import { desktopFormViewports, desktopViewports } from './desktop-viewports.mjs';
import {openPersonnel as menu} from './app-navigation';
import { randomUUID } from 'node:crypto';
import { execFileSync, spawn } from 'node:child_process';
import { existsSync, readFileSync } from 'node:fs';
import { chromium, type Request } from '@playwright/test';
import { test, expect, type Page } from './fixtures';
import { click308, menu308, reach308, reference308 } from './staffing308-support';
import { localConnection } from '../../scripts/local-runtime.mjs';
import { request } from './dated-object-support';

import { nativeZoomBrowser } from './native-zoom-support';

const cohort = '名录翻页' + randomUUID().slice(0, 8);
const referenceCode = 'LOG-' + randomUUID().slice(0, 8);
test.beforeAll(async ({ playwright }) => {
  const origin = localConnection().webOrigin,
    client = await playwright.request.newContext({ baseURL: origin, extraHTTPHeaders: { Origin: origin } });
  try {
    const login = await client.post('/api/v1/session/login', {
      data: { login: 'admin', password: 'admin1234567' },
    });
    expect(login.status()).toBe(200);
    const token = (await login.json()).csrf;
    const post = async (route: string, data: unknown) => {
      const response = await client.post('/api/v1/persons' + route, {
        headers: { 'X-CSRF-Token': token },
        data,
      });
      expect(response.status()).toBe(200);
      return response.json();
    };
    async function createReference(path: string, reason: string, extra: Record<string, string> = {}) {
      const response = await client.post('/api/v1/' + path, {
        headers: { 'X-CSRF-Token': token, 'Idempotency-Key': randomUUID() },
        data: {
          code: referenceCode, name: referenceCode, effectiveDate: '1900-01-01',
          operationType: 'CREATE', operationReason: reason, ...extra,
        },
      });
      expect(response.status()).toBe(201);
      return response.json();
    }
    await createReference('business-units', 'NEW_ORGANIZATION');
    const family = await createReference('job-families', 'NEW_JOB_FAMILY');
    const jobFunction = await createReference('job-functions', 'NEW_JOB_FUNCTION');
    await createReference('jobs', 'NEW_JOB', { jobFamilyId: family.objectId, jobFunctionId: jobFunction.objectId });

    for (let i = 0; i < 21; i++) {
      const identity = {
        name: cohort,
        evidence: { kind: 'form', reference: 'M-' + i + '-' + randomUUID().slice(0, 6) },
      };
      await post('', {
        ...identity,
        submissionId: randomUUID(),
        contacts: [
          { kind: 'phone', value: '10000001' },
          { kind: 'email', value: 'demo@example.test' },
        ],
      });
    }
  } finally {
    await client.dispose();
  }
});
async function login(page: Page, role: string) {
  await page.goto('/');
  await page.getByLabel('登录名').fill(role);
  await page.getByLabel('密码').fill('admin1234567');
  await page.getByRole('button', { name: '登录', exact: true }).click();
  await expect(page.getByRole('button', { name: '退出', exact: true })).toBeVisible();
}
async function select(page: Page, label: string, choice: string) {
  await page.getByRole('combobox', { name: label, exact: true }).click();
  await page.getByRole('option', { name: choice, exact: true }).click();
}
async function search(page: Page, value: string) {
  await page.getByLabel('姓名或人员编号').fill(value);
  const response = page.waitForResponse(
    (r) =>
      r.request().method() === 'POST' &&
      new URL(r.url()).pathname === '/api/v1/persons/queries' &&
      r.request().postDataJSON()?.keyword === value,
  );
  await page.getByLabel('姓名或人员编号').press('Enter');
  const result = await response;
  expect(result.status()).toBe(200);
  const { ref } = await result.json();
  await expect(page.locator('a[data-object-link]').first()).toHaveAttribute(
    'href',
    new RegExp('[?&]ref=' + ref + '(?:&|$)'),
  );
}
async function measureField(page: Page, label: string) {
  const field = page.getByRole('textbox', { name: label, exact: true });
  await reach308(page, field);
  return field.evaluate((el) => ({
    label: getComputedStyle(document.querySelector('.object-form .object-field-label')!).fontSize,
    value: getComputedStyle(el).fontSize,
    height: el.closest('.MuiInputBase-root')!.getBoundingClientRect().height,
    columns: getComputedStyle(document.querySelector('.object-detail-columns')!).gridTemplateColumns.split(
      ' ',
    ).length,
  }));
}
async function logLayout(page: Page) {
  const selected = page.getByRole('tab', { name: '变更日志', selected: true });
  const first = page.locator('.change-operation').first();
  await expect(first.getByRole('link', { name: '查看当次资料', exact: true })).toBeVisible();
  await expect(first.getByRole('button', { name: '来源明细', exact: true })).toHaveAttribute('aria-expanded', 'false');
  const tabs = await selected.evaluate((el) => {
    const tab = getComputedStyle(el);
    const indicator = getComputedStyle(el.closest('.MuiTabs-root')!.querySelector('.MuiTabs-indicator')!);
    return { minHeight: tab.minHeight, border: tab.borderTopStyle, borderColor: tab.borderTopColor,
      indicatorHeight: indicator.height, indicatorTop: indicator.top };
  });
  const row = await first.locator('tr').first().evaluate((el) => {
    const cell = el.querySelector('[data-label="原值"]')!;
    return { display: getComputedStyle(el).display, labels: getComputedStyle(cell, '::before').content !== 'none' };
  });
  return { ...tabs, ...row };
}

async function referenceLog(page: Page) {
  await page.getByRole('textbox', { name: '搜索编码或名称', exact: true }).fill(referenceCode);
  await click308(page, page.getByRole('button', { name: '查询', exact: true }));
  await click308(page, page.getByRole('link', { name: referenceCode, exact: true }).first());
  await click308(page, page.getByRole('tab', { name: '变更日志', exact: true }));
  return logLayout(page);
}

async function readPersonSummary(page: Page) {
  const summary = page.locator('.operation-summary');
  await expect(summary.getByText('操作者：admin', { exact: true })).toBeVisible();
  await expect(summary).toContainText(/操作时间：\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}（北京时间 UTC\+08:00）/);
  expect(await summary.evaluate(el => el.scrollWidth <= el.clientWidth + 1)).toBe(true);
  return summary.locator('time').getAttribute('datetime');
}

async function openPersonLog(page: Page, activate = 'Enter') {
  const tabs = page.getByRole('tablist');
  const profile = tabs.getByRole('tab', { name: '个人资料', exact: true });
  const history = tabs.getByRole('tab', { name: '变更日志', exact: true });
  await expect(tabs.getByRole('tab')).toHaveCount(2);
  await expect(profile).toHaveAttribute('aria-selected', 'true');
  const current = page.getByRole('tabpanel', { name: '个人资料', exact: true });
  await expect(current).toHaveCount(1);
  expect(await current.getAttribute('id')).toBe(await profile.getAttribute('aria-controls'));
  await click308(page, profile);
  await page.keyboard.press('ArrowRight');
  await expect(history).toBeFocused();
  await expect(profile).toHaveAttribute('aria-selected', 'true');
  await page.keyboard.press(activate);
  const panel = page.getByRole('tabpanel', { name: '变更日志', exact: true });
  await expect(history).toHaveAttribute('aria-selected', 'true');
  await expect(profile).toHaveAttribute('aria-selected', 'false');
  await expect(page.getByRole('tabpanel')).toHaveCount(1);
  await expect(panel).toHaveAttribute('aria-labelledby', (await history.getAttribute('id'))!);
  expect(await panel.getAttribute('id')).toBe(await history.getAttribute('aria-controls'));
  await expect(panel.getByRole('link', { name: '查看当次资料', exact: true }).first()).toBeVisible();
  await expect(history).toBeFocused();
  return panel;
}

