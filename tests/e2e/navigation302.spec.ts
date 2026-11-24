import { test, expect } from './fixtures';
import { open } from './bu-support';
import { apiBU, apiDepartment, menu, departmentKind } from './dated-object-support';

for (const role of ['admin', 'reader']) test(`302 ${role}详情位置总数、跨页切换无列表重读及返回`, async ({ page }) => {
  test.setTimeout(120000);
  await open(page);
  const bu = await apiBU(page), prefix = 'NAV302_' + Date.now();
  const root = await apiDepartment(page, bu.id, { code: prefix + '_00', name: prefix + '根' });
  const ids = [root.id];
  for (let i = 1; i < 22; i++) ids.push((await apiDepartment(page, bu.id, { code: prefix + '_' + String(i).padStart(2, '0'), name: prefix + i, parent: root.id })).id);
  if (role === 'reader') { await page.getByRole('button', { name: '退出', exact: true }).click(); await open(page, role); }
  await menu(page, departmentKind);
  const base = '/settings/organization/departments';
  await page.goto(base + '?' + new URLSearchParams({ q: prefix, scopeDepartmentId: root.id, includeDescendants: 'true' }));
  const grid = page.getByRole('grid', { name: '部门列表', exact: true });
  await grid.getByRole('link', { name: prefix + '根', exact: true }).click();
  const pager = page.getByRole('navigation', { name: '对象导航', exact: true });
  await expect(pager).toContainText('1 / 22');
  await expect(page.getByRole('button', { name: '上一个对象', exact: true })).toBeDisabled();
  await page.getByRole('link', { name: '返回列表', exact: true }).click();
  await expect(grid.getByRole('link', { name: prefix + '根', exact: true })).toBeFocused();
  await page.locator('.MuiDataGrid-virtualScroller').hover();
  await page.mouse.wheel(0, 3000);
  await grid.getByRole('link', { name: prefix + '19', exact: true }).click();
  await expect(pager).toContainText('20 / 22');
  const listReads: string[] = [];
  page.on('request', r => { if (r.method() === 'GET' && new URL(r.url()).pathname === '/api/v1/departments') listReads.push(r.url()); });
  for (const [direction, position] of [['下一个对象', 21], ['上一个对象', 20], ['下一个对象', 21], ['下一个对象', 22]] as const) {
    await page.getByRole('button', { name: direction, exact: true }).click();
    await expect(pager).toContainText(`${position} / 22`);
    await expect(page.getByRole('heading', { level: 1 })).toContainText(prefix + '_' + String(position - 1).padStart(2, '0'));
  }
  expect(listReads).toHaveLength(0);
  await expect(page.getByRole('button', { name: '下一个对象', exact: true })).toBeDisabled();
  for (const width of [390, 320]) {
    await page.setViewportSize({ width, height: 844 });
    await expect(pager.getByLabel('第22个，共22个匹配对象', { exact: true })).toBeVisible();
    expect(await pager.evaluate(el => el.getBoundingClientRect().right <= innerWidth)).toBe(true);
  }
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.getByRole('link', { name: '返回列表', exact: true }).click();
  await expect(grid.getByRole('link', { name: prefix + '21', exact: true })).toBeFocused();
  expect(new URL(page.url()).searchParams.get('page')).toBe('2');
  expect(new URL(page.url()).searchParams.get('scopeDepartmentId')).toBe(root.id);
});

test('302路径准确统计单独加载，延迟和失败均保留详情及切换', async ({ page }) => {
  await open(page);
  const bu = await apiBU(page), prefix = 'PATH302_' + Date.now();
  await apiDepartment(page, bu.id, { code: prefix + 'A', name: prefix + 'A' });
  await apiDepartment(page, bu.id, { code: prefix + 'B', name: prefix + 'B' });
  await menu(page, departmentKind);
  await page.goto('/settings/organization/departments?' + new URLSearchParams({ q: prefix, filterField: 'departmentPath', filterOperator: 'contains', filterValue: prefix }));
  let release!: () => void;
  const gate = new Promise<void>(resolve => { release = resolve; });
  await page.route('**/api/v1/departments/navigation', async route => {
    if (route.request().postDataJSON().direction === 'current') await gate;
    await route.continue();
  });
  await page.getByRole('grid', { name: '部门列表', exact: true }).getByRole('link', { name: prefix + 'A', exact: true }).click();
  await expect(page.getByRole('heading', { level: 1 })).toContainText(prefix + 'A');
  await expect(page.getByRole('button', { name: '下一个对象', exact: true })).toBeEnabled();
  const pager = page.getByRole('navigation', { name: '对象导航', exact: true });
  await expect(pager.locator('.record-pager-counter')).toHaveCount(0);
  release();
  await expect(pager).toContainText('1 / 2');
  await page.unroute('**/api/v1/departments/navigation');
  await page.route('**/api/v1/departments/navigation', route => route.request().postDataJSON().direction === 'current'
    ? route.fulfill({ status: 503, json: { code: 'M02_DEPENDENCY_UNAVAILABLE', message: '统计暂不可用' } }) : route.continue());
  await page.reload();
  await expect(page.getByText('位置与总数暂时无法加载', { exact: true })).toBeVisible();
  await expect(page.getByRole('heading', { level: 1 })).toContainText(prefix + 'A');
  await expect(page.getByRole('button', { name: '下一个对象', exact: true })).toBeEnabled();
  await page.unroute('**/api/v1/departments/navigation');
  await page.getByRole('button', { name: '重试', exact: true }).click();
  await expect(pager).toContainText('1 / 2');
  await expect(page.getByText('位置与总数暂时无法加载', { exact: true })).toHaveCount(0);
});

for (const [kind, label] of [['country-regions', '国家/地区'], ['geo-areas', '行政区划']] as const)
  for (const role of ['admin', 'reader']) test(`302 ${role}${label}共享导航位置及切换`, async ({ page }) => {
    await open(page, role);
    await page.getByRole('button', { name: '基础资料', exact: true }).click();
    const loaded = page.waitForResponse(r => r.status() === 200 && r.request().method() === 'GET' && new URL(r.url()).pathname === '/api/v1/' + kind);
    await page.getByRole('menuitem', { name: label, exact: true }).click();
    const list = await (await loaded).json();
    expect(list.count).toBe(kind === 'country-regions' ? 251 : 3630);
    await page.locator(`.MuiDataGrid-row[data-id="${list.items[0].id}"] a[data-object-link]`).click();
    const pager = page.getByRole('navigation', { name: '对象导航', exact: true });
    await expect(pager).toContainText(`1 / ${list.count}`);
    const listReads: string[] = [];
    page.on('request', r => { if (r.method() === 'GET' && new URL(r.url()).pathname === '/api/v1/' + kind) listReads.push(r.url()); });
    await page.getByRole('button', { name: '下一个对象', exact: true }).click();
    await expect(pager).toContainText(`2 / ${list.count}`);
    await expect(page).toHaveURL(new RegExp('/' + list.items[1].id + '[?]'));
    await page.getByRole('button', { name: '上一个对象', exact: true }).click();
    await expect(pager).toContainText(`1 / ${list.count}`);
    expect(listReads).toHaveLength(0);
    await page.getByRole('link', { name: '返回列表', exact: true }).click();
    await expect(page.locator(`.MuiDataGrid-row[data-id="${list.items[0].id}"] a[data-object-link]`)).toBeFocused();
  });
