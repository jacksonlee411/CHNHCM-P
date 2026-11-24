import { useEffect, useRef, useState } from 'react';
import { Alert, Button, MenuItem, Typography } from '@mui/material';
import { api, ApiError, sessionGuard, unwrap, type Models } from '@chnhcm/api-client';
import { ObjectTextField, ObjectDetailField } from '@chnhcm/ui';
import type { DataEditorProps, Definition, RecordView, SummaryView } from './dated/presentation';
import { columnStatuses, operatorsFor } from './BusinessUnitColumnFilter';
import { LocationRegionPicker } from './LocationRegionPicker';
const textFields = [{ key: 'addressLine', label: '详细地址', max: 1001, required: true }, { key: 'postalCode', label: '邮政编码', max: 20 }];
const canReadFullAddress = (fields: string[]) => ['geoAreaId', 'geoAreaPath', 'addressLine'].every(field => fields.includes(field));
const fullAddress = (path: Models['GeoAreaPathNode'][] | undefined, address: string | null | undefined) =>
    path && address ? path.map(node => node.name).join('') + address : undefined;
export const locationColumns: Pick<Definition, 'additionalColumns' | 'filterColumns' | 'hiddenColumns'> = {
    hiddenColumns: ['nameEn', 'postalCode'],
    additionalColumns: [
        { field: 'countryRegionId', headerName: '国家/地区', width: 150, sortable: false, filterable: false, valueGetter: (_, r) => r.values?.countryName ?? (r.values?.countryRegionId ? '名称无读取权限' : '') },
        ...[{ key: 'nameEn', label: '英文名称' }, ...textFields].map(f => ({ field: f.key, headerName: f.label, width: f.key === 'addressLine' ? 260 : 160, sortable: true, filterOperators: operatorsFor(f.key), valueGetter: (_: unknown, r: SummaryView) => r.values?.[f.key] })),
    ],
    filterColumns: [{ field: 'code', label: '编码' }, { field: 'name', label: '名称' }, { field: 'nameEn', label: '英文名称' }, ...textFields.map(f => ({ field: f.key, label: f.label })), { field: 'status', label: '状态', values: columnStatuses }, { field: 'effectiveDate', label: '生效日期' }],
};
export function LocationAddressView({ record, fields, restricted }: {
    record: RecordView | null;
    fields: string[];
    restricted: boolean;
}) {
    if (!record) return null;
    const path = record.values.geoAreaId === null ? [] : record.references?.geoAreaId?.path;
    return <>
        {fields.includes('countryRegionId') && <ObjectDetailField label="国家/地区">{record.references?.countryRegionId?.name || '名称无读取权限'}</ObjectDetailField>}
        {fields.includes('geoAreaPath') && <ObjectDetailField label="行政区划">{record.references?.geoAreaPath?.name || (record.values.geoAreaId ? '名称无读取权限' : '不适用')}</ObjectDetailField>}
        {textFields.filter(f => fields.includes(f.key)).map(f => <ObjectDetailField key={f.key} label={f.label}>{record.values[f.key] || '未提供'}</ObjectDetailField>)}
        {canReadFullAddress(fields) && <ObjectDetailField label="完整地址">{!restricted && fullAddress(path, record.values.addressLine) || '地址信息无读取权限'}</ObjectDetailField>}
        {restricted && <Typography variant="body2">地址名称无读取权限</Typography>}
    </>;
}
export function LocationAddress(props: DataEditorProps) {
    const input = props.context && 'kind' in props.context && props.context.kind === 'location' ? props.context.input : undefined;
    const identity = JSON.stringify([props.session.instanceId, props.session.subjectId, input]);
    const current = useRef(identity);
    current.current = identity;
    const checkedMissingCountry = useRef('');
    const [context, setContext] = useState<{
        key: string;
        value?: Models['LocationReferenceContext'];
        error?: string;
    }>(), [retry, setRetry] = useState(0);
    const [restricted, setRestricted] = useState(false), [countryChoice, setCountryChoice] = useState<Models['LocationCountryOption']>();
    const [chosen, setChosen] = useState<Models['GeoAreaCandidate']>();
    const country = props.values.countryRegionId ?? '', area = props.values.geoAreaId ?? '';
    const value = context?.key === identity ? context.value : undefined, error = context?.key === identity ? context.error : undefined;
    const writable = (f: string) => props.writes.includes(f) && !props.disabled;
    useEffect(() => { setContext(undefined); if (!input || props.disabled)
        return; const c = new AbortController(), same = sessionGuard(); const timer = setTimeout(() => { void api.POST('/api/v1/locations/reference-context', { params: { header: { 'X-CSRF-Token': props.session.csrf } }, body: input, signal: c.signal }).then(unwrap).then(v => { if (!c.signal.aborted && same() && current.current === identity) {
        setContext({ key: identity, value: v });
        if (country && !v.countries.some(option => option.id === country) && checkedMissingCountry.current !== `${identity}:${retry}`) {
            // A missing candidate can mean retired eligibility or withdrawn
            // read access. Ask the current detail once, without a reload loop.
            checkedMissingCountry.current = `${identity}:${retry}`;
            setCountryChoice(undefined);
            setChosen(undefined);
            setRestricted(true);
            props.onReferenceRestricted?.();
        }
    } }).catch(e => { if (!c.signal.aborted && same() && current.current === identity) {
        setContext({ key: identity, error: e instanceof Error ? e.message : '地址选项读取失败' });
        if (e instanceof ApiError && e.status === 403) {
            setRestricted(true);
            setChosen(undefined);
            setCountryChoice(undefined);
            props.onReferenceRestricted?.();
        }
    } }); }, 150); return () => { clearTimeout(timer); c.abort(); }; }, [identity, retry, props.disabled]);
    const selectedCountry = value?.countries.find(v => v.id === country);
    // Unchanged adoption must display the exact saved label, even after a
    // directory rename or when current candidate labels are more permissive.
    const preservesAdoption = value?.newAdoption === false;
    const originalCountry = props.baseline?.references?.countryRegionId;
    // Only a fresh authorized detail may restore a saved label after clearing
    // it. A failed read leaves the trimmed baseline and restriction intact.
    useEffect(() => {
        if (props.baseline?.references?.countryRegionId?.name) setRestricted(false);
    }, [props.baseline]);
    const countryName = preservesAdoption
        ? (!restricted && originalCountry?.id === country ? originalCountry.name : undefined)
        : selectedCountry?.name || (countryChoice?.id === country ? countryChoice.name : undefined) || (!restricted && !value && originalCountry?.id === country ? originalCountry.name : undefined);
    const countryOptions = value?.countries ?? [];
    const originalArea = props.baseline?.references?.geoAreaId;
    const areaLabel = restricted ? undefined : preservesAdoption
        ? (originalArea?.id === area ? originalArea.name : undefined)
        : chosen?.id === area ? chosen.path.filter(n => n.id !== country).map(n => n.name).join(' / ') : (originalArea?.id === area ? originalArea.name : undefined);
    const originalPath = originalArea?.id === area ? originalArea.path : undefined;
    const areaPath = restricted ? undefined : preservesAdoption ? originalPath : chosen?.id === area ? chosen.path : originalPath;
    const requiresArea = selectedCountry?.requiresArea ?? !!area;
    // Creation exposes writable inputs, not stored-record read fields. Its
    // preview uses only the entered address and the authorized candidate path.
    const showFullAddress = input?.action === 'CREATE'
        ? ['countryRegionId', 'geoAreaId', 'addressLine'].every(field => props.writes.includes(field))
        : canReadFullAddress(props.fields);
    const addressPath = area ? areaPath?.length ? areaPath.filter(node => node.id !== country) : undefined
        : country && (selectedCountry?.requiresArea === false || originalCountry?.id === country && props.baseline?.values?.geoAreaId === null) ? [] : undefined;
    return <>
 {props.fields.includes('countryRegionId') && <ObjectTextField select fullWidth id={`${props.prefix}-countryRegionId`} label="国家/地区" value={country} disabled={!writable('countryRegionId')} onChange={e => { setCountryChoice(countryOptions.find(c => c.id === e.target.value)); props.onChange('countryRegionId', e.target.value); props.onChange('geoAreaId', ''); setChosen(undefined); }} error={props.fieldError.replace('changes.', '') === 'countryRegionId'} helperText={props.fieldError.replace('changes.', '') === 'countryRegionId' ? props.error : undefined}>
 <MenuItem value="">请选择国家/地区</MenuItem>{country && !countryOptions.some(c => c.id === country) && <MenuItem value={country}>{countryName || '名称无读取权限'}</MenuItem>}{countryOptions.map(c => <MenuItem key={c.id} value={c.id}>{preservesAdoption && c.id === country ? countryName || '名称无读取权限' : c.name}</MenuItem>)}
 </ObjectTextField>}
 {requiresArea && props.fields.includes('geoAreaId') && <LocationRegionPicker id={`${props.prefix}-geoAreaId`} identity={identity}
   countryId={country} countryName={countryName || '所选国家/地区'}
   adoptedDate={value?.adoptedDate} areaId={area} label={area ? areaLabel || '名称无读取权限' : ''} path={areaPath}
   disabled={!writable('geoAreaId') || !value} error={props.fieldError.replace('changes.', '') === 'geoAreaId' ? props.error : undefined}
   onSelect={candidate => { props.onChange('geoAreaId', candidate?.id ?? ''); setChosen(candidate ?? undefined); if (candidate) setRestricted(false); }}
   onDenied={() => { setRestricted(true); setChosen(undefined); setCountryChoice(undefined); props.onReferenceRestricted?.(); }} />}
 {!requiresArea && country && <Typography variant="body2">此国家/地区使用基础地址。</Typography>}
 {error && <Alert severity="warning" action={<Button onClick={() => setRetry(n => n + 1)}>重试</Button>}>{error}</Alert>}
 {textFields.filter(f => props.fields.includes(f.key)).map(f => <ObjectTextField key={f.key} id={`${props.prefix}-${f.key}`} fullWidth label={f.label} required={f.required} value={props.values[f.key] ?? ''} disabled={!writable(f.key)} onChange={e => props.onChange(f.key, e.target.value)} error={props.fieldError.replace('changes.', '') === f.key} helperText={props.fieldError.replace('changes.', '') === f.key ? props.error : undefined}/>)}
 {showFullAddress && <ObjectDetailField label="完整地址">{restricted ? '地址信息无读取权限' : fullAddress(addressPath, props.values.addressLine) || '待完善地址'}</ObjectDetailField>}
 </>;
}
