import { readFile } from 'node:fs/promises';
import { withScopedGrants } from './scoped-grants';
import {checkPairPeriod} from './dated-object-support';
import { chooseApplication } from './app-navigation';
import { deferAutomaticConfirmation } from './fixtures';
import {randomUUID} from 'node:crypto';
import {test,expect,openMoreActions} from './fixtures';
import {open,choose} from './bu-support';
import {withPolicy} from './grants';
import {objects,departmentKind as dep,menu,reference,request,readObject,fillNew,createObject,apiBU,apiDepartment} from './dated-object-support';

for(const kind of objects) {
 test(`${kind.label}共享固定期间及退出失败后的草稿保护`,async({page})=>{
  test.setTimeout(90000);await open(page);const bu=kind===dep?await apiBU(page):undefined;
  await createObject(page,kind,{bu:bu?.code});
  await openMoreActions(page);await page.getByRole('menuitem',{name:'插入停用与重新启用',exact:true}).click();
  await page.locator('#pair-deactivationDate').fill('2026-06-01');await page.locator('#pair-reactivationDate').fill('2026-07-01');
  await choose(page,'停用原因','业务终止');await expect(page.locator('#pair-reactivationReason')).toHaveText('业务恢复');
  await page.getByRole('dialog').getByRole('button',{name:'保存',exact:true}).click();await expect(page.getByRole('dialog')).toHaveCount(0);
  const paired=await readObject(page,kind);expect(paired.timeline).toHaveLength(3);
  if (kind===dep) await checkPairPeriod(page,'departments','department','2026-07-01');
  await openMoreActions(page);await page.getByRole('menuitem',{name:'删除停用与重新启用',exact:true}).click();
  await page.getByRole('button',{name:'删除两条记录',exact:true}).click();await expect(page.getByRole('dialog')).toHaveCount(0);
  expect((await readObject(page,kind)).timeline).toHaveLength(1);
  await openMoreActions(page);await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();await page.locator('#record-date').fill('2026-02-01');
  await page.route('**/api/v1/session',route=>route.request().method()==='DELETE'?route.fulfill({status:503,json:{code:'SERVICE_UNAVAILABLE',message:'测试退出失败'}}):route.continue());
  await page.getByRole('button',{name:'退出',exact:true}).click();await page.getByRole('button',{name:'放弃修改',exact:true}).click();await expect(page.getByText('退出未完成，请重试',{exact:true})).toBeVisible();
  await page.getByRole('textbox',{name:kind.name,exact:true}).fill('退出失败后新增草稿');await page.getByRole('button',{name:'取消编辑',exact:true}).click();await expect(page.getByRole('dialog',{name:'尚有未保存修改',exact:true})).toBeVisible();await page.getByRole('button',{name:'继续编辑',exact:true}).click();
  expect(await page.evaluate(()=>{const e=new Event('beforeunload',{cancelable:true});window.dispatchEvent(e);return e.defaultPrevented})).toBe(true);
  await expect(page.getByRole('textbox',{name:kind.name,exact:true})).toHaveValue('退出失败后新增草稿');
 });
 test(`${kind.label}未来首条丢响应与原结果查回`,async({page})=>{
  await deferAutomaticConfirmation(page);
  test.setTimeout(60000);await open(page);const bu=kind===dep?await apiBU(page):undefined;
  await menu(page,kind);await fillNew(page,kind,{bu:bu?.code,date:'2027-01-01',name:'未来首条'});
  let accepted:any;await page.route(`**/api/v1/${kind.segment}`,async route=>{if(route.request().method()!=='POST')return route.continue();const response=await route.fetch();expect(response.status()).toBe(201);accepted=await response.json();await route.abort('connectionreset')});
  await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.getByRole('button',{name:'查看保存结果',exact:true})).toBeVisible();await page.reload();
  await page.getByRole('region',{name:`${kind.label}提交结果`,exact:true}).getByRole('button',{name:'查看保存结果',exact:true}).click();await page.getByRole('link',{name:'查看保存结果',exact:true}).click();
  await expect(page.getByText('正在查看只读历史资料',{exact:true})).toBeVisible();await expect(page.getByText('所选记录尚未生效。',{exact:true})).toBeVisible();
  const detail=await readObject(page,kind);expect(detail.resolvedSourceVersion).toBe(accepted.sourceVersion);expect(detail.selectedRecord.name).toBe('未来首条');
  await page.getByRole('button',{name:'返回当前资料',exact:true}).click();await expect(page.getByText('正在查看只读历史资料',{exact:true})).toHaveCount(0);
 });
}

