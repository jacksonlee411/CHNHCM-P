import { deferAutomaticConfirmation } from './fixtures';
import {test,expect,openMoreActions,type Page} from './fixtures';
import {open,create,add,life,enter,choose,save,select,read} from './bu-support';

async function correctFirst(page:Page){
 await select(page,'2026-06-01');await openMoreActions(page);
 for(const name of ['补全首条历史','补齐首条历史'])await expect(page.getByRole('menuitem',{name,exact:true})).toHaveCount(0);
 await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();
 await expect(page.getByRole('combobox',{name:'保留原日期资料',exact:true})).toHaveCount(0);
 await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-01-01');await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('早');
 return save(page);
}
async function startAdd(page:Page,label='新增记录'){
 await page.getByRole('button',{name:label,exact:true}).click();await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-06-01');await choose(page,'操作原因','名称调整');
}
async function submitAdd(page:Page,status=201){const response=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/records'));await page.getByRole('button',{name:'保存',exact:true}).click();const r=await response;expect(r.status()).toBe(status);if(status<300)await expect(page.locator('.object-form')).toHaveCount(0);return r.json()}

for(const later of [false,true])test(`TL302303 首条更正后独立${later?'插入':'新增'}原日期资料，每次保存可查且不自动复制`,async({page})=>{
 await open(page);await create(page,'2026-06-01');if(later){await life(page,'DEACTIVATE','2026-07-01');await life(page,'REACTIVATE','2026-08-01','丙','Gamma');await add(page,'2026-10-01','丁','Delta')}
 const original=await read(page);const corrected=await correctFirst(page);const intermediate=await read(page);
 expect(intermediate.timeline).toHaveLength(original.timeline.length);expect(intermediate.timeline[0]).toMatchObject({recordId:original.timeline[0].recordId,effectiveDate:'2026-01-01',name:'早',operationType:'CREATE'});expect(intermediate.timeline.some((r:any)=>r.effectiveDate==='2026-06-01')).toBe(false);
 expect((await read(page,'2026-06-01')).selectedRecord.name).toBe('早');await page.reload();await expect(page.getByRole('button',{name:'更多操作',exact:true})).toBeEnabled();await select(page,'2026-01-01');
 await startAdd(page,later?'插入记录':'新增记录');await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('甲');
 const added=await submitAdd(page);expect(added.submissionId).not.toBe(corrected.submissionId);expect(added.sourceVersion).not.toBe(corrected.sourceVersion);expect(added.targetRecordId).not.toBe(original.timeline[0].recordId);
 expect((await read(page,'2026-05-31')).selectedRecord.name).toBe('早');expect((await read(page,'2026-06-01')).selectedRecord.name).toBe('甲');
 if(later){expect((await read(page,'2026-07-01')).selectedRecord.name).toBe('甲');expect((await read(page,'2026-08-01')).selectedRecord.nameEn).toBe('Gamma');expect((await read(page,'2026-10-01')).selectedRecord.name).toBe('丁')}
 expect((await read(page,undefined,original.headSourceVersion)).timeline).toEqual(original.timeline);expect((await read(page,undefined,corrected.sourceVersion)).timeline).toEqual(intermediate.timeline);
 await page.getByRole('tab',{name:'变更日志',exact:true}).click();await expect(page.locator('.change-operation').first()).toContainText('新增');await expect(page.locator('.change-operation').nth(1)).toContainText('更正记录');
});

test('TL304305 第二步取消、非法名称及同值均不影响第一步，重新填写后可独立保存',async({page})=>{
 await open(page);await create(page,'2026-06-01');const corrected=await correctFirst(page);const intermediate=await read(page);
 await startAdd(page);await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('待办资料');await page.getByRole('button',{name:'取消编辑',exact:true}).click();await page.getByRole('button',{name:'放弃修改',exact:true}).click();
 expect((await read(page)).headSourceVersion).toBe(corrected.sourceVersion);
 await startAdd(page);await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('😀');await submitAdd(page,400);expect((await read(page)).timeline).toEqual(intermediate.timeline);
 await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('早');const unchanged=await submitAdd(page,200);expect(unchanged.outcome).toBe('no_change');expect((await read(page)).headSourceVersion).toBe(corrected.sourceVersion);
 await startAdd(page);await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('甲');await submitAdd(page);expect((await read(page,'2026-06-01')).selectedRecord.name).toBe('甲');expect((await read(page,undefined,corrected.sourceVersion)).timeline).toEqual(intermediate.timeline);
});

test('TL306 第二步已保存而响应丢失，查回只恢复第二步且第一步原结果可查',async({page})=>{
  await deferAutomaticConfirmation(page);
 await open(page);await create(page,'2026-06-01');const corrected=await correctFirst(page);const intermediate=await read(page);let saved:any;
 await page.route('**/api/v1/business-units/*/records',async route=>{if(route.request().method()!=='POST'){await route.continue();return}const response=await route.fetch();expect(response.status()).toBe(201);saved=await response.json();await route.abort('failed')});
 await startAdd(page);await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('甲');await page.getByRole('button',{name:'保存',exact:true}).click();
 await expect(page.getByRole('button',{name:'查看保存结果',exact:true})).toBeVisible();await page.getByRole('button',{name:'查看保存结果',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);
 expect((await read(page)).headSourceVersion).toBe(saved.sourceVersion);expect((await read(page)).timeline).toHaveLength(2);expect((await read(page,undefined,corrected.sourceVersion)).timeline).toEqual(intermediate.timeline);
 const restored=await page.evaluate(async(key)=>{const s=await(await fetch('/api/v1/session')).json();return(await fetch('/api/v1/organization/submissions/'+key+'/resolve',{method:'POST',headers:{'X-CSRF-Token':s.csrf}})).json()},corrected.submissionId);expect(restored.sourceVersion).toBe(corrected.sourceVersion);expect(restored.selectedRecordId).toBe(intermediate.timeline[0].recordId);
});
