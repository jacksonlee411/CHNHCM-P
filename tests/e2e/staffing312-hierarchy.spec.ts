import { desktopViewportCases } from './desktop-viewports.mjs';
import { randomUUID } from 'node:crypto';
import { writeFile } from 'node:fs/promises';
import { expect, test, type Page } from './fixtures';
import { request } from './dated-object-support';
import { open } from './bu-support';
import { withPolicy } from './grants';
import { namedMenu308, seed308 } from './staffing308-support';
import { hierarchyJourney,seedHierarchy,structureMenu,structureDetail,queryStructure,adjust,chooseParent } from './staffing312-hierarchy-support';
import { staffingZoomCases } from './staffing-zoom-suite';

for(const role of ['admin','reader']) for(const [cell,width,height] of desktopViewportCases){
 test(`312 hierarchy ${role} ${cell} 管理汇报日期与版本`,async({page})=>{
   test.setTimeout(180000);await open(page);const family=await seed308(page,'jobfamily'),sample=await seedHierarchy(page,'hierarchy',family.id);
   if(role==='reader'){await page.context().clearCookies();await open(page,role)}
   await page.setViewportSize({width,height});await hierarchyJourney(page,sample,family,role);
   await page.screenshot({path:test.info().outputPath('hierarchy.png'),mask:[page.locator('.app-account-name')]});
 });
}
staffingZoomCases(['hierarchy'] as const,'312',hierarchyJourney,seedHierarchy);

const hierarchyApi = '/api/v1/position-hierarchies/reporting';
const recoveryApi = '**/api/v1/position-hierarchies/submissions/*/resolve';
const effectiveDate = '2027-04-01';

test('139 查询者聚焦结构日期后收到默认日期仍可输入并查询', async ({page}) => {
  await open(page,'reader');
  let release!:()=>void;
  const pending=new Promise<void>(resolve=>{release=resolve});
  await page.route('**/position-hierarchies/reporting/structure?**',async route=>{
    if (!new URL(route.request().url()).searchParams.has('businessAt')) await pending;
    await route.continue();
  });
  const date=page.getByRole('textbox',{name:/结构日期/});
  try {
    await structureMenu(page);
    await date.focus();
    await expect(date).toHaveValue('');
  } finally { release() }
  await expect(page.locator('.standard-page')).toHaveAttribute('aria-busy','false');
  await expect(date).toBeFocused();
  await page.keyboard.insertText(effectiveDate);
  await expect(date).toHaveValue(effectiveDate);
  await page.getByRole('button',{name:'查询结构'}).click();
  await expect(page).toHaveURL(url=>url.searchParams.get('businessAt')===effectiveDate);
});

test('139 草稿日期取消后回到原结构日', async ({page}) => {
  await open(page);
  const family = await seed308(page, 'jobfamily');
  const sample=await seedHierarchy(page,'hierarchy',family.id);
  await structureMenu(page);await queryStructure(page,effectiveDate);await structureDetail(page,sample.child.name);await adjust(page);
  const date=page.getByRole('textbox',{name:/结构日期/});
  await page.getByRole('textbox', {name:/生效日期/}).fill('2027-05-01');
  await page.getByRole('button', {name:'取消', exact:true}).click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.getByRole('button', {name:'放弃修改'}).click();
  await expect(date).toHaveValue(effectiveDate);
});

test('139 办理岗位固定不因换岗丢失草稿', async ({page}) => {
  const {child, parent, sample} = await draftParentChange(page);
  await expect(child).toHaveCount(0);
  await page.getByRole('button', {name:'取消', exact:true}).click();
  await page.getByRole('button', {name:'继续编辑'}).click();
  await expect(parent).toHaveValue(new RegExp('^' + sample.second.name));
});

test('139 返回结构恢复查询日期', async ({page}) => {
  await open(page);
  let release!:()=>void, received!:()=>void;
  const responseGate=new Promise<void>(resolve=>{release=resolve});
  const started=new Promise<void>(resolve=>{received=resolve});
  await page.route('**/position-hierarchies/reporting/structure?**',async route=>{
    const url=new URL(route.request().url());
    if(url.searchParams.has('businessAt')){await route.continue();return}
    const response=await route.fetch();received();await responseGate;await route.fulfill({response});
  });
  await structureMenu(page);
  const purpose = page.getByRole('combobox', {name:/用途/});
  const date = page.getByRole('textbox', {name:/结构日期/});
  await started;
  await date.fill(effectiveDate);
  release();
  await expect(page.locator('.standard-page')).toHaveAttribute('aria-busy','false');
  await expect(date).toHaveValue(effectiveDate);
  await expect(purpose).toHaveCount(0);
  await date.fill(effectiveDate);
  await page.getByRole('button', {name:'查询结构'}).click();
  await expect(page).toHaveURL(/businessAt=2027-04-01/);
  await expect(page.locator('.standard-page')).toHaveAttribute('aria-busy','false');
  await page.getByRole('button', {name:'职务配置',exact:true}).click();
  await page.getByRole('menuitem', {name:'岗位',exact:true}).click();
  await expect(page).toHaveURL(/\/settings\/staffing\/positions(?:\?.*)?$/);
  await page.goBack();
  await expect(purpose).toHaveCount(0);
  await expect(date).toHaveValue(effectiveDate);
  await page.reload();
  await expect(purpose).toHaveCount(0);
  await expect(date).toHaveValue(effectiveDate);
});