async function journey(page: Page, role: string, relogin: boolean) {
  page.setDefaultTimeout(15000);
  await reference308(page, role);
  const buLayout = await referenceLog(page);
  await menu308(page, 'job');
  let baseline;
  if (role === 'admin') {
    await click308(page, page.getByRole('link', { name: '新建', exact: true }));
    baseline = await measureField(page, '编码');
    await click308(
      page,
      page
        .getByRole('button', { name: '取消', exact: true })
        .or(page.getByRole('link', { name: '取消', exact: true })),
    );
  }
  const jobLayout = await referenceLog(page);
  expect(jobLayout).toEqual(buLayout);
  expect(jobLayout).toMatchObject({ minHeight: '42px', indicatorHeight: '3px', indicatorTop: '0px' });
  const dimensions = await page.evaluate(() => ({
    width: innerWidth,
    height: innerHeight,
    zoom: devicePixelRatio,
    theme: getComputedStyle(document.querySelector('header')!).backgroundColor,
  }));
  test.info().annotations.push({ type: 'viewport', description: JSON.stringify(dimensions) });
  await menu(page);
  const theme = await page.locator('header').evaluate((el) => getComputedStyle(el).backgroundColor);
  expect(theme).toBe(dimensions.theme);
  console.info('T01 视口', JSON.stringify({ role, ...dimensions, field: baseline }));
  const btn = (label: string) => page.getByRole('button', { name: label, exact: true });
  if (role === 'admin') {
    let registrations = 0;
    const countRegistration = (r: import('@playwright/test').Request) => {
      if (r.method() === 'POST' && new URL(r.url()).pathname === '/api/v1/persons') registrations++;
    };
    page.on('request', countRegistration);
    await page.getByRole('link', { name: '登记人员', exact: true }).click();
    expect(await measureField(page, '姓名')).toEqual(baseline);
    await btn('保存').click();
    await expect(page.locator('#person-error')).toBeFocused();
    await page.getByRole('textbox', { name: '姓名', exact: true }).fill('取消用样本');
    await btn('取消').click();
    await expect(btn('继续编辑')).toBeVisible();
    await page.keyboard.press('Escape');
    await btn('取消').click();
    await btn('放弃修改').click();
    expect(registrations).toBe(0);
    page.off('request', countRegistration);
    await page.getByRole('link', { name: '登记人员', exact: true }).click();
    const key = randomUUID().slice(0, 8),
      name = '验收人员' + key;
    await page.getByRole('textbox', { name: '姓名', exact: true }).fill(name);
    await page.getByLabel('登记单号（选填）').fill('UI-1' + key);
    await select(page, '联系类型 1', '电话');
    await page.getByLabel('联系值 1').fill('10000001');
    await btn('添加联系方式').click();
    await select(page, '联系类型 2', '电子邮箱');
    await page.getByLabel('联系值 2').fill('demo@example.test');
    await page.keyboard.press('Tab');
    await page.keyboard.press('Shift+Tab');
    await page.mouse.wheel(0, 240);
    await expect(page.getByText('正在核对候选人员')).toHaveCount(0);
    if (relogin) {
      await page.route(
        '**/api/v1/persons',
        async (route) => {
          if (route.request().method() !== 'POST') {
            await route.continue();
            return;
          }
          const response = await route.fetch();
          expect(response.status()).toBe(200);
          await route.abort('failed');
        },
        { times: 1 },
      );
    }
    await btn('保存').click();
    await expect(btn('更正资料')).toBeVisible();
    const number = (await page.locator('.object-identity-summary').innerText()).trim();
    expect(number).toMatch(/^P\d{8}$/);
    if (relogin) {
      await Promise.all([
        page.waitForResponse(
          (r) => r.request().method() === 'DELETE' && r.url().endsWith('/session') && r.status() === 204,
        ),
        btn('退出').click(),
      ]);
      await expect(page.getByLabel('登录名')).toBeVisible();
      await login(page, 'admin');
      await menu(page);
    } else {
      await page.getByRole('link', { name: '返回列表', exact: true }).click();
    }
    await search(page, number);
    await page.getByRole('link', { name, exact: true }).click();
    await btn('更正资料').click();
    await page.getByRole('group',{name:'联系方式',exact:true}).getByRole('button',{name:'编辑'}).first().click();
    await page.getByRole('dialog').getByRole('textbox', { name: '电话', exact: true }).fill('10000002');
    await page.getByRole('dialog').getByRole('button',{name:'应用'}).click();
    await btn('保存').click();
    await expect(page.getByText('10000002', { exact: true })).toBeVisible();
    await expect(page.getByText('demo@example.test', { exact: true })).toBeVisible();
  } else {
    await expect(page.getByRole('link', { name: '登记人员', exact: true })).toHaveCount(0);
    await search(page, cohort);
    await page.getByRole('link', { name: cohort, exact: true }).first().click();
    await expect(btn('更正资料')).toHaveCount(0);
  }
  const currentTime = await readPersonSummary(page);
  const panel = await openPersonLog(page);
  await expect(page.locator('.operation-summary')).toHaveCount(0);
  const logTimes = panel.locator('[data-label="记录时间"] time');
  await expect(logTimes.first()).toHaveAttribute('datetime', currentTime!);
  const personLayout = await logLayout(page);
  expect(personLayout).toEqual(jobLayout);
  expect(personLayout).toMatchObject(dimensions.width < 900 ? { display: 'block', labels: true } : { display: 'table-row' });
  await expect(panel.locator('[data-label="生效记录"]')).toHaveCount(0);
  const originalLink = panel.getByRole('link', { name: '查看当次资料', exact: true }).last();
  const originalTime = await logTimes.last().getAttribute('datetime');
  if (role === 'admin') expect(originalTime).not.toBe(currentTime);
  const originalHref = await originalLink.getAttribute('href');
  expect(new URL(originalHref!, page.url()).searchParams.get('source')).toMatch(/^[a-f0-9-]{36}$/);
  await click308(page, originalLink);
  await expect(page.getByText('10000001', { exact: true })).toBeVisible();
  expect(await readPersonSummary(page)).toBe(originalTime);
  await expect(page.getByText('demo@example.test', { exact: true })).toBeVisible();
  await btn('返回当前资料').click();
  await expect(originalLink).toHaveAttribute('href', originalHref!);
  await expect(originalLink).toBeFocused();
  await page.getByRole('link', { name: '返回列表', exact: true }).click();
  await expect(page.locator('a[data-object-link]:focus')).toBeVisible();
  await search(page, cohort);
  const grid = page.getByRole('grid');
  await expect(grid.getByRole('link', { name: cohort, exact: true }).first()).toBeVisible();
  // End moves through the grid's keyboard navigation, including virtualized rows.
  await grid.getByRole('link', { name: cohort, exact: true }).first().focus();
  await page.keyboard.press('Control+End');
  const last = grid.locator('[data-rowindex="19"]').getByRole('link', { name: cohort, exact: true });
  await last.click();
  await expect(page.getByLabel('第20个，共21个匹配对象')).toBeVisible();
  await btn('下一个对象').click();
  await expect(page.getByLabel('第21个，共21个匹配对象')).toBeVisible();
  expect(new URL(page.url()).searchParams.get('page')).toBe('2');
  await openPersonLog(page, 'Space');
  await click308(page, panel.getByRole('link', { name: '查看当次资料', exact: true }).last());
  const returning = page.getByRole('link', { name: '返回列表', exact: true });
  if (role === 'admin' && dimensions.zoom === 2) {
    // Deliver the real journal while the operator is pressing the directory link.
    let release!: () => void;
    const resume = new Promise<void>(resolve => { release = resolve; });
    const journal = '**/api/v1/persons/*/changes';
    const requested = page.waitForRequest(journal);
    await page.route(journal, async route => {
      await resume;
      await route.continue();
    }, { times: 1 });
    try {
      await btn('返回当前资料').click();
      await requested;
      await expect(returning).toHaveAttribute('href', /page=2/);
      await returning.hover();
      await page.mouse.down();
      await expect(returning).toBeFocused();
      release();
      await expect(panel.getByRole('link', { name: '查看当次资料' }).last()).toBeAttached();
      await expect(returning).toBeFocused();
    } finally {
      release();
      await page.mouse.up();
    }
  } else {
    await btn('返回当前资料').click();
    await expect(panel.getByRole('link', { name: '查看当次资料' }).last()).toBeFocused();
    await expect(returning).toHaveAttribute('href', /page=2/);
    await returning.click();
  }
  await expect(page).toHaveURL(url => url.pathname === '/personnel/employees' && url.searchParams.get('page') === '2');
  await expect
    .poll(() =>
      page.evaluate(() => ({
        page: new URL(location.href).searchParams.get('page'),
        focus: document.activeElement?.getAttribute('data-object-link') !== null,
        rows: document.querySelectorAll('a[data-object-link]').length,
      })),
    )
    .toEqual({ page: '2', focus: true, rows: 1 });
  expect(await page.getByLabel('姓名或人员编号').inputValue()).toBe(cohort);
}
for (const role of ['admin', 'reader'])
  for (const [w, h] of desktopViewports) {
    test('T01 ' + role + ' ' + w + '×' + h, async ({ page }) => {
      test.setTimeout(90000);
      await page.setViewportSize({ width: w, height: h });
      await login(page, role);
      await journey(page, role, w === 1440 && h === 900);
    });
  }
for (const role of ['admin', 'reader'])
  test('T01 ' + role + ' 原生缩放', async () => {
    test.setTimeout(180000);
    let n = 3500 + (process.pid % 1000);
    while (existsSync('/tmp/.X' + n + '-lock')) n++;
    const env = { ...process.env, DISPLAY: ':' + n },
      display = spawn(
        'Xvfb',
        [env.DISPLAY, '-screen', '0', '1920x1200x24', '-nolisten', 'tcp', '-nolisten', 'unix'],
        { env, stdio: 'ignore' },
      );
    let browser;
    try {
      await expect.poll(() => existsSync('/tmp/.X' + n + '-lock')).toBe(true);
      browser = await chromium.launch({
        headless: false,
        env,
        args: ['--ozone-platform=x11', '--window-size=1280,1039'],
      });
      const context = await browser.newContext({
        viewport: null,
        deviceScaleFactor: undefined,
        baseURL: localConnection().webOrigin,
      });
      const page = await context.newPage();
      await login(page, role);
      const cdp = await context.newCDPSession(page),
        win = await cdp.send('Browser.getWindowForTarget'),
        size = await page.evaluate(() => [innerWidth, innerHeight]);
      await cdp.send('Browser.setWindowBounds', {
        windowId: win.windowId,
        bounds: { width: win.bounds.width! + 1280 - size[0], height: win.bounds.height! + 900 - size[1] },
      });
      await expect
        .poll(() => page.evaluate(() => [innerWidth, innerHeight, devicePixelRatio]))
        .toEqual([1280, 900, 1]);
      execFileSync('python3', ['docs/verification/data/100-ui-baseline/native-zoom.py', '5'], { env });
      await expect
        .poll(() => page.evaluate(() => [innerWidth, innerHeight, devicePixelRatio]))
        .toEqual([640, 450, 2]);
      await journey(page, role, false);
    } finally {
      await browser?.close();
      display.kill();
    }
  });

function personnelSQL() {
  const project = JSON.parse(
    execFileSync('python3', ['scripts/local-bu.py', 'connection'], { encoding: 'utf8' }),
  ).project;
  const container = execFileSync(
    'docker',
    [
      'ps',
      '-q',
      '--filter',
      'label=com.docker.compose.project=' + project,
      '--filter',
      'label=com.docker.compose.service=postgres',
    ],
    { encoding: 'utf8' },
  ).trim();
  expect(container).toMatch(/^[0-9a-f]+$/);
  return (input: string) =>
    execFileSync(
      'docker',
      ['exec', '-i', container, 'psql', '-XAtq', '-U', 'postgres', '-d', 'chnhcm', '-v', 'ON_ERROR_STOP=1'],
      { input, stdio: ['pipe', 'ignore', 'pipe'] },
    );
}
async function personFieldRestriction(page: Page, id: string) {
  const principal = await page.evaluate(
    async () => (await (await fetch('/api/v1/session')).json()).subjectId,
  );
  for (const value of [principal, id]) expect(value).toMatch(/^[0-9a-f-]{36}$/);
  const grant = randomUUID(),
    sql = personnelSQL();
  sql(
    `insert into access.person_grant(instance_id,principal_id,grant_id,action,purpose,person_ids,fields,denied) select instance_id,principal_id,'${grant}','read','directory',array['${id}'::uuid],array['phone','phone.masked'],true from access.principal where principal_id='${principal}'`,
  );
  return {
    clear: () => sql(`delete from access.person_grant where grant_id='${grant}'`),
    hideHistory: () =>
      sql(`update access.person_grant set action='history.read',purpose='history',fields=array['name','number'] where grant_id='${grant}'`),
    hideIdentity: () =>
      sql(`update access.person_grant set fields=array['name','number','phone'] where grant_id='${grant}'`),
  };
}
test('T01 同名直接登记、本人改号、导航重试及撤去旧来源字段', async ({ page }) => {
  test.setTimeout(90000);
  await login(page, 'admin');
  await menu(page);
  const btn = (name: string) => page.getByRole('button', { name, exact: true });
  const reference = 'Q-9' + randomUUID().slice(0, 6);
  await page.getByRole('link', { name: '登记人员', exact: true }).click();
  await page.getByRole('textbox', { name: '姓名', exact: true }).fill(cohort);
  await page.getByLabel('登记单号（选填）').fill(reference);
  await expect(btn('登记为另一人')).toHaveCount(0);
  await select(page, '联系类型 1', '电话');
  await page.getByLabel('联系值 1').fill('17700009990');
  await btn('保存').click();
  await expect(btn('更正资料')).toBeVisible();
  const id = new URL(page.url()).pathname.split('/').at(-1)!;
  const oldNumber = (await page.locator('.object-identity-summary').innerText()).trim();
  await btn('更正资料').click();
  const typed = '正在核对的姓名' + randomUUID().slice(0, 4);
  await page.getByRole('textbox', { name: '姓名', exact: true }).fill(typed);
  await page.getByRole('group',{name:'联系方式',exact:true}).getByRole('button',{name:'编辑'}).first().click();
  await page.getByRole('dialog').getByRole('textbox', { name: '电话', exact: true }).fill('17700009991');
  await page.getByRole('dialog').getByRole('button',{name:'应用'}).click();
  const editRestriction = await personFieldRestriction(page, id);
  try {
    await page.evaluate(() => window.dispatchEvent(new Event('focus')));
    await expect(page.locator('input[value="17700009991"]')).toHaveCount(0);
    await expect(page.getByRole('textbox', { name: '姓名', exact: true })).toHaveValue(typed);
    await btn('取消').click();
    await btn('放弃修改').click();
  } finally {
    editRestriction.clear();
  }
  await btn('更多操作').click();
  await page.getByRole('menuitem', { name: '改号' }).click();
  const newNumber = 'R-' + randomUUID().slice(0, 8).toUpperCase();
  await page.getByLabel('新号', { exact: true }).fill(newNumber);
  await select(page, '原因', '编号纠错');
  await page.getByLabel('客户决定文号').fill('DEC-7');
  await btn('保存').click();
  await expect(page.locator('.object-identity-summary')).toHaveText(newNumber);
  await page.getByRole('link', { name: '返回列表' }).click();
  await search(page, oldNumber);
  await expect(page.getByText('历史编号', { exact: false }).last()).toBeVisible();
  const directoryRoute = '**/api/v1/personnel/directory/query';
  await page.route(directoryRoute, (route) => {
    if (route.request().postDataJSON()?.anchor !== id) return route.continue();
    return route.fulfill({
      status: 503,
      contentType: 'application/json',
      body: JSON.stringify({ code: 'UNAVAILABLE', message: '暂不可用' }),
    });
  });
  await page.getByRole('link', { name: cohort, exact: true }).click();
  await expect(page.getByText('暂时无法切换记录')).toBeVisible();
  await page.unroute(directoryRoute);
  await expect(page.getByText('17700009990', { exact: true })).toBeVisible();
  await btn('重试').click();
  await expect(page.getByLabel('第1个，共1个匹配对象')).toBeVisible();
  const log = await openPersonLog(page);
  const renumber = log.locator('.change-operation').first();
  await expect(renumber.getByText(oldNumber, { exact: true })).toBeVisible();
  await expect(renumber.getByText(newNumber, { exact: true })).toBeVisible();
  await log.getByRole('link', { name: '查看当次资料', exact: true }).last().click();
  const historyURL = page.url();
  await expect(page.locator('.object-identity-summary')).toHaveText(oldNumber);
  await Promise.all([
    page.waitForResponse((r) => r.request().method() === 'DELETE' && r.status() === 204),
    btn('退出').click(),
  ]);
  await login(page, 'reader');
  await page.goto(historyURL);
  await expect(page.getByText('17700009990', { exact: true })).toBeVisible();
  const restriction = await personFieldRestriction(page, id);
  try {
    await page.evaluate(() => window.dispatchEvent(new Event('focus')));
    await expect(page.getByText('17700009990', { exact: true })).toHaveCount(0);
    await expect(page.locator('.object-identity-summary')).toHaveText(oldNumber);
    await btn('返回当前资料').click();
    await page.goBack();
    await expect(page.locator('.object-identity-summary')).toHaveText(oldNumber);
    await expect(page.getByText('17700009990', { exact: true })).toHaveCount(0);
    restriction.hideIdentity();
    await page.evaluate(() => window.dispatchEvent(new Event('focus')));
    await expect(page.locator('.object-identity-summary')).toHaveCount(0);
    await expect(page.getByText('所选资料不可用')).toBeVisible();
  } finally {
    restriction.clear();
  }
});

