import { randomUUID } from 'node:crypto';
import { expect, type Page } from './fixtures';
import { request, apiBU, apiDepartment } from './dated-object-support';
import { seed308 } from './staffing308-support';

export async function signIn(page: Page, login = 'admin') {
  await page.goto('/');
  await page.getByLabel('登录名').fill(login);
  await page.getByLabel('密码').fill('admin1234567');
  await page.getByRole('button', { name: '登录', exact: true }).click();
  await expect(page.getByRole('button', { name: '退出', exact: true })).toBeVisible();
}
export async function seed(
  page: Page,
  withDocument = true,
  employerFrom = '2027-04-01',
  options: { name?: string; familyId?: string; gradeId?: string; arrival?: string } = {},
) {
  const arrival = options.arrival ?? '2027-04-01';
  const suffix = randomUUID().slice(0, 8),
    name = options.name ?? '入职试办' + suffix,
    department = '研发' + suffix,
    employer = '用人单位' + suffix,
    job = '专员' + suffix;
  const bu = await apiBU(page, '管理单元' + suffix);
  const dep = await apiDepartment(page, bu.id, { name: department });
  const family = options.familyId ? { id: options.familyId } : await seed308(page, 'jobfamily');
  const positionJob = await seed308(page, 'job', family.id, {
    name: job,
    gradeIds: options.gradeId ? [options.gradeId] : undefined,
  });
  const send = async (path: string, input: object, status: number) => {
    const r = await request(page, 'POST', '/api/v1/' + path, input, randomUUID());
    expect(r.status, r.body.message).toBe(status);
    return r.body;
  };
  const countries = await request(page, 'GET', '/api/v1/country-regions?q=CN');
  const cn = countries.body.items.find((r: { code: string }) => r.code === 'CN');
  const entity = await send(
    'legal-entities',
    {
      code: 'H' + suffix,
      name: employer,
      countryRegionId: cn.id,
      subjectType: 'legal_person',
      registrationType: '登记证',
      registrationNumber: 'T' + suffix,
      establishingOrganization: '合成设立单位',
      operationType: 'CREATE',
      operationReason: 'NEW_LEGAL_ENTITY',
    },
    201,
  );
  await send(
    'legal-entities/' + entity.legalEntityId + '/purposes',
    {
      expectedVersion: '',
      row: {
        id: randomUUID(),
        role: 'employer',
        kind: 'declaration',
        basis: '设立及用工资格',
        from: employerFrom,
      },
    },
    200,
  );
  const policy = await request(page, 'GET', '/api/v1/assignment-policies');
  if (
    !policy.body.rows.some(
      (r: { category: string; purpose: string; from: string; until?: string }) =>
        r.category === 'Employee' && r.purpose === 'hire' && r.from <= arrival && (!r.until || arrival < r.until),
    )
  ) {
    await send(
      'assignment-policies',
      {
        expectedVersion: policy.body.version,
        row: { id: randomUUID(), category: 'Employee', purpose: 'hire', mode: 'JobOnly', from: arrival },
      },
      200,
    );
  }
  const saved = await send(
    'persons',
    {
      submissionId: randomUUID(),
      name,
      ...(withDocument ? { primaryDocument: { kind: 'passport', number: 'SYN-' + suffix } } : {}),
    },
    200,
  );
  return {
    name,
    department,
    employer,
    job,
    id: saved.id,
    buId: bu.id,
    departmentId: dep.id,
    employerId: entity.legalEntityId,
    jobId: positionJob.id,
  };
}
