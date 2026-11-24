import { Typography, Stack, Button } from '@mui/material';
import { Link } from 'react-router-dom';
import { useContext } from 'react';
import { ObjectDetailField } from '@chnhcm/ui';
import { ListNavigation } from './dated/listNavigation';
import { geoLevels, GeoAreaScope, childQuery, levelLabel } from './GeoAreaFilters';
import { api, ApiError, unwrap, sessionGuard, type Models } from '@chnhcm/api-client';
import { submissionChannel } from '../../shared/dated/submission';
import { ColumnFilterEditor, columnFilterQuery, columnStatuses, operatorsFor } from './BusinessUnitColumnFilter';
import { mapResponse, type Definition, type DataFields, type Result, type DetailView } from './dated/presentation';
export const GEOAREA_PATH = '/settings/basic-data/geo-areas';
const record = (r: Models['GeoAreaRecord']) => { const { nameEn, ...metadata } = r; return { ...metadata, values: { name: r.name, nameEn } }; };
const changes = (c?: DataFields): Models['GeoAreaChangesInput'] => ({ name: c?.name ?? undefined, nameEn: c?.nameEn });
const result = (r: Models['GeoAreaResult']): Result => ({ ...r, objectId: r.geoAreaId });
const detail = (d: Models['GeoAreaDetail']): DetailView => ({ ...d,
    basicFields: (_record, fields, restricted) => {
        const r = d.selectedRecord;
        const path = restricted || !fields.includes('geoAreaPath') ? undefined : d.geoAreaPath;
        return <>
            {fields.includes('nature') && r?.nature && <ObjectDetailField label="性质">{r.nature === 'address_region' ? '地址分区' : r.nature === 'administrative' ? '行政区划' : '未提供'}</ObjectDetailField>}
            {fields.includes('level') && r?.level && <ObjectDetailField label="层级">{levelLabel(r.level)}</ObjectDetailField>}
            {fields.includes('scheme') && r?.scheme && <ObjectDetailField label="代码方案">{r.scheme}</ObjectDetailField>}
            {fields.includes('countryRegionId') && r?.countryRegionId && <ObjectDetailField label="国家/地区">{!restricted && d.countryRegionName || '名称不可读'}</ObjectDetailField>}
            {fields.includes('geoAreaPath') && d.pathDate && <ObjectDetailField label={d.readOnly ? '原采用路径' : '路径'}>
                {path?.length ? <>{path.map(n => n.name).join(' / ')}（{d.pathDate}）</> : '当前无权读取完整路径'}
            </ObjectDetailField>}
        </>;
    },
    selectedRecord: d.selectedRecord && record(d.selectedRecord), timeline: d.timeline.map(record),
    additionContext: d.additionContext && { ...d.additionContext, defaults: record(d.additionContext.defaults), maxEffectiveDate: d.additionContext.maxEffectiveDate ?? undefined },
    lifecycleContext: d.lifecycleContext && { ...d.lifecycleContext, defaults: record(d.lifecycleContext.defaults), maxEffectiveDate: d.lifecycleContext.maxEffectiveDate ?? undefined },
});
async function header(signal: AbortSignal) {
    const same = sessionGuard();
    const v = await api.GET('/api/v1/session', { signal });
    signal.throwIfAborted();
    if (!same())
        throw new Error('会话已变化，请重新查询。');
    return { 'X-CSRF-Token': unwrap(v).csrf };
}
export const geoArea: Definition = {
    kind: 'organization.geoArea', basePath: GEOAREA_PATH, label: '行政区划与地址分区', listLabel: '行政区划', prefix: 'geoArea', statusReadField: 'state',
    fields: [{ key: 'name', label: '显示名称', required: true }, { key: 'nameEn', label: '英文名称' }],
    ListScope: GeoAreaScope,
    columnChoices: context => {
        const fields = context?.capabilities.readFields ?? [];
        return [
            ...(fields.includes('countryRegionId') && context?.filterCountries?.length ? [{field:'countryRegionId',label:'国家/地区',values:context.filterCountries.map(c => ({value:c.id,label:c.name}))}] : []),
            ...(fields.includes('level') ? [{field:'level',label:'层级',values:geoLevels}] : []),
        ];
    },
    scopedEmpty: query => query.has('parentAreaId') ? '当前范围没有可显示的直接下级' : undefined,
    summary: row => <Stack sx={{ px: 1, pb: 1, whiteSpace: 'normal', overflowWrap: 'anywhere' }}>
        <Stack direction="row" gap={1} flexWrap="wrap">
            {row.values?.countryRegionId && row.values.countryRegionName && <ColumnValue field="countryRegionId" label="国家/地区" value={row.values.countryRegionId} text={row.values.countryRegionName} />}
            {row.values?.level && <ColumnValue field="level" label="层级" value={row.values.level} text={levelLabel(row.values.level)} />}
        </Stack>
        {row.secondary && <Typography variant="body2">{row.secondary}</Typography>}
        {row.values?.parentAreaId !== undefined && <Children id={row.id} />}
    </Stack>,
    additionalColumns: [
        {field:'countryRegionId', headerName:'国家/地区', width:136, sortable:false, filterOperators:operatorsFor('countryRegionId'), renderCell:p => p.row.values?.countryRegionName ?? (p.row.values?.countryRegionId ? '名称不可读' : null)},
        {field:'geoAreaPath', headerName:'路径', flex:1.5, minWidth:230, sortable:false, filterable:false, renderCell:p => <Typography variant="body2" sx={{whiteSpace:'normal',overflowWrap:'anywhere'}}>{p.row.secondary}</Typography>},
        {field:'level', headerName:'层级', width:130, sortable:false, filterOperators:operatorsFor('level'), renderCell:p => p.row.values?.level ? levelLabel(p.row.values.level) : null},
        {field:'children', headerName:'下级', width:100, sortable:false, filterable:false, renderCell:p => p.row.values?.parentAreaId !== undefined ? <Children id={p.row.id}/> : null},
    ],
    filterColumns: [{ field: 'code', label: '编码' }, { field: 'name', label: '名称' }, { field: 'status', label: '状态', values: columnStatuses }],
    submissions: submissionChannel<string, Result>('organization.geoArea', 'chnhcm-pending-geoArea-submissions'),
    resolve: (session, pending) => mapResponse(api.POST('/api/v1/geo-areas/submissions/{id}/resolve', { params: { path: { id: pending.submissionId }, header: { 'X-CSRF-Token': session.csrf } } }), result),
    resultView: r => ({ kind: 'organization.geoArea', submissionId: r.submissionId, outcome: r.outcome, objectId: r.objectId ?? undefined, sourceVersion: r.sourceVersion ?? undefined, selectedRecordId: r.selectedRecordId ?? undefined, lines: [] }),
    navigation: async (params, anchorId, direction, signal) => api.POST('/api/v1/geo-areas/navigation', { signal, params: { header: await header(signal) }, body: {
            anchorId, direction,
            query: geoQuery(params),
        } }),
    api: {
        context: signal => api.GET('/api/v1/geo-areas/context', { signal }),
        reasons: (operationType, signal) => api.GET('/api/v1/geo-areas/operation-reasons', { params: { query: { operationType } }, signal }),
        list: (query, requestedPage = Number(query.get('page') ?? 1), signal) => mapResponse(api.GET('/api/v1/geo-areas', { params: { query: {
                    ...geoListQuery(query), page: requestedPage, ...columnFilterQuery(query), filterField: columnFilterQuery(query).filterField as NonNullable<Models['GeoAreaQuery']['filter']>['field'], filterOperator: columnFilterQuery(query).filterOperator as NonNullable<Models['GeoAreaQuery']['filter']>['operator'],
                } }, signal }), v => ({ ...v, items: v.items.map(r => ({ ...r, values: { countryRegionId: r.countryRegionId, countryRegionName: r.countryRegionName, level: r.level, parentAreaId: r.parentAreaId }, secondary: r.geoAreaPath?.map(n => n.name).join(' / ') })) })),
        detail: (id, query, signal) => mapResponse(api.GET('/api/v1/geo-areas/{id}', { params: { path: { id }, query }, signal }), detail),
        changes: (id, query, signal) => mapResponse(api.GET('/api/v1/geo-areas/{id}/changes', { params: { path: { id }, query }, signal }), v => ({ ...v, items: v.items.map(c => ({ ...c, items: c.items.map(i => ({ ...i, beforeContext: i.beforeContext && record(i.beforeContext), afterContext: i.afterContext && record(i.afterContext) })) })) })),
        preview: (id, csrf, body) => mapResponse(api.POST('/api/v1/geo-areas/{id}/preview', { params: { path: { id }, header: { 'X-CSRF-Token': csrf } }, body: {
                action: body.action, expectedSourceVersion: body.expectedSourceVersion,
                ...(body.action === 'CORRECT' ? { recordId: body.recordId, expectedRecordRevision: body.expectedRecordRevision, changes: changes(body.changes) } : {
                    anchorRecordId: body.anchorRecordId, effectiveDate: body.effectiveDate, operationType: body.operationType, operationReason: body.operationReason,
                    ...(body.action !== 'DEACTIVATE' && body.changes ? { changes: changes(body.changes) } : {}),
                }),
            } }), v => ({ ...v, afterRecord: v.afterRecord && record(v.afterRecord) })),
        record(id, command, header) {
            switch (command.action) {
                case 'ADD': return mapResponse(api.POST('/api/v1/geo-areas/{id}/records', { params: { path: { id }, header }, body: { ...command.body, changes: changes(command.body.changes) } }), result);
                case 'CORRECT': return mapResponse(api.POST('/api/v1/geo-areas/{id}/records/{recordId}/correct', { params: { path: { id, recordId: command.recordId }, header }, body: { expectedSourceVersion: command.body.expectedSourceVersion, expectedRecordRevision: command.body.expectedRecordRevision, changes: changes(command.body.changes) } }), result);
                case 'DEACTIVATE': return mapResponse(api.POST('/api/v1/geo-areas/{id}/deactivate', { params: { path: { id }, header }, body: command.body }), result);
                case 'REACTIVATE': return mapResponse(api.POST('/api/v1/geo-areas/{id}/reactivate', { params: { path: { id }, header }, body: { ...command.body, changes: command.body.changes && changes(command.body.changes) } }), result);
                case 'DELETE': throw new ApiError('M02_CAPABILITY_UNSUPPORTED', '行政区划与地址分区不提供删除记录', 400);
            }
        },
    },
};
function geoQuery(params: URLSearchParams): Models['GeoAreaQuery'] {
    const f = columnFilterQuery(params);
    return { countryRegionId: params.get('countryRegionId') || undefined, parentAreaId: params.get('parentAreaId') || undefined, level: params.get('level') || undefined, q: params.get('q') || undefined, status: (params.get('status') || 'all') as Models['GeoAreaQuery']['status'], sort: (params.get('sort') || 'code') as Models['GeoAreaQuery']['sort'], pageSize: Number(params.get('pageSize') || 20) as 20 | 50 | 100,
        filter: f.filterField ? { field: f.filterField as NonNullable<Models['GeoAreaQuery']['filter']>['field'], operator: (f.filterOperator || '') as NonNullable<Models['GeoAreaQuery']['filter']>['operator'], value: f.filterValue || '' } : undefined };
}
function ColumnValue({ field, label, value, text }: { field: string; label: string; value: string; text: string }) {
    const open = useContext(ColumnFilterEditor);
    return <Button size="small" aria-label={`筛选${label}：${text}`} onClick={event => {
        event.stopPropagation();
        open({ field, initialValue: value, source: event.currentTarget });
    }}>{text}</Button>;
}
function Children({ id }: { id: string }) {
    const navigation = useContext(ListNavigation)!;
    const source = new URL(navigation.rowBack(id), window.location.origin);
    const target = childQuery(source.searchParams, id);
    return <Button component={Link} data-child-link to={GEOAREA_PATH + '?' + target} size="small"
        onClick={event => {
            event.stopPropagation();
            if (event.button === 0 && !event.ctrlKey && !event.metaKey && !event.shiftKey && !event.altKey)
                navigation.rememberScroll(id, true);
        }}>查看下级</Button>;
}
function geoListQuery(params: URLSearchParams) { const { filter, ...query } = geoQuery(params); return query; }
