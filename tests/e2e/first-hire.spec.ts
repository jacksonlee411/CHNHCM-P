import { chromium } from '@playwright/test';
import { randomUUID } from 'node:crypto';
import { existsSync } from 'node:fs';
import { execFileSync, spawn } from 'node:child_process';
import { localConnection } from '../../scripts/local-runtime.mjs';
import { test, expect, type Page } from './fixtures';
import { request, apiBU, apiDepartment } from './dated-object-support';
import { database } from './community-list-support.mjs';
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
  const dep = await apiDepartment(page, bu.id, { name: department });
  const family = await seed308(page, 'jobfamily');
  const positionJob = await seed308(page, 'job', family.id, { name: job });
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
  return {
    name,
    department,
    employer,
    job,
    id: saved.id,
    buId:bu.id,
    departmentId: dep.id,
    employerId: entity.legalEntityId,
    jobId: positionJob.id,
  };
}
async function openPerson(page: Page, name: string) {
  if (await page.evaluate(() => matchMedia('(min-width:900px)').matches)) {
    await page.getByRole('button', { name: '切换应用', exact: true }).click();
    await page.getByRole('menuitem', { name: '人员', exact: true }).click();
  } else {
    await page.getByRole('button', { name: '打开导航', exact: true }).click();
    await page.getByRole('button', { name: '人员', exact: true }).click();
    const archive = page.getByRole('button', { name: '人员档案', exact: true });
    const heading = page.getByRole('heading', { name: '人员档案', exact: true });
    await expect(archive.or(heading).first()).toBeVisible();
    if (await archive.isVisible()) await archive.click();
  }
  const archive = page.getByRole('link', { name: '人员档案', exact: true });
  const heading = page.getByRole('heading', { name: '人员档案', exact: true });
  await expect(archive.or(heading).first()).toBeVisible();
  if (await archive.isVisible()) await archive.click();
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
async function selectIssuedDecision(page: Page, sample: Awaited<ReturnType<typeof seed>>, roster: boolean) {
  const number = '合成人事任字-' + randomUUID().slice(0, 8);
  const ref = { source: randomUUID(), version: randomUUID(), line: roster ? randomUUID() : undefined };
  const rows = [
    {
      person: sample.id,
      line: ref.line,
      employer: { id: sample.employerId, name: sample.employer },
      arrival: '2027-04-01',
      department: { id: sample.departmentId, name: sample.department },
      job: { id: sample.jobId, name: sample.job },
    },
  ];
  if (roster) {
    const other = await request(page, 'POST', '/api/v1/persons', {
      submissionId: randomUUID(),
      name: '同份名单另一人',
    });
    expect(other.status).toBe(200);
    rows.push({ ...rows[0], person: other.body.id, line: randomUUID() });
  }
  const issued = await request(
    page,
    'POST',
    '/api/v1/appointment-decisions',
    {
      source: ref.source,
      number,
      roster,
      withdrawn: false,
      rows,
    },
    ref.version,
  );
  expect(issued.status, issued.body.message).toBe(200);
  await openPerson(page, sample.name);
  await page.getByRole('link', { name: '办理入职', exact: true }).click();
  const basis = page.getByRole('combobox', { name: '任用依据' });
  await basis.focus();
  await basis.press('ArrowDown');
  await page.getByRole('option', { name: number, exact: true }).click();
  for (const text of [sample.employer, sample.department, sample.job, '2027-04-01'])
    await expect(page.getByText(text, { exact: true })).toBeVisible();
  await expect(page.getByRole('textbox', { name: /任用通知说明/ })).toHaveCount(0);
  await expect
    .poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1))
    .toBe(true);
  return { ...ref, number };
}
async function journey(
  page: Page,
  sample: Awaited<ReturnType<typeof seed>>,
  disconnect = false,
  roster = false,
) {
  const button = (name: string) => page.getByRole('button', { name, exact: true });
  const ref = await selectIssuedDecision(page, sample, roster);
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
  await expect(page.getByText('任用决定 ' + ref.number)).toBeVisible();
  await expect(page.getByText('原单 ' + ref.source + '，修订 ' + ref.version)).toBeVisible();
  if (ref.line) await expect(page.getByText('名单行 ' + ref.line + '，修订 ' + ref.version)).toBeVisible();
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
    await journey(page, sample, width === 1440 && height === 540, width <= 390);
  });
}
async function native200(run: (page: Page, sample: Awaited<ReturnType<typeof seed>>) => Promise<void>) {
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
    await run(page, sample);
    zoom(0);
    await expect.poll(dims).toEqual([1280, 900, 1]);
  } finally {
    await browser?.close();
    xvfb.kill();
  }
}

