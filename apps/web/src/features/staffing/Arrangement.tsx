import {useEffect,useRef,useState} from 'react';
import {Alert,Autocomplete,Box,Button,Stack,TextField,Typography} from '@mui/material';
import AccountTreeIcon from '@mui/icons-material/AccountTree';
import {ApiError,api,unwrap,type Models} from '@chnhcm/api-client';
import {RecordControl,ObjectDetailSheet,ObjectDetailIdentity,ObjectDetailSection,ObjectDetailField,ObjectForm,ObjectTextField,UnsavedChangesDialog} from '@chnhcm/ui';
import {useDraftExit} from '../../shared/dated/useDraftExit';
import {useObjectSubmission} from './useSubmission';
import {arrangementChannel,resolveArrangement,nodeLabel,useStructure,type Structure} from './structure';

type Choice={id:string;name:string;code?:string};
const unassigned:Choice={id:'',name:'不设上级'};
export function Arrangement({session,id,initial,onClose,onSaved}:{session:Models['Principal'];id:string;initial:Structure;onClose:()=>void;onSaved:(day:string,changed:boolean)=>void}){
  const [text,setText]=useState(initial.businessAt),[date,setDate]=useState(initial.businessAt);
  const [expected,setExpected]=useState(initial.sourceVersion);
  const [choice,setChoice]=useState<string|null>(),[search,setSearch]=useState(''),[open,setOpen]=useState(false),[page,setPage]=useState(1);
  const [error,setError]=useState(''),[retry,setRetry]=useState(0);
  const result=useStructure({mode:'relation',businessAt:date,nodeId:id,pageSize:1},true,retry);
  const current=result?.data;
  const picked=useStructure({mode:'node',businessAt:date,nodeId:choice||undefined},!!choice,retry);
  useEffect(()=>{
    if(choice && picked?.status && [401,403,404].includes(picked.status)) setChoice(null);
  },[choice,picked?.status]);
  const candidates=useStructure({mode:'candidates',businessAt:date,nodeId:id,q:search,page},open && !!current?.canPublish && text===date,retry);
  const needsChoice=choice===null || (!!choice && !picked?.data?.selected);
  const selected=choice===undefined ? current?.parent??unassigned : !choice ? (choice===null?null:unassigned) : picked?.data?.selected??null;
  const [previousParent,setPreviousParent]=useState(initial.parent?.id??'');
  useEffect(()=>{if(current)setPreviousParent(current.parent?.id??'')},[current]);
  const saved=useRef(false);
  const finish=(r:Models['StaffingResult'])=>{
    if(r.outcome==='accepted'||r.outcome==='no_change'){saved.current=true;onSaved(date,r.outcome==='accepted');return}
    setError(r.failure?.message??'原提交未生效，请核对后重新保存。');setRetry(n=>n+1);
  };
  const {flow,pending,sessionBlocked}=useObjectSubmission(session,arrangementChannel,async(body:Models['HierarchyPublish'],submissionId)=>{
    const result=unwrap(await api.POST('/api/v1/position-hierarchies/reporting/publish',{
      params:{header:{'Idempotency-Key':submissionId,'X-CSRF-Token':session.csrf}},body
    }));
    if(result.kind!=='positionhierarchy')throw new Error('保存结果尚未确认');
    return result;
  },ref=>resolveArrangement(session,ref),finish,e=>e instanceof ApiError && e.status>=400 && e.status<500 && e.status!==408);
  const locked=flow.inFlight||pending||sessionBlocked;
  const dirty=text!==initial.businessAt || (choice!==undefined && choice!==previousParent);
  const exit=useDraftExit(dirty||flow.inFlight||flow.unknown,()=>saved.current,()=>flow.clearPayload());
  const conflict=!!current && expected!==current.sourceVersion;
  const ready=!!current?.canPublish && text===current.businessAt && !locked && !conflict && !needsChoice;
  const submit=async()=>{
    if(!ready)return;
    setError('');
    const result=await flow.submit({expectedVersion:expected,effectiveDate:date,childId:id,parentId:selected?.id||null});
    if(result.kind==='saved') finish(result.result);
    if(result.kind==='failed'){setError(result.error instanceof Error?result.error.message:'保存结果尚未确认');if(result.definitive)setRetry(n=>n+1)}
  };
  return <Box className="standard-page object-page" aria-busy={flow.inFlight}>
    <RecordControl identity={<Typography>调整上级</Typography>} navigation={null} actions={<>
      <Button variant="contained" disabled={!ready} onClick={submit}>{flow.inFlight?'正在保存…':'保存'}</Button>
      <Button disabled={flow.inFlight} onClick={()=>exit.request(onClose)}>取消</Button></>}/>
    <Box className="page-scroll" data-page-scroll tabIndex={0}>
      <ObjectDetailSheet identity={<ObjectDetailIdentity icon={<AccountTreeIcon/>} title="调整上级" summary={current?.selected ? nodeLabel(current.selected):undefined}/>}>
        <ObjectForm><ObjectDetailSection title="岗位关系">
          <ObjectTextField required label="生效日期" type="date" value={text} disabled={locked} onChange={e=>{
            const value=e.target.value;setText(value);if(value.length===10 && e.target.validity.valid){setDate(value);setPage(1);setSearch('')}
          }}/>
          {current && <ObjectDetailField label="原上级">{current.parent?nodeLabel(current.parent):'无上级'}</ObjectDetailField>}
          <ObjectDetailField label="新上级" inputId="arrangement-parent">
            <Autocomplete<Choice> id="arrangement-parent" disabled={locked||!current?.canPublish||text!==date} value={selected}
              options={[unassigned,...(candidates?.data?.nodes??[])]} filterOptions={items=>items} getOptionLabel={nodeLabel}
              open={open} inputValue={open?search:(selected?nodeLabel(selected):'')}
              getOptionKey={n=>n.id} isOptionEqualToValue={(a,b)=>a.id===b.id}
              noOptionsText="没有匹配岗位" onOpen={()=>setOpen(true)} onClose={()=>{setOpen(false);setSearch('');setPage(1)}}
              onInputChange={(_,value,reason)=>{if(reason==='input'||reason==='clear'){setSearch(value);setPage(1)}}}
              onChange={(_,n)=>{setChoice(n?.id??null);setPage(1);setSearch('')}}
              renderInput={params=><TextField {...params} placeholder="搜索岗位名称或编码"/>}/>
          </ObjectDetailField>
          {(page>1||candidates?.data?.more) && <Stack direction="row">
            <Button disabled={page===1||locked} onMouseDown={e=>e.preventDefault()} onClick={()=>setPage(p=>p-1)}>上一页候选</Button>
            <Button disabled={!candidates?.data?.more||locked} onMouseDown={e=>e.preventDefault()} onClick={()=>setPage(p=>p+1)}>下一页候选</Button>
          </Stack>}
          {current?.nextStart && <ObjectDetailField label="后续安排">{current.nextStart} 起已有安排</ObjectDetailField>}
          {conflict && <Alert severity="warning" action={<Button disabled={locked} onClick={()=>setExpected(current!.sourceVersion)}>使用当前版本</Button>}>上级安排已变化，请核对原上级和拟选岗位。</Alert>}
          {(choice===null || picked?.error) && <Alert severity="warning">请重新选择上级岗位，或明确选择“不设上级”。</Alert>}
          {(error||result?.error||candidates?.error) && <Alert severity="error" action={<Button onClick={()=>setRetry(n=>n+1)}>重新查询</Button>}>{error||result?.error||candidates?.error}</Alert>}
        </ObjectDetailSection></ObjectForm>
      </ObjectDetailSheet>
    </Box><UnsavedChangesDialog {...exit.dialog} pending={flow.unknown||flow.inFlight}/>
  </Box>
}
