import { checkMiddleDelete } from './delete-history-support';
import { desktopViewports } from './desktop-viewports.mjs';
import { readFileSync } from "node:fs";
import { randomUUID } from 'node:crypto';
import { test, expect, openMoreActions, type Page } from './fixtures';
import { open, select, choose } from './bu-support';
import { checkAdoptedLabelOnPairDelete, checkPairPeriod, request } from './dated-object-support';
import {addressField,menu,create,detail,root} from './location204-support';
test.describe.configure({timeout:90000});
test('地点204 地区连续选择、上级路径、全局组合搜索与已选回填', async ({page}) => {
 await open(page); await menu(page); await page.getByRole('link',{name:'新建',exact:true}).click();
 await page.locator('#location-effectiveDate').fill('2020-01-01');
 await choose(page,'国家/地区','中国');
 const control=page.getByRole('combobox',{name:'行政区划',exact:true});
 await control.click(); await expect(page.getByRole('option')).toHaveCount(31);
 await expect(page.getByRole('button',{name:'下一页区域',exact:true})).toHaveCount(0);
 await page.getByRole('option',{name:'中国 / 广东省',exact:true}).click();
 await expect(control).toHaveAttribute('aria-expanded','true');
 await page.getByRole('option',{name:'中国 / 广东省 / 深圳市',exact:true}).click();
 await expect(control).toHaveAttribute('aria-expanded','true');
 await expect(page.getByRole('navigation',{name:'地区选择路径',exact:true})).toContainText('深圳市');
 const {mkdirSync}=await import('node:fs');mkdirSync('.local/location-region-picker',{recursive:true});
 await page.screenshot({path:'.local/location-region-picker/desktop-browse.png',mask:[page.locator('.app-account-name')]});
 await page.getByRole('button',{name:'返回上一级',exact:true}).click();
 await expect(page.getByRole('option',{name:'中国 / 广东省 / 广州市',exact:true})).toBeVisible();
 await page.getByRole('option',{name:'中国 / 广东省 / 深圳市',exact:true}).click();
 const global=page.waitForResponse(r=>r.url().includes('/geo-areas/candidates?')&&new URL(r.url()).searchParams.get('q')==='北京');
 await control.fill('北京'); const query=new URL((await global).url()); expect(query.searchParams.has('parentAreaId')).toBe(false);
 await expect(page.getByRole('option',{name:'中国 / 北京市',exact:true})).toBeVisible();
 await control.fill('广东省深圳市南山区');
 await page.getByRole('option',{name:'中国 / 广东省 / 深圳市 / 南山区',exact:true}).click();
 await expect(control).toHaveValue(/广东省.*深圳市.*南山区/); await expect(control).toHaveAttribute('aria-expanded','false');
 await expect(control).toBeEnabled(); await page.screenshot({path:'.local/location-region-picker/desktop-selected.png',mask:[page.locator('.app-account-name')]});
 await control.click(); await expect(page.getByRole('option',{name:'中国 / 广东省 / 深圳市 / 南山区',exact:true})).toHaveAttribute('aria-selected','true');
 await control.fill('南山'); await expect(page.getByRole('option',{name:'中国 / 广东省 / 深圳市 / 南山区',exact:true})).toBeVisible();
 await control.fill(''); await expect(page.getByRole('navigation',{name:'地区选择路径',exact:true})).toContainText('深圳市');
 await page.keyboard.press('Escape'); await expect(control).toHaveValue(/南山区/);
 await page.getByRole('textbox',{name:'编码',exact:true}).fill(`REGION-${randomUUID().slice(0,8)}`);
 await page.getByRole('textbox',{name:'地点名称',exact:true}).fill('地区选择验证地点');
 await page.getByRole('textbox',{name:'详细地址',exact:true}).fill('地区控件验证一号');
 await page.getByRole('button',{name:'保存',exact:true}).click(); await expect(page.locator('.object-form')).toHaveCount(0);
 const saved=await detail(page); expect(saved.selectedRecord.geoAreaPath.at(-1).name).toBe('南山区');
 await openMoreActions(page); await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();
 await expect(control).toHaveValue(/广东省.*深圳市.*南山区/); await control.click();
 await expect(page.getByRole('option',{name:'中国 / 广东省 / 深圳市 / 南山区',exact:true})).toHaveAttribute('aria-selected','true');
 await page.keyboard.press('Escape'); await page.getByRole('button',{name:'取消编辑',exact:true}).click();
});

