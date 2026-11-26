import { api, unwrap, type Models } from '@chnhcm/api-client';
import { paths } from './api';
import { submissionChannel, terminalOutcome, type Pending } from '../../shared/dated/submission';
export const quotaPath = '/settings/staffing/quotas';
export function positionQuotaPath(id: string, params = new URLSearchParams()) {
  const query = params.toString();
  return paths.position + '/' + id + '/quota' + (query ? '?' + query : '');
}
export function quotaResultLinks(result: Result) {
  const links: { href: string; label: string }[] = [];
  const add = (kind: string, id: string, source?: string, record?: string) => {
    if (!source || !record) return;
    const params = new URLSearchParams({ sourceVersion: source, selectedRecordId: record });
    links.push({
      href:
        kind === 'position' ? positionQuotaPath(id, params) : quotaPath + '/departments/' + id + '?' + params,
      label: kind === 'position' ? '查看岗位核编原件' : '查看部门额度原件',
    });
  };
  if (result.positionId || (result.departmentId && result.objectId))
    add(
      result.positionId ? 'position' : 'department',
      (result.positionId || result.departmentId)!,
      result.originalSourceVersion ?? result.sourceVersion,
      result.selectedRecordId,
    );
  for (const row of result.quotaChanges ?? [])
    add(row.targetKind, row.targetId, row.sourceVersion, row.recordId);
  return links;
}
export type Position = Models['StaffingQuotaPosition'];
export type RecordView = Models['StaffingQuotaRecord'];
export type Result = Models['StaffingResult'];
export const receipts = submissionChannel<string, Result>('staffing.quota', 'chnhcm-pending-staffing-quotas');
const priorReceipts = submissionChannel<string, Result>('staffing.quota', 'chnhcm-pending-quotas');
export function recoverStoredQuotaReferences() {
  for (const ref of priorReceipts.read()) {
    receipts.remember(ref);
    priorReceipts.forget(ref);
  }
}
export const labels = {
  create: '核编',
  adjust: '调编',
  correct: '更正记录',
  redate: '更改生效日',
  withdraw: '撤去核编',
  delete: '删除记录',
} as const;
export type Action = keyof typeof labels;
export const departmentRecordLabels = {
  correct: '更正部门总额',
  redate: '更改总额生效日',
  delete: '删除总额记录',
} as const;
export type Department = Models['DepartmentAllocation'];
export function readDepartment(
  departmentId: string,
  businessAt: string,
  sourceVersion: string,
  signal: AbortSignal,
  selectedRecordId?: string,
) {
  return api
    .GET('/api/v1/staffing-quotas/departments/{departmentId}', {
      params: { path: { departmentId }, query: { businessAt, sourceVersion, selectedRecordId } },
      signal,
    })
    .then(unwrap);
}
export function saveAllocation(
  departmentId: string,
  body: Models['AllocationInput'],
  csrf: string,
  id: string,
) {
  return api
    .POST('/api/v1/staffing-quotas/departments/{departmentId}/allocations', {
      params: { path: { departmentId }, header: { 'X-CSRF-Token': csrf, 'Idempotency-Key': id } },
      body,
    })
    .then(unwrap);
}
export type Command =
  | { action: 'create'; body: Models['StaffingQuotaCreateInput'] }
  | { action: 'adjust'; quotaId: string; body: Models['StaffingQuotaAdjustInput'] }
  | { action: 'withdraw'; quotaId: string; body: Models['StaffingQuotaWithdrawInput'] }
  | { action: 'correct'; quotaId: string; recordId: string; body: Models['StaffingQuotaCorrectInput'] }
  | { action: 'redate'; quotaId: string; recordId: string; body: Models['StaffingQuotaRedateInput'] }
  | { action: 'delete'; quotaId: string; recordId: string; body: Models['StaffingQuotaDeleteInput'] };
