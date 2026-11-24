import { test, expect, openMoreActions } from './fixtures';
import { withPolicy } from './grants';
import { open, create, read, enter, choose, add, life } from './bu-support';

test('普通更正直接保存：非法名称不落库，合法更正及清空英文不依赖辅助接口', async ({page}) => {
  await open(page); await create(page);
  const before = await read(page);
  let assistanceRequests = 0;
  await page.route('**/api/v1/business-units/*/preview', async route => {
    assistanceRequests++; await route.abort('failed');
  });
  await openMoreActions(page);
  await page.getByRole('menuitem', {name:'更正记录',exact:true}).click();
  const name = page.getByRole('textbox', {name:'中文名称',exact:true});
  await name.fill('甲'.repeat(201));
  await expect(page.getByRole('button', {name:'预览',exact:true})).toHaveCount(0);
  await page.getByRole('button', {name:'保存',exact:true}).click();
  await expect(name).toBeFocused();
  await expect(name).toHaveAttribute('aria-invalid','true');
  expect((await read(page)).headSourceVersion).toBe(before.headSourceVersion);
  await name.fill('测试1-1');
  await page.getByRole('textbox', {name:'英文名称',exact:true}).fill('');
  const request = page.waitForRequest(r => r.method()==='POST' && r.url().endsWith('/correct'));
  await page.getByRole('button', {name:'保存',exact:true}).click();
  expect((await request).postDataJSON().changes).toEqual({name:'测试1-1',nameEn:null});
  await expect(page.locator('.object-form')).toHaveCount(0);
  const after = await read(page);
  expect(after.timeline).toHaveLength(1);
  expect(after.timeline[0].recordId).toBe(before.timeline[0].recordId);
  expect(after.timeline[0].effectiveDate).toBe(before.timeline[0].effectiveDate);
  expect(after.timeline[0].name).toBe('测试1-1');
  expect(after.timeline[0].nameEn).toBeUndefined();
  expect((await read(page,undefined,before.headSourceVersion)).timeline).toEqual(before.timeline);
  expect(assistanceRequests).toBe(0);
});

test('首条前移改名直接更正：没有保留选择，辅助503仍一次保存且不新增原日节点', async ({page}) => {
  await open(page); await create(page,'2026-06-01'); const before = await read(page);
  await page.route('**/api/v1/business-units/*/preview', route => route.fulfill({status:503,json:{message:'辅助暂不可用'}}));
  await enter(page,'更正记录'); let commands=0,joints=0;
  page.on('request',r=>{if(r.method()==='POST'&&r.url().endsWith('/correct'))commands++;if(r.method()==='POST'&&r.url().endsWith('/timeline-revisions'))joints++});
  await expect(page.getByRole('combobox',{name:'保留原日期资料',exact:true})).toHaveCount(0);
  await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-01-01');
  await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('早期资料');
  await expect(page.getByText('操作影响暂时无法加载',{exact:true})).toBeVisible();
  await expect(page.getByRole('button',{name:'预览',exact:true})).toHaveCount(0);
  await page.getByRole('button',{name:'保存',exact:true}).click(); await expect(page.locator('.object-form')).toHaveCount(0);
  const after=await read(page,'2026-01-01'); expect(after.timeline).toHaveLength(1);
  expect(after.timeline[0].recordId).toBe(before.timeline[0].recordId);expect(after.selectedRecord.name).toBe('早期资料');
  expect(commands).toBe(1);expect(joints).toBe(0);expect((await read(page,undefined,before.headSourceVersion)).timeline).toEqual(before.timeline);
});

test('直接新增仍须用户选择业务原因，不借固定目录值绕过必填', async ({page}) => {
  await open(page); await create(page);
  const before = await read(page);
  await page.getByRole('button',{name:/^(新增记录|插入记录)$/,exact:true}).click();
  await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-03-01');
  await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('有明确业务原因的新资料');
  await page.getByRole('button',{name:'保存',exact:true}).click();
  await expect(page.getByRole('alert').first()).toBeVisible();
  expect((await read(page)).headSourceVersion).toBe(before.headSourceVersion);
  await choose(page,'操作原因','名称调整');
  await page.getByRole('button',{name:'保存',exact:true}).click();
  await expect(page.locator('.object-form')).toHaveCount(0);
  expect((await read(page,'2026-03-01')).selectedRecord.name).toBe('有明确业务原因的新资料');
});

