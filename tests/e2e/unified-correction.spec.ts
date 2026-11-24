import { deferAutomaticConfirmation } from './fixtures';
import { expectLogChange } from './fixtures';
import {test,expect,openMoreActions} from './fixtures';
import {open,create,add,life,choose,select,read,expectSingleClassification} from './bu-support';

async function edit(page: Parameters<typeof open>[0]) {
  await openMoreActions(page);
  await expect(page.getByRole('menuitem',{name:'修改生效日期',exact:true})).toHaveCount(0);
  await expect(page.getByRole('menuitem',{name:'更正业务分类',exact:true})).toHaveCount(0);
  await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();
  await expect(page.getByRole('button',{name:'保存',exact:true})).toBeVisible();
  await expectSingleClassification(page);
}
async function save(page: Parameters<typeof open>[0], status=200) {
  const response=page.waitForResponse(r=>r.url().endsWith('/correct')&&r.request().method()==='POST');
  await page.getByRole('button',{name:'保存',exact:true}).click();
  const r=await response;expect(r.status()).toBe(status);
  if(status===200)await expect(page.locator('.object-form')).toHaveCount(0);
  return r.json();
}

test('统一入口一次保存名称日期分类，原日默认、旧来源及逐项日志准确',async({page})=>{
  await open(page);await create(page,'2025-12-01','Initial');await add(page,'2026-01-01','甲','Alpha');await add(page,'2026-06-01','乙','Beta');
  await select(page,'2026-01-01');const before=await read(page);const first=before.timeline[1];
  await edit(page);
  await expect(page.getByLabel('生效日期',{exact:true}).and(page.locator('input'))).toHaveValue('2026-01-01');
  await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('甲2');
  await page.getByRole('textbox',{name:'英文名称',exact:true}).fill('');
  await page.getByLabel('生效日期',{exact:true}).and(page.locator('input')).fill('2026-02-01');
  await choose(page,'操作原因','其他数据调整');
  await expect(page.getByRole('button',{name:'预览',exact:true})).toHaveCount(0);
  const saved=await save(page);const after=await read(page);
  expect(after.timeline).toHaveLength(3);expect(after.timeline[1]).toMatchObject({recordId:first.recordId,name:'甲2',effectiveDate:'2026-02-01',operationReason:'OTHER_DATA_ADJUSTMENT'});
  expect(after.timeline[1].nameEn).toBeUndefined();expect(after.timeline[2]).toMatchObject({name:'乙',nameEn:'Beta'});
  expect(saved.previousEffectiveDate).toBe('2026-01-01');
  expect((await read(page,undefined,before.headSourceVersion)).timeline).toEqual(before.timeline);
  await page.getByRole('tab',{name:'变更日志',exact:true}).click();const log=page.locator('.change-operation').first();
  await expectLogChange(page, '中文名称', '甲', '甲2');await expectLogChange(page, '英文名称', 'Alpha', '未提供');
  await expectLogChange(page, '生效日期', '2026-01-01', '2026-02-01');await expectLogChange(page, '操作原因', '名称调整', '其他数据调整');
});

test('普通更正日期只能在原邻居之间，越界组合整笔拒绝',async({page})=>{
  await open(page);await create(page);await add(page,'2026-06-01','乙');await add(page,'2026-10-01','丙');await select(page,'2026-06-01');
  const before=await read(page);await edit(page);const date=page.getByLabel('生效日期',{exact:true}).and(page.locator('input'));
  await expect(date).toHaveAttribute('min','2026-01-02');await expect(date).toHaveAttribute('max','2026-09-30');
  await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('乙2');await date.fill('2026-11-01');await save(page,400);
  await expect(date).toBeFocused();expect((await read(page)).headSourceVersion).toBe(before.headSourceVersion);
  await date.fill('2026-05-01');await save(page);expect((await read(page)).timeline[1]).toMatchObject({effectiveDate:'2026-05-01',name:'乙2'});
});

