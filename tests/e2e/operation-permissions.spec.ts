import { accessFixture } from './operation-access';
import { test, expect, type Page } from './fixtures';
import { open, create, enter, save } from './bu-support';
import { execFileSync, spawn } from 'node:child_process';
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import { chromium } from '@playwright/test';

test.setTimeout(120_000);
const evidence=process.env.CHNHCM_EVIDENCE_DIR ?? '.local/122-browser';
mkdirSync(evidence,{recursive:true});
async function currentDetail(page:Page){return page.evaluate(async()=>{const path='/api/v1/business-units/'+location.pathname.split('/').at(-1);const r=await fetch(path);if(!r.ok)throw new Error('detail '+r.status);return r.json()})}

test('TL367/374/375/380/381 真实撤权、字段裁剪、旧链接和摘要分权',async({page})=>{
 await open(page);const rights=await accessFixture(page);
 try {
  rights.sql(`UPDATE access.principal SET display_name='合成操作员',principal_kind='human' WHERE ${rights.where}`);
  await create(page,'9998-11-10');const before=await currentDetail(page);const currentURL=page.url();
  await expect(page.locator('.operation-summary')).toContainText('合成操作员（admin）');
  rights.sql(`UPDATE access.principal SET display_name='更名后的账号',principal_kind='service' WHERE ${rights.where}`);
  await page.reload();await expect(page.locator('.operation-summary')).toContainText('合成操作员（admin）');
  await enter(page,'更正记录');await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('服务主体资料');await save(page);await expect(page.locator('.operation-summary')).toContainText('更名后的账号（admin）（服务主体）');
  const saved=await currentDetail(page);
  rights.mutate(`UPDATE access.subject_grant SET read_fields=ARRAY['code','name','nameEn'] WHERE ${rights.where} AND action IN ('bu.read','bu.history.read')`);
  await page.reload();await expect(page.locator('.operation-summary')).toBeVisible();await expect(page.getByText('所选记录尚未生效。',{exact:true})).toHaveCount(0);await expect(page.getByText('原适用期间',{exact:true})).toHaveCount(0);
  await page.getByRole('tab',{name:'变更日志',exact:true}).click();await expect(page.locator('.change-operation').first()).toBeVisible();await expect(page.locator('.change-log')).not.toContainText('9998-11-10');await expect(page.locator('.change-log')).not.toContainText('新设组织');
  await page.getByRole('button',{name:'列筛选…',exact:true}).click();await page.getByRole('dialog').getByRole('combobox',{name:'列',exact:true}).click();await expect(page.getByRole('option',{name:'生效记录',exact:true})).toHaveCount(0);await page.keyboard.press('Escape');await expect(page.getByRole('listbox')).toHaveCount(0);await page.getByRole('dialog').getByRole('button',{name:'取消',exact:true}).click();
  const original=page.locator('.change-operation').last().getByRole('link',{name:'查看当次资料',exact:true});const oldHref=await original.getAttribute('href');expect(oldHref).toContain(before.headSourceVersion);
  await original.click();await expect(page.locator('.operation-summary')).toContainText('合成操作员（admin）');await page.getByRole('link',{name:'返回变更日志',exact:true}).click();await expect(page.locator('.change-operation').first()).toBeVisible();
  const invalid = new URL(page.url());invalid.searchParams.set('logFilterField','unknown');invalid.searchParams.set('logFilterOperator','equals');invalid.searchParams.set('logFilterValue','x');await page.goto(invalid.href);
  await expect(page.getByRole('button',{name:'重置日志条件',exact:true})).toBeVisible();await page.getByRole('button',{name:'重置日志条件',exact:true}).click();await expect(page.locator('.change-operation')).toHaveCount(2);
  const forbidden = new URL(page.url());forbidden.searchParams.set('logFilterField','effectiveDate');forbidden.searchParams.set('logFilterOperator','is');forbidden.searchParams.set('logFilterValue','9998-11-10');await page.goto(forbidden.href);
  await expect(page.locator('.operation-summary')).toHaveCount(0);await expect(page.getByRole('button',{name:'重试',exact:true})).toBeVisible();expect(new URL(page.url()).searchParams.has('logFilterField')).toBe(false);await page.getByRole('button',{name:'重试',exact:true}).click();await expect(page.locator('.operation-summary')).toBeVisible();await page.getByRole('tab',{name:'变更日志',exact:true}).click();await expect(page.locator('.change-operation')).toHaveCount(2);
  rights.mutate(`DELETE FROM access.subject_grant WHERE ${rights.where} AND action='bu.history.read'`);
  await page.getByRole('button',{name:'刷新日志',exact:true}).click();await expect(page.getByText('当前权限不允许读取日志',{exact:true})).toBeVisible();await expect(page.locator('.change-operation')).toHaveCount(0);await expect(page.locator('.operation-summary')).toHaveCount(0);
  await page.getByRole('button',{name:'重试',exact:true}).click();await expect(page.locator('.operation-summary')).toContainText('服务主体');await expect(page.getByRole('tab',{name:'变更日志',exact:true})).toHaveCount(0);
  await page.goto(oldHref!);await expect(page.getByText('当前权限不允许读取历史',{exact:true})).toBeVisible();await expect(page.locator('.operation-summary')).toHaveCount(0);
  await page.getByRole('button',{name:'返回当前资料',exact:true}).click();await expect(page.locator('.operation-summary')).toContainText('服务主体');
  expect((await currentDetail(page)).selectedRecord.operationSummary).toEqual(saved.selectedRecord.operationSummary);
  await page.screenshot({path:`${evidence}/summary-without-history.png`,mask:[page.locator('.app-account-name')]});
  rights.mutate(`DELETE FROM access.subject_grant WHERE ${rights.where} AND action='bu.read'`);await page.reload();await expect(page.locator('.operation-summary')).toHaveCount(0);await expect(page.locator('.object-detail-field')).toHaveCount(0);expect(currentURL).toContain('/business-units/');
 } finally {rights.restore();rights.sql(`UPDATE access.principal SET display_name=NULL,principal_kind='human' WHERE ${rights.where}`)}
});