test('T01 变更日志跨页、筛选、查看敏感差异与撤权', async ({ page, context }) => {
  test.setTimeout(120000);
  await login(page, 'admin');
  const prefix = '办理日志' + randomUUID().slice(0, 8);
  const originalPhone = '13800001234', changedPhone = '13900001234';
  const registered = await request(page, 'POST', '/api/v1/persons', {
    submissionId: randomUUID(), name: prefix, contacts: [{ kind: 'phone', value: originalPhone }],
  });
  expect(registered.status).toBe(200);
  const { id, sourceVersion: first } = registered.body;
  const path = '/api/v1/persons/' + id;
  const initial = await request(page, 'GET', path);
  expect(initial.status).toBe(200);
  const phone = initial.body.contacts[0];
  const versions: string[] = [first];
  async function correct(index: number) {
    const response = await request(page, 'POST', path + '/corrections', {
      submissionId: randomUUID(), expectedSourceVersion: versions.at(-1), name: prefix + index,
      ...(index === 21 && { contacts: [{
        itemId: phone.itemId, expectedRevision: phone.revision, kind: 'phone', value: changedPhone,
      }] }),
    });
    expect(response.status).toBe(200);
    versions.push(response.body.sourceVersion);
  }
  for (let i = 1; i <= 21; i++) await correct(i);
  await menu(page);
  await search(page, initial.body.number);
  await page.getByRole('link', { name: prefix + 21, exact: true }).click();
  const reads: Request[] = [], differences: Request[] = [];
  page.on('request', req => {
    const pathname = new URL(req.url()).pathname;
    if (pathname === path + '/changes') reads.push(req);
    if (pathname.startsWith(path + '/changes/')) differences.push(req);
  });
  const panel = await openPersonLog(page);
  const groups = panel.locator('tbody[data-source-version]');
  const sources = () => groups.evaluateAll(rows => rows.map(row => row.getAttribute('data-source-version')));
  await expect.poll(sources).toEqual(versions.slice(2).reverse());
  expect(reads.map(req => req.method())).toEqual(['POST']);
  const initialLogBody = await (await reads[0].response())!.text();
  expect(initialLogBody).not.toContain(originalPhone);
  expect(initialLogBody).not.toContain(changedPhone);
  expect(differences).toHaveLength(0);
  const latest = panel.locator(`tbody[data-source-version="${versions[21]}"]`);
  await expect(latest.locator('[data-label="变更字段"]')).toHaveCount(2);
  await expect(latest).toContainText(prefix + 20);
  await expect(latest).toContainText(prefix + 21);
  await expect(latest.getByText('****1234', { exact: true })).toHaveCount(2);
  await expect(panel.getByText(changedPhone, { exact: true })).toHaveCount(0);
  await expect(panel.getByText(originalPhone, { exact: true })).toHaveCount(0);
  const source = latest.getByRole('button', { name: '来源明细', exact: true });
  await source.click();
  await expect(source).toHaveAttribute('aria-expanded', 'true');
  const detail = latest.locator('[id]').filter({ hasText: versions[20] }).last();
  await expect(detail).toContainText(versions[21]);
  await source.press('Enter');
  await expect(source).toHaveAttribute('aria-expanded', 'false');
  await expect(source).toBeFocused();
  const sensitive = latest.getByRole('button', { name: /敏感差异/ });
  await sensitive.click();
  await expect(latest).toContainText(originalPhone);
  await expect(latest).toContainText(changedPhone);
  expect(differences).toHaveLength(1);
  await sensitive.click();
  await expect(latest).not.toContainText(changedPhone);

  async function filter() {
    await panel.getByRole('button', { name: '日志更多操作', exact: true }).click();
    await page.getByRole('menuitem', { name: /列筛选/ }).click();
    return page.getByRole('dialog', { name: '列筛选', exact: true });
  }
  async function phoneFilter() {
    const dialog = await filter();
    await select(page, '列', '变更字段');
    await select(page, '值', '电话');
    await dialog.getByRole('button', { name: '应用', exact: true }).click();
    await expect.poll(sources).toEqual([versions[21], first]);
  }
  const draft = await filter();
  await select(page, '列', '变更字段');
  await select(page, '值', '电话');
  await draft.getByRole('button', { name: '取消', exact: true }).click();
  await expect(draft).toHaveCount(0);
  expect(reads).toHaveLength(1);
  await correct(22);
  await panel.getByRole('button', { name: '下一页日志', exact: true }).click();
  await expect.poll(sources).toEqual([versions[1], first]);
  expect(reads).toHaveLength(2);
  await expect(panel.getByText(/21[–−-]22\s*\/\s*22/)).toBeVisible();

  const trigger = groups.last().getByRole('link', { name: '查看当次资料', exact: true });
  const logURL = page.url();
  const firstURL = new URL((await trigger.getAttribute('href'))!, logURL);
  expect(firstURL.searchParams.get('source')).toBe(first);
  const popup = context.waitForEvent('page');
  await trigger.click({ modifiers: ['Control'] });
  const originalTab = await popup;
  try {
    await originalTab.waitForURL(firstURL.href, { waitUntil: 'load' });
    await expect(originalTab.getByRole('heading', { name: prefix, exact: true })).toBeVisible();
    await expect(originalTab.getByRole('button', { name: '返回当前资料', exact: true })).toBeVisible();
    await expect(page).toHaveURL(logURL);
  } finally {
    await originalTab.close();
  }
  await reach308(page, trigger);
  const before = await trigger.evaluate(el => el.getBoundingClientRect().top);
  await trigger.click();
  await expect(page.getByRole('heading', { name: prefix, exact: true })).toBeVisible();
  await page.goBack();
  await expect.poll(sources).toEqual([versions[1], first]);
  await expect(trigger).toBeFocused();
  await expect.poll(() => trigger.evaluate((el, top) => Math.abs(el.getBoundingClientRect().top - top), before)).toBeLessThanOrEqual(2);
  await page.goForward();
  await expect(page.getByRole('heading', { name: prefix, exact: true })).toBeVisible();
  await page.goBack();
  await expect.poll(sources).toEqual([versions[1], first]);
  await expect(trigger).toBeFocused();

  // Return via the history notice into a separate visit, then jump between
  // the two journal entries while both display the same personnel route.
  await trigger.click();
  await page.getByRole('button', { name: '返回当前资料', exact: true }).click();
  await expect(trigger).toBeFocused();
  await phoneFilter();
  await page.evaluate(() => history.go(-2));
  await expect.poll(sources).toEqual([versions[1], first]);
  await expect(trigger).toBeFocused();
  await page.evaluate(() => history.go(2));
  await expect.poll(sources).toEqual([versions[21], first]);
  await expect(panel.getByText(/1[–−-]2\s*\/\s*2/)).toBeVisible();
  const clear = await filter();
  await clear.getByRole('button', { name: '清除条件', exact: true }).click();
  await clear.getByRole('button', { name: '应用', exact: true }).click();
  await expect.poll(sources).toEqual(versions.slice(2, 22).reverse());
  const invalid = await filter();
  await select(page, '列', '操作者');
  await invalid.getByRole('textbox', { name: '值', exact: true }).fill('人'.repeat(201));
  await invalid.getByRole('button', { name: '应用', exact: true }).click();
  const filterError = panel.getByRole('alert');
  await expect(filterError).toContainText('筛选值无效');
  await filterError.getByRole('button', { name: '清除条件', exact: true }).click();
  await expect.poll(sources).toEqual(versions.slice(2, 22).reverse());
  await panel.getByRole('button', { name: '日志更多操作', exact: true }).click();
  await page.getByRole('menuitem', { name: '刷新日志', exact: true }).click();
  await expect.poll(sources).toEqual(versions.slice(3).reverse());
  // A field may be revoked while this browser retains focus. Re-read the log
  // in place, then hold the fresh profile response to check the transition.
  const readOnlyProfile = await personFieldRestriction(page, id);
  let continueProfile!: () => void;
  const resumeProfile = new Promise<void>(resolve => { continueProfile = resolve; });
  try {
    await panel.getByRole('button', { name: '日志更多操作', exact: true }).click();
    await page.getByRole('menuitem', { name: '刷新日志', exact: true }).click();
    await expect(latest.locator('[data-label="变更字段"]')).toHaveCount(1);
    const pending = page.waitForRequest(req => new URL(req.url()).pathname === path && req.method() === 'GET');
    await page.route('**' + path, async route => {
      await resumeProfile;
      await route.continue();
    }, { times: 1 });
    const profile = page.getByRole('tab', { name: '个人资料', exact: true });
    await profile.click();
    await pending;
    await expect(profile).toBeFocused();
    await expect(page.getByRole('tabpanel', { name: '个人资料' })).not.toContainText(changedPhone);
    continueProfile();
    await expect(page.getByRole('heading', { name: prefix + 22, exact: true })).toBeVisible();
    await expect(page.getByRole('tabpanel', { name: '个人资料' })).not.toContainText(changedPhone);
    await expect(profile).toBeFocused();
  } finally {
    continueProfile();
    readOnlyProfile.clear();
  }
  await openPersonLog(page);
  await phoneFilter();
  await sensitive.click();
  await expect(latest).toContainText(changedPhone);
  // The server has already authorized this response. Deliver it only after
  // the current principal has re-read the log under the narrower grant.
  let release!: () => void;
  let captured!: () => void;
  let delivered!: () => void;
  const waiting = new Promise<void>(resolve => { release = resolve; });
  const oldResponse = new Promise<void>(resolve => { captured = resolve; });
  const finished = new Promise<void>(resolve => { delivered = resolve; });
  await page.route('**' + path + '/changes', async route => {
    const response = await route.fetch();
    captured();
    await waiting;
    try { await route.fulfill({ response }); }
    finally { delivered(); }
  }, { times: 1 });
  await panel.getByRole('button', { name: '日志更多操作', exact: true }).click();
  await page.getByRole('menuitem', { name: '刷新日志', exact: true }).click();
  await oldResponse;
  const restriction = await personFieldRestriction(page, id);
  try {
    await page.evaluate(() => window.dispatchEvent(new Event('focus')));
    await expect.poll(sources).toEqual(versions.slice(3).reverse());
    release();
    await finished;
    await expect.poll(sources).toEqual(versions.slice(3).reverse());
    await expect(panel.getByText('电话', { exact: true })).toHaveCount(0);
    await expect(panel).not.toContainText(changedPhone);
    await expect(panel).not.toContainText('****1234');
    await expect(panel.getByRole('button', { name: /敏感差异/ })).toHaveCount(0);
    const oldLog = page.url();
    restriction.hideHistory();
    await page.evaluate(() => window.dispatchEvent(new Event('focus')));
    const title = page.getByRole('heading', { name: prefix + 22, exact: true });
    await expect(page.getByRole('tablist')).toHaveCount(0);
    await expect(title).toBeFocused();
    await page.goto(oldLog);
    await expect(title).toBeFocused();
    await expect(page.getByRole('tablist')).toHaveCount(0);
    await expect(page.getByRole('link', { name: '查看当次资料', exact: true })).toHaveCount(0);
    await page.goto(firstURL.href);
    await expect(page.getByText('所选资料不可用', { exact: true })).toBeVisible();
    await expect(title).toHaveCount(0);
  } finally {
    release();
    restriction.clear();
  }
  // A fresh grant lets the reader reopen the log. Now discover its loss only
  // from the profile request issued by a deliberate tab change.
  await page.goto(logURL);
  await expect(panel.getByRole('link', { name: '查看当次资料' }).first()).toBeVisible();
  const leavingHistory = await personFieldRestriction(page, id);
  try {
    leavingHistory.hideHistory();
    const tab = page.getByRole('tab', { name: '个人资料', exact: true });
    await tab.click();
    await expect(page.getByRole('tablist')).toHaveCount(0);
    await expect(page.getByRole('heading', { name: prefix + 22, exact: true })).toBeFocused();
    await expect(page.getByRole('link', { name: '查看当次资料' })).toHaveCount(0);
  } finally {
    leavingHistory.clear();
  }
});

