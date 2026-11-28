import { useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { Pagination, Alert, Button, MenuItem, Typography } from '@mui/material';
import { ApiError, type Models } from '@chnhcm/api-client';
import {
  ObjectDetailField,
  ObjectDetailSection,
  ObjectForm,
  ObjectFormError,
  ObjectTextField,
  UnsavedChangesDialog,
  focusObjectField,
} from '@chnhcm/ui';
import { useDraftExit } from '../../shared/dated/useDraftExit';
import { useObjectSubmission } from '../../shared/dated/useObjectSubmission';
import { useRead, message } from '../../shared/useRead';
import {
  awardChannel,
  awardLabels,
  awardOriginal,
  payContext,
  recoverAward,
  releaseAward,
  submitAward,
  type Award,
  type Command,
  type AwardResult,
  type SavedAward,
} from './basePay';
import { PayBasisFacts } from './BasePayFacts';
import { money } from './rates';

type Props = {
  session: Models['Principal'];
  detail: Models['PayDetail'];
  action: string;
  award?: Award;
  initialDay: string;
  host: HTMLElement | null;
  onCancel: () => void;
  onSaved: (r: SavedAward) => void;
  onDenied: () => void;
  onReload: () => void;
};
export function BasePayEditor({
  session,
  detail,
  action,
  award,
  initialDay,
  host,
  onCancel,
  onSaved,
  onDenied,
  onReload,
}: Props) {
  const target = award ?? detail.current,
    correction = action === 'CORRECT',
    withdrawing = action === 'WITHDRAW';
  const original = useRead(award?.revision, () => awardOriginal(award!.awardId, award!.revision));
  const [day, setDay] = useState(initialDay);
  const [amount, setAmount] = useState(target?.amount ?? '');
  const [reason, setReason] = useState(correction ? (award?.reason ?? '') : '');
  const [note, setNote] = useState('');
  const [plan, setPlan] = useState<string>();
  const [version, setVersion] = useState(detail.version);
  const [issue, setIssue] = useState<{ text: string; field?: string }>();
  const [conflict, setConflict] = useState(false);
  const [candidatePage, setCandidatePage] = useState(1);
  const originalPlan = original.data?.basis?.planId;
  const [autoPlan, setAutoPlan] = useState<string>();
  const requestedPlan = plan ?? (correction ? originalPlan : undefined);
  const chosen = requestedPlan ?? autoPlan;
  const basis = useRead(
    /^\d{4}-\d{2}-\d{2}$/.test(day)
      ? JSON.stringify([detail.assignmentId, day, candidatePage, requestedPlan])
      : undefined,
    () => payContext(detail.assignmentId, day, candidatePage, requestedPlan),
  );
  const candidates = basis.data?.candidates ?? [];
  const selected = chosen ?? (basis.data?.candidateCount === 1 ? candidates[0]?.planId : undefined);
  const candidate = basis.data?.selectedBasis ?? candidates.find((b) => b.planId === selected);
  const options =
    candidate && !candidates.some((b) => b.planId === candidate.planId) ? [candidate, ...candidates] : candidates;
  useEffect(() => {
    if (!correction && !chosen && basis.data?.candidateCount === 1) setAutoPlan(basis.data.candidates[0]?.planId);
  }, [day, basis.data, chosen, correction]);
  const dirty =
    day !== initialDay ||
    amount !== (target?.amount ?? '') ||
    reason !== (correction ? (award?.reason ?? '') : '') ||
    note !== '' ||
    plan !== undefined;
  const sent = useRef<{ key: string; body: Command } | undefined>(undefined);
  const [retry, setRetry] = useState<string>();
  const [amountEdited, setAmountEdited] = useState(false);
  useEffect(
    () => () => {
      if (sent.current) releaseAward(sent.current.body);
    },
    [],
  );
  useEffect(() => {
    if (action === 'ADJUST' && !amountEdited && basis.data) setAmount(basis.data.current?.amount ?? '');
  }, [action, amountEdited, basis.data]);
  function completed(r: AwardResult) {
    if (r.outcome === 'accepted' && r.award) {
      if (sent.current) releaseAward(sent.current.body);
      onSaved({ submissionId: r.submissionId, award: r.award });
      return;
    }
    if (r.outcome === 'not_applied' && sent.current?.key === r.submissionId) {
      setRetry(r.confirmation);
      setIssue({ text: '已确认原申请未执行，可按原申请重送。' });
    }
  }
  const { flow, pending, sessionBlocked } = useObjectSubmission<Command, AwardResult>(
    session,
    awardChannel,
    (body, key) => {
      if (sent.current) releaseAward(sent.current.body);
      sent.current = { body, key };
      return submitAward(session, detail.assignmentId, body, key, retry);
    },
    (ref) => recoverAward(session, ref),
    completed,
    (e) => e instanceof ApiError && [400, 401, 403, 404, 409, 422].includes(e.status),
  );
  const exit = useDraftExit(
    dirty,
    () => flow.saved,
    () => {
      if (sent.current) releaseAward(sent.current.body);
      flow.clearPayload();
    },
  );
  const [reloading, setReloading] = useState(false);
  const targetChanged = !!award && !detail.items.some((item) => item.revision === award.revision && item.current);
  const refreshSource = () => {
    setReloading(true);
    setIssue(undefined);
    basis.invalidate();
    onReload();
  };
  const snapshot = useRef<{ day: string; plan?: string; token?: string } | undefined>(undefined);
  useEffect(() => {
    if (reloading && basis.data) {
      snapshot.current = candidate ? { day, plan: candidate.planId, token: candidate.token } : undefined;
      setVersion(basis.data.version);
      setConflict(false);
      setReloading(false);
      return;
    }
    if (!candidate) return;
    const last = snapshot.current;
    if (last?.day === day && last.plan === candidate.planId && last.token !== candidate.token) setConflict(true);
    snapshot.current = { day, plan: candidate.planId, token: candidate.token };
  }, [day, candidate?.planId, candidate?.token, reloading, basis.data]);
  useEffect(() => {
    if (basis.status === 403 || basis.status === 404 || original.status === 403 || original.status === 404)
      onDenied();
  }, [basis.status, original.status]);
  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (
      flow.inFlight ||
      flow.unknown ||
      pending ||
      sessionBlocked ||
      conflict ||
      reloading ||
      targetChanged ||
      version !== detail.version
    )
      return;
    const command: Command = { action, effectiveDate: day, expectedVersion: version };
    if (award) {
      command.awardId = award.awardId;
      command.expectedRevision = award.revision;
      command.note = note;
    }
    if (!withdrawing) {
      if (!correction || reason !== award?.reason) command.reason = reason;
      if (!correction || amount !== award?.amount || (plan !== undefined && plan !== originalPlan)) {
        if (!day) {
          setIssue({ text: '请填写决定生效日期', field: 'effectiveDate' });
          focusObjectField('pay-effectiveDate', 'pay-error');
          return;
        }
        if (!candidate) {
          setIssue({ text: '请选择此生效日的基本月薪计划', field: 'planId' });
          focusObjectField('pay-planId', 'pay-error');
          return;
        }
        command.planId = candidate.planId;
        command.componentId = candidate.componentId;
        command.basisToken = candidate.token;
        command.amount = amount;
      }
    }
    setIssue(undefined);
    const result =
      retry && sent.current ? await flow.submit(sent.current.body, sent.current.key) : await flow.submit(command);
    if (result.kind === 'saved') {
      completed(result.result);
      return;
    }
    if (result.kind === 'failed') {
      const e = result.error;
      if (result.definitive && sent.current) {
        releaseAward(sent.current.body);
        sent.current = undefined;
        setRetry(undefined);
      }
      if (e instanceof ApiError && [401, 403, 404].includes(e.status)) {
        onDenied();
        return;
      }
      if (e instanceof ApiError && e.status === 409) setConflict(true);
      setIssue({ text: message(e), field: e instanceof ApiError ? e.field : undefined });
      focusObjectField('pay-' + (e instanceof ApiError ? e.field : ''), 'pay-error');
    }
  };
  const selectDay = (d: string) => {
    setDay(d);
    setAutoPlan(undefined);
    setCandidatePage(1);
    setPlan(undefined);
    setIssue(undefined);
  };
  const disabled = flow.inFlight || flow.unknown || pending || sessionBlocked;
  const readingBasis = !withdrawing && /^\d{4}-\d{2}-\d{2}$/.test(day) && !basis.data && !basis.error;
  const controls = (
    <>
      <Button disabled={flow.inFlight} onClick={() => exit.request(onCancel)}>
        取消
      </Button>
      <Button
        type="submit"
        form="pay-award-form"
        variant="contained"
        color={withdrawing ? 'error' : 'primary'}
        disabled={disabled || readingBasis || conflict || reloading || targetChanged || version !== detail.version}
      >
        {retry ? '按原申请重送' : withdrawing ? '确认撤去' : '保存'}
      </Button>
    </>
  );
  const restoration = award?.restoration;
  return (
    <ObjectForm>
      <form id="pay-award-form" onSubmit={submit} noValidate>
        {host ? createPortal(controls, host) : controls}
        <UnsavedChangesDialog {...exit.dialog} />
        {issue && <ObjectFormError id="pay-error" message={issue.text} />}
        {targetChanged && <Alert severity="warning">原笔已更正或撤去，请返回核定记录选择最新修订。</Alert>}
        {!targetChanged && (conflict || version !== detail.version) && (
          <Alert severity="warning" action={<Button onClick={refreshSource}>重读依据</Button>}>
            核定或依据已变化，请核对最新资料。
          </Alert>
        )}
        <fieldset disabled={disabled || !!retry} style={{ border: 0, padding: 0, margin: 0, minWidth: 0 }}>
          <ObjectDetailSection title={awardLabels[action]}>
            {award ? (
              <ObjectDetailField label="生效日期">{award.effectiveDate}</ObjectDetailField>
            ) : (
              <ObjectTextField
                type="date"
                required
                id="pay-effectiveDate"
                label="生效日期"
                value={day}
                onChange={(e) => selectDay(e.target.value)}
              />
            )}
            {withdrawing ? (
              <>
                <ObjectDetailField label="原基本月薪">{money(award?.amount)} 元/月</ObjectDetailField>
                <ObjectDetailField label="撤去后">
                  {restoration?.amount
                    ? money(restoration.amount) + ' 元/月'
                    : restoration
                      ? '尚未定薪'
                      : '办理影响不可用'}
                </ObjectDetailField>
              </>
            ) : (
              <>
                <ObjectTextField
                  select
                  required
                  id="pay-planId"
                  label="薪酬计划"
                  value={selected && options.some((b) => b.planId === selected) ? selected : ''}
                  onChange={(e) => setPlan(e.target.value || undefined)}
                >
                  <MenuItem value="">请选择</MenuItem>
                  {options.map((b) => (
                    <MenuItem key={b.planId} value={b.planId}>
                      {[b.planCode, b.planName].filter(Boolean).join(' · ')}
                    </MenuItem>
                  ))}
                </ObjectTextField>
                {basis.data && basis.data.candidateCount > basis.data.candidatePageSize && (
                  <Pagination
                    aria-label="薪酬计划候选分页"
                    count={Math.ceil(basis.data.candidateCount / basis.data.candidatePageSize)}
                    page={basis.data.candidatePage}
                    onChange={(_, page) => setCandidatePage(page)}
                  />
                )}
                <ObjectTextField
                  required
                  id="pay-amount"
                  label="个人基本工资（元/月）"
                  value={amount}
                  onChange={(e) => {
                    setAmountEdited(true);
                    setAmount(e.target.value);
                  }}
                  error={issue?.field === 'amount'}
                  helperText={issue?.field === 'amount' ? issue.text : undefined}
                  slotProps={{ htmlInput: { inputMode: 'decimal' } }}
                />
                <ObjectTextField
                  id="pay-reason"
                  label="业务原因"
                  required={action === 'ADJUST'}
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                />
              </>
            )}
            {award && (
              <ObjectTextField
                required
                id="pay-note"
                label={withdrawing ? '撤去缘由' : '更正缘由'}
                value={note}
                onChange={(e) => setNote(e.target.value)}
              />
            )}
          </ObjectDetailSection>
        </fieldset>
        {candidate && <PayBasisFacts basis={candidate} title="生效日制度" day={day} />}
        {basis.data?.message && <Typography>{basis.data.message}</Typography>}
        {basis.error && (
          <Alert severity="error" action={<Button onClick={basis.refresh}>重试</Button>}>
            {basis.error}
          </Alert>
        )}
      </form>
    </ObjectForm>
  );
}