export async function saveQuota(cmd: Command, csrf: string, id: string) {
  const header = { 'X-CSRF-Token': csrf, 'Idempotency-Key': id };
  switch (cmd.action) {
    case 'create':
      return unwrap(await api.POST('/api/v1/staffing-quotas', { params: { header }, body: cmd.body }));
    case 'adjust':
      return unwrap(
        await api.POST('/api/v1/staffing-quotas/{quotaId}/adjust', {
          params: { header, path: { quotaId: cmd.quotaId } },
          body: cmd.body,
        }),
      );
    case 'withdraw':
      return unwrap(
        await api.POST('/api/v1/staffing-quotas/{quotaId}/withdraw', {
          params: { header, path: { quotaId: cmd.quotaId } },
          body: cmd.body,
        }),
      );
    case 'correct':
      return unwrap(
        await api.POST('/api/v1/staffing-quotas/{quotaId}/records/{recordId}/correct', {
          params: { header, path: { quotaId: cmd.quotaId, recordId: cmd.recordId } },
          body: cmd.body,
        }),
      );
    case 'redate':
      return unwrap(
        await api.POST('/api/v1/staffing-quotas/{quotaId}/records/{recordId}/redate', {
          params: { header, path: { quotaId: cmd.quotaId, recordId: cmd.recordId } },
          body: cmd.body,
        }),
      );
    case 'delete':
      return unwrap(
        await api.POST('/api/v1/staffing-quotas/{quotaId}/records/{recordId}/delete', {
          params: { header, path: { quotaId: cmd.quotaId, recordId: cmd.recordId } },
          body: cmd.body,
        }),
      );
  }
}
export async function previewQuota(cmd: Command, csrf: string, signal: AbortSignal) {
  const header = { 'X-CSRF-Token': csrf };
  switch (cmd.action) {
    case 'create':
      return unwrap(
        await api.POST('/api/v1/staffing-quotas/preview', { params: { header }, body: cmd.body, signal }),
      );
    case 'adjust':
      return unwrap(
        await api.POST('/api/v1/staffing-quotas/{quotaId}/adjust/preview', {
          params: { header, path: { quotaId: cmd.quotaId } },
          body: cmd.body,
          signal,
        }),
      );
    case 'withdraw':
      return unwrap(
        await api.POST('/api/v1/staffing-quotas/{quotaId}/withdraw/preview', {
          params: { header, path: { quotaId: cmd.quotaId } },
          body: cmd.body,
          signal,
        }),
      );
    case 'correct':
      return unwrap(
        await api.POST('/api/v1/staffing-quotas/{quotaId}/records/{recordId}/correct/preview', {
          params: { header, path: { quotaId: cmd.quotaId, recordId: cmd.recordId } },
          body: cmd.body,
          signal,
        }),
      );
    case 'redate':
      return unwrap(
        await api.POST('/api/v1/staffing-quotas/{quotaId}/records/{recordId}/redate/preview', {
          params: { header, path: { quotaId: cmd.quotaId, recordId: cmd.recordId } },
          body: cmd.body,
          signal,
        }),
      );
    case 'delete':
      return unwrap(
        await api.POST('/api/v1/staffing-quotas/{quotaId}/records/{recordId}/delete/preview', {
          params: { header, path: { quotaId: cmd.quotaId, recordId: cmd.recordId } },
          body: cmd.body,
          signal,
        }),
      );
  }
}
export async function resolveQuota(csrf: string, ref: Pending) {
  const r = unwrap(
    await api.POST('/api/v1/staffing-quotas/submissions/{id}/resolve', {
      params: { path: { id: ref.submissionId }, header: { 'X-CSRF-Token': csrf } },
    }),
  );
  if (r.kind !== 'staffingquota' || r.submissionId !== ref.submissionId || !terminalOutcome(r.outcome))
    throw new Error('原提交结果尚未确认');
  return r;
}
export const quotaAmountFields = { position: '核定编制', department: '部门总额' } as const;
export const quotaTotals = { complete: '直属岗位核定合计', visible: '可见范围 · 岗位核定小计' } as const;
export const missingQuotaRecord = '当前资料中没有所选记录，请重新选择。';

export function quotaLabel(record: RecordView, target: keyof typeof quotaAmountFields = 'position') {
  if (record.quotaState === 'set') return String(record.quota);
  if (record.quotaState === 'unset') return target === 'department' ? '未核定' : '未核编';
  return '受限';
}

export function quotaSavedNotice(result: Result) {
  if (result.outcome === 'no_change') return '没有修改内容。';
  return result.action?.toLowerCase() === 'delete' ? '已删除。' : '已保存。';
}
export const period = (r: RecordView) =>
  !r.effectiveDate
    ? ''
    : r.endInclusive
      ? r.effectiveDate + ' 至 ' + r.endInclusive
      : '自 ' + r.effectiveDate + ' 起';
