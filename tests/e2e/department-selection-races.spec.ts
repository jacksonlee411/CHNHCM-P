import { randomUUID } from 'node:crypto';
import { test, expect, openMoreActions, type Page } from './fixtures';
import { open } from './bu-support';
import { apiBU, apiDepartment, departmentKind, menu, request } from './dated-object-support';

const toolbar = (page: Page) => page.getByRole('toolbar', { name: '所选部门', exact: true });
const row = (page: Page, index = 0) => page.getByRole('grid', { name: '部门列表', exact: true }).locator('.MuiDataGrid-row').nth(index).getByRole('checkbox');
const selected = (page: Page, n: number) => expect(page.getByText(`已选择${n}项`, { exact: true })).toBeVisible();

function deferred() {
  let resolve!: () => void;
  const promise = new Promise<void>(done => { resolve = done; });
  return { promise, resolve };
}

// Hold a real successful response after the server has finished. Observe both
// the request barrier and browser fetch settlement; no timing-based race guess.
async function hold(page: Page, endpoint: string) {
  const captured = deferred(), released = deferred(), completed = deferred();
  let calls = 0;
  await page.route('**/api/v1/departments/' + endpoint, async route => {
    calls++;
    const response = await route.fetch();
    expect(response.status()).toBe(200);
    captured.resolve();
    await released.promise;
    await route.fulfill({ response }).catch(() => {});
    completed.resolve();
  });
  return { captured: captured.promise, release: released.resolve, completed: completed.promise, calls: () => calls };
}

test.beforeEach(async ({ page }) => {
  // Deliberately ignore cancellation only in this test's transport: the product
  // must reject late success even when abort cannot stop delivery.
  await page.addInitScript(() => {
    const original = window.fetch;
    (window as any).__selectionSettled = 0;
    window.fetch = async (input, init) => {
      const request = new Request(input, init);
      const selection = /\/departments\/(selection\/(?:start|resolve)|export)/.test(request.url);
      const response = await original(selection ? new Request(request, { signal: new AbortController().signal }) : request);
      if (selection) (window as any).__selectionSettled++;
      return response;
    };
  });
});

async function setup(page: Page, count = 3) {
  await open(page);
  const bu = await apiBU(page), prefix = 'RACE131' + randomUUID().slice(0, 6);
  for (let i = 0; i < count; i++) await apiDepartment(page, bu.id, { code: prefix + i, name: prefix + i });
  await menu(page, departmentKind);
  await page.getByRole('textbox', { name: '搜索编码或名称', exact: true }).fill(prefix);
  await page.getByRole('button', { name: '查询', exact: true }).click();
  await expect(page.getByText(`1–${Math.min(count, 20)} / ${count}`, { exact: true })).toBeVisible();
  return prefix;
}

test('Q131-V09 建立中连续勾选只建立一次，随后核对最新意图', async ({ page }) => {
  await setup(page);
  const barrier = await hold(page, 'selection/start');
  try {
    await row(page).check(); await barrier.captured;
    await row(page, 1).check();
    await expect(toolbar(page).getByRole('button', { name: '导出所选', exact: true })).toBeDisabled();
    barrier.release(); await barrier.completed;
    await selected(page, 2);
    expect(barrier.calls()).toBe(1);
    await expect(row(page)).toBeChecked(); await expect(row(page, 1)).toBeChecked();
  } finally { barrier.release(); await page.unrouteAll({ behavior: 'wait' }); }
});

