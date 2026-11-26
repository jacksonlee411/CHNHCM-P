import { test, expect } from './fixtures';
import { open } from './bu-support';
import { apiBU, apiDepartment } from './dated-object-support';
import { chooseApplication } from './app-navigation';

const departmentPath = '/settings/organization/departments';
const buPath = '/settings/organization/business-units';

for (const role of ['admin', 'reader']) test(`303 ${role}一级顺序、直接入口、详情历史刷新及设置切换`, async ({ page }) => {
  test.setTimeout(120000);
  await open(page);
  const bu = await apiBU(page), name = '一级菜单验证' + Date.now();
  const dep = await apiDepartment(page, bu.id, { name });
  if (role === 'reader') {
    await page.getByRole('button', { name: '退出', exact: true }).click();
    await open(page, role);
  }
  await expect(page).toHaveURL(new RegExp(`${buPath}$`));
  const app = page.getByRole('button', { name: '切换应用', exact: true });
  await app.click();
  await expect(page.locator('#navigation-apps-menu').getByRole('menuitem')).toHaveText(['部门', '人员', '设置']);
  await page.getByRole('menuitem', { name: '部门', exact: true }).click();
  await expect(page.getByRole('heading', { level: 1, name: '部门', exact: true })).toBeVisible();
  await expect(app).toHaveText('部门');
  await expect(page.getByRole('button', { name: '组织配置', exact: true })).toHaveCount(0);
  await expect(page.getByRole('button', { name: '部门', exact: true })).toHaveCount(0);
  if (role === 'reader') await expect(page.getByRole('link', { name: '新建', exact: true })).toHaveAttribute('aria-disabled', 'true');
  await page.getByRole('textbox', { name: '搜索编码或名称', exact: true }).fill(dep.code);
  await page.getByRole('button', { name: '查询', exact: true }).click();
  await page.getByRole('link', { name, exact: true }).click();
  const detail = page.url();
  await page.reload();
  await expect(app).toHaveText('部门');
  await page.getByRole('tab', { name: '变更日志', exact: true }).click();
  await page.getByRole('link', { name: '查看当次资料', exact: true }).first().click();
  await expect(page.getByText('正在查看只读历史资料', { exact: true })).toBeVisible();
  await expect(app).toHaveText('部门');
  await page.reload();
  await expect(app).toHaveText('部门');
  await expect(page.getByText('正在查看只读历史资料', { exact: true })).toBeVisible();
  await page.goto(detail);
  await page.getByRole('link', { name: '返回列表', exact: true }).click();
  await expect(page.getByRole('textbox', { name: '搜索编码或名称', exact: true })).toHaveValue(dep.code);
  await expect(page.locator('.list-return-selected')).toContainText(dep.code);
  await expect(page.getByRole('link', { name, exact: true })).toBeFocused();
  await chooseApplication(page, '设置');
  await expect(app).toHaveText('设置');
  await page.getByRole('button', { name: '组织配置', exact: true }).click();
  await expect(page.locator('#navigation-section-menu').getByRole('menuitem')).toHaveText(['业务单元', '企业', '法律实体', '地点']);
  await page.keyboard.press('Escape');
  for (const name of ['国家/地区', '行政区划']) {
    await page.getByRole('button', { name: '基础资料', exact: true }).click();
    await page.getByRole('menuitem', { name, exact: true }).click();
    await expect(page.getByRole('heading', { level: 1, name, exact: true })).toBeVisible();
    await expect(page.getByRole('row').nth(1)).toBeVisible();
    await expect(app).toHaveText('设置');
  }
});

for (const width of [1440, 899]) test(`303 ${width}跨应用取消离开与放弃保留正确归属`, async ({ page }) => {
  test.setTimeout(60000);
  await open(page);
  await page.setViewportSize({ width, height: 900 });
  await expect(page.locator('[data-navigation-mode]')).toHaveAttribute('data-navigation-mode', width >= 900 ? 'desktop' : 'drawer');
  for (const [from, to, field, path] of [
    ['部门', '设置', '部门名称', departmentPath],
    ['设置', '部门', '中文名称', buPath],
  ] as const) {
    await chooseApplication(page, from);
    await expect(page.getByRole('heading', { level: 1, name: from === '部门' ? '部门' : '业务单元', exact: true })).toBeVisible();
    await page.getByRole('link', { name: '新建', exact: true }).click();
    await expect(page).toHaveURL(new RegExp(`${path}/new`));
    await page.getByRole('textbox', { name: field, exact: true }).fill('保留未保存输入');
    await chooseApplication(page, to);
    await page.getByRole('button', { name: '继续编辑', exact: true }).click();
    await expect(page).toHaveURL(new RegExp(`${path}/new`));
    await expect(page.getByRole('textbox', { name: field, exact: true })).toHaveValue('保留未保存输入');
    if (width >= 900) await expect(page.getByRole('button', { name: '切换应用', exact: true })).toHaveText(from);
    else await expect(page.locator('[data-navigation-mode]')).toContainText(from);
    await chooseApplication(page, to);
    await page.getByRole('button', { name: /^(放弃修改|离开此页)$/ }).click();
    await expect(page.getByRole('heading', { level: 1, name: to === '部门' ? '部门' : '业务单元', exact: true })).toBeVisible();
  }
});

test('303 一级菜单键盘、抽屉顺序与尺寸切换焦点', async ({ page }) => {
  test.setTimeout(60000);
  await open(page);
  const app = page.getByRole('button', { name: '切换应用', exact: true });
  await app.focus(); await page.keyboard.press('Enter');
  await expect(page.locator('#navigation-apps-menu').getByRole('menuitem')).toHaveText(['部门', '人员', '设置']);
  await expect(page.getByRole('menuitem', { name: '设置', exact: true })).toBeFocused();
  await page.keyboard.press('ArrowUp');
  await expect(page.getByRole('menuitem', { name: '人员', exact: true })).toBeFocused();
  await page.keyboard.press('ArrowUp');
  await expect(page.getByRole('menuitem', { name: '部门', exact: true })).toBeFocused();
  await page.keyboard.press('Escape'); await expect(app).toBeFocused();
  await app.click();
  await page.keyboard.press('ArrowUp');
  await page.keyboard.press('ArrowUp');
  await page.keyboard.press('Enter');
  await expect(page.getByRole('heading', { level: 1, name: '部门', exact: true })).toBeVisible();
  await page.setViewportSize({ width: 899, height: 900 });
  await expect(page.locator('[data-navigation-mode]')).toHaveAttribute('data-navigation-mode', 'drawer');
  const trigger = page.getByRole('button', { name: '打开导航', exact: true });
  await trigger.click();
  const drawer = page.locator('#navigation-drawer');
  await expect(drawer.getByRole('button', { name: /^(部门|人员|设置)$/ })).toHaveText(['部门', '人员', '设置']);
  await expect(drawer.getByRole('button', { name: '部门', exact: true })).not.toHaveAttribute('aria-expanded');
  await drawer.getByRole('button', { name: '部门', exact: true }).click();
  await expect(drawer).not.toBeVisible();
  await page.getByRole('button', { name: '打开部门树', exact: true }).click();
  await expect(page.getByRole('tree', { name: '部门树', exact: true })).toBeVisible();
  await page.keyboard.press('Escape');
  await trigger.click(); await page.keyboard.press('Escape'); await expect(trigger).toBeFocused();
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
  await page.getByRole('button', { name: '打开导航', exact: true }).click();
  await page.setViewportSize({ width: 1440, height: 900 });
  await expect(page.locator('#navigation-drawer')).not.toBeVisible();
  await expect(app).toBeFocused();
});
