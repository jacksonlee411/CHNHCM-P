import { test, expect, openMoreActions, type Page } from './fixtures';
import { open, select } from './bu-support';
import { request } from './dated-object-support';
import { randomUUID } from 'node:crypto';

test.describe.configure({ timeout: 90_000 });
const root = '/settings/basic-data/country-regions';
async function menu(page: Page, code?: string) {
  const listed = page.waitForResponse(r => r.status() === 200 && r.request().method() === 'GET' && new URL(r.url()).pathname === '/api/v1/country-regions' && !new URL(r.url()).searchParams.get('q') && (new URL(r.url()).searchParams.get('page') ?? '1') === '1');
  await page.getByRole('button', { name: '基础资料', exact: true }).click();
  await page.getByRole('menuitem', { name: '国家/地区', exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`${root}$`));
  await expect(page.getByRole('heading', { name: '国家/地区', exact: true })).toBeVisible();
  await listed;
  if (code) {
    await page.getByRole('textbox', { name: '搜索编码或名称' }).fill(code);
    await page.getByRole('button', { name: '查询', exact: true }).click();
    await page.getByRole('row').filter({ has: page.getByRole('gridcell', { name: code, exact: true }) }).getByRole('link').first().click();
    await expect(page.getByRole('button', { name: '1900-01-01', exact: true })).toBeVisible();
  }
}
async function read(page: Page) {
  const id = new URL(page.url()).pathname.split('/').at(-1);
  return (await request(page, 'GET', '/api/v1/country-regions/' + id)).body;
}

test('国家200 正常菜单完整目录、分页和无创建或选择导出', async ({ page }) => {
  const forbidden: string[] = [];
  page.on('request', r => { if (/country-regions\/(selection|references)/.test(r.url())) forbidden.push(r.url()); });
  await open(page); await menu(page);
  await expect(page.getByRole('link', { name: '新建', exact: true })).toHaveCount(0);
  await expect(page.getByRole('checkbox')).toHaveCount(0);
  await expect(page.getByRole('button', { name: '导出所选', exact: true })).toHaveCount(0);
  const first = await request(page, 'GET', '/api/v1/country-regions?pageSize=100');
  expect(first.body.count).toBe(251);
  await page.getByRole('button', { name: /下一页/ }).click();
  await expect(page).toHaveURL(/page=2/);
  await menu(page, 'GB');
  await expect(page.getByText('来源资料', { exact: true })).toHaveCount(0);
  await expect(page.getByText('base.uk', { exact: true })).toHaveCount(0);
  const detail = await read(page);
  expect(detail.code).toBe('GB');
  expect(detail).not.toHaveProperty('origin');
  expect(detail.capabilities.readFields).not.toContain('source');
  expect(forbidden).toEqual([]);
});

test('国家200 共用更正、未保存保护、日志和准确原来源', async ({ page }) => {
  await open(page); await menu(page, 'AD');
  const before = await read(page);
  await openMoreActions(page);
  await expect(page.getByRole('menuitem', { name: /删除|插入停用/ })).toHaveCount(0);
  await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
  const name = page.getByRole('textbox', { name: '显示名称', exact: true });
  await expect(name).toHaveValue(before.selectedRecord.name);
  await expect(page.locator('#record-date')).toHaveAttribute('readonly', '');
  await name.fill('安道尔本地显示');
  await page.getByRole('textbox', { name: '电话区号', exact: true }).fill('');
  await page.getByRole('button', { name: '取消编辑', exact: true }).click();
  await page.getByRole('button', { name: '继续编辑', exact: true }).click();
  await expect(name).toHaveValue('安道尔本地显示');
  const saving = page.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith('/correct'));
  await page.getByRole('button', { name: '保存', exact: true }).click();
  expect((await saving).status()).toBe(200);
  await expect(page.locator('.object-form')).toHaveCount(0);
  const after = await read(page);
  expect(after.selectedRecord.name).toBe('安道尔本地显示');
  expect(after.selectedRecord.phoneCode).toBeNull();
  expect(after).not.toHaveProperty('origin');
  const original = await request(page, 'GET', `/api/v1/country-regions/${after.id}?sourceVersion=${before.headSourceVersion}`);
  expect(original.body.selectedRecord.name).toBe(before.selectedRecord.name);
  await page.getByRole('tab', { name: /变更日志/ }).click();
  const history = page.locator('#country-history-panel');
  await expect(history.getByText('安道尔本地显示', { exact: true }).first()).toBeVisible();
  const source = history.locator('a[href*="sourceVersion="]').first();
  await expect(source).toBeVisible(); await source.click();
  await expect(page.getByText("正在查看只读历史资料", { exact: true }).first()).toBeVisible();
});