test('Q131-V09 双标签退出后迟到成功CSV不得下载或保留旧选择', async ({ page, context }) => {
  await setup(page);
  const other = await context.newPage(); await other.goto('/');
  await expect(other.getByRole('button', { name: '退出', exact: true })).toBeVisible();
  await row(page).check(); await selected(page, 1);
  const barrier = await hold(page, 'export'), downloads: string[] = [];
  page.on('download', file => downloads.push(file.suggestedFilename()));
  try {
    await toolbar(page).getByRole('button', { name: '导出所选', exact: true }).click(); await barrier.captured;
    await other.getByRole('button', { name: '退出', exact: true }).click();
    await expect(other.getByRole('textbox', { name: '登录名', exact: true })).toBeVisible();
    barrier.release(); await barrier.completed;
    await expect.poll(() => page.evaluate(() => (window as any).__selectionSettled)).toBe(2);
    expect(downloads).toEqual([]);
    await expect(page.getByRole('textbox', { name: '登录名', exact: true })).toBeVisible();
    await expect(toolbar(page)).toHaveCount(0);
  } finally { barrier.release(); await page.unrouteAll({ behavior: 'wait' }); await other.close(); }
});

for (const phase of ['establishing', 'checking', 'exporting'] as const) {
  for (const action of ['clear', 'query', 'leave', 'history', 'logout'] as const) {
    test(`Q131-V09 ${phase}期间${action}丢弃迟到响应`, async ({ page }) => {
      const prefix = await setup(page);
      if (phase !== 'establishing') { await row(page).check(); await selected(page, 1); }
      const endpoint = phase === 'establishing' ? 'selection/start' : phase === 'checking' ? 'selection/resolve' : 'export';
      const barrier = await hold(page, endpoint), downloads: string[] = [];
      page.on('download', file => downloads.push(file.suggestedFilename()));
      const before = await page.evaluate(() => (window as any).__selectionSettled);
      try {
        if (phase === 'exporting') await toolbar(page).getByRole('button', { name: '导出所选', exact: true }).click();
        else await row(page, phase === 'checking' ? 1 : 0).check();
        await barrier.captured;
        await expect(page.locator('[data-selection-state]')).toHaveAttribute('data-selection-state', phase);
        await expect(toolbar(page).getByRole('button', { name: /导出/ })).toBeDisabled();
        if (phase !== 'exporting') await expect(page.getByText(/已选择\d+项/, { exact: true })).toHaveCount(0);
        if (action === 'clear') await toolbar(page).getByRole('button', { name: '取消选择', exact: true }).click();
        if (action === 'query') {
          await openMoreActions(page);
          await page.getByRole('menuitem', { name: '状态筛选…', exact: true }).click();
          await page.getByRole('combobox', { name: '状态', exact: true }).click();
          await page.getByRole('option', { name: '停用', exact: true }).click();
          await page.getByRole('button', { name: '应用', exact: true }).click();
        }
        if (action === 'leave') await page.getByRole('link', { name: prefix + '0', exact: true }).click();
        if (action === 'history') await page.goBack();
        if (action === 'logout') {
          await page.getByRole('button', { name: '退出', exact: true }).click();
          await expect(page.getByRole('textbox', { name: '登录名', exact: true })).toBeVisible();
        }
        barrier.release(); await barrier.completed;
        await expect.poll(() => page.evaluate(() => (window as any).__selectionSettled)).toBe(before + 1);
        await expect(toolbar(page)).toHaveCount(0); expect(downloads).toEqual([]);
        if (action === 'clear') {
          await expect(page.locator('[data-selection-state]')).toHaveAttribute('data-selection-state', 'idle');
          await expect(page.getByText('1–3 / 3', { exact: true })).toBeVisible();
          await expect(row(page)).not.toBeChecked();
        }
        if (action === 'query') await expect(page.getByText('没有符合条件的部门', { exact: true })).toBeVisible();
      } finally { barrier.release(); await page.unrouteAll({ behavior: 'wait' }); }
    });
  }
}

