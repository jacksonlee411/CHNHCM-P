import { useEffect, useRef, useState } from 'react';
import { Alert, Button, MenuItem, Stack, TextField } from '@mui/material';
import { api, ApiError, sessionGuard, unwrap, type Models } from '@chnhcm/api-client';
import { ObjectTextField, ObjectDetailField, ObjectDetailSection } from '@chnhcm/ui';
import type { DataEditorProps, Definition, RecordView } from './dated/presentation';
import { columnStatuses, legalEntitySubjectTypes, operatorsFor } from './BusinessUnitColumnFilter';

export const subjectTypeLabel = (value?: string | null) => legalEntitySubjectTypes.find(v => v.value === value)?.label ?? '未提供';
const textFields = [{ key: 'registrationType', label: '登记类型' }, { key: 'registrationNumber', label: '登记号码' }, { key: 'establishingOrganization', label: '设立单位说明' }];
export const legalEntityColumns: Pick<Definition, 'additionalColumns' | 'filterColumns' | 'hiddenColumns'> = {
  hiddenColumns: ['nameEn', 'establishingOrganization'],
  additionalColumns: [
    { field: 'subjectType', headerName: '主体类别', width: 150, type: 'singleSelect', valueOptions: legalEntitySubjectTypes, sortable: true, filterOperators: operatorsFor('subjectType'), valueGetter: (_, r) => r.values?.subjectType },
    ...[{ key: 'nameEn', label: '英文名称' }, ...textFields].map(f => ({ field: f.key, headerName: f.label, width: 180, sortable: true, filterOperators: operatorsFor(f.key), valueGetter: (_: unknown, r: import('./dated/presentation').SummaryView) => r.values?.[f.key] })),
  ],
  filterColumns: [{ field: 'code', label: '编码' }, { field: 'name', label: '名称' }, { field: 'nameEn', label: '英文名称' }, { field: 'subjectType', label: '主体类别', values: legalEntitySubjectTypes }, ...textFields.map(f => ({ field: f.key, label: f.label })), { field: 'status', label: '状态', values: columnStatuses }, { field: 'effectiveDate', label: '生效日期' }],
};

export function LegalEntityFieldsView({ record, fields }: { record: RecordView | null; fields: string[]; restricted: boolean }) {
  if (!record) return null;
  return <ObjectDetailSection title="主体资料">
    {fields.includes('countryRegionId') && <ObjectDetailField label="国家/地区">{record.references?.countryRegionId?.name || '名称无读取权限'}</ObjectDetailField>}
    {fields.includes('subjectType') && <ObjectDetailField label="主体类别">{subjectTypeLabel(record.values.subjectType)}</ObjectDetailField>}
    {textFields.filter(f => fields.includes(f.key)).map(f => <ObjectDetailField key={f.key} label={f.label}>{record.values[f.key] || '未提供'}</ObjectDetailField>)}
  </ObjectDetailSection>;
}

export function LegalEntityFilters({ query, fields, onChange }: Parameters<NonNullable<Definition['ListFilters']>>[0]) {
  const [countries, setCountries] = useState<Models['CountrySummary'][]>([]), [error, setError] = useState(''), [retry, setRetry] = useState(0);
  useEffect(() => {
    const c = new AbortController(), same = sessionGuard(); setCountries([]); setError('');
    if (fields.includes('countryRegionId')) void (async () => {
      try {
        const all: Models['CountrySummary'][] = [];
        for (let page = 1; ; page++) {
          const v = unwrap(await api.GET('/api/v1/country-regions', { params: { query: { page, pageSize: 100 } }, signal: c.signal }));
          all.push(...v.items); if (page * 100 >= v.count) break;
        }
        if (!c.signal.aborted && same()) setCountries(all);
      } catch (e) { if (!c.signal.aborted && same()) setError(e instanceof Error ? e.message : '国家/地区读取失败'); }
    })();
    return () => c.abort();
  }, [fields.join('|'), retry]);
  return fields.includes('countryRegionId') && <Stack gap={1} sx={{ mb: 1 }}>
    <TextField select size="small" label="国家/地区" value={query.get('countryRegionId') ?? ''} onChange={e => onChange({ countryRegionId: e.target.value })} sx={{ width: 220, maxWidth: '100%' }}>
      <MenuItem value="">全部国家/地区</MenuItem>
      {query.get('countryRegionId') && !countries.some(c => c.id === query.get('countryRegionId')) && <MenuItem value={query.get('countryRegionId')!}>已选国家/地区</MenuItem>}
      {countries.map(c => <MenuItem key={c.id} value={c.id}>{c.name ?? '名称无读取权限'}</MenuItem>)}
    </TextField>
    {error && <Alert severity="error" action={<Button onClick={() => setRetry(n => n + 1)}>重试</Button>}>{error}</Alert>}
  </Stack>;
}

