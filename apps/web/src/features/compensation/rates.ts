import { api, unwrap, type Models } from '@chnhcm/api-client';
import { submissionChannel, type Pending } from '../../shared/dated/submission';

export const ratePath = '/settings/compensation/grade-rates';
export const rateTitle = '职级薪酬标准';
const base = '/api/v1/compensation/grade-rates';
export type Principal = Models['Principal'];
export type Rate = Models['CompensationRate'];
export type Detail = Models['CompensationRateDetail'];
export type RateRecord = Models['CompensationRateRecord'];
export type Command = Models['CompensationRateInput'];
export type Action = Command['action'];
export type Receipt = Models['CompensationReceipt'];
export const rateChannel = submissionChannel<string, Receipt>('compensation.gradeRate', 'chnhcm-compensation-grade-rate-pending');
export const menuAccess = () => api.GET('/api/v1/compensation/access').then(unwrap);
export const createContext = () => api.GET('/api/v1/compensation/grade-rates/create-context').then(unwrap);
export const rateDetail = (id: string, recordId?: string) =>
  api.GET('/api/v1/compensation/grade-rates/{id}', { params: { path: { id }, query: { recordId } } }).then(unwrap);
export const rateHistory = (id: string, sourceVersion?: string) =>
  api.GET('/api/v1/compensation/grade-rates/{id}/history', { params: { path: { id }, query: { sourceVersion } } }).then(unwrap);
export function rateList(query: { gradeId?: string; q?: string; currency?: string; frequency?: string; form?: string; status?: string; page: number; pageSize: number }) {
  return api.GET(base, { params: { query } }).then(unwrap);
}
export const actionLabel: Record<Action, string> = { CREATE: '新建', ADD: '新增记录', CORRECT: '更正记录', DEACTIVATE: '停用', REACTIVATE: '重新启用', DELETE: '删除此记录' };
export const formLabels: Record<string, string> = { FIXED: '定额', RANGE: '连续薪带' };
export const statusLabels: Record<string, string> = { ACTIVE: '启用', INACTIVE: '停用', FUTURE: '尚未生效' };
export const frequencies: Record<string, { name: string; unitName: string }> = {
  MONTH: { name: '月', unitName: '每月' },
  YEAR: { name: '年', unitName: '每年' },
  HOUR: { name: '小时', unitName: '每小时' },
  ONCE: { name: '一次性', unitName: '每次' },
};
export function moneyUnit(currency?: string, frequency?: string) {
  const unit = frequencies[frequency ?? '']?.unitName;
  return currency && unit ? (currency === 'CNY' ? '人民币元' : currency) + '/' + unit.replace('每', '') : '';
}
export const amountLabels = { amount: '定额', minimum: '下限', guidance: '指导值', maximum: '上限' };
export type AmountKey = keyof typeof amountLabels;
export const amountKeys = Object.keys(amountLabels) as AmountKey[];
export function money(value?: string) {
  if (value === undefined) return '';
  const [integer, fraction] = value.split('.');
  return integer.replace(/\B(?=(\d{3})+(?!\d))/g, ',') + (fraction === undefined ? '' : '.' + fraction);
}
export function gradeLabel(rate: RateRecord) {
  return [rate.grade?.code, rate.grade?.name].filter(Boolean).join(' ');
}
export function rateAmounts(rate: RateRecord) {
  if (rate.amount !== undefined) return money(rate.amount);
  if (rate.minimum !== undefined && rate.maximum !== undefined) return money(rate.minimum) + ' – ' + money(rate.maximum);
  return (['minimum', 'maximum', 'guidance'] as const).filter(key => rate[key] !== undefined)
    .map(key => amountLabels[key] + ' ' + money(rate[key])).join('，');
}
export function saveRate(session: Principal, id: string, body: Command, submission: string) {
  const header = { 'X-CSRF-Token': session.csrf, 'Idempotency-Key': submission };
  return id
    ? api.POST('/api/v1/compensation/grade-rates/{id}', { params: { path: { id }, header }, body }).then(unwrap)
    : api.POST(base, { params: { header }, body }).then(unwrap);
}
export function resolveRate(session: Principal, ref: Pending) {
  return api.POST('/api/v1/compensation/grade-rate-receipts/{submissionId}/resolve', {
    params: { path: { submissionId: ref.submissionId }, header: { 'X-CSRF-Token': session.csrf } },
  }).then(unwrap);
}