test('Q131-V09 取消最后一项返回GET，不发空选择核验', async ({ page }) => {
  await setup(page); await row(page).check(); await selected(page, 1);
  const calls: string[] = [];
  page.on('request', r => { const p = new URL(r.url()).pathname; if (p.startsWith('/api/v1/departments')) calls.push(r.method() + ' ' + p); });
  await row(page).uncheck();
  await expect(page.locator('[data-selection-state]')).toHaveAttribute('data-selection-state', 'idle');
  await expect(page.getByText('1–3 / 3', { exact: true })).toBeVisible();
  await expect(toolbar(page)).toHaveCount(0);
  expect(calls.filter(v => !v.endsWith('/selection/cancel'))).toEqual(['GET /api/v1/departments']); expect(calls.filter(v => v.endsWith('/selection/cancel'))).toHaveLength(1);
});

test('Q131-V09 首次建立后新增不改变固定意图，后续增量仍可完成', async ({ page }) => {
  const prefix = await setup(page), barrier = await hold(page, 'selection/start');
  try {
    await row(page).check(); await barrier.captured; await row(page, 1).check();
    const list = (await request(page, 'GET', '/api/v1/departments?q=' + prefix)).body;
    await apiDepartment(page, list.items[0].managementBu.id, { code: prefix + '3' });
    barrier.release(); await barrier.completed;
    await selected(page,2); await expect(row(page,0)).toBeChecked(); await expect(row(page,1)).toBeChecked(); await expect(row(page,3)).not.toBeChecked();
    await expect(page.getByText('1–4 / 4', { exact: true })).toBeVisible();
    expect(barrier.calls()).toBe(1);
  } finally { barrier.release(); await page.unrouteAll({ behavior: 'wait' }); }
});

test('Q131-V09 双标签换账号后旧建立结果不得进入新账号', async ({ page, context }) => {
  await setup(page);
  const other = await context.newPage(); await other.goto('/');
  await expect(other.getByRole('button', { name: '退出', exact: true })).toBeVisible();
  const barrier = await hold(page, 'selection/start');
  try {
    await test.step('捕获旧建立响应', async () => { await row(page).check(); await barrier.captured; });
    await test.step('另一标签退出', async () => { await other.getByRole('button', { name: '退出', exact: true }).click(); });
    await expect(other.getByRole('textbox', { name: '登录名', exact: true })).toBeVisible();
    await test.step('输入新账号', async () => { await other.getByRole('textbox', { name: '登录名', exact: true }).fill('reader'); });
    await test.step('输入开发预设密码', async () => { await other.getByLabel('密码').fill('admin1234567'); });
    const loginResponse = other.waitForResponse(r => r.url().endsWith('/api/v1/session/login')); await other.getByRole('button', { name: '登录', exact: true }).click(); expect((await loginResponse).status()).toBe(200);
    await expect(other.locator('.app-account-name')).toHaveText('reader');
    await expect(page.locator('.app-account-name')).toHaveText('reader');
    barrier.release(); await barrier.completed;
    await expect.poll(() => page.evaluate(() => (window as any).__selectionSettled)).toBe(1);
    await expect(toolbar(page)).toHaveCount(0);
    await expect(page.locator('[data-selection-state]')).toHaveAttribute('data-selection-state', 'idle');
  } finally { barrier.release(); await page.unrouteAll({ behavior: 'ignoreErrors' }).catch(() => {}); if (!other.isClosed()) await other.close().catch(() => {}); }
});

test('Q131-V09 翻页核对期间清空后轻量读取当前页，旧响应不得恢复', async ({ page }) => {
  await setup(page, 21); await row(page).check(); await selected(page, 1);
  const barrier = await hold(page, 'selection/resolve');
  const calls: string[] = [];
  page.on('request', r => { if (new URL(r.url()).pathname === '/api/v1/departments') calls.push(r.method()); });
  try {
    await page.getByRole('button', { name: '下一页', exact: true }).click(); await barrier.captured;
    await toolbar(page).getByRole('button', { name: '取消选择', exact: true }).click();
    await expect(page.getByText('21–21 / 21', { exact: true })).toBeVisible();
    barrier.release(); await barrier.completed;
    await expect.poll(() => page.evaluate(() => (window as any).__selectionSettled)).toBe(2);
    await expect(toolbar(page)).toHaveCount(0);
    await expect(page.locator('[data-selection-state]')).toHaveAttribute('data-selection-state', 'idle');
    await expect(page.getByRole('alert')).toHaveCount(0);
    expect(calls).toEqual(['GET']);
  } finally { barrier.release(); await page.unrouteAll({ behavior: 'wait' }); }
});


