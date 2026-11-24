import {test,expect,openMoreActions,type Page} from './fixtures';
import {open,create,add,life,choose,select,read} from './bu-support';
import {withPolicy} from './grants';

async function start(page:Page,name:'新增记录'|'插入记录'){
 await page.getByRole('button',{name,exact:true}).click();
 await expect(page.getByRole('heading',{name,exact:true})).toBeVisible();
}
async function save(page:Page,status=201){
 const response=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/records'));
 await page.getByRole('button',{name:'保存',exact:true}).click();const r=await response;expect(r.status()).toBe(status);
 if(status<300)await expect(page.locator('.object-form')).toHaveCount(0);
 return r.json();
}

test('TL258261265266 最新按日排序追加，未来及闰日默认，不复制分类原因',async({page})=>{
 await open(page);await create(page,'2028-02-28');await start(page,'新增记录');
 await expect(page.getByLabel('生效日期').and(page.locator('input'))).toHaveValue('2028-02-29');
 await expect(page.getByRole('textbox',{name:'中文名称',exact:true})).toHaveValue('甲');await expect(page.getByRole('textbox',{name:'英文名称',exact:true})).toHaveValue('Alpha');
 await expect(page.getByRole('combobox',{name:'操作原因',exact:true}).locator('..').locator('input')).toHaveValue('');await expect(page.getByText('数据更改',{exact:true})).toBeVisible();
 await page.getByRole('button',{name:'取消编辑',exact:true}).click();await expect(page.getByRole('dialog')).toHaveCount(0);
 await add(page,'2028-06-01','乙','Beta');await select(page,'2028-02-28');
 await openMoreActions(page);await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('甲2');await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);
 await expect(page.getByRole('button',{name:'插入记录',exact:true})).toBeEnabled();await start(page,'插入记录');await expect(page.getByRole('textbox',{name:'中文名称',exact:true})).toHaveValue('甲2');await page.getByRole('button',{name:'取消编辑',exact:true}).click();
 await select(page,'2028-06-01');await expect(page.getByRole('button',{name:'新增记录',exact:true})).toBeEnabled();
});

test('TL262268271280 中段从所选资料带值，默认次日且空变化不新增，真实插入保护后继',async({page})=>{
 await open(page);await create(page);await add(page,'2026-06-01','乙','Beta');await add(page,'2026-10-01','丙','Gamma');await select(page,'2026-06-01');const before=await read(page);
 await start(page,'插入记录');const date=page.getByLabel('生效日期').and(page.locator('input'));
 await expect(date).toHaveValue('2026-06-02');await expect(date).toHaveAttribute('min','2026-06-02');await expect(date).toHaveAttribute('max','2026-09-30');await expect(page.getByRole('textbox',{name:'中文名称',exact:true})).toHaveValue('乙');await expect(page.getByRole('textbox',{name:'英文名称',exact:true})).toHaveValue('Beta');
 await save(page,400);expect((await read(page)).headSourceVersion).toBe(before.headSourceVersion);await choose(page,'操作原因','名称调整');
 const request=page.waitForRequest(r=>r.method()==='POST'&&r.url().endsWith('/records'));const unchanged=await save(page,200);expect(unchanged.outcome).toBe('no_change');expect((await request).postDataJSON()).toMatchObject({anchorRecordId:before.timeline[1].recordId,changes:{},operationType:'CHANGE'});expect((await read(page)).headSourceVersion).toBe(before.headSourceVersion);
 await start(page,'插入记录');await choose(page,'操作原因','名称调整');await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('乙2');await page.getByRole('textbox',{name:'英文名称',exact:true}).fill('');await save(page);
 const after=await read(page);expect(after.timeline.map((r:{name:string})=>r.name)).toEqual(['甲','乙','乙2','丙']);expect(after.timeline[2].nameEn).toBeUndefined();expect(after.timeline[3].nameEn).toBe('Gamma');expect((await read(page,undefined,before.headSourceVersion)).timeline).toEqual(before.timeline);
});

