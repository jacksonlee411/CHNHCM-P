import { desktopViewports } from './desktop-viewports.mjs';
import { policy } from './community-list-support.mjs';
import { directoryReturnJourney, geoField, openGeoFilter } from './geoarea-directory-support';
import { test, expect, openMoreActions, type Page } from './fixtures';
import { open, select } from './bu-support';
import { request } from './dated-object-support';
import { randomUUID } from 'node:crypto';
test.describe.configure({ timeout: 90000 });
const root = '/settings/basic-data/geo-areas';
async function menu(page: Page, code?: string) {
  const listed = page.waitForResponse(r => r.status() === 200 && r.request().method() === 'GET' && new URL(r.url()).pathname === '/api/v1/geo-areas' && !new URL(r.url()).searchParams.get('q') && (new URL(r.url()).searchParams.get('page') ?? '1') === '1');
    const compact = await page.evaluate(() => matchMedia('(max-width:899px)').matches);
    const opener = page.getByRole('button', { name: '打开导航', exact: true });
    if (compact) await expect(opener).toBeVisible();
    else await expect(opener).toBeHidden();
    if (compact) {
        await page.getByRole('button', { name: '打开导航', exact: true }).click();
        const drawer = page.locator('#navigation-drawer');
        if (!await drawer.getByRole('button', { name: '行政区划', exact: true }).isVisible())
            await drawer.getByRole('button', { name: '基础资料', exact: true }).click();
        await drawer.getByRole('button', { name: '行政区划', exact: true }).click();
    }
    else {
        await page.getByRole('button', { name: '基础资料', exact: true }).click();
        await page.getByRole('menuitem', { name: '行政区划', exact: true }).click();
    }
    await expect(page).toHaveURL(new RegExp(`${root}$`));
    await expect(page.getByRole('heading', { name: '行政区划', exact: true })).toBeVisible();
  await listed;
    if (code) {
        await page.getByRole('textbox', { name: '搜索编码或名称' }).fill(code);
        const queried = page.waitForResponse(r => r.request().method() === 'GET' && new URL(r.url()).pathname === '/api/v1/geo-areas' && new URL(r.url()).searchParams.get('q') === code);
        await page.getByRole('button', { name: '查询', exact: true }).click();
        const response = await queried;
        expect(response.status()).toBe(200);
        const row = (await response.json()).items.find((r: {code: string}) => r.code === code);
        expect(row).toBeDefined();
        await page.getByRole('link', { name: row.name, exact: true }).click();
        const detail = await request(page, 'GET', `/api/v1/geo-areas/${row.id}`);
        expect(detail.status).toBe(200);
        expect(detail.body.timeline[0].effectiveDate).toBe('1900-01-01');
        await expect(page.getByRole('navigation', { name: '生效日期', exact: true }).locator('[aria-current="date"]')).toContainText(detail.body.selectedRecord.effectiveDate);
    }
}
async function read(page: Page) {
    const id = new URL(page.url()).pathname.split('/').at(-1);
    return (await request(page, 'GET', '/api/v1/geo-areas/' + id)).body;
}
test('区划203 正常菜单完整目录、分页和无创建或选择导出', async ({ page }) => {
    const forbidden: string[] = [];
    page.on('request', r => { if (/geo-areas\/(selection|references)/.test(r.url()))
        forbidden.push(r.url()); });
    await open(page);
    await menu(page);
    await expect(page.getByRole('link', { name: '新建', exact: true })).toHaveCount(0);
    await expect(page.getByRole('checkbox')).toHaveCount(0);
    await expect(page.getByRole('button', { name: '导出所选', exact: true })).toHaveCount(0);
    const first = await request(page, 'GET', '/api/v1/geo-areas?pageSize=100');
    expect(first.body.count).toBe(3630);
    await page.getByRole('button', { name: /下一页/ }).click();
    await expect(page).toHaveURL(/page=2/);
    await menu(page, '110101');
    await expect(page.getByText('来源资料', { exact: true })).toHaveCount(0);
    await expect(page.getByText('base.uk', { exact: true })).toHaveCount(0);
    const detail = await read(page);
    expect(detail.code).toBe('110101');
    expect(detail).not.toHaveProperty('origin');
    expect(detail.capabilities.readFields).not.toContain('source');
    expect(forbidden).toEqual([]);
});
test('区划203 共用更正、未保存保护、日志和准确原来源', async ({ page }) => {
    await open(page);
    await menu(page, '110102');
    const before = await read(page);
    await openMoreActions(page);
    await expect(page.getByRole('menuitem', { name: /删除|插入停用/ })).toHaveCount(0);
    await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
    const name = page.getByRole('textbox', { name: '显示名称', exact: true });
    await expect(name).toHaveValue(before.selectedRecord.name);
    await expect(page.locator('#record-date')).toHaveAttribute('readonly', '');
    await name.fill('西城本地显示');
    await page.getByRole('button', { name: '取消编辑', exact: true }).click();
    await page.getByRole('button', { name: '继续编辑', exact: true }).click();
    await expect(name).toHaveValue('西城本地显示');
    const saving = page.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith('/correct'));
    await page.getByRole('button', { name: '保存', exact: true }).click();
    expect((await saving).status()).toBe(200);
    await expect(page.locator('.object-form')).toHaveCount(0);
    const after = await read(page);
    expect(after.selectedRecord.name).toBe('西城本地显示');
    expect(after).not.toHaveProperty('origin');
    const original = await request(page, 'GET', `/api/v1/geo-areas/${after.id}?sourceVersion=${before.headSourceVersion}`);
    expect(original.body.selectedRecord.name).toBe(before.selectedRecord.name);
    await page.getByRole('tab', { name: /变更日志/ }).click();
    const history = page.locator('#geoArea-history-panel');
    await expect(history.getByText('西城本地显示', { exact: true }).first()).toBeVisible();
    const source = history.locator('a[href*="sourceVersion="]').first();
    await expect(source).toBeVisible();
    await source.click();
    await expect(page.getByText("正在查看只读历史资料", { exact: true }).first()).toBeVisible();
    await expect(geoField(page, '原采用路径')).toContainText('中国');
    await expect(geoField(page, '国家/地区')).toHaveText('中国');
});
test('区划203 追加与停用恢复复用时间线并保留初始日期', async ({ page }) => {
    await open(page);
    await menu(page, '110105');
    await page.getByRole('button', { name: '新增记录', exact: true }).click();
    await expect(page.locator('#record-date')).not.toHaveValue('1900-01-01');
    await page.locator('#record-date').fill('1901-01-01');
    await page.getByRole('textbox', { name: '显示名称', exact: true }).fill('朝阳本地显示');
    await page.getByRole('button', { name: '保存', exact: true }).click();
    await expect(page.locator('.object-form')).toHaveCount(0);
    await select(page, '1901-01-01');
    await openMoreActions(page);
    await page.getByRole('menuitem', { name: '停用行政区划与地址分区', exact: true }).click();
    await page.locator('#record-date').fill('1902-01-01');
    await page.getByRole('button', { name: '确认停用', exact: true }).click();
    await expect(page.locator('.object-form')).toHaveCount(0);
    expect((await read(page)).status).toBe('inactive');
    await select(page, '1902-01-01');
    await openMoreActions(page);
    await page.getByRole('menuitem', { name: '重新启用行政区划与地址分区', exact: true }).click();
    await page.locator('#record-date').fill('1903-01-01');
    await page.getByRole('button', { name: '确认重新启用', exact: true }).click();
    await expect(page.locator('.object-form')).toHaveCount(0);
    const restored = await read(page);
    expect(restored.status).toBe('active');
    expect(restored.timeline.map((r: {
        effectiveDate: string;
    }) => r.effectiveDate)).toEqual(['1900-01-01', '1901-01-01', '1902-01-01', '1903-01-01']);
});
test('区划203 查询者可读当前和历史、无来源资料或维护按钮', async ({ page }) => {
    await open(page, 'reader');
    await menu(page, '110108');
    await expect(page.getByRole('button', { name: '新增记录', exact: true })).toHaveCount(0);
    await expect(page.getByRole('button', { name: '更多操作', exact: true })).toHaveCount(0);
    await expect(page.getByText('来源资料', { exact: true })).toHaveCount(0);
    await expect(page.getByText('base.cn', { exact: true })).toHaveCount(0);
    const detail = await read(page);
    expect(detail.capabilities.history).toBe(true);
    expect(detail.code).toBe('110108');
    expect(detail).not.toHaveProperty('origin');
    expect(detail.capabilities.readFields).not.toContain('source');
    expect(new URL(page.url()).pathname.startsWith(root)).toBe(true);
});
test('区划203 已保存但丢响应、刷新后按原提交查回', async ({ page }) => {
    await open(page);
    await menu(page, '110106');
    await openMoreActions(page);
    await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
    const recoveredName = `原提交恢复名称 ${randomUUID().slice(0, 8)}`;
    await page.getByRole('textbox', { name: '显示名称', exact: true }).fill(recoveredName);
    let accepted: {
        sourceVersion: string;
        submissionId: string;
    } | undefined;
    await page.route('**/api/v1/geo-areas/*/records/*/correct', async (route) => {
        const response = await route.fetch();
        expect(response.status()).toBe(200);
        accepted = await response.json();
        await route.abort('connectionreset');
    });
    await page.getByRole('button', { name: '保存', exact: true }).click();
    await expect(page.getByRole('button', { name: '查看保存结果', exact: true })).toBeVisible();
    await page.reload();
    await page.getByRole('region', { name: '行政区划与地址分区提交结果', exact: true }).getByRole('button', { name: '查看保存结果', exact: true }).click();
    await page.getByRole('link', { name: '查看保存结果', exact: true }).click();
    await expect(page.getByText('正在查看只读历史资料', { exact: true })).toBeVisible();
    expect(new URL(page.url()).searchParams.get('sourceVersion')).toBe(accepted?.sourceVersion);
    expect((await read(page)).selectedRecord.name).toBe(recoveredName);
    const recovered = await request(page, 'POST', `/api/v1/geo-areas/submissions/${accepted?.submissionId}/resolve`);
    expect(recovered.body.sourceVersion).toBe(accepted?.sourceVersion);
});
test('区划203 国家列筛选与下级范围在详情返回时保持', async ({ page }) => {
    await open(page);
    await menu(page);
    const dialog = await openGeoFilter(page,'国家/地区');
    await dialog.getByRole('combobox',{name:'值',exact:true}).click();
    await page.getByRole('option',{name:'中国',exact:true}).click();
    expect(new URL(page.url()).searchParams.has('filterField')).toBe(false);
    const listed = page.waitForResponse(r => r.status()===200 && new URL(r.url()).pathname==='/api/v1/geo-areas' && new URL(r.url()).searchParams.get('filterField')==='countryRegionId');
    await dialog.getByRole('button',{name:'应用'}).click();
    expect((await (await listed).json()).count).toBe(3209);
    await expect(page.getByRole('button',{name:'国家/地区 等于 中国'})).toBeVisible();
    await expect(page.locator('.MuiDataGrid-row [data-field="countryRegionId"]').first()).toHaveText('中国');
    const before = page.url();
    const childrenLoaded = page.waitForResponse(r => r.status() === 200 && new URL(r.url()).pathname === '/api/v1/geo-areas' && !!new URL(r.url()).searchParams.get('parentAreaId'));
    await page.getByRole('link', { name: '查看下级', exact: true }).first().click();
    const children = await (await childrenLoaded).json();
    await expect(page).toHaveURL(/parentAreaId=/);
    const query = new URL(page.url()).search;
    const firstChild = page.locator(`.MuiDataGrid-row[data-id="${children.items[0].id}"] a[data-object-link]`);
    await expect(firstChild).toHaveAttribute('href', /parentAreaId/);
    await firstChild.click();
    await expect(page.getByRole('button', { name: '1900-01-01', exact: true })).toBeVisible();
    const first = await read(page);
    await expect(page.getByRole('button', { name: '下一个对象', exact: true })).toBeEnabled();
    await page.getByRole('button', { name: '下一个对象', exact: true }).click();
    await expect.poll(async () => (await read(page)).id).not.toBe(first.id);
    await expect(page.getByText('对象导航暂时不可用，请返回列表重新查询。', { exact: true })).toHaveCount(0);
    await page.getByRole('link', { name: '返回列表', exact: true }).click();
    await expect.poll(() => new URL(page.url()).searchParams.get('parentAreaId')).toBe(new URLSearchParams(query).get('parentAreaId'));
    expect(before).toContain('countryRegionId');
});
test('区划138 澳门目录可切换层级并清除列条件', async ({page}) => {
    await open(page,'reader');
    await menu(page);
    const dialog = await openGeoFilter(page,'国家/地区');
    const apply = async (field: string, count: number) => {
        const next = page.waitForResponse(r => {
            const url = new URL(r.url());
            return r.status() === 200 && url.pathname === '/api/v1/geo-areas' && (url.searchParams.get('filterField') ?? '') === field;
        });
        await dialog.getByRole('button',{name:'应用'}).click();
        expect((await (await next).json()).count).toBe(count);
    };
    await dialog.getByRole('combobox',{name:'值'}).click();
    await page.getByRole('option',{name:/澳门/}).click();
    await apply('countryRegionId',10);
    const parish = page.getByRole('row').filter({has:page.getByRole('link',{name:'望德堂區',exact:true})});
    await expect(parish.locator('[data-field="level"]')).toHaveText('堂区(澳门)');
    await expect(parish.locator('[data-field="geoAreaPath"]')).toContainText('澳门 / 望德堂區');
    await expect(page.getByRole('row').filter({has:page.getByRole('link',{name:'氹仔',exact:true})}).locator('[data-field="level"]')).toHaveText('区域(澳门)');
    await page.getByRole('button',{name:/国家\/地区 等于 .*澳门/}).click();
    await dialog.getByRole('combobox',{name:'列',exact:true}).click();
    await page.getByRole('option',{name:'层级',exact:true}).click();
    await dialog.getByRole('combobox',{name:'值'}).click();
    await page.getByRole('option',{name:'县市(台湾)',exact:true}).click();
    await apply('level',22);
    await expect(page.locator('.MuiDataGrid-row [data-field="level"]').first()).toHaveText('县市(台湾)');
    await page.getByRole('button',{name:'层级 等于 县市(台湾)'}).click();
    await dialog.getByRole('button',{name:'清除条件'}).click();
    await apply('',3630);
});
test('区划203 桌面显示字段和更正表单', async ({ page }) => {
    await open(page);
    await menu(page, '110109');
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
    await geoField(page, '代码方案').filter({hasText: 'CN-MCA'}).scrollIntoViewIfNeeded();
    await expect(geoField(page, '代码方案').filter({hasText: 'CN-MCA'})).toBeVisible();
    await openMoreActions(page);
    await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
    await page.getByRole('textbox', { name: '英文名称', exact: true }).scrollIntoViewIfNeeded();
    await expect(page.getByRole('textbox', { name: '英文名称', exact: true })).toBeVisible();
    await page.getByRole('button', { name: '保存', exact: true }).scrollIntoViewIfNeeded();
    await expect(page.getByRole('button', { name: '保存', exact: true })).toBeInViewport();
    await page.getByRole('button', { name: '取消编辑', exact: true }).click();
});