export function LegalEntityFields(props: DataEditorProps) {
  const input = props.context && 'kind' in props.context && props.context.kind === 'legalentity' ? props.context.input : undefined;
  const identity = JSON.stringify([props.session.instanceId, props.session.subjectId, input]);
  const current = useRef(identity); current.current = identity;
  const [context, setContext] = useState<{ key: string; value?: Models['LegalEntityReferenceContext']; error?: string }>();
  const [retry, setRetry] = useState(0), [restricted, setRestricted] = useState(false);
  const [choice, setChoice] = useState<Models['LegalEntityCountryOption']>();
  const checkedMissing = useRef('');
  const country = props.values.countryRegionId ?? '';
  const value = context?.key === identity ? context.value : undefined, error = context?.key === identity ? context.error : undefined;
  const writable = (key: string) => props.writes.includes(key) && !props.disabled;
  useEffect(() => {
    setContext(undefined);
    if (!input || props.disabled || !props.fields.includes('countryRegionId')) return;
    const c = new AbortController(), same = sessionGuard();
    const timer = setTimeout(() => {
      void api.POST('/api/v1/legal-entities/reference-context', { params: { header: { 'X-CSRF-Token': props.session.csrf } }, body: input, signal: c.signal }).then(unwrap).then(v => {
        if (c.signal.aborted || !same() || current.current !== identity) return;
        setContext({ key: identity, value: v });
        const unreadable = !v.newAdoption ? v.countryRestricted : country && !v.countries.some(option => option.id === country);
        if (unreadable && checkedMissing.current !== `${identity}:${retry}`) {
          checkedMissing.current = `${identity}:${retry}`; setChoice(undefined); setRestricted(true); props.onReferenceRestricted?.();
        }
      }).catch(e => {
        if (c.signal.aborted || !same() || current.current !== identity) return;
        setContext({ key: identity, error: e instanceof Error ? e.message : '国家选项读取失败' });
        if (e instanceof ApiError && e.status === 403) { setRestricted(true); setChoice(undefined); props.onReferenceRestricted?.(); }
      });
    }, 150);
    return () => { clearTimeout(timer); c.abort(); };
  }, [identity, retry, props.disabled, props.fields.join('|')]);
  useEffect(() => { if (props.baseline?.references?.countryRegionId?.name) setRestricted(false); }, [props.baseline]);
  const original = props.baseline?.references?.countryRegionId;
  const preserves = value?.newAdoption === false;
  const name = preserves ? value?.countryName
    : value?.countries.find(v => v.id === country)?.name || (choice?.id === country ? choice.name : undefined) || (!restricted && !value && original?.id === country ? original.name : undefined);
  const options = value?.countries ?? [];
  const fieldError = props.fieldError.replace('changes.', '');
  return <>
    {props.fields.includes('countryRegionId') && <ObjectTextField select fullWidth required id={`${props.prefix}-countryRegionId`} label="国家/地区" value={country} disabled={!writable('countryRegionId')} onChange={e => { setChoice(options.find(v => v.id === e.target.value)); props.onChange('countryRegionId', e.target.value); }} error={fieldError === 'countryRegionId'} helperText={fieldError === 'countryRegionId' ? props.error : undefined}>
      <MenuItem value="">请选择国家/地区</MenuItem>
      {country && !options.some(c => c.id === country) && <MenuItem value={country}>{name || '名称无读取权限'}</MenuItem>}
      {options.map(c => <MenuItem key={c.id} value={c.id}>{preserves && c.id === country ? name || '名称无读取权限' : c.name}</MenuItem>)}
    </ObjectTextField>}
    {error && <Alert severity="warning" action={<Button onClick={() => setRetry(n => n + 1)}>重试</Button>}>{error}</Alert>}
    {props.fields.includes('subjectType') && <ObjectTextField select fullWidth required id={`${props.prefix}-subjectType`} label="主体类别" value={props.values.subjectType ?? ''} disabled={!writable('subjectType')} onChange={e => props.onChange('subjectType', e.target.value)} error={fieldError === 'subjectType'} helperText={fieldError === 'subjectType' ? props.error : undefined}>
      <MenuItem value="">请选择主体类别</MenuItem>{legalEntitySubjectTypes.map(v => <MenuItem key={v.value} value={v.value}>{v.label}</MenuItem>)}
    </ObjectTextField>}
    {textFields.filter(f => props.fields.includes(f.key)).map(f => <ObjectTextField key={f.key} fullWidth id={`${props.prefix}-${f.key}`} label={f.label} value={props.values[f.key] ?? ''} disabled={!writable(f.key)} onChange={e => props.onChange(f.key, e.target.value)} error={fieldError === f.key} helperText={fieldError === f.key ? props.error : undefined} />)}
  </>;
}
