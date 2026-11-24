import { deferAutomaticConfirmation } from './fixtures';
import {test,expect,openMoreActions} from './fixtures';
import {open,create,read,add,life,select,choose} from './bu-support';
import {withPolicy} from './grants';
import type {Page,Locator} from '@playwright/test';
async function insert(page:Page){await openMoreActions(page);await page.getByRole('menuitem',{name:'插入停用与重新启用',exact:true}).click();return page.getByRole('dialog',{name:'插入停用与重新启用',exact:true})}
async function fill(dialog:Locator,_page:Page,stop='2026-03-01',resume='2026-04-01'){await dialog.getByLabel('停用日期',{exact:true}).fill(stop);await dialog.getByLabel('重新启用日期',{exact:true}).fill(resume);await choose(dialog,'停用原因','业务终止');await expect(dialog.locator('#pair-reactivationReason')).toHaveText('业务恢复')}
async function submit(page:Page,dialog:Locator,deletePair=false,status=200){const response=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith(deletePair?'/lifecycle-pairs/delete':'/lifecycle-pairs'));await dialog.getByRole('button',{name:deletePair?'删除两条记录':'保存',exact:true}).click();const r=await response;expect(r.status()).toBe(status);if(status===200){await expect(dialog).toHaveCount(0);await expect(page.getByRole('button',{name:'更多操作',exact:true})).toBeEnabled()}return r.json()}
test('固定插入只填两组日期原因，一次保存两条并保留后继及旧来源',async({page})=>{
 await open(page);await create(page);await add(page,'2026-06-01','乙','Beta');await add(page,'2026-10-01','丙','Gamma');await select(page,'2026-01-01');const before=await read(page);const d=await insert(page);
 await expect(d.getByLabel('停用日期',{exact:true})).toHaveValue('2026-01-02');await expect(d.getByLabel('重新启用日期',{exact:true})).toHaveValue('2026-01-03');await d.getByLabel('停用日期',{exact:true}).fill('');await expect(d.getByLabel('重新启用日期',{exact:true})).toHaveAttribute('min','2026-01-03');await d.getByLabel('停用日期',{exact:true}).fill('2026-01-02');await expect(d.getByRole('combobox')).toHaveCount(2);await expect(d.getByRole('textbox',{name:/中文名称|英文名称/})).toHaveCount(0);await expect(d.getByRole('button',{name:'预览',exact:true})).toHaveCount(0);
 await d.getByRole('button',{name:'取消',exact:true}).click();await expect(page.getByRole('dialog')).toHaveCount(0);expect((await read(page)).headSourceVersion).toBe(before.headSourceVersion);
 const dialog=await insert(page);await fill(dialog,page);let commands=0;page.on('request',r=>{if(r.method()==='POST'&&r.url().endsWith('/lifecycle-pairs'))commands++});await submit(page,dialog);const after=await read(page);expect(commands).toBe(1);expect(after.timeline).toHaveLength(5);expect(after.timeline.find((r:any)=>r.effectiveDate==='2026-03-01')).toMatchObject({operationType:'DEACTIVATE',name:'甲',state:'inactive'});expect(after.timeline.find((r:any)=>r.effectiveDate==='2026-04-01')).toMatchObject({operationType:'REACTIVATE',name:'甲',state:'active'});expect(after.timeline.find((r:any)=>r.effectiveDate==='2026-06-01')).toEqual(before.timeline.find((r:any)=>r.effectiveDate==='2026-06-01'));expect((await read(page,undefined,before.headSourceVersion)).timeline).toEqual(before.timeline);
 await openMoreActions(page);await expect(page.getByRole('menuitem',{name:'联合修订时间线',exact:true})).toHaveCount(0);await page.keyboard.press('Escape');
});
test('同一空档不足两日仅禁用，两日空档日期固定，停用段无可插入位置',async({page})=>{
 await open(page);await create(page,'2026-06-01');await add(page,'2026-06-03','乙');await select(page,'2026-06-01');await openMoreActions(page);await expect(page.getByRole('menuitem',{name:'插入停用与重新启用',exact:true})).toBeDisabled();await page.keyboard.press('Escape');
 await page.goto('/settings/organization/business-units');await create(page,'2026-06-01');await add(page,'2026-06-04','乙');await select(page,'2026-06-01');const d=await insert(page);await expect(d.getByLabel('停用日期',{exact:true})).toHaveValue('2026-06-02');await expect(d.getByLabel('停用日期',{exact:true})).toHaveAttribute('max','2026-06-02');await expect(d.getByLabel('重新启用日期',{exact:true})).toHaveValue('2026-06-03');await expect(d.getByLabel('重新启用日期',{exact:true})).toHaveAttribute('max','2026-06-03');await choose(d,'停用原因','业务终止');await expect(d.locator('#pair-reactivationReason')).toHaveText('业务恢复');await submit(page,d);await select(page,'2026-06-02');await openMoreActions(page);await expect(page.getByRole('menuitem',{name:'插入停用与重新启用',exact:true})).toBeDisabled();await page.keyboard.press('Escape');
});
test('成对删除显示真实资料变化，保留后继独立快照及两条原件',async({page})=>{
 await open(page);await create(page);await life(page,'DEACTIVATE','2026-03-01');await life(page,'REACTIVATE','2026-04-01','乙','Beta');await life(page,'DEACTIVATE','2026-07-01');await life(page,'REACTIVATE','2026-08-01','丙','Gamma');await select(page,'2026-04-01');const before=await read(page);
 await openMoreActions(page);await page.getByRole('menuitem',{name:'删除停用与重新启用',exact:true}).click();const d=page.getByRole('dialog',{name:'删除停用与重新启用',exact:true});await expect(d.getByText('2026-03-01 · 停用',{exact:true})).toBeVisible();await expect(d.getByText('2026-04-01 · 重新启用',{exact:true})).toBeVisible();await expect(d.getByLabel('删除后的资料变化')).toContainText('乙 → 甲');await expect(d.getByLabel('删除后的资料变化')).toContainText('2026-04-01 至 2026-07-31');await expect(d.getByRole('combobox')).toHaveCount(0);
 const result=await submit(page,d,true);expect(result.items).toHaveLength(2);const after=await read(page);expect(after.timeline).toHaveLength(3);expect(after.timeline.find((r:any)=>r.effectiveDate==='2026-07-01').name).toBe('甲');expect(after.timeline.find((r:any)=>r.effectiveDate==='2026-08-01').name).toBe('丙');expect((await read(page,undefined,before.headSourceVersion)).timeline).toEqual(before.timeline);
 await page.getByRole('tab',{name:'变更日志',exact:true}).click();await expect(page.locator('.change-operation').first()).toContainText('联合修订');await expect(page.locator('.change-operation').first().getByRole('link',{name:'查看删除前原件',exact:true})).toHaveCount(2);
});
test('日期联动不覆盖已编辑恢复日，取消保护；来源冲突重读保留草稿',async({page})=>{
 await open(page);await create(page);const d=await insert(page);await d.getByLabel('停用日期',{exact:true}).fill('2026-03-01');await expect(d.getByLabel('重新启用日期',{exact:true})).toHaveValue('2026-03-02');await d.getByLabel('重新启用日期',{exact:true}).fill('2026-05-01');await d.getByLabel('停用日期',{exact:true}).fill('2026-04-01');await expect(d.getByLabel('重新启用日期',{exact:true})).toHaveValue('2026-05-01');await d.getByRole('button',{name:'取消',exact:true}).click();await page.getByRole('button',{name:'继续编辑',exact:true}).click();await fill(d,page);
 expect(await page.evaluate(async()=>{const id=location.pathname.split('/').at(-1),s=await(await fetch('/api/v1/session')).json(),d=await(await fetch('/api/v1/business-units/'+id)).json();return(await fetch('/api/v1/business-units/'+id+'/records/'+d.timeline[0].recordId+'/correct',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':s.csrf,'Idempotency-Key':crypto.randomUUID()},body:JSON.stringify({expectedSourceVersion:d.headSourceVersion,expectedRecordRevision:d.timeline[0].revision,changes:{name:'并发资料'}})})).status})).toBe(200);
 await submit(page,d,false,409);await d.getByRole('button',{name:'重新核对当前资料',exact:true}).click();await expect(d.getByLabel('停用日期',{exact:true})).toHaveValue('2026-03-01');await submit(page,d);expect((await read(page)).timeline.find((r:any)=>r.operationType==='REACTIVATE').name).toBe('并发资料');
});
test('丢失响应可重试保存，零重复记录；未配对的停用无成对删除入口',async({page})=>{
  await deferAutomaticConfirmation(page);
 await open(page);await create(page);const d=await insert(page);await fill(d,page);let lost=true;
 await page.route('**/api/v1/business-units/*/lifecycle-pairs',async route=>{const r=await route.fetch();if(lost){lost=false;await route.abort('failed')}else await route.fulfill({response:r})});await d.getByRole('button',{name:'保存',exact:true}).click();await expect(d.getByRole('button',{name:'重试保存',exact:true})).toBeVisible();await d.getByRole('button',{name:'重试保存',exact:true}).click();await expect(d).toHaveCount(0);expect((await read(page)).timeline).toHaveLength(3);
 await select(page,'2026-04-01');await life(page,'DEACTIVATE','2026-09-01');await openMoreActions(page);await expect(page.getByRole('menuitem',{name:'删除停用与重新启用',exact:true})).toHaveCount(0);await page.keyboard.press('Escape');
});

test('保存校验直达原因字段，撤回一方权限后不显示办理入口，隐藏资料不泄露',async({page})=>{
 await open(page);await create(page);let d=await insert(page);await submit(page,d,false,400);await expect(d.getByLabel('停用原因',{exact:true})).toBeFocused();await d.getByRole('button',{name:'取消',exact:true}).click();
 await life(page,'DEACTIVATE','2026-03-01');await life(page,'REACTIVATE','2026-04-01','乙','Hidden');await select(page,'2026-03-01');
 await withPolicy(page,async(where,mutate)=>{
  mutate(`UPDATE access.subject_grant SET read_fields=ARRAY['code','effectiveDate'] WHERE ${where} AND action IN ('bu.read','bu.history.read')`);
  await page.reload();await openMoreActions(page);await page.getByRole('menuitem',{name:'删除停用与重新启用',exact:true}).click();d=page.getByRole('dialog',{name:'删除停用与重新启用',exact:true});await expect(d.getByLabel('删除后的资料变化')).toHaveCount(0);await expect(d).not.toContainText('Hidden');await expect(d).not.toContainText('2026-03-01 · 停用');await expect(d.getByText('类型不可读取',{exact:false})).toHaveCount(2);await d.getByRole('button',{name:'取消',exact:true}).click();
  mutate(`UPDATE access.subject_grant SET write_fields=ARRAY[]::text[] WHERE ${where} AND action='bu.reactivate'`);await page.reload();await openMoreActions(page);await expect(page.getByRole('menuitem',{name:'插入停用与重新启用',exact:true})).toHaveCount(0);await page.keyboard.press('Escape');
 });
});

test('只有两项启停权限仍可从菜单插入，取消焦点回到触发按钮',async({page})=>{
 await open(page);await create(page);
 await withPolicy(page,async(where,mutate)=>{
  mutate(`UPDATE access.subject_grant SET write_fields=ARRAY[]::text[] WHERE ${where} AND action NOT IN ('bu.deactivate','bu.reactivate')`);await page.reload();await openMoreActions(page);await expect(page.getByRole('menuitem',{name:'更正记录',exact:true})).toHaveCount(0);await page.getByRole('menuitem',{name:'插入停用与重新启用',exact:true}).click();const d=page.getByRole('dialog',{name:'插入停用与重新启用',exact:true});await d.getByRole('button',{name:'取消',exact:true}).click();await expect(page.getByRole('button',{name:'更多操作',exact:true})).toBeFocused();
  const again=await insert(page);await fill(again,page);await submit(page,again);expect((await read(page)).timeline).toHaveLength(3);
 });
});

test('固定两条丢响应后退出并刷新，只凭原引用恢复原来源且历史只读',async({page})=>{
  await deferAutomaticConfirmation(page);
 await open(page);await create(page);const d=await insert(page);await fill(d,page);let committed:any;
 await page.route('**/api/v1/business-units/*/lifecycle-pairs',async route=>{const response=await route.fetch();committed=await response.json();await route.abort('failed')});await d.getByRole('button',{name:'保存',exact:true}).click();await expect(d.getByRole('button',{name:'重试保存',exact:true})).toBeVisible();await d.getByRole('button',{name:'取消',exact:true}).click();await expect(page.getByRole('dialog',{name: /^(尚有未保存修改|保存结果尚未确认)$/,exact:true})).toContainText('离开不会撤销已提交的保存操作');await page.getByRole('button',{name: /^(放弃修改|离开此页)$/,exact:true}).click();await expect(d).toHaveCount(0);await page.reload();
 const resolve=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/resolve'));await page.getByRole('button',{name:'查看保存结果',exact:true}).click();expect((await resolve).status()).toBe(200);await expect(page.getByText('已保存',{exact:true})).toBeVisible();expect((await read(page)).timeline).toHaveLength(3);await page.getByRole('link',{name:'查看保存结果',exact:true}).click();await expect(page).toHaveURL(u=>u.searchParams.get('sourceVersion')===committed.sourceVersion);await expect(page.getByText('正在查看只读历史资料',{exact:true})).toBeVisible();await expect(page.getByRole('button',{name:'更多操作',exact:true})).toHaveCount(0);
});