test('区划203 香港两种性质、省直连县和台湾同名路径', async ({ page }) => {
    test.setTimeout(180000);
    await open(page);
    for (const item of [
        { code: 'HK', nature: '地址分区', parent: '香港' },
        { code: 'CW', nature: '行政区划', parent: '香港島' },
        { code: '419001', nature: '行政区划', parent: '河南省' },
        { code: '10018010', nature: '行政区划', parent: '新竹市' },
        { code: '10020010', nature: '行政区划', parent: '嘉義市' },
    ]) {
        await menu(page, item.code);
        await expect(geoField(page, '性质')).toHaveText(item.nature);
        await expect(geoField(page, '路径')).toContainText(item.parent);
        const value = await read(page);
        expect(value.geoAreaPath.at(-2).name).toBe(item.parent);
        expect(value.geoAreaPath.at(-1)).toMatchObject({id:value.id,name:value.selectedRecord.name});
        await expect(geoField(page, '路径')).toContainText(item.parent + ' / ' + value.selectedRecord.name);
        if (item.code.startsWith('100')) expect(value.selectedRecord.name).toBe('東區');
    }
});


test('区划203 同名摘要列直接显示完整父路径并可下钻',async({page})=>{
 await open(page);await page.setViewportSize({width:899,height:900});await menu(page);
 await page.getByRole('textbox',{name:'搜索编码或名称'}).fill('東區');
 await page.getByRole('button',{name:'查询',exact:true}).click();
 await expect(page.getByText(/新竹市/).first()).toBeVisible();
 await expect(page.getByText(/嘉義市/).first()).toBeVisible();
 await expect(page.getByRole('link',{name:'查看下级',exact:true}).first()).toBeVisible();
 await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
});

