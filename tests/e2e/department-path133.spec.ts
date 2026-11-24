import { test, expect, openMoreActions } from './fixtures';
import { open } from './bu-support';
import { menu, departmentKind, apiBU, apiDepartment } from './dated-object-support';

test('M1路径无总数分页，第三页详情返回后仍可前后翻页并完整选择', async ({ page }) => {
  // The full journey creates 41 objects and performs over 140 HTTP requests.
  // Its recorded 30s timeouts reached later navigation assertions with no
  // pending requests; this budget is separate from per-request latency goals.
  test.setTimeout(60000);
  await open(page);
  const bu = await apiBU(page), name = '路径游标' + Date.now();
  const codes: string[] = [];
  for (let i = 0; i < 41; i++) codes.push((await apiDepartment(page, bu.id, { name })).code);
  codes.sort();
  await menu(page, departmentKind);
  await openMoreActions(page);
  await page.getByRole('menuitem', { name: '列筛选…', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: '列筛选', exact: true });
  await dialog.getByRole('combobox', { name: '列', exact: true }).click();
  await page.getByRole('option', { name: '部门路径', exact: true }).click();
  await dialog.getByRole('textbox').fill(name);
  await dialog.getByRole('button', { name: '应用', exact: true }).click();
  const grid = page.getByRole('grid', { name: '部门列表', exact: true });
  await expect(grid.locator('.MuiDataGrid-row').filter({ hasText: codes[0] })).toBeVisible();
  for (const index of [20, 40]) {
    const response = page.waitForResponse(r => r.request().method() === 'GET' && new URL(r.url()).pathname === '/api/v1/departments');
    await page.getByRole('button', { name: '下一页', exact: true }).click();
    const received = await response; expect(received.status()).toBe(200);
    expect(new URL(received.url()).searchParams.has('cursor')).toBe(true);
    const result = await received.json();
    expect(result).not.toHaveProperty('count');
    expect(result.hasMore).toBe(index === 20);
    await expect(grid.locator('.MuiDataGrid-row').filter({ hasText: codes[index] })).toBeVisible();
  }
  await grid.locator('.MuiDataGrid-row').filter({ hasText: codes[40] }).getByRole('link').click();
  await expect(page.getByRole('navigation', { name: '对象导航', exact: true })).toContainText('41 / 41');
  await page.getByRole('link', { name: '返回列表', exact: true }).click();
  await expect(grid.locator('.MuiDataGrid-row').filter({ hasText: codes[40] })).toBeVisible();
  await page.reload();
  await expect(grid.locator('.MuiDataGrid-row').filter({ hasText: codes[40] })).toBeVisible();
  await page.getByRole('button', { name: '上一页', exact: true }).click();
  await expect(grid.locator('.MuiDataGrid-row').filter({ hasText: codes[20] })).toBeVisible();
  await page.getByRole('button', { name: '上一页', exact: true }).click();
  await expect(grid.locator('.MuiDataGrid-row').filter({ hasText: codes[0] })).toBeVisible();
  // Crossing the cursor boundary in detail returns to the actual target page.
  await page.locator('.MuiDataGrid-virtualScroller').evaluate(el => { el.scrollTop = el.scrollHeight; });
  await grid.locator('.MuiDataGrid-row').filter({ hasText: codes[19] }).getByRole('link').click();
  await page.getByRole('button', { name: '下一个对象', exact: true }).click();
  await expect(page.getByRole('heading', { level: 1 })).toContainText(codes[20]);
  await page.getByRole('link', { name: '返回列表', exact: true }).click();
  const returned = grid.locator('.MuiDataGrid-row').filter({ hasText: codes[20] }).getByRole('link');
  await expect(returned).toBeFocused();
  await expect(page.getByText('第2页 · 21–40', { exact: true })).toBeVisible();
  // Auxiliary neighbor failure leaves the independently loaded detail usable.
  await page.route('**/api/v1/departments/navigation', route => route.fulfill({ status: 503, json: { code: 'M02_DEPENDENCY_UNAVAILABLE', message: '受控导航失败' } }));
  await returned.click();
  await expect(page.getByRole('heading', { level: 1 })).toContainText(codes[20]);
  await expect(page.getByText('暂时无法切换记录', { exact: true })).toBeVisible();
  await page.getByRole('link', { name: '返回列表', exact: true }).click();
  await page.unroute('**/api/v1/departments/navigation');
  await page.getByRole('checkbox', { name: '全选查询结果', exact: true }).check();
  await expect(page.getByRole('toolbar', { name: '所选部门', exact: true })).toContainText('已选择41项');
});
