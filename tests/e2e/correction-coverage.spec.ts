import {test,expect,openMoreActions,type Page} from './fixtures';
import {open,create,add,choose,read,expectSingleClassification} from './bu-support';
import {withPolicy} from './grants';

async function edit(page:Page){await openMoreActions(page);await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();await expect(page.getByRole('button',{name:'保存',exact:true})).toBeVisible()}
async function save(page:Page,status=200){const response=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/correct'));await page.getByRole('button',{name:'保存',exact:true}).click();const r=await response;expect(r.status()).toBe(status);if(status===200)await expect(page.locator('.object-form')).toHaveCount(0);return r.json()}

test('TL298 日期分类组合在自动辅助503后仍直接保存，三面只落一次',async({page})=>{
 await open(page);await create(page,'2025-12-01','Initial');await add(page,'2026-01-01','甲','Alpha');const before=await read(page);let assists=0,commands=0;
 await page.route('**/api/v1/business-units/*/preview',route=>{assists++;return route.fulfill({status:503,json:{code:'DEPENDENCY_UNAVAILABLE',message:'辅助暂不可用'}})});
 page.on('request',r=>{if(r.method()==='POST'&&r.url().endsWith('/correct'))commands++});await edit(page);await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('组合更正');await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-02-01');await choose(page,'操作原因','其他数据调整');
 await expect.poll(()=>assists).toBeGreaterThan(0);await expect(page.getByText('操作影响暂时无法加载',{exact:true})).toBeVisible();await expect(page.getByRole('button',{name:'预览',exact:true})).toHaveCount(0);await expect(page.getByRole('button',{name:'保存',exact:true})).toBeEnabled();await save(page);
 expect(commands).toBe(1);expect((await read(page)).timeline[1]).toMatchObject({recordId:before.timeline[1].recordId,name:'组合更正',effectiveDate:'2026-02-01',operationReason:'OTHER_DATA_ADJUSTMENT'});expect((await read(page,undefined,before.headSourceVersion)).timeline).toEqual(before.timeline);
});

for(const facet of ['名称','日期','分类'] as const)test(`TL299 只有${facet}权限且无联合权仍可从正常菜单更正`,async({page})=>{
 await open(page);await create(page,'2025-12-01','Initial');await add(page,'2026-01-01','甲','Alpha');const before=await read(page);
 await withPolicy(page,async(where,mutate)=>{
  const action=facet==='名称'?'bu.record.correct':facet==='日期'?'bu.record.redate':'bu.record.reclassify';const fields=facet==='名称'?"ARRAY['name','nameEn']":facet==='日期'?"ARRAY['effectiveDate']":"ARRAY['operationType','operationReason']";
  mutate(`DELETE FROM access.subject_grant WHERE ${where} AND action='bu.timeline.revise';UPDATE access.subject_grant SET write_fields='{}' WHERE ${where};UPDATE access.subject_grant SET write_fields=${fields} WHERE ${where} AND action='${action}'`);
  await page.reload();await edit(page);const name=page.getByRole('textbox',{name:'中文名称',exact:true}),date=page.getByLabel('生效日期').and(page.locator('input'));
  await expectSingleClassification(page);
  await expect(page.getByRole('button',{name:'添加关联操作',exact:true})).toHaveCount(0);
  if(facet==='名称'){await expect(name).toBeEnabled();await expect(date).toHaveAttribute('readonly','');await expect(page.getByRole('combobox',{name:'操作原因',exact:true})).toHaveCount(0);await name.fill('仅名称')}
  if(facet==='日期'){await expect(name).toBeDisabled();await expect(date).not.toHaveAttribute('readonly');await expect(page.getByRole('combobox',{name:'操作原因',exact:true})).toHaveCount(0);await date.fill('2026-02-01')}
  if(facet==='分类'){await expect(name).toBeDisabled();await expect(date).toHaveAttribute('readonly','');await expect(page.getByRole('combobox',{name:'操作原因',exact:true})).toBeVisible();await choose(page,'操作原因','其他数据调整')}
  const request=page.waitForRequest(r=>r.method()==='POST'&&r.url().endsWith('/correct'));await save(page);const body=(await request).postDataJSON();expect(Object.keys(body).sort()).toEqual(['expectedSourceVersion','expectedRecordRevision',facet==='名称'?'changes':facet==='日期'?'newEffectiveDate':'classification'].sort());
  const after=await read(page);expect(after.timeline[1]).toMatchObject({name:facet==='名称'?'仅名称':'甲',effectiveDate:facet==='日期'?'2026-02-01':'2026-01-01',operationReason:facet==='分类'?'OTHER_DATA_ADJUSTMENT':'NAME_ADJUSTMENT'});expect((await read(page,undefined,before.headSourceVersion)).timeline).toEqual(before.timeline);
 });
});

for(const revoke of ['读权','写权'] as const)test(`TL300 分类${revoke}撤回不丢名称日期草稿，移除该面后仍可保存`,async({page})=>{
 await open(page);await create(page,'2025-12-01','Initial');await add(page,'2026-01-01','甲','Alpha');await edit(page);await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('保留名称');await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-02-01');
 const ready=page.waitForResponse(r=>r.url().endsWith('/preview')&&r.request().postDataJSON()?.classification?.operationReason==='OTHER_DATA_ADJUSTMENT');await choose(page,'操作原因','其他数据调整');expect((await ready).status()).toBe(200);
 await withPolicy(page,async(where,mutate)=>{
  if(revoke==='写权'){
   mutate(`UPDATE access.subject_grant SET write_fields=array_remove(array_remove(write_fields,'operationType'),'operationReason') WHERE ${where} AND action LIKE '%.reclassify'`);await save(page,403);await expect(page.getByRole('button',{name:'撤销这些字段的修改',exact:true})).toBeVisible();await expectSingleClassification(page);await expect(page.getByRole('combobox',{name:'操作原因',exact:true})).toContainText('其他数据调整');await page.getByRole('button',{name:'撤销这些字段的修改',exact:true}).click();await expectSingleClassification(page);
  }else{
   mutate(`UPDATE access.subject_grant SET read_fields=array_remove(array_remove(read_fields,'operationType'),'operationReason') WHERE ${where} AND action IN ('bu.read','bu.history.read')`);
   await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-02-02');await expect(page.getByRole('button',{name:'业务分类',exact:true})).toHaveCount(0);await expect(page.getByRole('combobox',{name:'操作原因',exact:true})).toHaveCount(0);await expectSingleClassification(page,[]);await expect(page.locator('body')).not.toContainText('其他数据调整');
  }
  await expect(page.getByRole('textbox',{name:'中文名称',exact:true})).toHaveValue('保留名称');await expect(page.getByLabel('生效日期').and(page.locator('input'))).toHaveValue(revoke==='读权'?'2026-02-02':'2026-02-01');
  const request=page.waitForRequest(r=>r.method()==='POST'&&r.url().endsWith('/correct'));await save(page);expect((await request).postDataJSON().classification).toBeUndefined();
 });
 expect((await read(page)).timeline[1]).toMatchObject({name:'保留名称',operationReason:'NAME_ADJUSTMENT'});
});

test('TL301 冲突重读保留日期分类草稿，未编辑名称取并发新值',async({page})=>{
 await open(page);await create(page,'2025-12-01','Initial');await add(page,'2026-01-01','甲','Alpha');await edit(page);await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-02-01');const ready=page.waitForResponse(r=>r.url().endsWith('/preview')&&r.request().postDataJSON()?.classification?.operationReason==='OTHER_DATA_ADJUSTMENT');await choose(page,'操作原因','其他数据调整');expect((await ready).status()).toBe(200);
 expect(await page.evaluate(async()=>{const base='/api/v1/business-units/'+location.pathname.split('/').at(-1),s=await(await fetch('/api/v1/session')).json(),d=await(await fetch(base)).json(),r=d.selectedRecord;return(await fetch(base+'/records/'+r.recordId+'/correct',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':s.csrf,'Idempotency-Key':crypto.randomUUID()},body:JSON.stringify({expectedSourceVersion:d.headSourceVersion,expectedRecordRevision:r.revision,changes:{name:'他人名称',nameEn:'Other'}})})).status})).toBe(200);
 await save(page,409);await page.getByRole('button',{name:'重新核对当前资料',exact:true}).click();await expect(page.getByRole('textbox',{name:'中文名称',exact:true})).toHaveValue('他人名称');await expect(page.getByRole('textbox',{name:'英文名称',exact:true})).toHaveValue('Other');await expect(page.getByLabel('生效日期').and(page.locator('input'))).toHaveValue('2026-02-01');await expect(page.getByRole('combobox',{name:'操作原因',exact:true})).toContainText('其他数据调整');
 const request=page.waitForRequest(r=>r.method()==='POST'&&r.url().endsWith('/correct'));await save(page);expect((await request).postDataJSON().changes).toBeUndefined();expect((await read(page)).timeline[1]).toMatchObject({name:'他人名称',nameEn:'Other',effectiveDate:'2026-02-01',operationReason:'OTHER_DATA_ADJUSTMENT'});
});

test('分类读权清理只作用于更正，新增仍可重新选择业务原因',async({page})=>{
 await open(page);await create(page);await page.getByRole('button',{name:'新增记录',exact:true}).click();await choose(page,'操作原因','名称调整');
 await withPolicy(page,async(where,mutate)=>{
  mutate(`UPDATE access.subject_grant SET read_fields=array_remove(array_remove(read_fields,'operationType'),'operationReason') WHERE ${where} AND action IN ('bu.read','bu.history.read')`);
  await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-06-01');
  await expect(page.getByText('已重读当前资料，请核对后保存。',{exact:true})).toHaveCount(0);
  await expect(page.getByRole('combobox',{name:'操作原因',exact:true})).not.toContainText('名称调整');
  await choose(page,'操作原因','名称调整');await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('新增资料');
  await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);
 });
 expect((await read(page)).timeline[1]).toMatchObject({name:'新增资料',operationType:'CHANGE',operationReason:'NAME_ADJUSTMENT'});
});

for(const hidden of ['operationType','operationReason'] as const)test(`分类${hidden}无读权只展示另一项只读字段`,async({page})=>{
 await open(page);await create(page);
 await withPolicy(page,async(where,mutate)=>{
  mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'${hidden}') WHERE ${where} AND action IN ('bu.read','bu.history.read')`);
  await page.reload();let catalogs=0;page.on('request',r=>{if(r.url().includes('/organization/operation-reasons'))catalogs++});await edit(page);
  await expectSingleClassification(page,[hidden==='operationType'?'操作原因':'操作类型']);
  await expect(page.locator('.object-form').getByRole('combobox')).toHaveCount(0);
  await expect(page.locator('.object-form')).toContainText(hidden==='operationType'?'新设组织':'新建');
  await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('仅改名称');
  const request=page.waitForRequest(r=>r.method()==='POST'&&r.url().endsWith('/correct'));await save(page);expect((await request).postDataJSON().classification).toBeUndefined();expect(catalogs).toBe(0);
 });
 expect((await read(page)).timeline[0]).toMatchObject({name:'仅改名称',operationType:'CREATE',operationReason:'NEW_ORGANIZATION'});
});

test('类型转换草稿撤写权后单组只读显示真实新类型，移除后恢复原值',async({page})=>{
 await open(page);await create(page);await add(page,'2026-06-01','乙');const before=await read(page);await edit(page);
 await choose(page,'操作类型','停用');const ready=page.waitForResponse(r=>r.url().endsWith('/preview')&&r.request().postDataJSON()?.classification?.operationReason==='BUSINESS_CLOSURE');await choose(page,'操作原因','业务终止');expect((await ready).status()).toBe(200);
 await withPolicy(page,async(where,mutate)=>{
  mutate(`UPDATE access.subject_grant SET write_fields=array_remove(array_remove(write_fields,'operationType'),'operationReason') WHERE ${where} AND action LIKE '%.reclassify'`);await save(page,403);
  await expect(page.getByRole('button',{name:'撤销这些字段的修改',exact:true})).toBeVisible();await expectSingleClassification(page);
  const type=page.getByRole('combobox',{name:'操作类型',exact:true}),reason=page.getByRole('combobox',{name:'操作原因',exact:true});
  await expect(type).toContainText('停用');await expect(type).toHaveAttribute('aria-disabled','true');await expect(reason).toContainText('业务终止');await expect(reason).toHaveAttribute('aria-disabled','true');
  await page.getByRole('button',{name:'撤销这些字段的修改',exact:true}).click();await expectSingleClassification(page);await expect(page.locator('.object-form').getByRole('combobox')).toHaveCount(0);
  await expect(page.locator('.object-form')).toContainText('数据更改');await expect(page.locator('.object-form')).toContainText('名称调整');await page.getByRole('button',{name:'取消编辑',exact:true}).click();await expect(page.getByRole('dialog')).toHaveCount(0);
 });
 expect((await read(page)).headSourceVersion).toBe(before.headSourceVersion);
});
