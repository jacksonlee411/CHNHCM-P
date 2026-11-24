import { useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { Alert, Box, Button, Stack, Typography } from '@mui/material';
import { api, ApiError, unwrap, type Models } from '@chnhcm/api-client';
import {
  EffectiveDates,
  ObjectDetailField,
  ObjectDetailSection,
  ObjectTextField,
  UnsavedChangesDialog,
  recordedTime,
} from '@chnhcm/ui';
import { RelationshipContracts } from './RelationshipContracts';
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
  type JobDraft,
} from './AssignmentRecordForm';

type Action = Models['EmploymentAction'];
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
  assignment,
  day,
  onDay,
  session,
  onSave,
  actionSlot,
  selectedAction,
}: {
  assignment: string;
  day: string;
  onDay: (date: string) => void;
  session: Models['Principal'];
  onSave: () => void;
  actionSlot: HTMLElement | null;
  selectedAction?: string;
}) {
  const [source, setSource] = useState('');
  const [contractDraft,setContractDraft]=useState<Models['EmploymentContractCommand']>();
  const contractInitial=useRef<Models['EmploymentContractCommand']|undefined>(undefined);
  const [draft, setDraft] = useState<JobDraft>();
  const initial = useRef<JobDraft | undefined>(undefined);
  const [chosen, setOpened] = useState<Action>();
  const [revision, setRevision] = useState('');
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const errorRef = useRef<HTMLDivElement>(null);
  const opener = useRef<HTMLButtonElement | null>(null);
  const saved = useRef(false);
  const header = { 'X-CSRF-Token': session.csrf };
  const state = useRead(JSON.stringify([assignment, day, source]), async () =>
    unwrap(
      await api.GET('/api/v1/assignments/{id}', {
        params: {
          path: { id: assignment },
          query: { businessAt: day || undefined, sourceVersion: source || undefined },
        },
      }),
    ),
  );
  const data = state.data;
  const opened = selectedAction ? data?.actions.find((a) => a.id === selectedAction && a.canHistory) : chosen;
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
      setOpened(undefined);
      setError('已重新核对权限，当前不可办理。');
    }
  }, [data, draft]);
  const original = useRead(
    opened ? JSON.stringify([opened.id, revision || opened.revision]) : undefined,
    async () =>
      unwrap(
        await api.GET('/api/v1/employment-actions/{id}', {
          params: { path: { id: opened!.id }, query: { adoptedRef: revision || opened!.revision } },
        }),
      ),
  );
  const close = () => {
    setContractDraft(undefined);
    setDraft(undefined);
    setError('');
    requestAnimationFrame(() => opener.current?.focus());
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
        if (draft) onDay(draft.day);
        close();
        setOpened(undefined);
        setRevision('');
        setNotice(recovery ? '已查回原办理。' : '职务信息已保存。');
      }
      state.invalidate();
      onSave();
    },
    failed: (e) => {
      setError(message(e));
      if (e instanceof ApiError && [401, 403, 404].includes(e.status)) {
        setDraft(undefined);
        setOpened(undefined);
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
    if (state.status === 403 || state.status === 404) {
      setDraft(undefined);
      setOpened(undefined);
      setRevision('');
    }
  }, [state.status]);
  function begin(button: HTMLButtonElement, mode: JobDraft['mode'], a?: Action, successor?:Action) {
    if (!data) return;
    opener.current = button;
    saved.current = false;
    setError('');
    setNotice('');
    submit.start();
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
      kind: after?.duty && a ? 'ActualDuty' : evidence?.exchange && a ? 'PositionExchange' : successor ? 'Promotion' : a ? (a.kind==='Promotion'?'Promotion':a.kind === 'Transfer' ? 'Transfer' : 'DataChange') : availableRecordTypes(data)[0],
      successor,
      grade:evidence?.promotion?.after,originalGrade:evidence?.promotion?.before ?? state?.grade,
      reason:evidence?.promotion?.reason,
      predecessor:data.actions.find(item=>item.id===evidence?.previous?.action),
      action: a,
      day: a?.businessDate ?? day,
      source: data.sourceVersion,
      value: a ? evidence?.locationChange?.after : state?.location,
      department: department
        ? { id: department, name: after?.segments?.[0]?.departmentName ?? state?.department ?? '' }
        : undefined,
      position: position ? {id: position, name: adoptedPosition?.positionName ?? state?.positionName ?? position} : undefined,
      job: job ? { id: job, name: after?.segments?.[0]?.jobName ?? state?.job ?? '' } : undefined,
      notice: evidence?.action.notice ?? '',
      requires: evidence?.requires,
    };
    if(successor && evidence?.transfer){
      const prior=evidence.transfer;
      next.department=prior.beforeDepartment ? {id:prior.beforeDepartment,name:prior.beforeDepartmentName ?? ''}:undefined;
      next.job=prior.beforeJob ? {id:prior.beforeJob,name:prior.beforeJobName ?? ''}:undefined;
      next.originalGrade=prior.beforeGrade;next.notice='';
    }
    initial.current = next;
    setDraft(next);
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
    const reference=(a:Action|undefined)=>a?{action:a.id,revision:a.revision}:undefined;
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
        },
      });
      return;
    }
    const changes = { ...input, insert: record.mode === 'insert' || undefined, requires: record.requires };
    if (record.kind === 'DataChange' && record.value)
      void submit.submit({ kind: 'location', action, input: { ...changes, location: record.value.id } });
    if (record.kind === 'Transfer')
      void submit.submit({
        kind: 'transfer',
        action,
        input: { ...changes, incumbent:record.handover&&record.incumbent?{assignment:record.incumbent.assignment,sourceVersion:record.incumbent.sourceVersion}:undefined, position:record.position?.id, department: record.department?.id, job: record.job?.id, notice: record.notice,previous:reference(record.predecessor) },
      });
  }
  function remove(a: Action) {
    if (!data || !a.canDelete) return;
    setError('');
    setNotice('');
    submit.start();
    const link=original.data?.exchange;
    if(link){
      if(link.canDelete && link.peerAssignment && link.peerSourceVersion)void submit.submit({kind:'exchange',input:{
        operation:'Delete',expectedSourceVersion:data.sourceVersion,peerAssignment:link.peerAssignment,peerSourceVersion:link.peerSourceVersion,action:{action:a.id,revision:a.revision}}});
      return;
    }
    void submit.submit({
      kind: 'delete',
      action: a.id,
      input: { expectedSourceVersion: data.sourceVersion, expectedRevision: a.revision },
    });
  }
  const at = data?.actions.filter((a) => a.businessDate === day) ?? [];
  const availableDays = Array.from(new Set(data?.actions.map((a) => a.businessDate) ?? []));
  return (
    <ObjectDetailSection title={selectedAction ? '办理明细' : '职务信息'}>
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
          {draft ? (
            <AssignmentRecordForm
              draft={draft}
              update={setDraft}
              view={data}
              csrf={session.csrf}
              busy={submit.busy}
              dirty={dirty}
              close={() => exit.request(close)}
              save={save}
            />
          ) : (
            <>
              {!contractDraft && (!selectedAction || source) && (
                <>
                  {availableDays.length > 0 && (
                    <EffectiveDates
                      dates={availableDays}
                      selected={day}
                      onSelect={(value) => {
                        onDay(value);
                        setOpened(undefined);
                        setRevision('');
                      }}
                    />
                  )}
                  <ObjectTextField
                    type="date"
                    label="查看业务日"
                    value={day}
                    onChange={(e) => {
                      onDay(e.target.value);
                      setOpened(undefined);
                      setRevision('');
                    }}
                  />
                  <ObjectDetailField label="该日任职">
                    {data.state
                      ? [data.state.department, data.state.job, data.state.location?.name]
                          .filter(Boolean)
                          .join(' · ')
                      : '该日无有效任职'}
                  </ObjectDetailField>
                  {data.managers.map(fact=><ObjectDetailField key={fact.purpose} label={managerPurposes[fact.purpose]}>{fact.name} · {fact.recordNumber} 号</ObjectDetailField>)}
                  {data.lastActive && <ObjectDetailField label="最后任职日">{data.lastActive}</ObjectDetailField>}
                  {data.state?.positionName && <ObjectDetailField label="岗位">{data.state.positionName} · {data.state.handover ? "交接占编" : data.state.seats ? "占编" : "免占编"}</ObjectDetailField>}
                  {data.state?.position && !source && <DutyUsage position={data.state.position} day={day} version={data.sourceVersion}/>}
                  {data.state && <ObjectDetailField label="个人职级">{data.state.grade?.name ?? '未记载'}</ObjectDetailField>}
                  {data.state && data.probation.status !== 'unrecorded' && (
                    <ObjectDetailField label="试用状态">
                      {data.probation.status === 'confirmed' ? '已转正' : '试用中'}
                    </ObjectDetailField>
                  )}
                  <Typography component="h3">当日办理</Typography>
                  <Stack direction="row" flexWrap="wrap" gap={1}>
                    {at.map((a) =>
                      a.canHistory ? (
                        <Button
                          key={a.id}
                          onClick={() => {
                            setOpened(a);
                            setRevision('');
                          }}
                        >
                          {actionLabel(a)} · {a.businessDate}
                        </Button>
                      ) : (
                        <Typography key={a.id}>{actionLabel(a)}</Typography>
                      ),
                    )}
                    {!at.length && <Typography>没有当日办理</Typography>}
                  </Stack>
                  {availableRecordTypes(data).length > 0 &&
                    actionSlot &&
                    createPortal(
                      <Stack direction="row" gap={1} flexWrap="wrap">
                        <Button variant="contained" onClick={(e) => begin(e.currentTarget, 'add')}>
                          新增记录
                        </Button>
                        {data.actions.some((a) => a.kind !== 'Hire' && a.operation !== 'Delete') && (
                          <Button onClick={(e) => begin(e.currentTarget, 'insert')}>插入记录</Button>
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
                <Box>
                  <OriginalEvidence original={original.data} setRevision={setRevision} />
                  {original.data.action.revision === opened.revision && !source && (
                    <Stack direction="row" gap={1} flexWrap="wrap">
                      {!original.data.exchange && (opened.canCorrectDuty || opened.canCorrectLocation || opened.canCorrectTransfer || opened.canCorrectPromotion) && (
                        <Button onClick={(e) => begin(e.currentTarget, 'correct', opened)}>更正记录</Button>
                      )}
                      {opened.canWithdrawPromotion && <Button onClick={e=>begin(e.currentTarget,'withdraw',opened)}>撤销晋升</Button>}
                      {!original.data.exchange && opened.kind==='Transfer' && opened.canCorrectTransfer && data.canPromote &&
                        <Button onClick={e=>begin(e.currentTarget,'insert',undefined,opened)}>在本次调动前补录晋升</Button>}
                      {opened.canMaintain && (!original.data.exchange || original.data.exchange.canRedate) && (
                        <Button onClick={(e) => begin(e.currentTarget, 'redate', opened)}>
                          {original.data.exchange ? '调整对调日期' : '调整生效日期'}
                        </Button>
                      )}
                      {opened.canDelete && (!original.data.exchange || original.data.exchange.canDelete) && (
                        <Button disabled={submit.busy} onClick={() => remove(opened)}>
                          {original.data.exchange ? '撤销两岗对调' : '删除记录'}
                        </Button>
                      )}
                    </Stack>
                  )}
                  {!source && original.data.promotion?.priorSource && <Button onClick={()=>{
                    setSource(original.data!.promotion!.priorSource!);setRevision('');setOpened(undefined);
                  }}>查看撤销前来源</Button>}
                  {!selectedAction && original.data.action.sourceVersion !== data.sourceVersion && (
                    <Button
                      onClick={() => {
                        setSource(original.data!.action.sourceVersion);
                        setOpened(undefined);
                      }}
                    >
                      查看当次任职
                    </Button>
                  )}
                </Box>
              )}
              {source && (
                <Button
                  onClick={() => {
                    setSource('');setRevision('');
                    setOpened(undefined);
                  }}
                >
                  返回当前职务信息
                </Button>
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
      {data?.canReadContracts && !draft && !selectedAction && !source && <RelationshipContracts view={data} session={session} draft={contractDraft} update={setContractDraft}
        begin={(cmd,b)=>{opener.current=b;saved.current=false;contractInitial.current=cmd;setContractDraft(cmd)}} cancel={()=>exit.request(close)} finish={()=>{saved.current=true;close()}}/>}
      {submit.pending.map((ref) => (
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
}: {
  original: Models['HireOriginal'];
  setRevision: (id: string) => void;
}) {
  const operator = original.revisions?.find((r) => r.id === original.action.revision)?.operator;
  return (
    <>
      <Typography component="h3">{actionLabel(original.action)}原件</Typography>
      <ObjectDetailField label="生效日期">{original.action.businessDate}</ObjectDetailField>
      {original.exchange && <ExchangeOriginal key={original.action.revision} link={original.exchange}/>}
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
        <ObjectDetailField label="任职安排">{[original.secondary.segments[0]?.departmentName,original.secondary.segments[0]?.jobName].filter(Boolean).join(' · ')}</ObjectDetailField>
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
      {original.noticeRef && <NoticeSource key={original.noticeRef.revision} reference={original.noticeRef} label="查看通知来源" />}
      {original.promotion?.withdrawalBasis && <NoticeSource key={original.promotion.withdrawalBasis.revision} reference={original.promotion.withdrawalBasis} label="查看撤销依据来源" />}
      {original.revisions?.map((r) => (
        <Button key={r.id} onClick={() => setRevision(r.id)} aria-pressed={r.id === original.action.revision}>
          查看修订 · {recordedTime(r.recordedAt)}
        </Button>
      ))}
      {operator && <ObjectDetailField label="经办人">{operator}</ObjectDetailField>}
    </>
  );
}