test('Q131-V09 在新标签打开详情不取消原页已确认选择', async ({ page, context }) => {
  const prefix = await setup(page); await row(page).check(); await selected(page, 1);
  const opened = context.waitForEvent('page');
  await page.getByRole('link', { name: prefix + '0', exact: true }).click({ modifiers: ['Control'] });
  const other = await opened;
  try {
    await other.waitForLoadState('domcontentloaded');
    await expect(other.getByRole('heading', { name: prefix + '0 · ' + prefix + '0', exact: true })).toBeVisible();
    await expect(page.locator('[data-selection-state]')).toHaveAttribute('data-selection-state', 'ready');
    await selected(page, 1);
    await expect(toolbar(page).getByRole('button', { name: '导出所选', exact: true })).toBeEnabled();
  } catch (error) {
    await test.info().attach('popup-masked', { body: await other.screenshot({ mask: [other.locator('.app-account-name'), other.getByLabel('登录名'), other.getByLabel('密码')] }), contentType: 'image/png' });
    await test.info().attach('popup-headings', { body: JSON.stringify(await other.getByRole('heading').allTextContents()), contentType: 'application/json' });
    throw error;
  } finally { await other.close(); }
});

test('Q131-V09 退出失败后重新读取列表，迟到旧GET不能使页面永久等待', async ({ page }) => {
  await setup(page);
  const captured = deferred(), release = deferred(), completed = deferred();
  let reads = 0;
  await page.route('**/api/v1/departments?*', async route => {
    reads++;
    if (reads !== 1) { await route.continue(); return; }
    const response = await route.fetch(); expect(response.status()).toBe(200);
    captured.resolve(); await release.promise;
    await route.fulfill({ response }).catch(() => {}); completed.resolve();
  });
  await page.route('**/api/v1/session', async route => {
    if (route.request().method() === 'DELETE') await route.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ code: 'SERVICE_ERROR', message: '测试退出失败' }) });
    else await route.continue();
  });
  try {
    await page.getByRole('button', { name: '查询', exact: true }).click(); await captured.promise;
    await page.getByRole('button', { name: '退出', exact: true }).click();
    await expect(page.getByText('退出未完成，请重试', { exact: true })).toBeVisible();
    release.resolve(); await completed.promise;
    await expect(page.getByText('1–3 / 3', { exact: true })).toBeVisible();
    expect(reads).toBe(2);
    await expect(page.locator('[data-selection-state]')).toHaveAttribute('data-selection-state', 'idle');
  } finally { release.resolve(); await page.unrouteAll({ behavior: 'wait' }); }
});

