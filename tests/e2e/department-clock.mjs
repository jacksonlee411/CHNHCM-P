// Ephemeral integration fixture only. Credentials enter on stdin and are never saved.
import {chromium,expect} from '@playwright/test';
import {readFileSync} from 'node:fs';
const input=JSON.parse(readFileSync(0,'utf8')),base=process.argv[2];
const browser=await chromium.launch();let step='start',diagnosticPage;const evidence=[];
try{
 for(const zone of ['Asia/Shanghai','America/Los_Angeles']){
  step=zone+' setup';await fetch(base+'/__fixture_clock?instant=2026-06-30T15%3A59%3A59Z',{method:'POST'});
  const context=await browser.newContext({baseURL:base,timezoneId:zone,viewport:{width:1440,height:900}}),page=await context.newPage();
  await page.goto('/');await page.getByLabel('登录名').fill(input.login);await page.getByLabel('密码').fill(input.password);await page.getByRole('button',{name:'登录',exact:true}).click();
  const enter=async p=>{diagnosticPage=p;step=zone+(p===page?' list menu':' detail menu');await p.getByRole('button',{name:'切换应用',exact:true}).click();await p.getByRole('menuitem',{name:'部门',exact:true}).click();await expect(p.getByRole('heading',{name:'部门',exact:true})).toBeVisible();await p.getByRole('textbox',{name:'搜索编码或名称',exact:true}).fill('R5_CLOCK_C');await p.getByRole('button',{name:'查询',exact:true}).click();await expect(p.getByRole('link',{name:'跨日转出部门',exact:true})).toBeVisible()};
  await enter(page);await page.getByRole('textbox',{name:'定位树中部门',exact:true}).fill('R5_CLOCK_C');await page.getByRole('listbox',{name:'部门定位结果'}).getByRole('option').click();await expect(page.getByRole('tree').locator(`[data-tree-id="${input.id}"]`)).toHaveAttribute('aria-selected','true');await page.getByRole('checkbox',{name:/全选查询结果/}).check();await expect(page.getByText('已选择1项',{exact:true})).toBeVisible();await expect(page.getByRole('button',{name:'导出所选',exact:true})).toBeEnabled();
  const detail=await context.newPage();await detail.goto('/');await enter(detail);await detail.getByRole('link',{name:'跨日转出部门',exact:true}).click();await detail.getByRole('button',{name:'2026-07-01',exact:true}).click();await expect(detail.getByRole('heading',{name:/七月转入部门/})).toBeVisible();await expect(detail.locator('.operation-summary')).toBeVisible();
  step=zone+' after midnight';await fetch(base+'/__fixture_clock?instant=2026-06-30T16%3A00%3A00Z',{method:'POST'});
  await page.getByRole('button',{name:'导出所选',exact:true}).click();await expect(page.getByRole('toolbar',{name:'所选部门',exact:true})).toHaveCount(0);await expect(page.getByRole('link',{name:'跨日转出部门',exact:true})).toHaveCount(0);await expect(page.getByRole('tree').locator(`[data-tree-id="${input.id}"]`)).toHaveCount(0);
  await detail.getByRole('tab',{name:'变更日志',exact:true}).click();await expect(detail.locator('#department-detail-error')).toBeVisible();await expect(detail.locator('.operation-summary')).toHaveCount(0);await expect(detail.locator('.department-path')).toHaveCount(0);await expect(detail.getByRole('heading',{name:/七月转入部门/})).toHaveCount(0);
  expect((await detail.request.get('/api/v1/departments/'+input.id+'?sourceVersion='+input.source)).status()).toBe(404);
  evidence.push({zone,before:'2026-06-30',after:'2026-07-01',futureRecordDoesNotMoveAuthority:true,oldSelectionAndProjectionCleared:true,oldSourceDenied:true,oldScopedTreeCleared:true});await context.close();
 }
 console.log(JSON.stringify({result:'passed',cases:evidence}));
} catch(error){if(diagnosticPage&&!diagnosticPage.isClosed())await diagnosticPage.screenshot({path:'../../.local/tree300-clock-failure.png',mask:[diagnosticPage.locator('.app-account-name'),diagnosticPage.getByLabel('登录名'),diagnosticPage.getByLabel('密码')]}).catch(()=>{});console.error(JSON.stringify({step,error:error.message.split('\n').slice(0,4).join(' ')}));process.exitCode=1}finally{await browser.close()}