async function draftParentChange(page: Page) {
  await open(page);
  const family = await seed308(page, 'jobfamily');
  const sample = await seedHierarchy(page, 'hierarchy', family.id);
  const original = (await request(page, 'GET', hierarchyApi)).body;
  await structureMenu(page);await queryStructure(page,effectiveDate);
  await structureDetail(page,sample.child.name);await adjust(page);await chooseParent(page,sample.second.name);
  const child=page.getByRole('combobox',{name:/子岗位/}),parent=page.getByRole('combobox',{name:/新上级/});
  return {sample,original,child,parent,publish:page.getByRole('button',{name:'保存',exact:true})};

}

test('312 层级发布恢复：未送达保留草稿，已发布自动查回且不重复写入', async ({ page }) => {
  const { sample, original, child, parent, publish } = await draftParentChange(page);
  const submissions: string[] = [];
  const lookups: string[] = [];
  await page.route('**' + hierarchyApi + '/publish', async route => {
    submissions.push(route.request().headers()['idempotency-key']);
    if (submissions.length === 2) expect((await route.fetch()).status()).toBe(200);
    await route.abort('failed');
  });
  await page.route(recoveryApi, async route => {
    lookups.push(new URL(route.request().url()).pathname.split('/').at(-2)!);
    await route.continue();
  });

  await publish.click();
  await expect(page.getByText('原提交未生效，请核对后重新保存。')).toBeVisible();
  await expect(child).toHaveCount(0);
  await expect(parent).toHaveValue(new RegExp('^' + sample.second.name));
  await expect(page.getByRole('textbox', { name: /生效日期/ })).toHaveValue(effectiveDate);
  expect((await request(page, 'GET', hierarchyApi)).body.sourceVersion).toBe(original.sourceVersion);
  await publish.click();
  await expect(page.getByText('已保存，2027-04-01 起生效。')).toBeVisible();
  await expect(page.getByRole('tab', {name:'岗位关系'})).toHaveAttribute('aria-selected','true');
  expect(submissions).toHaveLength(2);
  expect(new Set(submissions).size).toBe(2);
  expect(lookups).toEqual(submissions);
  const latest = (await request(page, 'GET', hierarchyApi + '?businessAt=' + effectiveDate)).body;
  expect(latest.versions).toHaveLength(original.versions.length + 1);
  expect(latest.nodes.find((node: {id: string}) => node.id === sample.child.id).parentId).toBe(sample.second.id);
});

test('312 层级发布恢复：自动查回仍未知，手动查回版本冲突后保留安排', async ({ page }) => {
  const { sample, original, child, parent, publish } = await draftParentChange(page);
  const concurrent = await request(page, 'POST', hierarchyApi + '/publish', {
    expectedVersion: original.sourceVersion, effectiveDate, childId: sample.child.id, parentId: null,
  }, randomUUID());
  expect(concurrent.status).toBe(200);
  const commands: {key: string; version: string}[] = [];
  const lookups: string[] = [];
  await page.route('**' + hierarchyApi + '/publish', async route => {
    commands.push({key: route.request().headers()['idempotency-key'], version: route.request().postDataJSON().expectedVersion});
    if (commands.length === 1) {
      expect((await route.fetch()).status()).toBe(409);
      await route.abort('failed');
    } else await route.continue();
  });
  await page.route(recoveryApi, async route => {
    const submissionId = new URL(route.request().url()).pathname.split('/').at(-2)!;
    lookups.push(submissionId);
    if (lookups.length === 1) {
      await route.fulfill({status: 202, json: {kind: 'positionhierarchy', submissionId, outcome: 'pending'}});
    } else await route.continue();
  });

  await publish.click();
  const recover = page.getByRole('button', { name: '查看保存结果' });
  await expect(recover).toBeEnabled();
  await expect(publish).toBeDisabled();
  expect(lookups).toEqual([commands[0].key]);
  await recover.click();
  await expect(page.getByText('岗位结构已发布新版本，请重新查询')).toBeVisible();
  await expect(child).toHaveCount(0);
  await expect(parent).toHaveValue(new RegExp('^' + sample.second.name));
  await expect(page.getByText('无上级',{exact:true})).toBeVisible();
  await expect(publish).toBeDisabled();
  await page.getByRole('button', {name: '使用当前版本'}).click();
  await publish.click();
  await expect(page.getByText('已保存，2027-04-01 起生效。')).toBeVisible();
  expect(commands.map(command => command.version)).toEqual([original.sourceVersion, concurrent.body.sourceVersion]);
  expect(commands[1].key).not.toBe(commands[0].key);
  expect(lookups).toEqual([commands[0].key, commands[0].key]);
});

test('139 从所属部门找到岗位并返回原行', async ({page})=>{
  await open(page);
  const family=await seed308(page,'jobfamily');
  const sample=await seedHierarchy(page,'hierarchy',family.id);
  await page.getByRole('button', { name: '职务配置', exact: true }).click();
  await expect(page.getByRole('menuitem', { name: '岗位层级', exact: true })).toHaveCount(0);
  await page.getByRole('menuitem', { name: '岗位', exact: true }).click();
  await page.getByRole('textbox',{name:'定位树中部门'}).fill(sample.child.department);
  const match=page.getByRole('listbox',{name:'部门定位结果'}).getByRole('option').filter({hasText:sample.child.department});
  await match.click();
  await expect(page).toHaveURL(/scopeDepartmentId=/);
  const include=page.getByRole('checkbox',{name:'含子部门'});
  await expect(include).toBeEnabled();
  await include.click();
  await expect(include).not.toBeChecked();
  await expect(page).toHaveURL(/includeDescendants=false/);
  const link=page.getByRole('link',{name:sample.child.name,exact:true});
  await link.click();
  await page.getByRole('tab',{name:'岗位关系'}).click();
  await queryStructure(page,effectiveDate);
  await expect(page.locator('#staffing-relations-panel')).toContainText(sample.first.name);
  await adjust(page);
  await page.getByRole('textbox',{name:/生效日期/}).fill('2027-04-');
  await page.getByRole('button',{name:'取消',exact:true}).click();
  await page.getByRole('button',{name:'继续编辑'}).click();
  await expect(page.getByRole('textbox',{name:/生效日期/})).toHaveValue('2027-04-');
  await page.getByRole('button',{name:'取消',exact:true}).click();
  await page.getByRole('button',{name:'放弃修改'}).click();
  await expect(page.getByRole('textbox',{name:/结构日期/})).toHaveValue(effectiveDate);
  await page.getByRole('link',{name:'返回列表'}).click();
  await expect(page).toHaveURL(/includeDescendants=false/);
  await expect(link).toBeFocused();
  await expect(page.getByRole('checkbox',{name:'含子部门'})).not.toBeChecked();
});

