import { useEffect, useReducer, useRef, useState } from 'react';
import { Alert, Box, Button } from '@mui/material';
import Business from '@mui/icons-material/Business';
import {
  ObjectDetailIdentity,
  ObjectDetailSection,
  ObjectDetailSheet,
  ObjectForm,
  ObjectTextField,
  RecordControl,
  UnsavedChangesDialog,
} from '@chnhcm/ui';
import { ApiError, sessionGuard, type Models } from '@chnhcm/api-client';
import { SubmissionFlow } from '../../shared/dated/submission';
import { useDraftExit } from '../../shared/dated/useDraftExit';
import { useSessionRecovery } from '../../shared/sessionRecovery';
import {
  readDepartment,
  receipts,
  resolveQuota,
  saveAllocation,
  type Department,
  type Result,
} from './quotaApi';

type Input = Models['AllocationInput'];
type Editable = { id: string; label: string; amount?: number | null; source?: string; withdraw?: boolean };
export function AllocationEditor({
  view,
  session,
  close,
  done,
}: {
  view: Department;
  session: Models['Principal'];
  close: () => void;
  done: (result: Result, date: string) => void;
}) {
  const initial = view.defaultEffectiveDate ?? view.businessAt;
  const [day, setDay] = useState(initial),
    [dateInput, setDateInput] = useState(initial),
    [pendingDay, setPendingDay] = useState(''),
    [baseline, setBaseline] = useState<Department>(),
    [reason, setReason] = useState(''),
    [values, setValues] = useState<Record<string, string>>({}),
    [loadError, setLoadError] = useState(''),
    [error, setError] = useState(''),
    [retry, setRetry] = useState(0);
  const mounted = useRef(false),
    [, render] = useReducer((n) => n + 1, 0),
    blocked = useSessionRecovery();
  const [flow] = useState(
    () =>
      new SubmissionFlow<Input, Result, Result>({
        owner: session,
        channel: receipts,
        uuid: () => crypto.randomUUID(),
        changed: render,
        current: () => {
          const same = sessionGuard();
          return () => mounted.current && same();
        },
        send: (body, id) => saveAllocation(view.departmentId, body, session.csrf, id),
        unwrap: (r) => r,
        resolve: (ref) => resolveQuota(session.csrf, ref),
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
            else setError(result.failure?.message ?? '原提交未生效');
          }
        },
        () => {},
      ),
    [flow, day, done],
  );
  useEffect(() => {
    const abort = new AbortController(),
      same = sessionGuard();
    setBaseline(undefined);
    setLoadError('');
    setError('');
    if (day)
      readDepartment(view.departmentId, day, '', abort.signal)
        .then((v) => {
          if (same() && !abort.signal.aborted) setBaseline(v);
        })
        .catch((e) => {
          if (same() && !abort.signal.aborted)
            setLoadError(
              e instanceof ApiError && e.status === 404
                ? view.firstEffectiveDate && day < view.firstEffectiveDate
                  ? '生效日不能早于部门首日 ' + view.firstEffectiveDate
                  : '该日没有可办理的部门资料，请选择有效的生效日'
                : e.message,
            );
        });
    return () => abort.abort();
  }, [day, retry, view.departmentId, view.firstEffectiveDate]);
  const current = baseline?.businessAt === day && dateInput === day ? baseline : undefined;
  const fields: Editable[] = [];
  if (current?.canSetTotal)
    fields.push({
      id: 'total',
      label: '部门总额',
      amount: current.quota,
      source: current.sourceVersion,
      withdraw: current.canWithdraw,
    });
  for (const m of current?.members ?? [])
    if (m.canAllocate)
      fields.push({
        id: m.positionId,
        label: m.name ?? m.code ?? '岗位核定编制',
        amount: m.quota,
        source: m.sourceVersion,
        withdraw: m.canWithdraw,
      });
  const editedAmounts = Object.keys(values).length > 0;
  const dirty = editedAmounts || reason !== '' || dateInput !== initial;
  const exit = useDraftExit(dirty, () => flow.saved, close);
  const locked = flow.inFlight || flow.unknown || blocked;
  const command = (): Input | undefined => {
    if (!current || !reason.trim()) return;
    const body: Input = { effectiveDate: day, reason };
    for (const f of fields) {
      const value = values[f.id];
      if (value === undefined || value === String(f.amount ?? '')) continue;
      const number = Number(value);
      if (value === '' ? !f.withdraw : !Number.isSafeInteger(number) || number < 0) return;
      const row: Models['AllocationRow'] = {
        expectedSourceVersion: f.source,
        ...(value === '' ? { withdraw: true } : { quota: number }),
      };
      if (f.id === 'total') body.total = row;
      else (body.positions ??= []).push({ ...row, positionId: f.id });
    }
    if (body.total || body.positions?.length) return body;
  };
  const body = command();
  const save = async () => {
    if (!body || locked) return;
    setError('');
    const r = await flow.submit(body);
    if (r.kind === 'saved') done(r.result, day);
    else if (r.kind === 'failed')
      setError(r.error instanceof Error ? r.error.message : '暂时无法确认保存结果');
  };
  return (
    <Box className="standard-page object-page">
      <RecordControl
        identity={view.name}
        navigation={null}
        actions={
          <>
            <Button variant="contained" disabled={!body || locked} onClick={() => void save()}>
              保存
            </Button>
            <Button disabled={locked} onClick={() => exit.request(close)}>
              取消
            </Button>
          </>
        }
      />
      <Box className="page-scroll" data-page-scroll tabIndex={0}>
        <ObjectDetailSheet
          identity={<ObjectDetailIdentity icon={<Business />} title="调整编制" summary={view.name} />}
        >
          <ObjectForm>
            <ObjectDetailSection title="记录信息">
              <ObjectTextField
                type="date"
                label="生效日期"
                required
                value={dateInput}
                onChange={(e) => {
                  const next = e.target.value;
                  if (next && e.target.validity.valid && next !== day) {
                    if (editedAmounts) {
                      setDateInput(day);
                      setPendingDay(next);
                      return;
                    }
                    setDay(next);
                  }
                  setDateInput(next);
                }}
                disabled={locked}
              />
              <ObjectTextField
                label="调整原因"
                required
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                disabled={locked}
              />
            </ObjectDetailSection>
            {loadError && <Alert severity="warning">{loadError}</Alert>}
            {current && (
              <ObjectDetailSection title="核定编制">
                {fields.map((f) => (
                  <ObjectTextField
                    key={f.id}
                    label={f.label}
                    type="number"
                    slotProps={{ htmlInput: { min: 0, step: 1 } }}
                    helperText={f.withdraw && f.amount != null ? '清空可撤去该项核定' : undefined}
                    value={values[f.id] ?? String(f.amount ?? '')}
                    disabled={locked}
                    onChange={(e) => {
                      const v = e.target.value;
                      setValues((old) => {
                        const next = { ...old };
                        if (v === String(f.amount ?? '')) delete next[f.id];
                        else next[f.id] = v;
                        return next;
                      });
                    }}
                  />
                ))}
              </ObjectDetailSection>
            )}
          </ObjectForm>
          {error && (
            <Alert
              severity="error"
              action={
                <Button disabled={locked} onClick={() => setRetry((n) => n + 1)}>
                  重新查询
                </Button>
              }
            >
              {error}
            </Alert>
          )}
        </ObjectDetailSheet>
      </Box>
      <UnsavedChangesDialog {...exit.dialog} pending={flow.unknown} />
      <UnsavedChangesDialog open={!!pendingDay} onContinue={() => setPendingDay('')} onDiscard={() => {
        setValues({});
        setDay(pendingDay);
        setDateInput(pendingDay);
        setPendingDay('');
      }} />
    </Box>
  );
}