test('国家200 追加与停用恢复复用时间线并保留初始日期', async ({ page }) => {
  await open(page); await menu(page, 'AE');
  await page.getByRole('button', { name: '新增记录', exact: true }).click();
  await expect(page.locator('#record-date')).not.toHaveValue('1900-01-01');
  await page.locator('#record-date').fill('1901-01-01');
  await page.getByRole('textbox', { name: '显示名称', exact: true }).fill('阿联酋本地显示');
  await expect(page.getByRole('combobox', { name: '操作原因', exact: true })).toHaveText('显示资料更新');
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.locator('.object-form')).toHaveCount(0);
  await select(page, '1901-01-01');
  await openMoreActions(page); await page.getByRole('menuitem', { name: '停用国家/地区', exact: true }).click();
  await page.locator('#record-date').fill('1902-01-01');
  await expect(page.getByRole('combobox', { name: '操作原因', exact: true })).toHaveText('停止选用');
  await page.getByRole('button', { name: '确认停用', exact: true }).click();
  await expect(page.locator('.object-form')).toHaveCount(0);
  expect((await read(page)).status).toBe('inactive');
  await select(page, '1902-01-01');
  await openMoreActions(page); await page.getByRole('menuitem', { name: '重新启用国家/地区', exact: true }).click();
  await page.locator('#record-date').fill('1903-01-01');
  await expect(page.getByRole('combobox', { name: '操作原因', exact: true })).toHaveText('恢复选用');
  await page.getByRole('button', { name: '确认重新启用', exact: true }).click();
  await expect(page.locator('.object-form')).toHaveCount(0);
  const restored = await read(page);
  expect(restored.status).toBe('active');
  expect(restored.timeline.map((r: { effectiveDate: string }) => r.effectiveDate)).toEqual(['1900-01-01','1901-01-01','1902-01-01','1903-01-01']);
});

test('国家200 查询者可读当前和历史、无来源资料或维护按钮', async ({ page }) => {
  await open(page, 'reader'); await menu(page, 'CN');
  await expect(page.getByRole('button', { name: '新增记录', exact: true })).toHaveCount(0);
  await expect(page.getByRole('button', { name: '更多操作', exact: true })).toHaveCount(0);
  await expect(page.getByText('来源资料', { exact: true })).toHaveCount(0);
  await expect(page.getByText('base.cn', { exact: true })).toHaveCount(0);
  const detail = await read(page);
  expect(detail.capabilities.history).toBe(true);
  expect(detail.code).toBe('CN');
  expect(detail).not.toHaveProperty('origin');
  expect(detail.capabilities.readFields).not.toContain('source');
  expect(new URL(page.url()).pathname.startsWith(root)).toBe(true);
});

test('国家200 已保存但丢响应、刷新后按原提交查回', async ({ page }) => {
  await open(page); await menu(page, 'AF');
  await openMoreActions(page); await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
  const recoveredName = `原提交恢复名称 ${randomUUID().slice(0, 8)}`;
  await page.getByRole('textbox', { name: '显示名称', exact: true }).fill(recoveredName);
  let accepted: { sourceVersion: string; submissionId: string } | undefined;
  await page.route('**/api/v1/country-regions/*/records/*/correct', async route => {
    const response = await route.fetch(); expect(response.status()).toBe(200);
    accepted = await response.json(); await route.abort('connectionreset');
  });
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.getByRole('button', { name: '查看保存结果', exact: true })).toBeVisible();
  await page.reload();
  await page.getByRole('region', { name: '国家/地区提交结果', exact: true }).getByRole('button', { name: '查看保存结果', exact: true }).click();
  await page.getByRole('link', { name: '查看保存结果', exact: true }).click();
  await expect(page.getByText('正在查看只读历史资料', { exact: true })).toBeVisible();
  expect(new URL(page.url()).searchParams.get('sourceVersion')).toBe(accepted?.sourceVersion);
  expect((await read(page)).selectedRecord.name).toBe(recoveredName);
  const recovered = await request(page, 'POST', `/api/v1/country-regions/submissions/${accepted?.submissionId}/resolve`);
  expect(recovered.body.sourceVersion).toBe(accepted?.sourceVersion);
});
