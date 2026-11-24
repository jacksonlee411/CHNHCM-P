import { useState } from 'react';
import { Alert, Box, Button, Stack, Tab, Tabs, Typography } from '@mui/material';
import { Link, useLocation, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { api, unwrap, type Models } from '@chnhcm/api-client';
import { ObjectForm, RecordBackButton, RecordControl, RecordPager } from '@chnhcm/ui';
import * as person from '../personnel/api';
import { usePageTitle } from '../../shared/navigation';
import { employment, HirePage } from './HirePage';
import { employees } from './EmployeeDirectory';
import { JobInformation } from './JobInformation';

export function EmploymentEntry({ session }: { session: Models['Principal'] }) {
  const { id = '' } = useParams(),
    route = useLocation(),
    [params] = useSearchParams();
  const manual = !!route.state?.hire || params.get('view') === 'history';
  const overview = person.useRead(manual ? undefined : id + (params.get('businessAt') ?? ''), () =>
    employment(id, params.get('businessAt') ?? undefined),
  );
  if (manual) return <HirePage session={session} />;
  if (overview.error)
    return (
      <Alert severity="error" action={<Button onClick={overview.refresh}>重试</Button>}>
        {overview.error}
      </Alert>
    );
  if (!overview.data) return null;
  const terms = overview.data.terms;
  if (terms.length === 1)
    return (
      <AssignmentDetail
        key={terms[0].assignment}
        id={id}
        assignment={terms[0].assignment}
        day={params.get('businessAt') ?? terms[0].from}
        session={session}
      />
    );
  if (terms.length > 1)
    return (
      <div className="standard-page">
        <Typography component="h1">选择任职</Typography>
        {terms.map((a) => (
          <Button
            key={a.assignment}
            component={Link}
            to={person.path + '/' + id + '/assignments/' + a.assignment + '?' + params}
          >
            {a.recordNumber} 号 · {a.from}
          </Button>
        ))}
      </div>
    );
  return <HirePage session={session} />;
}
export function AssignmentPage({ session }: { session: Models['Principal'] }) {
  const { id = '', assignment = '' } = useParams(),
    [params] = useSearchParams();
  return (
    <AssignmentDetail
      key={assignment}
      id={id}
      assignment={assignment}
      day={params.get('assignmentDay') ?? params.get('businessAt') ?? ''}
      session={session}
    />
  );
}
function AssignmentDetail({
  id,
  assignment,
  day,
  session,
}: {
  id: string;
  assignment: string;
  day: string;
  session: Models['Principal'];
}) {
  usePageTitle('职务信息');
  const navigate = useNavigate();
  const [actionSlot, setActionSlot] = useState<HTMLSpanElement | null>(null);
  const [params] = useSearchParams(),
    [date, setDate] = useState(day);
  const view = person.useRead(JSON.stringify([assignment, day]), async () =>
    unwrap(
      await api.GET('/api/v1/assignments/{id}', {
        params: { path: { id: assignment }, query: { businessAt: day || undefined } },
      }),
    ),
  );
  const data = view.data?.person === id ? view.data : undefined;
  const profile = person.useRead(data?.primaryJob ? id : undefined, () => person.profile(id));
  const listed = params.get('from') === 'assignments';
  const query = person.parse(params);
  const navigation = person.useRead(listed ? params.toString() + assignment : undefined, async () =>
    unwrap(
      await api.POST('/api/v1/assignments/query', {
        params: { header: { 'X-CSRF-Token': session.csrf } },
        body: {
          ...query,
          pageSize: query.pageSize as 20 | 50 | 100,
          businessAt: params.get('businessAt') ?? undefined,
          department: params.get('department') ?? undefined,
          person: params.get('person') ?? undefined,
          anchor: assignment,
        },
      }),
    ),
  );
  const nav = navigation.data?.navigation;
  function move(direction: 'previous' | 'next') {
    if (!nav) return;
    const target = direction === 'previous' ? nav.previous : nav.next;
    if (!target) return;
    const q = new URLSearchParams(params);
    q.set('selected', target.assignment);
    const position = nav.position + (direction === 'previous' ? -1 : 1);
    q.set('page', String(Math.ceil(position / query.pageSize)));
    navigate(person.path + '/' + target.person + '/assignments/' + target.assignment + '?' + q);
  }
  const backQuery = new URLSearchParams(params);
  backQuery.delete('from');
  backQuery.delete('view');
  const returnAction = params.get('returnAction');
  const registerQuery = new URLSearchParams(params);
  for (const key of ['returnAction', 'assignmentDay']) registerQuery.delete(key);
  const back = returnAction
    ? '/personnel/actions/' + returnAction + '?' + registerQuery
    : params.get('from') === 'assignments'
      ? employees + '?' + backQuery
      : person.path + '/' + id + '?' + backQuery;
  const personQuery = new URLSearchParams(params);
  personQuery.delete('from');
  if (listed) personQuery.set('returnTo', 'assignments');
  personQuery.set('assignment', assignment);
  const path = person.path + '/' + id;
  const secondary=new URLSearchParams(params);if(date||data?.businessAt)secondary.set("assignmentDay",date||data!.businessAt);
  const legacy = new URLSearchParams(params);
  legacy.set('view', 'history');
  return (
    <Box className="standard-page object-page">
      <RecordControl
        identity={
          <>
            <RecordBackButton component={Link} to={back} label={returnAction ? '返回办理明细' : undefined} />
            {data?.name} {data?.number}
          </>
        }
        navigation={
          data &&
          nav && (
            <RecordPager
              previous={!!nav.previous}
              next={!!nav.next}
              counter={{ position: nav.position, count: navigation.data!.total }}
              onChange={move}
            />
          )
        }
        actions={
          data && (
            <Stack direction="row" flexWrap="wrap" gap={1}>
              <span ref={setActionSlot} />
              {(data.canAddSecondary||data.canAddParallel)&&<Button component={Link} to={path+'/assignments/'+assignment+'/additional?'+secondary}>新增兼岗</Button>}
              {data.primaryJob&&<Button component={Link} to={path + '/employment?' + legacy}>
                办理历史
              </Button>}
            </Stack>
          )
        }
      />
      <Box className="page-scroll" data-page-scroll role="region" aria-label="任职详情" tabIndex={0}>
        {view.error && (
          <Alert severity="error" action={<Button onClick={view.refresh}>重试</Button>}>
            {view.error}
          </Alert>
        )}
        {view.data && !data && <Alert severity="error">所选任职不可用</Alert>}
        {data && (
          <>
            <Typography component="h1" variant="h5">
              {data.name} · {data.recordNumber} 号{data.primaryJob ? '主岗' : '兼岗'}
            </Typography>
            <Typography>
              {data.employerName} · {data.relationship === 'Employee' ? '员工关系' : data.relationship} · 自{' '}
              {data.from} 起
            </Typography>
            {data.relationshipPrimary && !data.primaryJob && <Typography>本工作关系主任职</Typography>}
            <Tabs value="job" aria-label="人员详情" variant="scrollable">
              {profile.data && (
                <Tab component={Link} to={path + '?' + personQuery} value="person" label="个人资料" />
              )}
              <Tab value="job" label="职务信息" />
              {profile.data?.history && (
                <Tab
                  component={Link}
                  to={path + '?' + personQuery + '&tab=history'}
                  value="history"
                  label="变更日志"
                />
              )}
            </Tabs>
            <Stack component="section" role="tabpanel">
              <ObjectForm>
                <JobInformation
                  assignment={assignment}
                  day={date || data.businessAt}
                  onDay={setDate}
                  session={session}
                  actionSlot={actionSlot}
                  onSave={view.refresh}
                />
              </ObjectForm>
            </Stack>
          </>
        )}
      </Box>
    </Box>
  );
}
