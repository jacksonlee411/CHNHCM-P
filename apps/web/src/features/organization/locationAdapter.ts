import { createElement } from 'react';
import { recoveryMessages } from './dated/recoveryMessages';
import { api, unwrap, sessionGuard, type Models } from '@chnhcm/api-client';
import { businessDate } from '@chnhcm/ui';
import { submissionChannel } from '../../shared/dated/submission';
import { mapResponse, type Definition, type DataFields, type PreviewInput, type Result, type RecordView } from './dated/presentation';
import { LocationAddress, locationColumns, LocationAddressView } from './LocationAddress';
export const LOCATION_PATH = '/settings/organization/locations';
const changes = (c?: DataFields): Models['LocationChangesInput'] | undefined => c && Object.fromEntries(Object.entries(c).filter(([k, v]) => ['name', 'nameEn', 'countryRegionId', 'geoAreaId', 'addressLine', 'postalCode'].includes(k) && v !== undefined).map(([k, v]) => [k, ['nameEn', 'geoAreaId', 'postalCode'].includes(k) && v === '' ? null : v])) as Models['LocationChangesInput'];
const result = (r: Models['LocationResult']): Result => ({ ...r, objectId: r.locationId });
const record = (r: Models['LocationRecord']): RecordView => ({ ...r, values: { name: r.name, nameEn: r.nameEn, countryRegionId: r.countryRegionId, geoAreaId: r.geoAreaId, addressLine: r.addressLine, postalCode: r.postalCode }, references: { countryRegionId: r.countryRegionId ? { id: r.countryRegionId, name: r.countryName } : null, geoAreaId: r.geoAreaId ? { id: r.geoAreaId, name: r.geoAreaPath?.map(n => n.name).join(' / '), path: r.geoAreaPath } : null, geoAreaPath: r.geoAreaId ? { id: r.geoAreaId, name: r.geoAreaPath?.map(n => n.name).join(' / ') } : null } });
const list = (v: Models['LocationList']) => ({ ...v, items: v.items.map(r => ({ ...r, values: { nameEn: r.nameEn, countryRegionId: r.countryRegionId, countryName: r.countryName, addressLine: r.addressLine, postalCode: r.postalCode } })) });
const detail = (d: Models['LocationDetail']) => ({ ...d, selectedRecord: d.selectedRecord && record(d.selectedRecord), timeline: d.timeline.map(record), additionContext: d.additionContext && { ...d.additionContext, maxEffectiveDate: d.additionContext.maxEffectiveDate ?? undefined, defaults: record(d.additionContext.defaults) }, lifecycleContext: d.lifecycleContext && { ...d.lifecycleContext, maxEffectiveDate: d.lifecycleContext.maxEffectiveDate ?? undefined, defaults: record(d.lifecycleContext.defaults) }, pairDeletion: d.pairDeletion && { ...d.pairDeletion, deactivation: record(d.pairDeletion.deactivation), reactivation: record(d.pairDeletion.reactivation), changes: d.pairDeletion.changes.map(c => ({ ...c, start: c.start ?? undefined, before: record(c.before), after: record(c.after) })) }, basicFields: (r: RecordView, fields: string[], restricted: boolean) => createElement(LocationAddressView, { record: r, fields, restricted }) });
function query(params: URLSearchParams): Models['LocationSelectionQuery'] { return { q: params.get('q') ?? undefined, status: (params.get('status') ?? 'all') as Models['LocationSelectionQuery']['status'], countryRegionId: params.get('countryRegionId') ?? undefined, filterField: params.get('filterField') as Models['LocationSelectionQuery']['filterField'] ?? undefined, filterOperator: params.get('filterOperator') as Models['LocationSelectionQuery']['filterOperator'] ?? undefined, filterValue: params.get('filterValue') ?? undefined }; }
function selectionQuery(q: import('./dated/presentation').SelectionQuery): Models['LocationSelectionQuery'] { return { q: q.q, status: q.status, countryRegionId: q.countryRegionId, filterField: q.filterField as Models['LocationSelectionQuery']['filterField'], filterOperator: q.filterOperator, filterValue: q.filterValue }; }
function preview(body: PreviewInput): Models['LocationPreviewInput'] {
    const expectedSourceVersion = body.expectedSourceVersion;
    switch (body.action) {
        case 'ADD': return { action: body.action, expectedSourceVersion, anchorRecordId: body.anchorRecordId ?? '', effectiveDate: body.effectiveDate ?? '', operationType: 'CHANGE', operationReason: body.operationReason ?? '', changes: changes(body.changes) ?? {} };
        case 'CORRECT': return { action: body.action, expectedSourceVersion, recordId: body.recordId ?? '', expectedRecordRevision: body.expectedRecordRevision ?? '', newEffectiveDate: body.newEffectiveDate, classification: body.classification, changes: changes(body.changes) };
        case 'DELETE': return { action: body.action, expectedSourceVersion, recordId: body.recordId ?? '', expectedRecordRevision: body.expectedRecordRevision ?? '' };
        case 'REDATE': return { action: body.action, expectedSourceVersion, recordId: body.recordId ?? '', expectedRecordRevision: body.expectedRecordRevision ?? '', newEffectiveDate: body.newEffectiveDate ?? '' };
        case 'DEACTIVATE': return { action: body.action, expectedSourceVersion, anchorRecordId: body.anchorRecordId ?? '', effectiveDate: body.effectiveDate ?? '', operationType: 'DEACTIVATE', operationReason: body.operationReason ?? '' };
        case 'REACTIVATE': return { action: body.action, expectedSourceVersion, anchorRecordId: body.anchorRecordId ?? '', effectiveDate: body.effectiveDate ?? '', operationType: 'REACTIVATE', operationReason: body.operationReason ?? '', changes: changes(body.changes) };
    }
}
async function header(signal: AbortSignal) {
    const same = sessionGuard();
    const response = await api.GET('/api/v1/session', { signal });
    signal.throwIfAborted();
    if (!same())
        throw new Error('会话已变化，请重新查询。');
    return { 'X-CSRF-Token': unwrap(response).csrf };
}
export const location: Definition = {
    ...locationColumns,
    kind: 'organization.location', basePath: LOCATION_PATH, label: '地点', prefix: 'location',
    statusReadField: 'status', logStateField: 'status', sortableFields: ['code', 'name', 'nameEn', 'addressLine', 'postalCode', 'status', 'effectiveDate'],
    fields: [{ key: 'name', label: '地点名称', required: true }, { key: 'nameEn', label: '英文名称' }],
    summaryFields: [{ key: 'name', label: '地点名称' }, { key: 'nameEn', label: '英文名称' }, { key: 'countryRegionId', label: '国家/地区', render: r => r.references?.countryRegionId?.name || '名称无读取权限' }, { key: 'geoAreaId', label: '行政区划', render: r => r.references?.geoAreaPath?.name || (r.values?.geoAreaId ? '名称无读取权限' : '不适用') }, { key: 'addressLine', label: '详细地址' }, { key: 'postalCode', label: '邮政编码' }],
    referenceContext: (id, body) => ({ kind: 'location', input: { action: body.action, locationId: id, expectedSourceVersion: body.expectedSourceVersion, recordId: body.recordId ?? undefined, expectedRecordRevision: body.expectedRecordRevision ?? undefined, anchorRecordId: body.anchorRecordId ?? undefined, effectiveDate: body.effectiveDate ?? undefined, newEffectiveDate: body.newEffectiveDate ?? undefined, classification: body.classification ?? undefined, changes: changes(Object.fromEntries(Object.entries(body.changes ?? {}).filter(([k]) => ['countryRegionId', 'geoAreaId'].includes(k)))) } }),
    createReferenceContext: (date, values) => ({ kind: 'location', input: { action: 'CREATE', effectiveDate: date, changes: changes({ countryRegionId: values.countryRegionId, geoAreaId: values.geoAreaId }) } }),
    Relations: LocationAddress,
    submissions: submissionChannel<string, Result>('organization.location', 'chnhcm-pending-location-submissions'),
    resolve: (session, pending) => mapResponse(api.POST('/api/v1/locations/submissions/{id}/resolve', { params: { path: { id: pending.submissionId }, header: { 'X-CSRF-Token': session.csrf } } }), result),
    resultView: r => ({ kind: 'organization.location', submissionId: r.submissionId, outcome: r.outcome, objectId: r.objectId ?? undefined, sourceVersion: r.sourceVersion ?? r.observedSourceVersion ?? undefined,
        selectedRecordId: (r.action === 'DELETE' ? r.targetRecordId : r.selectedRecordId) ?? undefined,
        lines: [...(r.items ?? []).map(i => `${i.outcome === 'no_change' ? '输入无变化' : '已保存'}${i.effectiveDate ? ` · ${businessDate(i.effectiveDate)}` : ''}${i.afterRecord?.name ? ` · ${i.afterRecord.name}` : ''}`),
            ...recoveryMessages(r.failureDetails)],
    }),
    api: {
        context: signal => api.GET('/api/v1/locations/context', { signal }),
        reasons: (operationType, signal) => api.GET('/api/v1/locations/operation-reasons', { params: { query: { operationType } }, signal }),
        list: (params, requestedPage = Number(params.get('page') ?? 1), signal) => mapResponse(api.GET('/api/v1/locations', { params: { query: { ...query(params), sort: (params.get('sort') ?? undefined) as 'code' | undefined, page: requestedPage, pageSize: Number(params.get('pageSize') ?? 20) as 20 | 50 | 100 } }, signal }), list),
        detail: (id, query, signal) => {
            return mapResponse(api.GET('/api/v1/locations/{id}', { params: { path: { id }, query: { sourceVersion: query.sourceVersion, selectedRecordId: query.selectedRecordId } }, signal }), detail);
        },
        create: (body, header) => mapResponse(api.POST('/api/v1/locations', { params: { header }, body: { code: body.code, name: body.name, nameEn: body.nameEn || undefined, countryRegionId: body.countryRegionId ?? '', geoAreaId: body.geoAreaId || undefined, addressLine: body.addressLine ?? '', postalCode: body.postalCode || undefined, effectiveDate: body.effectiveDate, operationType: 'CREATE', operationReason: body.operationReason } }), result),
        preview: (id, csrf, body) => mapResponse(api.POST('/api/v1/locations/{id}/preview', { params: { path: { id }, header: { 'X-CSRF-Token': csrf } }, body: preview(body) }), v => ({ ...v, afterRecord: v.afterRecord ? record(v.afterRecord) : undefined })),
        changes: (id, query, signal) => mapResponse(api.GET('/api/v1/locations/{id}/changes', { params: { path: { id }, query }, signal }), v => ({ ...v, items: v.items.map(c => ({ ...c, items: c.items.map(i => ({ ...i, beforeContext: i.beforeContext && record(i.beforeContext), afterContext: i.afterContext && record(i.afterContext) })) })) })),
        record(id, command, header) {
            switch (command.action) {
                case 'ADD': return mapResponse(api.POST('/api/v1/locations/{id}/records', { params: { path: { id }, header }, body: { ...command.body, changes: changes(command.body.changes) } }), result);
                case 'CORRECT': return mapResponse(api.POST('/api/v1/locations/{id}/records/{recordId}/correct', { params: { path: { id, recordId: command.recordId }, header }, body: { ...command.body, changes: changes(command.body.changes) } }), result);
                case 'DELETE': return mapResponse(api.POST('/api/v1/locations/{id}/records/{recordId}/delete', { params: { path: { id, recordId: command.recordId }, header }, body: command.body }), result);
                case 'DEACTIVATE': return mapResponse(api.POST('/api/v1/locations/{id}/deactivate', { params: { path: { id }, header }, body: command.body }), result);
                case 'REACTIVATE': return mapResponse(api.POST('/api/v1/locations/{id}/reactivate', { params: { path: { id }, header }, body: { ...command.body, changes: changes(command.body.changes) } }), result);
            }
        },
        maintainPair: (id, body, header) => mapResponse(api.POST('/api/v1/locations/{id}/lifecycle-pairs/maintain', { params: { path: { id }, header }, body }), result),
        pair: (id, command, header) => command.action === 'insert'
            ? mapResponse(api.POST('/api/v1/locations/{id}/lifecycle-pairs/insert', { params: { path: { id }, header }, body: command.body }), result)
            : mapResponse(api.POST('/api/v1/locations/{id}/lifecycle-pairs/delete', { params: { path: { id }, header }, body: command.body }), result),
    },
    navigation: async (params, anchorId, direction, signal) => api.POST('/api/v1/locations/navigation', { params: { header: await header(signal) }, body: { query: query(params), anchorId, direction, sort: params.get('sort') ?? undefined }, signal }),
    selection: {
        protocol: 'identity-draft',
        start: async (query, selection, position, signal) => mapResponse(api.POST('/api/v1/locations/selection/start', { params: { header: await header(signal) }, body: { query: selectionQuery(query), selection, page: position.page, pageSize: position.pageSize, sort: position.sort }, signal }), v => v.pageState === 'available' ? ({ ...v, page: list(v.page) }) : v),
        resolvePage: async (query, draft, edit, position, signal) => mapResponse(api.POST('/api/v1/locations/selection/resolve', { params: { header: await header(signal) }, body: { query: selectionQuery(query), draftId: draft.draftId, expectedRevision: draft.revision, ...edit, page: position.page, pageSize: position.pageSize, sort: position.sort }, signal }), v => v.pageState === 'available' ? ({ ...v, page: list(v.page) }) : v),
        cancel: async (draft) => {
            const controller = new AbortController(), timer = setTimeout(() => controller.abort(), 10000);
            try {
                return await api.POST('/api/v1/locations/selection/cancel', { params: { header: await header(controller.signal) }, body: { draftId: draft.draftId, expectedRevision: draft.revision }, signal: controller.signal });
            }
            finally {
                clearTimeout(timer);
            }
        },
        export: async (query, draft, sort, signal) => api.POST('/api/v1/locations/export', { params: { header: await header(signal) }, body: { query: selectionQuery(query), draftId: draft.draftId, expectedRevision: draft.revision, sort }, signal, parseAs: 'blob' }),
        fallbackFilename: p => `chnhcm-locations-${p.businessAt}.csv`, toolbarSelector: '[aria-label="所选地点"]', returnFocusSelector: '[data-selection-status], input[aria-label="搜索编码或名称"], [data-list-heading]',
    },
};