test('139 读结构末页岗位并在新标签保留原结构日期',async({page})=>{
  test.setTimeout(180000);
  await open(page);
  const family=await seed308(page,'jobfamily'),sample=await seedHierarchy(page,'hierarchy',family.id);
  const detail=(await request(page,'GET','/api/v1/positions/'+sample.child.id)).body.selectedRecord;
  const prefix=randomUUID().slice(0,6).toUpperCase();
  let target='',largestID='';
  for(let i=0;i<34;i++){
    const code='Z'+prefix+String(i).padStart(2,'0');
    const result=await request(page,'POST','/api/v1/positions',{code,name:'同名岗位'+prefix,effectiveDate:'2026-01-01',
      departmentId:detail.departmentId,locationId:detail.locationId,jobId:detail.jobId,operationType:'CREATE',operationReason:'NEW_POSITION'},randomUUID());
    expect(result.status).toBe(201);
    if(result.body.objectId>largestID){largestID=result.body.objectId;target=code}
  }
  let head = (await request(page,'GET',hierarchyApi)).body.sourceVersion;
  let parentId = largestID;
  const children:string[] = [];
  for (let level=1; level<=5; level++) {
    const code='D'+prefix+level;
    children.push(code);
    const result=await request(page,'POST','/api/v1/positions',{
      code, name:'区域销售与专业服务协作岗位'.repeat(3)+level, effectiveDate:'2026-01-01',
      departmentId:detail.departmentId, locationId:detail.locationId, jobId:detail.jobId,
      operationType:'CREATE', operationReason:'NEW_POSITION'
    },randomUUID());
    expect(result.status).toBe(201);
    const link=await request(page,'POST',hierarchyApi+'/publish',{
      childId:result.body.objectId, parentId, effectiveDate, expectedVersion:head
    },randomUUID());
    expect(link.status).toBe(200);
    parentId=result.body.objectId;
    head=link.body.sourceVersion;
  }
  const pages:{mode:string;page:number;rows:number;versions:number}[]=[];
  const raw:string[]=[];
  let listReads=0;
  page.on('request', request=>{
    if(request.method()==='GET' && new URL(request.url()).pathname==='/api/v1/positions') listReads++;
  });
  page.on('response',async response=>{
    const url=new URL(response.url());
    if(!url.pathname.includes('/position-hierarchies/')||response.request().method()!=='GET')return;
    if(!url.pathname.endsWith('/structure')){raw.push(url.pathname);return}
    if(!response.ok())return;
    const body=await response.json().catch(()=>undefined);
    if(body)pages.push({mode:url.searchParams.get('mode')!,page:Number(url.searchParams.get('page')??1),rows:body.nodes.length,versions:body.versions?.length??0});
  });
  await structureMenu(page);
  await queryStructure(page,effectiveDate);
  await page.getByRole('textbox',{name:'查找岗位'}).fill(target);
  await page.getByRole('button',{name:'查询',exact:true}).click();
  await page.getByRole('button',{name:'定位',exact:true}).click();
  const link=page.locator('[data-position-tree]').getByRole('link',{name:new RegExp(target)});
  await expect(link).toBeFocused();
  const beforeExpand=listReads;
  let row=link;
  for (const code of children) {
    await row.press('ArrowRight');
    const child=page.locator('[data-position-tree]').getByRole('link',{name:new RegExp(code)});
    await expect(child).toBeVisible();
    await row.press('ArrowDown');
    await expect(child).toBeFocused();
    row=child;
  }
  expect(listReads).toBe(beforeExpand);
  const popup=page.context().waitForEvent('page',{timeout:10000});
  await link.click({modifiers:['Control']});
  const tab=await popup;
  await expect(tab.getByRole('tab',{name:'岗位关系'})).toHaveAttribute('aria-selected','true');
  await expect(tab.getByRole('combobox',{name:/用途/})).toHaveCount(0);
  await expect(tab.getByRole('textbox',{name:/结构日期/})).toHaveValue(effectiveDate);
  await tab.close();
  expect(raw).toEqual([]);
  expect(pages.some(p=>p.mode==='path')).toBe(true);
  expect(pages.some(p=>p.mode==='nodes' && p.page>1)).toBe(true);
  expect(pages.every(p=>p.rows<=30 && p.versions===0)).toBe(true);
  const path = test.info().outputPath('structure-read.json');
  await writeFile(path, JSON.stringify({
    pages, expandedLevels:children.length, listReadsDuringExpansion:listReads-beforeExpand, domNodes: await page.locator('[data-position-tree] li').count(), viewport: page.viewportSize()
  }, null, 2));
  await test.info().attach('结构读取规模', { path, contentType: 'application/json' });
});

