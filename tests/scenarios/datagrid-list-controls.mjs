// UI adapter checks with synthetic HTTP responses; no DB/business acceptance.
// Run after `pnpm build`; mounts the actual source-built Community component.
import { chromium, expect } from '@playwright/test';
import { createServer } from 'node:http';
import { readFile, stat, mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import assert from 'node:assert/strict';

const out = path.resolve(process.argv[2] ?? '.local/111-community-t01-t02/controls');
await mkdir(out, { recursive: true });
const dist = path.resolve('apps/web/dist');
const types = { '.html': 'text/html', '.js': 'text/javascript', '.css': 'text/css', '.otf': 'font/otf', '.png': 'image/png', '.txt': 'text/plain' };
const server = createServer(async (req, res) => {
  try {
    const requested = path.resolve(dist, '.' + new URL(req.url, 'http://localhost').pathname);
    if (!requested.startsWith(dist + '/') && requested !== dist) { res.writeHead(403).end(); return; }
    let file = requested;
    try { if (!(await stat(file)).isFile()) file = path.join(dist, 'index.html'); }
    catch { file = path.join(dist, 'index.html'); }
    res.writeHead(200, { 'content-type': types[path.extname(file)] ?? 'application/octet-stream' });
    res.end(await readFile(file));
  } catch { res.writeHead(500).end(); }
});
await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
const origin = `http://127.0.0.1:${server.address().port}`;
const results = { scope: 'Actual source-built Community with synthetic API; checks UI adapters, not real Go/DB business acceptance', checks: [], warnings: [], consoleErrors: [], pageErrors: [], externalRequests: [], stopped: false };
let browser, page, releaseContext;
const check = name => results.checks.push({ name, result: 'passed' });
try {
  browser = await chromium.launch({ headless: true });
  page = await browser.newPage({ viewport: { width: 1440, height: 900 }, locale: 'zh-CN' });
  page.on('console', message => { if (message.type() === 'warning') results.warnings.push(message.text()); if (message.type() === 'error') results.consoleErrors.push(message.text()); });
  page.on('pageerror', error => results.pageErrors.push(error.message));
  const queries = [], writes = [];
  let manyRows = false, failNext = false, denyNext = false, canCreate = true, reasonFailure = false, holdContext = false, delayed;
  await page.route('**/*', async route => {
    const request = route.request(), url = new URL(request.url());
    if (url.origin !== origin) { results.externalRequests.push(url.origin); await route.abort(); return; }
    if (!url.pathname.startsWith('/api/')) { await route.continue(); return; }
    if (request.method() !== 'GET') writes.push({ method: request.method(), path: url.pathname });
    const json = value => route.fulfill({ json: value });
    if (url.pathname === '/api/v1/session') return json({ subjectId: 'synthetic-subject', login: 'UI合成角色', csrf: 'synthetic-only', instanceId: 'synthetic-instance' });
    if (url.pathname === '/api/v1/organization/context') {
      if (holdContext) { holdContext = false; await new Promise(resolve => { releaseContext = resolve; }); }
      return json({ currentDate: '2026-11-03', timezone: 'Asia/Shanghai', enterprise: { id: 'synthetic-enterprise', code: 'UI', name: '合成企业', validFrom: '2020-01-01' }, capabilities: { create: canCreate, readFields: ['code','name','status','effectiveDate'], writeFields: ['code','name','nameEn','effectiveDate','operationReason'] } });
    }
    if (url.pathname === '/api/v1/organization/operation-reasons') return reasonFailure ? route.fulfill({ status: 503, json: { code: 'UNAVAILABLE', message: '合成原因目录失败' } }) : json([{ code: 'NEW_ORGANIZATION', name: '新设组织' }]);
    if (url.pathname === '/api/v1/business-units') {
      queries.push(Object.fromEntries(url.searchParams));
      if (url.searchParams.get('q') === 'slow') { await new Promise(resolve => { delayed = resolve; }); return route.fulfill({ status: 403, json: { code: 'DENIED', message: '过期拒绝' } }); }
      if (failNext || denyNext) {
        const denied = denyNext; failNext = denyNext = false;
        return route.fulfill({ status: denied ? 403 : 503, json: { code: denied ? 'DENIED' : 'UNAVAILABLE', message: denied ? '当前无权读取' : '合成读取失败' } });
      }
      if (manyRows) {
        const size = Number(url.searchParams.get('pageSize') ?? 20), page = Number(url.searchParams.get('page') ?? 1);
        const rows = Array.from({ length: 73 }, (_, i) => ({ id: `page-row-${i + 1}`, code: `UI-${String(i + 1).padStart(3, '0')}`, name: `合成分页 ${i + 1}`, status: 'active', effectiveDate: '2026-10-01' }));
        return json({ items: rows.slice((page - 1) * size, page * size), count: rows.length, page, pageSize: size, businessAt: url.searchParams.get('businessAt'), currentDate: '2026-11-03' });
      }
      return json({ items: [{ id: 'one', code: 'UI-001', name: '合成单元', status: 'active', effectiveDate: '2026-10-01' }], count: 1, page: 1, pageSize: Number(url.searchParams.get('pageSize') ?? 20), businessAt: url.searchParams.get('businessAt'), currentDate: '2026-11-03' });
    }
    return route.fulfill({ status: 404, json: { code: 'NOT_FOUND', message: '未定义的合成请求' } });
  });
  await page.goto(origin);
  await expect(page.getByRole('heading', { name: '业务单元', exact: true })).toBeVisible();
  await page.getByRole('button', { name: '组织配置', exact: true }).click();
  await page.getByRole('menuitem', { name: '业务单元', exact: true }).click();
  await expect(page.getByText('列表暂不可用，请联系管理员。')).toHaveCount(0);
  await expect(page.getByRole('grid', { name: '业务单元列表' })).toBeVisible();
  await expect(page.getByRole('link', { name: '合成单元', exact: true })).toBeVisible();
  await expect(page.getByText('1–1 / 1', { exact: true })).toBeVisible();
  await expect(page.getByLabel('查看日期')).toHaveCount(0);
  await expect(page.getByRole('combobox', { name: '每页条数' })).toBeVisible();
  check('normal menu mounts source-built Community and native pagination');

  const listURL = page.url();
  await page.getByRole('gridcell', { name: 'UI-001', exact: true }).click();
  await page.keyboard.press('Enter');
  await expect(page).toHaveURL(listURL);
  check('ordinary cell click and Enter keep the list; only the name is a real detail link');

  const search = page.getByRole('textbox', { name: '搜索编码或名称' });
  const more = page.getByRole('button', { name: '更多操作', exact: true });
  const query = page.getByRole('button', { name: '查询', exact: true });
  const linkedQuery = async () => {
    const href = new URL(await page.getByRole('link', { name: '合成单元', exact: true }).getAttribute('href'), origin);
    return new URL(href.searchParams.get('back'), origin).searchParams;
  };
  const open = async name => { await more.click(); await page.getByRole('menuitem', { name, exact: true }).click(); };
  const beforeColumns = queries.length;
  const columnMenu = async label => {
    await page.getByRole('columnheader', { name: new RegExp(label) }).hover();
    await page.getByRole('button', { name: `${label}列操作`, exact: true }).click();
  };
  await columnMenu('编码');
  await expect(page.getByRole('menuitem', { name: '降序', exact: true })).toHaveCount(0);
  await expect(page.getByRole('menuitem', { name: /筛选/ })).toHaveCount(0);
  await expect(page.getByRole('menuitem', { name: /固定/ })).toHaveCount(0);
  await page.keyboard.press('Escape'); await expect(page.getByRole('menu')).toHaveCount(0);
  await columnMenu('名称');
  await page.getByRole('menuitem', { name: '隐藏列', exact: true }).click();
  await expect(page.getByRole('columnheader', { name: /名称/ })).toHaveCount(0);
  await open('列设置…');
  await page.getByRole('checkbox', { name: '名称', exact: true }).check();
  await page.keyboard.press('Escape'); await expect(page.getByRole('menu')).toHaveCount(0);
  await expect(page.getByRole('link', { name: '合成单元', exact: true })).toBeVisible();
  await open('列设置…');
  await page.getByRole('checkbox', { name: '显示/隐藏全部', exact: true }).uncheck();
  await page.keyboard.press('Escape'); await expect(page.getByRole('menu')).toHaveCount(0);
  await expect(page.getByRole('columnheader')).toHaveCount(0);
  await open('列设置…');
  await page.getByRole('button', { name: '重置', exact: true }).click();
  await page.keyboard.press('Escape'); await expect(page.getByRole('menu')).toHaveCount(0);
  await expect(page.getByRole('columnheader')).toHaveCount(4);
  const codeHeader = page.getByRole('columnheader', { name: /编码/ });
  const initialWidth = await codeHeader.evaluate(node => node.getBoundingClientRect().width);
  const separator = codeHeader.locator('.MuiDataGrid-columnSeparator');
  await codeHeader.hover();
  const grip = await separator.boundingBox();
  await page.mouse.move(grip.x + grip.width / 2, grip.y + grip.height / 2);
  await page.mouse.down(); await page.mouse.move(grip.x + 50, grip.y + grip.height / 2, { steps: 6 }); await page.mouse.up();
  await expect.poll(() => codeHeader.evaluate(node => node.getBoundingClientRect().width)).toBeGreaterThan(initialWidth + 20);
  const resizedWidth = await codeHeader.evaluate(node => node.getBoundingClientRect().width);
  await search.fill('布局草稿');
  await expect.poll(() => codeHeader.evaluate(node => node.getBoundingClientRect().width)).toBe(resizedWidth);
  await search.fill('');
  assert.equal(queries.length, beforeColumns);
  check('native hide/restore, all-hidden recovery and column resize are local; no pinning or unsupported query menus');

  const beforeDraft = queries.length;
  await search.fill('草稿甲');
  await expect(page.getByText('搜索词尚未应用', { exact: false })).toBeVisible();
  await open('查看日期…');
  await page.getByRole('dialog').locator('input[type=date]').fill('2026-10-10');
  await page.getByRole('button', { name: '取消', exact: true }).click();
  await expect(more).toBeFocused();
  assert.equal(queries.length, beforeDraft);
  await expect(search).toHaveValue('草稿甲');
  check('date local draft cancellation preserves A/D and trigger focus');

  await open('查看日期…');
  await page.getByRole('dialog').locator('input[type=date]').fill('');
  await expect(page.getByRole('button', { name: '应用', exact: true })).toBeDisabled();
  assert.equal(queries.length, beforeDraft);
  await page.getByRole('dialog').locator('input[type=date]').fill('2026-10-10');
  await page.getByRole('button', { name: '应用', exact: true }).click();
  await expect(page).toHaveURL(url => url.searchParams.get('q') === '草稿甲' && url.searchParams.get('businessAt') === '2026-10-10' && url.searchParams.get('page') === '1');
  await expect(page.getByRole('button', { name: '查看日期：2026-10-10' })).toBeVisible();
  await expect.poll(() => codeHeader.evaluate(node => node.getBoundingClientRect().width)).toBe(resizedWidth);
  await expect.poll(async () => (await linkedQuery()).get('q')).toBe('草稿甲');
  await expect.poll(async () => (await linkedQuery()).get('businessAt')).toBe('2026-10-10');
  check('incomplete date blocked; apply uses current D and preserves native column width');

  await search.fill('草稿乙');
  await expect.poll(async () => (await linkedQuery()).get('q')).toBe('草稿甲');
  let before = queries.length;
  await open('状态筛选…');
  await page.getByRole('combobox', { name: '状态' }).click();
  await page.getByRole('option', { name: '启用', exact: true }).click();
  await page.keyboard.press('Escape'); await expect(page.getByRole('menu')).toHaveCount(0);
  await expect(more).toBeFocused();
  assert.equal(queries.length, before);
  await expect(search).toHaveValue('草稿乙');
  await open('状态筛选…');
  await page.getByRole('combobox', { name: '状态' }).click();
  await page.getByRole('option', { name: '启用', exact: true }).click();
  await page.getByRole('button', { name: '应用', exact: true }).click();
  await expect(page).toHaveURL(url => url.searchParams.get('status') === 'active' && url.searchParams.get('q') === '草稿乙');
  await expect.poll(async () => (await linkedQuery()).get('q')).toBe('草稿乙');
  await expect.poll(async () => (await linkedQuery()).get('status')).toBe('active');
  check('status Escape cancellation and explicit application; stable cells carry current A only');

  await page.setViewportSize({ width: 390, height: 844 });
  await expect(page.getByRole('link', { name: '合成单元', exact: true })).toBeVisible();
  await expect(page.getByText('UI-001 · 启用 · 2026-10-01', { exact: true })).toBeVisible();
  await expect(page.getByRole('combobox', { name: '每页条数' })).toBeVisible();
  await expect(page.getByRole('columnheader')).toHaveCount(0);
  await page.screenshot({ path: path.join(out, 'summary-column.png'), fullPage: true });
  await open('排序：编码升序');
  await page.getByRole('combobox', { name: '排序方式' }).click();
  await page.getByRole('option', { name: '名称降序', exact: true }).click();
  await page.getByRole('button', { name: '应用', exact: true }).click();
  await expect(page).toHaveURL(url => url.searchParams.get('sort') === 'name_desc' && url.searchParams.get('page') === '1');
  check('narrow sorting uses server whitelist and applied URL');
  await open('排序：名称降序');
  await page.setViewportSize({ width: 1000, height: 900 });
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await expect(more).toBeFocused();
  check('breakpoint transition closes panel and restores an available trigger');

  await page.getByRole('button', { name: '查看日期：2026-10-10' }).click();
  await page.getByRole('button', { name: '今天', exact: true }).click();
  await expect(page).toHaveURL(url => url.searchParams.get('businessAt') === '2026-11-03');
  await expect(more).toBeFocused();
  check('server today and removed-summary focus fallback');

  failNext = true;
  await query.click();
  await expect(page.getByText('合成读取失败', { exact: true })).toBeVisible();
  await search.fill('失败后新草稿');
  await page.getByRole('button', { name: '重试', exact: true }).click();
  await expect(page.getByText('合成读取失败', { exact: true })).toHaveCount(0);
  await expect(search).toHaveValue('失败后新草稿');
  assert.equal(queries.at(-1).q, '草稿乙');
  check('retry uses A and preserves newer D');

  await search.fill('slow'); await query.click();
  await expect.poll(() => Boolean(delayed)).toBeTruthy();
  await search.fill('fast'); await query.click();
  await expect(page).toHaveURL(url => url.searchParams.get('q') === 'fast');
  delayed();
  await search.fill('请求后的新草稿');
  await expect(search).toHaveValue('请求后的新草稿');
  await expect(page.getByText('过期拒绝', { exact: true })).toHaveCount(0);
  check('late previous denial cannot clear current input or replace current error');

  denyNext = true; await query.click();
  await expect(page.getByText('当前无权读取', { exact: true })).toBeVisible();
  await expect(search).toHaveValue('');
  check('current authorization denial clears restricted search draft');
  reasonFailure = true;
  await page.getByRole('link', { name: '新建', exact: true }).click();
  await expect(page.getByText('合成原因目录失败', { exact: true })).toBeVisible();
  await page.getByRole('textbox', { name: '编码', exact: true }).fill('UI-NEW');
  await page.getByRole('textbox', { name: '中文名称', exact: true }).fill('可继续录入的合成名称');
  await expect(page.getByRole('button', { name: '保存', exact: true })).toBeDisabled();
  reasonFailure = false;
  await page.getByRole('button', { name: '重试加载', exact: true }).click();
  await expect(page.getByText('合成原因目录失败', { exact: true })).toHaveCount(0);
  await expect(page.getByRole('textbox', { name: '编码', exact: true })).toHaveValue('UI-NEW');
  await expect(page.getByRole('textbox', { name: '中文名称', exact: true })).toHaveValue('可继续录入的合成名称');
  check('reason directory failure permits independent input; retry retains it and missing reason still prevents save');
  await page.getByRole('link', { name: '取消', exact: true }).click();
  await page.getByRole('button', { name: '放弃并离开', exact: true }).click();
  canCreate = false; holdContext = true;
  await page.goto(origin + '/settings/organization/business-units');
  await expect.poll(() => Boolean(releaseContext)).toBeTruthy();
  await open('查看日期…');
  await expect(page.getByRole('button', { name: '今天', exact: true })).toBeDisabled();
  await page.getByRole('dialog').locator('input[type=date]').fill('2026-10-01');
  await page.getByRole('button', { name: '应用', exact: true }).click();
  await expect(page.getByRole('link', { name: '合成单元', exact: true })).toBeVisible();
  releaseContext();
  check('manual viewing date remains usable while current-date context is pending');
  await expect(page.getByRole('link', { name: '新建', exact: true })).toHaveAttribute('aria-disabled', 'true');
  check('read-only create remains unavailable');
  manyRows = true;
  await page.goto(origin + '/settings/organization/business-units');
  await expect(page.getByText('1–20 / 73', { exact: true })).toBeVisible();
  await search.fill('翻页未应用');
  await page.getByRole('button', { name: '下一页', exact: true }).click();
  await expect(page.getByText('21–40 / 73', { exact: true })).toBeVisible();
  await search.fill('第二页的新草稿');
  await page.getByRole('button', { name: '下一页', exact: true }).click();
  await expect(page.getByText('41–60 / 73', { exact: true })).toBeVisible();
  await expect(search).toHaveValue('第二页的新草稿');
  assert.equal(queries.at(-1).q ?? '', '');
  check('second/third server pages preserve A/D; input renders do not trigger native sort resets');
  manyRows = false;
  await page.goto(origin + '/settings/organization/business-units');
  await expect(page.getByRole('link', { name: '合成单元', exact: true })).toBeVisible();
  assert.deepEqual(writes, []);
  results.licenseMessages = [...results.warnings, ...results.consoleErrors].filter(text => /MUI X.*(license|Missing)/is.test(text));
  assert.deepEqual(results.licenseMessages, []);
  await expect(page.getByText(/Missing license key|Unlicensed product|MUI X Invalid license/)).toHaveCount(0);
  assert.deepEqual(results.warnings, []);
  assert.deepEqual(results.pageErrors, []);
  assert.deepEqual(results.externalRequests, []);
  assert(results.consoleErrors.every(text => /Failed to load resource.*(403|503)/.test(text)), JSON.stringify(results.consoleErrors));
  check('no writes, external requests or runtime exceptions; no license messages or watermarks');
  await page.screenshot({ path: path.join(out, 'desktop.png'), fullPage: true });
  results.status = 'passed';
} catch (error) {
  results.status = 'failed'; results.failure = error.stack;
  results.activeElement = await page?.evaluate(() => document.activeElement?.outerHTML);
  await page?.screenshot({ path: path.join(out, 'failure.png'), fullPage: true }); throw error;
} finally {
  releaseContext?.();
  await browser?.close();
  await new Promise(resolve => server.close(resolve));
  results.stopped = true;
  await writeFile(path.join(out, 'results.json'), JSON.stringify(results, null, 2) + '\n');
}
console.log(JSON.stringify({ status: results.status, checks: results.checks.length, scope: results.scope, stopped: results.stopped }));