test('地点204 地区搜索滚动追加和键盘连续选择', async ({page}) => {
 await open(page); await menu(page); await page.getByRole('link',{name:'新建',exact:true}).click();
 await page.locator('#location-effectiveDate').fill('2020-01-01'); await choose(page,'国家/地区','中国');
 const control=page.getByRole('combobox',{name:'行政区划',exact:true}); await control.fill('区');
 await expect(page.getByRole('option')).toHaveCount(50); const first=await page.getByRole('option').first().getAttribute('aria-label');
 await page.getByRole('listbox').evaluate(element=>{element.scrollTop=element.scrollHeight; element.dispatchEvent(new Event('scroll'));});
 await expect(page.getByRole('option')).toHaveCount(100); expect(await page.getByRole('option').first().getAttribute('aria-label')).toBe(first);
 await control.fill('海淀'); await expect(page.getByRole('option',{name:'中国 / 北京市 / 海淀区',exact:true})).toBeVisible();
 const restarted=page.waitForResponse(response=>{const url=new URL(response.url());return url.pathname==='/api/v1/geo-areas/candidates'&&url.searchParams.get('q')==='区';});
 await control.fill('区'); expect(new URL((await restarted).url()).searchParams.has('cursor')).toBe(false);
 await expect(page.getByRole('option')).toHaveCount(50); expect(await page.getByRole('option').first().getAttribute('aria-label')).toBe(first);
 await page.keyboard.press('Escape');
 for(const [width,height] of desktopViewports) {
  await page.setViewportSize({width,height}); await control.click();
  await page.getByRole('option',{name:'中国 / 北京市',exact:true}).click();
  await expect(control).toHaveAttribute('aria-expanded','true'); await expect(page.getByRole('option',{name:'中国 / 北京市 / 东城区',exact:true})).toBeVisible();
  const {mkdirSync:makeFolder}=await import('node:fs');makeFolder('.local/location-region-picker',{recursive:true});await page.screenshot({path:`.local/location-region-picker/browse-${width}-${height}.png`,mask:[page.locator('.app-account-name')]});
  await control.press('ArrowDown'); await control.press('Enter'); await expect(control).toHaveAttribute('aria-expanded','false'); await expect(control).toHaveValue(/北京市/);
  await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
  const {mkdirSync}=await import('node:fs');const folder='.local/location-region-picker';mkdirSync(folder,{recursive:true});
  await page.screenshot({path:`${folder}/selected-${width}-${height}.png`,mask:[page.locator('.app-account-name')]});
  await choose(page,'国家/地区','澳门'); await expect(control).toHaveCount(0); await choose(page,'国家/地区','中国');
 }
});
for (const [round, [region, area]] of ([['中国', '海淀'], ['香港', '中西區'], ['澳门', undefined], ['台湾', '松山區'], ['中国','济源'], ['香港','深水埗'], ['澳门',undefined], ['台湾','板橋']] as const).entries()) {
    test(`地点204 ${region}${round}正常菜单创建和原地址维护`, async ({ page }) => {
        await open(page);
        await create(page, region, area);
        const first = await detail(page);
        const regionPath = (first.selectedRecord.geoAreaPath ?? []).map((node:any) => node.name).join('');
        await expect(page.getByRole('heading', { name: '地址', exact: true })).toHaveCount(0);
        await expect(addressField(page, '地点名称')).toHaveText(first.selectedRecord.name);
        await expect(addressField(page, '国家/地区')).toHaveText(first.selectedRecord.countryName);
        await expect(addressField(page, '完整地址')).toHaveText(regionPath + '测试街一号');
        await expect(page.getByRole('textbox', { name: '完整地址', exact: true })).toHaveCount(0);
        expect(first.selectedRecord.postalCode).toBe('00100');
        await expect(page.getByText('地址采用日期', { exact: true })).toHaveCount(0);
        await expect(page.getByText('地址补充', { exact: true })).toHaveCount(0);
        expect(first.selectedRecord).not.toHaveProperty('addressExtra');
        expect(first.selectedRecord.adoptionDate).toBe('2020-01-01');
        await openMoreActions(page);
        await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
        await expect(page.getByRole('textbox', { name: '详细地址', exact: true })).toHaveValue('测试街一号');
        await expect(page.getByText(/地址采用日期/)).toHaveCount(0);
        await expect(page.getByRole('textbox', { name: '地址补充', exact: true })).toHaveCount(0);
        await page.getByRole('textbox', { name: '详细地址', exact: true }).fill('测试街二号');
        await expect(addressField(page, '完整地址')).toHaveText(regionPath + '测试街二号');
        await page.getByRole('button', { name: '保存', exact: true }).click();
        await expect(page.locator('.object-form')).toHaveCount(0);
        const after = await detail(page);
        expect(after.selectedRecord.addressLine).toBe('测试街二号');
        await expect(addressField(page, '完整地址')).toHaveText(regionPath + '测试街二号');
        expect(after.selectedRecord.adoptionDate).toBe('2020-01-01');
        const old = await request(page, 'GET', `/api/v1/locations/${first.id}?sourceVersion=${first.headSourceVersion}`);
        expect(old.body.selectedRecord.addressLine).toBe('测试街一号');
        await page.getByRole('tab', { name: /变更日志/ }).click();
        await expect(page.locator('#location-history-panel').getByText('测试街二号', { exact: true }).first()).toBeVisible();
        if (area) await expect(page.locator('#location-history-panel').getByRole('cell', { name: '行政区划', exact: true }).first()).toBeVisible();
        await page.goto(`${root}/${first.id}?sourceVersion=${first.headSourceVersion}`);
        await expect(addressField(page, '完整地址')).toHaveText(regionPath + '测试街一号');
        await expect(page.getByRole('textbox', { name: '完整地址', exact: true })).toHaveCount(0);
    });
}
test('地点204 完整地址无分隔符、准确采用与字段权限', async ({ page }) => {
    test.setTimeout(180000);
    const { withPolicy } = await import('./grants');
    await open(page);
    await create(page, '中国', '西乡塘');
    const original = await detail(page), areaId = original.selectedRecord.geoAreaId;
    const prefix = '广西壮族自治区南宁市西乡塘区';
    await openMoreActions(page);
    await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
    const address = page.getByRole('textbox', { name: '详细地址', exact: true });
    await address.fill('广西大学宿舍211');
    await expect(addressField(page, '完整地址')).toHaveText(prefix + '广西大学宿舍211');
    const { mkdirSync } = await import('node:fs');
    mkdirSync('.local/location-address-presentation', { recursive: true });
    await addressField(page, '完整地址').scrollIntoViewIfNeeded();
    await page.screenshot({ path: '.local/location-address-presentation/form.png', mask: [page.locator('.app-account-name')] });
    const longAddress = '长'.repeat(1000) + '𠀀';
    await address.fill(longAddress);
    await expect(addressField(page, '完整地址')).toHaveText(prefix + longAddress);
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
    await page.screenshot({ path: '.local/location-address-presentation/long-address.png', mask: [page.locator('.app-account-name')] });
    await page.setViewportSize({ width: 1440, height: 900 });
    await address.fill('A / B 栋 2-11');
    await expect(addressField(page, '完整地址')).toHaveText(prefix + 'A / B 栋 2-11');
    const saving = page.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith('/correct'));
    await page.getByRole('button', { name: '保存', exact: true }).click();
    expect((await saving).request().postDataJSON().changes).toEqual({ addressLine: 'A / B 栋 2-11' });
    await expect(page.locator('.object-form')).toHaveCount(0);
    await expect(addressField(page, '完整地址')).toHaveText(prefix + 'A / B 栋 2-11');
    await expect(addressField(page, '地点名称')).toHaveText(original.selectedRecord.name);
    await page.screenshot({ path: '.local/location-address-presentation/detail.png', mask: [page.locator('.app-account-name')] });
    const directory = (await request(page, 'GET', `/api/v1/geo-areas/${areaId}`)).body;
    const rename = async (name: string) => {
        const current = (await request(page, 'GET', `/api/v1/geo-areas/${areaId}`)).body;
        const saved = await request(page, 'POST', `/api/v1/geo-areas/${areaId}/records/${current.selectedRecord.recordId}/correct`, {
            expectedSourceVersion: current.headSourceVersion, expectedRecordRevision: current.selectedRecord.revision, changes: { name },
        }, randomUUID());
        expect(saved.status).toBe(200);
    };
    await rename('西乡塘区（目录更名）');
    try {
        await page.reload();
        await expect(addressField(page, '完整地址')).toHaveText(prefix + 'A / B 栋 2-11');
        await openMoreActions(page);
        await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
        await expect(addressField(page, '完整地址')).toHaveText(prefix + 'A / B 栋 2-11');
        await page.getByRole('button', { name: '取消编辑', exact: true }).click();
    } finally { await rename(directory.selectedRecord.name); }
    for (const field of ['addressLine', 'geoAreaPath', 'geoAreaId']) {
        await withPolicy(page, async (where, mutate) => {
            mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'${field}') WHERE ${where} AND action='location.read'`);
            await page.reload();
            await expect(page.getByRole('heading', { level: 1 })).toContainText(original.code);
            await expect(addressField(page, '完整地址')).toHaveCount(0);
        });
    }
    await withPolicy(page, async (where, mutate) => {
        mutate(`DELETE FROM access.subject_grant WHERE ${where} AND action='geoarea.history.read'`);
        await page.reload();
        await expect(addressField(page, '完整地址')).toHaveText('地址信息无读取权限');
        await expect(addressField(page, '详细地址')).toHaveText('A / B 栋 2-11');
        await expect(page.getByText(prefix + 'A / B 栋 2-11', { exact: true })).toHaveCount(0);
    });
});

test('地点204 完整时间线、启停、固定期间和查询者菜单', async ({ page }) => {
    test.setTimeout(180000);
    await open(page);
    await create(page, '澳门');
    await page.getByRole('button', { name: '新增记录', exact: true }).click();
    await page.locator('#record-date').fill('2021-01-01');
    await page.getByRole('textbox', { name: '地点名称', exact: true }).fill('地点后继独立资料');
    await choose(page, '操作原因', '地址调整');
    await page.getByRole('button', { name: '保存', exact: true }).click();
    await expect(page.locator('.object-form')).toHaveCount(0);
    await select(page, '2021-01-01');
    await openMoreActions(page);
    await page.getByRole('menuitem', { name: '停用地点', exact: true }).click();
    await page.locator('#record-date').fill('2022-01-01');
    await page.getByRole('button', { name: '确认停用', exact: true }).click();
    await expect(page.locator('.object-form')).toHaveCount(0);
    await select(page, '2022-01-01');
    await openMoreActions(page);
    await page.getByRole('menuitem', { name: '重新启用地点', exact: true }).click();
    await page.locator('#record-date').fill('2023-01-01');
    await page.getByRole('button', { name: '确认重新启用', exact: true }).click();
    await expect(page.locator('.object-form')).toHaveCount(0);
    const after = await detail(page);
    expect(after.timeline).toHaveLength(4);
    expect(after.selectedRecord.adoptionDate).toBe('2023-01-01');
    await openMoreActions(page);
    await page.getByRole('menuitem', {name:'更正记录',exact:true}).click();
    await page.getByLabel('生效日期',{exact:true}).and(page.locator('input')).fill('2023-02-01');
    await page.getByRole('textbox',{name:'邮政编码',exact:true}).fill('00200');
    await page.getByRole('button',{name:'保存',exact:true}).click();
    await expect(page.locator('.object-form')).toHaveCount(0);
    expect((await detail(page)).selectedRecord.adoptionDate).toBe('2023-02-01');
    const remove=async()=>{await openMoreActions(page);await page.getByRole('menuitem',{name:'删除此记录',exact:true}).click();await page.getByRole('button',{name:'确认删除',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0)};
    await remove();
    await select(page,'2022-01-01');await remove();
    await select(page,'2021-01-01');await openMoreActions(page);await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();
    await choose(page,'操作类型','停用');await choose(page,'操作原因','地点停用');
    await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);
    expect((await detail(page)).selectedRecord.state).toBe('inactive');
    await openMoreActions(page);await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();
    await choose(page,'操作类型','数据更改');await choose(page,'操作原因','地址调整');
    await page.getByRole('textbox',{name:'详细地址',exact:true}).fill('分类恢复后的独立地址');
    await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);
    expect((await detail(page)).selectedRecord.adoptionDate).toBe('2021-01-01');
    await remove();await select(page,'2020-01-01');expect((await detail(page)).timeline).toHaveLength(1);
    await page.getByRole('tab',{name:'变更日志',exact:true}).click();await page.getByRole('button',{name:'变更字段列菜单',exact:true}).click();await page.getByRole('menuitem',{name:'筛选…',exact:true}).click();
    await page.getByRole('dialog').getByRole('combobox',{name:'值',exact:true}).click();await expect(page.getByRole('option',{name:'状态',exact:true})).toHaveCount(1);await page.getByRole('option',{name:'状态',exact:true}).click();
    const filtered=page.waitForResponse(r=>r.request().method()==='GET'&&r.url().includes('/changes?')&&new URL(r.url()).searchParams.get('filterValue')==='status');
    await page.getByRole('dialog').getByRole('button',{name:'应用',exact:true}).click();expect((await filtered).status()).toBe(200);

});
for (const role of ['admin', 'reader'])
    test(`地点204 ${role}列表国家列和正常菜单`, async ({ page }) => {
        await open(page); const code=await create(page,'澳门'); const saved=await detail(page);
        if(role==='reader'){await page.getByRole('button',{name:'退出',exact:true}).click();await open(page,role);}
        await menu(page);
        await expect(page.getByRole('link', { name: '新建', exact: true }))[role === 'reader' ? 'toBeDisabled' : 'toBeEnabled']();
        await expect(page.getByRole('combobox', { name: '国家/地区', exact: true })).toHaveCount(0);
        await expect(page.getByRole('columnheader',{name:'国家/地区',exact:true})).toBeVisible();
        await page.getByRole('textbox',{name:'搜索编码或名称',exact:true}).fill(code);await page.getByRole('button',{name:'查询',exact:true}).click();
        const row=page.locator(`.MuiDataGrid-row[data-id="${saved.id}"]`);
        await expect(row.locator('[data-field="countryRegionId"]')).toHaveText(saved.selectedRecord.countryName);
        await row.getByRole('link').click();await expect(page.getByRole('heading',{level:1})).toContainText(code);
        await page.getByRole('link',{name:'返回列表',exact:true}).click();await expect(row.getByRole('link')).toBeFocused();
        await expect(row.locator('[data-field="countryRegionId"]')).toHaveText(saved.selectedRecord.countryName);
        const {mkdirSync}=await import('node:fs');mkdirSync('.local/location-country-column',{recursive:true});
        await page.screenshot({path:`.local/location-country-column/list-${role}.png`,mask:[page.locator('.app-account-name')]});
    });

test('地点204 固定期间插入、维护、删除与原来源', async({page})=>{
 await open(page);await create(page,'澳门');const original=await detail(page);
 await openMoreActions(page);await page.getByRole('menuitem',{name:'插入停用与重新启用',exact:true}).click();
 let dialog=page.getByRole('dialog',{name:'插入停用与重新启用',exact:true});
 await dialog.getByLabel('停用日期',{exact:true}).fill('2021-01-01');await dialog.getByLabel('重新启用日期',{exact:true}).fill('2022-01-01');
 await dialog.getByRole('button',{name:'保存',exact:true}).click();await expect(dialog).toHaveCount(0);
 await select(page,'2021-01-01');await openMoreActions(page);await page.getByRole('menuitem',{name:'维护停用与重新启用',exact:true}).click();
 dialog=page.getByRole('dialog',{name:'维护停用与重新启用',exact:true});await dialog.getByLabel('停用日期',{exact:true}).fill('2021-02-01');await dialog.getByLabel('重新启用日期',{exact:true}).fill('2022-02-01');
 await dialog.getByRole('button',{name:'保存',exact:true}).click();await expect(dialog).toHaveCount(0);
 const maintained=await detail(page);expect(maintained.timeline.map((r:any)=>r.effectiveDate)).toEqual(['2020-01-01','2021-02-01','2022-02-01']);
  await checkPairPeriod(page,'locations','location','2022-02-01');
 await select(page,'2021-02-01');await openMoreActions(page);await page.getByRole('menuitem',{name:'删除停用与重新启用',exact:true}).click();
 dialog=page.getByRole('dialog',{name:'删除停用与重新启用',exact:true});await dialog.getByRole('button',{name:'删除两条记录',exact:true}).click();await expect(dialog).toHaveCount(0);
 expect((await detail(page)).timeline).toHaveLength(1);const old=await request(page,'GET',`/api/v1/locations/${original.id}?sourceVersion=${maintained.headSourceVersion}`);expect(old.body.timeline).toHaveLength(3);
});

test('地点204 地址候选迟到、国家切换清理、辅助失败保留普通更正',async({page})=>{
 await open(page);await create(page,'中国','海淀');
 const original=await detail(page),countryID=original.selectedRecord.countryRegionId;
 const oldCountry=(await request(page,'GET',`/api/v1/country-regions/${countryID}`)).body;
 const rename=async(name:string)=>{const current=(await request(page,'GET',`/api/v1/country-regions/${countryID}`)).body;const r=current.selectedRecord;const saved=await request(page,'POST',`/api/v1/country-regions/${countryID}/records/${r.recordId}/correct`,{expectedSourceVersion:current.headSourceVersion,expectedRecordRevision:r.revision,changes:{name}},randomUUID());expect(saved.status).toBe(200)};
 await rename('中国（目录更名验证）');
 try {
 const context=page.waitForResponse(r=>r.url().endsWith('/api/v1/locations/reference-context')&&r.request().method()==='POST');
 await openMoreActions(page);await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();
 expect((await(await context).json()).newAdoption).toBe(false);
 await expect(page.getByRole('combobox',{name:'国家/地区',exact:true})).toHaveText(original.selectedRecord.countryName);
 let release!:()=>void;const waiting=new Promise<void>(r=>{release=r});let entered!:()=>void;const sent=new Promise<void>(r=>{entered=r});
 await page.route('**/api/v1/geo-areas/candidates?**',async route=>{const response=await route.fetch();entered();await waiting;await route.fulfill({response}).catch(()=>{})});
 const control=page.getByRole('combobox',{name:'行政区划',exact:true});await control.fill('海淀');await sent;await page.keyboard.press('Escape');
 await choose(page,'国家/地区','澳门');release();await expect(page.getByRole('combobox',{name:'行政区划',exact:true})).toHaveCount(0);
 await page.getByRole('button',{name:'取消编辑',exact:true}).click();await page.getByRole('button',{name:'放弃修改',exact:true}).click();
 await page.unroute('**/api/v1/geo-areas/candidates?**');await page.route('**/api/v1/locations/reference-context',route=>route.fulfill({status:503,contentType:'application/json',body:JSON.stringify({code:'SERVICE_UNAVAILABLE',message:'测试辅助读取暂不可用'})}));
 await openMoreActions(page);await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();await page.getByRole('textbox',{name:'详细地址',exact:true}).fill('不变引用普通门牌更正');await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);expect((await detail(page)).selectedRecord.addressLine).toBe('不变引用普通门牌更正');expect((await detail(page)).selectedRecord.countryName).toBe(original.selectedRecord.countryName);
 await page.unroute('**/api/v1/locations/reference-context');
 // Stop selection retroactively across the saved adoption day. This is still
 // readable history, not a permission withdrawal. Restore eligibility after
 // the assertion without deleting either directory source.
 const directory=(await request(page,'GET',`/api/v1/country-regions/${countryID}`)).body;
 const previous=directory.timeline.at(-1).effectiveDate;
 const nextDay=(day:string)=>new Date(Date.parse(day+'T00:00:00Z')+86400000).toISOString().slice(0,10);
 const stopDay=previous<'2019-01-01'?'2019-01-01':nextDay(previous),resumeDay=nextDay(stopDay);
 expect(resumeDay<'2020-01-01').toBe(true);
 const countryLifecycle=async(action:'DEACTIVATE'|'REACTIVATE',day:string)=>{const current=(await request(page,'GET',`/api/v1/country-regions/${countryID}`)).body;const result=await request(page,'POST',`/api/v1/country-regions/${countryID}/${action==='DEACTIVATE'?'deactivate':'reactivate'}`,{expectedSourceVersion:current.headSourceVersion,anchorRecordId:current.selectedRecord.recordId,effectiveDate:day,operationType:action,operationReason:action==='DEACTIVATE'?'STOP_SELECTION':'RESUME_SELECTION'},randomUUID());expect(result.status).toBe(201)};
 await countryLifecycle('DEACTIVATE',stopDay);
 try {
  const unavailable=page.waitForResponse(r=>r.url().endsWith('/api/v1/locations/reference-context')&&r.request().method()==='POST');
  const reloaded=page.waitForResponse(r=>r.request().method()==='GET'&&new URL(r.url()).pathname===`/api/v1/locations/${original.id}`);
  await openMoreActions(page);await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();
  const options=await(await unavailable).json();expect(options.newAdoption).toBe(false);expect(options.countries.some((c:any)=>c.id===countryID)).toBe(false);
  expect((await reloaded).status()).toBe(200);
  await expect(page.getByRole('combobox',{name:'国家/地区',exact:true})).toHaveText(original.selectedRecord.countryName);
  await page.getByRole('textbox',{name:'详细地址',exact:true}).fill('目录停选后沿用原采用');
  await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);
  expect((await detail(page)).selectedRecord.countryName).toBe(original.selectedRecord.countryName);
 } finally {await countryLifecycle('REACTIVATE',resumeDay)}
 } finally {await rename(oldCountry.selectedRecord.name)}
});

test('地点204 固定137身份、跨页排除、九列CSV和失权零下载',async({page})=>{
 test.setTimeout(180000);const {readFileSync}=await import('node:fs');const {withPolicy}=await import('./grants');await open(page);await create(page,'澳门');const example=await detail(page);const prefix=`SELECT204-${Date.now()}-`;
 const ids=await page.evaluate(async({country,prefix})=>{const session=await(await fetch('/api/v1/session')).json();const ids=[];for(let i=0;i<137;i++){const response=await fetch('/api/v1/locations',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':session.csrf,'Idempotency-Key':crypto.randomUUID()},body:JSON.stringify({code:prefix+String(i).padStart(3,'0'),name:i===0?'+1+1':`地点选择${i}`,countryRegionId:country,addressLine:'澳门测试街一号',postalCode:'00100',effectiveDate:'2020-01-01',operationType:'CREATE',operationReason:'NEW_LOCATION'})});if(response.status!==201)throw Error(`seed ${i}: ${response.status}`);ids.push((await response.json()).locationId)}return ids},{country:example.selectedRecord.countryRegionId,prefix});expect(ids).toHaveLength(137);
 await menu(page);await page.getByRole('textbox',{name:'搜索编码或名称',exact:true}).fill(prefix);await page.getByRole('button',{name:'查询',exact:true}).click();await expect(page.getByText('1–20 / 137',{exact:true})).toBeVisible();await page.getByRole('checkbox',{name:/全选查询结果/}).check();await expect(page.getByText('已选择137项',{exact:true})).toBeVisible();await page.getByRole('button',{name:'下一页',exact:true}).click();await expect(page.getByText('21–40 / 137',{exact:true})).toBeVisible();await page.getByRole('grid',{name:'地点列表',exact:true}).locator('.MuiDataGrid-row').first().getByRole('checkbox').uncheck();await expect(page.getByText('已选择136项',{exact:true})).toBeVisible();await choose(page,'每页条数','50');await expect(page.getByText('已选择136项',{exact:true})).toBeVisible();
 const downloaded=page.waitForEvent('download');await page.getByRole('button',{name:'导出所选',exact:true}).click();const file=await downloaded;const csv=readFileSync((await file.path())!,'utf8').replace(/^\ufeff/,'').trim().split(/\r?\n/);expect(csv).toHaveLength(137);expect(csv[0].split(',')).toEqual(['编码','名称','英文名','国家/地区','行政区划路径','详细地址','邮编','状态','生效日期']);expect(csv.join('\n')).toContain('00100');expect(csv.join('\n')).toContain("'+1+1");expect(csv.join('\n')).not.toContain(prefix+'020');
 await withPolicy(page,async(where,mutate)=>{let downloads=0;page.on('download',()=>downloads++);mutate(`DELETE FROM access.subject_grant WHERE ${where} AND action='location.export'`);await page.getByRole('button',{name:'导出所选',exact:true}).click();await expect(page.getByRole('toolbar',{name:'所选地点',exact:true})).toHaveCount(0);expect(downloads).toBe(0)});
});

test('地点204 已提交丢响应后的原提交恢复与零重复来源',async({page})=>{
 const {deferAutomaticConfirmation}=await import('./fixtures');await deferAutomaticConfirmation(page);await open(page);await create(page,'澳门');const before=await detail(page);await openMoreActions(page);await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();await page.getByRole('textbox',{name:'详细地址',exact:true}).fill('丢响应后的确切地址');let lost=true;let saved:any;
 await page.route('**/api/v1/locations/*/records/*/correct',async route=>{const response=await route.fetch();if(lost){lost=false;saved=await response.json();await route.abort('failed')}else await route.fulfill({response})});
 await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.getByRole('button',{name:'重试保存',exact:true})).toBeVisible();await page.getByRole('button',{name:'重试保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);const after=await detail(page);expect(after.headSourceVersion).toBe(saved.sourceVersion);expect(after.timeline).toHaveLength(before.timeline.length);expect(after.selectedRecord.addressLine).toBe('丢响应后的确切地址');const history=await request(page,'GET',`/api/v1/locations/${before.id}/changes`);expect(history.body.items).toHaveLength(2);
});

test('地点204 目录失权清理旧标签，后续503不恢复缓存',async({page})=>{
 const {withPolicy}=await import('./grants');await open(page);await create(page,'中国','海淀');await withPolicy(page,async(where,mutate)=>{
  await openMoreActions(page);await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();await expect(page.getByRole('combobox',{name:'行政区划',exact:true})).toHaveValue(/海淀/);mutate(`DELETE FROM access.subject_grant WHERE ${where} AND action LIKE 'geoarea.%'`);
  const control=page.getByRole('combobox',{name:'行政区划',exact:true});
  const candidates='**/api/v1/geo-areas/candidates?**';
  let received!: (status: number) => void, release!: () => void;
  const denied = new Promise<number>(resolve => { received = resolve; });
  const closed = new Promise<void>(resolve => { release = resolve; });
  await page.route(candidates, async route => {
    if (new URL(route.request().url()).searchParams.get('q') !== '海淀') return route.continue();
    const response = await route.fetch();
    received(response.status());
    await closed;
    await route.fulfill({response}).catch(() => {});
  });
  try {
    await control.fill('海淀');
    expect(await denied).toBe(403);
    await page.keyboard.press('Escape');
    await expect(control).toHaveAttribute('aria-expanded','false');
    release();
    await expect(control).not.toHaveValue(/海淀/);
    await expect(addressField(page,'完整地址')).not.toContainText('海淀');
  } finally { release(); await page.unroute(candidates); }
  await page.route('**/api/v1/geo-areas/candidates?**',route=>route.fulfill({status:503,contentType:'application/json',body:JSON.stringify({code:'UNAVAILABLE',message:'读取暂不可用'})}));
  const unavailableArea=page.waitForResponse(r=>r.url().includes('/geo-areas/candidates?')&&new URL(r.url()).searchParams.get('q')==='昌平'&&r.status()===503);
  await control.fill('昌平');await unavailableArea;await expect(page.getByRole('combobox',{name:'行政区划',exact:true})).not.toHaveValue(/海淀/);await page.keyboard.press('Escape');await page.getByRole('textbox',{name:'详细地址',exact:true}).fill('无区域读取权的普通更正');await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);expect((await detail(page)).selectedRecord.addressLine).toBe('无区域读取权的普通更正');
 });
 await page.unroute('**/api/v1/geo-areas/candidates?**');
 await withPolicy(page,async(where,mutate)=>{
  await page.reload();await openMoreActions(page);
  // The saved country label appears before the initial directory request.
  // Finish that read before revoking access and changing its input date.
  const initialContext=page.waitForResponse(r=>r.url().endsWith('/api/v1/locations/reference-context')&&r.request().method()==='POST');
  await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();
  expect((await(await initialContext).json()).countries.some((country:any)=>country.name==='中国')).toBe(true);
  await expect(page.getByRole('combobox',{name:'行政区划',exact:true})).toBeEnabled();
  await expect(page.getByRole('combobox',{name:'国家/地区',exact:true})).toHaveText('中国');
  mutate(`DELETE FROM access.subject_grant WHERE ${where} AND action='country.read'`);
  const refreshed=page.waitForResponse(r=>r.url().endsWith('/api/v1/locations/reference-context')&&r.request().method()==='POST'&&r.request().postDataJSON().newEffectiveDate==='2020-02-01');
  await page.locator('#record-date').fill('2020-02-01');const deniedContext=await refreshed;expect(deniedContext.status()).toBe(200);expect((await deniedContext.json()).countries).toHaveLength(0);
  await expect(page.getByRole('combobox',{name:'国家/地区',exact:true})).not.toHaveText('中国');
  await expect(page.getByRole('combobox',{name:'行政区划',exact:true})).not.toHaveValue(/海淀/);
  await page.route('**/api/v1/locations/reference-context',route=>route.fulfill({status:503,contentType:'application/json',body:JSON.stringify({code:'UNAVAILABLE',message:'读取暂不可用'})}));
  await page.locator('#record-date').fill('2020-01-01');await expect(page.getByRole('alert').filter({hasText:'读取暂不可用'})).toBeVisible();
  await expect(page.getByRole('combobox',{name:'国家/地区',exact:true})).not.toHaveText('中国');
  await page.getByRole('textbox',{name:'详细地址',exact:true}).fill('无国家读取权的普通更正');await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);
  expect((await detail(page)).selectedRecord.addressLine).toBe('无国家读取权的普通更正');
 });
});

test('140 地点候选两页与固定吉林目录逐项一致，策略变化清空并重读，撤权清空', async ({page}) => {
  const {withPolicy} = await import('./grants');
  const catalog = JSON.parse(readFileSync('internal/contexts/organization/infra/geoareaseed/areas.json', 'utf8'));
  const expected = catalog.areas.filter((a: {countryKey:string;code:string}) => a.countryKey === 'base.cn' && a.code.startsWith('22'))
    .map((a: {code:string}) => a.code).sort();
  expect(expected).toHaveLength(70);
  await open(page);
  await menu(page);
  const count = async () => {
    const response = await request(page, 'GET', '/api/v1/locations');
    expect(response.status).toBe(200);
    return response.body.count;
  };
  const originalCount = await count();
  const writes: string[] = [];
  page.on('request', req => {
    if (req.method() === 'POST' && new URL(req.url()).pathname === '/api/v1/locations') writes.push(req.method());
  });
  await page.getByRole('link', {name:'新建',exact:true}).click();
  await page.locator('#location-effectiveDate').fill('2020-01-01');
  await choose(page, '国家/地区', '中国');
  const picker = page.getByRole('combobox', {name:'行政区划',exact:true});
  const options = page.getByRole('option');
  const more = page.getByRole('button', {name:'加载更多地区'});
  const response = (cursor:boolean) => page.waitForResponse(r => {
    const url = new URL(r.url());
    return url.pathname === '/api/v1/geo-areas/candidates' && url.searchParams.get('q') === '吉林省' && url.searchParams.has('cursor') === cursor;
  });
  const first = response(false);
  await picker.fill('吉林省');
  const firstPage = await (await first).json();
  await expect(options).toHaveCount(50);
  const next = response(true);
  await more.click();
  const secondPage = await (await next).json();
  expect(secondPage.hasMore).toBe(false);
  const all = [...firstPage.items, ...secondPage.items];
  expect(all.map(item => item.code)).toEqual(expected);
  await expect(options).toHaveText(all.map(item => item.path.map((n:{name:string})=>n.name).join(' / ')));

  await picker.fill('长春');
  const again = response(false);
  await picker.fill('吉林省');
  await again;
  for (const restore of [true, false]) {
    await withPolicy(page, async (where, mutate) => {
      const removeNature = `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'nature') WHERE ${where} AND action='geoarea.read'`;
      if (restore) await withPolicy(page, async (_,change) => change(removeNature));
      else mutate(removeNature);
      // Hold only the automatic first-page request to observe the cleared DOM.
      let release!: () => void;
      const released = new Promise<void>(resolve => {release = resolve});
      const pattern = '**/geo-areas/candidates?**';
      await page.route(pattern, async route => {
        if (!new URL(route.request().url()).searchParams.has('cursor')) await released;
        await route.continue();
      });
      const stale = response(true), fresh = response(false);
      try {
        await more.click();
        const fail = await stale;
        expect(fail.status()).toBe(409);
        const error = await fail.json();
        expect(error.code).toBe('M02_CURSOR_CHANGED');
        expect(error).not.toHaveProperty('items');
        await expect(options).toHaveCount(0);
      } finally {release()}
      const res = await fresh;
      expect(res.status()).toBe(200);
      expect(new URL(res.url()).searchParams.toString()).toBe(new URL((await first).url()).searchParams.toString());
      const result = await res.json();
      expect(result.items.map((item:{code:string})=>item.code)).toEqual(expected.slice(0,50));
      expect(result.items.every((item:object) => Object.hasOwn(item,'nature') === restore)).toBe(true);
      await expect(options).toHaveCount(50);
      await page.unroute(pattern);
    });
    // Restoring grants also advances the version. Open a new first page.
    await picker.fill('长春');
    const restart = response(false);
    await picker.fill('吉林省');
    await restart;
  }
  await withPolicy(page, async (where, mutate) => {
    mutate(`DELETE FROM access.subject_grant WHERE ${where} AND action='geoarea.read'`);
    const denied = response(true);
    await more.click();
    const res = await denied;
    expect(res.status()).toBe(403);
    expect(await res.json()).toMatchObject({code:'ACCESS_DENIED'});
    await expect(options).toHaveCount(0);
  });
  await page.keyboard.press('Escape');
  await page.getByRole('link',{name:'取消',exact:true}).click();
  await page.getByRole('button',{name:'放弃修改',exact:true}).click();
  await expect(page).toHaveURL(url => url.pathname === root);
  expect(await count()).toBe(originalCount);
  expect(writes).toEqual([]);
});

test('140 地点成对删除显示准确采用的国家名称', async ({page}) => {
  await open(page);
  await create(page, '澳门');
  await checkAdoptedLabelOnPairDelete(page, 'locations');
});

test('140 地点成对删除显示准确采用的区划路径', async ({page}) => {
  await open(page);
  await create(page, '中国', '西乡塘');
  await checkAdoptedLabelOnPairDelete(page, 'locations', true);
});

test('140 地点删除中段后选中适用记录，查询者可回看删除前原件', async ({page}) => {
  await checkMiddleDelete(page, 'locations', 'ADDRESS_ADJUSTMENT', create, menu);
});
