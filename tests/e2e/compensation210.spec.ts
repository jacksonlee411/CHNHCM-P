import { randomUUID } from 'node:crypto';
import { expect, test, type Page } from './fixtures';
import { open } from './bu-support';
import { namedMenu308 } from './staffing308-support';
import { calibrateDesktopWindow, nativeZoomBrowser } from './native-zoom-support';

const listPath = '/settings/compensation/grade-rates';
const menu = (page: Page) => namedMenu308(page, '职级薪酬标准', '薪酬配置', listPath);
async function gradeFixture(page: Page, code: string) {
  const session = await (await page.request.get('/api/v1/session')).json();
  const grade = await page.request.post('/api/v1/grades', { headers: { Origin: new URL(page.url()).origin, 'X-CSRF-Token': session.csrf, 'Idempotency-Key': randomUUID() },
    data: { code, name: '薪标验收职级', effectiveDate: '2027-01-01', operationType: 'CREATE', operationReason: 'NEW_GRADE' } });
  const detail = await grade.json();
  expect(grade.status(), detail.code + " " + detail.message).toBe(201);
}
async function choose(page: Page, field: string, option: string) {
  await page.getByRole('combobox', { name: field, exact: true }).click();
  await page.getByRole('option', { name: option, exact: true }).click();
}
async function more(page: Page, name: string) {
  await page.getByRole('button', { name: '更多操作', exact: true }).click();
  await page.getByRole('menuitem', { name, exact: true }).click();
}
async function save(page: Page, label = '保存') {
  const request = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname.includes('/compensation/grade-rates'));
  await page.getByRole('button', { name: label, exact: true }).click();
  expect((await request).status()).toBe(200);
  await expect(page.locator('.object-form')).toHaveCount(0);
}
async function createRate(page: Page, code: string) {
  await menu(page);
  await page.getByRole('link', { name: '新建', exact: true }).click();
  const date = page.getByRole('textbox', { name: '生效日期' });
  await expect(date).toHaveValue('1900-01-01');
  await date.fill('2027-01-01');
  await choose(page, '办理原因', '建立标准');
  await page.getByRole('combobox', { name: '职级', exact: true }).fill(code);
  await page.getByRole('option', { name: code + ' 薪标验收职级', exact: true }).click();
  await choose(page, '标准形式', '连续薪带');
  for (const [name, value] of [['下限','10000'], ['指导值','12500'], ['上限','15000']]) await page.getByRole('textbox', { name, exact: true }).fill(value);
  await save(page);
  await expect(page.getByText('15,000.00', { exact: true })).toBeVisible();
}
test.describe.configure({ timeout: 160000 });
for (const height of [900,540]) {
  test('210 职级薪标正常菜单、未来调整、更正撤去与原件 ' + height, async ({ page }) => {
    await page.setViewportSize({width:1440, height}); await open(page);
    const code = 'G210' + randomUUID().slice(0,8).toUpperCase();
    await gradeFixture(page, code); await createRate(page, code);
    await page.getByRole('button', { name: '新增记录', exact: true }).click();
    await page.getByRole('textbox', {name:'生效日期'}).fill('2027-07-01');
    await choose(page, '办理原因', '薪酬制度调整');
    for (const [name,value] of [['下限','11000'],['指导值','13500'],['上限','16500']]) await page.getByRole('textbox',{name,exact:true}).fill(value);
    await save(page); await expect(page.getByText('16,500.00',{exact:true})).toBeVisible();
    await more(page,'更正记录');
    const maximum = page.getByRole('textbox',{name:'上限',exact:true});
    await expect(maximum).toHaveValue('16500.00'); await maximum.fill('16000');
    await page.getByRole('button',{name:'取消',exact:true}).click();
    await page.getByRole('button',{name:'放弃修改',exact:true}).click();
    await expect(page.getByText('16,500.00',{exact:true})).toBeVisible();
    await more(page,'更正记录'); await maximum.fill('16000'); await save(page);
    await expect(page.getByText('16,000.00',{exact:true})).toBeVisible();
    await page.getByRole('tab',{name:'变更日志'}).click();
    await page.getByRole('button',{name:'来源明细'}).first().click();
    await page.getByRole('link',{name:'办理前资料'}).first().click();
    await expect(page.getByText('正在查看只读历史资料',{exact:true})).toBeVisible();
    await expect(page.locator('.object-detail-field').filter({has:page.getByText('上限',{exact:true})})).toContainText('16,500.00');
    await expect(page.getByRole('button',{name:'更多操作'})).toHaveCount(0);
    await page.getByRole('button',{name:'返回变更日志'}).click();
    await page.getByRole('tab',{name:'基本资料'}).click();
    await more(page,'删除此记录'); await save(page,'确认删除');
    await expect(page.getByText('15,000.00',{exact:true})).toBeVisible();
    await page.getByRole('link',{name:'返回列表'}).click();
    const search=page.getByRole('textbox',{name:'搜索职级编码或名称'}); await search.fill(code); await search.press('Enter');
    await expect(page.getByRole('link',{name:code+' 薪标验收职级'})).toBeVisible();
    expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
  });
}
test('210 职级薪标原生 200% 菜单与表单保存', async () => {
  const zoom = await nativeZoomBrowser();
  try {
    const {page}=zoom; await open(page); await calibrateDesktopWindow(page); await zoom.setZoom(2);
    const code='Z210'+randomUUID().slice(0,8).toUpperCase(); await gradeFixture(page,code); await createRate(page,code);
    expect(await page.evaluate(()=>devicePixelRatio)).toBe(2);
    expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
  } finally { await zoom.close(); }
});

test('210 职级薪标并发重读保留输入及响应丢失查回', async ({ page }) => {
  await open(page);
  const code='C210'+randomUUID().slice(0,8).toUpperCase();
  await gradeFixture(page,code);
  await createRate(page,code);
  const object= new URL(page.url()).pathname.split('/').at(-1)!;
  const path='/api/v1/compensation/grade-rates/'+object;
  await more(page,'更正记录');
  const upper=page.getByRole('textbox',{name:'上限',exact:true});
  await upper.fill('15666');
  const remote=await page.evaluate(async path=>{
    const principal=await(await fetch('/api/v1/session')).json();
    const head=await(await fetch(path)).json();
    const response=await fetch(path,{method:'POST',
      headers:{'Content-Type':'application/json','X-CSRF-Token':principal.csrf,'Idempotency-Key':crypto.randomUUID()},
      body:JSON.stringify({action:'CORRECT',expectedVersion:head.sourceVersion,expectedRevision:head.record.revision,
        recordId:head.record.recordId,changes:{minimum:'10600'}})});
    return response.status;
  },path);
  expect(remote).toBe(200);
  await page.getByRole('button',{name:'保存',exact:true}).click();
  await page.getByRole('button',{name:'重新读取',exact:true}).click();
  await expect(upper).toHaveValue('15666');
  await expect(page.getByRole('textbox',{name:'下限',exact:true})).toHaveValue('10600.00');
  await save(page);
  await more(page,'更正记录');
  await upper.fill('16000');
  await page.route(url=>url.pathname===path, async route=>{
    if (route.request().method()!=='POST') return route.continue();
    const applied=await route.fetch();
    expect(applied.status()).toBe(200);
    await route.abort('failed');
  });
  await page.getByRole('button',{name:'保存',exact:true}).click();
  await expect(page.getByText('16,000.00',{exact:true})).toBeVisible();
  const results=await page.request.get(path+'/history');
  expect((await results.json()).items).toHaveLength(4);
});
