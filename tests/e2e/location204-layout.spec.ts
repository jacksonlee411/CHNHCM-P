import { nativeZoomCases } from './desktop-viewports.mjs';
import {chromium} from '@playwright/test';
import {spawn,execFileSync} from 'node:child_process';
import {existsSync,mkdirSync,writeFileSync} from 'node:fs';
import {test,expect,openMoreActions} from './fixtures';
import {open} from './bu-support';
import {addressField,create,menu} from './location204-support';
import {captureViewport} from '../../docs/verification/data/100-ui-capture-fix/capture-viewport.mjs';

test('地点204 双角色桌面视口与列表断点、表单键盘和地址可达',async({page})=>{
 test.setTimeout(360000);const evidence=`.local/g4-implementation/layout/${Date.now()}-viewports`;mkdirSync(evidence,{recursive:true});
 await open(page);const code=await create(page,'澳门');const path=new URL(page.url()).pathname;const matrix=[];
 for(const role of ['admin','reader']){
  await page.setViewportSize({width:1440,height:900});await page.context().clearCookies();await open(page,role);
  for(const [width,height] of [[1440,900],[1000,900],[900,900],[899,900],[1440,540]]){
   await page.setViewportSize({width,height});await menu(page);await page.getByRole('textbox',{name:'搜索编码或名称',exact:true}).fill(code);await page.getByRole('button',{name:'查询',exact:true}).click();
   const link=page.getByRole('link',{name:code+'办公地点',exact:true});await link.scrollIntoViewIfNeeded();await expect(link).toBeInViewport();await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
   await captureViewport(page,`${evidence}/${role}-${width}-${height}-list.png`,[page.locator('.app-account-name')]);
   const loaded = page.waitForResponse(r => r.request().method() === 'GET' && new URL(r.url()).pathname === path.replace('/settings/organization', '/api/v1'));
   await link.click();
   await loaded;
   await expect(page).toHaveURL(new RegExp(path));
   await expect(page.getByRole('region', { name: '地点详情正文', exact: true })).toHaveAttribute('data-page-scroll-ready', 'true');
   // Navigation resolves after the detail. Its second row changes the available
   // body height on narrow screens, so measure the settled layout.
   await expect(page.getByRole('navigation', { name: '对象导航', exact: true })).toContainText('1 / 1');
   const street = addressField(page, '详细地址'), fullAddress = addressField(page, '完整地址');
   await expect(street).toBeVisible();
   await street.scrollIntoViewIfNeeded();
   await expect(street).toBeInViewport();
   await expect(fullAddress).toHaveText('测试街一号');
   await fullAddress.scrollIntoViewIfNeeded();
   await expect(fullAddress).toBeInViewport();
   if(role==='admin'){
    await openMoreActions(page);await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();const address=page.getByRole('textbox',{name:'详细地址',exact:true});await address.scrollIntoViewIfNeeded();await address.focus();await page.keyboard.press('Tab');await expect(page.getByRole('textbox',{name:'邮政编码',exact:true})).toBeFocused();
    await captureViewport(page,`${evidence}/${role}-${width}-${height}-form.png`,[page.locator('.app-account-name')]);await page.getByRole('button',{name:'保存',exact:true}).scrollIntoViewIfNeeded();await expect(page.getByRole('button',{name:'保存',exact:true})).toBeInViewport();await page.getByRole('button',{name:'取消编辑',exact:true}).click();
   }
   await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);matrix.push({role,width,height,result:'passed'});
  }
 }
 writeFileSync(`${evidence}/matrix.json`,JSON.stringify(matrix,null,2)+'\n');console.log(JSON.stringify({event:'G4-layout',evidence,matrix}));
});

