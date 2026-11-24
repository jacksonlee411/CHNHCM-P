import { useState } from 'react';
import { Autocomplete, MenuItem, TextField } from '@mui/material';
import { api, unwrap, type Models } from '@chnhcm/api-client';
import { ObjectDetailField, ObjectTextField } from '@chnhcm/ui';
import { useRead } from '../personnel/api';
import type { JobDraft } from './AssignmentRecordForm';

export const managerPurposes = { direct: '直属汇报', functional: '专业汇报' };
export function LifecycleFields({ draft, update, view, csrf }: {draft: JobDraft; update: (draft: JobDraft) => void; view: Models['AssignmentView']; csrf: string}) {
  const [search,setSearch]=useState('');
  const manager=draft.kind==='SetManager';
  const matches=useRead(manager && !draft.endReporting && draft.day && search.trim() ? JSON.stringify([draft.day,search]) : undefined, async()=>{
    const header={'X-CSRF-Token':csrf};
    const query=unwrap(await api.POST('/api/v1/persons/queries',{params:{header},body:{keyword:search.trim()}}));
    const result=unwrap(await api.POST('/api/v1/assignments/query',{params:{header},body:{businessAt:draft.day,ref:query.ref,page:1,pageSize:50}}));
    return result.rows.filter(r=>r.id!==view.assignment).map(r=>({id:r.id,name:r.name+' · '+r.recordNumber+' 号 · '+r.departmentName+' · '+r.jobName}));
  });
  if (!manager) return <>
    <ObjectTextField select label="原通知日期含义" value={draft.meaning ?? 'last-active'} onChange={e=>update({...draft,meaning:e.target.value as JobDraft['meaning']})}>
      <MenuItem value="last-active">最后在职日（含当日）</MenuItem><MenuItem value="first-inactive">首个不在职日</MenuItem>
    </ObjectTextField>
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
