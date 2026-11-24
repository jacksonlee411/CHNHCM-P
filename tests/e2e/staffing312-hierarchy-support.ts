import { randomUUID } from 'node:crypto';
import { expect, test, type Page } from './fixtures';
import { request } from './dated-object-support';
import { click308, namedMenu308, reference308 } from './staffing308-support';
import { seedPosition } from './staffing312-support';

const path='/api/v1/position-hierarchies/reporting';
export async function seedHierarchy(page: Page, _kind:'hierarchy', family:string) {
  const child=await seedPosition(page,'position',family);
  const detail=await request(page,'GET','/api/v1/positions/'+child.id);
  const {departmentId,jobId,locationId}=detail.body.selectedRecord;
  const position=async(prefix:string)=>{
    const name=prefix+child.code;
    const result=await request(page,'POST','/api/v1/positions',{departmentId,jobId,locationId,effectiveDate:'2026-01-01',operationType:'CREATE',operationReason:'NEW_POSITION',code:prefix==='原上级'?'OLD'+child.code:'NEW'+child.code,name},randomUUID());
    expect(result.status).toBe(201);return {id:result.body.objectId as string,name};
  };
  const first=await position('原上级'), second=await position('新上级');
  const initial=await request(page,'GET',path);
  const publish=async(expectedVersion:string,effectiveDate:string,parentId:string)=>{
    const result=await request(page,'POST',path+'/publish',{expectedVersion,effectiveDate,childId:child.id,parentId},randomUUID());
    expect(result.status).toBe(200);return result.body.sourceVersion as string;
  };
  const older=await publish(initial.body.sourceVersion,'2027-01-01',first.id);
  await publish(older,'2027-05-01',second.id);
  const view=await request(page,'GET',path);
  return {child:{...child,code:detail.body.code as string},first,second,older,number:view.body.versions.find((v:{sourceVersion:string})=>v.sourceVersion===older).number as number};
}
export async function structureMenu(page:Page) {
  await namedMenu308(page,'岗位','职务配置','/settings/staffing/positions');
  await click308(page,page.getByRole('button',{name:'更多操作',exact:true}));
  await click308(page,page.getByRole('menuitem',{name:'查看岗位上下级',exact:true}));
  await expect(page.getByRole('heading',{name:'岗位上下级'})).toBeVisible();
}
export async function queryStructure(page:Page,date:string) {
  const input = page.getByRole('textbox',{name:/结构日期/});
  const parameter = new URL(page.url()).pathname.endsWith('/position-hierarchies') ? 'businessAt' : 'structureDate';
  await input.fill(date);
  await click308(page,page.getByRole('button',{name:'查询结构'}));
  await expect(page).toHaveURL(url => url.searchParams.get(parameter) === date);
  await expect(page.locator('.standard-page')).toHaveAttribute('aria-busy','false');
  await expect(input).toHaveValue(date);
}
export async function structureDetail(page:Page,name:string) {
  await page.getByRole('textbox',{name:'查找岗位'}).fill(name);
  await click308(page,page.getByRole('button',{name:'查询',exact:true}));
  await click308(page,page.getByRole('link',{name:new RegExp('^'+name)}).first());
  await expect(page.getByRole('tab',{name:'岗位关系'})).toHaveAttribute('aria-selected','true');
}
export async function adjust(page:Page){
  await click308(page,page.getByRole('button',{name:'更多操作'}));
  await click308(page,page.getByRole('menuitem',{name:'调整上级'}));
  await expect(page.getByRole('heading',{name:'调整上级'})).toBeVisible();
}
export async function chooseParent(page:Page,name:string){
  const input=page.getByRole('combobox',{name:/新上级/});
  await click308(page,input);
  await input.fill(name);
  await click308(page,page.getByRole('option',{name:new RegExp('^'+name)}));
  await expect(input).toHaveValue(new RegExp('^'+name));
}
export async function hierarchyJourney(page:Page,sample:Awaited<ReturnType<typeof seedHierarchy>>,_family:unknown,role:string){
  page.setDefaultTimeout(15000);
  await reference308(page,role);await structureMenu(page);
  await queryStructure(page,'2027-04-01');await structureDetail(page,sample.child.name);
  const relation=page.locator('#staffing-relations-panel');
  await expect(relation).toContainText(sample.first.name);
  await expect(page.getByRole('combobox',{name:/用途/})).toHaveCount(0);
  if(role==='admin'){
    await adjust(page);await chooseParent(page,sample.second.name);
    await click308(page,page.getByRole('button',{name:'取消',exact:true}));await click308(page,page.getByRole('button',{name:'放弃修改'}));
    await expect(relation).toContainText(sample.first.name);
    await adjust(page);await chooseParent(page,sample.second.name);
    for(const day of ['2027-03-31','2027-04-01']){
      await page.getByRole('textbox',{name:/生效日期/}).fill(day);
      await expect(page.getByRole('combobox',{name:/新上级/})).toHaveValue(new RegExp('^'+sample.second.name));
      await expect(page.getByRole('button',{name:'保存',exact:true})).toBeEnabled();
    }
    await page.screenshot({path:test.info().outputPath('arrange-'+await page.evaluate(()=>innerWidth)+'.png'),mask:[page.locator('.app-account-name')]});
    const saved=page.waitForResponse(r=>r.request().method()==='POST' && r.url().endsWith(path+'/publish'));
    await click308(page,page.getByRole('button',{name:'保存',exact:true}));expect((await saved).status()).toBe(200);
    await expect(relation).toContainText(sample.second.name);
  } else {
    await expect(page.getByRole('menuitem',{name:'调整上级'})).toHaveCount(0);
    await queryStructure(page,'2027-05-01');await expect(relation).toContainText(sample.second.name);
  }
  await click308(page,page.getByRole('link',{name:'返回岗位上下级'}));
  await click308(page,page.getByRole('button',{name:'发布历史'}));
  const version=page.getByRole('button',{name:new RegExp('^第 '+sample.number+' 版')});
  await click308(page,version);await structureDetail(page,sample.child.name);
  await expect(relation).toContainText(sample.first.name);
  await expect(page.getByRole('button',{name:'更多操作'})).toHaveCount(0);
  await click308(page,page.getByRole('button',{name:'返回当前结构'}));
  await expect(page.getByRole('heading',{name:'岗位上下级'})).toBeVisible();
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
}
