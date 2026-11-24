import { chromium } from '@playwright/test';
import { randomUUID } from 'node:crypto';
import { existsSync } from 'node:fs';
import { execFileSync, spawn } from 'node:child_process';
import { localConnection } from '../../scripts/local-runtime.mjs';
import { test, expect, type Page } from './fixtures';
import { request, apiBU, apiDepartment } from './dated-object-support';
import { seed308 } from './staffing308-support';

async function signIn(page: Page, login = 'admin') {
  await page.goto('/');
  await page.getByLabel('登录名').fill(login);
  await page.getByLabel('密码').fill('admin1234567');
  await page.getByRole('button', { name: '登录', exact: true }).click();
  await expect(page.getByRole('button', { name: '退出', exact: true })).toBeVisible();
}
async function seed(page: Page, withDocument = true) {
  const suffix = randomUUID().slice(0, 8),
    name = '入职试办' + suffix,
    department = '研发' + suffix,
    employer = '用人单位' + suffix,
    job = '专员' + suffix;
  const bu = await apiBU(page, '管理单元' + suffix);
  await apiDepartment(page, bu.id, { name: department });
  const family = await seed308(page, 'jobfamily');
  await seed308(page, 'job', family.id, { name: job });
  const send = async (path: string, input: object, status: number) => {
    const r = await request(page, 'POST', '/api/v1/' + path, input, randomUUID());
    expect(r.status, r.body.message).toBe(status);
    return r.body;
  };
  const countries = await request(page, 'GET', '/api/v1/country-regions?q=CN');
  const cn = countries.body.items.find((r: { code: string }) => r.code === 'CN');
  const entity = await send(
    'legal-entities',
    {
      code: 'H' + suffix,
      name: employer,
      countryRegionId: cn.id,
      subjectType: 'legal_person',
      registrationType: '登记证',
      registrationNumber: 'T' + suffix,
      establishingOrganization: '合成设立单位',
      operationType: 'CREATE',
      operationReason: 'NEW_LEGAL_ENTITY',
    },
    201,
  );
  await send(
    'legal-entities/' + entity.legalEntityId + '/purposes',
    {
      expectedVersion: '',
      row: {
        id: randomUUID(),
        role: 'employer',
        kind: 'declaration',
        basis: '设立及用工资格',
        from: '2027-04-01',
      },
    },
    200,
  );
  const policy = await request(page, 'GET', '/api/v1/assignment-policies');
  if (
    !policy.body.rows.some(
      (r: { category: string; purpose: string }) => r.category === 'Employee' && r.purpose === 'hire',
    )
  ) {
    await send(
      'assignment-policies',
      {
        expectedVersion: policy.body.version,
        row: { id: randomUUID(), category: 'Employee', purpose: 'hire', mode: 'JobOnly', from: '2027-04-01' },
      },
      200,
    );
  }
  const saved = await send(
    'persons',
    {
      submissionId: randomUUID(),
      name,
      ...(withDocument ? { primaryDocument: { kind: 'passport', number: 'SYN-' + suffix } } : {}),
    },
    200,
  );
  return { name, department, employer, job, id: saved.id };
}
async function openPerson(page: Page, name: string) {
  if (await page.evaluate(() => matchMedia('(min-width:900px)').matches)) {
    await page.getByRole('button', { name: '切换应用', exact: true }).click();
    await page.getByRole('menuitem', { name: '人员', exact: true }).click();
  } else {
    await page.getByRole('button', { name: '打开导航', exact: true }).click();
    await page.getByRole('button', { name: '人员', exact: true }).click();
  }
  await page.getByLabel('姓名或人员编号').fill(name);
  await page.getByLabel('姓名或人员编号').press('Enter');
  await page.locator('a[data-object-link]').filter({ hasText: name }).click();
}
async function fillHire(page: Page, sample: Awaited<ReturnType<typeof seed>>) {
  await openPerson(page, sample.name);
  await page.getByRole('link', { name: '办理入职', exact: true }).click();
  const date = page.getByRole('textbox', { name: /^到岗日/ });
  await date.fill('2027-04-01');
  await date.press('Tab');
  for (const [label, value] of [
    ['部门', sample.department],
    ['职务', sample.job],
    ['用人单位', sample.employer],
  ]) {
    const target = page.getByRole('combobox', { name: new RegExp('^' + label) });
    await target.fill(value);
    await page.getByRole('option', { name: value, exact: true }).click();
  }
  await page.getByRole('textbox', { name: /^任用通知说明/ }).fill('三月人事任用通知：四月一日到岗。');
  await expect
    .poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1))
    .toBe(true);
}
async function journey(page: Page, sample: Awaited<ReturnType<typeof seed>>, disconnect = false) {
  const button = (name: string) => page.getByRole('button', { name, exact: true });
  await fillHire(page, sample);
  if (disconnect) {
    await page.route(
      '**/api/v1/hires',
      async (route) => {
        await route.fetch();
        await route.abort('failed');
      },
      { times: 1 },
    );
  }
  await expect(button('保存')).toBeEnabled();
  await button('保存').focus();
  await page.keyboard.press('Enter');
  await expect(page.getByText('可见在职 1 人，任职 1 份，占编 0')).toBeVisible();
  await button('查看原办理').click();
  await expect(page.getByText('采用规则 T02-employee-1')).toBeVisible();
  await page.getByRole('textbox', { name: /^查询日期/ }).fill('2027-03-31');
  await expect(page.getByText('可见在职 0 人，任职 0 份，占编 0')).toBeVisible();
  await button('撤销未来入职').click();
  await page.getByRole('textbox', { name: /^撤销说明/ }).fill('任用单位于到岗前撤回通知');
  await button('保存').click();
  await expect(page.getByText(/入职（已撤销）/)).toBeVisible();
  await page.getByRole('textbox', { name: /^查询日期/ }).fill('2027-04-01');
  await expect(page.getByText('可见在职 0 人，任职 0 份，占编 0')).toBeVisible();
}
test('T03 查回前一人的原号时保留当前入职草稿', async ({ page }) => {
  test.setTimeout(110000);
  await signIn(page);
  const first = await seed(page);
  await fillHire(page, first);
  const [requestA] = await Promise.all([
    page.waitForRequest((r) => new URL(r.url()).pathname === '/api/v1/hires' && r.method() === 'POST'),
    page.getByRole('button', { name: '保存', exact: true }).click(),
  ]);
  await expect(page.getByText('可见在职 1 人，任职 1 份，占编 0')).toBeVisible();
  const key = await requestA.headerValue('Idempotency-Key');
  expect(key).toBeTruthy();
  await page.goto('/');
  const next = await seed(page);
  await fillHire(page, next);
  await expect(page.getByRole('button', { name: '保存', exact: true })).toBeEnabled();
  await page.evaluate(async (submissionId) => {
    const p = await (await fetch('/api/v1/session')).json();
    localStorage.setItem(
      'chnhcm-pending-employee-hire',
      JSON.stringify([
        {
          instanceRef: p.instanceId,
          subjectRef: p.subjectId,
          submissionId,
        },
      ]),
    );
    window.dispatchEvent(new Event('storage'));
  }, key);
  await page.getByRole('button', { name: '查回原提交' }).click();
  await expect(page.getByText('已查回原办理。')).toBeVisible();
  await expect(page.getByRole('textbox', { name: /^任用通知说明/ })).toHaveValue(/四月一日/);
  await expect(page.getByRole('button', { name: '保存', exact: true })).toBeEnabled();
  const state = await request(
    page,
    'GET',
    '/api/v1/persons/' + next.id + '/employment?businessAt=2027-04-01',
  );
  expect(state.body.assignments).toBe(0);
});

