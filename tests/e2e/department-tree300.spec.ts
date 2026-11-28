import { nativeZoomCases } from './desktop-viewports.mjs';
import {test, expect, type Page} from './fixtures';
import {open} from './bu-support';
import {apiBU, apiDepartment, menu, departmentKind} from './dated-object-support';

test('300部门树正常菜单、独立展开定位、范围交集、固定选择及返回', async({page}) => {
  test.setTimeout(120000);
  await open(page);
  const bu = await apiBU(page), prefix = 'TREE'+Date.now();
  const root = await apiDepartment(page,bu.id,{code:prefix,name:'树根'+prefix});
  const child = await apiDepartment(page,bu.id,{code:prefix+'A',name:'子部门'+prefix,parent:root.id});
  await apiDepartment(page,bu.id,{code:prefix+'B',name:'末端'+prefix,parent:child.id});
  await apiDepartment(page,bu.id,{code:prefix+'F',name:'未来'+prefix,parent:root.id,date:'2099-01-01'});
  await menu(page,departmentKind);
  const grid=page.getByRole('grid',{name:'部门列表',exact:true});
  await expect(grid).toBeVisible();
  const search=page.getByRole('textbox',{name:'定位树中部门',exact:true});
  await expect(search).toBeVisible();
  const requests:string[]=[];
  page.on('request',r=>{const u=new URL(r.url());if(u.pathname==='/api/v1/departments'&&r.method()==='GET'||u.pathname.includes('/departments/selection/'))requests.push(u.pathname)});
  const query=page.getByRole('textbox',{name:'搜索编码或名称',exact:true});
  await query.fill('尚未应用');
  const before=requests.length;
  await search.fill(prefix);
  const results=page.getByRole('listbox',{name:'部门定位结果'});
  await expect(results.getByRole('option')).toHaveCount(3);
  expect(requests.length).toBe(before);
  await results.getByRole('option').filter({hasText:'树根'+prefix}).click();
  const tree=page.getByRole('tree',{name:'部门树',exact:true});
  await expect(tree).toBeVisible();
  await expect(grid.locator('.MuiDataGrid-row')).toHaveCount(3);
  await expect(query).toHaveValue('尚未应用');
  expect(new URL(page.url()).searchParams.get('q')).toBeNull();
  expect(new URL(page.url()).searchParams.get('scopeDepartmentId')).toBe(root.id);
  const row=tree.locator(`[data-tree-id="${root.id}"]`);
  expect(Math.round((await row.boundingBox())!.height)).toBe(28);
  expect(await row.locator('span').last().evaluate(el=>getComputedStyle(el).fontSize)).toBe('14px');
  const beforeExpand=requests.length;
  await row.getByRole('button',{name:'展开树根'+prefix,exact:true}).click();
  await expect(tree.locator(`[data-tree-id="${child.id}"]`)).toBeVisible();
  expect(requests.length).toBe(beforeExpand);
  await page.getByRole('checkbox',{name:'全选查询结果',exact:true}).check();
  await expect(page.getByText('已选择3项',{exact:true})).toBeVisible();
  const beforeRepeat=requests.length;
  await row.locator('span').last().click();
  await page.getByRole('button',{name:'折叠部门树',exact:true}).click();
  await expect(page.getByText('已选择3项',{exact:true})).toBeVisible();
  expect(requests.length).toBe(beforeRepeat);
  await page.getByRole('button',{name:'展开部门树',exact:true}).click();
  await page.getByRole('checkbox',{name:'含子部门',exact:true}).uncheck();
  await expect(grid.locator('.MuiDataGrid-row')).toHaveCount(1);
  await expect(page.getByRole('toolbar',{name:'所选部门',exact:true})).toHaveCount(0);
  await page.getByRole('button',{name:'全部',exact:true}).click();
  await expect(page.getByRole('checkbox',{name:'含子部门',exact:true})).not.toBeChecked();
  await expect(page.getByRole('checkbox',{name:'含子部门',exact:true})).toBeDisabled();
  await row.locator('span').last().click();
  await expect(grid.locator('.MuiDataGrid-row')).toHaveCount(1);
  await grid.getByRole('link',{name:'树根'+prefix,exact:true}).click();
  await page.getByRole('link',{name:'返回列表',exact:true}).click();
  await expect(grid.locator('.MuiDataGrid-row')).toHaveCount(1);
  await expect(page.getByRole('tree').locator(`[data-tree-id="${root.id}"]`)).toHaveAttribute('aria-selected','true');
  for(const width of [1000,900,899]) {
    await page.setViewportSize({width,height:844});
    if(width>=900) await expect(page.getByRole('tree')).toBeVisible();
    else {
      const trigger=page.getByRole('button',{name:'打开部门树',exact:true});await trigger.click();
      await expect(page.getByRole('tree')).toBeVisible();
      await page.keyboard.press('Escape');await expect(trigger).toBeFocused();
    }
  }
});