test('T03 原生 200% 首单办理及撤销', async () => {
  await native200(journey);
});

async function hireAtLocation(page: Page, sample: Awaited<ReturnType<typeof seed>>) {
  const suffix = randomUUID().slice(0, 7),
    locations: { id: string; name: string }[] = [];
  const countries = await request(page, 'GET', '/api/v1/country-regions?q=MO');
  const mo = countries.body.items.find((c: { code: string }) => c.code === 'MO');
  for (const label of ['甲', '乙', '丙']) {
    const name = '工作地点' + label + suffix;
    const created = await request(
      page,
      'POST',
      '/api/v1/locations',
      {
        code: 'L' + locations.length + suffix,
        name,
        countryRegionId: mo.id,
        addressLine: '合成地址',
        effectiveDate: '2026-01-01',
        operationType: 'CREATE',
        operationReason: 'NEW_LOCATION',
      },
      randomUUID(),
    );
    expect(created.status, created.body.message).toBe(201);
    locations.push({ name, id: created.body.locationId });
  }
  const hired = await request(
    page,
    'POST',
    '/api/v1/hires',
    {
      person: sample.id,
      employer: sample.employerId,
      department: sample.departmentId,
      job: sample.jobId,
      location: locations[0].id,
      arrival: '2027-04-01',
      notice: '合成入职通知',
    },
    randomUUID(),
  );
  expect(hired.status, hired.body.message).toBe(200);
  return { locations, hire: hired.body };
}
async function openEmployeeAssignment(page: Page,sample:Awaited<ReturnType<typeof seed>>){
  const desktop = page.getByRole('button', { name: '切换应用', exact: true });
  if (await desktop.isVisible()) {
    await desktop.click();
    await page.getByRole('menuitem', { name: '人员', exact: true }).click();
  } else {
    await page.getByRole('button', { name: '打开导航', exact: true }).click();
    await page.getByRole('button', { name: '人员', exact: true }).click();
    await page.getByRole('button', { name: '员工列表', exact: true }).click();
  }
  await expect(page.getByRole('heading', { name: '员工', exact: true })).toBeVisible();
  const date = page.getByRole('textbox', { name: /^业务日期/ });
  await date.fill('2027-04-01');
  await date.press('Tab');
  const search = page.getByRole('textbox', { name: '姓名或人员编号' });
  await search.fill(sample.name);
  await search.press('Enter');
  await expect(page).toHaveURL((url) => url.searchParams.has('ref'));
  await expect(page.getByText('1 份任职，1 人', { exact: true })).toBeVisible();
  const row = page.locator('a[data-object-link]').filter({ hasText: sample.name });
  await row.focus();
  await page.keyboard.press('Enter');
  await expect(page.getByRole('tab', { name: '职务信息' })).toHaveAttribute('aria-selected', 'true');
}
async function assignmentJourney(page: Page, sample: Awaited<ReturnType<typeof seed>>) {
  const { locations } = await hireAtLocation(page, sample);
  await openEmployeeAssignment(page,sample);
  await expect(page.getByText(new RegExp(locations[0].name))).toBeVisible();
  await page.getByRole('button', { name: '新增记录', exact: true }).click();
  const effective = page.getByRole('textbox', { name: /^生效日期/ });
  await effective.fill('2027-05-01');
  await effective.press('Tab');
  const changeTo = async (name: string) => {
    const loc = page.getByRole('combobox', { name: /工作地点/ });
    await loc.fill(name);
    await page.getByRole('option', { name, exact: true }).click();
    await page.getByRole('button', { name: '保存', exact: true }).click();
  };
  await changeTo(locations[1].name);
  await expect(page.getByText('职务信息已保存。')).toBeVisible();
  await page.getByRole('button', { name: '数据更改 · 2027-05-01', exact: true }).click();
  await page.getByRole('button', { name: '更正记录', exact: true }).click();
  await expect(page.getByRole('combobox', { name: /工作地点/ })).toHaveValue(locations[1].name);
  await changeTo(locations[2].name);
  await page.getByRole('button', { name: '数据更改 · 2027-05-01', exact: true }).click();
  const revisions = page.getByRole('button', { name: /查看修订/ });
  await expect(revisions).toHaveCount(2);
  await revisions.last().click();
  await expect(page.getByText(locations[0].name + ' → ' + locations[1].name, { exact: true })).toBeVisible();
  await expect
    .poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1))
    .toBe(true);
  await page.getByRole('link', { name: '返回列表', exact: true }).click();
  await expect(row).toBeFocused();
  await expect(search).toHaveValue(sample.name);
}
for (const [width, height] of [
  [1440, 900],
  [1440, 540],
  [390, 844],
  [320, 900],
  [320, 256],
]) {
  test(`M04 职务信息地点更改与原笔纠错 ${width}x${height}`, async ({ page }) => {
    test.setTimeout(150000);
    await signIn(page);
    const sample = await seed(page);
    await page.setViewportSize({ width, height });
    await assignmentJourney(page, sample);
  });
}
test('M04 职务信息地点更改与原笔纠错 原生200%', async () => {
  await native200(assignmentJourney);
});

