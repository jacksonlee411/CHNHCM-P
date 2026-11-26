import { randomUUID } from 'node:crypto';
import { expect, type Page } from './fixtures';
import { choose } from './bu-support';
import { chooseApplication } from './app-navigation';
export const objects = [
  { label: '业务单元', segment: 'business-units', name: '中文名称', prefix: 'bu' },
  { label: '部门', segment: 'departments', name: '部门名称', prefix: 'department' },
] as const;
export type ObjectKind = typeof objects[number];
export const departmentKind = objects[1];
export async function menu(page: Page, kind: ObjectKind) {
  await chooseApplication(page, kind.segment === 'departments' ? '部门' : '设置');
  await expect(page.getByRole('heading', { level: 1, name: kind.label, exact: true })).toBeVisible();
}
export async function reference(page: Page, label: string, text: string) {
  const field = page.getByRole('combobox', { name: label, exact: true });
  await field.click(); await field.fill(text);
  await page.getByRole('option').filter({ hasText: text }).first().click();
}
export async function request(page: Page, method: string, path: string, body?: object, key?: string) {
  return page.evaluate(async ({method,path,body,key}) => {
    const session = await (await fetch('/api/v1/session')).json();
    const response = await fetch(path, { method, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': session.csrf, ...(key ? { 'Idempotency-Key': key } : {}) }, ...(body ? {body:JSON.stringify(body)} : {}) });
    return { status:response.status, body:await response.json() };
  }, {method,path,body,key});
}
export async function readObject(page: Page, kind: ObjectKind) {
  const url=new URL(page.url()),q=new URLSearchParams();
  for(const key of ['sourceVersion','selectedRecordId'])if(url.searchParams.has(key))q.set(key,url.searchParams.get(key)!);
  const result=await request(page,'GET',`/api/v1/${kind.segment}/`+url.pathname.split('/').at(-1)+'?'+q);
  expect(result.status).toBe(200);return result.body;
}
export async function fillNew(page: Page, kind: ObjectKind, options: {name?:string;date?:string;bu?:string;parent?:string;code?:string} = {}) {
  await page.getByRole('link',{name:'新建',exact:true}).click();
  await page.getByRole('textbox',{name:'编码',exact:true}).fill(options.code ?? 'R5_'+randomUUID().slice(0,8));
  await page.getByRole('textbox',{name:kind.name,exact:true}).fill(options.name ?? '验证组织');
  await page.locator(`#${kind.prefix}-effectiveDate`).fill(options.date ?? '2026-01-01');
  if(options.bu)await reference(page,'管理业务单元',options.bu);
  if(options.parent)await reference(page,'行政上级',options.parent);
  await choose(page,'操作原因','新设组织');
}
export async function createObject(page: Page, kind: ObjectKind, options: Parameters<typeof fillNew>[2] = {}) {
  await menu(page,kind);await fillNew(page,kind,options);
  await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);
  return readObject(page,kind);
}
export async function apiBU(page:Page,name='测试事业部',date='2026-01-01') {
  const code='R5BU_'+randomUUID().slice(0,8), r=await request(page,'POST','/api/v1/business-units',{code,name,effectiveDate:date,operationType:'CREATE',operationReason:'NEW_ORGANIZATION'},randomUUID());expect(r.status).toBe(201);return {code,id:r.body.buId,name};
}
export async function apiDepartment(page:Page,bu:string,options:{code?:string;name?:string;parent?:string;date?:string}={}) {
  const code=options.code??'R5D_'+randomUUID().slice(0,8),r=await request(page,'POST','/api/v1/departments',{code,name:options.name??code,managementBuId:bu,parentDepartmentId:options.parent??null,effectiveDate:options.date??'2026-01-01',operationType:'CREATE',operationReason:'NEW_ORGANIZATION'},randomUUID());expect(r.status).toBe(201);return {...r.body,code,id:r.body.departmentId};
}

// Both real first-record editors expose the same finite leap-day gap.
export async function cancelLeapDayPreview(page: Page) {
  await page.getByRole('menuitem', {name:'更正记录',exact:true}).click();
  const preview = page.waitForResponse(r => r.url().endsWith('/preview') && r.request().postDataJSON()?.newEffectiveDate === '2024-03-01');
  await page.getByRole('textbox', {name:'生效日期',exact:true}).fill('2024-03-01');
  expect((await (await preview).json()).newNoFactRange).toEqual({ start:'2024-02-28', endExclusive:'2024-03-01', endInclusive:'2024-02-29' });
  await expect(page.getByText('无适用资料：2024-02-28 至 2024-02-29', {exact:true})).toBeVisible();
  await page.getByRole('button',{name:'取消编辑',exact:true}).click();
  await page.getByRole('button',{name:'放弃修改',exact:true}).click();
  await expect(page.locator('.object-form')).toHaveCount(0);
}

export async function checkPairPeriod(page: Page, segment: string, action: string, start: string, last?: string) {
  const {withPolicy} = await import('./grants');
  const {openMoreActions} = await import('./fixtures');
  const url = new URL(page.url());
  const apiPath = '/api/v1/'+segment+'/'+url.pathname.split('/').at(-1);
  const head = (await request(page,'GET',apiPath)).body;
  const resume = head.timeline.find((r:{effectiveDate:string})=>r.effectiveDate===start);
  const exact = apiPath+'?selectedRecordId='+resume.recordId;
  const load = async () => (await request(page,'GET',exact)).body;
  expect((await load()).pairDeletion.changes).toEqual([]);
  const saved = await request(page,'POST',apiPath+'/records/'+resume.recordId+'/correct',{
    expectedSourceVersion:head.headSourceVersion, expectedRecordRevision:resume.revision, changes:{name:'恢复端的独立名称'},
  },randomUUID());
  expect(saved.status).toBe(200);
  const info = await load();
  expect(info.pairDeletion.changes).toHaveLength(1);
  expect(info.pairDeletion.changes[0]).toMatchObject({start, endInclusive:last ?? null});
  url.searchParams.set('selectedRecordId',resume.recordId);
  url.searchParams.delete('saved');
  await page.goto(url.toString());
  const openDialog = async () => {
    await openMoreActions(page);
    await page.getByRole('menuitem',{name:'删除停用与重新启用',exact:true}).click();
    return page.getByRole('dialog',{name:'删除停用与重新启用',exact:true});
  };
  let dialog = await openDialog();
  await expect(dialog.getByText(last ? start+' 至 '+last : '自 '+start+' 起',{exact:true})).toBeVisible();
  await dialog.getByRole('button',{name:'取消',exact:true}).click();
  expect((await load()).headSourceVersion).toBe(info.headSourceVersion);
  await withPolicy(page, async (where, mutate) => {
    mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE ${where} AND action='${action}.read'`);
    expect((await load()).pairDeletion.changes).toEqual([]);
  });
  await withPolicy(page, async (where, mutate) => {
    mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'effectiveDate') WHERE ${where} AND action='${action}.read'`);
    const limited = await load();
    await page.reload();
    if (action==='location' || action==='legalentity') {
      expect(limited.pairDeletion).toBeUndefined();
      await openMoreActions(page);
      await expect(page.getByRole('menuitem',{name:'删除停用与重新启用',exact:true})).toHaveCount(0);
      await page.keyboard.press('Escape');
    } else {
      const [change] = limited.pairDeletion.changes;
      for (const key of ['start','endExclusive','endInclusive']) expect(change).not.toHaveProperty(key);
      dialog = await openDialog();
      await expect(dialog.getByLabel('删除后的资料变化')).toContainText('恢复端的独立名称');
      await expect(dialog.getByLabel('删除后的资料变化')).not.toContainText(start);
      await dialog.getByRole('button',{name:'取消',exact:true}).click();
    }
  });
  await page.reload();
}

