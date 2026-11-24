// Real source names, IDs and screenshots remain in the external private directory.
import { chromium, expect } from '@playwright/test';
import { spawn } from 'node:child_process';
import { createInterface } from 'node:readline';
import { readFileSync, writeFileSync, appendFileSync, createWriteStream } from 'node:fs';
import { resolve } from 'node:path';

const [directory, destination, fixedOrigin] = process.argv.slice(2);
if (!directory || !destination) throw new Error('private directory and aggregate output required');
if (fixedOrigin && !['http://127.0.0.1:4388', 'http://127.0.0.1:4383'].includes(fixedOrigin)) throw new Error('unexpected fixed verification origin');
const plan = JSON.parse(readFileSync(resolve(directory, 'browser-plan.json')));
const role = process.env.CHNHCM_VANKE_ROLE || 'admin';
if (!['admin', 'reader'].includes(role)) throw new Error('unknown verification role');
const result = { scope: 'complete V127 dataset, normal runtime and normal menus', role, fixedOrigin, preserveBusinessFacts: Boolean(fixedOrigin) || role === 'reader', cases: [], apiResponses: [], pageErrors: 0, passed: false };
const log = createWriteStream(resolve(directory, 'browser-server.log'), { flags: 'a', mode: 0o600 });
const server = fixedOrigin ? null : spawn(process.env.CHNHCM_VANKE_BINARY || '.local/vanke-127/vanke127', ['-mode', 'serve', '-private', directory], { stdio: ['ignore', 'pipe', 'pipe'] });
server?.stderr.pipe(log);
const lines = server ? createInterface({ input: server.stdout }) : null;
let browser, page, stage = 'startup';
const save = () => writeFileSync(destination, JSON.stringify(result, null, 2) + '\n');
const screenshot = async label => page.screenshot({ path: resolve(directory, role + '-' + label + '.png'), mask: [page.locator('header'), page.locator('input[type=password]')] }).catch(() => {});
try {
  const ready = fixedOrigin ? { url: fixedOrigin } : await Promise.race([
    new Promise((done, fail) => { lines.once('line', value => done(JSON.parse(value))); server.once('exit', () => fail(new Error('startup failed'))); }),
    new Promise((_, fail) => setTimeout(() => fail(new Error('startup timeout')), 15000).unref()),
  ]);
  browser = await chromium.launch({ headless: true });
  page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
  page.setDefaultTimeout(20000);
  page.on('pageerror', () => result.pageErrors++);
  page.on('response', response => {
    const url = new URL(response.url());
    if (url.pathname.startsWith('/api/')) result.apiResponses.push({
      route: url.pathname.replace(/[0-9a-f]{8}-[0-9a-f-]{27,}/g, '{id}'),
      queryApplied: url.searchParams.has('q'), status: response.status(),
      responseStartMS: response.request().timing().responseStart,
    });
  });
  stage = 'login';
  await page.goto(ready.url);
  await page.getByLabel('登录名').fill(role);
  await page.getByLabel('密码').fill('admin1234567');
  await page.getByRole('button', { name: '登录', exact: true }).click();
  await page.getByRole('button', { name: '切换应用', exact: true }).click();
  await page.getByRole('menuitem', { name: '设置', exact: true }).click();
  for (const item of plan) {
    const entry = { label: item.label, passed: false }, begin = performance.now();
    result.cases.push(entry);
    try {
      const label = item.kind === 'business-units' ? '业务单元' : '部门', api = '/api/v1/' + item.kind;
      stage = 'normal-menu';
      await page.getByRole('button', { name: '切换应用', exact: true }).click();
      await page.getByRole('menuitem', { name: label === '部门' ? '部门' : '设置', exact: true }).click();
      await expect(page.getByRole('heading', { name: label, exact: true })).toBeVisible();
      // Selecting the current/default menu may reuse its existing query. Reload
      // the selected normal route to observe a real request in every case.
      const [response] = await Promise.all([page.waitForResponse(r => r.request().method() === 'GET' && new URL(r.url()).pathname === api), page.reload()]);
      entry.listStatus = response.status(); expect(response.status()).toBe(200);
      const list = await response.json(); expect(list.count).toBe(item.count); entry.listCount = list.count;
      await expect(page.getByRole('heading', { name: label, exact: true })).toBeVisible();
      // Development StrictMode may discard the first response and issue another
      // owned read. A response alone does not prove the visible table is ready.
      await expect(page.getByRole('gridcell', { name: list.items[0].code, exact: true })).toBeVisible();
      await expect(page.locator('.MuiAlert-standardError')).toHaveCount(0);
      stage = 'search';
      await page.getByPlaceholder('搜索编码或名称').fill(item.code);
      const [searchResponse] = await Promise.all([
        page.waitForResponse(r => r.request().method() === 'GET' && new URL(r.url()).pathname === api && new URL(r.url()).searchParams.get('q') === item.code),
        page.getByRole('button', { name: '查询', exact: true }).click(),
      ]);
      entry.searchStatus = searchResponse.status(); expect(searchResponse.status()).toBe(200);
      stage = 'detail';
      const row = page.getByRole('row').filter({ has: page.getByRole('gridcell', { name: item.code, exact: true }) });
      const [responseDetail] = await Promise.all([page.waitForResponse(r => r.request().method() === 'GET' && new URL(r.url()).pathname === api + '/' + item.id), row.getByRole('link').click()]);
      entry.detailStatus = responseDetail.status(); expect(responseDetail.status()).toBe(200);
      const value = await responseDetail.json(); expect(value.timeline.length).toBe(item.recordCount);
      const selected = item.records.find(r => r.id === value.selectedRecord.recordId);
      expect(value.selectedRecord.name).toBe(selected.name);
      await expect(page.locator('.object-detail-columns')).toBeVisible();
      if (item.kind === 'departments') {
        await expect(page.locator('.department-path')).toContainText(selected.name);
        const relations=page.locator('.object-detail-section').filter({has:page.getByRole('heading',{name:'组织关系',exact:true})});
        await expect(relations.getByRole('link')).toHaveCount(0);
      }
      await screenshot(item.label + '-detail');
      stage = 'exact-source';
      const [oldResponse] = await Promise.all([page.waitForResponse(r => new URL(r.url()).pathname === api + '/' + item.id && new URL(r.url()).searchParams.get('sourceVersion') === item.originalSource), page.goto(ready.url + '/settings/organization/' + item.kind + '/' + item.id + '?sourceVersion=' + item.originalSource)]);
      expect(oldResponse.status()).toBe(200); const old = await oldResponse.json();
      expect(old.resolvedSourceVersion).toBe(item.originalSource); expect(old.readOnly).toBe(true);
      expect(old.timeline.length).toBe(1); expect(old.selectedRecord.name).toBe(item.originalName);
      await expect(page.getByText('正在查看只读历史资料', { exact: true })).toBeVisible();
      if (item.kind === 'departments') await expect(page.locator('.department-path')).toContainText(item.originalName);
      entry.originalSourceAndPath = true;
      await screenshot(item.label + '-original');
      if (item.edit && !fixedOrigin && role === 'admin') {
        stage = 'ordinary-edit';
        await page.getByRole('button', { name: '返回当前资料', exact: true }).click();
        await page.getByRole('button', { name: '更多操作', exact: true }).click();
        await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
        await expect(page.getByRole('textbox', { name: '部门名称', exact: true })).toHaveValue(item.originalName);
        await page.getByRole('textbox', { name: '简称', exact: true }).fill('历史原名页面核验');
        const [saved] = await Promise.all([page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname.endsWith('/correct')), page.getByRole('button', { name: '保存', exact: true }).click()]);
        entry.saveStatus = saved.status(); expect(saved.status()).toBe(200);
        const changes = saved.request().postDataJSON().changes;
        expect(changes).not.toHaveProperty('name'); expect(changes.shortName).toBe('历史原名页面核验');
        expect((await saved.json()).outcome).toBe('accepted');
        await expect(page.locator('.object-form')).toHaveCount(0);
        await expect(page.locator('.object-detail-columns')).toContainText('历史原名页面核验');
        await expect(page.locator('.department-path')).toContainText(item.originalName);
        entry.ordinaryUnrelatedEditPreservedHistoricalName = true;
        await screenshot(item.label + '-edited');
      }
      entry.passed = true;
    } catch (error) {
      entry.failedStage = stage; entry.errorType = error.name;
      appendFileSync(resolve(directory, 'browser-errors.log'), item.label + ' ' + stage + '\n' + error.stack + '\n', { mode: 0o600 });
      // Error messages and page snapshots can contain original source values.
      await screenshot(item.label + '-failure');
    }
    entry.milliseconds = performance.now() - begin; save();
  }
  result.passed = result.cases.length === plan.length && result.cases.every(c => c.passed) && result.pageErrors === 0;
} catch (error) {
  result.failure = { stage, errorType: error.name };
  appendFileSync(resolve(directory, 'browser-errors.log'), stage + '\n' + error.stack + '\n', { mode: 0o600 });
  if (page) await screenshot('browser-failure');
} finally {
  if (browser) await browser.close();
  lines?.close();
  if (server) {
    server.kill('SIGTERM');
    await new Promise(done => server.exitCode !== null ? done() : server.once('exit', done));
  }
  log.end(); save(); console.log(JSON.stringify(result));
  if (!result.passed) process.exitCode = 1;
}
