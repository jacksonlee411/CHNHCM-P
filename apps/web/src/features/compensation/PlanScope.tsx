import { useState } from 'react';
import { Autocomplete, MenuItem, TextField } from '@mui/material';
import { ObjectDetailField, ObjectTextField } from '@chnhcm/ui';
import { useRead } from '../../shared/useRead';
import { dimensionNames, scopeCandidates, type NamedChoice, type ScopeChoice } from './plans';

export type ScopeDraft = { mode: '' | 'ENTERPRISE' | 'SELECTED'; choices: NamedChoice[] };

function ScopePicker({
  dimension,
  exclude,
  value,
  onChange,
}: {
  dimension: ScopeChoice['dimension'];
  exclude: boolean;
  value: NamedChoice[];
  onChange: (v: NamedChoice[]) => void;
}) {
  const [search, setSearch] = useState('');
  const list = useRead(JSON.stringify([dimension, search]), () => scopeCandidates(dimension, search), 'fresh');
  const label = (exclude ? '排除' : '适用') + dimensionNames[dimension];
  const id = 'plan-scope-' + dimension + (exclude ? '-exclude' : '');
  const options = (list.data ?? []).map((r) => ({ ...r, dimension, exclude }));
  return (
    <ObjectDetailField inputId={id} label={label}>
      <Autocomplete<NamedChoice, true>
        multiple
        id={id}
        options={options}
        value={value}
        filterOptions={(items) => items}
        filterSelectedOptions
        loading={!list.data && !list.error}
        loadingText="正在查询"
        noOptionsText="没有符合条件的对象"
        getOptionKey={(v) => v.id}
        renderOption={({ key, ...props }, v) => (
          <li key={key} {...props}>
            {[
              v.code,
              v.name,
              v.effectiveDate && '自 ' + v.effectiveDate + ' 起',
              v.endExclusive && '至 ' + v.endExclusive + ' 前',
            ]
              .filter(Boolean)
              .join(' ')}
          </li>
        )}
        getOptionLabel={(v) => [v.code, v.name].filter(Boolean).join(' ')}
        isOptionEqualToValue={(a, b) => a.id === b.id}
        onChange={(_, v) => onChange(v)}
        onInputChange={(_, v, why) => {
          if (why === 'input' || why === 'clear') setSearch(v);
        }}
        renderInput={(p) => <TextField {...p} variant="standard" error={!!list.error} helperText={list.error} />}
      />
    </ObjectDetailField>
  );
}
export function PlanScopeEditor({
  dimensions,
  value,
  onChange,
  error,
}: {
  dimensions: ScopeChoice['dimension'][];
  value: ScopeDraft;
  onChange: (v: ScopeDraft) => void;
  error?: string;
}) {
  return (
    <div className="object-detail-columns">
      <ObjectTextField
        id="plan-scope"
        select
        required
        label="适用方式"
        value={value.mode}
        error={!!error}
        helperText={error}
        onChange={(e) => onChange({ ...value, mode: e.target.value as ScopeDraft['mode'] })}
      >
        <MenuItem value="ENTERPRISE">企业通用</MenuItem>
        {(dimensions.length > 0 || value.mode === 'SELECTED') && <MenuItem value="SELECTED">指定范围</MenuItem>}
      </ObjectTextField>
      {value.mode === 'SELECTED' &&
        dimensions.flatMap((dimension) =>
          [false, true].map((exclude) => {
            const matches = (m: NamedChoice) => m.dimension === dimension && !!m.exclude === exclude;
            return (
              <ScopePicker
                key={dimension + exclude}
                dimension={dimension}
                exclude={exclude}
                value={value.choices.filter(matches)}
                onChange={(items) =>
                  onChange({ ...value, choices: [...value.choices.filter((m) => !matches(m)), ...items] })
                }
              />
            );
          }),
        )}
    </div>
  );
}
