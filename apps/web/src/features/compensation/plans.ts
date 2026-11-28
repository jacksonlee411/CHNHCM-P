import { api, unwrap, type Models } from '@chnhcm/api-client';
import { submissionChannel, type Pending } from '../../shared/dated/submission';
import { type Principal, type Receipt } from './rates';

export const planPath = '/settings/compensation/plans';
export const planTitle = '薪酬计划';
export type Plan = Models['CompensationPlan'];
export type PlanDetail = Models['CompensationPlanDetail'];
export type PlanRecord = Models['CompensationPlanRecord'];
export type PlanChange = Models['CompensationPlanChange'];
export type Component = Models['CompensationPlanComponent'];
export type PlanInput = Models['CompensationPlanInput'];
export type RateChoice = Models['CompensationRateCandidate'];
export type ScopeChoice = Models['CompensationScopeChoice'];
export type NamedChoice = ScopeChoice & { name: string; code?: string; effectiveDate?: string; endExclusive?: string };
export const planChannel = submissionChannel<string, Receipt>('compensation.plan', 'chnhcm-compensation-plan-pending');
const base = '/api/v1/compensation/plans';
export const planContext = () => api.GET('/api/v1/compensation/plans/create-context').then(unwrap);
export const listPlans = (query: { q?: string; status?: string; page: number; pageSize: number }) =>
  api.GET(base, { params: { query } }).then(unwrap);
export const readPlan = (id: string, recordId?: string, businessAt?: string) =>
  api
    .GET('/api/v1/compensation/plans/{id}', { params: { path: { id }, query: { recordId, businessAt } } })
    .then(unwrap);
export const planHistory = (id: string, sourceVersion?: string) =>
  api
    .GET('/api/v1/compensation/plans/{id}/history', { params: { path: { id }, query: { sourceVersion } } })
    .then(unwrap);
export const rateCandidates = (businessAt: string, q: string, currency: string, frequency: string) =>
  api
    .GET('/api/v1/compensation/plans/rate-candidates', { params: { query: { businessAt, q, currency, frequency } } })
    .then(unwrap);
export async function scopeCandidates(dimension: ScopeChoice['dimension'], q: string) {
  const query = { q: q || undefined, page: 1, pageSize: 50 as const };
  if (dimension === 'legalEntity') {
    const list = unwrap(await api.GET('/api/v1/legal-entities', { params: { query } }));
    return list.items.map((r) => ({
      id: r.id,
      code: r.code ?? undefined,
      name: r.name ?? '',
      effectiveDate: r.effectiveDate ?? undefined,
    }));
  }
  const list =
    dimension === 'jobFamily'
      ? unwrap(await api.GET('/api/v1/job-families', { params: { query } }))
      : unwrap(await api.GET('/api/v1/grades', { params: { query } }));
  return list.items.map((r) => ({
    id: r.id,
    code: r.code ?? undefined,
    name: r.name ?? '',
    effectiveDate: r.effectiveDate ?? undefined,
    endExclusive: r.endExclusive ?? undefined,
  }));
}
export function savePlan(session: Principal, id: string, body: PlanInput, submission: string) {
  const header = { 'X-CSRF-Token': session.csrf, 'Idempotency-Key': submission };
  return id
    ? api.POST('/api/v1/compensation/plans/{id}', { params: { path: { id }, header }, body }).then(unwrap)
    : api.POST(base, { params: { header }, body }).then(unwrap);
}
export function resolvePlan(session: Principal, ref: Pending) {
  return api
    .POST('/api/v1/compensation/plan-receipts/{submissionId}/resolve', {
      params: { path: { submissionId: ref.submissionId }, header: { 'X-CSRF-Token': session.csrf } },
    })
    .then(unwrap);
}
export const natures: Record<string, string> = {
  BASIC: '基本工资',
  ALLOWANCE: '津贴补贴',
  PERFORMANCE: '绩效工资',
  BONUS: '奖金',
  OTHER: '其他',
};
export const methods: Record<Component['method'], string> = {
  FIXED: '计划固定额',
  GRADE_RATE: '职级标准',
  PERCENT: '目标比例',
};
export const dimensionNames: Record<ScopeChoice['dimension'], string> = {
  legalEntity: '法律实体',
  jobFamily: '职务系列',
  grade: '职级',
};
export function scopeLabel(scope?: PlanRecord['scope']) {
  if (!scope) return '';
  if (scope.mode === 'ENTERPRISE') return '企业通用';
  const result = (scope.members ?? []).map((m) => (m.exclude ? '排除' : '') + (m.name || dimensionNames[m.dimension]));
  if (scope.restricted) result.push('部分范围受限');
  return result.join('、') || '指定范围';
}
