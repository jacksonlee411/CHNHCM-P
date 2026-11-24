import { test, expect } from './fixtures';
import { open } from './bu-support';
import { apiBU, apiDepartment, departmentKind, menu, reference } from './dated-object-support';

test('Q131-V15 正常新建首次无搜索父候选，无选择和导航预热', async ({ page }) => {
  await open(page);
  const bu = await apiBU(page), parent = await apiDepartment(page, bu.id);
  const calls: string[] = [];
  page.on('request', request => {
    const path = new URL(request.url()).pathname;
    if (path.startsWith('/api/v1/departments')) calls.push(path);
  });
  await menu(page, departmentKind);
  await expect(page.getByRole('grid', { name: '部门列表', exact: true })).toBeVisible();
  expect(calls.filter(path => /selection|navigation|references\/parents/.test(path))).toEqual([]);
  await page.getByRole('link', { name: '新建', exact: true }).click();
  // The candidate fixture starts in 2026; initial master-data default remains 1900.
  await expect(page.locator('#department-effectiveDate')).toHaveValue('1900-01-01');
  await page.locator('#department-effectiveDate').fill('2026-01-01');
  await reference(page, '管理业务单元', bu.code);
  const response = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/departments/references/parents');
  await page.getByRole('combobox', { name: '行政上级', exact: true }).click();
  const received = await response;
  expect(received.status()).toBe(200);
  expect(received.request().postDataJSON().q ?? '').toBe('');
  const data = await received.json();
  expect(data.count).toBeUndefined(); expect(data.hasMore).toBe(false); expect(data.items.map((v: { id: string }) => v.id)).toEqual([parent.id]);
  await expect(page.getByRole('option').filter({ hasText: parent.code })).toBeVisible();
  expect(calls.filter(path => /selection|navigation/.test(path))).toEqual([]);
});
