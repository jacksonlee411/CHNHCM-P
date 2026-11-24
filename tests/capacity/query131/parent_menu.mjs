import {chromium, expect} from '@playwright/test';

let raw = '';
for await (const chunk of process.stdin) raw += chunk;
const input = JSON.parse(raw), origin = new URL(input.url);
if (origin.protocol !== 'http:' || origin.hostname !== '127.0.0.1') throw Error('owned loopback runtime required');
if (!Number.isInteger(input.count) || input.count <= 20 || input.pages.length !== 2) throw Error('independent first and second pages required');
const browser = await chromium.launch({headless: true});
const result = {passed: false, role: 'admin', requests: [], pageErrors: 0};
try {
  const context = await browser.newContext({baseURL: input.url, viewport: {width: 1440, height: 900}});
  const page = await context.newPage();
  page.on('pageerror', () => result.pageErrors++);
  // Observe request initiation, so an unfinished speculative request cannot
  // evade the no-prewarm check merely because its response has not arrived.
  page.on('request', request => {
    if (new URL(request.url()).pathname !== '/api/v1/departments/references/parents') return;
    const body = request.postDataJSON();
    result.requests.push({page: body.page, searchEmpty: !body.q, mode: body.context?.mode});
  });
  await page.goto('/');
  await page.getByLabel('登录名').fill('admin');
  await page.getByLabel('密码').fill('admin1234567');
  await page.getByRole('button', {name: '登录', exact: true}).click();
  await page.getByRole('button', {name: '切换应用', exact: true}).click();
  await page.getByRole('menuitem', {name: '设置', exact: true}).click();
  await page.getByRole('button', {name: '切换应用', exact: true}).click();
  const listing = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/departments' && r.request().method() === 'GET');
  await page.getByRole('menuitem', {name: '部门', exact: true}).click();
  expect((await listing).status()).toBe(200);
  await page.getByRole('link', {name: '新建', exact: true}).click();
  await expect(page.getByRole('heading', {name: '新建部门', exact: true})).toBeVisible();
  await expect(page.locator('#department-effectiveDate')).toHaveValue(input.day);
  const bu = page.getByRole('combobox', {name: '管理业务单元', exact: true});
  await bu.click();
  await bu.fill(input.buCode);
  const option = page.getByRole('option').filter({hasText: input.buCode});
  await expect(option).toHaveCount(1);
  await option.click();
  expect(result.requests).toHaveLength(0);
  const field = page.getByRole('combobox', {name: '行政上级', exact: true});
  await expect(field).toHaveValue('');
  const parentResponse = pageNumber => page.waitForResponse(r => {
    if (new URL(r.url()).pathname !== '/api/v1/departments/references/parents') return false;
    const body = r.request().postDataJSON();
    return !body.q && body.page === pageNumber && body.context?.mode === 'create';
  });
  const first = parentResponse(1), started = performance.now();
  await field.click();
  const a = await first;
  result.firstStatus = a.status();
  result.firstResponseMS = performance.now() - started;
  expect(a.status()).toBe(200);
  const firstBody = await a.json();
  expect(firstBody.count).toBe(input.count);
  expect(firstBody.items).toEqual(input.pages[0]);
  await expect(page.getByRole('option')).toHaveCount(input.pages[0].length);
  result.firstOpenMS = performance.now() - started;
  const second = parentResponse(2);
  await page.getByRole('button', {name: '下一页候选', exact: true}).click();
  const b = await second;
  result.secondStatus = b.status();
  expect(b.status()).toBe(200);
  const secondBody = await b.json();
  expect(secondBody.count).toBe(input.count);
  expect(secondBody.items).toEqual(input.pages[1]);
  await page.getByRole('option').filter({hasText: input.pages[1][0].code}).click();
  await expect(field).toHaveValue(new RegExp(input.pages[1][0].code));
  await page.getByRole('textbox', {name: '部门名称', exact: true}).fill('131未保存候选草稿');
  await page.getByRole('link', {name: '取消', exact: true}).click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.getByRole('button', {name: '放弃并离开', exact: true}).click();
  await expect(page.getByRole('heading', {name: '部门', exact: true})).toBeVisible();
  expect(result.pageErrors).toBe(0);
  result.passed = true;
  result.noParentWarmup = true;
  result.count = input.count;
  result.independentPagesCompared = 2;
  result.selectedSecondPageAndUnsavedProtection = true;
} catch (error) {
  result.failure = {errorType: error.name};
  // The Python owner retains stderr privately; no DOM values enter the report.
  process.stderr.write(String(error.stack || error) + '\n');
} finally {
  await browser.close();
}
process.stdout.write(JSON.stringify(result) + '\n');
if (!result.passed) process.exitCode = 1;
