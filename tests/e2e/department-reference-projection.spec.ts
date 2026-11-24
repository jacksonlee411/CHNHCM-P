import { randomUUID } from 'node:crypto';
import { test, expect, openMoreActions, type Page } from './fixtures';
import { open } from './bu-support';
import { withPolicy } from './grants';
import { menu, departmentKind as dep, request, reference, fillNew, apiBU, apiDepartment } from './dated-object-support';

const detailField = (page: Page, label: string) => page.locator('.object-detail-field').filter({ has: page.getByText(label, { exact: true }) });
async function edit(page: Page, code: string, action = 'CORRECT') {
  await menu(page, dep);
  await page.getByPlaceholder('搜索编码或名称').fill(code);
  await page.getByRole('button', { name: '查询', exact: true }).click();
  await page.getByRole('link', { name: code, exact: true }).click();
  if (action === 'ADD') await page.getByRole('button', { name: '新增记录', exact: true }).click();
  else { await openMoreActions(page); await page.getByRole('menuitem', { name: action === 'CORRECT' ? '更正记录' : '重新启用部门', exact: true }).click(); }
}

for (const action of ['CORRECT', 'ADD', 'REACTIVATE']) for (const endpoint of ['business-units', 'parents']) test(`${action}先改名称后仍能读取${endpoint}候选，未改关系不发送空changes`, async ({ page }) => {
  await open(page); const bu = await apiBU(page), parent = await apiDepartment(page, bu.id), child = await apiDepartment(page, bu.id, { parent: parent.id });
  const initial = await request(page, 'GET', '/api/v1/departments/' + child.id); expect(initial.status).toBe(200);
  if (action === 'REACTIVATE') {
    const stopped = await request(page, 'POST', '/api/v1/departments/' + child.id + '/deactivate', { anchorRecordId: initial.body.selectedRecord.recordId,
      expectedSourceVersion: initial.body.headSourceVersion, effectiveDate: '2026-02-01', operationType: 'DEACTIVATE', operationReason: 'ORG_RESTRUCTURING' }, randomUUID());
    expect(stopped.status).toBe(201);
  }
  await edit(page, child.code, action);
  if (action !== 'CORRECT') await page.locator('#record-date').fill('2026-03-01');
  await page.getByRole('textbox', { name: '部门名称', exact: true }).fill('只改名称的草稿');
  const code = endpoint === 'parents' ? parent.code : bu.code;
  const response = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/departments/references/' + endpoint && r.request().postDataJSON().q === code);
  const field = page.getByRole('combobox', { name: endpoint === 'parents' ? '行政上级' : '管理业务单元', exact: true });
  await field.click(); await field.fill(code);
  const received = await response;
  expect(received.status()).toBe(200);
  expect(received.request().postDataJSON().context).not.toHaveProperty('changes');
  await expect(page.getByRole('option').filter({ hasText: endpoint === 'parents' ? parent.code : bu.code })).toBeVisible();
});

