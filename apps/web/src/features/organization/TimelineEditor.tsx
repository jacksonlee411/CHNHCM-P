import { SubmissionError } from '../../shared/dated/submission';
import { selectedReason } from './dated/reasons';
import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import {
  Alert,
  Box,
  Button,
  MenuItem,
  Stack,
  Typography,
} from "@mui/material";
import { useDraftExit } from "../../shared/dated/useDraftExit";
import {
  unwrap,
  ApiError,
  sessionGuard,
  type Models,
} from "@chnhcm/api-client";
import {
  ObjectTextField,
  ObjectForm,
  ObjectFormError,
  ObjectDetailField,
  ObjectDetailSection,
  UnsavedChangesDialog,
  businessDate, businessPeriod,
} from "@chnhcm/ui";
import { useDefinition, inputValues, type RecordCommand, type DetailView, type Result, type Preview, type PreviewInput, type CorrectInput, type DataKey, type DataFields, type ReferenceContext } from './dated/presentation';
import { DataEditor, recordLabels as businessUnitFields } from './dated/DataEditor';
import { trimDetailFields, trimDetailReferences } from './businessUnitProjection';
import { useSubmission } from './dated/useSubmission';

type Props = {
  session: Models["Principal"];
  detail: DetailView;
  action: "ADD" | "CORRECT" | "DEACTIVATE" | "REACTIVATE" | "DELETE";
  actionsContainer: HTMLElement | null;
  onClose: () => void;
  onDenied: () => void;
  onReloaded: (detail: DetailView) => void;
  onSaved: (r: Result) => void;
};
const message = (e: unknown) =>
  e instanceof ApiError || e instanceof SubmissionError ? e.message : "暂时无法完成请求";
const operationTypeLabels: Record<string, string> = {
  CREATE: "新建", CHANGE: "数据更改", DEACTIVATE: "停用", REACTIVATE: "重新启用",
};
const errorFieldId = (field: string, prefix: string) => field === 'anchorRecordId' ? `${prefix}-record-editor-title` : field === 'classification.operationType' ? 'record-classification-type' : field === 'classification.operationReason' ? 'record-classification-reason' : (field === 'effectiveDate' || field === 'newEffectiveDate') ? 'record-date' : field === 'operationReason' ? 'record-reason' : `record-${field.replace('changes.', '')}`;

