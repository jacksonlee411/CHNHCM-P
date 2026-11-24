import { lazyCredentials } from './credentials.mjs';
import { expectLogChange } from './fixtures';
import { openMoreActions, applyListOption,recordRange, readCurrentDate,  test, expect, type Page } from './fixtures';
import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { randomUUID } from 'node:crypto';
import path from 'node:path';
import { localConnection } from '../../scripts/local-runtime.mjs';

const root = path.resolve(import.meta.dirname, '../..');
const origin = localConnection().webOrigin;
const credentials = lazyCredentials();
const buPath = '/settings/organization/business-units';
async function enter(page: Page) {
  await page.goto('/');
  await page.getByLabel('登录名').fill(credentials.login);
  await page.getByLabel('密码').fill(credentials.password);
  // Pace the shared real account, including the transition from the original suite.
  await page.waitForTimeout(3100);
  await page.getByRole('button', { name: '登录', exact: true }).click();
  await page.getByRole('button', { name: '组织配置', exact: true }).click();
  await page.getByRole('menuitem', { name: '业务单元', exact: true }).click();
  expect(await readCurrentDate(page)).toMatch(/^\d{4}-\d{2}-\d{2}$/);
}
async function seed(page: Page, count: number) {
  const prefix = 'UI100_' + randomUUID().slice(0, 8);
  const date = await readCurrentDate(page);
  const session = await (await page.request.get('/api/v1/session')).json();
  for (let i = 0; i < count; i++) {
    const code = prefix + '_' + String(i).padStart(2, '0');
    const response = await page.request.post('/api/v1/business-units', {
      headers: { Origin: origin, 'X-CSRF-Token': session.csrf, 'Idempotency-Key': randomUUID() },
      data: { code, name: code + ' 青禾', nameEn: 'Qinghe ' + i, effectiveDate: date, operationType: 'CREATE', operationReason: 'NEW_ORGANIZATION' },
    });
    expect(response.status()).toBe(201);
  }
  await search(page, prefix);
  await expect(page.getByText(recordRange(page, count, 1), { exact: true })).toBeVisible();
  return { prefix, date };
}
async function search(page: Page, q: string) {
  await page.getByLabel('搜索编码或名称').fill(q);
  const response = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/business-units' && (new URL(r.url()).searchParams.get('q') ?? '') === q);
  await page.getByRole('button', { name: '查询', exact: true }).click();
  await response;
}

test('查询草稿、分页焦点、筛选整组应用、前后退刷新及不完整日期', async ({ page }) => {
  await enter(page);
  const { prefix, date } = await seed(page, 21);
  const input = page.getByLabel('搜索编码或名称');
  let releasePage!: () => void;
  const heldPage = new Promise<void>(resolve => { releasePage = resolve; });
  await page.route('**/api/v1/business-units?*', async route => {
    if (new URL(route.request().url()).searchParams.get('page') === '2') await heldPage;
    await route.continue();
  });
  await input.fill(prefix + '_00');
  await page.getByRole('button', { name: '下一页', exact: true }).focus();
  await page.keyboard.press('Enter');
  await expect(page.getByText('正在查询…', { exact: true })).toBeVisible();
  await expect(page.locator('.list-pager').getByRole('status')).toBeFocused();
  await expect(page.getByRole('button', { name: '下一页', exact: true })).toBeDisabled();
  releasePage();
  await expect(page.getByText(recordRange(page, 21, 2), { exact: true })).toBeVisible();
  await page.unrouteAll({ behavior: 'wait' });
  await expect(page.locator('.list-pager').getByRole('status')).toBeFocused();
  await expect(input).toHaveValue(prefix + '_00');
  await expect(page).toHaveURL(url => url.searchParams.get('q') === prefix);
  await page.getByRole('combobox', { name: '每页条数' }).click();
  await page.getByRole('option', { name: '50' }).click();
  await expect(page.getByText(recordRange(page, 21, 1), { exact: true })).toBeVisible();
  await expect(input).toHaveValue(prefix + '_00');
  await expect(page.getByRole('combobox', { name: '每页条数' })).toBeFocused();
  await applyListOption(page, '状态', '启用');
  await expect(page.getByText(recordRange(page, 1, 1), { exact: true })).toBeVisible();
  await expect(page).toHaveURL(url => url.searchParams.get('q') === prefix + '_00');
  await input.fill('');
  await page.getByRole('combobox', { name: '每页条数' }).click();
  await page.getByRole('option', { name: '20' }).click();
  await expect(input).toHaveValue('');
  await expect(page).toHaveURL(url => url.searchParams.get('q') === prefix + '_00');
  await expect(page.getByText(/输入已修改，点击查询后生效/)).toBeVisible();
  await applyListOption(page, '排序', '名称降序');
  await expect(page).toHaveURL(url => !url.searchParams.has('q'));
  await search(page, prefix);
  await input.fill('离开即丢弃');
  await page.goBack();
  await expect(input).toHaveValue('');
  await page.goForward();
  await expect(input).toHaveValue(prefix);
  await input.fill('刷新前草稿');
  await page.reload();
  await expect(input).toHaveValue(prefix);
  await expect(page.getByText(recordRange(page, 21, 1), { exact: true })).toBeVisible();
  let requests = 0;
  page.on('request', r => { if (new URL(r.url()).pathname === '/api/v1/business-units') requests++; });
  await openMoreActions(page);
  await expect(page.getByRole('menuitem', {name:'查看日期…', exact:true})).toHaveCount(0);
  await page.getByRole('menuitem', {name:'状态筛选…', exact:true}).click();
  await page.keyboard.press('Escape'); expect(requests).toBe(0);
  await input.fill(prefix + '_01');
  await applyListOption(page, '状态', '全部');
  await expect(page.getByText(recordRange(page, 1, 1), { exact: true })).toBeVisible();
  await expect(page).toHaveURL(url => url.searchParams.get('q') === prefix + '_01');
  await search(page, '');
  await expect(page).toHaveURL(url => !url.searchParams.has('q'));
  await expect(input).toHaveValue('');
  await expect(page.getByRole('button', { name: '查询', exact: true })).toBeFocused();
});

