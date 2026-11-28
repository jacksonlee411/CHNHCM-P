import { api, unwrap, type Models } from '@chnhcm/api-client';
import { submissionChannel, type Pending } from '../../shared/dated/submission';

const base = '/api/v1/compensation/assignments/{assignmentId}/base-pay';
export type Award = Models['PayAward'];
export type Basis = Models['PayBasis'];
export type Command = Models['ConfirmAward'];
export type AwardResult = Models['AwardResolution'];
export type SavedAward = Models['AwardReceipt'];
export const awardChannel = submissionChannel<string, AwardResult>(
  'compensation.award',
  'chnhcm-compensation-award-pending',
);
export const awardLabels: Record<string, string> = {
  FIRST: '首次定薪',
  ADJUST: '调薪',
  CORRECT: '更正',
  WITHDRAW: '撤去',
};
export const payAccess = (assignmentId: string, businessAt?: string) =>
  api
    .GET('/api/v1/compensation/assignments/{assignmentId}/base-pay/access', {
      params: { path: { assignmentId }, query: { businessAt } },
    })
    .then(unwrap);
export const payDetail = (assignmentId: string, businessAt?: string, candidatePage?: number) =>
  api
    .GET(base, {
      params: {
        path: { assignmentId },
        query: { businessAt, candidatePage: candidatePage === 1 ? undefined : candidatePage },
      },
    })
    .then(unwrap);
export const payContext = (assignmentId: string, businessAt: string, candidatePage: number, planId?: string) =>
  api
    .GET('/api/v1/compensation/assignments/{assignmentId}/base-pay/context', {
      params: {
        path: { assignmentId },
        query: { businessAt, candidatePage: candidatePage === 1 ? undefined : candidatePage, planId },
      },
    })
    .then(unwrap);
export const awardOriginal = (awardId: string, revision: string) =>
  api
    .GET('/api/v1/compensation/awards/{awardId}/revisions/{revision}', { params: { path: { awardId, revision } } })
    .then(unwrap);
const accepted = (r: Models['AwardReceipt']): AwardResult => ({ ...r, outcome: 'accepted' });
// Only the mounted form keeps the original payload. Persistent recovery stores identities only.
const payloads = new WeakMap<Command, string>();
const pendingBodies = new Map<string, { assignmentId: string; command: Command }>();
const ownerKey = (session: Models['Principal'], id: string) =>
  [session.instanceId, session.subjectId, id].join('/');
export function releaseAward(body: Command) {
  const key = payloads.get(body);
  if (key) {
    pendingBodies.delete(key);
    payloads.delete(body);
  }
}
export const recoverAward = (session: Models['Principal'], ref: Pending) =>
  api
    .POST('/api/v1/compensation/award-receipts/{submissionId}/resolve', {
      params: { path: { submissionId: ref.submissionId }, header: { 'X-CSRF-Token': session.csrf } },
      body: pendingBodies.get(ownerKey(session, ref.submissionId)) ?? {},
    })
    .then(unwrap);
export function submitAward(
  session: Models['Principal'],
  assignmentId: string,
  body: Command,
  key: string,
  confirmation?: string,
) {
  const owned = ownerKey(session, key);
  payloads.set(body, owned);
  pendingBodies.set(owned, { assignmentId, command: body });
  return api
    .POST(base, {
      params: {
        path: { assignmentId },
        header: { 'X-CSRF-Token': session.csrf, 'Idempotency-Key': key, 'X-Award-Confirmation': confirmation },
      },
      body,
    })
    .then(unwrap)
    .then(accepted);
}
export function payReturn(path: string) {
  const [route, query] = path.split('?');
  const uuid = '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}';
  return (
    new RegExp('^/personnel/employees/' + uuid + '/assignments/' + uuid + '$', 'i').test(route) &&
    new URLSearchParams(query).get('tab') === 'pay'
  );
}
