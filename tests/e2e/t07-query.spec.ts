import {test,expect,openMoreActions,applyListOption,expectCurrentDate,type Page} from './fixtures';
import {open,create,add,life,readyToSave,read,select} from './bu-support';
import {randomUUID} from 'node:crypto';
import {execFileSync} from 'node:child_process';
import {readFileSync} from 'node:fs';

async function menuCreate(page:Page,code:string,date:string,name='甲'){
 await page.getByRole('link',{name:'新建',exact:true}).click();await page.getByRole('textbox',{name:'编码',exact:true}).fill(code);await page.getByRole('textbox',{name:'中文名称',exact:true}).fill(name);await page.getByRole('textbox',{name:'英文名称',exact:true}).fill('Alpha');await page.getByLabel('生效日期').and(page.locator('input')).fill(date);await page.getByRole('combobox',{name:'操作原因',exact:true}).click();await page.getByRole('option',{name:'新设组织',exact:true}).click();await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);
}
async function option(page:Page,label:string,value:string){await page.getByRole('combobox',{name:label,exact:true}).click();await page.getByRole('option',{name:value,exact:true}).click()}

test('TL019 今日停用与未来首条的四类筛选，不需要查看日期',async({page})=>{
 test.setTimeout(120000);await open(page);const prefix='T07Q_'+randomUUID().slice(0,8);
 await menuCreate(page,prefix+'_F3','2026-01-01');await life(page,'DEACTIVATE','2026-04-01');await life(page,'REACTIVATE','9998-01-01','乙','Beta');await page.getByRole('link',{name:'返回列表',exact:true}).click();
 await menuCreate(page,prefix+'_F6','9998-01-01','乙');await page.getByRole('link',{name:'返回列表',exact:true}).click();await page.getByLabel('搜索编码或名称').fill(prefix);await page.getByRole('button',{name:'查询',exact:true}).click();
 for(const [status,ids] of Object.entries({全部:['F3','F6'],启用:[],停用:['F3'],未生效:['F6']})){
  const response=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1/business-units');await applyListOption(page,'状态',status);await response;
  const rows=page.getByRole('row').filter({hasText:prefix});await expect(rows).toHaveCount(ids.length);for(const id of ids)await expect(rows.filter({hasText:prefix+'_'+id})).toHaveCount(1);
 }
});

test('TL020 选择十月记录，新增与更正取消保持明确记录',async({page})=>{
 await open(page);await create(page);await add(page,'2026-06-01','乙','Beta');await add(page,'2026-10-01','丙','Gamma');
 await select(page,'2026-10-01');
 expect((await read(page,'2026-03-15')).selectedRecord.name).toBe('甲');await expect(page.getByText('丙',{exact:true}).first()).toBeVisible();await page.getByRole('button',{name:/^(新增记录|插入记录)$/,exact:true}).click();await expect(page.getByLabel('生效日期').and(page.locator('input'))).toHaveValue((await read(page)).currentDate);await expect(page.getByRole('textbox',{name:'中文名称',exact:true})).toHaveValue('丙');await page.getByRole('button',{name:'取消编辑',exact:true}).click();
 await openMoreActions(page);await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();await expect(page.getByLabel('生效日期').and(page.locator('input'))).toHaveValue('2026-10-01');await page.getByRole('button',{name:'取消编辑',exact:true}).click();await expect(page.getByRole('button',{name:'2026-10-01',exact:true})).toHaveAttribute('aria-current','date');expect((await read(page,'2026-03-15')).selectedRecord.name).toBe('甲');
});

