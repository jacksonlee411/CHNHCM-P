import {chromium, expect} from '@playwright/test';

// The Python owner starts the isolated API and serves the matching built dist.
// No trace, storage state, response body, query values or customer labels leave
// the process. Screenshots, when requested, stay in the private task directory.
let raw=''; for await (const chunk of process.stdin) raw+=chunk;
const input=JSON.parse(raw), origin=new URL(input.url);
if(origin.hostname!=='127.0.0.1' || origin.protocol!=='http:') throw Error('loopback owner required');
const browser=await chromium.launch({headless:true});
const result={role:input.role,segment:input.segment,network:[],steps:[],passed:false};
try {
  const context=await browser.newContext({baseURL:input.url,viewport:{width:1440,height:900}});
  const page=await context.newPage(),pending=new Map();
  // Login routes '/' to the BU list. Capture that first request before login;
  // returning through the BU menu can legitimately reuse the same mounted list.
  const listing=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1/'+input.segment && r.request().method()==='GET');
  page.on('request', r=>{if(new URL(r.url()).pathname.startsWith('/api/'))pending.set(r,performance.now());});
  page.on('requestfinished', async r=>{
    const started=pending.get(r);if(started===undefined)return;pending.delete(r);
    const response=await r.response();result.network.push({method:r.method(),route:new URL(r.url()).pathname.replace(/[0-9a-f]{8}-[0-9a-f-]{27}/gi,':id'),status:response?.status(),ms:performance.now()-started});
  });
  const step=async(label,fn)=>{const start=performance.now();await fn();result.steps.push({label,ms:performance.now()-start});};
  await step('登录',async()=>{
    await page.goto('/');await page.getByLabel('登录名').fill(input.role);await page.getByLabel('密码').fill('admin1234567');
    await page.getByRole('button',{name:'登录',exact:true}).click();
    await expect(page.locator('[data-navigation-mode]')).toBeVisible();
    await page.evaluate(()=>document.fonts.ready);
  });
  const label=input.segment==='departments'?'部门':'业务单元';
  await step('正常菜单首次进入',async()=>{
    await page.getByRole('button',{name:'切换应用',exact:true}).click();
    await page.getByRole('menuitem',{name:'设置',exact:true}).click();
    await page.getByRole('button',{name:'切换应用',exact:true}).click();
    await page.getByRole('menuitem',{name:label === '部门' ? '部门' : '设置',exact:true}).click();
    const received=await listing;expect(received.status()).toBe(200);
    const list=await received.json();expect(list.count).toBe(input.count);expect(list.items.length).toBe(Math.min(20,input.count));
    await expect(page.getByRole('heading',{name:label,exact:true})).toBeVisible();
    const table=page.getByRole('grid');await expect(table).toBeVisible();
    result.count=list.count;result.listHasSelectionVersion=Object.hasOwn(list,'selectionVersion');
    if(input.role==='reader')await expect(page.getByRole('link',{name:'新建',exact:true})).toHaveAttribute('aria-disabled','true');
    else await expect(page.getByRole('link',{name:'新建',exact:true})).toBeVisible();
    result.first=list.items[0];
  });
  await step('当前详情及返回',async()=>{
    const first=result.first;await page.getByRole('link',{name:first.name,exact:true}).first().click();
    await expect(page.getByRole('heading',{name:new RegExp(first.name.replace(/[.*+?^${}()|[\]\\]/g,'\\$&'))})).toBeVisible();
    await expect(page.locator('.effective-dates')).toBeVisible();
    await page.goBack();await expect(page.getByRole('heading',{name:label,exact:true})).toBeVisible();
    if(input.screenshot)await page.screenshot({path:input.screenshot,mask:[page.locator('.app-account-name')]});
  });
  if(input.role==='admin')await step('正常新建入口及直接可见字段',async()=>{
    await page.getByRole('link',{name:'新建',exact:true}).click();
    await expect(page.getByRole('heading',{name:'新建'+label,exact:true})).toBeVisible();
    await expect(page.getByRole('textbox',{name:'编码',exact:true})).toBeVisible();
    await expect(page.getByRole('textbox',{name:input.segment==='departments'?'部门名称':'中文名称',exact:true})).toBeVisible();
    await expect(page.locator('#'+(input.segment==='departments'?'department':'bu')+'-effectiveDate')).toHaveValue(/^\d{4}-\d{2}-\d{2}$/);
    if(input.segment==='departments'){
      await expect(page.getByRole('combobox',{name:'管理业务单元',exact:true})).toBeVisible();
      await expect(page.getByRole('combobox',{name:'行政上级',exact:true})).toBeVisible();
    }
  });
  delete result.first;result.passed=true;result.pending=pending.size;
} finally {await browser.close();}
process.stdout.write(JSON.stringify(result)+'\n');