for (const [title, revokeRead] of [
  ['139 响应丢失后离开表单从岗位列表查回', false],
  ['139 跨入口查回前失去结构读权时保留岗位列表', true],
] as const) test(title,async({page})=>{
  test.setTimeout(120000);
  const {sample,original,publish}=await draftParentChange(page);
  let sent='',count=0,lookups=0;
  let submitted: {effectiveDate:string; childId:string; parentId:string} | undefined;
  await page.route('**'+hierarchyApi+'/publish',async route=>{
    count++;sent=route.request().headers()['idempotency-key'];submitted=route.request().postDataJSON();
    expect((await route.fetch()).ok()).toBe(true);await route.abort('failed');
  });
  await page.route(recoveryApi,async route=>{
    expect(new URL(route.request().url()).pathname.split('/').at(-2)).toBe(sent);
    if(++lookups===1) await route.fulfill({status:202,json:{kind:'positionhierarchy',submissionId:sent,outcome:'pending'}});
    else await route.continue();
  });
  await publish.click();
  await expect(page.getByRole('button',{name:'查看保存结果'})).toBeEnabled();
  await page.getByRole('button',{name:'职务配置',exact:true}).click();
  await page.getByRole('menuitem',{name:'岗位',exact:true}).click();
  await page.getByRole('button',{name:'离开此页'}).click();
  await expect(page).toHaveURL(/\/settings\/staffing\/positions$/);
  await page.reload();
  await page.getByRole('button',{name:'更多操作',exact:true}).click();
  await expect(page.getByRole('menuitem',{name:'查看岗位上下级'})).toBeVisible();
  await page.keyboard.press('Escape');
  await withPolicy(page, async (where, mutate) => {
    if (revokeRead) mutate(`DELETE FROM access.subject_grant WHERE ${where} AND action='positionhierarchy.read'`);
    const permission = revokeRead ? page.waitForResponse(response =>
      new URL(response.url()).pathname === '/api/v1/positions/context') : undefined;
    await page.getByRole('button',{name:'查看保存结果'}).click();
    await expect(page.getByText('原提交已保存。',{exact:true})).toBeVisible();
    if (permission) {
      expect((await (await permission).json()).capabilities.actions.positionRelations).toBeFalsy();
      await expect(page).toHaveURL(/\/settings\/staffing\/positions$/);
      await expect(page.getByRole('heading',{name:'岗位上下级'})).toHaveCount(0);
      await page.reload();
      await expect(page.getByRole('tree',{name:'部门树'})).toBeVisible();
      await page.getByRole('button',{name:'更多操作',exact:true}).click();
      await expect(page.getByRole('menuitem',{name:'列筛选…'})).toBeVisible();
      await expect(page.getByRole('menuitem',{name:'查看岗位上下级'})).toHaveCount(0);
    } else {
      await expect(page).toHaveURL(/\/settings\/staffing\/position-hierarchies$/);
      await expect(page.getByRole('heading',{name:'岗位上下级'})).toBeVisible();
    }
  });
  expect(count).toBe(1);expect(lookups).toBe(2);
  expect(submitted).toMatchObject({effectiveDate, childId:sample.child.id, parentId:sample.second.id});
  const latest=(await request(page,'GET',hierarchyApi+'?businessAt='+effectiveDate)).body;
  expect(latest.versions.length).toBe(original.versions.length+1);
  expect(latest.nodes.find((n:{id:string})=>n.id===sample.child.id).parentId).toBe(sample.second.id);
});

test('139 改日保留有效输入并要求重新选择已停用上级',async({page})=>{
  test.setTimeout(120000);
  await open(page);
  const family=await seed308(page,'jobfamily'),sample=await seedHierarchy(page,'hierarchy',family.id);
  const data=(await request(page,'GET','/api/v1/positions/'+sample.child.id)).body.selectedRecord;
  const code='TEMP'+randomUUID().slice(0,8),name='短期上级'+code;
  const created=await request(page,'POST','/api/v1/positions',{code,name,effectiveDate:'2026-01-01',operationType:'CREATE',operationReason:'NEW_POSITION',departmentId:data.departmentId,jobId:data.jobId,locationId:data.locationId},randomUUID());
  expect(created.status).toBe(201);
  const stopped=await request(page,'POST','/api/v1/positions/'+created.body.objectId+'/deactivate',{expectedSourceVersion:created.body.sourceVersion,anchorRecordId:created.body.selectedRecordId,effectiveDate:'2027-06-01',operationType:'DEACTIVATE',operationReason:'POSITION_CLOSURE'},randomUUID());
  expect(stopped.status).toBe(200);
  await structureMenu(page);await queryStructure(page,effectiveDate);await structureDetail(page,sample.child.name);await adjust(page);await chooseParent(page,name);
  let ready!:()=>void,release!:()=>void;
  const started=new Promise<void>(r=>{ready=r}),held=new Promise<void>(r=>{release=r});
  await page.route('**/reporting/structure?**',async route=>{
    const url=new URL(route.request().url());
    if(url.searchParams.get('businessAt')!=='2027-05-01' || url.searchParams.get('mode')!=='relation'){await route.continue();return}
    const response=await route.fetch();ready();await held;await route.fulfill({response}).catch(()=>{});
  });
  const date=page.getByRole('textbox',{name:/生效日期/}),save=page.getByRole('button',{name:'保存',exact:true});
  await date.fill('2027-05-01');await started;await date.fill('2027-04-');
  release();await expect(date).toHaveValue('2027-04-');await expect(save).toBeDisabled();
  await date.fill('2027-06-01');
  await expect(page.getByText(/请重新选择上级岗位/)).toBeVisible();await expect(save).toBeDisabled();
  await chooseParent(page,sample.first.name);await expect(save).toBeEnabled();
  await page.getByRole('button',{name:'取消',exact:true}).click();await page.getByRole('button',{name:'放弃修改'}).click();
  await expect(page.getByRole('textbox',{name:/结构日期/})).toHaveValue(effectiveDate);
});

