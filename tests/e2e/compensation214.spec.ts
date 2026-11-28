import { expect, test, type Page } from './fixtures';
import { openPersonnel } from './app-navigation';
import { desktopViewports } from './desktop-viewports.mjs';
import { signIn } from './hire-fixture';
import { calibrateDesktopWindow, nativeZoomBrowser } from './native-zoom-support';
import { request } from './dated-object-support';
import { preparePay, openPay, first, january } from './compensation214-support';

async function payJourney(page: Page) {
  const sample = await preparePay(page);
  await openPay(page, sample);
  const button = (name: string) => page.getByRole('button', { name, exact: true });
  const amount = page.getByLabel('个人基本工资（元/月）');
  await expect(page.getByText('尚未定薪', { exact: true })).toBeVisible();
  await button('首次定薪').click();
  await page.getByLabel('生效日期').fill(first);
  await expect(page.getByText('生效日制度', { exact: true })).toBeVisible();
  await amount.fill('999');
  await button('取消').click();
  await button('放弃修改').click();
  const root = '/api/v1/compensation/assignments/' + sample.assignment + '/base-pay';
  expect((await request(page, 'GET', root)).body.items).toHaveLength(0);
  await button('首次定薪').click();
  await page.getByLabel('生效日期').fill(first);
  await expect(page.getByText('生效日制度', { exact: true })).toBeVisible();
  await amount.fill('15500');
  await button('保存').click();
  await expect(amount).toBeFocused();
  await amount.fill('12100');
  await page.route(
    '**' + root,
    async (route) => {
      if (route.request().method() === 'POST') {
        await route.fetch();
        await route.abort('failed');
      } else await route.continue();
    },
    { times: 1 },
  );
  await button('保存').click();
  await expect(page.getByText(/已保存 · 申请号/)).toBeVisible({ timeout: 25000 });
  const records = page.getByRole('table', { name: '按日核定记录' });
  await records.getByRole('button', { name: '更正' }).click();
  await expect(amount).toHaveValue('12100.00');
  await amount.fill('12000');
  await page.getByLabel('更正缘由').fill('核对原通知');
  await expect(page.getByText('生效日制度', { exact: true })).toBeVisible();
  await button('保存').click();
  // Open the next decision as soon as the prior form closes. Its timeline version must include the correction.
  await expect(page.locator('#pay-award-form')).toHaveCount(0);
  await button('调薪').click();
  await page.getByLabel('生效日期').fill(january);
  await expect(page.getByText('350.00 元/月', { exact: true })).toBeVisible();
  await amount.fill('13000');
  await page.getByLabel('业务原因').fill('新年度聘任决定');
  await button('保存').click();
  await expect(records).toContainText('13,000.00');
  await page.getByLabel('工资查询日').fill('2026-12-31');
  await button('查询').click();
  await expect(page.getByText('12,000.00 元/月', { exact: true })).toBeVisible();
  await expect(page.getByRole('link', { name: /16,000.00/ })).toBeVisible();
  const original = records
    .getByRole('row')
    .filter({ hasText: '12,100.00' })
    .getByRole('button', { name: '查看原件' });
  await original.click();
  await expect(page.getByText('12,100.00 元/月', { exact: true })).toBeVisible();
  await page.getByRole('link', { name: /15,000.00/ }).click();
  await expect(page.getByText('正在查看只读历史资料', { exact: true })).toBeVisible();
  await page.getByRole('link', { name: '返回员工薪酬' }).click();
  await expect(page.getByRole('link', { name: /15,000.00/ })).toBeFocused();
  await button('返回核定记录').click();
  await expect(original).toBeFocused();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
  await page.getByRole('link', { name: '返回人员' }).click();
  await expect(page.locator('a[data-object-link][href*="' + sample.assignment + '"]')).toBeFocused();
}
for (const [width, height] of desktopViewports) {
  test(`214 人员名录同级薪酬、断线查回及跨日原件 ${width}×${height}`, async ({ page }) => {
    test.setTimeout(120000);
    await page.setViewportSize({ width, height });
    await signIn(page);
    await payJourney(page);
  });
}
test('214 同级薪酬键盘办理及原件返回 原生200%', async () => {
  test.setTimeout(150000);
  const zoom = await nativeZoomBrowser();
  try {
    await signIn(zoom.page);
    await openPersonnel(zoom.page);
    await calibrateDesktopWindow(zoom.page);
    await zoom.setZoom(2);
    await payJourney(zoom.page);
  } finally {
    await zoom.close();
  }
});