test('TL380/382 在途日志失效、历史切换和原上界刷新',async({page})=>{
 await open(page);await create(page);const original=await currentDetail(page);const url=page.url();let release!:()=>void;
 const held=new Promise<void>(r=>{release=r});let captured=false;
 await page.route('**/api/v1/business-units/*/changes?*',async route=>{const response=await route.fetch();captured=true;await held;await route.fulfill({response}).catch(()=>{})});
 await page.getByRole('tab',{name:'变更日志',exact:true}).click();await expect.poll(()=>captured).toBe(true);
 const basicTab = page.getByRole('tab', {name: '基本资料', exact: true});
 await basicTab.click();
 release();
 await page.unroute('**/api/v1/business-units/*/changes?*');
 // The pending log also has zero rows. Wait for the selected tab before clicking history again.
 await expect(basicTab).toHaveAttribute('aria-selected', 'true');
 await expect(page).toHaveURL(url => url.searchParams.get('tab') === 'basic');
 await expect(page.locator('.change-operation')).toHaveCount(0);
 await page.getByRole('tab',{name:'变更日志',exact:true}).click();await expect(page.locator('.change-operation')).toHaveCount(1);const pinned=new URL(page.url()).searchParams.get('logThrough');
 await page.evaluate(async id=>{const session=await(await fetch('/api/v1/session')).json();const detail=await(await fetch('/api/v1/business-units/'+id)).json();const r=detail.selectedRecord;const response=await fetch(`/api/v1/business-units/${id}/records/${r.recordId}/correct`,{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':session.csrf,'Idempotency-Key':crypto.randomUUID()},body:JSON.stringify({expectedSourceVersion:detail.headSourceVersion,expectedRecordRevision:r.revision,changes:{name:'并发的新资料'}})});if(!response.ok)throw new Error('write '+response.status)},original.id);
 await page.reload();await expect(page.locator('.change-operation')).toHaveCount(1);expect(new URL(page.url()).searchParams.get('logThrough')).toBe(pinned);
 await page.getByRole('button',{name:'刷新日志',exact:true}).click();await expect(page.locator('.change-operation')).toHaveCount(2);expect(new URL(page.url()).searchParams.get('logThrough')).not.toBe(pinned);
 await page.getByRole('link',{name:'查看当次资料',exact:true}).last().click();
 await expect(page.getByRole('button',{name:'返回当前资料',exact:true})).toBeVisible();
 await expect(page.getByRole('tab',{name:'基本资料',exact:true})).toHaveAttribute('aria-selected','true');
 await page.getByRole('tab',{name:'变更日志',exact:true}).click();
 await expect(page.locator('.change-operation')).toHaveCount(1);
 await page.getByRole('button',{name:'刷新日志',exact:true}).click();
 await expect(page).toHaveURL(u=>u.searchParams.get('logThrough')===pinned);
 await page.getByRole('button',{name:'返回当前资料',exact:true}).click();await expect(page).not.toHaveURL(/sourceVersion=/);await expect(page.locator('.change-operation')).toHaveCount(2);expect(url).toContain('/business-units/');
});

test('TL383 桌面与原生Chromium 200%长姓名账号及字段值',async()=>{
 let n=1500+process.pid%1000;while(existsSync(`/tmp/.X11-unix/X${n}`)||existsSync(`/tmp/.X${n}-lock`))n++;
 const env={...process.env,DISPLAY:`:${n}`},xvfb=spawn('Xvfb',[`:${n}`,'-screen','0','2200x1400x24','-nolisten','tcp','-nolisten','unix'],{stdio:'ignore'});
 let browser:Awaited<ReturnType<typeof chromium.launch>>|undefined;let rights:Awaited<ReturnType<typeof accessFixture>>|undefined;
 try {
  await expect.poll(()=>{try{execFileSync('xdpyinfo',[],{env,stdio:'ignore',timeout:1000});return true}catch{return false}}).toBe(true);
  browser=await chromium.launch({headless:false,env,args:['--window-size=1920,1219','--window-position=0,0','--ozone-platform=x11']});
  const context=await browser.newContext({viewport:null,deviceScaleFactor:undefined}),page=await context.newPage();await open(page);rights=await accessFixture(page);
  rights.sql(`UPDATE access.principal SET display_name='合成的超长姓名用于检验自然换行与操作身份可复制显示' WHERE ${rights.where}`);
  await create(page);await expect(page.locator('.operation-summary')).toContainText('合成的超长姓名');
  await enter(page,'更正记录');await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('包含较长中文组织名称的更正后资料用于检查差异默认展示');await save(page);
  const cdp=await context.newCDPSession(page),{windowId}=await cdp.send('Browser.getWindowForTarget');
  const dimensions=()=>page.evaluate(()=>({width:innerWidth,height:innerHeight,dpr:devicePixelRatio,outer:[outerWidth,outerHeight]}));
  const zoom=(steps:number)=>execFileSync('python3',['docs/verification/data/100-ui-baseline/native-zoom.py',String(steps)],{env,stdio:'pipe'});
  zoom(0);mkdirSync(evidence,{recursive:true});const matrix=[];
  for(const [width,height] of [[1920,1080],[1366,768]]){
   const current=await dimensions();await cdp.send('Browser.setWindowBounds',{windowId,bounds:{width:current.outer[0]+width-current.width,height:current.outer[1]+height-current.height}});await expect.poll(async()=>(await dimensions()).width).toBe(width);
   await page.getByRole('tab',{name:'基本资料',exact:true}).click();await expect(page.locator('.operation-summary')).toBeVisible();await page.getByRole('tab',{name:'变更日志',exact:true}).click();await expect(page.locator('.change-operation').first()).toContainText('包含较长中文组织名称');
   await page.getByRole('button',{name:'来源明细',exact:true}).first().focus();await page.keyboard.press('Enter');await expect(page.getByRole('button',{name:'来源明细',exact:true}).first()).toHaveAttribute('aria-expanded','true');await page.keyboard.press('Enter');expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
   await page.screenshot({path:`${evidence}/viewport-${width}.png`,mask:[page.locator('.app-account-name')]});matrix.push({...await dimensions(),result:'passed'});
  }
  const current=await dimensions();await cdp.send('Browser.setWindowBounds',{windowId,bounds:{width:current.outer[0]+1366-current.width,height:current.outer[1]+768-current.height}});zoom(5);await expect.poll(async()=>(await dimensions()).dpr).toBe(2);
  await page.getByRole('button',{name:'列筛选…',exact:true}).focus();await page.keyboard.press('Enter');await page.keyboard.press('Escape');await expect(page.getByRole('button',{name:'列筛选…',exact:true})).toBeFocused();await page.mouse.wheel(0,450);expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);matrix.push({...await dimensions(),zoom:'native 200%',result:'passed'});const target=page.locator('.change-operation').first().getByRole('link',{name:'查看当次资料',exact:true}).first();await target.scrollIntoViewIfNeeded();await expect(target).toBeInViewport();await expect(page.locator('.change-operation').first().locator('[data-label="新值"]').first()).toBeInViewport();
  const mask=await page.addStyleTag({content:'.app-account-name{visibility:hidden!important}'});const screenshot=await cdp.send('Page.captureScreenshot',{format:'png',captureBeyondViewport:false,fromSurface:true});const pixels=Buffer.from(screenshot.data,'base64');expect(pixels.readUInt32BE(16)).toBe(1366);expect(pixels.readUInt32BE(20)).toBe(768);writeFileSync(`${evidence}/native-200.png`,pixels);await mask.evaluate(e=>e.remove());zoom(0);await expect.poll(async()=>(await dimensions()).dpr).toBe(1);writeFileSync(`${evidence}/viewport-matrix.json`,JSON.stringify(matrix,null,2)+'\n');
 } finally {if(rights){rights.restore();rights.sql(`UPDATE access.principal SET display_name=NULL WHERE ${rights.where}`)}await browser?.close();xvfb.kill('SIGTERM');await new Promise<void>(resolve=>{if(xvfb.exitCode!==null)resolve();else xvfb.once('exit',()=>resolve())})}
});
