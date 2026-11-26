import { useState } from 'react';
import { Alert, Box, Button, Stack, Typography } from '@mui/material';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { api, unwrap, type Models } from '@chnhcm/api-client';
import { ObjectForm, EffectiveDates, ObjectDetailIdentity, ObjectDetailSheet, ObjectTabPanel, RecordBackButton, RecordControl, RecordPager } from '@chnhcm/ui';
import * as person from '../personnel/api';
import { usePageTitle } from '../../shared/navigation';
import { employment } from './HirePage';
import PersonIcon from '@mui/icons-material/Person';
import {PersonTabs} from '../personnel/PersonTabs';
import { JobInformation, type JobPageMode } from './JobInformation';
import { employmentHistoryReturn, personnelReturn } from './jobLinks';

export function AssignmentPage({ session }: { session: Models['Principal'] }) {
  const { id = '', assignment = '' } = useParams(),
    [params] = useSearchParams();
  return (
    <AssignmentDetail
      key={assignment}
      id={id}
      assignment={assignment}
      day={params.get('assignmentDay') ?? ''}
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
  const [historySlot, setHistorySlot] = useState<HTMLDivElement | null>(null);
  const [pageMode, setPageMode] = useState<JobPageMode>('browse');
  const [params, setParams] = useSearchParams();
  const source = params.get('sourceVersion') || undefined;
  const view = person.useRead(JSON.stringify([assignment, day, source]), async () =>
    unwrap(
      await api.GET('/api/v1/assignments/{id}', {
        params: { path: { id: assignment }, query: { businessAt: day || undefined, sourceVersion:source } },
      }),
    ),
  );
  const data = view.data?.person === id ? view.data : undefined;
  const profile = person.useRead(data?.primaryJob ? id : undefined, () => person.profile(id));
  const history = person.useRead(data && !source ? id : undefined,()=>employment(id));
  const otherTerms = history.data?.terms.filter(t=>t.assignment!==assignment);
  const listed=params.get('from')==='directory', query=person.parse(params);
  const anchor=params.get('selected') || assignment;
  const navigation=person.useRead(listed ? anchor+person.directoryParams(params) : undefined,()=>person.directory(session,params,anchor));
  const nav=navigation.data?.navigation;
  async function move(direction:'previous'|'next') {
    const entry=location.href;
    try {
      const fresh=await person.directory(session,params,anchor);
      if(location.href!==entry)return;
      const position=fresh.navigation,target=position?.[direction];
      if(!position || !target){navigation.refresh();return;}
      const next=person.directoryParams(params);
      next.set('page',String(Math.ceil((position.position+(direction==='previous'?-1:1))/query.pageSize)));
      navigate(person.directoryTarget(next,target.person,target.assignment));
    } catch {if(location.href===entry)navigation.refresh();}
  }
  const backQuery=person.directoryParams(params);
  const returnAction = params.get('returnAction');
  const registerQuery = new URLSearchParams(params);
  for (const key of ['returnAction', 'returnTo', 'assignmentDay', 'action', 'sourceVersion', 'sourceEntry']) registerQuery.delete(key);
  if (params.get('action') !== returnAction) registerQuery.delete('adoptedRef');
  const originalReturn = personnelReturn(params.get('returnTo'));
  const registerOrigin = originalReturn && originalReturn.split('?')[0] === '/personnel/actions/' + returnAction ? originalReturn : undefined;
  const historyOrigin = employmentHistoryReturn(params.get('returnTo'), id);
  const back = returnAction
    ? registerOrigin || '/personnel/actions/' + returnAction + '?' + registerQuery
    : historyOrigin || (listed ? person.path+'?'+backQuery : person.path+'/'+id+'?'+backQuery);
  const personQuery=person.directoryParams(params);
  if(listed)personQuery.set('from','directory');
  personQuery.set('assignment',assignment);
  const selectedDay=day || data?.businessAt;
  if(selectedDay)personQuery.set('assignmentDay',selectedDay);
  const path=person.path+'/'+id;
  const secondary = new URLSearchParams(params);
  if (selectedDay) secondary.set('assignmentDay', selectedDay);
  const legacy = new URLSearchParams(params);
  legacy.set('view', 'history');
  const sourceAssignment = personnelReturn(path + '/assignments/' + assignment + '?' + params);
  if (sourceAssignment) legacy.set('returnTo', sourceAssignment);
  const browsing = pageMode === 'browse' && !source;
  const showRelatedTabs = pageMode !== 'history' && !source;
  const dates=data?.effectiveDates || [];
  const tabs=showRelatedTabs && !!profile.data;
  const explicitUnavailable=!!data && !!day && !source && !dates.includes(day);
  return (
    <Box className="standard-page object-page">
      <RecordControl
        identity={
          <>
            <RecordBackButton component={Link} to={back} label={returnAction ? '返回办理明细' : historyOrigin ? '返回办理历史' : listed ? '返回人员' : '返回本人'} />
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
            {browsing && (data?.canAddSecondary||data?.canAddParallel)&&<Button component={Link} to={path+'/assignments/'+assignment+'/additional?'+secondary}>新增兼岗</Button>}
            {browsing && !!history.data?.actions.length &&<Button component={Link} to={path + '/employment?' + legacy}>
              本人办理历史
            </Button>}
          </Stack>
        }
      />
      <div ref={setHistorySlot} style={{display:'contents'}} />
      <Box className="page-scroll" data-page-scroll role="region" aria-label="任职详情" tabIndex={0}>
        {view.error && (
          <Alert severity="error" action={<Button onClick={view.refresh}>重试</Button>}>
            {view.error}
          </Alert>
        )}
        {view.data && !data && <Alert severity="error">所选任职不可用</Alert>}
        {explicitUnavailable && <Alert severity="info">所选生效节点不可用</Alert>}
        {data && !explicitUnavailable && <ObjectDetailSheet
          identity={<ObjectDetailIdentity icon={<PersonIcon/>} title={data.name} summary={
            [data.number,`${data.recordNumber} 号${data.primaryJob?'主岗':'兼岗'}`,data.employerName,
              data.relationship==='Employee'?'员工关系':data.relationship,data.from+' 起',data.futureTerm?'尚未生效':'',
              data.relationshipPrimary && !data.primaryJob?'本工作关系主任职':''].filter(Boolean).join(' · ')}/>}
          tabs={tabs && <PersonTabs selected="job" targets={{profile:path+'?'+personQuery, job:location.pathname+location.search,
            history:profile.data?.history?path+'?'+personQuery+'&tab=history':undefined}}/>}
          timeline={browsing && dates.length>0 && <EffectiveDates dates={dates} selected={selectedDay} onSelect={value=>{
            setParams(previous=>{const next=new URLSearchParams(previous);next.set('assignmentDay',value);
              for(const key of ['action','sourceVersion','adoptedRef','sourceEntry'])next.delete(key);return next;});
          }}/>}
        >
          {browsing && !!otherTerms?.length && <Stack direction="row" alignItems="center" flexWrap="wrap"><Typography>其他任职</Typography>
            {otherTerms.map(term=>{
              const next=person.directoryParams(params);
              if(listed)next.set('from','directory');
              return <Button component={Link} key={term.assignment} to={path+'/assignments/'+term.assignment+'?'+next}>
                {term.recordNumber} 号{term.recordNumber===0?'主岗':'兼岗'} · {term.employerName} · {term.from} 起{term.until?'，'+term.until+' 起不在任':''}
              </Button>;
            })}</Stack>}
          <Box role={tabs?'tabpanel':undefined} id="person-job-panel" aria-labelledby={tabs?'person-job-tab':undefined}>
            <ObjectForm><JobInformation
              enabled={!!data}
              current={{...view, data}}
              assignment={assignment}
              day={day || data?.businessAt || ''}
              session={session}
              actionSlot={actionSlot}
              historySlot={historySlot}
              onModeChange={setPageMode}
              onSave={view.refresh}
            /></ObjectForm>
          </Box>
          {tabs && <ObjectTabPanel prefix="person" value="profile" selected="job">{null}</ObjectTabPanel>}
          {tabs && profile.data?.history && <ObjectTabPanel prefix="person" value="history" selected="job">{null}</ObjectTabPanel>}
        </ObjectDetailSheet>}
      </Box>
    </Box>
  );
}
