import { randomUUID } from 'node:crypto';
import { expect, test, type Page } from './fixtures';
import { open } from './bu-support';
import { withPolicy } from './grants';
import { enter308, namedMenu308, seed308 } from './staffing308-support';
import { calibrateDesktopWindow, nativeZoomBrowser } from './native-zoom-support';

const base = '/api/v1/compensation/plans',
  route = '/settings/compensation/plans';
const menu = (page: Page) => namedMenu308(page, '薪酬计划', '薪酬配置', route);
const text = (page: Page, name: string) => page.getByRole('textbox', { name, exact: true });
const button = (page: Page, name: string) => page.getByRole('button', { name, exact: true });
const dialog = (page: Page) => page.getByRole('dialog', { name: '维护组成项' });
const code210 = (prefix: string) => prefix + randomUUID().slice(0, 8).toUpperCase();
async function select(page: Page, name: string, label: string) {
  await page.getByRole('combobox', { name, exact: true }).click();
  await page.getByRole('option', { name: label, exact: true }).click();
}
async function post(page: Page, path: string, data: unknown, expected = 200) {
  const { csrf } = await (await page.request.get('/api/v1/session')).json();
  const response = await page.request.post(path, {
    headers: { Origin: new URL(page.url()).origin, 'X-CSRF-Token': csrf, 'Idempotency-Key': randomUUID() },
    data,
  });
  const result = await response.json();
  expect(response.status(), JSON.stringify(result)).toBe(expected);
  return result;
}
async function start(page: Page, code: string, name: string) {
  await menu(page);
  await page.getByRole('link', { name: '新建', exact: true }).click();
  await expect(text(page, '生效日期')).toHaveValue('1900-01-01');
  await text(page, '生效日期').fill('2027-01-01');
  await select(page, '办理原因', '建立计划');
  await text(page, '计划编码').fill(code);
  await text(page, '计划名称').fill(name);
  await select(page, '适用方式', '企业通用');
}
async function fixed(page: Page, amount: string) {
  await button(page, '添加组成项').click();
  await text(page, '项编码').fill('ALLOW');
  await text(page, '项名称').fill('岗位津贴');
  await select(page, '性质', '津贴补贴');
  await text(page, '固定金额').fill(amount);
  await dialog(page).getByRole('button', { name: '确定' }).click();
}
async function save(page: Page, label = '保存') {
  const result = page.waitForResponse(
    (r) => r.request().method() === 'POST' && new URL(r.url()).pathname.startsWith(base),
  );
  await button(page, label).click();
  const r = await result;
  expect(r.status(), await r.text()).toBe(200);
  await expect(page.locator('#compensation-plan-form')).toHaveCount(0);
}
async function more(page: Page, action: string) {
  await button(page, '更多操作').click();
  await page.getByRole('menuitem', { name: action, exact: true }).click();
}
for (const height of [900, 540]) {
  test('210 薪酬计划企业通用、十月津贴、取消与原件 ' + height, async ({ page }) => {
    await page.setViewportSize({ width: 1440, height });
    await open(page);
    const code = code210('PF'),
      name = '单项津贴 ' + code;
    await start(page, code, name);
    await fixed(page, '300');
    await save(page);
    const table = page.getByRole('table', { name: '薪酬组成', exact: true });
    await expect(table).toContainText('300.00 人民币元/月');
    await button(page, '新增记录').click();
    await text(page, '生效日期').fill('2027-10-01');
    await select(page, '办理原因', '薪酬制度调整');
    await button(page, '编辑 岗位津贴').click();
    await expect(text(page, '固定金额')).toHaveValue('300.00');
    await text(page, '固定金额').fill('999');
    await dialog(page).getByRole('button', { name: '取消' }).click();
    await button(page, '编辑 岗位津贴').click();
    await expect(text(page, '固定金额')).toHaveValue('300.00');
    await text(page, '固定金额').fill('350');
    await dialog(page).getByRole('button', { name: '确定' }).click();
    await save(page);
    await expect(table).toContainText('350.00 人民币元/月');
    await button(page, '2027-01-01').click();
    await expect(table).toContainText('300.00 人民币元/月');
    await page.getByRole('tab', { name: '变更日志' }).click();
    await page.getByRole('link', { name: '查看当次资料' }).first().click();
    await expect(page.getByText('正在查看只读历史资料', { exact: true })).toBeVisible();
    await expect(table).toContainText('350.00 人民币元/月');
    await expect(button(page, '更多操作')).toHaveCount(0);
    await button(page, '返回变更日志').click();
    await page.getByRole('link', { name: '返回列表' }).click();
    await text(page, '搜索计划编码或名称').fill(code);
    await text(page, '搜索计划编码或名称').press('Enter');
    await expect(page.getByRole('link', { name, exact: true })).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
  });
}
test('210 计划引用年标准、百分比年目标及保存时的薪标原件', async ({ page }) => {
  await open(page);
  const code = code210('PY');
  const grade = await post(
    page,
    '/api/v1/grades',
    { code, name: '年薪职级', effectiveDate: '2027-01-01', operationType: 'CREATE', operationReason: 'NEW_GRADE' },
    201,
  );
  await post(page, '/api/v1/compensation/grade-rates', {
    action: 'CREATE',
    gradeId: grade.objectId,
    currency: 'CNY',
    frequency: 'YEAR',
    unit: 'PER_YEAR',
    effectiveDate: '2027-01-01',
    reason: '建立标准',
    changes: { form: 'FIXED', amount: '180000' },
  });
  await start(page, code, '年薪制度 ' + code);
  await button(page, '添加组成项').click();
  await text(page, '项编码').fill('BASIC');
  await text(page, '项名称').fill('基本工资');
  await select(page, '约定频率', '年');
  await select(page, '取值方式', '职级标准');
  await page.getByRole('combobox', { name: '职级及引用标准' }).fill(code);
  await page.getByRole('option', { name: new RegExp(code) }).click();
  await expect(page.getByRole('listbox')).toHaveCount(0);
  await dialog(page).getByRole('button', { name: '确定' }).click();
  await button(page, '添加组成项').click();
  await text(page, '项编码').fill('BONUS');
  await text(page, '项名称').fill('目标奖金');
  await select(page, '性质', '奖金');
  await select(page, '约定频率', '年');
  await select(page, '取值方式', '目标比例');
  await select(page, '基准项', 'BASIC 基本工资');
  await text(page, '目标比例（%）').fill('12.50');
  await dialog(page).getByRole('button', { name: '确定' }).click();
  await save(page);
  const table = page.getByRole('table', { name: '薪酬组成', exact: true });
  await expect(table).toContainText('参考额 22,500.00');
  await expect(table).toContainText('180,000.00');
  await text(page, '适用日期').fill('2027-06-30');
  await button(page, '查看该日').click();
  await table.getByRole('link', { name: new RegExp(code) }).click();
  await expect(
    page
      .locator('.object-detail-field')
      .filter({ has: page.getByText('定额', { exact: true }) })
      .last(),
  ).toContainText('180,000.00');
  await page.getByRole('link', { name: '返回薪酬计划' }).click();
  await expect(text(page, '适用日期')).toHaveValue('2027-06-30');
  await table.getByRole('button', { name: '来源说明' }).click();
  await table.getByRole('link', { name: /查看保存时依据/ }).click();
  await expect(page.getByText('正在查看只读历史资料', { exact: true })).toBeVisible();
  await page.getByRole('link', { name: '返回薪酬计划' }).click();
  await expect(table.getByRole('link', { name: /查看保存时依据/ })).toBeFocused();
});
test('210 计划并发重读和丢失响应查回', async ({ page }) => {
  await open(page);
  const code = code210('PC');
  await start(page, code, code);
  await fixed(page, '0');
  await save(page);
  await more(page, '更正记录');
  await button(page, '编辑 岗位津贴').click();
  await text(page, '固定金额').fill('999');
  await page.evaluate(() => window.history.back());
  const guard = page.getByRole('dialog', { name: '尚有未保存修改' });
  await expect(guard).toBeVisible();
  await guard.getByRole('button', { name: '继续编辑' }).click();
  await expect(text(page, '固定金额')).toHaveValue('999');
  await dialog(page).getByRole('button', { name: '取消' }).click();
  await button(page, '取消').click();
  await expect(page.getByRole('table', { name: '薪酬组成', exact: true })).toContainText('0.00');
  const path = base + '/' + new URL(page.url()).pathname.split('/').at(-1);
  await more(page, '更正记录');
  await text(page, '说明').fill('经办人填写的说明');
  const old = await (await page.request.get(path)).json();
  await post(page, path, {
    action: 'CORRECT',
    recordId: old.record.recordId,
    expectedRevision: old.record.revision,
    expectedVersion: old.sourceVersion,
    changes: { name: '另一位管理员更名' },
  });
  await button(page, '保存').click();
  await button(page, '重新读取').click();
  await expect(text(page, '说明')).toHaveValue('经办人填写的说明');
  await expect(text(page, '计划名称')).toHaveValue('另一位管理员更名');
  await save(page);
  await more(page, '更正记录');
  await text(page, '说明').fill('查回原提交');
  await page.route(
    (url) => url.pathname === path,
    async (r) => {
      if (r.request().method() !== 'POST') return r.continue();
      expect((await r.fetch()).status()).toBe(200);
      await r.abort('failed');
    },
  );
  await button(page, '保存').click();
  await expect(page.getByText('查回原提交', { exact: true })).toBeVisible();
  expect((await (await page.request.get(path + '/history')).json()).items).toHaveLength(4);
});
test('210 计划原生 200% 通过键盘提交组成与保存', async () => {
  const zoom = await nativeZoomBrowser();
  try {
    const { page } = zoom;
    await open(page);
    await calibrateDesktopWindow(page);
    await zoom.setZoom(2);
    const code = code210('PZ');
    await start(page, code, code);
    await fixed(page, '200');
    await button(page, '保存').focus();
    await page.keyboard.press('Enter');
    await expect(page.getByRole('table', { name: '薪酬组成', exact: true })).toContainText('200.00 人民币元/月');
    expect(await page.evaluate(() => devicePixelRatio)).toBe(2);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
  } finally {
    await zoom.close();
  }
});
test.describe.configure({ timeout: 150000 });

