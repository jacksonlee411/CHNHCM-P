import { useEffect, useRef, useState } from 'react';
import { Link, useLocation, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { Alert, Box, Button, Stack, Typography } from '@mui/material';
import { api, sessionGuard, unwrap, type Models } from '@chnhcm/api-client';
import { HistoryNotice, ObjectForm, RecordBackButton, RecordControl, RecordPager } from '@chnhcm/ui';
import { usePageTitle } from '../../shared/navigation';
import * as person from '../personnel/api';
import { EmployeeHire, employment } from './HirePage';
import { actionLabel, relationAction } from './AssignmentRecordForm';
import { JobInformation, OriginalEvidence } from './JobInformation';
import { registerPath, registerQuery } from './EmploymentRegister';
import { registerListParams } from './jobLinks';

export function RegisterActionPage({
  session,
  cancel = false,
}: {
  session: Models['Principal'];
  cancel?: boolean;
}) {
  const { actionId = '' } = useParams();
  return <ActionDetail key={actionId} actionId={actionId} session={session} cancel={cancel} />;
}

function ActionDetail({
  session,
  cancel,
  actionId,
}: {
  session: Models['Principal'];
  cancel: boolean;
  actionId: string;
}) {
  const [params, setParams] = useSearchParams(),
    navigate = useNavigate(),
    route = useLocation();
  const revision = params.get('adoptedRef') || '';
  const entered = params.get('action') || actionId;
  const original = person.useRead(actionId, async () =>
    unwrap(await api.GET('/api/v1/employment-actions/{id}', { params: { path: { id: actionId } } })),
  );
  const a = original.data?.action;
  const visibleAssignment = person.useRead(
    a ? JSON.stringify([a.assignment, a.businessDate, a.revision]) : undefined,
    async () =>
      unwrap(
        await api.GET('/api/v1/assignments/{id}', {
          params: { path: { id: a!.assignment }, query: { businessAt: a!.businessDate } },
        }),
      ),
  );
  const fallback = visibleAssignment.status === 403 || visibleAssignment.status === 404;
  const archived = person.useRead(
    fallback && (revision || entered !== actionId) ? JSON.stringify([entered, revision]) : undefined,
    async () =>
      unwrap(
        await api.GET('/api/v1/employment-actions/{id}', {
          params: { path: { id: entered }, query: { adoptedRef: revision || undefined } },
        }),
      ),
  );
  const overview = person.useRead(a?.person, () => employment(a!.person));
  const current = overview.data?.actions.find((v) => v.id === actionId);
  const related = overview.data?.actions.find((v) => v.original === actionId && v.canHistory);
  const canReturn = person.useRead(session.subjectId, async () =>
    unwrap(await api.GET('/api/v1/employment-actions/context')),
  );
  const list = registerListParams(params);
  list.set('selected', actionId);
  const listed = params.get('fromRegister') === '1';
  const applied = registerQuery(list);
  const readPosition = async () => unwrap(
    await api.GET('/api/v1/employment-actions', { params: { query: { ...applied, anchor: actionId } } }),
  );
  const listing = person.useRead(
    listed && canReturn.data?.canList && a ? JSON.stringify([applied, actionId]) : undefined,
    readPosition,
    'fresh',
  );
  const stepRequest = useRef(0);
  const [stepFailed, setStepFailed] = useState(false);
  useEffect(() => () => { stepRequest.current++; }, [route.key]);
  const nav = listing.data?.navigation;
  if (nav) list.set('page', String(Math.ceil(nav.position / applied.pageSize)));
  const details = registerPath + '/' + actionId + '?' + params;
  const [mode, setMode] = useState<'browse' | 'editing' | 'history'>('browse');
  const [actionSlot, setActionSlot] = useState<HTMLDivElement | null>(null);
  const [historySlot, setHistorySlot] = useState<HTMLDivElement | null>(null);
  const historical = !!params.get('sourceVersion') || (!!revision && revision !== a?.revision);
  const browsing = mode === 'browse' && !historical;
  const queryForAssignment = new URLSearchParams(params);
  queryForAssignment.set('returnAction', actionId);
  queryForAssignment.set('returnTo', details);
  queryForAssignment.set('action', entered);
  if (!queryForAssignment.has('assignmentDay') && a) queryForAssignment.set('assignmentDay', a.businessDate);
  function goRevision(value: string) {
    const q = new URLSearchParams(params);
    if (value) q.set('adoptedRef', value);
    else {
      q.delete('adoptedRef');
      q.delete('sourceVersion');
      original.invalidate();
      overview.invalidate();
    }
    setParams(q);
  }
  function toggleRelated() {
    if (!related) return;
    const q = new URLSearchParams(params);
    if (entered === related.id) q.delete('action');
    else q.set('action', related.id);
    q.delete('adoptedRef');
    q.delete('sourceVersion');
    setParams(q);
  }
  async function step(direction: 'next' | 'previous') {
    const request = ++stepRequest.current, address = location.href, current = sessionGuard();
    const active = () => request === stepRequest.current && location.href === address && current();
    let position;
    try {
      position = (await readPosition()).navigation;
    } catch {
      if (active()) setStepFailed(true);
      return;
    }
    if (!active()) return;
    const id = position?.[direction];
    if (!position || !id) {
      listing.invalidate();
      return;
    }
    const next = registerListParams(params);
    next.set('fromRegister', '1');
    next.set('selected', id);
    next.set('page', String(Math.ceil((position.position + (direction === 'next' ? 1 : -1)) / applied.pageSize)));
    navigate(registerPath + '/' + id + '?' + next, { state: { pagerFocus: direction } });
  }
  usePageTitle(cancel ? '撤销未来入职' : '办理明细');
  if (cancel && a && browsing && current?.canCancel)
    return (
      <EmployeeHire
        id={a.person}
        session={session}
        cancellation={{
          action: current,
          returnTo: details,
          done: () => navigate(details, { replace: true }),
        }}
      />
    );
  const readOnlyOriginal = revision || entered !== actionId ? archived.data : original.data;
  return (
    <Box className="standard-page object-page">
      <RecordControl
        identity={
          <>
            {canReturn.data?.canList && (
              <RecordBackButton component={Link} to={registerPath + '?' + list} label="返回办理记录" />
            )}
            办理记录
          </>
        }
        navigation={
          nav && !stepFailed && mode !== 'editing' ? (
            <RecordPager
              previous={!!nav.previous}
              next={!!nav.next}
              counter={{ position: nav.position, count: listing.data!.total }}
              onChange={step}
              focusOnMount={route.state?.pagerFocus}
            />
          ) : null
        }
        actions={
          <Stack direction="row" gap={1} flexWrap="wrap" alignItems="center">
            {a && browsing && (
              <>
                {entered === actionId && current?.canRehire && (
                  <Button
                    component={Link}
                    to={
                      person.path +
                      '/' +
                      a.person +
                      '/employment?' +
                      new URLSearchParams({
                        rehireAction: a.id,
                        rehireRevision: a.revision,
                        returnTo: details,
                      })
                    }
                  >
                    办理再聘
                  </Button>
                )}
                {entered === actionId && current?.canCancel && (
                  <Button component={Link} to={registerPath + '/' + actionId + '/cancel?' + params}>
                    撤销未来入职
                  </Button>
                )}
                {visibleAssignment.data && (
                  <Button
                    component={Link}
                    to={
                      person.path + '/' + a.person + '/assignments/' + a.assignment + '?' + queryForAssignment
                    }
                  >
                    查看任职
                  </Button>
                )}
              </>
            )}
            <div ref={setActionSlot} style={{ display: 'contents' }} />
          </Stack>
        }
      />
      <div ref={setHistorySlot} style={{ display: 'contents' }} />
      {fallback && historical && <HistoryNotice onReturn={() => goRevision('')} label="返回当前原件" />}
      <Box className="page-scroll" role="region" aria-label="办理明细" tabIndex={0} data-page-scroll>
        {original.error && (
          <Alert severity="error" action={<Button onClick={original.refresh}>重试</Button>}>
            {original.error}
          </Alert>
        )}
        {(listing.error || stepFailed || (listing.data && !nav)) && (
          <Alert severity="info" action={<Button onClick={() => { setStepFailed(false); listing.invalidate(); }}>重试导航</Button>}>
            {listing.error || stepFailed ? '暂时无法读取查询内的位置。' : '这笔办理已不在当前查询范围内。'}
          </Alert>
        )}
        {a && (
          <>
            <Typography component="h1" variant="h5">
              {actionLabel(a)} ·{' '}
              {relationAction(a.kind)
                ? '工作关系'
                : a.recordNumber + ' 号' + (a.primaryJob ? '主岗' : '兼岗')}
            </Typography>
            {cancel && overview.data && !current?.canCancel && (
              <Alert severity="info">当前不可撤销这笔办理。</Alert>
            )}
            {related && browsing && (
              <Button onClick={toggleRelated}>
                {entered === related.id ? '返回入职原件' : '查看撤销依据'}
              </Button>
            )}
            <ObjectForm>
              {visibleAssignment.data && (
                <JobInformation
                  key={entered}
                  session={session}
                  assignment={a.assignment}
                  day={params.get('assignmentDay') || a.businessDate}
                  selectedAction={entered}
                  onDay={(day) => {
                    const q = new URLSearchParams(params);
                    q.set('assignmentDay', day);
                    setParams(q);
                  }}
                  actionSlot={actionSlot}
                  historySlot={historySlot}
                  onModeChange={setMode}
                  onSave={() => {
                    original.invalidate();
                    overview.invalidate();
                    listing.invalidate();
                  }}
                />
              )}
              {fallback && !params.get('sourceVersion') && readOnlyOriginal && (
                <OriginalEvidence original={readOnlyOriginal} setRevision={goRevision} />
              )}
              {fallback && params.get('sourceVersion') && (
                <Alert severity="info">当前无法读取所选任职来源。</Alert>
              )}
              {archived.error && <Alert severity="error">{archived.error}</Alert>}
              {visibleAssignment.error && !fallback && (
                <Alert severity="error" action={<Button onClick={visibleAssignment.refresh}>重试</Button>}>
                  {visibleAssignment.error}
                </Alert>
              )}
            </ObjectForm>
          </>
        )}
      </Box>
    </Box>
  );
}
