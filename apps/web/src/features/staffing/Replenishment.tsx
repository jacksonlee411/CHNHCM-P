import { useEffect, useRef, useState } from 'react';
import { Alert, Box, Button, Paper, Stack, Typography } from '@mui/material';
import { Link, useParams, useSearchParams } from 'react-router-dom';
import { api, unwrap, type Models } from '@chnhcm/api-client';
import { ObjectDetailField, ObjectDetailSection, ObjectForm, ObjectTextField, RecordBackButton, RecordControl, UnsavedChangesDialog, recordedTime } from '@chnhcm/ui';
import { useDraftExit } from '../../shared/dated/useDraftExit';
import { usePageTitle } from '../../shared/navigation';
import { useRead } from '../personnel/api';
import { useRuleSubmission } from '../readiness/useRuleSubmission';
import { paths } from './api';

const route = '/api/v1/positions/{id}/replenishment' as const;
const read = async (id: string, day: string) =>
  unwrap(await api.GET(route, { params: {path: {id},query: {businessAt:day||undefined}} }));

export function ReplenishmentLink({id,day,back}:{id:string;day:string;back:string}) {
  const view=useRead(JSON.stringify([id,day]),()=>read(id,day));
  if(!view.data)return null;
  return <ObjectDetailSection title="岗位补员">
    <ObjectDetailField label="补员状态">{view.data.paused?'暂停补员':view.data.covered||view.data.records.some(r=>r.date<=day)?'允许补员':'尚未接管'}</ObjectDetailField>
    {view.data.exempt!==undefined&&<ObjectDetailField label="免占编任职">{view.data.exempt}</ObjectDetailField>}
    <Button component={Link} to={paths.position+'/'+id+'/replenishment?'+new URLSearchParams({day,back})}>查看补员安排</Button>
  </ObjectDetailSection>;
}

export function PositionReplenishment({session}:{session:Models['Principal']}) {
  const {id=''}=useParams(), [params]=useSearchParams();
  const base=paths.position+'/'+id, origin=params.get('back')??base;
  const back=origin.split('?')[0]===base?origin:base;
  const [day,setDay]=useState(params.get('day')??'');
  const [editing,setEditing]=useState(false),[reason,setReason]=useState('');
  const [message,setMessage]=useState(''),[error,setError]=useState('');
  const saved=useRef(false),failure=useRef<HTMLDivElement>(null),trigger=useRef<HTMLButtonElement>(null);
  usePageTitle('岗位补员');
  const view=useRead(JSON.stringify([id,day]),()=>read(id,day));
  const close=()=>{setEditing(false);setReason('');requestAnimationFrame(()=>trigger.current?.focus())};
  const leave=useDraftExit(editing&&!!reason.trim(),()=>saved.current,close);
  const header={'X-CSRF-Token':session.csrf};
  const send=useRuleSubmission<Models['ReplenishmentChange']>('position-replenishment-'+id,session,{
    send:async(cmd,key)=>{
      await unwrap(await api.POST(route,{params:{path:{id},header:{...header,'Idempotency-Key':key}},body:cmd}));
      return {submissionId:key,outcome:'accepted',version:key};
    },
    resolve:(sub)=>api.POST('/api/v1/positions/{id}/replenishment/submissions/{sub}/resolve',{params:{path:{id,sub},header},body:{}}).then(unwrap),
    finished:(r)=>{saved.current=true;close();view.refresh();setMessage(r.outcome==='accepted'?'已保存补员安排。':'原提交未生效。');setError('')},
    failed:(e)=>{setError(e instanceof Error?e.message:'暂时无法保存');view.refresh()},
  });
  useEffect(()=>{if(error)failure.current?.focus()},[error]);
  const action=view.data?.paused?'恢复补员':'暂停补员';
  return <Box className="standard-page object-page">
    <RecordControl navigation={null} identity={<><RecordBackButton component={Link} to={back}/>岗位补员</>} actions={null}/>
    <Box className="page-scroll" data-page-scroll tabIndex={0} role="region" aria-label="补员安排">
      <Paper sx={{p:{xs:1.5,sm:3},maxWidth:1000,mx:'auto'}}>
        <Typography variant="h5" component="h1">岗位补员</Typography>
        {(error||view.error)&&<Alert severity="error" ref={failure} tabIndex={-1}>{error||view.error}</Alert>}
        {message&&<Alert severity="success">{message}</Alert>}
        {send.pending.map(ref=><Button key={ref.submissionId} onClick={()=>void send.recover(ref)}>查回原提交</Button>)}
        {send.storageError&&<Alert severity="error">无法保存提交号，请恢复浏览器存储后办理。</Alert>}
        <ObjectForm>
          <ObjectTextField label="业务日期" type="date" value={day} onChange={e=>{setDay(e.target.value);setMessage('')}}/>
          {view.data&&<ObjectDetailField label="当日补员状态">{view.data.paused?'暂停补员':view.data.covered||view.data.records.some(r=>r.date<=day)?'允许补员':'尚未接管'}</ObjectDetailField>}
          {view.data?.exempt!==undefined&&<ObjectDetailField label="免占编任职">{view.data.exempt}</ObjectDetailField>}
          {view.data?.reserved!==undefined&&<ObjectDetailField label="招聘预留">{view.data.reserved}</ObjectDetailField>}
          {view.data?.occupied!==undefined&&<ObjectDetailField label="常规实占">{view.data.occupied}</ObjectDetailField>}
          {view.data?.canChange&&(editing?<><ObjectTextField autoFocus required label="补员原因" value={reason} onChange={e=>setReason(e.target.value)}/>
            <Stack direction="row" gap={1}><Button variant="contained" disabled={send.busy||!day||!reason.trim()} onClick={()=>void send.submit({date:day,reason,paused:!view.data!.paused,expected:view.data!.version})}>保存</Button><Button onClick={()=>leave.request(close)}>取消</Button></Stack></>
            :<Button ref={trigger} onClick={()=>{saved.current=false;setEditing(true);setError('');setMessage('');send.start()}}>{action}</Button>)}
          {!!view.data?.records.length&&<ObjectDetailSection title="补员记录">
            {view.data.records.map(r=><ObjectDetailField key={r.revision} label={r.date+' · '+(r.paused?'暂停补员':'恢复补员')}>{r.reason}<Typography component="time" dateTime={r.recordedAt} variant="body2">{recordedTime(r.recordedAt)}</Typography></ObjectDetailField>)}
          </ObjectDetailSection>}
        </ObjectForm>
      </Paper></Box><UnsavedChangesDialog {...leave.dialog}/>
  </Box>;
}
