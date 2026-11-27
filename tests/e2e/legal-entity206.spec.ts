import { checkMiddleDelete } from './delete-history-support';
import { randomUUID } from 'node:crypto';
import { test, expect, openMoreActions, deferAutomaticConfirmation } from './fixtures';
import { open, choose, select } from './bu-support';
import { checkAdoptedLabelOnPairDelete, checkPairPeriod, request } from './dated-object-support';
import { create, detail, menu, root } from './legal-entity206-support';

test.describe.configure({ timeout: 90000 });
for (const [country, category, code] of [['中国', '法人组织', 'legal_person'], ['香港', '分支机构', 'branch'], ['台湾', '非法人组织', 'unincorporated'], ['美国', '其他组织', 'other']]) {
  test(`法律实体206 ${category} ${country}正常菜单和独立历史`, async ({ page }) => {
    await open(page); await create(page, country, category); const before = await detail(page);
    expect(before.selectedRecord.subjectType).toBe(code); expect(before.selectedRecord.registrationNumber).toBe('00100');
    await openMoreActions(page); await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
    await expect(page.getByRole('textbox', { name: '登记类型', exact: true })).toHaveValue('登记证');
    await page.getByRole('textbox', { name: '登记号码', exact: true }).fill('000aB');
    await page.getByRole('button', { name: '保存', exact: true }).click(); await expect(page.locator('.object-form')).toHaveCount(0);
    const after = await detail(page); expect(after.selectedRecord.registrationNumber).toBe('000aB'); expect(after.selectedRecord.adoptionDate).toBe('2020-01-01');
    const old = await request(page, 'GET', `/api/v1/legal-entities/${before.id}?sourceVersion=${before.headSourceVersion}`); expect(old.body.selectedRecord.registrationNumber).toBe('00100');
    await page.getByRole('tab', { name: /变更日志/ }).click(); await expect(page.locator('#legalentity-history-panel').getByText('000aB', { exact: true }).first()).toBeVisible();
  });
}

test('法律实体206 登记半对定位、完整清空与未保存保护', async ({ page }) => {
  await open(page); await create(page); const before = await detail(page);
  await openMoreActions(page); await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
  await page.getByRole('textbox', { name: '登记号码', exact: true }).fill('');
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.getByText('登记类型和登记号码须同时填写或同时清空', { exact: false }).first()).toBeVisible();
  await expect(page.getByRole('textbox', { name: '登记类型', exact: true })).toHaveValue('登记证');
  expect((await detail(page)).headSourceVersion).toBe(before.headSourceVersion);
  await page.getByRole('textbox', { name: '登记类型', exact: true }).fill('');
  await page.getByRole('button', { name: '保存', exact: true }).click(); await expect(page.locator('.object-form')).toHaveCount(0);
  expect((await detail(page)).selectedRecord.registrationNumber).toBeNull();
  await openMoreActions(page); await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
  await page.getByRole('textbox', { name: '设立单位说明', exact: true }).fill('未保存草稿');
  await page.getByRole('button', { name: '取消编辑', exact: true }).click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.getByRole('button', { name: '继续编辑', exact: true }).click();
  await expect(page.getByRole('textbox', { name: '设立单位说明', exact: true })).toHaveValue('未保存草稿');
});

test('法律实体206 国家候选失败保留资料并允许不变引用更正', async ({ page }) => {
  await open(page); await create(page); const before = await detail(page);
  await page.route('**/api/v1/legal-entities/reference-context', route => route.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ code: 'SERVICE_UNAVAILABLE', message: '候选暂不可用' }) }));
  await openMoreActions(page); await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
  await expect(page.getByText('候选暂不可用', { exact: true })).toBeVisible();
  await page.getByRole('textbox', { name: '登记号码', exact: true }).fill('000new');
  await page.getByRole('button', { name: '保存', exact: true }).click(); await expect(page.locator('.object-form')).toHaveCount(0);
  const after = await detail(page); expect(after.selectedRecord.countryRegionId).toBe(before.selectedRecord.countryRegionId); expect(after.selectedRecord.registrationNumber).toBe('000new');
});

