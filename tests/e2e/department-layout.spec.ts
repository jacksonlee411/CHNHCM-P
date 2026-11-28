import { nativeZoomCases } from './desktop-viewports.mjs';
import {chromium} from '@playwright/test';
import {spawn,execFileSync} from 'node:child_process';
import {existsSync,mkdirSync,writeFileSync} from 'node:fs';
import {test,expect,openMoreActions,type Page} from './fixtures';
import {open} from './bu-support';
import {departmentKind as dep,menu,apiBU,apiDepartment} from './dated-object-support';
import {captureViewport} from '../../docs/verification/data/100-ui-capture-fix/capture-viewport.mjs';
const evidence = process.env.CHNHCM_EVIDENCE_DIR ?? '.local/department-layout-' + process.env.CHNHCM_LOCAL_PROFILE;
async function screens(page:Page,name:string,fixture:{url:string;code:string;name:string}){
 const detailURL=fixture.url;
 const entries=[];
 const capture=async(view:string)=>{await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);const dimensions=await page.evaluate(()=>({width:innerWidth,height:innerHeight,dpr:devicePixelRatio}));await captureViewport(page,`${evidence}/${name}-${view}.png`,[page.locator('.app-account-name')]);entries.push({name,view,...dimensions,result:'passed'});};
 await menu(page,dep);await page.getByPlaceholder('搜索编码或名称').fill(fixture.code);await page.getByRole('button',{name:'查询',exact:true}).click();await expect(page.getByRole('link',{name:fixture.name,exact:true})).toBeVisible();await expect(page.getByText('正在查询…',{exact:true})).toHaveCount(0);await page.getByRole('link',{name:fixture.name,exact:true}).scrollIntoViewIfNeeded();await expect(page.getByRole('link',{name:fixture.name,exact:true})).toBeInViewport();await capture('list');
 await openMoreActions(page);await page.getByRole('menuitem',{name:'列筛选…',exact:true}).click();await expect(page.getByRole('dialog',{name:'列筛选',exact:true})).toBeVisible();const filter=page.getByRole('dialog',{name:'列筛选',exact:true});await filter.getByRole('combobox',{name:'列',exact:true}).click();await page.getByRole('option',{name:'管理业务单元',exact:true}).click();await expect(filter.getByRole('combobox',{name:'管理业务单元',exact:true})).toBeInViewport();await capture('filter');await page.keyboard.press('Escape');await expect(page.getByRole('button',{name:'更多操作',exact:true})).toBeFocused();
 await page.getByRole('link',{name:'新建',exact:true}).click();await expect(page.getByRole('combobox',{name:'管理业务单元',exact:true})).toBeVisible();const picker=page.getByRole('combobox',{name:'管理业务单元',exact:true});await picker.focus();await picker.press('ArrowDown');await expect(page.getByRole('option').first()).toBeVisible();await picker.press('Escape');await expect(picker).toBeFocused();await picker.press('Tab');await expect(page.getByRole('combobox',{name:'行政上级',exact:true})).toBeFocused();await page.getByRole('combobox',{name:'行政上级',exact:true}).scrollIntoViewIfNeeded();await expect(page.getByRole('combobox',{name:'行政上级',exact:true})).toBeInViewport();await capture('create');await page.getByRole('link',{name:'取消',exact:true}).click();
 await page.goto(detailURL);await page.locator('.department-path').scrollIntoViewIfNeeded();await expect(page.locator('.department-path')).toBeInViewport();await capture('detail');
 await openMoreActions(page);await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();await expect(page.getByRole('combobox',{name:'行政上级',exact:true})).toBeVisible();await page.getByRole('button',{name:'保存',exact:true}).scrollIntoViewIfNeeded();await expect(page.getByRole('button',{name:'保存',exact:true})).toBeInViewport();await page.getByRole('combobox',{name:'行政上级',exact:true}).scrollIntoViewIfNeeded();await expect(page.getByRole('combobox',{name:'行政上级',exact:true})).toBeInViewport();await capture('correction');await page.getByRole('button',{name:'取消编辑',exact:true}).click();
 await openMoreActions(page);await page.getByRole('menuitem',{name:'插入停用与重新启用',exact:true}).click();await page.getByRole('dialog').getByRole('button',{name:'取消',exact:true}).scrollIntoViewIfNeeded();await capture('pair');await page.getByRole('dialog').getByRole('button',{name:'取消',exact:true}).click();
 await page.getByRole('tab',{name:'变更日志',exact:true}).click();await expect(page.getByRole('table',{name:'变更日志',exact:true})).toBeVisible();await page.getByRole('link',{name:'查看当次资料',exact:true}).first().scrollIntoViewIfNeeded();await expect(page.getByRole('link',{name:'查看当次资料',exact:true}).first()).toBeInViewport();await capture('log');
 return entries;
}
async function setup(page:Page){
 await open(page);const bu=await apiBU(page,'跨地区管理事业部','1900-01-01'),p=await apiDepartment(page,bu.id,{name:'华东地区行政管理与专业支持服务中心'.repeat(3)}),name='人力资源及组织发展服务部门'.repeat(4),c=await apiDepartment(page,bu.id,{parent:p.id,name});return {url:'/settings/organization/departments/'+c.id,code:c.code,name};
}
test('部门七类页面桌面键盘布局',async({page})=>{
 test.setTimeout(180000);page.setDefaultTimeout(15000);mkdirSync(evidence,{recursive:true});const url=await setup(page);
 await expect(page.locator('[data-navigation-mode]')).toHaveAttribute('data-navigation-mode','desktop');
 const matrix=await screens(page,'desktop',url);
 writeFileSync(`${evidence}/responsive.json`,JSON.stringify(matrix,null,2)+'\n');
});
test('部门七类页面Chromium 原生缩放',async()=>{
 test.setTimeout(180000);mkdirSync(evidence,{recursive:true});let number=1700+process.pid%1000;while(existsSync(`/tmp/.X11-unix/X${number}`)||existsSync(`/tmp/.X${number}-lock`))number++;
 const env={...process.env,DISPLAY:`:${number}`},xvfb=spawn('Xvfb',[env.DISPLAY,'-screen','0','1920x1200x24','-nolisten','tcp','-nolisten','unix'],{stdio:'ignore'});let browser:Awaited<ReturnType<typeof chromium.launch>>|undefined;
 try{
  await expect.poll(()=>{try{execFileSync('xdpyinfo',[],{env,stdio:'ignore',timeout:1000});return true}catch{return false}}).toBe(true);
  browser=await chromium.launch({headless:false,env,args:['--window-size=1440,1039','--window-position=0,0','--ozone-platform=x11']});const connection=JSON.parse(execFileSync('python3',['scripts/local-bu.py','connection'],{encoding:'utf8'}));
  const actual=await browser.newContext({viewport:null,deviceScaleFactor:undefined,baseURL:connection.webOrigin}),view=await actual.newPage();view.setDefaultTimeout(15000);const url=await setup(view);
  const cdp=await actual.newCDPSession(view),{windowId}=await cdp.send('Browser.getWindowForTarget');const dimensions=()=>view.evaluate(()=>({width:innerWidth,height:innerHeight,dpr:devicePixelRatio,outer:[outerWidth,outerHeight]}));
  const zoom=(steps:number)=>execFileSync('python3',['docs/verification/data/100-ui-baseline/native-zoom.py',String(steps)],{env,stdio:'pipe'});zoom(0);const initial=await dimensions();await cdp.send('Browser.setWindowBounds',{windowId,bounds:{width:initial.outer[0]+1280-initial.width,height:initial.outer[1]+900-initial.height}});const matrix=[];
  for(const [steps,dpr] of nativeZoomCases){zoom(steps);await expect.poll(async()=>(await dimensions()).dpr).toBe(dpr);matrix.push(...await screens(view,`zoom-${dpr*100}`,url));}
  writeFileSync(`${evidence}/native-zoom.json`,JSON.stringify(matrix,null,2)+'\n');zoom(0);
 }finally{xvfb.kill('SIGTERM');await browser?.close();await new Promise<void>(resolve=>{if(xvfb.exitCode!==null)resolve();else xvfb.once('exit',()=>resolve())})}
});