test('慢响应和失败重试保留新输入，旧响应不覆盖新查询，权限拒绝清理草稿', async ({ page }) => {
  await enter(page);
  const { prefix } = await seed(page, 2);
  const input = page.getByLabel('搜索编码或名称');
  let release!: () => void;
  const held = new Promise<void>(resolve => { release = resolve; });
  let arrived = false;
  await page.route('**/api/v1/business-units?*', async route => {
    if (new URL(route.request().url()).searchParams.get('q') !== prefix + '_00') return route.continue();
    const response = await route.fetch(); arrived = true;
    await held; await route.fulfill({ response });
  });
  await input.fill(prefix + '_00');
  await page.getByRole('button', { name: '查询', exact: true }).click();
  await expect.poll(() => arrived).toBe(true);
  await input.fill('请求期间新输入');
  release();
  await expect(page.getByText(recordRange(page, 1, 1), { exact: true })).toBeVisible();
  await expect(input).toHaveValue('请求期间新输入');
  await expect(input).toBeFocused();
  await page.unrouteAll({ behavior: 'wait' });
  let fail = true, releaseFailure!: () => void;
  const heldFailure = new Promise<void>(resolve => { releaseFailure = resolve; });
  await page.route('**/api/v1/business-units?*', async route => {
    if (fail) { await heldFailure; await route.fulfill({ status: 503, json: { code: 'M02_DEPENDENCY_UNAVAILABLE', message: '查询暂时不可用' } }); }
    else await route.continue();
  });
  await input.fill(prefix); await page.getByRole('button', { name: '查询', exact: true }).click();
  await expect(page.getByText('正在查询…', { exact: true })).toBeVisible();
  await input.fill('失败后新草稿'); releaseFailure();
  await expect(page.getByText('查询暂时不可用')).toBeVisible();
  await expect(input).toHaveValue('失败后新草稿'); fail = false;
  await page.getByRole('button', { name: '重试', exact: true }).click();
  await expect(page.getByText(recordRange(page, 2, 1), { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: '查询', exact: true })).toBeFocused();
  await expect(input).toHaveValue('失败后新草稿');
  await expect(page).toHaveURL(url => url.searchParams.get('q') === prefix);
  await page.unrouteAll({ behavior: 'wait' });
  let releaseOld!: () => void;
  const old = new Promise<void>(resolve => { releaseOld = resolve; }); arrived = false;
  await page.route('**/api/v1/business-units?*', async route => {
    if (new URL(route.request().url()).searchParams.get('q') !== prefix + '_00') return route.continue();
    const response = await route.fetch(); arrived = true; await old; await route.fulfill({ response });
  });
  await input.fill(prefix + '_00'); await page.getByRole('button', { name: '查询', exact: true }).click();
  await expect.poll(() => arrived).toBe(true);
  await search(page, prefix + '_01');
  await input.fill('保留后写草稿'); releaseOld();
  await page.unrouteAll({ behavior: 'wait' });
  await expect(page.getByRole('link', { name: prefix + '_01 青禾', exact: true })).toBeVisible();
  await expect(page.getByRole('link', { name: prefix + '_00 青禾', exact: true })).toHaveCount(0);
  await expect(input).toHaveValue('保留后写草稿');
  await page.route('**/api/v1/business-units?*', route => route.fulfill({ status: 403, json: { code: 'ACCESS_DENIED', message: '当前无读取权限' } }));
  await search(page, prefix);
  await expect(page.getByText('当前无读取权限')).toBeVisible();
  await expect(input).toHaveValue('');
  await expect(page.locator('a[data-object-link]')).toHaveCount(0);
  await expect(page.getByText(/^\d+–\d+ \/ \d+$/)).toHaveCount(0);
});

