import {checkPairPeriod} from './dated-object-support';
import { randomUUID } from 'node:crypto';
import { test, expect, openMoreActions, deferAutomaticConfirmation, type Page } from './fixtures';
import { open, choose, select } from './bu-support';
import { request, apiBU, apiDepartment, cancelLeapDayPreview } from './dated-object-support';
import { chooseApplication } from './app-navigation';
const base = '/settings/organization/enterprise';
async function menu(page: Page) {
  await page.getByRole('button', { name: '组织配置', exact: true }).click();
  await page.getByRole('menuitem', { name: '企业', exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`${base}/[0-9a-f-]+$`));
  await expect(page.getByRole('button', { name: '更多操作', exact: true })).toBeEnabled();
  await expect(page).not.toHaveTitle(/页面不存在/);
}
async function read(page: Page) { return (await request(page, 'GET', '/api/v1/enterprise/' + new URL(page.url()).pathname.split('/').at(-1))).body; }
async function submit(page: Page, suffix: string, button = '保存', status = 200) {
  const wait = page.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith(suffix));
  await page.getByRole('button', { name: button, exact: true }).click();
  const response = await wait; expect(response.status()).toBe(status);
  await expect(page.locator('.object-form')).toHaveCount(0);
  return response.json();
}
async function correct(page: Page, name: string, date?: string) {
  await openMoreActions(page); await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
  await page.getByRole('textbox', { name: '中文名称', exact: true }).fill(name);
  if (date) await page.getByLabel('生效日期').and(page.locator('input')).fill(date);
  return submit(page, '/correct');
}
async function reset(page: Page) {
  const context = await request(page, 'GET', '/api/v1/enterprise/context');
  const path = '/api/v1/enterprise/' + context.body.id;
  let d = (await request(page, 'GET', path)).body;
  for (const row of [...d.timeline].reverse().slice(0, -1)) {
    d = (await request(page, 'GET', path)).body;
    const actual = d.timeline.find((r: any) => r.recordId === row.recordId);
    const result = await request(page, 'POST', path + '/records/' + row.recordId + '/delete', { expectedSourceVersion: d.headSourceVersion, expectedRecordRevision: actual.revision }, randomUUID());
    expect(result.status).toBe(200);
  }
  d = (await request(page, 'GET', path)).body;
  const result = await request(page, 'POST', path + '/records/' + d.timeline[0].recordId + '/correct', { expectedSourceVersion: d.headSourceVersion, expectedRecordRevision: d.timeline[0].revision, newEffectiveDate: '1900-01-01', changes: { name: '本地验证企业', nameEn: null } }, randomUUID());
  expect(result.status).toBe(200);
  await page.goto(base); await expect(page).toHaveURL(new RegExp(`${base}/[0-9a-f-]+$`));
}

test('140 企业首条后移预览跨闰日，取消后保留来源', async ({page}) => {
  test.setTimeout(90000);
  await open(page);
  await menu(page);
  await reset(page);
  try {
    await correct(page, '闰日样本', '2024-02-28');
    const before = await read(page);
    await openMoreActions(page);
    await cancelLeapDayPreview(page);
    expect((await read(page)).headSourceVersion).toBe(before.headSourceVersion);
  } finally { await reset(page); }
});

