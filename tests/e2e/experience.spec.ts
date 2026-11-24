import { test, expect } from './fixtures';

test('日期轴多节点、隐藏日期菜单、键盘和窄屏选择保持同一展示记录', async ({ page }) => {
  await page.goto('/test-fixtures/experience.html');
  const axis = page.getByRole('navigation', { name: '生效日期', exact: true });
  await expect(axis).toBeVisible();
  await page.evaluate(() => document.fonts.ready);
  await expect(axis.locator('[aria-current="date"]')).toContainText('2026-06-01');
  await axis.getByRole('button', { name: '较早的生效日期', exact: true }).click();
  await page.getByRole('menuitem', { name: '2026-01-01', exact: true }).click();
  await expect(page.getByRole('status')).toHaveText('选中记录：2026-01-01');
  await axis.locator('[aria-current="date"]').focus();
  await page.keyboard.press('ArrowRight');
  await page.keyboard.press('Enter');
  await expect(page.getByRole('status')).toHaveText('选中记录：2026-02-01');
  await page.setViewportSize({ width: 390, height: 480 });
  await axis.getByRole('button', { name: /选择生效日期/ }).click();
  await page.getByRole('menuitem', { name: '2026-12-01', exact: true }).click();
  await expect(page.getByRole('status')).toHaveText('选中记录：2026-12-01');
  await expect(axis.getByRole('button', { name: /选择生效日期/ })).toBeFocused();
  await page.getByRole('button', { name: '核对输入' }).focus();
  await expect(page.getByRole('button', { name: '核对输入' })).toBeInViewport();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  await page.getByRole('button', { name: '单记录', exact: true }).click();
  await axis.getByRole('button', { name: /选择生效日期/ }).click();
  await expect(page.getByRole('menuitem')).toHaveCount(1);
  await page.keyboard.press('Escape');
  await page.getByRole('button', { name: '空集合', exact: true }).click();
  await expect(axis.getByRole('button')).toHaveCount(0);
  await expect(page.getByRole('status')).toHaveText('选中记录：');

});

test('跨页签错误定位展开隐藏分组，来回切换保留输入且短屏可滚动', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 480 });
  await page.goto('/test-fixtures/experience.html');
  await page.getByRole('button', { name: '核对输入' }).click();
  await expect(page.getByRole('tab', { name: '补充资料' })).toHaveAttribute('aria-selected', 'true');
  await expect(page.getByLabel('明细名称')).toBeFocused();
  await expect(page.getByLabel('明细名称')).toBeInViewport();
  await page.getByLabel('明细名称').fill('未保存的展示输入');
  await page.getByRole('tab', { name: '基本资料', exact: true }).click();
  await expect(page.getByLabel('明细名称')).not.toBeVisible();
  await page.getByRole('tab', { name: '补充资料', exact: true }).click();
  await expect(page.getByLabel('明细名称')).toHaveValue('未保存的展示输入');
  await page.getByRole('region', { name: '夹具正文' }).hover();
  await page.mouse.wheel(0, 500);
  await expect.poll(() => page.evaluate(() => scrollY)).toBeGreaterThan(0);

});
