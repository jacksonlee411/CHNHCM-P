import { SubmissionError } from '../../shared/dated/submission';
import { selectedReason } from './dated/reasons';
import {useEffect,useRef,useState} from 'react';
import {Alert,Box,Button,Dialog,DialogActions,DialogContent,DialogTitle,MenuItem,Typography} from '@mui/material';
import {useDraftExit} from '../../shared/dated/useDraftExit';
import {unwrap,ApiError,sessionGuard,type Models} from '@chnhcm/api-client';
import {ObjectTextField,UnsavedChangesDialog,businessDate,businessPeriod} from '@chnhcm/ui';
import {useDefinition,type Definition,type DetailView,type Result,type PairCommand,type MaintainPairInput} from './dated/presentation';
import {trimDetailFields} from './businessUnitProjection';
import {useSubmission} from './dated/useSubmission';

type Props={session:Models['Principal'];detail:DetailView;mode:'insert'|'delete'|'maintain';onClose:()=>void;onReloaded:(d:DetailView)=>void;onDenied:()=>void;onSaved:(r:Result)=>void};
// Calendar input assistance only; Go owns all availability and final constraints.
function shift(value:string,n:number){const d=new Date(value+'T12:00:00Z');if(!Number.isFinite(d.valueOf()))return '';d.setUTCDate(d.getUTCDate()+n);const s=d.toISOString().slice(0,10);return /^\d{4}-\d{2}-\d{2}$/.test(s)?s:''}
function PairDeletionChanges({changes, fields, showDates}: {
  changes: NonNullable<DetailView['pairDeletion']>['changes'];
  fields: NonNullable<Definition['summaryFields']>;
  showDates: boolean;
}) {
  const periods = changes.map(change => {
    const rows = fields.flatMap(field => {
      const before = change.before.values?.[field.key];
      const after = change.after.values?.[field.key];
      const beforeText = field.render?.(change.before) ?? before ?? '—';
      const afterText = field.render?.(change.after) ?? after ?? '—';
      if (before === after && beforeText === afterText) return [];
      return [{key: field.key, label: field.label, beforeText, afterText}];
    });
    return {change, rows};
  }).filter(period => period.rows.length > 0);
  if (!periods.length) return null;

  return <Box sx={{mt:2}} aria-label="删除后的资料变化">
    <Typography>删除后的资料变化</Typography>
    {periods.map(({change, rows}, i) => <Box key={i} sx={{mt:1}}>
      {showDates && change.start && <Typography variant="body2">
        {businessPeriod(change.start, change.endExclusive, change.endInclusive)}
      </Typography>}
      {rows.map(row => <Typography key={row.key}>
        {row.label}：{row.beforeText} → {row.afterText}
      </Typography>)}
    </Box>)}
  </Box>;
}

