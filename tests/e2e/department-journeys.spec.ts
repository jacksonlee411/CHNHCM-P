import { randomUUID } from 'node:crypto';
import { test, expect, openMoreActions, type Page } from './fixtures';
import { open, create as createBU, choose } from './bu-support';
import { chooseApplication } from './app-navigation';
const root = '/settings/organization/';
const kinds = [
  { label: '业务单元', segment: 'business-units', name: '中文名称', prefix: 'bu' },
  { label: '部门', segment: 'departments', name: '部门名称', prefix: 'department' },
] as const;
async function menu(page: Page, label: string) { await chooseApplication(page, label === '部门' ? '部门' : '设置'); }
async function reference(page: Page, label: string, text: string) {
  const field = page.getByRole('combobox', { name: label, exact: true });
  await field.click(); await field.fill(text);
  await page.getByRole('option').filter({ hasText: text }).first().click();
}
async function read(page: Page, kind: typeof kinds[number]) {
  return page.evaluate(async segment => { const q=new URLSearchParams();const current=new URLSearchParams(location.search);for(const key of ['selectedRecordId','sourceVersion'])if(current.has(key))q.set(key,current.get(key)!); const r = await fetch(`/api/v1/${segment}/` + location.pathname.split('/').at(-1) + '?' + q); if (!r.ok) throw Error('read: ' + r.status); return r.json(); }, kind.segment);
}
for (const kind of kinds) test(`${kind.label}真实共用旅程：正常菜单、创建、更正、改日、新增、历史、启停和删除`, async ({ page }) => {
  test.setTimeout(90000);
  await open(page);
  let bu = '';
  if (kind.segment === 'departments') { bu = await createBU(page); await menu(page, kind.label); }
  const code = 'R5_' + randomUUID().slice(0, 8);
  await page.getByRole('link', { name: '新建', exact: true }).click();
  await page.getByRole('textbox', { name: '编码', exact: true }).fill(code);
  await page.getByRole('textbox', { name: kind.name, exact: true }).fill('首批验证组织');
  await page.locator(`#${kind.prefix}-effectiveDate`).fill('2026-01-01');
  if (bu) await reference(page, '管理业务单元', bu);
  await choose(page, '操作原因', '新设组织');
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.locator('.object-form')).toHaveCount(0);
  await expect(page).toHaveURL(new RegExp(root + kind.segment + '/[a-f0-9-]+\\?'));
  const created = await read(page, kind);
  expect(created.selectedRecord.name).toBe('首批验证组织');
  await expect(page.locator('.operation-summary')).toContainText('操作者：');
  if (bu) { await expect(page.getByText('企业根部门（无行政上级）', { exact: true })).toBeVisible(); await expect(page.locator('.department-path')).toContainText('首批验证组织'); }
  await openMoreActions(page); await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
  await page.getByRole('textbox', { name: kind.name, exact: true }).fill('更正后的组织');
  await page.locator('#record-date').fill('2026-01-02');
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.locator('.object-form')).toHaveCount(0);
  const corrected = await read(page, kind); expect(corrected.selectedRecord.name).toBe('更正后的组织'); expect(corrected.selectedRecord.effectiveDate).toBe('2026-01-02');
  await page.getByRole('button', { name: '新增记录', exact: true }).click();
  await page.locator('#record-date').fill('2026-06-01');
  await page.getByRole('textbox', { name: kind.name, exact: true }).fill('后续组织');
  await choose(page, '操作原因', '名称调整');
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.locator('.object-form')).toHaveCount(0);
  await page.getByRole('tab', { name: '变更日志', exact: true }).click();
  await expect(page.getByRole('table', { name: '变更日志', exact: true })).toBeVisible();
  await page.locator('.change-operation').last().getByRole('link', { name: '查看当次资料', exact: true }).first().click();
  await expect(page.getByText('正在查看只读历史资料', { exact: true })).toBeVisible();
  expect((await read(page, kind)).selectedRecord.name).toBe('首批验证组织');
  await page.getByRole('button', { name: '返回当前资料', exact: true }).click();
  await page.getByRole('button', { name: '2026-06-01', exact: true }).click();
  await expect(page.locator('.effective-dates [data-day="2026-06-01"][aria-current="date"]')).toBeVisible();
  await openMoreActions(page); await page.getByRole('menuitem', { name: `停用${kind.label}`, exact: true }).click();
  await page.locator('#record-date').fill('2026-09-01'); await choose(page, '操作原因', '业务终止');
  await page.getByRole('button', { name: '确认停用', exact: true }).click(); await expect(page.locator('.object-form')).toHaveCount(0);
  expect((await read(page, kind)).selectedRecord.state).toBe('inactive');
  if (bu) await expect(page.locator('.department-path')).toHaveText('后续组织');
  await openMoreActions(page); await page.getByRole('menuitem', { name: `重新启用${kind.label}`, exact: true }).click();
  await page.locator('#record-date').fill('2026-10-01'); await choose(page, '操作原因', '业务恢复');
  await page.getByRole('button', { name: '确认重新启用', exact: true }).click(); await expect(page.locator('.object-form')).toHaveCount(0);
  expect((await read(page, kind)).selectedRecord.state).toBe('active');
  await openMoreActions(page); await page.getByRole('menuitem', { name: '删除此记录', exact: true }).click();
  await page.getByRole('button', { name: '确认删除', exact: true }).click(); await expect(page.locator('.object-form')).toHaveCount(0);
  expect((await read(page, kind)).timeline).toHaveLength(3);
});
