import { randomUUID } from 'node:crypto';
import { test, expect, openMoreActions } from './fixtures';
import { open, choose } from './bu-support';
import { menu, objects, request, apiBU, apiDepartment } from './dated-object-support';

for (const kind of objects) test(`${kind.label}清空原因后自动辅助不抢焦点或关闭关系候选，保存错误仍定位字段`, async ({ page }) => {
  test.setTimeout(60000); await open(page);
  const bu = await apiBU(page), parent = kind.segment === 'departments' ? await apiDepartment(page, bu.id) : undefined;
  const child = parent ? await apiDepartment(page, bu.id, { parent: parent.id }) : bu;
  const path = '/api/v1/' + kind.segment + '/' + child.id;
  const before = await request(page, 'GET', path);
  expect((await request(page, 'POST', path + '/deactivate', { anchorRecordId: before.body.selectedRecord.recordId,
    expectedSourceVersion: before.body.headSourceVersion, effectiveDate: '2026-02-01', operationType: 'DEACTIVATE', operationReason: 'ORG_RESTRUCTURING' }, randomUUID())).status).toBe(201);
  await menu(page, kind); await page.getByPlaceholder('搜索编码或名称').fill(child.code);
  await page.getByRole('button', { name: '查询', exact: true }).click();
  await page.getByRole('link', { name: parent ? child.code : bu.name, exact: true }).click();
  await openMoreActions(page); await page.getByRole('menuitem', { name: '重新启用' + kind.label, exact: true }).click();
  await expect(page.getByRole('combobox', { name: '操作原因' })).toHaveText('业务恢复');
  await page.getByRole('button', { name: '取消编辑', exact: true }).click();
  await expect(page.locator('.object-form')).toHaveCount(0);
  await expect(page.getByRole('dialog')).toHaveCount(0);
  let release!: () => void, held = false;
  const delay = new Promise<void>(resolve => { release = resolve; });
  await page.route('**' + path + '/preview', async route => {
    if (held || route.request().postDataJSON()?.operationReason) return route.continue();
    held = true; const response = await route.fetch(); expect(response.status()).toBe(parent ? 400 : 200);
    await delay; await route.fulfill({ response });
  });
  try {
    await openMoreActions(page); await page.getByRole('menuitem', { name: '重新启用' + kind.label, exact: true }).click();
    await expect(page.getByRole('combobox', { name: '操作原因' })).toHaveText('业务恢复');
    await choose(page, '操作原因', '请选择');
    await expect.poll(() => held).toBe(true);
    const focused = parent ? page.getByRole('combobox', { name: '行政上级', exact: true }) : page.getByRole('textbox', { name: kind.name, exact: true });
    await focused.click();
    if (parent) { await focused.fill(parent.code); await expect(page.getByRole('option').filter({ hasText: parent.code })).toBeVisible(); }
    const received = page.waitForResponse(r => new URL(r.url()).pathname === path + '/preview' && r.status() === (parent ? 400 : 200));
    release(); await received;
    if (parent) await expect(page.locator('.object-form-error')).toBeVisible();
    await expect(focused).toBeFocused();
    if (parent) { await expect(page.getByRole('option').filter({ hasText: parent.code })).toBeVisible(); await page.keyboard.press('Escape'); }
    await choose(page, '操作原因', '业务恢复');
    await page.getByRole('textbox', { name: kind.name, exact: true }).fill('恢复后名称');
    await expect(page.getByRole('button', { name: '确认重新启用', exact: true })).toBeEnabled();
    const saved = page.waitForResponse(r => new URL(r.url()).pathname === path + '/reactivate');
    await page.getByRole('button', { name: '确认重新启用', exact: true }).click(); expect((await saved).status()).toBe(201);
    await expect(page.locator('.object-form')).toHaveCount(0);
    await openMoreActions(page); await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
    await page.getByRole('textbox', { name: kind.name, exact: true }).fill('😀');
    await page.getByRole('button', { name: '保存', exact: true }).click();
    await expect(page.locator('.object-form-error')).toBeVisible();
    await expect(page.getByRole('textbox', { name: kind.name, exact: true })).toBeFocused();
  } finally { release(); await page.unroute('**' + path + '/preview'); }
});