test('人事试办：同名两人、两人同时登记同一证号、同一张登记单办了两次', async ({ page, browser }) => {
  test.setTimeout(120000);
  const colleague = await browser.newContext({ baseURL: localConnection().webOrigin });
  const other = await colleague.newPage();
  const sql = personnelSQL(),
    registerGrant = randomUUID(),
    verifyGrant = randomUUID();
  const button = (p: Page, name: string) => p.getByRole('button', { name, exact: true });
  const openForm = async (p: Page, name: string, number: string, ref?: string) => {
    await menu(p);
    await p.getByRole('link', { name: '登记人员', exact: true }).click();
    await p.getByRole('textbox', { name: '姓名', exact: true }).fill(name);
    await p.getByRole('textbox', { name: '姓名', exact: true }).press('Tab');
    await select(p, '证件类型', '居民身份证');
    await p.getByRole('textbox', { name: '证件号码', exact: true }).fill(number);
    if (ref) await p.getByLabel('登记单号（选填）').fill(ref);
  };
  const save = async (p: Page) => {
    const response = p.waitForResponse(
      (r) => r.request().method() === 'POST' && new URL(r.url()).pathname === '/api/v1/persons',
    );
    await button(p, '保存').click();
    return await response;
  };
  const numberShown = async (p: Page) => {
    await expect(button(p, '更正资料')).toBeVisible();
    return (await p.locator('.object-identity-summary').innerText()).trim();
  };
  try {
    await Promise.all([login(page, 'admin'), login(other, 'reader')]);
    const subject = await other.evaluate(
      async () => (await (await fetch('/api/v1/session')).json()).subjectId,
    );
    expect(subject).toMatch(/^[0-9a-f-]{36}$/);
    sql(`insert into access.person_grant(instance_id,principal_id,grant_id,action,purpose,fields)
      select instance_id,principal_id,g.id::uuid,g.action,'registration',array['name','number','document','evidence']
      from access.principal cross join (values ('${registerGrant}','register'),('${verifyGrant}','verify')) g(id,action) where principal_id='${subject}'`);
    const namesQueried: string[] = [];
    const observe = (req: import('@playwright/test').Request) => {
      if (req.method() === 'POST' && /\/persons\/(queries|candidates)$/.test(new URL(req.url()).pathname))
        namesQueried.push(req.url());
    };
    page.on('request', observe);
    // The area code 000000 is reserved for these synthetic inputs.
    await openForm(page, '王伟', '000000199001010010');
    expect((await save(page)).status()).toBe(200);
    const first = await numberShown(page);
    await openForm(page, '王伟', '000000199001010029');
    await expect(page.getByText(/核对同名人员|登记为另一人/)).toHaveCount(0);
    expect((await save(page)).status()).toBe(200);
    const second = await numberShown(page);
    page.off('request', observe);
    expect(namesQueried).toEqual([]);
    expect(first).not.toBe(second);
    await menu(page);
    await search(page, '王伟');
    await expect(page.getByRole('link', { name: '王伟', exact: true })).toHaveCount(2);
    console.info('试办一：王伟两份档案', first, second, '登记时姓名查询为零');

    await Promise.all([
      openForm(page, '张敏', '000000199001010037'),
      openForm(other, '张旻', '000000199001010037'),
    ]);
    let arrived = 0,
      release!: () => void;
    const together = new Promise<void>((resolve) => {
      release = resolve;
    });
    for (const p of [page, other])
      await p.route(
        '**/api/v1/persons',
        async (route) => {
          if (route.request().method() === 'POST') {
            if (++arrived === 2) release();
            await together;
          }
          await route.continue();
        },
        { times: 1 },
      );
    const responses = await Promise.all([save(page), save(other)]);
    expect(responses.map((r) => r.status()).sort()).toEqual([200, 409]);
    const winner = responses[0].status() === 200 ? page : other,
      loser = winner === page ? other : page;
    await expect(loser.locator('#person-error')).toContainText('证件');
    await expect(loser.getByRole('textbox', { name: '证件号码', exact: true })).toHaveValue('000000199001010037');
    const created = await responses[winner === page ? 0 : 1].json();
    await expect(winner).toHaveURL(new RegExp(created.id));
    const wonNumber = (await winner.locator('.object-identity-summary').innerText()).trim();
    await button(loser, '取消').click();
    await button(loser, '放弃修改').click();
    await expect(loser.getByRole('heading', { name: '人员', level:1, exact: true })).toBeVisible();
    // The losing clerk already returned to the directory when discarding the form.
    if (winner === page) await menu(page);
    await expect(page).toHaveURL((url) => url.pathname === '/personnel/employees');
    await expect(page.locator('a[data-object-link]').first()).toBeVisible();
    await search(page, '张');
    await expect(page.getByRole('link', { name: /^张[敏旻]$/ })).toHaveCount(1);
    console.info('试办二：两位经办人同时保存，一笔成功，一笔被拒', wonNumber);

    await openForm(page, '李娜', '000000199001010045', 'RS-018');
    const original = await (await save(page)).json();
    const originalNumber = await numberShown(page);
    await openForm(other, '李娜', '000000199001010045', 'RS-018');
    const replay = await save(other);
    expect(replay.status()).toBe(200);
    expect(await replay.json()).toMatchObject({
      id: original.id,
      sourceVersion: original.sourceVersion,
      originalNumber,
    });
    await expect(other.getByRole('alert').filter({ hasText: '这张登记单已办过' })).toContainText(
      originalNumber,
    );
    await menu(page);
    await search(page, '李娜');
    await expect(page.getByRole('link', { name: '李娜', exact: true })).toHaveCount(1);
    console.info('试办三：登记单 RS-018 两次办理均指向原档', originalNumber);
  } finally {
    sql(`delete from access.person_grant where grant_id in ('${registerGrant}','${verifyGrant}')`);
    await colleague.close();
  }
});

test('仅填姓名登记后，在原档案补填和更换证件', async ({ page }) => {
  await login(page, 'admin');
  await menu(page);
  const button = (name: string) => page.getByRole('button', { name, exact: true });
  await page.getByRole('link', { name: '登记人员', exact: true }).click();
  await page.getByRole('textbox', { name: '姓名', exact: true }).fill('补证试办');
  await button('保存').click();
  await expect(button('更正资料')).toBeVisible();
  const number = (await page.locator('.object-identity-summary').innerText()).trim();
  await button('更正资料').click();
  const docs = page.getByRole('group', { name: '身份证件', exact: true });
  await docs.getByRole('button', { name: '添加一行' }).click();
  await select(page, '证件类型', '护照');
  await button('应用').click();
  await expect(page.getByRole('dialog')).toContainText('请填写证件号码');
  await page.getByRole('textbox', { name: '证件号码', exact: true }).fill('TEST-PASSPORT-A');
  await button('应用').click();
  await button('设为主要').click();
  await button('保存').click();
  await docs.getByRole('button', { name: '查看' }).click();
  await expect(page.getByRole('dialog')).toContainText('********');
  await button('查看完整号码').click();
  await expect(page.getByRole('dialog')).toContainText('TEST-PASSPORT-A');
  await page.keyboard.press('Escape');
  await button('更正资料').click();
  await docs.getByRole('button', { name: '编辑' }).click();
  await expect(page.getByRole('textbox', { name: '证件号码', exact: true })).toHaveValue('TEST-PASSPORT-A');
  await page.getByRole('textbox', { name: '证件号码', exact: true }).fill('TEST-PASSPORT-B');
  await button('应用').click();
  await button('保存').click();
  await expect(page.locator('.object-identity-summary')).toHaveText(number);
  await docs.getByRole('button', { name: '查看' }).click();
  await button('查看完整号码').click();
  await expect(page.getByRole('dialog')).toContainText('TEST-PASSPORT-B');
});

