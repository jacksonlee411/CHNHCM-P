import { useState } from 'react';
import { Autocomplete, MenuItem, TextField } from '@mui/material';
import { api, unwrap, type Models } from '@chnhcm/api-client';
import { ObjectDetailField, ObjectTextField } from '@chnhcm/ui';
import { useRead } from '../personnel/api';
import { actionLabel, TransferTarget, type JobDraft } from './AssignmentRecordForm';

type Props = {
  draft: JobDraft;
  view: Models['AssignmentView'];
  csrf: string;
  update: (draft: JobDraft) => void;
};

export function EarlierAction({ draft, view, update }: Omit<Props, 'csrf'>) {
  const choices = view.actions.filter(
    (a) =>
      a.canHistory &&
      a.id !== draft.action?.id &&
      a.id !== draft.successor?.id &&
      a.businessDate === draft.day &&
      ['ConfirmEmployment', 'Promotion', 'Transfer'].includes(a.kind) &&
      !['Withdraw', 'Delete'].includes(a.operation),
  );
  if (!choices.length) return null;
  return (
    <ObjectTextField
      select
      label="通知载明的前序办理"
      value={draft.predecessor?.id ?? ''}
      onChange={(e) => update({ ...draft, predecessor: choices.find((a) => a.id === e.target.value) })}
    >
      <MenuItem value="">未载明</MenuItem>
      {choices.map((a) => (
        <MenuItem key={a.id} value={a.id}>
          {actionLabel(a)} · {a.notice}
        </MenuItem>
      ))}
    </ObjectTextField>
  );
}

export function PromotionFields({ draft, view, csrf, update }: Props) {
  const [search, setSearch] = useState('');
  const withdraw = draft.mode === 'withdraw';
  const initial = draft.kind === 'GradeRecord';
  const pair = draft.kind === 'PromotionTransfer';
  const job = draft.job?.id;
  const grades = useRead(
    !withdraw && draft.day ? JSON.stringify([view.assignment, draft.day, job, search]) : undefined,
    async () =>
      unwrap(
        await api.POST('/api/v1/assignments/{id}/grade-options', {
          params: { path: { id: view.assignment }, header: { 'X-CSRF-Token': csrf } },
          body: { businessDate: draft.day, job, search, page: 1 },
        }),
      ),
  );
  if (withdraw)
    return (
      <>
        <ObjectDetailField label="原晋升通知">{draft.action?.notice}</ObjectDetailField>
        <ObjectTextField
          select
          label="引用通知"
          value={draft.basis?.id ?? ''}
          onChange={(e) => update({ ...draft, basis: view.actions.find((a) => a.id === e.target.value) })}
        >
          <MenuItem value="">未选择</MenuItem>
          {view.actions
            .filter(
              (a) =>
                a.canHistory &&
                a.notice &&
                a.businessDate === draft.day &&
                a.operation !== 'Withdraw' &&
                a.operation !== 'Delete',
            )
            .map((a) => (
              <MenuItem key={a.id} value={a.id}>
                {actionLabel(a)} · {a.notice}
              </MenuItem>
            ))}
        </ObjectTextField>
        <ObjectTextField
          label="撤销依据（误录原因）"
          required={!draft.basis}
          value={draft.withdrawal ?? ''}
          onChange={(e) => update({ ...draft, withdrawal: e.target.value })}
        />
      </>
    );
  return (
    <>
      {!initial && (
        <ObjectDetailField label="原任职级">{draft.originalGrade?.name ?? '未记载'}</ObjectDetailField>
      )}
      {!initial && view.canTransfer && (
        <>
          <TransferTarget
            kind="department"
            draft={draft}
            assignment={view.assignment}
            csrf={csrf}
            change={(department) => update({ ...draft, department })}
          />
          <TransferTarget
            kind="job"
            draft={draft}
            assignment={view.assignment}
            csrf={csrf}
            change={(job) => update({ ...draft, job, grade: undefined })}
          />
        </>
      )}
      <ObjectDetailField label={initial ? '实际任职级' : '拟任职级'} inputId="assignment-grade" required>
        <Autocomplete
          fullWidth
          size="small"
          id="assignment-grade"
          value={draft.grade ?? null}
          options={grades.data ?? []}
          filterOptions={(items) => items}
          getOptionLabel={(item) => item.name}
          isOptionEqualToValue={(a, b) => a.id === b.id}
          onInputChange={(_, value, reason) => {
            if (reason === 'input' || reason === 'clear') setSearch(value);
          }}
          onChange={(_, grade) => update({ ...draft, grade: grade ?? undefined })}
          noOptionsText={grades.error ?? '没有匹配的可用职级'}
          renderInput={(props) => (
            <TextField
              {...props}
              variant="standard"
              required
              error={!!grades.error}
              helperText={grades.error}
            />
          )}
        />
      </ObjectDetailField>
      {!initial && (
        <ObjectTextField
          label="晋升原因"
          value={draft.reason ?? '职级晋升'}
          onChange={(e) => update({ ...draft, reason: e.target.value })}
        />
      )}
      <ObjectTextField
        label={initial ? '原职级任用依据' : '晋升依据'}
        required
        value={draft.notice}
        onChange={(e) => update({ ...draft, notice: e.target.value })}
      />
      {pair && (
        <ObjectTextField
          label="调动另有通知时填写"
          value={draft.transferNotice ?? ''}
          onChange={(e) => update({ ...draft, transferNotice: e.target.value })}
        />
      )}
      {!initial && <EarlierAction draft={draft} view={view} update={update} />}
      {draft.successor && <ObjectDetailField label="后续调动">{draft.successor.notice}</ObjectDetailField>}
    </>
  );
}
