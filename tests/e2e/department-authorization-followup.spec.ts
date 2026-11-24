import { randomUUID } from 'node:crypto';
import { test, expect, openMoreActions, type Page } from './fixtures';
import { open, choose } from './bu-support';
import { withPolicy } from './grants';
import { objects, departmentKind as dep, menu, request, apiBU, apiDepartment, type ObjectKind } from './dated-object-support';

const detailField = (page: Page, label: string) => page.locator('.object-detail-field').filter({ has: page.getByText(label, { exact: true }) });
const logTable = (page: Page) => page.getByRole('table', { name: '变更日志', exact: true });
async function openDetails(page: Page, kind: ObjectKind, code: string, name: string) {
  await menu(page, kind);
  await page.getByPlaceholder('搜索编码或名称').fill(code);
  await page.getByRole('button', { name: '查询', exact: true }).click();
  await page.getByRole('link', { name, exact: true }).click();
}
async function fieldFilter(page: Page) {
  await page.getByRole('button', { name: '变更字段列菜单', exact: true }).click();
  await page.getByRole('menuitem', { name: '筛选…', exact: true }).click();
  return page.getByRole('dialog');
}
async function applyFieldFilter(page: Page, field: string) {
  const response = page.waitForResponse(r => r.request().method() === 'GET' && new URL(r.url()).pathname.endsWith('/changes') && new URL(r.url()).searchParams.get('filterValue') === field);
  await page.getByRole('dialog').getByRole('button', { name: '应用', exact: true }).click();
  expect((await response).status()).toBe(200);
  await expect(logTable(page)).toBeVisible();
  await expect(page).toHaveURL(u => u.searchParams.get('tab') === 'history' && u.searchParams.get('logFilterValue') === field);
}

for (const scenario of [
  { field: 'parentDepartmentId', reference: 'parentDepartment', label: '行政上级', endpoint: 'parents', fault: '503' },
  { field: 'parentDepartmentId', reference: 'parentDepartment', label: '行政上级', endpoint: 'parents', fault: '断连' },
  { field: 'managementBuId', reference: 'managementBu', label: '管理业务单元', endpoint: 'business-units', fault: '503' },
] as const) test(`关系候选撤${scenario.label}读权后重读${scenario.fault}，立即清理编辑及父详情，恢复授权可重读`, async ({ page }) => {
  test.setTimeout(60000);
  await open(page);
  const bu = await apiBU(page, '候选撤权事业部'), parent = await apiDepartment(page, bu.id, { name: '候选撤权上级' });
  const child = await apiDepartment(page, bu.id, { parent: parent.id });
  const path = '/api/v1/departments/' + child.id, pattern = '**' + path + '?*';
  const expectedName = scenario.field === 'parentDepartmentId' ? '候选撤权上级' : bu.name;
  const before = await request(page, 'GET', path); expect(before.status).toBe(200);
  await openDetails(page, dep, child.code, child.code);
  await test.step('打开更正记录并等候首次候选响应，尚未撤权', async () => {
    const initial = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === '/api/v1/departments/references/' + scenario.endpoint);
    await openMoreActions(page); await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
    expect((await initial).status()).toBe(200);
  });
  await expect(page.getByRole('combobox', { name: scenario.label, exact: true })).toHaveValue(new RegExp(expectedName));
  await withPolicy(page, async (where, mutate) => {
    mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'${scenario.field}') WHERE ${where} AND action='department.read'`);
    const response = await page.request.get(path); expect(response.status()).toBe(200);
    const current = await response.json();
    expect(current.capabilities.readFields).not.toContain(scenario.field);
    expect(current.selectedRecord).not.toHaveProperty(scenario.field);
    expect(current.selectedRecord).not.toHaveProperty(scenario.reference);
    let reloads = 0, release!: () => void;
    const held = new Promise<void>(resolve => { release = resolve; });
    await page.route(pattern, async route => {
      reloads++; await held;
      return scenario.fault === '503'
        ? route.fulfill({ status: 503, json: { code: 'M02_EXECUTION_LIMIT', message: '测试详情重读失败' } })
        : route.abort('failed');
    });
    try {
      const rejected = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === '/api/v1/departments/references/' + scenario.endpoint);
      await page.getByRole('combobox', { name: scenario.label, exact: true }).click();
      const denied = await rejected; expect(denied.status()).toBe(403);
      expect(await denied.json()).not.toHaveProperty('readFields');
      await expect.poll(() => reloads).toBeGreaterThan(0);
      // A slow or failed refresh must not extend the lifetime of revoked data.
      await expect(detailField(page, scenario.label)).toContainText('无读取权限');
      await expect(page.getByRole('combobox', { name: scenario.label, exact: true })).toHaveCount(0);
      release();
      await expect(page.locator('.object-form-error')).toBeVisible();
      await page.getByRole('button', { name: '取消编辑', exact: true }).click();
      await expect(page.locator('.object-form')).toHaveCount(0);
      await expect(detailField(page, scenario.label)).toContainText('无读取权限');
      await expect(detailField(page, scenario.label)).not.toContainText(expectedName);
    } finally { release(); await page.unroute(pattern); }
  });
  await page.reload();
  await expect(detailField(page, scenario.label)).toContainText(expectedName);
  const after = await request(page, 'GET', path); expect(after.status).toBe(200);
  expect(after.body.headSourceVersion).toBe(before.body.headSourceVersion);
  expect(after.body.selectedRecord).toEqual(before.body.selectedRecord);
});

for (const actions of ["'department.history.read'", "'department.read','department.history.read'"]) test(`日志按当前历史字段权隐藏状态筛选并恢复，撤权范围${actions}`, async ({ page }) => {
  test.setTimeout(60000);
  await open(page); const bu = await apiBU(page), d = await apiDepartment(page, bu.id);
  const path = '/api/v1/departments/' + d.id;
  const before = await request(page, 'GET', path); expect(before.status).toBe(200);
  await withPolicy(page, async (where, mutate) => {
    mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'status') WHERE ${where} AND action IN (${actions})`);
    const control = await request(page, 'GET', path + '/changes');
    expect(control.status).toBe(200); expect(control.body.readFields).not.toContain('status');
    await openDetails(page, dep, d.code, d.code);
    await page.getByRole('tab', { name: '变更日志', exact: true }).click();
    await expect(logTable(page)).toBeVisible();
    const dialog = await fieldFilter(page);
    await dialog.getByRole('combobox', { name: '值', exact: true }).click();
    await expect(page.getByRole('option', { name: '部门名称', exact: true })).toBeVisible();
    await expect(page.getByRole('option', { name: '状态', exact: true })).toHaveCount(0);
    await page.getByRole('option', { name: '部门名称', exact: true }).click();
    await applyFieldFilter(page, 'name');
  });
  const refreshed = page.waitForResponse(r => r.request().method() === 'GET' && new URL(r.url()).pathname === path + '/changes');
  await page.getByRole('button', { name: '刷新日志', exact: true }).click();
  expect((await refreshed).status()).toBe(200); await expect(logTable(page)).toBeVisible();
  const dialog = await fieldFilter(page); await choose(dialog, '值', '状态');
  await applyFieldFilter(page, 'state');
  const after = await request(page, 'GET', path); expect(after.status).toBe(200);
  expect(after.body.headSourceVersion).toBe(before.body.headSourceVersion);
});