test('304 正常菜单完整维护：首条、插入、更正、删除、固定期间、启停及历史', async ({ page }) => {
  test.setTimeout(150000); await open(page); await menu(page); await reset(page);
  await expect(page.getByRole('link', { name: '返回列表', exact: true })).toHaveCount(0);
  await expect(page.getByRole('textbox', { name: '管理业务单元', exact: true })).toHaveCount(0);
  const original = await read(page); const id = original.id;
  await correct(page, '企业首条修订', '1800-01-01');
  await page.getByRole('button', { name: '新增记录', exact: true }).click();
  await page.getByLabel('生效日期').and(page.locator('input')).fill('1900-01-01');
  await choose(page, '操作原因', '名称调整'); await page.getByRole('textbox', { name: '中文名称', exact: true }).fill('企业后继');
  await submit(page, '/records', '保存', 201);
  await select(page, '1800-01-01'); await page.getByRole('button', { name: '插入记录', exact: true }).click();
  await page.getByLabel('生效日期').and(page.locator('input')).fill('1850-01-01');
  await choose(page, '操作原因', '名称调整'); await page.getByRole('textbox', { name: '中文名称', exact: true }).fill('企业中间记录');
  await submit(page, '/records', '保存', 201); await select(page, '1850-01-01');
  await correct(page, '企业中间更正', '1860-01-01');
  await select(page, '1860-01-01'); await openMoreActions(page); await page.getByRole('menuitem', { name: '删除此记录', exact: true }).click();
  await submit(page, '/delete', '确认删除');
  await select(page, '1800-01-01'); await openMoreActions(page); await page.getByRole('menuitem', { name: '插入停用与重新启用', exact: true }).click();
  let dialog = page.getByRole('dialog', { name: '插入停用与重新启用', exact: true });
  await dialog.getByLabel('停用日期', { exact: true }).fill('1810-01-01'); await dialog.getByLabel('重新启用日期', { exact: true }).fill('1820-01-01');
  await choose(dialog, '停用原因', '业务终止'); await submit(page, '/lifecycle-pairs', '保存', 200); await expect(dialog).toHaveCount(0);
  await checkPairPeriod(page,'enterprise','enterprise','1820-01-01','1899-12-31');
  await select(page, '1810-01-01'); await openMoreActions(page); await page.getByRole('menuitem', { name: '删除停用与重新启用', exact: true }).click();
  dialog = page.getByRole('dialog', { name: '删除停用与重新启用', exact: true }); await submit(page, '/lifecycle-pairs/delete', '删除两条记录'); await expect(dialog).toHaveCount(0);
  await select(page, '1900-01-01'); await openMoreActions(page); await page.getByRole('menuitem', { name: '停用企业', exact: true }).click();
  await page.getByLabel('生效日期').and(page.locator('input')).fill('2000-01-01'); await choose(page, '操作原因', '业务终止'); await submit(page, '/deactivate', '确认停用', 201);
  await select(page, '2000-01-01'); await openMoreActions(page); await page.getByRole('menuitem', { name: '重新启用企业', exact: true }).click();
  await page.getByLabel('生效日期').and(page.locator('input')).fill('2001-01-01'); await page.getByRole('textbox', { name: '中文名称', exact: true }).fill('企业恢复'); await submit(page, '/reactivate', '确认重新启用', 201);
  const head = await read(page); expect(head.id).toBe(id); expect(head.timeline[0].recordId).toBe(original.timeline[0].recordId);
  await page.getByRole('tab', { name: '变更日志', exact: true }).click(); await expect(page.locator('.change-operation').first()).toContainText('重新启用');
  await page.getByRole('link', { name: '查看当次资料', exact: true }).first().click(); await expect(page.getByText('正在查看只读历史资料', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: '更多操作', exact: true })).toHaveCount(0); await page.reload(); await expect(page.getByText('正在查看只读历史资料', { exact: true })).toBeVisible();
  await reset(page);
});