test('只有原日期可选时日期只读，名称仍可更正',async({page})=>{
  await open(page);await create(page,'2026-06-01');await add(page,'2026-06-02','乙');await add(page,'2026-06-03','丙');await select(page,'2026-06-02');
  await edit(page);const date=page.getByLabel('生效日期',{exact:true}).and(page.locator('input'));
  await expect(date).toHaveAttribute('readonly','');await expect(page.getByRole('button',{name:'保存',exact:true})).toBeEnabled();
  await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('乙2');await save(page);
  expect((await read(page)).timeline[1]).toMatchObject({effectiveDate:'2026-06-02',name:'乙2'});
});

test('分类初始直接显示且不算修改，真实修改保护且恢复初值可退出',async({page})=>{
  await open(page);await create(page);const before=await read(page);await edit(page);
  await expect(page.getByRole('button',{name:'业务分类',exact:true})).toHaveCount(0);
  await expect(page.getByRole('combobox',{name:'操作类型',exact:true})).toHaveCount(0);
  await expect(page.locator('.object-form .object-detail-field').filter({has:page.getByText('操作类型',{exact:true})})).toContainText('新建');
  const reason=page.getByRole('combobox',{name:'操作原因',exact:true});await expect(reason).toBeVisible();await expect(reason).toContainText('新设组织');
  let commands=0;page.on('request',r=>{if(r.method()==='POST'&&r.url().endsWith('/correct'))commands++});
  await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.getByRole('status').filter({hasText:'没有修改内容'})).toBeVisible();expect(commands).toBe(0);
  await page.getByRole('button',{name:'取消编辑',exact:true}).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);await expect(page.locator('.object-form')).toHaveCount(0);
  await edit(page);const date=page.getByLabel('生效日期',{exact:true}).and(page.locator('input'));await date.fill('2026-02-01');
  await page.getByRole('button',{name:'取消编辑',exact:true}).click();await page.getByRole('button',{name:'继续编辑',exact:true}).click();
  await date.fill('2026-01-01');await page.getByRole('button',{name:'取消编辑',exact:true}).click();
  await expect(page.locator('.object-form')).toHaveCount(0);await expect(page.getByRole('dialog')).toHaveCount(0);expect((await read(page)).headSourceVersion).toBe(before.headSourceVersion);
});

test('分类改回原值仅保存名称，不自动匹配分类或增加分类日志',async({page})=>{
  await open(page);await create(page);await add(page,'2026-06-01','乙');const before=await read(page);await edit(page);
  const type=page.getByRole('combobox',{name:'操作类型',exact:true}),reason=page.getByRole('combobox',{name:'操作原因',exact:true});
  await expect(type).toBeVisible();await expect(type).toContainText('数据更改');await expect(reason).toContainText('名称调整');
  await choose(page,'操作类型','停用');await expect(reason).not.toContainText('名称调整');
  await choose(page,'操作类型','数据更改');await choose(page,'操作原因','名称调整');
  await page.getByRole('button',{name:'取消编辑',exact:true}).click();await expect(page.getByRole('dialog')).toHaveCount(0);
  expect((await read(page)).headSourceVersion).toBe(before.headSourceVersion);
  await edit(page);await choose(page,'操作类型','停用');await choose(page,'操作类型','数据更改');await choose(page,'操作原因','名称调整');
  await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('只改名称');
  const request=page.waitForRequest(r=>r.url().endsWith('/correct')&&r.method()==='POST');await save(page);
  const body=(await request).postDataJSON();expect(body.changes).toEqual({name:'只改名称'});expect(body.classification).toBeUndefined();
  expect((await read(page)).selectedRecord).toMatchObject({name:'只改名称',operationType:'CHANGE',operationReason:'NAME_ADJUSTMENT'});
  await page.getByRole('tab',{name:'变更日志',exact:true}).click();const log=page.locator('.change-operation').first();
  await expectLogChange(page, '中文名称', '乙', '只改名称');await expect(log.locator('[data-label="变更字段"]')).toHaveText(['中文名称']);
});