test('TL029 被删原件从确切历史返回当前，不能继续维护原目标',async({page})=>{
 await open(page);await create(page);await add(page,'2026-06-01','乙','Beta');const old=await read(page);const target=old.timeline.find((r:any)=>r.effectiveDate==='2026-06-01');
 await select(page,'2026-06-01');await openMoreActions(page);await page.getByRole('menuitem',{name:'删除此记录',exact:true}).click();await readyToSave(page);await page.getByRole('button',{name:'确认删除',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);
 await page.getByRole('tab',{name:'变更日志',exact:true}).click();await page.locator(`.change-operation[data-source-version="${old.headSourceVersion}"] a[href*="selectedRecordId=${old.selectedRecord.recordId}"]`).filter({hasText:'查看当次资料'}).click();await expect(page.getByText('正在查看只读历史资料',{exact:true})).toBeVisible();expect(new URL(page.url()).searchParams.get('sourceVersion')).toBe(old.headSourceVersion);await expect(page.getByText('乙',{exact:true}).first()).toBeVisible();
 await page.getByRole('button',{name:'返回当前资料',exact:true}).click();await expect(page.getByText('当前资料中没有所选记录，请重新选择。',{exact:true})).toBeVisible();await expect(page.getByRole('button',{name:'更多操作',exact:true})).toHaveCount(0);const d=await read(page);expect(d.timeline.some((r:any)=>r.recordId===target.recordId)).toBe(false);expect((await read(page,undefined,old.headSourceVersion)).timeline).toEqual(old.timeline);
});

test('TL032 原来源读取503或撤历史权后清正文，恢复后仍读原来源',async({page})=>{
 test.setTimeout(60000);await open(page);await create(page);await add(page,'2026-06-01','乙','Beta');const old=await read(page);await correctName(page,'2026-06-01','新');
 await page.getByRole('tab',{name:'变更日志',exact:true}).click();await page.locator(`.change-operation[data-source-version="${old.headSourceVersion}"] a[href*="selectedRecordId=${old.selectedRecord.recordId}"]`).filter({hasText:'查看当次资料'}).click();await expect(page.getByText('正在查看只读历史资料',{exact:true})).toBeVisible();expect(new URL(page.url()).searchParams.get('sourceVersion')).toBe(old.headSourceVersion);
 const pattern='**/api/v1/business-units/*?*sourceVersion=*';await page.route(pattern,route=>route.fulfill({status:503,json:{code:'M02_DEPENDENCY_UNAVAILABLE',message:'受控来源读取中断'}}));await page.reload();await expect(page.getByText('受控来源读取中断',{exact:true})).toBeVisible();await expect(page.getByRole('heading',{name:'基本资料',exact:true})).toHaveCount(0);expect(new URL(page.url()).searchParams.get('sourceVersion')).toBe(old.headSourceVersion);await page.unroute(pattern);await page.reload();await expect(page.getByText('乙',{exact:true}).first()).toBeVisible();
 const subject=await page.evaluate(async()=>(await(await fetch('/api/v1/session')).json()).subjectId);const cp=execFileSync('python3',['scripts/local-bu.py','credentials-path'],{encoding:'utf8'}).trim();const creds=JSON.parse(readFileSync(cp,'utf8'));const key=cp.split('/').at(-2)!;expect(key).toMatch(/^[0-9a-f]{12}$/);for(const id of [subject,creds.instance])expect(id).toMatch(/^[0-9a-f-]{36}$/);
 const container=execFileSync('docker',['ps','-q','--filter',`label=com.docker.compose.project=chnhcm-bu-${key}`,'--filter','label=com.docker.compose.service=postgres'],{encoding:'utf8'}).trim();expect(container).toMatch(/^[0-9a-f]+$/);
 const sql=(q:string)=>execFileSync('docker',['exec','-i',container,'psql','-U','postgres','-d','chnhcm','-X','-Atq','-v','ON_ERROR_STOP=1'],{input:q,encoding:'utf8'}).trim();const where=`instance_id='${creds.instance}' AND principal_id='${subject}'`;const grants=sql(`SELECT json_agg(g) FROM access.subject_grant g WHERE ${where} AND action='bu.history.read'`);const mutate=(q:string)=>sql(`BEGIN;SELECT principal_id FROM access.policy_guard WHERE ${where} FOR UPDATE;${q};UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE ${where};COMMIT;`);
 try{mutate(`DELETE FROM access.subject_grant WHERE ${where} AND action='bu.history.read'`);await page.reload();await expect(page.getByText('当前权限不允许读取历史',{exact:true})).toBeVisible();await expect(page.getByRole('heading',{name:'基本资料',exact:true})).toHaveCount(0);expect(new URL(page.url()).searchParams.get('sourceVersion')).toBe(old.headSourceVersion)}finally{mutate(`INSERT INTO access.subject_grant SELECT * FROM json_populate_recordset(NULL::access.subject_grant,'${grants.replaceAll("'","''")}')`)}
 await page.reload();await expect(page.getByText('乙',{exact:true}).first()).toBeVisible();expect((await read(page,undefined,old.headSourceVersion)).timeline).toEqual(old.timeline);
});
async function correctName(page:Page,date:string,name:string){await select(page,date);await openMoreActions(page);await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();await page.getByRole('textbox',{name:'中文名称',exact:true}).fill(name);await readyToSave(page);await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0)}

