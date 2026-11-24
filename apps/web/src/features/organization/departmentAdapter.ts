import { createElement } from 'react';
import { recoveryMessages } from './dated/recoveryMessages';
import { DepartmentTree } from './DepartmentTree';
import { api, ApiError, unwrap, sessionGuard, type Models } from '@chnhcm/api-client';
import { businessDate } from '@chnhcm/ui';
import { submissionChannel } from '../../shared/dated/submission';
import { mapResponse, type Definition, type DataFields, type PreviewInput, type Result, type RecordView } from './dated/presentation';
import { columnFilterQuery } from './BusinessUnitColumnFilter';
import { departmentColumns } from './DepartmentColumns';
import { DepartmentRelations, DepartmentRelationsView, DepartmentPath, referenceLabel } from './DepartmentRelations';

export const DEPARTMENT_PATH = '/settings/organization/departments';
const changes = (c?: DataFields): Models['DepartmentChangesInput'] | undefined => c && ({ name: c.name ?? undefined, shortName: c.shortName, managementBuId: c.managementBuId ?? undefined, parentDepartmentId: c.parentDepartmentId });
const result = (r: Models['DepartmentResult']): Result => ({ ...r, objectId: r.departmentId });
const record = <T extends Partial<Models['DepartmentRecord']>>(r: T) => {
  const { shortName, managementBuId, parentDepartmentId, managementBu, parentDepartment, ...metadata } = r;
  return { ...metadata, values: { name: r.name, shortName, managementBuId, parentDepartmentId }, references: { managementBuId: managementBu, parentDepartmentId: parentDepartment } };
};
const list = (v: Models['DepartmentList']) => ({ ...v, items: v.items.map(r => ({ ...r, values: { shortName: r.shortName }, references: { managementBuId: r.managementBu }, secondary: createElement(DepartmentPath, { path: r.path }) })) });
const detail = (d: Models['DepartmentDetail']) => ({ ...d, id: d.departmentId,
  selectedRecord: d.selectedRecord && record(d.selectedRecord), timeline: d.timeline.map(record),
  additionContext: d.additionContext && { ...d.additionContext, defaults: d.additionContext.defaults && record(d.additionContext.defaults) },
  lifecycleContext: d.lifecycleContext && { ...d.lifecycleContext, defaults: d.lifecycleContext.defaults && record(d.lifecycleContext.defaults) },
  pairDeletion: d.pairDeletion && { ...d.pairDeletion, deactivation: record(d.pairDeletion.deactivation), reactivation: record(d.pairDeletion.reactivation), changes: d.pairDeletion.changes.map(c => ({ ...c, before: record(c.before), after: record(c.after) })) },
  related: (selectedRecord: RecordView | null, readFields: string[], referencesRestricted: boolean) => createElement(DepartmentRelationsView, { record: selectedRecord, readFields, path: referencesRestricted ? { pathMode: 'restricted', pathVersion: d.path.pathVersion } : d.path }),
});
function preview(body: PreviewInput): Models['DepartmentPreviewInput'] {
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
  if (!same()) throw new Error('会话已变化，请重新查询。');
  return { 'X-CSRF-Token': unwrap(response).csrf };
}
export const department: Definition = {
  ...departmentColumns,
  ListSidebar: DepartmentTree,
  kind: 'organization.department', basePath: DEPARTMENT_PATH, label: '部门', prefix: 'department',
  statusReadField: 'status', pathCursorField: 'departmentPath',
  fields: [{ key: 'name', label: '部门名称', required: true }, { key: 'shortName', label: '简称' }],
  summaryFields: [{ key: 'name', label: '部门名称' }, { key: 'shortName', label: '简称' }, { key: 'managementBuId', label: '管理业务单元', render: r => referenceLabel(r.references?.managementBuId) || (r.values?.managementBuId ? '名称无读取权限' : '未提供') }, { key: 'parentDepartmentId', label: '行政上级', render: r => r.values?.parentDepartmentId ? referenceLabel(r.references?.parentDepartmentId) || '名称无读取权限' : '企业根部门' }],
  referenceContext: (id, body) => {
    const relationChanges = {
      ...(body.changes?.managementBuId !== undefined ? { managementBuId: body.changes.managementBuId ?? undefined } : {}),
      ...(body.changes?.parentDepartmentId !== undefined ? { parentDepartmentId: body.changes.parentDepartmentId } : {}),
    };
    const changes = Object.keys(relationChanges).length ? relationChanges : undefined;
    const common = { mode: 'maintain' as const, departmentId: id, expectedSourceVersion: body.expectedSourceVersion };
    if (body.action === 'CORRECT') return { ...common, action: body.action, recordId: body.recordId ?? '', expectedRecordRevision: body.expectedRecordRevision ?? '', newEffectiveDate: body.newEffectiveDate, classification: body.classification, changes };
    if (body.action === 'ADD') return { ...common, action: body.action, anchorRecordId: body.anchorRecordId ?? '', effectiveDate: body.effectiveDate ?? '', operationType: 'CHANGE', operationReason: body.operationReason || undefined, changes };
    if (body.action === 'REACTIVATE') return { ...common, action: body.action, anchorRecordId: body.anchorRecordId ?? '', effectiveDate: body.effectiveDate ?? '', operationReason: body.operationReason || undefined, changes };
    return undefined;
  },
  createReferenceContext: (date, values) => ({ mode: 'create', effectiveDate: date, managementBuId: values.managementBuId ?? undefined }),
  Relations: DepartmentRelations,
  submissions: submissionChannel<string, Result>('organization.department', 'chnhcm-pending-department-submissions'),
  resolve: (session, pending) => mapResponse(api.POST('/api/v1/departments/submissions/{id}/resolve', { params: { path: { id: pending.submissionId }, header: { 'X-CSRF-Token': session.csrf } } }), result),
  resultView: r => ({ kind: 'organization.department', submissionId: r.submissionId, outcome: r.outcome, objectId: r.objectId ?? undefined, sourceVersion: r.sourceVersion ?? r.observedSourceVersion ?? undefined,
    selectedRecordId: (r.action === 'DELETE' ? r.targetRecordId : r.selectedRecordId) ?? undefined,
    lines: [...(r.items ?? []).map(i => `${i.outcome === 'no_change' ? '输入无变化' : '已保存'}${i.effectiveDate ? ` · ${businessDate(i.effectiveDate)}` : ''}${i.afterRecord?.name ? ` · ${i.afterRecord.name}` : ''}`),
      ...recoveryMessages(r.failureDetails)],
  }),
  api: {
    context: signal => api.GET('/api/v1/departments/context', { signal }),
    reasons: (operationType, signal) => api.GET('/api/v1/departments/operation-reasons', { params: { query: { operationType } }, signal }),
    list: (query, requestedPage = Number(query.get('page') ?? 1), signal) => mapResponse(api.GET('/api/v1/departments', { params: { query: {
      scopeDepartmentId: query.get('scopeDepartmentId') ?? undefined, includeDescendants: query.has('scopeDepartmentId') ? query.get('includeDescendants') !== 'false' : undefined,
      cursor: query.get('cursor') ?? undefined, q: query.get('q') ?? undefined, status: (query.get('status') ?? 'all') as Models['DepartmentSelectionQuery']['status'],
      sort: (query.get('sort') ?? 'code') as 'code' | 'name' | 'name_desc', page: requestedPage, pageSize: Number(query.get('pageSize') ?? 20) as 20 | 50 | 100, ...columnFilterQuery(query),
    } }, signal }), list),
    detail: (id, query, signal) => {
      return mapResponse(api.GET('/api/v1/departments/{id}', { params: { path: { id }, query: { sourceVersion: query.sourceVersion, selectedRecordId: query.selectedRecordId } }, signal }), detail);
    },
    create: (body, header) => mapResponse(api.POST('/api/v1/departments', { params: { header }, body: { code: body.code, name: body.name, shortName: body.shortName || undefined, managementBuId: body.managementBuId ?? '', parentDepartmentId: body.parentDepartmentId || undefined, effectiveDate: body.effectiveDate, operationType: 'CREATE', operationReason: body.operationReason } }), result),
    preview: (id, csrf, body) => api.POST('/api/v1/departments/{id}/preview', { params: { path: { id }, header: { 'X-CSRF-Token': csrf } }, body: preview(body) }),
    changes: (id, query, signal) => mapResponse(api.GET('/api/v1/departments/{id}/changes', { params: { path: { id }, query }, signal }), v => ({ ...v, items: v.items.map(c => ({ ...c, items: c.items.map(i => ({ ...i, beforeContext: i.beforeContext && record(i.beforeContext), afterContext: i.afterContext && record(i.afterContext) })) })) })),
    record(id, command, header) {
      switch (command.action) {
        case 'ADD': return mapResponse(api.POST('/api/v1/departments/{id}/records', { params: { path: { id }, header }, body: { ...command.body, changes: changes(command.body.changes) } }), result);
        case 'CORRECT': return mapResponse(api.POST('/api/v1/departments/{id}/records/{recordId}/correct', { params: { path: { id, recordId: command.recordId }, header }, body: { ...command.body, changes: changes(command.body.changes) } }), result);
        case 'DELETE': return mapResponse(api.POST('/api/v1/departments/{id}/records/{recordId}/delete', { params: { path: { id, recordId: command.recordId }, header }, body: command.body }), result);
        case 'DEACTIVATE': return mapResponse(api.POST('/api/v1/departments/{id}/deactivate', { params: { path: { id }, header }, body: command.body }), result);
        case 'REACTIVATE': return mapResponse(api.POST('/api/v1/departments/{id}/reactivate', { params: { path: { id }, header }, body: { ...command.body, changes: changes(command.body.changes) } }), result);
      }
    },
    pair: (id, command, header) => command.action === 'insert'
      ? mapResponse(api.POST('/api/v1/departments/{id}/lifecycle-pairs', { params: { path: { id }, header }, body: command.body }), result)
      : mapResponse(api.POST('/api/v1/departments/{id}/lifecycle-pairs/delete', { params: { path: { id }, header }, body: command.body }), result),
  },
  navigation: async (params, anchorId, direction, signal) => {
    return api.POST('/api/v1/departments/navigation', { params: { header: await header(signal) }, body: {
      query: { scopeDepartmentId: params.get('scopeDepartmentId') ?? undefined, includeDescendants: params.has('scopeDepartmentId') ? params.get('includeDescendants') !== 'false' : undefined, q: params.get('q') ?? undefined, status: (params.get('status') ?? 'all') as Models['DepartmentSelectionQuery']['status'], ...columnFilterQuery(params) },
      anchorId, direction,
      sort: (params.get('sort') ?? 'code') as 'code' | 'name' | 'name_desc',
    }, signal });
  },
  selection: {
    protocol: 'identity-draft',
    start: async (query, selection, position, signal) => mapResponse(api.POST('/api/v1/departments/selection/start', { params: { header: await header(signal) }, body: { query: {...query,filterField:query.filterField as Models['DepartmentSelectionQuery']['filterField']}, selection, ...position, sort:position.sort as 'code'|'name'|'name_desc', pageSize: position.pageSize as 20 | 50 | 100 }, signal }), v => v.pageState === 'available' ? ({ ...v, page: list(v.page) }) : v),
    resolvePage: async (query, draft, edit, position, signal) => mapResponse(api.POST('/api/v1/departments/selection/resolve', { params: { header: await header(signal) }, body: { query: {...query,filterField:query.filterField as Models['DepartmentSelectionQuery']['filterField']}, draftId: draft.draftId, expectedRevision: draft.revision, ...edit, ...position, sort:position.sort as 'code'|'name'|'name_desc', pageSize: position.pageSize as 20 | 50 | 100 }, signal }), v => v.pageState === 'available' ? ({ ...v, page: list(v.page) }) : v),
    cancel: async draft => {
      const controller = new AbortController(), timer = setTimeout(() => controller.abort(), 10000);
      try { return await api.POST('/api/v1/departments/selection/cancel', { params: { header: await header(controller.signal) }, body: { draftId: draft.draftId, expectedRevision: draft.revision }, signal: controller.signal }); }
      finally { clearTimeout(timer); }
    },
    export: async (query, draft, sort, signal) => api.POST('/api/v1/departments/export', { params: { header: await header(signal) }, body: { query: {...query,filterField:query.filterField as Models['DepartmentSelectionQuery']['filterField']}, draftId: draft.draftId, expectedRevision: draft.revision, sort:sort as 'code'|'name'|'name_desc' }, signal, parseAs: 'blob' }),
    fallbackFilename: p => `chnhcm-departments-${p.businessAt}.csv`, toolbarSelector: '[aria-label="所选部门"]', returnFocusSelector: '[data-selection-status], input[aria-label="搜索编码或名称"], [data-list-heading]',
  },
};