test('BU日志保留自身无需独立status字段权的状态筛选', async ({ page }) => {
  await open(page); const bu = await apiBU(page, 'BU状态筛选_' + randomUUID().slice(0, 8));
  const path = '/api/v1/business-units/' + bu.id;
  const control = await request(page, 'GET', path + '/changes');
  expect(control.status).toBe(200); expect(control.body.readFields).not.toContain('status');
  await openDetails(page, objects[0], bu.code, bu.name);
  await page.getByRole('tab', { name: '变更日志', exact: true }).click();
  await expect(logTable(page)).toBeVisible();
  const dialog = await fieldFilter(page); await choose(dialog, '值', '状态');
  await applyFieldFilter(page, 'state');
});


test('关系候选拒绝后成功重读不循环，授权恢复后可显式重试', async ({ page }) => {
  test.setTimeout(60000);
  await open(page);
  const bu = await apiBU(page, '候选重试事业部'), parent = await apiDepartment(page, bu.id), child = await apiDepartment(page, bu.id, { parent: parent.id });
  const path = '/api/v1/departments/' + child.id;
  const before = await request(page, 'GET', path); expect(before.status).toBe(200);
  await openDetails(page, dep, child.code, child.code);
  const first = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/departments/references/parents');
  await openMoreActions(page); await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
  expect((await first).status()).toBe(200);
  await withPolicy(page, async (where, mutate) => {
    mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'parentDepartmentId') WHERE ${where} AND action='department.read'`);
    let rejected = 0, reloads = 0;
    const observe = (r: import('@playwright/test').Response) => {
      const url = new URL(r.url());
      if (url.pathname.includes('/departments/references/') && r.status() === 403) rejected++;
      if (url.pathname === path && r.status() === 200) reloads++;
    };
    page.on('response', observe);
    try {
      await page.getByRole('combobox', { name: '行政上级', exact: true }).click();
      await expect.poll(() => reloads).toBeGreaterThan(0);
      // Observe beyond several candidate debounce cycles; a detail refresh must
      // settle even though both relationship selectors lose candidate authority.
      await page.waitForTimeout(1600);
      expect(reloads).toBeLessThanOrEqual(2); expect(rejected).toBeLessThanOrEqual(2);
      await expect(detailField(page, '行政上级')).toContainText('无读取权限');
      await expect(page.getByRole('combobox', { name: '管理业务单元', exact: true })).toHaveValue(new RegExp(bu.name));
      mutate(`UPDATE access.subject_grant SET read_fields=array_append(read_fields,'parentDepartmentId') WHERE ${where} AND action='department.read'`);
      await page.getByRole('combobox', { name: '管理业务单元', exact: true }).click();
      const retried = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/departments/references/business-units');
      await page.getByRole('button', { name: '重试读取候选', exact: true }).click();
      expect((await retried).status()).toBe(200);
      // A full suite has more than one page of BUs. The selected identity need
      // not be on the first unfiltered page returned by an explicit retry.
      await page.getByRole('combobox', { name: '管理业务单元', exact: true }).fill(bu.code);
      await expect(page.getByRole('option').filter({ hasText: bu.code })).toBeVisible();
      await page.keyboard.press('Escape');
      await page.getByRole('button', { name: '取消编辑', exact: true }).click();
      await expect(page.locator('.object-form')).toHaveCount(0);
    } finally { page.off('response', observe); }
  });
  await page.reload(); await expect(detailField(page, '行政上级')).toContainText(parent.code);
  const after = await request(page, 'GET', path); expect(after.status).toBe(200);
  expect(after.body.headSourceVersion).toBe(before.body.headSourceVersion);
  expect(after.body.selectedRecord).toEqual(before.body.selectedRecord);
});
