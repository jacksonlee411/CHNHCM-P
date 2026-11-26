import { desktopViewports } from './desktop-viewports.mjs';
import { randomUUID } from 'node:crypto';
import { test, expect, type Page } from './fixtures';
import { open } from './bu-support';
import { click308, date308, namedMenu308, seed308 } from './staffing308-support';
import { seedPosition } from './staffing312-support';
import { request } from './dated-object-support';
import { staffingZoomCases } from './staffing-zoom-suite';
import { withPolicy } from './grants';

const jan = '2027-01-01',
  may = '2027-05-01',
  sep = '2027-09-01';
async function seed(page: Page, _kind: string, family: string) {
  const a = await seedPosition(page, 'position', family),
    pos = await request(page, 'GET', '/api/v1/positions/' + a.id);
  const { departmentId, jobId, locationId } = pos.body.selectedRecord;
  const name = '测试岗' + a.code;
  const qa = await request(
    page,
    'POST',
    '/api/v1/positions',
    {
      departmentId,
      jobId,
      locationId,
      name,
      code: 'T' + a.code,
      effectiveDate: '2026-01-01',
      operationType: 'CREATE',
      operationReason: 'NEW_POSITION',
    },
    randomUUID(),
  );
  expect(qa.status).toBe(201);
  const saved = await request(
    page,
    'POST',
    '/api/v1/staffing-quotas/departments/' + departmentId + '/allocations',
    {
      effectiveDate: jan,
      reason: '年度额度',
      total: { quota: 10 },
      positions: [
        { positionId: a.id, quota: 6 },
        { positionId: qa.body.objectId, quota: 3 },
      ],
    },
    randomUUID(),
  );
  expect(saved.status, saved.body.message).toBe(200);
  const swap = await request(page, 'POST', '/api/v1/staffing-quotas/departments/' + departmentId + '/allocations', {
    effectiveDate: sep, reason: '岗位等额调剂',
    positions: saved.body.quotaChanges
      .filter((r: { targetKind: string }) => r.targetKind === 'position')
      .map((r: { targetId: string; sourceVersion: string }) => ({
        positionId: r.targetId, expectedSourceVersion: r.sourceVersion, quota: r.targetId === a.id ? 5 : 4,
      })),
  }, randomUUID());
  expect(swap.status, swap.body.message).toBe(200);
  return { a, qa: name };
}
const value = (page: Page, name: string) =>
  page
    .locator('.object-detail-field')
    .filter({ has: page.locator('label').filter({ hasText: new RegExp('^' + name + '$') }) })
    .locator('.object-field-value');
