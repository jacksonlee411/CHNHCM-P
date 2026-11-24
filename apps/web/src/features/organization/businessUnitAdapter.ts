import { recoveryMessages } from './dated/recoveryMessages';
import { api, unwrap, sessionGuard, type Models } from '@chnhcm/api-client';
import { businessDate } from '@chnhcm/ui';
import { submissionChannel } from '../../shared/dated/submission';
import type { FieldCatalog } from './dated/object';
import { columnFilterQuery } from './BusinessUnitColumnFilter';
import { mapResponse, type Definition, type Result, type DetailView, type DataFields, type RecordCommand, type PairCommand } from './dated/presentation';

export const BU_PATH = '/settings/organization/business-units';
type Header = { 'X-CSRF-Token': string; 'Idempotency-Key': string };
export const businessUnitFields = {
  code: { label: '编码' }, name: { label: '中文名称' }, nameEn: { label: '英文名称' },
  effectiveDate: { label: '生效日期' }, operationType: { label: '操作类型' }, operationReason: { label: '操作原因' },
} satisfies FieldCatalog<Models['CreateInput']>;
const changes = (c?: DataFields): Models['ChangesInput'] | undefined => c && ({ name: c.name ?? undefined, nameEn: c.nameEn });
const result = (r: Models['Result']): Result => ({ ...r, objectId: r.buId });
const record = <T extends { name?: string | null; nameEn?: string | null }>(r: T) => { const { nameEn, ...metadata } = r; return { ...metadata, values: { name: r.name, nameEn } }; };
const detail = (d: Models['Detail']): DetailView => ({ ...d,
  selectedRecord: d.selectedRecord && record(d.selectedRecord), timeline: d.timeline.map(record), viewFact: d.viewFact && record(d.viewFact),
  additionContext: d.additionContext && { ...d.additionContext, defaults: d.additionContext.defaults && record(d.additionContext.defaults) },
  lifecycleContext: d.lifecycleContext && { ...d.lifecycleContext, defaults: d.lifecycleContext.defaults && record(d.lifecycleContext.defaults) },
  pairDeletion: d.pairDeletion && {
    deactivation: record(d.pairDeletion.deactivation), reactivation: record(d.pairDeletion.reactivation),
    changes: d.pairDeletion.nameChanges.map(c => ({ start: c.start, endExclusive: c.endExclusive, endInclusive: c.endInclusive, before: { values: { name: c.beforeName, nameEn: c.beforeNameEn } }, after: { values: { name: c.afterName, nameEn: c.afterNameEn } } })),
  },
});

