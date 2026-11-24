import { useEffect, useRef, useState } from 'react';
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom';
import { Alert, Box, Button, IconButton, Menu, MenuItem, Tab, Tabs, Typography } from '@mui/material';
import MoreVert from '@mui/icons-material/MoreVert';
import WorkOutline from '@mui/icons-material/WorkOutline';
import {
  EffectiveDates,
  HistoryNotice,
  ObjectDetailField,
  ObjectDetailIdentity,
  ObjectDetailSection,
  ObjectDetailSheet,
  ObjectTabPanel,
  RecordBackButton,
  RecordControl,
  RecordPager,
  recordedTime,
} from '@chnhcm/ui';
import { api, ApiError, sessionGuard, unwrap, type Models } from '@chnhcm/api-client';
import { useMaintenanceSearchParams, usePageTitle } from '../../shared/navigation';
import { StaffingChangeLog, logKeys, logReturnHref } from './ChangeLog';
import { QuotaEditor, permittedQuotaDraft, type QuotaDraft } from './QuotaEditor';
import {
  labels,
  missingQuotaRecord,
  quotaSavedNotice,
  receipts,
  period,
  positionQuotaPath,
  quotaLabel,
  quotaPath,
  type Action,
  type Position,
  type Result,
} from './quotaApi';
import { ownedReference } from '../../shared/dated/submission';
import * as staff from './api';

