import { chromium } from '@playwright/test';
import { randomUUID } from 'node:crypto';
import { existsSync } from 'node:fs';
import { execFileSync, spawn } from 'node:child_process';
import { localConnection } from '../../scripts/local-runtime.mjs';
import { test, expect, type Page } from './fixtures';
import { request, apiBU, apiDepartment } from './dated-object-support';
import { database } from './community-list-support.mjs';
import { namedMenu308, seed308 } from './staffing308-support';
import { seedPosition } from './staffing312-support';

async function signIn(page: Page, login = 'admin') {
  await page.goto('/');
  await page.getByLabel('登录名').fill(login);
  await page.getByLabel('密码').fill('admin1234567');
  await page.getByRole('button', { name: '登录', exact: true }).click();
  await expect(page.getByRole('button', { name: '退出', exact: true })).toBeVisible();
}
async function seed(page: Page, withDocument = true, employerFrom='2027-04-01') {
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
        from: employerFrom,
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
async function native200(run: (page: Page, sample: Awaited<ReturnType<typeof seed>>) => Promise<void>,prepare:(page:Page)=>ReturnType<typeof seed>=seed) {
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
    const sample = await prepare(page);
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
async function openEmployeeAssignment(page: Page,sample:Awaited<ReturnType<typeof seed>>,businessAt='2027-04-01'){
  const desktop = page.getByRole('button', { name: '切换应用', exact: true });
  if (await desktop.isVisible()) {
    await desktop.click();
    await page.getByRole('menuitem', { name: '人员', exact: true }).click();
  } else {
    await page.getByRole('button', { name: '打开导航', exact: true }).click();
    const personnel = page.getByRole('button', { name: '人员', exact: true });
    if ((await personnel.getAttribute('aria-expanded')) !== 'true') await personnel.click();
    await page.getByRole('button', { name: '员工列表', exact: true }).click();
  }
  await expect(page.getByRole('heading', { name: '员工', exact: true })).toBeVisible();
  const date = page.getByRole('textbox', { name: /^业务日期/ });
  await date.fill(businessAt);
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
  return { row, search };
}
async function assignmentJourney(page: Page, sample: Awaited<ReturnType<typeof seed>>) {
  const { locations } = await hireAtLocation(page, sample);
  const { row, search } = await openEmployeeAssignment(page,sample);
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
  await expect(page.getByText('1 份任职，1 人', { exact: true })).toBeVisible({timeout:15000});
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
    const transactions = page.getByRole('button', { name: '人事事务', exact: true });
    if (!(await transactions.isVisible())) {
      await desktop.click();
      await page.getByRole('menuitem', { name: '人员', exact: true }).click();
      await expect(page.getByRole('heading', { name: /^(员工|办理记录|人员档案)$/ }).first()).toBeVisible();
    }
    await transactions.focus();
    await page.keyboard.press('Enter');
    const register = page.getByRole('menuitem', { name: '办理记录', exact: true });
    await register.focus();
    await page.keyboard.press('Enter');
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

async function promotionJourney(page: Page, sample: Awaited<ReturnType<typeof seed>>, mode: 'forward' | 'reverse' | 'pair' = 'forward') {
  const suffix = randomUUID().slice(0, 5);
  const p5 = await seed308(page, 'grade', undefined, {name:'P5-' + suffix});
  const p6 = await seed308(page, 'grade', undefined, {name:'P6-' + suffix});
  const detail = await request(page,'GET','/api/v1/jobs/'+sample.jobId);
  const record = detail.body.selectedRecord;
  const accepted = await request(page,'POST','/api/v1/jobs/'+sample.jobId+'/records/'+record.recordId+'/correct',{
    expectedSourceVersion:detail.body.headSourceVersion,expectedRecordRevision:record.revision,changes:{gradeIds:mode === 'pair' ? [p5.id] : [p5.id,p6.id]},
  },randomUUID());
  expect(accepted.status,accepted.body.message).toBe(200);
  await hireAtLocation(page,sample);
  const dest = '产品部' + suffix;
  await apiDepartment(page,sample.buId,{name:dest});
  const policy=await request(page,'GET','/api/v1/assignment-policies');
  if (!policy.body.rows.some((r:{category:string;purpose:string})=>r.category==='Employee' && r.purpose==='transfer')) {
    const result=await request(page,'POST','/api/v1/assignment-policies',{expectedVersion:policy.body.version,
      row:{id:randomUUID(),category:'Employee',purpose:'transfer',mode:'JobOnly',from:'2027-04-01'}},randomUUID());
    expect(result.status,result.body.message).toBe(200);
  }
  await openEmployeeAssignment(page,sample);
  const date='2027-07-01';
  const button=(name:string)=>page.getByRole('button',{name,exact:true});
  const choose=async(label:string,value:string)=>{
    await page.getByRole('combobox',{name:new RegExp('^'+label)}).fill(value);
    await page.getByRole('option',{name:value,exact:true}).click();
  };
  const add=async(kind:string,day:string)=>{
    await button('新增记录').click();
    await page.getByRole('combobox',{name:/办理类型/}).click();
    await page.getByRole('option',{name:kind,exact:true}).click();
    const field=page.getByRole('textbox',{name:/^生效日期/});await field.fill(day);await field.press('Tab');
  };
  const prior=async(label:string)=>{
    await page.getByRole('combobox',{name:/通知载明的前序办理/}).click();
    await page.getByRole('option',{name:label,exact:true}).click();
  };
  await add('补录实际职级','2027-04-01');
  await choose('实际任职级',p5.name);await page.getByLabel(/原职级任用依据/).fill('原任期职级聘书');
  await button('保存').click();await expect(page.getByText(p5.name,{exact:true})).toBeVisible();
  await add('试用开始','2027-04-02');
  await page.getByLabel(/试用依据/).fill('试用通知');await button('保存').click();
  await expect(page.getByText('试用中',{exact:true})).toBeVisible();
  await add('转正',date);await page.getByLabel(/转正依据/).fill('本关系转正通知');
  await button('保存').click();await expect(page.getByText('已转正',{exact:true})).toBeVisible();
  if (mode !== 'forward') {
    const family = await seed308(page,'jobfamily');
    const job = await seed308(page,'job',family.id,{name:'产品专员' + suffix,gradeIds:mode === 'pair' ? [p6.id] : [p5.id,p6.id]});
    const pair = mode === 'pair';
    await add(pair ? '晋升并调动' : '调动',date);
    await choose('调入部门',dest);
    await choose('拟任职务',job.name);
    await page.getByLabel(pair ? /晋升依据/ : /调动依据/).fill(pair ? '晋升及调动共同通知' : '独立调动通知');
    await prior('转正 · 本关系转正通知');
    if (pair) await choose('拟任职级',p6.name);
    await button('保存').click();
    await button('调动 · '+date).click();
    if (pair) {
      await button('查看通知来源').click();
      await expect(page.getByText('晋升 · '+date+' · 晋升及调动共同通知',{exact:true})).toBeVisible();
      await button('晋升 · '+date).click();
      await button('撤销晋升').click();
      await page.getByLabel(/撤销依据/).fill('核对原通知');
      await button('保存').click();
      await expect(page.getByRole('alert').filter({hasText:'职级在拟定职务或期间内不可用'})).toBeVisible();
    } else {
      await button('在本次调动前补录晋升').click();
      await choose('拟任职级',p6.name);
      await page.getByLabel(/晋升依据/).fill('研发晋升通知');
      await button('保存').click();
      await button('晋升 · '+date).click();
      await expect(page.getByText(sample.department,{exact:true}).first()).toBeVisible();
      await button('撤销晋升').click();
      await page.getByRole('combobox',{name:'引用通知'}).click();
      await page.getByRole('option',{name:'调动 · 独立调动通知',exact:true}).click();
      await button('保存').click();
      await button('晋升 · '+date).click();
      await button('查看撤销依据来源').click();
      await expect(page.getByText('调动 · '+date+' · 独立调动通知',{exact:true})).toBeVisible();
      await expect(page.getByText(p5.name,{exact:true})).toBeVisible();
    }
    return;
  }
  await add('晋升',date);await choose('拟任职级',p6.name);
  await page.getByLabel(/晋升依据/).fill('研发晋升通知');await prior('转正 · 本关系转正通知');
  await button('保存').focus();await page.keyboard.press('Enter');
  await expect(page.getByText(p6.name,{exact:true})).toBeVisible();
  await add('调动',date);await choose('调入部门',dest);await page.getByLabel(/调动依据/).fill('独立调动通知');
  await prior('晋升 · 研发晋升通知');await button('保存').click();
  await expect(button('调动 · '+date)).toBeVisible();
  await button('晋升 · '+date).click();await expect(page.getByText(p5.name+' → '+p6.name,{exact:true})).toBeVisible();
  await button('更正记录').click();await page.getByLabel('晋升原因',{exact:true}).fill('专业职级晋升');
  await button('保存').click();await button('晋升 · '+date).click();
  await expect(page.getByText('专业职级晋升',{exact:true})).toBeVisible();
  await openRegister(page);
  const name=page.getByRole('textbox',{name:'姓名或人员编号'});await name.fill(sample.name);await name.press('Enter');
  const promotion=page.locator('a[data-object-link]').filter({hasText:'晋升'});
  await expect(promotion).toHaveCount(1);await promotion.focus();await page.keyboard.press('Enter');
  await button('撤销晋升').click();await page.getByLabel(/撤销依据/).fill('核对原件，通知仅载调动');
  await button('保存').click();await expect(page.getByText('已撤销',{exact:true}).first()).toBeVisible();
  await button('查看撤销前来源').click();
  await expect(page.getByText(p6.name,{exact:true}).first()).toBeVisible();
  await button('返回当前职务信息').click();
  await page.getByRole('link',{name:'返回办理记录'}).click();await expect(promotion).toBeFocused();
  await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
}
for(const [width,height] of [[1440,900],[1440,540],[390,844],[320,900],[320,256]]) {
  test(`M04 同日转正晋升调动、从登记撤销 ${width}x${height}`,async({page})=>{
    test.setTimeout(210000);await signIn(page);const sample=await seed(page);
    await page.setViewportSize({width,height});await promotionJourney(page,sample);
  });
}
test('M04 同日转正晋升调动、从登记撤销 原生200%',async()=>{await native200(promotionJourney)});

for (const mode of ['pair','reverse'] as const) {
  test('M04 晋升共同办理与反序补录 '+mode,async({ page })=>{
    test.setTimeout(210000);
    await signIn(page);
    await promotionJourney(page,await seed(page),mode);
  });
}

async function lifecycleJourney(page:Page,sample:Awaited<ReturnType<typeof seed>>){
  await hireAtLocation(page,sample);
  const manager=await seed(page);
  await hireAtLocation(page,manager);
  await openEmployeeAssignment(page,sample);
  const button=(name:string)=>page.getByRole('button',{name,exact:true});
  const choose=async(label:string,name:string)=>{
    await page.getByRole('combobox',{name:new RegExp('^'+label)}).fill(name);
    await page.getByRole('option',{name,exact:true}).click();
  };
  const add=async(kind:string,label:string,day:string)=>{
    await button('新增记录').click();
    await page.getByRole('combobox',{name:/办理类型/}).click();
    await page.getByRole('option',{name:kind,exact:true}).click();
    const field=page.getByRole('textbox',{name:new RegExp('^'+label)});await field.fill(day);await field.press('Tab');
  };
  await add('实际汇报','生效日期','2027-05-01');
  await page.getByRole('combobox',{name:/经理任职/}).fill(manager.name);
  await page.getByRole('option',{name:new RegExp('^'+manager.name+' · 0 号')}).click();
  await button('保存').focus();await page.keyboard.press('Enter');
  await expect(page.getByText(manager.name+' · 0 号',{exact:true})).toBeVisible();
  await button('实际汇报 · 2027-05-01').click();
  await expect(page.getByRole('heading',{name:'实际汇报原件'})).toBeVisible();
  await add('离职','原通知日期','2027-08-31');
  await page.getByLabel(/退出依据/).fill('本关系最后在职日为八月三十一日');await button('保存').click();
  await button('离职 · 2027-08-31').click();await expect(page.getByRole('heading',{name:'离职原件'})).toBeVisible();
  await openRegister(page);
  const name=page.getByRole('textbox',{name:'姓名或人员编号'});await name.fill(sample.name);await name.press('Enter');
  const left=page.locator('a[data-object-link]').filter({hasText:'离职'});await expect(left).toHaveCount(1);
  await left.focus();await page.keyboard.press('Enter');
  await page.getByRole('link',{name:'办理再聘',exact:true}).click();
  await expect(page.getByRole('heading',{name:'办理再聘'})).toBeVisible();
  const day=page.getByRole('textbox',{name:/^到岗日/});await day.fill('2027-10-01');await day.press('Tab');
  for(const [label,value] of [['部门',sample.department],['职务',sample.job],['用人单位',sample.employer]])await choose(label,value);
  await page.getByLabel(/任用通知说明/).fill('十月重新聘任');
  await button('保存').click();
  await expect(page.getByText('已保存。',{exact:true})).toBeVisible();
  await expect(page.getByText(/再聘 ·/).first()).toBeVisible();
  await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
}
for(const [width,height] of [[1440,900],[1440,540],[390,844],[320,900],[320,256]]){
  test(`M04 实际汇报离职及原记录再聘 ${width}x${height}`,async({page})=>{
    test.setTimeout(180000);await signIn(page);const sample=await seed(page);
    await page.setViewportSize({width,height});await lifecycleJourney(page,sample);
  });
}
test('M04 实际汇报离职及原记录再聘 原生200%',async()=>{await native200(lifecycleJourney)});

async function positionHireJourney(page: Page, sample: Awaited<ReturnType<typeof seed>>) {
  const family = await seed308(page,'jobfamily');
  const position = await seedPosition(page,'position',family.id);
  const policyPath='/api/v1/assignment-policies';
  const before=await request(page,'GET',policyPath);
  const original=before.body.rows.find((r:{category:string;purpose:string;from:string})=>r.category==='Employee'&&r.purpose==='hire'&&r.from==='2027-04-01');
  expect(original.mode).toBe('JobOnly');
  const publish=await request(page,'POST',policyPath,{expectedVersion:before.body.version,row:{...original,mode:'PositionExempt'}},randomUUID());
  expect(publish.status,publish.body.message).toBe(200);
  const button=(name:string)=>page.getByRole('button',{name,exact:true});
  try {
    const number='按岗任用'+sample.name;
    const issued=await request(page,'POST','/api/v1/appointment-decisions',{source:randomUUID(),number,roster:false,
      rows:[{person:sample.id,arrival:'2027-04-01',employer:{id:sample.employerId,name:sample.employer}}]},randomUUID());
    expect(issued.status,issued.body.message).toBe(200);
    await openPerson(page,sample.name);
    await page.getByRole('link',{name:'办理入职',exact:true}).click();
    await page.getByRole('combobox',{name:'任用依据'}).click();
    await page.getByRole('option',{name:number,exact:true}).click();
    await page.getByRole('combobox',{name:/^岗位/}).fill(position.name);
    await page.getByRole('option',{name:position.name,exact:true}).click();
    await expect(page.getByText(position.department,{exact:true})).toBeVisible();
    await expect(page.getByText(position.job.name,{exact:true})).toBeVisible();
    await expect(button('保存')).toBeEnabled();await button('保存').focus();await page.keyboard.press('Enter');
    await expect(page.getByText('可见在职 1 人，任职 1 份，占编 0')).toBeVisible();
    await button('查看原办理').click();
    await expect(page.getByText(new RegExp(position.name+' · 免占编'))).toBeVisible();
    await namedMenu308(page,'岗位','职务配置','/settings/staffing/positions');
    await page.getByRole('textbox',{name:'搜索编码或名称'}).fill(position.code);await button('查询').click();
    await page.getByRole('link',{name:position.name,exact:true}).click();
    await page.getByRole('link',{name:'查看补员安排'}).click();
    const day=page.getByLabel('业务日期',{exact:true});
    await day.fill('2027-05-01');await day.press('Tab');
    await expect(page.getByText('1',{exact:true})).toBeVisible();
    await button('暂停补员').click();await page.getByLabel(/补员原因/).fill('五月保留已有任职');
    await button('保存').click();await expect(page.getByText('已保存补员安排。')).toBeVisible();
    await day.fill('2027-06-01');await day.press('Tab');await button('恢复补员').click();
    await page.getByLabel(/补员原因/).fill('六月重开');await button('保存').click();
    await expect(page.getByText('2027-06-01 · 恢复补员',{exact:true})).toBeVisible();
    await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
  } finally {
    const now=await request(page,'GET',policyPath);
    const applied=now.body.rows.find((r:{id:string})=>r.id===original.id);
    if(applied?.mode==='PositionExempt'&&now.body.version===publish.body.version){
      const restored=await request(page,'POST',policyPath,{expectedVersion:now.body.version,row:{...applied,mode:original.mode}},randomUUID());
      expect(restored.status,restored.body.message).toBe(200);
    }
  }
}
for(const [width,height] of [[1440,900],[1440,540],[390,844],[320,900],[320,256]]){
  test(`M04 按岗免占首单与岗位补员 ${width}x${height}`,async({page})=>{
    test.setTimeout(180000);await signIn(page);const sample=await seed(page);
    await page.setViewportSize({width,height});await positionHireJourney(page,sample);
  });
}
test('M04 按岗免占首单与岗位补员 原生200%',async()=>{await native200(positionHireJourney)});

async function positionTransferJourney(page: Page, sample: Awaited<ReturnType<typeof seed>>) {
  await hireAtLocation(page,sample);
  const family = await seed308(page,'jobfamily');
  const first = await seedPosition(page,'position',family.id);
  const actual = await seedPosition(page,'position',family.id);
  const policyPath='/api/v1/assignment-policies';
  let policy=await request(page,'GET',policyPath);
  const find=(rows: {id:string;category:string;purpose:string;from:string;mode:string}[]) =>
    rows.find(r=>r.category==='Employee'&&r.purpose==='transfer'&&r.from==='2027-04-01');
  if (!find(policy.body.rows)) {
    const added=await request(page,'POST',policyPath,{expectedVersion:policy.body.version,row:{id:randomUUID(),category:'Employee',purpose:'transfer',from:'2027-04-01',mode:'JobOnly'}},randomUUID());
    expect(added.status,added.body.message).toBe(200);
    policy=await request(page,'GET',policyPath);
  }
  const row=find(policy.body.rows)!;
  expect(row.mode).toBe('JobOnly');
  const published=await request(page,'POST',policyPath,{expectedVersion:policy.body.version,row:{...row,mode:'PositionExempt'}},randomUUID());
  expect(published.status,published.body.message).toBe(200);
  try {
    await openEmployeeAssignment(page,sample);
    const button=(name:string)=>page.getByRole('button',{name,exact:true});
    await button('新增记录').click();
    await page.getByRole('combobox',{name:/办理类型/}).click();
    await page.getByRole('option',{name:'调动',exact:true}).click();
    const day=page.getByRole('textbox',{name:/^生效日期/});
    await day.fill('2027-06-01');await day.press('Tab');
    const position=page.getByRole('combobox',{name:/调入岗位/});
    await position.fill(first.name);await page.getByRole('option',{name:first.name,exact:true}).click();
    await expect(page.getByText(first.department,{exact:true})).toBeVisible();
    await expect(page.getByText(first.job.name,{exact:true})).toBeVisible();
    await page.getByLabel(/调动依据/).fill('六月按岗调动通知');
    await button('保存').focus();await page.keyboard.press('Enter');
    const action=button('调动 · 2027-06-01');
    await action.click();await expect(page.getByRole('heading',{name:'调动原件'})).toBeVisible();
    const arrangement=(p:typeof actual)=>[p.department,p.job.name,p.name,'免占编'].join(' · ');
    await expect(page.getByText(arrangement(first),{exact:true})).toBeVisible();
    await button('更正记录').click();await expect(position).toHaveValue(first.name);
    await position.fill(actual.name);await page.getByRole('option',{name:actual.name,exact:true}).click();
    await button('保存').click();
    await expect(page.getByText(actual.name+' · 免占编',{exact:true})).toBeVisible();
    await action.click();
    await expect(page.getByText(arrangement(actual),{exact:true})).toBeVisible();
    const revisions=page.getByRole('button',{name:/查看修订/});
    await expect(revisions).toHaveCount(2);await revisions.last().click();
    await expect(page.getByText(arrangement(first),{exact:true})).toBeVisible();
    await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
  } finally {
    const current=await request(page,'GET',policyPath);
    if (current.body.version===published.body.version) {
      const applied=find(current.body.rows)!;
      const restored=await request(page,'POST',policyPath,{expectedVersion:current.body.version,row:{...applied,mode:row.mode}},randomUUID());
      expect(restored.status,restored.body.message).toBe(200);
    }
  }
}
for (const [width,height] of [[1440,900],[1440,540],[390,844],[320,900],[320,256]]) {
  test(`M04 按岗位调入并更正原岗 ${width}x${height}`,async({page})=>{
    test.setTimeout(160000);
    await signIn(page);const sample=await seed(page);await page.setViewportSize({width,height});
    await positionTransferJourney(page,sample);
  });
}
test('M04 按岗位调入并更正原岗 原生200%',async()=>{await native200(positionTransferJourney)});

async function recruitingJourney(page:Page,sample:Awaited<ReturnType<typeof seed>>) {
  const path='/api/v1/assignment-policies';
  const policy=await request(page,'GET',path);
  const row=policy.body.rows.find((r:{category:string;purpose:string})=>r.category==='Employee'&&r.purpose==='recruit');
  if (!row) {
    const added=await request(page,'POST',path,{expectedVersion:policy.body.version,row:{id:randomUUID(),category:'Employee',purpose:'recruit',mode:'PositionCounted',from:'2027-04-01'}},randomUUID());
    expect(added.status,added.body.message).toBe(200);
  } else expect(row.mode).toBe('PositionCounted');
  const family=await seed308(page,'jobfamily');
  const position=await seedPosition(page,'position',family.id);
  const quota=await request(page,'POST','/api/v1/staffing-quotas',{positionId:position.id,effectiveDate:'2027-04-01',quota:1,reason:'为本次招聘核定一人'},randomUUID());
  expect(quota.status,quota.body.message).toBe(200);
  await openPerson(page,sample.name);
  await page.getByRole('link',{name:'招聘录用',exact:true}).click();
  await page.getByRole('link',{name:'发出录用',exact:true}).click();
  await expect(page.getByRole('heading',{name:'发出录用',exact:true})).toBeVisible();
  const number='招聘通知'+sample.name;
  const text=(name:string)=>page.getByRole('textbox',{name:new RegExp('^'+name)});
  const day=async(name:string,value:string)=>{await text(name).fill(value);await text(name).press('Tab')};
  const button=(name:string)=>page.getByRole('button',{name,exact:true});
  const pick=async(label:string,value:string)=>{await page.getByRole('combobox',{name:new RegExp('^'+label)}).fill(value);await page.getByRole('option',{name:value,exact:true}).click()};
  await text('录用编号').fill(number);await expect(text('录用编号')).toHaveValue(number);
  await day('约定到岗日','2027-05-01');await day('安排起日','2027-04-01');
  await pick('用人单位',sample.employer);await pick('岗位',position.name);
  await expect(page.getByText(position.department,{exact:true})).toBeVisible();
  await day('业务日期','2027-04-01');await text('办理依据').fill('四月发出一份已签录用通知');
  await expect(button('保存')).toBeEnabled();await button('保存').focus();await page.keyboard.press('Enter');
  await expect(page.getByRole('heading',{name:number,exact:true})).toBeVisible();
  await expect(page.getByText(new RegExp(position.name+' · 预留 1 编'))).toBeVisible();
  await button('登记接受').click();await day('业务日期','2027-04-03');await text('办理依据').fill('候选人本人已签收接受');
  await button('保存').click();await expect(page.getByText('已接受',{exact:true})).toBeVisible();
  await button('2027-04-01 · 发出录用').click();await expect(page.getByText('已发出',{exact:true})).toBeVisible();
  await button('返回当前录用').click();await expect(page.getByText('已接受',{exact:true})).toBeVisible();
  await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
  return {number,position};
}
for(const [width,height] of [[1440,900],[1440,540],[390,844],[320,900],[320,256]]) {
  test(`M05 录用预留与本人接受 ${width}x${height}`,async({page})=>{
    test.setTimeout(160000);await signIn(page);const sample=await seed(page);
    await page.setViewportSize({width,height});await recruitingJourney(page,sample);
  });
}
test('M05 录用预留与本人接受 原生200%',async()=>{await native200(recruitingJourney)});

async function countedPromotionJourney(page: Page, sample: Awaited<ReturnType<typeof seed>>) {
  const p5=await seed308(page,'grade',undefined,{name:'P5-'+randomUUID().slice(0,8)});
  const job=await request(page,'GET','/api/v1/jobs/'+sample.jobId);
  const old=job.body.selectedRecord;
  const grades=await request(page,'POST','/api/v1/jobs/'+sample.jobId+'/records/'+old.recordId+'/correct',{
    expectedSourceVersion:job.body.headSourceVersion,expectedRecordRevision:old.revision,changes:{gradeIds:[p5.id]},
  },randomUUID());
  expect(grades.status,grades.body.message).toBe(200);
  await hireAtLocation(page,sample);
  const family=await seed308(page,'jobfamily');
  const target=await seedPosition(page,'position',family.id);
  const quota=await request(page,'POST','/api/v1/staffing-quotas',{positionId:target.id,effectiveDate:'2027-04-01',quota:1,reason:'按岗晋升核一编'},randomUUID());
  expect(quota.status,quota.body.message).toBe(200);
  const policy='/api/v1/assignment-policies';
  const rules=await request(page,'GET',policy);
  const oldRule=rules.body.rows.find((r:{category:string;purpose:string;from:string})=>r.category==='Employee'&&r.purpose==='transfer'&&r.from==='2027-04-01');
  const rule=oldRule??{id:randomUUID(),category:'Employee',purpose:'transfer',from:'2027-04-01'};
  const next=await request(page,'POST',policy,{expectedVersion:rules.body.version,row:{...rule,mode:'PositionCounted'}},randomUUID());
  expect(next.status,next.body.message).toBe(200);
  try {
    await openEmployeeAssignment(page,sample);
    const button=(name:string)=>page.getByRole('button',{name,exact:true});
    const add=async(kind:string,date:string)=>{
      await button('新增记录').click();
      await page.getByRole('combobox',{name:/办理类型/}).click();
      await page.getByRole('option',{name:kind,exact:true}).click();
      const day=page.getByRole('textbox',{name:/^生效日期/});await day.fill(date);await day.press('Tab');
    };
    const pick=async(label:string,name:string)=>{
      await page.getByRole('combobox',{name:new RegExp('^'+label)}).fill(name);
      await page.getByRole('option',{name,exact:true}).click();
    };
    await add('补录实际职级','2027-04-01');await pick('实际任职级',p5.name);
    await page.getByLabel(/原职级任用依据/).fill('原岗位个人任用级别');await button('保存').click();
    await expect(page.getByText(p5.name,{exact:true})).toBeVisible();
    await add('晋升并调动','2027-07-01');await pick('调入岗位',target.name);
    await expect(page.getByText(target.department,{exact:true})).toBeVisible();
    await pick('拟任职级',target.grade.name);
    await page.getByLabel(/晋升依据/).fill('同一份通知晋升并调入新岗');await button('保存').focus();await page.keyboard.press('Enter');
    await expect(page.getByText(target.name+' · 占编',{exact:true})).toBeVisible();
    await expect(page.getByText(target.grade.name,{exact:true})).toBeVisible();
    await button('调动 · 2027-07-01').click();
    await expect(page.getByText([target.department,target.job.name,target.name,'占编'].join(' · '),{exact:true})).toBeVisible();
    const usage=await request(page,'GET','/api/v1/positions/'+target.id+'/replenishment?businessAt=2027-07-01');
    expect(usage.body.occupied).toBe(1);expect(usage.body.reserved).toBe(0);
    await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
  } finally {
    const current=await request(page,'GET',policy);
    if(current.body.version===next.body.version) {
      const row=current.body.rows.find((r:{id:string})=>r.id===rule.id);
      const reset=await request(page,'POST',policy,{expectedVersion:current.body.version,row:{...row,mode:oldRule?.mode??'JobOnly'}},randomUUID());
      expect(reset.status,reset.body.message).toBe(200);
    }
  }
}
for(const [width,height] of [[1440,900],[1440,540],[390,844],[320,900],[320,256]]) {
  test(`M04 占编岗位晋升与调动 ${width}x${height}`,async({page})=>{
    test.setTimeout(160000);await signIn(page);const sample=await seed(page);
    await page.setViewportSize({width,height});await countedPromotionJourney(page,sample);
  });
}
test('M04 占编岗位晋升与调动 原生200%',async()=>{await native200(countedPromotionJourney)});

async function withCountedPolicies(page:Page,run:()=>Promise<void>,purposes=['hire','transfer']){
  const policy='/api/v1/assignment-policies', from='2027-04-01';
  const altered:{id:string; prior:string;revision:string}[]=[];
  try {
    for(const purpose of purposes){
      const current=await request(page,'GET',policy);
      const old=current.body.rows.find((r:{category:string;purpose:string;from:string})=>r.category==='Employee'&&r.purpose===purpose&&r.from===from);
      const rule=old??{id:randomUUID(),purpose,category:'Employee',from};
      if(rule.mode==='PositionCounted')continue;
      const applied=await request(page,'POST',policy,{expectedVersion:current.body.version,row:{...rule,mode:'PositionCounted'}},randomUUID());
      expect(applied.status,applied.body.message).toBe(200);
      const read=await request(page,'GET',policy);
      altered.push({id:rule.id,prior:old?.mode??'JobOnly',revision:read.body.rows.find((v:{id:string})=>v.id===rule.id).revision});
    }
    await run();
  } finally {
    for(const prev of altered.reverse()){
      const current=await request(page,'GET',policy);
      const r=current.body.rows.find((v:{id:string})=>v.id===prev.id);
      if(r?.revision!==prev.revision)continue;
      const saved=await request(page,'POST',policy,{expectedVersion:current.body.version,row:{...r,mode:prev.prior}},randomUUID());
      expect(saved.status,saved.body.message).toBe(200);
    }
  }
}
async function exchangeJourney(page: Page, first: Awaited<ReturnType<typeof seed>>) {
  const second=await seed(page);
  const family=await seed308(page,'jobfamily');
  const a=await seedPosition(page,'position',family.id);
  const b=await seedPosition(page,'position',family.id);
  const from='2027-04-01';
  await withCountedPolicies(page,async()=>{
    for(const [i,p] of [a,b].entries()){
      const quota=await request(page,'POST','/api/v1/staffing-quotas',{positionId:p.id,effectiveDate:from,quota:1,reason:'两岗各核一编'},randomUUID());
      expect(quota.status,quota.body.message).toBe(200);
      const employee=[first,second][i];
      const hire=await request(page,'POST','/api/v1/hires',{person:employee.id,employer:employee.employerId,position:p.id,arrival:from,notice:'正式在岗通知'},randomUUID());
      expect(hire.status,hire.body.message).toBe(200);
    }
    await openEmployeeAssignment(page,first);
    const button=(name:string)=>page.getByRole('button',{name,exact:true});
    const day=page.getByRole('textbox',{name:/^生效日期/});
    await button('新增记录').click();await page.getByRole('combobox',{name:/办理类型/}).click();
    await page.getByRole('option',{name:'同日对调',exact:true}).click();
    await day.fill('2027-05-01');await day.press('Tab');
    await page.getByRole('combobox',{name:/对调员工任职/}).fill(second.name);
    await page.getByRole('option',{name:new RegExp(second.name)}).click();
    await expect(page.getByText(a.name,{exact:true}).last()).toBeVisible();
    await expect(page.getByText(b.name,{exact:true}).last()).toBeVisible();
    await page.getByLabel(/对调依据/).fill('双方同日互换岗位');
    await button('保存').focus();await page.keyboard.press('Enter');
    await expect(page.getByText(b.name+' · 占编',{exact:true})).toBeVisible();
    await button('调动 · 2027-05-01').click();
    await button('查看另一份调动原件').click();
    await expect(page.getByText(['2027-05-01',b.name,a.name,'双方同日互换岗位'].join(' · '),{exact:true})).toBeVisible();
    await button('调整对调日期').click();await day.fill('2027-06-01');await day.press('Tab');
    await button('保存').click();await button('调动 · 2027-06-01').click();
    await button('撤销两岗对调').click();
    await expect(page.getByText(a.name+' · 占编',{exact:true})).toBeVisible();
    await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
  });
}

for(const [width,height] of [[1440,900],[1440,540],[390,844],[320,900],[320,256]]){
  test(`M04 同日对调两个满编岗 ${width}x${height}`,async({page})=>{
    test.setTimeout(160000);await signIn(page);const sample=await seed(page);
    await page.setViewportSize({width,height});await exchangeJourney(page,sample);
  });
}
test('M04 同日对调两个满编岗 原生200%',async()=>{await native200(exchangeJourney)});

async function successionJourney(page:Page, successor:Awaited<ReturnType<typeof seed>>){
  const outgoing=await seed(page), family=await seed308(page,'jobfamily');
  const a=await seedPosition(page,'position',family.id), b=await seedPosition(page,'position',family.id);
  await withCountedPolicies(page,async()=>{
    for(const [i,target] of [a,b].entries()){
      const p=[successor,outgoing][i];
      const quota=await request(page,'POST','/api/v1/staffing-quotas',{positionId:target.id,effectiveDate:'2027-04-01',quota:1,reason:'常规一人'},randomUUID());
      expect(quota.status,quota.body.message).toBe(200);
      const original=await request(page,'POST','/api/v1/hires',{person:p.id,employer:p.employerId,position:target.id,arrival:'2027-04-01',lastDay:i===1?'2027-04-10':undefined,notice:'原在岗安排'},randomUUID());
      expect(original.status,original.body.message).toBe(200);
    }
    await openEmployeeAssignment(page,successor);
    const button=(name:string)=>page.getByRole('button',{name,exact:true});
    const select=async(label:string,value:string)=>{await page.getByRole('combobox',{name:new RegExp(label)}).click();await page.getByRole('option',{name:value,exact:true}).click()};
    await button('新增记录').click();await select('办理类型','调动');
    await select('调动方式','内部接任');
    await page.getByRole('combobox',{name:/调入岗位/}).fill(b.name);
    await page.getByRole('option',{name:b.name,exact:true}).click();
    await page.getByRole('combobox',{name:/被接替任职/}).fill(outgoing.name);
    await page.getByRole('option',{name:new RegExp(outgoing.name)}).click();
    await expect(page.getByText('2027-04-01 起占交接名额，2027-04-11 起占常规名额')).toBeVisible();
    await page.getByLabel(/调动依据/).fill('已确定十日交接，十一日起接任');
    await button('保存').focus();await page.keyboard.press('Enter');
    await expect(page.getByText(b.name+' · 交接占编',{exact:true})).toBeVisible();
    await button('调动 · 2027-04-01').click();
    await expect(page.getByText(outgoing.name+' · 2027-04-01 起交接，2027-04-11 起占常规名额',{exact:true})).toBeVisible();
    const date=page.getByRole('textbox',{name:'查看业务日'});await date.fill('2027-04-11');await date.press('Tab');
    await expect(page.getByText(b.name+' · 占编',{exact:true})).toBeVisible();
    const usage=await request(page,'GET','/api/v1/positions/'+b.id+'/replenishment?businessAt=2027-04-10');
    expect([usage.body.occupied,usage.body.handoverOccupied,usage.body.totalOccupied,usage.body.reserved]).toEqual([1,1,2,0]);
    await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
  });
}
for(const [width,height] of [[1440,900],[1440,540],[390,844],[320,900],[320,256]]){
  test(`M04 内部接任十天交接 ${width}x${height}`,async({page})=>{
    test.setTimeout(160000);await signIn(page);const sample=await seed(page);
    await page.setViewportSize({width,height});await successionJourney(page,sample);
  });
}
test('M04 内部接任十天交接 原生200%',async()=>{await native200(successionJourney)});

async function dutyJourney(page:Page,sample:Awaited<ReturnType<typeof seed>>){
  const entry='2026-04-01', first='2026-04-05', last='2026-04-07', after='2026-04-08';
  const send=async(path:string,body:object,status=200)=>{
    const r=await request(page,'POST','/api/v1/'+path,body,randomUUID());expect(r.status,r.body.message).toBe(status);return r.body;
  };
  for(const purpose of ['hire','transfer']){
    const rules=(await request(page,'GET','/api/v1/assignment-policies')).body;
    const prior=rules.rows.find((r:{category:string;purpose:string;from:string})=>r.category==='Employee'&&r.purpose===purpose&&r.from===entry);
    if(prior){expect(prior.mode).toBe('PositionCounted');continue}
    await send('assignment-policies',{expectedVersion:rules.version,row:{id:randomUUID(),purpose,category:'Employee',mode:'PositionCounted',from:entry,until:'2026-05-01'}});
  }
  const family=await seed308(page,'jobfamily');
  const store=await seedPosition(page,'position',family.id),home=await seedPosition(page,'position',family.id);
  for(const [p,n] of [[store,3],[home,1]] as const){
    await send('staffing-quotas',{positionId:p.id,effectiveDate:entry,quota:n,reason:'四月门店核编'});
  }
  const hire=(person:string,position:string)=>send('hires',{person,employer:sample.employerId,position,arrival:entry,lastDay:'2026-04-30',notice:'四月正式在岗通知'});
  for(const name of ['早班','中班','晚班']){
    const id=(await send('persons',{submissionId:randomUUID(),name:name+randomUUID().slice(0,8),primaryDocument:{kind:'passport',number:randomUUID()}})).id;
    await hire(id,store.id);
  }
  await hire(sample.id,home.id);
  const number='门店已执行派令'+sample.name;
  await send('appointment-decisions',{source:randomUUID(),number,roster:false,rows:[{person:sample.id,employer:{id:sample.employerId,name:sample.employer},position:{id:store.id,name:store.name},arrival:first}]});
  await openEmployeeAssignment(page,sample,first);
  const button=(name:string)=>page.getByRole('button',{name,exact:true});
  const select=async(label:string,choice:string)=>{await page.getByRole('combobox',{name:new RegExp(label)}).click();await page.getByRole('option',{name:choice,exact:true}).click()};
  const date=async(label:string,date:string)=>{const field=page.getByRole('textbox',{name:new RegExp('^'+label)});await field.fill(date);await field.press('Tab')};
  await button('新增记录').click();await select('办理类型','调动');
  await page.getByRole('combobox',{name:/调入岗位/}).fill(store.name);
  await page.getByRole('option',{name:store.name,exact:true}).click();
  await page.getByLabel(/调动依据/).fill('常规拟调入');
  await expect(page.getByText(new RegExp(store.name+'.*缺 1 编')).last()).toBeVisible();
  await expect(button('保存')).toBeDisabled();
  await button('取消').click();await page.getByRole('dialog').getByRole('button',{name:/放弃/}).click();
  const enter=async()=>{
    await button('新增记录').click();await select('办理类型','补录实际履职');await select('履职资料','登记出勤原件');
    await select('派岗决定',number+' · '+store.name);
    await page.getByLabel(/出勤出具方/).fill('门店考勤');await page.getByLabel(/出勤原件编号/).fill('考勤'+sample.name);
    await date('最后履职日',last);await date('出勤核实至',after);
  };
  await enter();await button('取消').click();await page.getByRole('dialog').getByRole('button',{name:/放弃/}).click();
  const usage=async(day:string)=>(await request(page,'GET','/api/v1/positions/'+store.id+'/replenishment?businessAt='+day)).body;
  expect((await usage(first)).occupied).toBe(3);
  await enter();await button('保存').focus();await page.keyboard.press('Enter');
  await expect(page.getByText('核定 3 · 实占 4 · 超编 1',{exact:true})).toBeVisible();
  await button('调动 · '+first).click();
  await expect(page.getByText('门店考勤 · 考勤'+sample.name,{exact:true})).toBeVisible();
  await expect(page.getByText(first+' 至 '+last,{exact:true})).toBeVisible();
  for(const day of [first,'2026-04-06',last,after]){const v=await usage(day);expect([v.quota,v.occupied,v.excess]).toEqual(day===after?[3,3,0]:[3,4,1])}
  await date('查看业务日',after);await expect(page.getByText(home.name+' · 占编',{exact:true})).toBeVisible();
  await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
}
const seedDuty=(page:Page)=>seed(page,true,'2026-04-01');
for(const [width,height] of [[1440,900],[1440,540],[390,844],[320,900],[320,256]]){
  test(`M04 满编门店的三天实况补录 ${width}x${height}`,async({page})=>{
    test.setTimeout(150000);await signIn(page);const sample=await seedDuty(page);await page.setViewportSize({width,height});await dutyJourney(page,sample);
  });
}
test('M04 满编门店的三天实况补录 原生200%',async()=>{await native200(dutyJourney,seedDuty)});

async function contractJourney(page:Page,sample:Awaited<ReturnType<typeof seed>>){
  await hireAtLocation(page,sample);
  await openEmployeeAssignment(page,sample);
  const button=(name:string)=>page.getByRole('button',{name,exact:true});
  const day=async(label:string,value:string)=>{const field=page.getByRole('textbox',{name:new RegExp('^'+label)});await field.fill(value);await field.press('Tab')};
  const first='A-'+sample.name,second='B-'+sample.name;
  await button('订立合同').click();
  await page.getByLabel(/合同编号/).fill('未保存合同');
  await button('取消').click();await page.getByRole('dialog').getByRole('button',{name:'放弃修改'}).click();
  await expect(page.getByText('本关系尚无合同。',{exact:true})).toBeVisible();
  await button('订立合同').click();
  await page.getByLabel(/合同编号/).fill(first);
  await day('签订日','2027-03-25');await day('合同起日','2027-04-06');await day('合同至','2027-06-30');
  await button('保存合同').focus();await page.keyboard.press('Enter');
  await expect(page.getByText('2027-04-06 至 2027-06-30',{exact:true})).toBeVisible();
  await button('续订合同').click();await page.getByLabel(/合同编号/).fill(second);
  await day('签订日','2027-06-20');await day('合同至','2028-06-30');
  await button('保存合同').click();
  await expect(page.getByText('2027-07-01 至 2028-06-30',{exact:true})).toBeVisible();
  await button(first).last().click();
  await expect(page.getByText('2027-03-25',{exact:true})).toBeVisible();
  await button(second).last().click();
  await button('新增记录').click();await page.getByRole('combobox',{name:/办理类型/}).click();await page.getByRole('option',{name:'离职',exact:true}).click();
  await day('原通知日期','2027-08-31');await page.getByLabel(/退出依据/).fill('依通知终结本关系和续订合同');
  await page.getByRole('combobox',{name:/合同处置/}).click();await page.getByRole('option',{name:'按离职通知终结上述合同',exact:true}).click();
  await button('保存').click();
  await expect(page.getByText('职务信息已保存。',{exact:true})).toBeVisible();
  await button(second).first().click();
  await expect(page.getByText('2027-09-01',{exact:true}).last()).toBeVisible();
  await expect(page.getByText(/依通知终结本关系和续订合同 · admin/)).toBeVisible();
  await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
}
for(const [width,height] of [[1440,900],[1440,540],[390,844],[320,900],[320,256]]){
  test(`M04 工作关系合同续订与离职 ${width}x${height}`,async({page})=>{
    test.setTimeout(160000);await signIn(page);const sample=await seed(page);await page.setViewportSize({width,height});await contractJourney(page,sample);
  });
}
test('M04 工作关系合同续订与离职 原生200%',async()=>{await native200(contractJourney)});

async function offerHireJourney(page:Page,sample:Awaited<ReturnType<typeof seed>>) {
  await withCountedPolicies(page,async()=>{
    const {number,position}=await recruitingJourney(page,sample);
    const currentOffer=page.url();
    const usage=async(day:string)=>{
      const r=await request(page,'GET','/api/v1/positions/'+position.id+'/replenishment?businessAt='+day);
      expect(r.status,r.body.message).toBe(200);
      return [r.body.reserved,r.body.occupied];
    };
    const arrival='2027-05-07';
    const button=(name:string)=>page.getByRole('button',{name,exact:true});
    const date=async(label:string,value:string)=>{const input=page.getByRole('textbox',{name:new RegExp('^'+label)});await input.fill(value);await input.press('Tab')};
    await button('调整录用日期').click();
    await expect(page.getByRole('textbox',{name:/^约定到岗日/})).toHaveValue('2027-05-01');
    await date('约定到岗日',arrival);await date('业务日期','2027-04-04');
    await page.getByLabel(/办理依据/).fill('双方确认推迟六天到岗');await button('保存').click();
    await expect(page.getByText('已发出',{exact:true})).toBeVisible();
    await button('登记接受').click();await date('业务日期','2027-04-05');
    await page.getByLabel(/办理依据/).fill('本人接受新的到岗日');await button('保存').click();
    await expect(page.getByText('已接受',{exact:true})).toBeVisible();
    expect(await usage(arrival)).toEqual([1,0]);
    await button('按录用办理入职').focus();await page.keyboard.press('Enter');
    await expect(page.getByText('已入职',{exact:true})).toBeVisible();
    await page.getByRole('link',{name:'查看入职任职'}).click();
    await expect(page.getByText(position.name+' · 占编',{exact:true})).toBeVisible();
    expect(await usage('2027-04-30')).toEqual([1,0]);expect(await usage(arrival)).toEqual([0,1]);
    await button('入职 · '+arrival).click();
    await page.getByRole('link',{name:number,exact:true}).click();
    await expect(page.getByText('已接受',{exact:true})).toBeVisible();
    await expect(button('按录用办理入职')).toHaveCount(0);
    await openRegister(page);
    const search=page.getByRole('textbox',{name:'姓名或人员编号'});await search.fill(sample.name);await search.press('Enter');
    await expect(page.getByText('1 笔办理',{exact:true})).toBeVisible();
    await page.locator('a[data-object-link]').filter({hasText:'入职'}).click();
    await page.getByRole('link',{name:'撤销未来入职'}).click();
    await page.getByRole('textbox',{name:/^撤销说明/}).fill('本人已在到岗前撤回');
    await button('保存').click();
    await button('查看撤销依据').click();await expect(page.getByText('本人已在到岗前撤回',{exact:true})).toBeVisible();
    expect(await usage(arrival)).toEqual([0,0]);
    await page.goto(currentOffer);await expect(page.getByText('已发出',{exact:true})).toBeVisible();
    await button('重新预留').click();
    await date('业务日期','2027-04-15');
    await page.getByLabel(/办理依据/).fill('招聘重新取得该份额');await button('保存').click();
    await expect(page.getByText('招聘重新取得该份额',{exact:true})).toBeVisible();expect(await usage(arrival)).toEqual([1,0]);
    await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
  },['hire']);
}
for(const [width,height] of [[1440,900],[1440,540],[390,844],[320,900],[320,256]]){
  test(`T06 已接受录用转入职与未来撤销 ${width}x${height}`,async({page})=>{
    test.setTimeout(170000);await signIn(page);const sample=await seed(page);await page.setViewportSize({width,height});await offerHireJourney(page,sample);
  });
}
test('T06 已接受录用转入职与未来撤销 原生200%',async()=>{await native200(offerHireJourney)});