for(const catalog of ['原原因不在候选','目录读取失败'] as const)test(`分类直接显示后${catalog}仍可仅更正名称`,async({page})=>{
  await open(page);await create(page);const before=await read(page);
  await page.route('**/api/v1/organization/operation-reasons?*',async route=>{
    if(catalog==='目录读取失败'){await route.fulfill({status:503,json:{code:'DEPENDENCY_UNAVAILABLE',message:'原因目录暂不可用'}});return;}
    const response=await route.fetch();const reasons=await response.json();await route.fulfill({response,json:reasons.filter((r:{code:string})=>r.code!=='NEW_ORGANIZATION')});
  });
  const loaded=page.waitForResponse(r=>r.url().includes('/operation-reasons?'));await edit(page);await loaded;
  await expect(page.getByRole('combobox',{name:'操作原因',exact:true})).toContainText('新设组织');
  await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('保留原分类');
  const request=page.waitForRequest(r=>r.url().endsWith('/correct')&&r.method()==='POST');await save(page);expect((await request).postDataJSON().classification).toBeUndefined();
  expect((await read(page)).timeline[0]).toMatchObject({name:'保留原分类',operationType:'CREATE',operationReason:'NEW_ORGANIZATION'});
  expect((await read(page,undefined,before.headSourceVersion)).timeline).toEqual(before.timeline);
});

test('分类更正造成孤立恢复正常拒绝，无关联编排入口',async({page})=>{await open(page);await create(page);await life(page,'DEACTIVATE','2026-04-01');await life(page,'REACTIVATE','2026-05-01','乙');await select(page,'2026-04-01');const before=await read(page);await edit(page);await choose(page,'操作类型','数据更改');await choose(page,'操作原因','名称调整');await save(page,409);await expect(page.getByRole('button',{name:'添加关联操作',exact:true})).toHaveCount(0);expect((await read(page)).headSourceVersion).toBe(before.headSourceVersion)});

test('统一更正响应丢失后按原引用恢复原日期与分类结果',async({page})=>{
  await deferAutomaticConfirmation(page);
  await open(page);await create(page,'2025-12-01','Initial');await add(page,'2026-01-01','甲','Alpha');await edit(page);await page.getByLabel('生效日期',{exact:true}).and(page.locator('input')).fill('2026-02-01');await choose(page,'操作原因','其他数据调整');
  const pattern='**/api/v1/business-units/*/records/*/correct';await page.route(pattern,async route=>{await route.fetch();await route.abort('connectionfailed')});
  await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.getByText(/暂时无法确认保存结果/,{exact:false})).toBeVisible();await page.unroute(pattern);await page.reload();
  await page.getByRole('button',{name:'查看保存结果',exact:true}).click();await page.getByRole('link',{name:'查看保存结果',exact:true}).click();await expect(page.getByText('正在查看只读历史资料',{exact:true})).toBeVisible();
  const result=await read(page);expect(result.timeline[1]).toMatchObject({effectiveDate:'2026-02-01',operationReason:'OTHER_DATA_ADJUSTMENT'});
});

test('转为停用明确移除名称输入，原件保持，缺分类原因定位字段',async({page})=>{
  await open(page);await create(page);await add(page,'2026-06-01','乙','Beta');const before=await read(page);await edit(page);
  const name=page.getByRole('textbox',{name:'中文名称',exact:true});await name.fill('保留输入');await choose(page,'操作类型','停用');
  await page.getByRole('button',{name:'保留当前输入',exact:true}).click();await expect(name).toHaveValue('保留输入');await choose(page,'操作类型','停用');await page.getByRole('button',{name:'移除名称修改并转为停用',exact:true}).click();
  await save(page,400);await expect(page.getByRole('combobox',{name:'操作原因',exact:true})).toBeFocused();await choose(page,'操作原因','业务终止');await save(page);
  expect((await read(page)).timeline[1]).toMatchObject({operationType:'DEACTIVATE',name:'甲',nameEn:'Alpha'});expect((await read(page,undefined,before.headSourceVersion)).timeline).toEqual(before.timeline);
});

