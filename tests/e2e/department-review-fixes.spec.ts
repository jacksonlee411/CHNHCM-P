import { randomUUID } from 'node:crypto';
import { test, expect, openMoreActions, type Page } from './fixtures';
import { open, choose } from './bu-support';
import { withPolicy } from './grants';
import { objects, departmentKind as dep, menu, fillNew, request, readObject, apiBU, apiDepartment } from './dated-object-support';

const statusField = (page: Page) => page.locator('.object-detail-field').filter({ has: page.getByText('状态', { exact: true }) });
const detailPath = (segment: string, id: string) => `/api/v1/${segment}/${id}`;
async function clearParent(page: Page) {
  await page.getByRole('combobox', { name: '行政上级', exact: true }).focus();
  await page.getByRole('button', { name: '清除行政上级，设为企业根部门', exact: true }).click();
}

test('无行政上级写权仍可从正常菜单创建根部门，未选可选字段不声明写入', async ({ page }) => {
  await open(page);
  const bu = await apiBU(page);
  await withPolicy(page, async (where, mutate) => {
    mutate(`UPDATE access.subject_grant SET write_fields=array_remove(write_fields,'parentDepartmentId') WHERE ${where} AND action='department.create'`);
    const context = (await request(page, 'GET', '/api/v1/departments/context')).body;
    expect(context.capabilities.create).toBe(true);
    expect(context.capabilities.writeFields).not.toContain('parentDepartmentId');
    const control = await request(page, 'POST', '/api/v1/departments', {
      code: 'ROOT_' + randomUUID().slice(0, 8), name: 'API根部门对照', managementBuId: bu.id,
      effectiveDate: '2026-01-01', operationType: 'CREATE', operationReason: 'NEW_ORGANIZATION',
    }, randomUUID());
    expect(control.status).toBe(201);
    await menu(page, dep);
    await fillNew(page, dep, { name: '获权页面根部门', bu: bu.code });
    const posted = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === '/api/v1/departments');
    await page.getByRole('button', { name: '保存', exact: true }).click();
    const response = await posted;
    expect(response.status()).toBe(201);
    expect(response.request().postDataJSON()).not.toHaveProperty('parentDepartmentId');
    expect(response.request().postDataJSON()).not.toHaveProperty('shortName');
    await expect(page.locator('.object-form')).toHaveCount(0);
    const detail = await readObject(page, dep);
    expect(detail.capabilities.readFields).toContain('parentDepartmentId');
    expect(detail.selectedRecord).toMatchObject({ name: '获权页面根部门', managementBuId: bu.id });
    expect(detail.selectedRecord.parentDepartmentId ?? null).toBeNull();
    await expect(page.getByText('企业根部门（无行政上级）', { exact: true })).toBeVisible();
  });
});

test('明确选择上级可创建子部门，更正清除仍提交null，新建选后清除保存为根部门', async ({ page }) => {
  test.setTimeout(60000);
  await open(page);
  const bu = await apiBU(page), parent = await apiDepartment(page, bu.id);
  await menu(page, dep);
  await fillNew(page, dep, { bu: bu.code, parent: parent.code });
  let posted = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === '/api/v1/departments');
  await page.getByRole('button', { name: '保存', exact: true }).click();
  const childResponse = await posted;
  expect(childResponse.status()).toBe(201);
  expect(childResponse.request().postDataJSON().parentDepartmentId).toBe(parent.id);
  await expect(page.locator('.object-form')).toHaveCount(0);
  const child = await readObject(page, dep);
  expect(child.selectedRecord.parentDepartmentId).toBe(parent.id);
  await openMoreActions(page);
  await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
  await clearParent(page);
  posted = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname.endsWith('/correct'));
  await page.getByRole('button', { name: '保存', exact: true }).click();
  const corrected = await posted;
  expect(corrected.status()).toBe(200);
  expect(corrected.request().postDataJSON().changes).toMatchObject({ parentDepartmentId: null });
  await expect(page.locator('.object-form')).toHaveCount(0);
  const root = await readObject(page, dep);
  expect(root.capabilities.readFields).toContain('parentDepartmentId');
  expect(root.selectedRecord.parentDepartmentId ?? null).toBeNull();
  const historical = await request(page, 'GET', detailPath('departments', child.departmentId) + '?sourceVersion=' + child.resolvedSourceVersion);
  expect(historical.status).toBe(200);
  expect(historical.body.selectedRecord.parentDepartmentId).toBe(parent.id);
  await menu(page, dep);
  await fillNew(page, dep, { bu: bu.code, parent: parent.code });
  await clearParent(page);
  posted = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === '/api/v1/departments');
  await page.getByRole('button', { name: '保存', exact: true }).click();
  expect((await posted).status()).toBe(201);
  await expect(page.locator('.object-form')).toHaveCount(0);
  const created = await readObject(page, dep);
  expect(created.capabilities.readFields).toContain('parentDepartmentId');
  expect(created.selectedRecord.parentDepartmentId ?? null).toBeNull();
});