async function openRegister(page: Page) {
  const desktop = page.getByRole('button', { name: '切换应用', exact: true });
  if (await desktop.isVisible()) {
    await desktop.click();
    await page.getByRole('menuitem', { name: '人员', exact: true }).click();
    await page.getByRole('button', { name: '人事事务', exact: true }).click();
    await page.getByRole('menuitem', { name: '办理记录', exact: true }).click();
  } else {
    await page.getByRole('button', { name: '打开导航', exact: true }).click();
    const personnel = page.getByRole('button', { name: '人员', exact: true });
    if ((await personnel.getAttribute('aria-expanded')) !== 'true') await personnel.click();
    const transactions = page.getByRole('button', { name: '人事事务', exact: true });
    if ((await transactions.getAttribute('aria-expanded')) !== 'true') await transactions.click();
    await page.getByRole('button', { name: '办理记录', exact: true }).click();
  }
  await expect(page.getByRole('heading', { name: '办理记录', exact: true })).toBeVisible();
}
async function registerJourney(page: Page, sample: Awaited<ReturnType<typeof seed>>) {
  const { locations, hire } = await hireAtLocation(page, sample);
  const view = await request(
    page,
    'GET',
    '/api/v1/assignments/' + hire.assignment + '?businessAt=2027-04-01',
  );
  const created = await request(
    page,
    'POST',
    '/api/v1/assignments/' + hire.assignment + '/location-changes',
    { businessDate: '2027-05-01', expectedSourceVersion: view.body.sourceVersion, location: locations[1].id },
    randomUUID(),
  );
  expect(created.status, created.body.message).toBe(200);
  await openRegister(page);
  const search = page.getByRole('textbox', { name: '姓名或人员编号' });
  await search.fill(sample.name);
  await search.press('Enter');
  await expect(page).toHaveURL((url) => url.searchParams.has('ref'));
  await expect(page.getByText('2 笔办理', { exact: true })).toBeVisible();
  await page.getByText('筛选条件', { exact: true }).click();
  await page.getByLabel('办理类型', { exact: true }).selectOption('DataChange');
  await page.getByLabel('生效起日', { exact: true }).fill('2027-05-01');
  await page.getByLabel('生效止日', { exact: true }).fill('2027-05-01');
  await page.getByRole('button', { name: '应用筛选' }).click();
  await expect(page.getByText('1 笔办理', { exact: true })).toBeVisible();
  const row = page.locator('a[data-object-link]').filter({ hasText: '数据更改' });
  await row.focus();
  await page.keyboard.press('Enter');
  await expect(page).toHaveURL((url) => url.pathname.endsWith('/' + created.body.action));
  await expect(page.getByRole('heading', { name: '数据更改原件', exact: true })).toBeVisible();
  await expect(page.getByText(sample.department, { exact: true })).toBeVisible();
  await page.getByRole('button', { name: '更正记录', exact: true }).click();
  const loc = page.getByRole('combobox', { name: /工作地点/ });
  await expect(loc).toHaveValue(locations[1].name);
  await loc.fill(locations[2].name);
  await page.getByRole('option', { name: locations[2].name, exact: true }).click();
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.getByText(locations[0].name + ' → ' + locations[2].name, { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: /查看修订/ })).toHaveCount(2);
  await page.getByRole('link', { name: '查看任职', exact: true }).click();
  await page.getByRole('button', { name: '数据更改 · 2027-05-01', exact: true }).click();
  await expect(page.getByText(locations[0].name + ' → ' + locations[2].name, { exact: true })).toBeVisible();
  await page.getByRole('link', { name: '返回办理明细', exact: true }).click();
  await expect(page.getByRole('heading', { name: '数据更改原件', exact: true })).toBeVisible();
  await page.getByRole('link', { name: '返回办理记录', exact: true }).click();
  await expect(row).toBeFocused();
  await expect(search).toHaveValue(sample.name);
  await expect(page.getByText('1 笔办理', { exact: true })).toBeVisible();
  await expect
    .poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1))
    .toBe(true);
}
for (const [width, height] of [
  [1440, 900],
  [1440, 540],
  [390, 844],
  [320, 900],
  [320, 256],
]) {
  test(`M04 办理记录查询和原笔更正 ${width}x${height}`, async ({ page }) => {
    test.setTimeout(150000);
    await signIn(page);
    const sample = await seed(page);
    await page.setViewportSize({ width, height });
    await registerJourney(page, sample);
  });
}
test('M04 办理记录查询和原笔更正 原生200%', async () => {
  await native200(registerJourney);
});