test('139 跨页选择部门内岗位并导出固定名单',async({page})=>{
  test.setTimeout(180000);
  await open(page);
  const family=await seed308(page,'jobfamily'),sample=await seedHierarchy(page,'hierarchy',family.id);
  const data=(await request(page,'GET','/api/v1/positions/'+sample.child.id)).body.selectedRecord;
  const prefix='CSV'+randomUUID().slice(0,6).toUpperCase();
  const make=async(n:number,dep:string)=>{
    const code=prefix+String(n).padStart(2,'0');
    const response=await request(page,'POST','/api/v1/positions',{code,name:code,effectiveDate:'2026-01-01',operationType:'CREATE',operationReason:'NEW_POSITION',departmentId:dep,jobId:data.jobId,locationId:data.locationId},randomUUID());
    expect(response.status).toBe(201);return code;
  };
  const expected:string[]=[];
  for(let i=1;i<=21;i++)expected.push(await make(i,data.departmentId));
  await namedMenu308(page,'岗位','职务配置','/settings/staffing/positions');
  await page.getByRole('textbox',{name:'定位树中部门'}).fill(sample.child.department);
  await page.getByRole('listbox',{name:'部门定位结果'}).getByRole('option').filter({hasText:sample.child.department}).click();
  await expect(page).toHaveURL(/scopeDepartmentId=/);
  let observed!: () => void, release!: () => void;
  const started = new Promise<void>(done => { observed = done });
  const resume = new Promise<void>(done => { release = done });
  await page.route('**/api/v1/positions?**', async route => {
    const applied = new URL(route.request().url()).searchParams;
    if (applied.get('q') !== prefix) { await route.continue(); return }
    expect(applied.get('scopeDepartmentId')).toBe(data.departmentId);
    const result = await route.fetch();
    observed();
    await resume;
    await route.fulfill({response:result});
  });
  await page.getByRole('textbox',{name:'搜索编码或名称'}).fill(prefix);
  await page.getByRole('button',{name:'查询',exact:true}).click();
  const selectAll = page.getByRole('checkbox',{name:'全选查询结果',exact:true});
  await started;
  try { await expect(selectAll).toBeDisabled() } finally { release() }
  await selectAll.click();
  await page.getByRole('button',{name:'更多操作',exact:true}).click();
  await expect(page.getByRole('menuitem',{name:'导出所选（21）'})).toBeEnabled();
  await page.keyboard.press('Escape');
  await make(22,data.departmentId);
  await page.getByRole('button',{name:'下一页',exact:true}).click();
  const last=expected.pop()!;
  const lastLink = page.getByRole('link', { name: last, exact:true });
  await lastLink.click();
  await expect(page.locator('#staffing-basic-panel')).toContainText(last);
  await page.getByRole('link', { name:'返回列表' }).click();
  await expect(lastLink).toBeFocused();
  await page.screenshot({path:test.info().outputPath('department-position-return.png'),mask:[page.locator('.app-account-name')]});
  await expect(page).toHaveURL(/page=2/);
  await expect(page).toHaveURL(/scopeDepartmentId=/);
  const row=page.getByRole('row').filter({has:lastLink});
  await row.getByRole('checkbox',{name:'取消选择岗位'}).click();
  await page.getByRole('button',{name:'更多操作',exact:true}).click();
  const download=page.waitForEvent('download');
  await page.getByRole('menuitem',{name:'导出所选（20）'}).click();
  const stream=await(await download).createReadStream();let csv='';
  for await(const chunk of stream!)csv+=chunk;
  const codes=csv.trim().split(/\r?\n/).slice(1).map(row=>row.split(',')[0]);
  expect(codes).toEqual(expected);
});

