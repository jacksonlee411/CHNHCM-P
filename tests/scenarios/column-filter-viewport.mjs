import { chromium, expect } from '@playwright/test';
import { spawn, execFileSync } from 'node:child_process';
import { readFileSync, writeFileSync, existsSync, mkdtempSync, chmodSync, rmSync, mkdirSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import assert from 'node:assert/strict';
import { login, query, panel, option, out as fixtureOut, connection } from '../e2e/community-list-support.mjs';
import { captureViewport } from '../../docs/verification/data/100-ui-capture-fix/capture-viewport.mjs';

const out = path.resolve(process.argv[2] ?? '.local/111-column-filter-implementation/matrix'); mkdirSync(out, { recursive: true });
const fixture = JSON.parse(readFileSync(fixtureOut + '/fixtures.json', 'utf8'));
const conditions = [['desktop',1440,900,0],['short',1440,540,0],['narrow',390,844,0],['width320',320,900,0],['stress',320,256,0],['zoom200',1280,900,5],['zoom400',1280,900,8]];
const result = { base: execFileSync('git',['rev-parse','HEAD'],{encoding:'utf8'}).trim(), profile: process.env.CHNHCM_LOCAL_PROFILE, origin: connection.webOrigin, rows: [], errors: [], licenseMessages: [], stopped: false };
const save = () => writeFileSync(path.join(out,'results.json'),JSON.stringify(result,null,2)+'\n');
const pause = ms => new Promise(r => setTimeout(r,ms));
const dimensions = page => page.evaluate(()=>({width:innerWidth,height:innerHeight,outer:[outerWidth,outerHeight],dpr:devicePixelRatio,visualScale:visualViewport.scale}));
const zoom = n => execFileSync('python3',['docs/verification/data/100-ui-baseline/native-zoom.py',String(n)],{stdio:'pipe'});
let display, context, browser, profile;
async function close(){if(context)await context.close();if(browser?.isConnected())await browser.close();if(profile)rmSync(profile,{recursive:true,force:true});context=browser=profile=undefined;}
async function fullyVisible(locator){await expect.poll(()=>locator.evaluate(e=>{const r=e.getBoundingClientRect();return r.top>=-1&&r.bottom<=innerHeight+1&&r.left>=-1&&r.right<=innerWidth+1;})).toBe(true);}
async function reach(page,locator){
  for(let n=0;n<130;n++){
    if(await locator.evaluate(e=>e===document.activeElement)){await expect(locator).toBeInViewport();return;}
    await page.keyboard.press('Tab');
  }
  throw new Error('Keyboard target unreachable: '+locator.toString());
}
try {
  const displayName=':219';assert.ok(!existsSync('/tmp/.X219-lock'));
  display=spawn('Xvfb',[displayName,'-screen','0','2560x1600x24','-nolisten','tcp'],{stdio:'ignore'});
  for(let n=0;n<50&&!existsSync('/tmp/.X11-unix/X219');n++)await pause(100);process.env.DISPLAY=displayName;
  for(const [label,width,height,steps] of conditions){
    const row={label,result:'not_run',checks:[],captures:[]};result.rows.push(row);save();
    let page;
    try {
      if(steps){
        profile=mkdtempSync(path.join(tmpdir(),'chnhcm-column-filter-zoom-'));chmodSync(profile,0o700);
        context=await chromium.launchPersistentContext(profile,{headless:false,viewport:null,locale:'zh-CN',timezoneId:'Asia/Shanghai',args:['--window-size=1280,1039','--window-position=0,0','--ozone-platform=x11']});browser=context.browser();page=context.pages()[0];
        await page.goto(connection.webOrigin);zoom(0);await pause(150);
        const cdp=await context.newCDPSession(page),{windowId}=await cdp.send('Browser.getWindowForTarget');
        const d=await dimensions(page);
        await cdp.send('Browser.setWindowBounds',{windowId,bounds:{width:1280,height:1039+900-d.height,left:0,top:0}});await pause(150);
        row.baseline=await dimensions(page);assert.equal(row.baseline.width,1280);assert.equal(row.baseline.height,900);
        zoom(steps);await pause(200);row.dimensions=await dimensions(page);
        assert.equal(row.dimensions.width,steps===5?640:320);assert.equal(row.dimensions.height,steps===5?450:225);assert.deepEqual(row.dimensions.outer,row.baseline.outer);assert.equal(row.dimensions.visualScale,1);
        // Native browser menu shows the actual percentage before any credentials are entered.
        await page.keyboard.press('Alt+f');await pause(150);
        execFileSync('python3',['docs/verification/data/100-ui-baseline/x11-shot.py',path.join(out,label+'-browser-menu.png'),'1400','1100'],{stdio:'pipe'});
        await page.keyboard.press('Escape');await cdp.detach();
      }else{
        browser=await chromium.launch({headless:true});context=await browser.newContext({viewport:{width,height},locale:'zh-CN',timezoneId:'Asia/Shanghai'});page=await context.newPage();
        row.dimensions=await dimensions(page);
      }
      page.setDefaultTimeout(12000);page.on('pageerror',e=>result.errors.push(e.message));page.on('console',m=>{if(/(?:MUI X:.*licen[cs]e|Missing licen[cs]e|Invalid licen[cs]e|licen[cs]e key)/i.test(m.text()))result.licenseMessages.push(m.text());});
      row.browser=browser.version();page.on('pageerror',e=>result.errors.push(e.message));page.on('console',m=>{if(/(?:MUI X:.*licen[cs]e|Missing licen[cs]e|Invalid licen[cs]e|licen[cs]e key)/i.test(m.text()))result.licenseMessages.push(m.text());});
      await login(page);await query(page,fixture.prefix);
      await panel(page,'列筛选…');
      const dialog=page.getByRole('dialog'),value=dialog.getByLabel('值',{exact:true});
      await value.fill(fixture.items[72].code);
      const apply=page.getByRole('button',{name:'应用',exact:true});await reach(page,apply);await fullyVisible(apply);
      assert.ok(await dialog.evaluate(e=>e.scrollWidth<=e.clientWidth+1));assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
      const file=label+'-filter.png';row.captures.push({file,...await captureViewport(page,path.join(out,file),[page.locator('.app-account-name'),page.getByLabel('登录名'),page.getByLabel('密码')])});
      await page.keyboard.press('Enter');await expect(page.getByText('1–1 / 1',{exact:true})).toBeVisible();
      assert.equal(new URL(page.url()).searchParams.get('filterField'),'code');
      const url=page.url(),input=page.getByLabel('搜索编码或名称');await input.fill('独立搜索草稿');
      await panel(page,'列筛选…');await value.fill('取消列条件草稿');await page.keyboard.press('Escape');await expect(page).toHaveURL(url);await expect(input).toHaveValue('独立搜索草稿');await expect(page.getByRole('button',{name:'更多操作',exact:true})).toBeFocused();
      await input.fill(fixture.prefix);await panel(page,'列筛选…');const clear=page.getByRole('button',{name:'清除条件',exact:true});await reach(page,clear);await page.keyboard.press('Enter');await reach(page,apply);await page.keyboard.press('Enter');await expect(page.getByText('1–20 / 73',{exact:true})).toBeVisible();
      assert.equal(new URL(page.url()).searchParams.has('filterField'),false);
      row.checks.push('normal-menu real API/DB','one column condition outside initial page','keyboard Apply/clear/Escape reachable and visible','independent search/column drafts','no horizontal dialog/document overflow','native zoom or stated viewport');row.result='passed';save();
    }catch(error){row.result='failed';row.error=String(error);if(page)try{await captureViewport(page,path.join(out,label+'-failure.png'),[page.locator('.app-account-name'),page.getByLabel('登录名'),page.getByLabel('密码')]);}catch{}save();process.exitCode=1;break;}
    finally{if(steps&&context)try{zoom(0);row.restored100=await dimensions(context.pages()[0]);}catch{}await close();save();}
  }
}finally{await close();if(display){display.kill('SIGTERM');await new Promise(r=>display.once('exit',r));}result.stopped=true;result.status=result.rows.length===7&&result.rows.every(x=>x.result==='passed')&&!result.errors.length&&!result.licenseMessages.length?'passed':'failed';save();}
console.log(JSON.stringify({status:result.status,rows:result.rows.map(({label,result,error,dimensions})=>({label,result,error,dimensions})),stopped:result.stopped},null,2));