test('M04 办理记录中的未来首单撤销后原件仍占一行', async ({ page }) => {
  test.setTimeout(150000);
  await signIn(page);
  const sample = await seed(page);
  await hireAtLocation(page, sample);
  await openRegister(page);
  const search = page.getByRole('textbox', { name: '姓名或人员编号' });
  await search.fill(sample.name);
  await search.press('Enter');
  await expect(page).toHaveURL((u) => u.searchParams.has('ref'));
  await expect(page.getByText('1 笔办理', { exact: true })).toBeVisible();
  const row = page.locator('a[data-object-link]').filter({ hasText: '入职' });
  await row.click();
  await page.getByRole('link', { name: '撤销未来入职' }).click();
  await expect(page.getByRole('heading', { name: '撤销未来入职', exact: true })).toBeVisible();
  await page.getByRole('textbox', { name: /^撤销说明/ }).fill('任用通知在到岗前撤回');
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await page.getByRole('button', { name: '查看撤销依据' }).click();
  await expect(page.getByText('任用通知在到岗前撤回', { exact: true })).toBeVisible();
  await page.getByRole('link', { name: '返回办理记录' }).click();
  await expect(row).toBeFocused();
  await expect(page.getByText('1 笔办理', { exact: true })).toBeVisible();
  await expect(page.getByRole('gridcell', { name: '已撤销', exact: true })).toBeVisible();
});

