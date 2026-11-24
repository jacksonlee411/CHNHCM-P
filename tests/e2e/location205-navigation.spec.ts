import {randomUUID} from 'node:crypto';
import {test, expect} from './fixtures';
import {open} from './bu-support';
import {request} from './dated-object-support';
import {menu, root} from './location204-support';

for (const role of ['admin', 'reader']) test(`205 ${role}地点主线导航位置、跨页无列表重读及返回`, async ({page}) => {
  test.setTimeout(120000);
  await open(page);
  const prefix = 'NAV205-' + Date.now();
  const countries = await request(page, 'GET', '/api/v1/country-regions?q=MO');
  const mo = countries.body.items.find((row: {code: string}) => row.code === 'MO');
  expect(mo).toBeDefined();
  for (let i=0;i<22;i++) {
    const result = await request(page, 'POST', '/api/v1/locations', {code: prefix+'-'+String(i).padStart(2,'0'), name: prefix+'-'+i, countryRegionId: mo.id, addressLine: '导航验收地址', effectiveDate: '2020-01-01', operationType: 'CREATE', operationReason: 'NEW_LOCATION'}, randomUUID());
    expect(result.status).toBe(201);
  }
  if (role === 'reader') { await page.getByRole('button', {name:'退出',exact:true}).click(); await open(page, role); }
  await menu(page);
  await page.getByRole('textbox', {name:'搜索编码或名称',exact:true}).fill(prefix);
  await page.getByRole('button', {name:'查询',exact:true}).click();
  const grid = page.getByRole('grid', {name:'地点列表',exact:true});
  await grid.getByRole('link', {name:prefix+'-0',exact:true}).click();
  const pager = page.getByRole('navigation', {name:'对象导航',exact:true});
  await expect(pager).toContainText('1 / 22');
  await expect(page.getByRole('button', {name:'上一个对象',exact:true})).toBeDisabled();
  await page.getByRole('link', {name:'返回列表',exact:true}).click();
  await page.locator('.MuiDataGrid-virtualScroller').hover(); await page.mouse.wheel(0,3000);
  await grid.getByRole('link', {name:prefix+'-19',exact:true}).click();
  await expect(pager).toContainText('20 / 22');
  const listReads: string[] = [];
  page.on('request', r => { if(r.method()==='GET' && new URL(r.url()).pathname==='/api/v1/locations')listReads.push(r.url()); });
  for(const [direction,position] of [['下一个对象',21],['上一个对象',20],['下一个对象',21],['下一个对象',22]] as const) {
    await page.getByRole('button', {name:direction,exact:true}).click();
    await expect(pager).toContainText(`${position} / 22`);
    await expect(page.getByRole('heading', {level:1})).toContainText(prefix+'-'+String(position-1).padStart(2,'0'));
  }
  expect(listReads).toHaveLength(0);
  await expect(page.getByRole('button', {name:'下一个对象',exact:true})).toBeDisabled();
  for(const width of [390,320]) {
    await page.setViewportSize({width,height:844});
    await expect(pager.getByLabel('第22个，共22个匹配对象',{exact:true})).toBeVisible();
    expect(await pager.evaluate(el=>el.getBoundingClientRect().right<=innerWidth)).toBe(true);
  }
  await page.setViewportSize({width:1440,height:900});
  await page.getByRole('link', {name:'返回列表',exact:true}).click();
  await expect(grid.getByRole('link', {name:prefix+'-21',exact:true})).toBeFocused();
  await expect(page).toHaveURL(url=>url.pathname===root && url.searchParams.get('page')==='2' && url.searchParams.get('q')===prefix);
});
