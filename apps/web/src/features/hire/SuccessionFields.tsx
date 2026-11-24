import {useState} from 'react';
import {Autocomplete, TextField} from '@mui/material';
import {api,unwrap} from '@chnhcm/api-client';
import {ObjectDetailField} from '@chnhcm/ui';
import {useRead} from '../personnel/api';
import type {JobDraft} from './AssignmentRecordForm';

export function SuccessionFields({draft,update,assignment,csrf}:{draft:JobDraft;update:(v:JobDraft)=>void;assignment:string;csrf:string}){
  const [search,setSearch]=useState('');
  const position=draft.position?.id;
  const options=useRead(position&&draft.day?JSON.stringify([assignment,position,draft.day,search]):undefined,async()=>{
    const header={'X-CSRF-Token':csrf};
    const ref=search.trim()?unwrap(await api.POST('/api/v1/persons/queries',{params:{header},body:{keyword:search.trim()}})).ref:undefined;
    return unwrap(await api.POST('/api/v1/assignments/{id}/succession-options',{params:{path:{id:assignment},header},body:{position:position!,ref,businessDate:draft.day,page:1}}));
  });
  return <>
    <ObjectDetailField label="被接替任职" inputId="incumbent" required>
      <Autocomplete id="incumbent" fullWidth size="small" options={options.data??[]} value={draft.incumbent??null}
        filterOptions={v=>v} isOptionEqualToValue={(a,b)=>a.assignment===b.assignment}
        getOptionLabel={v=>v.name+' · '+v.recordNumber+' 号'}
        onInputChange={(_,v,r)=>{if(r==='input'||r==='clear')setSearch(v)}}
        onChange={(_,v)=>update({...draft,incumbent:v??undefined})}
        noOptionsText={options.error??'没有已确定离岗日期的原任职'}
        renderInput={params=><TextField {...params} variant="standard" required error={!!options.error} helperText={options.error}/>}/>
    </ObjectDetailField>
    {draft.incumbent && <>
      <ObjectDetailField label="岗位所属部门">{draft.incumbent.departmentName}</ObjectDetailField>
      <ObjectDetailField label="岗位对应职务">{draft.incumbent.jobName}</ObjectDetailField>
      <ObjectDetailField label="名额安排">{draft.day} 起占交接名额，{draft.incumbent.until}{draft.incumbent.continues?' 起占常规名额':' 起结束交接'}</ObjectDetailField>
    </>}
  </>;
}
