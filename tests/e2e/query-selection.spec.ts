import { nativeZoomCases } from './desktop-viewports.mjs';
import { test, expect } from './fixtures';
import { login, query, panel, option, policy, connection } from './community-list-support.mjs';
import { randomUUID } from 'node:crypto';
import { mkdirSync, appendFileSync } from 'node:fs';
import { nativeZoomBrowser } from './native-zoom-support';

const prefix = 'QS' + randomUUID().slice(0, 7);
const items: { id: string; code: string; name: string }[] = [];
const grid = (page: any) => page.getByRole('grid', { name: '业务单元列表' });
const head = (page: any) => page.getByRole('checkbox', { name: /全选查询结果/ });
const rowBox = (page: any, index = 0) => grid(page).locator('.MuiDataGrid-row').nth(index).getByRole('checkbox');
const selected = async (page: any, count: number) => expect(page.getByText(`已选择${count}项`, { exact: true })).toBeVisible();
test.beforeAll(async ({ browser }) => {
  test.setTimeout(120000); const context = await browser.newContext();
  try { const page = await context.newPage(); await login(page);
    const session = await (await page.request.get('/api/v1/session')).json();
    for (let i = 0; i < 137; i++) {
      const code = prefix + String(i).padStart(3, '0'), name = i === 0 ? '+1+1' : `选择样本${i}`;
      const response = await page.request.post('/api/v1/business-units', { headers: { Origin: connection.webOrigin, 'X-CSRF-Token': session.csrf, 'Idempotency-Key': randomUUID() }, data: { code, name, effectiveDate: '2026-01-01', operationType: 'CREATE', operationReason: 'NEW_ORGANIZATION' } });
      expect(response.status()).toBe(201); items.push({ id: (await response.json()).buId, code, name });
    }
  } finally { await context.close(); }
});
test.beforeEach(async ({ page }) => { await login(page); await query(page, prefix); });
test.afterEach(async ({}, info) => {
  const out = '.local/115-selection'; mkdirSync(out, { recursive: true });
  appendFileSync(out + '/browser.jsonl', JSON.stringify({ title: info.title, status: info.status, durationMs: info.duration }) + '\n');
});
test('TL349/351/353/360 全137、跨页排除、大小排序保持及完整CSV', async ({ page }) => {
  await head(page).check(); await selected(page, 137);
  await page.getByRole('button', { name: '下一页', exact: true }).click(); await expect(page.getByText('21–40 / 137', { exact: true })).toBeVisible(); await selected(page, 137);
  await rowBox(page).uncheck(); await selected(page, 136); await expect(head(page)).toHaveAttribute('aria-checked', 'mixed');
  await page.getByRole('button', { name: '上一页', exact: true }).click(); await selected(page, 136); await expect(rowBox(page)).toBeChecked();
  await page.getByRole('combobox', { name: '每页条数', exact: true }).click(); await page.getByRole('option', { name: '50', exact: true }).click(); await selected(page, 136);
  await page.getByRole('combobox', { name: '每页条数', exact: true }).click(); await page.getByRole('option', { name: '100', exact: true }).click(); await selected(page, 136);
  const sorted = page.waitForResponse(response => new URL(response.url()).pathname === '/api/v1/business-units' && new URL(response.url()).searchParams.get('sort') === 'name');
  await page.getByRole('columnheader', { name: /名称/ }).click(); await (await sorted).finished(); await selected(page, 136); await expect(page.getByRole('button', { name: '导出所选', exact: true })).toBeEnabled();
  const pending = page.waitForEvent('download'); await page.getByRole('button', { name: '导出所选', exact: true }).click(); const file = await pending;
  let csv = ''; for await (const chunk of await file.createReadStream()) csv += chunk;
  expect(csv.trim().split(/\r?\n/)).toHaveLength(137); expect(csv).not.toContain(items[20].code); expect(csv).toContain("'+1+1");
  expect(csv.replace(/^\uFEFF/, '').split(/\r?\n/)[0]).toBe('编码,名称,状态,生效日期');
  await page.getByRole('button', { name: '取消选择', exact: true }).click(); await expect(page.getByLabel('搜索编码或名称')).toHaveValue(prefix);
});
test('TL350/352/355 跨页逐项、空集、返回高亮与刷新独立', async ({ page }) => {
  await rowBox(page).check(); await selected(page, 1); await page.getByRole('button', { name: '下一页', exact: true }).click(); await expect(page.getByText('21–40 / 137', { exact: true })).toBeVisible(); await rowBox(page).check(); await selected(page, 2);
  await page.getByRole('button', { name: '上一页', exact: true }).click(); await expect(page.getByText('1–20 / 137', { exact: true })).toBeVisible(); await expect(rowBox(page)).toBeChecked(); await rowBox(page).uncheck(); await selected(page, 1);
  await page.reload(); await expect(page.getByRole('toolbar', { name: '所选业务单元' })).toHaveCount(0);
  await grid(page).locator('a[data-object-link]').first().click(); await page.getByRole('link', { name: '返回列表', exact: true }).click();
  await expect(grid(page).locator('.list-return-selected')).toHaveCount(1); await expect(rowBox(page)).not.toBeChecked();
  await query(page, prefix + 'NONE'); await expect(head(page)).toBeDisabled();
});
test('TL354 搜索草稿、取消与应用条件', async ({ page }) => {
  await page.getByLabel('搜索编码或名称').fill('未应用草稿'); await head(page).check(); await selected(page, 137);
  await page.getByRole('button', { name: '取消选择', exact: true }).click(); await expect(page.getByLabel('搜索编码或名称')).toHaveValue('未应用草稿');
  await page.getByLabel('搜索编码或名称').fill(prefix); await head(page).check(); await selected(page, 137);
  await panel(page, '状态筛选…'); await page.getByRole('button', { name: '取消', exact: true }).click(); await selected(page, 137);
  await panel(page, '列筛选…'); await option(page, '列', '编码'); await option(page, '条件', '等于'); await page.getByRole('dialog').getByLabel('值', { exact: true }).fill(items[0].code); await page.getByRole('button', { name: '应用', exact: true }).click();
  await expect(page.getByText('1–1 / 1', { exact: true })).toBeVisible(); await expect(page.getByRole('toolbar', { name: '所选业务单元' })).toHaveCount(0);
});
test('TL359 取消时旧核对响应不可恢复选择', async ({ page }) => {
  let release!: () => void; const held = new Promise<void>(resolve => { release = resolve; });
  await page.route('**/business-units/selection/resolve', async route => { await held; await route.continue().catch(() => {}); });
  await head(page).check(); await expect(page.getByText('正在核对选择…', { exact: true })).toBeVisible(); await page.getByRole('button', { name: '取消选择', exact: true }).click(); release();
  await expect(page.getByRole('toolbar', { name: '所选业务单元' })).toHaveCount(0); await expect(head(page)).not.toBeChecked();
});
test('TL359/363 查询开始停止导出，取消后迟到CSV不产生下载', async ({ page }) => {
  await head(page).check(); await selected(page, 137);
  let release!: () => void; let held = new Promise<void>(resolve => { release = resolve; });
  await page.route('**/api/v1/business-units?*', async route => {
    if (new URL(route.request().url()).searchParams.get('sort') === 'name') await held;
    await route.continue().catch(() => {});
  });
  try {
    await page.getByRole('columnheader', { name: /名称/ }).click();
    await expect(page.getByRole('button', { name: '导出所选', exact: true })).toBeDisabled();
    release(); await expect(page.getByRole('button', { name: '导出所选', exact: true })).toBeEnabled(); await selected(page, 137);
  } finally { release(); await page.unrouteAll({ behavior: 'ignoreErrors' }); }
  held = new Promise<void>(resolve => { release = resolve; });
  let fetched!: () => void; const ready = new Promise<void>(resolve => { fetched = resolve; });
  const downloads: string[] = []; page.on('download', file => downloads.push(file.suggestedFilename()));
  await page.route('**/api/v1/business-units/export', async route => {
    const response = await route.fetch(); fetched(); await held; await route.fulfill({ response }).catch(() => {});
  });
  try {
    await page.getByRole('button', { name: '导出所选', exact: true }).click(); await ready;
    await page.getByRole('button', { name: '取消选择', exact: true }).click(); release();
    await expect(page.getByLabel('搜索编码或名称')).toHaveValue(prefix);
    await page.unrouteAll({ behavior: 'wait' }); expect(downloads).toEqual([]);
  } finally { release(); await page.unrouteAll({ behavior: 'ignoreErrors' }); }
});
test('TL356/357/361 真实权限变化、同数来源变化及只读导出分权', async ({ page }) => {
  await head(page).check(); await selected(page, 137);
  const downloads: string[] = []; page.on('download', file => downloads.push(file.suggestedFilename()));
  await policy(page, async (where: string, mutate: (sql: string) => void) => {
    mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE ${where} AND action='bu.read'`);
    await page.getByRole('button', { name: '导出所选', exact: true }).click(); await expect(page.getByText('列表已变化，请重新选择', { exact: true })).toBeVisible();
    await expect(page.getByRole('toolbar', { name: '所选业务单元' })).toHaveCount(0); expect(downloads).toEqual([]);
  });
  await page.getByRole('button', { name: '退出', exact: true }).click(); await expect(page.getByLabel('登录名')).toBeVisible(); await login(page, true); await query(page, prefix); await head(page).check(); await selected(page, 137); await expect(page.getByRole('button', { name: '导出所选', exact: true })).toBeDisabled();
});
test('TL365 Space、Shift、网格全选、摘要列', async ({ page }) => {
  await rowBox(page).focus(); await page.keyboard.press('Space'); await selected(page, 1);
  await rowBox(page, 3).click({ modifiers: ['Shift'] }); await selected(page, 4);
  await page.keyboard.press('Control+a'); await selected(page, 137);
  for (const size of [{ width: 800, height: 900 }, { width: 1440, height: 900 }]) {
    await page.setViewportSize(size); await selected(page, 137); await expect(head(page)).toBeVisible(); await expect(page.getByRole('button', { name: '取消选择', exact: true })).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
  }
});

test('TL365 Chromium 原生缩放，选择和操作可达', async () => {
  test.setTimeout(120000);
  const native=await nativeZoomBrowser(),page=native.page;
  try {
    await login(page); await query(page, prefix); await head(page).check(); await selected(page, 137);
    const dimensions = () => page.evaluate(() => ({ width: innerWidth, height: innerHeight, dpr: devicePixelRatio, outer: [outerWidth, outerHeight] }));
    for (const [, dpr] of nativeZoomCases) {
      await native.setZoom(dpr); await selected(page,137);
      await head(page).scrollIntoViewIfNeeded(); await expect(head(page)).toBeInViewport();
      const clear = page.getByRole('button', { name: '取消选择', exact: true }); await clear.scrollIntoViewIfNeeded(); await expect(clear).toBeInViewport();
      const exportButton = page.getByRole('button', { name: '导出所选', exact: true }); await exportButton.scrollIntoViewIfNeeded(); await expect(exportButton).toBeInViewport();
      await page.mouse.wheel(0, 400); expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
      mkdirSync('.local/115-selection', { recursive: true });
      await exportButton.scrollIntoViewIfNeeded();
      await page.screenshot({path: `.local/115-selection/zoom${dpr * 100}-visible-${prefix}.png`,mask:[page.locator('.app-account-name')]});
      appendFileSync('.local/115-selection/zoom.jsonl', JSON.stringify({ dpr, dimensions: await dimensions(), selected: 137, result: 'passed' }) + '\n');
    }
  } finally { await native.close(); }
});