// Create the prerequisite policies through the same authenticated API as the forms.
// Exercise the switches of date, shared and original references, and withdrawal from the menus.
test('210 两份月计划与年计划联验共享薪带、独立津贴及撤去后的原采用', async ({ page }) => {
  await open(page);
  const grade = await seed308(page, 'grade', undefined, { date: '2027-01-01' });
  const rateAPI = '/api/v1/compensation/grade-rates';
  const makeRate = (frequency: string, changes: object) =>
    post(page, rateAPI, {
      action: 'CREATE',
      gradeId: grade.id,
      currency: 'CNY',
      frequency,
      unit: 'PER_' + frequency,
      effectiveDate: '2027-01-01',
      reason: '建立标准',
      changes,
    });
  const shared = await makeRate('MONTH', { form: 'RANGE', minimum: '10000', guidance: '12500', maximum: '15000' });
  const annual = await makeRate('YEAR', { form: 'FIXED', amount: '180000' });
  async function policy(label: string, amount?: string) {
    const frequency = amount ? 'MONTH' : 'YEAR';
    const units = { currency: 'CNY', frequency, unit: 'PER_' + frequency, precision: 2 };
    const row = {
      ...units,
      id: randomUUID(),
      code: 'BASIC',
      name: '基本工资',
      nature: 'BASIC',
      method: 'GRADE_RATE',
      rates: [{ gradeId: grade.id, rateId: (amount ? shared : annual).objectId }],
    };
    const second = amount
      ? { ...units, id: randomUUID(), code: 'ALLOW', name: '岗位津贴', nature: 'ALLOWANCE', method: 'FIXED', amount }
      : {
          ...units,
          id: randomUUID(),
          code: 'BONUS',
          name: '目标奖金',
          nature: 'BONUS',
          method: 'PERCENT',
          baseId: row.id,
          percent: '12.50',
          rounding: 'HALF_UP',
        };
    const code = code210(label);
    const receipt = await post(page, base, {
      action: 'CREATE',
      code,
      reason: '建立计划',
      effectiveDate: '2027-01-01',
      changes: { name: code, scope: { mode: 'ENTERPRISE' }, components: [row, second] },
    });
    return { code, receipt };
  }
  const pro = await policy('PRO', '300'),
    ops = await policy('OPS', '200'),
    mgt = await policy('MGT');
  const table = page.getByRole('table', { name: '薪酬组成', exact: true });
  async function readPlan(code: string, date: string) {
    await menu(page);
    const query = text(page, '搜索计划编码或名称');
    await query.fill(code);
    await query.press('Enter');
    await page.getByRole('link', { name: code, exact: true }).click();
    await text(page, '适用日期').fill(date);
    await button(page, '查看该日').click();
    await expect(table.getByRole('row')).toHaveCount(3);
  }
  await readPlan(pro.code, '2027-06-30');
  await expect(table).toContainText('15,000.00');
  const link = table.getByRole('link', { name: new RegExp(grade.code) });
  await link.click();
  await button(page, '新增记录').click();
  await text(page, '生效日期').fill('2027-07-01');
  await select(page, '办理原因', '薪酬制度调整');
  for (const [label, value] of [
    ['下限', '11000'],
    ['指导值', '13500'],
    ['上限', '16000'],
  ])
    await text(page, label).fill(value);
  await button(page, '保存').click();
  await expect(page.getByText('16,000.00', { exact: true })).toBeVisible();
  for (const plan of [pro, ops]) {
    await readPlan(plan.code, '2027-06-30');
    await expect(table).toContainText('15,000.00');
    await text(page, '适用日期').fill('2027-07-01');
    await button(page, '查看该日').click();
    await expect(table).toContainText('16,000.00');
    await expect(link).toHaveAttribute('href', new RegExp(shared.objectId));
    await table.getByRole('button', { name: '来源说明' }).click();
    await table.getByRole('link', { name: /查看保存时依据/ }).click();
    await expect(page.getByText('15,000.00', { exact: true })).toBeVisible();
    await page.getByRole('link', { name: '返回薪酬计划' }).click();
    await expect(text(page, '适用日期')).toHaveValue('2027-07-01');
  }
  await readPlan(pro.code, '2027-09-30');
  await expect(table).toContainText('300.00 人民币元/月');
  await button(page, '新增记录').click();
  await text(page, '生效日期').fill('2027-10-01');
  await select(page, '办理原因', '薪酬制度调整');
  await button(page, '编辑 岗位津贴').click();
  await text(page, '固定金额').fill('350');
  await dialog(page).getByRole('button', { name: '确定' }).click();
  await save(page);
  await expect(table).toContainText('350.00 人民币元/月');
  await readPlan(ops.code, '2027-10-01');
  await expect(table).toContainText('200.00 人民币元/月');
  await link.click();
  await expect(page.getByText('16,000.00', { exact: true })).toBeVisible();
  await more(page, '删除此记录');
  await button(page, '确认删除').click();
  await expect(page.getByText('15,000.00', { exact: true })).toBeVisible();
  await readPlan(pro.code, '2027-10-01');
  await expect(table).toContainText('15,000.00');
  await expect(table).toContainText('350.00 人民币元/月');
  await table.getByRole('button', { name: '来源说明' }).click();
  await table.getByRole('link', { name: /查看保存时依据/ }).click();
  await expect(page.getByText('16,000.00', { exact: true })).toBeVisible();
  await expect(page.getByText('正在查看只读历史资料', { exact: true })).toBeVisible();
  await readPlan(mgt.code, '2027-10-01');
  await expect(table).toContainText('180,000.00');
  await expect(table).toContainText('参考额 22,500.00');
  await expect(link).toHaveAttribute('href', new RegExp(annual.objectId));
});