const reference = JSON.parse(
  readFileSync('docs/verification/data/504-personal-profile/dialog-layout.json', 'utf8'),
);
async function open504(page: Page, role = 'admin') {
  const name = '资料布局' + randomUUID().slice(0, 8);
  await login(page, 'admin');
  const root = await request(page, 'POST', '/api/v1/persons', { name, submissionId: randomUUID() });
  expect(root.status).toBe(200);
  const id = root.body.id;
  const saved = await request(page, 'POST', '/api/v1/persons/' + id + '/corrections', {
    submissionId: randomUUID(),
    expectedSourceVersion: root.body.sourceVersion,
    basic: { englishName: 'Alexandria'.repeat(8) },
    documents: [
      {
        fields: {
          documentKind: 'passport',
          documentNumber: 'SYN-' + randomUUID(),
          issuingCountry: '原文地区',
          issueDate: '2020-01-02',
          expiryDate: '2030-01-02',
        },
      },
      { fields: { documentKind: 'other', documentNumber: 'SYN-' + randomUUID() }, longTerm: true },
    ],
    contacts: [
      { kind: 'phone', value: '123456', purpose: '工作', primary: true },
      { kind: 'email', value: 'sample@example.test' },
    ],
  });
  expect(saved.status).toBe(200);
  if (role === 'reader') {
    await page.getByRole('button', { name: '退出', exact: true }).click();
    await login(page, role);
  }
  await menu(page);
  await search(page, name);
  await page.getByRole('link', { name, exact: true }).click();
  return name;
}
async function check504(page: Page, section: string, variant: string, mode: string) {
  const dlg = page.getByRole('dialog');
  const src = reference.layouts.find(
    (x: { section: string; variant: string }) => x.section === section && x.variant === variant,
  );
  expect(src).toBeTruthy();
  await expect(dlg.getByRole('heading')).toHaveText(section + ' · ' + reference.modes[mode].title_suffix);
  await expect(dlg.locator('.MuiDialogActions-root button')).toHaveText(reference.modes[mode].footer_right);
  const fields = dlg.locator('.profile-grid .object-detail-field');
  const requiredFields: Record<string, string[]> = {
    银行账户: ['银行名称', '户名', '银行账号', '币种'],
    地址: ['地址类型', '国家/地区', '详细地址'],
    家庭成员与紧急联系人: ['联系人姓名', '与员工关系'],
    教育经历: ['教育记录类型', '学校/授予院校', ...(variant === '已取得学历' ? ['学历'] : variant === '学位授予' ? ['学位'] : [])],
    '专业技术职称、职业资格与技能等级证书': ['凭证类别', '职称/资格名称'],
  };
  const required = requiredFields[section] || ['证件类型', '证件号码', '联系渠道', '电子邮箱', '电话'];
  const labels = src.desktop_rows
    .flat()
    .map((label: string) => (mode !== '查看' && required.includes(label) ? label + '*' : label));
  await expect(fields.locator('.object-field-label')).toHaveText(labels);
  const geometry = await dlg.evaluate((el) => {
    const paper = el.getBoundingClientRect(),
      grid = el.querySelector('.profile-grid')!,
      css = getComputedStyle(grid);
    const rows = Array.from(grid.querySelectorAll('.object-detail-field')).map((x) => {
      const r = x.getBoundingClientRect(),
        label = x.querySelector('label')!.getBoundingClientRect();
      return { x: r.x - paper.x, y: r.y, w: label.width, gap: parseFloat(getComputedStyle(x).columnGap) };
    });
    return {
      w: paper.width,
      h: paper.height,
      left: paper.left,
      top: paper.top,
      inner: innerWidth,
      height: innerHeight,
      rows,
      gap: css.rowGap,
      col: css.columnGap,
    };
  });
  const narrow = geometry.inner < 600;
  expect(geometry.w).toBe(narrow ? geometry.inner : Math.min(900, geometry.inner - 64));
  expect(geometry.left).toBeCloseTo((geometry.inner - geometry.w) / 2, 0);
  if (narrow) {
    expect(geometry.h).toBe(geometry.height);
    expect(geometry.top).toBe(0);
  }
  expect(geometry.gap).toBe(narrow ? '8px' : '10px');
  expect(geometry.col).toBe(narrow ? '0px' : '36px');
  geometry.rows.forEach((row, i) => {
    expect(row.w).toBe(narrow ? 108 : 132);
    expect(row.gap).toBe(narrow ? 12 : 14);
    if (narrow || i % 2 === 0) expect(row.x).toBe(narrow ? 16 : 24);
    if (!narrow && i % 2 === 1) {
      expect(row.x).toBeCloseTo(geometry.w / 2 + 18, 0);
      expect(row.y).toBe(geometry.rows[i - 1].y);
    }
    if (i > 0 && (narrow || i % 2 === 0)) expect(row.y).toBeGreaterThan(geometry.rows[i - 1].y);
  });
  if (mode !== '查看' && narrow) {
    const hint = await dlg.locator('.MuiDialogActions-root>span').boundingBox(),
      btn = await dlg.getByRole('button', { name: '应用' }).boundingBox();
    expect(hint!.y + hint!.height).toBeLessThanOrEqual(btn!.y);
  }
}
for (const [width, height] of desktopViewports)
  test('504 六种弹框与变体 ' + width + '×' + height, async ({ page }) => {
    test.setTimeout(150000);
    await page.setViewportSize({ width, height });
    await open504(page);
    const btn = (s: string) => page.getByRole('button', { name: s, exact: true });
    for (const mode of ['查看', '编辑', '新增']) {
      if (mode === '编辑') await btn('更正资料').click();
      if (mode !== '新增') {
        const basic = page.locator('#profile-basic');
        if (mode === '编辑') await expect(basic.getByRole('textbox', { name: '姓名', exact: true })).toBeVisible();
        const labels = basic.locator('.object-field-label');
        await expect(labels).toHaveCount(14);
        await expect(labels.nth(7)).toBeVisible();
        const layout = await labels.evaluateAll((nodes) =>
          nodes.map((label) => {
            const r = label.getBoundingClientRect(),
              v = label.nextElementSibling!.getBoundingClientRect();
            return { name: label.textContent?.replace('*', ''), x: r.x, y: r.y, valueX: v.x, valueY: v.y };
          }),
        );
        expect(layout.map((f) => f.name)).toEqual([
          '工号',
          '姓名',
          '曾用名',
          '英文名',
          '出生日期',
          '性别',
          '婚姻状况',
          '国籍',
          '民族',
          '籍贯',
          '出生地',
          '户籍类别',
          '户口性质',
          '政治面貌',
        ]);
        expect(layout[0].y).toBe(layout[7].y);
        expect(layout[7].x).toBeGreaterThan(layout[0].x);
        expect(layout[1].y - layout[0].y).toBeLessThanOrEqual(40);
        for (const f of layout) {
          expect(f.valueX).toBeGreaterThan(f.x + 90);
          expect(Math.abs(f.valueY - f.y)).toBeLessThan(8);
        }
        const docs = page.getByRole('group', { name: '身份证件', exact: true });
        await expect(docs).toHaveAccessibleDescription('2 条');
        await expect(docs.getByRole('button', { name: mode + ' · 护照', exact: true })).toBeVisible();
        await expect(docs).toContainText('********');
        await expect(docs).not.toContainText('SYN-');
        for (const text of ['至 2030-01-02', '长期有效', '原文地区'])
          await expect(docs.getByText(text, { exact: true })).toBeVisible();
        const contacts = page.getByRole('group', { name: '联系方式', exact: true });
        await expect(contacts.getByRole('button', { name: mode + ' · 123456', exact: true })).toBeVisible();
        await expect(contacts.getByText('主要', { exact: true })).toBeVisible();
        await expect(contacts.getByText('工作', { exact: true })).toBeVisible();
        if (mode === '查看') {
          const longValue = await basic.getByText('Alexandria'.repeat(8), { exact: true }).evaluate((el) => ({
            wraps: el.clientHeight > 32,
            fits: el.scrollWidth <= el.clientWidth + 1,
          }));
          expect(longValue).toEqual({ wraps: true, fits: true });
        }
        if (mode === '查看') await expect(docs.getByRole('columnheader', { name: '操作' })).toHaveCount(0);
        else
          await expect(basic.getByRole('textbox', { name: '姓名', exact: true })).toHaveAttribute(
            'required',
            '',
          );
      }
      for (const [section, variants] of [
        ['身份证件', ['有期限', '长期有效']],
        ['联系方式', ['电话', '电子邮箱']],
      ] as const) {
        const group = page.getByRole('group', { name: section, exact: true });
        for (const [i, variant] of variants.entries()) {
          await group
            .getByRole('button', {
              name: mode === '新增' ? '添加一行' : new RegExp('^' + mode + ' · '),
              exact: true,
            })
            .nth(mode === '新增' ? 0 : i)
            .click();
          if (mode === '新增' && i === 1) {
            if (section === '身份证件') await page.getByLabel('长期有效', { exact: true }).check();
            else await select(page, '联系渠道', '电子邮箱');
          }
          await check504(page, section, variant, mode);
          await page.keyboard.press('Escape');
          if (mode === '新增' && i === 1) await btn('放弃修改').click();
        }
      }
    }
    const email = page.getByRole('group', { name: '联系方式', exact: true });
    await email.getByRole('button', { name: '添加一行' }).click();
    await page.getByRole('textbox', { name: '电话', exact: true }).fill('987654');
    await btn('应用').click();
    await expect(email).toHaveAccessibleDescription('3 条');
    await expect(email).toContainText('987654');
    await btn('取消').click();
    await btn('放弃修改').click();
    await expect(email).toHaveAccessibleDescription('2 条');
    await expect(email).not.toContainText('987654');
  });
test('504 共用查看与长表单的视口和原生缩放', async () => {
  test.setTimeout(240000);
  const browser = await nativeZoomBrowser();
  const p = browser.page;
  try {
    await open504(p);
    for (const editing of [false, true]) {
      if (editing) await p.getByRole('button', { name: '更正资料', exact: true }).click();
      await p
        .getByRole('group', { name: '身份证件', exact: true })
        .getByRole('button', { name: editing ? /^编辑 · / : /^查看 · / })
        .first()
        .click();
      for (const [w, h, z] of desktopFormViewports) {
        await p.setViewportSize({ width: w, height: h });
        if (z > 1) await browser.setZoom(z);
        await check504(p, '身份证件', '有期限', editing ? '编辑' : '查看');
        const body = p.locator('.profile-dialog .MuiDialogContent-root');
        const last = body.locator('.object-detail-field').last();
        const title = p.locator('.profile-dialog .MuiDialogTitle-root');
        const actions = p.locator('.profile-dialog .MuiDialogActions-root');
        const originalTitle = await title.boundingBox(),
          originalActions = await actions.boundingBox();
        await body.hover();
        await p.mouse.wheel(0, 1600);
        await expect
          .poll(async () => {
            const l = await last.boundingBox(),
              b = await body.boundingBox();
            return !!l && !!b && l.y + l.height <= b.y + b.height + 1;
          })
          .toBeTruthy();
        expect(await title.boundingBox()).toEqual(originalTitle);
        expect(await actions.boundingBox()).toEqual(originalActions);
        const viewport = await p.evaluate(() => ({ width: innerWidth, height: innerHeight }));
        expect(originalTitle!.y).toBeGreaterThanOrEqual(0);
        expect(originalActions!.y + originalActions!.height).toBeLessThanOrEqual(viewport.height + 1);
        const first = p.getByRole('dialog').locator('.MuiDialogTitle-root button');
        await first.focus();
        await p.keyboard.press('Shift+Tab');
        await expect(p.getByRole('dialog').locator('.MuiDialogActions-root button').last()).toBeFocused();
        if (editing) {
          await p.keyboard.press('Shift+Tab');
          await expect(p.getByRole('button', { name: '取消', exact: true }).last()).toBeFocused();
          await p.keyboard.press('Tab');
          await expect(p.getByRole('button', { name: '应用', exact: true })).toBeFocused();
        }
      }
      await browser.setZoom(1);
      await p.keyboard.press('Escape');
    }
  } finally {
    await browser.close();
  }
});

test('504 受限字段重排、只读遮罩和原行焦点', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await open504(page, 'reader');
  const principal = await page.evaluate(
      async () => (await (await fetch('/api/v1/session')).json()).subjectId,
    ),
    id = new URL(page.url()).pathname.split('/').at(-1)!;
  expect(id).toMatch(/^[a-f0-9-]{36}$/);
  expect(principal).toMatch(/^[a-f0-9-]{36}$/);
  const grant = randomUUID(),
    channelGrant = randomUUID(),
    sql = personnelSQL();
  sql(
    `insert into access.person_grant(instance_id,principal_id,action,purpose,fields,person_ids,grant_id,denied) select instance_id,principal_id,'profile.read','personal-profile',array['issuer','issuingCountry','contactPurpose'],array['${id}'::uuid],'${grant}'::uuid,true from access.principal where principal_id='${principal}'`,
  );
  sql(`insert into access.person_grant(instance_id,principal_id,action,purpose,fields,person_ids,grant_id,denied)
    select instance_id,principal_id,'read','directory',array['email','email.masked'],array['${id}'::uuid],'${channelGrant}'::uuid,true from access.principal where principal_id='${principal}'`);
  try {
    await page.evaluate(() => window.dispatchEvent(new Event('focus')));
    const group = page.getByRole('group', { name: '身份证件', exact: true }),
      row = group.getByRole('button', { name: '查看' }).first();
    const contacts = page.getByRole('group', { name: '联系方式', exact: true });
    await expect(contacts).toHaveAccessibleDescription('1 条');
    await expect(contacts).not.toContainText('sample@example.test');
    await expect(contacts).not.toContainText('联系用途');
    await expect(group).not.toContainText('原文地区');
    await expect(group).not.toContainText('签发国家/地区');
    await row.focus();
    await page.keyboard.press('Enter');
    const dlg = page.getByRole('dialog');
    await expect(dlg.locator('.object-field-label')).toHaveText([
      '证件类型',
      '证件号码',
      '证件姓名',
      '签发日期',
      '到期日期',
      '长期有效',
    ]);
    await expect(dlg).toContainText('********');
    await expect(dlg.getByText('查看完整号码')).toHaveCount(0);
    await page.keyboard.press('Escape');
    await expect(row).toBeFocused();
    await expect(page.getByRole('button', { name: '更正资料' })).toHaveCount(0);
    await expect(page.getByRole('button', { name: '添加一行' })).toHaveCount(0);
  } finally {
    sql(`delete from access.person_grant where grant_id in ('${grant}','${channelGrant}')`);
  }
});
test('504 原文、目录采用与跨栏错误定位后整页保存', async ({ page }) => {
  test.setTimeout(140000);
  const name = await open504(page);
  const btn = (s: string) => page.getByRole('button', { name: s, exact: true });
  await btn('更正资料').click();
  await page.getByLabel('曾用名', { exact: true }).fill('曾用' + name);
  await page.getByLabel('国籍', { exact: true }).fill('中国');
  await page.getByRole('option', { name: '中国', exact: true }).click();
  const docs = page.getByRole('group', { name: '身份证件', exact: true });
  await docs.getByRole('button', { name: '编辑' }).first().click();
  await page.getByLabel('签发国家/地区', { exact: true }).fill('历史地名');
  await page.getByLabel('签发日期', { exact: true }).fill('2031-01-01');
  await btn('应用').click();
  await btn('保存').click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.setViewportSize({ width: 1440, height: 540 });
  const error = page.getByText('到期日期须不早于签发日期，长期有效时请清空到期日期', { exact: true }).last();
  await expect(error).toBeVisible();
  await error.scrollIntoViewIfNeeded();
  const e = await error.boundingBox(),
    b = await page.locator('.profile-dialog .MuiDialogContent-root').boundingBox();
  expect(e!.y).toBeGreaterThanOrEqual(b!.y - 1);
  expect(e!.y + e!.height).toBeLessThanOrEqual(b!.y + b!.height + 1);
  await page.getByLabel('到期日期', { exact: true }).fill('2040-01-01');
  await btn('应用').click();
  await page.setViewportSize({ width: 1440, height: 900 });
  await btn('保存').click();
  await expect(btn('更正资料')).toBeVisible();
  const id = new URL(page.url()).pathname.split('/').at(-1);
  const read = await request(page, 'GET', '/api/v1/persons/' + id);
  expect(read.body.basic.formerName).toBe('曾用' + name);
  expect(read.body.basic.nationality).toBe('中国');
  expect(read.body.nationalityAdoption.id).toBeTruthy();
  expect(read.body.documents[0].fields.issuingCountry).toBe('历史地名');
  expect(read.body.documents[0].issuingCountryAdoption).toBeUndefined();
});