test('300末级保留筛选交集，清空关键词立即生效且不残留旧定位提示', async({page}) => {
  test.setTimeout(120000);
  await open(page);
  const bu=await apiBU(page),prefix='LEAF'+Date.now();
  const root=await apiDepartment(page,bu.id,{code:prefix,name:'旧根'+prefix});
  const leaf=await apiDepartment(page,bu.id,{code:prefix+'L',name:'末级'+prefix,parent:root.id});
  const base='/settings/organization/departments';
  const filters={status:'active',filterField:'code',filterOperator:'contains',filterValue:prefix,sort:'name_desc',pageSize:'50'};
  await menu(page,departmentKind);
  await page.goto(base+'?'+new URLSearchParams({...filters,q:'旧'}));
  const grid=page.getByRole('grid',{name:'部门列表',exact:true});
  const query=page.getByRole('textbox',{name:'搜索编码或名称',exact:true});
  await grid.getByRole('link',{name:'旧根'+prefix,exact:true}).click();
  await page.getByRole('link',{name:'返回列表',exact:true}).click();
  await expect(grid.getByRole('link',{name:'旧根'+prefix,exact:true})).toBeVisible();
  expect(new URL(page.url()).searchParams.get('selected')).toBe(root.id);
  await page.getByRole('textbox',{name:'定位树中部门',exact:true}).fill(prefix+'L');
  await page.getByRole('listbox',{name:'部门定位结果'}).getByRole('option').click();
  await expect(page.getByText('没有符合条件的部门',{exact:true})).toBeVisible();
  await expect(query).toHaveValue('旧');
  await expect(page.getByText(/原选中对象不在当前页/)).toHaveCount(0);
  expect(new URL(page.url()).searchParams.get('q')).toBe('旧');
  expect(new URL(page.url()).searchParams.get('selected')).toBeNull();
  await query.fill(''); // No query button: removing the keyword is itself applied.
  await expect(grid.getByRole('link',{name:'末级'+prefix,exact:true})).toBeVisible();
  await expect(grid.locator('.MuiDataGrid-row')).toHaveCount(1);
  await expect(page.getByText(/搜索词尚未应用|原选中对象不在当前页/)).toHaveCount(0);
  const params=new URL(page.url()).searchParams;
  expect(params.get('q')).toBeNull();expect(params.get('scopeDepartmentId')).toBe(leaf.id);
  expect(params.get('page')).toBe('1');expect(params.get('cursor')).toBeNull();
  for(const [key,value] of Object.entries(filters))expect(params.get(key),key).toBe(value);
  await page.goBack();
  await expect(query).toHaveValue('旧');
  await expect(page.getByText('没有符合条件的部门',{exact:true})).toBeVisible();
  await page.goForward();
  await expect(query).toHaveValue('');
  await expect(grid.getByRole('link',{name:'末级'+prefix,exact:true})).toBeVisible();
  await grid.getByRole('link',{name:'末级'+prefix,exact:true}).click();
  await page.getByRole('link',{name:'返回列表',exact:true}).click();
  await expect(grid.getByRole('link',{name:'末级'+prefix,exact:true})).toBeVisible();
  expect(new URL(page.url()).searchParams.get('selected')).toBe(leaf.id);
  await page.getByRole('checkbox',{name:'含子部门',exact:true}).uncheck();
  await expect(grid.getByRole('link',{name:'末级'+prefix,exact:true})).toBeVisible();
  expect(new URL(page.url()).searchParams.get('selected')).toBeNull();
  await query.fill('尚未应用');
  await page.getByRole('button',{name:'清除范围',exact:true}).click();
  await expect(grid.locator('.MuiDataGrid-row')).toHaveCount(2);
  await expect(query).toHaveValue('尚未应用');
  for(const [key,value] of Object.entries(filters))expect(new URL(page.url()).searchParams.get(key),key).toBe(value);
});

