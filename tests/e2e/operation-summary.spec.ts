import {test, expect, expectLogChange, type Page} from './fixtures';
import {open,create,add,read,select,enter,save,choose,readyToSave} from './bu-support';
import {mkdirSync} from 'node:fs';

test.setTimeout(90_000);
async function displayed(page:Page){return page.evaluate(async()=>{const p=new URLSearchParams(location.search),q=new URLSearchParams();for(const key of ['sourceVersion','selectedRecordId'])if(p.has(key))q.set(key,p.get(key)!);const r=await fetch('/api/v1/business-units/'+location.pathname.split('/').at(-1)+'?'+q);if(!r.ok)throw new Error('displayed '+r.status);return r.json()})}
const evidence=process.env.CHNHCM_EVIDENCE_DIR ?? '.local/122-browser';
mkdirSync(evidence,{recursive:true});
async function screenshot(page:Page,name:string){mkdirSync(evidence,{recursive:true});await page.screenshot({path:`${evidence}/${name}.png`,fullPage:true,mask:[page.locator('.app-account-name')]})}
async function logs(page:Page){await page.getByRole('tab',{name:'变更日志',exact:true}).click();await expect(page.locator('.change-operation').first()).toBeVisible();await expect(page.getByRole('region',{name:'业务单元详情正文'})).toHaveAttribute('data-page-scroll-ready','true')}
async function changeName(page:Page,name:string){await enter(page,'更正记录');await page.getByRole('textbox',{name:'中文名称',exact:true}).fill(name);await save(page)}

test('TL367–375 正常菜单摘要、真实差异、删除两类原件和准确链接返回',async({page,context})=>{
 await open(page);await create(page);const s1=await displayed(page);
 await expect(page.locator('.operation-summary')).toContainText('操作者：admin');await expect(page.locator('.operation-summary')).toContainText('操作时间：');
 await expect(page.getByRole('button',{name:'来源说明',exact:true})).toHaveCount(0);
 await enter(page,'更正记录');await page.getByRole('button',{name:'取消编辑',exact:true}).click();expect((await displayed(page)).selectedRecord.operationSummary).toEqual(s1.selectedRecord.operationSummary);
 await add(page,'2026-06-01','乙','Beta');await add(page,'2026-10-01','丙');await select(page,'2026-06-01');await changeName(page,'乙新');
 const s4=await displayed(page);expect(s4.selectedRecord.operationSummary.operationSourceVersion).toBe(s4.headSourceVersion);
 await logs(page);await expectLogChange(page,'中文名称','乙','乙新');
 const group=page.locator('.change-operation').first();await expect(group.locator('[data-label="生效记录"]')).toContainText('2026-06-01');await expect(group.getByRole('button',{name:'来源明细'})).toHaveAttribute('aria-expanded','false');
 await group.getByRole('button',{name:'来源明细'}).focus();await page.keyboard.press('Enter');await expect(group.getByText('来源版本：'+s4.headSourceVersion,{exact:true})).toBeVisible();await page.keyboard.press('Enter');await expect(group.getByText('来源版本：'+s4.headSourceVersion,{exact:true})).toHaveCount(0);
 const link=group.getByRole('link',{name:'查看当次资料',exact:true});const href=await link.getAttribute('href');expect(href).toContain('sourceVersion='+s4.headSourceVersion);expect(href).toContain('selectedRecordId='+s4.selectedRecord.recordId);expect(href).not.toContain('businessAt');
 const originURL=page.url();const opened=context.waitForEvent('page');await link.click({modifiers:['Control']});const child=await opened;await child.waitForLoadState();await expect(child.getByText('正在查看只读历史资料',{exact:true})).toBeVisible();expect(page.url()).toBe(originURL);await child.close();
 await screenshot(page,'log-desktop');await link.click();await expect(page.getByText('正在查看只读历史资料',{exact:true})).toBeVisible();await expect(page.locator('.operation-summary')).toContainText('操作者：admin');await page.getByRole('link',{name:'返回变更日志',exact:true}).click();await expect(link).toBeFocused();expect(page.url()).toBe(originURL);
 await page.getByRole('tab',{name:'基本资料',exact:true}).click();await enter(page,'删除此记录');await readyToSave(page);const deleting=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/delete'));await page.getByRole('button',{name:'确认删除',exact:true}).click();const deleted=await(await deleting).json();await expect(page.locator('.object-form')).toHaveCount(0);
 await logs(page);const deletion=page.locator('.change-operation').first();const before=deletion.getByRole('link',{name:'查看删除前原件',exact:true});const after=deletion.getByRole('link',{name:'查看删除后版本',exact:true});expect(await before.getAttribute('href')).toContain('sourceVersion='+s4.headSourceVersion);expect(await after.getAttribute('href')).not.toContain('selectedRecordId=');
 await before.click();expect((await displayed(page)).selectedRecord.operationSummary).toEqual(s4.selectedRecord.operationSummary);await page.getByRole('link',{name:'返回变更日志',exact:true}).click();await expect(before).toBeFocused();await after.click();const afterDetail=await displayed(page);expect(afterDetail.resolvedSourceVersion).toBe(deleted.sourceVersion);expect(afterDetail.timeline).toHaveLength(2);expect(afterDetail.selectedRecord.recordId).not.toBe(s4.selectedRecord.recordId);
 const tombstoneURL=new URL(page.url());tombstoneURL.searchParams.set('selectedRecordId',s4.selectedRecord.recordId);await page.goto(tombstoneURL.toString());await expect(page.getByText('已删除记录原件，只读。下方保留删除前内容与期间。',{exact:true})).toBeVisible();const tombstone=await displayed(page);expect(tombstone.selectedRecord.revision).toBe(deleted.targetRevision);expect(tombstone.selectedRecord.operationSummary).toEqual(s4.selectedRecord.operationSummary);
 await screenshot(page,'deleted-original');
});