test('TL263264 无日期仅禁用插入，一天空档只允许该日',async({page})=>{
 await open(page);await create(page,'2026-06-01');await add(page,'2026-06-02','乙');await add(page,'2026-06-04','丙');await select(page,'2026-06-01');
 const disabled=page.getByRole('button',{name:'插入记录',exact:true});await expect(disabled).toBeDisabled();await expect(disabled).not.toHaveAttribute('title');await expect(disabled).not.toHaveAttribute('aria-describedby');await expect(page.locator('.object-form')).toHaveCount(0);
 await select(page,'2026-06-02');await start(page,'插入记录');const date=page.getByLabel('生效日期').and(page.locator('input'));await expect(date).toHaveValue('2026-06-03');await expect(date).toHaveAttribute('min','2026-06-03');await expect(date).toHaveAttribute('max','2026-06-03');
 await choose(page,'操作原因','名称调整');await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('中间');await save(page);expect((await read(page)).timeline[2]).toMatchObject({effectiveDate:'2026-06-03',name:'中间'});
});

test('TL260266280 当前停用仍可插入此前启用段，停用段不可隐式恢复',async({page})=>{
 await open(page);await create(page);await life(page,'DEACTIVATE','2026-04-01');await life(page,'REACTIVATE','2028-08-01','乙','Beta');
 await select(page,'2026-04-01');await expect(page.getByRole('button',{name:'插入记录',exact:true})).toBeDisabled();
 await select(page,'2026-01-01');await start(page,'插入记录');await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('早期资料');await choose(page,'操作原因','名称调整');await save(page);
 const after=await read(page);expect(after.timeline[2]).toMatchObject({operationType:'DEACTIVATE',name:'早期资料'});expect(after.timeline[3]).toMatchObject({operationType:'REACTIVATE',name:'乙',nameEn:'Beta'});await select(page,'2028-08-01');await start(page,'新增记录');await expect(page.getByRole('textbox',{name:'中文名称',exact:true})).toHaveValue('乙');await expect(page.getByText('数据更改',{exact:true})).toBeVisible();await expect(page.getByRole('combobox',{name:'操作原因',exact:true}).locator('..').locator('input')).toHaveValue('');await page.getByRole('button',{name:'取消编辑',exact:true}).click();
});