for (const fault of ['503', '断连'] as const) test(`部门状态撤读权后详情重读${fault}，取消编辑清除旧状态且恢复授权后可重读`, async ({ page }) => {
  await open(page);
  const bu = await apiBU(page), d = await apiDepartment(page, bu.id, { name: '状态撤权部门' });
  const path = detailPath('departments', d.id), reloadPattern = '**' + path + '?*';
  await page.goto('/settings/organization/departments/' + d.id);
  const before = await readObject(page, dep);
  await expect(statusField(page)).toContainText('启用');
  await openMoreActions(page);
  await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
  await withPolicy(page, async (where, mutate) => {
    mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'status') WHERE ${where} AND action='department.read'`);
    const actual = await page.request.get(path);
    expect(actual.status()).toBe(200);
    const current = await actual.json();
    expect(current.selectedRecord).not.toHaveProperty('state');
    for (const record of current.timeline) expect(record).not.toHaveProperty('state');
    let reloads = 0;
    await page.route(reloadPattern, route => {
      if (route.request().method() !== 'GET') return route.continue();
      reloads++;
      return fault === '503'
        ? route.fulfill({ status: 503, json: { code: 'M02_EXECUTION_LIMIT', message: '测试详情暂不可用' } })
        : route.abort('failed');
    });
    try {
      const preview = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === path + '/preview');
      await page.locator('#record-date').fill('2026-02-01');
      const response = await preview;
      expect(response.status()).toBe(200);
      expect((await response.json()).readFields).not.toContain('status');
      await expect.poll(() => reloads).toBeGreaterThan(0);
      await expect(page.locator('.object-form-error')).toBeVisible();
      await page.getByRole('button', { name: '取消编辑', exact: true }).click();
      await page.getByRole('button', { name: '放弃修改', exact: true }).click();
      await expect(page.locator('.object-form')).toHaveCount(0);
      await expect(statusField(page)).toContainText('无读取权限');
      await expect(statusField(page)).not.toContainText('启用');
      expect((await (await page.request.get(path)).json()).headSourceVersion).toBe(before.headSourceVersion);
    } finally { await page.unroute(reloadPattern); }
  });
  await page.reload();
  await expect(statusField(page)).toContainText('启用');
  expect((await readObject(page, dep)).headSourceVersion).toBe(before.headSourceVersion);
});

test('BU撤英文读权且重读失败仍保留其无需独立字段授权的状态', async ({ page }) => {
  await open(page);
  const bu = await apiBU(page), kind = objects[0], path = detailPath(kind.segment, bu.id), pattern = '**' + path + '?*';
  await page.goto('/settings/organization/business-units/' + bu.id);
  await expect(statusField(page)).toContainText('启用');
  await openMoreActions(page);
  await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
  await withPolicy(page, async (where, mutate) => {
    mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'nameEn') WHERE ${where} AND action='bu.read'`);
    await page.route(pattern, route => route.fulfill({ status: 503, json: { code: 'M02_EXECUTION_LIMIT', message: '测试详情暂不可用' } }));
    try {
      const preview = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === path + '/preview');
      await page.locator('#record-date').fill('2026-02-01');
      const response = await preview;
      expect(response.status()).toBe(200);
      expect((await response.json()).readFields).not.toContain('nameEn');
      await expect(page.locator('.object-form-error')).toContainText('测试详情暂不可用');
      await page.getByRole('button', { name: '取消编辑', exact: true }).click();
      await page.getByRole('button', { name: '放弃修改', exact: true }).click();
      await expect(page.locator('.object-form')).toHaveCount(0);
      await expect(statusField(page)).toContainText('启用');
    } finally { await page.unroute(pattern); }
  });
});

