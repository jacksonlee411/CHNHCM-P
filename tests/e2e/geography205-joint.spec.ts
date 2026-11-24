import {randomUUID} from 'node:crypto';
import {test, expect, openMoreActions, type Page} from './fixtures';
import {open} from './bu-support';
import {request} from './dated-object-support';
import {create, detail, menu as locationMenu} from './location204-support';

async function directory(page: Page, type: 'country-regions' | 'geo-areas', code: string) {
  const listed = page.waitForResponse(r => r.request().method() === 'GET' && new URL(r.url()).pathname === `/api/v1/${type}` && !new URL(r.url()).searchParams.get('q'));
  await page.getByRole('button', {name: '基础资料', exact: true}).click();
  await page.getByRole('menuitem', {name: type === 'country-regions' ? '国家/地区' : '行政区划', exact: true}).click();
  await expect(page).toHaveURL(new RegExp(`/settings/basic-data/${type}$`));
  // The URL can change while the previous object's shared search control is
  // still mounted. Wait for this directory before entering its query.
  await expect(page.getByRole('heading', {name: type === 'country-regions' ? '国家/地区' : '行政区划', exact: true})).toBeVisible();
  expect((await listed).status()).toBe(200);
  const query = page.getByRole('textbox', {name: '搜索编码或名称', exact: true});
  await query.fill(code);
  const loaded = page.waitForResponse(r => r.request().method() === 'GET' && new URL(r.url()).pathname === `/api/v1/${type}` && new URL(r.url()).searchParams.get('q') === code);
  await page.getByRole('button', {name: '查询', exact: true}).click();
  const response = await loaded; expect(response.status()).toBe(200);
  const row = (await response.json()).items.find((v: {code: string}) => v.code === code);
  expect(row).toBeDefined();
  await page.getByRole('link', {name: row.name, exact: true}).click();
  await expect(page.getByRole('navigation', {name: '生效日期', exact: true})).toBeVisible();
  return (await request(page, 'GET', `/api/v1/${type}/${row.id}`)).body;
}

async function rename(page: Page, value: string) {
  await openMoreActions(page);
  await page.getByRole('menuitem', {name: '更正记录', exact: true}).click();
  await page.getByRole('textbox', {name: '显示名称', exact: true}).fill(value);
  const saved = page.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith('/correct'));
  await page.getByRole('button', {name: '保存', exact: true}).click();
  expect((await saved).status()).toBe(200);
  await expect(page.locator('.object-form')).toHaveCount(0);
}