async function historyJourney(page:Page,sample:Awaited<ReturnType<typeof seed>>){
  const {locations}=await hireAtLocation(page,sample);
  const destination='产品部'+randomUUID().slice(0,6);
  await apiDepartment(page,sample.buId,{name:destination});
  const policy=await request(page,'GET','/api/v1/assignment-policies');
  if(!policy.body.rows.some((r:{purpose:string;category:string})=>r.purpose==='transfer'&&r.category==='Employee')){
    const change=await request(page,'POST','/api/v1/assignment-policies',{
      expectedVersion:policy.body.version,row:{id:randomUUID(),category:'Employee',purpose:'transfer',mode:'JobOnly',from:'2027-04-01'},
    },randomUUID());
    expect(change.status,change.body.message).toBe(200);
  }
  await openEmployeeAssignment(page,sample);
  const button=(name:string)=>page.getByRole('button',{name,exact:true});
  async function date(label:string,value:string){
    const input=page.getByRole('textbox',{name:new RegExp('^'+label)});
    await input.fill(value);await input.press('Tab');
  }
  await button('新增记录').click();
  await page.getByRole('combobox',{name:/办理类型/}).click();
  await page.getByRole('option',{name:'调动',exact:true}).click();
  await date('生效日期','2027-07-01');
  await page.getByRole('combobox',{name:/调入部门/}).fill(destination);
  await page.getByRole('option',{name:destination,exact:true}).click();
  await expect(page.getByRole('combobox',{name:/拟任职务/})).toHaveValue(sample.job);
  await page.getByLabel(/调动依据/).fill('合成七月跨部门调令');
  await button('保存').click();
  await expect(button('调动 · 2027-07-01')).toBeVisible();
  await button('调动 · 2027-07-01').click();
  await expect(page.getByRole('heading',{name:'调动原件'})).toBeVisible();
  await expect(page.getByText('合成七月跨部门调令',{exact:true})).toBeVisible();
  await button('插入记录').click();
  await date('生效日期','2027-06-01');
  await page.getByRole('combobox',{name:/工作地点/}).fill(locations[1].name);
  await page.getByRole('option',{name:locations[1].name,exact:true}).click();
  await button('保存').click();
  await expect(button('数据更改 · 2027-06-01')).toBeVisible();
  await date('查看业务日','2027-07-01');
  await expect(page.getByText([destination,sample.job,locations[1].name].join(' · '),{exact:true})).toBeVisible();
  await date('查看业务日','2027-06-01');
  await button('数据更改 · 2027-06-01').click();
  await button('调整生效日期').click();
  await date('生效日期','2027-06-10');
  await button('保存').click();
  await button('数据更改 · 2027-06-10').click();
  await button('删除记录').click();
  await expect(page.getByText([sample.department,sample.job,locations[0].name].join(' · '),{exact:true})).toBeVisible();
  await button('数据更改 · 2027-06-10').click();
  await expect(page.getByText('已删除',{exact:true})).toBeVisible();
  await expect(button('删除记录')).toHaveCount(0);
  await expect(page.getByRole('button',{name:/查看修订/})).toHaveCount(3);
  await date('查看业务日','2027-07-01');
  await expect(page.getByText([destination,sample.job,locations[0].name].join(' · '),{exact:true})).toBeVisible();
  await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
}
for(const [width,height] of [[1440,900],[1440,540],[390,844],[320,900],[320,256]]){
  test(`M04 历史插入与按职务调动 ${width}x${height}`,async({page})=>{
    test.setTimeout(160000);
    await signIn(page);const sample=await seed(page);await page.setViewportSize({width,height});
    await historyJourney(page,sample);
  });
}
test('M04 历史插入与按职务调动 原生200%',async()=>{await native200(historyJourney)});