for (const operation of ['preview', 'correct'] as const) test(`部门${operation}撤权拒绝未带字段列表，详情重读503后清除旧状态`, async ({ page }) => {
  await open(page);
  const bu = await apiBU(page), d = await apiDepartment(page, bu.id), path = detailPath('departments', d.id), pattern = '**' + path + '?*';
  await page.goto('/settings/organization/departments/' + d.id);
  const before = await readObject(page, dep);
  await expect(statusField(page)).toContainText('启用');
  await openMoreActions(page);
  await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
  await withPolicy(page, async (where, mutate) => {
    mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'status') WHERE ${where} AND action='department.read'; UPDATE access.subject_grant SET write_fields=ARRAY[]::text[] WHERE ${where} AND action IN ('department.record.correct','department.first.redate')`);
    await page.route(pattern, route => route.fulfill({ status: 503, json: { code: 'M02_EXECUTION_LIMIT', message: '测试详情暂不可用' } }));
    try {
      const posted = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname.endsWith('/' + operation));
      if (operation === 'preview') await page.locator('#record-date').fill('2026-02-01');
      else {
        await page.getByRole('textbox', { name: '部门名称', exact: true }).fill('未保存草稿');
        await page.getByRole('button', { name: '保存', exact: true }).click();
      }
      const response = await posted;
      expect(response.status()).toBe(403);
      expect(await response.json()).not.toHaveProperty('readFields');
      await expect(page.locator('.object-form-error')).toContainText('测试详情暂不可用');
      await page.getByRole('button', { name: '取消编辑', exact: true }).click();
      if (await page.getByRole('dialog', { name: '尚有未保存修改', exact: true }).isVisible()) {
        await page.getByRole('button', { name: '放弃修改', exact: true }).click();
      }
      await expect(page.locator('.object-form')).toHaveCount(0);
      await expect(statusField(page)).toContainText('无读取权限');
    } finally { await page.unroute(pattern); }
  });
  const after = await readObject(page, dep);
  expect(after.headSourceVersion).toBe(before.headSourceVersion);
  expect(after.timeline).toEqual(before.timeline);
});

for (const kind of objects) test(`${kind.label}固定期间保存被撤权且重读失败，父详情按本对象状态读权投影`, async ({ page }) => {
  await open(page);
  const bu = await apiBU(page), object = kind === dep ? await apiDepartment(page, bu.id) : bu;
  const actionPrefix = kind === dep ? 'department' : 'bu', revokedField = kind === dep ? 'status' : 'nameEn';
  const path = detailPath(kind.segment, object.id), pattern = '**' + path + '?*';
  await page.goto('/settings/organization/' + kind.segment + '/' + object.id);
  const before = await readObject(page, kind);
  await expect(statusField(page)).toContainText('启用');
  await openMoreActions(page);
  await page.getByRole('menuitem', { name: '插入停用与重新启用', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: '插入停用与重新启用', exact: true });
  await dialog.getByLabel('停用日期', { exact: true }).fill('2026-03-01');
  await dialog.getByLabel('重新启用日期', { exact: true }).fill('2026-04-01');
  await choose(dialog, '停用原因', '业务终止');
  await expect(dialog.locator('#pair-reactivationReason')).toHaveText('业务恢复');
  await withPolicy(page, async (where, mutate) => {
    mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'${revokedField}') WHERE ${where} AND action='${actionPrefix}.read'; UPDATE access.subject_grant SET write_fields=ARRAY[]::text[] WHERE ${where} AND action='${actionPrefix}.deactivate'`);
    await page.route(pattern, route => route.fulfill({ status: 503, json: { code: 'M02_EXECUTION_LIMIT', message: '测试详情暂不可用' } }));
    try {
      const posted = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === path + '/lifecycle-pairs');
      await dialog.getByRole('button', { name: '保存', exact: true }).click();
      const response = await posted;
      expect(response.status()).toBe(403);
      const error = await response.json();
      // A rejection may omit the field projection; the UI must then clear all
      // independently protected fields before attempting its detail reload.
      expect(error.readFields ?? []).not.toContain(revokedField);
      await expect(dialog).toContainText('测试详情暂不可用');
      await dialog.getByRole('button', { name: '取消', exact: true }).click();
      await page.getByRole('button', { name: '放弃修改', exact: true }).click();
      await expect(dialog).toHaveCount(0);
      await expect(statusField(page)).toContainText(kind === dep ? '无读取权限' : '启用');
    } finally { await page.unroute(pattern); }
  });
  const after = await readObject(page, kind);
  expect(after.headSourceVersion).toBe(before.headSourceVersion);
  expect(after.timeline).toEqual(before.timeline);
});
