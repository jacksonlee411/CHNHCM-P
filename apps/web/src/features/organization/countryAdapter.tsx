import { api, ApiError, unwrap, sessionGuard, type Models } from '@chnhcm/api-client';
import { submissionChannel } from '../../shared/dated/submission';
import { columnFilterQuery, columnStatuses } from './BusinessUnitColumnFilter';
import { mapResponse, type Definition, type DataFields, type Result, type DetailView } from './dated/presentation';

export const COUNTRY_PATH = '/settings/basic-data/country-regions';
const record = (r: Models['CountryRecord']) => { const { nameEn, phoneCode, ...metadata } = r; return { ...metadata, values: { name: r.name, nameEn, phoneCode } }; };
const changes = (c?: DataFields): Models['CountryChangesInput'] => ({ name: c?.name ?? undefined, nameEn: c?.nameEn, phoneCode: c?.phoneCode });
const result = (r: Models['CountryResult']): Result => ({ ...r, objectId: r.countryRegionId });
const detail = (d: Models['CountryDetail']): DetailView => ({ ...d,
  selectedRecord: d.selectedRecord && record(d.selectedRecord), timeline: d.timeline.map(record),
  additionContext: d.additionContext && { ...d.additionContext, defaults: record(d.additionContext.defaults), maxEffectiveDate: d.additionContext.maxEffectiveDate ?? undefined },
  lifecycleContext: d.lifecycleContext && { ...d.lifecycleContext, defaults: record(d.lifecycleContext.defaults), maxEffectiveDate: d.lifecycleContext.maxEffectiveDate ?? undefined },
});
async function header(signal: AbortSignal) {
  const same = sessionGuard(); const v = await api.GET('/api/v1/session', { signal }); signal.throwIfAborted();
  if (!same()) throw new Error('会话已变化，请重新查询。');
  return { 'X-CSRF-Token': unwrap(v).csrf };
}
export const country: Definition = {
  kind: 'organization.country', basePath: COUNTRY_PATH, label: '国家/地区', prefix: 'country', statusReadField: 'state',
  fields: [{ key: 'name', label: '显示名称', required: true }, { key: 'nameEn', label: '英文名称' }, { key: 'phoneCode', label: '电话区号' }],
  filterColumns: [{ field: 'code', label: '编码' }, { field: 'name', label: '名称' }, { field: 'status', label: '状态', values: columnStatuses }],
  submissions: submissionChannel<string, Result>('organization.country', 'chnhcm-pending-country-submissions'),
  resolve: (session, pending) => mapResponse(api.POST('/api/v1/country-regions/submissions/{id}/resolve', { params: { path: { id: pending.submissionId }, header: { 'X-CSRF-Token': session.csrf } } }), result),
  resultView: r => ({ kind: 'organization.country', submissionId: r.submissionId, outcome: r.outcome, objectId: r.objectId ?? undefined, sourceVersion: r.sourceVersion ?? undefined, selectedRecordId: r.selectedRecordId ?? undefined, lines: [] }),
  navigation: async (params, anchorId, direction, signal) => api.POST('/api/v1/country-regions/navigation', { signal, params: { header: await header(signal) }, body: {
    anchorId, direction, sort: (params.get('sort') ?? 'code') as 'code' | 'name' | 'name_desc',
    query: { q: params.get('q') ?? undefined, status: (params.get('status') ?? 'all') as Models['SelectionQuery']['status'], ...columnFilterQuery(params), filterField: columnFilterQuery(params).filterField as Models['SelectionQuery']['filterField'] },
  } }),
  api: {
    context: signal => api.GET('/api/v1/country-regions/context', { signal }),
    reasons: (operationType, signal) => api.GET('/api/v1/country-regions/operation-reasons', { params: { query: { operationType } }, signal }),
    list: (query, requestedPage = Number(query.get('page') ?? 1), signal) => api.GET('/api/v1/country-regions', { params: { query: {
      q: query.get('q') ?? undefined, status: (query.get('status') ?? 'all') as Models['SelectionQuery']['status'], sort: (query.get('sort') ?? 'code') as 'code' | 'name' | 'name_desc',
      page: requestedPage, pageSize: Number(query.get('pageSize') ?? 20) as 20 | 50 | 100, ...columnFilterQuery(query), filterField: columnFilterQuery(query).filterField as Models['SelectionQuery']['filterField'],
    } }, signal }),
    detail: (id, query, signal) => mapResponse(api.GET('/api/v1/country-regions/{id}', { params: { path: { id }, query }, signal }), detail),
    changes: (id, query, signal) => mapResponse(api.GET('/api/v1/country-regions/{id}/changes', { params: { path: { id }, query }, signal }), v => ({ ...v, items: v.items.map(c => ({ ...c, items: c.items.map(i => ({ ...i, beforeContext: i.beforeContext && record(i.beforeContext), afterContext: i.afterContext && record(i.afterContext) })) })) })),
    preview: (id, csrf, body) => mapResponse(api.POST('/api/v1/country-regions/{id}/preview', { params: { path: { id }, header: { 'X-CSRF-Token': csrf } }, body: {
      action: body.action, expectedSourceVersion: body.expectedSourceVersion,
      ...(body.action === 'CORRECT' ? { recordId: body.recordId, expectedRecordRevision: body.expectedRecordRevision, changes: changes(body.changes) } : {
        anchorRecordId: body.anchorRecordId, effectiveDate: body.effectiveDate, operationType: body.operationType, operationReason: body.operationReason,
        ...(body.action !== 'DEACTIVATE' && body.changes ? { changes: changes(body.changes) } : {}),
      }),
    } }), v => ({ ...v, afterRecord: v.afterRecord && record(v.afterRecord) })),
    record(id, command, header) {
      switch (command.action) {
        case 'ADD': return mapResponse(api.POST('/api/v1/country-regions/{id}/records', { params: { path: { id }, header }, body: { ...command.body, changes: changes(command.body.changes) } }), result);
        case 'CORRECT': return mapResponse(api.POST('/api/v1/country-regions/{id}/records/{recordId}/correct', { params: { path: { id, recordId: command.recordId }, header }, body: { expectedSourceVersion: command.body.expectedSourceVersion, expectedRecordRevision: command.body.expectedRecordRevision, changes: changes(command.body.changes) } }), result);
        case 'DEACTIVATE': return mapResponse(api.POST('/api/v1/country-regions/{id}/deactivate', { params: { path: { id }, header }, body: command.body }), result);
        case 'REACTIVATE': return mapResponse(api.POST('/api/v1/country-regions/{id}/reactivate', { params: { path: { id }, header }, body: { ...command.body, changes: command.body.changes && changes(command.body.changes) } }), result);
        case 'DELETE': throw new ApiError('M02_CAPABILITY_UNSUPPORTED', '国家/地区不提供删除记录', 400);
      }
    },
  },
};