async function department(page: Page, name: string) {
  await namedMenu308(page, '部门编制分配', '职务配置', '/settings/staffing/quotas', '部门编制统计');
  const date = page.getByLabel('统计日期').and(page.locator('input'));
  await click308(page, date);
  await date.fill(jan);
  await expect(date).toHaveValue(jan);
  await click308(page, page.getByRole('button', { name: '查询统计', exact: true }));
  await expect.poll(() => new URL(page.url()).searchParams.get('businessAt')).toBe(jan);
  await expect(page.locator('.standard-page[aria-busy]')).toHaveAttribute('aria-busy', 'false');
  const filter = page.getByRole('combobox', { name: '所属部门' });
  await click308(page, filter);
  const option = page.getByRole('option', { name, exact: true });
  const expected = await option.getAttribute('data-value');
  await click308(page, option);
  await expect.poll(() => new URL(page.url()).searchParams.get('departmentId')).toBe(expected);
  await expect(filter).toHaveText(name);
  await expect(page.getByText(/部门总额 10 · 直属岗位核定合计 9 · 待分配 1/)).toBeVisible();
  await click308(page, page.getByRole('link', { name: '部门编制', exact: true }));
  await expect(value(page, '待分配')).toHaveText('1');
}
async function journey(page: Page, sample: Awaited<ReturnType<typeof seed>>, _family: unknown, role: string) {
  const click = (name: string) => click308(page, page.getByRole('button', { name, exact: true }));
  await department(page, sample.a.department);
  const axis = page.getByRole('navigation', { name: '生效日期', exact: true });
  const dateControl = () => axis.locator('[aria-current="date"]').evaluate((el) => ({
    mode: el.closest('[data-mode]')?.getAttribute('data-mode'),
    font: getComputedStyle(el).fontSize,
    height: Math.round(el.getBoundingClientRect().height),
  }));
  await date308(page, sep);
  const allocationControl = await dateControl();
  await expect(value(page, '直属岗位核定合计')).toHaveText('9');
  await expect(value(page, sample.a.name)).toHaveText('5');
  await expect(value(page, sample.qa)).toHaveText('4');
  await click308(page, page.getByRole('link', { name: sample.a.name, exact: true }));
  await expect(value(page, '核定编制')).toHaveText('5');
  await expect(axis.locator('[aria-current="date"]')).toContainText(sep);
  expect(await dateControl()).toEqual(allocationControl);
  await click308(page, page.getByRole('link', { name: '返回部门编制', exact: true }));
  await expect(axis.locator('[aria-current="date"]')).toContainText(sep);
  await click('查看指定日');
  const queryDay = page.getByLabel('统计日期').and(page.locator('input'));
  await click308(page, queryDay);
  await queryDay.fill('2027-04-30');
  await queryDay.press('Enter');
  await expect(page.locator('time[datetime="2027-04-30"]')).toBeVisible();
  await expect(value(page, sample.a.name)).toHaveText('6');
  await expect(axis.locator('[aria-current="date"]')).toHaveCount(0);
  await date308(page, jan);
  if (role === 'reader') {
    await expect(page.getByRole('button', { name: '调整编制' })).toHaveCount(0);
    await click308(page, page.getByRole('tab', { name: '变更日志', exact: true }));
    await click308(page, page.getByRole('link', { name: '查看当次资料' }).first());
    await expect(value(page, '部门总额')).toHaveText('10');
    await click('返回当前资料');
  } else {
    await click('调整编制');
    const dev = page.getByRole('spinbutton', { name: sample.a.name, exact: true }),
      qa = page.getByRole('spinbutton', { name: sample.qa, exact: true });
    await expect(dev).toHaveValue('6');
    await qa.fill('5');
    await page.getByLabel('调整原因').and(page.locator('input')).fill('同日调剂');
    await click('保存');
    await expect(page.getByRole('alert').filter({ hasText: '超出部门总额 1 编' })).toBeVisible();
    await dev.fill('5');
    const editDay = page.getByLabel('生效日期').and(page.locator('input'));
    await editDay.fill(sep);
    await click('继续编辑');
    await expect(editDay).toHaveValue(jan);
    await expect(qa).toHaveValue('5');
    await editDay.fill(sep);
    await click('放弃修改');
    await expect(qa).toHaveValue('4');
    await click('取消');
    await click('放弃修改');
    await expect(value(page, '待分配')).toHaveText('1');
    await click('调整编制');
    await expect(dev).toHaveValue('6');
    await expect(qa).toHaveValue('3');
    await page.getByLabel('生效日期').and(page.locator('input')).fill(may);
    await expect(dev).toHaveValue('6');
    await dev.fill('5');
    await qa.fill('5');
    await page.getByLabel('调整原因').and(page.locator('input')).fill('五月调剂');
    await click('保存');
    await expect(value(page, '待分配')).toHaveText('0');
    await expect(value(page, sample.qa)).toHaveText('5');
    await date308(page, jan);
    await expect(value(page, '待分配')).toHaveText('1');
  }
  await click308(page, page.getByRole('link', { name: '返回列表', exact: true }));
  await expect(page.getByText(/直属岗位核定合计 9 · 待分配 1/)).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
}
for (const [width, height] of desktopViewports) {
  test('315 部门同日调剂及只读复核 ' + width + 'x' + height, async ({ page }) => {
    test.setTimeout(180000);
    await open(page);
    const family = await seed308(page, 'jobfamily'),
      sample = await seed(page, 'department', family.id);
    await page.setViewportSize({ width, height });
    await journey(page, sample, family, 'admin');
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.context().clearCookies();
    await open(page, 'reader');
    await page.setViewportSize({ width, height });
    await journey(page, sample, family, 'reader');
    await click308(page, page.getByRole('link', { name: '部门编制', exact: true }));
    await date308(page, may);
    await expect(value(page, '待分配')).toHaveText('0');
    await expect(value(page, sample.a.name)).toHaveText('5');
  });
}
staffingZoomCases(['department'] as const, '315', journey, seed);

