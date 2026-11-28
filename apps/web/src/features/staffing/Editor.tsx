import { createPortal } from 'react-dom';
import { useEffect, useMemo, useRef, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  MenuItem,
  Stack,
  Typography,
} from '@mui/material';
import {
  ObjectForm,
  ObjectTextField,
  ObjectDetailSection,
  ObjectDetailField,
  ObjectFormError,
  focusObjectField,
  UnsavedChangesDialog,
  businessDate,
} from '@chnhcm/ui';
import { type Models, sessionGuard, ApiError } from '@chnhcm/api-client';
import { useDraftExit } from '../../shared/dated/useDraftExit';
import { useStaffingSubmission } from './useSubmission';
import * as staff from './api';
import { JobClassificationReference } from './JobClassificationReference';
import { PositionReference } from './PositionReference';
import { GradeMemberList, GradeMembersEditor } from './GradeMembers';
export type StaffingDraft = {
  values: Record<string, string>;
  code?: string;
  date?: string;
  stop?: string;
  start?: string;
  type?: string;
  reason?: string;
  stopReason?: string;
  startReason?: string;
  reasonTouched?: boolean;
};
export function permittedDraft(draft: StaffingDraft, fields: string[]): StaffingDraft {
  return {
    values: Object.fromEntries(Object.entries(draft.values).filter(([key]) => fields.includes(key))),
    ...(fields.includes('code') && { code: draft.code }),
    ...(fields.includes('effectiveDate') && { date: draft.date, stop: draft.stop, start: draft.start }),
    ...(fields.includes('operationType') && { type: draft.type }),
    ...(fields.includes('operationReason') && {
      reason: draft.reason, stopReason: draft.stopReason, startReason: draft.startReason, reasonTouched: draft.reasonTouched,
    }),
  };
}
type Props = {
  draft?: StaffingDraft;
  kind: staff.Kind;
  session: staff.Principal;
  action: string;
  detail?: staff.Detail;
  context: Models['StaffingContext'];
  actionsContainer?: HTMLDivElement | null;
  onClose: () => void;
  onSaved: (r: staff.Result) => void;
  onAccessChanged: (cause: 'denied' | 'period', draft?: StaffingDraft) => void;
  onGradesRestricted: () => void;
};
const text = (r: staff.RecordView | undefined, k: string) => String(r?.[k as keyof staff.RecordView] ?? '');
export function StaffingEditor({
  kind: k,
  draft,
  session,
  action,
  detail,
  context,
  onClose,
  onSaved,
  actionsContainer,
  onAccessChanged,
  onGradesRestricted,
}: Props) {
  const pair = action.includes('LIFECYCLE_PAIR'),
    maintain = action === 'MAINTAIN_LIFECYCLE_PAIR',
    deleting = action === 'DELETE' || action === 'DELETE_LIFECYCLE_PAIR';
  const baseline = maintain ? detail?.pairContext?.reactivation : detail?.selectedRecord;
  const fields = (detail?.capabilities ?? context.capabilities).actionFields[action] ?? [];
  const [values, setValues] = useState<Record<string, string>>(() =>
      Object.fromEntries(staff.dataFields(k).map((f) => [f, draft?.values[f] ?? text(baseline, f)])),
    ),
    [touched, setTouched] = useState<string[]>(Object.keys(draft?.values ?? {})),
    [code, setCode] = useState(draft?.code ?? '');
  const [gradeIds, setGradeIds] = useState<string[]>(baseline?.gradeIds ?? []);
  const [gradesTouched, setGradesTouched] = useState(false);
  const [gradesRestricted, setGradesRestricted] = useState(baseline?.gradesRestricted ?? false);
  const gradesDirty = gradesTouched && JSON.stringify(gradeIds) !== JSON.stringify(baseline?.gradeIds ?? []);
  const initialDate =
    action === 'CREATE'
      ? context.defaultEffectiveDate
      : action === 'ADD'
        ? (detail?.additionContext?.defaultEffectiveDate ?? '')
        : action === 'DEACTIVATE' || action === 'REACTIVATE'
          ? (detail?.lifecycleContext?.defaultEffectiveDate ?? '')
          : (baseline?.effectiveDate ?? '');
  const initialStop = (maintain ? detail?.pairContext?.deactivation.effectiveDate : detail?.pairInsertion?.defaultStopDate) ?? '';
  const initialStart = (maintain ? detail?.pairContext?.reactivation.effectiveDate : detail?.pairInsertion?.defaultResumeDate) ?? '';
  const [date, setDate] = useState(draft?.date ?? initialDate),
    [stop, setStop] = useState(draft?.stop ?? initialStop),
    [start, setStart] = useState(draft?.start ?? initialStart);
  const initialType =
    action === 'CREATE'
      ? 'CREATE'
      : action === 'ADD'
        ? 'CHANGE'
        : action === 'REACTIVATE' || action === 'DEACTIVATE'
          ? action
          : (baseline?.operationType ?? 'CHANGE');
  const [type, setType] = useState(draft?.type ?? initialType),
    [reason, setReason] = useState(draft?.reason ?? (action === 'CORRECT' ? (baseline?.operationReason ?? '') : '')),
    [stopReason, setStopReason] = useState(draft?.stopReason ?? (maintain ? (detail?.pairContext?.deactivation.operationReason ?? '') : '')),
    [startReason, setStartReason] = useState(draft?.startReason ?? (maintain ? (baseline?.operationReason ?? '') : ''));
  const [reasons, setReasons] = useState<Models['StaffingReasons']['items']>([]),
    [stopReasons, setStopReasons] = useState<Models['StaffingReasons']['items']>([]),
    [startReasons, setStartReasons] = useState<Models['StaffingReasons']['items']>([]),
    [loadedReasonKey, setLoadedReasonKey] = useState(''),
    [reasonError, setReasonError] = useState(''),
    [retry, setRetry] = useState(0);
  const [fieldError, setFieldError] = useState(''),
    [reasonTouched, setReasonTouched] = useState(draft?.reasonTouched ?? false);
  const [error, setError] = useState(''),
    [preview, setPreview] = useState<Models['StaffingPreview']>(),
    [previewBusy, setPreviewBusy] = useState(false);
  const reasonGeneration = useRef(0),
    saved = useRef(false),
    form = useRef<HTMLFormElement>(null);
  const scalarDraft = (): StaffingDraft => ({
    values: Object.fromEntries(touched.map(key => [key, values[key]])),
    code, date, stop, start, type, reason, stopReason, startReason, reasonTouched,
  });
  const period = JSON.stringify([date, stop, start, type]);
  const lastPeriod = useRef(period);
  useEffect(() => {
    const changed = lastPeriod.current !== period;
    lastPeriod.current = period;
    if (!changed || !gradesRestricted || !fields.includes('gradeIds')) return;
    if (action === 'CREATE') {
      setGradesRestricted(false);
    } else {
      onAccessChanged('period', scalarDraft());
    }
  }, [period, gradesRestricted]);
  const finish = (r: staff.Result) => {
    if (r.outcome === 'accepted' || r.outcome === 'no_change') {
      saved.current = true;
      onSaved(r);
    } else if (r.failureStatus === 403) {
      onAccessChanged('denied', scalarDraft());
    } else {
      setError(
        r.outcome === 'not_applied' ? '原提交未生效，请核对输入后重新保存。' : (r.failure?.message ?? '原提交已拒绝。'),
      );
    }
  };
  const submission = useStaffingSubmission(k, session, detail?.id ?? '', finish),
    flow = submission.flow;
  const dirty =
    code !== '' || gradesDirty ||
    touched.some((f) => values[f] !== text(baseline, f)) ||
    date !== initialDate ||
    type !== initialType ||
    reasonTouched ||
    stop !== initialStop ||
    start !== initialStart;
  const exit = useDraftExit(
    dirty || flow.unknown || flow.inFlight,
    () => saved.current,
    () => {
      flow.clearPayload();
      setValues({});
      setGradeIds([]);
      setCode('');
    },
  );
  const body = useMemo(() => {
    const changes: Models['StaffingChangesInput'] = {};
    if (gradesDirty && fields.includes('gradeIds') && !gradesRestricted) changes.gradeIds = gradeIds;
    for (const key of touched) {
      if (fields.includes(key)) (changes as Record<string, string | null>)[key] = values[key] === '' ? null : values[key];
    }
    if (action === 'CREATE') {
      const v: staff.Input = { action, code, effectiveDate: date, operationType: 'CREATE', operationReason: reason };
      for (const f of staff.dataFields(k)) if (fields.includes(f) && (f !== 'gradeId' || values[f])) (v as Record<string, unknown>)[f] = values[f];
      if ((k === 'job' || k === 'gradeladder') && fields.includes('gradeIds') && !gradesRestricted) v.gradeIds = gradeIds;
      return v;
    }
    const v: staff.Input = { action, expectedSourceVersion: detail!.headSourceVersion };
    if (['CORRECT', 'REDATE', 'DELETE'].includes(action)) {
      v.recordId = baseline?.recordId;
      v.expectedRecordRevision = baseline?.revision;
      if (action === 'REDATE' || date !== baseline?.effectiveDate) v.newEffectiveDate = date;
      if (action === 'CORRECT') {
        v.changes = changes;
        if (type !== baseline?.operationType || reason !== baseline?.operationReason)
          v.classification = { operationType: type, operationReason: reason };
      }
    } else if (pair) {
      if (action === 'INSERT_LIFECYCLE_PAIR') v.anchorRecordId = detail?.selectedRecord.recordId;
      else {
        v.deactivationRecordId = detail?.pairContext?.deactivation.recordId;
        v.deactivationRevision = detail?.pairContext?.deactivation.revision;
        v.reactivationRecordId = detail?.pairContext?.reactivation.recordId;
        v.reactivationRevision = detail?.pairContext?.reactivation.revision;
      }
      if (!deleting) {
        v.deactivationDate = stop;
        v.reactivationDate = start;
        v.deactivationReason = stopReason;
        v.reactivationReason = startReason;
        v.changes = changes;
      }
    } else {
      v.anchorRecordId = detail?.selectedRecord.recordId;
      v.effectiveDate = date;
      v.operationType = action === 'ADD' ? 'CHANGE' : action;
      v.operationReason = reason;
      if (action !== 'DEACTIVATE') v.changes = changes;
    }
    return v;
  }, [k, action, detail, code, date, type, reason, values, touched, stop, start, stopReason, startReason, gradeIds, gradesDirty, gradesRestricted, fields]);
  const signature = JSON.stringify(body);
  useEffect(() => setPreview(undefined), [signature]);
  const needsReasons = !deleting && action !== 'REDATE' && (pair || fields.includes('operationReason'));
  const reasonKey = JSON.stringify([k, action, type, retry]);
  const reasonsPending = needsReasons && loadedReasonKey !== reasonKey;
  useEffect(() => {
    if (deleting || action === 'REDATE') return;
    const a = new AbortController(),
      own = ++reasonGeneration.current;
    setReasonError('');
    setLoadedReasonKey('');
    if (pair) {
      void Promise.all([staff.reasons(k, 'DEACTIVATE', a.signal), staff.reasons(k, 'REACTIVATE', a.signal)])
        .then(([x, y]) => {
          if (a.signal.aborted || own !== reasonGeneration.current) return;
          setStopReasons(x.items);
          setStartReasons(y.items);
          if (!stopReason && x.items.length === 1) setStopReason(x.items[0].code);
          if (!startReason && y.items.length === 1) setStartReason(y.items[0].code);
          setLoadedReasonKey(reasonKey);
        })
        .catch(() => {
          if (!a.signal.aborted) setReasonError('业务原因加载失败，请重试。');
        });
    } else {
      void staff
        .reasons(k, type, a.signal)
        .then((v) => {
          if (a.signal.aborted || own !== reasonGeneration.current) return;
          setReasons(v.items);
          if (!reason && v.items.length === 1) setReason(v.items[0].code);
          setLoadedReasonKey(reasonKey);
        })
        .catch(() => {
          if (!a.signal.aborted) setReasonError('业务原因加载失败，请重试。');
        });
    }
    return () => a.abort();
  }, [k, type, pair, deleting, action, reasonKey]);
  const requestPreview = async () => {
    if (!detail) return;
    const same = sessionGuard();
    setPreviewBusy(true);
    setError('');
    try {
      const v = await staff.preview(k, session, detail.id, body);
      if (same()) setPreview(v);
    } catch (e) {
      if (same()) setError(e instanceof Error ? e.message : '预览失败');
    } finally {
      if (same()) setPreviewBusy(false);
    }
  };
  const submit = async () => {
    setError('');
    setFieldError('');
    if (reasonsPending || reasonError || !form.current?.reportValidity()) return;
    const v = await flow.submit(body);
    if (v.kind === 'saved') finish(v.result);
    else if (v.kind === 'failed') {
      setError(v.error instanceof Error ? v.error.message : '暂时无法确认提交结果，请查看保存结果。');
      if (v.error instanceof ApiError) {
        if ([401, 403].includes(v.error.status)) {
          onAccessChanged('denied', v.error.status === 403 ? scalarDraft() : undefined);
          return;
        }
        const field = (v.error.field ?? '').replace('changes.', '').replace('newEffectiveDate', 'effectiveDate');
        setFieldError(field);
        requestAnimationFrame(() => focusObjectField(`staffing-${field}`, 'staffing-error'));
      }
    }
  };
  useEffect(() => {
    if (deleting) void requestPreview();
  }, [signature]);
  const disabled = flow.inFlight || flow.unknown || submission.sessionBlocked;
  const readOnlyData = deleting || action === 'DEACTIVATE' || action === 'REDATE' || type === 'DEACTIVATE';
  const formId = `staffing-${k}-form`;
  const actions = (
    <>
      <Button
        variant="contained"
        type="submit"
        form={formId}
        disabled={disabled || reasonsPending || !!reasonError || (submission.pending && !flow.attempt)}
      >
        {flow.inFlight ? '正在保存…' : deleting ? '确认删除' : '保存'}
      </Button>
      <Button onClick={() => exit.request(onClose)}>取消</Button>
    </>
  );
  const dataInputs = (
    <>
      {action === 'CREATE' && fields.includes('code') && (
        <ObjectTextField
          id="staffing-code"
          error={fieldError === 'code'}
          helperText={fieldError === 'code' ? error : undefined}
          label="编码"
          required
          inputProps={{ maxLength: 30 }}
          value={code}
          disabled={disabled}
          onChange={(e) => setCode(e.target.value)}
        />
      )}
      {staff
        .dataFields(k)
        .filter((f) => fields.includes(f) || (readOnlyData && detail?.capabilities.readFields.includes(f)))
        .map((f) =>
          readOnlyData ? (
            <ObjectDetailField key={f} label={staff.fieldLabels[f]}>
              {staff.isPositionField(f) ? staff.positionName(baseline,f) ?? '未指定' : staff.isJobClassification(f) ? staff.classificationName(baseline, f) ?? '受限' : text(baseline, f) || '未提供'}
            </ObjectDetailField>
          ) : staff.isPositionField(f) ? (
            <PositionReference key={f} field={f} body={{...body,objectId:detail?.id}} session={session} id={values[f]} original={baseline} disabled={disabled}
              onDenied={() => { const draft = scalarDraft(); draft.values = Object.fromEntries(Object.entries(draft.values).filter(([key]) => !staff.isPositionField(key))); onAccessChanged('denied', draft); }}
              error={fieldError === f ? error : undefined} onChange={id => {
                if (fieldError === f && values[f] !== id) { setFieldError(''); setError(''); }
                setValues(previous => ({ ...previous, [f]: id }));
                setTouched(previous => [...new Set([...previous, f])]);
              }} />
          ) : staff.isJobClassification(f) ? (
            <JobClassificationReference
              key={f}
              field={f}
              session={session}
              body={{ ...body, objectId: detail?.id }}
              value={values[f]}
              originalLabel={values[f] === (baseline?.[f] ?? '') ? staff.classificationName(baseline, f) : undefined}
              required={f === 'jobFamilyId' || ['CREATE', 'ADD', 'REACTIVATE', 'INSERT_LIFECYCLE_PAIR'].includes(action)}
              disabled={disabled}
              error={fieldError === f ? error : undefined}
              onChange={(id) => {
                if (fieldError === f) { setFieldError(''); setError(''); }
                setValues(previous => ({ ...previous, [f]: id }));
                setTouched(previous => [...new Set([...previous, f])]);
              }}
            />
          ) : (
            <ObjectTextField
              key={f}
              id={`staffing-${f}`}
              label={staff.fieldLabels[f]}
              required={f === 'name'}
              error={fieldError === f}
              helperText={fieldError === f ? error : undefined}
              value={values[f] ?? ''}
              disabled={disabled}
              multiline={['description', 'responsibilities', 'requirements'].includes(f)}
              minRows={['description', 'responsibilities', 'requirements'].includes(f) ? 3 : undefined}
              onChange={(e) => {
                setValues((v) => ({ ...v, [f]: e.target.value }));
                setTouched((v) => [...new Set([...v, f])]);
              }}
            />
          ),
        )}
      {(k === 'job' || k === 'gradeladder') && (fields.includes('gradeIds') || (detail?.capabilities ?? context.capabilities).readFields.includes('gradeIds')) && (
        readOnlyData || !fields.includes('gradeIds') ? <ObjectDetailField label={k === 'job' ? '可用职级' : '职级成员（低至高）'}><GradeMemberList members={baseline?.gradeMembers} restricted={baseline?.gradesRestricted} ordered={k === 'gradeladder'} /></ObjectDetailField> :
        <GradeMembersEditor kind={k} session={session} ids={gradeIds} disabled={disabled} restricted={gradesRestricted}
          body={action === 'CREATE' ? { ...body, gradeIds, objectId: detail?.id } : { ...body, changes: { ...body.changes, gradeIds }, objectId: detail?.id }}
          error={fieldError === 'gradeIds' ? error : undefined}
          onChange={ids => { setGradeIds(ids); setGradesTouched(true); }}
          onRestricted={() => { setGradeIds([]); setGradesTouched(false); setGradesRestricted(true); flow.clearPayload(); onGradesRestricted(); }} />
      )}
    </>
  );
  const content = (
    <ObjectForm>
      <form
        id={formId}
        ref={form}
        onSubmit={(e) => {
          e.preventDefault();
          void submit();
        }}
      >
        <ObjectFormError
          id="staffing-error"
          message={error}
          issues={fieldError ? [{ id: `staffing-${fieldError}`, message: error }] : []}
        />
        {reasonError && (
          <Alert severity="error" action={<Button onClick={() => setRetry((n) => n + 1)}>重试</Button>}>
            {reasonError}
          </Alert>
        )}
        {deleting && (
          <Alert severity="warning">
            {pair
              ? '将删除这两条停用与重新启用记录。'
              : `将删除${baseline?.effectiveDate ? businessDate(baseline.effectiveDate) : '所选'}记录。`}
            {preview && preview.changed ? '相邻记录的适用期间将相应调整。' : ''}
          </Alert>
        )}
        <Box className="object-detail-columns">
          <ObjectDetailSection title={pair && !deleting ? '重新启用资料' : '基本资料'}>
            {dataInputs}
          </ObjectDetailSection>
          <ObjectDetailSection title="记录信息">
            {!deleting && !pair && fields.includes('effectiveDate') && (
              <ObjectTextField
                id="staffing-effectiveDate"
                error={fieldError === 'effectiveDate'}
                helperText={fieldError === 'effectiveDate' ? error : undefined}
                label="生效日期"
                type="date"
                required
                value={date}
                disabled={disabled}
                onChange={(e) => setDate(e.target.value)}
              />
            )}
            {!deleting &&
              !pair &&
              action !== 'REDATE' &&
              (action === 'CORRECT' && fields.includes('operationType') ? (
                <ObjectTextField
                  select
                  label="操作类型"
                  value={type}
                  disabled={disabled}
                  onChange={(e) => {
                    setType(e.target.value);
                    setReason('');
                  }}
                >
                  {(baseline?.operationType === 'CREATE' ? ['CREATE'] : ['CHANGE', 'DEACTIVATE', 'REACTIVATE']).map(
                    (v) => (
                      <MenuItem key={v} value={v}>
                        {staff.operationLabel[v]}
                      </MenuItem>
                    ),
                  )}
                </ObjectTextField>
              ) : (
                <ObjectDetailField label="操作类型">{staff.operationLabel[initialType]}</ObjectDetailField>
              ))}
            {!deleting && !pair && action !== 'REDATE' && fields.includes('operationReason') && (
              <ObjectTextField
                id="staffing-operationReason"
                select
                label="操作原因"
                required
                value={reason}
                disabled={disabled || reasonsPending}
                onChange={(e) => {
                  setReasonTouched(true);
                  setReason(e.target.value);
                }}
              >
                <MenuItem value="">请选择原因</MenuItem>
                {reasons.map((r) => (
                  <MenuItem key={r.code} value={r.code}>
                    {r.label}
                  </MenuItem>
                ))}
              </ObjectTextField>
            )}
            {!deleting && pair && (
              <>
                <ObjectTextField
                  id="staffing-deactivationDate"
                  label="停用日期"
                  type="date"
                  required
                  value={stop}
                  disabled={disabled}
                  onChange={(e) => setStop(e.target.value)}
                />
                <ObjectTextField
                  select
                  label="停用原因"
                  required
                  value={stopReason}
                  disabled={disabled || reasonsPending}
                  onChange={(e) => {
                    setReasonTouched(true);
                    setStopReason(e.target.value);
                  }}
                >
                  {stopReasons.map((r) => (
                    <MenuItem key={r.code} value={r.code}>
                      {r.label}
                    </MenuItem>
                  ))}
                </ObjectTextField>
                <ObjectTextField
                  id="staffing-reactivationDate"
                  label="重新启用日期"
                  type="date"
                  required
                  value={start}
                  disabled={disabled}
                  onChange={(e) => setStart(e.target.value)}
                />
                <ObjectTextField
                  select
                  label="重新启用原因"
                  required
                  value={startReason}
                  disabled={disabled || reasonsPending}
                  onChange={(e) => {
                    setReasonTouched(true);
                    setStartReason(e.target.value);
                  }}
                >
                  {startReasons.map((r) => (
                    <MenuItem key={r.code} value={r.code}>
                      {r.label}
                    </MenuItem>
                  ))}
                </ObjectTextField>
              </>
            )}
          </ObjectDetailSection>
        </Box>
        {flow.unknown && <Alert severity="warning">保存结果尚未确认，请使用“查看保存结果”。</Alert>}
      </form>
    </ObjectForm>
  );
  return (
    <>
      {pair || deleting ? (
        <Dialog
          open
          fullWidth
          maxWidth="md"
          onClose={() => exit.request(onClose)}
          aria-labelledby="staffing-operation-title"
        >
          <DialogTitle id="staffing-operation-title">{staff.operationLabel[action]}</DialogTitle>
          <DialogContent>{content}</DialogContent>
          <DialogActions sx={{ flexWrap: 'wrap' }}>{actions}</DialogActions>
        </Dialog>
      ) : (
        <>
          {actionsContainer && createPortal(actions, actionsContainer)}
          {content}
        </>
      )}
      <UnsavedChangesDialog pending={flow.unknown || flow.inFlight} {...exit.dialog} />
    </>
  );
}