async function concurrentJourney(page: Page, sample: Awaited<ReturnType<typeof seed>>) {
  const { hire } = await hireAtLocation(page, sample);
  const policy = await request(page, 'GET', '/api/v1/assignment-policies');
  if (
    !policy.body.rows.some(
      (r: { purpose: string; category: string }) => r.purpose === 'secondary' && r.category === 'Employee',
    )
  ) {
    const saved = await request(
      page,
      'POST',
      '/api/v1/assignment-policies',
      {
        expectedVersion: policy.body.version,
        row: {
          id: randomUUID(),
          category: 'Employee',
          purpose: 'secondary',
          mode: 'JobOnly',
          from: '2027-04-01',
        },
      },
      randomUUID(),
    );
    expect(saved.status, saved.body.message).toBe(200);
  }
  const newDept = '培训部' + randomUUID().slice(0, 5);
  const training = await apiDepartment(page, sample.buId, { name: newDept });
  const employerName = '培训公司' + randomUUID().slice(0, 6);
  const countries = await request(page, 'GET', '/api/v1/country-regions?q=CN');
  const country = countries.body.items.find((r: { code: string }) => r.code === 'CN');
  const entity = await request(
    page,
    'POST',
    '/api/v1/legal-entities',
    {
      code: 'P' + randomUUID().slice(0, 7),
      name: employerName,
      countryRegionId: country.id,
      subjectType: 'legal_person',
      registrationType: '登记证',
      registrationNumber: 'PAR-' + randomUUID().slice(0, 8),
      operationType: 'CREATE',
      operationReason: 'NEW_LEGAL_ENTITY',
    },
    randomUUID(),
  );
  expect(entity.status, entity.body.message).toBe(201);
  const purpose = await request(
    page,
    'POST',
    '/api/v1/legal-entities/' + entity.body.legalEntityId + '/purposes',
    {
      expectedVersion: '',
      row: {
        id: randomUUID(),
        role: 'employer',
        kind: 'declaration',
        basis: '培训公司独立聘任资格',
        from: '2027-04-01',
      },
    },
    randomUUID(),
  );
  expect(purpose.status, purpose.body.message).toBe(200);
  await openEmployeeAssignment(page, sample);
  const button = (name: string) => page.getByRole('button', { name, exact: true });
  const go = async (name: string) => page.getByRole('link', { name, exact: true }).click();
  const choose = async (label: string, target: string) => {
    await page.getByRole('combobox', { name: new RegExp('^' + label) }).fill(target);
    await page.getByRole('option', { name: target, exact: true }).click();
  };
  await go('新增兼岗');
  await page.getByLabel(/任用依据/).fill('放弃本次草稿');
  await go('取消');
  await button('放弃修改').click();
  await expect(page.getByRole('heading', { name: new RegExp('0 号主岗') })).toBeVisible();
  await go('新增兼岗');
  await choose('部门', sample.department);
  await choose('职务', sample.job);
  await page.getByLabel(/任用依据/).fill('兼任部门评审员');
  await button('保存').focus();
  await page.keyboard.press('Enter');
  await expect(page.getByRole('heading', { name: new RegExp('1 号兼岗') })).toBeVisible();
  await expect(page.getByRole('tab', { name: '个人资料' })).toHaveCount(0);
  await expect(page.getByRole('tab', { name: /个人日志/ })).toHaveCount(0);
  await go('新增兼岗');
  await page.getByRole('combobox', { name: /所属关系/ }).click();
  await page.getByRole('option', { name: '新增并行工作关系', exact: true }).click();
  await choose('用人单位', employerName);
  await choose('部门', newDept);
  await choose('职务', sample.job);
  await page.getByLabel(/任用依据/).fill('培训关系独立聘任');
  await button('保存').click();
  await expect(page.getByRole('heading', { name: new RegExp('2 号兼岗') })).toBeVisible();
  await expect(page.getByText('本工作关系主任职', { exact: true })).toBeVisible();
  await expect(page.getByRole('tab', { name: '个人资料' })).toHaveCount(0);
  await expect
    .poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1))
    .toBe(true);
  const link = page.getByRole('link', { name: /返回/ }).first();
  await link.click();
  await expect(page.getByText('3 份任职，1 人', { exact: true })).toBeVisible();
  await expect(page.locator('a[data-object-link]').filter({ hasText: sample.name })).toHaveCount(3);
  return { department: training.id, primary: hire.assignment };
}
for (const [width, height] of [
  [1440, 900],
  [1440, 540],
  [390, 844],
  [320, 900],
  [320, 256],
]) {
  test(`M04 新增兼岗与并行工作关系 ${width}x${height}`, async ({ page }) => {
    test.setTimeout(180000);
    await signIn(page);
    const sample = await seed(page);
    await page.setViewportSize({ width, height });
    await concurrentJourney(page, sample);
  });
}
test('M04 新增兼岗与并行工作关系 原生200%', async () => {
  await native200(async (page, sample) => {
    await concurrentJourney(page, sample);
  });
});