test('Q131-V07 全选捕获新增成员时，排除旧页全部三项仍保留新增的一项', async ({ page }) => {
  const prefix = await setup(page);
  const current = (await request(page, 'GET', '/api/v1/departments?q=' + prefix)).body;
  await apiDepartment(page, current.items[0].managementBu.id, { code: prefix + '3', name: prefix + '3' });
  const barrier = await hold(page, 'selection/start');
  try {
    await test.step('建立前页面仍为3项，服务器捕获4项', async () => {
      await page.getByRole('checkbox', { name: '全选查询结果', exact: true }).check(); await barrier.captured;
    });
    await test.step('排除旧页三项', async () => { for (let i = 0; i < 3; i++) await row(page, i).uncheck(); });
    barrier.release(); await barrier.completed;
    await test.step('仍选择捕获时新增的一项', async () => { await selected(page, 1); });
    await expect(page.getByText('1–4 / 4', { exact: true })).toBeVisible();
    for (let i = 0; i < 3; i++) await expect(row(page, i)).not.toBeChecked();
    await expect(row(page, 3)).toBeChecked();
    expect(barrier.calls()).toBe(1);
    const calls: string[] = [];
    page.on('request', r => { const path = new URL(r.url()).pathname; if (path.startsWith('/api/v1/departments')) calls.push(r.method() + ' ' + path); });
    await row(page, 3).uncheck();
    await expect(toolbar(page)).toHaveCount(0);
    await expect(page.locator('[data-selection-state]')).toHaveAttribute('data-selection-state', 'idle');
    await expect(page.getByText('1–4 / 4', { exact: true })).toBeVisible();
    expect(calls.filter(v => !v.endsWith('/selection/cancel'))).toEqual(['GET /api/v1/departments']); expect(calls.filter(v => v.endsWith('/selection/cancel'))).toHaveLength(1);
  } finally { barrier.release(); await page.unrouteAll({ behavior: 'wait' }).catch(() => {}); }
});

test('Q131-V09 另一标签退出响应未返回时，已在途CSV也必须失效', async ({ page, context }) => {
  await setup(page);
  const other = await context.newPage(); await other.goto('/');
  await expect(other.getByRole('button', { name: '退出', exact: true })).toBeVisible();
  await row(page).check(); await selected(page, 1);
  const csv = await hold(page, 'export'), logoutCaptured = deferred(), logoutRelease = deferred();
  const downloads: string[] = [];
  page.on('download', file => downloads.push(file.suggestedFilename()));
  await other.route('**/api/v1/session', async route => {
    if (route.request().method() !== 'DELETE') { await route.continue(); return; }
    const response = await route.fetch(); expect(response.status()).toBe(204);
    logoutCaptured.resolve(); await logoutRelease.promise;
    await route.fulfill({ response }).catch(() => {});
  });
  try {
    await toolbar(page).getByRole('button', { name: '导出所选', exact: true }).click(); await csv.captured;
    await other.getByRole('button', { name: '退出', exact: true }).click(); await logoutCaptured.promise;
    await expect(toolbar(page)).toHaveCount(0);
    csv.release(); await csv.completed;
    await expect.poll(() => page.evaluate(() => (window as any).__selectionSettled)).toBe(2);
    expect(downloads).toEqual([]);
    logoutRelease.resolve();
    await expect(other.getByRole('textbox', { name: '登录名', exact: true })).toBeVisible();
  } finally {
    csv.release(); logoutRelease.resolve();
    await page.unrouteAll({ behavior: 'ignoreErrors' }).catch(() => {});
    await other.unrouteAll({ behavior: 'ignoreErrors' }).catch(() => {});
    await other.close().catch(() => {});
  }
});

test('Q131-V15 退出失败不得清除另一标签的未保存输入', async ({ page, context }) => {
  await setup(page);
  const other = await context.newPage(); await other.goto('/');
  try {
    await menu(other, departmentKind);
    await other.getByRole('link', { name: '新建', exact: true }).click();
    await other.getByRole('textbox', { name: '部门名称', exact: true }).fill('另一标签未保存输入');
    await page.route('**/api/v1/session', async route => {
      if (route.request().method() === 'DELETE') await route.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ code: 'SERVICE_ERROR', message: '测试退出失败' }) });
      else await route.continue();
    });
    await page.getByRole('button', { name: '退出', exact: true }).click();
    await expect(page.getByText('退出未完成，请重试', { exact: true })).toBeVisible();
    await expect(other.getByRole('textbox', { name: '部门名称', exact: true })).toHaveValue('另一标签未保存输入');
    await other.getByRole('link', { name: '取消', exact: true }).click();
    await expect(other.getByRole('dialog', { name: '尚有未保存修改', exact: true })).toBeVisible();
  } finally { await page.unrouteAll({ behavior: 'ignoreErrors' }).catch(() => {}); await other.close().catch(() => {}); }
});


