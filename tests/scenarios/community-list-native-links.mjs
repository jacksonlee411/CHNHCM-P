import { chromium, expect } from '@playwright/test';
import { spawn, execFileSync } from 'node:child_process';
import { mkdirSync, existsSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { login, query, out as fixtureOut } from '../e2e/community-list-support.mjs';

const out=path.resolve(process.argv[2]??'.local/111-community-t03/native-links');mkdirSync(out,{recursive:true});
const fixture=JSON.parse(readFileSync(fixtureOut+'/fixtures.json','utf8'));
const result={scope:'Linux Chromium native link context menu and clipboard, real authorized Go/API/DB',rows:[],stopped:false};
const save=()=>writeFileSync(path.join(out,'results.json'),JSON.stringify(result,null,2)+'\n');
let display,browser;
try{
  if(existsSync('/tmp/.X221-lock'))throw new Error('Task display occupied');
  display=spawn('Xvfb',[':221','-screen','0','1800x1200x24','-nolisten','tcp'],{stdio:'ignore'});
  for(let n=0;n<50&&!existsSync('/tmp/.X11-unix/X221');n++)await new Promise(r=>setTimeout(r,100));process.env.DISPLAY=':221';
  browser=await chromium.launch({headless:false,args:['--ozone-platform=x11','--window-position=0,0','--window-size=1440,1039']});
  for(const [label,width,height] of [['desktop',1440,900],['summary',390,844]]){
    const context=await browser.newContext({viewport:{width,height},permissions:['clipboard-read','clipboard-write']});
    const row={label,result:'not_run'};result.rows.push(row);save();
    try{
      const page=await context.newPage();await login(page);await query(page,fixture.items[0].code);
      await page.getByLabel('搜索编码或名称').fill('复制地址时保留的草稿');
      const link=page.getByRole('link',{name:fixture.items[0].name,exact:true}),href=await link.getAttribute('href');
      await page.locator('.app-account-name').evaluateAll(xs=>xs.forEach(e=>e.style.visibility='hidden'));
      await link.click({button:'right'});await new Promise(r=>setTimeout(r,150));
      execFileSync('python3',['docs/verification/data/100-ui-baseline/x11-shot.py',path.join(out,label+'-context-menu.png'),'1500','1100'],{stdio:'pipe'});
      execFileSync('python3',['tests/scenarios/native-browser-keys.py','Home','Down','Down','Down','Return'],{stdio:'pipe'});
      const copied=await page.evaluate(()=>navigator.clipboard.readText());expect(copied).toBe(new URL(href,page.url()).href);
      await expect(page.getByLabel('搜索编码或名称')).toHaveValue('复制地址时保留的草稿');
      const target=await context.newPage();await target.goto(copied);await expect(target.getByRole('heading',{level:1})).toContainText(fixture.items[0].code);await target.close();
      row.result='passed';row.checks=['native right-click Copy link address','actual clipboard equals authorized href','copied address opens real detail','original draft preserved'];save();
    }finally{await context.close();}
  }
  result.status='passed';
}catch(error){result.status='failed';result.error=String(error);process.exitCode=1;}
finally{if(browser)await browser.close();if(display){display.kill('SIGTERM');await new Promise(r=>display.once('exit',r));}result.stopped=true;save();}
console.log(JSON.stringify(result,null,2));