test('冲突重读更新未编辑日期和分类，名称草稿不覆盖并发更正',async({page})=>{
  await open(page);await create(page,'2025-12-01','Initial');await add(page,'2026-01-01','甲','Alpha');await edit(page);
  await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('我的名称');
  const status=await page.evaluate(async()=>{
    const s=await(await fetch('/api/v1/session')).json();const base='/api/v1/business-units/'+location.pathname.split('/').at(-1);const d=await(await fetch(base)).json();const r=d.selectedRecord;
    return(await fetch(base+'/records/'+r.recordId+'/correct',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':s.csrf,'Idempotency-Key':crypto.randomUUID()},body:JSON.stringify({expectedSourceVersion:d.headSourceVersion,expectedRecordRevision:r.revision,newEffectiveDate:'2026-02-01',classification:{operationType:'CHANGE',operationReason:'OTHER_DATA_ADJUSTMENT'}})})).status;
  });expect(status).toBe(200);await save(page,409);await page.getByRole('button',{name:'重新核对当前资料',exact:true}).click();
  await expect(page.getByLabel('生效日期',{exact:true}).and(page.locator('input'))).toHaveValue('2026-02-01');await expect(page.getByRole('textbox',{name:'中文名称',exact:true})).toHaveValue('我的名称');
  await expect(page.getByRole('combobox',{name:'操作原因',exact:true})).toContainText('其他数据调整');
  const request=page.waitForRequest(r=>r.url().endsWith('/correct')&&r.method()==='POST');await save(page);const body=(await request).postDataJSON();expect(body.changes).toEqual({name:'我的名称'});expect(body.newEffectiveDate).toBeUndefined();expect(body.classification).toBeUndefined();
  expect((await read(page)).timeline[1]).toMatchObject({name:'我的名称',effectiveDate:'2026-02-01',operationReason:'OTHER_DATA_ADJUSTMENT'});
});

test('跨节点改日拒绝且不提供通用联合回退',async({page})=>{await open(page);await create(page);await add(page,'2026-06-01','乙');await add(page,'2026-10-01','丙');await select(page,'2026-06-01');const before=await read(page);await edit(page);await page.getByLabel('生效日期',{exact:true}).and(page.locator('input')).fill('2026-11-01');await save(page,400);await expect(page.getByRole('button',{name:'添加关联操作',exact:true})).toHaveCount(0);expect((await read(page)).headSourceVersion).toBe(before.headSourceVersion)});

test('日期写权撤回保留可读草稿，移除该修改后仍能保存名称',async({page})=>{
  await open(page);await create(page);await edit(page);await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('保留名称');
  const {execFileSync}=await import('node:child_process');const {readFileSync}=await import('node:fs');
  const creds=JSON.parse(readFileSync(execFileSync('python3',['scripts/local-bu.py','credentials-path'],{encoding:'utf8'}).trim(),'utf8'));
  const connection=JSON.parse(execFileSync('python3',['scripts/local-bu.py','connection'],{encoding:'utf8'}));const container=execFileSync('docker',['ps','-q','--filter','label=com.docker.compose.project='+connection.project,'--filter','label=com.docker.compose.service=postgres'],{encoding:'utf8'}).trim();
  const subject=await page.evaluate(async()=>(await(await fetch('/api/v1/session')).json()).subjectId);expect(subject).toMatch(/^[0-9a-f-]{36}$/);expect(container).toMatch(/^[0-9a-f]+$/);
  const sql=(query:string)=>execFileSync('docker',['exec','-i',container,'psql','-U','postgres','-d','chnhcm','-X','-Atq','-v','ON_ERROR_STOP=1'],{input:query,encoding:'utf8'}).trim();const where=`instance_id='${creds.instance}' AND principal_id='${subject}'`;
  const grants=sql(`SELECT json_agg(g) FROM (SELECT grant_id,write_fields FROM access.subject_grant WHERE ${where}) g;`);const mutate=(query:string)=>sql(`BEGIN;SELECT principal_id FROM access.policy_guard WHERE ${where} FOR UPDATE;${query};UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE ${where};COMMIT;`);
  try{
    mutate(`UPDATE access.subject_grant SET write_fields=array_remove(write_fields,'effectiveDate') WHERE ${where} AND action LIKE '%.redate'`);
    const date=page.getByLabel('生效日期',{exact:true}).and(page.locator('input'));await date.fill('2026-02-01');
    await expect(page.getByRole('button',{name:'撤销这些字段的修改',exact:true})).toBeVisible();await expect(date).toHaveValue('2026-02-01');await expect(date).toHaveAttribute('readonly','');await expect(page.getByRole('button',{name:'保存',exact:true})).toBeDisabled();
    await page.getByRole('button',{name:'撤销这些字段的修改',exact:true}).click();await expect(date).toHaveValue('2026-01-01');await expect(page.getByRole('textbox',{name:'中文名称',exact:true})).toHaveValue('保留名称');await save(page);expect((await read(page)).timeline[0]).toMatchObject({name:'保留名称',effectiveDate:'2026-01-01'});
  }finally{mutate(`UPDATE access.subject_grant g SET write_fields=s.write_fields FROM json_to_recordset('${grants.replaceAll("'","''")}') AS s(grant_id uuid,write_fields text[]) WHERE g.instance_id='${creds.instance}' AND g.grant_id=s.grant_id`);}
});


test('R1 旧分类原名称在详情和更正表单保持，普通更正不重解释目录',async({page})=>{
  await open(page);await create(page);await add(page,'2026-06-01','乙');const current=await read(page),target=current.selectedRecord;
  expect(target.revision).toMatch(/^[0-9a-f-]{36}$/);
  // The helper admits only this task's explicitly temporary browser database.
  const {execFileSync}=await import('node:child_process');
  execFileSync('go',['run','-tags=integration','./tests/e2e/history-fixture'],{input:JSON.stringify({Object:current.id,Source:current.headSourceVersion,Revision:target.revision}),encoding:'utf8'});
  {
    await page.reload();await expect(page.getByRole('button',{name:'更多操作',exact:true})).toBeEnabled();const historical=await read(page);
    expect(historical.selectedRecord).toMatchObject({operationType:'CHANGE',typeLabel:'资料变更',operationReason:'NAME_CHANGE',reasonLabel:'名称变更'});
    await edit(page);await expect(page.getByRole('combobox',{name:'操作类型',exact:true})).toContainText('资料变更');await expect(page.getByRole('combobox',{name:'操作原因',exact:true})).toContainText('名称变更');
    if(process.env.CHNHCM_EVIDENCE_DIR){const {mkdirSync}=await import('node:fs');const {join}=await import('node:path');mkdirSync(process.env.CHNHCM_EVIDENCE_DIR,{recursive:true});await page.screenshot({path:join(process.env.CHNHCM_EVIDENCE_DIR,'old-directory-edit.png'),mask:[page.locator('.app-account-name')]})}
    await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('旧分类下更正');await save(page);
    expect((await read(page)).selectedRecord).toMatchObject({name:'旧分类下更正',typeLabel:'资料变更',operationReason:'NAME_CHANGE',reasonLabel:'名称变更'});
    expect((await read(page,undefined,historical.headSourceVersion)).timeline).toEqual(historical.timeline);
  }
});