test('M04 仅见兼岗的读者从员工目录直达获权职务信息', async ({ page, browser }) => {
  test.setTimeout(150000);
  await signIn(page);
  const sample = await seed(page);
  const issued = await concurrentJourney(page, sample);
  const sql = database(),
    grants = ['directory', 'read', 'history'].map((action) => ({ id: randomUUID(), action }));
  const reader = await browser.newContext({ baseURL: localConnection().webOrigin });
  try {
    for (const grant of grants)
      sql(`INSERT INTO access.employment_grant(instance_id,principal_id,grant_id,action,person_id,department_id,fields)
      SELECT instance_id,principal_id,'${grant.id}','${grant.action}','${sample.id}','${issued.department}',ARRAY['arrangement','basis','location']
      FROM access.principal WHERE login='reader'`);
    const screen = await reader.newPage();
    await signIn(screen, 'reader');
    await openEmployeeAssignment(screen, sample);
    await expect(screen.getByRole('heading', { name: /2 号兼岗/ })).toBeVisible();
    await expect(screen.getByRole('tab', { name: '个人资料' })).toHaveCount(0);
    await expect(screen.getByRole('link', { name: '新增兼岗' })).toHaveCount(0);
    const hidden = await request(
      screen,
      'GET',
      '/api/v1/assignments/' + issued.primary + '?businessAt=2027-04-01',
    );
    expect(hidden.status).toBe(403);
  } finally {
    for (const grant of grants) sql(`DELETE FROM access.employment_grant WHERE grant_id='${grant.id}'`);
    await reader.close();
  }
});

async function probationJourney(page:Page,sample:Awaited<ReturnType<typeof seed>>){
  await hireAtLocation(page,sample);await openEmployeeAssignment(page,sample);
  const button=(name:string)=>page.getByRole('button',{name,exact:true});
  const add=async(kind:string,day:string)=>{
    await button('新增记录').click();
    await page.getByRole('combobox',{name:/办理类型/}).click();
    await page.getByRole('option',{name:kind,exact:true}).click();
    const date=page.getByRole('textbox',{name:/^生效日期/});await date.fill(day);await date.press('Tab');
  };
  await add('试用开始','2027-04-09');
  await page.getByLabel(/试用依据/).fill('实际试用开始通知');
  await button('保存').focus();await page.keyboard.press('Enter');
  await expect(page.getByText('试用中',{exact:true})).toBeVisible();
  await button('试用开始 · 2027-04-09').click();
  await expect(page.getByRole('heading',{name:'试用开始原件',exact:true})).toBeVisible();
  await add('转正','2027-07-01');await expect(page.getByText('2027-04-09',{exact:true}).last()).toBeVisible();
  await page.getByLabel(/转正依据/).fill('试用转正通知');await button('保存').click();
  await expect(page.getByText('已转正',{exact:true})).toBeVisible();await button('转正 · 2027-07-01').click();
  await expect(page.getByRole('heading',{name:'转正原件',exact:true})).toBeVisible();
  await expect(page.getByText('试用转正通知',{exact:true})).toBeVisible();
  await openRegister(page);
  const search=page.getByRole('textbox',{name:'姓名或人员编号'});await search.fill(sample.name);await search.press('Enter');
  const record=page.locator('a[data-object-link]').filter({hasText:'转正'});
  await expect(record).toHaveCount(1);await record.focus();await page.keyboard.press('Enter');
  await expect(page.getByRole('heading',{name:'转正 · 工作关系',exact:true})).toBeVisible();
  await expect(page.getByText('试用转正通知',{exact:true})).toBeVisible();
  await page.getByRole('link',{name:'返回办理记录'}).click();
  await expect(record).toBeFocused();
  await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
}
for(const [width,height] of [[1440,900],[1440,540],[390,844],[320,900],[320,256]]){
  test(`M04 真实试用开始与关系转正 ${width}x${height}`,async({page})=>{
    test.setTimeout(150000);await signIn(page);const sample=await seed(page);
    await page.setViewportSize({width,height});await probationJourney(page,sample);
  });
}
test('M04 真实试用开始与关系转正 原生200%',async()=>{await native200(probationJourney)});
