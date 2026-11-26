import { lazyCredentials } from './credentials.mjs';
import { deferAutomaticConfirmation } from './fixtures';
import { readRecord } from './fixtures';
import {readCurrentDate,  test, expect, openMoreActions, expectCurrentDate, type Page } from './fixtures';
import { readFileSync } from 'node:fs';
import { execFileSync } from 'node:child_process';
import { randomUUID } from 'node:crypto';
const credentials= lazyCredentials();
const list='/settings/organization/business-units';
async function open(page:Page) {
 await new Promise(r=>setTimeout(r,3100));
 await page.goto('/');await page.getByLabel('登录名').fill(credentials.login);await page.getByLabel('密码').fill(credentials.password);await page.getByRole('button',{name:'登录',exact:true}).click();
 await page.getByRole('button',{name:'切换应用',exact:true}).click();await page.getByRole('menuitem',{name:'设置',exact:true}).click();await page.getByRole('button',{name:'组织配置',exact:true}).click();await page.getByRole('menuitem',{name:'业务单元',exact:true}).click();
 expect(await readCurrentDate(page)).toMatch(/^\d{4}-\d{2}-\d{2}$/);return readCurrentDate(page);
}
async function create(page:Page) {
 const code='DEL_'+randomUUID().slice(0,8);
 await page.getByRole('link',{name:'新建',exact:true}).click();
 await page.getByRole('textbox',{name:'编码',exact:true}).fill(code);await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('甲');await page.getByRole('textbox',{name:'英文名称',exact:true}).fill('Alpha');
 await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-01-01');await page.getByRole('combobox',{name:'操作原因',exact:true}).click();await page.getByRole('option',{name:'新设组织',exact:true}).click();
 await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);await expect(page.getByRole('button',{name:'更多操作',exact:true})).toBeEnabled();return code;
}
async function add(page:Page,date:string,name:string,en?:string) {
 const d=await read(page), anchor=d.timeline.filter((r:{effectiveDate:string})=>r.effectiveDate<date).at(-1)??d.timeline[0];
 const [ay,am,ad]=anchor.effectiveDate.split('-');await page.getByRole('button',{name:`${ay}-${am}-${ad}`,exact:true}).click();await expect(page.getByRole('button',{name:'更多操作',exact:true})).toBeEnabled();
 await page.getByRole('button',{name:/^(新增记录|插入记录)$/,exact:true}).click();await page.getByLabel('生效日期').and(page.locator('input')).fill(date);
 await page.getByRole('combobox',{name:'操作原因',exact:true}).click();await page.getByRole('option',{name:'名称调整',exact:true}).click();await page.getByRole('textbox',{name:'中文名称',exact:true}).fill(name);
 if(en!==undefined)await page.getByRole('textbox',{name:'英文名称',exact:true}).fill(en);
 await expect(page.getByRole('button', {name:'预览',exact:true})).toHaveCount(0);await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);await expect(page.locator('.effective-dates')).toBeVisible();
}
async function life(page:Page,kind:'DEACTIVATE'|'REACTIVATE',date:string) {
 await openMoreActions(page);await page.getByRole('menuitem',{name:kind==='DEACTIVATE'?'停用业务单元':'重新启用业务单元',exact:true}).click();
 await page.getByLabel('生效日期').and(page.locator('input')).fill(date);await page.getByRole('combobox',{name:'操作原因',exact:true}).click();await page.getByRole('option',{name:kind==='DEACTIVATE'?'业务终止':'业务恢复',exact:true}).click();
 await expect(page.getByRole('button', {name:'预览',exact:true})).toHaveCount(0);await page.getByRole('button',{name:kind==='DEACTIVATE'?'确认停用':'确认重新启用',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);
}
async function read(page: Page, date?: string, source?: string) { return readRecord(page, date, source); }
async function select(page:Page,date:string) {
 const [y,m,d]=date.split('-');const response=page.waitForResponse(r=>r.request().method()==='GET'&&r.url().includes('/api/v1/business-units/')&&r.url().includes('selectedRecordId='));
 await page.getByRole('button',{name:`${y}-${m}-${d}`,exact:true}).click();await response;await expect(page.locator('.effective-dates')).toBeVisible();
}
async function enterDelete(page:Page) {
 await openMoreActions(page);await page.getByRole('menuitem',{name:'删除此记录',exact:true}).click();
 await expect(page.getByRole('heading',{name:'删除此记录',exact:true})).toBeVisible();
 await expect(page.locator('.object-form input')).toHaveCount(0);await expect(page.locator('.object-form').getByRole('combobox')).toHaveCount(0);
 await expect(page.getByRole('button',{name:'确认删除',exact:true})).toBeVisible();
}
async function readyToSave(page: Page) { await expect(page.getByRole('button', { name: '预览', exact: true })).toHaveCount(0); await expect(page.getByText('正在读取操作影响…', {exact:true})).toHaveCount(0); }
async function confirm(page:Page) {
 const response=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/delete'));
 await page.getByRole('button',{name:'确认删除',exact:true}).click();const r=await response;expect(r.status()).toBe(200);const result=await r.json();
 await expect(page.getByText('已删除。',{exact:true})).toBeVisible();await expect(page.locator('.object-form')).toHaveCount(0);return result;
}

test('正常菜单删除中段、保留查看和返回上下文、原件与新来源、同日新身份',async({page})=>{
 const today=await open(page);const code=await create(page);
 await openMoreActions(page);await expect(page.getByRole('menuitem',{name:'删除此记录',exact:true})).toHaveCount(0);await page.keyboard.press('Escape');
 await add(page,'2026-06-01','乙','Beta');await add(page,'2026-10-01','丙','Gamma');
 await page.getByRole('link',{name:'返回列表',exact:true}).click();await page.getByLabel('搜索编码或名称').fill(code);
 const queried=page.waitForResponse(response=>response.request().method()==='GET'&&new URL(response.url()).pathname==='/api/v1/business-units'&&new URL(response.url()).searchParams.get('q')===code);
 await page.getByRole('button',{name:'查询',exact:true}).click();const listResponse=await queried;expect(listResponse.status()).toBe(200);expect((await listResponse.json()).count).toBe(1);
 await expect(page).toHaveURL(url=>url.searchParams.get('q')===code);
 const nameLink=page.getByRole('row').filter({hasText:code}).getByRole('link');
 await expect.poll(async()=>{const href=new URL((await nameLink.getAttribute('href'))!,page.url());return new URL(href.searchParams.get('back')!,page.url()).searchParams.get('q')}).toBe(code);
 await nameLink.click();const old=await read(page);
 const target=old.timeline.find((r:any)=>r.effectiveDate==='2026-06-01');const first=old.timeline[0];
 await select(page,'2026-10-01');await enterDelete(page);
 await expect(page.getByText('自 2026-10-01 起',{exact:true})).toBeVisible();
 await page.getByRole('button',{name:'取消编辑',exact:true}).click();
 await select(page,'2026-06-01');await expect(page.getByText('2026-06-01 至 2026-09-30',{exact:true})).toBeVisible();
 await enterDelete(page);await expect(page.getByText('2026-06-01 至 2026-09-30',{exact:true})).toBeVisible();await page.getByRole('button',{name:'取消编辑',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);
 expect((await read(page)).headSourceVersion).toBe(old.headSourceVersion);await enterDelete(page);await readyToSave(page);await expect(page.getByText(/删除后适用资料：甲/)).toBeVisible();
 await page.screenshot({path:test.info().outputPath('delete-preview-masked.png'),mask:[page.locator('.app-account-name')]});
 const result=await confirm(page);expect(result.targetRecordId).toBe(target.recordId);expect(result.selectedRecordId).toBe(first.recordId);expect(result.targetRevision).not.toBe(target.revision);
 await expectCurrentDate(page,today);const current=await read(page,'2026-06-15');expect(current.timeline).toHaveLength(2);expect(current.selectedRecord.name).toBe('甲');expect(current.timeline[0].endExclusive).toBe('2026-10-01');
 expect((await read(page,'2026-06-15',old.headSourceVersion)).selectedRecord.name).toBe('乙');
 await page.getByRole('tab',{name:'变更日志',exact:true}).click();await expect(page.locator('.change-operation').first()).toContainText('乙');await expect(page.locator('.change-operation').first()).toContainText('Beta');
 await page.getByRole('link',{name:'查看删除前原件',exact:true}).first().click();await expect(page.locator('.object-detail-field').filter({has:page.getByText('中文名称',{exact:true})})).toContainText('乙');
 await expect(page.getByRole('button',{name:'更多操作',exact:true})).toHaveCount(0);
 await page.getByRole('button',{name:'返回当前资料',exact:true}).click();await add(page,'2026-06-01','新');expect((await read(page,'2026-06-15')).selectedRecord.recordId).not.toBe(target.recordId);
 await page.reload();await expectCurrentDate(page,today);expect((await read(page,'2026-06-15')).selectedRecord.name).toBe('新');
 await page.getByRole('link',{name:'返回列表',exact:true}).click();await expect(page.getByLabel('搜索编码或名称')).toHaveValue(code);await expectCurrentDate(page, today);await expect(page).toHaveURL(u=>u.searchParams.get('q')===code&&u.searchParams.get('selected')===result.buId);
 console.log(JSON.stringify({code,buId:result.buId,deletedRecordId:result.targetRecordId,deletionRevision:result.targetRevision,sourceVersion:result.sourceVersion,originalSource:old.headSourceVersion}));
});

test('删除启停按完整依赖拒绝，合法删除末尾恢复和停用准确改变今日状态',async({page})=>{
 await open(page);await create(page);await life(page,'DEACTIVATE','2026-04-01');await life(page,'REACTIVATE','2026-05-01');
 await select(page,'2026-04-01');await enterDelete(page);await page.getByRole('button',{name:'确认删除',exact:true}).click();
 await expect(page.getByText('2026-05-01 的重新启用缺少停用前提，请核对前后启停记录',{exact:true})).toBeVisible();await expect(page.getByRole('button',{name:'确认删除',exact:true})).toBeEnabled();await page.getByRole('button',{name:'取消编辑',exact:true}).click();
 await select(page,'2026-05-01');await enterDelete(page);await readyToSave(page);await expect(page.getByText('今日状态：启用 → 停用',{exact:true})).toBeVisible();await confirm(page);await expect(page.locator('.object-detail-field').filter({has:page.getByText('状态',{exact:true})}).getByText('停用',{exact:true})).toBeVisible();
 await enterDelete(page);await readyToSave(page);await expect(page.getByText('今日状态：停用 → 启用',{exact:true})).toBeVisible();await confirm(page);await expect(page.locator('.object-detail-field').filter({has:page.getByText('状态',{exact:true})}).getByText('启用',{exact:true})).toBeVisible();expect((await read(page)).timeline).toHaveLength(1);
});

for(const recovery of ['retry','refresh'] as const) test(`删除丢响应后${recovery}准确查回原删除修订`,async({page})=>{
  await deferAutomaticConfirmation(page);
 await open(page);await create(page);await add(page,'2026-06-01','乙');await enterDelete(page);await readyToSave(page);
 const pattern='**/api/v1/business-units/*/records/*/delete';let result:any;
 await page.route(pattern,async route=>{const response=await route.fetch();expect(response.status()).toBe(200);result=await response.json();await route.abort('failed')});
 const failed=page.waitForEvent('requestfailed',r=>r.url().endsWith('/delete'));await page.getByRole('button',{name:'确认删除',exact:true}).click();await failed;await expect(page.getByText(/暂时无法确认保存结果/,{exact:true})).toBeVisible();await page.unroute(pattern);
 if(recovery==='retry') {await page.getByRole('button',{name:'重试保存',exact:true}).click();await expect(page.getByText('已删除。',{exact:true})).toBeVisible();}
 else {
  await page.reload();await page.getByRole('button',{name:'查看保存结果',exact:true}).click();await expect(page.getByText("已保存", { exact: true })).toBeVisible();await page.getByRole('link',{name:'查看保存结果',exact:true}).click();
  await expect(page).toHaveURL(u=>u.searchParams.get('selectedRecordId')===result.targetRecordId&&u.searchParams.get('sourceVersion')===result.sourceVersion);
  await expect(page.getByText('已删除记录原件，只读。下方保留删除前内容与期间。',{exact:true})).toBeVisible();
  const original=await page.evaluate(async(result)=>(await fetch(`/api/v1/business-units/${result.buId}?`+new URLSearchParams({sourceVersion:result.sourceVersion,selectedRecordId:result.targetRecordId}))).json(),result);
  expect(original.selectedRecord.revision).toBe(result.targetRevision);expect(original.selectedRecord.deleted).toBe(true);await page.getByRole('button',{name:'返回当前资料',exact:true}).click();
 }
 expect((await read(page)).timeline).toHaveLength(1);
});

for(const recovery of ['retry','refresh'] as const) test(`删除拒绝的COMMIT失败后${recovery}保留原提交引用`,async({page})=>{
  await deferAutomaticConfirmation(page);
 await open(page);await create(page);await add(page,'2026-06-01','乙');const original=await read(page);await enterDelete(page);await readyToSave(page);
 // A concurrent legitimate change makes the preview stale. Fail only this
 // synthetic BU's rejection at real PostgreSQL COMMIT, with no HTTP mocking.
 const changed=await page.evaluate(async()=>{
  const session=await(await fetch('/api/v1/session')).json();const id=location.pathname.split('/').at(-1);
  const current=await(await fetch('/api/v1/business-units/'+id)).json();
  const response=await fetch('/api/v1/business-units/'+id+'/records',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':session.csrf,'Idempotency-Key':crypto.randomUUID()},body:JSON.stringify({expectedSourceVersion:current.headSourceVersion,anchorRecordId:current.timeline.at(-1).recordId,effectiveDate:'2026-10-01',operationType:'CHANGE',operationReason:'NAME_ADJUSTMENT',changes:{name:'丙'}})});
  return {status:response.status,body:await response.json()};
 });expect(changed.status).toBe(201);
 expect(original.id).toMatch(/^[0-9a-f-]{36}$/);
 const connection=JSON.parse(execFileSync('python3',['scripts/local-bu.py','connection'],{encoding:'utf8'}));
 const container=execFileSync('docker',['ps','-q','--filter','label=com.docker.compose.project='+connection.project,'--filter','label=com.docker.compose.service=postgres'],{encoding:'utf8'}).trim();expect(container).toMatch(/^[0-9a-f]+$/);
 const sql=(query:string)=>execFileSync('docker',['exec','-i',container,'psql','-U','postgres','-d','chnhcm','-X','-Atq','-v','ON_ERROR_STOP=1'],{input:query,encoding:'utf8'}).trim();
 const trigger='test_rejected_commit_'+original.id.replaceAll('-','');let key='';
 try {
  sql(`CREATE FUNCTION organization.${trigger}() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'controlled deferred failure'; END $$; CREATE CONSTRAINT TRIGGER ${trigger} AFTER INSERT ON organization.bu_terminal DEFERRABLE INITIALLY DEFERRED FOR EACH ROW WHEN (NEW.request_object_id='${original.id}' AND NEW.outcome='rejected') EXECUTE FUNCTION organization.${trigger}()`);
  const failed=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/delete'));
  await page.getByRole('button',{name:'确认删除',exact:true}).click();const r=await failed;expect(r.status()).toBe(503);
  const body=await r.json();expect(body.code).toBe('M02_RESULT_PENDING');expect(body.outcome).toBeUndefined();expect(body.failureDetails).toBeUndefined();
  key=r.request().headers()['idempotency-key'];expect(key).toMatch(/^[0-9a-f-]{36}$/);
  await expect(page.getByText(/暂时无法确认保存结果/,{exact:true})).toBeVisible();await expect(page.getByRole('button',{name:'查看保存结果',exact:true})).toBeVisible();
  expect(sql(`SELECT count(*) FROM organization.bu_terminal WHERE submission_id='${key}'`)).toBe('0');
 } finally {sql(`DROP TRIGGER IF EXISTS ${trigger} ON organization.bu_terminal; DROP FUNCTION IF EXISTS organization.${trigger}()`);}
 if(recovery==='retry') {
  const retry=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/delete'));
  await page.getByRole('button',{name:'重试保存',exact:true}).click();const r=await retry;expect(r.status()).toBe(409);expect(r.request().headers()['idempotency-key']).toBe(key);
  await expect(page.getByRole('button',{name:'查看保存结果',exact:true})).toBeVisible();
 } else await page.reload();
 const resolve=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/resolve'));
 await page.getByRole('button',{name:'查看保存结果',exact:true}).click();const result=await(await resolve).json();expect(result.submissionId).toBe(key);expect(result.outcome).toBe(recovery==='retry'?'rejected':'not_applied');
 await expect(page.getByText('未保存',{exact:true})).toBeVisible();await expect(page.getByRole('button',{name:'查看保存结果',exact:true})).toHaveCount(0);
 expect(sql(`SELECT count(*) FROM organization.bu_terminal WHERE submission_id='${key}'`)).toBe('1');
 expect((await read(page)).headSourceVersion).toBe(changed.body.sourceVersion);expect((await read(page,undefined,original.headSourceVersion)).timeline).toEqual(original.timeline);
});

test('未来停用删除不提前改变今日状态，短屏确认可达且取消恢复焦点',async({page})=>{
 const today=await open(page);await create(page);const date=new Date(today+'T12:00:00Z');date.setUTCDate(date.getUTCDate()+2);const future=date.toISOString().slice(0,10);
 await life(page,'DEACTIVATE',future);const old=await read(page);await enterDelete(page);await readyToSave(page);await expect(page.getByText('今日状态：启用 → 启用',{exact:true})).toHaveCount(0);
 await page.setViewportSize({width:1440,height:540});await expect(page.getByRole('button',{name:'确认删除',exact:true})).toBeInViewport();
 await page.getByRole('button',{name:'取消编辑',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);await expect(page.getByRole('button',{name:'更多操作',exact:true})).toBeFocused();
 await enterDelete(page);await readyToSave(page);await confirm(page);expect((await read(page)).status).toBe('active');expect((await read(page,future,old.headSourceVersion)).selectedRecord.state).toBe('inactive');await expectCurrentDate(page,today);
});

test('同值前序删除仍显示期间变化并保存结构事实，查询者不开放删除入口',async({page})=>{
 await open(page);const code=await create(page);await add(page,'2026-03-01','乙','Beta');await life(page,'DEACTIVATE','2026-05-01');await life(page,'REACTIVATE','2026-07-01');
 await select(page,'2026-01-01');await openMoreActions(page);await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('乙');await page.getByRole('textbox',{name:'英文名称',exact:true}).fill('Beta');await readyToSave(page);await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);
 const old=await read(page);await select(page,'2026-03-01');await enterDelete(page);await readyToSave(page);await expect(page.getByText('没有修改内容。',{exact:true})).toHaveCount(0);await expect(page.getByText('拟保存时间线',{exact:true})).toHaveCount(0);const result=await confirm(page);expect(result.outcome).toBe('accepted');expect(result.impact.dataAffectedRanges).toHaveLength(0);expect(result.impact.structuralChanges).toHaveLength(3);
 const current=await read(page);expect(current.timeline.find((r:any)=>r.effectiveDate==='2026-07-01')).toEqual(old.timeline.find((r:any)=>r.effectiveDate==='2026-07-01'));
 await page.getByRole('button',{name:'退出',exact:true}).click();await page.getByLabel('登录名').fill(credentials.reader_login);await page.getByLabel('密码').fill(credentials.reader_password);await page.getByRole('button',{name:'登录',exact:true}).click();
 await page.getByRole('button',{name:'切换应用',exact:true}).click();await page.getByRole('menuitem',{name:'设置',exact:true}).click();await page.getByRole('button',{name:'组织配置',exact:true}).click();await page.getByRole('menuitem',{name:'业务单元',exact:true}).click();await page.getByLabel('搜索编码或名称').fill(code);await page.getByRole('button',{name:'查询',exact:true}).click();await page.getByRole('row').filter({hasText:code}).getByRole('link').click();
 await expect(page.getByRole('button',{name:'更多操作',exact:true})).toHaveCount(0);
});

for (const status of [200,409]) test(`删除辅助重试${status}时单独撤日期读权，清除目标缓存并重新核对`,async({page})=>{
 await open(page);await create(page);await add(page,'2026-06-01','乙');
 const helperPattern='**/api/v1/business-units/*/preview';
 await page.route(helperPattern,route=>route.fulfill({status:503,json:{message:'controlled assistance failure'}}));
 await enterDelete(page);await expect(page.getByRole('button',{name:'重试加载',exact:true})).toBeVisible();await page.unroute(helperPattern);
 if(status===409){
  const result=await page.evaluate(async()=>{
   const session=await(await fetch('/api/v1/session')).json();const id=location.pathname.split('/').at(-1);
   const current=await(await fetch('/api/v1/business-units/'+id)).json();
   return(await fetch('/api/v1/business-units/'+id+'/records',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':session.csrf,'Idempotency-Key':crypto.randomUUID()},body:JSON.stringify({expectedSourceVersion:current.headSourceVersion,anchorRecordId:current.timeline.at(-1).recordId,effectiveDate:'2026-10-01',operationType:'CHANGE',operationReason:'NAME_ADJUSTMENT',changes:{name:'丙'}})})).status;
  });expect(result).toBe(201);
 }
 const subject=await page.evaluate(async()=>(await(await fetch('/api/v1/session')).json()).subjectId);
 for(const id of [subject,credentials.instance])expect(id).toMatch(/^[0-9a-f-]{36}$/);
 const connection=JSON.parse(execFileSync('python3',['scripts/local-bu.py','connection'],{encoding:'utf8'}));
 const container=execFileSync('docker',['ps','-q','--filter','label=com.docker.compose.project='+connection.project,'--filter','label=com.docker.compose.service=postgres'],{encoding:'utf8'}).trim();expect(container).toMatch(/^[0-9a-f]+$/);
 const sql=(query:string)=>execFileSync('docker',['exec','-i',container,'psql','-U','postgres','-d','chnhcm','-X','-Atq','-v','ON_ERROR_STOP=1'],{input:query,encoding:'utf8'}).trim();
 const where=`instance_id='${credentials.instance}' AND principal_id='${subject}'`;
 const grants=sql(`SELECT json_agg(g) FROM (SELECT grant_id,read_fields FROM access.subject_grant WHERE ${where}) g;`);
 const mutate=(query:string)=>sql(`BEGIN;SELECT principal_id FROM access.policy_guard WHERE ${where} FOR UPDATE;${query};UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE ${where};COMMIT;`);
 try{
  mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'effectiveDate') WHERE ${where} AND action IN ('bu.read','bu.history.read')`);
  const response=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/preview'));
  await page.getByRole('button',{name:'重试加载',exact:true}).click();const r=await response;expect(r.status()).toBe(status);const safe=await r.json();if(status===200)expect(safe.afterRecord.effectiveDate).toBeUndefined();expect(safe.readFields).toBeDefined();expect(safe.readFields).not.toContain('effectiveDate');
  await expect(page.getByText('已重读当前资料，请核对后保存。',{exact:true})).toHaveCount(0);
  await expect(page.locator('body')).not.toContainText('2026-06-01');await expect(page.getByRole('button',{name:'确认删除',exact:true})).toBeEnabled();
  await readyToSave(page);await expect(page.getByText(/删除后适用资料：/)).not.toContainText('开放终点');await confirm(page);await expect(page.getByText('适用期间',{exact:true})).toHaveCount(0);
 }finally{mutate(`UPDATE access.subject_grant g SET read_fields=s.read_fields FROM json_to_recordset('${grants.replaceAll("'","''")}') AS s(grant_id uuid,read_fields text[]) WHERE g.instance_id='${credentials.instance}' AND g.grant_id=s.grant_id`);}
});