test('TL376–382 日志筛选草稿、操作分页、固定上界、历史刷新和窄屏键盘',async({page})=>{
 await open(page);await create(page,'2026-01-01','');const initial=await displayed(page);
 // Synthetic volume is created through the authenticated public write API. The
 // actual navigation, filters, paging and links below use normal page controls.
 await page.evaluate(async({id,record,source,revision})=>{
  const session=await(await fetch('/api/v1/session')).json();
  for(let n=2;n<=45;n++){
   const response=await fetch(`/api/v1/business-units/${id}/records/${record}/correct`,{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':session.csrf,'Idempotency-Key':crypto.randomUUID()},body:JSON.stringify({expectedSourceVersion:source,expectedRecordRevision:revision,changes:{name:`样本${n}`,...(n%3===0?{nameEn:`EN${n}`}:{})}})});
   if(!response.ok)throw new Error('fixture correction '+response.status);const saved=await response.json();source=saved.sourceVersion;revision=saved.targetRevision;
  }
 },{id:initial.id,record:initial.selectedRecord.recordId,source:initial.headSourceVersion,revision:initial.selectedRecord.revision});
 await page.reload();await logs(page);await expect(page.getByText('1–20 / 45 项操作',{exact:true})).toBeVisible();await expect(page.locator('.change-operation')).toHaveCount(20);const upper=new URL(page.url()).searchParams.get('logThrough');expect(upper).toBeTruthy();
 await page.getByRole('button',{name:'下一页',exact:true}).click();await expect(page.getByText('21–40 / 45 项操作',{exact:true})).toBeVisible();await page.getByRole('button',{name:'下一页',exact:true}).click();await expect(page.locator('.change-operation')).toHaveCount(5);await expect(page.getByText('41–45 / 45 项操作',{exact:true})).toBeVisible();
 await page.getByRole('button',{name:'变更字段列菜单'}).click();await page.getByRole('menuitem',{name:'筛选…',exact:true}).click();let dialog=page.getByRole('dialog');await choose(dialog,'值','英文名称');await dialog.getByRole('button',{name:'取消',exact:true}).click();await expect(page.getByText('41–45 / 45 项操作',{exact:true})).toBeVisible();
 await page.getByRole('button',{name:'变更字段列菜单'}).click();await page.getByRole('menuitem',{name:'筛选…',exact:true}).click();dialog=page.getByRole('dialog');await choose(dialog,'值','英文名称');await dialog.getByRole('button',{name:'应用',exact:true}).click();await expect(page.getByText('1–15 / 15 项操作',{exact:true})).toBeVisible();expect(new URL(page.url()).searchParams.get('logThrough')).toBe(upper);
 const appliedURL=page.url();await page.getByRole('button',{name:'列筛选…',exact:true}).click();await page.getByRole('dialog').getByRole('button',{name:'清除条件',exact:true}).click();await page.keyboard.press('Escape');expect(page.url()).toBe(appliedURL);
 await page.getByRole('button',{name:'列筛选…',exact:true}).click();await page.getByRole('dialog').getByRole('button',{name:'清除条件',exact:true}).click();await page.getByRole('dialog').getByRole('button',{name:'应用',exact:true}).click();await expect(page.getByText('1–20 / 45 项操作',{exact:true})).toBeVisible();
 // Width changes keep the same applied URL and groups, while values stay visible.
 for(const width of [390,320]){await page.setViewportSize({width,height:900});await expect(page.locator('.change-operation').first().getByText('样本45',{exact:true}).first()).toBeVisible();await page.getByRole('button',{name:'列筛选…',exact:true}).click();await page.keyboard.press('Escape');await expect(page.getByRole('button',{name:'列筛选…',exact:true})).toBeFocused();expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true)}
 await screenshot(page,'log-320');
 await page.setViewportSize({width:1440,height:900});await page.getByRole('link',{name:'查看当次资料',exact:true}).last().click();const historical=await displayed(page);await logs(page);const firstContext=new URL(page.url()).searchParams.get('sourceVersion');expect(firstContext).toBe(historical.resolvedSourceVersion);await page.getByRole('button',{name:'刷新日志',exact:true}).click();await expect(page.locator('.change-operation').first()).toBeVisible();expect(new URL(page.url()).searchParams.get('sourceVersion')).toBe(firstContext);await expect(page).toHaveURL(u=>u.searchParams.get('logThrough')===firstContext);
});

test('TL367/381 未来首条有摘要，无变化与失败不刷新摘要',async({page})=>{
 await open(page);await create(page,'9998-11-10');const before=await displayed(page);expect(before.result).toBe('M02_NO_FACT');await expect(page.locator('.operation-summary')).toContainText('操作者：admin');
 await enter(page,'更正记录');await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.getByText('没有修改内容。',{exact:true})).toBeVisible();await page.getByRole('button',{name:'取消编辑',exact:true}).click();expect((await displayed(page)).selectedRecord.operationSummary).toEqual(before.selectedRecord.operationSummary);
 await screenshot(page,'future-summary');
});