test('部门父子关系、跨BU重选与同头显式来源的封存路径',async({page})=>{
 test.setTimeout(90000);await open(page);const a=await apiBU(page,'第一事业部'),b=await apiBU(page,'第二事业部');
 const parent=await createObject(page,dep,{bu:a.code,name:'行政上级原名'}),parentID=parent.departmentId;
 const child=await createObject(page,dep,{bu:a.code,parent:parent.code,name:'所属子部门'}),childURL=page.url();
 await expect(page.locator('.department-path')).toContainText('行政上级原名 / 所属子部门');
 const relations=page.locator('.object-detail-section').filter({has:page.getByRole('heading',{name:'组织关系',exact:true})});
 await expect(relations).toContainText('第一事业部');await expect(relations.getByRole('link')).toHaveCount(0);
 let d=(await request(page,'GET','/api/v1/departments/'+parentID)).body;
 const renamed=await request(page,'POST',`/api/v1/departments/${parentID}/records`,{expectedSourceVersion:d.headSourceVersion,anchorRecordId:d.selectedRecord.recordId,effectiveDate:'2026-07-01',operationType:'CHANGE',operationReason:'NAME_ADJUSTMENT',changes:{name:'行政上级七月名'}},randomUUID());expect(renamed.status).toBe(201);
 await page.reload();await expect(page.locator('.department-path')).toContainText('2026-01-01 至 2026-06-30');await expect(page.locator('.department-path')).toContainText('行政上级七月名');expect((await readObject(page,dep)).headSourceVersion).toBe(child.headSourceVersion);
 await page.getByRole('tab',{name:'变更日志',exact:true}).click();await page.getByRole('link',{name:'查看当次资料',exact:true}).first().click();
 await expect(page.locator('.department-path')).toContainText('行政上级原名');await expect(page.locator('.department-path')).not.toContainText('行政上级七月名');expect((await readObject(page,dep)).resolvedSourceVersion).toBe(child.headSourceVersion);
 await expect(relations.getByRole('link')).toHaveCount(0);
 await page.getByRole('button',{name:'返回当前资料',exact:true}).click();
 const other=await apiDepartment(page,b.id,{name:'第二事业部上级'});
 await openMoreActions(page);await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();await reference(page,'管理业务单元',b.code);
 await expect(page.getByRole('combobox',{name:'行政上级',exact:true})).toHaveValue(new RegExp(parent.code));
 await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form-error')).toBeVisible();await expect(page.getByRole('combobox',{name:'行政上级',exact:true})).toHaveValue(new RegExp(parent.code));
 await reference(page,'行政上级',other.code);await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);
 const reassigned=await readObject(page,dep);expect(reassigned.selectedRecord.managementBuId).toBe(b.id);expect(reassigned.selectedRecord.parentDepartmentId).toBe(other.id);expect(reassigned.selectedRecord.operationType).toBe('CREATE');
 await page.getByRole('tab',{name:'变更日志',exact:true}).click();await expect(page.locator('.change-operation').first()).toContainText('管理业务单元');await expect(page.locator('.change-operation').first()).toContainText('第二事业部');
 // A BU source pasted into the Department route cannot restore cached Department facts.
 const buDetail=(await request(page,'GET','/api/v1/business-units/'+a.id)).body;
 await page.goto(childURL.split('?')[0]+'?sourceVersion='+buDetail.headSourceVersion);await expect(page.locator('#department-detail-error')).toBeVisible();await expect(page.locator('.department-path')).toHaveCount(0);
 await page.goto(childURL.split('?')[0]+'?adoptedRef=unsupported');await expect(page.locator('#department-detail-error')).toContainText('未知的查询条件');await expect(page.locator('.department-path')).toHaveCount(0);
 await page.getByRole('button',{name:'返回当前资料',exact:true}).click();await expect(page.locator('.department-path')).toContainText('第二事业部上级');
});