test('210 薪酬计划对照本构建的职级阶梯控制区、正文与键盘维护', async ({ page }) => {
  await open(page);
  const sample = await seed308(page, 'gradeladder');
  const code = code210('COMPARE');
  await start(page, code, code);
  await fixed(page, '300');
  await save(page);
  const own = page.url();
  for (const height of [900, 540]) {
    await page.setViewportSize({ width: 1440, height });
    const tops: number[] = [];
    for (const ref of [true, false]) {
      if (ref) await enter308(page, sample);
      else await page.goto(own);
      const body = page.getByRole('region', { name: ref ? '职级阶梯资料' : '薪酬计划资料', exact: true });
      await expect(body.getByRole('heading', { name: ref ? sample.name : code, exact: true })).toBeVisible();
      const control = page.locator('.record-control');
      tops.push(Math.round((await body.boundingBox())!.y));
      expect(await control.boundingBox()).not.toBeNull();
      expect(await page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
      await body.focus();
      await page.keyboard.press('PageDown');
      if (height === 540) await expect.poll(() => page.evaluate(() => window.scrollY)).toBeGreaterThan(0);
      await button(page, '更多操作').focus();
      await page.keyboard.press('Enter');
      await expect(page.getByRole('menuitem', { name: '更正记录', exact: true })).toBeVisible();
      await page.keyboard.press('Escape');
    }
    expect(Math.abs(tops[1] - tops[0])).toBeLessThanOrEqual(3);
    await test.info().attach('相同构建正文顶部-' + height, {
      body: JSON.stringify({ height, gradeLadder: tops[0], plan: tops[1] }),
      contentType: 'application/json',
    });
  }
});

test('210 计划撤去金额权后保留可维护的名称与原津贴', async ({ page }) => {
  await open(page);
  const code = code210('PRIGHTS');
  await start(page, code, code);
  await fixed(page, '300');
  await save(page);
  const id = new URL(page.url()).pathname.split('/').at(-1)!;
  const path = base + '/' + id;
  const before = await (await page.request.get(path)).json();
  await more(page, '更正记录');
  const name = code + ' 核对名称';
  await text(page, '计划名称').fill(name);
  await button(page, '编辑 岗位津贴').click();
  await text(page, '固定金额').fill('380');
  await dialog(page).getByRole('button', { name: '确定' }).click();
  await withPolicy(page, async (where, mutate) => {
    mutate(
      `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'amount') WHERE ${where} AND action IN ('compplan.read','compplan.history.read'); UPDATE access.subject_grant SET write_fields=array_remove(write_fields,'amount') WHERE ${where} AND action='compplan.correct'`,
    );
    const denied = page.waitForResponse((r) => new URL(r.url()).pathname === path && r.request().method() === 'POST');
    await button(page, '保存').click();
    expect((await denied).status()).toBe(403);
    await expect(text(page, '计划名称')).toHaveValue(name);
    await button(page, '编辑 岗位津贴').click();
    await expect(text(page, '固定金额')).toHaveCount(0);
    await dialog(page).getByRole('button', { name: '取消' }).click();
    const limited = await (await page.request.get(path)).json();
    expect(limited.sourceVersion).toBe(before.sourceVersion);
    expect(limited.record.components[0].amount).toBeUndefined();
    await save(page);
    await expect(page.getByRole('heading', { level: 1, name, exact: true })).toBeVisible();
  });
  const latest = await (await page.request.get(path)).json();
  expect(latest.record.name).toBe(name);
  expect(latest.record.components[0].amount).toBe('300.00');
});
