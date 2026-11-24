import { test, expect } from './fixtures';
import { login, seed, query, panel, option, isList, policy, out, listPath } from './community-list-support.mjs';
import { appendFileSync } from 'node:fs';

let fixture: Awaited<ReturnType<typeof seed>>;
const pageErrors = new WeakMap<object, string[]>();
test.beforeAll(async ({ browser }) => {
  const context = await browser.newContext();
  try { const page = await context.newPage(); await login(page); fixture = await seed(page); }
  finally { await context.close(); }
});
test.beforeEach(async ({ page }) => {
  const errors: string[] = []; pageErrors.set(page, errors); page.on('pageerror', e => errors.push(e.message));
  await login(page); await query(page, fixture.prefix);
});
test.afterEach(async ({ page }, info) => {
  const errors = pageErrors.get(page) ?? [];
  appendFileSync(out + '/column-filter-results.jsonl', JSON.stringify({ title: info.title, result: info.status, durationMs: info.duration, errors }) + '\n');
  expect(errors).toEqual([]);
});
async function edit(page: any, field: string, operator: string, value: string) {
  await panel(page, '列筛选…'); await option(page, '列', field); await option(page, '条件', operator);
  await page.getByRole('dialog').getByLabel('值', { exact: true }).fill(value);
}
async function apply(page: any) {
  const pending = page.waitForResponse(isList); await page.getByRole('dialog').getByRole('button', { name: '应用', exact: true }).click();
  const response = await pending; expect(response.status()).toBe(200); const result = await response.json();
  await expect(page.getByText(`${result.count ? (result.page - 1) * result.pageSize + 1 : 0}–${Math.min(result.page * result.pageSize, result.count)} / ${result.count}`, { exact: true })).toBeVisible();
  return result;
}
test('TL335 桌面列菜单、独立编码/名称及同步应用', async ({ page }) => {
  const header = page.getByRole('columnheader', { name: /编码/ }); await header.hover(); await header.getByLabel('列操作').click();
  await page.getByRole('menuitem', { name: '筛选…', exact: true }).click();
  await expect(page.getByRole('dialog').getByRole('combobox', { name: '列', exact: true })).toHaveText('编码');
  await option(page, '条件', '等于'); await page.getByRole('dialog').getByLabel('值', { exact: true }).fill(fixture.items[72].code);
  let actual = await apply(page); expect(actual.count).toBe(1); expect(actual.items[0].id).toBe(fixture.items[72].id);
  await expect(page.getByRole('button', { name: '编辑列筛选', exact: true })).toHaveCount(1);
  await edit(page, '名称', '等于', fixture.items[72].code); expect((await apply(page)).count).toBe(0);
  await edit(page, '名称', '等于', fixture.items[72].name); actual = await apply(page); expect(actual.count).toBe(1); expect(actual.items[0].id).toBe(fixture.items[72].id);
});
test('TL336 取消/Escape/外部关闭不推进A，清除须应用', async ({ page }) => {
  await edit(page, '编码', '等于', fixture.items[0].code); await apply(page);
  let requests = 0; page.on('request', r => { if (isListRequest(r)) requests++; });
  const url = page.url(); await page.getByLabel('搜索编码或名称').fill('独立搜索草稿');
  for (const close of ['cancel', 'escape', 'outside']) {
    await panel(page, '列筛选…'); await page.getByRole('dialog').getByLabel('值', { exact: true }).fill('未应用列草稿');
    if (close === 'cancel') await page.getByRole('button', { name: '取消', exact: true }).click();
    else if (close === 'escape') await page.keyboard.press('Escape'); else await page.locator('.MuiDialog-container').click({ position: { x: 2, y: 2 } });
    await expect(page.getByRole('dialog')).toHaveCount(0); await expect(page).toHaveURL(url);
    await expect(page.getByRole('button', { name: '更多操作', exact: true })).toBeFocused();
    await expect(page.getByLabel('搜索编码或名称')).toHaveValue('独立搜索草稿');
  }
  expect(requests).toBe(0);
  await panel(page, '列筛选…'); await page.getByRole('button', { name: '清除条件', exact: true }).click(); await page.getByRole('button', { name: '取消', exact: true }).click();
  expect(requests).toBe(0); await expect(page).toHaveURL(url);
  await page.getByLabel('搜索编码或名称').fill(fixture.prefix); await panel(page, '列筛选…'); await page.getByRole('button', { name: '清除条件', exact: true }).click();
  expect((await apply(page)).count).toBe(73); expect(requests).toBe(1); await expect(page).toHaveURL(u => !u.searchParams.has('filterField'));
});
const isListRequest = (r: any) => new URL(r.url()).pathname === '/api/v1/business-units' && r.method() === 'GET';
test('TL337 状态同列多值、替换为纯日期及未来首条', async ({ page }) => {
  await panel(page, '列筛选…'); await option(page, '列', '状态'); await option(page, '条件', '属于任一');
  for (const label of ['启用', '停用']) { await page.getByRole('dialog').getByRole('combobox', { name: '值', exact: true }).click(); await page.getByRole('option', { name: label, exact: true }).click(); }
  expect((await apply(page)).count).toBe(72); expect(new URL(page.url()).searchParams.get('filterValue')).toBe('active,inactive');
  await panel(page, '列筛选…'); await option(page, '条件', '等于');
  await page.getByRole('dialog').getByRole('combobox', { name: /^值/ }).click(); await page.getByRole('option', { name: '停用', exact: true }).click();
  expect((await apply(page)).count).toBe(1);
  await edit(page, '生效日期', '晚于', '2026-02-01');
  const dateValue = page.getByRole('dialog').getByLabel('值', { exact: true });
  await expect(dateValue).toHaveAttribute('type', 'text');
  await expect(dateValue).toHaveAttribute('placeholder', 'YYYY-MM-DD');
  let invalidRequests = 0;
  const invalidRequest = (request: any) => { if (isListRequest(request)) invalidRequests++; };
  page.on('request', invalidRequest);
  await dateValue.fill('2026-02-30');
  await page.getByRole('dialog').getByRole('button', { name: '应用', exact: true }).click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await expect(dateValue).toBeFocused();
  await page.waitForTimeout(150); expect(invalidRequests).toBe(0);
  page.off('request', invalidRequest);
  await dateValue.fill('2026-02-01'); let actual = await apply(page); expect(actual.count).toBe(1); expect(actual.items[0].status).toBe('not_effective');
  await edit(page, '生效日期', '等于', '2026-02-01'); actual = await apply(page); expect(actual.count).toBe(1); expect(actual.items[0].status).toBe('inactive');
});
test('TL338 A/D、分页、大小、排序、刷新及历史保留列条件', async ({ page }) => {
  await edit(page, '编码', '包含', fixture.prefix); await apply(page);
  await page.getByLabel('搜索编码或名称').fill('未应用分页草稿'); await page.getByRole('button', { name: '下一页', exact: true }).click();
  await expect(page.getByText('21–40 / 73', { exact: true })).toBeVisible(); await option(page, '每页条数', '50'); await expect(page.getByText('1–50 / 73', { exact: true })).toBeVisible();
  await expect(page.getByLabel('搜索编码或名称')).toHaveValue('未应用分页草稿');
  await page.getByLabel('搜索编码或名称').fill(fixture.prefix + 'G20'); await page.getByRole('columnheader', { name: /名称/ }).click(); await expect(page.getByText('1–20 / 20', { exact: true })).toBeVisible();
  const appliedURL = page.url(); await page.getByLabel('搜索编码或名称').fill('刷新丢弃'); await page.reload(); await expect(page.getByLabel('搜索编码或名称')).toHaveValue(fixture.prefix + 'G20');
  await edit(page, '编码', '等于', fixture.items[0].code); await apply(page); await page.goBack(); await expect(page).toHaveURL(appliedURL);
  await expect(page.getByText('1–20 / 20', { exact: true })).toBeVisible(); await page.goForward(); await expect(page.getByText('1–1 / 1', { exact: true })).toBeVisible();
});
test('TL339 详情跨页对象导航及返回携带列条件', async ({ page }) => {
  await edit(page, '名称', '包含', fixture.prefix + 'G21'); expect((await apply(page)).count).toBe(21);
  await page.getByRole('link', { name: fixture.items[19].name, exact: true }).click();
  await page.getByRole('button', { name: '下一个对象', exact: true }).click(); await expect(page.getByRole('heading', { level: 1 })).toContainText(fixture.items[20].code);
  await expect(page.getByRole('button', { name: '下一个对象', exact: true })).toBeDisabled();
  await page.getByRole('link', { name: '返回列表', exact: true }).click(); await expect(page.getByText('21–21 / 21', { exact: true })).toBeVisible();
  const target = page.getByRole('link', { name: fixture.items[20].name, exact: true }); await expect(target).toBeFocused(); expect(new URL(page.url()).searchParams.get('filterField')).toBe('name');
  await target.click(); await page.goBack(); await expect(target).toBeFocused(); await expect(target).toBeInViewport();
});
test('TL340 查询者字段权、撤权及无效直达', async ({ page, browser }) => {
  const started = performance.now();
  const mark = (step: string) => { if (process.env.CHNHCM_BROWSER_TIMINGS === '1') console.log(JSON.stringify({ event: 'TL340_step', step, elapsedMS: performance.now() - started })); };
  mark('start');
  const context = await browser.newContext();
  try {
    const reader = await context.newPage(); await login(reader, true); mark('reader-login'); await query(reader, fixture.prefix); mark('reader-query');
    await policy(reader, async (where, mutate) => {
      mark('policy-ready');
      mutate(`UPDATE access.subject_grant SET read_fields=ARRAY['code'] WHERE ${where} AND action='bu.read'`);
      await edit(reader, '名称', '包含', '列表样本'); expect((await apply(reader)).count).toBe(0); mark('hidden-field-filter');
      await edit(reader, '生效日期', '等于', '2026-01-01'); expect((await apply(reader)).count).toBe(0); mark('hidden-field-filter');
      await edit(reader, '编码', '等于', fixture.items[0].code); expect((await apply(reader)).count).toBe(1); mark('code-filter');
      mutate(`DELETE FROM access.subject_grant WHERE ${where} AND action='bu.read'`);
      const pending = reader.waitForResponse(isList); await reader.getByRole('button', { name: '查询', exact: true }).click(); expect((await pending).status()).toBe(403); await expect(reader.getByLabel('搜索编码或名称')).toHaveValue(''); mark('read-revoked');
    });
  } finally { await context.close(); mark('reader-closed'); }
  const pending = page.waitForResponse(isList); await page.goto(listPath + '?filterField=code&filterOperator=isEmpty&filterValue=xx'); expect((await pending).status()).toBe(400);
  await expect(page.getByText('列筛选条件无效', { exact: true })).toBeVisible(); await expect(page.locator('.MuiDataGrid-row')).toHaveCount(0); await expect(page.getByText('尚无查询结果', { exact: true })).toBeVisible(); mark('invalid-direct-complete');
});
test('TL341 当前失败重试及迟到真实响应不盖新条件', async ({ page }) => {
  await edit(page, '编码', '包含', fixture.prefix); await apply(page);
  await page.route('**/api/v1/business-units?*', route => route.fulfill({ status: 503, json: { code: 'UNAVAILABLE', message: '筛选查询受控失败' } }));
  await page.getByRole('button', { name: '下一页', exact: true }).click(); await expect(page.getByText('筛选查询受控失败')).toBeVisible(); await expect(page.locator('.MuiDataGrid-row')).toHaveCount(0);
  await page.getByLabel('搜索编码或名称').fill('重试保留草稿'); await page.unrouteAll(); await page.getByRole('button', { name: '重试', exact: true }).click(); await expect(page.getByText('21–40 / 73', { exact: true })).toBeVisible();
  await expect(page.getByLabel('搜索编码或名称')).toHaveValue('重试保留草稿'); await page.getByLabel('搜索编码或名称').fill(fixture.prefix);
  let release!: () => void, arrived!: () => void; const held = new Promise<void>(r => release = r), fetched = new Promise<void>(r => arrived = r);
  await page.route('**/api/v1/business-units?*', async route => { const response = await route.fetch(); if (new URL(route.request().url()).searchParams.get('filterValue') === fixture.items[0].code) { arrived(); await held; } await route.fulfill({ response }); });
  try {
    await edit(page, '编码', '等于', fixture.items[0].code); await page.getByRole('button', { name: '应用', exact: true }).click(); await fetched;
    await edit(page, '编码', '等于', fixture.items[1].code); expect((await apply(page)).items[0].id).toBe(fixture.items[1].id);
    release(); await page.unrouteAll({ behavior: 'wait' }); await expect(page.getByRole('link', { name: fixture.items[1].name, exact: true })).toBeVisible(); await expect(page.getByRole('link', { name: fixture.items[0].name, exact: true })).toHaveCount(0);
  } finally { release(); }
});
test('TL342 窄屏/短屏布局、键盘、焦点及断点关闭', async ({ page }) => {
  for (const [width, height] of [[390, 844], [320, 568], [1200, 420]]) {
    await page.setViewportSize({ width, height }); await panel(page, '列筛选…'); const dialog = page.getByRole('dialog'); await dialog.getByLabel('值', { exact: true }).fill(fixture.items[0].code);
    expect(await dialog.evaluate(el => el.scrollWidth <= el.clientWidth + 1)).toBe(true); expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    await page.getByRole('button', { name: '应用', exact: true }).focus(); await page.keyboard.press('Enter'); await expect(page.getByText('1–1 / 1', { exact: true })).toBeVisible();
    await panel(page, '列筛选…'); await dialog.getByLabel('值', { exact: true }).fill('取消草稿'); await page.keyboard.press('Escape'); await expect(page.getByRole('button', { name: '更多操作', exact: true })).toBeFocused();
  }
  await page.setViewportSize({ width: 1200, height: 900 }); await panel(page, '列筛选…'); await page.setViewportSize({ width: 390, height: 844 }); await expect(page.getByRole('dialog')).toHaveCount(0); await expect(page.getByRole('button', { name: '更多操作', exact: true })).toBeFocused();
});
