// Exercise normal page overlap without waiting for the first list to render.
// Only aggregate response metadata is saved; original source values stay in memory.
import { chromium, expect } from '@playwright/test';
import { writeFileSync } from 'node:fs';

const [destination] = process.argv.slice(2);
if (!destination) throw new Error('aggregate output path required');
const origin = 'http://127.0.0.1:4388', route = '/api/v1/departments';
const result = { origin, businessWrites: 0, cases: [], responses: [], pageErrors: 0, passed: false };
const browser = await chromium.launch({ headless: true });
try {
  for (const login of ['admin', 'reader']) {
    const context = await browser.newContext();
    const page = await context.newPage();
    page.setDefaultTimeout(20000);
    page.on('pageerror', () => result.pageErrors++);
    const pending = new Set();
    page.on('request', request => { if (new URL(request.url()).pathname.startsWith(route)) pending.add(request); });
    page.on('requestfinished', request => pending.delete(request));
    page.on('requestfailed', request => pending.delete(request));
    page.on('response', response => {
      const url = new URL(response.url());
      if (url.pathname.startsWith(route)) result.responses.push({ login, route: url.pathname, status: response.status(), queryApplied: url.searchParams.has('q') });
    });
    const item = { login, queriedBeforeInitialResponse: false, passed: false };
    result.cases.push(item);
    let stage = 'login';
    try {
      await page.goto(origin);
      await page.getByLabel('登录名').fill(login);
      await page.getByLabel('密码').fill('admin1234567');
      await page.getByRole('button', { name: '登录', exact: true }).click();
      await page.getByRole('button', { name: '切换应用', exact: true }).click();
      await page.getByRole('menuitem', { name: '设置', exact: true }).click();
      stage = 'initial-list';
      await page.getByRole('button', { name: '切换应用', exact: true }).click();
      await Promise.all([
        page.waitForRequest(r => new URL(r.url()).pathname === route),
        page.getByRole('menuitem', { name: '部门', exact: true }).click(),
      ]);
      const query = '__NO_SUCH_ORGANIZATION_127__';
      await page.getByPlaceholder('搜索编码或名称').fill(query);
      item.queriedBeforeInitialResponse = [...pending].some(r => new URL(r.url()).pathname === route && !new URL(r.url()).searchParams.has('q'));
      expect(item.queriedBeforeInitialResponse).toBe(true);
      stage = 'overlapping-search';
      const [search] = await Promise.all([
        page.waitForResponse(r => new URL(r.url()).pathname === route && new URL(r.url()).searchParams.get('q') === query),
        page.getByRole('button', { name: '查询', exact: true }).click(),
      ]);
      expect(search.status()).toBe(200);
      expect((await search.json()).count).toBe(0);
      stage = 'empty-result-page';
      await expect(page.locator('.MuiAlert-standardError')).toHaveCount(0);
      await page.getByPlaceholder('搜索编码或名称').fill('');
      stage = 'clear-search';
      const [all] = await Promise.all([
        page.waitForResponse(r => new URL(r.url()).pathname === route && !new URL(r.url()).searchParams.has('q')),
        page.getByRole('button', { name: '查询', exact: true }).click(),
      ]);
      expect(all.status()).toBe(200);
      const value = await all.json(); expect(value.count).toBe(87343);
      stage = 'rendered-full-list';
      await expect(page.getByRole('gridcell', { name: value.items[0].code, exact: true })).toBeVisible();
      stage = 'pending-reads-finish';
      await expect.poll(() => pending.size, { timeout: 15000 }).toBe(0);
      stage = 'no-visible-error';
      await expect(page.locator('.MuiAlert-standardError')).toHaveCount(0);
      item.passed = true;
    } catch (error) {
      item.errorType = error.name;
      item.failedStage = stage;
      item.failedLine = error.stack?.match(/browser_overlap\.mjs:(\d+)/)?.[1];
    } finally {
      await context.close();
    }
  }
  result.passed = result.cases.every(c => c.passed) && result.pageErrors === 0 && result.responses.every(r => r.status === 200);
} finally {
  await browser.close();
  writeFileSync(destination, JSON.stringify(result, null, 2) + '\n');
  console.log(JSON.stringify(result));
  if (!result.passed) process.exitCode = 1;
}