export function LifecyclePairDialog(props:Props){
 const businessUnit=useDefinition(),businessUnitApi=businessUnit.api;
 const {session,detail,mode}=props;const callbacks=useRef(props);callbacks.current=props;
 const [basis,setBasis]=useState(detail),initial=useRef(detail.pairInsertion),pair=useRef(detail.pairDeletion),selected=useRef(detail.selectedRecord?.recordId);
 const originalStop=mode==='maintain'?pair.current?.deactivation.effectiveDate:initial.current?.defaultStopDate,originalResume=mode==='maintain'?pair.current?.reactivation.effectiveDate:initial.current?.defaultResumeDate;
 const [stop,setStop]=useState(originalStop??''),[resume,setResume]=useState(originalResume??'');
 const [stopReason,setStopReason]=useState(mode==='maintain'?pair.current?.deactivation.operationReason??'':''),[resumeReason,setResumeReason]=useState(mode==='maintain'?pair.current?.reactivation.operationReason??'':''),[reasons,setReasons]=useState<Record<string,Models['Reason'][]>>({});
 const [reasonRetry,retryReasons]=useState(0),[reasonError,setReasonError]=useState('');const resumeEdited=useRef(false);
 const [reloading,setReloading]=useState(false),[conflict,setConflict]=useState(false),[error,setError]=useState(''),[field,setField]=useState('');
 const mounted=useRef(false),epoch=useRef(0);
 const {flow,pending,sessionBlocked}=useSubmission({channel:businessUnit.submissions,owner:session,resolve: reference => businessUnit.resolve(session, reference),
  send:(command:PairCommand | {action:'maintain';body:MaintainPairInput},id)=>command.action==='maintain'?businessUnitApi.maintainPair!(detail.id,command.body,{'X-CSRF-Token':session.csrf,'Idempotency-Key':id}):businessUnitApi.pair!(detail.id,command,{'X-CSRF-Token':session.csrf,'Idempotency-Key':id}),
  onResolved:r=>{if(r.outcome==='accepted'||r.outcome==='no_change')callbacks.current.onSaved(r);else setError('')},
  onDenied:()=>{setStop('');setResume('');flow.clearPayload();callbacks.current.onDenied()},
 });
 const {inFlight:busy,unknown}=flow;
 const dirty=mode!=='delete'&&(stop!==(originalStop??'')||resume!==(originalResume??'')||stopReason!==selectedReason('',reasons.DEACTIVATE??[])||resumeReason!==selectedReason('',reasons.REACTIVATE??[]));
 const leave=useDraftExit(dirty,()=>flow.saved,()=>{flow.clearPayload();setStop('');setResume('');setStopReason('');setResumeReason('')});
 const title=mode==='insert'?'插入停用与重新启用':mode==='maintain'?'维护停用与重新启用':'删除停用与重新启用';
 const available=mode==='insert'?!!basis.pairInsertion?.available&&basis.pairInsertion.anchorRecordId===initial.current?.anchorRecordId:!!basis.pairDeletion&&basis.pairDeletion.deactivation.recordId===pair.current?.deactivation.recordId&&basis.pairDeletion.reactivation.recordId===pair.current?.reactivation.recordId;
 useEffect(()=>{mounted.current=true;return()=>{mounted.current=false;epoch.current++}},[]);
 useEffect(()=>{
  const guard=sessionGuard();let active=true;if(mode==='delete')return;
  setReasonError('');void Promise.all(['DEACTIVATE','REACTIVATE'].map(async kind=>[kind,unwrap(await businessUnitApi.reasons(kind as 'DEACTIVATE'|'REACTIVATE'))] as const)).then(rows=>{if(active&&guard()){const values=Object.fromEntries(rows);setReasons(values);setStopReason(previous=>selectedReason(previous,values.DEACTIVATE));setResumeReason(previous=>selectedReason(previous,values.REACTIVATE));}}).catch(e=>{if(active&&guard())setReasonError(e instanceof ApiError?e.message:'原因读取失败')});return()=>{active=false};
 },[reasonRetry,mode,session.instanceId,session.subjectId]);
 useEffect(()=>{if(field&&!busy)document.getElementById('pair-'+field)?.focus()},[field,busy]);
 async function reload(){
  const own=++epoch.current,same=sessionGuard();setReloading(true);setError('');
  try{const v=unwrap(await businessUnitApi.detail(detail.id,{selectedRecordId:mode==='insert'?initial.current?.anchorRecordId:selected.current}));if(!mounted.current||!same()||own!==epoch.current)return;
   if(!v.capabilities.readFields.includes('effectiveDate')){setStop('');setResume('')}
   if(mode==='delete'&&(v.pairDeletion?.deactivation.recordId!==pair.current?.deactivation.recordId||v.pairDeletion?.reactivation.recordId!==pair.current?.reactivation.recordId))delete v.pairDeletion;
   setBasis(v);callbacks.current.onReloaded(v);setConflict(false);setError((mode==='insert'?v.pairInsertion?.available:!!v.pairDeletion)?'':'当前记录无法完成此操作，可取消编辑');
  }catch(e){if(!mounted.current||!same()||own!==epoch.current)return;setError(e instanceof ApiError?e.message:'读取失败');if(e instanceof ApiError&&[403,404].includes(e.status))callbacks.current.onDenied()}
  finally{if(mounted.current&&same()&&own===epoch.current)setReloading(false)}
 }
 async function save(){
  if(sessionBlocked||busy||reloading||(!flow.attempt&&(!available||pending||conflict)))return;
  const pairBody={expectedSourceVersion:basis.headSourceVersion??'',deactivationRecordId:basis.pairDeletion?.deactivation.recordId??'',deactivationRevision:basis.pairDeletion?.deactivation.revision??'',reactivationRecordId:basis.pairDeletion?.reactivation.recordId??'',reactivationRevision:basis.pairDeletion?.reactivation.revision??''};
  const command:PairCommand | {action:'maintain';body:MaintainPairInput}=mode==='maintain'?{action:mode,body:{...pairBody,deactivationDate:stop,reactivationDate:resume,deactivationReason:stopReason,reactivationReason:resumeReason}}:mode==='insert'?{action:mode,body:{expectedSourceVersion:basis.headSourceVersion??'',anchorRecordId:initial.current?.anchorRecordId??'',deactivationDate:stop,deactivationReason:stopReason,reactivationDate:resume,reactivationReason:resumeReason}}:{action:mode,body:{expectedSourceVersion:basis.headSourceVersion??'',deactivationRecordId:basis.pairDeletion?.deactivation.recordId??'',deactivationRevision:basis.pairDeletion?.deactivation.revision??'',reactivationRecordId:basis.pairDeletion?.reactivation.recordId??'',reactivationRevision:basis.pairDeletion?.reactivation.revision??''}};
  setError('');setField('');
  const outcome=await flow.submit(command);
  if(outcome.kind==='saved')callbacks.current.onSaved(outcome.result);
  else if(outcome.kind==='failed'){
   const e=outcome.error;setError(flow.unknown?'':e instanceof ApiError||e instanceof SubmissionError?e.message:'暂时无法开始保存，请重试');setField(e instanceof ApiError?e.field??'':'');
   if(outcome.definitive&&e instanceof ApiError){
    if(e.status===409)setConflict(true);
    if(e.status===403&&e.code!=='AUTH_REQUEST_VALIDATION_FAILED'){const safe=trimDetailFields(basis,e.readFields??[],businessUnit.statusReadField);setBasis(safe);callbacks.current.onReloaded(safe);if(!safe.capabilities.readFields.includes('effectiveDate')){setStop('');setResume('')}void reload()}
    if(e.status===404)callbacks.current.onDenied();
   }
  }
 }
 const change=()=>{setField('');setError('')};const disabled=busy||reloading||unknown;
 const earliestResume=shift(basis.pairInsertion?.minDate??'',1),followingStop=shift(stop,1);
 const resumeMin=followingStop>earliestResume?followingStop:earliestResume;
 const close=()=>{if(!busy)leave.request(()=>callbacks.current.onClose())};
 return <><Dialog open fullWidth maxWidth="sm" aria-labelledby="pair-dialog-title" onClose={close}>
  <DialogTitle id="pair-dialog-title">{title}</DialogTitle>
  <DialogContent dividers>
   {mode!=='delete'?<Box sx={{display:'grid',gridTemplateColumns:{xs:'1fr',sm:'1fr 1fr'},gap:2}}>
    {(mode==='maintain'||!!basis.pairInsertion)&&<ObjectTextField id="pair-deactivationDate" label="停用日期" type="date" value={stop} disabled={disabled} error={field==='deactivationDate'} onChange={e=>{change();setStop(e.target.value);if(!resumeEdited.current)setResume(shift(e.target.value,1))}} inputProps={{min:basis.pairInsertion?.minDate,max:resumeEdited.current&&resume&&shift(resume,-1)<(basis.pairInsertion?.maxStopDate??'9999-12-31')?shift(resume,-1):basis.pairInsertion?.maxStopDate}}/>}
    <ObjectTextField id="pair-deactivationReason" label="停用原因" select value={stopReason} disabled={disabled} error={field==='deactivationReason'} onChange={e=>{change();setStopReason(e.target.value)}}><MenuItem value="">请选择</MenuItem>{reasons.DEACTIVATE?.map(r=><MenuItem key={r.code} value={r.code}>{r.name}</MenuItem>)}</ObjectTextField>
    {(mode==='maintain'||!!basis.pairInsertion)&&<ObjectTextField id="pair-reactivationDate" label="重新启用日期" type="date" value={resume} disabled={disabled} error={field==='reactivationDate'} onChange={e=>{change();resumeEdited.current=true;setResume(e.target.value)}} inputProps={{min:resumeMin,max:basis.pairInsertion?.maxDate}}/>}
    <ObjectTextField id="pair-reactivationReason" label="重新启用原因" select value={resumeReason} disabled={disabled} error={field==='reactivationReason'} onChange={e=>{change();setResumeReason(e.target.value)}}><MenuItem value="">请选择</MenuItem>{reasons.REACTIVATE?.map(r=><MenuItem key={r.code} value={r.code}>{r.name}</MenuItem>)}</ObjectTextField>
   </Box>:basis.pairDeletion&&<Box>
    {[basis.pairDeletion.deactivation,basis.pairDeletion.reactivation].map(r=><Typography key={r.recordId}>{r.effectiveDate?businessDate(r.effectiveDate):'日期不可读取'} · {r.typeLabel||'类型不可读取'}</Typography>)}
    <PairDeletionChanges changes={basis.pairDeletion.changes} fields={businessUnit.summaryFields ?? businessUnit.fields}
      showDates={basis.capabilities.readFields.includes('effectiveDate')} />
   </Box>}
   {reasonError&&<Alert severity="error" action={<Button onClick={()=>retryReasons(n=>n+1)}>重试加载</Button>}>{reasonError}</Alert>}
   {error&&<Alert severity="error" sx={{mt:2}}>{error}{conflict&&<Button disabled={disabled} onClick={()=>void reload()}>重新核对当前资料</Button>}</Alert>}
   {unknown&&flow.attempt&&<Button disabled={sessionBlocked||busy} onClick={()=>void save()}>重试保存</Button>}
  </DialogContent>
  <DialogActions sx={{flexWrap:'wrap'}}><Button disabled={sessionBlocked||disabled||!available||pending||conflict} variant="contained" color={mode==='delete'?'error':'primary'} onClick={()=>void save()}>{mode!=='delete'?'保存':'删除两条记录'}</Button><Button disabled={busy} onClick={close}>取消</Button></DialogActions>
 </Dialog><UnsavedChangesDialog pending={unknown} {...leave.dialog}/></>;
}
