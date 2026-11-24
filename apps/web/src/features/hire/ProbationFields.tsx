import { useState } from 'react';
import { Autocomplete, MenuItem, TextField } from '@mui/material';
import { api, unwrap, type Models } from '@chnhcm/api-client';
import { ObjectTextField, ObjectDetailField } from '@chnhcm/ui';
import { useRead } from '../personnel/api';
import type { JobDraft } from './AssignmentRecordForm';

export function ProbationFields({
  draft,
  view,
  csrf,
  update,
}: {
  draft: JobDraft;
  view: Models['AssignmentView'];
  csrf: string;
  update: (v: JobDraft) => void;
}) {
  const [search, setSearch] = useState('');
  const confirm = draft.kind === 'ConfirmEmployment';
  const departments = useRead(
    confirm && draft.day ? JSON.stringify([view.assignment, draft.day, search]) : undefined,
    async () =>
      unwrap(
        await api.POST('/api/v1/assignments/{id}/probation-departments', {
          params: { path: { id: view.assignment }, header: { 'X-CSRF-Token': csrf } },
          body: { businessDate: draft.day, search, page: 1 },
        }),
      ),
  );
  const earlier = view.actions.filter(
    (a) => a.kind === 'Transfer' && a.operation !== 'Delete' && a.canHistory && a.businessDate === draft.day,
  );
  return (
    <>
      <ObjectDetailField label="所属工作关系">{view.employerName}</ObjectDetailField>
      {confirm && (
        <>
          <ObjectDetailField label="实际试用开始日">{view.probation.start}</ObjectDetailField>
          <ObjectDetailField label="通知记载的发生部门" inputId="confirmation-department">
            <Autocomplete
              id="confirmation-department"
              size="small"
              fullWidth
              value={draft.occurrence ?? null}
              options={departments.data ?? []}
              filterOptions={(v) => v}
              getOptionLabel={(v) => v.name}
              isOptionEqualToValue={(a, b) => a.id === b.id}
              onInputChange={(_, v, r) => {
                if (r === 'clear' || r === 'input') setSearch(v);
              }}
              onChange={(_, v) => update({ ...draft, occurrence: v ?? undefined, predecessor: undefined })}
              noOptionsText={departments.error ?? '没有匹配的部门'}
              renderInput={(props) => (
                <TextField
                  {...props}
                  variant="standard"
                  error={!!departments.error}
                  helperText={departments.error}
                />
              )}
            />
          </ObjectDetailField>
          {!draft.occurrence && earlier.length > 0 && (
            <ObjectTextField
              select
              label="通知明确在转正前生效的调动"
              value={draft.predecessor?.id ?? ''}
              onChange={(e) =>
                update({ ...draft, predecessor: earlier.find((a) => a.id === e.target.value) })
              }
            >
              <MenuItem value="">未载明</MenuItem>
              {earlier.map((a) => (
                <MenuItem key={a.id} value={a.id}>
                  {a.businessDate} · {a.notice}
                </MenuItem>
              ))}
            </ObjectTextField>
          )}
        </>
      )}
      <ObjectTextField
        label={confirm ? '转正依据' : '试用依据'}
        required
        value={draft.notice}
        onChange={(e) => update({ ...draft, notice: e.target.value })}
      />
    </>
  );
}