test('139 查询者从历史节点打开结构日的岗位原件', async ({ page }) => {
  test.setTimeout(120000);
  await open(page);
  const family = await seed308(page, 'jobfamily');
  const { child, first, second } = await seedHierarchy(page, 'hierarchy', family.id);
  const base = '/api/v1/positions/' + child.id;
  const current = (await request(page, 'GET', base)).body;
  const oldName = '区域专员' + child.code;
  const revisedName = '区域客户专员' + child.code;
  const addition = await request(page, 'POST', base + '/records', {
    expectedSourceVersion: current.headSourceVersion, anchorRecordId: current.selectedRecord.recordId,
    effectiveDate, operationType: 'CHANGE', operationReason: 'NAME_ADJUSTMENT', changes: { name: oldName }
  }, randomUUID());
  expect(addition.status).toBe(200);
  const prior = (await request(page, 'GET', hierarchyApi)).body;
  const sealed = await request(page, 'POST', hierarchyApi+'/publish', {
    expectedVersion:prior.sourceVersion, effectiveDate, childId:first.id, parentId:second.id
  },randomUUID());
  expect(sealed.status).toBe(200);
  const version = sealed.body.sourceVersion;
  const number = (await request(page,'GET',hierarchyApi)).body.versions.find(
    (entry:{sourceVersion:string})=>entry.sourceVersion===version
  ).number;
  const selected = (await request(page, 'GET', base + '?selectedRecordId=' + addition.body.selectedRecordId)).body;
  const correction = await request(page, 'POST', base + '/records/' + addition.body.selectedRecordId + '/correct', {
    expectedSourceVersion: selected.headSourceVersion, expectedRecordRevision: selected.selectedRecord.revision,
    changes: { name: revisedName }
  }, randomUUID());
  expect(correction.status).toBe(200);

  await page.context().clearCookies();
  await open(page, 'reader');
  await structureMenu(page);
  await queryStructure(page, effectiveDate);
  await page.getByRole('button', { name: '发布历史' }).click();
  await page.getByRole('button', { name: new RegExp('^第 ' + number + ' 版') }).click();
  const purpose = page.getByRole('combobox', { name: /用途/ });
  await expect(page.getByText('正在查看只读历史资料')).toBeVisible();
  await queryStructure(page, '2027-03-31');
  expect(new URL(page.url()).searchParams.get('sourceVersion')).toBe(version);
  // Consecutive date and name submissions must keep both conditions, even
  // when the router has not yet rendered the first navigation.
  await page.getByRole('textbox',{name:/结构日期/}).fill(effectiveDate);
  await page.getByRole('textbox',{name:'查找岗位'}).fill(oldName);
  const searchButton = await page.getByRole('button',{name:'查询',exact:true}).elementHandle();
  await page.getByRole('button',{name:'查询结构'}).evaluate((button,search) => {
    (button as HTMLButtonElement).click();
    (search as HTMLButtonElement).click();
  },searchButton);
  await expect(page).toHaveURL(url =>
    url.searchParams.get('businessAt')===effectiveDate && url.searchParams.get('q')===oldName &&
    url.searchParams.get('sourceVersion')===version);
  await page.getByRole('link',{name:new RegExp('^'+oldName)}).click();
  await expect(page.getByText('正在查看只读历史资料')).toBeVisible();
  await expect(page.locator('#staffing-relations-panel')).toContainText(first.name);
  await page.getByRole('tab', { name: '基本资料' }).click();
  await expect(page.locator('#staffing-basic-panel')).toContainText(oldName);
  await expect(page.locator('#staffing-basic-panel')).not.toContainText(revisedName);
  await expect(page.getByRole('button', { name: '更多操作' })).toHaveCount(0);
  const sealedAddress = page.url();
  for (const grant of ['position.history.read', 'positionhierarchy.history.read']) {
    await withPolicy(page, async (where, mutate) => {
      mutate(`DELETE FROM access.subject_grant WHERE ${where} AND action='${grant}'`);
      await page.reload();
      await expect(page.getByRole('alert').filter({hasText: /权/}).first()).toBeVisible();
      await expect(page.getByText(oldName, { exact: true })).toHaveCount(0);
      await expect(page.getByText(revisedName, { exact: true })).toHaveCount(0);
      await page.goto('/settings/staffing/positions/' + child.id);
      await expect(page.locator('#staffing-basic-panel')).toContainText(child.name);
    });
    await page.goto(sealedAddress);
    await expect(page.locator('#staffing-basic-panel')).toContainText(oldName);
  }
  await page.getByRole('button', { name: '返回当前结构' }).click();
  await expect(page.getByRole('textbox', { name: /结构日期/ })).toHaveValue(effectiveDate);
  await expect(purpose).toHaveCount(0);
  expect(new URL(page.url()).searchParams.has('purpose')).toBe(false);
  expect(new URL(page.url()).searchParams.has('sourceVersion')).toBe(false);
  for (const address of [
    '/settings/staffing/position-hierarchies?purpose=functional&businessAt='+effectiveDate,
    '/settings/staffing/positions/'+child.id+'?purpose=functional&tab=relations&structureDate='+effectiveDate,
  ]) {
    await page.goto(address);
    await expect(page.getByRole('heading',{name:'页面不存在'})).toBeVisible();
    await expect(page.getByRole('button',{name:'发布历史'})).toHaveCount(0);
    await expect(page.getByRole('button',{name:'更多操作'})).toHaveCount(0);
  }
});

test('139 解除管理上级保留下一次安排', async ({ page }) => {
  test.setTimeout(120000);
  await open(page);
  const family = await seed308(page, 'jobfamily');
  const { child, first, second } = await seedHierarchy(page, 'hierarchy', family.id);
  await structureMenu(page);
  await queryStructure(page, effectiveDate);
  await structureDetail(page, child.name);
  const relations = page.locator('#staffing-relations-panel');
  await expect(relations).toContainText(first.name);
  await expect(relations).toContainText('2027-04-30');
  await adjust(page);
  await page.getByRole('combobox', { name: /新上级/ }).click();
  await page.getByRole('option', { name: '不设上级', exact: true }).click();
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(relations).toContainText('无上级');
  await expect(relations).toContainText('2027-04-30');
  const cleared = (await request(page, 'GET', hierarchyApi)).body.sourceVersion;
  await adjust(page);
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(relations).toContainText('无上级');
  expect((await request(page, 'GET', hierarchyApi)).body.sourceVersion).toBe(cleared);
  await queryStructure(page, '2027-03-31');
  await expect(relations).toContainText(first.name);
  await queryStructure(page, '2027-05-01');
  await expect(relations).toContainText(second.name);
});