test('区划138 查询草稿、第二页下钻和两次返回定位', async ({page}) => {
    test.setTimeout(120000);
    await open(page, 'reader');
    for (const [width, height] of desktopViewports) {
        await page.setViewportSize({width, height});
        await menu(page);
        await directoryReturnJourney(page);
    }
});

test('区划138 逐层返回、刷新与末级空范围', async ({page}) => {
    await open(page);
    await menu(page);
    const search = page.getByRole('textbox', {name:'搜索编码或名称'});
    await search.fill('連江縣');
    await search.press('Enter');
    const drill = (name: string) => page.getByRole('row').filter({has:page.getByRole('link', {name, exact:true})}).getByRole('link', {name:'查看下级'});
    await drill('連江縣').click();
    await drill('南竿鄉').click();
    await page.reload();
    await expect(page.getByText('当前范围没有可显示的直接下级', {exact:true})).toBeVisible();
    await expect(page.getByText('南竿鄉的直接下级', {exact:true})).toBeVisible();
    const back = page.getByRole('link', {name:'返回上次列表'});
    await back.click();
    await expect(drill('南竿鄉')).toBeFocused();
    await back.click();
    await expect(drill('連江縣')).toBeFocused();
    await expect(search).toHaveValue('連江縣');
    await drill('連江縣').click();
    await page.getByRole('link', {name:'清除上级范围'}).click();
    await expect(page.getByRole('link', {name:'北竿鄉', exact:true})).toBeVisible();
    expect(new URL(page.url()).searchParams.has('fromList')).toBe(false);
    await expect(back).toHaveCount(0);
});

test('区划138 下钻期间收回上级字段后仍可返回获权列表', async ({page}) => {
    await open(page, 'reader');
    await menu(page);
    const county = page.getByRole('link', {name:'連江縣', exact:true});
    await page.getByRole('row').filter({has:county}).getByRole('link', {name:'查看下级'}).click();
    const namedScope = page.getByText('連江縣的直接下级', {exact:true});
    await expect(namedScope).toBeVisible();
    await policy(page, async (where: string, mutate: (statement: string) => void) => {
        mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'parentAreaId') WHERE ${where} AND action='geoarea.read'`);
        const denied = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/geo-areas' && r.status() === 403);
        await page.getByRole('button', {name:'查询', exact:true}).click();
        await denied;
        await expect(namedScope).toHaveCount(0);
        await expect(page.getByText('直接下级范围不可用', {exact:true})).toBeVisible();
        await expect(page.getByText('当前范围没有可显示的直接下级', {exact:true})).toHaveCount(0);
        await page.getByRole('link', {name:'返回上次列表'}).click();
        await expect(county).toBeFocused();
        await expect(page.getByRole('link', {name:'查看下级'})).toHaveCount(0);
    });
});