test('真实名称链接保留原标签草稿，新标签及两种返回恢复焦点，目标缺失回到列表标题', async ({ page, context }) => {
  await enter(page);
  const { prefix } = await seed(page, 2);
  const input = page.getByLabel('搜索编码或名称');
  const object = page.getByRole('link', { name: prefix + '_00 青禾', exact: true });
  await input.fill('原标签尚未应用');
  const href = await object.getAttribute('href');
  expect(href).toBeTruthy(); expect(href).not.toContain('尚未应用');
  const popupPromise = context.waitForEvent('page');
  await object.click({ modifiers: ['Control'] });
  const popup = await popupPromise;
  await expect(popup.getByRole('heading', { level: 1 })).toContainText(prefix + '_00');
  await expect(input).toHaveValue('原标签尚未应用');
  await popup.getByRole('link', { name: '返回列表', exact: true }).click();
  await expect(popup.getByLabel('搜索编码或名称')).toHaveValue(prefix);
  await expect(popup.getByRole('link', { name: prefix + '_00 青禾', exact: true })).toBeFocused();
  await popup.close(); await page.bringToFront();
  for (const back of ['link', 'browser']) {
    await object.focus(); await page.keyboard.press('Enter');
    await expect(page).toHaveTitle(new RegExp(prefix + '_00'));
    if (back === 'link') await page.getByRole('link', { name: '返回列表', exact: true }).click();
    else await page.goBack();
    await expect(object).toBeFocused(); await expect(object).toBeInViewport();
    await expect(input).toHaveValue(prefix);
  }
  await object.click();
  await expect(page.getByRole('heading', { level: 1 })).toContainText(prefix + '_00');
  // Keep one interceptor while detail navigation has in-flight list requests.
  // Removing/reinstalling Chromium interception can leave the next request outside the fixture.
  let denyReturn = false;
  await page.route('**/api/v1/business-units?*', async route => {
    if (denyReturn) {
      await route.fulfill({ status: 403, json: { code: 'ACCESS_DENIED', message: '返回时列表权限已收回' } });
      return;
    }
    const response = await route.fetch(), body = await response.json();
    await route.fulfill({ response, json: { ...body, count: 0, items: [] } });
  });
  await page.getByRole('link', { name: '返回列表', exact: true }).click();
  await expect(page.locator('[data-list-heading]')).toBeFocused();
  await expect(page.getByText('原选中对象不在当前页或筛选结果中，查询条件已保留。')).toHaveCount(0);
  await expect(page.getByRole('link', { name: '查看原对象' })).toHaveCount(0);
  await expect(page.getByText('没有符合条件的业务单元', { exact: true })).toBeVisible();
  await expect(input).toHaveValue(prefix);
  await page.goto(new URL(href!, page.url()).href);
  await expect(page.getByRole('heading', { level: 1 })).toContainText(prefix + '_00');
  denyReturn = true;
  const deniedReturn = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/business-units' && r.status() === 403);
  await page.getByRole('link', { name: '返回列表', exact: true }).click();
  await deniedReturn;
  await expect(page.getByText('返回时列表权限已收回')).toBeVisible();
  await expect(page.locator('[data-list-return-fallback]')).toBeFocused();
  await expect(page.locator('a[data-object-link]')).toHaveCount(0);
  await expect(page.getByLabel('搜索编码或名称')).toHaveValue('');
});

