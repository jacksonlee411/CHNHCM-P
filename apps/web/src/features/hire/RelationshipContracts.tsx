import { useEffect, useRef, useState } from 'react';
import { Alert, Box, Button, Stack, Typography } from '@mui/material';
import { api, unwrap, type Models } from '@chnhcm/api-client';
import { ObjectDetailField, ObjectTextField, recordedTime } from '@chnhcm/ui';
import { useRead, message } from '../personnel/api';
import { useRuleSubmission } from '../readiness/useRuleSubmission';
import { lastDutyDay } from './ActualDutyFields';

type Command=Models['EmploymentContractCommand'];
type Contract=Models['EmploymentContract'];
const labels={sign:'订立合同',renew:'续订合同',end:'终结合同'};

export function RelationshipContracts({view,session,draft,update,begin,cancel,finish}:{
  view:Models['AssignmentView'];session:Models['Principal'];draft?:Command;update:(v:Command)=>void;
  begin:(cmd:Command,button:HTMLButtonElement)=>void;cancel:()=>void;finish:()=>void;
}) {
  const [chosen,select]=useState(''),[error,setError]=useState('');
  const [notice,setNotice]=useState('');
  const alert=useRef<HTMLDivElement>(null);
  const list=useRead(JSON.stringify([view.assignment,view.businessAt,view.sourceVersion]),async()=>unwrap(await api.GET('/api/v1/assignments/{id}/contracts',{params:{path:{id:view.assignment},query:{businessAt:view.businessAt}}})));
  const items=list.data?.items??[];
  const current=items.find(c=>c.id===chosen);
  const header={'X-CSRF-Token':session.csrf};
  const submission=useRuleSubmission<Command,Models['ContractReceipt']>('employment-contract',session,{
    send:async(body,key)=>unwrap(await api.POST('/api/v1/assignments/{id}/contracts',{params:{path:{id:view.assignment},header:{...header,'Idempotency-Key':key}},body})),
    resolve:async(sub)=>unwrap(await api.POST('/api/v1/employment-contracts/submissions/{sub}/resolve',{params:{path:{sub},header}})),
    finished:(r,recovery)=>{
      if(r.outcome!=='accepted'){setError('原号已确认未执行。');return}
      setError('');list.invalidate();if(recovery!=='other'){select(r.id??'');setNotice(recovery?'已查回原合同办理。':'合同办理已保存。');finish()}
    },
    failed:e=>setError(message(e)),
  });
  useEffect(()=>{submission.start()},[submission.start]);
  useEffect(()=>{if(error)alert.current?.focus()},[error]);
  const edit=(op:Command['operation'],b:HTMLButtonElement,c?:Contract)=>{
    setError('');setNotice('');submission.start();
    begin({operation:op,expectedSourceVersion:view.sourceVersion,ref:c?{id:c.id,version:c.version}:undefined,
      ...(op==='end'?{firstInactive:view.businessAt}:{signed:view.businessAt,from:op==='renew'?c?.firstInactive||c?.until||view.businessAt:view.businessAt})},b);
  };
  const available=!!draft && (draft.operation==='end'?!!draft.firstInactive&&!!draft.basis?.trim():!!draft.number?.trim()&&!!draft.signed&&!!draft.from);
  return <Stack spacing={1.5}>
    <Typography component="h3">工作关系合同</Typography>
    <ObjectDetailField label="用人单位">{view.employerName}</ObjectDetailField>
    {list.error && <Alert severity="error" action={<Button onClick={list.refresh}>重试</Button>}>{list.error}</Alert>}
    {draft && list.data?.canManage ? <Box component="form" onSubmit={e=>{e.preventDefault();if(available&&!submission.busy)void submission.submit(draft)}}>
      <Stack spacing={1.5}>
        <Typography component="h4">{labels[draft.operation]}</Typography>
        {draft.ref && <ObjectDetailField label="原合同">{items.find(c=>c.id===draft.ref?.id)?.number}</ObjectDetailField>}
        {draft.operation==='end'?<>
          <ObjectTextField label="合同首个无效日" type="date" required value={draft.firstInactive??''} onChange={e=>update({...draft,firstInactive:e.target.value})}/>
          <ObjectTextField label="合同终结依据" required value={draft.basis??''} onChange={e=>update({...draft,basis:e.target.value})}/>
        </>:<>
          <ObjectTextField label="合同编号" required value={draft.number??''} onChange={e=>update({...draft,number:e.target.value})}/>
          <ObjectTextField label="签订日" type="date" required value={draft.signed??''} onChange={e=>update({...draft,signed:e.target.value})}/>
          <ObjectTextField label="合同起日" type="date" required value={draft.from??''} onChange={e=>update({...draft,from:e.target.value})}/>
          <ObjectTextField label="合同至（含当日）" type="date" value={draft.lastDay??''} onChange={e=>update({...draft,lastDay:e.target.value||undefined})}/>
        </>}
        <Stack direction="row" gap={1}><Button variant="contained" type="submit" disabled={!available||submission.busy}>保存合同</Button><Button onClick={cancel}>取消</Button></Stack>
      </Stack>
    </Box>:<>
      <Stack direction="row" flexWrap="wrap" gap={1}>
        {items.map(c=><Button key={c.id} aria-pressed={c.id===chosen} onClick={()=>select(c.id)}>{c.number}</Button>)}
        {list.data?.canManage && <Button onClick={e=>edit('sign',e.currentTarget)}>订立合同</Button>}
      </Stack>
      {list.data && !items.length && <Typography>本关系尚无合同。</Typography>}
      {current && <>
        <ObjectDetailField label="合同编号">{current.number}</ObjectDetailField>
        <ObjectDetailField label="签订日">{current.signed}</ObjectDetailField>
        <ObjectDetailField label="约定期限">{current.from} 至 {lastDutyDay(current.until)||'无固定期限'}</ObjectDetailField>
        {current.firstInactive && <ObjectDetailField label="合同首个无效日">{current.firstInactive}</ObjectDetailField>}
        {([['前任合同',current.previous],['后继合同',current.next]] as const).map(([label,id])=>id&&<ObjectDetailField key={label} label={label}><Button onClick={()=>select(id)}>{items.find(c=>c.id===id)?.number}</Button></ObjectDetailField>)}
        {current.processes.map(p=><ObjectDetailField key={p.version} label={labels[p.operation]}>{p.businessDate} · {p.basis} · {p.operator} · <time dateTime={p.recordedAt}>{recordedTime(p.recordedAt)}</time></ObjectDetailField>)}
        {list.data?.canManage && <Stack direction="row" gap={1} flexWrap="wrap">
          {!current.next && <Button onClick={e=>edit('renew',e.currentTarget,current)}>续订合同</Button>}
          {!current.firstInactive && <Button onClick={e=>edit('end',e.currentTarget,current)}>终结合同</Button>}
        </Stack>}
      </>}
    </>}
    {(error||notice) && <Alert ref={alert} tabIndex={-1} severity={error?'error':'success'}>{error||notice}</Alert>}
    {submission.storageError && <Alert severity="error">无法保存提交号，请恢复浏览器存储后办理。</Alert>}
    {submission.pending.map(p=><Button key={p.submissionId} onClick={()=>void submission.recover(p)}>查回合同办理</Button>)}
  </Stack>;
}