test('部门摘要独立日志读权，关系及名称撤权清除旧投影',async({page})=>{
 test.setTimeout(60000);await open(page);const bu=await apiBU(page);const parent=await apiDepartment(page,bu.id,{name:'受控上级'});
 await createObject(page,dep,{bu:bu.code,parent:parent.code,name:'受控部门名称'});
 await withPolicy(page,async(where,mutate)=>{
  mutate(`DELETE FROM access.subject_grant WHERE ${where} AND action='department.history.read'`);await page.reload();await expect(page.getByRole('tab',{name:'变更日志',exact:true})).toHaveCount(0);await expect(page.locator('.operation-summary')).toContainText('操作者：');await expect(page.locator('.department-path')).toContainText('受控上级');
  mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE ${where} AND action='department.reference.read'`);await page.reload();await expect(page.locator('.department-path')).toHaveCount(0);await expect(page.locator('.object-detail-field').filter({has:page.getByText('部门路径',{exact:true})})).toContainText('无读取权限');
  await openMoreActions(page);await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();await page.getByRole('textbox',{name:'部门名称',exact:true}).fill('尚未保存的敏感输入');
  mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE ${where} AND action='department.read'`);
  await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.getByRole('textbox',{name:'部门名称',exact:true})).toHaveCount(0);await expect(page.getByText('尚未保存的敏感输入',{exact:false})).toHaveCount(0);await expect(page.getByRole('heading').filter({hasText:'受控部门名称'})).toHaveCount(0);
 });
});