test('表单帮助、未保存提示、日志已有字段和时刻表达、历史短条及受限标题', async ({ page }) => {
  await enter(page);
  const { prefix } = await seed(page, 1);
  await page.route('**/api/v1/organization/operation-reasons?*', route => route.fulfill({ status: 503, json: { code: 'M02_DEPENDENCY_UNAVAILABLE', message: '原因目录暂不可用' } }));
  await page.getByRole('link', { name: '新建', exact: true }).click();
  await expect(page.getByText('原因目录暂不可用', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: '保存', exact: true })).toBeDisabled();
  await expect(page.getByRole('button', { name: '保存', exact: true })).toHaveAccessibleDescription(/原因目录暂不可用/);
  await expect(page.getByText('原因目录暂不可用', { exact: true })).toHaveCount(1);
  await expect(page.getByText(/办理资料读取失败|正在读取办理资料/)).toHaveCount(0);
  await page.unroute('**/api/v1/organization/operation-reasons?*');
  await page.getByRole('button', { name: '重试加载', exact: true }).click();
  await expect(page).toHaveTitle('新建业务单元 · CHNHCM');
  await expect(page.getByRole('heading', { level: 1 })).toHaveCount(1);
  await expect(page.getByRole('tab')).toHaveCount(0);
  await expect(page.getByText(/1 至 40 位字母/)).toBeVisible();
  await expect(page.getByRole('combobox', { name: '操作原因', exact: true })).toContainText('新设组织');
  await expect(page.getByRole('button', { name: '保存', exact: true })).toHaveAccessibleDescription('');
  await page.getByLabel('中文名称').fill('未保存输入');
  await page.getByRole('link', { name: '取消', exact: true }).click();
  await expect(page.getByRole('dialog')).not.toContainText('待确认提交');
  await page.getByRole('button', { name: '放弃修改' }).click();
  await page.getByRole('link', { name: prefix + '_00 青禾', exact: true }).click();
  await page.getByRole('button', { name: '更多操作', exact: true }).click();
  await expect(page.getByRole('menuitem',{name:'查看日期…',exact:true})).toHaveCount(0);
  await page.keyboard.press('Escape');
  await page.getByRole('tab', { name: '变更日志', exact: true }).click();
  await expectLogChange(page, '英文名称', '未登记', 'Qinghe 0');
  await expect(page.locator('.change-operation').first().getByRole('row').filter({has: page.getByRole('cell', {name: '生效日期', exact: true})}).locator('[data-label="新值"]')).toHaveText(/\d{4}-\d{2}-\d{2}/);
  await expect(page.locator('.change-operation').getByText(/北京时间 UTC\+08:00/)).toBeVisible();
  await page.getByRole('link', { name: '查看当次资料', exact: true }).click();
  await expect(page.getByText('正在查看只读历史资料', { exact: true })).toBeVisible();
  await page.setViewportSize({ width: 320, height: 256 });
  await expect.poll(() => page.evaluate(() => ({ width: innerWidth, height: innerHeight, scrollable: document.documentElement.scrollHeight > innerHeight }))).toEqual({ width: 320, height: 256, scrollable: true });
  await page.mouse.move(160, 248); await page.mouse.wheel(0, 1600);
  await expect.poll(() => page.evaluate(() => scrollY)).toBeGreaterThan(0);
  await expect(page.getByText('正在查看只读历史资料', { exact: true })).toBeInViewport();
  await expect(page.getByRole('button', { name: '返回当前资料', exact: true })).toBeInViewport();
  await page.getByRole('button', { name: '返回当前资料', exact: true }).click();
  await expect(page).not.toHaveURL(/sourceVersion=/);
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.route('**/api/v1/business-units/*/changes?*', route => route.fulfill({ status: 403, json: { code: 'ACCESS_DENIED', message: '来源权限已收回' } }));
  await page.getByRole('tab', { name: '变更日志', exact: true }).click();
  await expect(page.getByText('来源权限已收回')).toBeVisible();
  await expect(page).toHaveTitle('业务单元详情 · CHNHCM');
  await expect(page.getByRole('heading', { name: new RegExp(prefix) })).toHaveCount(0);
});


test('首次列表慢读取完成时不抢走正在输入的焦点和草稿', async ({ page }) => {
  let reading = false;
  let release!: () => void;
  const held = new Promise<void>(resolve => { release = resolve; });
  await page.route('**/api/v1/business-units?*', async route => {
    reading = true;
    await held;
    await route.continue();
  });
  try {
    await enter(page);
    await expect.poll(() => reading).toBe(true);
    const input = page.getByLabel('搜索编码或名称');
    await input.fill('正在编辑的首个搜索词');
    const response = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/business-units' && r.ok());
    release();
    await response;
    await expect(page.locator('a[data-object-link]').first()).toBeVisible();
    await expect(input).toHaveValue('正在编辑的首个搜索词');
    await expect(input).toBeFocused();
  } finally {
    release();
    await page.unrouteAll({ behavior: 'wait' });
  }
});
