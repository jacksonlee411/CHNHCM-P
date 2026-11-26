import { MenuItem, Typography } from '@mui/material';
import { api, unwrap, type Models } from '@chnhcm/api-client';
import { ObjectDetailField, ObjectTextField } from '@chnhcm/ui';
import { useRead } from '../personnel/api';
import type { JobDraft } from './AssignmentRecordForm';

export function lastDutyDay(until?:string) {
  if (!until) return '';
  const date=new Date(until+'T00:00:00Z');
  date.setUTCDate(date.getUTCDate()-1);
  return date.toISOString().slice(0,10);
}

export function DutyOriginal({evidence}:{evidence:Models['DutyEvidence']}) {
  return <>
    <ObjectDetailField label="派岗决定">{evidence.dispatchNumber}</ObjectDetailField>
    <ObjectDetailField label="出勤原件">{evidence.issuer} · {evidence.number}</ObjectDetailField>
    <ObjectDetailField label="实际履职期间">{evidence.from} 至 {lastDutyDay(evidence.until)||'持续履职'}</ObjectDetailField>
    <ObjectDetailField label="出勤核实至">{evidence.observedThrough}</ObjectDetailField>
  </>;
}

export function DutyUsage({position,day,version}:{position:string;day:string;version:string}) {
  const read=useRead(JSON.stringify([position,day,version]),async()=>unwrap(await api.GET('/api/v1/positions/{id}/replenishment',{params:{path:{id:position},query:{businessAt:day}}})));
  const v=read.data;
  if (!v && !read.error) return <ObjectDetailField label="岗位用量">正在读取…</ObjectDetailField>;
  if(v?.quota==null||v.occupied==null||v.excess==null)return null;
  return <ObjectDetailField label="岗位用量">核定 {v.quota} · 实占 {v.totalOccupied??v.occupied} · 超编 {v.excess}</ObjectDetailField>;
}

export function ActualDutyFields({draft,update,view}:{draft:JobDraft;update:(d:JobDraft)=>void;view:Models['AssignmentView']}) {
  const read=useRead(view.assignment,async()=>unwrap(await api.GET('/api/v1/assignments/{id}/duty-sources',{params:{path:{id:view.assignment}}})));
  const choices=read.data?.decisions??[];
  const original=draft.originalDuty;
  const observed=(read.data?.observations??[]).filter(v=>!original||v.id===original.id && v.version!==original.version);
  const paper=draft.paper;
  const edition=(ref:Models['DecisionRef'])=>JSON.stringify([ref.source,ref.version,ref.line]);
  const setPaper=(changes:Partial<Models['DutyObservation']>)=>update({...draft,paper:{...paper,...changes},day:changes.from??draft.day});
  const newPaper=original?{dispatch:original.dispatch,issuer:original.issuer,number:original.number,from:original.from,lastDay:lastDutyDay(original.until),observedThrough:original.observedThrough,previous:{id:original.id,version:original.version}}:{from:draft.day};
  return <>
    {read.error && <Typography role="alert" color="error">{read.error}</Typography>}
    <ObjectTextField select required label="履职资料" value={paper?'paper':draft.observation?.version??''} onChange={e=>{
      if(e.target.value==='paper') {update({...draft,observation:undefined,paper:newPaper,day:newPaper.from});return}
      const observation=observed.find(v=>v.version===e.target.value);
      update({...draft,paper:undefined,observation,day:observation?.from??draft.day});
    }}>
      {observed.map(v=><MenuItem key={v.version} value={v.version}>{v.issuer} · {v.number} · {v.from}</MenuItem>)}
      {view.canVerifyDuty && <MenuItem value="paper">{original?'更正原出勤资料':'登记出勤原件'}</MenuItem>}
    </ObjectTextField>
    {draft.observation && <DutyOriginal evidence={draft.observation}/>}
    {paper && <>
      <ObjectTextField select required label="派岗决定" value={paper.dispatch?edition(paper.dispatch):''} onChange={e=>{
        const v=choices.find(v=>edition(v.ref)===e.target.value);
        if(v)setPaper({dispatch:v.ref,from:original?paper.from:v.arrival, lastDay:original?paper.lastDay:undefined});
      }}>
        {choices.map(v=><MenuItem key={edition(v.ref)} value={edition(v.ref)}>{v.number} · {v.position?.name}</MenuItem>)}
      </ObjectTextField>
      <ObjectTextField required label="出勤出具方" value={paper.issuer??''} disabled={!!original} onChange={e=>setPaper({issuer:e.target.value})}/>
      <ObjectTextField required label="出勤原件编号" value={paper.number??''} disabled={!!original} onChange={e=>setPaper({number:e.target.value})}/>
      <ObjectTextField type="date" required label="实际履职开始日" value={paper.from??''} onChange={e=>setPaper({from:e.target.value})}/>
      <ObjectTextField type="date" label="最后履职日" value={paper.lastDay??''} onChange={e=>setPaper({lastDay:e.target.value||undefined})}/>
      <ObjectTextField type="date" required label="出勤核实至" value={paper.observedThrough??''} onChange={e=>setPaper({observedThrough:e.target.value})}/>
    </>}
  </>;
}