test('地点204 Chromium 原生缩放地址表单键盘',async()=>{
 test.setTimeout(180000);const evidence=`.local/g4-implementation/layout/${Date.now()}-native`;mkdirSync(evidence,{recursive:true});let n=3800+process.pid%1000;while(existsSync(`/tmp/.X11-unix/X${n}`)||existsSync(`/tmp/.X${n}-lock`))n++;
 const env={...process.env,DISPLAY:`:${n}`};const xvfb=spawn('Xvfb',[env.DISPLAY,'-screen','0','1920x1200x24','-nolisten','tcp','-nolisten','unix'],{env,stdio:'ignore'});let browser:Awaited<ReturnType<typeof chromium.launch>>|undefined;
 try{
  await expect.poll(()=>{try{execFileSync('xdpyinfo',[],{env,stdio:'ignore',timeout:1000});return true}catch{return false}}).toBe(true);
  browser=await chromium.launch({headless:false,env,args:['--window-size=1440,1039','--window-position=0,0','--ozone-platform=x11']});const connection=JSON.parse(execFileSync('python3',['scripts/local-bu.py','connection'],{encoding:'utf8'}));const actual=await browser.newContext({viewport:null,deviceScaleFactor:undefined,baseURL:connection.webOrigin});const page=await actual.newPage();page.setDefaultTimeout(15000);
  await open(page);const code=await create(page,'中国','海淀');const zoom=(steps:number)=>execFileSync('python3',['docs/verification/data/100-ui-baseline/native-zoom.py',String(steps)],{env,stdio:'pipe'});const dims=()=>page.evaluate(()=>({width:innerWidth,height:innerHeight,dpr:devicePixelRatio,outer:[outerWidth,outerHeight]}));const cdp=await actual.newCDPSession(page);const {windowId}=await cdp.send('Browser.getWindowForTarget');const initial=await dims();await cdp.send('Browser.setWindowBounds',{windowId,bounds:{width:initial.outer[0]+1280-initial.width,height:initial.outer[1]+900-initial.height}});const matrix=[];
  for(const [steps,dpr]of nativeZoomCases()){
   zoom(steps);await expect.poll(async()=>(await dims()).dpr).toBe(dpr);await menu(page);await page.getByRole('textbox',{name:'搜索编码或名称',exact:true}).fill(code);await page.getByRole('button',{name:'查询',exact:true}).click();const link=page.getByRole('link',{name:code+'办公地点',exact:true});await link.scrollIntoViewIfNeeded();await expect(link).toBeInViewport();await captureViewport(page,`${evidence}/${dpr}-list.png`,[page.locator('.app-account-name')]);await link.click();await expect(page).toHaveURL(/\/locations\/[0-9a-f-]+/);await expect(page.getByRole('region',{name:'地点详情正文',exact:true})).toHaveAttribute('data-page-scroll-ready','true');await openMoreActions(page);await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();const region=page.getByRole('combobox',{name:'行政区划',exact:true});await region.click();await expect(page.getByRole('option').filter({hasText:'海淀'})).toBeVisible();await captureViewport(page,`${evidence}/${dpr}-region.png`,[page.locator('.app-account-name')]);await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);await page.keyboard.press('Escape');const address=page.getByRole('textbox',{name:'详细地址',exact:true});await address.scrollIntoViewIfNeeded();await address.focus();await page.keyboard.press('Tab');await expect(page.getByRole('textbox',{name:'邮政编码',exact:true})).toBeFocused();await captureViewport(page,`${evidence}/${dpr}-fields.png`,[page.locator('.app-account-name')]);await page.getByRole('button',{name:'保存',exact:true}).scrollIntoViewIfNeeded();await expect(page.getByRole('button',{name:'保存',exact:true})).toBeInViewport();await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);matrix.push({...await dims(),result:'passed'});await page.getByRole('button',{name:'取消编辑',exact:true}).click();
  }
  writeFileSync(`${evidence}/matrix.json`,JSON.stringify(matrix,null,2)+'\n');console.log(JSON.stringify({event:'G4-native-layout',evidence,matrix}));zoom(0);
 }finally{await browser?.close();xvfb.kill('SIGTERM');await new Promise<void>(resolve=>{if(xvfb.exitCode!==null)resolve();else xvfb.once('exit',()=>resolve())})}
});