test('法律实体206 丢响应查回原提交且不重复来源', async ({ page }) => {
  await deferAutomaticConfirmation(page); await open(page); await create(page); const before = await detail(page);
  await openMoreActions(page); await page.getByRole('menuitem', { name: '更正记录', exact: true }).click(); await page.getByRole('textbox', { name: '登记号码', exact: true }).fill('000lost');
  let lost = true, saved: any;
  await page.route('**/api/v1/legal-entities/*/records/*/correct', async route => { const response = await route.fetch(); if (lost) { lost = false; saved = await response.json(); await route.abort('failed'); } else await route.fulfill({ response }); });
  await page.getByRole('button', { name: '保存', exact: true }).click(); await expect(page.getByRole('button', { name: '重试保存', exact: true })).toBeVisible();
  await page.getByRole('button', { name: '重试保存', exact: true }).click(); await expect(page.locator('.object-form')).toHaveCount(0);
  const after = await detail(page); expect(after.headSourceVersion).toBe(saved.sourceVersion); expect(after.selectedRecord.registrationNumber).toBe('000lost');
  expect((await request(page, 'GET', `/api/v1/legal-entities/${before.id}/changes`)).body.total).toBe(2);
});

test('法律实体206 查询者菜单、完整条件导航、选择与CSV', async ({ page }) => {
  await open(page); const prefix = `CSV206-${Date.now()}`;
  const countries = (await request(page, 'GET', '/api/v1/country-regions?q=US')).body.items; const country = countries.find((v: any) => v.code === 'US');
  for (let i = 0; i < 23; i++) {
    const saved = await request(page, 'POST', '/api/v1/legal-entities', { code: `${prefix}-${String(i).padStart(2, '0')}`, name: `${prefix}主体${i}`, countryRegionId: country.id, subjectType: 'branch', registrationType: '登记证', registrationNumber: `00${i}`, effectiveDate: '2020-01-01', operationType: 'CREATE', operationReason: 'NEW_LEGAL_ENTITY' }, randomUUID()); expect(saved.status).toBe(201);
  }
  await page.context().clearCookies(); await open(page, 'reader'); await menu(page); await expect(page.getByRole('link', { name: '新建', exact: true })).toBeDisabled();
  await page.getByRole('textbox', { name: '搜索编码或名称', exact: true }).fill(prefix); await page.getByRole('button', { name: '查询', exact: true }).click();
  await expect(page.getByRole('link', { name: `${prefix}主体0`, exact: true })).toBeVisible(); await page.getByRole('link', { name: `${prefix}主体0`, exact: true }).click();
  await expect(page.getByRole('button', { name: '新增记录', exact: true })).toHaveCount(0);
  await page.getByRole('button', { name: '下一个对象', exact: true }).click(); await expect.poll(async () => (await detail(page)).code).toBe(`${prefix}-01`);
  await page.getByRole('link', { name: '返回列表', exact: true }).click(); await expect(page.getByRole('textbox', { name: '搜索编码或名称', exact: true })).toHaveValue(prefix);
  await page.getByRole('checkbox', { name: '全选查询结果', exact: true }).check();
  const downloaded = page.waitForEvent('download'); await page.getByRole('button', { name: /导出所选/ }).click(); const file = await downloaded;
  expect(file.suggestedFilename()).toMatch(/chnhcm-legal-entities/); const stream = await file.createReadStream(); let csv = ''; for await (const chunk of stream!) csv += chunk;
  expect(csv).toContain('登记号码'); expect(csv).toContain('分支机构'); expect(csv.split('\r\n').filter(Boolean)).toHaveLength(24);
});

