import { api, ApiError, unwrap, sessionGuard, type Models } from '@chnhcm/api-client';
import { submissionChannel, type Pending } from '../../shared/dated/submission';
export const kinds = ['jobfunction', 'jobfamily', 'job', 'grade', 'gradeladder', 'position'] as const;
export type Kind = typeof kinds[number];
export const positionFields = ['departmentId', 'jobId', 'locationId', 'gradeId'] as const;
export type PositionField = typeof positionFields[number];
export const isPositionField = (field: string): field is PositionField => positionFields.some(key => key === field);
export const positionName = (record: RecordView | undefined, field: PositionField) => record?.[`${field.slice(0, -2)}Name` as 'departmentName' | 'jobName' | 'locationName' | 'gradeName'];
export type JobClassificationField = 'jobFamilyId' | 'jobFunctionId';
export const isJobClassification = (field: string): field is JobClassificationField =>
  field === 'jobFamilyId' || field === 'jobFunctionId';
export const classificationName = (record: RecordView | undefined, field: JobClassificationField) =>
  field === 'jobFamilyId' ? record?.jobFamilyName : record?.jobFunctionName;
export type Input = Models['StaffingInput'];
export type Result = Models['StaffingResult'];
export type Detail = Models['StaffingDetail'];
export type Row = Models['StaffingSummary'];
export type RecordView = Models['StaffingRecord'];
export type Query = Models['StaffingQuery'];
export type Selection = Models['StaffingSelection'];
export type Principal = Models['Principal'];
export const paths = { position:'/settings/staffing/positions', jobfunction: '/settings/staffing/job-functions', jobfamily: '/settings/staffing/job-families', job: '/settings/staffing/jobs', grade: '/settings/staffing/grades', gradeladder: '/settings/staffing/grade-ladders' };
export const labels = { position: '岗位', jobfunction: '职类', jobfamily: '职务系列', job: '职务', grade: '职级', gradeladder: '职级阶梯' };
const bases = { position:'/api/v1/positions', jobfunction: '/api/v1/job-functions', jobfamily: '/api/v1/job-families', job: '/api/v1/jobs', grade: '/api/v1/grades', gradeladder: '/api/v1/grade-ladders' } as const;
export const channels = { position: submissionChannel<string, Result>('staffing.position', 'chnhcm-pending-position-submissions'), gradeladder: submissionChannel<string, Result>('staffing.gradeladder', 'chnhcm-pending-gradeladder-submissions'), jobfunction: submissionChannel<string, Result>('staffing.jobfunction', 'chnhcm-pending-jobfunction-submissions'), jobfamily: submissionChannel<string, Result>('staffing.jobfamily', 'chnhcm-pending-jobfamily-submissions'), grade: submissionChannel<string, Result>('staffing.grade', 'chnhcm-pending-grade-submissions'), job: submissionChannel<string, Result>('staffing.job', 'chnhcm-pending-job-submissions') };
export const fieldLabels: Record<string, string> = { departmentId:'所属部门', jobId:'职务', locationId:'工作地点', gradeId:'职级', gradeIds: '职级成员', code: '编码', name: '名称', nameEn: '英文名称', description: '说明', jobFamilyId: '职务系列', jobFunctionId: '职类', responsibilities: '主要职责', requirements: '任职要求', status: '状态', effectiveDate: '生效日期' };
export const dataFields = (k: Kind) => ['name', 'nameEn', ...(k === 'position' ? positionFields : k !== 'job' ? ['description'] : ['jobFamilyId', 'jobFunctionId', 'responsibilities', 'requirements'])];
export const statusLabel = (s?: string) => ({ active: '启用', inactive: '停用', not_effective: '未生效' }[s ?? ''] ?? '受限');
export const operationLabel: Record<string, string> = { CREATE: '新建', ADD: '新增记录', CHANGE: '数据更改', CORRECT: '更正记录', REDATE: '调整日期', DELETE: '删除记录', DEACTIVATE: '停用', REACTIVATE: '重新启用', INSERT_LIFECYCLE_PAIR: '插入停用与重新启用', MAINTAIN_LIFECYCLE_PAIR: '维护停用与重新启用', DELETE_LIFECYCLE_PAIR: '删除停用与重新启用' };
export function queryFrom(params: URLSearchParams): Query {
  const query: Query = {};
  for (const key of ['q','status','jobFamilyId','jobFunctionId','scopeDepartmentId','filterField','filterOperator','filterValue','sort'] as const) {
    if (params.has(key)) query[key] = params.get(key)!;
  }
  if (params.get('jobFunctionMissing') === 'true') query.jobFunctionMissing = true;
  if (params.has('scopeDepartmentId') || params.has('includeDescendants')) {
    query.includeDescendants = params.get('includeDescendants') !== 'false';
  }
  query.page = Number(params.get('page') ?? 1);
  query.pageSize = Number(params.get('pageSize') ?? 20);
  return query;
}
const headers = (p: Principal) => ({ 'X-CSRF-Token': p.csrf });
export const context = (k: Kind, signal?: AbortSignal) => api.GET(`${bases[k]}/context`, { signal }).then(unwrap);
export const list = (k: Kind, q: Query, signal?: AbortSignal) => api.GET(bases[k], { params: { query: q }, signal }).then(unwrap);
export const detail = (k: Kind, id: string, q: {
    sourceVersion?: string;
    selectedRecordId?: string;
}, signal?: AbortSignal) => api.GET(`${bases[k]}/{id}`, { params: { path: { id }, query: q }, signal }).then(unwrap);
export const navigation = (k: Kind, id: string, q: Query, signal?: AbortSignal) => api.GET(`${bases[k]}/{id}/navigation`, { params: { path: { id }, query: q }, signal }).then(unwrap);
export const log = (k: Kind, id: string, query: Models['StaffingLogQuery'], signal?: AbortSignal) => api.GET(`${bases[k]}/{id}/changes`, { params: { path: { id }, query }, signal }).then(unwrap);
export const reasons = (k: Kind, operationType: string, signal?: AbortSignal) => api.GET(`${bases[k]}/operation-reasons`, { params: { query: { operationType } }, signal }).then(unwrap);
export const classificationReferences = (field: JobClassificationField, p: Principal, body: Input, signal: AbortSignal) =>
  api.POST(field === 'jobFamilyId' ? '/api/v1/jobs/reference-context' : '/api/v1/jobs/function-reference-context',
    { params: { header: headers(p) }, body, signal }).then(unwrap);