test('300清空关键词的慢读取和失败重试保留新输入与其他筛选',async({page})=>{
  test.setTimeout(120000);
  await open(page);
  const bu=await apiBU(page),prefix='CLEAR'+Date.now();
  await apiDepartment(page,bu.id,{code:prefix,name:prefix});
  const base='/settings/organization/departments';
  const params={q:'旧',filterField:'code',filterOperator:'equals',filterValue:prefix};
  await page.goto(base+'?'+new URLSearchParams(params));
  await expect(page.getByText('没有符合条件的部门',{exact:true})).toBeVisible();
  const query=page.getByRole('textbox',{name:'搜索编码或名称',exact:true});
  let release!:()=>void,arrive!:()=>void;
  const held=new Promise<void>(resolve=>release=resolve),arrived=new Promise<void>(resolve=>arrive=resolve);
  await page.route('**/api/v1/departments?*',async route=>{const response=await route.fetch();arrive();await held;await route.fulfill({response});});
  try{
    await query.fill('');await arrived;
    await query.fill('较新的草稿');release();
    await expect(page.getByRole('grid',{name:'部门列表',exact:true}).getByRole('link',{name:prefix,exact:true})).toBeVisible();
    await expect(query).toHaveValue('较新的草稿');
    expect(new URL(page.url()).searchParams.get('q')).toBeNull();
  }finally{release();await page.unrouteAll({behavior:'wait'});}
  await page.goto(base+'?'+new URLSearchParams(params));
  await expect(page.getByText('没有符合条件的部门',{exact:true})).toBeVisible();
  await page.route('**/api/v1/departments?*',route=>route.fulfill({status:503,json:{code:'M02_DEPENDENCY_UNAVAILABLE',message:'清除关键词读取失败'}}));
  await query.fill('');
  await expect(page.getByText('清除关键词读取失败',{exact:true})).toBeVisible();
  expect(new URL(page.url()).searchParams.get('q')).toBeNull();
  await query.fill('失败后的新草稿');await page.unrouteAll({behavior:'wait'});
  await page.getByRole('button',{name:'重试',exact:true}).click();
  await expect(page.getByRole('grid',{name:'部门列表',exact:true}).getByRole('link',{name:prefix,exact:true})).toBeVisible();
  await expect(query).toHaveValue('失败后的新草稿');
  expect(new URL(page.url()).searchParams.get('filterValue')).toBe(prefix);
});