test('法律实体206 完整时间线、启停、固定期间和查询者菜单', async ({ page }) => {
    test.setTimeout(180000);
    await open(page);
    await create(page, '澳门');
    await page.getByRole('button', { name: '新增记录', exact: true }).click();
    await page.locator('#record-date').fill('2021-01-01');
    await page.getByRole('textbox', { name: '法律实体名称', exact: true }).fill('法律实体后继独立资料');
    await choose(page, '操作原因', '登记资料调整');
    await page.getByRole('button', { name: '保存', exact: true }).click();
    await expect(page.locator('.object-form')).toHaveCount(0);
    await select(page, '2021-01-01');
    await openMoreActions(page);
    await page.getByRole('menuitem', { name: '停用法律实体', exact: true }).click();
    await page.locator('#record-date').fill('2022-01-01');
    await page.getByRole('button', { name: '确认停用', exact: true }).click();
    await expect(page.locator('.object-form')).toHaveCount(0);
    await select(page, '2022-01-01');
    await openMoreActions(page);
    await page.getByRole('menuitem', { name: '重新启用法律实体', exact: true }).click();
    await page.locator('#record-date').fill('2023-01-01');
    await page.getByRole('button', { name: '确认重新启用', exact: true }).click();
    await expect(page.locator('.object-form')).toHaveCount(0);
    const after = await detail(page);
    expect(after.timeline).toHaveLength(4);
    expect(after.selectedRecord.adoptionDate).toBe('2023-01-01');
    await openMoreActions(page);
    await page.getByRole('menuitem', {name:'更正记录',exact:true}).click();
    await page.getByLabel('生效日期',{exact:true}).and(page.locator('input')).fill('2023-02-01');
    await page.getByRole('textbox',{name:'登记号码',exact:true}).fill('00200');
    await page.getByRole('button',{name:'保存',exact:true}).click();
    await expect(page.locator('.object-form')).toHaveCount(0);
    expect((await detail(page)).selectedRecord.adoptionDate).toBe('2023-02-01');
    const remove=async()=>{await openMoreActions(page);await page.getByRole('menuitem',{name:'删除此记录',exact:true}).click();await page.getByRole('button',{name:'确认删除',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0)};
    await remove();
    await select(page,'2022-01-01');await remove();
    await select(page,'2021-01-01');await openMoreActions(page);await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();
    await choose(page,'操作类型','停用');await choose(page,'操作原因','法律实体停用');
    await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);
    expect((await detail(page)).selectedRecord.state).toBe('inactive');
    await openMoreActions(page);await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();
    await choose(page,'操作类型','数据更改');await choose(page,'操作原因','登记资料调整');
    await page.getByRole('textbox',{name:'设立单位说明',exact:true}).fill('分类恢复后的独立资料');
    await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);
    expect((await detail(page)).selectedRecord.adoptionDate).toBe('2021-01-01');
    await remove();await select(page,'2020-01-01');expect((await detail(page)).timeline).toHaveLength(1);
    await page.getByRole('tab',{name:'变更日志',exact:true}).click();await page.getByRole('button',{name:'变更字段列菜单',exact:true}).click();await page.getByRole('menuitem',{name:'筛选…',exact:true}).click();
    await page.getByRole('dialog').getByRole('combobox',{name:'值',exact:true}).click();await expect(page.getByRole('option',{name:'状态',exact:true})).toHaveCount(1);await page.getByRole('option',{name:'状态',exact:true}).click();
    const filtered=page.waitForResponse(r=>r.request().method()==='GET'&&r.url().includes('/changes?')&&new URL(r.url()).searchParams.get('filterValue')==='status');
    await page.getByRole('dialog').getByRole('button',{name:'应用',exact:true}).click();expect((await filtered).status()).toBe(200);

});
for (const role of ['admin', 'reader'])
    test(`法律实体206 ${role}列表筛选和正常菜单`, async ({ page }) => { await open(page, role); await menu(page); await expect(page.getByRole('link', { name: '新建', exact: true }))[role === 'reader' ? 'toBeDisabled' : 'toBeEnabled'](); await expect(page.getByRole('combobox', { name: '国家/地区', exact: true })).toBeVisible(); });

test('法律实体206 固定期间插入、维护、删除与原来源', async({page})=>{
 await open(page);await create(page,'澳门');const original=await detail(page);
 await openMoreActions(page);await page.getByRole('menuitem',{name:'插入停用与重新启用',exact:true}).click();
 let dialog=page.getByRole('dialog',{name:'插入停用与重新启用',exact:true});
 await dialog.getByLabel('停用日期',{exact:true}).fill('2021-01-01');await dialog.getByLabel('重新启用日期',{exact:true}).fill('2022-01-01');
 await dialog.getByRole('button',{name:'保存',exact:true}).click();await expect(dialog).toHaveCount(0);
 await select(page,'2021-01-01');await openMoreActions(page);await page.getByRole('menuitem',{name:'维护停用与重新启用',exact:true}).click();
 dialog=page.getByRole('dialog',{name:'维护停用与重新启用',exact:true});await dialog.getByLabel('停用日期',{exact:true}).fill('2021-02-01');await dialog.getByLabel('重新启用日期',{exact:true}).fill('2022-02-01');
 await dialog.getByRole('button',{name:'保存',exact:true}).click();await expect(dialog).toHaveCount(0);
 const maintained=await detail(page);expect(maintained.timeline.map((r:any)=>r.effectiveDate)).toEqual(['2020-01-01','2021-02-01','2022-02-01']);
  await checkPairPeriod(page,'legal-entities','legalentity','2022-02-01');
 await select(page,'2021-02-01');await openMoreActions(page);await page.getByRole('menuitem',{name:'删除停用与重新启用',exact:true}).click();
 dialog=page.getByRole('dialog',{name:'删除停用与重新启用',exact:true});await dialog.getByRole('button',{name:'删除两条记录',exact:true}).click();await expect(dialog).toHaveCount(0);
 expect((await detail(page)).timeline).toHaveLength(1);const old=await request(page,'GET',`/api/v1/legal-entities/${original.id}?sourceVersion=${maintained.headSourceVersion}`);expect(old.body.timeline).toHaveLength(3);
});


