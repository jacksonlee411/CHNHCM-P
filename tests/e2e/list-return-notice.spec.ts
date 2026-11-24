import { randomUUID } from 'node:crypto';
import { test, expect, openMoreActions } from './fixtures';
import { open, choose } from './bu-support';
import { apiBU, menu, objects, request } from './dated-object-support';

for (const kind of objects) test(`${kind.label}返回定位随主动查询结束，缺失目标不额外提示`, async ({ page }) => {
  test.setTimeout(120000);
  await open(page);
  const bu = kind.segment === 'departments' ? await apiBU(page) : undefined;
  const prefix = 'RET_' + randomUUID().slice(0, 8);
  const items: { id: string; code: string; name: string }[] = [];
  for (let i = 0; i < 21; i++) {
    const code = prefix + '_' + String(i).padStart(2, '0'), name = prefix + '名称' + i;
    const result = await request(page, 'POST', `/api/v1/${kind.segment}`, {
      code, name, effectiveDate: '2026-01-01', operationType: 'CREATE', operationReason: 'NEW_ORGANIZATION',
      ...(bu ? { managementBuId: bu.id, parentDepartmentId: null } : {}),
    }, randomUUID());
    expect(result.status).toBe(201);
    items.push({ id: result.body.buId ?? result.body.departmentId, code, name });
  }
  await menu(page, kind);
  const input = page.getByRole('textbox', { name: '搜索编码或名称', exact: true });
  const grid = page.getByRole('grid', { name: `${kind.label}列表`, exact: true });
  const notice = page.locator('[data-list-return-fallback]');
  const query = async (q: string) => {
    await input.fill(q);
    const response = page.waitForResponse(r => new URL(r.url()).pathname === `/api/v1/${kind.segment}`
      && r.request().method() === 'GET' && new URL(r.url()).searchParams.get('q') === q);
    await page.getByRole('button', { name: '查询', exact: true }).click();
    expect((await response).status()).toBe(200);
    await expect(page.getByText('正在查询…', { exact: true })).toHaveCount(0);
  };
  const returnFromFirst = async (browserBack = false) => {
    const link = grid.locator('a[data-object-link]').first();
    const name = await link.innerText();
    // Virtual row recycling can change what .first() refers to before a click.
    await grid.getByRole('link', { name, exact: true }).click();
    await expect(page.getByRole('heading', { level: 1 })).toContainText(name);
    if (browserBack) await page.goBack();
    else await page.getByRole('link', { name: '返回列表', exact: true }).click();
    await expect(grid.getByRole('link', { name, exact: true })).toBeFocused();
    await expect(grid.locator('.list-return-selected')).toHaveCount(1);
    await expect(notice).toHaveCount(0);
  };
  await query(prefix);
  await returnFromFirst();
  await input.fill(items[20].code); // A draft alone does not end return positioning.
  await expect(grid.locator('.list-return-selected')).toHaveCount(1);
  await query(items[20].code);
  await expect(page.getByText('1–1 / 1', { exact: true })).toBeVisible();
  await expect(notice).toHaveCount(0);
  expect(new URL(page.url()).searchParams.has('selected')).toBe(false);

  for (const action of ['same-query', 'page', 'page-size', 'sort', 'status', 'column-filter']) {
    await query(prefix);
    await choose(page, '每页条数', '20');
    await expect(page.getByText('1–20 / 21', { exact: true })).toBeVisible();
    // Browser Back has no selected URL parameter; its saved row must also expire.
    await returnFromFirst(action === 'same-query');
    if (['page', 'page-size', 'sort'].includes(action)) {
      await grid.locator('.MuiDataGrid-row').first().getByRole('checkbox').check();
      await expect(page.getByText('已选择1项', { exact: true })).toBeVisible();
    }
    if (action === 'same-query') await query(prefix);
    if (action === 'page') {
      await page.getByRole('button', { name: '下一页', exact: true }).click();
      await expect(page.getByText('21–21 / 21', { exact: true })).toBeVisible();
    }
    if (action === 'page-size') {
      await choose(page, '每页条数', '50');
      await expect(page.getByText('1–21 / 21', { exact: true })).toBeVisible();
    }
    if (action === 'sort') {
      await page.getByRole('columnheader', { name: /名称/ }).click();
      await expect(page.getByRole('columnheader', { name: /名称/ })).toHaveAttribute('aria-sort', 'ascending');
      await expect(page.getByText('正在核对选择…', { exact: true })).toHaveCount(0);
    }
    if (action === 'status') {
      await openMoreActions(page);
      await page.getByRole('menuitem', { name: '状态筛选…', exact: true }).click();
      await choose(page, '状态', '启用');
      await page.getByRole('button', { name: '应用', exact: true }).click();
      await expect(page).toHaveURL(u => u.searchParams.get('status') === 'active');
    }
    if (action === 'column-filter') {
      await openMoreActions(page);
      await page.getByRole('menuitem', { name: '列筛选…', exact: true }).click();
      const dialog = page.getByRole('dialog', { name: '列筛选', exact: true });
      await choose(page, '列', '编码');
      await dialog.getByRole('textbox', { name: '值', exact: true }).fill(items[20].code);
      await dialog.getByRole('button', { name: '应用', exact: true }).click();
      await expect(page.getByText('1–1 / 1', { exact: true })).toBeVisible();
    }
    await expect(notice).toHaveCount(0);
    await expect(grid.locator('.list-return-selected')).toHaveCount(0);
    expect(new URL(page.url()).searchParams.has('selected')).toBe(false);
    if (['page', 'page-size', 'sort'].includes(action)) {
      await expect(page.getByText('已选择1项', { exact: true })).toBeVisible();
      // The selection toolbar replaces search; finish it before the next journey.
      await page.getByRole('button', { name: '取消选择', exact: true }).click();
      await expect(page.getByRole('toolbar', { name: `所选${kind.label}`, exact: true })).toHaveCount(0);
    }
  }

  await menu(page, kind);
  // The same list heading remains visible while menu navigation resets filters.
  // Wait for that navigation to commit before typing into the new query draft.
  await expect(page).toHaveURL(u => u.pathname === `/settings/organization/${kind.segment}` && u.search === '');
  await expect(input).toHaveValue('');
  await query(items[0].name);
  await page.getByRole('link', { name: items[0].name, exact: true }).click();
  await openMoreActions(page);
  await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
  await page.getByRole('textbox', { name: kind.name, exact: true }).fill('更名后不匹配原查询');
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.locator('.object-form')).toHaveCount(0);
  await page.getByRole('link', { name: '返回列表', exact: true }).click();
  await expect(page.getByText('0–0 / 0', { exact: true })).toBeVisible();
  await expect(notice).toHaveCount(0);
  await expect(page.locator('[data-list-heading]')).toBeFocused();
  await expect(page.getByText(`刚查看的${kind.label}未显示在本页。`, { exact: true })).toHaveCount(0);
  await expect(page.getByRole('link', { name: '查看原对象', exact: true })).toHaveCount(0);
  await expect(page.getByRole('alert')).toHaveCount(0);
  await query(items[0].name);
  await expect(page.getByText('0–0 / 0', { exact: true })).toBeVisible();
  await expect(notice).toHaveCount(0);
  expect(new URL(page.url()).searchParams.has('selected')).toBe(false);
  await query(items[0].code);
  await grid.getByRole('link', { name: '更名后不匹配原查询', exact: true }).click();
  await expect(page.getByRole('heading', { level: 1 })).toContainText('更名后不匹配原查询');
});
