import {offerAccess,offerPath} from '../recruiting/Offers';
import { useEffect, useLayoutEffect, useRef, useState } from 'react';
import { useSearchParams, useParams, Link, useLocation, useNavigate } from 'react-router-dom';
import { Alert, Box, Button, IconButton, Menu, MenuItem, Tab, Tabs } from '@mui/material';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import PersonIcon from '@mui/icons-material/Person';
import {
  ObjectDetailSheet,
  ObjectTabPanel,
  ObjectDetailIdentity,
  ObjectDetailSection,
  ObjectDetailField,
  RecordControl,
  RecordBackButton,
  RecordPager,
  HistoryNotice,
} from '@chnhcm/ui';
import { employment } from "../hire/HirePage";
import { OperationSummary } from '../../shared/dated/OperationLog';
import { usePageTitle } from '../../shared/navigation';
import * as person from './api';
import { ProfileSections } from './ProfileSections';
import { Editor } from './Editor';
import { PersonChangeLog, type LogPosition } from './ChangeLog';
export { actionLabel } from './ChangeLog';
export const kindLabel: Record<string, string> = {
  personal: '本人核对记录',
  form: '人事登记单',
  phone: '电话',
  email: '电子邮箱',
};
export const documentLabels: Record<string, string> = {
  resident_id: '居民身份证',
  passport: '护照',
  other: '其他身份证件',
};
function DocumentFields({ data, version }: { data: person.Profile; version?: string }) {
  const [show, setShow] = useState(false);
  const full = person.useRead(
    show && data.canReadDocument ? data.id + data.sourceVersion : undefined,
    () => person.profile(data.id, version, 'fullDocument'),
    'fresh',
  );
  const doc = full.data?.primaryDocument || data.primaryDocument;
  if (!doc) return null;
  return (
    <ObjectDetailSection title="主要证件">
      <ObjectDetailField label="证件类型">{documentLabels[doc.kind]}</ObjectDetailField>
      <ObjectDetailField label="证件号码">{doc.number}</ObjectDetailField>
      {data.canReadDocument && (
        <Button onClick={() => setShow(!show)}>{show ? '隐藏号码' : '查看完整号码'}</Button>
      )}
      {show && full.error && (
        <Alert action={<Button onClick={full.refresh}>重试</Button>}>{full.error}</Alert>
      )}
    </ObjectDetailSection>
  );
}
type DetailProps = { session: person.Principal; creating?: boolean };
export function PersonDetail(props: DetailProps) {
  const { id } = useParams();
  return (
    <PersonDetailView key={props.session.instanceId + props.session.subjectId + (id ?? '')} {...props} />
  );
}