test('T03 只读经办人从原档查看但不能办理入职', async ({ page, browser }) => {
  await signIn(page);
  const sample = await seed(page);
  const viewer = await browser.newContext({ baseURL: localConnection().webOrigin });
  try {
    const read = await viewer.newPage();
    await signIn(read, 'reader');
    await openPerson(read, sample.name);
    await expect(read.getByRole('link', { name: '办理入职', exact: true })).toHaveCount(0);
    const scope = await request(read, 'GET', '/api/v1/persons/' + sample.id + '/employment');
    expect(scope.body.canHire).toBe(false);
    const denied = await request(read, 'POST', '/api/v1/hires/targets', {
      person: sample.id,
      kind: 'job',
      arrival: '2027-04-01',
      page: 1,
    });
    expect(denied.status).toBe(403);
  } finally {
    await viewer.close();
  }
});

test('T03 原人补证后放弃入职', async ({ page }) => {
  test.setTimeout(110000);
  await signIn(page);
  const sample = await seed(page, false);
  await fillHire(page, sample);
  await expect(page.getByText(/待补齐：适用的身份证件/)).toBeVisible();
  await expect(page.getByRole('button', { name: '保存', exact: true })).toBeDisabled();
  const [editor] = await Promise.all([
    page.waitForEvent('popup'),
    page.getByRole('link', { name: '在新窗口补充原档' }).click(),
  ]);
  await editor.getByRole('group', { name: '身份证件' }).getByRole('button', { name: '添加一行' }).click();
  await editor.getByRole('combobox', { name: '证件类型', exact: true }).click();
  await editor.getByRole('option', { name: '护照', exact: true }).click();
  await editor
    .getByRole('textbox', { name: '证件号码', exact: true })
    .fill('SYN-' + randomUUID().slice(0, 8));
  await editor.getByRole('dialog').getByRole('button', { name: '应用' }).click();
  await editor.getByRole('button', { name: '保存', exact: true }).click();
  await expect(editor.getByRole('button', { name: '更正资料', exact: true })).toBeVisible();
  await editor.close();
  const retry = page.getByRole('button', { name: '重新核对' });
  if (await retry.isVisible()) await retry.click();
  await expect(page.getByRole('button', { name: '保存', exact: true })).toBeEnabled();
  await expect(page.getByRole('textbox', { name: /^任用通知说明/ })).toHaveValue(/四月一日/);
  await page.getByRole('button', { name: '取消', exact: true }).click();
  await page.getByRole('button', { name: '放弃修改' }).click();
  await page.getByRole('textbox', { name: /^查询日期/ }).fill('2027-04-01');
  await expect(page.getByText('可见在职 0 人，任职 0 份，占编 0')).toBeVisible();
  const current = await request(page, 'GET', '/api/v1/persons/' + sample.id + '?view=edit');
  expect(current.body.documents).toHaveLength(1);
});
for (const [width, height] of [
  [1440, 900],
  [1440, 540],
  [390, 844],
  [320, 900],
  [320, 256],
]) {
  test('T03 首单入职与未来撤销 ' + width + '×' + height, async ({ page }) => {
    test.setTimeout(110000);
    await signIn(page);
    const sample = await seed(page);
    await page.setViewportSize({ width, height });
    await journey(page, sample, width === 1440 && height === 540);
  });
}
test('T03 原生 200% 首单办理及撤销', async () => {
  test.setTimeout(140000);
  let display = 5800 + (process.pid % 1000);
  while (existsSync('/tmp/.X11-unix/X' + display) || existsSync('/tmp/.X' + display + '-lock')) display++;
  const env = { ...process.env, DISPLAY: ':' + display };
  const xvfb = spawn(
    'Xvfb',
    [env.DISPLAY, '-screen', '0', '1920x1200x24', '-nolisten', 'tcp', '-nolisten', 'unix'],
    { env, stdio: ['ignore', 'ignore', 'pipe'] },
  );
  xvfb.stderr?.on('data', (data) => process.stderr.write(data));
  let browser: Awaited<ReturnType<typeof chromium.launch>> | undefined;
  try {
    await expect
      .poll(
        () => {
          try {
            execFileSync('xdpyinfo', [], { env, stdio: 'ignore', timeout: 1000 });
            return true;
          } catch {
            return false;
          }
        },
        { timeout: 15000 },
      )
      .toBe(true);
    browser = await chromium.launch({
      headless: false,
      env,
      args: ['--window-size=1280,1039', '--ozone-platform=x11'],
    });
    const ctx = await browser.newContext({
      viewport: null,
      deviceScaleFactor: undefined,
      baseURL: localConnection().webOrigin,
    });
    const page = await ctx.newPage();
    await signIn(page);
    const sample = await seed(page);
    const cdp = await ctx.newCDPSession(page),
      win = await cdp.send('Browser.getWindowForTarget');
    const size = await page.evaluate(() => ({ w: innerWidth, h: innerHeight }));
    await cdp.send('Browser.setWindowBounds', {
      windowId: win.windowId,
      bounds: { width: win.bounds.width! + 1280 - size.w, height: win.bounds.height! + 900 - size.h },
    });
    const dims = () => page.evaluate(() => [innerWidth, innerHeight, devicePixelRatio]);
    await expect.poll(dims).toEqual([1280, 900, 1]);
    const zoom = (n: number) =>
      execFileSync('python3', ['docs/verification/data/100-ui-baseline/native-zoom.py', String(n)], {
        env,
        stdio: 'pipe',
      });
    zoom(5);
    await expect.poll(dims).toEqual([640, 450, 2]);
    await journey(page, sample);
    zoom(0);
    await expect.poll(dims).toEqual([1280, 900, 1]);
  } finally {
    await browser?.close();
    xvfb.kill();
  }
});
