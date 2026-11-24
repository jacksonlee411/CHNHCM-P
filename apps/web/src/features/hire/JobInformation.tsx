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
import { useDraftExit } from '../../shared/dated/useDraftExit';
import { useRead, message } from '../personnel/api';
import { useRuleSubmission } from '../readiness/useRuleSubmission';

import { AssignmentRecordForm, actionLabel, draftChanged, type JobDraft } from './AssignmentRecordForm';

type Action = Models['EmploymentAction'];
type Command =
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
      draft.mode === 'redate'
        ? a?.canMaintain
        : draft.action
          ? draft.kind === 'Transfer'
            ? a?.canCorrectTransfer
            : a?.canCorrectLocation
          : draft.kind === 'Transfer'
            ? data.canTransfer
            : data.canChangeLocation;
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
    setDraft(undefined);
    setError('');
    requestAnimationFrame(() => opener.current?.focus());
  };
  const dirty = !!draft && !!initial.current && draftChanged(draft, initial.current);
  const exit = useDraftExit(dirty, () => saved.current, close);
  const submit = useRuleSubmission<Command, Models['HireReceipt']>('assignment-location', session, {
    send: async (cmd, key) => {
      const headers = { ...header, 'Idempotency-Key': key };
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
        await api.POST('/api/v1/hires/submissions/{sub}/resolve', { params: { path: { sub }, header } }),
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
  function begin(button: HTMLButtonElement, mode: JobDraft['mode'], a?: Action) {
    if (!data) return;
    opener.current = button;
    saved.current = false;
    setError('');
    setNotice('');
    submit.start();
    const evidence = a ? original.data : undefined;
    const after = evidence?.transfer?.after;
    const state = data.state;
    const department = after?.department ?? state?.departmentId;
    const job = after?.job ?? state?.jobId;
    const next: JobDraft = {
      mode,
      kind: a?.kind === 'Transfer' || (!a && !data.canChangeLocation) ? 'Transfer' : 'DataChange',
      action: a,
      day: a?.businessDate ?? day,
      source: data.sourceVersion,
      value: a ? evidence?.locationChange?.after : state?.location,
      department: department
        ? { id: department, name: after?.segments?.[0]?.departmentName ?? state?.department ?? '' }
        : undefined,
      job: job ? { id: job, name: after?.segments?.[0]?.jobName ?? state?.job ?? '' } : undefined,
      notice: evidence?.action.notice ?? '',
      requires: evidence?.requires,
    };
    initial.current = next;
    setDraft(next);
  }
  function save(record: JobDraft) {
    const action = record.action?.id;
    const input = {
      expectedSourceVersion: record.source,
      expectedRevision: record.action?.revision,
      businessDate: record.day,
    };
    if (record.mode === 'redate') {
      void submit.submit({ kind: 'redate', action, input });
      return;
    }
    const changes = { ...input, insert: record.mode === 'insert' || undefined, requires: record.requires };
    if (record.kind === 'DataChange' && record.value)
      void submit.submit({ kind: 'location', action, input: { ...changes, location: record.value.id } });
    if (record.kind === 'Transfer')
      void submit.submit({
        kind: 'transfer',
        action,
        input: { ...changes, department: record.department?.id, job: record.job?.id, notice: record.notice },
      });
  }
  function remove(a: Action) {
    if (!data || !a.canDelete) return;
    setError('');
    setNotice('');
    submit.start();
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
              {!selectedAction && (
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
                  {(data.canChangeLocation || data.canTransfer) &&
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
              {opened && original.error && (
                <Alert severity="error" action={<Button onClick={original.refresh}>重试</Button>}>
                  {original.error}
                </Alert>
              )}
              {opened && original.data && (
                <Box>
                  <OriginalEvidence original={original.data} setRevision={setRevision} />
                  {original.data.action.revision === opened.revision && !source && (
                    <Stack direction="row" gap={1} flexWrap="wrap">
                      {(opened.canCorrectLocation || opened.canCorrectTransfer) && (
                        <Button onClick={(e) => begin(e.currentTarget, 'correct', opened)}>更正记录</Button>
                      )}
                      {opened.canMaintain && (
                        <Button onClick={(e) => begin(e.currentTarget, 'redate', opened)}>
                          调整生效日期
                        </Button>
                      )}
                      {opened.canDelete && (
                        <Button disabled={submit.busy} onClick={() => remove(opened)}>
                          删除记录
                        </Button>
                      )}
                    </Stack>
                  )}
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
                    setSource('');
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
      {original.action.operation === 'Delete' && (
        <ObjectDetailField label="办理状态">已删除</ObjectDetailField>
      )}
      {original.secondary && <>
        <ObjectDetailField label="任职安排">{[original.secondary.segments[0]?.departmentName,original.secondary.segments[0]?.jobName].filter(Boolean).join(' · ')}</ObjectDetailField>
        <ObjectDetailField label="所属工作关系">{original.secondary.legal[0]?.entityName}{original.secondary.relationshipPrimary?' · 关系主任职':''}</ObjectDetailField>
      </>}
      {original.transfer?.beforeDepartmentName && (
        <ObjectDetailField label="调出安排">
          {[original.transfer.beforeDepartmentName, original.transfer.beforeJobName]
            .filter(Boolean)
            .join(' · ')}
        </ObjectDetailField>
      )}
      {original.transfer?.after.segments?.[0] && (
        <ObjectDetailField label="调入安排">
          {[original.transfer.after.segments[0].departmentName, original.transfer.after.segments[0].jobName]
            .filter(Boolean)
            .join(' · ')}
        </ObjectDetailField>
      )}
      <ObjectDetailField label="首次办理时间">{recordedTime(original.action.recordedAt)}</ObjectDetailField>
      {original.departmentName && (
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
      {original.revisions?.map((r) => (
        <Button key={r.id} onClick={() => setRevision(r.id)} aria-pressed={r.id === original.action.revision}>
          查看修订 · {recordedTime(r.recordedAt)}
        </Button>
      ))}
      {operator && <ObjectDetailField label="经办人">{operator}</ObjectDetailField>}
    </>
  );
}
