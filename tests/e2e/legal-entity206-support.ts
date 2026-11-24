import { expect, type Page } from './fixtures';
import { request } from './dated-object-support';
export const root = '/settings/organization/legal-entities';
export async function menu(page: Page) {
  if (await page.evaluate(() => matchMedia('(max-width:899px)').matches)) {
    await page.getByRole('button', { name: '打开导航', exact: true }).click();
    const drawer = page.locator('#navigation-drawer');
    if (!await drawer.getByRole('button', { name: '法律实体', exact: true }).isVisible()) await drawer.getByRole('button', { name: '组织配置', exact: true }).click();
    await drawer.getByRole('button', { name: '法律实体', exact: true }).click();
  } else {
    await page.getByRole('button', { name: '组织配置', exact: true }).click();
    await page.getByRole('menuitem', { name: '法律实体', exact: true }).click();
  }
  await expect(page).toHaveURL(new RegExp(`${root}$`));
  await expect(page.getByRole('heading', { name: '法律实体', exact: true })).toBeVisible();
}
export async function detail(page: Page) { return (await request(page, 'GET', '/api/v1/legal-entities/' + new URL(page.url()).pathname.split('/').at(-1))).body; }
export async function create(page: Page, country = '美国', category = '分支机构') {
  await menu(page); await page.getByRole('link', { name: '新建', exact: true }).click();
  const code = `LE206-${Date.now()}`;
  await page.getByRole('textbox', { name: '编码', exact: true }).fill(code);
  await page.getByRole('textbox', { name: '法律实体名称', exact: true }).fill(code + '主体');
  await expect(page.locator('#legalentity-effectiveDate')).toHaveValue('1900-01-01');
  await page.locator('#legalentity-effectiveDate').fill('2020-01-01');
  await page.getByRole('combobox', { name: '国家/地区', exact: true }).click();
  await page.getByRole('option', { name: country, exact: true }).click();
  await page.getByRole('combobox', { name: '主体类别', exact: true }).click();
  await page.getByRole('option', { name: category, exact: true }).click();
  await page.getByRole('textbox', { name: '登记类型', exact: true }).fill('登记证');
  await page.getByRole('textbox', { name: '登记号码', exact: true }).fill('00100');
  await page.getByRole('textbox', { name: '设立单位说明', exact: true }).fill('设立单位原始资料');
  const saved = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === '/api/v1/legal-entities');
  await page.getByRole('button', { name: '保存', exact: true }).click();
  expect((await saved).status()).toBe(201);
  await expect(page).toHaveURL(new RegExp(`${root}/[0-9a-f-]+`));
  await expect(page.getByRole('button', { name: '新增记录', exact: true })).toBeVisible();
  return code;
}