test('TL005 合法闰日从正常菜单创建并准确回读',async({page})=>{await open(page);await create(page,'2028-02-29');const d=await read(page,'2028-02-29');expect(d.timeline[0].effectiveDate).toBe('2028-02-29');expect(d.selectedRecord.name).toBe('甲');await expect(page.getByRole('button',{name:'2028-02-29',exact:true})).toBeVisible()});

test('TL030 列表搜索百分号和下划线按字面处理',async({page})=>{
 await open(page);const code='T07_LITERAL_'+randomUUID().slice(0,8);await menuCreate(page,code,'2026-01-01','甲%_乙');await page.getByRole('link',{name:'返回列表',exact:true}).click();for(const q of ['甲%','%_','_乙']){await page.getByLabel('搜索编码或名称').fill(q);await page.getByRole('button',{name:'查询',exact:true}).click();await expect(page.getByRole('row').filter({hasText:code})).toHaveCount(1)}
});

test('TL013 Go默认唯一创建原因，直接保存并回读名称快照',async({page})=>{
 await open(page);await page.getByRole('link',{name:'新建',exact:true}).click();
 await page.getByRole('textbox',{name:'编码',exact:true}).fill('T07_REASON_'+randomUUID().slice(0,8));await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('新设组织案例');await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-01-01');
 await expect(page.getByRole('combobox',{name:'操作原因',exact:true})).toContainText('新设组织');await expect(page.getByText('请选择操作原因。',{exact:true})).toHaveCount(0);await expect(page.getByRole('button',{name:'保存',exact:true})).toBeEnabled();
 await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);const d=await read(page);expect(d.timeline[0].operationReason).toBe('NEW_ORGANIZATION');expect(d.timeline[0].reasonLabel).toBe('新设组织');
});

test('TL257 正常菜单不开放整BU作废、恢复已删记录或脚本导入',async({page})=>{
 await open(page);await create(page);await add(page,'2026-06-01','乙');const old=await read(page);await select(page,'2026-06-01');await openMoreActions(page);await page.getByRole('menuitem',{name:'删除此记录',exact:true}).click();await readyToSave(page);await page.getByRole('button',{name:'确认删除',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);await select(page,'2026-01-01');
 await openMoreActions(page);await expect(page.getByRole('menuitem',{name:/作废|恢复已删|导入|脚本/})).toHaveCount(0);await page.keyboard.press('Escape');expect((await read(page,undefined,old.headSourceVersion)).timeline).toEqual(old.timeline);
 await page.getByRole('link',{name:'返回列表',exact:true}).click();for(const role of ['button','link'] as const)await expect(page.getByRole(role,{name:/作废|恢复已删|导入|脚本/})).toHaveCount(0);
});