for (const scenario of [
  { action: 'department.reference.read', label: '行政上级', endpoint: 'parents', property: 'parentDepartment', fault: '503' },
  { action: 'department.reference.read', label: '行政上级', endpoint: 'parents', property: 'parentDepartment', fault: '断连' },
  { action: 'bu.reference.read', label: '管理业务单元', endpoint: 'business-units', property: 'managementBu', fault: '503' },
] as const) test(`引用200撤权清除${scenario.label}标签，详情重读${scenario.fault}仍保留字段输入及关系ID`, async ({ page }) => {
  test.setTimeout(60000); await open(page);
  const bu = await apiBU(page, '需保护的事业部名称'), parent = await apiDepartment(page, bu.id, { name: '需保护的上级名称' });
  const child = await apiDepartment(page, bu.id, { parent: parent.id });
  const path = '/api/v1/departments/' + child.id, pattern = '**' + path + '?*';
  const before = await request(page, 'GET', path); expect(before.status).toBe(200);
  const initial = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/departments/references/' + scenario.endpoint);
  await edit(page, child.code); expect((await initial).status()).toBe(200);
  const field = page.getByRole('combobox', { name: scenario.label, exact: true });
  await expect(field).toHaveValue(/需保护的/);
  await page.getByRole('textbox', { name: '部门名称', exact: true }).fill('仍可读的未保存名称');
  await withPolicy(page, async (where, mutate) => {
    mutate(`UPDATE access.subject_grant SET read_fields=ARRAY[]::text[] WHERE ${where} AND action='${scenario.action}'`);
    const current = await request(page, 'GET', path); expect(current.status).toBe(200);
    expect(current.body.selectedRecord[scenario.property]).not.toHaveProperty('name');
    expect(current.body.selectedRecord[scenario.property + 'Id']).toBe(before.body.selectedRecord[scenario.property + 'Id']);
    let reloads = 0, release!: () => void;
    const held = new Promise<void>(resolve => { release = resolve; });
    await page.route(pattern, async route => {
      reloads++; await held;
      return scenario.fault === '503' ? route.fulfill({ status: 503, json: { code: 'M02_EXECUTION_LIMIT', message: '测试详情重读失败' } }) : route.abort('failed');
    });
    try {
      const refreshed = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/departments/references/' + scenario.endpoint);
      await field.click(); const response = await refreshed; expect(response.status()).toBe(200);
      const result = await response.json(); expect(result.items).toHaveLength(0);
      expect(result.selectedReference.id).toBe(before.body.selectedRecord[scenario.property].id);
      expect(result.selectedReference).not.toHaveProperty('name');
      await expect.poll(() => reloads).toBeGreaterThan(0);
      await expect(field).not.toHaveValue(/需保护的/);
      await expect(page.getByRole('textbox', { name: '部门名称', exact: true })).toHaveValue('仍可读的未保存名称');
      release(); await expect(page.locator('.object-form-error')).toBeVisible();
      await page.getByRole('button', { name: '取消编辑', exact: true }).click();
      const discard = page.getByRole('dialog');
      await expect(discard).toBeVisible(); await discard.getByRole('button', { name: '放弃修改', exact: true }).click();
      await expect(page.locator('.object-form')).toHaveCount(0);
      await expect(detailField(page, scenario.label)).not.toContainText('需保护的');
    } finally { release(); await page.unroute(pattern); }
  });
  await page.reload(); await expect(detailField(page, scenario.label)).toContainText('需保护的');
  const after = await request(page, 'GET', path);
  expect(after.body.headSourceVersion).toBe(before.body.headSourceVersion);
  expect(after.body.selectedRecord).toEqual(before.body.selectedRecord);
});

test('仍可读但因切换管理BU而不合资格的上级保留名称及原ID，提交由Go拒绝', async ({ page }) => {
  test.setTimeout(60000); await open(page);
  const a = await apiBU(page), b = await apiBU(page), parent = await apiDepartment(page, a.id, { name: '原关系仍可读' });
  const child = await apiDepartment(page, a.id, { parent: parent.id });
  const initial = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/departments/references/parents');
  await edit(page, child.code); expect((await initial).status()).toBe(200);
  await reference(page, '管理业务单元', b.code);
  const selected = page.getByRole('combobox', { name: '行政上级', exact: true });
  const refreshed = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/departments/references/parents' && r.request().postDataJSON().context.changes?.managementBuId === b.id);
  await selected.click(); const result = await (await refreshed).json();
  expect(result.items.some((r: { id: string }) => r.id === parent.id)).toBe(false);
  expect(result.selectedReference.name).toBe('原关系仍可读');
  await page.keyboard.press('Escape'); await expect(selected).toHaveValue(/原关系仍可读/);
  const save = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname.endsWith('/correct'));
  await page.getByRole('button', { name: '保存', exact: true }).click();
  expect((await save).status()).toBe(409);
  await expect(page.locator('.object-form-error')).toBeVisible();
  await expect(selected).toHaveValue(/原关系仍可读/);
  const current = await request(page, 'GET', '/api/v1/departments/' + child.id);
  expect(current.body.selectedRecord.managementBuId).toBe(a.id);
  expect(current.body.selectedRecord.parentDepartmentId).toBe(parent.id);
});

