import { randomUUID } from 'node:crypto';
import { test, expect, openMoreActions } from './fixtures';
import { open, choose } from './bu-support';
import { apiBU, apiDepartment, request } from './dated-object-support';

test('BU覆盖变化自动预览识别部门引用，取消不落库，合法日期可直接保存', async ({ page }) => {
  await open(page);
  const bu = await apiBU(page, '覆盖预览事业部');
  const dep = await apiDepartment(page, bu.id, { name: '截止日验证部门' });
  const departmentPath = '/api/v1/departments/' + dep.id;
  const department = (await request(page, 'GET', departmentPath)).body;
  const stop = await request(page, 'POST', departmentPath + '/deactivate', {
    expectedSourceVersion: department.headSourceVersion,
    anchorRecordId: department.timeline.at(-1).recordId,
    effectiveDate: '2035-01-01', operationType: 'DEACTIVATE', operationReason: 'ORG_RESTRUCTURING',
  }, randomUUID());
  expect(stop.status).toBe(201);
  await page.getByLabel('搜索编码或名称').fill(bu.code);
  await page.getByRole('button', { name: '查询', exact: true }).click();
  await page.getByRole('row').filter({ hasText: bu.code }).getByRole('link', { name: bu.name, exact: true }).click();
  const path = '/api/v1/business-units/' + bu.id;
  const before = (await request(page, 'GET', path)).body;
  const form = async () => {
    await openMoreActions(page);
    await page.getByRole('menuitem', { name: '停用业务单元', exact: true }).click();
    await choose(page, '操作原因', '业务终止');
  };
  const date = page.getByLabel('生效日期').and(page.locator('input'));
  await form();
  const conflict = page.waitForResponse(r => r.url().endsWith('/preview') && r.request().postDataJSON()?.effectiveDate === '2034-12-31');
  await date.fill('2034-12-31');
  const denied = await conflict;
  expect(denied.status()).toBe(409);
  expect((await denied.json()).code).toBe('M02_DEPARTMENT_REFERENCE_CONFLICT');
  await expect(page.getByText('当前关系不满足保存条件', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: '确认停用', exact: true })).toBeDisabled();
  await page.getByRole('button', { name: '取消编辑', exact: true }).click();
  await page.getByRole('button', { name: '放弃修改', exact: true }).click();
  expect((await request(page, 'GET', path)).body.headSourceVersion).toBe(before.headSourceVersion);
  await form();
  const accepted = page.waitForResponse(r => r.url().endsWith('/preview') && r.request().postDataJSON()?.effectiveDate === '2035-01-01');
  await date.fill('2035-01-01');
  expect((await accepted).status()).toBe(200);
  await expect(page.getByRole('button', { name: '预览', exact: true })).toHaveCount(0);
  const saved = page.waitForResponse(r => r.url().endsWith('/deactivate') && r.request().method() === 'POST');
  await page.getByRole('button', { name: '确认停用', exact: true }).click();
  expect((await saved).status()).toBe(201);
  await expect(page.locator('.object-form')).toHaveCount(0);
  const after = (await request(page, 'GET', path)).body;
  expect(after.timeline.at(-1)).toMatchObject({ state: 'inactive', effectiveDate: '2035-01-01' });
  const old = await request(page, 'GET', path + '?sourceVersion=' + before.headSourceVersion);
  expect(old.status).toBe(200);
  expect(old.body.timeline).toEqual(before.timeline);
});
