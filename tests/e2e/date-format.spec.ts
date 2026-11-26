import { desktopViewports } from './desktop-viewports.mjs';
import { expectLogChange } from './fixtures';
import {test, expect, openMoreActions} from './fixtures';
import { cancelLeapDayPreview } from './dated-object-support';
import { withPolicy } from './grants';
import type {Locator} from '@playwright/test';

async function expectValid(input: Locator, valid = true) {
  await expect.poll(() => input.evaluate(element => (element as HTMLInputElement).validity.valid)).toBe(valid);
}
import {open, create, add, select, read, enter, readyToSave, save} from './bu-support';

test('详情适用期间显示最后适用日，开放期间及历史来源保持准确', async ({page}) => {
  await open(page);
  await create(page, '2026-03-13');
  const period = page.locator('.object-detail-field').filter({has: page.getByText('适用期间', {exact:true})});
  const status = page.locator('.object-detail-field').filter({has: page.getByText('状态', {exact:true})});
  await expect(period).toContainText('自 2026-03-13 起');
  await expect(status).toContainText('启用');
  await add(page, '2026-03-24', '乙');
  await expect(period).toContainText('自 2026-03-24 起');
  await select(page, '2026-03-13');
  await expect(period).toContainText('2026-03-13 至 2026-03-23');
  await expect(period).not.toContainText('不含当日');
  const detail = await read(page);
  const first = detail.timeline.find((r: {effectiveDate:string}) => r.effectiveDate === '2026-03-13');
  expect(first.endExclusive).toBe('2026-03-24');
  expect(first.endInclusive).toBe('2026-03-23');
  await page.reload();
  await expect(period).toContainText('2026-03-13 至 2026-03-23');
  await page.getByRole('tab', {name:'变更日志',exact:true}).click();
  await page.getByRole('link', {name:'查看当次资料',exact:true}).first().click();
  await select(page, '2026-03-13');
  await expect(period).toContainText('2026-03-13 至 2026-03-23');
  await expect(page.getByText('记录生效后状态', {exact:true})).toHaveCount(0);
  await expect(status).toContainText('启用');
  await expect(page.getByText('只读', {exact:false}).first()).toBeVisible();
});

for (const locale of ['zh-CN', 'en-US']) {
  test(`日期组件在${locale}保持ISO输入、选日、有效性及只读状态`, async ({browser, baseURL}) => {
    const context = await browser.newContext({locale});
    try {
      const page = await context.newPage();
      await page.goto(new URL('/test-fixtures/experience.html?date-fields=1', baseURL).href);
      const input = page.getByRole('textbox', {name:'测试日期', exact:true});
      await expect(input).toHaveAttribute('type', 'text');
      await expect(input).toHaveValue('2026-01-13');
      await expect(input).toHaveAttribute('placeholder', 'YYYY-MM-DD');
      await input.fill('2026-01-');
      await expectValid(input, false);
      await input.fill('2026-02-30');
      await expectValid(input, false);
      await expect(page.getByText('请输入 YYYY-MM-DD 格式的有效日期', {exact:true})).toBeVisible();
      await input.clear();
      await expectValid(input, false);
      for (const day of ['0001-01-01','9999-12-31','2028-02-29']) {
        await input.fill(day);
        await expect(input).toHaveValue(day);
        await expectValid(input);
      }
      const bounded = page.getByRole('textbox', {name:'范围日期', exact:true});
      await bounded.fill('2026-01-09');
      await expectValid(bounded, false);
      await expect(page.getByText('日期不得早于 2026-01-10', {exact:true})).toBeVisible();
      await bounded.fill('2026-01-21');
      await expectValid(bounded, false);
      await bounded.fill('2026-01-20');
      await expectValid(bounded);
      await page.getByRole('button', {name:'选择日期',exact:true}).first().click();
      await page.keyboard.press('Escape');
      await page.locator('input[type="date"]').first().fill('2026-01-13', {force:true});
      await expect(input).toHaveValue('2026-01-13');
      await expect(input).toBeFocused();
      await expect(page.getByRole('textbox', {name:'只读日期',exact:true})).not.toBeEditable();
      await expect(page.getByRole('textbox', {name:'禁用日期',exact:true})).toBeDisabled();
      await expect(page.getByRole('button', {name:'选择日期',exact:true}).nth(2)).toBeDisabled();
      await expect(page.getByText('2026-01-13 00:30:05（北京时间 UTC+08:00）', {exact:true})).toBeVisible();
      await page.evaluate(() => document.fonts.ready);
      for (const [width, height] of desktopViewports) {
        await page.setViewportSize({width, height});
        await expect(input).toHaveValue('2026-01-13');
        await expect.poll(() => input.evaluate(element => element.scrollWidth <= element.clientWidth)).toBe(true);
        await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
        await page.screenshot({path:`.local/date-format-20261103/component-${locale}-${width}x${height}.png`});
      }
    } finally { await context.close(); }
  });
}

test('正常菜单创建、更正日期、历史来源使用ISO格式', async ({page}) => {
  await open(page);
  await create(page, '2026-01-13');
  const original = await read(page);
  await expect(page.getByRole('button', {name:'2026-01-13',exact:true})).toBeVisible();
  await openMoreActions(page);
  await expect(page.getByRole('menuitem',{name:'查看日期…',exact:true})).toHaveCount(0);
  await page.keyboard.press('Escape');
  await enter(page, '更正记录');
  await page.getByRole('textbox', {name:'生效日期',exact:true}).fill('2026-01-14');
  await readyToSave(page);
  await save(page);
  await expect(page.getByRole('button', {name:'2026-01-14',exact:true})).toBeVisible();
  await page.getByRole('tab', {name:'变更日志',exact:true}).click();
  await expectLogChange(page, '生效日期', '2026-01-13', '2026-01-14');
  await page.getByRole('link', {name:'查看当次资料',exact:true}).last().click();
  await expect(page.getByRole('button', {name:'2026-01-13',exact:true})).toBeVisible();
  await expect(page.locator('.operation-summary').getByText(/^操作时间：\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}（北京时间 UTC\+08:00）$/)).toBeVisible();
  const historical = await read(page, '2026-01-13', original.headSourceVersion);
  expect(historical.timeline[0].effectiveDate).toBe('2026-01-13');
});

test('140 BU 首条后移跨闰日，受限查询者看不到期间', async ({page}) => {
  test.setTimeout(90000);
  await open(page);
  const code = await create(page, '2024-02-28');
  const before = await read(page);
  await page.setViewportSize({width:1440,height:540});
  await openMoreActions(page);
  await cancelLeapDayPreview(page);
  expect((await read(page)).headSourceVersion).toBe(before.headSourceVersion);
  await page.setViewportSize({width:1440,height:900});
  await add(page,'2024-03-01','三月');
  await select(page,'2024-02-28');
  await expect(page.getByText('2024-02-28 至 2024-02-29',{exact:true})).toBeVisible();
  const url = page.url();
  await page.getByRole('button',{name:'退出',exact:true}).click();
  await open(page,'reader');
  await withPolicy(page, async (where,mutate) => {
    mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'effectiveDate') WHERE ${where} AND action LIKE 'bu.%'`);
    await page.goto(url);
    await expect(page.getByRole('heading',{level:1})).toContainText(code);
    await expect(page.getByText('适用期间',{exact:true})).toHaveCount(0);
    await expect(page.getByText(/2024-02-29/)).toHaveCount(0);
  });
});