test('新建引用撤名称读权后仅显示仍可读编码，保留名称草稿和选定BU保存', async ({ page }) => {
  await open(page); const bu = await apiBU(page, '新建引用名称');
  await menu(page, dep); await fillNew(page, dep, { bu: bu.code, name: '应保留的新建输入' });
  const field = page.getByRole('combobox', { name: '管理业务单元', exact: true });
  await expect(field).toHaveValue(/新建引用名称/);
  await withPolicy(page, async (where, mutate) => {
    mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE ${where} AND action='bu.reference.read'`);
    const read = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/departments/references/business-units');
    await field.click(); const response = await read; expect(response.status()).toBe(200);
    expect((await response.json()).selectedReference).not.toHaveProperty('name');
    await page.keyboard.press('Escape'); await expect(field).toHaveValue(bu.code);
    await expect(page.getByRole('textbox', { name: '部门名称', exact: true })).toHaveValue('应保留的新建输入');
    const saved = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === '/api/v1/departments');
    await page.getByRole('button', { name: '保存', exact: true }).click(); const result = await saved;
    expect(result.status()).toBe(201); expect(result.request().postDataJSON().managementBuId).toBe(bu.id);
  });
});

test('列表筛选引用撤名称读权后不回显旧标签，稳定BU身份仍可应用', async ({ page }) => {
  await open(page); const bu = await apiBU(page, '筛选引用名称'), child = await apiDepartment(page, bu.id);
  await menu(page, dep); await openMoreActions(page); await page.getByRole('menuitem', { name: '列筛选…', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: '列筛选', exact: true });
  await dialog.getByRole('combobox', { name: '列', exact: true }).click(); await page.getByRole('option', { name: '管理业务单元', exact: true }).click();
  await reference(page, '管理业务单元', bu.code);
  const field = page.getByRole('combobox', { name: '管理业务单元', exact: true });
  await withPolicy(page, async (where, mutate) => {
    mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE ${where} AND action='bu.reference.read'`);
    const read = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/departments/references/business-units' && r.request().postDataJSON().selectedId === bu.id);
    await field.click(); expect((await read).status()).toBe(200);
    await page.keyboard.press('Escape'); await expect(field).toHaveValue(bu.code);
    await dialog.getByRole('button', { name: '应用', exact: true }).click();
    await expect(page).toHaveURL(u => u.searchParams.get('filterValue') === bu.id);
    await expect(page.getByRole('link', { name: child.code, exact: true })).toBeVisible();
  });
});

for (const kind of ['bu', 'parent']) test(`查询其他${kind}时相同所选投影不回填旧标签，实际搜索词保持`, async ({ page }) => {
  await open(page);
  const bu = await apiBU(page, '原事业部'), other = await apiBU(page, '待选择事业部');
  const parent = await apiDepartment(page, bu.id), otherParent = await apiDepartment(page, bu.id), child = await apiDepartment(page, bu.id, { parent: parent.id });
  await edit(page, child.code);
  const field = page.getByRole('combobox', { name: kind === 'bu' ? '管理业务单元' : '行政上级', exact: true });
  const code = kind === 'bu' ? other.code : otherParent.code, endpoint = kind === 'bu' ? 'business-units' : 'parents';
  const response = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/departments/references/' + endpoint && r.request().postDataJSON().q === code);
  await field.click(); await field.fill(code); expect((await response).status()).toBe(200);
  const option = page.getByRole('option').filter({ hasText: code });
  await expect(option).toBeVisible(); await expect(field).toHaveValue(code);
  await option.click(); await expect(field).toHaveValue(new RegExp(code));
});

