import {expect,type Page} from './fixtures';
import {request} from './dated-object-support';
export const root = '/settings/organization/locations';
export const addressField = (page: Page, label: string) => page.locator('.object-detail-section')
    .filter({ has: page.getByRole('heading', { name: '基本资料', exact: true }) })
    .locator('.object-detail-field').filter({ has: page.getByText(label, { exact: true }) })
    .locator('.object-field-value');
export async function menu(page: Page) {
    const compact = await page.evaluate(() => matchMedia('(max-width:899px)').matches);
    if (compact) {
        await page.getByRole('button', { name: '打开导航', exact: true }).click();
        const d = page.locator('#navigation-drawer');
        if (!await d.getByRole('button', {name:'地点',exact:true}).isVisible()) await d.getByRole('button',{name:'组织配置',exact:true}).click();
        await d.getByRole('button', { name: '地点', exact: true }).click();
    }
    else {
        await page.getByRole('button', { name: '组织配置', exact: true }).click();
        await page.getByRole('menuitem', { name: '地点', exact: true }).click();
    }
    await expect(page).toHaveURL(new RegExp(`${root}$`));
    await expect(page.getByRole('heading', { name: '地点', exact: true })).toBeVisible();
}
export async function detail(page: Page) { return (await request(page, 'GET', '/api/v1/locations/' + new URL(page.url()).pathname.split('/').at(-1))).body; }
export async function create(page: Page, region: string, area?: string) {
    await menu(page);
    await page.getByRole('link', { name: '新建', exact: true }).click();
    const code = `L204-${Date.now()}`;
    await page.getByRole('textbox', { name: '编码', exact: true }).fill(code);
    await page.getByRole('textbox', { name: '地点名称', exact: true }).fill(code + '办公地点');
    // The isolated browser enterprise starts in 2020; the product still offers 1900.
    await expect(page.locator('#location-effectiveDate')).toHaveValue('1900-01-01');
    await page.locator('#location-effectiveDate').fill('2020-01-01');
    await page.getByRole('combobox', { name: '国家/地区', exact: true }).click();
    await page.getByRole('option', { name: region, exact: true }).click();
    if (area) {
        const control = page.getByRole('combobox', { name: '行政区划', exact: true });
        await expect(control).toBeEnabled();
        await control.fill(area);
        await page.getByRole('option').filter({ hasText: area }).first().click();
    }
    await page.getByRole('textbox', { name: '详细地址', exact: true }).fill('测试街一号');
    const path = area ? (await page.getByRole('combobox', { name: '行政区划', exact: true }).inputValue()).split(' / ').join('') : '';
    await expect(addressField(page, '完整地址')).toHaveText(path + '测试街一号');
    await expect(page.getByRole('textbox', { name: '完整地址', exact: true })).toHaveCount(0);
    await page.getByRole('textbox', { name: '邮政编码', exact: true }).fill('00100');
    const saving = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === '/api/v1/locations');
    await page.getByRole('button', { name: '保存', exact: true }).click();
    expect((await saving).status()).toBe(201);
    await expect(page).toHaveURL(new RegExp(`${root}/[0-9a-f-]+`));
    await expect(page.getByRole('button', { name: '新增记录', exact: true })).toBeVisible();
    return code;
}