test('部门停用后原上级转BU，重新启用保留原关系并要求有效重选',async({page})=>{
 test.setTimeout(90000);await open(page);const a=await apiBU(page),b=await apiBU(page),parent=await apiDepartment(page,a.id,{name:'原行政上级'}),replacement=await apiDepartment(page,a.id,{name:'可恢复上级'});
 await createObject(page,dep,{bu:a.code,parent:parent.code,name:'待恢复子部门'});
 await openMoreActions(page);await page.getByRole('menuitem',{name:'停用部门',exact:true}).click();await page.locator('#record-date').fill('2026-06-01');await choose(page,'操作原因','业务终止');await page.getByRole('button',{name:'确认停用',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);
 const d=(await request(page,'GET','/api/v1/departments/'+parent.id)).body;
 expect((await request(page,'POST',`/api/v1/departments/${parent.id}/records`,{expectedSourceVersion:d.headSourceVersion,anchorRecordId:d.selectedRecord.recordId,effectiveDate:'2026-07-01',operationType:'CHANGE',operationReason:'ORGANIZATION_REASSIGNMENT',changes:{managementBuId:b.id}},randomUUID())).status).toBe(201);
 await page.reload();await expect(page.locator('.department-path')).not.toContainText('停用前路径');await expect(page.locator('.department-path')).toContainText('原行政上级');
 await openMoreActions(page);await page.getByRole('menuitem',{name:'重新启用部门',exact:true}).click();await page.locator('#record-date').fill('2026-08-01');await expect(page.locator('#record-reason')).toHaveText('业务恢复');
 await expect(page.getByRole('combobox',{name:'行政上级',exact:true})).toHaveValue(new RegExp(parent.code));
 await expect(page.locator('.object-form-error')).toBeVisible();await reference(page,'行政上级',replacement.code);await page.getByRole('button',{name:'确认重新启用',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);expect((await readObject(page,dep)).selectedRecord.parentDepartmentId).toBe(replacement.id);
});

test('部门拟日期与分类刷新候选，迟到旧查询不覆盖新候选',async({page})=>{
 test.setTimeout(90000);await open(page);const bu=await apiBU(page);const later=await apiDepartment(page,bu.id,{name:'七月才生效上级',date:'2026-07-01'});
 await createObject(page,dep,{bu:bu.code,name:'候选变化部门'});await openMoreActions(page);await page.getByRole('menuitem',{name:'停用部门',exact:true}).click();await page.locator('#record-date').fill('2026-06-01');await choose(page,'操作原因','业务终止');await page.getByRole('button',{name:'确认停用',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);
 await openMoreActions(page);await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();await choose(page,'操作类型','数据更改');await choose(page,'操作原因','变更归属组织');
 let release!:()=>void,held=false;const hold=new Promise<void>(r=>release=r);
 await page.route('**/api/v1/departments/references/parents',async route=>{const body=route.request().postDataJSON();if(!held&&body.q==='七月才生效上级'&&!body.context.newEffectiveDate){held=true;const response=await route.fetch();expect((await response.json()).items).toHaveLength(0);await hold;await route.fulfill({response}).catch(()=>{});}else await route.continue()});
 const field=page.getByRole('combobox',{name:'行政上级',exact:true});await field.click();await field.fill('七月才生效上级');await expect.poll(()=>held).toBe(true);
 await page.locator('#record-date').fill('2026-08-01');await field.click();await field.fill('七月才生效上级');await expect(page.getByRole('option').filter({hasText:later.code})).toBeVisible();release();await page.waitForTimeout(300);await expect(page.getByRole('option').filter({hasText:later.code})).toBeVisible();await page.getByRole('option').filter({hasText:later.code}).click();
 await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);const saved=await readObject(page,dep);expect(saved.selectedRecord.operationType).toBe('CHANGE');expect(saved.selectedRecord.effectiveDate).toBe('2026-08-01');expect(saved.selectedRecord.parentDepartmentId).toBe(later.id);
});

test('两对象相同提交号查回隔离，跨会话隐藏原主体待确认引用',async({page})=>{
  await deferAutomaticConfirmation(page);
 test.setTimeout(120000);await open(page);const bu=await apiBU(page),submission=randomUUID();
 await page.evaluate(value=>{crypto.randomUUID=()=>value as `${string}-${string}-${string}-${string}-${string}`},submission);
 for(const kind of objects){
  if(kind===dep){await chooseApplication(page,'部门');await page.getByRole('button',{name: /^(放弃修改|离开此页)$/,exact:true}).click();await expect(page.getByRole('heading',{name:kind.label,exact:true})).toBeVisible();}else await menu(page,kind);
  await fillNew(page,kind,{name:`${kind.label}隔离提交`,bu:kind===dep?bu.code:undefined});
  await page.route(`**/api/v1/${kind.segment}`,async route=>{if(route.request().method()!=='POST')return route.continue();const response=await route.fetch();expect(response.status()).toBe(201);expect((await response.json()).submissionId).toBe(submission);await route.abort('connectionreset')});
  await page.getByRole('button',{name:'保存',exact:true}).click();
  await expect(page.getByRole('region',{name:`${kind.label}提交结果`,exact:true}).getByRole('button',{name:'查看保存结果',exact:true})).toBeEnabled();
 }
 const storage=await page.evaluate(()=>[localStorage.getItem('chnhcm-pending-submissions'),localStorage.getItem('chnhcm-pending-department-submissions')]);for(const raw of storage){expect(raw).toContain(submission);expect(raw).not.toContain('隔离提交');expect(raw).not.toContain('csrf')}
 await page.getByRole('button',{name:'退出',exact:true}).click();await page.getByRole('button',{name: /^(放弃修改|离开此页)$/,exact:true}).click();await expect(page.getByRole('button',{name:'登录',exact:true})).toBeEnabled();
 await page.waitForTimeout(3100);await page.getByLabel('登录名').fill('reader');await page.getByLabel('密码').fill('admin1234567');await page.getByRole('button',{name:'登录',exact:true}).click();await expect(page.getByRole('button',{name:'退出',exact:true})).toBeVisible();await expect(page.getByRole('button',{name:'查看保存结果',exact:true})).toHaveCount(0);
 await page.getByRole('button',{name:'退出',exact:true}).click();await expect(page.getByRole('button',{name:'登录',exact:true})).toBeEnabled();await page.waitForTimeout(3100);await page.getByLabel('登录名').fill('admin');await page.getByLabel('密码').fill('admin1234567');await page.getByRole('button',{name:'登录',exact:true}).click();await expect(page.getByRole('button',{name:'查看保存结果',exact:true})).toHaveCount(2);
 for(const kind of objects){const region=page.getByRole('region',{name:`${kind.label}提交结果`,exact:true});await region.getByRole('button',{name:'查看保存结果',exact:true}).click();await region.getByRole('link',{name:'查看保存结果',exact:true}).click();await expect(page).toHaveURL(new RegExp('/'+kind.segment+'/'));expect((await readObject(page,kind)).selectedRecord.name).toBe(`${kind.label}隔离提交`)}
});

test('306 部门路径只显示真实集团层级，当前停用未来表达一致', async ({page}) => {
 test.setTimeout(90000); await open(page);
 const bu = await apiBU(page), enterprise = (await request(page,'GET','/api/v1/departments/context')).body.enterprise.name;
 const group = await apiDepartment(page,bu.id,{name:enterprise});
 const rows = [];
 for (const [name,date] of [['当前路径306','2026-01-01'],['停用路径306','2026-01-01'],['未来路径306','2099-12-01']]) {
  const created = await apiDepartment(page,bu.id,{name,date,parent:group.id});
  if(name==='停用路径306') {
   const d=(await request(page,'GET','/api/v1/departments/'+created.id)).body;
   expect((await request(page,'POST',`/api/v1/departments/${created.id}/deactivate`,{expectedSourceVersion:d.headSourceVersion,anchorRecordId:d.selectedRecord.recordId,effectiveDate:'2026-10-01',operationType:'DEACTIVATE',operationReason:'ORG_RESTRUCTURING'},randomUUID())).status).toBe(201);
  }
  rows.push({...created,name});
 }
 for(const row of rows) {
  await menu(page,dep);
  await page.getByRole('textbox',{name:'搜索编码或名称',exact:true}).fill(row.code);
  await page.getByRole('button',{name:'查询',exact:true}).click();
  await page.getByRole('grid',{name:'部门列表',exact:true}).locator('.MuiDataGrid-row').filter({hasText:row.code}).getByRole('link').first().click();
  await expect(page.locator('.department-path')).toHaveText(`${enterprise} / ${row.name}`);
  await expect(page.locator('.department-path')).not.toContainText(/未生效|停用前路径/);
 }
});

test('027 甲 BU 名称更正、乙 BU 只读、历史与固定选择逐份导出', async ({ page }) => {
  test.setTimeout(90000);
  await open(page);
  const prefix = 'X03' + randomUUID().slice(0, 8);
  const a = await apiBU(page), b = await apiBU(page);
  const left = await apiDepartment(page, a.id, { code: prefix + 'A', name: prefix + '甲部门' });
  const right = await apiDepartment(page, b.id, { code: prefix + 'B', name: prefix + '乙部门' });
  const hidden = await apiDepartment(page, (await apiBU(page)).id, { code: prefix + 'C', name: prefix + '未授部门' });
  const base = '/api/v1/departments';
  await page.context().clearCookies();
  await open(page, 'reader');
  await withScopedGrants(page, async publish => {
    const read = ['code', 'name', 'effectiveDate', 'status'];
    publish({ scope: 'bu', target: a.id, actions: ['department.read', 'department.history.read', 'department.record.correct'], read, write: ['name'] }).issue();
    publish({ scope: 'bu', target: b.id, actions: ['department.read'], read: ['code', 'name'] }).issue();
    const search = async (q: string) => {
      await page.getByRole('textbox', { name: '搜索编码或名称', exact: true }).fill(q);
      await page.getByRole('button', { name: '查询', exact: true }).click();
    };
    await menu(page, dep);
    await search(prefix);
    await expect(page.getByRole('link', { name: prefix + '甲部门' })).toBeVisible();
    await expect(page.getByRole('link', { name: prefix + '乙部门' })).toBeVisible();
    await expect(page.getByText(hidden.code, { exact: true })).toHaveCount(0);
    await page.getByRole('link', { name: prefix + '乙部门' }).click();
    await expect(page.getByRole('tab', { name: '变更日志', exact: true })).toHaveCount(0);
    expect((await request(page, 'GET', base + '/' + right.id + '/changes')).status).toBe(403);
    const detail = await readObject(page, dep);
    expect(detail.selectedRecord).not.toHaveProperty('managementBuId');
    const denied = await request(page, 'POST', base + '/' + right.id + '/records/' + detail.selectedRecord.recordId + '/correct', {
      expectedSourceVersion: detail.headSourceVersion, expectedRecordRevision: detail.selectedRecord.revision, changes: { name: '乙部门不应改变' }
    }, randomUUID());
    expect(denied.status).toBe(403);
    expect((await readObject(page, dep)).headSourceVersion).toBe(detail.headSourceVersion);
    await page.getByRole('link', { name: '返回列表', exact: true }).click();
    await page.getByRole('link', { name: prefix + '甲部门' }).click();
    const before = (await readObject(page, dep)).headSourceVersion;
    await openMoreActions(page);
    await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
    const name = page.getByRole('textbox', { name: '部门名称', exact: true });
    await expect(name).toHaveValue(prefix + '甲部门');
    await page.getByRole('button', { name: '取消编辑', exact: true }).click();
    expect((await readObject(page, dep)).headSourceVersion).toBe(before);
    await openMoreActions(page);
    await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
    const corrected = prefix + '甲更正';
    await name.fill(corrected);
    await page.getByRole('button', { name: '保存', exact: true }).click();
    await expect(page.getByRole('link', { name: '返回列表', exact: true })).toBeVisible();
    await page.getByRole('tab', { name: '变更日志', exact: true }).click();
    const log = page.getByRole('table', { name: '变更日志', exact: true });
    await expect(log).toContainText(corrected);
    const returned = await request(page, 'GET', base + '/' + left.id + '/changes');
    expect(returned.status).toBe(200);
    const original = await request(page, 'GET', base + '/' + left.id + '?sourceVersion=' + before);
    expect(original.status).toBe(200);
    expect(original.body.selectedRecord.name).toBe(prefix + '甲部门');
    expect(original.body.selectedRecord).not.toHaveProperty('managementBuId');
    await page.getByRole('link', { name: '返回列表', exact: true }).click();
    const head = page.getByRole('checkbox', { name: /全选查询结果/ });
    await head.check();
    const exportButton = page.getByRole('button', { name: '导出所选', exact: true });
    await expect(exportButton).toBeDisabled();
    await page.getByRole('button', { name: '取消选择', exact: true }).click();
    const exp = (bu: string) => publish({ scope: 'bu', target: bu, actions: ['department.export'], read: [] });
    const aExport = exp(a.id);
    aExport.issue();
    const proxy = aExport.issue('proxy');
    const revokeB = exp(b.id).issue();
    await head.check();
    await expect(page.getByText('已选择2项', { exact: true })).toBeVisible();
    proxy();
    const download = page.waitForEvent('download');
    await exportButton.click();
    const csv = await readFile((await (await download).path())!, 'utf8');
    const lines = csv.replace(/^\uFEFF/, '').trim().split(/\r?\n/);
    expect(lines).toHaveLength(3);
    expect(lines[1]).toContain(corrected);
    expect(lines[2].split(',')).toEqual([right.code, prefix + '乙部门', '', '', '', '', '', '']);
    expect(csv).not.toContain(hidden.code);
    const laterDownloads: string[] = [];
    page.on('download', d => laterDownloads.push(d.suggestedFilename()));
    revokeB();
    const refusal = page.waitForResponse(r => new URL(r.url()).pathname === base + '/export');
    await exportButton.click();
    const response = await refusal;
    expect(response.status()).toBe(403);
    expect(response.headers()['content-disposition']).toBeUndefined();
    await expect(page.getByRole('toolbar', { name: '所选部门', exact: true })).toHaveCount(0);
    expect(laterDownloads).toEqual([]);
  });
});