function PersonDetailView({ session, creating = false }: DetailProps) {
  const [positions] = useState(() => new Map<string, LogPosition>());
  const identity = useRef<HTMLDivElement>(null);
  function focusIdentity() {
    const heading = identity.current?.querySelector('h1');
    if (heading) {
      heading.tabIndex = -1;
      heading.focus();
    }
  }
  const route = useLocation();
  const { id = '' } = useParams(),
    [params, setParams] = useSearchParams(),
    navigate = useNavigate();
  const mode = creating ? 'register' : params.get('action'),
    version = params.get('source') || undefined,
    log = params.get('tab') === 'history';
  const restoredSection = useRef('');
  const wasSource = useRef(log);
  const query = person.parse(params),
    access = person.useRead(creating ? 'entry' : undefined, person.entry);
  const record = person.useRead(id ? JSON.stringify([id, version, mode]) : undefined, () =>
    person.profile(id, version, mode === 'correct' && !version ? 'edit' : undefined),
  );
  const offers=person.useRead(record.data && !creating && !version && !mode ? id:undefined,()=>offerAccess(id));
  const assignment = params.get('assignment');
  const assignmentDay = assignment ? params.get('assignmentDay') ?? params.get('businessAt') ?? undefined : undefined;
  const hire = person.useRead(
    record.data && !creating && !version && !mode ? JSON.stringify([id, assignmentDay]) : undefined,
    () => employment(id, assignmentDay),
  );
  const hasJobTab = !!assignment && !!hire.data?.terms.some(
    (term) => term.assignment === assignment && term.recordNumber === 0,
  );
  const data = record.data,
    nav = person.useRead(data && params.has('from') ? id + JSON.stringify(query) : undefined, () =>
      person.navigation(id, query),
    );
  // Reauthorize the basic record when returning from history, including browser Back.
  // Keep the tab controls mounted while its fields and actions await the fresh projection.
  const [checkingProfile, setCheckingProfile] = useState(false);
  const lastLog = useRef(log);
  useLayoutEffect(() => {
    const leaving = lastLog.current && !log;
    lastLog.current = log;
    if (leaving && !version && !mode && data?.history) {
      setCheckingProfile(true);
      record.invalidate();
    } else if (data || record.error) {
      if (checkingProfile && data && !data.history) focusIdentity();
      setCheckingProfile(false);
    }
  }, [log, data, mode, version, record.error, record.invalidate, checkingProfile]);
  const showHistory = !version && (data?.history || (checkingProfile && !data && !record.error));
  const hasTabs = showHistory || hasJobTab;
  const selected = showHistory && log ? 'history' : 'profile';
  useEffect(() => {
    if (version) {
      wasSource.current = true;
      return;
    }
    if (record.error || (data && !showHistory)) positions.clear();
    if (data && log && !showHistory) {
      const current = new URLSearchParams(params);
      current.delete('tab');
      setParams(current, { replace: true });
      focusIdentity();
    }
  }, [data, showHistory, log, params, record.error, setParams, version]);
  useEffect(() => {
    const target = route.state?.profileSection;
    if (
      !data ||
      mode ||
      version ||
      selected !== 'profile' ||
      !target ||
      restoredSection.current === route.key
    )
      return;
    const section = document.getElementById('profile-' + target);
    if (section) {
      restoredSection.current = route.key;
      section.focus({ preventScroll: true });
      section.scrollIntoView({ block: 'nearest' });
    }
  }, [data, mode, version, selected, route.key, route.state]);
  const replayId: string | undefined = route.state?.replayedSubmission;
  const receipt = person.useRead(data && replayId && !mode && !version ? id + replayId : undefined, () =>
    person.resolve(session, replayId!),
  );
  const [anchor, setAnchor] = useState<HTMLElement | null>(null),
    [navError, setNavError] = useState('');
  usePageTitle(creating ? '登记人员' : data?.name || '员工资料');
  function edit(values: Record<string, string>) {
    const next = new URLSearchParams(params);
    for (const [k, v] of Object.entries(values)) {
      if (v) next.set(k, v);
      else next.delete(k);
    }
    setParams(
      next,
      version && values.source === '' ? { state: { logVisit: route.state?.logVisit } } : undefined,
    );
  }
  const backParams = new URLSearchParams(params);
  for (const k of ['from', 'source', 'tab', 'action']) backParams.delete(k);
  if (data) {
    backParams.set('selected', id);
    if (nav.data) backParams.set('page', String(Math.ceil(nav.data.position / query.pageSize)));
  }
  const cancelParams = new URLSearchParams(params);
  cancelParams.delete('action');
  const assignmentParams = new URLSearchParams(params);
  for (const key of ['assignment', 'tab', 'action', 'source']) assignmentParams.delete(key);
  if (assignmentParams.get('returnTo') === 'assignments') {
    assignmentParams.set('from', 'assignments');
    assignmentParams.delete('returnTo');
  }
  const assignmentPath = person.path + '/' + id + '/assignments/' + assignment + '?' + assignmentParams;
  const back = assignment ? assignmentPath : person.path + '?' + backParams;
  const permitted =
    mode === 'register'
      ? access.data?.canRegister
      : mode === 'correct'
        ? !!data?.editable?.length
        : mode === 'renumber' && data?.renumber;
  async function move(dir: 'next' | 'previous') {
    const identity = location.href;
    try {
      const fresh = await person.navigation(id, query),
        target = dir === 'next' ? fresh.next : fresh.prev;
      if (location.href !== identity) return;
      if (!target) {
        nav.refresh();
        return;
      }
      setNavError('');
      const next = new URLSearchParams(params);
      next.delete('source');
      next.delete('tab');
      next.set('page', String(Math.ceil((fresh.position + (dir === 'next' ? 1 : -1)) / query.pageSize)));
      next.set('selected', target);
      navigate(person.path + '/' + target + '?' + next);
    } catch (e) {
      if (location.href === identity) setNavError(person.message(e));
    }
  }
  if (mode && permitted && !version)
    return (
      <Editor
        key={session.instanceId + session.subjectId + id + mode}
        mode={mode}
        session={session}
        profile={data}
        entry={access.data}
        back={creating ? back : person.path + '/' + id + '?' + cancelParams}
        recheck={creating ? access.refresh : record.refresh}
        saved={(result, section) => {
          if (result.id) {
            const next = new URLSearchParams(params);
            next.delete('action');
            next.delete('source');
            navigate(person.path + '/' + result.id + '?' + next, {
              replace: true,
              state: {
                replayedSubmission: result.originalNumber ? result.submissionId : undefined,
                profileSection: section,
                noChange: result.outcome === 'no_change',
              },
            });
            record.refresh();
          } else navigate(back, { replace: true, state: { registered: mode === 'register' } });
        }}
      />
    );
  return (
    <Box className="standard-page object-page">
      <RecordControl
        identity={
          <>
            <RecordBackButton component={Link} to={back} />
            {data?.number}
            <>
              {data?.renumber && !version && (
                <IconButton aria-label="更多操作" onClick={(e) => setAnchor(e.currentTarget)}>
                  <MoreVertIcon />
                </IconButton>
              )}
            </>
          </>
        }
        actions={
          <>
            {data?.editable?.length && !version ? (
              <Button variant="contained" onClick={() => edit({ action: 'correct' })}>
                更正资料
              </Button>
            ) : null}
            {offers.data?.canRead&&<Button component={Link} to={offerPath+'?person='+id}>招聘录用</Button>}
            {hire.data?.rehireFrom && <Button component={Link} to={person.path+'/'+id+'/employment?rehireAction='+hire.data.rehireFrom.action+'&rehireRevision='+hire.data.rehireFrom.revision}>办理再聘</Button>}
            {!hasJobTab && hire.data && (hire.data.canHire || hire.data.actions.length > 0) && (
              <Button component={Link} to={person.path + '/' + id + '/employment?' + params}
                state={{ hire: hire.data.actions.length === 0 }}>
                {hire.data.actions.length ? '工作关系与任职' : '办理入职'}
              </Button>
            )}
          </>
        }
        navigation={
          nav.data && (
            <RecordPager
              previous={!!nav.data.prev}
              next={!!nav.data.next}
              counter={{ position: nav.data.position, count: nav.data.total }}
              onChange={move}
            />
          )
        }
      />
      <Menu open={!!anchor && !!data?.renumber} anchorEl={anchor} onClose={() => setAnchor(null)}>
        <MenuItem
          onClick={() => {
            setAnchor(null);
            edit({ action: 'renumber' });
          }}
        >
          改号
        </MenuItem>
      </Menu>
      {version && (
        <HistoryNotice
          onReturn={() => {
            if (params.has('from') || log) edit({ source: '', tab: 'history' });
            else if (window.history.state?.idx > 0) navigate(-1);
            else navigate('/');
          }}
        />
      )}
      <Box className="page-scroll" data-page-scroll role="region" tabIndex={0} aria-label="人员资料">
        {data && receipt.data?.id === id && receipt.data.originalNumber && (
          <Alert severity="info">这张登记单已办过，原登记的人员编号为 {receipt.data.originalNumber}。</Alert>
        )}
        {data && route.state?.noChange && <Alert severity="info">没有修改内容</Alert>}
        {(record.error || access.error) && (
          <Alert
            severity="error"
            action={
              <Button
                onClick={() => {
                  record.refresh();
                  access.refresh();
                }}
              >
                重试
              </Button>
            }
          >
            {record.error || access.error}
          </Alert>
        )}
        {(nav.error || navError) && data && (
          <Alert
            action={
              <Button
                onClick={() => {
                  setNavError('');
                  nav.refresh();
                }}
              >
                重试
              </Button>
            }
          >
            暂时无法切换记录
          </Alert>
        )}
        {mode && !permitted && (data || access.data) && <Alert severity="warning">没有本次办理权限</Alert>}
        {(data || hasTabs) && (
          <ObjectDetailSheet
            identity={
              <div ref={identity}>
                <ObjectDetailIdentity
                  icon={<PersonIcon />}
                  title={data?.name || (version ? '办理原件' : '个人资料')}
                  summary={data?.number}
                />
              </div>
            }
            tabs={
              hasTabs && (
                <Tabs
                  aria-label="人员详情"
                  value={selected}
                  variant="scrollable"
                  allowScrollButtonsMobile
                  onChange={(_, tab: string) => {
                    if (tab === 'job') return;
                    wasSource.current = false;
                    edit({ tab: tab === 'history' ? 'history' : '' });
                  }}
                >
                  <Tab
                    value="profile"
                    id="person-profile-tab"
                    aria-controls="person-profile-panel"
                    label="个人资料"
                  />
                  {hasJobTab && <Tab component={Link} to={assignmentPath} value="job" label="职务信息" />}
                  {showHistory && (
                    <Tab
                      value="history"
                      id="person-history-tab"
                      aria-controls="person-history-panel"
                      label="变更日志"
                    />
                  )}
                </Tabs>
              )
            }
          >
            {hasTabs && (
              <ObjectTabPanel prefix="person" value="history" selected={selected}>
                {selected === 'history' && (
                  <PersonChangeLog
                    key={route.key}
                    id={id}
                    session={session}
                    params={params}
                    positions={positions}
                    returning={wasSource.current}
                    recheckProfile={record.refresh}
                  />
                )}
              </ObjectTabPanel>
            )}
            {profilePanel(
              !!hasTabs,
              selected,
              selected === 'profile' && data && (
                <>
                  {!version && (
                    <ProfileSections data={data} recheck={record.refresh} />
                  )}
                  {!version && data.primaryDocument && !data.readable?.includes('documentKind') && (
                    <DocumentFields data={data} />
                  )}
                  {version && (
                    <>
                      <Box className="object-detail-columns">
                        <ObjectDetailSection title="基本资料">
                          {data.name && <ObjectDetailField label="姓名">{data.name}</ObjectDetailField>}
                          {data.number && (
                            <ObjectDetailField label="人员编号">{data.number}</ObjectDetailField>
                          )}
                        </ObjectDetailSection>
                        {data.primaryDocument && (
                          <DocumentFields
                            key={id + data.sourceVersion + (version || '')}
                            data={data}
                            version={version}
                          />
                        )}
                        {version && data.evidence && (
                          <ObjectDetailSection title="登记来源">
                            <ObjectDetailField label="材料类型">
                              {kindLabel[data.evidence.kind]}
                            </ObjectDetailField>
                            <ObjectDetailField label="材料编号">{data.evidence.reference}</ObjectDetailField>
                          </ObjectDetailSection>
                        )}
                      </Box>
                      {data.contacts && (
                        <ObjectDetailSection title="联系方式">
                          {data.contacts.map((x) => (
                            <ObjectDetailField key={x.itemId} label={kindLabel[x.kind]}>
                              {x.value}
                            </ObjectDetailField>
                          ))}
                        </ObjectDetailSection>
                      )}
                    </>
                  )}
                  {data.decision && (
                    <ObjectDetailSection title="改号决定">
                      <ObjectDetailField label="原号">{data.decision.oldNumber}</ObjectDetailField>
                      <ObjectDetailField label="客户决定文号">{data.decision.decisionRef}</ObjectDetailField>
                      <ObjectDetailField label="原因">
                        {data.decision.reason === 'correction' ? '编号纠错' : '制度迁移'}
                      </ObjectDetailField>
                      <ObjectDetailField label="有效截止日">
                        {data.decision.validUntil || '长期有效'}
                      </ObjectDetailField>
                    </ObjectDetailSection>
                  )}
                  {data.disposed && <Alert severity="info">资料已按保留政策处置</Alert>}
                  <OperationSummary summary={data} />
                </>
              ),
            )}
          </ObjectDetailSheet>
        )}
      </Box>
    </Box>
  );
}

function profilePanel(tabs: boolean, selected: string, content: React.ReactNode) {
  return tabs ? (
    <ObjectTabPanel prefix="person" value="profile" selected={selected}>
      {content}
    </ObjectTabPanel>
  ) : (
    content
  );
}