for(const action of ['ADD','DEACTIVATE','REACTIVATE','DELETE'])test(`TL164 ${action}自动辅助503仍可直接提交，由Go完成最后校验`,async({page})=>{
 await open(page);await create(page);
 if(action==='REACTIVATE')await life(page,'DEACTIVATE','2026-04-01');
 if(action==='DELETE')await add(page,'2026-06-01','乙','Beta');
 const before=await read(page);let writes=0;
 await page.route('**/api/v1/business-units/*/preview',route=>route.fulfill({status:503,json:{code:'M02_DEPENDENCY_UNAVAILABLE',message:'辅助暂不可用'}}));
 if(action==='ADD')await page.getByRole('button',{name:'新增记录',exact:true}).click();else{await openMoreActions(page);await page.getByRole('menuitem',{name:action==='DEACTIVATE'?'停用业务单元':action==='REACTIVATE'?'重新启用业务单元':'删除此记录',exact:true}).click()}
 if(action!=='DELETE'){
  await page.getByLabel('生效日期').and(page.locator('input')).fill(action==='REACTIVATE'?'2026-05-01':'2026-03-01');await choose(page,'操作原因',action==='ADD'?'名称调整':action==='DEACTIVATE'?'业务终止':'业务恢复');
  if(action!=='DEACTIVATE'){await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('直接办理');await page.getByRole('textbox',{name:'英文名称',exact:true}).fill('Direct')}
 }
 await expect(page.getByText('操作影响暂时无法加载',{exact:true})).toBeVisible();await expect(page.getByRole('button',{name:'预览',exact:true})).toHaveCount(0);
 const label=action==='DEACTIVATE'?'确认停用':action==='REACTIVATE'?'确认重新启用':action==='DELETE'?'确认删除':'保存';await expect(page.getByRole('button',{name:label,exact:true})).toBeEnabled();
 const suffix=action==='DEACTIVATE'?'/deactivate':action==='REACTIVATE'?'/reactivate':action==='DELETE'?'/delete':'/records';page.on('request',r=>{if(r.method()==='POST'&&r.url().endsWith(suffix))writes++});const response=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith(suffix));await page.getByRole('button',{name:label,exact:true}).click();expect((await response).status()).toBe(action==='DELETE'?200:201);await expect(page.locator('.object-form')).toHaveCount(0);expect(writes).toBe(1);
 const after=await read(page);if(action==='DELETE')expect(after.timeline).toHaveLength(1);else expect(after.timeline.at(-1)).toMatchObject({operationType:action==='ADD'?'CHANGE':action,...(action!=='DEACTIVATE'?{name:'直接办理',nameEn:'Direct'}:{})});expect((await read(page,undefined,before.headSourceVersion)).timeline).toEqual(before.timeline);
});

for(const action of ['DEACTIVATE','REACTIVATE'] as const)test(`启停日期权限边界 ${action}可写新日期但不可读历史日期时仍可办理`,async({page})=>{
 await open(page);await create(page);if(action==='REACTIVATE')await life(page,'DEACTIVATE','2026-04-01');const before=await read(page);const date=action==='DEACTIVATE'?'2026-03-01':'2026-05-01';
 await withPolicy(page,async(where,mutate)=>{
  mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'effectiveDate') WHERE ${where} AND action IN ('bu.read','bu.history.read')`);
  await page.reload();await page.route('**/api/v1/business-units/*/preview',route=>route.fulfill({status:503,json:{message:'辅助暂不可用'}}));
  await openMoreActions(page);await page.getByRole('menuitem',{name:action==='DEACTIVATE'?'停用业务单元':'重新启用业务单元',exact:true}).click();
  await page.getByLabel('生效日期').and(page.locator('input')).fill(date);await choose(page,'操作原因',action==='DEACTIVATE'?'业务终止':'业务恢复');await expect(page.getByText('操作影响暂时无法加载',{exact:true})).toBeVisible();
  const response=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith(action==='DEACTIVATE'?'/deactivate':'/reactivate'));await page.getByRole('button',{name:action==='DEACTIVATE'?'确认停用':'确认重新启用',exact:true}).click();expect((await response).status()).toBe(201);await expect(page.locator('.object-form')).toHaveCount(0);
 });
 const after=await read(page);expect(after.timeline.at(-1)).toMatchObject({operationType:action,effectiveDate:date,name:'甲',nameEn:'Alpha'});expect((await read(page,undefined,before.headSourceVersion)).timeline).toEqual(before.timeline);
});