test('304 草稿取消、冲突重读和丢失响应查回', async ({ page }) => {
  test.setTimeout(120000); await deferAutomaticConfirmation(page); await open(page); await menu(page); await reset(page);
  const before = await read(page); await openMoreActions(page); await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
  await page.getByRole('textbox', { name: '中文名称', exact: true }).fill('未保存草稿');
  await page.getByRole('button', { name: '取消编辑', exact: true }).click(); await page.getByRole('button', { name: '继续编辑', exact: true }).click();
  await expect(page.getByRole('textbox', { name: '中文名称', exact: true })).toHaveValue('未保存草稿');
  await page.getByRole('button', { name: '取消编辑', exact: true }).click(); await page.getByRole('button', { name: '放弃修改', exact: true }).click();
  expect((await read(page)).headSourceVersion).toBe(before.headSourceVersion);
  await openMoreActions(page); await page.getByRole('menuitem', { name: '更正记录', exact: true }).click(); await page.getByRole('textbox', { name: '中文名称', exact: true }).fill('保留输入');
  const path = '/api/v1/enterprise/' + before.id + '/records/' + before.timeline[0].recordId + '/correct';
  expect((await request(page, 'POST', path, { expectedSourceVersion: before.headSourceVersion, expectedRecordRevision: before.timeline[0].revision, changes: { name: '并发名称' } }, randomUUID())).status).toBe(200);
  const conflict = page.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith('/correct')); await page.getByRole('button', { name: '保存', exact: true }).click(); expect((await conflict).status()).toBe(409);
  await page.getByRole('button', { name: '重新核对当前资料', exact: true }).click(); await expect(page.getByRole('textbox', { name: '中文名称', exact: true })).toHaveValue('保留输入');
  let committed: any;
  await page.route('**/api/v1/enterprise/*/records/*/correct', async route => { const response = await route.fetch(); expect(response.status()).toBe(200); committed = await response.json(); await route.abort('failed'); });
  await page.getByRole('button', { name: '保存', exact: true }).click(); await expect(page.getByRole('button', { name: '查看保存结果', exact: true })).toBeVisible(); await page.getByRole('button', { name: '查看保存结果', exact: true }).click();
  await expect(page.locator('.object-form')).toHaveCount(0); expect((await read(page)).headSourceVersion).toBe(committed.sourceVersion);
  await page.unroute('**/api/v1/enterprise/*/records/*/correct'); await reset(page);
});

test('304 查询者和窄屏：唯一企业、记录与历史只读', async ({ page }) => {
  test.setTimeout(60000); await open(page, 'reader');
  await page.getByRole('button', { name: '组织配置', exact: true }).click(); await page.getByRole('menuitem', { name: '企业', exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`${base}/[0-9a-f-]+$`)); await expect(page.getByRole('tab', { name: '变更日志', exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: '更多操作', exact: true })).toHaveCount(0); await expect(page.getByRole('button', { name: '新增记录', exact: true })).toHaveCount(0);
  for (const width of [1440, 720, 390, 320]) {
    await page.setViewportSize({ width, height: 900 }); await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
    await page.getByRole('tab', { name: '变更日志', exact: true }).click(); await expect(page.locator('.change-operation').first()).toBeVisible(); await page.getByRole('tab', { name: '基本资料', exact: true }).click();
  }
});


test('304 企业更名保持部门当前及历史路径独立', async ({ page }) => {
  test.setTimeout(90000);
  await open(page); await menu(page); await reset(page);
  const bu = await apiBU(page, '企业关联BU304');
  const dep = await apiDepartment(page, bu.id, {name:'企业关联部门304'});
  await correct(page, '企业关联更名304');
  try {
    await chooseApplication(page, '部门');
    await page.getByRole('textbox', {name:'搜索编码或名称',exact:true}).fill(dep.code);
    await page.getByRole('button', {name:'查询',exact:true}).click();
    await expect(page.getByRole('row').filter({hasText:dep.code})).not.toContainText('企业关联更名304');
    await page.getByRole('link', {name:'企业关联部门304',exact:true}).click();
    await expect(page.locator('.department-path')).toHaveText('企业关联部门304');
    await page.getByRole('tab', {name:'变更日志',exact:true}).click();
    await page.getByRole('link', {name:'查看当次资料',exact:true}).first().click();
    await expect(page.locator('.department-path')).toHaveText('企业关联部门304');
    await expect(page.locator('.department-path')).not.toContainText('企业关联更名304');
  } finally {
    await chooseApplication(page, '设置'); await menu(page); await reset(page);
  }
});