test('T03首次无搜索父候选精确分页，第二页所选标签保留并直接保存', async ({ page }) => {
  await open(page);
  const bu = await apiBU(page), prefix = 'P131_' + randomUUID().slice(0, 6);
  const parents = [];
  for (let i = 0; i < 21; i++) parents.push(await apiDepartment(page, bu.id, { code: prefix + '_' + String(i).padStart(2, '0') }));
  const child = await apiDepartment(page, bu.id);
  await edit(page, child.code);
  const field = page.getByRole('combobox', { name: '行政上级', exact: true });
  const first = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/departments/references/parents' && !r.request().postDataJSON().q);
  await field.click();
  const firstResponse = await first;
  expect(firstResponse.status()).toBe(200);
  const firstPage = await firstResponse.json();
  expect(firstPage.count).toBeUndefined(); expect(firstPage.hasMore).toBe(true); expect(firstPage.items).toHaveLength(20);
  const second = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/departments/references/parents' && r.request().postDataJSON().cursor === firstPage.nextCursor);
  await page.getByRole('button', { name: '下一页候选', exact: true }).click();
  const secondPage = await (await second).json();
  expect(secondPage.count).toBeUndefined(); expect(secondPage.hasMore).toBe(false); expect(secondPage.items.map((r: {id: string}) => r.id)).toEqual([parents[20].id]);
  await page.getByRole('option').filter({ hasText: parents[20].code }).click();
  await expect(field).toHaveValue(new RegExp(parents[20].code));
  const save = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname.endsWith('/correct'));
  await page.getByRole('button', { name: '保存', exact: true }).click();
  expect((await save).status()).toBe(200);
  const current = await request(page, 'GET', '/api/v1/departments/' + child.id);
  expect(current.body.selectedRecord.parentDepartmentId).toBe(parents[20].id);
});

test('T03候选选定后上级未来停用，保存拒绝并保留关系和名称草稿', async ({ page }) => {
  await open(page);
  const bu = await apiBU(page), parent = await apiDepartment(page, bu.id, { name: '候选时可用的上级' }), child = await apiDepartment(page, bu.id);
  const before = await request(page, 'GET', '/api/v1/departments/' + child.id);
  await edit(page, child.code);
  await reference(page, '行政上级', parent.code);
  const name = page.getByRole('textbox', { name: '部门名称', exact: true });
  await name.fill('拒绝后仍保留的名称');
  const parentBefore = await request(page, 'GET', '/api/v1/departments/' + parent.id);
  const stopped = await request(page, 'POST', '/api/v1/departments/' + parent.id + '/deactivate', {
    expectedSourceVersion: parentBefore.body.headSourceVersion, anchorRecordId: parentBefore.body.selectedRecord.recordId,
    effectiveDate: '2099-01-01', operationType: 'DEACTIVATE', operationReason: 'ORG_RESTRUCTURING',
  }, randomUUID());
  expect(stopped.status).toBe(201);
  const save = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname.endsWith('/correct'));
  await page.getByRole('button', { name: '保存', exact: true }).click();
  expect((await save).status()).toBe(409);
  await expect(page.locator('.object-form-error')).toBeVisible();
  await expect(name).toHaveValue('拒绝后仍保留的名称');
  await expect(page.getByRole('combobox', { name: '行政上级', exact: true })).toHaveValue(/候选时可用的上级/);
  const after = await request(page, 'GET', '/api/v1/departments/' + child.id);
  expect(after.body.headSourceVersion).toBe(before.body.headSourceVersion);
  expect(after.body.selectedRecord.parentDepartmentId).toBeFalsy();
  await page.getByRole('button', { name: '取消编辑', exact: true }).click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.getByRole('dialog').getByRole('button', { name: '放弃修改', exact: true }).click();
});
