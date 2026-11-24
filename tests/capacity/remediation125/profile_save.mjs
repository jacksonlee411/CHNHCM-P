import {chromium,expect} from '@playwright/test';
import {writeFile} from 'node:fs/promises';
let raw='';for await(const chunk of process.stdin)raw+=chunk;
const input=JSON.parse(raw);
if(new URL(input.url).hostname!=='127.0.0.1')throw Error('owned loopback required');
const browser=await chromium.launch({headless:true});
const result={passed:false,steps:[],correctRequests:[],network:[]};
let page,before,changed,restored;
try{
  const context=await browser.newContext({baseURL:input.url,viewport:{width:1440,height:900}});page=await context.newPage();
  page.on('response',async response=>{
    const request=response.request(),url=new URL(response.url());
    if(url.pathname.startsWith('/api/'))result.network.push({method:request.method(),route:url.pathname.replace(/[0-9a-f]{8}-[0-9a-f-]{27}/gi,':id'),status:response.status(),hasQuery:url.searchParams.has('q'),matchesExpectedQuery:url.searchParams.get('q')===input.code});
    if(request.method()==='POST' && /\/records\/[^/]+\/correct$/.test(new URL(response.url()).pathname)){
      const value=await response.json();
      if(response.status()===200 && value.outcome==='accepted'){
        result.correctRequests.push({status:response.status()});
        if(!changed)changed=value;else restored=value;
      }
    }
  });
  const step=async(label,fn)=>{result.currentStep=label;const start=performance.now();await fn();result.steps.push({label,ms:performance.now()-start});};
  const read=()=>page.evaluate(async id=>{const r=await fetch('/api/v1/departments/'+id);if(!r.ok)throw Error('detail '+r.status);return r.json();},input.id);
  await step('正常菜单及目标查询',async()=>{
    await page.goto('/');await page.getByLabel('登录名').fill('admin');await page.getByLabel('密码').fill('admin1234567');await page.getByRole('button',{name:'登录',exact:true}).click();
    await page.getByRole('button',{name:'切换应用',exact:true}).click();await page.getByRole('menuitem',{name:'设置',exact:true}).click();
    await page.getByRole('button',{name:'切换应用',exact:true}).click();
    const initial=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1/departments');
    await page.getByRole('menuitem',{name:'部门',exact:true}).click();expect((await initial).status()).toBe(200);
    await expect(page.getByRole('grid')).toBeVisible();
    await page.getByRole('textbox',{name:'搜索编码或名称',exact:true}).fill(input.code);
    const request=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1/departments' && new URL(r.url()).searchParams.get('q')===input.code);
    await expect(page.getByRole('textbox',{name:'搜索编码或名称',exact:true})).toHaveValue(input.code);
    await page.getByRole('button',{name:'查询',exact:true}).click();const listing=await(await request).json();expect(listing.count).toBe(1);
    await page.getByRole('link',{name:listing.items[0].name,exact:true}).click();await expect(page.locator('.effective-dates')).toBeVisible();before=await read();
  });
  const correct=async value=>{
    const trigger=page.getByRole('button',{name:'更多操作',exact:true});await expect(trigger).toBeEnabled();await trigger.click();
    await expect(page.getByRole('button',{name:'更多操作',exact:true,includeHidden:true})).toHaveAttribute('aria-expanded','true');
    await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();await expect(page.getByRole('textbox',{name:'简称',exact:true})).toBeVisible();
    await page.getByRole('textbox',{name:'简称',exact:true}).fill(value);
    await expect(page.getByRole('button',{name:'保存',exact:true})).toBeEnabled();await page.getByRole('button',{name:'保存',exact:true}).click();
    await expect(page.locator('.object-form')).toHaveCount(0);
  };
  await step('更正简称并实际保存',async()=>{
    await correct('125浏览器交接');const actual=await read();expect(actual.selectedRecord.shortName).toBe('125浏览器交接');expect(actual.selectedRecord.name).toBe(before.selectedRecord.name);expect(actual.selectedRecord.recordId).toBe(before.selectedRecord.recordId);
  });
  await step('通过同一表单恢复原值',async()=>{
    await correct(before.selectedRecord.shortName??'');const actual=await read();expect(actual.selectedRecord.shortName??'').toBe(before.selectedRecord.shortName??'');expect(actual.timeline.map(r=>[r.recordId,r.effectiveDate,r.endExclusive])).toEqual(before.timeline.map(r=>[r.recordId,r.effectiveDate,r.endExclusive]));
  });
  await expect.poll(()=>Boolean(changed&&restored)).toBe(true);
  await step('当前授权下查回原保存结果',async()=>{
    const value=await page.evaluate(async key=>{const session=await(await fetch('/api/v1/session')).json();const r=await fetch('/api/v1/departments/submissions/'+key+'/resolve',{method:'POST',headers:{'X-CSRF-Token':session.csrf}});if(!r.ok)throw Error('resolve '+r.status);return r.json();},changed.submissionId);
    expect(value.sourceVersion).toBe(changed.sourceVersion);expect(value.currentChanged).toBe(true);
  });
  await writeFile(input.receipt,JSON.stringify({changed,restored}),{mode:0o600,flag:'wx'});
  await page.screenshot({path:input.screenshot,mask:[page.locator('.app-account-name')]});result.passed=true;
}catch(error){
  result.errorType=error.name;
  if(input.diagnostics)await writeFile(input.diagnostics,JSON.stringify(result),{mode:0o600,flag:'wx'});
  throw error;
}finally{
  if(changed&&!restored&&page){
    // An incomplete browser journey stays failed. Restore only this task's
    // optional short name through the real API and retain both history sources.
    await page.evaluate(async({id,shortName})=>{
      const session=await(await fetch('/api/v1/session')).json();const d=await(await fetch('/api/v1/departments/'+id)).json();const r=d.selectedRecord;
      const response=await fetch('/api/v1/departments/'+id+'/records/'+r.recordId+'/correct',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':session.csrf,'Idempotency-Key':crypto.randomUUID()},body:JSON.stringify({expectedSourceVersion:d.headSourceVersion,expectedRecordRevision:r.revision,changes:{shortName}})});
      if(!response.ok)throw Error('browser cleanup failed '+response.status);
    },{id:input.id,shortName:before.selectedRecord.shortName??null});
  }
  await browser.close();
}
process.stdout.write(JSON.stringify(result)+'\n');
