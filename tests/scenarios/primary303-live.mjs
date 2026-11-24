// Read-only checks against the verified local profile; no business mutations.
import { chromium, expect as baseExpect } from '@playwright/test';
import { mkdirSync, writeFileSync, existsSync } from 'node:fs';
import { dirname } from 'node:path';
import { localConnection, verifyLocalTarget } from '../../scripts/local-runtime.mjs';

const expect = baseExpect.configure({ timeout: 20000 });
const output = process.argv[2];
if (!output || existsSync(output)) throw new Error('New evidence path required');
verifyLocalTarget();
const connection = localConnection();
const report = { profile: process.env.CHNHCM_LOCAL_PROFILE, origin: connection.webOrigin, cases: [], responses: [], readOnlyPosts: [], businessWrites: [], passed: false };
mkdirSync(dirname(output), { recursive: true });
const browser = await chromium.launch();
let lastPage;
try {
  for (const role of ['admin', 'reader']) {
    const context = await browser.newContext({ baseURL: connection.webOrigin, viewport: { width: 1440, height: 900 } });
    const page = await context.newPage();
    lastPage = page;
    page.on('response', response => { const path = new URL(response.url()).pathname; if (path.startsWith('/api/')) report.responses.push({ role, path, status: response.status() }); });
    page.setDefaultTimeout(20000);
    page.on('request', request => {
      const path = new URL(request.url()).pathname;
      if (request.method() === 'POST' && path === '/api/v1/departments/navigation') { report.readOnlyPosts.push({ method: request.method(), path }); return; }
      if (path.startsWith('/api/') && !['GET', 'HEAD'].includes(request.method()) && !path.startsWith('/api/v1/session')) report.businessWrites.push({ method: request.method(), path });
    });
    await page.goto('/');
    await page.getByLabel('登录名').fill(role);
    await page.getByLabel('密码').fill('admin1234567');
    await page.getByRole('button', { name: '登录', exact: true }).click();
    await expect(page.getByRole('heading', { level: 1, name: '业务单元', exact: true })).toBeVisible();
    const app = page.getByRole('button', { name: '切换应用', exact: true });
    await app.click();
    await expect(page.locator('#navigation-apps-menu').getByRole('menuitem')).toHaveText(['部门', '设置']);
    await page.screenshot({ path: output.replace(/\.json$/, `-${role}-menu.png`), mask: [page.locator('.app-account-name')] });
    const listed = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/departments' && r.status() === 200);
    await page.getByRole('menuitem', { name: '部门', exact: true }).click();
    const list = await (await listed).json();
    expect(list.count).toBe(87345);
    await expect(app).toHaveText('部门');
    const first = page.getByRole('grid', { name: '部门列表', exact: true }).getByRole('link').first();
    await first.click();
    await page.reload();
    await expect(app).toHaveText('部门');
    await page.getByRole('tab', { name: '变更日志', exact: true }).click();
    await page.getByRole('link', { name: '查看当次资料', exact: true }).first().click();
    await expect(page.getByText('正在查看只读历史资料', { exact: true })).toBeVisible();
    await expect(app).toHaveText('部门');
    report.cases.push({ role, check: 'desktop order, original department count, refreshed detail and historical source', count: list.count, passed: true });
    await app.click(); await page.getByRole('menuitem', { name: '设置', exact: true }).click();
    await page.getByRole('button', { name: '组织配置', exact: true }).click();
    await expect(page.locator('#navigation-section-menu').getByRole('menuitem')).toHaveText(['业务单元']);
    await page.keyboard.press('Escape');
    for (const [label, segment, count] of [['国家/地区', 'country-regions', 251], ['行政区划', 'geo-areas', 3630]]) {
      await page.getByRole('button', { name: '基础资料', exact: true }).click();
      const response = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/' + segment && r.status() === 200);
      await page.getByRole('menuitem', { name: label, exact: true }).click();
      expect((await (await response).json()).count).toBe(count);
      await expect(page.getByRole('grid').getByRole('link').first()).toBeVisible();
      await expect(app).toHaveText('设置');
      report.cases.push({ role, check: label, count, passed: true });
    }
    for (const width of [390, 320]) {
      await page.setViewportSize({ width, height: 844 });
      await page.getByRole('button', { name: '打开导航', exact: true }).click();
      const drawer = page.locator('#navigation-drawer');
      await expect(drawer.getByRole('button').filter({ hasText: /^(部门|设置)$/ })).toHaveText(['部门', '设置']);
      await expect(drawer.getByRole('button', { name: '部门', exact: true })).not.toHaveAttribute('aria-expanded');
      await drawer.getByRole('button', { name: '部门', exact: true }).click();
      await expect(drawer).not.toBeVisible();
      await expect(page.getByRole('heading', { level: 1, name: '部门', exact: true })).toBeVisible();
      await expect(page.getByRole('grid').getByRole('link').first()).toBeVisible();
      if (role === 'reader') await expect(page.getByRole('link', { name: '新建', exact: true })).toHaveAttribute('aria-disabled', 'true');
      await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
      report.cases.push({ role, check: 'drawer direct application', width, passed: true });
    }
    await context.close();
  }
  expect(report.businessWrites).toEqual([]);
  report.passed = true;
} catch (error) {
  await lastPage?.screenshot({ path: output.replace(/\.json$/, '-failure.png'), mask: [lastPage.locator('.app-account-name'), lastPage.getByLabel('登录名'), lastPage.getByLabel('密码')] }).catch(() => {});
  throw error;
} finally {
  writeFileSync(output, JSON.stringify(report, null, 2) + '\n');
  await browser.close();
}