export function TimelineEditor({
  session,
  detail,
  action,
  actionsContainer,
  onClose,
  onDenied,
  onReloaded,
  onSaved,
}: Props) {
  const businessUnit = useDefinition(), businessUnitApi = businessUnit.api;
  const [basis, setBasis] = useState(detail);
  const isLifecycle = action === "DEACTIVATE" || action === "REACTIVATE";
  const [lifecycleAnchor] = useState(detail.lifecycleContext?.anchorRecordId ?? "");
  const [lifecycleBaseline, setLifecycleBaseline] = useState<DataFields & {date:string}>({date:detail.lifecycleContext?.defaultEffectiveDate??"",...inputValues(detail.lifecycleContext?.defaults)});
  const [addAnchor]=useState(detail.additionContext?.anchorRecordId??'');
  const [addBaseline,setAddBaseline]=useState<DataFields & {date:string}>({date:detail.additionContext?.defaultEffectiveDate??'',...inputValues(detail.additionContext?.defaults)});
  const isDelete = action === "DELETE";
  const isNew = action !== "CORRECT" && !isDelete;
  const operationType = action === "ADD" ? "CHANGE" : action === "CORRECT" || isDelete ? undefined : action;
  const title = {ADD:basis.additionContext?.mode==="INSERT"?"插入记录":"新增记录",CORRECT:"更正记录",DEACTIVATE:`停用${businessUnit.label}`,REACTIVATE:`重新启用${businessUnit.label}`,DELETE:"删除此记录"}[action];
  const saveLabel = {ADD:"保存",CORRECT:"保存",DEACTIVATE:"确认停用",REACTIVATE:"确认重新启用",DELETE:"确认删除"}[action];
  const [reasonRetry, retryReasons] = useState(0);
  const [reasonError, setReasonError] = useState(false);
  const [reasons,setReasons] = useState<Models["Reason"][]>([]);


  const target = basis.selectedRecord;
  const [date, setDate] = useState(action === "CORRECT" ? target?.effectiveDate ?? "" : action==="ADD"?addBaseline.date:isLifecycle?lifecycleBaseline.date:"");
  const [reason, setReason] = useState("");
  const [values, setValues] = useState<DataFields>(action === 'CORRECT' ? inputValues(target) : action === 'ADD' ? { ...addBaseline } : isLifecycle ? { ...lifecycleBaseline } : {});
  const [touched, setTouched] = useState<DataKey[]>([]);
  const [classificationType, setClassificationType] = useState(target?.operationType ?? "");
  const [classificationReason, setClassificationReason] = useState(target?.operationReason ?? "");
  const [pendingStop, setPendingStop] = useState(false);
  const [dateTouched,setDateTouched]=useState(false);
  const [classificationTouched,setClassificationTouched]=useState(false);
  const canReclassify = action === "CORRECT" && !!basis.capabilities.correction?.facets.includes("RECLASSIFY");
  const catalogType = action === "CORRECT" ? (canReclassify ? classificationType : undefined) : operationType;
  useEffect(()=>{
    if(!catalogType) return;
    setReasons([]); if (reasonError) setError(''); setReasonError(false);
    let active=true;const same=sessionGuard();
    void businessUnitApi.reasons(catalogType as "CREATE"|"CHANGE"|"DEACTIVATE"|"REACTIVATE").then(unwrap).then(v=>{if(active&&same()){setReasons(v);if(action==='CORRECT'){if(classificationType!==target?.operationType)setClassificationReason(previous=>selectedReason(previous,v));}else setReason(previous=>selectedReason(previous,v));}}).catch(e=>{if(active&&same()){setReasonError(true);setError(message(e))}});
    return ()=>{active=false};
  },[reasonRetry, catalogType,session.instanceId,session.subjectId]);
  const [assistance, setAssistance] = useState<Preview>();
  const [assisting, setAssisting] = useState(false);
  const [assistanceError, setAssistanceError] = useState("");
  const [assistanceBlocked, setAssistanceBlocked] = useState(false);
  const [assistanceRevision, retryAssistance] = useState(0);
  const [noChanges, setNoChanges] = useState(false);
  const [error, setError] = useState("");
  const [field, setField] = useState("");
  const [focusRequest, setFocusRequest] = useState<{ field: string }>();
  const [reloading, setReloading] = useState(false);
  const reloadRequest = useRef(0);
  const [conflict, setConflict] = useState(false);
  const mounted = useRef(false);
  const { flow, pending, sessionBlocked } = useSubmission({
    channel: businessUnit.submissions, owner: session, resolve: reference => businessUnit.resolve(session, reference),
    send: (command: RecordCommand, id) => businessUnitApi.record(detail.id, command, { 'X-CSRF-Token': session.csrf, 'Idempotency-Key': id }),
    onResolved: result => {
      setError('');
      if (result.outcome === 'accepted' || result.outcome === 'no_change') { setTouched([]); setDate(''); setReason(''); onSaved(result); }
      else setError('');
    },
    onDenied: () => { clear(); onDenied(); },
  });
  const { inFlight: busy, unknown } = flow;
  const epoch = useRef(0);
  const correctedNames = touched.filter(f => (values[f] ?? '') !== (target?.values[f] ?? ''));
  const addedNames = touched.filter(f => (values[f] ?? '') !== (addBaseline[f] ?? ''));
  const lifecycleNames = touched.filter(f => (values[f] ?? '') !== (lifecycleBaseline[f] ?? ''));
  const edits = (fields: DataKey[]): DataFields => Object.fromEntries(fields.map(f => [f, values[f] || (f === 'name' || f === 'managementBuId' ? '' : null)]));
  const dateChanged = action === 'CORRECT' && dateTouched && date !== (target?.effectiveDate ?? '');
  const classificationChanged = action === 'CORRECT' && classificationTouched && (classificationType !== (target?.operationType ?? '') || classificationReason !== (target?.operationReason ?? ''));
  const dirty = action === 'CORRECT' ? correctedNames.length > 0 || dateChanged || classificationChanged : action==="ADD" ? addedNames.length>0 || date!==addBaseline.date || reason !== selectedReason('', reasons) : isLifecycle ? lifecycleNames.length>0 || date!==lifecycleBaseline.date || reason !== selectedReason('', reasons) : touched.length > 0 || !!date || !!reason;
  const correctionInput = (): CorrectInput => ({expectedSourceVersion:basis.headSourceVersion??'',expectedRecordRevision:target?.revision??'',
    ...(correctedNames.length ? {changes:edits(correctedNames)} : {}),
    ...(dateChanged ? {newEffectiveDate:date} : {}),
    ...(classificationChanged ? {classification:{operationType:classificationType as Models['Classification']['operationType'],operationReason:classificationReason}} : {})});
  useEffect(() => {
    if (!focusRequest || busy) return;
    document.getElementById(errorFieldId(focusRequest.field, businessUnit.prefix))?.focus();
    setFocusRequest(undefined);
  }, [focusRequest, busy, businessUnit.prefix]);
  const capability = (c:Models["Capabilities"],addition:DetailView["additionContext"],lifecycle:DetailView["lifecycleContext"]) => {
    switch(action) {
      case "ADD": return {allowed:c.add && !!addition?.available && addition.anchorRecordId===addAnchor,fields:c.addWriteFields ?? []};
      case "CORRECT": return {allowed:c.correct,fields:c.correctWriteFields ?? []};
      case "DEACTIVATE": return {allowed:c.deactivate && lifecycle?.anchorRecordId===lifecycleAnchor,fields:c.deactivateWriteFields ?? []};
      case "REACTIVATE": return {allowed:c.reactivate && lifecycle?.anchorRecordId===lifecycleAnchor,fields:c.reactivateWriteFields ?? []};
      case "DELETE": return {allowed:c.delete,fields:[]};
    }
  };
  const {allowed:mayAct,fields:writes}=capability(basis.capabilities,basis.additionContext,basis.lifecycleContext);
  const can = (f:string) => basis.capabilities.readFields.includes(f);
  const showClassificationInputs = action === "CORRECT" && (canReclassify || classificationChanged && can("operationType") && can("operationReason"));
  const changed = () => {
    epoch.current++;
    setNoChanges(false);
    setAssistance(undefined);
    setAssistanceError("");
    setAssistanceBlocked(false);
    setError("");
    setField("");
  };
  const mark = (f: DataKey) =>
    setTouched((v) => (v.includes(f) ? v : [...v, f]));
  const clear = () => {
    setValues(inputValues());
    setDate("");
    setReason("");
    setTouched([]);
    flow.clearPayload();
    epoch.current++;
    setAssistance(undefined);
  };
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      epoch.current++;
    };
  }, []);
  const leave = useDraftExit(dirty, () => flow.saved, clear);
  const input = (): PreviewInput => action === "CORRECT" ? {action,recordId:target?.recordId,...correctionInput()} : ({
    action,
    ...(action==='ADD'?{anchorRecordId:addAnchor}:isLifecycle?{anchorRecordId:lifecycleAnchor}:{}),
    expectedSourceVersion: basis.headSourceVersion ?? "",
    ...(action==='ADD'?{changes:edits(addedNames)}:action==='REACTIVATE' && lifecycleNames.length ? {changes:edits(lifecycleNames)} : {}),
    ...(isNew
      ? {
          effectiveDate: date,
          operationType,
          operationReason: reason as PreviewInput["operationReason"],
        }
      : {
          recordId: target?.recordId ?? "",
          expectedRecordRevision: target?.revision ?? "",
        }),
  });
  const clearUnreadInputs = (fields: string[]) => {
    setValues(v => Object.fromEntries(Object.entries(v).filter(([key]) => fields.includes(key))));
    if (!fields.includes("effectiveDate")) {setDate("");setDateTouched(false);}
    if (!fields.includes("operationReason")) {setReason("");setClassificationReason("");}
    if (!fields.includes("operationType")) setClassificationType("");
    if (!fields.includes("operationReason") || !fields.includes("operationType")) {
      if(action === "CORRECT") setReasons([]);
      setClassificationTouched(false);
    }
    setTouched(touched => touched.filter(field => fields.includes(field)));
    flow.clearPayload();
  };
  const refreshReadFields = async (fields?: string[]) => {
    if (!fields || fields.length === basis.capabilities.readFields.length && fields.every(f => basis.capabilities.readFields.includes(f))) return false;
    const safe = trimDetailFields(basis, fields, businessUnit.statusReadField);
    setAssistance(undefined);
    setField("");
    clearUnreadInputs(fields);
    setBasis(safe);
    onReloaded(safe);
    await reload();
    return true;
  };
  // Read-only assistance is automatic; it never authorizes a command.
  const assistanceInput = JSON.stringify(input());
  useEffect(() => {
    setAssistance(undefined);
    setAssistanceError("");
    setAssistanceBlocked(false);
    if (action === "CORRECT" && !dateChanged && !classificationChanged || busy || reloading || unknown || conflict || !mayAct || (isNew && !date)) {
      setAssisting(false);
      return;
    }
    const own = ++epoch.current, same = sessionGuard();
    let active = true;
    setAssisting(true);
    const timer = setTimeout(() => {
      void businessUnitApi.preview(detail.id, session.csrf, JSON.parse(assistanceInput)).then(unwrap).then(async response => {
        if (!active || !mounted.current || !same() || epoch.current !== own) return;
        const value = response;
        if (await refreshReadFields(value.readFields)) return;
        setAssistance(value);
      }).catch(async e => {
        if (!active || !mounted.current || !same() || epoch.current !== own) return;
        if (e instanceof ApiError) {
          if (e.code === 'AUTH_REQUEST_VALIDATION_FAILED') { setAssistanceError('登录状态需要刷新，请重试'); return; }
          if (await refreshReadFields(e.readFields ?? (e.status === 403 ? [] : undefined))) return;
          if (e.status === 404) { clear(); onDenied(); return; }
          if (e.status === 403 && e.code !== 'AUTH_REQUEST_VALIDATION_FAILED') { void reload(); return; }
          if (e.status === 409 && ["M02_SOURCE_CONFLICT", "M02_REVISION_CONFLICT"].includes(e.code)) { setConflict(true); setError(flow.unknown ? '' : message(e)); return; }
          // Business diagnostics are visible immediately; other actions keep their direct-save behavior.
          if (e.status === 400 || e.status === 409 || e.status === 422) {
            if(isLifecycle){setError(flow.unknown ? '' : message(e));setField(e.field??"");setAssistanceBlocked(true);}
            return;
          }
        }
        setAssistanceError("操作影响暂时无法加载");
      }).finally(() => {
        if (active && mounted.current && same() && epoch.current === own) setAssisting(false);
      });
    }, 300);
    return () => { active = false; clearTimeout(timer); };
  }, [assistanceInput, assistanceRevision, busy, reloading, unknown, conflict, mayAct]);
  const reload = async () => {
    const edits={names:[...(action==="ADD"?addedNames:isLifecycle?lifecycleNames:correctedNames)],date:dateChanged,classification:classificationChanged};
    const same = sessionGuard();
    const own = ++epoch.current;
    const request = ++reloadRequest.current;
    setReloading(true);
    setAssistance(undefined);
    try {
      const v = unwrap(
        await businessUnitApi.detail(detail.id, { selectedRecordId: action === 'ADD' ? addAnchor : isLifecycle ? lifecycleAnchor : target?.recordId }),
      );
      if (!mounted.current || !same() || own !== epoch.current) return;
      clearUnreadInputs(v.capabilities.readFields);
      if(action==='CORRECT'){
        setValues(previous => ({ ...inputValues(v.selectedRecord), ...Object.fromEntries(edits.names.filter(f => v.capabilities.readFields.includes(f)).map(f => [f, previous[f]])) }));
        if(!edits.date)setDate(v.selectedRecord?.effectiveDate??'');
        if(!edits.classification){setClassificationType(v.selectedRecord?.operationType??'');setClassificationReason(v.selectedRecord?.operationReason??'');}
      }
      if(action==='ADD'){
        const d=v.additionContext?.defaults;
        setValues(previous => ({ ...inputValues(d), ...Object.fromEntries(edits.names.filter(f => v.capabilities.readFields.includes(f)).map(f => [f, previous[f]])) }));
        setAddBaseline(previous=>({...previous,...inputValues(d)}));
      }
      if(isLifecycle){
        const d=v.lifecycleContext?.defaults ?? v.selectedRecord;
        setValues(previous => ({ ...inputValues(d), ...Object.fromEntries(edits.names.filter(f => v.capabilities.readFields.includes(f)).map(f => [f, previous[f]])) }));
        setLifecycleBaseline(previous=>({...previous,...inputValues(d)}));
      }
      setField("");
      setBasis(v);
      onReloaded(v);
      setConflict(false);
      setError(
        capability(v.capabilities,v.additionContext,v.lifecycleContext).allowed
          ? action === "CORRECT" && !v.capabilities.readFields.includes("effectiveDate")
            ? "无法查看生效日期，可修改其他有权字段"
            : ""
          : isLifecycle ? "此处无法完成当前操作，可取消编辑" : "没有操作权限，可取消编辑",
      );
    } catch (e) {
      if (!mounted.current || !same() || own !== epoch.current) return;
      setError(flow.unknown ? '' : message(e));
      if (e instanceof ApiError && [403, 404].includes(e.status)) {
        clear();
        onDenied();
      }
    } finally {
      if (mounted.current && same() && request === reloadRequest.current) setReloading(false);
    }
  };
  const save = async () => {
    if (!flow.attempt && action === 'CORRECT' && !dirty) { setError(''); setNoChanges(true); return; }
    if (!flow.attempt && dateChanged && !date) { setError('请填写生效日期'); setField('newEffectiveDate'); setFocusRequest({ field: 'newEffectiveDate' }); return; }
    if (sessionBlocked || busy || reloading || pendingStop || (!flow.attempt && (pending || conflict || !mayAct))) return;
    const body = input(), expectedSourceVersion = body.expectedSourceVersion;
    const command: RecordCommand = action === 'ADD'
      ? { action, body: { expectedSourceVersion, anchorRecordId: addAnchor, effectiveDate: date, changes: body.changes ?? {}, operationType: 'CHANGE', operationReason: reason as Models['AddInput']['operationReason'] } }
      : action === 'DELETE' ? { action, recordId: target?.recordId ?? '', body: { expectedSourceVersion, expectedRecordRevision: target?.revision ?? '' } }
      : action === 'DEACTIVATE' ? { action, body: { anchorRecordId: lifecycleAnchor, expectedSourceVersion, effectiveDate: date, operationType: action, operationReason: reason as Models['DeactivateInput']['operationReason'] } }
      : action === 'REACTIVATE' ? { action, body: { anchorRecordId: lifecycleAnchor, expectedSourceVersion, effectiveDate: date, operationType: action, operationReason: reason as Models['ReactivateInput']['operationReason'], ...(body.changes ? { changes: body.changes } : {}) } }
      : { action, recordId: target?.recordId ?? '', body: correctionInput() };
    setError('');
    const outcome = await flow.submit(command);
    if (outcome.kind === 'saved') { setTouched([]); setDate(''); setReason(''); onSaved(outcome.result); }
    else if (outcome.kind === 'failed') {
      const e = outcome.error; setError(flow.unknown ? '' : message(e));
      if (e instanceof ApiError) { setField(e.field ?? ''); if (e.field) setFocusRequest({ field: e.field }); }
      if (outcome.definitive) {
        setAssistance(undefined);
        if (e instanceof ApiError && ['M02_SOURCE_CONFLICT', 'M02_REVISION_CONFLICT'].includes(e.code)) setConflict(true);
      }
      if (e instanceof ApiError && e.code === 'AUTH_REQUEST_VALIDATION_FAILED') return;
      if (e instanceof ApiError && !outcome.wasUnknown && await refreshReadFields(e.readFields ?? (e.status === 403 ? [] : undefined))) return;
      if (e instanceof ApiError && e.status === 404) { clear(); onDenied(); }
      else if (e instanceof ApiError && e.status === 403 && e.code !== 'AUTH_REQUEST_VALIDATION_FAILED') {
        if (outcome.wasUnknown) { clear(); onDenied(); } else void reload();
      }
    }
  };
  const blockedNames=action==='CORRECT'?correctedNames.filter(f=>!writes.includes(f)):[];
  const blockedDate=dateChanged&&!basis.capabilities.correction?.facets.includes('REDATE');
  const blockedClassification=classificationChanged&&(!basis.capabilities.correction?.facets.includes('RECLASSIFY')||!basis.capabilities.correction?.classificationTypes.includes(classificationType as Models['Classification']['operationType']));
  const blockedEdits=blockedNames.length>0||blockedDate||blockedClassification;
  const disabled = busy || reloading || unknown || !mayAct;
  useEffect(() => { document.getElementById(`${businessUnit.prefix}-record-editor-title`)?.focus(); }, []);
  const actions = <Stack direction="row" gap={1} useFlexGap flexWrap="wrap" role="group" aria-label="记录编辑操作">
    <Button size="small" variant="contained" disabled={sessionBlocked || disabled || pending || conflict || pendingStop || blockedEdits || assistanceBlocked} onClick={() => void save()}>{saveLabel}</Button>
    <Button size="small" disabled={busy} onClick={() => leave.request(onClose)}>取消编辑</Button>
  </Stack>;
  return (
    <ObjectForm><ObjectDetailSection>
      <Stack spacing={1}>
        <Typography id={`${businessUnit.prefix}-record-editor-title`} component="h2" className="object-section-title" tabIndex={-1}>
          {action === 'CORRECT' ? `更正记录：${businessDate(target?.effectiveDate)}` : title}
        </Typography>
        <ObjectFormError id={`${businessUnit.prefix}-record-form-error`} message={error} issues={field ? [{ id: errorFieldId(field, businessUnit.prefix), message: error }] : []}>
          {reasonError && <Button onClick={() => retryReasons(n => n + 1)}>重试加载操作原因</Button>}
          {conflict && <Button disabled={busy || reloading} onClick={() => void reload()}>重新核对当前资料</Button>}
        </ObjectFormError>
        {unknown && flow.attempt && <Button disabled={sessionBlocked || busy || reloading} onClick={() => void save()}>重试保存</Button>}
        <Box className="object-detail-columns">
          {isDelete && <ObjectDetailSection title="待删除记录">
            {(businessUnit.summaryFields ?? businessUnit.fields).filter(f => can(f.key)).map(f => <ObjectDetailField key={f.key} label={f.label}>{'render' in f && f.render && target ? f.render(target) : target?.values[f.key] ?? '未提供'}</ObjectDetailField>)}
            {can("effectiveDate") && target?.effectiveDate && <ObjectDetailField label="原适用期间">{businessPeriod(target.effectiveDate, target.endExclusive, target.endInclusive)}</ObjectDetailField>}
          </ObjectDetailSection>}
          {!isDelete && action !== "DEACTIVATE" && <ObjectDetailSection title="基本资料">
            <DataEditor session={session} values={values} baseline={action === 'ADD' ? basis.additionContext?.defaults : isLifecycle ? basis.lifecycleContext?.defaults : target} fields={basis.capabilities.readFields} writes={writes}
              disabled={disabled || action === 'CORRECT' && classificationType === 'DEACTIVATE'} prefix="record" fieldError={field} error={error} guidance={basis.nameGuidance}
              onReadDenied={() => void refreshReadFields([])} onReferenceRestricted={() => {
                const safe = trimDetailReferences(basis);
                setBasis(safe); onReloaded(safe); void reload();
              }} context={businessUnit.referenceContext?.(basis.id, input())} onChange={(f, value) => { changed(); mark(f); setValues(v => ({ ...v, [f]: value })); }} />
          </ObjectDetailSection>}
          <ObjectDetailSection title="记录信息">
            {action === 'CORRECT' && can('effectiveDate') && <ObjectTextField id="record-date" label={businessUnitFields.effectiveDate.label} type="date" value={date} disabled={disabled}
              slotProps={{htmlInput:{min:basis.capabilities.correction?.minEffectiveDate,max:basis.capabilities.correction?.maxEffectiveDate,readOnly:!basis.capabilities.correction?.facets.includes('REDATE') || !!basis.capabilities.correction?.maxEffectiveDate && basis.capabilities.correction?.minEffectiveDate===basis.capabilities.correction?.maxEffectiveDate}}}
              onChange={e=>{changed();setDateTouched(true);setDate(e.target.value)}} error={field==='newEffectiveDate'} helperText={field==='newEffectiveDate'?error:undefined}/>}
            {isNew ? (
              <>
                {(isLifecycle || can('effectiveDate')) && <ObjectTextField
                  id="record-date"
                  label={businessUnitFields.effectiveDate.label}
                  type="date"
                  required
                  slotProps={{htmlInput:{min:action==='ADD'?basis.additionContext?.minEffectiveDate:basis.lifecycleContext?.minEffectiveDate,max:action==='ADD'?basis.additionContext?.maxEffectiveDate:basis.lifecycleContext?.maxEffectiveDate}}}
                  value={date}
                  disabled={disabled}
                  onChange={(e) => {
                    changed();
                    setDate(e.target.value);
                  }}
                  error={field === "effectiveDate"}
                  helperText={field === 'effectiveDate' ? error : undefined}
                />}
                <ObjectTextField
                  id="record-reason"
                  label={businessUnitFields.operationReason.label}
                  select
                  required
                  value={reason}
                  disabled={disabled}
                  onChange={(e) => {
                    changed();
                    setReason(e.target.value);
                  }}
                  error={field === 'operationReason'}
                  helperText={field === 'operationReason' ? error : undefined}
                >
                  <MenuItem value="">请选择</MenuItem>
                  {reasons.map(item => <MenuItem key={item.code} value={item.code}>{item.name}</MenuItem>)}
                </ObjectTextField>
                <ObjectDetailField label={businessUnitFields.operationType.label}>{action === "ADD" ? "数据更改" : action === "DEACTIVATE" ? "停用" : "重新启用"}</ObjectDetailField>
              </>
            ) : (
              <>
                {action !== "CORRECT" && can("effectiveDate") && <ObjectDetailField label={businessUnitFields.effectiveDate.label}>{businessDate(target?.effectiveDate)}</ObjectDetailField>}
                {!showClassificationInputs && can('operationType') && <ObjectDetailField label={action === 'CORRECT' ? '操作类型' : '原操作类型'}>{target?.typeLabel}</ObjectDetailField>}
                {!showClassificationInputs && can('operationReason') && <ObjectDetailField label={action === 'CORRECT' ? '操作原因' : '原操作原因'}>{target?.reasonLabel}</ObjectDetailField>}
                {showClassificationInputs && <>
                    {basis.capabilities.correction?.classificationTypes.length === 1 && basis.capabilities.correction.classificationTypes[0] === classificationType
                      ? <ObjectDetailField label={businessUnitFields.operationType.label}>{target?.typeLabel}</ObjectDetailField>
                      : <ObjectTextField id="record-classification-type" label={businessUnitFields.operationType.label} select value={classificationType} disabled={disabled||!canReclassify} onChange={e=>{
                      if(e.target.value===classificationType) return;
                      if(e.target.value==='DEACTIVATE' && correctedNames.length){setPendingStop(true);return;}
                      changed();setClassificationTouched(true);setClassificationType(e.target.value);setClassificationReason('');
                    }}>{classificationChanged && !basis.capabilities.correction?.classificationTypes.includes(classificationType as Models['Classification']['operationType']) && <MenuItem value={classificationType}>{classificationType === target?.operationType ? target?.typeLabel : operationTypeLabels[classificationType]}</MenuItem>}{basis.capabilities.correction?.classificationTypes.map(t=><MenuItem key={t} value={t}>{t===target?.operationType ? target.typeLabel : operationTypeLabels[t]}</MenuItem>)}</ObjectTextField>}
                    <ObjectTextField id="record-classification-reason" label={businessUnitFields.operationReason.label} select value={classificationReason} disabled={disabled||!canReclassify} error={field==='classification.operationReason'} helperText={field==='classification.operationReason'?error:undefined} onChange={e=>{changed();setClassificationTouched(true);setClassificationReason(e.target.value)}}>
                      <MenuItem value="">请选择</MenuItem>
                      {classificationType===target?.operationType && target?.operationReason && !reasons.some(r=>r.code===target.operationReason) && <MenuItem value={target.operationReason}>{target.reasonLabel}</MenuItem>}
                      {reasons.map(r=><MenuItem key={r.code} value={r.code}>{classificationType===target?.operationType && r.code===target?.operationReason ? target.reasonLabel : r.name}</MenuItem>)}
                    </ObjectTextField>
                </>}
              </>
            )}
          </ObjectDetailSection>
        </Box>
        {blockedEdits && <Alert severity="warning">部分字段的修改权限已变化
          <Button disabled={busy||reloading||unknown} onClick={()=>{changed();setValues(v=>({...v,...Object.fromEntries(blockedNames.map(f=>[f,target?.values[f]??'']))}));setTouched(v=>v.filter(f=>!blockedNames.includes(f)));if(blockedDate){setDate(target?.effectiveDate??'');setDateTouched(false);}if(blockedClassification){setClassificationType(target?.operationType??'');setClassificationReason(target?.operationReason??'');setClassificationTouched(false);}}}>撤销这些字段的修改</Button></Alert>}
        {pendingStop && <Alert severity="warning">转为停用须移除{businessUnit.prefix === 'bu' ? '名称' : '资料'}修改。
          <Button disabled={disabled} onClick={()=>{changed();setValues(inputValues(target));setTouched([]);setClassificationTouched(true);setClassificationType('DEACTIVATE');setClassificationReason('');setPendingStop(false)}}>移除{businessUnit.prefix === 'bu' ? '名称' : '资料'}修改并转为停用</Button>
          <Button disabled={disabled} onClick={()=>setPendingStop(false)}>保留当前输入</Button></Alert>}
        {noChanges && <Typography variant="body2" role="status">没有修改内容。</Typography>}
        {assisting && <Typography variant="body2" role="status">正在读取操作影响…</Typography>}
        {assistanceError && <Alert severity="error" action={<Button onClick={() => retryAssistance(n => n + 1)}>重试加载</Button>}>{assistanceError}</Alert>}
        {assistance && (isDelete || action === 'CORRECT' && dateChanged) && <Box aria-label="操作影响">
          {assistance.previousCurrentStatus !== assistance.currentStatus && <Typography variant="body2">今日状态：{assistance.previousCurrentStatus === 'inactive' ? '停用' : assistance.previousCurrentStatus === 'active' ? '启用' : '未生效'} → {assistance.currentStatus === 'inactive' ? '停用' : assistance.currentStatus === 'active' ? '启用' : '未生效'}</Typography>}
          {isDelete && assistance.afterRecord && <Typography variant="body2">删除后适用资料：{assistance.afterRecord.name ?? '无读取权限'}{assistance.afterRecord.values.nameEn ? ` / ${assistance.afterRecord.values.nameEn}` : ''}{can('effectiveDate') ? `；${businessDate(assistance.afterRecord.effectiveDate)}` : ''}</Typography>}
          {can("effectiveDate") && assistance.newNoFactRange && <Typography variant="body2">无适用资料：{businessPeriod(assistance.newNoFactRange.start, assistance.newNoFactRange.endExclusive, assistance.newNoFactRange.endInclusive)}</Typography>}
          {action === "CORRECT" && dateChanged && can("effectiveDate") && <Typography variant="body2">生效日期：{businessDate(target?.effectiveDate)} → {businessDate(date)}</Typography>}
        </Box>}
        {actionsContainer && createPortal(actions, actionsContainer)}
      </Stack>
      <UnsavedChangesDialog pending={unknown || pending} {...leave.dialog} />
    </ObjectDetailSection></ObjectForm>
  );
}
