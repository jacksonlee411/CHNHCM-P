import { lazyCredentials } from './credentials.mjs';
import { deferAutomaticConfirmation } from './fixtures';
import { expectLogChange } from './fixtures';
import { readRecord } from './fixtures';
import {readCurrentDate,  test, expect, openMoreActions, expectCurrentDate, type Page } from './fixtures';
import { readFileSync } from 'node:fs';
import { execFileSync } from 'node:child_process';
import { randomUUID } from 'node:crypto';
const credentials= lazyCredentials();
async function open(page:Page) {
 await new Promise(r=>setTimeout(r,3100));await page.goto('/');await page.getByLabel('登录名').fill(credentials.login);await page.getByLabel('密码').fill(credentials.password);await page.getByRole('button',{name:'登录',exact:true}).click();
 await page.getByRole('button',{name:'切换应用',exact:true}).click();await page.getByRole('menuitem',{name:'设置',exact:true}).click();await page.getByRole('button',{name:'组织配置',exact:true}).click();await page.getByRole('menuitem',{name:'业务单元',exact:true}).click();expect(await readCurrentDate(page)).toMatch(/^\d{4}-\d{2}-\d{2}$/);return readCurrentDate(page);
}
async function create(page:Page) {
 const code='REDATE_'+randomUUID().slice(0,8);await page.getByRole('link',{name:'新建',exact:true}).click();await page.getByRole('textbox',{name:'编码',exact:true}).fill(code);await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('甲');await page.getByRole('textbox',{name:'英文名称',exact:true}).fill('Alpha');await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-01-01');await page.getByRole('combobox',{name:'操作原因',exact:true}).click();await page.getByRole('option',{name:'新设组织',exact:true}).click();await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);await expect(page.getByRole('button',{name:'更多操作',exact:true})).toBeEnabled();return code;
}
async function add(page:Page,date:string,name:string,en?:string) {
 const d=await read(page), anchor=d.timeline.filter((r:{effectiveDate:string})=>r.effectiveDate<date).at(-1)??d.timeline[0];await select(page,anchor.effectiveDate);
 await page.getByRole('button',{name:/^(新增记录|插入记录)$/,exact:true}).click();await page.getByLabel('生效日期').and(page.locator('input')).fill(date);await page.getByRole('combobox',{name:'操作原因',exact:true}).click();await page.getByRole('option',{name:'名称调整',exact:true}).click();await page.getByRole('textbox',{name:'中文名称',exact:true}).fill(name);if(en!==undefined)await page.getByRole('textbox',{name:'英文名称',exact:true}).fill(en);await readyToSave(page);await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);
}
async function life(page:Page,kind:'DEACTIVATE'|'REACTIVATE',date:string) {
 await openMoreActions(page);await page.getByRole('menuitem',{name:kind==='DEACTIVATE'?'停用业务单元':'重新启用业务单元',exact:true}).click();await page.getByLabel('生效日期').and(page.locator('input')).fill(date);await page.getByRole('combobox',{name:'操作原因',exact:true}).click();await page.getByRole('option',{name:kind==='DEACTIVATE'?'业务终止':'业务恢复',exact:true}).click();if(kind==='REACTIVATE'){await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('乙');await page.getByRole('textbox',{name:'英文名称',exact:true}).fill('Beta')};await readyToSave(page);await page.getByRole('button',{name:kind==='DEACTIVATE'?'确认停用':'确认重新启用',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);
}
async function read(page: Page, date?: string, source?: string) { return readRecord(page, date, source); }
async function select(page:Page,date:string) {
 const [y,m,d]=date.split('-');await page.getByRole('button',{name:`${y}-${m}-${d}`,exact:true}).click();await expect(page.locator('.effective-dates')).toBeVisible();
}
async function enter(page:Page) {
 await openMoreActions(page);await expect(page.getByRole('menuitem',{name:'修改生效日期',exact:true})).toHaveCount(0);await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();await expect(page.getByLabel('生效日期').and(page.locator('input'))).toBeVisible();await expect(page.getByRole('combobox',{name:'操作原因',exact:true})).toBeVisible();await expect(page.getByRole('button',{name:'业务分类',exact:true})).toHaveCount(0);await expect(page.getByRole('button',{name:'保存',exact:true})).toBeVisible();
}
async function readyToSave(page:Page) {await expect(page.getByRole('button', {name:'预览',exact:true})).toHaveCount(0);await expect(page.getByText('正在读取操作影响…',{exact:true})).toHaveCount(0);}
async function save(page:Page) {
 const response=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/correct'));await page.getByRole('button',{name:'保存',exact:true}).click();const r=await response;expect(r.status()).toBe(200);const result=await r.json();await expect(page.locator('.object-form')).toHaveCount(0);return result;
}

test('正常菜单统一更正改日保持顺序，保留独立资料、原新来源及返回上下文',async({page})=>{
 const today=await open(page);const code=await create(page);await add(page,'2026-06-01','乙','Beta');await add(page,'2026-10-01','丙','Gamma');
 await page.getByRole('link',{name:'返回列表',exact:true}).click();await page.getByLabel('搜索编码或名称').fill(code);await page.getByRole('button',{name:'查询',exact:true}).click();await page.getByRole('row').filter({hasText:code}).getByRole('link').click();const original=await read(page);const target=original.timeline.find((r:any)=>r.effectiveDate==='2026-06-01');
 await select(page,'2026-06-01');await enter(page);await expect(page.getByLabel('生效日期').and(page.locator('input'))).toHaveValue('2026-06-01');await page.getByRole('button',{name:'取消编辑',exact:true}).click();await expect(page.getByRole('button',{name:'更多操作',exact:true})).toBeFocused();
 await enter(page);await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-09-01');await readyToSave(page);await expect(page.getByText('生效日期：2026-06-01 → 2026-09-01',{exact:true})).toBeVisible();const saved=await save(page);expect(saved.targetRecordId).toBe(target.recordId);await expectCurrentDate(page,today);
 expect((await read(page,'2026-06-15')).selectedRecord.name).toBe('甲');expect((await read(page,'2026-10-15')).selectedRecord.name).toBe('丙');expect((await read(page,'2026-09-01')).selectedRecord.nameEn).toBe('Beta');expect((await read(page,'2026-06-15',original.headSourceVersion)).selectedRecord.name).toBe('乙');
 await page.getByRole('tab',{name:'变更日志',exact:true}).click();await expectLogChange(page, '生效日期', '2026-06-01', '2026-09-01');await page.getByRole('link',{name:'查看当次资料',exact:true}).first().click();await expect(page.getByRole('button',{name:'更多操作',exact:true})).toHaveCount(0);await page.getByRole('button',{name:'返回当前资料',exact:true}).click();await page.reload();await expectCurrentDate(page,today);
 await page.getByRole('link',{name:'返回列表',exact:true}).click();await expect(page.getByLabel('搜索编码或名称')).toHaveValue(code);await expect(page).toHaveURL(u=>u.searchParams.get('selected')===saved.buId);
 console.log(JSON.stringify({code,buId:saved.buId,targetRecordId:saved.targetRecordId,sourceVersion:saved.sourceVersion,originalSource:original.headSourceVersion}));
});

test('唯一首条后移到未来显示未生效，原来源仍启用',async({page})=>{
 const today=await open(page);await create(page);const original=await read(page);const date=new Date(today+'T12:00:00Z');date.setUTCDate(date.getUTCDate()+2);const future=date.toISOString().slice(0,10);
 await enter(page);await page.getByLabel('生效日期').and(page.locator('input')).fill(future);const previewResponse=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/preview'));await readyToSave(page);const gap=(await(await previewResponse).json()).newNoFactRange;date.setUTCDate(date.getUTCDate()-1);expect(gap).toEqual({start:'2026-01-01',endExclusive:future,endInclusive:date.toISOString().slice(0,10)});await expect(page.getByText(/^无适用资料：2026-01-01 至/)).toBeVisible();await expect(page.getByText('今日状态：启用 → 未生效',{exact:true})).toBeVisible();await save(page);
 const current=await read(page);expect(current.status).toBe('not_effective');expect(current.viewFact).toBeNull();expect(current.selectedRecordIsFuture).toBe(true);expect(current.timeline[0].recordId).toBe(original.timeline[0].recordId);expect((await read(page,today,original.headSourceVersion)).status).toBe('active');await expect(page.getByText('所选记录尚未生效。',{exact:true})).toBeVisible();
});

test('停用和恢复改日保留独立快照，非法后继拒绝并定位新日期',async({page})=>{
 await open(page);await create(page);await life(page,'DEACTIVATE','2026-04-01');await life(page,'REACTIVATE','2026-08-01');const old=await read(page);
 await select(page,'2026-04-01');await enter(page);await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-03-01');await readyToSave(page);await save(page);expect((await read(page,'2026-03-15')).selectedRecord.state).toBe('inactive');
 await select(page,'2026-08-01');await enter(page);await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-09-01');await readyToSave(page);await save(page);expect((await read(page,'2026-08-15')).selectedRecord.state).toBe('inactive');expect((await read(page,'2026-09-01')).selectedRecord.nameEn).toBe('Beta');expect((await read(page,'2026-08-15',old.headSourceVersion)).selectedRecord.nameEn).toBe('Beta');
 await enter(page);await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-02-01');await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.getByLabel('生效日期').and(page.locator('input'))).toBeFocused();await expect(page.getByRole('button',{name:'保存',exact:true})).toBeEnabled();await expect(page.getByLabel('生效日期').and(page.locator('input'))).toHaveValue('2026-02-01');
});

for(const recovery of ['retry','refresh'] as const) test(`改日丢响应后${recovery}恢复准确原日期及修订`,async({page})=>{
  await deferAutomaticConfirmation(page);
 await open(page);await create(page);await add(page,'2026-06-01','乙');await enter(page);await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-07-01');await readyToSave(page);const pattern='**/api/v1/business-units/*/records/*/correct';let saved:any;
 await page.route(pattern,async route=>{const r=await route.fetch();expect(r.status()).toBe(200);saved=await r.json();await route.abort('failed')});const failed=page.waitForEvent('requestfailed',r=>r.url().endsWith('/correct'));await page.getByRole('button',{name:'保存',exact:true}).click();await failed;await expect(page.getByText(/暂时无法确认保存结果/,{exact:true})).toBeVisible();await page.unroute(pattern);
 if(recovery==='retry'){await page.getByRole('button',{name:'重试保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0)}else{await page.reload();await page.getByRole('button',{name:'查看保存结果',exact:true}).click();await expect(page.getByText("已保存", { exact: true })).toBeVisible();await page.getByRole('link',{name:'查看保存结果',exact:true}).click();await expect(page).toHaveURL(u=>u.searchParams.get('sourceVersion')===saved.sourceVersion&&u.searchParams.get('selectedRecordId')===saved.targetRecordId)}
 const original=await read(page,'2026-07-01',saved.sourceVersion);expect(original.selectedRecord.revision).toBe(saved.targetRevision);expect(original.selectedRecord.effectiveDate).toBe('2026-07-01');
});

test('同值改日无新事实，查询者无改日入口，取消与未保存保护可达',async({page})=>{
 await open(page);const code=await create(page);const original=await read(page);await enter(page);await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.getByText('没有修改内容。',{exact:true})).toBeVisible();await page.getByRole('button',{name:'取消编辑',exact:true}).click();expect((await read(page)).headSourceVersion).toBe(original.headSourceVersion);
 await enter(page);await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-02-01');await page.getByRole('button',{name:'取消编辑',exact:true}).click();await expect(page.getByRole('dialog')).toBeVisible();await page.getByRole('button',{name:'继续编辑',exact:true}).click();await expect(page.getByLabel('生效日期').and(page.locator('input'))).toHaveValue('2026-02-01');await readyToSave(page);await expect(page.getByRole('button',{name:'保存',exact:true})).toBeInViewport();await save(page);
 await page.setViewportSize({width:1440,height:900});await page.getByRole('button',{name:'退出',exact:true}).click();await page.getByLabel('登录名').fill(credentials.reader_login);await page.getByLabel('密码').fill(credentials.reader_password);await page.getByRole('button',{name:'登录',exact:true}).click();await page.getByRole('button',{name:'切换应用',exact:true}).click();await page.getByRole('menuitem',{name:'设置',exact:true}).click();await page.getByRole('button',{name:'组织配置',exact:true}).click();await page.getByRole('menuitem',{name:'业务单元',exact:true}).click();await page.getByLabel('搜索编码或名称').fill(code);await page.getByRole('button',{name:'查询',exact:true}).click();await page.getByRole('row').filter({hasText:code}).getByRole('link').click();await expect(page.getByRole('button',{name:'更多操作',exact:true})).toHaveCount(0);
});

test('改日旧辅助迟到不覆盖新日期，来源冲突保留仍可读日期',async({page})=>{
 await open(page);await create(page);await add(page,'2026-06-01','乙');await enter(page);await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-07-01');let release!:()=>void;const barrier=new Promise<void>(r=>{release=r});let reached!:()=>void;const waiting=new Promise<void>(r=>{reached=r});
 let finished!:()=>void;const sent=new Promise<void>(r=>{finished=r});const pattern='**/api/v1/business-units/*/preview';await page.route(pattern,async route=>{const response=await route.fetch();reached();await barrier;await route.fulfill({response});finished()});await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-07-02');await waiting;await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-08-01');release();await sent;await page.unroute(pattern);await expect(page.getByRole('button',{name:'保存',exact:true})).toBeEnabled();await expect(page.getByLabel('生效日期').and(page.locator('input'))).toHaveValue('2026-08-01');await readyToSave(page);
 await page.evaluate(async()=>{const s=await(await fetch('/api/v1/session')).json();const id=location.pathname.split('/').at(-1);const d=await(await fetch('/api/v1/business-units/'+id)).json();await fetch('/api/v1/business-units/'+id+'/records',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':s.csrf,'Idempotency-Key':crypto.randomUUID()},body:JSON.stringify({expectedSourceVersion:d.headSourceVersion,anchorRecordId:d.timeline.at(-1).recordId,effectiveDate:'2026-10-01',operationType:'CHANGE',operationReason:'NAME_ADJUSTMENT',changes:{name:'丙'}})})});
 const refused=page.waitForResponse(r=>r.url().endsWith('/correct'));await page.getByRole('button',{name:'保存',exact:true}).click();expect((await refused).status()).toBe(409);await expect(page.getByLabel('生效日期').and(page.locator('input'))).toHaveValue('2026-08-01');await page.getByRole('button',{name:'重新核对当前资料',exact:true}).click();await readyToSave(page);await save(page);
});

for(const path of ['preview','reload'] as const) test(`改日${path}时撤日期读权，清除日期输入、辅助与父详情缓存`,async({page})=>{
 await open(page);await create(page);await add(page,'2026-06-01','乙');await enter(page);
 // Establish the readable preview before revoking rights. Otherwise its late
 // response can correctly remove the date input before the next fill starts.
 const initialPreview=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/preview')&&r.request().postDataJSON()?.newEffectiveDate==='2026-07-01');
 await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-07-01');
 const initialResponse=await initialPreview;expect(initialResponse.status()).toBe(200);expect((await initialResponse.json()).readFields).toContain('effectiveDate');
 await expect(page.getByText('生效日期：2026-06-01 → 2026-07-01',{exact:true})).toBeVisible();
 if(path==='reload') {
  await readyToSave(page);
  const correction=await page.evaluate(async()=>{
   const s=await(await fetch('/api/v1/session')).json();const id=location.pathname.split('/').at(-1);const d=await(await fetch('/api/v1/business-units/'+id)).json();const target=d.timeline.find((r:any)=>r.effectiveDate==='2026-06-01');
   return(await fetch('/api/v1/business-units/'+id+'/records/'+target.recordId+'/correct',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':s.csrf,'Idempotency-Key':crypto.randomUUID()},body:JSON.stringify({expectedSourceVersion:d.headSourceVersion,expectedRecordRevision:target.revision,changes:{name:'并发更正'}})})).status;
  });expect(correction).toBe(200);
  const conflict=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/correct'));
  await page.getByRole('button',{name:'保存',exact:true}).click();expect((await conflict).status()).toBe(409);await expect(page.getByRole('button',{name:'重新核对当前资料',exact:true})).toBeVisible();
 }
 const subject=await page.evaluate(async()=>(await(await fetch('/api/v1/session')).json()).subjectId);expect(subject).toMatch(/^[0-9a-f-]{36}$/);expect(credentials.instance).toMatch(/^[0-9a-f-]{36}$/);
 const connection=JSON.parse(execFileSync('python3',['scripts/local-bu.py','connection'],{encoding:'utf8'}));const container=execFileSync('docker',['ps','-q','--filter','label=com.docker.compose.project='+connection.project,'--filter','label=com.docker.compose.service=postgres'],{encoding:'utf8'}).trim();expect(container).toMatch(/^[0-9a-f]+$/);
 const sql=(query:string)=>execFileSync('docker',['exec','-i',container,'psql','-U','postgres','-d','chnhcm','-X','-Atq','-v','ON_ERROR_STOP=1'],{input:query,encoding:'utf8'}).trim();const where=`instance_id='${credentials.instance}' AND principal_id='${subject}'`;const grants=sql(`SELECT json_agg(g) FROM (SELECT grant_id,read_fields FROM access.subject_grant WHERE ${where}) g;`);const mutate=(query:string)=>sql(`BEGIN;SELECT principal_id FROM access.policy_guard WHERE ${where} FOR UPDATE;${query};UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE ${where};COMMIT;`);
 try {
  mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'effectiveDate') WHERE ${where} AND action IN ('bu.read','bu.history.read')`);
  if(path==='preview') {
   const response=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/preview')&&r.request().postDataJSON()?.newEffectiveDate==='2026-07-02');await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-07-02');const r=await response;expect(r.status()).toBe(200);const body=await r.json();expect(body.beforeRecord).toBeUndefined();expect(body.afterRecord.effectiveDate).toBeUndefined();expect(body.readFields).not.toContain('effectiveDate');
  } else {
   const response=page.waitForResponse(r=>r.request().method()==='GET'&&/\/api\/v1\/business-units\/[^/?]+(?:\?|$)/.test(r.url()));await page.getByRole('button',{name:'重新核对当前资料',exact:true}).click();const r=await response;expect(r.status()).toBe(200);const body=await r.json();expect(body.capabilities.readFields).not.toContain('effectiveDate');expect(body.capabilities.redate).toBe(true);
  }
  await expect(page.getByText('无法查看生效日期，可修改其他有权字段',{exact:true})).toBeVisible();await expect(page.getByLabel('生效日期').and(page.locator('input'))).toHaveCount(0);await expect(page.locator('input[type="date"]')).toHaveCount(0);await expect(page.locator('body')).not.toContainText('2026-06-01');await expect(page.locator('body')).not.toContainText('2026-07-01');await expect(page.getByRole('button',{name:'预览',exact:true})).toHaveCount(0);await expect(page.getByRole('button',{name:'保存',exact:true})).toBeEnabled();await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('仅名称更正');await save(page);
 } finally {mutate(`UPDATE access.subject_grant g SET read_fields=s.read_fields FROM json_to_recordset('${grants.replaceAll("'","''")}') AS s(grant_id uuid,read_fields text[]) WHERE g.instance_id='${credentials.instance}' AND g.grant_id=s.grant_id`);}
});
