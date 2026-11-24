import { useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { Alert, Autocomplete, Box, Button, Stack, TextField, Typography } from '@mui/material';
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

type Location = Models['EmploymentLocationRef'];
type Action = Models['EmploymentAction'];
type Command = { action?: string; input: Models['EmploymentLocationChange'] };
type Draft = {
  action?: Action;
  day: string;
  initialDay: string;
  original?: Location;
  value?: Location;
  source: string;
};
const actionLabel = (a: Action) =>
  a.kind === 'DataChange' ? '数据更改' : a.kind === 'CancelHire' ? '撤销入职' : '入职';

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
  const [draft, setDraft] = useState<Draft>();
  const [search, setSearch] = useState('');
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
    const allowed = draft.action
      ? data.actions.some((a) => a.id === draft.action?.id && a.canCorrectLocation)
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
  const choices = useRead(
    draft?.day ? JSON.stringify([assignment, draft.day, search]) : undefined,
    async () =>
      unwrap(
        await api.POST('/api/v1/assignments/{id}/location-options', {
          params: { path: { id: assignment }, header },
          body: { businessDate: draft!.day, search, page: 1 },
        }),
      ),
  );
  const close = () => {
    setDraft(undefined);
    setSearch('');
    setError('');
    requestAnimationFrame(() => opener.current?.focus());
  };
  const dirty = !!draft && (draft.day !== draft.initialDay || draft.value?.id !== draft.original?.id);
  const exit = useDraftExit(dirty, () => saved.current, close);
  const submit = useRuleSubmission<Command, Models['HireReceipt']>('assignment-location', session, {
    send: async (cmd, key) => {
      const headers = { ...header, 'Idempotency-Key': key };
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
  function begin(button: HTMLButtonElement, a?: Action) {
    if (!data) return;
    opener.current = button;
    saved.current = false;
    setError('');
    setNotice('');
    setSearch('');
    submit.start();
    const date = a?.businessDate ?? day;
    const initial = a ? original.data?.locationChange?.after : data.state?.location;
    setDraft({
      action: a,
      day: date,
      initialDay: date,
      source: data.sourceVersion,
      original: initial,
      value: initial,
    });
  }
  function save() {
    if (!draft?.value) return;
    void submit.submit({
      action: draft.action?.id,
      input: {
        businessDate: draft.day,
        expectedSourceVersion: draft.source,
        expectedRevision: draft.action?.revision,
        location: draft.value.id,
      },
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
            <Box
              component="form"
              onSubmit={(e) => {
                e.preventDefault();
                save();
              }}
            >
              <Stack spacing={1.5}>
                <Typography component="h3">{draft.action ? '更正记录' : '新增记录'} · 数据更改</Typography>
                <ObjectTextField
                  type="date"
                  label="生效日期"
                  value={draft.day}
                  required
                  disabled={!!draft.action}
                  onChange={(e) => setDraft({ ...draft, day: e.target.value })}
                />
                <ObjectDetailField label="业务原因">工作地点变更</ObjectDetailField>
                <ObjectDetailField label="工作地点" inputId="assignment-location" required>
                  <Autocomplete
                    id="assignment-location"
                    fullWidth
                    size="small"
                    value={draft.value ?? null}
                    options={choices.data ?? []}
                    filterOptions={(v) => v}
                    isOptionEqualToValue={(a, b) => a.id === b.id}
                    getOptionLabel={(v) => v.name ?? v.id}
                    onInputChange={(_, text, reason) => {
                      if (reason === 'input' || reason === 'clear') setSearch(text);
                    }}
                    onChange={(_, v) => setDraft({ ...draft, value: v ?? undefined })}
                    noOptionsText={choices.error ?? '没有匹配的地点'}
                    renderInput={(props) => (
                      <TextField
                        {...props}
                        variant="standard"
                        required
                        error={!!choices.error}
                        helperText={choices.error}
                      />
                    )}
                  />
                </ObjectDetailField>
                <Stack direction="row" gap={1}>
                  <Button
                    variant="contained"
                    type="submit"
                    disabled={!draft.value || !draft.day || !dirty || submit.busy}
                  >
                    保存
                  </Button>
                  <Button onClick={() => exit.request(close)}>取消</Button>
                </Stack>
              </Stack>
            </Box>
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
                  {data.canChangeLocation &&
                    actionSlot &&
                    createPortal(
                      <Button variant="contained" onClick={(e) => begin(e.currentTarget)}>
                        新增记录
                      </Button>,
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
                  {opened.canCorrectLocation &&
                    original.data.action.revision === opened.revision &&
                    !source && <Button onClick={(e) => begin(e.currentTarget, opened)}>更正记录</Button>}
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
          查回地点变更提交
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
