// Controlled-clock supplementary browser lane. Credentials arrive through stdin
// and are never printed, persisted, or passed as process arguments.
import { chromium, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';
const credentials=JSON.parse(readFileSync(0,'utf8'));
const browser=await chromium.launch(process.env.CHNHCM_BROWSER_EXECUTABLE?{executablePath:process.env.CHNHCM_BROWSER_EXECUTABLE}:{});
const page=await browser.newPage({viewport:{width:1440,height:900}});
let step='login';
try{
 await page.goto(process.argv[2]);await page.getByLabel('登录名').fill(credentials.login);await page.getByLabel('密码').fill(credentials.password);await page.getByRole('button',{name:'登录',exact:true}).click();
 step='normal-menu';await page.getByRole('button',{name:'切换应用',exact:true}).click();await page.getByRole('menuitem',{name:'设置',exact:true}).click();await page.getByRole('button',{name:'组织配置',exact:true}).click();await page.getByRole('menuitem',{name:'业务单元',exact:true}).click();
 step='create';await page.getByRole('link',{name:'新建',exact:true}).click();await page.getByRole('textbox',{name:'编码',exact:true}).fill('JOINT_MIDNIGHT_BROWSER');await page.getByRole('textbox',{name:'中文名称',exact:true}).fill('甲');await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-01-01');await page.getByRole('combobox',{name:'操作原因',exact:true}).click();await page.getByRole('option',{name:'新设组织',exact:true}).click();await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);
 step='pair';await page.getByRole('button',{name:'更多操作',exact:true}).click();await page.getByRole('menuitem',{name:'插入停用与重新启用',exact:true}).click();const dialog=page.getByRole('dialog',{name:'插入停用与重新启用',exact:true});await dialog.getByLabel('停用日期',{exact:true}).fill('2026-06-15');await dialog.getByLabel('重新启用日期',{exact:true}).fill('2026-06-16');await dialog.getByRole('combobox',{name:'停用原因',exact:true}).click();await page.getByRole('option',{name:'业务终止',exact:true}).click();await expect(dialog.locator('#pair-reactivationReason')).toHaveText('业务恢复');
 step='save-and-recover';const response=page.waitForResponse(r=>r.url().endsWith('/lifecycle-pairs'));await dialog.getByRole('button',{name:'保存',exact:true}).click();const saved=await(await response).json();await expect(dialog).toHaveCount(0);
 const recovered=await page.evaluate(async key=>{const s=await(await fetch('/api/v1/session')).json();return(await fetch('/api/v1/organization/submissions/'+key+'/resolve',{method:'POST',headers:{'X-CSRF-Token':s.csrf}})).json()},saved.submissionId);
 expect(saved.evaluatedBusinessDate).toBe('2026-06-14');expect(saved.recordedAt).toBe('2026-06-14T16:00:01Z');expect(recovered.items).toHaveLength(2);expect(recovered.items.every(i=>i.dateDescription==='未来安排')).toBe(true);expect(recovered.sourceVersion).toBe(saved.sourceVersion);
 console.log(JSON.stringify({result:'passed',normalMenu:true,fixedPair:true,clock:'injected Go Clock',evaluatedBusinessDate:saved.evaluatedBusinessDate,recordedAt:saved.recordedAt,dateDescription:recovered.items[0].dateDescription,submissionId:saved.submissionId,sourceVersion:saved.sourceVersion}));
}catch{console.error('Controlled midnight browser failed at '+step);process.exitCode=1}
finally{await browser.close()}