test('504 证号字段隐藏时可以单独更正签发机关', async ({ page }) => {
  await open504(page);
  const id = new URL(page.url()).pathname.split('/').at(-1)!;
  expect(id).toMatch(/^[a-f0-9-]{36}$/);
  const grant = randomUUID(),
    sql = personnelSQL();
  sql(`insert into access.person_grant(instance_id,principal_id,action,purpose,fields,person_ids,grant_id,denied)
    select instance_id,principal_id,'profile.read','personal-profile',array['documentNumber','documentNumber.masked'],
      array['${id}'::uuid],'${grant}'::uuid,true from access.principal where login='admin'`);
  try {
    await page.reload();
    await page.getByRole('button', { name: '更正资料' }).click();
    const docs = page.getByRole('group', { name: '身份证件', exact: true });
    await docs.getByRole('button', { name: '编辑' }).first().click();
    await expect(page.getByRole('dialog').getByText('证件号码', { exact: true })).toHaveCount(0);
    await page.getByLabel('签发机关', { exact: true }).fill('合成签发机关');
    await page.getByRole('button', { name: '应用' }).click();
    await page.getByRole('button', { name: '保存', exact: true }).click();
    await docs.getByRole('button', { name: '查看' }).first().click();
    await expect(page.getByRole('dialog')).toContainText('合成签发机关');
  } finally {
    sql(`delete from access.person_grant where grant_id='${grant}'`);
  }
});

const familySection = '家庭成员与紧急联系人';
async function openFamilySample(page: Page, role = 'admin') {
  const name = '住址样本' + randomUUID().slice(0, 8);
  await login(page, 'admin');
  const root = await request(page, 'POST', '/api/v1/persons', { name, submissionId: randomUUID() });
  expect(root.status).toBe(200);
  const path = '/api/v1/persons/' + root.body.id;
  const saved = await request(page, 'POST', path + '/corrections', {
    expectedSourceVersion: root.body.sourceVersion,
    submissionId: randomUUID(),
    addresses: [
      {
        fields: {
          addressType: '居住地址',
          addressCountry: '原文地区',
          street: '合成住址',
          postalCode: '00123',
          addressPrimary: 'yes',
        },
      },
    ],
    family: [
      { fields: { familyName: '合成甲', relationship: '配偶', familyPhone: '12345', emergency: 'yes' } },
      { fields: { familyName: '合成乙', relationship: '子女' } },
    ],
  });
  expect(saved.status).toBe(200);
  if (role === 'reader') {
    await page.getByRole('button', { name: '退出', exact: true }).click();
    await login(page, role);
  }
  await menu(page);
  await search(page, name);
  await page.getByRole('link', { name, exact: true }).click();
  return { path, source: saved.body.sourceVersion, id: root.body.id };
}
for (const [width, height] of desktopViewports) {
  test('504 第二批两栏六个弹框及取消 ' + width + '×' + height, async ({ page }) => {
    await page.setViewportSize({ width, height });
    const { path, source } = await openFamilySample(page);
    const button = (name: string) => page.getByRole('button', { name, exact: true });
    for (const mode of ['查看', '编辑', '新增']) {
      if (mode === '编辑') await button('更正资料').click();
      for (const section of ['地址', familySection]) {
        const group = page.getByRole('group', { name: section, exact: true });
        const opener = group
          .getByRole('button', { name: mode === '新增' ? '添加一行' : new RegExp('^' + mode + ' · ') })
          .first();
        await opener.focus();
        await opener.press('Enter');
        await check504(page, section, '通用', mode);
        if (mode === '新增' && section === '地址') {
          await select(page, '地址类型', '通讯地址');
          await page.locator('#row-addressCountry').fill('历史地名');
          await page.locator('#row-street').fill('合成新址');
          await page.getByLabel('主要地址').check();
          await button('应用').click();
          await expect(group).toHaveAccessibleDescription('2 条');
          await expect(group.getByRole('button', { name: '编辑 · 通讯地址' })).toBeFocused();
        } else if (mode === '编辑' && section === familySection) {
          await page.locator('#row-familyJob').fill('行内取消');
          await page.getByRole('dialog').getByRole('button', { name: '取消' }).click();
          await expect(opener).toBeFocused();
        } else {
          await page.keyboard.press('Escape');
          await expect(opener).toBeFocused();
        }
      }
    }
    const family = page.getByRole('group', { name: familySection, exact: true });
    await family.getByRole('button', { name: '移除' }).last().click();
    await expect(family).toHaveAccessibleDescription('1 条');
    await page.getByLabel('曾用名', { exact: true }).fill('整页放弃');
    await button('取消').click();
    await button('放弃修改').click();
    await expect(family).toHaveAccessibleDescription('2 条');
    const data = (await request(page, 'GET', path)).body;
    expect(data.sourceVersion).toBe(source);
    expect(data.addresses).toHaveLength(1);
    expect(data.addresses[0].fields.addressPrimary).toBe('yes');
  });
}
test('504 第二批目录选择、错误归焦与跨栏保存', async ({ page }) => {
  const { path } = await openFamilySample(page);
  const btn = (name: string) => page.getByRole('button', { name, exact: true });
  await btn('更正资料').click();
  await page.getByLabel('曾用名').fill('同笔更正');
  await page
    .getByRole('group', { name: '地址', exact: true })
    .getByRole('button', { name: '添加一行' })
    .click();
  await select(page, '地址类型', '通讯地址');
  for (const [id, name] of [
    ['addressCountry', '中国'],
    ['province', '广东省'],
    ['area', '南山区'],
  ]) {
    await page.locator('#row-' + id).fill(name);
    await page.getByRole('option', { name, exact: true }).first().click();
  }
  await page.getByLabel('详细地址').fill('合成街道');
  await page.getByLabel('邮政编码').fill('00009');
  await btn('应用').click();
  const family = page.getByRole('group', { name: familySection, exact: true });
  await family.getByRole('button', { name: '编辑 · 合成甲' }).click();
  await page.locator('#row-familyPhone').fill('');
  await btn('应用').click();
  const area = (await request(page, 'GET', '/api/v1/geo-areas?q=440305')).body.items[0];
  const geo = '/api/v1/geo-areas/' + area.id;
  const seen = (await request(page, 'GET', geo)).body;
  const revision = await request(
    page,
    'POST',
    geo + '/records/' + seen.selectedRecord.recordId + '/correct',
    {
      expectedSourceVersion: seen.headSourceVersion,
      expectedRecordRevision: seen.selectedRecord.revision,
      changes: { nameEn: 'Test ' + randomUUID().slice(0, 8) },
    },
    randomUUID(),
  );
  expect(revision.status).toBe(200);
  await btn('保存').click();
  await expect(page.locator('#row-area')).toBeFocused();
  await expect(page.locator('#row-postalCode')).toHaveValue('00009');
  await page.locator('#row-area').fill('南山');
  await page.getByRole('option', { name: '南山区', exact: true }).click();
  await btn('应用').click();
  await btn('保存').click();
  await expect(
    page.getByText('紧急联系人须有电话或电子邮箱，请补填或撤下紧急联系人标记', { exact: true }).last(),
  ).toBeVisible();
  await expect(page.getByLabel('紧急联系人', { exact: true })).toBeFocused();
  await page.locator('#row-familyEmail').fill('family@example.test');
  await btn('应用').click();
  await btn('保存').click();
  await expect(btn('更正资料')).toBeVisible();
  const data = (await request(page, 'GET', path)).body;
  expect(data.basic.formerName).toBe('同笔更正');
  expect(data.addresses[1].fields.postalCode).toBe('00009');
  expect(data.addresses[1].adoptions.area.id).toBeTruthy();
  expect(data.family[0].fields.familyEmail).toBe('family@example.test');
});
test('504 第二批只读和受限家庭明细重排与条数', async ({ page }) => {
  const { path, id } = await openFamilySample(page, 'reader');
  const group = page.getByRole('group', { name: familySection, exact: true });
  await expect(group).toHaveAccessibleDescription('2 条');
  await expect(page.getByRole('button', { name: '更正资料' })).toHaveCount(0);
  await expect(group).toContainText('****2345');
  const data = (await request(page, 'GET', path)).body;
  const grants = [randomUUID(), randomUUID()];
  const sql = personnelSQL();
  expect(id).toMatch(/^[0-9a-f-]{36}$/);
  expect(data.family[1].itemId).toMatch(/^[0-9a-f-]{36}$/);
  try {
    sql(`insert into access.person_grant(instance_id,principal_id,action,purpose,fields,person_ids,grant_id,item_id,denied)
  select instance_id,principal_id,'family.read','family-contact',v.fields,array['${id}'::uuid],v.grant_id,v.item_id,true from access.principal cross join
  (values(array['familyName'],'${grants[0]}'::uuid,'${data.family[1].itemId}'::uuid),(array['familyBirthDate','familyEmail','familyEmail.masked'],'${grants[1]}'::uuid,null::uuid)) v(fields,grant_id,item_id) where login='reader'`);
    await page.evaluate(() => window.dispatchEvent(new Event('focus')));
    await expect(group).toHaveAccessibleDescription('1 条');
    for (const [width, height] of desktopViewports) {
      await page.setViewportSize({ width, height });
      const row = group.getByRole('button', { name: '查看 · 合成甲' });
      await row.focus();
      await row.press('Enter');
      const fields = page.getByRole('dialog').locator('.object-detail-field');
      await expect(fields.locator('label')).toHaveText([
        '联系人姓名',
        '与员工关系',
        '工作单位',
        '职务',
        '电话',
        '紧急联系人',
      ]);
      const positions = await fields.evaluateAll((elements) =>
        elements.map((el) => {
          const r = el.getBoundingClientRect();
          return [r.x, r.y];
        }),
      );
      expect(positions[2][0]).toBe(positions[0][0]);
      expect(positions[2][1]).toBeGreaterThan(positions[0][1]);
      expect(positions[0][1]).toBe(positions[1][1]);
      await page.keyboard.press('Escape');
      await expect(row).toBeFocused();
    }
  } finally {
    sql(`delete from access.person_grant where grant_id in ('${grants[0]}','${grants[1]}')`);
  }
});

test('504 第二批地址目录控件、桌面视口与原生缩放', async () => {
  test.setTimeout(150000);
  const browser = await nativeZoomBrowser(),
    p = browser.page;
  try {
    await openFamilySample(p);
    await p.getByRole('button', { name: '更正资料', exact: true }).click();
    const row = p.getByRole('button', { name: '编辑 · 居住地址' });
    for (const [width, height, zoom] of desktopFormViewports) {
      await p.setViewportSize({ width, height });
      if (zoom !== 1) await browser.setZoom(zoom);
      await row.focus();
      await row.press('Enter');
      const dlg = p.getByRole('dialog');
      await p.locator('#row-addressCountry').fill('中国');
      const option = p.getByRole('option', { name: '中国', exact: true });
      await expect(option).toBeVisible();
      await p.locator('#row-addressCountry').press('ArrowDown');
      await p.keyboard.press('Enter');
      await p.locator('#row-province').fill('广东省');
      await expect(p.getByRole('option', { name: '广东省', exact: true })).toBeVisible();
      await p.locator('#row-province').press('ArrowDown');
      await p.keyboard.press('Enter');
      await p.locator('#row-area').fill('南山');
      await expect(p.getByRole('option', { name: '南山区', exact: true })).toBeVisible();
      await p.locator('#row-area').press('ArrowDown');
      await p.keyboard.press('Enter');
      await expect(p.locator('#row-area')).toHaveValue('南山区');
      await p.locator('#row-postalCode').focus();
      await p.keyboard.press('Tab');
      await expect(p.getByLabel('主要地址')).toBeFocused();
      await p.keyboard.press('Tab');
      await expect(dlg.getByRole('button', { name: '取消' })).toBeFocused();
      await p.keyboard.press('Tab');
      await expect(dlg.getByRole('button', { name: '应用' })).toBeFocused();
      await p.keyboard.press('Shift+Tab');
      await p.keyboard.press('Enter');
      await expect(row).toBeFocused();
    }
  } finally {
    await browser.close();
  }
});