test('地理205 三对象正常菜单：目录更名停选、旧采用保持、新采用拒绝恢复和双角色', async ({page}) => {
  test.setTimeout(240000);
  await open(page);
  const code = await create(page, '香港', '中西區');
  const original = await detail(page), countryID = original.selectedRecord.countryRegionId, areaID = original.selectedRecord.geoAreaId;
  const countryBefore = await directory(page, 'country-regions', 'HK');
  expect(countryBefore.id).toBe(countryID);
  await rename(page, '香港（G5联合验收）');
  const areaBefore = await directory(page, 'geo-areas', 'CW');
  expect(areaBefore.id).toBe(areaID);
  await rename(page, '中西區（G5联合验收）');
  const nextDay = (day: string) => new Date(Date.parse(day + 'T00:00:00Z') + 86400000).toISOString().slice(0, 10);
  const lastDay = areaBefore.timeline.at(-1).effectiveDate;
  const stopDay = lastDay < '2019-01-01' ? '2019-01-01' : nextDay(lastDay), resumeDay = nextDay(stopDay);
  expect(resumeDay < '2020-01-01').toBe(true);
  let stopped = false;
  const restoreName = async (type: string, id: string, name: string) => {
    const current = (await request(page, 'GET', `/api/v1/${type}/${id}`)).body, r = current.selectedRecord;
    const saved = await request(page, 'POST', `/api/v1/${type}/${id}/records/${r.recordId}/correct`, {expectedSourceVersion: current.headSourceVersion, expectedRecordRevision: r.revision, changes: {name}}, randomUUID());
    expect(saved.status).toBe(200);
  };
  const resume = async () => {
    const current = (await request(page, 'GET', `/api/v1/geo-areas/${areaID}`)).body;
    const saved = await request(page, 'POST', `/api/v1/geo-areas/${areaID}/reactivate`, {expectedSourceVersion: current.headSourceVersion, anchorRecordId: current.selectedRecord.recordId, effectiveDate: resumeDay, operationType: 'REACTIVATE', operationReason: 'RESUME_SELECTION'}, randomUUID());
    expect(saved.status).toBe(201); stopped = false;
  };
  try {
    const old = (await request(page, 'GET', `/api/v1/locations/${original.id}`)).body;
    expect(old.headSourceVersion).toBe(original.headSourceVersion);
    expect(old.selectedRecord).toEqual(original.selectedRecord);
    await create(page, '香港（G5联合验收）', '中西區（G5联合验收）');
    const adopted = await detail(page);
    expect(adopted.selectedRecord.countryName).toBe('香港（G5联合验收）');
    expect(JSON.stringify(adopted.selectedRecord.geoAreaPath)).toContain('中西區（G5联合验收）');
    await directory(page, 'geo-areas', 'CW');
    await openMoreActions(page);
    await page.getByRole('menuitem', {name: '停用行政区划与地址分区', exact: true}).click();
    await page.locator('#record-date').fill(stopDay);
    const stop = page.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith('/deactivate'));
    await page.getByRole('button', {name: '确认停用', exact: true}).click();
    expect((await stop).status()).toBe(201); stopped = true;
    await expect(page.locator('.object-form')).toHaveCount(0);
    await locationMenu(page);
    await page.getByRole('textbox', {name: '搜索编码或名称', exact: true}).fill(code);
    await page.getByRole('button', {name: '查询', exact: true}).click();
    await page.getByRole('link', {name: original.selectedRecord.name, exact: true}).click();
    await openMoreActions(page);
    await page.getByRole('menuitem', {name: '更正记录', exact: true}).click();
    await expect(page.getByRole('combobox', {name: '国家/地区', exact: true})).toHaveText(original.selectedRecord.countryName);
    await expect(page.getByRole('combobox', {name:'行政区划',exact:true})).toHaveValue(/中西區/);
    await page.getByRole('textbox', {name: '详细地址', exact: true}).fill('G5目录停选后的原地点门牌');
    await page.getByRole('button', {name: '保存', exact: true}).click();
    await expect(page.locator('.object-form')).toHaveCount(0);
    const maintained = await detail(page);
    expect(maintained.selectedRecord.countryName).toBe(original.selectedRecord.countryName);
    expect(maintained.selectedRecord.geoAreaPath).toEqual(original.selectedRecord.geoAreaPath);
    const newInput = {code: `G5-${randomUUID().slice(0,8)}`, name: 'G5候选恢复验证', countryRegionId: countryID, geoAreaId: areaID, addressLine: 'G5新采用地址', effectiveDate: '2020-01-01', operationType: 'CREATE', operationReason: 'NEW_LOCATION'};
    const denied = await request(page, 'POST', '/api/v1/locations', newInput, randomUUID());
    expect(denied.status).toBe(409);
    await resume();
    const accepted = await request(page, 'POST', '/api/v1/locations', newInput, randomUUID());
    expect(accepted.status).toBe(201);
    const exact = (await request(page, 'GET', `/api/v1/locations/${original.id}?sourceVersion=${original.headSourceVersion}`)).body;
    expect(exact.selectedRecord).toEqual(original.selectedRecord);
    const countryOld = (await request(page, 'GET', `/api/v1/country-regions/${countryID}?sourceVersion=${countryBefore.headSourceVersion}`)).body;
    const areaOld = (await request(page, 'GET', `/api/v1/geo-areas/${areaID}?sourceVersion=${areaBefore.headSourceVersion}`)).body;
    expect(countryOld.selectedRecord.name).toBe(countryBefore.selectedRecord.name);
    expect(areaOld.selectedRecord.name).toBe(areaBefore.selectedRecord.name);
  } finally {
    if (stopped) await resume();
    await restoreName('country-regions', countryID, countryBefore.selectedRecord.name);
    await restoreName('geo-areas', areaID, areaBefore.selectedRecord.name);
  }
  await page.getByRole('button', {name: '退出', exact: true}).click();
  await open(page, 'reader');
  await directory(page, 'country-regions', 'HK');
  await expect(page.getByRole('button', {name: '新增记录', exact: true})).toHaveCount(0);
  await directory(page, 'geo-areas', 'CW');
  await expect(page.getByRole('button', {name: '新增记录', exact: true})).toHaveCount(0);
  await locationMenu(page);
  await expect(page.getByRole('link', {name: '新建', exact: true})).toBeDisabled();
  await page.getByRole('textbox', {name: '搜索编码或名称', exact: true}).fill(code);
  await page.getByRole('button', {name: '查询', exact: true}).click();
  await page.getByRole('link', {name: original.selectedRecord.name, exact: true}).click();
  await expect(page.getByRole('heading', {name: `${original.code} · ${original.selectedRecord.name}`, exact: true})).toBeVisible();
  const denied = await request(page, 'POST', '/api/v1/locations', {code:'G5-READER',name:'不可创建',countryRegionId:countryID,geoAreaId:areaID,addressLine:'拒绝',effectiveDate:'2020-01-01',operationType:'CREATE',operationReason:'NEW_LOCATION'}, randomUUID());
  expect(denied.status).toBe(403);
});
