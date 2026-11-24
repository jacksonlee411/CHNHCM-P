import { api, type Models } from '@chnhcm/api-client';
import { submissionChannel } from '../../shared/dated/submission';
import { mapResponse, type Definition, type DataFields, type Result, type DetailView } from './dated/presentation';

export const ENTERPRISE_PATH = '/settings/organization/enterprise';
const record = (r: Models['EnterpriseRecord']) => { const { nameEn, ...metadata } = r; return { ...metadata, values: { name: r.name, nameEn } }; };
const changes = (c?: DataFields): Models['EnterpriseChangesInput'] => ({ name: c?.name ?? undefined, nameEn: c?.nameEn });
const result = (r: Models['EnterpriseResult']): Result => ({ ...r, objectId: r.enterpriseId });
const detail = (d: Models['EnterpriseDetail']): DetailView => ({ ...d,
  selectedRecord: d.selectedRecord && record(d.selectedRecord), timeline: d.timeline.map(record),
  additionContext: d.additionContext && { ...d.additionContext, defaults: record(d.additionContext.defaults), maxEffectiveDate: d.additionContext.maxEffectiveDate ?? undefined },
  pairDeletion: d.pairDeletion && { ...d.pairDeletion, deactivation: record(d.pairDeletion.deactivation), reactivation: record(d.pairDeletion.reactivation), changes: d.pairDeletion.changes.map(c => ({ start: c.start, endExclusive: c.endExclusive, endInclusive: c.endInclusive, before: record(c.before), after: record(c.after) })) },
  lifecycleContext: d.lifecycleContext && { ...d.lifecycleContext, defaults: record(d.lifecycleContext.defaults), maxEffectiveDate: d.lifecycleContext.maxEffectiveDate ?? undefined },
});
export const enterprise: Definition = {
  singleton: true, kind: 'organization.enterprise', basePath: ENTERPRISE_PATH, label: '企业', prefix: 'enterprise', statusReadField: 'state',
  fields: [{ key: 'name', label: '中文名称', required: true }, { key: 'nameEn', label: '英文名称' }],
  submissions: submissionChannel<string, Result>('organization.enterprise', 'chnhcm-pending-enterprise-submissions'),
  resolve: (session, pending) => mapResponse(api.POST('/api/v1/enterprise/submissions/{id}/resolve', { params: { path: { id: pending.submissionId }, header: { 'X-CSRF-Token': session.csrf } } }), result),
  resultView: r => ({ kind: 'organization.enterprise', submissionId: r.submissionId, outcome: r.outcome, objectId: r.objectId ?? undefined, sourceVersion: r.sourceVersion ?? undefined, selectedRecordId: r.selectedRecordId ?? undefined, lines: [] }),
  api: {
    context: signal => api.GET('/api/v1/enterprise/context', { signal }),
    reasons: (operationType, signal) => api.GET('/api/v1/enterprise/operation-reasons', { params: { query: { operationType } }, signal }),
    detail: (id, query, signal) => mapResponse(api.GET('/api/v1/enterprise/{id}', { params: { path: { id }, query }, signal }), detail),
    changes: (id, query, signal) => mapResponse(api.GET('/api/v1/enterprise/{id}/changes', { params: { path: { id }, query }, signal }), v => ({ ...v, items: v.items.map(c => ({ ...c, items: c.items.map(i => ({ ...i, beforeContext: i.beforeContext && record(i.beforeContext), afterContext: i.afterContext && record(i.afterContext) })) })) })),
    preview: (id, csrf, body) => mapResponse(api.POST('/api/v1/enterprise/{id}/preview', { params: { path: { id }, header: { 'X-CSRF-Token': csrf } }, body: {
      action: body.action, expectedSourceVersion: body.expectedSourceVersion,
      ...(body.action === 'DELETE' ? { recordId: body.recordId, expectedRecordRevision: body.expectedRecordRevision } : body.action === 'CORRECT' ? { recordId: body.recordId, expectedRecordRevision: body.expectedRecordRevision, newEffectiveDate: body.newEffectiveDate, classification: body.classification, changes: body.changes && changes(body.changes) } : {
        anchorRecordId: body.anchorRecordId, effectiveDate: body.effectiveDate, operationType: body.operationType, operationReason: body.operationReason,
        ...(body.action !== 'DEACTIVATE' && body.changes ? { changes: changes(body.changes) } : {}),
      }),
    } }), v => ({ ...v, afterRecord: v.afterRecord && record(v.afterRecord) })),
    pair(id, command, header) {
      return command.action === 'insert'
        ? mapResponse(api.POST('/api/v1/enterprise/{id}/lifecycle-pairs', { params: { path: { id }, header }, body: command.body }), result)
        : mapResponse(api.POST('/api/v1/enterprise/{id}/lifecycle-pairs/delete', { params: { path: { id }, header }, body: command.body }), result);
    },
    record(id, command, header) {
      switch (command.action) {
        case 'ADD': return mapResponse(api.POST('/api/v1/enterprise/{id}/records', { params: { path: { id }, header }, body: { ...command.body, changes: changes(command.body.changes) } }), result);
        case 'CORRECT': return mapResponse(api.POST('/api/v1/enterprise/{id}/records/{recordId}/correct', { params: { path: { id, recordId: command.recordId }, header }, body: { ...command.body, changes: command.body.changes && changes(command.body.changes) } }), result);
        case 'DEACTIVATE': return mapResponse(api.POST('/api/v1/enterprise/{id}/deactivate', { params: { path: { id }, header }, body: command.body }), result);
        case 'REACTIVATE': return mapResponse(api.POST('/api/v1/enterprise/{id}/reactivate', { params: { path: { id }, header }, body: { ...command.body, changes: command.body.changes && changes(command.body.changes) } }), result);
        case 'DELETE': return mapResponse(api.POST('/api/v1/enterprise/{id}/records/{recordId}/delete', { params: { path: { id, recordId: command.recordId }, header }, body: command.body }), result);
      }
    },
  },
};
