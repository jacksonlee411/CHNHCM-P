import { deferAutomaticConfirmation } from './fixtures';
import { test, expect, openMoreActions, type Page } from './fixtures';
import { open, create, read } from './bu-support';

const unsaved = (page: Page) => page.getByRole('dialog', { name: '尚有未保存修改', exact: true });
async function failLogout(page: Page, failure: '503' | 'network') {
  await page.route('**/api/v1/session', async route => {
    if (route.request().method() !== 'DELETE') return route.continue();
    if (failure === 'network') await route.abort('connectionreset');
    else await route.fulfill({ status: 503, json: { code: 'SERVICE_UNAVAILABLE', message: '退出暂时失败' } });
  });
  await page.getByRole('button', { name: '退出', exact: true }).click();
  await expect(unsaved(page)).toBeVisible();
  await unsaved(page).getByRole('button', { name: '放弃修改', exact: true }).click();
  await expect(page.getByText(failure === '503' ? '退出未完成，请重试' : '退出结果尚未确认，请重试', { exact: true })).toBeVisible();
}

async function dismissNativeClose(page: Page) {
  // Use Chromium's actual beforeunload dialog, then keep editing the same page.
  const confirmation = page.waitForEvent('dialog');
  await page.close({ runBeforeUnload: true });
  const dialog = await confirmation;
  expect(dialog.type()).toBe('beforeunload');
  await dialog.dismiss();
  expect(page.isClosed()).toBe(false);
}

for (const failure of ['503', 'network'] as const) test(`退出${failure}后更正新草稿恢复取消、路由和关闭保护`, async ({ page }) => {
  await open(page); await create(page);
  const before = await read(page);
  await openMoreActions(page); await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
  await page.locator('#record-date').fill('2026-02-01');
  await page.getByRole('button', { name: '取消编辑', exact: true }).click();
  await expect(unsaved(page)).toBeVisible();
  await unsaved(page).getByRole('button', { name: '继续编辑', exact: true }).click();
  await failLogout(page, failure);
  const name = page.getByRole('textbox', { name: '中文名称', exact: true });
  await expect(name).toBeVisible(); await name.fill('退出失败后的新草稿');
  expect((await read(page)).headSourceVersion).toBe(before.headSourceVersion);
  expect.soft(await page.evaluate(() => {
    const event = new Event('beforeunload', { cancelable: true });
    window.dispatchEvent(event); return event.defaultPrevented;
  })).toBe(true);
  await page.getByRole('button', { name: '取消编辑', exact: true }).click();
  await expect(unsaved(page)).toBeVisible();
  await unsaved(page).getByRole('button', { name: '继续编辑', exact: true }).click();
  await expect(name).toHaveValue('退出失败后的新草稿');
  await page.getByRole('button', { name: '组织配置', exact: true }).click();
  await page.getByRole('menuitem', { name: '业务单元', exact: true }).click();
  await expect(unsaved(page)).toBeVisible();
  await unsaved(page).getByRole('button', { name: '继续编辑', exact: true }).click();
  await dismissNativeClose(page);
  await expect(name).toHaveValue('退出失败后的新草稿');
  expect((await read(page)).headSourceVersion).toBe(before.headSourceVersion);
  await page.unroute('**/api/v1/session');
  await page.getByRole('button', { name: '退出', exact: true }).click();
  await expect(unsaved(page)).toBeVisible();
  await unsaved(page).getByRole('button', { name: '放弃修改', exact: true }).click();
  await expect(page.getByRole('button', { name: '登录', exact: true })).toBeEnabled();
});

test('退出503后新建草稿重新保护，明确放弃只通过当次路由', async ({ page }) => {
  await open(page);
  await page.getByRole('link', { name: '新建', exact: true }).click();
  const name = page.getByRole('textbox', { name: '中文名称', exact: true });
  await name.fill('退出前的新建草稿');
  await failLogout(page, '503');
  await name.fill('退出失败后的新建草稿');
  await page.getByRole('link', { name: '取消', exact: true }).click();
  await expect(unsaved(page)).toBeVisible();
  await unsaved(page).getByRole('button', { name: '继续编辑', exact: true }).click();
  await dismissNativeClose(page);
  await expect(name).toHaveValue('退出失败后的新建草稿');
  await page.getByRole('link', { name: '取消', exact: true }).click();
  await unsaved(page).getByRole('button', { name: '放弃修改', exact: true }).click();
  await expect(page.getByRole('heading', { name: '业务单元', exact: true })).toBeVisible();
  await page.getByRole('link', { name: '新建', exact: true }).click();
  await name.fill('返回后第三份草稿');
  await page.getByRole('link', { name: '取消', exact: true }).click();
  await expect(unsaved(page)).toBeVisible();
  await unsaved(page).getByRole('button', { name: '放弃修改', exact: true }).click();
  await expect(page.getByRole('heading', { name: '业务单元', exact: true })).toBeVisible();
});

test('301确认期间禁止原内容重试，成功后可开始新草稿', async ({ page }) => {
  await deferAutomaticConfirmation(page);
  await open(page); await create(page);
  await openMoreActions(page); await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
  await page.getByRole('textbox', { name: '中文名称', exact: true }).fill('第一次已保存');
  let first = true;
  await page.route('**/api/v1/business-units/*/records/*/correct', async route => {
    const response = await route.fetch();
    if (first) { first = false; expect(response.status()).toBe(200); await route.abort('connectionreset'); }
    else await route.fulfill({ response });
  });
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.getByRole('button', { name: '重试保存', exact: true })).toBeVisible();
  let release!: () => void, started = false;
  const held = new Promise<void>(resolve => { release = resolve; });
  await page.route('**/api/v1/organization/submissions/*/resolve', async route => {
    const response = await route.fetch();
    started = true; await held;
    await route.fulfill({ response });
  });
  await page.getByRole('button', { name: '查看保存结果', exact: true }).click();
  await expect.poll(() => started).toBe(true);
  await expect(page.getByRole('button', { name: '重试保存', exact: true })).toBeDisabled();
  release();
  await expect(page.locator('.object-form')).toHaveCount(0);
  const saved = await read(page); expect(saved.selectedRecord.name).toBe('第一次已保存');
  await openMoreActions(page); await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
  await page.getByRole('textbox', { name: '中文名称', exact: true }).fill('第二次尚未保存');
  await expect(page.getByRole('textbox', { name: '中文名称', exact: true })).toHaveValue('第二次尚未保存');
  await expect(page.getByText('迟到查回拒绝', { exact: true })).toHaveCount(0);
  await expect(page.getByRole('button', { name: '查看保存结果', exact: true })).toHaveCount(0);
  expect((await read(page)).headSourceVersion).toBe(saved.headSourceVersion);
});