const object = {
  kind: 'organization.business-unit', basePath: BU_PATH,
  submissions: submissionChannel<string, Result>('organization.business-unit', 'chnhcm-pending-submissions'),
  resolve: (session: { instanceId: string; subjectId: string; csrf: string }, pending: {submissionId:string}) => mapResponse(api.POST('/api/v1/organization/submissions/{id}/resolve', { params: { path: { id: pending.submissionId }, header: { 'X-CSRF-Token': session.csrf } } }), result),
  resultView: (result: Result) => ({
    kind: 'organization.business-unit', submissionId: result.submissionId, outcome: result.outcome,
    objectId: result.objectId ?? undefined, sourceVersion: result.sourceVersion ?? result.observedSourceVersion ?? undefined,
    selectedRecordId: (result.action === 'DELETE' ? result.targetRecordId : result.selectedRecordId) ?? undefined,
    lines: [
      ...(result.items ?? []).map(item => `${item.outcome === 'no_change' ? '输入无变化' : '已保存'}${item.effectiveDate ? ` · ${businessDate(item.effectiveDate)}` : ''}${item.afterRecord?.name ? ` · ${item.afterRecord.name}` : ''}`),
      ...recoveryMessages(result.failureDetails),
    ],
  }),
};
const businessUnitApi: Definition['api'] = {
  context: signal => api.GET('/api/v1/organization/context', { signal }),
  reasons: (operationType: 'CREATE' | 'CHANGE' | 'DEACTIVATE' | 'REACTIVATE', signal?: AbortSignal) => api.GET('/api/v1/organization/operation-reasons', { params: { query: { operationType } }, signal }),
  list: (query: URLSearchParams, requestedPage = Number(query.get('page') ?? 1), signal?: AbortSignal) => api.GET('/api/v1/business-units', { params: { query: {
    q: query.get('q') ?? undefined, status: (query.get('status') ?? 'all') as Models['SelectionQuery']['status'],
    sort: (query.get('sort') ?? 'code') as 'code' | 'name' | 'name_desc', page: requestedPage,
    pageSize: Number(query.get('pageSize') ?? 20) as 20 | 50 | 100, ...columnFilterQuery(query), filterField: columnFilterQuery(query).filterField as Models['SelectionQuery']['filterField'],
  } }, signal }),
  detail: (id, query, signal) => mapResponse(api.GET('/api/v1/business-units/{id}', { params: { path: { id }, query }, signal }), detail),
  create: (body, header) => mapResponse(api.POST('/api/v1/business-units', { params: { header }, body: { code: body.code, name: body.name, nameEn: body.nameEn, effectiveDate: body.effectiveDate, operationType: 'CREATE', operationReason: body.operationReason as Models['CreateInput']['operationReason'] } }), result),
  preview: (id, csrf, body) => mapResponse(api.POST('/api/v1/business-units/{id}/preview', { params: { path: { id }, header: { 'X-CSRF-Token': csrf } }, body: { ...body, changes: changes(body.changes), operationReason: body.operationReason as Models['PreviewInput']['operationReason'] } }), v => ({ ...v, afterRecord: v.afterRecord && record(v.afterRecord) })),
  changes: (id, query, signal) => mapResponse(api.GET('/api/v1/business-units/{id}/changes', { params: { path: { id }, query }, signal }), v => ({ ...v, items: v.items.map(c => ({ ...c, items: c.items.map(i => ({ ...i, beforeContext: i.beforeContext && record(i.beforeContext), afterContext: i.afterContext && record(i.afterContext) })) })) })),
  record(id: string, command: RecordCommand, header: Header) {
    switch (command.action) {
      case 'ADD': return mapResponse(api.POST('/api/v1/business-units/{id}/records', { params: { path: { id }, header }, body: { ...command.body, changes: changes(command.body.changes) ?? {}, operationReason: command.body.operationReason as Models['AddInput']['operationReason'] } }), result);
      case 'CORRECT': return mapResponse(api.POST('/api/v1/business-units/{id}/records/{recordId}/correct', { params: { path: { id, recordId: command.recordId }, header }, body: { ...command.body, changes: changes(command.body.changes) } }), result);
      case 'DELETE': return mapResponse(api.POST('/api/v1/business-units/{id}/records/{recordId}/delete', { params: { path: { id, recordId: command.recordId }, header }, body: command.body }), result);
      case 'DEACTIVATE': return mapResponse(api.POST('/api/v1/business-units/{id}/deactivate', { params: { path: { id }, header }, body: { ...command.body, operationReason: command.body.operationReason as Models['DeactivateInput']['operationReason'] } }), result);
      case 'REACTIVATE': return mapResponse(api.POST('/api/v1/business-units/{id}/reactivate', { params: { path: { id }, header }, body: { ...command.body, changes: changes(command.body.changes), operationReason: command.body.operationReason as Models['ReactivateInput']['operationReason'] } }), result);
    }
  },
  pair(id: string, command: PairCommand, header: Header) {
    return command.action === 'insert'
      ? mapResponse(api.POST('/api/v1/business-units/{id}/lifecycle-pairs', { params: { path: { id }, header }, body: command.body }), result)
      : mapResponse(api.POST('/api/v1/business-units/{id}/lifecycle-pairs/delete', { params: { path: { id }, header }, body: command.body }), result);
  },
};

async function selectionHeader(signal: AbortSignal) {
  const same = sessionGuard();
  const response = await api.GET('/api/v1/session', { signal });
  signal.throwIfAborted();
  if (!same()) throw new Error('会话已变化，请重新查询。');
  return { 'X-CSRF-Token': unwrap(response).csrf };
}
const businessUnitSelection: Definition['selection'] = {
  protocol: 'list-version',
  version: page => { if ('selectionVersion' in page && typeof page.selectionVersion === 'string') return page.selectionVersion; throw new Error('业务单元列表缺少选择版本'); },
  resolve: async (query, selection, expectedSelectionVersion, signal) => api.POST('/api/v1/business-units/selection/resolve', {
    body: { query: { ...query, filterField: query.filterField as Models['SelectionQuery']['filterField'] }, selection, expectedSelectionVersion }, params: { header: await selectionHeader(signal) }, signal,
  }),
  export: async (query, selection, expectedSelectionVersion, sort, signal) => api.POST('/api/v1/business-units/export', {
    body: { query: { ...query, filterField: query.filterField as Models['SelectionQuery']['filterField'] }, selection, expectedSelectionVersion, sort:sort as 'code'|'name'|'name_desc' }, params: { header: await selectionHeader(signal) }, signal, parseAs: 'blob',
  }),
  fallbackFilename: page => `chnhcm-business-units-${page.businessAt}.csv`,
  toolbarSelector: '[aria-label="所选业务单元"]',
  returnFocusSelector: '[data-selection-status], input[aria-label="搜索编码或名称"], [data-list-heading]',
};

export const businessUnit: Definition = { ...object, label: '业务单元', prefix: 'bu',
  fields: [{ key: 'name', label: '中文名称', required: true }, { key: 'nameEn', label: '英文名称' }],
  api: businessUnitApi, selection: businessUnitSelection,
  navigation: async (params, anchorId, direction, signal) => api.POST('/api/v1/business-units/navigation', {
    params: { header: await selectionHeader(signal) }, signal,
    body: { anchorId, direction, sort: (params.get('sort') ?? 'code') as 'code' | 'name' | 'name_desc',
      query: { q: params.get('q') ?? undefined, status: (params.get('status') ?? 'all') as Models['SelectionQuery']['status'],
        ...columnFilterQuery(params), filterField: columnFilterQuery(params).filterField as Models['SelectionQuery']['filterField'] } },
  }),
};

export type { RecordCommand } from "./dated/presentation";
