import { useEffect, useId, useLayoutEffect, useRef, useState } from 'react';
import { Link, useLocation, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { createPortal } from 'react-dom';
import { Alert, Box, Button, Menu, MenuItem, Stack, Typography } from '@mui/material';
import { api, ApiError, unwrap, type Models } from '@chnhcm/api-client';
import {
  HistoryNotice,
  ObjectDetailField,
  ObjectDetailSection,
  UnsavedChangesDialog,
  recordedTime,
} from '@chnhcm/ui';
import { RelationshipContracts } from './RelationshipContracts';
import { adoptedOfferURL } from './jobLinks';
import { DutyOriginal, DutyUsage, lastDutyDay } from './ActualDutyFields';
import {ExchangeOriginal} from './ExchangeFields';
import {managerPurposes} from './LifecycleFields';
import { useDraftExit } from '../../shared/dated/useDraftExit';
import { useRead, message } from '../personnel/api';
import { useRuleSubmission } from '../readiness/useRuleSubmission';

import {
  AssignmentRecordForm,
  actionLabel,
  availableRecordTypes,
  draftChanged,
  relationAction,
  type JobDraft,
} from './AssignmentRecordForm';

type Action = Models['EmploymentAction'];
export type JobPageMode = 'browse' | 'editing' | 'history';
type Command =
  | { kind:'duty'; input:Models['DutyCommand'] }
  | { kind:'exchange'; input:Models['PositionExchangeCommand'] }
  | { kind:'manager'; input:Models['ActualManagerCommand'] }
  | { kind:'exit'; leave:boolean; input:Models['AssignmentExitCommand'] }
  | { kind: 'grade'; form:'Promotion'|'GradeRecord'|'PromotionTransfer'; action?:string; input:Models['AssignmentPromotionCommand'] }
  | { kind: 'withdraw'; action:string; input:Models['HireCancel'] }
  | { kind: 'probation'; confirm: boolean; input: Models['ProbationCommand'] }
  | { kind: 'location'; action?: string; input: Models['EmploymentLocationChange'] }
  | { kind: 'transfer' | 'redate' | 'delete'; action?: string; input: Models['EmploymentRecordMaintenance'] };

export function JobInformation({
  enabled = true,
  current,
  assignment,
  day,
  session,
  onSave,
  actionSlot,
  historySlot,
  onModeChange,
  selectedAction,
}: {
  assignment: string;
  day: string;
  session: Models['Principal'];
  onSave: () => void;
  actionSlot: HTMLElement | null;
  historySlot?: HTMLElement | null;
  onModeChange?: (mode: JobPageMode) => void;
  enabled?: boolean;
  current?: ReturnType<typeof useRead<Models['AssignmentView']>>;
  selectedAction?: string;
}) {
  const [params, setParams] = useSearchParams();
  const navigate = useNavigate();
  const route = useLocation();
  const { id: personId } = useParams();
  const source = params.get('sourceVersion') || '';
  const revision = params.get('adoptedRef') || '';
  const chosenId = selectedAction || params.get('action');
  function select(values: Record<string, string | undefined>, replace = false) {
    const next = new URLSearchParams(params);
    for (const [key, value] of Object.entries(values)) {
      if (value) next.set(key, value);
      else next.delete(key);
    }
    setParams(next, {replace});
  }
  const [contractDraft,setContractDraft]=useState<Models['EmploymentContractCommand']>();
  const contractInitial=useRef<Models['EmploymentContractCommand']|undefined>(undefined);
  const [draft, setDraft] = useState<JobDraft>();
  const initial = useRef<JobDraft | undefined>(undefined);
  const [menu, setMenu] = useState<HTMLElement | null>(null);
  const [removal, setRemoval] = useState<{action: Action; source: string; exchange?: Models['PositionExchangeLink']}>();
  const [conflict, setConflict] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const errorRef = useRef<HTMLDivElement>(null);
  const controlId = useId();
  const focusTarget = useRef('');
  const restoredLocation = useRef('');
  const saved = useRef(false);
  const header = { 'X-CSRF-Token': session.csrf };
  const lookup = useRead(enabled && !current ? JSON.stringify([assignment, day, source]) : undefined, async () =>
    unwrap(
      await api.GET('/api/v1/assignments/{id}', {
        params: {
          path: { id: assignment },
          query: { businessAt: day || undefined, sourceVersion: source || undefined },
        },
      }),
    ),
  );
  const state = current ?? lookup;
  const data = state.data;
  const opened = data?.actions.find(a => a.id === chosenId && a.canHistory);
  const historical = !!source || !!revision && !draft && (!opened || revision !== opened.revision);
  useEffect(() => {
    if (!data || !draft) return;
    const a = data.actions.find((v) => v.id === draft.action?.id);
    const allowed =
      draft.mode==='withdraw' ? a?.canWithdrawPromotion : draft.mode === 'redate'
        ? a?.canMaintain
        : draft.action
          ? draft.kind === 'ActualDuty' ? a?.canCorrectDuty : draft.kind === 'Promotion' ? a?.canCorrectPromotion : draft.kind === 'Transfer'
            ? a?.canCorrectTransfer
            : a?.canCorrectLocation
          : availableRecordTypes(data).includes(draft.kind);
    if (!allowed) {
      setDraft(undefined);
      setError('已重新核对权限，当前不可办理。');
    }
  }, [data, draft]);
  const displayedRevision = !draft && revision || opened?.revision;
  const original = useRead(
    opened ? JSON.stringify([opened.id, displayedRevision]) : undefined,
    async () =>
      unwrap(
        await api.GET('/api/v1/employment-actions/{id}', {
          params: { path: { id: opened!.id }, query: { adoptedRef: displayedRevision } },
        }),
      ),
  );
  const originalSource = historical && revision ? original.data?.action : undefined;
  const exact = useRead(originalSource && originalSource.sourceVersion!==data?.sourceVersion ?
    JSON.stringify([originalSource.assignment,originalSource.businessDate,originalSource.sourceVersion]) : undefined, async () =>
      unwrap(await api.GET('/api/v1/assignments/{id}', {params:{path:{id:originalSource!.assignment},
        query:{businessAt:originalSource!.businessDate,sourceVersion:originalSource!.sourceVersion}}})));
  const facts = historical && revision ? (originalSource && (originalSource.sourceVersion===data?.sourceVersion ? data : exact.data)) : data;
  const unavailable = !!chosenId && !!data && (!opened || original.status === 403 || original.status === 404);
  const mode: JobPageMode = historical || unavailable ? 'history' : draft || removal || contractDraft ? 'editing' : 'browse';
  useLayoutEffect(() => { onModeChange?.(mode); }, [mode, onModeChange]);
  useLayoutEffect(() => {
    if (!route.state?.focusJobOriginal || restoredLocation.current === route.key || historical || !data || opened && !original.data && !original.error) return;
    if (!selectedAction && unavailable) {
      const next = new URLSearchParams(params);
      next.delete('action');
      setParams(next, {replace:true, state:route.state});
      return;
    }
    const target = document.getElementById(controlId + '-record') ?? document.getElementById(controlId + '-original') ??
      document.getElementById(controlId + '-add') ?? document.getElementById(controlId + '-insert');
    if (target) {
      target.focus();
      restoredLocation.current = route.key;
    }
  }, [route, historical, data, opened, original.data, original.error, controlId, selectedAction, unavailable, params, setParams]);
  useEffect(() => {
    if (removal && data && (!opened?.canDelete || original.data?.exchange && !original.data.exchange.canDelete)) {
      setRemoval(undefined);
      setError('已重新核对权限，当前不可处置这笔办理。');
    }
  }, [removal, data, opened, original.data]);
  const review = useRead(conflict && draft?.day ? JSON.stringify([assignment, draft.day]) : undefined, () =>
    api.GET('/api/v1/assignments/{id}', {params:{path:{id:assignment},query:{businessAt:draft!.day}}}).then(unwrap), 'fresh');
  const reviewTarget = draft?.action || draft?.successor;
  const reviewedAction = reviewTarget && review.data?.actions.find(a => a.id === reviewTarget.id && a.canHistory);
  const aligned = !reviewTarget || !!reviewedAction && original.data?.action.revision === reviewedAction.revision;
  const close = () => {
    setContractDraft(undefined);
    setDraft(undefined);
    setRemoval(undefined);
    setConflict(false);
    setError('');
    requestAnimationFrame(() => {
      const target = document.getElementById(focusTarget.current) ?? document.getElementById(controlId + '-record');
      target?.focus();
    });
  };
  const dirty = !!draft && !!initial.current && draftChanged(draft, initial.current) || !!contractDraft && JSON.stringify(contractDraft)!==JSON.stringify(contractInitial.current);
  const exit = useDraftExit(dirty, () => saved.current, close);
  const submit = useRuleSubmission<Command, Models['HireReceipt']>('assignment-location', session, {
    send: async (cmd, key) => {
      const headers = { ...header, 'Idempotency-Key': key };
      if (cmd.kind==='duty') return unwrap(await api.POST('/api/v1/assignments/{id}/actual-duty',{params:{path:{id:assignment},header:headers},body:cmd.input}));
      if (cmd.kind==='exchange') return unwrap(await api.POST('/api/v1/assignments/{id}/position-exchanges',{params:{path:{id:assignment},header:headers},body:cmd.input}));
      if (cmd.kind==='manager') return unwrap(await api.POST('/api/v1/assignments/{id}/managers',{params:{path:{id:assignment},header:headers},body:cmd.input}));
      if (cmd.kind==='exit') return unwrap(await api.POST(cmd.leave ? '/api/v1/assignments/{id}/leave' : '/api/v1/assignments/{id}/end',{params:{path:{id:assignment},header:headers},body:cmd.input}));
      if (cmd.kind==='withdraw') return unwrap(await api.POST('/api/v1/employment-actions/{id}/cancel',{
        params:{path:{id:cmd.action},header:headers},body:cmd.input}));
      if (cmd.kind==='grade') {
        const params={path:{id:assignment},header:headers};
        if(cmd.action) return unwrap(await api.POST('/api/v1/assignments/{id}/promotions/{action}/correct',
          {params:{...params,path:{id:assignment,action:cmd.action}},body:cmd.input}));
        const path=cmd.form==='GradeRecord'?'/api/v1/assignments/{id}/grade-records':
          cmd.form==='PromotionTransfer'?'/api/v1/assignments/{id}/promotion-transfers':'/api/v1/assignments/{id}/promotions';
        return unwrap(await api.POST(path,{params,body:cmd.input}));
      }
      if (cmd.kind === 'probation') {
        const params = { path: { id: assignment }, header: headers };
        return cmd.confirm
          ? unwrap(await api.POST('/api/v1/assignments/{id}/confirm-employment', { params, body: cmd.input }))
          : unwrap(await api.POST('/api/v1/assignments/{id}/trial-start', { params, body: cmd.input }));
      }
      if (cmd.kind !== 'location') {
        const input = cmd.input;
        if (cmd.kind === 'transfer' && !cmd.action)
          return unwrap(
            await api.POST('/api/v1/assignments/{id}/transfers', {
              params: { path: { id: assignment }, header: headers },
              body: input,
            }),
          );
        const params = { path: { id: assignment, action: cmd.action! }, header: headers };
        if (cmd.kind === 'transfer')
          return unwrap(
            await api.POST('/api/v1/assignments/{id}/transfers/{action}/correct', { params, body: input }),
          );
        if (cmd.kind === 'redate')
          return unwrap(
            await api.POST('/api/v1/assignments/{id}/actions/{action}/redate', { params, body: input }),
          );
        return unwrap(
          await api.POST('/api/v1/assignments/{id}/actions/{action}/delete', { params, body: input }),
        );
      }
      return cmd.action
        ? unwrap(
            await api.POST('/api/v1/assignments/{id}/location-changes/{action}/correct', {
              params: { path: { id: assignment, action: cmd.action }, header: headers },
              body: cmd.input,
            }),
          )
        : unwrap(
            await api.POST('/api/v1/assignments/{id}/location-changes', {
              params: { path: { id: assignment }, header: headers },
              body: cmd.input,
            }),
          );
    },
    resolve: async (sub) =>
      unwrap(
        await api.POST('/api/v1/employment-actions/submissions/{sub}/resolve', { params: { path: { sub }, header } }),
      ),
    finished: (r, recovery) => {
      if (r.outcome !== 'accepted') {
        setError(r.outcome === 'not_applied' ? '原号已确认未执行。' : '原提交已拒绝。');
        return;
      }
      if (recovery !== 'other') {
        saved.current = true;
        close();
        select({
          assignmentDay: selectedAction ? params.get('assignmentDay') || undefined : draft?.day || day,
          action: selectedAction ? undefined : r.action,
          adoptedRef: undefined,
          sourceVersion: undefined,
        }, true);
        setNotice(recovery ? '已查回原办理。' : '职务信息已保存。');
      }
      state.invalidate();
      if (state !== current) onSave();
    },
    failed: (e) => {
      setError(message(e));
      if (e instanceof ApiError && e.status === 409 &&
          (e.code === 'ASSIGNMENT_CHANGED' || e.code === 'CONTRACT_SOURCE' && draft?.kind === 'Leave')) {
        setConflict(!!draft);
        setRemoval(undefined);
        original.invalidate();
        state.invalidate();
      }
      if (e instanceof ApiError && [401, 403, 404].includes(e.status)) {
        setDraft(undefined);
        setRemoval(undefined);
        setConflict(false);
        original.invalidate();
        state.invalidate();
      }
    },
  });
  useEffect(() => {
    submit.start();
  }, [submit.start]);
  useEffect(() => {
    if (error) errorRef.current?.focus();
  }, [error]);
  useEffect(() => {
    if ([state.status, original.status, review.status].some(s => s === 401 || s === 403 || s === 404)) {
      setDraft(undefined);
      setRemoval(undefined);
      setConflict(false);
    }
  }, [state.status, original.status, review.status]);
  function makeDraft(mode: JobDraft['mode'], a?: Action, successor?: Action, data = state.data): JobDraft | undefined {
    if (!data) return;
    const evidence = a || successor ? original.data : undefined;
    const after = evidence?.transfer?.after;
    const state = data.state;
    const department = after?.department ?? state?.departmentId;
    const job = after?.job ?? state?.jobId;
    const adoptedPosition = after?.segments?.find(segment => segment.position);
    const position = after ? after.position : state?.position;
    const next: JobDraft = {
      mode,
      originalDuty:after?.duty,
      paper:after?.duty && data.canVerifyDuty ? {dispatch:after.duty.dispatch,issuer:after.duty.issuer,number:after.duty.number,from:after.duty.from,lastDay:lastDutyDay(after.duty.until)||undefined,observedThrough:after.duty.observedThrough,previous:{id:after.duty.id,version:after.duty.version}}:undefined,
      handover:after?.succession?true:undefined,
      incumbent:after?.succession?{...after.succession,departmentName:adoptedPosition?.departmentName??'',jobName:adoptedPosition?.jobName??''}:undefined,
      exchange:evidence?.exchange,
      kind: after?.duty && a ? 'ActualDuty' : evidence?.exchange && a ? 'PositionExchange' : successor ? (successor.kind === 'Promotion' ? 'ConfirmEmployment' : 'Promotion') : a ? (a.kind==='Promotion'?'Promotion':a.kind === 'Transfer' ? 'Transfer' : 'DataChange') : availableRecordTypes(data)[0],
      successor,
      grade:evidence?.promotion?.after,originalGrade:evidence?.promotion?.before ?? state?.grade,
      reason:evidence?.promotion?.reason,
      predecessor:data.actions.find(item=>item.id===evidence?.previous?.action && (successor?.kind !== 'Promotion' || item.kind === 'Transfer')),
      action: a,
      day: a?.businessDate ?? successor?.businessDate ?? day,
      source: data.sourceVersion,
      value: a ? evidence?.locationChange?.after : state?.location,
      department: department
        ? { id: department, name: after?.segments?.[0]?.departmentName ?? state?.department ?? '' }
        : undefined,
      position: position ? {id: position, name: adoptedPosition?.positionName ?? state?.positionName ?? position} : undefined,
      job: job ? { id: job, name: after?.segments?.[0]?.jobName ?? state?.job ?? '' } : undefined,
      notice: successor ? '' : evidence?.action.notice ?? '',
      requires: evidence?.requires,
    };
    if(successor && evidence?.transfer){
      const prior=evidence.transfer;
      next.department=prior.beforeDepartment ? {id:prior.beforeDepartment,name:prior.beforeDepartmentName ?? ''}:undefined;
      next.job=prior.beforeJob ? {id:prior.beforeJob,name:prior.beforeJobName ?? ''}:undefined;
      next.originalGrade=prior.beforeGrade;next.notice='';
    }
    return next;
  }
  function begin(target: string, mode: JobDraft['mode'], a?: Action, successor?: Action) {
    const next = makeDraft(mode, a, successor);
    if (!next || historical) return;
    focusTarget.current = controlId + '-' + target;
    setMenu(null);
    saved.current = false;
    setError('');
    setNotice('');
    setConflict(false);
    submit.start();
    initial.current = next;
    setDraft(next);
  }
  function acceptCurrent() {
    if (!draft || !initial.current || !review.data || !aligned) return;
    const latestView = review.data;
    const target = draft.action && latestView.actions.find(a => a.id === draft.action?.id);
    const successor = draft.successor && latestView.actions.find(a => a.id === draft.successor?.id);
    if (draft.action && !target || draft.successor && !successor) return;
    const latestMode = !draft.action && !draft.successor ? latestView.canInsertRecord ? 'insert' : 'add' : draft.mode;
    const latest = makeDraft(latestMode, target, successor, latestView);
    if (!latest) return;
    const edited = Object.entries(draft).filter(([key, value]) =>
      !['mode', 'source', 'action', 'successor', 'exchange', 'originalDuty', 'originalGrade'].includes(key) &&
      (key !== 'paper' || latestView.canVerifyDuty) && (!['handover', 'incumbent'].includes(key) || latestView.canHandover) &&
      JSON.stringify(value) !== JSON.stringify(initial.current?.[key as keyof JobDraft]));
    initial.current = latest;
    setDraft({...latest, ...Object.fromEntries(edited)});
    setConflict(false);
    setError('');
  }
  function save(record: JobDraft) {
    if(record.kind==='ActualDuty'){
      const p=record.paper;
      const paper=p?.dispatch && p.from && p.observedThrough && p.issuer && p.number ? {dispatch:p.dispatch,issuer:p.issuer,number:p.number,from:p.from,observedThrough:p.observedThrough,lastDay:p.lastDay||undefined,previous:p.previous}:undefined;
      const o=record.observation;
      void submit.submit({kind:'duty',input:{expectedSourceVersion:record.source,action:record.action?{action:record.action.id,revision:record.action.revision}:undefined,observation:o?{id:o.id,version:o.version}:undefined,paper}});
      return;
    }
    if(record.kind==='PositionExchange'){
      const peerAssignment=record.exchange?.peerAssignment ?? record.peer?.assignment;
      const peerSourceVersion=record.exchange?.peerSourceVersion ?? record.peer?.sourceVersion;
      if(!peerAssignment || !peerSourceVersion)return;
      void submit.submit({kind:'exchange',input:{operation:record.action?'Redate':'Add',expectedSourceVersion:record.source,peerAssignment,peerSourceVersion,
        businessDate:record.day,notice:record.action?undefined:record.notice,action:record.action?{action:record.action.id,revision:record.action.revision}:undefined}});
      return;
    }
    if(record.kind==='SetManager'){
      void submit.submit({kind:'manager',input:{businessDate:record.day,expectedSourceVersion:record.source,purpose:record.purpose ?? 'direct',manager:record.endReporting ? undefined : record.manager?.id,lastDay:record.endReporting ? undefined : record.managerLastDay || undefined}});
      return;
    }
    if(record.kind==='Leave'||record.kind==='EndAssignment'){
      void submit.submit({kind:'exit',leave:record.kind==='Leave',input:{date:record.day,meaning:record.meaning ?? 'last-active',expectedSourceVersion:record.source,notice:record.notice,contracts:record.contracts}});
      return;
    }
    const action = record.action?.id;
    const input = {
      expectedSourceVersion: record.source,
      expectedRevision: record.action?.revision,
      businessDate: record.day,
    };
    if(record.mode==='withdraw' && record.action){
      void submit.submit({kind:'withdraw',action:record.action.id,input:{expectedSourceVersion:record.source,
        revision:record.action.revision,reason:record.withdrawal?.trim(),basis:record.basis?{action:record.basis.id,revision:record.basis.revision}:undefined}});
      return;
    }
    const reference=(a:Pick<Action, 'id' | 'revision'>|undefined)=>a?{action:a.id,revision:a.revision}:undefined;
    if(record.kind==='GradeRecord'||record.kind==='Promotion'||record.kind==='PromotionTransfer'){
      const changing=record.kind==='PromotionTransfer'||record.kind==='Promotion' &&
        (record.position?.id!==initial.current?.position?.id || record.job?.id!==initial.current?.job?.id || record.department?.id!==initial.current?.department?.id);
      void submit.submit({kind:'grade',form:record.kind,action,input:{...input,grade:record.grade?.id,
        reason:record.reason,notice:record.notice,transferNotice:record.transferNotice,
        department:changing?record.department?.id:undefined,job:changing?record.job?.id:undefined,position:changing?record.position?.id:undefined,
        previous:reference(record.predecessor),precedes:reference(record.successor)}});
      return;
    }
    if (record.mode === 'redate') {
      void submit.submit({ kind: 'redate', action, input });
      return;
    }
    if (record.kind === 'ConfirmEmployment' || record.kind === 'ProbationStart') {
      void submit.submit({
        kind: 'probation',
        confirm: record.kind === 'ConfirmEmployment',
        input: {
          businessDate: record.day,
          expectedSourceVersion: record.source,
          notice: record.notice,
          department: record.occurrence?.id,
          afterAction: record.predecessor?.id,
          afterRevision: record.predecessor?.revision,
          precedes: reference(record.successor),
        },
      });
      return;
    }
    const changes = { ...input, insert: record.mode === 'insert' ? true : record.mode === 'add' ? false : undefined, requires: record.requires };
    if (record.kind === 'DataChange' && record.value)
      void submit.submit({ kind: 'location', action, input: { ...changes, location: record.value.id } });
    if (record.kind === 'Transfer')
      void submit.submit({
        kind: 'transfer',
        action,
        input: { ...changes, incumbent:record.handover&&record.incumbent?{assignment:record.incumbent.assignment,sourceVersion:record.incumbent.sourceVersion}:undefined, position:record.position?.id, department: record.department?.id, job: record.job?.id, notice: record.notice,previous:reference(record.predecessor) },
      });
  }
  function confirmRemoval() {
    if (!removal || !opened?.canDelete || historical || !data || submit.busy) return;
    const a = removal.action;
    setError('');
    setNotice('');
    const link=removal.exchange;
    if(link){
      if(link.canDelete && link.peerAssignment && link.peerSourceVersion)void submit.submit({kind:'exchange',input:{
        operation:'Delete',expectedSourceVersion:removal.source,peerAssignment:link.peerAssignment,peerSourceVersion:link.peerSourceVersion,action:{action:a.id,revision:a.revision}}});
      return;
    }
    void submit.submit({
      kind: 'delete',
      action: a.id,
      input: { expectedSourceVersion: removal.source, expectedRevision: a.revision },
    });
  }
  function returnCurrent() {
    const entry = params.get('sourceEntry');
    state.invalidate();
    original.invalidate();
    if (!current) onSave();
    const next = new URLSearchParams(params);
    for (const key of ['sourceVersion', 'adoptedRef', 'sourceEntry']) next.delete(key);
    if (!selectedAction && unavailable && !source) next.delete('action');
    const options = {state:{...route.state, focusJobOriginal:true}};
    if (entry && personId && entry !== assignment && /^[a-f\d-]{36}$/i.test(entry)) {
      navigate('/personnel/employees/' + personId + '/assignments/' + entry + '?' + next, options);
    } else setParams(next, options);
  }
  const at = facts?.actions.filter(a => a.businessDate === facts.businessAt) ?? [];
  const historyNotice = <HistoryNotice onReturn={returnCurrent} label="返回当前职务信息" />;
  const evidence = original.data;
  const canCorrect = opened && !evidence?.exchange && (opened.canCorrectDuty || opened.canCorrectLocation || opened.canCorrectTransfer || opened.canCorrectPromotion);
  const canRedate = opened?.canMaintain && (!evidence?.exchange || evidence.exchange.canRedate);
  const canDelete = opened?.canDelete && (!evidence?.exchange || evidence.exchange.canDelete);
  const canInsertBefore = opened?.canCorrectPromotion && data?.canConfirm || opened?.kind === 'Transfer' && opened.canCorrectTransfer && data?.canPromote && !evidence?.exchange;
  const canMaintain = canCorrect || canRedate || canDelete || opened?.canWithdrawPromotion || canInsertBefore;
  function originalSourceURL(target: Action) {
    const next = new URLSearchParams(params);
    next.set('sourceVersion', target.sourceVersion);
    next.set('assignmentDay', target.businessDate);
    next.set('action', target.id);
    next.set('adoptedRef', target.revision);
    if (assignment !== target.assignment) next.set('sourceEntry', assignment);
    return '/personnel/employees/' + target.person + '/assignments/' + target.assignment + '?' + next;
  }
  return (
    <ObjectDetailSection title={selectedAction ? '办理明细' : '职务信息'}>
      {historical && (historySlot ? createPortal(historyNotice, historySlot) : actionSlot ? createPortal(historyNotice, actionSlot) : historyNotice)}
      {data && chosenId && !opened && <Alert severity="info" action={!selectedAction && !historical && <Button onClick={returnCurrent}>返回当前职务信息</Button>}>该办理或来源当前不可用。</Alert>}
      {state.error && (
        <Alert severity="error" action={<Button onClick={state.refresh}>重试</Button>}>
          {state.error}
        </Alert>
      )}
      {data && (
        <>
          <ObjectDetailField label="雇用记录编号">
            {data.recordNumber} · {data.primaryJob ? '主岗' : '兼岗'}
          </ObjectDetailField>
          {removal && opened?.canDelete && evidence ? <>
            <Typography component="h3">{removal.exchange ? '撤销两岗对调' : '删除记录'}</Typography>
            <ObjectDetailField label="处置目标">{actionLabel(opened)} · {opened.businessDate} · {data.recordNumber} 号{data.primaryJob ? '主岗' : '兼岗'}</ObjectDetailField>
            {removal.exchange && <ObjectDetailField label="影响范围">双方的岗位调动将一并撤销</ObjectDetailField>}
            {evidence.locationChange && <ObjectDetailField label="影响范围">移除本次地点变更。办理前地点为{evidence.locationChange.before?.name || '未记载'}。</ObjectDetailField>}
            {!removal.exchange && evidence.transfer && <ObjectDetailField label="影响范围">移除该任职的本次调动，按剩余办理重算任职安排。</ObjectDetailField>}
            <OriginalEvidence original={evidence} showExchangePeer={!!removal.exchange}/>
            {actionSlot && createPortal(<Stack direction="row" gap={1}>
              <Button color="error" variant="contained" disabled={submit.busy} onClick={confirmRemoval}>
                {removal.exchange ? '确认撤销两岗对调' : '确认删除'}
              </Button>
              <Button onClick={close}>取消</Button>
            </Stack>, actionSlot)}
          </> : draft ? (<>
            {conflict && <Box>
              <Alert severity="warning">任职或原办理已更新。请对照当前资料和本次输入，再继续办理。</Alert>
              {review.data && <ObjectDetailField label="当前任职">{[review.data.state?.department, review.data.state?.job, review.data.state?.location?.name].filter(Boolean).join(' · ') || '该日无有效任职'}</ObjectDetailField>}
              {review.error && <Alert severity="error" action={<Button onClick={review.refresh}>重试</Button>}>{review.error}</Alert>}
              {reviewTarget && evidence && <OriginalEvidence original={evidence} />}
              {review.data && reviewTarget && !aligned && (original.error || evidence) &&
                <Button onClick={()=>{review.invalidate(); original.invalidate(); state.invalidate();}}>重新读取当前办理</Button>}
              <Button disabled={!review.data || !aligned} onClick={acceptCurrent}>采用当前来源继续办理</Button>
            </Box>}
            <AssignmentRecordForm
              draft={draft}
              update={setDraft}
              view={data}
              csrf={session.csrf}
              busy={submit.busy || conflict}
              dirty={dirty}
              close={() => exit.request(close)}
              save={save}
              actionSlot={actionSlot}
            />
          </>) : (
            <>
              {!contractDraft && (!selectedAction || source) && (
                <>
                  {exact.error && <Alert severity="error" action={<Button onClick={exact.refresh}>重试</Button>}>{exact.error}</Alert>}
                  {facts && <div className="object-detail-columns"><ObjectDetailSection title="任职安排">
                  <ObjectDetailField label="该日任职">
                    {facts.state
                      ? [facts.state.department, facts.state.job, facts.state.location?.name]
                          .filter(Boolean)
                          .join(' · ')
                      : '该日无有效任职'}
                  </ObjectDetailField>
                  {facts.managers.map(fact=><ObjectDetailField key={fact.purpose} label={managerPurposes[fact.purpose]}>{fact.name} · {fact.recordNumber} 号</ObjectDetailField>)}
                  {facts.lastActive && <ObjectDetailField label="最后任职日">{facts.lastActive}</ObjectDetailField>}
                  {facts.state?.positionName && <ObjectDetailField label="岗位">{facts.state.positionName} · {facts.state.handover ? "交接占编" : facts.state.seats ? "占编" : "免占编"}</ObjectDetailField>}
                  {facts.state?.position && !historical && <DutyUsage position={facts.state.position} day={day} version={facts.sourceVersion}/>}
                  </ObjectDetailSection><ObjectDetailSection title="记录信息">
                  <ObjectDetailField label="生效日期">{facts.businessAt}</ObjectDetailField>
                  {facts.state && <ObjectDetailField label="个人职级">{facts.state.grade?.name ?? '未记载'}</ObjectDetailField>}
                  {facts.state && facts.probation.status !== 'unrecorded' && (
                    <ObjectDetailField label="试用状态">
                      {facts.probation.status === 'confirmed' ? '已转正' : '试用中'}
                    </ObjectDetailField>
                  )}
                  </ObjectDetailSection></div>}
                  <Typography component="h3">当日办理</Typography>
                  <Stack direction="row" flexWrap="wrap" gap={1}>
                    {at.map((a) =>
                      a.canHistory ? (
                        <Button
                          key={a.id}
                          id={chosenId === a.id ? controlId + '-record' : undefined}
                          onClick={() => select({action:a.id, assignmentDay:a.businessDate, adoptedRef:historical?a.revision:undefined, sourceVersion:historical?facts?.sourceVersion:undefined})}
                        >
                          {actionLabel(a)}{a.operation==='Withdraw'?'（已撤销）':a.operation==='Delete'?'（已删除）':''} · {a.businessDate}
                        </Button>
                      ) : (
                        <Typography key={a.id}>{actionLabel(a)}{a.operation==='Withdraw'?'（已撤销）':a.operation==='Delete'?'（已删除）':''}</Typography>
                      ),
                    )}
                    {!at.length && <Typography>没有当日办理</Typography>}
                  </Stack>
                  {mode === 'browse' && availableRecordTypes(data).length > 0 &&
                    actionSlot &&
                    createPortal(
                      <Stack direction="row" gap={1} flexWrap="wrap">
                        {data.canAddRecord && <Button id={controlId+'-add'} variant="contained" onClick={() => begin('add', 'add')}>
                          新增记录
                        </Button>}
                        {data.canInsertRecord && (
                          <Button id={controlId+'-insert'} onClick={() => begin('insert', 'insert')}>插入记录</Button>
                        )}
                      </Stack>,
                      actionSlot,
                    )}
                </>
              )}
              {!contractDraft && opened && original.error && (
                <Alert severity="error" action={<Button onClick={original.refresh}>重试</Button>}>
                  {original.error}
                </Alert>
              )}
              {!contractDraft && opened && original.data && (
                <Box id={controlId + '-original'} tabIndex={-1}>
                  <OriginalEvidence original={original.data} setRevision={value => {
                    if (!source && historical && value === opened.revision) returnCurrent();
                    else select({adoptedRef:value});
                  }} />
                  {!historical && original.data.action.revision === opened.revision && canMaintain && <>
                    <Button id={controlId + '-more'} aria-haspopup="menu" aria-expanded={!!menu} onClick={e => setMenu(e.currentTarget)}>更多操作</Button>
                    <Menu anchorEl={menu} open={!!menu} onClose={() => setMenu(null)}>
                      {canCorrect && <MenuItem onClick={() => begin('more', 'correct', opened)}>更正记录</MenuItem>}
                      {canInsertBefore && <MenuItem onClick={() => begin('more', 'insert', undefined, opened)}>
                        {opened.kind === 'Promotion' ? '在本次晋升前补录转正' : '在本次调动前补录晋升'}
                      </MenuItem>}
                      {opened.canWithdrawPromotion && <MenuItem onClick={() => begin('more', 'withdraw', opened)}>撤销晋升</MenuItem>}
                      {canRedate && <MenuItem onClick={() => begin('more', 'redate', opened)}>
                        {original.data.exchange ? '调整对调日期' : '调整生效日期'}
                      </MenuItem>}
                      {canDelete && <MenuItem disabled={submit.busy} onClick={() => {
                        focusTarget.current = controlId + '-more';
                        setMenu(null);
                        setNotice('');
                        setError('');
                        submit.start();
                        setRemoval({action:opened, source:data.sourceVersion, exchange:original.data?.exchange});
                      }}>{original.data.exchange ? '撤销两岗对调' : '删除记录'}</MenuItem>}
                    </Menu>
                  </>}
                  {!source && original.data.promotion?.priorSource && <Button onClick={() =>
                    select({sourceVersion:original.data!.promotion!.priorSource, adoptedRef:undefined, assignmentDay:opened.businessDate})}>查看撤销前来源</Button>}
                  {!source && !selectedAction && (original.data.action.sourceVersion !== data.sourceVersion || relationAction(opened.kind)) &&
                    <Button component={Link} to={originalSourceURL(original.data.action)}>查看当次任职</Button>}
                </Box>
              )}
            </>
          )}
        </>
      )}
      {error && (
        <Alert tabIndex={-1} ref={errorRef} severity="error">
          {error}
        </Alert>
      )}
      {notice && <Alert severity="success">{notice}</Alert>}
      {data?.canReadContracts && !draft && !removal && !selectedAction && !historical && !unavailable && <RelationshipContracts actionSlot={actionSlot} view={data} session={session} draft={contractDraft} update={setContractDraft}
        begin={(cmd,b)=>{if (!b.id) b.id=controlId+'-contract';focusTarget.current=b.id;saved.current=false;contractInitial.current=cmd;setContractDraft(cmd)}} cancel={()=>exit.request(close)} finish={()=>{saved.current=true;close()}}/>}
      {!historical && submit.pending.map((ref) => (
        <Button key={ref.submissionId} onClick={() => void submit.recover(ref)}>
          查回职务信息提交
        </Button>
      ))}
      {submit.storageError && <Alert severity="error">无法保存提交号，请恢复浏览器存储后办理。</Alert>}
      <UnsavedChangesDialog {...exit.dialog} />
    </ObjectDetailSection>
  );
}

function NoticeSource({ reference, label }: { reference: Models['EmploymentActionRef']; label: string }) {
  const [open, setOpen] = useState(false);
  const source = useRead(open ? JSON.stringify(reference) : undefined, async () =>
    unwrap(await api.GET('/api/v1/employment-actions/{id}', {
      params: { path: { id: reference.action }, query: { adoptedRef: reference.revision } },
    })));
  const original = source.data;
  return <Box>
    <Button aria-expanded={open} onClick={() => setOpen(!open)}>{label}</Button>
    {open && source.error && <Alert severity="error">{source.error}</Alert>}
    {open && original && <ObjectDetailField label="引用的原件">
      {actionLabel(original.action)} · {original.action.businessDate} · {original.action.notice}
    </ObjectDetailField>}
  </Box>;
}

export function OriginalEvidence({
  original,
  setRevision,
  showExchangePeer,
}: {
  original: Models['HireOriginal'];
  setRevision?: (id: string) => void;
  showExchangePeer?: boolean;
}) {
  const route = useLocation();
  const order = original.previous ?? (original.probation?.afterAction && original.probation.afterRevision
    ? {action: original.probation.afterAction, revision: original.probation.afterRevision} : undefined);
  const operator = original.revisions?.find((r) => r.id === original.action.revision)?.operator;
  return (
    <>
      <Typography component="h3">{actionLabel(original.action)}原件</Typography>
      <ObjectDetailField label="生效日期">{original.action.businessDate}</ObjectDetailField>
      {original.initialRecordPosition && <>
        <ObjectDetailField label="原登记操作">{original.initialRecordPosition.recorded === 'Add' ? '新增记录' : '插入记录'}</ObjectDetailField>
        <ObjectDetailField label="当时履历位置">{original.initialRecordPosition.effective
          ? (original.initialRecordPosition.effective === 'Add' ? '末端新增' : '历史插入')
          : '待核对当时的办理原件'}</ObjectDetailField>
      </>}
      {original.exchange && <ExchangeOriginal key={original.action.revision} link={original.exchange} initiallyOpen={showExchangePeer}/>}
      {original.manager && <ObjectDetailField label={managerPurposes[original.manager.purpose]}>{original.manager.manager ? original.manager.name+' · '+original.manager.recordNumber+' 号' : '汇报结束'}</ObjectDetailField>}
      {original.exit && <>
        <ObjectDetailField label={original.exit.meaning==='last-active'?'最后在职日':'首个不在职日'}>{original.exit.date}</ObjectDetailField>
        {original.exit.meaning==='last-active' && <ObjectDetailField label="首个不在职日">{original.exit.firstInactive}</ObjectDetailField>}
      </>}
      {original.promotion && <>
        <ObjectDetailField label="个人职级">{original.promotion.before?.name ?? '未记载'} → {original.promotion.after.name}</ObjectDetailField>
        <ObjectDetailField label="发生部门">{original.promotion.departmentName}</ObjectDetailField>
        <ObjectDetailField label="办理原因">{original.promotion.reason}</ObjectDetailField>
        {original.promotion.withdrawal && <ObjectDetailField label="撤销依据">{original.promotion.withdrawal}</ObjectDetailField>}
      </>}
      {original.action.operation==='Withdraw' && <ObjectDetailField label="办理状态">已撤销</ObjectDetailField>}
      {original.relation && (
        <>
          <ObjectDetailField label="所属工作关系">
            {original.relation.employerName} · {original.relation.from}
            {original.relation.until && ' 至 ' + original.relation.until + '（不含）'}
          </ObjectDetailField>
          {original.relation.terms.length > 0 && (
            <ObjectDetailField label="相关任职">
              {original.relation.terms
                .map((t) => t.recordNumber + ' 号' + (t.recordNumber === 0 ? '主岗' : '兼岗'))
                .join('、')}
            </ObjectDetailField>
          )}
        </>
      )}
      {original.sources?.offer && <ObjectDetailField label="采用的招聘录用"><Link to={adoptedOfferURL(original.sources.offer.offer, original.sources.offer.version, route.pathname + route.search)}>{original.sources.offer.number}</Link></ObjectDetailField>}
      {original.sources?.grade && <ObjectDetailField label="首任职级">{original.sources.grade.name}</ObjectDetailField>}
      {original.trialStart && (
        <ObjectDetailField label="实际试用开始日">{original.trialStart}</ObjectDetailField>
      )}
      {original.probation && (
        <>
          <ObjectDetailField label="实际试用开始日">{original.probation.start}</ObjectDetailField>
          {original.probation.departmentName && (
            <ObjectDetailField label="发生部门">{original.probation.departmentName}</ObjectDetailField>
          )}
          {['same-day-order-undetermined', 'primary-unavailable'].includes(
            original.probation.attribution,
          ) && (
            <ObjectDetailField label="发生部门">
              {original.probation.attribution === 'same-day-order-undetermined'
                ? '未确定 · 同日调动与转正的实际先后未载明'
                : '未确定 · 缺少当日关系主任职归属'}
            </ObjectDetailField>
          )}
        </>
      )}
      {original.action.operation === 'Delete' && (
        <ObjectDetailField label="办理状态">已删除</ObjectDetailField>
      )}
      {original.secondary && <>
        <ObjectDetailField label="任职安排">{[original.secondary.segments[0]?.departmentName,original.secondary.segments[0]?.jobName,original.secondary.segments[0]?.positionName].filter(Boolean).join(' · ')}</ObjectDetailField>
        {original.secondary.grade && <ObjectDetailField label="个人职级">{original.secondary.grade.name}</ObjectDetailField>}
        <ObjectDetailField label="所属工作关系">{original.secondary.legal[0]?.entityName}{original.secondary.relationshipPrimary?' · 关系主任职':''}</ObjectDetailField>
      </>}
      {original.transfer?.after.duty && <DutyOriginal evidence={original.transfer.after.duty}/>}
      {original.transfer?.after.succession && <ObjectDetailField label="接任交接">{original.transfer.after.succession.name} · {original.transfer.after.succession.from} 起交接，{original.transfer.after.succession.until}{original.transfer.after.succession.continues?' 起占常规名额':' 起结束交接'}</ObjectDetailField>}
      {original.transfer?.beforeDepartmentName && (
        <ObjectDetailField label="调出安排">
          {[original.transfer.beforeDepartmentName, original.transfer.beforeJobName, original.transfer.beforePositionName]
            .filter(Boolean)
            .join(' · ')}
        </ObjectDetailField>
      )}
      {original.transfer?.after.segments?.[0] && (
        <ObjectDetailField label="调入安排">
          {[original.transfer.after.segments[0].departmentName, original.transfer.after.segments[0].jobName,original.transfer.after.segments[0].positionName, original.transfer.after.segments[0].position ? (original.transfer.after.segments[0].handoverSource ? '交接占编' : original.transfer.after.segments[0].seats ? '占编' : '免占编') : undefined]
            .filter(Boolean)
            .join(' · ')}
        </ObjectDetailField>
      )}
      <ObjectDetailField label="首次办理时间">{recordedTime(original.action.recordedAt)}</ObjectDetailField>
      {original.departmentName && !original.promotion && (
        <ObjectDetailField label="发生部门">{original.departmentName}</ObjectDetailField>
      )}
      {original.locationChange && <ObjectDetailField label="业务原因">工作地点变更</ObjectDetailField>}
      {original.locationChange && (
        <ObjectDetailField label="工作地点">
          {original.locationChange.before?.name ?? '—'} → {original.locationChange.after.name}
        </ObjectDetailField>
      )}
      {original.locationChange?.replaced && (
        <ObjectDetailField label="更正前记录值">{original.locationChange.replaced.name}</ObjectDetailField>
      )}
      {original.action.notice && (
        <ObjectDetailField label="人事依据">{original.action.notice}</ObjectDetailField>
      )}
      {original.probation && original.reason && <ObjectDetailField label="修订原因">{original.reason}</ObjectDetailField>}
      {order && <NoticeSource key={order.revision} reference={order} label="查看前序办理" />}
      {original.noticeRef && <NoticeSource key={original.noticeRef.revision} reference={original.noticeRef} label="查看通知来源" />}
      {original.promotion?.withdrawalBasis && <NoticeSource key={original.promotion.withdrawalBasis.revision} reference={original.promotion.withdrawalBasis} label="查看撤销依据来源" />}
      {setRevision && original.revisions?.map((r,index) => (
        <Button key={r.id} onClick={() => setRevision(r.id)} aria-pressed={r.id === original.action.revision}>
          查看修订 · {recordedTime(r.recordedAt)} · 第 {(original.revisions?.length ?? 0)-index} 次
        </Button>
      ))}
      {operator && <ObjectDetailField label="经办人">{operator}</ObjectDetailField>}
    </>
  );
}
