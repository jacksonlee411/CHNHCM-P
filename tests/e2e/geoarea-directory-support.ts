import { test, expect, type Page } from './fixtures';

export const geoField = (page: Page, label: string) => page.locator('.object-detail-field')
    .filter({ has: page.getByText(label, { exact: true }) }).locator('.object-field-value');

export async function openGeoFilter(page: Page, label: '国家/地区' | '层级') {
    if (await page.evaluate(() => innerWidth < 900)) {
        await page.getByRole('button', { name: new RegExp('^筛选' + label + '：') }).first().click();
    } else {
        const header = page.getByRole('columnheader', { name: new RegExp(label) });
        await header.hover();
        await header.getByLabel('列操作').click();
        await page.getByRole('menuitem', {name:'筛选…',exact:true}).click();
    }
    const dialog = page.getByRole('dialog', { name: '列筛选' });
    await expect(dialog.getByRole('combobox',{name:'列',exact:true})).toHaveText(label);
    await expect(dialog.getByRole('combobox',{name:'条件',exact:true})).toHaveText('等于');
    return dialog;
}
export async function directoryReturnJourney(page: Page) {
    const search = page.getByRole('textbox', { name: '搜索编码或名称' });
    const requests: URL[] = [];
    const observe = (req: import('@playwright/test').Request) => {
        const url = new URL(req.url());
        if (url.pathname === '/api/v1/geo-areas') requests.push(url);
    };
    await expect(page.getByRole('link', { name: '連江縣', exact: true })).toBeVisible();
    page.on('request', observe);
    try {
        const toolbar = page.locator('.list-control');
        await expect(toolbar.getByRole('combobox', {name: /国家|层级/})).toHaveCount(0);
        await expect(page.getByRole('checkbox', {name:'仅根层'})).toHaveCount(0);
        const more = toolbar.getByRole('button',{name:'更多操作'});
        await more.click();
        await expect(page.getByRole('menuitem',{name:/区划筛选|仅根层|国家|层级/})).toHaveCount(0);
        await page.keyboard.press('Escape');
        const screenshot = page.viewportSize()?.width === 1440 && page.viewportSize()?.height === 900;
        const mask = [page.locator('.app-account-name')];
        if (screenshot) await page.screenshot({path:test.info().outputPath('columns-default.png'),mask});
        if (screenshot) {
            const header = page.getByRole('columnheader',{name:/国家\/地区/});
            await header.hover();
            const handle = await header.locator('.MuiDataGrid-columnSeparator').first().boundingBox();
            if (!handle) throw new Error('国家列缺少调宽手柄');
            const x = handle.x + handle.width / 2, y = handle.y + handle.height / 2;
            const initial = await header.evaluate(el => el.getBoundingClientRect().width);
            await page.mouse.move(x,y);
            await page.mouse.down();
            await page.mouse.move(x+45,y,{steps:10});
            await page.mouse.up();
            const resized = await header.evaluate(el => el.getBoundingClientRect().width);
            expect(resized).toBeGreaterThan(initial+30);
            await search.fill('連江縣');
            await expect(toolbar.getByText(/输入已修改/)).toBeVisible();
            expect(await header.evaluate(el => el.getBoundingClientRect().width)).toBe(resized);
        } else {
            await search.fill('連江縣');
        }
        const compactLevel = await page.evaluate(() => innerWidth < 900)
            ? await page.getByRole('button',{name:/^筛选层级：/}).first().innerText() : undefined;
        const dialog = await openGeoFilter(page,'层级');
        if (compactLevel) {
            await expect(dialog.getByRole('combobox',{name:'值',exact:true})).toHaveText(compactLevel);
        }
        await dialog.getByRole('combobox',{name:'值',exact:true}).click();
        await page.getByRole('option',{name:'地方行政区(香港)',exact:true}).click();
        await dialog.getByRole('button',{name:'取消'}).click();
        await expect(search).toHaveValue('連江縣');
        await search.fill('');
        await openGeoFilter(page,'层级');
        await dialog.getByRole('combobox',{name:'值',exact:true}).click();
        await expect(page.getByRole('option')).toHaveText(['省级','地市级','区县级','区域(香港)','地方行政区(香港)','区域(澳门)','堂区(澳门)','县市(台湾)','乡镇市区(台湾)']);
        await page.getByRole('option',{name:'省级',exact:true}).click();
        if (screenshot) await page.screenshot({path:test.info().outputPath('columns-level.png'),mask});
        expect(requests).toHaveLength(0);
        await dialog.getByRole('button',{name:'应用'}).click();
        await expect(toolbar.getByRole('button',{name:'层级 等于 省级'})).toBeVisible();
        await expect(page.getByRole('link', { name: '北京市', exact: true })).toBeVisible();
        expect(requests).toHaveLength(1);
        await search.fill('尚未提交');
        await page.getByRole('button', { name: /下一页/ }).click();
        const trigger = page.getByRole('row').filter({ has: page.getByRole('link', { name: '云南省', exact: true }) })
            .getByRole('link', { name: '查看下级' });
        await expect(trigger).toBeVisible();
        await expect(search).toHaveValue('尚未提交');
        if (screenshot) await page.screenshot({path: test.info().outputPath('geo138-list.png'), mask});
        const source = new URL(page.url()).searchParams;
        expect(source.get('page')).toBe('2');
        expect(source.has('q')).toBe(false);
        await trigger.press('Enter');
        await expect(page.getByText('云南省的直接下级', { exact: true })).toBeVisible();
        await expect(search).toHaveValue('');
        const child = page.getByRole('link', { name: '昆明市', exact: true });
        const childScope = new URL(page.url()).searchParams.get('parentAreaId');
        await expect(child).toBeVisible();
        if (screenshot) await page.screenshot({path: test.info().outputPath('geo138-children.png'), mask});
        await child.press('Enter');
        await expect(geoField(page, '国家/地区')).toHaveText('中国');
        await expect(geoField(page, '路径')).toContainText('中国 / 云南省');
        if (screenshot) await page.screenshot({path: test.info().outputPath('geo138-detail.png'), mask});
        await page.getByRole('link', { name: '返回列表' }).press('Enter');
        await expect(child).toBeFocused();
        expect(new URL(page.url()).searchParams.get('parentAreaId')).toBe(childScope);
        await page.getByRole('link', { name: '返回上次列表' }).press('Enter');
        await expect(trigger).toBeFocused();
        await expect(trigger).toBeInViewport();
        const returned = new URL(page.url()).searchParams;
        returned.delete('selected');
        expect([...returned].sort()).toEqual([...source].sort());
        await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
    } finally {
        page.off('request', observe);
    }
}