test('139 查询者失去部门字段及层级权后仍可打开普通岗位', async ({ page }) => {
  test.setTimeout(120000);
  await open(page);
  const family = await seed308(page, 'jobfamily');
  const { child } = await seedHierarchy(page, 'hierarchy', family.id);
  const record = (await request(page, 'GET', '/api/v1/positions/' + child.id)).body.selectedRecord;
  const create = async (prefix: string, date: string) => {
    const name = prefix + child.code;
    const result = await request(page, 'POST', '/api/v1/positions', {
      departmentId: record.departmentId, locationId: record.locationId, jobId: record.jobId,
      name, code: prefix === '未来岗' ? 'F' + child.code : 'I' + child.code, effectiveDate: date,
      operationType: 'CREATE', operationReason: 'NEW_POSITION'
    }, randomUUID());
    expect(result.status).toBe(201);
    return { name, result: result.body };
  };
  const future = await create('未来岗', '2099-01-01');
  const inactive = await create('已停用岗', '2026-01-01');
  expect((await request(page, 'POST', '/api/v1/positions/' + inactive.result.objectId + '/deactivate', {
    expectedSourceVersion: inactive.result.sourceVersion, anchorRecordId: inactive.result.selectedRecordId,
    effectiveDate:'2026-01-02', operationType:'DEACTIVATE', operationReason:'POSITION_CLOSURE'
  }, randomUUID())).status).toBe(200);
  await page.context().clearCookies();
  await open(page, 'reader');
  await namedMenu308(page, '岗位', '职务配置', '/settings/staffing/positions');
  await page.getByRole('textbox', { name: '定位树中部门' }).fill(child.department);
  await page.getByRole('listbox', { name: '部门定位结果' }).getByRole('option').filter({hasText:child.department}).click();
  await expect(page.getByRole('link', { name: child.name, exact:true })).toBeVisible();
  for (const [name, status] of [[future.name,'未生效'],[inactive.name,'停用']]) {
    const row = page.getByRole('row').filter({has: page.getByRole('link',{name,exact:true})});
    await expect(row).toContainText(status);
  }
  await withPolicy(page, async (where, mutate) => {
    mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'departmentId') WHERE ${where} AND action='position.read';
      DELETE FROM access.subject_grant WHERE ${where} AND action='positionhierarchy.read'`);
    await page.reload();
    await expect(page.getByText('当前部门范围不可用')).toBeVisible();
    await expect(page.getByRole('link', { name: child.name, exact:true })).toHaveCount(0);
    await expect(page.getByRole('textbox', { name: '定位树中部门' })).toHaveCount(0);
    await page.getByRole('button', { name: '清除范围' }).click();
    await page.getByRole('textbox', {name:'搜索编码或名称'}).fill(child.code);
    await page.getByRole('button', {name:'查询', exact:true}).click();
    await page.getByRole('link', { name: child.name, exact:true }).click();
    await expect(page.locator('#staffing-basic-panel')).toContainText(child.name);
    await expect(page.getByRole('tab', {name:'岗位关系'})).toHaveCount(0);
    const publish = await request(page, 'POST', hierarchyApi + '/publish', {
      expectedVersion:'', effectiveDate, childId:child.id, parentId:null
    }, randomUUID());
    expect(publish.status).toBe(403);
  });
});

test('139 从部门范围内的岗位调整至另一个部门的上级', async ({ page }) => {
  test.setTimeout(120000);
  await open(page);
  const family = await seed308(page, 'jobfamily');
  const { child, first } = await seedHierarchy(page, 'hierarchy', family.id);
  const path = '/api/v1/positions/' + child.id;
  const originalDetail = (await request(page, 'GET', path)).body;
  const original = originalDetail.selectedRecord;
  const department = (await request(page, 'GET', '/api/v1/departments/' + original.departmentId)).body;
  const positions = [];
  for (const [label, parent] of [['子部门', original.departmentId], ['其他部门', undefined]] as const) {
    const suffix = randomUUID().slice(0, 8);
    const dep = await request(page, 'POST', '/api/v1/departments', {
      code: 'DP' + suffix, name: label + suffix, managementBuId: department.selectedRecord.managementBuId,
      parentDepartmentId: parent, effectiveDate:'2026-01-01', operationType:'CREATE', operationReason:'NEW_ORGANIZATION'
    }, randomUUID());
    expect(dep.status).toBe(201);
    const name = label + '岗位' + suffix;
    const result = await request(page, 'POST', '/api/v1/positions', {
      code: 'PS' + suffix, name, departmentId: dep.body.departmentId, jobId: original.jobId,
      locationId: original.locationId, effectiveDate:'2026-01-01', operationType:'CREATE', operationReason:'NEW_POSITION'
    }, randomUUID());
    expect(result.status).toBe(201);
    positions.push(name);
  }
  const [subordinateDepartment, outsideDepartment] = positions;
  for (const role of ['admin', 'reader']) {
    if (role === 'reader') { await page.context().clearCookies(); await open(page, role) }
    await namedMenu308(page, '岗位', '职务配置', '/settings/staffing/positions');
    await page.getByRole('textbox', {name:'定位树中部门'}).fill(child.department);
    await page.getByRole('listbox', {name:'部门定位结果'}).getByRole('option').filter({hasText:child.department}).click();
    const scope = page.getByRole('checkbox', {name:'含子部门'});
    const object = page.getByRole('link', {name:child.name, exact:true});
    const below = page.getByRole('link', {name:subordinateDepartment, exact:true});
    await expect(scope).toBeChecked();
    await expect(below).toBeVisible();
    await expect(page.getByRole('link',{name:outsideDepartment,exact:true})).toHaveCount(0);
    await scope.click();
    await expect(below).toHaveCount(0);
    await object.click();
    await page.getByRole('tab', {name:'岗位关系'}).click();
    await queryStructure(page, effectiveDate);
    const relations = page.locator('#staffing-relations-panel');
    if (role === 'admin') {
      await expect(relations).toContainText(first.name);
      await adjust(page);
      await chooseParent(page, outsideDepartment);
      await page.getByRole('button', {name:'保存',exact:true}).click();
    }
    await expect(relations).toContainText(outsideDepartment);
    await expect(page.getByRole('combobox',{name:/用途/})).toHaveCount(0);
    await queryStructure(page, '2027-03-31');
    await expect(relations).toContainText(first.name);
    await page.getByRole('link',{name:'返回列表'}).click();
    await expect(object).toBeFocused();
    await expect(scope).not.toBeChecked();
  }
  const after = (await request(page,'GET',path)).body;
  expect(after.selectedRecord.departmentId).toBe(original.departmentId);
  expect(after.headSourceVersion).toBe(originalDetail.headSourceVersion);
});

test('139 部门内按编码排序后恢复第二页的滚动位置和岗位焦点', async ({page}, info) => {
  test.setTimeout(140000);
  await open(page);
  const family=await seed308(page,'jobfamily'), sample=await seedHierarchy(page,'hierarchy',family.id);
  const position=(await request(page,'GET','/api/v1/positions/'+sample.child.id)).body.selectedRecord;
  const prefix='NAV'+randomUUID().slice(0,6).toUpperCase();
  const names=Array.from({length:45},(_,i)=>prefix+String(i+1).padStart(2,'0'));
  for(const code of names){
    const res=await request(page,'POST','/api/v1/positions',{code,name:code,effectiveDate:'2026-01-01',
      departmentId:position.departmentId,jobId:position.jobId,locationId:position.locationId,
      operationType:'CREATE',operationReason:'NEW_POSITION'},randomUUID());
    expect(res.status).toBe(201);
  }
  await page.setViewportSize({width:1152,height:680});
  await namedMenu308(page,'岗位','职务配置','/settings/staffing/positions');
  await page.getByRole('textbox',{name:'定位树中部门'}).fill(sample.child.department);
  await page.getByRole('listbox',{name:'部门定位结果'}).getByRole('option').filter({hasText:sample.child.department}).click();
  await page.getByRole('checkbox',{name:'含子部门'}).click();
  await expect(page).toHaveURL(url=>url.searchParams.get('includeDescendants')==='false');
  await page.getByRole('textbox',{name:'搜索编码或名称'}).fill(prefix);
  const grid=page.getByRole('grid',{name:'岗位列表'});
  const listReply=(sort:string)=>page.waitForResponse(response=>{
    const u=new URL(response.url());
    return u.pathname==='/api/v1/positions' && u.searchParams.get('q')===prefix &&
      (u.searchParams.get('sort')??'')===sort && response.ok();
  });
  const searched=listReply('');
  await page.getByRole('button',{name:'查询',exact:true}).click();
  await searched;
  const header=page.getByRole('columnheader',{name:/编码/});
  for(const sort of ['code','-code']){
    await expect(grid.getByRole('link').first()).toBeVisible();
    await expect(page.locator('.standard-page')).toHaveAttribute('aria-busy','false');
    const sorted=listReply(sort);
    await header.click();
    await sorted;
    await expect(page).toHaveURL(url=>url.searchParams.get('sort')===sort);
  }
  await expect(grid.getByRole('link',{name:names[44],exact:true})).toBeVisible();
  await page.getByRole('button',{name:'下一页',exact:true}).click();
  await expect(page).toHaveURL(url=>url.searchParams.get('page')==='2');
  const scroller=grid.locator('.MuiDataGrid-virtualScroller');
  const target=grid.getByRole('link',{name:names[11],exact:true});
  const scrollTop=()=>scroller.evaluate(el=>el.scrollTop);
  const rowY=()=>target.evaluate(el=>el.getBoundingClientRect().y);
  const frame=()=>page.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));
  const inView=async()=> (await target.count())>0 && await target.evaluate(el=>{
    const r=el.getBoundingClientRect(),s=el.closest('.MuiDataGrid-virtualScroller')!.getBoundingClientRect();
    return r.top>=s.top+30 && r.bottom<=s.bottom-8 && el.contains(document.elementFromPoint((r.left+r.right)/2,(r.top+r.bottom)/2));
  });
  const bounds=await scroller.boundingBox();expect(bounds).not.toBeNull();
  await page.mouse.move(bounds!.x+bounds!.width/2,bounds!.y+bounds!.height/2);
  for(let i=0;i<20 && !(await inView());i++){await page.mouse.wheel(0,84);await frame()}
  expect(await inView()).toBe(true);
  const before={top:await scrollTop(),linkY:await rowY()};
  expect(before.top).toBeGreaterThan(0);
  const applied = new URL(page.url()).searchParams;
  await target.click();await expect(page.locator('#staffing-basic-panel')).toContainText(names[11]);
  await page.getByRole('link',{name:'返回列表'}).click();
  await expect(target).toBeFocused();await frame();
  await expect.poll(async () => Math.abs(await scrollTop()-before.top)).toBeLessThanOrEqual(1);
  await expect.poll(async () => Math.abs(await rowY()-before.linkY)).toBeLessThanOrEqual(1);
  const after={top:await scrollTop(),linkY:await rowY()};
  await expect(page.getByRole('navigation',{name:'岗位视图'})).toHaveCount(0);
  await page.getByRole('button',{name:'更多操作',exact:true}).click();
  await page.getByRole('menuitem',{name:'查看岗位上下级'}).click();
  await expect(page.getByRole('heading',{name:'岗位上下级'})).toBeVisible();
  await expect(page.getByRole('tree',{name:'部门树'})).toHaveCount(0);
  await page.getByRole('link',{name:'返回列表'}).click();
  await expect(target).toBeFocused();
  await expect.poll(async () => Math.abs(await scrollTop()-after.top)).toBeLessThanOrEqual(1);
  await expect.poll(async () => Math.abs(await rowY()-after.linkY)).toBeLessThanOrEqual(1);
  for(const key of ['scopeDepartmentId','includeDescendants','q','sort','page']){
    expect(new URL(page.url()).searchParams.get(key)).toBe(applied.get(key));
  }
  await expect.poll(inView).toBe(true);
  await writeFile(info.outputPath('scroll.json'),JSON.stringify({viewport:page.viewportSize(),
    positions:45,page:2,sort:'-code',includeDescendants:false,selectedCodeNumber:12,before,after,
    structureReturn:{top:await scrollTop(),linkY:await rowY()},linkFocused:true},null,2));
});
