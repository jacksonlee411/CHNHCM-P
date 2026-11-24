import { lazyCredentials } from './credentials.mjs';
import { expect } from '@playwright/test';
import { execFileSync } from 'node:child_process';
import { readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { randomUUID } from 'node:crypto';
import { localConnection, verifyLocalTarget } from '../../scripts/local-runtime.mjs';

export const connection = localConnection();
export const credentials = lazyCredentials();
export const listPath = '/settings/organization/business-units';
export const out = process.env.CHNHCM_T03_OUT ?? '.local/111-community-t03';
export async function openList(page) {
  await expect(page.locator('[data-navigation-mode]')).toBeVisible();
  await page.evaluate(() => document.fonts.ready);
  if (await page.getByRole('button', { name: '打开导航', exact: true }).isVisible()) {
    await page.getByRole('button', { name: '打开导航', exact: true }).click();
    await page.locator('#navigation-drawer').getByRole('button', { name: '业务单元', exact: true }).click();
  } else {
    await page.getByRole('button', { name: '组织配置', exact: true }).click();
    await page.getByRole('menuitem', { name: '业务单元', exact: true }).click();
  }
  await expect(page.getByRole('grid', { name: '业务单元列表' })).toBeVisible();
}
export async function login(page, reader = false) {
  verifyLocalTarget();
  await page.goto(connection.webOrigin);
  await page.getByLabel('登录名').fill(reader ? credentials.reader_login : credentials.login);
  await page.getByLabel('密码').fill(reader ? credentials.reader_password : credentials.password);
  await new Promise(r => setTimeout(r, 3100));
  await page.getByRole('button', { name: '登录', exact: true }).click();
  await openList(page);
}
export const isList = response => new URL(response.url()).pathname === '/api/v1/business-units' && response.request().method() === 'GET';
export async function query(page, q) {
  await page.getByLabel('搜索编码或名称').fill(q);
  const pending = page.waitForResponse(response => isList(response) && (new URL(response.url()).searchParams.get('q') ?? '') === q);
  await page.getByRole('button', { name: '查询', exact: true }).click();
  const response = await pending;
  expect(response.status()).toBe(200);
  const body = await response.json();
  await expect(page.getByText(`${body.count ? (body.page - 1) * body.pageSize + 1 : 0}–${Math.min(body.page * body.pageSize, body.count)} / ${body.count}`, { exact: true })).toBeVisible();
  return body;
}
export async function panel(page, name) {
  await page.getByRole('button', { name: '更多操作', exact: true }).click();
  await page.getByRole('menuitem', { name, exact: true }).click();
}
export async function option(page, label, value) {
  await page.getByRole('combobox', { name: label, exact: true }).click();
  await page.getByRole('option', { name: value, exact: true }).click();
}
export async function seed(page) {
  const prefix = 'T03_' + randomUUID().slice(0, 8), items = [];
  const session = await (await page.request.get('/api/v1/session')).json();
  const context = await (await page.request.get('/api/v1/organization/context')).json();
  const future = String(Number(context.currentDate.slice(0, 4)) + 1) + '-12-01';
  const headers = () => ({ Origin: connection.webOrigin, 'X-CSRF-Token': session.csrf, 'Idempotency-Key': randomUUID() });
  for (let i = 0; i < 73; i++) {
    const code = prefix + '_' + String(i).padStart(3, '0') + (i === 22 ? '_LONG_RESPONSIVE_CODE' : '');
    const name = (i < 20 ? prefix + 'G20 ' : '') + (i < 21 ? prefix + 'G21 ' : '') + '列表样本 ' + String(73 - i).padStart(3, '0') + (i === 22 ? '长名称换行验证'.repeat(18) : '');
    const effectiveDate = i === 72 ? future : '2026-01-01';
    const response = await page.request.post('/api/v1/business-units', { headers: headers(), data: { code, name, nameEn: 'PrivateEnglish' + i, effectiveDate, operationType: 'CREATE', operationReason: 'NEW_ORGANIZATION' } });
    expect(response.status(), JSON.stringify(await response.json())).toBe(201);
    const result = await response.json();
    items.push({ id: result.buId, code, name, effectiveDate });
    if (i === 71) {
      const deactivated = await page.request.post('/api/v1/business-units/' + result.buId + '/deactivate', { headers: headers(), data: { anchorRecordId: result.selectedRecordId, expectedSourceVersion: result.sourceVersion, effectiveDate: '2026-02-01', operationType: 'DEACTIVATE', operationReason: 'BUSINESS_CLOSURE' } });
      expect(deactivated.status()).toBe(201);
    }
  }
  const fixture = { prefix, items, currentDate: context.currentDate, future };
  mkdirSync(out, { recursive: true });
  writeFileSync(out + '/fixtures.json', JSON.stringify(fixture, null, 2) + '\n');
  return fixture;
}
export function database() {
  const container = execFileSync('docker', ['ps', '-q', '--filter', 'label=com.docker.compose.project=' + connection.project, '--filter', 'label=com.docker.compose.service=postgres'], { encoding: 'utf8' }).trim();
  expect(container).toMatch(/^[0-9a-f]+$/);
  return statement => execFileSync('docker', ['exec', '-i', container, 'psql', '-U', 'postgres', '-d', 'chnhcm', '-X', '-Atq', '-v', 'ON_ERROR_STOP=1'], { input: statement, encoding: 'utf8' }).trim();
}
export async function policy(page, run) {
  const sql = database(), session = await (await page.request.get('/api/v1/session')).json();
  const where = `instance_id='${session.instanceId}' AND principal_id='${session.subjectId}'`;
  const snapshot = sql(`SELECT json_agg(g) FROM access.subject_grant g WHERE ${where}`);
  const mutate = statement => sql(`BEGIN; SELECT principal_id FROM access.policy_guard WHERE ${where} FOR UPDATE; ${statement}; UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE ${where}; COMMIT;`);
  try { return await run(where, mutate); }
  finally { mutate(`DELETE FROM access.subject_grant WHERE ${where}; INSERT INTO access.subject_grant SELECT * FROM json_populate_recordset(NULL::access.subject_grant,'${snapshot.replaceAll("'", "''")}')`); }
}
