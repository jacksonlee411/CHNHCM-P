import { desktopViewports } from './desktop-viewports.mjs';
import {openPersonnel} from './app-navigation';
import { chromium, type Locator } from '@playwright/test';
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
      (r: { category: string; purpose: string; from: string; until?: string }) => r.category === 'Employee' && r.purpose === 'hire' && r.from <= '2027-04-01' && (!r.until || '2027-04-01' < r.until),
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
  await openPersonnel(page);
  await page.getByLabel('姓名或人员编号').fill(name);
  await page.getByLabel('姓名或人员编号').press('Enter');
  await expect(page).toHaveURL((url) => url.pathname === '/personnel/employees' && url.searchParams.has('ref'));
  const target = page.locator('a[data-object-link]').filter({ hasText: name }).first();
  await expect(target).toBeVisible();
  await target.click();
  if(new URL(page.url()).pathname.includes('/assignments/')) await page.getByRole('tab',{name:'个人资料'}).click();
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
  await expect(page.getByRole('heading',{name:'本人办理历史'})).toBeVisible();
  await button('查看原办理').click();
  await expect(page.getByText('采用规则 T02-employee-1')).toBeVisible();
  await expect(page.getByText('任用决定 ' + ref.number)).toBeVisible();
  await expect(page.getByText('原单 ' + ref.source + '，修订 ' + ref.version)).toBeVisible();
  if (ref.line) await expect(page.getByText('名单行 ' + ref.line + '，修订 ' + ref.version)).toBeVisible();
  await expect(page.getByLabel('查询日期')).toHaveCount(0);
  expect((await request(page,'GET','/api/v1/persons/'+sample.id+'/employment?businessAt=2027-03-31')).body).toMatchObject({people:0,assignments:0,seats:0});
  await button('撤销未来入职').click();
  await page.getByRole('textbox', { name: /^撤销说明/ }).fill('任用单位于到岗前撤回通知');
  await button('保存').click();
  await expect(page.getByText(/入职（已撤销）/)).toBeVisible();
  await expect(page.getByLabel('查询日期')).toHaveCount(0);
  expect((await request(page,'GET','/api/v1/persons/'+sample.id+'/employment?businessAt=2027-04-01')).body).toMatchObject({people:0,assignments:0,seats:0});
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
  await expect(page.getByRole('heading',{name:'本人办理历史'})).toBeVisible();
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
  await expect(page.getByLabel('查询日期')).toHaveCount(0);
  expect((await request(page,'GET','/api/v1/persons/'+sample.id+'/employment?businessAt=2027-04-01')).body).toMatchObject({people:0,assignments:0,seats:0});
  const current = await request(page, 'GET', '/api/v1/persons/' + sample.id + '?view=edit');
  expect(current.body.documents).toHaveLength(1);
});
for (const [width, height] of desktopViewports) {
  test('T03 首单入职与未来撤销 ' + width + '×' + height, async ({ page }) => {
    test.setTimeout(110000);
    await signIn(page);
    const sample = await seed(page);
    await page.setViewportSize({ width, height });
    await journey(page, sample, height === 540, height === 900);
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
  await ensureJobOnlyPolicy(page, 'hire');
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
async function openEmployeeAssignment(page: Page,sample:Awaited<ReturnType<typeof seed>>,businessAt='2027-04-01', count=1, assignment?:string){
  await openPersonnel(page);
  await expect(page.getByRole('textbox',{name:'业务日期'})).toHaveCount(0);
  const search = page.getByRole('textbox', { name: '姓名或人员编号' });
  await search.fill(sample.name);
  const fetched = page.waitForResponse(response => {
    if (new URL(response.url()).pathname !== '/api/v1/personnel/directory/query') return false;
    const input = response.request().postDataJSON();
    return !!input?.ref && !input.businessAt && !input.anchor && response.ok();
  });
  await search.press('Enter');
  const loaded = await fetched;
  const ref = loaded.request().postDataJSON().ref as string;
  expect((await loaded.json()).keyword).toBe(sample.name);
  await expect(page).toHaveURL((url) => url.searchParams.get('ref') === ref);
  await expect(page.getByText(count + ' 份任职，1 人', { exact: true })).toBeVisible();
  const row = page.locator(assignment ? 'a[data-object-link][href*="/assignments/'+assignment+'"]' : 'a[data-object-link]').filter({ hasText: sample.name });
  await expect(row).toHaveAttribute('href',new RegExp('ref='+ref));
  await row.focus();
  await page.keyboard.press('Enter');
  await expect(page.locator('.object-detail-identity h1')).toHaveText(sample.name);
  if (count === 0) await page.getByRole('tab', { name: '职务信息', exact: true }).click();
  if(businessAt!=='2027-04-01'){
    await expect(page).toHaveURL(url => url.pathname.includes('/assignments/'));
    const projection=await request(page,'GET','/api/v1/assignments/'+new URL(page.url()).pathname.split('/').pop());
    const node=projection.body.effectiveDates.filter((d:string)=>d<=businessAt).at(-1);
    if(node)await chooseEffectiveDate(page,node);
  }
  return { row, search };
}
async function chooseEffectiveDate(page: Page, day: string) {
  const axis=page.getByRole('navigation',{name:'生效日期'});
  await expect(axis.locator('button[aria-current="date"]')).toBeVisible();
  const point=axis.locator('button[data-day="'+day+'"]');
  if(await point.isVisible()) await point.click();
  else {
    const compact=axis.locator('[data-date-menu="all"]');
    if(await compact.isVisible()) await compact.click();
    else {
      const first=await axis.locator('button[data-day]').first().getAttribute('data-day');
      await axis.locator(day<(first || '')?'[data-date-menu="before"]':'[data-date-menu="after"]').click();
    }
    await page.getByRole('menuitem',{name:day,exact:true}).click();
  }
  await expect(axis.locator('[aria-current=date]')).toContainText(day);
}
async function chooseRecordAction(page: Page, label: string) {
  const trigger = page.getByRole('button', {name: '更多操作', exact: true});
  await trigger.focus();
  await page.keyboard.press('Enter');
  const choice = page.getByRole('menuitem', {name: label, exact: true});
  await choice.focus();
  await page.keyboard.press('Enter');
}
// Delay the real read so it completes during a pointer gesture or while a filter is being edited.
async function holdRead(page: Page, match: (url: URL) => boolean) {
  let release!: () => void;
  const held = new Promise<void>(resolve => { release = resolve; });
  await page.route(match, async route => {
    await held;
    await route.continue();
  });
  page.once('close', release);
  return release;
}

async function clickWhileReadCompletes(button: Locator, release: () => void, loaded: Locator) {
  const mouse = button.page().mouse;
  try {
    await button.scrollIntoViewIfNeeded();
    const box = await button.boundingBox();
    expect(box).not.toBeNull();
    await mouse.move(box!.x + box!.width / 2, box!.y + box!.height / 2);
    await mouse.down();
    release();
    await expect(loaded).toBeVisible();
    await mouse.up();
  } finally {
    release();
  }
}

async function assignmentJourney(page: Page, sample: Awaited<ReturnType<typeof seed>>) {
  const { locations } = await hireAtLocation(page, sample);
  const { row, search } = await openEmployeeAssignment(page,sample);
  await expect(page.getByText(new RegExp(locations[0].name))).toBeVisible();
  const add = page.getByRole('button', { name: '新增记录', exact: true });
  await add.focus();
  await page.keyboard.press('Enter');
  await page.getByRole('button', { name:'取消', exact:true }).click();
  await expect(add).toBeFocused();
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
  await chooseRecordAction(page, '更正记录');
  const save = page.locator('.record-control').getByRole('button', {name:'保存', exact:true});
  await expect(save).toBeVisible();
  await expect(page.getByRole('link', {name:'新增兼岗'})).toHaveCount(0);
  if (await page.evaluate(() => innerWidth >= 900)) {
    const groups = page.locator('form .object-detail-columns > *');
    await expect(groups).toHaveCount(2);
    // Compare both columns in the same layout snapshot.
    const [left, right] = await groups.evaluateAll(columns => columns.map(column => {
      const { x, y, right } = column.getBoundingClientRect();
      return { x, y, right };
    }));
    expect(right.x).toBeGreaterThan(left.right);
    expect(Math.abs(right.y - left.y)).toBeLessThan(2);
  }
  await page.getByRole('button',{name:'取消',exact:true}).click();
  await expect(page.getByRole('button',{name:'更多操作',exact:true})).toBeFocused();
  await chooseRecordAction(page, '更正记录');
  await expect(page.getByRole('combobox', { name: /工作地点/ })).toHaveValue(locations[1].name);
  const releaseProfile = await holdRead(page, url => url.pathname === '/api/v1/persons/' + sample.id);
  await changeTo(locations[2].name);
  await page.getByRole('button', { name: '数据更改 · 2027-05-01', exact: true }).click();
  const revisions = page.getByRole('button', { name: /查看修订/ });
  await expect(revisions).toHaveCount(2);
  const firstRevision = page.getByRole('button', { name: /查看修订.*第 1 次$/ });
  await clickWhileReadCompletes(firstRevision, releaseProfile, page.getByRole('tab', {name:'个人资料'}));
  await expect(page.getByText(locations[0].name + ' → ' + locations[1].name, { exact: true })).toBeVisible();
  const versionURL = page.url();
  expect(new URL(versionURL).searchParams.get('adoptedRef')).toBeTruthy();
  await page.goBack();
  await expect(page.getByText(locations[0].name + ' → ' + locations[2].name, { exact: true })).toBeVisible();
  await page.goForward();
  await expect(page).toHaveURL(versionURL);
  await page.reload();
  await expect(page).toHaveURL(versionURL);
  await expect(page.getByText(locations[0].name + ' → ' + locations[1].name, { exact: true })).toBeVisible();
  await expect(page.getByRole('link',{name:'新增兼岗'})).toHaveCount(0);
  await expect(page.getByRole('button',{name:'更多操作',exact:true})).toHaveCount(0);
  await page.getByRole('region',{name:'任职详情'}).focus();
  await page.keyboard.press('End');
  await expect.poll(() => page.locator('.history-notice').evaluate(el => {
    const box = el.getBoundingClientRect();
    return box.top >= -1 && box.bottom <= innerHeight + 1;
  })).toBe(true);
  const currentRead = page.waitForResponse(response => response.url().includes('/api/v1/assignments/') &&
    response.request().method() === 'GET' && !new URL(response.url()).searchParams.has('sourceVersion') && response.ok());
  await page.getByRole('button',{name:'返回当前职务信息'}).click();
  await currentRead;
  await expect(page.getByText(locations[0].name + ' → ' + locations[2].name, { exact: true })).toBeVisible();
  await expect
    .poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1))
    .toBe(true);
  await page.getByRole('link', { name: '返回人员', exact: true }).click();
  await expect(page.getByText('1 份任职，1 人', { exact: true })).toBeVisible({timeout:15000});
  await expect(row).toBeFocused();
  await expect(search).toHaveValue(sample.name);
}
for (const [width, height] of desktopViewports) {
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

test('M04 地点补录与原笔改日遇到真实并发后采用最新来源', async ({page}) => {
  test.setTimeout(150000);
  await signIn(page);
  const sample = await seed(page);
  const {locations, hire} = await hireAtLocation(page, sample);
  await openEmployeeAssignment(page, sample);
  const base = '/api/v1/assignments/' + hire.assignment;
  const button = (label: string) => page.getByRole('button', {name:label,exact:true});
  const date = page.getByRole('textbox', {name:/^生效日期/});
  const latest = async (day:string) => (await request(page,'GET',base+'?businessAt='+day)).body;
  const other = await page.context().newPage();
  await other.goto(page.url());
  await button('新增记录').click();
  await date.fill('2027-05-01');
  await date.press('Tab');
  const location = page.getByRole('combobox',{name:/工作地点/});
  await location.fill(locations[1].name);
  await page.getByRole('option',{name:locations[1].name,exact:true}).click();
  const june = await request(other, 'POST', base+'/location-changes', {businessDate:'2027-06-01',
    expectedSourceVersion:(await latest('2027-06-01')).sourceVersion, location:locations[2].id}, randomUUID());
  expect(june.status,june.body.message).toBe(200);
  async function conflict(path: string) {
    const response = page.waitForResponse(r => new URL(r.url()).pathname === path && r.request().method()==='POST');
    await button('保存').click();
    expect((await response).status()).toBe(409);
    await expect(button('保存')).toBeDisabled();
    await expect(button('采用当前来源继续办理')).toBeEnabled();
  }
  await conflict(base+'/location-changes');
  await expect(location).toHaveValue(locations[1].name);
  await button('采用当前来源继续办理').click();
  await expect(page.getByRole('heading',{name:'插入记录 · 数据更改'})).toBeVisible();
  await button('保存').click();
  await expect(button('数据更改 · 2027-05-01')).toBeVisible();
  const may = await latest('2027-05-01');
  const action = may.actions.find((a:{businessDate:string})=>a.businessDate==='2027-05-01');
  expect(may.state.location.id).toBe(locations[1].id);
  await button('数据更改 · 2027-05-01').click();
  await chooseRecordAction(page,'调整生效日期');
  await date.fill('2027-05-03');
  await date.press('Tab');
  const edited = await request(other,'POST',base+'/actions/'+action.id+'/redate', {
    expectedSourceVersion:may.sourceVersion, expectedRevision:action.revision, businessDate:'2027-05-02'},randomUUID());
  expect(edited.status,edited.body.message).toBe(200);
  await conflict(base+'/actions/'+action.id+'/redate');
  await expect(date).toHaveValue('2027-05-03');
  await expect(page.getByText('2027-05-02',{exact:true})).toBeVisible();
  await button('采用当前来源继续办理').click();
  await button('保存').click();
  await expect(button('数据更改 · 2027-05-03')).toBeVisible();
  const proof = await request(page,'GET','/api/v1/employment-actions/'+action.id+'?adoptedRef='+edited.body.revision);
  expect(proof.body.action.businessDate).toBe('2027-05-02');
  await button('数据更改 · 2027-05-03').click();
  await page.getByRole('button',{name:/查看修订/}).nth(1).click();
  await button('数据更改 · 2027-05-02').click();
  await button('返回当前职务信息').click();
  await expect(button('数据更改 · 2027-05-03')).toBeVisible();
  await expect(page).toHaveURL(url=>url.searchParams.get('assignmentDay')==='2027-05-03');
  await other.close();
});

async function openRegister(page: Page) {
  const heading = page.getByRole('heading', {name:'办理记录', exact:true});
  if (new URL(page.url()).pathname === '/personnel/actions') {
    await expect(heading).toBeVisible();
    return;
  }
  const desktop = page.getByRole('button', { name: '切换应用', exact: true });
  if (await desktop.isVisible()) {
    const transactions = page.getByRole('button', { name: '人事事务', exact: true });
    if (!(await transactions.isVisible())) {
      await desktop.click();
      await page.getByRole('menuitem', { name: '人员', exact: true }).click();
      await expect(page.getByRole('heading', { name: /^(人员|办理记录)$/ }).first()).toBeVisible();
      if (new URL(page.url()).pathname === '/personnel/actions') {
        await expect(heading).toBeVisible();
        return;
      }
    }
    await transactions.focus();
    await page.keyboard.press('Enter');
    const register = page.getByRole('menuitem', { name: '办理记录', exact: true });
    await register.focus();
    await page.keyboard.press('Enter');
  } else {
    await page.getByRole('button', { name: '打开导航', exact: true }).click();
    const personnel = page.locator('#navigation-drawer').getByRole('button', { name: '人员', exact: true }).first();
    if ((await personnel.getAttribute('aria-expanded')) !== 'true') await personnel.click();
    const transactions = page.getByRole('button', { name: '人事事务', exact: true });
    if ((await transactions.getAttribute('aria-expanded')) !== 'true') await transactions.click();
    await page.getByRole('button', { name: '办理记录', exact: true }).click();
  }
  await expect(heading).toBeVisible();
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
  function appliedList(kind: string | null) {
    return page.waitForResponse(response => {
      const url = new URL(response.url());
      return url.pathname === '/api/v1/employment-actions' && url.searchParams.has('ref') &&
        url.searchParams.get('kind') === kind && !url.searchParams.has('anchor');
    });
  }
  const search = page.getByRole('textbox', { name: '姓名或人员编号' });
  await search.fill(sample.name);
  const named = appliedList(null);
  await search.press('Enter');
  expect((await named).status()).toBe(200);
  await expect(page).toHaveURL((url) => url.searchParams.has('ref'));
  await expect(page.getByText('2 笔办理', { exact: true })).toBeVisible();
  await page.getByText('筛选条件', { exact: true }).click();
  await search.fill('无匹配'+sample.name);
  await expect(page.getByRole('status').filter({hasText:'姓名或人员编号已修改'})).toBeVisible();
  await page.getByLabel('办理类型', {exact:true}).selectOption('Leave');
  const empty = appliedList('Leave');
  await page.getByRole('button',{name:'应用筛选'}).click();
  expect((await empty).status()).toBe(200);
  await expect(page.getByText('0 笔办理',{exact:true})).toBeVisible();
  expect(page.url()).not.toContain(encodeURIComponent(sample.name));
  await page.locator('summary').filter({hasText:'离职'}).click();
  await search.fill(sample.name);
  await page.getByLabel('办理类型', { exact: true }).selectOption('DataChange');
  await page.getByLabel('生效起日', { exact: true }).fill('2027-05-01');
  await page.getByLabel('生效止日', { exact: true }).fill('2027-05-01');
  const changes = appliedList('DataChange');
  await page.getByRole('button', { name: '应用筛选' }).click();
  expect((await changes).status()).toBe(200);
  await expect(page.getByText('1 笔办理', { exact: true })).toBeVisible();
  const row = page.locator('a[data-object-link]').filter({ hasText: '数据更改' });
  await row.focus();
  await page.keyboard.press('Enter');
  await expect(page).toHaveURL((url) => url.pathname.endsWith('/' + created.body.action));
  await expect(page.getByRole('heading', { name: '数据更改原件', exact: true })).toBeVisible();
  await expect(page.getByText(sample.department, { exact: true })).toBeVisible();
  await chooseRecordAction(page, '更正记录');
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
for (const [width, height] of desktopViewports) {
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
  await chooseEffectiveDate(page,'2027-04-01');
  await button('插入记录').click();
  await date('生效日期','2027-06-01');
  await page.getByRole('combobox',{name:/工作地点/}).fill(locations[1].name);
  await page.getByRole('option',{name:locations[1].name,exact:true}).click();
  await button('保存').click();
  await expect(button('数据更改 · 2027-06-01')).toBeVisible();
  await chooseEffectiveDate(page,'2027-07-01');
  await expect(page.getByText([destination,sample.job,locations[1].name].join(' · '),{exact:true})).toBeVisible();
  await chooseEffectiveDate(page,'2027-06-01');
  await button('数据更改 · 2027-06-01').click();
  await chooseRecordAction(page, '调整生效日期');
  await date('生效日期','2027-06-10');
  await button('保存').click();
  await button('数据更改 · 2027-06-10').click();
  const deletions: string[] = [];
  page.on('request', req => {
    if (req.method() === 'POST' && req.url().endsWith('/delete')) deletions.push(req.url());
  });
  await chooseRecordAction(page, '删除记录');
  await expect(button('确认删除')).toBeVisible();
  expect(deletions).toHaveLength(0);
  await button('取消').click();
  await expect(button('更多操作')).toBeFocused();
  expect(deletions).toHaveLength(0);
  await chooseRecordAction(page, '删除记录');
  await button('确认删除').click();
  await expect.poll(() => deletions.length).toBe(1);
  await expect(page.getByText([sample.department,sample.job,locations[0].name].join(' · '),{exact:true})).toBeVisible();
  await button('数据更改（已删除） · 2027-06-10').click();
  await expect(page.getByText('已删除',{exact:true})).toBeVisible();
  await expect(button('删除记录')).toHaveCount(0);
  await expect(page.getByRole('button',{name:/查看修订/})).toHaveCount(3);
  await chooseEffectiveDate(page,'2027-07-01');
  await expect(page.getByText([destination,sample.job,locations[0].name].join(' · '),{exact:true})).toBeVisible();
  await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
}
for(const [width,height] of desktopViewports){
  test(`M04 历史插入与按职务调动 ${width}x${height}`,async({page})=>{
    test.setTimeout(160000);
    await signIn(page);const sample=await seed(page);await page.setViewportSize({width,height});
    await historyJourney(page,sample);
  });
}
test('M04 历史插入与按职务调动 原生200%',async()=>{await native200(historyJourney)});

async function ensureJobOnlyPolicy(page: Page, purpose: 'hire' | 'secondary') {
  const path = '/api/v1/assignment-policies', from = '2027-04-01';
  const policy = await request(page, 'GET', path);
  expect(policy.status, policy.body.message).toBe(200);
  const current = policy.body.rows.find((r: { category: string; purpose: string; from: string; until?: string }) =>
    r.category === 'Employee' && r.purpose === purpose && r.from <= from && (!r.until || from < r.until));
  if (current?.mode === 'JobOnly') return;
  const row = current ?? { id: randomUUID(), category: 'Employee', purpose, from };
  const saved = await request(page, 'POST', path, {
    expectedVersion: policy.body.version, row: { ...row, mode: 'JobOnly' },
  }, randomUUID());
  expect(saved.status, saved.body.message).toBe(200);
}
async function concurrentJourney(page: Page, sample: Awaited<ReturnType<typeof seed>>) {
  const { hire } = await hireAtLocation(page, sample);
  await ensureJobOnlyPolicy(page, 'secondary');
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
  await expect(page.locator('.object-detail-identity').filter({hasText:'0 号主岗'})).toBeVisible();
  await go('新增兼岗');
  await choose('部门', sample.department);
  await choose('职务', sample.job);
  await page.getByLabel(/任用依据/).fill('兼任部门评审员');
  await button('保存').focus();
  await page.keyboard.press('Enter');
  await expect(page.locator('.object-detail-identity').filter({hasText:'1 号兼岗'})).toBeVisible();
  await expect(page.getByRole('tab', { name: '个人资料' })).toHaveCount(0);
  await expect(page.getByRole('tab', { name: /个人日志/ })).toHaveCount(0);
  await button('新增记录').click();
  const type = page.getByRole('combobox',{name:/办理类型/});
  await type.click();
  await page.getByRole('option',{name:'离职',exact:true}).click();
  await expect(page.getByText('0 号主岗、1 号兼岗',{exact:true})).toBeVisible();
  await expect(page.getByText(sample.employer+' · 自 2027-04-01 起',{exact:true}).last()).toBeVisible();
  await type.click();
  await page.getByRole('option',{name:'结束任职',exact:true}).click();
  await expect(page.getByText('1 号兼岗 · '+sample.employer,{exact:true})).toBeVisible();
  await expect(page.getByText('0 号主岗、1 号兼岗',{exact:true})).toHaveCount(0);
  await button('取消').click();
  await button('放弃修改').click();
  await expect(button('新增记录')).toBeFocused();
  await go('新增兼岗');
  await page.getByRole('combobox', { name: /所属关系/ }).click();
  await page.getByRole('option', { name: '新增并行工作关系', exact: true }).click();
  await choose('用人单位', employerName);
  await choose('部门', newDept);
  await choose('职务', sample.job);
  await page.getByLabel(/任用依据/).fill('培训关系独立聘任');
  await button('保存').click();
  await expect(page.locator('.object-detail-identity').filter({hasText:'2 号兼岗'})).toBeVisible();
  await expect(page.locator('.object-detail-identity')).toContainText('本工作关系主任职');
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
for (const [width, height] of desktopViewports) {
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
    await expect(screen.locator('.object-detail-identity').filter({hasText:'2 号兼岗'})).toBeVisible();
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

test('M04 从兼岗打开关系转正的获权任职来源并返回原岗', async ({page,browser}) => {
  test.setTimeout(150000);
  await signIn(page);
  const sample = await seed(page);
  const {hire} = await hireAtLocation(page,sample);
  await ensureJobOnlyPolicy(page, 'secondary');
  const other = await apiDepartment(page, sample.buId,{name:'兼岗来源部'+randomUUID().slice(0,6)});
  const base = '/api/v1/assignments/'+hire.assignment;
  async function save(path:string, day:string, fields:object) {
    const current = await request(page,'GET',base+'?businessAt='+day);
    const result = await request(page,'POST',base+'/'+path,{...fields,expectedSourceVersion:current.body.sourceVersion},randomUUID());
    expect(result.status,result.body.message).toBe(200);
    return result.body;
  }
  const secondary = await save('additional-assignments','2027-04-01',{from:'2027-04-01',parallel:false,department:other.id,job:sample.jobId,notice:'兼岗原始通知'});
  const earlySource = await request(page,'GET','/api/v1/assignments/'+secondary.assignment+'?businessAt=2027-04-01');
  expect(earlySource.status).toBe(200);
  await save('trial-start','2027-04-09',{businessDate:'2027-04-09',notice:'关系试用开始'});
  const confirmed = await save('confirm-employment','2027-07-01',{businessDate:'2027-07-01',notice:'获权关系转正通知'});
  async function check(screen:Page, count:number, expectedAssignment:string) {
    await openEmployeeAssignment(screen,sample,'2027-07-01',count,secondary.assignment);
    const record = screen.getByRole('button',{name:'转正 · 2027-07-01'});
    await record.click();
    await expect(screen.getByText('获权关系转正通知',{exact:true})).toBeVisible();
    const source = screen.getByRole('link',{name:'查看当次任职'});
    const response = screen.waitForResponse(r => new URL(r.url()).pathname==='/api/v1/assignments/'+expectedAssignment &&
      new URL(r.url()).searchParams.has('sourceVersion'));
    await source.click();
    const read = await response;
    expect(read.status()).toBe(200);
    const returned = await read.json();
    expect(returned.assignment).toBe(expectedAssignment);
    expect(returned.actions.some((a:{id:string;revision:string})=>a.id===confirmed.action && a.revision===confirmed.revision)).toBe(true);
    await expect(screen.getByText('正在查看只读历史资料')).toBeVisible();
    expect(new URL(screen.url()).searchParams.get('sourceVersion')).toBe(returned.sourceVersion);
    await screen.reload();
    await expect(screen.getByText('获权关系转正通知',{exact:true})).toBeVisible();
    await screen.getByRole('button',{name:'返回当前职务信息'}).click();
    await expect(screen).toHaveURL(url => url.pathname.endsWith('/'+secondary.assignment) && !url.searchParams.has('sourceVersion'));
    await expect(screen.locator('.object-detail-identity').filter({hasText:'1 号兼岗'})).toBeVisible();
    await expect(screen.getByText('获权关系转正通知',{exact:true})).toBeVisible();
    await expect(record).toBeFocused();
  }
  await check(page,2,hire.assignment);
  const sql=database(), grants=['directory','read','history','employment.leave'].map(action=>({id:randomUUID(),action}));
  const reader=await browser.newContext({baseURL:localConnection().webOrigin});
  try {
    for(const grant of grants) sql(`INSERT INTO access.employment_grant(instance_id,principal_id,grant_id,action,person_id,department_id,fields)
      SELECT instance_id,principal_id,'${grant.id}','${grant.action}','${sample.id}','${other.id}',ARRAY['arrangement','basis','location']
      FROM access.principal WHERE login='reader'`);
    const screen=await reader.newPage();
    await signIn(screen,'reader');
    await check(screen,1,secondary.assignment);
    const impact = await request(screen,'GET','/api/v1/assignments/'+secondary.assignment+'/leave-impact?date=2027-09-01&meaning=first-inactive');
    expect(impact.status,impact.body.message).toBe(200);
    expect(impact.body.terms.map((t:{recordNumber:number})=>t.recordNumber)).toEqual([1]);
    for (const [version,status] of [[earlySource.body.sourceVersion,200],[randomUUID(),404]] as const) {
      const location = await screen.getByRole('link',{name:'查看当次任职'}).getAttribute('href');
      const target = new URL(location!,screen.url());
      target.searchParams.set('sourceVersion',version);
      const response = screen.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/assignments/'+secondary.assignment &&
        new URL(r.url()).searchParams.get('sourceVersion') === version);
      await screen.goto(target.href);
      expect((await response).status()).toBe(status);
      await expect(screen.getByText('获权关系转正通知',{exact:true})).toHaveCount(0);
      if (status === 200) await expect(screen.getByText('该办理或来源当前不可用。')).toBeVisible();
      await screen.getByRole('button',{name:'返回当前职务信息'}).click();
      await expect(screen.locator('.object-detail-identity').filter({hasText:'1 号兼岗'})).toBeVisible();
      await expect(screen.getByText('获权关系转正通知',{exact:true})).toBeVisible();
    }
    const beforeRevoke = screen.waitForResponse(r => new URL(r.url()).pathname==='/api/v1/assignments/'+secondary.assignment &&
      new URL(r.url()).searchParams.has('sourceVersion'));
    await screen.getByRole('link',{name:'查看当次任职'}).click();
    expect((await beforeRevoke).status()).toBe(200);
    await expect(screen.getByText('正在查看只读历史资料')).toBeVisible();
    await expect(screen.getByText('获权关系转正通知',{exact:true})).toBeVisible();
    const historyGrant = grants.find(g=>g.action==='history')!;
    sql(`DELETE FROM access.employment_grant WHERE grant_id='${historyGrant.id}'`);
    const denied = screen.waitForResponse(r => new URL(r.url()).pathname==='/api/v1/assignments/'+secondary.assignment &&
      new URL(r.url()).searchParams.has('sourceVersion'));
    await screen.reload();
    expect((await denied).status()).toBe(403);
    await expect(screen.getByText('获权关系转正通知',{exact:true})).toHaveCount(0);
    await screen.getByRole('button',{name:'返回当前职务信息'}).click();
    await expect(screen.locator('.object-detail-identity').filter({hasText:'1 号兼岗'})).toBeVisible();
    await expect(screen).toHaveURL(url => !url.searchParams.has('action') && !url.searchParams.has('sourceVersion'));
    await expect(screen.getByRole('button',{name:'新增记录',exact:true})).toBeFocused();
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
  const profile = await request(page,'GET','/api/v1/persons/'+sample.id);
  expect(profile.status).toBe(200);
  await expect(page.getByRole('heading',{name:sample.name+' · '+profile.body.number+' · 转正 · 工作关系',exact:true})).toBeVisible();
  await expect(page.getByText('试用转正通知',{exact:true})).toBeVisible();
  await page.getByRole('link',{name:'返回办理记录'}).click();
  await expect(record).toBeFocused();
  await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
}
for(const [width,height] of desktopViewports){
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
  if (mode !== 'reverse') {
    await add('转正',date);await page.getByLabel(/转正依据/).fill('本关系转正通知');
    await button('保存').click();await expect(page.getByText('已转正',{exact:true})).toBeVisible();
  }
  if (mode !== 'forward') {
    const family = await seed308(page,'jobfamily');
    const job = await seed308(page,'job',family.id,{name:'产品专员' + suffix,gradeIds:mode === 'pair' ? [p6.id] : [p5.id,p6.id]});
    const pair = mode === 'pair';
    await add(pair ? '晋升并调动' : '调动',date);
    await choose('调入部门',dest);
    await choose('拟任职务',job.name);
    await page.getByLabel(pair ? /晋升依据/ : /调动依据/).fill(pair ? '晋升及调动共同通知' : '独立调动通知');
    if (pair) await prior('转正 · 本关系转正通知');
    if (pair) await choose('拟任职级',p6.name);
    await button('保存').click();
    await button('调动 · '+date).click();
    await expect(page.getByRole('heading', {name: '调动原件', exact: true})).toBeVisible();
    if (pair) {
      const noticeSource = button('查看通知来源');
      await expect(noticeSource).toHaveAttribute('aria-expanded', 'false');
      await noticeSource.click();
      await expect(noticeSource).toHaveAttribute('aria-expanded', 'true');
      await expect(page.getByText('晋升 · '+date+' · 晋升及调动共同通知',{exact:true})).toBeVisible();
      await button('晋升 · '+date).click();
      await expect(page.getByRole('heading', {name: '晋升原件', exact: true})).toBeVisible();
      await chooseRecordAction(page, '撤销晋升');
      await page.getByLabel(/撤销依据/).fill('核对原通知');
      await button('保存').click();
      await expect(page.getByRole('alert').filter({hasText:'职级在拟定职务或期间内不可用'})).toBeVisible();
    } else {
      await chooseRecordAction(page, '在本次调动前补录晋升');
      await choose('拟任职级',p6.name);
      await page.getByLabel(/晋升依据/).fill('研发晋升通知');
      await button('保存').click();
      await button('晋升 · '+date).click();
      await chooseRecordAction(page, '在本次晋升前补录转正');
      await expect(page.getByLabel(/^生效日期/)).toHaveValue(date);
      await page.getByLabel(/转正依据/).fill('本关系转正通知');
      await button('保存').click();
      await expect(page.getByText('已转正',{exact:true})).toBeVisible();
      await button('转正 · '+date).click();
      await expect(page.getByText(sample.department,{exact:true}).first()).toBeVisible();
      await button('晋升 · '+date).click();
      await chooseRecordAction(page, '撤销晋升');
      await page.getByRole('combobox',{name:'引用通知'}).click();
      await page.getByRole('option',{name:'调动 · 独立调动通知',exact:true}).click();
      await button('保存').click();
      await button('晋升（已撤销） · '+date).click();
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
  await chooseRecordAction(page, '更正记录');await page.getByLabel('晋升原因',{exact:true}).fill('专业职级晋升');
  await button('保存').click();await button('晋升 · '+date).click();
  await expect(page.getByText('专业职级晋升',{exact:true})).toBeVisible();
  await openRegister(page);
  const name=page.getByRole('textbox',{name:'姓名或人员编号'});await name.fill(sample.name);await name.press('Enter');
  const promotion=page.locator('a[data-object-link]').filter({hasText:'晋升'});
  await expect(promotion).toHaveCount(1);await promotion.focus();await page.keyboard.press('Enter');
  await chooseRecordAction(page, '撤销晋升');await page.getByLabel(/撤销依据/).fill('核对原件，通知仅载调动');
  await button('保存').click();await expect(page.getByText('已撤销',{exact:true}).first()).toBeVisible();
  await button('查看撤销前来源').click();
  await expect(page.getByText(p6.name,{exact:true}).first()).toBeVisible();
  await expect(page).toHaveURL(url=>!!url.searchParams.get('sourceVersion'));
  await page.reload();
  await expect(page.getByText(p6.name,{exact:true}).first()).toBeVisible();
  await expect(button('更多操作')).toHaveCount(0);
  await expect(page.getByText('正在查看只读历史资料',{exact:true})).toBeVisible();
  await button('返回当前职务信息').click();
  await page.getByRole('link',{name:'返回办理记录'}).click();await expect(promotion).toBeFocused();
  await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
}
for(const [width,height] of desktopViewports) {
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

async function lifecycleJourney(page:Page,sample:Awaited<ReturnType<typeof seed>>, recover=false, fromHistory=false){
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
  const managerValues=page.getByText(manager.name+' · 0 号',{exact:true});
  await expect(managerValues).toHaveCount(2);
  await expect(managerValues.first()).toBeVisible();
  await button('实际汇报 · 2027-05-01').click();
  await expect(page.getByRole('heading',{name:'实际汇报原件'})).toBeVisible();
  await add('离职','原通知日期','2027-08-31');
  await page.getByLabel(/退出依据/).fill('本关系最后在职日为八月三十一日');await button('保存').click();
  await button('离职 · 2027-08-31').click();await expect(page.getByRole('heading',{name:'离职原件'})).toBeVisible();
  let sourceAddress = '';
  if (fromHistory) {
    await page.getByRole('link',{name:'本人办理历史',exact:true}).click();
    await button('从原离职办理再聘').click();
    await expect(page).toHaveURL(url => !!url.searchParams.get('rehireAction') && !!url.searchParams.get('rehireRevision'));
    sourceAddress = new URL(page.url()).searchParams.get('returnTo')!;
  } else {
    await openRegister(page);
    const name=page.getByRole('textbox',{name:'姓名或人员编号'});await name.fill(sample.name);await name.press('Enter');
    const left=page.locator('a[data-object-link]').filter({hasText:'离职'});await expect(left).toHaveCount(1);
    await left.focus();await page.keyboard.press('Enter');
    await page.getByRole('link',{name:'办理再聘',exact:true}).click();
  }
  await expect(page.getByRole('heading',{name:'办理再聘'})).toBeVisible();
  const day=page.getByRole('textbox',{name:/^到岗日/});await day.fill('2027-10-01');await day.press('Tab');
  for(const [label,value] of [['部门',sample.department],['职务',sample.job],['用人单位',sample.employer]])await choose(label,value);
  await page.getByLabel(/任用通知说明/).fill('十月重新聘任');
  let submissionId: string | null = null;
  if (recover) {
    await page.route('**/api/v1/hires', async route => {
      submissionId = await route.request().headerValue('Idempotency-Key');
      await route.fetch();
      await route.abort('failed');
    }, {times: 1});
    // Keep the receipt unknown until the user reopens the page and requests it.
    await page.route('**/api/v1/hires/submissions/*/resolve', route => route.abort('failed'), {times: 1});
  }
  await button('保存').click();
  if (recover) {
    await expect(page.getByRole('alert').filter({hasText: '网络连接失败，请重试'})).toBeVisible();
    await expect(button('查回原提交')).toBeVisible();
    expect(submissionId).toBeTruthy();
    page.once('dialog', dialog => dialog.accept());
    await page.reload({waitUntil: 'commit'});
    await expect(button('查回原提交')).toBeVisible();
    const [receipt] = await Promise.all([
      page.waitForResponse(response => new URL(response.url()).pathname === '/api/v1/hires/submissions/' + submissionId + '/resolve'),
      button('查回原提交').click(),
    ]);
    expect(receipt.ok()).toBe(true);
    expect(await receipt.json()).toMatchObject({submissionId, outcome: 'accepted'});
  }
  await expect(page).toHaveURL(url=>url.pathname.startsWith('/personnel/employees/'+sample.id+'/assignments/') && url.searchParams.get('assignmentDay')==='2027-10-01' && !!url.searchParams.get('action'));
  const landed = new URL(page.url());
  const exact = await request(page,'GET','/api/v1/employment-actions/'+landed.searchParams.get('action')+'?adoptedRef='+landed.searchParams.get('adoptedRef'));
  expect(exact.status).toBe(200);
  expect(exact.body.action).toMatchObject({kind:'Rehire',person:sample.id,recordNumber:0,businessDate:'2027-10-01',assignment:landed.pathname.split('/').pop()});
  await expect(page.getByText('十月重新聘任',{exact:true}).first()).toBeVisible();
  if (fromHistory) {
    await page.getByRole('link',{name:'返回办理历史'}).click();
    await expect(page).toHaveURL(new URL(sourceAddress,page.url()).href);
    const exit = await request(page,'GET','/api/v1/employment-actions/'+exact.body.rehireOf.action+'?adoptedRef='+exact.body.rehireOf.revision);
    expect(exit.status).toBe(200);
    expect(exit.body.action.employment).not.toBe(exact.body.action.employment);
    expect(exit.body.relation).toMatchObject({ employerName: sample.employer, from: '2027-04-01', until: '2027-09-01' });
    await expect(page.getByText(`工作关系 ${sample.employer} · 2027-04-01 起，2027-09-01 起不在该关系`,{exact:true})).toBeVisible();
  }
  await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
}
for(const [width,height] of desktopViewports){
  test(`M04 实际汇报离职及原记录再聘 ${width}x${height}`,async({page})=>{
    test.setTimeout(180000);await signIn(page);const sample=await seed(page);
    await page.setViewportSize({width,height});await lifecycleJourney(page,sample,height === 540);
  });
}
test('M04 从本人历史再聘，回执中断后查回并返回原离职办理',async({page})=>{
  test.setTimeout(180000);
  await signIn(page);
  const sample = await seed(page);
  await lifecycleJourney(page, sample, true, true);
});
test('M04 实际汇报离职及原记录再聘 原生200%',async()=>{await native200(lifecycleJourney)});

async function positionHireJourney(page: Page, sample: Awaited<ReturnType<typeof seed>>) {
  const family = await seed308(page,'jobfamily');
  const position = await seedPosition(page,'position',family.id);
  const policyPath='/api/v1/assignment-policies';
  const before=await request(page,'GET',policyPath);
  const original=before.body.rows.find((r:{category:string;purpose:string;from:string})=>r.category==='Employee'&&r.purpose==='hire'&&r.from==='2027-04-01');
  let changedVersion: string | undefined;
  if (original.mode !== 'PositionExempt') {
    const saved = await request(page, 'POST', policyPath, { expectedVersion: before.body.version, row: { ...original, mode: 'PositionExempt' } }, randomUUID());
    expect(saved.status, saved.body.message).toBe(200);
    changedVersion = saved.body.version;
  }
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
    await expect(page.getByRole('heading',{name:'本人办理历史'})).toBeVisible();
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
    if (changedVersion) {
      const now = await request(page, 'GET', policyPath);
      const applied = now.body.rows.find((r: { id: string }) => r.id === original.id);
      if (applied?.mode === 'PositionExempt' && now.body.version === changedVersion) {
        const restored = await request(page, 'POST', policyPath, { expectedVersion: now.body.version, row: { ...applied, mode: original.mode } }, randomUUID());
        expect(restored.status, restored.body.message).toBe(200);
      }
    }
  }
}
for(const [width,height] of desktopViewports){
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
    await chooseRecordAction(page, '更正记录');await expect(position).toHaveValue(first.name);
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
for (const [width,height] of desktopViewports) {
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
for(const [width,height] of desktopViewports) {
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
    const releaseUsage = await holdRead(page, url => url.pathname === '/api/v1/positions/' + target.id + '/replenishment' &&
      url.searchParams.get('businessAt') === '2027-07-01');
    await page.getByLabel(/晋升依据/).fill('同一份通知晋升并调入新岗');await button('保存').focus();await page.keyboard.press('Enter');
    await expect(page.getByText(target.name+' · 占编',{exact:true})).toBeVisible();
    await expect(page.getByText(target.grade.name,{exact:true})).toBeVisible();
    await clickWhileReadCompletes(button('调动 · 2027-07-01'), releaseUsage,
      page.getByText('核定 1 · 实占 1 · 超编 0', {exact:true}));
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
for(const [width,height] of desktopViewports) {
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
    await chooseRecordAction(page, '调整对调日期');await day.fill('2027-06-01');await day.press('Tab');
    await button('保存').click();await button('调动 · 2027-06-01').click();
    await chooseRecordAction(page, '撤销两岗对调');
    await expect(page.getByText('双方的岗位调动将一并撤销')).toBeVisible();
    await button('取消').click();
    await expect(button('更多操作')).toBeFocused();
    await expect(page.getByText(b.name+' · 占编',{exact:true})).toBeVisible();
    await chooseRecordAction(page, '撤销两岗对调');
    await button('确认撤销两岗对调').click();
    await expect(page.getByText(a.name+' · 占编',{exact:true})).toBeVisible();
    await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
  });
}

for(const [width,height] of desktopViewports){
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
    await chooseEffectiveDate(page,'2027-04-11');
    await expect(page.getByText(b.name+' · 占编',{exact:true})).toBeVisible();
    const usage=await request(page,'GET','/api/v1/positions/'+b.id+'/replenishment?businessAt=2027-04-10');
    expect([usage.body.occupied,usage.body.handoverOccupied,usage.body.totalOccupied,usage.body.reserved]).toEqual([1,1,2,0]);
    await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
  });
}
for(const [width,height] of desktopViewports){
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
  await openEmployeeAssignment(page,sample,first,0);
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
  await chooseEffectiveDate(page,after);await expect(page.getByText(home.name+' · 占编',{exact:true})).toBeVisible();
  await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
}
const seedDuty=(page:Page)=>seed(page,true,'2026-04-01');
for(const [width,height] of desktopViewports){
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
  const form = page.locator('form').filter({has:page.getByLabel(/退出依据/)});
  await expect(form.getByText(second,{exact:true})).toBeVisible();
  await expect(button('保存')).toBeDisabled();
  if (await page.evaluate(() => innerWidth >= 900 && innerHeight >= 800)) {
    for (const [meaning,label] of [['first-inactive','首个不在职日'],['last-active','最后在职日（含当日）']]) {
      const preview = page.waitForResponse(response => {
        const url = new URL(response.url());
        return url.pathname.endsWith('/leave-impact') && url.searchParams.get('date') === '2027-08-31' && url.searchParams.get('meaning') === meaning;
      });
      await page.getByRole('combobox',{name:/原通知日期含义/}).click();
      await page.getByRole('option',{name:label,exact:true}).click();
      const response = await preview;
      expect(response.status()).toBe(200);
      expect((await response.json()).contracts.map((contract:{number:string})=>contract.number)).toEqual([second]);
    }
  }
  await page.getByRole('combobox',{name:/合同处置/}).click();await page.getByRole('option',{name:'按离职通知终结上述合同',exact:true}).click();
  await button('保存').click();
  await expect(page.getByText('职务信息已保存。',{exact:true})).toBeVisible();
  await button(second).first().click();
  await expect(page.getByText('2027-09-01',{exact:true}).last()).toBeVisible();
  await expect(page.getByText(/依通知终结本关系和续订合同 · admin/)).toBeVisible();
  await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
}
for(const [width,height] of desktopViewports){
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
    const hireRow=page.locator('a[data-object-link]').filter({hasText:'入职'});
    await hireRow.click();
    const originalAddress=page.url();
    await page.getByRole('link',{name:number,exact:true}).click();
    await expect(page.getByText('已接受',{exact:true})).toBeVisible();
    await page.reload();
    await page.getByRole('link',{name:'返回办理明细'}).click();
    await expect(page).toHaveURL(originalAddress);
    await page.getByRole('link',{name:'返回办理记录'}).click();
    await expect(hireRow).toBeFocused(); await expect(search).toHaveValue(sample.name);
    await hireRow.click();
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
for(const [width,height] of desktopViewports){
  test(`T06 已接受录用转入职与未来撤销 ${width}x${height}`,async({page})=>{
    test.setTimeout(170000);await signIn(page);const sample=await seed(page);await page.setViewportSize({width,height});await offerHireJourney(page,sample);
  });
}
test('T06 已接受录用转入职与未来撤销 原生200%',async()=>{await native200(offerHireJourney)});

test('M04 采用录用的跨页返回按招聘和人事原件的现权读取', async ({page, browser}) => {
  test.setTimeout(180000);
  await signIn(page);
  const sample = await seed(page);
  await withCountedPolicies(page, async () => {
    const { number } = await recruitingJourney(page, sample);
    await page.getByRole('button', {name:'按录用办理入职'}).click();
    await expect(page.getByText('已入职', {exact:true})).toBeVisible();
    const reader = await browser.newContext({baseURL:localConnection().webOrigin});
    const sql = database(), offerGrant = randomUUID();
    const jobs = ['directory','register.read','read','history'].map(action => ({id:randomUUID(),action}));
    function grantOffer() {
      sql(`INSERT INTO access.offer_grant(instance_id,principal_id,grant_id,action,person_id)
        SELECT instance_id,principal_id,'${offerGrant}','read','${sample.id}' FROM access.principal WHERE login='reader'`);
    }
    try {
      for (const g of jobs) sql(`INSERT INTO access.employment_grant(instance_id,principal_id,grant_id,action,person_id,fields)
        SELECT instance_id,principal_id,'${g.id}','${g.action}','${sample.id}',ARRAY['arrangement','basis','department','job']
        FROM access.principal WHERE login='reader'`);
      grantOffer();
      const screen = await reader.newPage();
      await signIn(screen,'reader');
      await openRegister(screen);
      const name = screen.getByRole('textbox',{name:'姓名或人员编号'});
      await name.fill(sample.name); await name.press('Enter');
      await expect(screen.getByText('1 笔办理',{exact:true})).toBeVisible();
      await screen.locator('a[data-object-link]').filter({hasText:'入职'}).click();
      const original = screen.url();
      const source = screen.getByRole('link',{name:number,exact:true});
      await source.click();
      await expect(screen.getByText('已接受',{exact:true})).toBeVisible();
      const adopted = screen.url();
      const offerPath = '/api/v1/recruiting/offers/'+new URL(adopted).pathname.split('/').pop();
      const actionPath = '/api/v1/employment-actions/'+new URL(original).pathname.split('/').pop();

      sql(`DELETE FROM access.offer_grant WHERE grant_id='${offerGrant}'`);
      const deniedOffer = screen.waitForResponse(r => new URL(r.url()).pathname === offerPath);
      await screen.reload();
      expect((await deniedOffer).status()).toBe(403);
      await expect(screen.getByRole('heading',{name:number})).toHaveCount(0);
      await screen.getByRole('link',{name:'返回办理明细'}).click();
      await expect(screen).toHaveURL(original);
      await expect(screen.getByRole('heading',{name:'入职原件',exact:true})).toBeVisible();
      await expect(source).toHaveCount(0);

      grantOffer();
      await screen.reload(); await source.click();
      await expect(screen.getByText('已接受',{exact:true})).toBeVisible();
      const historical = jobs.find(g => g.action === 'history')!;
      sql(`DELETE FROM access.employment_grant WHERE grant_id='${historical.id}'`);
      const deniedAction = screen.waitForResponse(r => new URL(r.url()).pathname === actionPath);
      await screen.getByRole('link',{name:'返回办理明细'}).click();
      expect((await deniedAction).status()).toBe(403);
      await expect(screen).toHaveURL(original);
      await expect(screen.getByRole('heading',{name:'入职原件',exact:true})).toHaveCount(0);
      await expect(source).toHaveCount(0);
      await screen.getByRole('link',{name:'返回办理记录'}).click();
      await expect(screen.getByRole('heading',{name:'办理记录',exact:true})).toBeVisible();
    } finally {
      sql(`DELETE FROM access.offer_grant WHERE grant_id='${offerGrant}'`);
      for (const g of jobs) sql(`DELETE FROM access.employment_grant WHERE grant_id='${g.id}'`);
      await reader.close();
    }
  }, ['hire']);
});

async function openAdditionalAssignment(page: Page, sample: Awaited<ReturnType<typeof seed>>, assignment: string) {
  const link = page.getByRole('link', { name: '新增兼岗', exact: true });
  if (page.viewportSize()?.width !== 1440 || page.viewportSize()?.height !== 900) {
    await openEmployeeAssignment(page, sample);
    await link.click();
    return;
  }

  const routePattern = '**/api/v1/personnel/directory/query';
  let release!: () => void;
  const held = new Promise<void>(resolve => { release = resolve; });
  let requested = false;
  await page.route(routePattern, async route => {
    const request = route.request();
    if (request.method() === 'POST' && request.postDataJSON()?.anchor === assignment) {
      requested = true;
      await held;
    }
    await route.continue();
  });
  try {
    await openEmployeeAssignment(page, sample);
    await expect.poll(() => requested).toBe(true);
    const pager = page.getByRole('navigation', { name: '对象导航' });
    await expect(pager).toHaveCount(0);
    await expect(page.getByRole('button', { name: '新增记录', exact: true })).toBeVisible();
    await link.scrollIntoViewIfNeeded();
    const box = await link.boundingBox();
    expect(box).not.toBeNull();
    const point = { x: box!.x + box!.width / 2, y: box!.y + box!.height / 2 };
    await page.mouse.move(point.x, point.y);
    await page.mouse.down();
    release();
    await expect(pager.getByText('1 / 1', { exact: true })).toBeVisible();
    const hit = await page.evaluate(({ x, y }) => document.elementFromPoint(x, y)?.closest('a,button')?.textContent, point);
    expect(hit).toBe('新增兼岗');
    await page.mouse.up();
    await expect(page).toHaveURL(url => url.pathname.endsWith('/additional'));
  } finally {
    release();
    await page.unroute(routePattern);
  }
}

async function secondaryPositionJourney(page:Page,sample:Awaited<ReturnType<typeof seed>>) {
  const first='2027-04-01', policy='/api/v1/assignment-policies';
  const previous:{id:string;mode:string;revision:string}[]=[];
  const setMode=async(purpose:string,mode:string)=>{
    const current=(await request(page,'GET',policy)).body;
    const before=current.rows.find((r:{category:string;purpose:string;from:string})=>r.category==='Employee'&&r.purpose===purpose&&r.from===first);
    if(before?.mode===mode)return;
    const row=before??{id:randomUUID(),category:'Employee',purpose,from:first};
    const saved=await request(page,'POST',policy,{expectedVersion:current.version,row:{...row,mode}},randomUUID());
    expect(saved.status,saved.body.message).toBe(200);
    const after=(await request(page,'GET',policy)).body.rows.find((r:{id:string})=>r.id===row.id);
    previous.push({id:row.id,mode:before?.mode??'JobOnly',revision:after.revision});
  };
  try{
    await setMode('hire','PositionCounted');await setMode('secondary','PositionCounted');
    const family=await seed308(page,'jobfamily');
    const main=await seedPosition(page,'position',family.id),second=await seedPosition(page,'position',family.id);
    for(const p of [main,second]) {
      const q=await request(page,'POST','/api/v1/staffing-quotas',{positionId:p.id,effectiveDate:first,quota:2,reason:'本岗位核两编'},randomUUID());
      expect(q.status,q.body.message).toBe(200);
    }
    const hire=await request(page,'POST','/api/v1/hires',{person:sample.id,employer:sample.employerId,position:main.id,arrival:first,notice:'主岗聘任'},randomUUID());
    expect(hire.status,hire.body.message).toBe(200);
    await openAdditionalAssignment(page, sample, hire.body.assignment);
    const positions=page.getByRole('combobox',{name:/^岗位/});
    await positions.fill(second.name);await page.getByRole('option',{name:second.name,exact:true}).click();
    await expect(page.getByText(second.department,{exact:true})).toBeVisible();await expect(page.getByText(second.job.name,{exact:true})).toBeVisible();
    const end=page.getByRole('textbox',{name:/^最后任职日/});await end.fill('2027-04-30');await end.press('Tab');
    await page.getByLabel(/任用依据/).fill('兼任第二岗位至四月底');
    const save=page.getByRole('button',{name:'保存',exact:true});await expect(save).toBeEnabled();await save.focus();await expect(save).toBeFocused();await page.keyboard.press('Enter');
    await expect(page.locator('.object-detail-identity').filter({hasText:'1 号兼岗'})).toBeVisible();
    await expect(page.getByText(second.name+' · 占编',{exact:true})).toBeVisible();
    for(const [p,day,n] of [[main,first,1],[second,first,1],[second,'2027-05-01',0]] as const) {
      const usage=await request(page,'GET','/api/v1/positions/'+p.id+'/replenishment?businessAt='+day);expect(usage.body.occupied).toBe(n);
    }
    await page.getByRole('button',{name:'新增兼岗 · '+first,exact:true}).click();
    await expect(page.getByText([second.department,second.job.name,second.name].join(' · '),{exact:true})).toBeVisible();
    await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
  } finally {
    for(const p of previous.reverse()){
      const current=(await request(page,'GET',policy)).body,row=current.rows.find((r:{id:string})=>r.id===p.id);
      if(row.revision===p.revision){
        const saved=await request(page,'POST',policy,{expectedVersion:current.version,row:{...row,mode:p.mode}},randomUUID());
        expect(saved.status,saved.body.message).toBe(200);
      }
    }
  }
}
for(const [width,height] of desktopViewports) {
  test(`M04 一人两个占编岗位 ${width}x${height}`,async({page})=>{
    test.setTimeout(160000);await signIn(page);const sample=await seed(page);await page.setViewportSize({width,height});await secondaryPositionJourney(page,sample);
  });
}
test('M04 一人两个占编岗位 原生200%',async()=>{await native200(secondaryPositionJourney)});

test('M04 人员名录的延迟响应保留姓名草稿及搜索返回位置',async({page})=>{
  test.setTimeout(90000);
  await signIn(page);
  const sample=await seed(page);
  await hireAtLocation(page,sample);
  await page.setViewportSize({width:1440,height:540});
  await openPersonnel(page);
  const input=page.getByRole('textbox',{name:'姓名或人员编号'});
  await input.fill(sample.name);
  await input.press('Enter');
  await expect(page.locator('a[data-object-link]').first()).toContainText(sample.name);
  const prior=new URL(page.url()).searchParams.get('ref');
  let finish!:()=>void,received!:()=>void;
  const arrived=new Promise<void>(resolve=>{received=resolve});
  const hold = new Promise<void>(resolve => { finish = resolve; });
  await page.route('**/api/v1/personnel/directory/query',async route => {
    const response = await route.fetch();
    received(); await hold; await route.fulfill({response});
  },{times:1});
  await input.fill(sample.name); await input.press('Enter');
  try {
    await arrived;
    await expect(page.locator('a[data-object-link]')).toHaveCount(0);
  } finally { finish(); }
  await expect(page.getByText('1 份任职，1 人',{exact:true})).toBeVisible();
  const current = new URL(page.url()).searchParams.get('ref');
  expect(current).toBeTruthy(); expect(current).not.toBe(prior);
  const employee=page.locator('a[data-object-link]').filter({hasText:sample.name});
  await expect(employee).toHaveAttribute('href',new RegExp('ref='+current));
  await employee.click(); await page.getByRole('link',{name:'返回人员',exact:true}).click();
  await expect(employee).toBeFocused(); await expect(input).toHaveValue(sample.name);
});

test('人员名录重试失败的查询步骤并保留后来输入的草稿', async ({ page }) => {
  await signIn(page);
  const suffix = randomUUID().slice(0, 8);
  const name = '名录重查' + suffix;
  const registered = await request(page, 'POST', '/api/v1/persons', { name, submissionId: randomUUID() });
  expect(registered.status).toBe(200);
  await openPersonnel(page);
  const input = page.getByRole('textbox', { name: '姓名或人员编号' });
  const link = page.getByRole('link', { name, exact: true });
  const searchPath = '/api/v1/persons/queries';
  let references = 0;
  page.on('request', req => { if (new URL(req.url()).pathname === searchPath) references++; });

  for (const [path, keyword] of [[searchPath, name], ['/api/v1/personnel/directory/query', suffix]]) {
    const endpoint = '**' + path;
    await page.route(endpoint, route => route.fulfill({
      status: 503, json: { code: 'TEMPORARY', message: '本次查询暂时失败' },
    }), { times: 1 });
    const failedRequest = page.waitForRequest(endpoint);
    await input.fill(keyword);
    await input.press('Enter');
    const attempted = (await failedRequest).postDataJSON();
    await expect(page.getByRole('alert')).toContainText('本次查询暂时失败');
    await expect(link).toHaveCount(0);
    await expect(page.getByText('没有匹配的人员', { exact: true })).toHaveCount(0);

    const beforeRetry = references;
    await input.fill('尚未提交的输入');
    const retry = page.waitForRequest(endpoint);
    await page.getByRole('button', { name: '重试', exact: true }).click();
    expect((await retry).postDataJSON()).toEqual(attempted);
    await expect(link).toBeVisible();
    await expect(input).toHaveValue('尚未提交的输入');
    await expect(page.getByRole('status').filter({ hasText: '输入已修改' })).toContainText(keyword);
    expect(references - beforeRetry).toBe(path === searchPath ? 1 : 0);
  }
});

async function filterPrimary(page: Page, value?: string) {
  await page.getByRole('button', {name:'更多操作', exact:true}).click();
  await page.getByRole('menuitem', {name:'列筛选…'}).click();
  const dialog = page.getByRole('dialog', {name:'列筛选'});
  if (value) {
    await dialog.getByRole('combobox', {name:'值'}).click();
    await page.getByRole('option', {name:value, exact:true}).click();
  } else await dialog.getByRole('button', {name:'清除条件'}).click();
  await dialog.getByRole('button', {name:'应用'}).click();
}

test('M04 桌面按主兼岗搜索并用键盘打开未来任职',async({page})=>{
  await signIn(page);const sample=await seed(page);await hireAtLocation(page,sample);
  await page.setViewportSize({width:1440,height:900});await openPersonnel(page);
  await expect(page.getByRole('columnheader',{name:'主兼岗'})).toBeVisible();
  await filterPrimary(page,'主岗');
  const input=page.getByLabel('姓名或人员编号');await input.fill(sample.name);await input.press('Enter');
  const row=page.locator('a[data-object-link]').filter({hasText:sample.name});await row.focus();await row.press('Enter');
  await expect(page.getByRole('tab',{name:'职务信息'})).toHaveAttribute('aria-selected','true');
  await expect(page.getByLabel('查看业务日')).toHaveCount(0);
});

test('按人员编号筛选兼岗并返回原行，资料读者只见本人', async ({ page, browser }) => {
  await signIn(page);
  const sample = await seed(page);
  const person = await request(page, 'GET', '/api/v1/persons/' + sample.id);
  expect(person.status).toBe(200);
  const number = person.body.number;
  const delivery = sample.department + '交付部';
  let parentId = sample.departmentId;
  for (const name of [sample.department + '区域', sample.department + '中心', delivery]) {
    const child = await apiDepartment(page, sample.buId, { name, parent: parentId });
    parentId = child.id;
  }
  const { hire } = await hireAtLocation(page, { ...sample, department: delivery, departmentId: parentId });
  await ensureJobOnlyPolicy(page, 'secondary');
  const primaryPath = '/api/v1/assignments/' + hire.assignment;
  const teams = [
    { name: sample.department + '一线组', from: '2027-05-01', record: 1 },
    { name: sample.department + '质量组', from: '2027-05-02', record: 2 },
  ];
  for (const { name, from, record } of teams) {
    const team = await apiDepartment(page, sample.buId, { name, parent: parentId });
    const current = await request(page, 'GET', primaryPath + '?businessAt=' + from);
    expect(current.status).toBe(200);
    const added = await request(page, 'POST', primaryPath + '/additional-assignments', {
      from, expectedSourceVersion: current.body.sourceVersion, parallel: false,
      department: team.id, job: sample.jobId, notice: '兼岗名录验收',
    }, randomUUID());
    expect(added.status, added.body.message).toBe(200);
    expect(added.body.recordNumber).toBe(record);
  }

  await page.setViewportSize({ width: 1440, height: 900 });
  await openPersonnel(page);
  const search = page.getByRole('textbox', { name: '姓名或人员编号' });
  await search.fill(number);
  await search.press('Enter');
  await expect(page.getByText('3 份任职，1 人', { exact: true })).toBeVisible();
  const node = (name: string) => page.getByRole('treeitem', { name, exact: true });
  const row = (record: number) => page.getByRole('row').filter({
    has: page.getByRole('gridcell', { name: String(record), exact: true }),
  });
  await expect(page.getByRole('columnheader').first().getByRole('checkbox')).toHaveAccessibleName('全选查询结果');
  await expect(page.getByRole('combobox', { name: '主兼岗', exact: true })).toHaveCount(0);
  await expect(row(0).getByRole('gridcell', { name: sample.job, exact: true })).toBeVisible();
  const openColumns = async () => {
    await page.getByRole('button', { name: '更多操作', exact: true }).click();
    await page.getByRole('menuitem', { name: '列设置…' }).click();
  };
  await openColumns();
  await page.getByRole('checkbox', { name: '职务', exact: true }).uncheck();
  await page.keyboard.press('Escape');
  await expect(page.getByRole('columnheader', { name: '职务', exact: true })).toHaveCount(0);
  await expect(page.getByRole('columnheader', { name: '岗位', exact: true })).toBeVisible();
  await openColumns();
  await page.getByRole('checkbox', { name: '职务', exact: true }).check();
  await page.keyboard.press('Escape');
  await expect(page.getByRole('columnheader', { name: '职务', exact: true })).toBeVisible();
  await expect(node(delivery)).toHaveAttribute('aria-level', '4');
  await node(delivery).click();
  await expect(page.getByRole('button', { name: '清除部门范围 · ' + delivery, exact: true })).toBeVisible();
  const descendants = page.getByRole('checkbox', { name: '含子部门', exact: true });
  await expect(descendants).toBeChecked();
  await descendants.click();
  await expect(page.getByText('1 份任职，1 人', { exact: true })).toBeVisible();
  await expect(descendants).not.toBeChecked();
  await expect(row(0)).toContainText(delivery);
  await descendants.click();
  await expect(page.getByText('3 份任职，1 人', { exact: true })).toBeVisible();
  await expect(descendants).toBeChecked();
  for (const team of teams) {
    await expect(node(team.name)).toHaveAttribute('aria-level', '5');
    await node(team.name).click();
    await expect(page.getByText('1 份任职，1 人', { exact: true })).toBeVisible();
    await expect(row(team.record)).toContainText(team.name);
  }
  await node(delivery).click();
  await expect(page.getByText('3 份任职，1 人', { exact: true })).toBeVisible();
  await filterPrimary(page, '兼岗');
  await expect(page.getByText('2 份任职，1 人', { exact: true })).toBeVisible();
  await page.getByRole('checkbox', { name: '全选查询结果', exact: true }).click();
  await expect(page.getByText('已选择2项', { exact: true })).toBeVisible();
  await row(1).getByRole('checkbox', { name: '取消选择人员', exact: true }).click();
  await expect(page.getByText('已选择1项', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: '取消选择', exact: true }).click();
  const second = row(2).getByRole('link', { name: sample.name, exact: true });
  await second.focus();
  await page.keyboard.press('Enter');
  await expect(page.locator('.object-detail-identity')).toContainText('2 号兼岗');
  await page.getByRole('link', { name: '返回人员', exact: true }).click();
  await expect(second).toBeFocused();
  await expect(search).toHaveValue(number);
  await expect(node(delivery)).toHaveAttribute('aria-selected', 'true');
  await expect(descendants).toBeChecked();
  await expect(page.getByRole('button',{name:'主兼岗：兼岗'})).toBeVisible();

  const reader = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  try {
    const limited = await reader.newPage();
    await signIn(limited, 'reader');
    await openPersonnel(limited);
    const limitedSearch = limited.getByRole('textbox', { name: '姓名或人员编号' });
    await limitedSearch.fill(number);
    await limitedSearch.press('Enter');
    await expect(limited.getByText('0 份任职，1 人', { exact: true })).toBeVisible();
    await expect(limited.getByRole('tree')).toHaveCount(0);
    for (const name of ['主兼岗', '部门', '职务', '岗位']) {
      await expect(limited.getByRole('columnheader', { name, exact: true })).toHaveCount(0);
    }
    await limited.getByRole('button', { name: '更多操作', exact: true }).click();
    await expect(limited.getByRole('menuitem', { name: '列筛选…' })).toHaveCount(0);
    await limited.keyboard.press('Escape');
    const self = limited.getByRole('link', { name: sample.name, exact: true });
    await expect(self).toHaveCount(1);
    await self.click();
    await expect(limited.getByRole('tab', { name: '个人资料' })).toBeVisible();
    await expect(limited.getByRole('tab', { name: '职务信息' })).toHaveCount(0);
  } finally {
    await reader.close();
  }
});

async function assignmentTabsJourney(page: Page, sample: Awaited<ReturnType<typeof seed>>) {
  await concurrentJourney(page, sample);
  await expect(page.getByLabel('姓名或人员编号')).toHaveValue(sample.name);
  await expect(page.getByText('3 份任职，1 人',{exact:true})).toBeVisible();
  const rows = page.locator('a[data-object-link]').filter({ hasText: sample.name });
  await expect(rows).toHaveCount(3);
  await filterPrimary(page, '兼岗');
  await expect(page.getByText('2 份任职，1 人', { exact: true })).toBeVisible();
  await expect(rows).toHaveCount(2);
  await filterPrimary(page);
  await expect(rows).toHaveCount(3);
  const primary = rows.first();
  await primary.focus();
  await page.keyboard.press('Enter');
  await expect(page.locator('.object-detail-identity').filter({hasText:'0 号主岗'})).toBeVisible();
  const tab = (name: string) => page.getByRole('tab', { name, exact: true });
  await tab('个人资料').click();
  await expect(tab('个人资料')).toHaveAttribute('aria-selected', 'true');
  await expect(tab('职务信息')).toBeVisible();
  await tab('变更日志').focus();
  await page.keyboard.press('Enter');
  await expect(tab('变更日志')).toHaveAttribute('aria-selected', 'true');
  await tab('职务信息').focus();
  await page.keyboard.press('Enter');
  await expect(page.locator('.object-detail-identity').filter({hasText:'0 号主岗'})).toBeVisible();
  await expect(tab('职务信息')).toHaveAttribute('aria-selected', 'true');
  await page.getByRole('link', { name: /返回/ }).first().click();
  await expect(primary).toBeFocused();
  await expect(page.getByRole('textbox',{name:'业务日期'})).toHaveCount(0);
  await rows.nth(1).click();
  await expect(page.locator('.object-detail-identity').filter({hasText:'1 号兼岗'})).toBeVisible();
  await expect(page.getByRole('tab')).toHaveCount(0);
  await expect(page.getByRole('heading',{name:'职务信息',exact:true})).toBeVisible();
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
}

for (const [width, height] of desktopViewports) {
  test(`M04 从人员选择本人任职并切换主岗页签 ${width}x${height}`, async ({ page }) => {
    test.setTimeout(180000);
    await signIn(page);
    const sample = await seed(page);
    await page.setViewportSize({ width, height });
    await assignmentTabsJourney(page, sample);
  });
}
test('M04 从人员选择本人任职并切换主岗页签 原生200%', async () => {
  await native200(assignmentTabsJourney);
});

test('M04 主岗、个人日志和撤销原单各按当前授权重新读取', async ({ page, browser }) => {
  test.setTimeout(150000);
  await signIn(page);
  const sample = await seed(page);
  const { hire } = await hireAtLocation(page, sample);
  const reader = await browser.newContext({ baseURL: localConnection().webOrigin });
  const sql = database();
  const denied = randomUUID();
  const grants = ['directory', 'read', 'history'].map((action) => ({ id: randomUUID(), action }));
  function allowJob() {
    for (const grant of grants) {
      sql(`INSERT INTO access.employment_grant(instance_id,principal_id,grant_id,action,person_id,fields)
        SELECT instance_id,principal_id,'${grant.id}','${grant.action}','${sample.id}',ARRAY['arrangement','basis','location']
        FROM access.principal WHERE login='reader'`);
    }
  }
  function removeJob() {
    for (const grant of grants) sql(`DELETE FROM access.employment_grant WHERE grant_id='${grant.id}'`);
  }
  try {
    allowJob();
    sql(`INSERT INTO access.person_grant(instance_id,principal_id,grant_id,action,purpose,person_ids,fields,denied)
      SELECT instance_id,principal_id,'${denied}','history.read','history',ARRAY['${sample.id}'::uuid],ARRAY['name','number'],true
      FROM access.principal WHERE login='reader'`);
    const screen = await reader.newPage();
    await signIn(screen, 'reader');
    await openEmployeeAssignment(screen, sample);
    const assignmentURL = screen.url();
    const tab = (name: string) => screen.getByRole('tab', { name, exact: true });
    await tab('个人资料').click();
    await expect(tab('职务信息')).toBeVisible();
    await expect(tab('变更日志')).toHaveCount(0);
    const profileURL = screen.url();
    removeJob();
    sql(`DELETE FROM access.person_grant WHERE grant_id='${denied}'`);
    await screen.reload();
    await tab('变更日志').click();
    await expect(tab('变更日志')).toHaveAttribute('aria-selected', 'true');
    await expect(tab('职务信息')).toHaveCount(0);
    await screen.goto(assignmentURL);
    await expect(screen.getByRole('alert')).toBeVisible();
    await expect(tab('职务信息')).toHaveCount(0);
    await screen.goBack();
    await expect(tab('变更日志')).toHaveAttribute('aria-selected', 'true');
    await expect(tab('职务信息')).toHaveCount(0);

    allowJob();
    const cancelled = await request(page, 'POST', '/api/v1/employment-actions/' + hire.action + '/cancel',
      { revision: hire.revision, reason: '原通知在到岗前已撤回' }, randomUUID());
    expect(cancelled.status, cancelled.body.message).toBe(200);
    await screen.goto(profileURL);
    await screen.getByRole('link', { name: '本人办理历史', exact: true }).click();
    await expect(screen.getByRole('heading',{name:'本人办理历史'})).toBeVisible();
    await expect(screen.getByText(/入职（已撤销）/)).toBeVisible();
    await screen.getByRole('button', { name: '查看原办理', exact: true }).first().click();
    await expect(screen.getByText('首任职 ' + hire.assignment, { exact: true })).toBeVisible();
  } finally {
    removeJob();
    sql(`DELETE FROM access.person_grant WHERE grant_id='${denied}'`);
    await reader.close();
  }
});

test('M05 交接应答在按下和松开之间到达时仍打开改期', async ({ page }) => {
  test.setTimeout(90000);
  await signIn(page);
  const person = await seed(page);
  await page.setViewportSize({ width: 1440, height: 540 });

  await withCountedPolicies(page, async () => {
    const { position } = await recruitingJourney(page, person);
    const button = (name: string) => page.getByRole('button', { name, exact: true });
    const offerId = new URL(page.url()).pathname.split('/').pop();
    const offerPath = '/api/v1/recruiting/offers/' + offerId;
    const seats = async () => {
      const usage = await request(page, 'GET', '/api/v1/positions/' + position.id + '/replenishment?businessAt=2027-05-01');
      expect(usage.status).toBe(200);
      return [usage.body.reserved, usage.body.occupied];
    };
    const current = await request(page, 'GET', offerPath);
    expect(await seats()).toEqual([1, 0]);
    await expect(button('按录用办理入职')).toBeVisible();
    await button('2027-04-01 · 发出录用').click();
    await expect(page.getByText('已发出', { exact: true })).toBeVisible();

    await expect(page).toHaveURL(url=>!!url.searchParams.get('sourceVersion'));
    await page.reload();
    await expect(page.getByText('已发出',{exact:true})).toBeVisible();
    await expect(button('调整录用日期')).toHaveCount(0);
    const handoff = (url: URL) => url.pathname === offerPath + '/hire-handoff' &&
      url.searchParams.get('sourceVersion') === current.body.version;
    let release!: () => void, requested = false;
    const held = new Promise<void>(resolve => { release = resolve; });
    await page.route(handoff, async route => {
      requested = true;
      await held;
      await route.continue();
    });
    const reschedule = button('调整录用日期');
    try {
      await button('返回当前录用').click();
      await expect(page.getByText('已接受', { exact: true })).toBeVisible();
      await expect.poll(() => requested).toBe(true);
      await expect(button('按录用办理入职')).toHaveCount(0);
      await reschedule.scrollIntoViewIfNeeded();
      const box = await reschedule.boundingBox();
      expect(box).not.toBeNull();
      const x = box!.x + box!.width / 2, y = box!.y + box!.height / 2;
      const hit = () => page.evaluate(([x, y]) => document.elementFromPoint(x, y)?.closest('button')?.textContent, [x, y]);
      expect(await hit()).toBe('调整录用日期');
      await page.mouse.move(x, y);
      await page.mouse.down();
      release();
      await expect(button('按录用办理入职')).toBeVisible();
      expect(await hit()).toBe('调整录用日期');
      await page.mouse.up();
    } finally {
      release();
      await page.unroute(handoff);
    }

    const arrival = page.getByRole('textbox', { name: /^约定到岗日/ });
    await expect(arrival).toHaveValue('2027-05-01');
    await button('取消').click();
    await expect(reschedule).toBeFocused();
    await page.keyboard.press('Enter');
    await expect(arrival).toHaveValue('2027-05-01');
    await button('取消').click();
    const after = await request(page, 'GET', offerPath);
    expect(after.body.version).toBe(current.body.version);
    expect(await seats()).toEqual([1, 0]);
    await expect(button('按录用办理入职')).toBeVisible();
    await reschedule.click();
    await expect(arrival).toBeVisible();
    await page.goBack();
    await expect(page.getByText('已发出',{exact:true})).toBeVisible();
    await expect(arrival).toHaveCount(0);
    await button('返回当前录用').click();
    await expect(reschedule).toBeVisible();
    await expect(arrival).toHaveCount(0);
  }, ['hire']);
});

async function profileAssignmentDateJourney(page:Page,sample:Awaited<ReturnType<typeof seed>>) {
  const post = async (path:string, input:object) => {
    const r = await request(page,'POST','/api/v1/'+path,input,randomUUID());
    expect(r.status,r.body.message).toBe(200);
    return r.body;
  };
  type Rule = {id:string; purpose:string; category:string; mode:string; from:string; until?:string; revision:string};
  const policyPath = 'assignment-policies';
  const policies = async () => (await request(page, 'GET', '/api/v1/'+policyPath)).body as {version:string; rows:Rule[]};
  const restored: { rule:Rule; revision:string }[] = [];
  try {
    for (const [purpose, at] of [['hire','2026-04-01'], ['transfer','2026-07-01']]) {
      const rows = (await policies()).rows
        .filter(row => row.purpose === purpose && row.category === 'Employee' && (!row.until || row.until > at))
        .sort((a,b) => a.from.localeCompare(b.from));
      const fillGap = async (from:string, until?:string) => {
        await post(policyPath, {expectedVersion:(await policies()).version,
          row:{id:randomUUID(), category:'Employee', purpose, from, until, mode:'JobOnly'}});
      };
      // An open-ended hire and its subsequent transfer each need a continuous
      // policy. Other journeys may have adopted an April-only counted policy.
      let from: string | undefined = at;
      for (const rule of rows) {
        if (from && from < rule.from) await fillGap(from, rule.from);
        if (rule.mode !== 'JobOnly') {
          await post(policyPath, {expectedVersion:(await policies()).version, row:{...rule, mode:'JobOnly'}});
          const written = (await policies()).rows.find(r => r.id === rule.id)!;
          restored.push({rule, revision:written.revision});
        }
        from = rule.until;
      }
      if (from) await fillGap(from);
    }
    const hire = await post('hires',{person:sample.id,employer:sample.employerId,department:sample.departmentId,job:sample.jobId,arrival:'2026-04-01',notice:'档案日期样本'});
    const path = 'assignments/'+hire.assignment;
    const term = async () => (await request(page,'GET','/api/v1/'+path)).body;
    const destinationName = '七月调入'+randomUUID().slice(0,8);
    const dest = await apiDepartment(page,sample.buId,{name:destinationName});
    await post(path+'/transfers',{expectedSourceVersion:(await term()).sourceVersion,businessDate:'2026-07-01',department:dest.id,job:sample.jobId,notice:'七月调动'});
    const current = await term();
    expect(current.state.departmentId).toBe(dest.id);
    await openPerson(page,sample.name);
    await page.getByRole('tab',{name:'职务信息'}).click();
    await expect(page.locator('[aria-current=date]')).toHaveText(current.businessAt);
    await expect(page.getByText(destinationName+' · '+sample.job,{exact:true})).toBeVisible();
    await chooseEffectiveDate(page,'2026-04-01');
    await page.reload();await expect(page.locator('[aria-current=date]')).toHaveText('2026-04-01');
    await expect(page.getByText(sample.department+' · '+sample.job,{exact:true})).toBeVisible();
  } finally {
    for (const {rule,revision} of restored.reverse()) {
      const policy = await policies();
      const current = policy.rows.find(r => r.id === rule.id);
      if (current?.revision === revision) {
        await post(policyPath, {expectedVersion:policy.version, row:{...current, mode:rule.mode}});
      }
    }
  }
}

for (const [width,height] of desktopViewports) {
  test(`M04 档案唯一任职的初入日与刷新 ${width}x${height}`, async ({page}) => {
    test.setTimeout(100000);
    await signIn(page);
    const sample=await seed(page,true,'2026-01-01');
    await page.setViewportSize({width,height});
    await profileAssignmentDateJourney(page,sample);
  });
}
test('M04 档案唯一任职的初入日与刷新 原生200%',async()=>{
  await native200(profileAssignmentDateJourney,page=>seed(page,true,'2026-01-01'));
});

test('M04 员工列表跨第21份任职后返回姓名、页码和原行', async ({page}) => {
  test.setTimeout(150000);
  await signIn(page);
  const sample = await seed(page);
  const {hire} = await hireAtLocation(page,sample);
  await ensureJobOnlyPolicy(page, 'secondary');
  const path = '/api/v1/assignments/'+hire.assignment;
  for (let n=1; n<=20; n++) {
    const from = '2027-05-'+String(n).padStart(2,'0');
    const current = await request(page,'GET',path+'?businessAt='+from);
    const added = await request(page,'POST',path+'/additional-assignments',{
      from,expectedSourceVersion:current.body.sourceVersion,parallel:false,department:sample.departmentId,job:sample.jobId,notice:'按原任用期间增加兼岗',
    },randomUUID());
    expect(added.status,added.body.message).toBe(200);
    expect(added.body.recordNumber).toBe(n);
  }
  await openEmployeeAssignment(page,sample,'2027-05-21',21,hire.assignment);
  await page.getByRole('link',{name:'返回人员',exact:true}).click();
  const search = page.getByRole('textbox',{name:'姓名或人员编号'});
  await expect(search).toHaveValue(sample.name);
  await page.getByRole('button',{name:'下一页',exact:true}).first().click();
  const last = page.locator('a[data-object-link]').filter({hasText:sample.name});
  await expect(last).toHaveCount(1);
  await expect(last).toHaveAttribute('href',/page=2/);
  await search.fill('仅在第二页输入的草稿');
  await page.goBack();
  await expect(page).toHaveURL(u => u.searchParams.get('page') === '1');
  await expect(search).toHaveValue(sample.name);
  await page.goForward();
  await expect(last).toHaveCount(1);
  await expect(last).toHaveAttribute('href',/page=2/);
  await expect(search).toHaveValue(sample.name);
  await last.click();
  await expect(page.locator('.object-detail-identity').filter({hasText:'20 号兼岗'})).toBeVisible();
  await page.getByRole('link',{name:'返回人员',exact:true}).click();
  await expect(last).toBeFocused();
  await expect(search).toHaveValue(sample.name);
  await expect(page).toHaveURL(u=>u.searchParams.get('page')==='2' && !u.searchParams.has('businessAt'));
});

test('M04 办理列表的姓名慢应答保留新输入并遵从浏览器返回', async ({page}) => {
  test.setTimeout(120000);
  await signIn(page);
  const sample = await seed(page);
  await hireAtLocation(page,sample);
  await openRegister(page);
  const search = page.getByRole('textbox',{name:'姓名或人员编号'});
  async function submitWhile(change: () => Promise<void>) {
    let arrived!: () => void, release!: () => void;
    const received = new Promise<void>(resolve => {arrived = resolve;});
    const hold = new Promise<void>(resolve => {release = resolve;});
    await page.route('**/api/v1/persons/queries', async route => {
      const response = await route.fetch();
      arrived();
      await hold;
      await route.fulfill({response});
    }, {times:1});
    const finished = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/persons/queries' && r.ok());
    await search.press('Enter');
    try {
      await received;
      await change();
    } finally {
      release();
    }
    await finished;
  }
  await search.fill(sample.name);
  await submitWhile(() => search.fill('后输入'+sample.name));
  await expect(page.getByText('1 笔办理',{exact:true})).toBeVisible();
  await expect(search).toHaveValue('后输入'+sample.name);
  await expect(page.getByRole('status').filter({hasText:'姓名或人员编号已修改'})).toBeVisible();
  const applied = page.url();
  await submitWhile(async () => {
    await page.goBack();
    await expect(page).toHaveURL(url => url.pathname === '/personnel/actions' && !url.searchParams.has('ref'));
  });
  await expect(search).toHaveValue('');
  await expect(page).toHaveURL(url => !url.searchParams.has('ref'));
  await page.goForward();
  await expect(page).toHaveURL(applied);
  await expect(page.getByText('1 笔办理',{exact:true})).toBeVisible();
  await expect(search).toHaveValue(sample.name);
  await search.fill('无匹配排序'+sample.name);
  await page.getByRole('columnheader',{name:/生效日期/}).click();
  await expect(page).toHaveURL(url => url.searchParams.get('sort') === 'businessDate');
  await expect(page.getByText('0 笔办理',{exact:true})).toBeVisible();
  await expect(search).toHaveValue('无匹配排序'+sample.name);
  await expect(page.getByRole('status').filter({hasText:'姓名或人员编号已修改'})).toHaveCount(0);
});

test('M04 到期姓名引用可重新查询，暂时失败和撤权各按当前条件处理', async ({page,browser}) => {
  test.setTimeout(150000);
  await signIn(page);
  const sample = await seed(page);
  await hireAtLocation(page,sample);
  const sql = database();
  const grants = ['read','register.read'].map(action => ({action,id:randomUUID()}));
  const reader = await browser.newContext({baseURL:localConnection().webOrigin});
  try {
    for (const grant of grants) sql(`INSERT INTO access.employment_grant(instance_id,principal_id,grant_id,action,person_id,fields)
      SELECT instance_id,principal_id,'${grant.id}','${grant.action}','${sample.id}',ARRAY['arrangement','basis','department','operator']
      FROM access.principal WHERE login='reader'`);
    const screen = await reader.newPage();
    await signIn(screen,'reader');
    await openRegister(screen);
    const search = screen.getByRole('textbox',{name:'姓名或人员编号'});
    let created = 0;
    screen.on('request',r => {
      if (r.method() === 'POST' && new URL(r.url()).pathname === '/api/v1/persons/queries') created++;
    });
    const list = (status:number, ref?:string) => screen.waitForResponse(response => {
      const url = new URL(response.url());
      return url.pathname === '/api/v1/employment-actions' && !!url.searchParams.get('ref') && response.status() === status &&
        (!ref || url.searchParams.get('ref') === ref);
    });
    await search.fill(sample.name);
    const first = list(200);
    await search.press('Enter');
    const firstRef = new URL((await first).url()).searchParams.get('ref')!;
    await expect(screen.getByText('1 笔办理',{exact:true})).toBeVisible();
    expect(created).toBe(1);
    expect(sql(`UPDATE personnel.query SET expires_at=clock_timestamp()-interval '1 second' WHERE ref='${firstRef}' RETURNING ref`)).toBe(firstRef);
    const expired = list(404,firstRef);
    await screen.evaluate(() => window.dispatchEvent(new Event('focus')));
    await expired;
    await expect(screen.getByRole('alert')).toBeVisible();
    await expect(search).toHaveValue(sample.name);
    const renewed = list(200);
    await search.press('Enter');
    const nextRef = new URL((await renewed).url()).searchParams.get('ref')!;
    expect(nextRef).not.toBe(firstRef);
    expect(created).toBe(2);
    await expect(screen.getByText('1 笔办理',{exact:true})).toBeVisible();

    await screen.route('**/api/v1/employment-actions?*', route => route.fulfill({
      status:503, contentType:'application/json', body:JSON.stringify({code:'TEMPORARY',message:'办理查询暂不可用'}),
    }), {times:1});
    const failed = list(503,nextRef);
    await screen.evaluate(() => window.dispatchEvent(new Event('focus')));
    await failed;
    const retried = list(200,nextRef);
    await screen.getByRole('button',{name:'重试',exact:true}).click();
    await retried;
    await expect(screen.getByText('1 笔办理',{exact:true})).toBeVisible();
    expect(created).toBe(2);

    sql(`UPDATE personnel.query SET expires_at=clock_timestamp()-interval '1 second' WHERE ref='${nextRef}'`);
    sql(`DELETE FROM access.employment_grant WHERE grant_id='${grants[1].id}'`);
    const denied = list(403,nextRef);
    await search.press('Enter');
    await denied;
    await expect(search).toHaveCount(0);
    await expect(screen.getByText(sample.name,{exact:true})).toHaveCount(0);
    expect(created).toBe(2);
  } finally {
    for (const grant of grants) sql(`DELETE FROM access.employment_grant WHERE grant_id='${grant.id}'`);
    await reader.close();
  }
});

test('M04 办理原件沿姓名与类型跨第 20 和 21 笔', async ({page}) => {
  test.setTimeout(150000);
  await signIn(page);
  const sample=await seed(page);
  const {locations,hire}=await hireAtLocation(page,sample);
  const path='/api/v1/assignments/'+hire.assignment;
  const changes: {action:string; revision:string}[] = [];
  for(let n=1;n<=21;n++){
    const date='2027-05-'+String(n).padStart(2,'0');
    const current=await request(page,'GET',path+'?businessAt='+date);
    const saved=await request(page,'POST',path+'/location-changes',{businessDate:date,expectedSourceVersion:current.body.sourceVersion,location:locations[n%2].id},randomUUID());
    expect(saved.status,saved.body.message).toBe(200);
    changes.push(saved.body);
  }
  const releaseList = await holdRead(page, url => url.pathname === '/api/v1/employment-actions' &&
    !url.searchParams.has('ref') && !url.searchParams.has('kind'));
  const search=page.getByRole('textbox',{name:'姓名或人员编号'});
  const kind=page.getByLabel('办理类型',{exact:true});
  try {
    await openRegister(page);
    await search.fill(sample.name);
    await page.getByText('筛选条件',{exact:true}).click();
    await kind.selectOption('DataChange');
  } finally {
    releaseList();
  }
  await expect(page.getByText(/^\d+ 笔办理$/)).toBeVisible();
  await expect(kind).toHaveValue('DataChange');
  await page.getByRole('button',{name:'应用筛选'}).click();
  await expect(page.getByText('21 笔办理',{exact:true})).toBeVisible();
  const applied = new URL(page.url()).searchParams.get('ref');
  await search.fill('只留在输入框'+sample.name);
  await page.getByRole('button',{name:'下一页',exact:true}).first().click();
  const row=page.locator('a[data-object-link]').filter({hasText:'数据更改'});
  await expect(row).toHaveCount(1);
  await expect(search).toHaveValue('只留在输入框'+sample.name);
  await expect(page).toHaveURL(url=>url.searchParams.get('ref')===applied && url.searchParams.get('page')==='2');
  await page.goBack();
  await expect(page.getByText('21 笔办理',{exact:true})).toBeVisible();
  await expect(search).toHaveValue(sample.name);
  await search.fill('下一次输入'+sample.name);
  await page.goForward();
  await expect(row).toHaveCount(1);
  await expect(search).toHaveValue(sample.name);
  await row.click();
  await expect(page.getByLabel('第21个，共21个匹配对象')).toBeVisible();
  await expect(page.getByRole('button',{name:'下一个对象'})).toBeDisabled();
  await page.getByRole('button',{name:'上一个对象'}).click();
  await expect(page.getByLabel('第20个，共21个匹配对象')).toBeVisible();
  await expect(page).toHaveURL(url=>url.searchParams.get('page')==='1');
  await page.getByRole('button',{name:'下一个对象'}).click();
  await expect(page.getByLabel('第21个，共21个匹配对象')).toBeVisible();
  await page.getByRole('link',{name:'返回办理记录'}).click();
  await expect(row).toBeFocused(); await expect(search).toHaveValue(sample.name);
  await expect(page).toHaveURL(url=>url.searchParams.get('page')==='2' && url.searchParams.get('kind')==='DataChange');
  const detail=await row.getAttribute('href');
  const plain=new URL(detail!,page.url()).pathname;
  await page.goto(plain);
  await expect(page.getByRole('heading',{name:'数据更改原件',exact:true})).toBeVisible();
  await expect(page.getByRole('navigation',{name:'对象导航'})).toHaveCount(0);
  await page.route('**/api/v1/employment-actions?*',route=>route.abort());
  await page.goto(detail!);
  await expect(page.getByRole('heading',{name:'数据更改原件',exact:true})).toBeVisible();
  await expect(page.getByRole('button',{name:'重试导航'})).toBeVisible();
  await page.unroute('**/api/v1/employment-actions?*');
  await page.getByRole('button',{name:'重试导航'}).click();
  await expect(page.getByLabel('第21个，共21个匹配对象')).toBeVisible();

  const excluded = new URL(detail!,page.url());
  excluded.searchParams.set('businessFrom','2028-01-01');
  await page.goto(excluded.href);
  await expect(page.getByText('这笔办理已不在当前查询范围内。')).toBeVisible();
  await expect(page.getByRole('heading',{name:'数据更改原件',exact:true})).toBeVisible();
  const recheck = page.waitForResponse(r => {
    const url = new URL(r.url());
    return url.pathname === '/api/v1/employment-actions' && url.searchParams.has('anchor');
  });
  await page.getByRole('button',{name:'重试导航'}).click();
  const result = await recheck;
  expect(result.status()).toBe(200);
  expect((await result.json()).navigation).toBeUndefined();
  await expect(page.getByText('这笔办理已不在当前查询范围内。')).toBeVisible();
  await expect(page).toHaveURL(excluded.href);

  const current = new URL(detail!,page.url());
  current.pathname = '/personnel/actions/'+changes[19].action;
  current.searchParams.set('selected',changes[19].action);
  current.searchParams.set('businessFrom','2027-05-20');
  current.searchParams.set('status','future');
  current.searchParams.set('sort','businessDate');
  current.searchParams.set('page','1');
  await page.goto(current.href);
  await expect(page.getByLabel('第1个，共2个匹配对象')).toBeVisible();
  const beforeDelete = await request(page,'GET',path+'?businessAt=2027-05-21');
  const deleted = await request(page,'POST',path+'/actions/'+changes[20].action+'/delete',{
    expectedSourceVersion:beforeDelete.body.sourceVersion,expectedRevision:changes[20].revision,
  },randomUUID());
  expect(deleted.status,deleted.body.message).toBe(200);
  await page.getByRole('button',{name:'下一个对象'}).click();
  await expect(page.getByLabel('第1个，共1个匹配对象')).toBeVisible();
  await expect(page.getByRole('button',{name:'下一个对象'})).toBeDisabled();
  await expect(page).toHaveURL(current.href);
});

test('人员名录跨页选择、排除及集合变动', async ({ page }) => {
  await signIn(page);
  const prefix = '跨页名录' + randomUUID().slice(0, 8);
  const add = async (i: number) => {
    const r = await request(page, 'POST', '/api/v1/persons', {
      name: prefix + String(i).padStart(2, '0'), submissionId: randomUUID(),
    });
    expect(r.status, r.body.message).toBe(200);
  };
  for (let i=0; i<22; i++) await add(i);
  await openPersonnel(page);
  const input = page.getByRole('textbox', {name:'姓名或人员编号'});
  await input.fill(prefix);
  await input.press('Enter');
  await expect(page.getByText('0 份任职，22 人')).toBeVisible();
  await page.getByRole('checkbox', {name:'全选查询结果'}).click();
  await expect(page.getByText('已选择22项')).toBeVisible();
  await page.getByRole('checkbox', {name:'取消选择人员',exact:true}).first().click();
  await expect(page.getByText('已选择21项')).toBeVisible();
  await page.getByRole('button', {name:'下一页'}).click();
  await expect(page.getByRole('checkbox', {name:'取消选择人员',exact:true})).toHaveCount(2);
  await expect(page.getByText('已选择21项')).toBeVisible();
  await page.locator('[role=columnheader][data-field=name]').click();
  await expect(page.getByText('已选择21项')).toBeVisible();
  await add(22);
  await page.evaluate(() => window.dispatchEvent(new Event('focus')));
  await expect(page.getByText('查询结果已变化，请重新选择。')).toBeVisible();
  await expect(input).toHaveValue(prefix);
  await expect(page.getByRole('checkbox', {name:'取消选择人员',exact:true})).toHaveCount(0);
});