export const positionReferences = (p: Principal, field: PositionField, body: Input, signal: AbortSignal) => api.POST('/api/v1/positions/reference-context', { params: { header: headers(p), query:{field} }, body:body as Models['PositionInput'], signal }).then(unwrap);
export const gradeReferences = (k: 'job' | 'gradeladder', p: Principal, body: Input, signal?: AbortSignal) => api.POST(`${bases[k]}/grade-reference-context`, { params: { header: headers(p) }, body, signal }).then(unwrap);
export const preview = (k: Kind, p: Principal, id: string, body: Input, signal?: AbortSignal) => api.POST(`${bases[k]}/{id}/preview`, { params: { path: { id }, header: headers(p) }, body, signal }).then(unwrap);
function checkedResult(k: Kind, id: string, result: Result): Result {
    if (result.kind !== k || result.submissionId !== id || !['accepted', 'rejected', 'no_change', 'not_applied', 'pending'].includes(result.outcome)) throw new Error('暂时无法确认原提交结果');
    return result;
}
export const resolve = (k: Kind, p: Principal, ref: Pending) => api.POST(`${bases[k]}/submissions/{id}/resolve`, { params: { path: { id: ref.submissionId }, header: headers(p) } }).then(unwrap).then(r => checkedResult(k, ref.submissionId, r));
export class RejectedSubmission extends ApiError {
    readonly terminal = true;
}
export async function send(k: Kind, p: Principal, objectId: string, body: Input, id: string): Promise<Result> {
    const action = body.action ?? 'CREATE', recordId = body.recordId ?? '';
    const suffix: Record<string, string> = { CREATE: '', ADD: '/{id}/records', CORRECT: '/{id}/records/{recordId}/correct', REDATE: '/{id}/records/{recordId}/redate', DELETE: '/{id}/records/{recordId}/delete', DEACTIVATE: '/{id}/deactivate', REACTIVATE: '/{id}/reactivate', INSERT_LIFECYCLE_PAIR: '/{id}/lifecycle-pairs/insert', MAINTAIN_LIFECYCLE_PAIR: '/{id}/lifecycle-pairs/maintain', DELETE_LIFECYCLE_PAIR: '/{id}/lifecycle-pairs/delete' };
    // Dynamic dispatch is confined to this adapter. Each route has an independent
    // generated input shape; Go also validates the selected operation before writing.
    const route = (bases[k] + suffix[action]) as '/api/v1/jobs/{id}/records/{recordId}/correct';
    const { action: _action, recordId: _record, objectId: _object, ...input } = body;
    const params = { path: { id: objectId, recordId }, header: { ...headers(p), 'Idempotency-Key': id } };
    const r = k === 'grade'
        ? await api.POST((bases.grade + suffix[action]) as '/api/v1/grades/{id}/records/{recordId}/correct', { params, body: input as Models['GradeCorrectInput'] })
        : k === 'gradeladder' ? await api.POST((bases.gradeladder + suffix[action]) as '/api/v1/grade-ladders/{id}/records/{recordId}/correct', { params, body: input as Models['GradeLadderCorrectInput'] })
        : k === 'position' ? await api.POST((bases.position + suffix[action]) as '/api/v1/positions/{id}/records/{recordId}/correct', { params, body: input as Models['PositionCorrectInput'] })
        : await api.POST(route, { params, body: input as Models['JobCorrectInput'] });
    const result = (r.data ?? r.error) as Result | undefined;
    if (result?.kind === k && result.submissionId === id && result.outcome === 'rejected')
        throw new RejectedSubmission(result.failure?.code ?? 'M03_REJECTED', result.failure?.message ?? '提交已拒绝', result.failureStatus ?? 409, result.failure?.field);
    if (result?.kind === k && result.submissionId === id && result.outcome === 'not_applied')
        throw new RejectedSubmission('M03_SUBMISSION_CLOSED', '原提交已关闭，请核对输入后重新保存', 409);
    return checkedResult(k, id, unwrap<Result>(r));
}
export async function select(k: Kind, p: Principal, op: 'start' | 'resolve' | 'cancel', body: Models['StaffingSelectionInput'], signal?: AbortSignal) { return unwrap(await api.POST(`${bases[k]}/selection/${op}`, { params: { header: headers(p) }, body, signal })); }
export async function exportCSV(k: Kind, p: Principal, body: Models['StaffingSelectionInput'], signal: AbortSignal) { const same = sessionGuard(); const r = await api.POST(`${bases[k]}/export`, { params: { header: headers(p) }, body, signal, parseAs: 'blob' }); const blob = unwrap(r); signal.throwIfAborted(); if (!same())
    throw new Error('会话已变化'); const url = URL.createObjectURL(blob); const a = document.createElement('a'); a.href = url; a.download = `chnhcm-${k}.csv`; a.click(); setTimeout(() => URL.revokeObjectURL(url), 1000); }