test('法律实体206 切换采用日后迟到国家候选不覆盖当前输入', async ({ page }) => {
  await open(page); await create(page, '美国');
  let release!: () => void;
  const gate = new Promise<void>(resolve => { release = resolve; });
  let observed!: () => void;
  const started = new Promise<void>(resolve => { observed = resolve; });
  await page.route('**/api/v1/legal-entities/reference-context', async route => {
    const body = route.request().postDataJSON();
    const response = await route.fetch();
    if (body.newEffectiveDate === '2021-01-01') {
      const data = await response.json();
      for (const option of data.countries) option.name = '迟到旧日期候选';
      observed(); await gate;
      await route.fulfill({ response, json: data });
    } else await route.fulfill({ response });
  });
  await openMoreActions(page); await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
  await page.getByRole('textbox', { name: '登记号码', exact: true }).fill('001-draft');
  const date = page.getByLabel('生效日期', { exact: true }).and(page.locator('input'));
  await date.fill('2021-01-01'); await started;
  const updated = page.waitForResponse(r => r.url().endsWith('/legal-entities/reference-context') && r.request().postDataJSON()?.newEffectiveDate === '2022-01-01');
  await date.fill('2022-01-01'); await updated; release();
  await page.getByRole('combobox', { name: '国家/地区', exact: true }).click();
  await expect(page.getByRole('option', { name: '美国', exact: true })).toBeVisible();
  await expect(page.getByRole('option', { name: '迟到旧日期候选', exact: true })).toHaveCount(0);
  await page.keyboard.press('Escape');
  await expect(page.getByRole('textbox', { name: '登记号码', exact: true })).toHaveValue('001-draft');
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.locator('.object-form')).toHaveCount(0);
  const saved = await detail(page);
  expect(saved.selectedRecord.effectiveDate).toBe('2022-01-01');
  expect(saved.selectedRecord.adoptionDate).toBe('2022-01-01');
  expect(saved.selectedRecord.registrationNumber).toBe('001-draft');
});

test('法律实体206 编辑上下文重新核权原标签且候选缺失不清空获权旧名', async ({ page }) => {
  await open(page); await create(page, '美国');
  let restricted = false;
  await page.route('**/api/v1/legal-entities/reference-context', async route => {
    const response = await route.fetch();
    const data = await response.json();
    if (route.request().postDataJSON().action === 'CORRECT') {
      expect(data.newAdoption).toBe(false);
      expect(data.countryName).toBe('美国');
      if (restricted) {
        // Current candidates can remain readable when exact history is denied.
        data.countryName = undefined; data.countryRestricted = true;
      } else {
        // Eligibility may change while the original adopted label stays readable.
        data.countries = []; data.countryRestricted = false;
      }
    }
    await route.fulfill({ response, json: data });
  });
  await openMoreActions(page); await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
  const country = page.getByRole('combobox', { name: '国家/地区', exact: true });
  await expect(country).toHaveText('美国');
  await page.getByRole('textbox', { name: '登记号码', exact: true }).fill('000-preserved');
  await page.getByRole('button', { name: '保存', exact: true }).click(); await expect(page.locator('.object-form')).toHaveCount(0);
  restricted = true;
  await openMoreActions(page); await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
  await expect(country).toHaveText('名称无读取权限');
  await expect(page.getByRole('textbox', { name: '登记号码', exact: true })).toHaveValue('000-preserved');
  await page.getByRole('textbox', { name: '登记号码', exact: true }).fill('000-after-restriction');
  await page.getByRole('button', { name: '保存', exact: true }).click(); await expect(page.locator('.object-form')).toHaveCount(0);
  expect((await detail(page)).selectedRecord.registrationNumber).toBe('000-after-restriction');
});

test('140 法律实体成对删除显示准确采用的登记国家名称', async ({page}) => {
  await open(page);
  await create(page, '美国');
  await checkAdoptedLabelOnPairDelete(page, 'legal-entities');
});

test('140 法律实体删除中段后选中适用记录，查询者可回看删除前原件', async ({page}) => {
  await checkMiddleDelete(page, 'legal-entities', 'REGISTRATION_ADJUSTMENT', create, menu);
});