test('504 第二批保存前撤回家庭职务写权保留其余草稿', async ({ page }) => {
  const { path, source, id } = await openFamilySample(page);
  const btn = (name: string) => page.getByRole('button', { name, exact: true });
  await btn('更正资料').click();
  await page.getByLabel('曾用名').fill('仍可保存');
  await btn('编辑 · 合成甲').click();
  await page.locator('#row-familyJob').fill('已撤权草稿');
  await page.locator('#row-employer').fill('获权单位');
  await btn('应用').click();
  const grant = randomUUID(),
    sql = personnelSQL();
  expect(id).toMatch(/^[0-9a-f-]{36}$/);
  try {
    sql(`insert into access.person_grant(instance_id,principal_id,action,purpose,fields,person_ids,grant_id,denied)
   select instance_id,principal_id,'family.correct','family-contact',array['familyJob'],array['${id}'::uuid],'${grant}',true from access.principal where login='admin'`);
    await btn('保存').click();
    await expect(page.getByText('没有本次办理权限', { exact: true }).first()).toBeVisible();
    expect((await request(page, 'GET', path)).body.sourceVersion).toBe(source);
    await btn('编辑 · 合成甲').click();
    await expect(page.locator('#row-familyJob')).toHaveCount(0);
    await expect(page.getByRole('dialog')).not.toContainText('已撤权草稿');
    await expect(page.locator('#row-employer')).toHaveValue('获权单位');
    await page.keyboard.press('Escape');
    await expect(page.locator('#person-error')).toContainText('没有本次办理权限');
    await btn('保存').click();
    await expect(btn('更正资料')).toBeVisible();
    const result = (await request(page, 'GET', path)).body;
    expect(result.family[0].fields.familyJob).toBeUndefined();
    expect(result.family[0].fields.employer).toBe('获权单位');
    expect(result.basic.formerName).toBe('仍可保存');
  } finally {
    sql(`delete from access.person_grant where grant_id='${grant}'`);
  }
});

test('504 复核已打开的家庭明细使用当前行和字段权限', async ({ page }) => {
  const { path, id } = await openFamilySample(page);
  const entry = page.getByRole('button', { name: '查看 · 合成甲' });
  await entry.click();
  const dialog = page.getByRole('dialog'),
    family = page.getByRole('group', { name: familySection, exact: true });
  await expect(dialog).toContainText('12345');
  const sql = personnelSQL(),
    grants = [randomUUID(), randomUUID(), randomUUID()];
  const item = (await request(page, 'GET', path)).body.family[0].itemId;
  const deny = (key: string, grant: string) =>
    sql(`insert into access.person_grant(instance_id,principal_id,action,purpose,fields,person_ids,item_id,grant_id,denied)
    select instance_id,principal_id,'family.read','family-contact',array['${key}'],array['${id}'::uuid],'${item}','${grant}',true from access.principal where login='admin'`);
  try {
    sql(`insert into access.person_grant(instance_id,principal_id,action,purpose,fields,person_ids,grant_id)
      select instance_id,principal_id,'family.read','family-contact',array['familyPhone.masked'],array['${id}'::uuid],'${grants[2]}' from access.principal where login='admin'`);
    deny('familyPhone', grants[0]);
    await page.evaluate(() => window.dispatchEvent(new Event('focus')));
    await expect(dialog).toContainText('****2345');
    await expect(dialog).not.toContainText('12345');
    deny('familyName', grants[1]);
    await page.evaluate(() => window.dispatchEvent(new Event('focus')));
    await expect(dialog).toHaveCount(0);
    await expect(family).toHaveAccessibleDescription('1 条');
    await expect(family).toBeFocused();
  } finally {
    sql(`delete from access.person_grant where grant_id in ('${grants[0]}','${grants[1]}','${grants[2]}')`);
  }
});

test('504 复核行内撤权时保留可写输入并应用同一草稿', async ({ page }) => {
  const { path, id } = await openFamilySample(page);
  await page.getByRole('button', { name: '更正资料' }).click();
  await page.getByLabel('曾用名').fill('整页获权值');
  await page.getByRole('button', { name: '编辑 · 合成甲' }).click();
  await page.locator('#row-familyJob').fill('撤权时清除');
  await page.locator('#row-employer').fill('保留行内输入');
  const item = (await request(page, 'GET', path)).body.family[0].itemId;
  const sql = personnelSQL(),
    grant = randomUUID();
  try {
    sql(`insert into access.person_grant(instance_id,principal_id,action,purpose,fields,person_ids,item_id,grant_id,denied)
      select instance_id,principal_id,'family.correct','family-contact',array['familyJob'],array['${id}'::uuid],'${item}','${grant}',true from access.principal where login='admin'`);
    await page.evaluate(() => window.dispatchEvent(new Event('focus')));
    await expect(page.locator('#row-familyJob')).toHaveCount(0);
    await expect(page.locator('#row-employer')).toHaveValue('保留行内输入');
    await page.getByRole('button', { name: '应用', exact: true }).click();
    await page.getByRole('button', { name: '保存', exact: true }).click();
    await expect(page.getByRole('button', { name: '更正资料' })).toBeVisible();
    const detail = (await request(page, 'GET', path)).body;
    expect(detail.family[0].fields.familyJob).toBeUndefined();
    expect(detail.family[0].fields.employer).toBe('保留行内输入');
    expect(detail.basic.formerName).toBe('整页获权值');
  } finally {
    sql(`delete from access.person_grant where grant_id='${grant}'`);
  }
});

test('504 复核按服务业务日选择同省同名区县', async ({ page }) => {
  const { path } = await openFamilySample(page);
  const day = (await request(page, 'GET', '/api/v1/geo-areas/context')).body.currentDate;
  await page.clock.setFixedTime(new Date('2001-01-03T00:00:00Z'));
  const btn = (name: string) => page.getByRole('button', { name, exact: true });
  await btn('更正资料').click();
  await page
    .getByRole('group', { name: '地址', exact: true })
    .getByRole('button', { name: '添加一行' })
    .click();
  await select(page, '地址类型', '通讯地址');
  for (const [key, name] of [
    ['addressCountry', '中国'],
    ['province', '四川省'],
  ]) {
    await page.locator('#row-' + key).fill(name);
    await page.getByRole('option', { name, exact: true }).click();
  }
  const searched = page.waitForResponse(
    (r) =>
      new URL(r.url()).pathname === '/api/v1/geo-areas/candidates' &&
      new URL(r.url()).searchParams.get('q') === '市中区',
  );
  await page.locator('#row-area').fill('市中区');
  const response = await searched;
  expect(new URL(response.url()).searchParams.get('adoptedDate')).toBe(day);
  const items = (await response.json()).items;
  const leshan = items.find((x: { code: string }) => x.code === '511102');
  expect(leshan).toBeTruthy();
  await expect(page.getByRole('option', { name: /内江市.*市中区/ })).toBeVisible();
  await page.getByRole('option', { name: /乐山市.*市中区/ }).click();
  await page.getByLabel('详细地址').fill('合成同名地区样本');
  await btn('应用').click();
  await btn('保存').click();
  await expect(btn('更正资料')).toBeVisible();
  const address = (await request(page, 'GET', path)).body.addresses[1];
  expect(address.adoptions.area.id).toBe(leshan.id);
  expect(address.adoptions.area.date).toBe(day);
});

const credentialSection = '专业技术职称、职业资格与技能等级证书';
const thirdBatchRecords = [
  { section: '教育经历', variant: '学习经历', name: '在读合成院校' },
  { section: '教育经历', variant: '已取得学历', name: '学历合成院校' },
  { section: '教育经历', variant: '学位授予', name: '学位合成院校' },
  { section: credentialSection, variant: '长期有效', name: '长期合成职称' },
  { section: credentialSection, variant: '有期限', name: '有期合成资格' },
];

for (const [width, height] of desktopViewports) {
  test('504 第三批教育凭证录入、弹框与取消 ' + width + '×' + height, async ({ page }) => {
    test.setTimeout(180000);
    await page.setViewportSize({ width, height });
    const { path } = await openFamilySample(page);
    const button = (name: string) => page.getByRole('button', { name, exact: true });
    const dlg = page.getByRole('dialog');
    await button('更正资料').click();
    for (const { section, variant, name } of thirdBatchRecords) {
      await page
        .getByRole('group', { name: section, exact: true })
        .getByRole('button', { name: '添加一行' })
        .click();
      if (section === '教育经历') {
        if (variant !== '学习经历') await select(page, '教育记录类型', variant);
        await dlg.locator('#row-school').fill(name);
        if (variant === '学习经历') {
          await select(page, '教育完成状态', '在读');
          await dlg.locator('#row-startMonth').fill('2025-09');
        } else if (variant === '已取得学历') {
          await dlg.locator('#row-educationLevel').fill('本科');
          await dlg.getByLabel('最高学历').check();
        } else {
          await dlg.locator('#row-degree').fill('学士');
          await dlg.locator('#row-awardDate').fill('2023-06-30');
          await dlg.getByLabel('最高学位').check();
        }
      } else {
        await select(page, '凭证类别', variant === '长期有效' ? '职称' : '职业资格');
        await dlg.locator('#row-credentialName').fill(name);
        await dlg.locator('#row-issuedOn').fill('2023-01-01');
        if (variant === '长期有效') await dlg.getByLabel('长期有效').check();
        else {
          await dlg.locator('#row-certificateNumber').fill('SYNTH-CERT-504');
          await dlg.locator('#row-validFrom').fill('2023-01-01');
          await dlg.locator('#row-validUntil').fill('2022-12-31');
        }
      }
      await check504(page, section, variant, '新增');
      await dlg.getByRole('button', { name: '应用' }).click();
    }
    await page.getByLabel('曾用名', { exact: true }).fill('教育跨栏同存');
    await button('保存').click();
    await expect(dlg.getByText(/截止日期须不早于/)).toBeVisible();
    await dlg.locator('#row-validUntil').fill('2027-12-31');
    await dlg.getByRole('button', { name: '应用' }).click();
    await button('保存').click();
    await expect(button('更正资料')).toBeVisible();
    const saved = (await request(page, 'GET', path)).body;
    expect(saved.education).toHaveLength(3);
    expect(saved.credentials).toHaveLength(2);
    expect(saved.basic.formerName).toBe('教育跨栏同存');
    expect(
      saved.education
        .filter((x: { fields: Record<string, string> }) => x.fields.highestEducation)
        .map((x: { fields: Record<string, string> }) => x.fields.school),
    ).toEqual(['学历合成院校']);
    expect(
      saved.education
        .filter((x: { fields: Record<string, string> }) => x.fields.highestDegree)
        .map((x: { fields: Record<string, string> }) => x.fields.school),
    ).toEqual(['学位合成院校']);
    for (const mode of ['查看', '编辑']) {
      if (mode === '编辑') await button('更正资料').click();
      for (const { section, variant, name } of thirdBatchRecords) {
        const opener = button(mode + ' · ' + name);
        await opener.focus();
        await opener.press('Enter');
        await check504(page, section, variant, mode);
        if (mode === '查看' && variant === '长期有效') {
          await expect(dlg.getByRole('button', {name: '查看完整号码'})).toHaveCount(0);
        }
        if (mode === '查看' && variant === '有期限') {
          await expect(dlg).toContainText('********');
          await dlg.getByRole('button', { name: '查看完整号码' }).click();
          await expect(dlg).toContainText('SYNTH-CERT-504');
          await dlg.getByRole('button', { name: '隐藏号码' }).click();
          await expect(dlg).not.toContainText('SYNTH-CERT-504');
        }
        if (mode === '编辑' && variant === '学习经历') {
          await select(page, '教育记录类型', '学位授予');
          await expect(dlg.getByRole('status')).toContainText('入学年月');
          await dlg.locator('#row-degree').fill('硕士');
          await dlg.getByRole('button', { name: '应用' }).click();
        } else await page.keyboard.press('Escape');
        await expect(opener).toBeFocused();
      }
    }
    await page
      .getByRole('group', { name: credentialSection })
      .getByRole('button', { name: '移除' })
      .last()
      .click();
    await button('取消').click();
    await button('放弃修改').click();
    expect((await request(page, 'GET', path)).body.sourceVersion).toBe(saved.sourceVersion);
    await button('退出').click();
    await expect(page.getByLabel('密码')).toBeVisible();
    await login(page, 'reader');
    await menu(page);
    await search(page, saved.name);
    await page.getByRole('link', { name: saved.name, exact: true }).click();
    const readOnly = button('查看 · 有期合成资格');
    await readOnly.press('Enter');
    await expect(dlg).toContainText('********');
    await expect(dlg.getByRole('button', { name: '查看完整号码' })).toHaveCount(0);
    await page.keyboard.press('Escape');
    await expect(readOnly).toBeFocused();
    await expect(button('更正资料')).toHaveCount(0);
  });
}

