import { useState } from 'react';
import { Autocomplete, TextField } from '@mui/material';
import { api, unwrap, type Models } from '@chnhcm/api-client';
import { ObjectDetailField } from '@chnhcm/ui';
import { useRead } from '../personnel/api';

export type TargetKind = 'department' | 'job' | 'employer' | 'bu' | 'location' | 'grade';
export type Option = { id: string; name: string };
export const labels = {
  grade:'个人职级',
  department: '部门',
  job: '职务',
  employer: '用人单位',
  bu: '适用业务单元',
  location: '适用地点',
};

export function Target({
  kind,
  choice,
  onChange,
  draft,
  session,
}: {
  kind: TargetKind;
  choice?: Option;
  onChange: (value?: Option) => void;
  draft: Models['HireInput'];
  session: Models['Principal'];
}) {
  const [search, setSearch] = useState('');
  const enabled = !!draft.arrival && (kind !== 'grade' || !!draft.job);
  const key = [
    kind,
    search,
    draft.arrival,
    draft.lastDay,
    draft.department,
    draft.job,
    draft.employer,
    draft.bu,
    draft.location,
  ];
  const candidates = useRead(enabled ? JSON.stringify(key) : undefined, async (): Promise<Option[]> => {
    const params = { header: { 'X-CSRF-Token': session.csrf } };
    const query = {
      person: draft.person,
      arrival: draft.arrival,
      lastDay: draft.lastDay || undefined,
      department: draft.department || undefined,
      search,
      page: 1,
    };
    if (kind === 'employer') {
      return unwrap(
        await api.POST('/api/v1/hires/employers', {
          params,
          body: { ...query, bu: draft.bu || undefined, location: draft.location || undefined },
        }),
      );
    }
    return unwrap(
      await api.POST('/api/v1/hires/targets', {
        params,
        body: { ...query, kind, job:kind==='grade'?draft.job:undefined, employer: draft.employer || undefined },
      }),
    );
  });
  const required = kind !== 'bu' && kind !== 'location' && kind !== 'grade';
  return (
    <ObjectDetailField label={labels[kind]} inputId={'hire-' + kind} required={required}>
      <Autocomplete
        id={'hire-' + kind}
        disabled={!enabled}
        options={candidates.data ?? []}
        value={choice ?? null}
        size="small"
        fullWidth
        filterOptions={(v) => v}
        getOptionLabel={(v) => v.name}
        isOptionEqualToValue={(a, b) => a.id === b.id}
        onInputChange={(_, text, reason) => {
          if (reason === 'input' || reason === 'clear') setSearch(text);
        }}
        onChange={(_, v) => onChange(v ?? undefined)}
        noOptionsText={candidates.error ?? '没有匹配记录'}
        renderInput={(props) => (
          <TextField
            {...props}
            variant="standard"
            required={required}
            error={!!candidates.error}
            helperText={candidates.error}
          />
        )}
      />
    </ObjectDetailField>
  );
}
