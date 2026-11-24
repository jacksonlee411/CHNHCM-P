import { test, expect, type Page, type Locator } from './fixtures';
import { randomUUID } from 'node:crypto';
import { request } from './dated-object-support';
import { choose } from './bu-support';
import {
  action308,
  evidence308,
  reference308,
  base308,
  click308,
  contrast308,
  enter308,
  family308,
  chooseClassification308,
  menu308,
  reach308,
  seed308,
} from './staffing308-support';
async function fill(page: Page, target: Locator, value: string) {
  await reach308(page, target);
  await target.fill(value);
}
export async function matrixJourney308(
  page: Page,
  o: Awaited<ReturnType<typeof seed308>>,
  family: Awaited<ReturnType<typeof seed308>>,
  role: string,
) {
  const kind = o.kind,
    path = base308[kind] + '/' + o.id;
  await reference308(page, role);
  await menu308(page, kind);
  await page.getByRole('textbox', { name: '搜索编码或名称', exact: true }).fill(o.code);
  const applied = page.waitForResponse(
    (r) =>
      r.request().method() === 'GET' &&
      new URL(r.url()).pathname === base308[kind] &&
      new URL(r.url()).searchParams.get('q') === o.code,
  );
  await click308(page, page.getByRole('button', { name: '查询', exact: true }));
  await (await applied).finished();
  await expect(page.getByText('1–20 / 21', { exact: true })).toBeVisible();
  await expect(page.locator('.standard-page[aria-busy]')).toHaveAttribute('aria-busy', 'false');
  await expect(page.getByRole('button', { name: '下一页', exact: true })).toBeEnabled();
  await click308(page, page.getByRole('button', { name: '下一页', exact: true }));
  await expect(page.getByText('21–21 / 21', { exact: true })).toBeVisible();
  await expect(page.locator('.standard-page[aria-busy]')).toHaveAttribute('aria-busy', 'false');
  await click308(page, page.getByRole('button', { name: '上一页', exact: true }));
  await expect(page.getByText('1–20 / 21', { exact: true })).toBeVisible();
  await expect(page.locator('.standard-page[aria-busy]')).toHaveAttribute('aria-busy', 'false');
  await click308(page, page.getByRole('button', { name: '更多操作', exact: true }));
  await click308(page, page.getByRole('menuitem', { name: '列筛选…', exact: true }));
  await choose(page, '列', '编码');
  await choose(page, '条件', '等于');
  await page.getByRole('textbox', { name: '值', exact: true }).fill(o.code);
  await click308(page, page.getByRole('button', { name: '应用', exact: true }));
  await expect(page.getByText('1–1 / 1', { exact: true })).toBeVisible();
  await click308(page, page.getByRole('link', { name: o.name, exact: true }));
  await expect(page.getByRole('heading', { name: o.name, level: 1, exact: true })).toBeVisible();
  await contrast308(page);
  await expect(page.getByRole('heading', { level: 1 })).toHaveCount(1);
  await expect(page.getByRole('combobox', { name: '维护操作' })).toHaveCount(0);
  await expect(page.getByRole('combobox', { name: '生效记录' })).toHaveCount(0);
  await expect
    .poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1))
    .toBe(true);
  const sheet = await page.locator('.object-detail-sheet').boundingBox();
  expect(sheet!.width).toBeLessThanOrEqual(1260);
  await reach308(page, page.getByRole('navigation', { name: '生效日期', exact: true }));
  await click308(page, page.getByRole('tab', { name: '变更日志', exact: true }));
  await expect(page.locator('.change-operation')).toHaveCount(1);
  await click308(page, page.getByRole('button', { name: '日志更多操作', exact: true }));
  await click308(page, page.getByRole('menuitem', { name: '列筛选…', exact: true }));
  await click308(page, page.getByRole('button', { name: '取消', exact: true }));
  await click308(page, page.getByRole('link', { name: '查看当次资料', exact: true }));
  await reach308(page, page.getByText('正在查看只读历史资料', { exact: true }));
  await click308(page, page.getByRole('link', { name: '返回变更日志', exact: true }));
  await expect(page).toHaveURL((url) => !url.searchParams.has('sourceVersion') && url.searchParams.get('tab') === 'history');
  await expect(page.getByRole('tab', { name: '变更日志', exact: true })).toHaveAttribute('aria-selected', 'true');
  await click308(page, page.getByRole('tab', { name: '基本资料', exact: true }));
  await expect(page.getByRole('tab', { name: '基本资料', exact: true })).toHaveAttribute('aria-selected', 'true');
  if (role === 'reader') {
    await expect(page.getByRole('button', { name: '更多操作', exact: true })).toHaveCount(0);
    await evidence308(page, 'normal-journey');
    return;
  }
  const before = (await request(page, 'GET', path)).body.headSourceVersion;
  await action308(page, '更正记录');
  await contrast308(page);
  await expect(page.locator('.object-form')).toHaveCount(1);
  await expect(
    page.locator('.record-control').getByRole('button', { name: '保存', exact: true }),
  ).toHaveCount(1);
  const name = page.getByRole('textbox', { name: '名称', exact: true }),
    last = page.getByRole('textbox', { name: kind === 'job' ? '任职要求' : '说明', exact: true });
  const style = await name.evaluate((el) => ({
    label: getComputedStyle(document.querySelector('#staffing-name-label')!).fontSize,
    value: getComputedStyle(el).fontSize,
    height: el.closest('.MuiInputBase-root')!.getBoundingClientRect().height,
  }));
  expect(style.label).toBe('13px');
  expect(style.value).toBe('14px');
  expect(style.height).toBe(32);
  const sections = await page
    .locator('.object-form .object-detail-columns > section')
    .evaluateAll((nodes) =>
      nodes.map((n) => ({ left: n.getBoundingClientRect().left, top: n.getBoundingClientRect().top })),
    );
  const width = await page.evaluate(() => innerWidth);
  if (width >= 900) expect(sections[1].left).toBeGreaterThan(sections[0].left);
  else expect(sections[1].top).toBeGreaterThan(sections[0].top);
  const long = '长内容与换行\n'.repeat(300).slice(0, kind === 'job' ? 2000 : 1000);
  await fill(page, last, long);
  await page.keyboard.press('Tab');
  await expect
    .poll(() =>
      page.evaluate(() => {
        const el = document.activeElement;
        if (!el || el === document.body) return false;
        const b = el.getBoundingClientRect();
        return b.top >= 0 && b.top < innerHeight;
      }),
    )
    .toBe(true);
  await page.keyboard.press('Shift+Tab');
  await expect(last).toBeFocused();
  await reach308(page, last);
  await fill(page, name, '长'.repeat(201));
  await click308(page, page.getByRole('button', { name: '保存', exact: true }));
  await expect(page.locator('#staffing-error')).toBeVisible();
  await expect(name).toBeFocused();
  const errorStyle = await page
    .locator('#staffing-name-helper-text')
    .evaluate((el) => getComputedStyle(el).fontSize);
  expect(errorStyle).toBe('12px');
  await test
    .info()
    .attach('candidate-form', {
      body: JSON.stringify({ style, sections, error: errorStyle }),
      contentType: 'application/json',
    });
  await expect(last).toHaveValue(long);
  await fill(page, name, o.name);
  await click308(page, page.getByRole('button', { name: '取消', exact: true }));
  await expect(page.getByRole('dialog', { name: '尚有未保存修改', exact: true })).toBeVisible();
  await click308(page, page.getByRole('button', { name: '继续编辑', exact: true }));
  await expect(last).toHaveValue(long);
  await click308(page, page.getByRole('button', { name: '取消', exact: true }));
  await click308(page, page.getByRole('button', { name: '放弃修改', exact: true }));
  await expect(page.locator('.object-form')).toHaveCount(0);
  expect((await request(page, 'GET', path)).body.headSourceVersion).toBe(before);
  await action308(page, '更正记录');
  await click308(page, page.getByRole('button', { name: '取消', exact: true }));
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await expect(page.locator('.object-form')).toHaveCount(0);
  await action308(page, '插入停用与重新启用');
  await fill(
    page,
    page.getByRole('textbox', { name: kind === 'job' ? '任职要求' : '说明', exact: true }),
    '恢复资料\n'.repeat(70),
  );
  await reach308(page, page.getByLabel('重新启用日期').and(page.locator('input')));
  await click308(page, page.getByRole('button', { name: '取消', exact: true }));
  await click308(page, page.getByRole('button', { name: '放弃修改', exact: true }));
  await expect(page.getByRole('button', { name: '更多操作', exact: true })).toBeFocused();
  await menu308(page, kind);
  await click308(page, page.getByRole('link', { name: '新建', exact: true }));
  await expect(page.getByLabel('生效日期').and(page.locator('input'))).toHaveValue('1900-01-01');
  const code = 'M308-' + randomUUID().slice(0, 8);
  await fill(page, page.getByRole('textbox', { name: '编码', exact: true }), code);
  await fill(page, page.getByRole('textbox', { name: '名称', exact: true }), '新建 ' + code);
  await fill(page, page.getByLabel('生效日期').and(page.locator('input')), '2026-01-01');
  if (kind === 'job') {
    await family308(page, family.name);
    await chooseClassification308(page, 'jobFunctionId', o.jobFunction!.name);
  }
  if (kind === 'gradeladder') {
    await click308(page, page.getByRole('button', { name: '选择职级', exact: true }));
    await page.getByRole('textbox', { name: '查找职级', exact: true }).fill(o.gradeNames![0]);
    await click308(page, page.getByRole('dialog', { name: '选择职级', exact: true }).getByRole('checkbox').first());
    await click308(page, page.getByRole('button', { name: '完成选择', exact: true }));
  }
  await click308(page, page.getByRole('button', { name: '保存', exact: true }));
  await expect(page.locator('.object-form')).toHaveCount(0);
  await expect(page.getByRole('heading', { name: '新建 ' + code, level: 1, exact: true })).toBeVisible();
  await expect
    .poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1))
    .toBe(true);
  await evidence308(page, 'normal-journey');
}