test('300范围页失权后固定成员汇总和CSV仍可用，读取503可重新核验',async({page})=>{
  test.setTimeout(120000);
  const {withPolicy}=await import('./grants');
  const {readFileSync}=await import('node:fs');
  await open(page);const bu=await apiBU(page),prefix='UNAV'+Date.now();
  const root=await apiDepartment(page,bu.id,{code:prefix}),leaf=await apiDepartment(page,bu.id,{code:prefix+'L',parent:root.id});
  await menu(page,departmentKind);await page.getByRole('textbox',{name:'定位树中部门',exact:true}).fill(prefix);
  await page.getByRole('listbox').getByRole('option').filter({hasText:prefix}).first().click();
  const grid=page.getByRole('grid',{name:'部门列表',exact:true});await expect(grid.locator('.MuiDataGrid-row')).toHaveCount(2);
  await page.getByRole('checkbox',{name:'全选查询结果',exact:true}).check();await expect(page.getByText('已选择2项',{exact:true})).toBeVisible();
  await grid.locator('.MuiDataGrid-row').filter({has:page.getByRole('link',{name:prefix,exact:true})}).getByRole('checkbox').uncheck();
  await expect(page.getByText('已选择1项',{exact:true})).toBeVisible();
  await withPolicy(page,async(where,mutate)=>{
    mutate(`INSERT INTO access.subject_grant(instance_id,grant_id,principal_id,scope_kind,scope_id,action,read_fields,write_fields) SELECT instance_id,gen_random_uuid(),principal_id,'department','${leaf.id}',action,read_fields,write_fields FROM access.subject_grant WHERE ${where} AND action='department.read'; DELETE FROM access.subject_grant WHERE ${where} AND action='department.read' AND scope_kind<>'department'`);
    try {
      await page.getByRole('button',{name:'刷新',exact:true}).click();
      await expect(page.getByText('当前范围不可用',{exact:true})).toBeVisible();
      await expect(page.getByText('已选择1项',{exact:true})).toBeVisible();
      await expect(grid.locator('.MuiDataGrid-row')).toHaveCount(0);
      const file=page.waitForEvent('download');await page.getByRole('button',{name:'导出所选',exact:true}).click();
      expect(readFileSync((await(await file).path())!,'utf8')).toContain(prefix+'L');
      await page.route('**/api/v1/departments/selection/resolve',r=>r.fulfill({status:503,json:{code:'M02_DEPENDENCY_UNAVAILABLE',message:'受控读取失败'}}));
      await page.getByRole('button',{name:'刷新',exact:true}).click();
      await expect(page.getByRole('button',{name:'导出所选',exact:true})).toBeDisabled();
      await page.unroute('**/api/v1/departments/selection/resolve');
      await page.getByRole('button',{name:'刷新',exact:true}).click();
      await expect(page.getByText('已选择1项',{exact:true})).toBeVisible();
      await expect(page.getByRole('button',{name:'导出所选',exact:true})).toBeEnabled();
    } finally {mutate(`DELETE FROM access.subject_grant WHERE ${where} AND action='department.read' AND scope_kind='department'`)}
  });
});

test('300查询者正常菜单、独立树失败不阻断无范围列表',async({page})=>{
  await open(page,'reader');
  await page.route('**/api/v1/departments/tree/nodes*',r=>r.fulfill({status:503,json:{code:'M02_DEPENDENCY_UNAVAILABLE',message:'受控树读取失败'}}));
  await menu(page,departmentKind);
  await expect(page.getByText('受控树读取失败',{exact:true})).toBeVisible();
  await expect(page.getByRole('grid',{name:'部门列表',exact:true})).toBeVisible();
  await expect(page.getByRole('link',{name:'新建',exact:true})).toHaveAttribute('aria-disabled','true');
  await page.unroute('**/api/v1/departments/tree/nodes*');
  await page.getByRole('button',{name:'重试',exact:true}).click();
  await expect(page.getByText('受控树读取失败',{exact:true})).toHaveCount(0);
});

test('300迟到定位不能覆盖新选择，末页定位支持前后同级且DOM有界',async({page})=>{
  test.setTimeout(180000);
  await page.addInitScript(()=>{const original=window.fetch;window.fetch=async(input,init)=>{const r=new Request(input,init);return original(r.url.includes('/tree-location')?new Request(r,{signal:new AbortController().signal}):r)}});
  await open(page);const bu=await apiBU(page),prefix='LATE'+Date.now();const root=await apiDepartment(page,bu.id,{code:prefix});
  const children=[];for(let i=0;i<51;i++)children.push(await apiDepartment(page,bu.id,{code:prefix+String(i).padStart(3,'0'),parent:root.id}));
  await menu(page,departmentKind);
  let release!:()=>void,arrived!:()=>void;const wait=new Promise<void>(r=>{release=r}),captured=new Promise<void>(r=>{arrived=r});
  await page.route('**/api/v1/departments/'+children[0].id+'/tree-location',async route=>{const response=await route.fetch();expect(response.status()).toBe(200);arrived();await wait;await route.fulfill({response}).catch(()=>{})});
  try {
    const search=page.getByRole('textbox',{name:'定位树中部门',exact:true});
    await search.fill(prefix+'000');await page.getByRole('listbox').getByRole('option').click();await captured;
    await search.fill(prefix+'050');await page.getByRole('listbox').getByRole('option').click();
    await expect(page.getByRole('tree').locator(`[data-tree-id="${children[50].id}"]`)).toBeVisible();
    await expect(page).toHaveURL(new RegExp('scopeDepartmentId='+children[50].id));
    release();await page.unrouteAll({behavior:'wait'});
    await expect(page).toHaveURL(new RegExp('scopeDepartmentId='+children[50].id));
    const tree=page.getByRole('tree');await expect(tree.getByRole('treeitem',{name:'加载较早部门',exact:true})).toBeVisible();
    await tree.getByRole('treeitem',{name:'加载较早部门',exact:true}).press('Enter');
    await expect(tree.locator(`[data-tree-id="${children[0].id}"]`)).toBeVisible();
    expect(await tree.getByRole('treeitem').count()).toBeLessThanOrEqual(200);
    await tree.locator(`[data-tree-id="${children[0].id}"]`).focus();await page.keyboard.press('ArrowDown');
    await expect(tree.locator(`[data-tree-id="${children[1].id}"]`)).toBeFocused();
  } finally {release();await page.unrouteAll({behavior:'wait'})}
});

