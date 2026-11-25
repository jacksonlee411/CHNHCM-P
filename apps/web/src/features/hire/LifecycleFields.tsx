import { useState } from 'react';
import { Alert, Autocomplete, Button, MenuItem, TextField } from '@mui/material';
import { api, unwrap, type Models } from '@chnhcm/api-client';
import { ObjectDetailField, ObjectTextField } from '@chnhcm/ui';
import { useRead } from '../personnel/api';
import type { JobDraft } from './AssignmentRecordForm';

export const managerPurposes = { direct: '直属汇报', functional: '专业汇报' };
export function LifecycleFields({ draft, update, view, csrf, impact, contractsConfirmed }: {draft: JobDraft; update: (draft: JobDraft) => void; view: Models['AssignmentView']; csrf: string; impact: ReturnType<typeof useRead<Models['EmploymentLeaveImpact']>>; contractsConfirmed: boolean}) {
  const [search,setSearch]=useState('');
  const manager=draft.kind==='SetManager';
  const matches=useRead(manager && !draft.endReporting && draft.day && search.trim() ? JSON.stringify([draft.day,search]) : undefined, async()=>{
    const header={'X-CSRF-Token':csrf};
    const query=unwrap(await api.POST('/api/v1/persons/queries',{params:{header},body:{keyword:search.trim()}}));
    const result=unwrap(await api.POST('/api/v1/assignments/query',{params:{header},body:{businessAt:draft.day,ref:query.ref,page:1,pageSize:50}}));
    return result.rows.filter(r=>r.id!==view.assignment).map(r=>({id:r.id,name:r.name+' · '+r.recordNumber+' 号 · '+r.departmentName+' · '+r.jobName}));
  });
  const required = impact.data?.contracts ?? [];
  if (!manager) return <>
    {draft.kind === 'Leave' && <>
      {impact.data && <>
        <ObjectDetailField label="所属工作关系">{impact.data.employerName} · 自 {impact.data.from} 起</ObjectDetailField>
        <ObjectDetailField label="本次离职将结束的任职">{impact.data.terms.map(term =>
          term.recordNumber + ' 号' + (term.recordNumber === 0 ? '主岗' : '兼岗')).join('、')}</ObjectDetailField>
      </>}
      {impact.error && <Alert severity="error" action={<Button onClick={impact.refresh}>重试</Button>}>{impact.error}</Alert>}
    </>}
    {draft.kind === 'EndAssignment' && <ObjectDetailField label="本次结束的任职">{view.recordNumber} 号兼岗 · {view.employerName}</ObjectDetailField>}
    <ObjectTextField select label="原通知日期含义" value={draft.meaning ?? 'last-active'} onChange={e=>update({...draft,meaning:e.target.value as JobDraft['meaning'],contracts:undefined})}>
      <MenuItem value="last-active">最后在职日（含当日）</MenuItem><MenuItem value="first-inactive">首个不在职日</MenuItem>
    </ObjectTextField>
    {required.length>0 && <>
      <ObjectDetailField label="涉及的合同">{required.map(contract => contract.number).join('、')}</ObjectDetailField>
      {impact.data?.canManageContracts ? <ObjectTextField select required label="合同处置" value={contractsConfirmed?'end':''} onChange={e=>update({...draft,contracts:e.target.value==='end'?required.map(({id,version})=>({id,version})):undefined})}>
        <MenuItem value="">请选择</MenuItem><MenuItem value="end">按离职通知终结上述合同</MenuItem>
      </ObjectTextField> : <Alert severity="warning">请由有合同处置权限的经办人办理离职。</Alert>}
    </>}
    <ObjectTextField required label="退出依据" value={draft.notice} onChange={e=>update({...draft,notice:e.target.value})}/>
  </>;
  return <>
    <ObjectTextField select label="汇报用途" value={draft.purpose ?? 'direct'} onChange={e=>update({...draft,purpose:e.target.value as JobDraft['purpose']})}>
      {Object.entries(managerPurposes).map(([key,label])=><MenuItem key={key} value={key}>{label}</MenuItem>)}
    </ObjectTextField>
    <ObjectTextField select label="汇报安排" value={draft.endReporting ? 'end' : 'assign'} onChange={e=>update({...draft,endReporting:e.target.value==='end',manager:undefined})}>
      <MenuItem value="assign">指定实际经理</MenuItem><MenuItem value="end">结束该用途汇报</MenuItem>
    </ObjectTextField>
    {!draft.endReporting && <>
      <ObjectDetailField label="经理任职" inputId="actual-manager" required>
        <Autocomplete id="actual-manager" fullWidth size="small" options={matches.data ?? []} value={draft.manager ?? null}
          getOptionLabel={o=>o.name} filterOptions={v=>v} isOptionEqualToValue={(a,b)=>a.id===b.id}
          onInputChange={(_,value,reason)=>{if(reason==='input'||reason==='clear')setSearch(value)}}
          onChange={(_,option)=>update({...draft,manager:option ?? undefined})}
          noOptionsText={matches.error ?? (search.trim() ? '没有匹配的有效任职' : '输入经理姓名或人员编号')}
          renderInput={params=><TextField {...params} variant="standard" required error={!!matches.error} helperText={matches.error}/>}/>
      </ObjectDetailField>
      <ObjectTextField type="date" label="汇报至（含当日）" value={draft.managerLastDay ?? ''} onChange={e=>update({...draft,managerLastDay:e.target.value})}/>
    </>}
  </>;
}
