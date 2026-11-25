import { useState } from 'react';
import { Alert, Box, Button, Stack, Tab, Tabs, Typography } from '@mui/material';
import { Link, Navigate, useLocation, useNavigate, useParams, useSearchParams } from 'react-router-dom';
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
  const manual = !!route.state?.hire || params.get('view') === 'history' || params.has('rehireAction');
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
    return <Navigate replace to={person.path + '/' + id + '/assignments/' + terms[0].assignment + route.search} state={route.state} />;
  if (terms.length > 1) {
    const selection = new URLSearchParams({ person: id });
    const businessAt = params.get('businessAt');
    if (businessAt) selection.set('businessAt', businessAt);
    return <Navigate replace to={employees + '?' + selection} />;
  }
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
  const [params, setParams] = useSearchParams();
  function applyDay(value: string) {
    setParams(previous => {
      const next = new URLSearchParams(previous);
      next.set('assignmentDay', value);
      return next;
    }, { replace: true });
  }
  const view = person.useRead(JSON.stringify([assignment, day]), async () =>
    unwrap(
      await api.GET('/api/v1/assignments/{id}', {
        params: { path: { id: assignment }, query: { businessAt: day || undefined } },
      }),
    ),
  );
  const data = view.data?.person === id ? view.data : undefined;
  const profile = person.useRead(data?.primaryJob ? id : undefined, () => person.profile(id));
  const history = person.useRead(data?.primaryJob ? JSON.stringify([id,day]) : undefined,()=>employment(id,day||undefined));
  const otherTerms = history.data?.terms.filter(t=>t.recordNumber===0 && t.assignment!==assignment);
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
  backQuery.delete('assignmentDay');
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
  const selectedDay = day || data?.businessAt;
  if (selectedDay) personQuery.set('assignmentDay', selectedDay);
  const path = person.path + '/' + id;
  const secondary = new URLSearchParams(params);
  if (selectedDay) secondary.set('assignmentDay', selectedDay);
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
          <Stack direction="row" flexWrap="wrap" gap={1}>
            <span ref={setActionSlot} />
            {(data?.canAddSecondary||data?.canAddParallel)&&<Button component={Link} to={path+'/assignments/'+assignment+'/additional?'+secondary}>新增兼岗</Button>}
            {data?.primaryJob&&<Button component={Link} to={path + '/employment?' + legacy}>
              办理历史
            </Button>}
          </Stack>
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
            {data.futureTerm && <Typography>尚未生效</Typography>}
            {data.relationshipPrimary && !data.primaryJob && <Typography>本工作关系主任职</Typography>}
            {otherTerms && otherTerms.length>0 && <Stack direction="row" flexWrap="wrap" alignItems="center">
              <Typography>同号任期</Typography>
              {otherTerms.map(term=>{
                const q=new URLSearchParams(params);q.set('assignmentDay',term.from);
                return <Button component={Link} key={term.assignment} to={person.path+'/'+id+'/assignments/'+term.assignment+'?'+q}>{term.from} 起</Button>
              })}
            </Stack>}
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
          </>
        )}
        <Stack component="section" role="tabpanel" sx={{display: data ? undefined : 'none'}}>
          <ObjectForm>
            <JobInformation
              enabled={!!data}
              current={{...view, data}}
              assignment={assignment}
              day={day || data?.businessAt || ''}
              onDay={applyDay}
              session={session}
              actionSlot={actionSlot}
              onSave={view.refresh}
            />
          </ObjectForm>
        </Stack>
      </Box>
    </Box>
  );
}