test('300Chromium 原生缩放下抽屉与范围操作',async()=>{
  test.setTimeout(120000);
  const {nativeZoomBrowser}=await import('./native-zoom-support');
  const native=await nativeZoomBrowser(),page=native.page;
  try {
    await open(page);const bu=await apiBU(page),root=await apiDepartment(page,bu.id,{name:'缩放部门树'+Date.now()});await menu(page,departmentKind);
    for(const [, factor] of nativeZoomCases) {
      await native.setZoom(factor);
      const trigger=page.getByRole('button',{name:'打开部门树',exact:true});await expect(trigger).toBeInViewport();await trigger.click();
      await page.getByRole('textbox',{name:'定位树中部门',exact:true}).fill(root.code);
      await page.getByRole('listbox').getByRole('option').click();
      await expect(trigger).toBeFocused();await expect(page.getByRole('button',{name:'清除范围',exact:true})).toBeInViewport();
      await page.getByRole('button',{name:'清除范围',exact:true}).click();
      const dimensions=await page.evaluate(()=>({width:innerWidth,document:document.documentElement.scrollWidth}));
      if(dimensions.document>dimensions.width+1)await page.screenshot({path:test.info().outputPath('native-overflow.png'),mask:[page.locator('.app-account-name')]});
      expect(dimensions.document,JSON.stringify({factor,...dimensions})).toBeLessThanOrEqual(dimensions.width+1);
    }
  } finally {await native.close()}
});

test('300展示预算反例：2501层定位骨架完整可见且不预读全部层',async({page})=>{
  test.setTimeout(120000);await open(page);const bu=await apiBU(page),target=await apiDepartment(page,bu.id,{code:'DEPTH'+Date.now()});
  const day=await page.evaluate(async()=>(await(await fetch('/api/v1/organization/context')).json()).currentDate);
  const {randomUUID}=await import('node:crypto');
  const ids=Array.from({length:2500},()=>randomUUID());ids.push(target.id);
  const chain=ids.map((id,i)=>({id,code:'DEPTH_'+String(i).padStart(4,'0'),name:'深链'+i,parentId:i?ids[i-1]:null,kind:'node',hasChildren:i<2500}));
  let reads=0;
  // This is a display-budget injection only. The real deep Go projection and
  // database hierarchy correctness are validated separately.
  await page.route('**/api/v1/departments/*/tree-location',r=>{reads++;return r.fulfill({json:{items:chain,businessAt:day}})});
  await page.route('**/api/v1/departments/tree/nodes*',r=>{reads++;const parent=new URL(r.request().url()).searchParams.get('parentId');const index=parent?ids.indexOf(parent)+1:0;return r.fulfill({json:{items:index>=0&&index<chain.length?[chain[index]]:[],businessAt:day,hasMore:false,hasPrevious:false}})});
  await menu(page,departmentKind);await page.getByRole('textbox',{name:'定位树中部门',exact:true}).fill(target.code);await page.getByRole('listbox').getByRole('option').click();
  const tree=page.getByRole('tree');await expect(tree.locator(`[data-tree-id="${target.id}"]`)).toBeInViewport();
  expect(await tree.getByRole('treeitem').count()).toBeLessThanOrEqual(200);expect(reads).toBeLessThanOrEqual(6);
});

