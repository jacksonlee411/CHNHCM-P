import { chromium, expect } from '@playwright/test';
import { spawn, execFileSync } from 'node:child_process';
import { readFileSync, writeFileSync, existsSync, mkdtempSync, chmodSync, rmSync, mkdirSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import assert from 'node:assert/strict';
import { login, query, panel, option, out as fixtureOut, connection } from '../e2e/community-list-support.mjs';
import { captureViewport } from '../../docs/verification/data/100-ui-capture-fix/capture-viewport.mjs';

const out = path.resolve(process.argv[2] ?? '.local/111-community-t03/matrix'); mkdirSync(out, { recursive: true });
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
        profile=mkdtempSync(path.join(tmpdir(),'chnhcm-community-zoom-'));chmodSync(profile,0o700);
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
      row.browser=browser.version();await login(page);await query(page,fixture.prefix);
      const capture=async name=>{await pause(150);const file=label+'-'+name+'.png';const metadata=await captureViewport(page,path.join(out,file),[page.locator('.app-account-name'),page.getByLabel('登录名'),page.getByLabel('密码')]);row.captures.push({file,...metadata});save();};
      assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
      if(row.dimensions.width>=900){
        await expect(page.getByRole('columnheader').filter({hasNot:page.getByRole('checkbox')})).toHaveCount(4);
        const heights=await page.locator('.MuiDataGrid-row').evaluateAll(rows=>rows.slice(0,8).map(r=>r.getBoundingClientRect().height));assert.ok(heights.every(h=>h===39));row.desktopRowHeights=heights;
      }else{await expect(page.getByRole('columnheader').filter({hasNot:page.getByRole('checkbox')})).toHaveCount(0);await expect(page.getByRole('checkbox',{name:/全选查询结果/})).toBeVisible();await expect(page.locator('.list-row-summary').first()).toBeVisible();await panel(page,'排序：编码升序');await option(page,'排序方式','名称降序');await page.getByRole('button',{name:'应用',exact:true}).click();await expect(page).toHaveURL(u=>u.searchParams.get('sort')==='name_desc');}
      const input=page.getByLabel('搜索编码或名称');await reach(page,input);await input.fill(fixture.prefix);
      const next=page.getByRole('button',{name:'下一页',exact:true});await reach(page,next);await page.keyboard.press('Enter');await expect(page.getByText('21–40 / 73',{exact:true})).toBeVisible();
      await capture('list');
      const before=await page.evaluate(()=>({document:scrollY,grid:document.querySelector('.MuiDataGrid-virtualScroller')?.scrollTop??0}));
      await page.mouse.move(Math.floor(row.dimensions.width/2),row.dimensions.height-35);await page.mouse.wheel(0,650);await pause(200);
      const after=await page.evaluate(()=>({document:scrollY,grid:document.querySelector('.MuiDataGrid-virtualScroller')?.scrollTop??0}));
      assert.ok(after.document>before.document||after.grid>before.grid);row.wheel={before,after};
      const visible=await page.locator('a[data-object-link]').evaluateAll(nodes=>nodes.flatMap(e=>{const r=e.getBoundingClientRect();return r.top>=0&&r.bottom<=innerHeight?[{name:e.textContent,id:e.closest('[data-id]').getAttribute('data-id')}]:[]}));
      assert.ok(visible.length);const target=visible.at(-1),link=page.getByRole('link',{name:target.name,exact:true});
      await link.click();await expect(page.getByRole('heading',{level:1})).toContainText(fixture.items.find(x=>x.id===target.id).code);
      const basic=page.getByRole('tab',{name:'基本资料',exact:true});await reach(page,basic);await page.keyboard.press('ArrowRight');await page.keyboard.press('Enter');await expect(page.getByRole('tab',{name:'变更日志',exact:true})).toHaveAttribute('aria-selected','true');
      const source=page.getByRole('link',{name:'查看当次资料',exact:true}).first();await reach(page,source);await page.keyboard.press('Enter');await expect(page.getByText('正在查看只读历史资料',{exact:true})).toBeVisible();
      const current=page.getByRole('button',{name:'返回当前资料',exact:true});await reach(page,current);await page.keyboard.press('Enter');await expect(page.getByText('正在查看只读历史资料',{exact:true})).toHaveCount(0);
      const back=page.getByRole('link',{name:'返回列表',exact:true});await reach(page,back);await page.keyboard.press('Enter');await expect(link).toBeFocused();await fullyVisible(link);
      const size=page.getByRole('combobox',{name:'每页条数',exact:true});await reach(page,size);await page.keyboard.press('Enter');await page.getByRole('option',{name:'50',exact:true}).press('Enter');await expect(page.getByText('1–50 / 73',{exact:true})).toBeVisible();
      const create=page.getByRole('link',{name:'新建',exact:true});await reach(page,create);await page.keyboard.press('Enter');
      for(const name of ['编码','中文名称','英文名称']){const field=page.getByRole('textbox',{name,exact:true});await reach(page,field);await expect(field).toBeInViewport();}
      const date=page.getByLabel('生效日期').and(page.locator('input'));await reach(page,date);
      const reason=page.getByRole('combobox',{name:'操作原因',exact:true});await reach(page,reason);await page.keyboard.press('Enter');await page.getByRole('option',{name:'新设组织',exact:true}).press('Enter');
      const submit=page.getByRole('button',{name:'保存',exact:true});await reach(page,submit);await page.keyboard.press('Enter');await expect(page.getByRole('textbox',{name:'编码',exact:true})).toBeFocused();assert.equal(await page.getByRole('textbox',{name:'编码',exact:true}).evaluate(e=>e.validity.valueMissing),true);
      await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('真实修改用于保护');
      const cancel=page.getByRole('link',{name:'取消',exact:true});await reach(page,cancel);await page.keyboard.press('Enter');
      await expect(page.getByRole('button',{name:'继续编辑',exact:true})).toBeVisible();await page.getByRole('button',{name:'继续编辑',exact:true}).press('Enter');await expect(page.getByRole('textbox',{name:'中文名称',exact:true})).toHaveValue('真实修改用于保护');
      await reach(page,cancel);await page.keyboard.press('Enter');await page.getByRole('button',{name:'放弃并离开',exact:false}).press('Enter');
      await expect(page.getByRole('grid',{name:'业务单元列表'})).toBeVisible();await expect(link).toBeFocused();await fullyVisible(link);await capture('returned');
      if(!steps){
        await input.fill('断点保持草稿');await page.setViewportSize({width:899,height:900});await expect(input).toHaveValue('断点保持草稿');await page.setViewportSize({width:900,height:900});await expect(input).toHaveValue('断点保持草稿');
      }
      row.checks.push('normal-menu login/list','server pagination and reachable page size','actual wheel movement','detail/log/source-history/current return','same-object virtual link focus','keyboard new form fields/error/unsaved protection','no horizontal document overflow','native browser zoom or specified CSS viewport');row.result='passed';save();
    }catch(error){row.result='failed';row.error=String(error);if(page)try{await captureViewport(page,path.join(out,label+'-failure.png'),[page.locator('.app-account-name'),page.getByLabel('登录名'),page.getByLabel('密码')]);}catch{}save();process.exitCode=1;break;}
    finally{if(steps&&context)try{zoom(0);row.restored100=await dimensions(context.pages()[0]);}catch{}await close();save();}
  }
}finally{await close();if(display){display.kill('SIGTERM');await new Promise(r=>display.once('exit',r));}result.stopped=true;result.status=result.rows.length===7&&result.rows.every(x=>x.result==='passed')&&!result.errors.length&&!result.licenseMessages.length?'passed':'failed';save();}
console.log(JSON.stringify({status:result.status,rows:result.rows.map(({label,result,error,dimensions})=>({label,result,error,dimensions})),stopped:result.stopped},null,2));
