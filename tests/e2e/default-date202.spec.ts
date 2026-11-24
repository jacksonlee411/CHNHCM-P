import { randomUUID } from 'node:crypto';
import { test, expect } from './fixtures';
import { open, choose } from './bu-support';
import { menu, objects, reference, request, readObject, fillNew } from './dated-object-support';

// These journeys include real fixture writes, error recovery and a second form.
// Use the existing department-journey budget; API latency is not measured here.
test.describe.configure({ timeout: 90_000 });

for (const kind of objects) {
  test(`${kind.label}202 保存拒绝后保留显式日期并可修正提交`, async ({ page }) => {
    await open(page);
    const code = 'DATE_CONFLICT_' + randomUUID().slice(0, 8);
    const buCode = 'DATE_OWNER_' + randomUUID().slice(0, 8);
    const bu = await request(page, 'POST', '/api/v1/business-units', { code: buCode, name: buCode, effectiveDate: '1900-01-01', operationType: 'CREATE', operationReason: 'NEW_ORGANIZATION' }, randomUUID());
    expect(bu.status).toBe(201);
    const path = `/api/v1/${kind.segment}`;
    const body = { code, name: '已存在的编码', effectiveDate: '1900-01-01', operationType: 'CREATE', operationReason: 'NEW_ORGANIZATION', ...(kind.segment === 'departments' ? { managementBuId: bu.body.buId } : {}) };
    expect((await request(page, 'POST', path, body, randomUUID())).status).toBe(201);
    await menu(page, kind);
    await fillNew(page, kind, { code, date: '2001-01-01', ...(kind.segment === 'departments' ? { bu: buCode } : {}) });
    const rejected = page.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith(path));
    await page.getByRole('button', { name: '保存', exact: true }).click();
    expect((await rejected).status()).toBe(409);
    await expect(page.locator(`#${kind.prefix}-effectiveDate`)).toHaveValue('2001-01-01');
    await page.getByRole('textbox', { name: '编码', exact: true }).fill('DATE_FIXED_' + randomUUID().slice(0, 8));
    await page.getByRole('button', { name: '保存', exact: true }).click();
    await expect(page.locator('.object-form')).toHaveCount(0);
    expect((await readObject(page, kind)).selectedRecord.effectiveDate).toBe('2001-01-01');
  });

  test(`${kind.label}202 默认日期经正常菜单创建并保留原提交`, async ({ page }) => {
    await open(page);
    const buCode = 'DEFAULT_BU_' + randomUUID().slice(0, 8);
    if (kind.segment === 'departments') {
      const r = await request(page, 'POST', '/api/v1/business-units', { code: buCode, name: buCode, effectiveDate: '1900-01-01', operationType: 'CREATE', operationReason: 'NEW_ORGANIZATION' }, randomUUID());
      expect(r.status).toBe(201);
    }
    await menu(page, kind);
    await page.getByRole('link', { name: '新建', exact: true }).click();
    await expect(page.locator(`#${kind.prefix}-effectiveDate`)).toHaveValue('1900-01-01');
    await page.getByRole('textbox', { name: '编码', exact: true }).fill('DEFAULT_' + randomUUID().slice(0, 8));
    await page.getByRole('textbox', { name: kind.name, exact: true }).fill('初始默认日期验证');
    if (kind.segment === 'departments') await reference(page, '管理业务单元', buCode);
    await choose(page, '操作原因', '新设组织');
    const saved = page.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith(`/api/v1/${kind.segment}`));
    await page.getByRole('button', { name: '保存', exact: true }).click();
    const response = await saved;
    expect(response.status()).toBe(201);
    const result = await response.json();
    await expect(page.locator('.object-form')).toHaveCount(0);
    expect((await readObject(page, kind)).selectedRecord.effectiveDate).toBe('1900-01-01');
    const resolve = kind.segment === 'departments' ? 'departments' : 'organization';
    const recovered = await request(page, 'POST', `/api/v1/${resolve}/submissions/${result.submissionId}/resolve`);
    expect(recovered.status).toBe(200);
    expect(recovered.body.sourceVersion).toBe(result.sourceVersion);
  });

  test(`${kind.label}202 日期手工清空及改日不被辅助重读或继续编辑覆盖`, async ({ page }) => {
    await open(page);
    await menu(page, kind);
    const reasonPath = `**/api/v1/${kind.segment === 'departments' ? 'departments' : 'organization'}/operation-reasons*`;
    // Only assistance failure is injected. Retrying context and reasons uses
    // the real API; this case asserts draft ownership, not a backend result.
    await page.route(reasonPath, route => route.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ code: 'TEST_ASSISTANCE_FAILURE', message: '辅助读取暂不可用' }) }));
    await page.getByRole('link', { name: '新建', exact: true }).click();
    const date = page.locator(`#${kind.prefix}-effectiveDate`);
    await expect(date).toHaveValue('1900-01-01');
    await expect(page.getByRole('button', { name: '重试加载', exact: true })).toBeVisible();
    await date.fill('');
    await page.unroute(reasonPath);
    await page.getByRole('button', { name: '重试加载', exact: true }).click();
    await expect(page.getByRole('button', { name: '重试加载', exact: true })).toHaveCount(0);
    await expect(date).toHaveValue('');
    await date.fill('1984-01-01');
    await page.getByRole('link', { name: '取消', exact: true }).click();
    await page.getByRole('button', { name: '继续编辑', exact: true }).click();
    await expect(date).toHaveValue('1984-01-01');
    await page.getByRole('link', { name: '取消', exact: true }).click();
    await page.getByRole('button', { name: '放弃修改', exact: true }).click();
    await page.getByRole('link', { name: '新建', exact: true }).click();
    await expect(date).toHaveValue('1900-01-01');
  });
}