test('300观察新父后，尚未缓存的旧父迟到窗口不能重新挂载部门',async({page})=>{
  test.setTimeout(120000);
  const {request}=await import('./dated-object-support');const {randomUUID}=await import('node:crypto');
  await page.addInitScript(()=>{const original=window.fetch;window.fetch=async(input,init)=>{const r=new Request(input,init);return original(r.url.includes('/tree/nodes')?new Request(r,{signal:new AbortController().signal}):r)}});
  await open(page);const bu=await apiBU(page),prefix='A300MOVE'+Date.now();
  const a=await apiDepartment(page,bu.id,{code:prefix+'A'}),b=await apiDepartment(page,bu.id,{code:prefix+'B'});
  const leaf=await apiDepartment(page,bu.id,{code:prefix+'L',parent:a.id});await apiDepartment(page,bu.id,{code:prefix+'Z',parent:b.id});
  let release!:()=>void,arrived!:()=>void,done!:()=>void;
  const held=new Promise<void>(r=>release=r),captured=new Promise<void>(r=>arrived=r),completed=new Promise<void>(r=>done=r);
  await page.route('**/api/v1/departments/tree/nodes*',async route=>{
    if(new URL(route.request().url()).searchParams.get('parentId')!==a.id){await route.continue();return;}
    const response=await route.fetch();expect(response.status()).toBe(200);arrived();await held;
    await route.fulfill({response}).catch(()=>{});done();
  });
  try {
    await menu(page,departmentKind);const tree=page.getByRole('tree',{name:'部门树',exact:true});
    await tree.getByRole('button',{name:'展开'+a.code,exact:true}).click();await captured;
    const changed=await request(page,'POST',`/api/v1/departments/${leaf.id}/records/${leaf.targetRecordId}/correct`,{expectedSourceVersion:leaf.sourceVersion,expectedRecordRevision:leaf.targetRevision,changes:{parentDepartmentId:b.id}},randomUUID());expect(changed.status).toBe(200);
    await tree.getByRole('button',{name:'展开'+b.code,exact:true}).click();
    const node=tree.locator(`[data-tree-id="${leaf.id}"]`),parent=tree.locator(`[data-tree-id="${b.id}"]`);
    await expect(node).toBeVisible();expect((await node.boundingBox())!.y).toBeGreaterThan((await parent.boundingBox())!.y);
    release();await completed;await page.evaluate(()=>new Promise<void>(r=>requestAnimationFrame(()=>requestAnimationFrame(()=>r()))));
    await expect(node).toHaveCount(1);expect((await node.boundingBox())!.y).toBeGreaterThan((await parent.boundingBox())!.y);
    await expect(tree.getByRole('treeitem',{name:'读取失败，重试',exact:true})).toBeVisible();
  } finally {release();await page.unrouteAll({behavior:'ignoreErrors'});}
});

test('300定位结果双向分页保留早期结果可达',async({page})=>{
  test.setTimeout(120000);await open(page);const bu=await apiBU(page),prefix='A300SEARCH'+Date.now();
  for(let i=0;i<21;i++)await apiDepartment(page,bu.id,{code:prefix+String(i).padStart(2,'0'),name:'同名定位结果'});
  await menu(page,departmentKind);await page.getByRole('textbox',{name:'定位树中部门',exact:true}).fill(prefix);
  const results=page.getByRole('listbox',{name:'部门定位结果'});
  const first=()=>results.getByRole('option').filter({hasText:prefix+'00'});
  await expect(first()).toBeVisible();await results.getByRole('option',{name:'加载更多定位结果',exact:true}).click();
  await expect(results.getByRole('option').filter({hasText:prefix+'20'})).toBeVisible();
  await results.getByRole('option',{name:'加载较早定位结果',exact:true}).click();await expect(first()).toBeVisible();
});

