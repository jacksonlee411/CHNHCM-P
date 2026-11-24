// Complete-case UI verification; no cookies, traces or private config are saved.
import { chromium, expect } from '@playwright/test';
import { spawn } from 'node:child_process';
import { createHash } from 'node:crypto';
import { createInterface } from 'node:readline';
import { readFileSync, writeFileSync, createWriteStream } from 'node:fs';
import { resolve, dirname } from 'node:path';

const [directory, destination] = process.argv.slice(2);
if (!directory || !destination) throw new Error('external case directory and output JSON required');
const result = { scope: 'original complete case through normal menus; one browser session', cases: [], failures: [] };
const log = createWriteStream(resolve(dirname(destination), 'case-browser-server.log'), { flags: 'a' });
const server = spawn('.local/storage-126-implementation/storage126', ['-mode', 'serve-web', '-input', directory], { stdio: ['ignore', 'pipe', 'pipe'] });
server.stderr.pipe(log);
const lines = createInterface({ input: server.stdout });
let browser, page;
try {
  const ready = await Promise.race([
    new Promise((done, fail) => { lines.once('line', value => done(JSON.parse(value))); server.once('exit', () => fail(new Error('case server startup failed'))); }),
    new Promise((_, fail) => setTimeout(() => fail(new Error('case server startup timeout')), 15000).unref()),
  ]);
  browser = await chromium.launch({ headless: true });
  page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
  page.setDefaultTimeout(20000);
  page.on('pageerror', error => result.failures.push({ pageError: error.message }));
  await page.goto(ready.url);
  await page.getByLabel('登录名').fill('admin');
  await page.getByLabel('密码').fill('admin1234567');
  await page.getByRole('button', { name: '登录', exact: true }).click();
  await page.getByRole('button', { name: '切换应用', exact: true }).click();
  await page.getByRole('menuitem', { name: '设置', exact: true }).click();
  const owners = JSON.parse(readFileSync(resolve(directory, 'owners.json')));
  for (const [kind, label, route, code, count] of [
    ['bu', '业务单元', 'business-units', 'R6_B_00000', 2023],
    ['dep', '部门', 'departments', 'R6_D_00000', 22104],
    ['dep', '部门', 'departments', 'R6_DEPTH_099', 22104],
  ]) {
    const begin = performance.now();
    await page.getByRole('button', { name: '切换应用', exact: true }).click();
    const [response] = await Promise.all([page.waitForResponse(r => r.request().method() === 'GET' && new URL(r.url()).pathname === '/api/v1/' + route), page.getByRole('menuitem', { name: label === '部门' ? '部门' : '设置', exact: true }).click()]);
    expect(response.status()).toBe(200);
    const list = await response.json();
    expect(list.count).toBe(count);
    await expect(page.getByRole('heading', { name: label, exact: true })).toBeVisible();
    await page.getByPlaceholder('搜索编码或名称').fill(code);
    await page.getByRole('button', { name: '查询', exact: true }).click();
    const target = owners[kind].find(o => o.code === code);
    const id = target[kind === 'bu' ? 'bu_id' : 'department_id'];
    const [detail] = await Promise.all([page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/' + route + '/' + id), page.locator(`a[href^="/settings/organization/${route}/${id}"]`).first().click()]);
    expect(detail.status()).toBe(200);
    const value = await detail.json();
    expect(value.headSourceVersion).toBe(target.head_source_id);
    await expect(page.locator('.object-detail-columns')).toBeVisible();
    if (kind === 'dep') await expect(page.locator('.department-path')).toBeVisible();
    result.cases.push({ code, normalMenu: true, count, head: value.headSourceVersion, timeline: value.timeline.length, milliseconds: performance.now() - begin });
    await page.screenshot({ path: resolve(dirname(destination), code + '.png'), mask: [page.locator('header')] });
    if (code === 'R6_D_00000') {
      const prototype = JSON.parse(readFileSync(resolve(directory, 'dep-prototype.json')));
      const raw = createHash('md5').update(code + ':' + prototype.Frames[0].Operation.Source.ID).digest();
      raw[6] = (raw[6] & 15) | 0x30; raw[8] = (raw[8] & 63) | 0x80;
      const h = raw.toString('hex'), source = `${h.slice(0, 8)}-${h.slice(8, 12)}-${h.slice(12, 16)}-${h.slice(16, 20)}-${h.slice(20)}`;
      const [old] = await Promise.all([page.waitForResponse(r => new URL(r.url()).pathname.startsWith('/api/') && new URL(r.url()).searchParams.get('sourceVersion') === source), page.goto(ready.url + '/settings/organization/departments/' + id + '?sourceVersion=' + source)]);
      expect(old.status()).toBe(200);
      const historical = await old.json();
      expect(historical.resolvedSourceVersion).toBe(source);
      expect(historical.readOnly).toBe(true);
      expect(historical.timeline.length).toBe(1);
      await expect(page.locator('.department-path')).toBeVisible();
      result.cases.push({ code, explicitSource: source, readOnly: true, timeline: 1 });
      await page.screenshot({ path: resolve(dirname(destination), 'department-original-source.png'), mask: [page.locator('header')] });
    }
  }
  result.passed = result.failures.length === 0;
} catch (error) {
  result.passed = false;
  result.failures.push({ error: String(error) });
  if (page) await page.screenshot({ path: resolve(dirname(destination), 'case-browser-failure.png'), mask: [page.locator('header'), page.locator('input[type=password]'), page.getByLabel('登录名')] }).catch(() => {});
  process.exitCode = 1;
} finally {
  if (browser) await browser.close();
  lines.close();
  server.kill('SIGTERM');
  await new Promise(done => server.exitCode !== null ? done() : server.once('exit', done));
  log.end();
  writeFileSync(destination, JSON.stringify(result, null, 2) + '\n');
  console.log(JSON.stringify({ passed: result.passed, cases: result.cases.length, failures: result.failures }));
}