test('504 第三批学习经历桌面视口及原生缩放', async () => {
  test.setTimeout(135000);
  const browser = await nativeZoomBrowser(),
    p = browser.page;
  try {
    await openFamilySample(p);
    await p.getByRole('button', { name: '更正资料', exact: true }).click();
    const opener = p
      .getByRole('group', { name: '教育经历', exact: true })
      .getByRole('button', { name: '添加一行' });
    for (const [width, height, zoom] of desktopFormViewports) {
      await p.setViewportSize({ width, height });
      if (zoom > 1) await browser.setZoom(zoom);
      await opener.focus();
      await opener.press('Enter');
      await check504(p, '教育经历', '学习经历', '新增');
      const dlg = p.getByRole('dialog'),
        body = dlg.locator('.MuiDialogContent-root');
      const header = dlg.locator('.MuiDialogTitle-root'),
        footer = dlg.locator('.MuiDialogActions-root');
      const top = await header.boundingBox(),
        bottom = await footer.boundingBox();
      await body.hover();
      await expect.poll(async () => {
        await p.mouse.wheel(0, 800);
        return body.evaluate((node) => {
          const row = node.querySelector('.object-detail-field:last-child')!;
          return Math.round(row.getBoundingClientRect().bottom - node.getBoundingClientRect().bottom);
        });
      }, { message: '滚轮到达末字段 ' + [width, height, zoom].join('/') }).toBeLessThanOrEqual(1);
      expect(await header.boundingBox()).toEqual(top);
      expect(await footer.boundingBox()).toEqual(bottom);
      await dlg.getByLabel('最高学位').focus();
      await p.keyboard.press('Tab');
      await expect(dlg.getByRole('button', { name: '取消' })).toBeFocused();
      await p.keyboard.press('Tab');
      await expect(dlg.getByRole('button', { name: '应用' })).toBeFocused();
      await p.keyboard.press('Escape');
      await expect(opener).toBeFocused();
    }
  } finally {
    await browser.close();
  }
});

for (const [width, height] of desktopViewports) {
  test('504 第四批银行三态、健康与整页保存取消 ' + width + '×' + height, async ({ page }) => {
    await page.setViewportSize({ width, height });
    const { path } = await openFamilySample(page);
    const button = (name: string) => page.getByRole('button', { name, exact: true });
    const bank = page.getByRole('group', { name: '银行账户', exact: true });
    const dlg = page.getByRole('dialog');
    await button('更正资料').click();
    await bank.getByRole('button', { name: '添加一行' }).click();
    await check504(page, '银行账户', '通用', '新增');
    for (const [id, value] of Object.entries({
      bankName: '合成银行',
      branchName: '合成支行',
      accountHolder: '合成户名',
      accountNumber: '000abc504',
      currency: 'CNY',
    })) {
      await dlg.locator('#row-' + id).fill(value);
    }
    await dlg.getByLabel('主要账户').check();
    await button('应用').click();
    await page.getByLabel('健康状况简述').fill('合成摘要甲');
    await page.getByLabel('曾用名').fill('合成九栏');
    await button('保存').click();
    const opener = button('查看 · 合成银行');
    await opener.press('Enter');
    await check504(page, '银行账户', '通用', '查看');
    await expect(dlg).toContainText('********');
    await button('查看完整号码').click();
    await expect(dlg).toContainText('000abc504');
    if (height === 900) {
      await button('隐藏号码').click();
      await expect(dlg).not.toContainText('000abc504');
      const sql = personnelSQL(), deny = randomUUID();
      const personId = path.split('/').at(-1)!;
      expect(personId).toMatch(/^[0-9a-f-]{36}$/);
      sql(`insert into access.person_grant(instance_id,principal_id,action,purpose,fields,person_ids,grant_id,denied)
        select instance_id,principal_id,'bank.read','personal-bank',array['accountNumber'],array['${personId}'::uuid],'${deny}',true from access.principal where login='admin'`);
      let resume!: () => void, entered!: () => void;
      const pending = new Promise<void>((resolve) => { resume = resolve; });
      const intercepted = new Promise<void>((resolve) => { entered = resolve; });
      try {
        await page.route((url) => url.searchParams.get('view')?.startsWith('bank-') === true, async (route) => {
          entered();
          await pending;
          await route.continue();
        }, { times: 1 });
        await button('查看完整号码').click();
        await intercepted;
        await expect(dlg).not.toContainText('000abc504');
        await expect(dlg).toContainText('********');
        resume();
        await expect(dlg.getByText('银行账号', { exact: true })).toHaveCount(0);
      } finally {
        resume();
        sql(`delete from access.person_grant where grant_id='${deny}'`);
      }
      await page.evaluate(() => window.dispatchEvent(new Event('focus')));
      await expect(button('查看完整号码')).toBeVisible();
      await expect(dlg).not.toContainText('000abc504');
    }
    await page.keyboard.press('Escape');
    await expect(opener).toBeFocused();
    const saved = (await request(page, 'GET', path)).body;
    expect(saved.healthSummary).toBe('合成摘要甲');
    expect(saved.basic.formerName).toBe('合成九栏');
    expect(saved.bankAccounts[0].fields.accountNumber).toBe('********');
    await button('更正资料').click();
    await button('编辑 · 合成银行').click();
    await expect(dlg.locator('#row-accountNumber')).toHaveValue('000abc504');
    await check504(page, '银行账户', '通用', '编辑');
    if (height === 900) {
      await dlg.locator('#row-accountNumber').fill('000different');
      await dlg.locator('#row-accountNumber').fill('000abc504');
      await page.keyboard.press('Escape');
      await expect(dlg).toHaveCount(0);
      await button('编辑 · 合成银行').click();
    }
    await dlg.locator('#row-branchName').fill('草稿支行');
    await button('应用').click();
    await button('编辑 · 合成甲').click();
    await dlg.locator('#row-employer').fill('草稿单位');
    await button('应用').click();
    await page
      .getByRole('group', { name: '地址', exact: true })
      .getByRole('button', { name: '添加一行' })
      .click();
    await select(page, '地址类型', '通讯地址');
    await dlg.locator('#row-addressCountry').fill('原文国家');
    await dlg.locator('#row-street').fill('草稿地址');
    await button('应用').click();
    await page.getByLabel('健康状况简述').fill('');
    await button('取消').click();
    await button('放弃修改').click();
    await expect(opener).toBeVisible();
    expect((await request(page, 'GET', path)).body).toEqual(saved);
    await button('退出').click();
    await expect(page.getByLabel('密码')).toBeVisible();
    await login(page, 'reader');
    await menu(page);
    await search(page, saved.name);
    await page.getByRole('link', { name: saved.name, exact: true }).click();
    await expect(page.getByRole('heading', { name: '基本信息', exact: true })).toBeVisible();
    await expect(bank).toHaveCount(0);
    await expect(page.getByText('健康信息', { exact: true })).toHaveCount(0);
    await expect(page.getByText('合成摘要甲')).toHaveCount(0);
    const maskedNumberOnly = height === 540;
    const grant = randomUUID(), sql = personnelSQL();
    expect(saved.id).toMatch(/^[0-9a-f-]{36}$/);
    try {
      sql(`insert into access.person_grant(instance_id,principal_id,action,purpose,fields,person_ids,grant_id)
        select instance_id,principal_id,'bank.read','personal-bank',${maskedNumberOnly ? "array['accountNumber.masked']" : "array['bankName','accountNumber.masked']"},array['${saved.id}'::uuid],'${grant}' from access.principal where login='reader'`);
      await page.reload();
      const restrictedOpener = maskedNumberOnly ? button('查看 · 银行账户') : opener;
      await restrictedOpener.press('Enter');
      await expect(dlg.locator('.object-field-label')).toHaveText(maskedNumberOnly ? ['银行账号'] : ['银行名称', '银行账号']);
      await expect(dlg).toContainText('********');
      await expect(dlg.getByRole('button', {name: '查看完整号码'})).toHaveCount(0);
      await expect(dlg).not.toContainText('000abc504');
      await page.keyboard.press('Escape');
      await expect(restrictedOpener).toBeFocused();
    } finally { sql(`delete from access.person_grant where grant_id='${grant}'`); }
  });
}

test('504 第四批健康撤权时整页回滚并清理草稿', async ({ page }) => {
  const { id, path, source } = await openFamilySample(page);
  const button = (name: string) => page.getByRole('button', { name, exact: true });
  await button('更正资料').click();
  await page.getByLabel('曾用名').fill('获权草稿');
  await page.getByLabel('健康状况简述').fill('甲'.repeat(201));
  await button('保存').click();
  await expect(page.getByLabel('健康状况简述')).toBeFocused();
  await expect(page.getByLabel('健康状况简述')).toHaveAttribute('aria-invalid', 'true');
  await page.getByLabel('健康状况简述').fill('撤权前输入');
  const sql = personnelSQL(),
    grant = randomUUID();
  expect(id).toMatch(/^[0-9a-f-]{36}$/);
  try {
    sql(`insert into access.person_grant(instance_id,principal_id,action,purpose,fields,person_ids,grant_id,denied)
      select instance_id,principal_id,'health.read','health-summary',array['healthSummary'],array['${id}'::uuid],'${grant}',true from access.principal where login='admin'`);
    await button('保存').click();
    await expect(page.locator('#person-error')).toContainText('没有本次办理权限');
    await expect(page.getByText('健康信息', { exact: true })).toHaveCount(0);
    await expect(page.getByLabel('曾用名')).toHaveValue('获权草稿');
    const detail = (await request(page, 'GET', path)).body;
    expect(detail.sourceVersion).toBe(source);
    expect(detail.healthSummary).toBeUndefined();
    expect(detail.basic.formerName).toBe('');
  } finally {
    sql(`delete from access.person_grant where grant_id='${grant}'`);
  }
});

test('504 第四批整页末栏和银行弹框的视口与原生缩放', async () => {
  test.setTimeout(150000);
  const browser = await nativeZoomBrowser(),
    page = browser.page;
  try {
    await openFamilySample(page);
    await page.getByRole('button', { name: '更正资料', exact: true }).click();
    for (const [width, height, zoom] of desktopFormViewports) {
      await page.setViewportSize({ width, height });
      if (zoom !== 1) await browser.setZoom(zoom);
      const summary = page.getByLabel('健康状况简述');
      await summary.focus();
      await expect(summary).toBeInViewport();
      await summary.fill('合成末栏');
      const add = page
        .getByRole('group', { name: '银行账户', exact: true })
        .getByRole('button', { name: '添加一行' });
      await add.focus();
      await add.press('Enter');
      await check504(page, '银行账户', '通用', '新增');
      const primary = page.getByLabel('主要账户');
      await primary.focus();
      await expect(primary).toBeInViewport();
      await primary.press('Tab');
      await expect(page.getByRole('dialog').getByRole('button', { name: '取消' })).toBeFocused();
      await page.keyboard.press('Escape');
      await expect(add).toBeFocused();
    }
  } finally {
    await browser.close();
  }
});