for (const phase of ['establishing', 'checking'] as const) {
  for (const position of ['page', 'pageSize', 'sort'] as const) {
    test(`Q131-V09 ${phase}期间${position}保留最新勾选及原基准`, async ({ page }) => {
      await setup(page, 21);
      const ids = await page.getByRole('grid', { name: '部门列表', exact: true }).locator('.MuiDataGrid-row').evaluateAll(rows => rows.slice(0, 2).map(row => row.getAttribute('data-id')));
      const calls: { path: string; body: any }[] = [];
      page.on('request', request => {
        const path = new URL(request.url()).pathname;
        if (path.includes('/departments/selection/')) calls.push({ path, body: request.postDataJSON() });
      });
      if (phase === 'checking') { await row(page).check(); await selected(page, 1); }
      const barrier = await hold(page, phase === 'establishing' ? 'selection/start' : 'selection/resolve');
      try {
        await row(page, phase === 'checking' ? 1 : 0).check(); await barrier.captured;
        if (phase === 'establishing') await row(page, 1).check();
        if (position === 'page') await page.getByRole('button', { name: '下一页', exact: true }).click();
        if (position === 'pageSize') {
          await page.getByRole('combobox', { name: '每页条数', exact: true }).click();
          await page.getByRole('option', { name: '50', exact: true }).click();
        }
        if (position === 'sort') await page.getByRole('columnheader', { name: /名称/ }).click();
        await expect(toolbar(page).getByRole('button', { name: '导出所选', exact: true })).toBeDisabled();
        barrier.release(); await barrier.completed;
        if (position === 'page') await expect(page.getByText('21–21 / 21', { exact: true })).toBeVisible();
        if (position === 'pageSize') await expect(page.getByText('1–21 / 21', { exact: true })).toBeVisible();
        if (position === 'sort') await expect(page).toHaveURL(/sort=name/);
        await selected(page, 2);
        expect(calls.filter(call => call.path.endsWith('/start'))).toHaveLength(1);
        const resolutions = calls.filter(call => call.path.endsWith('/resolve'));
        expect(resolutions.length).toBeGreaterThan(0);
        expect(new Set(resolutions.map(call => call.body.draftId)).size).toBe(1);
        const actual = new Set<string>(calls.find(call => call.path.endsWith('/start'))!.body.selection.ids);
        let revision = 1;
        for (const { body } of resolutions) {
          expect(body).not.toHaveProperty('expectedSelectionVersion');
          expect(body.expectedRevision).toBeGreaterThanOrEqual(revision); revision = body.expectedRevision;
          for (const id of body.addIds ?? []) actual.add(id);
          for (const id of body.removeIds ?? []) actual.delete(id);
        }
        expect([...actual].sort()).toEqual(ids.sort());
      } finally { barrier.release(); await page.unrouteAll({ behavior: 'wait' }); }
    });
  }
}

test('Q131-V09 取消选择同时丢弃等待建立完成的翻页意图', async ({ page }) => {
  await setup(page, 21);
  const barrier = await hold(page, 'selection/start');
  try {
    await row(page).check(); await barrier.captured;
    await page.getByRole('button', { name: '下一页', exact: true }).click();
    await toolbar(page).getByRole('button', { name: '取消选择', exact: true }).click();
    barrier.release(); await barrier.completed;
    await expect.poll(() => page.evaluate(() => (window as any).__selectionSettled)).toBe(1);
    await expect(toolbar(page)).toHaveCount(0);
    await expect(page.getByText('1–20 / 21', { exact: true })).toBeVisible();
    expect(new URL(page.url()).searchParams.get('page')).not.toBe('2');
  } finally { barrier.release(); await page.unrouteAll({ behavior: 'wait' }); }
});
