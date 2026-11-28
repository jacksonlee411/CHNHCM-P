import { randomUUID } from 'node:crypto';
import { open, select } from './bu-support';
import { request } from './dated-object-support';
import { expect, openMoreActions, type Page } from './fixtures';
import { nativeZoomBrowser } from './native-zoom-support';

export async function checkMiddleDelete(
  page: Page, segment: string, reason: string,
  create: (page: Page, country: string) => Promise<string>, menu: (page: Page) => Promise<void>,
) {
  await open(page);
  const code = await create(page, '澳门');
  const path = new URL(page.url()).pathname;
  const api = '/api/v1/' + segment + '/' + path.split('/').at(-1);
  const read = async () => (await request(page, 'GET', api)).body;
  let detail = await read();
  for (const date of ['2021-01-01', '2022-01-01', '2023-01-01']) {
    const result = await request(page, 'POST', api + '/records', {
      expectedSourceVersion: detail.headSourceVersion, anchorRecordId: detail.selectedRecord.recordId,
      effectiveDate: date, operationType: 'CHANGE', operationReason: reason, changes: { name: code + ' ' + date },
    }, randomUUID());
    expect(result.status).toBe(201);
    detail = await read();
  }
  const [, second, removed] = detail.timeline;
  const deletion = api + '/records/' + removed.recordId + '/delete';
  async function editDelete(p: Page, compact = false) {
    if (compact) {
      await p.getByRole('button', {name: /选择生效日期/}).click();
      await p.getByRole('menuitem', {name: '2022-01-01', exact: true}).click();
      await expect(p.getByRole('button', {name: /选择生效日期/})).toContainText('2022-01-01');
    } else {
      await select(p, '2022-01-01');
    }
    await openMoreActions(p);
    await p.getByRole('menuitem', {name: '删除此记录', exact: true}).click();
    await expect(p.getByText('2022-01-01 至 2022-12-31', {exact: true})).toBeVisible();
  }
  await page.goto(path);
  let deleteRequests = 0;
  page.on('request', req => { if (req.method() === 'POST' && new URL(req.url()).pathname === deletion) deleteRequests++; });
  await editDelete(page);
  await page.getByRole('button', {name: '取消编辑', exact: true}).click();
  await expect(page.locator('.object-form')).toHaveCount(0);
  expect((await read()).headSourceVersion).toBe(detail.headSourceVersion);
  expect(deleteRequests).toBe(0);

  const zoom = await nativeZoomBrowser();
  try {
    const p = zoom.page;
    await open(p);
    await p.goto(path);
    await zoom.setZoom(2);
    await editDelete(p, true);
    const submitted = p.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === deletion);
    await p.getByRole('button', {name: '确认删除', exact: true}).press('Enter');
    const response = await submitted;
    expect(response.status()).toBe(200);
    const receipt = await response.json();
    expect(receipt).toMatchObject({targetRecordId: removed.recordId, targetRevision: removed.revision,
      effectiveDate: '2022-01-01', selectedRecordId: second.recordId});
    const dates = p.getByRole('button', {name: /选择生效日期/});
    await expect(dates).toContainText('2021-01-01');
    await expect(dates).toHaveAttribute('aria-current', 'date');
    expect(new URL(p.url()).searchParams.get('selectedRecordId')).toBe(second.recordId);
    await dates.click();
    await expect(p.getByRole('menuitem', {name: '2022-01-01', exact: true})).toHaveCount(0);
    await expect(p.getByRole('menuitem', {name: '2023-01-01', exact: true})).toBeVisible();
    await p.keyboard.press('Escape');
    const recovered = await request(p, 'POST', '/api/v1/' + segment + '/submissions/' + receipt.submissionId + '/resolve');
    expect(recovered.status).toBe(200);
    expect(recovered.body).toMatchObject({sourceVersion: receipt.sourceVersion, targetRevision: removed.revision, selectedRecordId: second.recordId});
  } finally {
    await zoom.close();
  }

  await page.setViewportSize({width: 1440, height: 900});
  await page.context().clearCookies();
  await open(page, 'reader');
  await menu(page);
  await page.getByRole('textbox', {name: '搜索编码或名称', exact: true}).fill(code);
  await page.getByRole('button', {name: '查询', exact: true}).click();
  await page.getByRole('link', {name: code + ' 2023-01-01', exact: true}).click();
  await page.getByRole('tab', {name: '变更日志', exact: true}).click();
  await page.getByRole('link', {name: '查看删除前原件', exact: true}).click();
  expect(new URL(page.url()).searchParams.get('selectedRecordId')).toBe(removed.recordId);
  expect(new URL(page.url()).searchParams.get('sourceVersion')).toBe(detail.headSourceVersion);
  await expect(page.getByText(code + ' 2022-01-01', {exact: true}).first()).toBeVisible();
  await expect(page.getByRole('button', {name: '新增记录', exact: true})).toHaveCount(0);
}