test('315 空部门定额与原件、未来冲突及获权小计', async ({ page }) => {
  test.setTimeout(240000);
  await open(page);
  const family = await seed308(page, 'jobfamily'),
    sample = await seed(page, 'department', family.id);
  const suffix = randomUUID().slice(0, 8),
    code = 'E' + suffix,
    name = '客服' + suffix;
  const bu = await request(
    page,
    'POST',
    '/api/v1/business-units',
    { code, name, effectiveDate: '2026-01-01', operationType: 'CREATE', operationReason: 'NEW_ORGANIZATION' },
    randomUUID(),
  );
  expect(bu.status).toBe(201);
  const dep = await request(
    page,
    'POST',
    '/api/v1/departments',
    {
      code,
      name,
      effectiveDate: '2026-01-01',
      managementBuId: bu.body.buId,
      operationType: 'CREATE',
      operationReason: 'NEW_ORGANIZATION',
    },
    randomUUID(),
  );
  expect(dep.status).toBe(201);
  const click = (name: string) => click308(page, page.getByRole('button', { name, exact: true }));
  const date = (label: string) => page.getByLabel(label).and(page.locator('input'));
  await namedMenu308(page, '部门编制分配', '职务配置', '/settings/staffing/quotas', '部门编制统计');
  const initialStatisticsDay = await date('统计日期').inputValue();
  await click308(page, page.getByRole('combobox', { name: '所属部门' }));
  await click308(page, page.getByRole('option', { name: '查找其他部门…', exact: true }));
  const lookup = page.getByRole('dialog', { name: '查找部门' });
  await lookup.getByLabel('搜索部门编码或名称').fill(code);
  await click308(page, lookup.getByRole('button', { name: name + '（' + code + '）', exact: true }));
  await click308(page, page.getByRole('link', { name: '部门编制', exact: true }));
  await expect(value(page, '部门总额')).toHaveText('未核定');
  expect(new URL(page.url()).searchParams.get('businessAt')).toBe(initialStatisticsDay);
  await click('调整编制');
  await expect(date('生效日期')).toHaveValue('1900-01-01');
  await date('生效日期').fill(jan);
  const amount = page.getByRole('spinbutton', { name: '部门总额', exact: true });
  await amount.fill('20');
  await date('调整原因').fill('客服独立定额');
  await page.route(
    '**/departments/*/allocations',
    async (route) => {
      const result = await route.fetch();
      expect(result.status()).toBe(200);
      await route.abort('failed');
    },
    { times: 1 },
  );
  await page.route('**/staffing-quotas/submissions/*/resolve', (r) => r.abort('failed'), { times: 1 });
  await click('保存');
  await click('查看保存结果');
  await expect(value(page, '待分配')).toHaveText('20');
  await expect(value(page, '操作时间').locator('time')).toHaveAttribute('datetime', /.+/);
  await click('更多操作');
  await click308(page, page.getByRole('menuitem', { name: '更正部门总额', exact: true }));
  await expect(page.getByRole('spinbutton', { name: /部门总额/ })).toHaveValue('20');
  await click('保存');
  await expect(page.getByText('没有修改内容。', { exact: true })).toBeVisible();
  await click('更多操作');
  await click308(page, page.getByRole('menuitem', { name: '更正部门总额', exact: true }));
  await page.getByRole('spinbutton', { name: /部门总额/ }).fill('0');
  await click('保存');
  await expect(value(page, '待分配')).toHaveText('0');
  await click308(page, page.getByRole('tab', { name: '变更日志', exact: true }));
  const history = page.getByRole('tabpanel', { name: '变更日志', exact: true });
  const changedAmount = history.locator('[data-label="变更字段"]').filter({ hasText: /^部门总额$/ }).first();
  await expect(changedAmount).toBeVisible();
  const logRequest = '**/staffing-quotas/history/*/changes?*';
  await page.route(logRequest, async (route) => {
    const response = await route.fetch();
    expect(response.ok()).toBe(true);
    await route.fulfill({ response, json: { ...(await response.json()), items: [], total: 0 } });
  });
  await click('日志更多操作');
  await click308(page, page.getByRole('menuitem', { name: '刷新日志', exact: true }));
  await expect(history.getByText('没有可读取的变更日志', { exact: true })).toBeVisible();
  await page.unroute(logRequest);
  await click('日志更多操作');
  await click308(page, page.getByRole('menuitem', { name: '刷新日志', exact: true }));
  await expect(changedAmount).toBeVisible();
  await click('日志更多操作');
  await click308(page, page.getByRole('menuitem', { name: '列筛选…', exact: true }));
  const filterDialog = page.getByRole('dialog', { name: '列筛选' });
  await click308(page, filterDialog.getByRole('combobox', { name: '列', exact: true }));
  await expect(page.getByRole('option', { name: '生效记录', exact: true })).toBeVisible();
  await click308(page, page.getByRole('option', { name: '记录处理', exact: true }));
  await click308(page, filterDialog.getByRole('combobox', { name: '值', exact: true }));
  await click308(page, page.getByRole('option', { name: '删除记录', exact: true }));
  await click('应用');
  await expect(history.getByText('记录处理 等于 删除记录', { exact: true })).toBeVisible();
  await expect(history.getByText('没有符合条件的变更日志', { exact: true })).toBeVisible();
  await click('日志更多操作');
  await click308(page, page.getByRole('menuitem', { name: '列筛选…', exact: true }));
  await click('清除条件');
  await click('应用');
  await click308(page, page.getByRole('link', { name: '查看当次资料' }).last());
  await expect(value(page, '部门总额')).toHaveText('20');
  await expect(page.getByRole('navigation', { name: '生效日期' })).toHaveCount(0);
  await click('返回当前资料');
  await expect(value(page, '部门总额')).toHaveText('0');
  await click('调整编制');
  await date('生效日期').fill(may);
  await amount.fill('3');
  await date('调整原因').fill('五月补三编');
  await click('保存');
  await expect(value(page, '部门总额')).toHaveText('3');
  await click('更多操作');
  await click308(page, page.getByRole('menuitem', { name: '更改总额生效日' }));
  await date('生效日期').fill('2027-06-01');
  await click('保存');
  const axis = page.getByRole('navigation', { name: '生效日期', exact: true });
  await expect(axis.locator('[aria-current="date"]')).toContainText('2027-06-01');
  await expect(axis.getByRole('button', { name: may, exact: true })).toHaveCount(0);
  await click('更多操作');
  await click308(page, page.getByRole('menuitem', { name: '删除总额记录' }));
  await expect(value(page, '删除后部门总额')).toHaveText('0');
  await click('确认删除');
  await expect(page.getByText('已删除。', { exact: true })).toBeVisible();
  await expect(page.getByText('正在查看只读历史资料')).toBeVisible();
  await expect(value(page, '部门总额')).toHaveText('3');
  await click('返回当前资料');
  await expect(page.getByText('当前资料中没有所选记录，请重新选择。', { exact: true })).toBeVisible();
  await click('查看默认记录');
  await expect(value(page, '部门总额')).toHaveText('0');
  await expect(axis.getByRole('button', { name: '2027-06-01', exact: true })).toHaveCount(0);
  const original = await request(
    page,
    'GET',
    '/api/v1/staffing-quotas/statistics?businessAt=' + jan + '&q=T' + sample.a.code,
  );
  const qa = original.body.items.find((r: { name: string }) => r.name === sample.qa);
  const source = (
    await request(
      page,
      'GET',
      '/api/v1/staffing-quotas/statistics/positions/' + qa.positionId + '?businessAt=' + jan,
    )
  ).body;
  expect(
    (
      await request(
        page,
        'POST',
        '/api/v1/staffing-quotas/' + source.quotaId + '/adjust',
        {
          quota: 4,
          effectiveDate: '2027-06-01',
          reason: '六月岗位独立增编',
          expectedSourceVersion: source.sourceVersion,
        },
        randomUUID(),
      )
    ).status,
  ).toBe(200);
  await department(page, sample.a.department);
  await click('调整编制');
  await date('生效日期').fill(may);
  await amount.fill('8');
  await page.getByRole('spinbutton', { name: sample.a.name, exact: true }).fill('5');
  await date('调整原因').fill('预约减编');
  await click('保存');
  await expect(page.getByRole('alert').filter({ hasText: '2027-06-01' })).toContainText('1 编');
  await click('取消');
  await click('放弃修改');
  await expect(value(page, '待分配')).toHaveText('1');
  await page.context().clearCookies();
  await open(page, 'reader');
  await withPolicy(page, async (where, mutate) => {
    const condition = where + " AND action LIKE 'staffingquota.%'";
    try {
      expect(sample.a.id).toMatch(/^[0-9a-f-]{36}$/);
      mutate(
        "UPDATE access.subject_grant SET scope_kind='position',scope_id='" +
          sample.a.id +
          "' WHERE " +
          condition,
      );
      await namedMenu308(page, '部门编制分配', '职务配置', '/settings/staffing/quotas', '部门编制统计');
      await date('统计日期').fill(jan);
      await click('查询统计');
      await expect(page.getByRole('columnheader', { name: '可见范围 · 岗位核定小计', exact: true })).toBeVisible();
      await click308(page, page.getByRole('combobox', { name: '所属部门' }));
      await click308(page, page.getByRole('option', { name: sample.a.department, exact: true }));
      await expect(page.getByText(/可见范围 · 岗位核定小计 6/)).toBeVisible();
      await click308(page, page.getByRole('link', { name: '部门编制', exact: true }));
      await expect(value(page, '可见范围 · 岗位核定小计')).toHaveText('6');
      await expect(page.getByText('部门总额', { exact: true })).toHaveCount(0);
      await expect(page.getByText('待分配', { exact: true })).toHaveCount(0);
      await expect(page.getByText(sample.qa, { exact: true })).toHaveCount(0);
      await expect(axis.getByRole('button', { name: '2027-06-01', exact: true })).toHaveCount(0);
      await date308(page, sep);
      await expect(value(page, '可见范围 · 岗位核定小计')).toHaveText('5');
    } finally {
      mutate('DELETE FROM access.subject_grant WHERE ' + condition);
    }
  });
});
