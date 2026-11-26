import { useState } from 'react';
import { Autocomplete, Button, TextField } from '@mui/material';
import { api, unwrap, type Models } from '@chnhcm/api-client';
import { ObjectDetailField, ObjectTextField } from '@chnhcm/ui';
import { useRead } from '../personnel/api';
import type { JobDraft } from './AssignmentRecordForm';

export function ExchangeFields({draft,update,assignment,csrf}:{
  draft:JobDraft;update:(value:JobDraft)=>void;assignment:string;csrf:string;
}) {
  const [search,setSearch]=useState('');
  const key=draft.day && search.trim() ? JSON.stringify([assignment,draft.day,search.trim()]) : undefined;
  const options=useRead(key,async()=>{
    const header={'X-CSRF-Token':csrf};
    const query=unwrap(await api.POST('/api/v1/persons/queries',{params:{header},body:{keyword:search.trim()}}));
    return unwrap(await api.POST('/api/v1/assignments/{id}/exchange-options',{
      params:{path:{id:assignment},header},body:{businessDate:draft.day,ref:query.ref,page:1},
    }));
  });
  return <>
    <ObjectDetailField label="对调员工任职" inputId="exchange-peer" required>
      <Autocomplete id="exchange-peer" size="small" fullWidth options={options.data ?? []} value={draft.peer ?? null}
        filterOptions={v=>v} isOptionEqualToValue={(a,b)=>a.assignment===b.assignment}
        getOptionLabel={o=>o.name+' · '+o.recordNumber+' 号 · '+o.positionName}
        onInputChange={(_,value,reason)=>{if(reason==='input'||reason==='clear')setSearch(value)}}
        onChange={(_,peer)=>update({...draft,peer:peer ?? undefined})}
        noOptionsText={options.error ?? (search.trim()?'没有匹配的可对调任职':'输入员工姓名或编号')}
        renderInput={params=><TextField {...params} variant="standard" required error={!!options.error} helperText={options.error}/>}/>
    </ObjectDetailField>
    {draft.peer && <>
      <ObjectDetailField label="本任职调入">{draft.peer.positionName}</ObjectDetailField>
      <ObjectDetailField label={draft.peer.name+'调入'}>{draft.peer.targetName}</ObjectDetailField>
    </>}
    <ObjectTextField required label="对调依据" value={draft.notice} onChange={e=>update({...draft,notice:e.target.value})}/>
  </>;
}

export function ExchangeOriginal({link, initiallyOpen=false}:{link:Models['PositionExchangeLink']; initiallyOpen?:boolean}) {
  const [expanded,setExpanded]=useState(initiallyOpen);
  const ref=link.peerAction;
  const peer=useRead(expanded&&ref?JSON.stringify([ref.action,ref.revision]):undefined,
    async()=>unwrap(await api.GET('/api/v1/employment-actions/{id}',{params:{path:{id:ref!.action},query:{adoptedRef:ref!.revision}}})));
  return <>
    <ObjectDetailField label="调动方式">同日对调</ObjectDetailField>
    {ref && <Button aria-expanded={expanded} onClick={()=>setExpanded(!expanded)}>{expanded?'收起另一份调动原件':'查看另一份调动原件'}</Button>}
    {expanded && peer.data && <ObjectDetailField label="另一份调动原件">
      {[peer.data.action.businessDate,peer.data.transfer?.beforePositionName,peer.data.transfer?.after.segments?.[0]?.positionName,peer.data.action.notice].filter(Boolean).join(' · ')}
    </ObjectDetailField>}
    {expanded && peer.error && <div role="alert">{peer.error}</div>}
  </>;
}