export function QuotaDetail({ session }: { session: Models['Principal'] }) {
  const { id = '' } = useParams(),
    [params, setParams] = useMaintenanceSearchParams(staff.paths.position),
    navigate = useNavigate(),
    location = useLocation();
  const version = params.get('sourceVersion') ?? '',
    record = params.get('selectedRecordId') ?? '';
  const proposedBack = params.get('back') ?? quotaPath;
  const backPath = proposedBack.split('?')[0];
  const departmentBack = /^\/settings\/staffing\/quotas\/departments\/[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}$/i.test(backPath);
  const back = departmentBack || [quotaPath, staff.paths.position].includes(backPath) ? proposedBack : quotaPath;
  const [retry, setRetry] = useState(0),
    [error, setError] = useState(''),
    [notice, setNotice] = useState('');
  const [edit, setEdit] = useState<Action>(),
    [draft, setDraft] = useState<QuotaDraft>(),
    [menu, setMenu] = useState<HTMLElement | null>(null);
  const pending = useRef<
    { identity: string; action: Action; target?: string; draft: QuotaDraft } | undefined
  >(undefined);
  const [response, setResponse] = useState<{ key: string; view: Position }>();
  const [nav, setNav] = useState<Models['StaffingNavigation']>(),
    [navError, setNavError] = useState(false),
    [navRetry, setNavRetry] = useState(0);
  const restoreAction = useRef(false);
  const more = useRef<HTMLButtonElement>(null),
    primary = useRef<HTMLButtonElement>(null);
  const identity = JSON.stringify([id, record, version, session.instanceId, session.subjectId]),
    key = JSON.stringify([identity, retry, session.csrf]);
  const view = response?.key === key ? response.view : undefined;
  const history = params.get('tab') === 'history' && !!view?.historyReadFields;
  const returnLog = logReturnHref(params.get('logReturn'), location.pathname);
  usePageTitle(view?.name ?? '岗位编制');
  function patch(values: Record<string, string>, clearLog = false) {
    const next = new URLSearchParams(params);
    if (clearLog) logKeys.forEach((k) => next.delete(k));
    for (const [k, v] of Object.entries(values)) {
      if (v) next.set(k, v);
      else next.delete(k);
    }
    setParams(next);
  }
  useEffect(() => {
    const abort = new AbortController(),
      same = sessionGuard();
    setResponse(undefined);
    setEdit(undefined);
    setDraft(undefined);
    setMenu(null);
    setError('');
    if (pending.current?.identity !== identity) pending.current = undefined;
    api
      .GET('/api/v1/staffing-quotas/positions/{positionId}', {
        params: {
          path: { positionId: id },
          query: { sourceVersion: version || undefined, selectedRecordId: record || undefined },
        },
        signal: abort.signal,
      })
      .then(unwrap)
      .then((v) => {
        if (abort.signal.aborted || !same()) return;
        const attempt = pending.current;
        pending.current = undefined;
        if (
          attempt &&
          !v.selectionMissing &&
          v.actions?.includes(attempt.action) &&
          v.recordId === attempt.target
        ) {
          setDraft(permittedQuotaDraft(attempt.draft, attempt.action, v));
          setEdit(attempt.action);
        }
        setResponse({ key, view: v });
      })
      .catch((e) => {
        if (!abort.signal.aborted && same()) {
          if (e instanceof ApiError && [401, 403, 404].includes(e.status)) {
            pending.current = undefined;
            if (
              e.status !== 401 &&
              location.state?.fromPositionPager &&
              back.split('?')[0] === staff.paths.position
            ) {
              navigate(staff.paths.position + '/' + id + '?' + new URLSearchParams({ back }), {
                replace: true,
              });
              return;
            }
          }
          setError(e.message);
        }
      });
    return () => abort.abort();
  }, [key]);
  useEffect(() => {
    const clear = () => {
      pending.current = undefined;
      setDraft(undefined);
      setEdit(undefined);
      setResponse(undefined);
      setNav(undefined);
    };
    window.addEventListener('chnhcm-session-invalidated', clear);
    return () => window.removeEventListener('chnhcm-session-invalidated', clear);
  }, []);
  useEffect(() => {
    const abort = new AbortController(),
      same = sessionGuard(),
      q = new URLSearchParams(back.split('?')[1]);
    setNav(undefined);
    setNavError(false);
    if (!params.has('back') || departmentBack) return;
    const request =
      back.split('?')[0] === staff.paths.position
        ? staff.navigation('position', id, staff.queryFrom(q), abort.signal)
        : api
            .GET('/api/v1/staffing-quotas/statistics', {
              params: {
                query: {
                  anchor: id,
                  q: q.get('q') ?? '',
                  departmentId: q.get('departmentId') ?? '',
                  businessAt: q.get('businessAt') ?? '',
                  pageSize: 1,
                },
              },
              signal: abort.signal,
            })
            .then(unwrap)
            .then((v) => v.navigation);
    request
      .then((v) => {
        if (!abort.signal.aborted && same()) setNav(v);
      })
      .catch(() => {
        if (!abort.signal.aborted && same()) setNavError(true);
      });
    return () => abort.abort();
  }, [id, back, navRetry, session.csrf]);
  function move(dir: 'previous' | 'next') {
    const target = dir === 'next' ? nav?.nextId : nav?.previousId;
    if (!target || !nav) return;
    const [path, query] = back.split('?'),
      q = new URLSearchParams(query);
    q.set('selected', target);
    q.set(
      'page',
      String(Math.ceil((nav.position + (dir === 'next' ? 1 : -1)) / Number(q.get('pageSize') ?? 20))),
    );
    const dest = new URLSearchParams({ back: path + '?' + q });
    const recordId = dir === 'next' ? nav.nextRecordId : nav.previousRecordId;
    if (recordId) dest.set('selectedRecordId', recordId);
    navigate(positionQuotaPath(target, dest), {
      state: { fromPositionPager: path === staff.paths.position },
    });
  }
  function close() {
    setEdit(undefined);
    setDraft(undefined);
    setRetry((n) => n + 1);
    restoreAction.current = true;
  }
  function saved(r: Result) {
    setEdit(undefined);
    setDraft(undefined);
    setRetry((n) => n + 1);
    setNotice(quotaSavedNotice(r));
    const original = r.originalSourceVersion || (r.replayed ? r.sourceVersion : '');
    patch({ sourceVersion: original ?? '', selectedRecordId: r.selectedRecordId ?? '', tab: 'basic' }, true);
  }
  useEffect(() => {
    if (view && restoreAction.current) {
      restoreAction.current = false;
      (more.current ?? primary.current)?.focus({ preventScroll: true });
    }
  }, [view]);
  useEffect(
    () =>
      receipts.subscribe(
        ({ reference, result }) => {
          if (
            !edit &&
            !version &&
            ownedReference(reference, session) &&
            result.positionId === id &&
            result.sourceVersion &&
            result.selectedRecordId
          )
            saved(result);
        },
        () => {},
      ),
    [edit, version, id, session, location.key],
  );
  if (edit && view && !view.selectionMissing)
    return (
      <QuotaEditor
        key={key + edit}
        session={session}
        view={view}
        record={view}
        action={edit}
        draft={draft}
        close={close}
        done={saved}
        reread={(input, denied) => {
          pending.current = { identity, action: edit, target: view.recordId, draft: input };
          setDraft(undefined);
          setEdit(undefined);
          setResponse(undefined);
          setRetry((n) => n + 1);
          setNotice(denied ? '权限已变化，已重新读取可见资料。' : '已重读记录，请核对当前资料和输入。');
        }}
      />
    );
  const actions = (view?.actions ?? [])
    .filter((a): a is Action => Object.hasOwn(labels, a))
    .filter((a) => {
      if (a === 'withdraw') return view?.quotaState === 'set';
      if (['correct', 'redate', 'delete'].includes(a) && !view?.recordId) return false;
      return a !== 'delete' || (view?.records?.findIndex((r) => r.recordId === view.recordId) ?? 0) > 0;
    });
  const definition = new URLSearchParams({
    back: back.split('?')[0] === staff.paths.position ? back : staff.paths.position,
  });
  if (params.has('positionRecordId')) definition.set('selectedRecordId', params.get('positionRecordId')!);
  const tab = history ? 'history' : 'basic';
  return (
    <Box className="standard-page object-page">
      <RecordControl
        identity={
          <>
            <RecordBackButton
              component={Link}
                label={departmentBack ? '返回部门编制' : '返回列表'}
              to={
                view?.canReadDefinition === false && back.split('?')[0] === staff.paths.position
                  ? quotaPath
                  : back
              }
            />
            {view?.code && <Typography>{view.code}</Typography>}
            {!version && !history && actions.length > 1 && (
              <IconButton
                ref={more}
                aria-label="更多操作"
                aria-haspopup="menu"
                onClick={(e) => setMenu(e.currentTarget)}
              >
                <MoreVert />
              </IconButton>
            )}
          </>
        }
        actions={
          !version &&
          !history &&
          actions.length > 0 && (
            <Button ref={primary} variant="contained" onClick={() => setEdit(actions[0])}>
              {labels[actions[0]]}
            </Button>
          )
        }
        navigation={
          nav && (
            <RecordPager
              previous={!!nav.previousId}
              next={!!nav.nextId}
              counter={nav.position ? { position: nav.position, count: nav.totalCount } : undefined}
              onChange={move}
            />
          )
        }
      />
      <Menu anchorEl={menu} open={!!menu} onClose={() => setMenu(null)}>
        {actions.slice(1).map((a) => (
          <MenuItem
            key={a}
            onClick={() => {
              setMenu(null);
              setEdit(a);
            }}
          >
            {labels[a]}
          </MenuItem>
        ))}
      </Menu>
      {version && <HistoryNotice onReturn={() => patch({ sourceVersion: '', tab: 'basic' }, true)} />}
      {returnLog && (
        <Box>
          <Button component={Link} to={returnLog}>
            返回变更日志
          </Button>
        </Box>
      )}
      <Box className="page-scroll" data-page-scroll role="region" tabIndex={0} aria-label="岗位编制资料">
        {notice && <Alert onClose={() => setNotice('')}>{notice}</Alert>}
        {error && (
          <Alert severity="error" action={<Button onClick={() => setRetry((n) => n + 1)}>重试</Button>}>
            {error}
          </Alert>
        )}
        {navError && (
          <Alert action={<Button onClick={() => setNavRetry((n) => n + 1)}>重试</Button>}>
            暂时无法切换记录
          </Alert>
        )}
        {nav?.position === 0 && <Alert>当前岗位不在原查询结果中。</Alert>}
        {!view && !error && <Typography role="status">正在读取…</Typography>}
        {view?.selectionMissing && (
          <Alert
            severity="warning"
            action={<Button onClick={() => patch({ selectedRecordId: '' })}>查看默认记录</Button>}
          >
            {missingQuotaRecord}
          </Alert>
        )}
        {view?.selectedRecordIsFuture && !history && <Alert severity="info">所选记录尚未生效。</Alert>}
        {view && (
          <ObjectDetailSheet
            identity={
              <ObjectDetailIdentity icon={<WorkOutline />} title={view.name ?? view.code ?? '岗位编制'} />
            }
            tabs={
              <Tabs
                value={tab}
                variant="scrollable"
                allowScrollButtonsMobile
                onChange={(_, value) => {
                  if (value === 'position') {
                    navigate(staff.paths.position + '/' + id + '?' + definition);
                    return;
                  }
                  patch({ tab: value }, true);
                }}
              >
                {view.canReadDefinition && <Tab value="position" label="基本资料" />}
                <Tab id="quota-basic-tab" aria-controls="quota-basic-panel" value="basic" label="岗位编制" />
                {view.historyReadFields && (
                  <Tab
                    id="quota-history-tab"
                    aria-controls="quota-history-panel"
                    value="history"
                    label="变更日志"
                  />
                )}
              </Tabs>
            }
            timeline={
              !history &&
              !!view.records?.some((r) => r.effectiveDate) && (
                <EffectiveDates
                  dates={view.records.flatMap((r) => (r.effectiveDate ? [r.effectiveDate] : []))}
                  selected={view.selectionMissing ? undefined : view.effectiveDate}
                  onSelect={(day) =>
                    patch({
                      selectedRecordId: view.records?.find((r) => r.effectiveDate === day)?.recordId ?? '',
                    })
                  }
                />
              )
            }
          >
            <ObjectTabPanel prefix="quota" value="basic" selected={tab}>
              {!view.selectionMissing && (
                <>
                  <Box className="object-detail-columns">
                    <ObjectDetailSection title="编制资料">
                      {view.departmentName && (
                        <ObjectDetailField label="所属部门">{view.departmentName}</ObjectDetailField>
                      )}
                      {view.active !== undefined && (
                        <ObjectDetailField label="岗位状态">
                          {view.active ? '启用' : '停用'}
                        </ObjectDetailField>
                      )}
                      <ObjectDetailField label="核定编制">{quotaLabel(view)}</ObjectDetailField>
                    </ObjectDetailSection>
                    <ObjectDetailSection title="记录信息">
                      {view.effectiveDate && (
                        <ObjectDetailField label="适用期间">{period(view)}</ObjectDetailField>
                      )}
                      {view.reason && <ObjectDetailField label="调整原因">{view.reason}</ObjectDetailField>}
                    </ObjectDetailSection>
                  </Box>
                  {view.operator && (
                    <Typography className="operation-summary" variant="body2">
                      操作者 {view.operator}　操作时间{' '}
                      {view.recordedAt && (
                        <time dateTime={view.recordedAt}>{recordedTime(view.recordedAt)}</time>
                      )}
                    </Typography>
                  )}
                  {view.usageState && (
                    <details>
                      <summary>编制使用情况</summary>{view.usageState === 'known' ? '用量已核齐' : '用量未知'}
                    </details>
                  )}
                </>
              )}
            </ObjectTabPanel>
            {history && view.quotaId && (
              <ObjectTabPanel prefix="quota" value="history" selected={tab}>
                <StaffingChangeLog
                  kind="staffingquota"
                  id={view.quotaId}
                  onReadFields={(fields) => {
                    if (
                      fields.length !== view.historyReadFields?.length ||
                      fields.some((f) => !view.historyReadFields?.includes(f))
                    ) {
                      setResponse(undefined);
                      setRetry((n) => n + 1);
                    }
                  }}
                  onDenied={() => {
                    setResponse(undefined);
                    setRetry((n) => n + 1);
                  }}
                />
              </ObjectTabPanel>
            )}
          </ObjectDetailSheet>
        )}
      </Box>
    </Box>
  );
}