async function displayBudget(page:Page) {
  await open(page);
  const day=await page.evaluate(async()=>(await(await fetch('/api/v1/organization/context')).json()).currentDate);
  const {randomUUID}=await import('node:crypto');
  const roots=Array.from({length:2100},(_,i)=>({id:randomUUID(),code:'BUDGET'+String(i).padStart(4,'0'),name:'预算根'+i,parentId:null,kind:'node',hasChildren:true}));
  const children=roots.map((p,i)=>({id:randomUUID(),code:'CHILD'+i,name:'预算子'+i,parentId:p.id,kind:'node',hasChildren:false}));
  const reads=new Map<string,number>();
  // Display cache only: controlled pages do not count as DB or capacity proof.
  await page.route('**/api/v1/departments/tree/nodes*',route=>{
    const q=new URL(route.request().url()).searchParams,parent=q.get('parentId')??'',offset=Number(q.get('cursor')??0);
    reads.set(parent,(reads.get(parent)??0)+1);
    const items=parent?[children[roots.findIndex(p=>p.id===parent)]]:roots.slice(offset,offset+50);
    return route.fulfill({json:{items,businessAt:day,hasPrevious:!parent&&offset>0,previousCursor:!parent&&offset>0?String(offset-50):undefined,hasMore:!parent&&offset+50<roots.length,nextCursor:!parent&&offset+50<roots.length?String(offset+50):undefined}});
  });
  await menu(page,departmentKind);const tree=page.getByRole('tree',{name:'部门树',exact:true});
  return {roots,reads,tree};
}

test('300展示缓存超过40分支后根窗口可淘汰并重读',async({page})=>{
  test.setTimeout(120000);const {roots,reads,tree}=await displayBudget(page);
  for(let i=0;i<40;i++){
    await tree.getByRole('button',{name:'展开预算根'+i,exact:true}).click();
    await expect.poll(()=>reads.get(roots[i].id)).toBe(1);
  }
  await tree.getByRole('treeitem',{name:'加载根部门',exact:true}).click();
  await tree.getByRole('button',{name:'展开预算根0',exact:true}).click();
  await expect.poll(()=>reads.get(roots[0].id)).toBe(2);
});

test('300展示缓存超过2000节点后同级窗口可双向重读',async({page})=>{
  // 81 sequential real UI page actions; endpoint/assertion budgets stay fixed.
  test.setTimeout(300000);const {roots,tree}=await displayBudget(page);
  const action=async(name:string)=>{
    await tree.evaluate((el,previous)=>new Promise<void>(done=>{el.scrollTop=previous?0:el.scrollHeight;requestAnimationFrame(()=>requestAnimationFrame(()=>done()))}),name==='加载较早部门');
    const height=await tree.evaluate(el=>el.scrollHeight);
    await tree.getByRole('treeitem',{name,exact:true}).click();
    await expect.poll(()=>tree.evaluate(el=>el.scrollHeight)).not.toBe(height);
    expect(await tree.getByRole('treeitem').count()).toBeLessThanOrEqual(200);
  };
  for(let i=0;i<41;i++)await action('加载更多部门');
  await tree.evaluate(el=>{el.scrollTop=el.scrollHeight});
  await expect(tree.locator(`[data-tree-id="${roots[2099].id}"]`)).toBeVisible();
  for(let i=0;i<40;i++)await action('加载较早部门');
  await tree.evaluate(el=>{el.scrollTop=0});
  await expect(tree.locator(`[data-tree-id="${roots[0].id}"]`)).toBeVisible();
});

