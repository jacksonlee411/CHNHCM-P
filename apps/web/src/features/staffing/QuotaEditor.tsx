import { useEffect, useMemo, useReducer, useRef, useState } from 'react';
import { Box, Button } from '@mui/material';
import WorkOutline from '@mui/icons-material/WorkOutline';
import { ApiError, sessionGuard, type Models } from '@chnhcm/api-client';
import {
  ObjectDetailField,
  ObjectDetailIdentity,
  ObjectDetailSection,
  ObjectDetailSheet,
  ObjectForm,
  ObjectFormError,
  focusObjectField,
  ObjectTextField,
  RecordControl,
  UnsavedChangesDialog,
} from '@chnhcm/ui';
import { SubmissionFlow } from '../../shared/dated/submission';
import { useDraftExit } from '../../shared/dated/useDraftExit';
import { useSessionRecovery } from '../../shared/sessionRecovery';
import {
  departmentRecordLabels,
  labels,
  period,
  previewQuota,
  quotaLabel,
  quotaAmountFields,
  receipts,
  resolveQuota,
  saveQuota,
  type Action,
  type Command,
  type Position,
  type RecordView,
  type Result,
} from './quotaApi';

export type QuotaDraft = { amount?: string; day?: string; reason?: string; denied?: boolean };
export function permittedQuotaDraft(draft: QuotaDraft, action: Action, view: Partial<Position>): QuotaDraft {
  return {
    amount:
      action === 'create' ||
      action === 'adjust' ||
      (action === 'correct' && view.correctFields?.includes('quota'))
        ? draft.amount
        : undefined,
    reason:
      ['create', 'adjust', 'withdraw'].includes(action) ||
      (action === 'correct' && view.correctFields?.includes('reason'))
        ? draft.reason
        : undefined,
    day: ['create', 'adjust', 'withdraw', 'redate'].includes(action) ? draft.day : undefined,
    denied: draft.denied,
  };
}
export function QuotaEditor({
  view,
  record,
  action,
  session,
  close,
  done,
  draft,
  reread,
}: {
  view: Partial<Position> & { targetKind?: 'department' };
  record: RecordView;
  action: Action;
  session: Models['Principal'];
  close: () => void;
  done: (result: Result, date: string) => void;
  draft?: QuotaDraft;
  reread: (input: QuotaDraft, denied: boolean) => void;
}) {
  const target = view.targetKind ?? 'position';
  const amountLabel = quotaAmountFields[target];
  const initialDate =
    action === 'create'
      ? '1900-01-01'
      : action === 'redate' || action === 'correct' || action === 'delete'
        ? (record.effectiveDate ?? '')
        : (view.businessAt ?? '');
  const initialAmount = action === 'create' ? '' : String(record.quota ?? '');
  const initialReason = action === 'correct' ? (record.reason ?? '') : '';
  const [day, setDay] = useState(draft?.day ?? initialDate),
    [amount, setAmount] = useState(draft?.amount ?? initialAmount),
    [reason, setReason] = useState(draft?.reason ?? initialReason),
    [error, setError] = useState(draft?.denied ? '权限已变化，请核对仍可办理的字段。' : ''),
    [suspended, setSuspended] = useState(!!draft?.denied),
    [canReload, setCanReload] = useState(false),
    [field, setField] = useState(''),
    [proposed, setProposed] = useState<RecordView>();
  const mounted = useRef(false),
    [, render] = useReducer((n) => n + 1, 0),
    blocked = useSessionRecovery();
  const [flow] = useState(
    () =>
      new SubmissionFlow<Command, Result, Result>({
        owner: session,
        channel: receipts,
        uuid: () => crypto.randomUUID(),
        changed: render,
        current: () => {
          const same = sessionGuard();
          return () => mounted.current && same();
        },
        send: (c, id) => saveQuota(c, session.csrf, id),
        unwrap: (r) => r,
        resolve: (r) => resolveQuota(session.csrf, r),
        definitive: (e) => e instanceof ApiError && e.status >= 400 && e.status < 500 && e.status !== 408,
      }),
  );
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      flow.clearPayload();
    };
  }, [flow]);
  useEffect(
    () =>
      receipts.subscribe(
        ({ reference, result }) => {
          if (flow.resolved(reference, result)) {
            if (flow.saved) done(result, day);
            else if (result.failureStatus === 403 || result.failureStatus === 404) refresh(true);
            else setError(result.failure?.message ?? '原提交未生效');
          }
        },
        () => {},
      ),
    [flow, day, amount, reason, done],
  );
  const dirty = amount !== initialAmount || reason !== initialReason || day !== initialDate;
  const exit = useDraftExit(dirty, () => flow.saved, close);
  const editableAmount =
    action === 'create' ||
    action === 'adjust' ||
    (action === 'correct' && record.quotaState === 'set' && view.correctFields?.includes('quota'));
  const editableReason =
    ['create', 'adjust', 'withdraw'].includes(action) ||
    (action === 'correct' && view.correctFields?.includes('reason'));
  function refresh(denied: boolean) {
    flow.clearPayload();
    reread(
      {
        amount: amount !== initialAmount ? amount : undefined,
        reason: reason !== initialReason ? reason : undefined,
        day: day !== initialDate ? day : undefined,
        denied,
      },
      denied,
    );
  }
  function failed(e: unknown, fromSave = false) {
    if (e instanceof ApiError && [401, 403, 404].includes(e.status)) {
      refresh(true);
      return;
    }
    const message = e instanceof Error ? e.message : '暂时无法确认保存结果';
    setError(message);
    setCanReload(e instanceof ApiError && e.status === 409);
    const id = e instanceof ApiError && e.field ? e.field : '';
    setField(id);
    if (fromSave) focusObjectField(id ? 'quota-' + id : 'quota-error', 'quota-error');
  }
  const amountInvalid =
    editableAmount && (amount.trim() === '' || !Number.isSafeInteger(Number(amount)) || Number(amount) < 0);
  const reasonInvalid = editableReason && !reason.trim();
  const command = useMemo((): Command | undefined => {
    const quota = Number(amount);
    if (amountInvalid || reasonInvalid || (action === 'redate' && !day)) return;
    if ((action === 'create' || action === 'adjust' || action === 'withdraw') && (!day || !reason.trim()))
      return;
    if (action === 'create' && !view.positionId) return;
    if (action === 'create')
      return { action, body: { positionId: view.positionId!, quota, effectiveDate: day, reason } };
    const quotaId = view.quotaId,
      expectedSourceVersion = view.sourceVersion;
    if (!quotaId || !expectedSourceVersion) return;
    if (action === 'adjust')
      return { action, quotaId, body: { expectedSourceVersion, quota, effectiveDate: day, reason } };
    if (action === 'withdraw')
      return { action, quotaId, body: { expectedSourceVersion, effectiveDate: day, reason } };
    const recordId = record.recordId,
      expectedRecordRevision = record.revision;
    if (!recordId || !expectedRecordRevision) return;
    const base = { expectedSourceVersion, expectedRecordRevision };
    if (action === 'redate') return { action, quotaId, recordId, body: { ...base, newEffectiveDate: day } };
    if (action === 'delete') return { action, quotaId, recordId, body: base };
    return {
      action,
      quotaId,
      recordId,
      body: {
        ...base,
        ...(editableAmount && amount !== initialAmount ? { quota } : {}),
        ...(editableReason && reason !== initialReason ? { reason } : {}),
      },
    };
  }, [
    action,
    amountInvalid,
    reasonInvalid,
    view,
    record,
    amount,
    day,
    reason,
    initialAmount,
    initialReason,
    editableAmount,
    editableReason,
  ]);
  useEffect(() => {
    const abort = new AbortController(),
      same = sessionGuard();
    setProposed(undefined);
    if (suspended) return;
    setError('');
    setCanReload(false);
    setField('');
    if (!command) return;
    const timer = setTimeout(
      () =>
        previewQuota(command, session.csrf, abort.signal)
          .then((v) => {
            if (!abort.signal.aborted && same()) setProposed(v);
          })
          .catch((e) => {
            if (!abort.signal.aborted && same() && !flow.inFlight) failed(e);
          }),
      200,
    );
    return () => {
      clearTimeout(timer);
      abort.abort();
    };
  }, [command, session.csrf, suspended]);
  const save = async () => {
    if (!command || blocked) return;
    setError('');
    const result = await flow.submit(command);
    if (result.kind === 'saved') done(result.result, day);
    else if (result.kind === 'failed') failed(result.error, true);
  };
  const locked = flow.inFlight || flow.unknown || blocked;
  return (
    <Box className="standard-page object-page">
      <RecordControl
        identity={view.name ?? view.code}
        navigation={null}
        actions={
          <>
            <Button
              variant="contained"
              disabled={!command || locked || suspended || (action === 'delete' && !proposed)}
              onClick={() => void save()}
            >
              {action === 'delete' ? '确认删除' : '保存'}
            </Button>
            <Button disabled={locked} onClick={() => exit.request(close)}>
              取消
            </Button>
          </>
        }
      />
      <Box className="page-scroll" data-page-scroll tabIndex={0}>
        <ObjectDetailSheet
          identity={
            <ObjectDetailIdentity
              icon={<WorkOutline />}
              title={view.targetKind === 'department' && (action === 'correct' || action === 'redate' || action === 'delete')
                ? departmentRecordLabels[action] : labels[action]}
              summary={view.name ?? view.code}
            />
          }
        >
          <ObjectForm>
            <Box className="object-detail-columns">
              <ObjectDetailSection title="基本资料">
                <ObjectDetailField label={view.targetKind === 'department' ? '部门' : '岗位'}>
                  {view.name ?? view.code}
                </ObjectDetailField>
                {editableAmount ? (
                  <ObjectTextField
                    id="quota-quota"
                    label={amountLabel}
                    required
                    value={amount}
                    type="number"
                    error={field === 'quota' || (!!amount && amountInvalid)}
                    helperText={amountInvalid ? '请输入非负整数' : undefined}
                    disabled={locked}
                    onChange={(e) => {
                      setSuspended(false);
                      setAmount(e.target.value);
                    }}
                    slotProps={{ htmlInput: { min: 0, step: 1 } }}
                  />
                ) : (
                  <ObjectDetailField label={amountLabel}>{quotaLabel(record, target)}</ObjectDetailField>
                )}
              </ObjectDetailSection>
              <ObjectDetailSection title="记录信息">
                {['create', 'adjust', 'redate', 'withdraw'].includes(action) ? (
                  <ObjectTextField
                    id="quota-effectiveDate"
                    label="生效日期"
                    required
                    type="date"
                    value={day}
                    error={field === 'effectiveDate'}
                    disabled={locked}
                    onChange={(e) => {
                      setSuspended(false);
                      setDay(e.target.value);
                    }}
                  />
                ) : (
                  record.effectiveDate && (
                    <ObjectDetailField label="生效日期">{record.effectiveDate}</ObjectDetailField>
                  )
                )}
                {editableReason ? (
                  <ObjectTextField
                    id="quota-reason"
                    label="调整原因"
                    required
                    value={reason}
                    error={field === 'reason' || (!!initialReason && reasonInvalid)}
                    helperText={reasonInvalid ? '请填写调整原因' : undefined}
                    disabled={locked}
                    onChange={(e) => {
                      setSuspended(false);
                      setReason(e.target.value);
                    }}
                  />
                ) : (
                  record.reason && <ObjectDetailField label="调整原因">{record.reason}</ObjectDetailField>
                )}
                {proposed?.effectiveDate && (
                  <ObjectDetailField label="适用期间">{period(proposed)}</ObjectDetailField>
                )}
              </ObjectDetailSection>
            </Box>
          </ObjectForm>
          {error && (
            <ObjectFormError
              id="quota-error"
              message={error}
              issues={field ? [{ id: 'quota-' + field, message: error }] : []}
            >
              {suspended && (
                <Button disabled={locked} onClick={() => setSuspended(false)}>
                  按当前权限预览
                </Button>
              )}
              {canReload && (
                <Button disabled={locked} onClick={() => refresh(false)}>
                  重读当前记录
                </Button>
              )}
            </ObjectFormError>
          )}
          {action === 'delete' && proposed && (
            <ObjectDetailField label={'删除后' + amountLabel}>{quotaLabel(proposed, target)}</ObjectDetailField>
          )}
        </ObjectDetailSheet>
      </Box>
      <UnsavedChangesDialog {...exit.dialog} pending={flow.unknown} />
    </Box>
  );
}