// The stable directory ID is the same on both sides of the deletion. Only the
// exact label adopted by the restored record changes.
export async function checkAdoptedLabelOnPairDelete(page: Page, segment: 'legal-entities' | 'locations', area = false) {
  const { openMoreActions } = await import('./fixtures');
  const { withPolicy } = await import('./grants');
  const route = new URL(page.url()).pathname;
  const api = '/api/v1/' + segment + '/' + route.split('/').at(-1);
  const get = async (path: string) => {
    const response = await request(page, 'GET', path);
    expect(response.status).toBe(200);
    return response.body;
  };
  const head = await get(api);
  const first = head.selectedRecord;
  const originalLabel = area ? first.geoAreaPath.map((node: {name: string}) => node.name).join(' / ') : first.countryName;
  const target = area ? first.geoAreaPath[0].id : first.countryRegionId;
  const catalog = '/api/v1/' + (area ? 'geo-areas' : 'country-regions') + '/' + target;
  const catalogHead = await get(catalog);
  const adopted = catalogHead.timeline.findLast((row: {effectiveDate: string}) => row.effectiveDate <= '2022-01-01');
  const originalName = adopted.name;
  const newName = originalName + '采用新名';
  const renamedLabel = area ? [newName, ...first.geoAreaPath.slice(1).map((node: {name: string}) => node.name)].join(' / ') : newName;
  const rename = async (name: string) => {
    const current = await get(catalog + '?selectedRecordId=' + adopted.recordId);
    expect((await request(page, 'POST', catalog + '/records/' + adopted.recordId + '/correct', {
      expectedSourceVersion: current.headSourceVersion, expectedRecordRevision: current.selectedRecord.revision, changes: {name},
    }, randomUUID())).status).toBe(200);
  };
  await rename(newName);
  try {
    const prefix = segment === 'locations' ? 'LOCATION' : 'ENTITY';
    expect((await request(page, 'POST', api + '/lifecycle-pairs/insert', {
      expectedSourceVersion: head.headSourceVersion, anchorRecordId: first.recordId,
      deactivationDate: '2021-01-01', deactivationReason: prefix + '_CLOSURE',
      reactivationDate: '2022-01-01', reactivationReason: prefix + '_RESUMPTION',
    }, randomUUID())).status).toBe(200);
  } finally {
    await rename(originalName);
  }
  const updated = await get(api);
  const resume = updated.timeline.find((row: {effectiveDate: string}) => row.effectiveDate === '2022-01-01');
  const selection = '?selectedRecordId=' + resume.recordId;
  const label = (row: {countryName: string; geoAreaPath: {name: string}[]}) =>
    area ? row.geoAreaPath.map(node => node.name).join(' / ') : row.countryName;
  const idKey = area ? 'geoAreaId' : 'countryRegionId';
  const changes = (await get(api + selection)).pairDeletion.changes;
  expect(changes).toHaveLength(1);
  expect(changes[0]).toMatchObject({start: '2022-01-01', endExclusive: null, endInclusive: null});
  expect(changes[0].before[idKey]).toBe(first[idKey]);
  expect(changes[0].after[idKey]).toBe(first[idKey]);
  expect(label(changes[0].before)).toBe(renamedLabel);
  expect(label(changes[0].after)).toBe(originalLabel);

  if (area) await page.setViewportSize({width: 1440, height: 540});
  const openDelete = async () => {
    await page.goto(route + selection);
    await openMoreActions(page);
    await page.getByRole('menuitem', {name: '删除停用与重新启用', exact: true}).click();
    return page.getByRole('dialog', {name: '删除停用与重新启用', exact: true});
  };
  let dialog = await openDelete();
  const summary = dialog.getByLabel('删除后的资料变化');
  await expect(summary.getByText('自 2022-01-01 起', {exact:true})).toBeVisible();
  await expect(summary.getByText((area ? '行政区划' : '国家/地区') + '：' + renamedLabel + ' → ' + originalLabel, {exact:true})).toBeVisible();
  expect(await summary.evaluate(element => element.scrollWidth <= element.clientWidth)).toBe(true);
  await dialog.getByRole('button', {name:'取消', exact:true}).click();
  expect((await get(api)).headSourceVersion).toBe(updated.headSourceVersion);

  await withPolicy(page, async (where, mutate) => {
    const action = area ? 'geoarea.history.read' : 'country.history.read';
    mutate(`DELETE FROM access.subject_grant WHERE ${where} AND action='${action}'`);
    expect((await get(api + selection)).pairDeletion.changes).toEqual([]);
    dialog = await openDelete();
    await expect(dialog.getByLabel('删除后的资料变化')).toHaveCount(0);
    await dialog.getByRole('button', {name: '取消', exact: true}).click();
  });

  dialog = await openDelete();
  await dialog.getByRole('button', {name:'删除两条记录', exact:true}).click();
  await expect(dialog).toHaveCount(0);
  const after = await get(api);
  expect(after.timeline).toHaveLength(1);
  expect(label(after.selectedRecord)).toBe(originalLabel);
  const original = await get(api + selection + '&sourceVersion=' + updated.headSourceVersion);
  expect(label(original.selectedRecord)).toBe(renamedLabel);
}
