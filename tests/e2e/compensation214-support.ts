import { randomUUID } from 'node:crypto';
import { expect, type Page } from './fixtures';
import { openPersonnel } from './app-navigation';
import { seed } from './hire-fixture';
import { request } from './dated-object-support';
import { database } from './community-list-support.mjs';
import { seed308 } from './staffing308-support';

export const entry = '2026-11-01',
  first = '2026-11-30',
  december = '2026-12-01',
  january = '2027-01-01';
export async function postAwardFixture(page: Page, path: string, input: object) {
  const v = await request(page, 'POST', '/api/v1/' + path, input, randomUUID());
  expect(v.status, v.body.message + ' ' + path).toBe(200);
  return v.body;
}
export async function prepareTerm(page: Page, gradeId: string, name?: string, familyId?: string) {
  const sample = await seed(page, true, entry, { gradeId, familyId, name, arrival: entry });
  const hire = await postAwardFixture(page, 'hires', {
    person: sample.id,
    employer: sample.employerId,
    department: sample.departmentId,
    job: sample.jobId,
    arrival: entry,
    notice: '个人月薪样本任用决定',
  });
  const root = 'assignments/' + hire.assignment;
  const term = await request(page, 'GET', '/api/v1/' + root + '?businessAt=' + entry);
  await postAwardFixture(page, root + '/grade-records', {
    expectedSourceVersion: term.body.sourceVersion,
    businessDate: entry,
    grade: gradeId,
    notice: '专业职级原任用决定',
  });
  return { ...sample, assignment: hire.assignment as string };
}
export function grantPay(instance: string, actor: string, term: string, actions: string[]) {
  for (const id of [instance, actor, term]) expect(id).toMatch(/^[0-9a-f-]{36}$/);
  for (const a of actions)
    expect(['read', 'history', 'first', 'adjust', 'correct', 'withdraw', 'use']).toContain(a);
  database()(
    `INSERT INTO access.base_pay_grant(instance_id,principal_id,assignment_id,action) SELECT '${instance}','${actor}','${term}',unnest(ARRAY[${actions.map((a) => "'" + a + "'").join(',')}]) ON CONFLICT DO NOTHING`,
  );
}
export async function preparePay(page: Page) {
  const code = 'P' + randomUUID().slice(0, 7);
  const grade = await seed308(page, 'grade', undefined, { code });
  const term = await prepareTerm(page, grade.id);
  const rate = await postAwardFixture(page, 'compensation/grade-rates', {
    action: 'CREATE',
    gradeId: grade.id,
    currency: 'CNY',
    frequency: 'MONTH',
    unit: 'PER_MONTH',
    effectiveDate: entry,
    reason: '建立标准',
    changes: { form: 'RANGE', minimum: '10000', maximum: '15000', guidance: '12500' },
  });
  await postAwardFixture(page, 'compensation/grade-rates/' + rate.objectId, {
    action: 'ADD',
    recordId: rate.recordId,
    expectedRevision: rate.revision,
    expectedVersion: rate.sourceVersion,
    effectiveDate: december,
    reason: '薪酬制度调整',
    changes: { minimum: '11000', maximum: '16000' },
  });
  const basic = {
    id: randomUUID(),
    code: 'BASIC',
    name: '基本工资',
    nature: 'BASIC',
    method: 'GRADE_RATE',
    currency: 'CNY',
    frequency: 'MONTH',
    unit: 'PER_MONTH',
    precision: 2,
    rates: [{ gradeId: grade.id, rateId: rate.objectId }],
  };
  const extra = {
    id: randomUUID(),
    code: 'ALLOW',
    name: '岗位津贴',
    nature: 'ALLOWANCE',
    method: 'FIXED',
    currency: 'CNY',
    frequency: 'MONTH',
    unit: 'PER_MONTH',
    precision: 2,
    amount: '300',
  };
  const plan = await postAwardFixture(page, 'compensation/plans', {
    action: 'CREATE',
    code,
    effectiveDate: entry,
    reason: '建立计划',
    changes: { name: '专业月薪' + code, scope: { mode: 'ENTERPRISE' }, components: [basic, extra] },
  });
  await postAwardFixture(page, 'compensation/plans/' + plan.objectId, {
    action: 'ADD',
    recordId: plan.recordId,
    expectedRevision: plan.revision,
    expectedVersion: plan.sourceVersion,
    effectiveDate: january,
    reason: '薪酬制度调整',
    changes: { components: [basic, { ...extra, amount: '350' }] },
  });
  const who = (await request(page, 'GET', '/api/v1/session')).body;
  grantPay(who.instanceId, who.subjectId, term.assignment, [
    'read',
    'history',
    'first',
    'adjust',
    'correct',
    'withdraw',
    'use',
  ]);
  return term;
}
export async function openPay(page: Page, sample: { name: string; assignment: string }) {
  await openPersonnel(page);
  const search = page.getByRole('textbox', { name: '姓名或人员编号' });
  await search.fill(sample.name);
  await search.press('Enter');
  const row = page
    .locator('a[data-object-link][href*="/assignments/' + sample.assignment + '"]')
    .filter({ hasText: sample.name });
  await row.click();
  const job = page.getByRole('tab', { name: '职务信息', exact: true }),
    pay = page.getByRole('tab', { name: '薪酬', exact: true });
  await expect(job).toHaveAttribute('aria-selected', 'true');
  await expect(pay).toBeVisible();
  await job.focus();
  await job.press('ArrowRight');
  await pay.press('Enter');
  await expect(pay).toHaveAttribute('aria-selected', 'true');
  await expect(pay).toBeFocused();
  await expect(page.getByLabel('工资查询日')).toBeVisible();
}