test('TL268269 初始日期资料不脏，真实修改保护且还原基线直接取消',async({page})=>{
 await open(page);await create(page);const day=(await read(page)).currentDate;await start(page,'新增记录');const date=page.getByLabel('生效日期').and(page.locator('input'));const original=await date.inputValue();expect(original).toBe(day);
 await date.fill('2026-02-01');await page.getByRole('button',{name:'取消编辑',exact:true}).click();await page.getByRole('button',{name:'继续编辑',exact:true}).click();await expect(date).toHaveValue('2026-02-01');
 await date.fill(original);await page.getByRole('button',{name:'取消编辑',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);await expect(page.getByRole('dialog')).toHaveCount(0);
});

for (const conflictPath of ['直接保存', '自动辅助'] as const) test(`TL274276 旧位置冲突保留草稿，重读不改锚点、不重套日期（${conflictPath}）`,async({page})=>{
 let release!:()=>void;const held=new Promise<void>(resolve=>release=resolve);
 // Exercise both Go conflict paths deterministically; response timing must not choose the case.
 await page.route('**/api/v1/business-units/*/preview',async route=>{
  if(conflictPath==='直接保存')return route.fulfill({status:503,json:{code:'UNAVAILABLE',message:'受控辅助不可用，仍由保存事务校验'}});
  if(route.request().postDataJSON()?.changes?.name!=='我的资料')return route.continue();
  await held;const response=await route.fetch();await route.fulfill({response});
 });
 try {
 await open(page);await create(page);await start(page,'新增记录');await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-04-01');await choose(page,'操作原因','名称调整');await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('我的资料');
 expect(await page.evaluate(async()=>{const base='/api/v1/business-units/'+location.pathname.split('/').at(-1),s=await(await fetch('/api/v1/session')).json(),d=await(await fetch(base)).json();return(await fetch(base+'/records',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':s.csrf,'Idempotency-Key':crypto.randomUUID()},body:JSON.stringify({expectedSourceVersion:d.headSourceVersion,anchorRecordId:d.timeline[0].recordId,effectiveDate:'2026-03-01',operationType:'CHANGE',operationReason:'NAME_ADJUSTMENT',changes:{name:'并发资料'}})})).status})).toBe(201);
 if(conflictPath==='直接保存')await save(page,409);
 else {release();await expect(page.getByRole('button',{name:'重新核对当前资料',exact:true})).toBeVisible();await expect(page.getByRole('button',{name:'保存',exact:true})).toBeDisabled();}
 } finally {release();await page.unrouteAll({behavior:'wait'});}
 await page.getByRole('button',{name:'重新核对当前资料',exact:true}).click();const date=page.getByLabel('生效日期').and(page.locator('input'));await expect(date).toHaveValue('2026-04-01');await expect(date).toHaveAttribute('max','2026-02-28');await expect(page.getByRole('textbox',{name:'中文名称',exact:true})).toHaveValue('我的资料');await expect(page.getByRole('heading',{name:'插入记录',exact:true})).toBeVisible();
 await save(page,400);await date.fill('2026-02-01');await save(page);expect((await read(page)).timeline.map((r:{name:string})=>r.name)).toEqual(['甲','我的资料','并发资料']);
});

test('TL270 迟到辅助不覆盖已编辑名称，也不把初始带值变为提交字段',async({page})=>{
 await open(page);await create(page);let release!:()=>void,reached!:()=>void;const barrier=new Promise<void>(r=>release=r),waiting=new Promise<void>(r=>reached=r);let parked=false;
 await page.route('**/api/v1/business-units/*/preview',async route=>{if(parked){await route.continue();return}parked=true;const response=await route.fetch();reached();await barrier;await route.fulfill({response})});
 await start(page,'新增记录');const originalDate=await page.getByLabel('生效日期').and(page.locator('input')).inputValue();await waiting;await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('手工名称');release();await expect(page.getByRole('textbox',{name:'中文名称',exact:true})).toHaveValue('手工名称');await expect(page.getByLabel('生效日期').and(page.locator('input'))).toHaveValue(originalDate);
 await choose(page,'操作原因','名称调整');const request=page.waitForRequest(r=>r.method()==='POST'&&r.url().endsWith('/records'));await save(page);expect((await request).postDataJSON().changes).toEqual({name:'手工名称'});
});

test('TL258267268270 所选记录决定位置，初始返回退出不拦截，真实草稿离开受保护',async({page})=>{
 await open(page);const code=await create(page);await add(page,'2026-06-01','乙','Beta');await add(page,'2026-10-01','丙','Gamma');const before=await read(page);
 await select(page,'2026-06-01');await start(page,'插入记录');await expect(page.getByRole('textbox',{name:'中文名称',exact:true})).toHaveValue('乙');await expect(page.getByLabel('生效日期').and(page.locator('input'))).toHaveValue('2026-06-02');await page.getByRole('button',{name:'取消编辑',exact:true}).click();
 await start(page,'插入记录');await expect(page.getByRole('textbox',{name:'中文名称',exact:true})).toHaveValue('乙');await expect(page.getByLabel('生效日期').and(page.locator('input'))).toHaveValue('2026-06-02');await page.getByRole('link',{name:'返回列表',exact:true}).click();await expect(page.getByLabel('搜索编码或名称')).toBeVisible();await expect(page.getByRole('dialog')).toHaveCount(0);
 await page.getByLabel('搜索编码或名称').fill(code);await page.getByRole('button',{name:'查询',exact:true}).click();await page.getByRole('row').filter({hasText:code}).getByRole('link').click();await select(page,'2026-06-01');await start(page,'插入记录');await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('未保存名称');await page.getByRole('link',{name:'返回列表',exact:true}).click();await page.getByRole('button',{name:'继续编辑',exact:true}).click();await expect(page.getByRole('textbox',{name:'中文名称',exact:true})).toHaveValue('未保存名称');await page.getByRole('link',{name:'返回列表',exact:true}).click();await page.getByRole('button',{name:'放弃修改',exact:true}).click();
 await page.getByRole('row').filter({hasText:code}).getByRole('link').click();await select(page,'2026-06-01');await start(page,'插入记录');await expect(page.getByRole('textbox',{name:'中文名称',exact:true})).toHaveValue('乙');expect((await read(page)).headSourceVersion).toBe(before.headSourceVersion);await page.getByRole('button',{name:'退出',exact:true}).click();await expect(page.getByLabel('登录名')).toBeVisible();await expect(page.getByRole('dialog')).toHaveCount(0);
});

test('TL277 锚点已删除时重读保留草稿并禁存，不自动改到其他位置',async({page})=>{
 await open(page);await create(page);await add(page,'2026-06-01','乙');await add(page,'2026-10-01','丙');await select(page,'2026-06-01');await start(page,'插入记录');await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('保留草稿');const ready=page.waitForResponse(r=>r.url().endsWith('/preview')&&r.request().postDataJSON()?.operationReason==='NAME_ADJUSTMENT');await choose(page,'操作原因','名称调整');expect((await ready).status()).toBe(200);
 expect(await page.evaluate(async()=>{const base='/api/v1/business-units/'+location.pathname.split('/').at(-1),s=await(await fetch('/api/v1/session')).json(),d=await(await fetch(base)).json(),r=d.timeline.find((x:{effectiveDate:string})=>x.effectiveDate==='2026-06-01');return(await fetch(base+'/records/'+r.recordId+'/delete',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':s.csrf,'Idempotency-Key':crypto.randomUUID()},body:JSON.stringify({expectedSourceVersion:d.headSourceVersion,expectedRecordRevision:r.revision})})).status})).toBe(200);
 const head=(await read(page)).headSourceVersion;await save(page,409);await page.getByRole('button',{name:'重新核对当前资料',exact:true}).click();await expect(page.getByRole('textbox',{name:'中文名称',exact:true})).toHaveValue('保留草稿');await expect(page.getByLabel('生效日期').and(page.locator('input'))).toHaveValue('2026-06-02');await expect(page.getByRole('button',{name:'保存',exact:true})).toBeDisabled();expect((await read(page)).headSourceVersion).toBe(head);await page.getByRole('button',{name:'取消编辑',exact:true}).click();await page.getByRole('button',{name:'放弃修改',exact:true}).click();
});

for(const field of ['nameEn','effectiveDate'])test(`TL272 新增表单撤回${field}读权清除输入和默认，保留仍可读名称`,async({page})=>{
 await open(page);await create(page);await add(page,'2026-06-01','乙','Beta');await add(page,'2026-10-01','丙','Gamma');await select(page,'2026-06-01');const original=await read(page);
 await start(page,'插入记录');await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('我的资料');const ready=page.waitForResponse(r=>r.url().endsWith('/preview')&&r.request().postDataJSON()?.operationReason==='NAME_ADJUSTMENT');await choose(page,'操作原因','名称调整');expect((await ready).status()).toBe(200);
 await withPolicy(page,async(where,mutate)=>{
  mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'${field}') WHERE ${where} AND action IN ('bu.read','bu.history.read')`);
  await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('保留我的输入');
  if(field==='nameEn'){
   await expect(page.getByRole('textbox',{name:'英文名称',exact:true})).toHaveCount(0);await expect(page.locator('body')).not.toContainText('Beta');await expect(page.getByRole('textbox',{name:'中文名称',exact:true})).toHaveValue('保留我的输入');
   const request=page.waitForRequest(r=>r.method()==='POST'&&r.url().endsWith('/records'));await save(page);expect((await request).postDataJSON().changes).toEqual({name:'保留我的输入'});
  }else{
   await expect(page.getByLabel('生效日期').and(page.locator('input'))).toHaveCount(0);await expect(page.locator('input[type="date"]')).toHaveCount(0);await expect(page.locator('.object-form')).not.toContainText('2026-06-02');await expect(page.getByRole('button',{name:'保存',exact:true})).toBeDisabled();await expect(page.getByText('没有操作权限，可取消编辑',{exact:true})).toBeVisible();await expect(page.getByRole('textbox',{name:'中文名称',exact:true})).toHaveValue('保留我的输入');
   await page.getByRole('button',{name:'取消编辑',exact:true}).click();await page.getByRole('button',{name:'放弃修改',exact:true}).click();expect((await read(page)).headSourceVersion).toBe(original.headSourceVersion);
  }
 });
 const after=await read(page);if(field==='nameEn')expect(after.timeline.find((r:any)=>r.effectiveDate==='2026-06-02')).toMatchObject({name:'保留我的输入',nameEn:'Beta'});expect((await read(page,undefined,original.headSourceVersion)).timeline).toEqual(original.timeline);
});

for(const move of ['anchor','next'])test(`TL277 并发改动${move}日期后重读保留锚点及草稿，按新位置验证`,async({page})=>{
 await open(page);await create(page);await add(page,'2026-06-01','乙','Beta');await add(page,'2026-10-01','丙','Gamma');await select(page,'2026-06-01');const original=await read(page);const anchor=original.timeline.find((r:any)=>r.effectiveDate==='2026-06-01');
 await start(page,'插入记录');const date=page.getByLabel('生效日期').and(page.locator('input'));const draft=move==='anchor'?'2026-06-02':'2026-09-01';await date.fill(draft);await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('我的草稿');const ready=page.waitForResponse(r=>r.url().endsWith('/preview')&&r.request().postDataJSON()?.operationReason==='NAME_ADJUSTMENT');await choose(page,'操作原因','名称调整');expect((await ready).status()).toBe(200);
 expect(await page.evaluate(async(move)=>{const base='/api/v1/business-units/'+location.pathname.split('/').at(-1),s=await(await fetch('/api/v1/session')).json(),d=await(await fetch(base)).json();const target=d.timeline.find((r:any)=>r.effectiveDate===(move==='anchor'?'2026-06-01':'2026-10-01'));return(await fetch(base+'/records/'+target.recordId+'/correct',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':s.csrf,'Idempotency-Key':crypto.randomUUID()},body:JSON.stringify({expectedSourceVersion:d.headSourceVersion,expectedRecordRevision:target.revision,newEffectiveDate:move==='anchor'?'2026-07-01':'2026-08-01'})})).status},move)).toBe(200);
 const concurrent=await read(page);await save(page,409);await page.getByRole('button',{name:'重新核对当前资料',exact:true}).click();await expect(date).toHaveValue(draft);await expect(page.getByRole('textbox',{name:'中文名称',exact:true})).toHaveValue('我的草稿');await expect(date).toHaveAttribute(move==='anchor'?'min':'max',move==='anchor'?'2026-07-02':'2026-07-31');
 await save(page,400);expect((await read(page)).headSourceVersion).toBe(concurrent.headSourceVersion);await date.fill(move==='anchor'?'2026-07-02':'2026-07-01');const request=page.waitForRequest(r=>r.method()==='POST'&&r.url().endsWith('/records'));await save(page);expect((await request).postDataJSON().anchorRecordId).toBe(anchor.recordId);
 expect((await read(page,undefined,original.headSourceVersion)).timeline).toEqual(original.timeline);expect((await read(page,undefined,concurrent.headSourceVersion)).timeline).toEqual(concurrent.timeline);
});
