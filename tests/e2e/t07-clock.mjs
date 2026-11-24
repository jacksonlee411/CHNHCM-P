// Credentials are received only on stdin; host and browser clocks are unchanged.
import {chromium,expect} from '@playwright/test';
import {readFileSync} from 'node:fs';
const credentials=JSON.parse(readFileSync(0,'utf8'));
const base=process.argv[2];
const browser=await chromium.launch(process.env.CHNHCM_BROWSER_EXECUTABLE?{executablePath:process.env.CHNHCM_BROWSER_EXECUTABLE}:{});
let step='start';const evidence=[];
try{
 for(const zone of ['Asia/Shanghai','America/Los_Angeles']){
  for(const [instant,day] of [['2026-06-14T15:59:59Z','2026-06-14'],['2026-06-14T16:00:00Z','2026-06-15']]){
   step=zone+' '+instant;
   await fetch(base+'/__fixture_clock?instant='+encodeURIComponent(instant),{method:'POST'});
   const context=await browser.newContext({timezoneId:zone,viewport:{width:1440,height:900}});const page=await context.newPage();
   await page.goto(base);await page.getByLabel('登录名').fill(credentials.login);await page.getByLabel('密码').fill(credentials.password);await page.getByRole('button',{name:'登录',exact:true}).click();
   await page.getByRole('button',{name:'切换应用',exact:true}).click();await page.getByRole('menuitem',{name:'设置',exact:true}).click();await page.getByRole('button',{name:'组织配置',exact:true}).click();await page.getByRole('menuitem',{name:'业务单元',exact:true}).click();
   await expect(page.getByLabel('查看日期')).toHaveCount(0);expect(await page.evaluate(async()=>(await(await fetch('/api/v1/business-units')).json()).businessAt)).toBe(day);
   await page.getByRole('link',{name:'新建',exact:true}).click();await expect(page.getByLabel('生效日期').and(page.locator('input'))).toHaveValue(day);
   evidence.push({zone,instant,queryDay:day,createDefault:day});await context.close();
  }
 }
 step='TL031 future lifecycle';
 await fetch(base+'/__fixture_clock?instant=2026-06-30T15%3A59%3A59Z',{method:'POST'});
 const context=await browser.newContext({timezoneId:'America/Los_Angeles',viewport:{width:1440,height:900}});const page=await context.newPage();
 await page.goto(base);await page.getByLabel('登录名').fill(credentials.login);await page.getByLabel('密码').fill(credentials.password);await page.getByRole('button',{name:'登录',exact:true}).click();
 const menu=async()=>{await page.getByRole('button',{name:'切换应用',exact:true}).click();await page.getByRole('menuitem',{name:'设置',exact:true}).click();await page.getByRole('button',{name:'组织配置',exact:true}).click();await page.getByRole('menuitem',{name:'业务单元',exact:true}).click()};
 await menu();await page.getByRole('link',{name:'新建',exact:true}).click();await page.getByRole('textbox',{name:'编码',exact:true}).fill('T07_NATURAL_MENU');await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('自然到日');await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-01-01');await page.getByRole('combobox',{name:'操作原因',exact:true}).click();await page.getByRole('option',{name:'新设组织',exact:true}).click();await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);
 for(const [action,date,reason,save] of [['停用业务单元','2026-07-01','业务终止','确认停用'],['重新启用业务单元','2026-08-01','业务恢复','确认重新启用']]){
  await page.getByRole('button',{name:'更多操作',exact:true}).click();await page.getByRole('menuitem',{name:action,exact:true}).click();await page.getByLabel('生效日期').and(page.locator('input')).fill(date);await page.getByRole('combobox',{name:'操作原因',exact:true}).click();await page.getByRole('option',{name:reason,exact:true}).click();await expect(page.getByRole('button', {name:'预览',exact:true})).toHaveCount(0);await page.getByRole('button',{name:save,exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);
 }
 const id=new URL(page.url()).pathname.split('/').at(-1);
 const read=()=>page.evaluate(async id=>(await fetch('/api/v1/business-units/'+id)).json(),id);const before=await read();
 for(const [instant,day,state,label] of [['2026-06-30T15:59:59Z','2026-06-30','active','启用'],['2026-06-30T16:00:00Z','2026-07-01','inactive','停用'],['2026-07-31T16:00:00Z','2026-08-01','active','启用']]){
  await fetch(base+'/__fixture_clock?instant='+encodeURIComponent(instant),{method:'POST'});await page.goto(base);await menu();await expect(page.getByLabel('查看日期')).toHaveCount(0);expect(await page.evaluate(async()=>(await(await fetch('/api/v1/business-units')).json()).businessAt)).toBe(day);await page.getByRole('link',{name:'自然到日',exact:true}).click();await expect(page.locator('.object-detail-field').filter({has:page.getByText('状态',{exact:true})})).toContainText(label);const d=await read();expect(d.status).toBe(state);expect(d.headSourceVersion).toBe(before.headSourceVersion);expect(d.timeline).toEqual(before.timeline);evidence.push({case:'TL031',instant,day,state,sourceVersion:d.headSourceVersion});
 }
 step='TL391 midnight selection';
 await fetch(base+'/__fixture_clock?instant=2026-06-30T15%3A59%3A59Z',{method:'POST'});
 await page.getByRole('link',{name:'返回列表',exact:true}).click();
 const row=page.getByRole('row').filter({hasText:'T07_NATURAL_MENU'});
 await expect(row).toContainText('启用');
 await page.getByRole('checkbox',{name:'全选查询结果',exact:true}).check();
 await expect(page.getByRole('button',{name:'导出所选',exact:true})).toBeEnabled();
 let downloads=0;page.on('download',()=>{downloads++});
 await fetch(base+'/__fixture_clock?instant=2026-06-30T16%3A00%3A00Z',{method:'POST'});
 const rejected=page.waitForResponse(r=>r.url().endsWith('/business-units/export'));
 await page.getByRole('button',{name:'导出所选',exact:true}).click();
 const response=await rejected;expect(response.status()).toBe(409);expect((await response.json()).code).toBe('M02_SELECTION_CHANGED');
 await expect(page.getByRole('checkbox',{name:'全选查询结果',exact:true})).not.toBeChecked();
 await expect(row).toContainText('停用');expect(downloads).toBe(0);
 await page.getByRole('checkbox',{name:'全选查询结果',exact:true}).check();
 await expect(page.getByRole('button',{name:'导出所选',exact:true})).toBeEnabled();
 const downloading=page.waitForEvent('download');await page.getByRole('button',{name:'导出所选',exact:true}).click();
 const download=await downloading;expect(download.suggestedFilename()).toBe('chnhcm-business-units-2026-07-01.csv');
 const stream=await download.createReadStream();const chunks=[];for await(const chunk of stream)chunks.push(chunk);
 expect(Buffer.concat(chunks).toString('utf8')).toContain('T07_NATURAL_MENU,自然到日,停用,2026-07-01');
 evidence.push({case:'TL391',oldDay:'2026-06-30',newDay:'2026-07-01',oldSelectionStatus:409,staleDownloads:0,reselectedFilename:download.suggestedFilename()});
 await context.close();
 console.log(JSON.stringify({result:'passed',normalMenu:true,clock:'injected Go Clock',cases:evidence}));
}catch{console.error('T07 business-day browser failed at '+step);process.exitCode=1}finally{await browser.close()}