test('300树撤权后503及迟到成功均不能恢复已清除内容',async({page})=>{
  test.setTimeout(120000);
  await page.addInitScript(()=>{const fetchOriginal=window.fetch;window.fetch=async(input,init)=>{const r=new Request(input,init);return fetchOriginal(r.url.includes('/tree/nodes')?new Request(r,{signal:new AbortController().signal}):r)}});
  await open(page);const bu=await apiBU(page),prefix='000_REVOKE300'+Date.now();
  const root=await apiDepartment(page,bu.id,{code:prefix+'A'}),child=await apiDepartment(page,bu.id,{code:prefix+'C',parent:root.id});
  await menu(page,departmentKind);
  // Other cases share this temporary database. Restrict the independent list
  // to this case's records before asserting that tree failure leaves it intact.
  const listSearch=page.getByRole('textbox',{name:'搜索编码或名称',exact:true});
  await listSearch.fill(prefix);await listSearch.press('Enter');
  await expect(page.getByRole('grid',{name:'部门列表',exact:true}).locator('.MuiDataGrid-row')).toHaveCount(2);
  let release!:()=>void,captured!:()=>void,finished!:()=>void;
  const held=new Promise<void>(r=>release=r),arrived=new Promise<void>(r=>captured=r),done=new Promise<void>(r=>finished=r);
  await page.route('**/api/v1/departments/tree/nodes*',async route=>{
    if(new URL(route.request().url()).searchParams.get('parentId')!==root.id){await route.continue();return;}
    const response=await route.fetch();expect(response.status()).toBe(200);captured();await held;
    await route.fulfill({response}).catch(()=>{});finished();
  });
  let searches=0;
  await page.route('**/api/v1/departments/tree/search*',route=>{searches++;if(searches>2)return route.continue();return route.fulfill({status:searches===1?403:503,json:{code:searches===1?'ACCESS_DENIED':'M02_DEPENDENCY_UNAVAILABLE',message:searches===1?'受控撤权':'受控暂时失败'}})});
  try {
    await page.getByRole('button',{name:'展开'+root.code,exact:true}).click();await arrived;
    const search=page.getByRole('textbox',{name:'定位树中部门',exact:true});await search.fill(prefix);
    await expect(page.getByText('受控撤权',{exact:true})).toBeVisible();
    await page.getByRole('button',{name:'重试',exact:true}).click();await expect.poll(()=>searches).toBe(2);
    await expect(page.getByText('受控暂时失败',{exact:true})).toBeVisible();
    await expect(page.getByText('受控撤权',{exact:true})).toHaveCount(0);
    release();await done;await search.fill('');
    await expect(page.getByRole('tree')).toBeVisible();
    await expect(page.getByRole('treeitem',{name:'加载根部门',exact:true})).toBeVisible();
    for(const id of [root.id,child.id])await expect(page.locator(`[data-tree-id="${id}"]`)).toHaveCount(0);
    await expect(page.getByRole('grid',{name:'部门列表',exact:true}).locator('.MuiDataGrid-row')).toHaveCount(2);
    await search.fill(prefix);
    for(const id of [root.id,child.id])await expect(page.locator(`[data-tree-id="${id}"]`)).toBeVisible();
    await expect(page.getByText('受控暂时失败',{exact:true})).toHaveCount(0);
  } finally {release();await page.unrouteAll({behavior:'ignoreErrors'});}
});

test('027 首次定位保留在部门上下文就绪前输入的最后一个编码', async ({ page }) => {
  await open(page);
  const bu = await apiBU(page);
  const department = await apiDepartment(page, bu.id);
  let signalReady!: () => void, release!: () => void;
  const received = new Promise<void>(resolve => { signalReady = resolve; });
  const resume = new Promise<void>(resolve => { release = resolve; });
  const calls: string[] = [];
  page.on('request', request => {
    const url = new URL(request.url());
    if (url.pathname === '/api/v1/departments/tree/search') calls.push(url.searchParams.get('q') ?? '');
  });
  const endpoint = '**/api/v1/departments/context';
  await page.route(endpoint, async route => {
    const response = await route.fetch();
    expect(response.status()).toBe(200);
    signalReady();
    await resume;
    await route.fulfill({ response });
  });
  try {
    await menu(page, departmentKind);
    await received;
    const input = page.getByRole('textbox', { name: '定位树中部门', exact: true });
    await page.clock.install();
    await input.fill('已替换');
    await input.fill(department.code);
    await page.clock.runFor(350);
    expect(calls).toEqual([]);
    release();
    const candidate = page.getByRole('option').filter({ hasText: department.code });
    await expect(candidate).toHaveCount(1);
    expect(calls).toEqual([department.code]);
    await input.press('Tab');
    await candidate.click();
    await expect(input).toHaveValue('');
  } finally {
    release();
    await page.unrouteAll({ behavior: 'wait' });
  }
});
