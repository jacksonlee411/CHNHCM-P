import { test, expect, openMoreActions, type Page } from './fixtures';
import { open, create, add, read, select, enter, save, choose, readyToSave } from './bu-support';
import { accessFixture } from './operation-access';
import { randomUUID } from 'node:crypto';

test.setTimeout(150_000);
const basic = (page: Page) => page.getByRole('tab', { name: '基本资料', exact: true });
// Positive reads target the active basic panel; revocation checks remain page-wide.
const basicEnglishName = (page: Page) => page.getByRole('tabpanel', { name: '基本资料', exact: true }).getByText('Alpha', { exact: true });
async function logs(page: Page) {
  await page.getByRole('tab', { name: '变更日志', exact: true }).click();
  await expect(page.locator('.change-operation').first()).toBeVisible();
  await expect(page.getByRole('region', { name: '业务单元详情正文' })).toHaveAttribute('data-page-scroll-ready', 'true');
}
async function displayed(page: Page) {
  return page.evaluate(async () => {
    const url = new URL(location.href), query = new URLSearchParams();
    for (const key of ['sourceVersion', 'selectedRecordId']) if (url.searchParams.has(key)) query.set(key, url.searchParams.get(key)!);
    const result = await fetch('/api/v1/business-units/' + url.pathname.split('/').at(-1) + '?' + query);
    if (!result.ok) throw new Error('displayed ' + result.status);
    return result.json();
  });
}
async function rename(page: Page, value: string) {
  await enter(page, '更正记录'); await page.getByRole('textbox', { name: '中文名称', exact: true }).fill(value); return save(page);
}
async function remove(page: Page) {
  await enter(page, '删除此记录'); await readyToSave(page);
  const response = page.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith('/delete'));
  await page.getByRole('button', { name: '确认删除', exact: true }).click();
  const result = await response; expect(result.status()).toBe(200);
  await expect(page.locator('.object-form')).toHaveCount(0); return result.json();
}
async function openFromList(page: Page, code: string) {
  await page.getByRole('textbox', { name: '搜索编码或名称', exact: true }).fill(code);
  await page.getByRole('textbox', { name: '搜索编码或名称', exact: true }).press('Enter');
  await page.getByRole('row').filter({ hasText: code }).getByRole('link').click();
  await expect(page.locator('.operation-summary')).toBeVisible();
}
// Two real sessions and two distinct subjects; only disposable fixture grants change.
async function secondWriter(admin: Page, reader: Page) {
  const a = await accessFixture(admin); await open(reader, 'reader');
  reader.setDefaultTimeout(15_000); const b = await accessFixture(reader);
  b.mutate(`DELETE FROM access.subject_grant WHERE ${b.where};INSERT INTO access.subject_grant SELECT instance_id,gen_random_uuid(),'${b.session.subject}',scope_kind,scope_id,action,read_fields,write_fields FROM access.subject_grant WHERE ${a.where}`);
  expect(a.session.subject).not.toBe(b.session.subject);
  return { a, b };
}
async function original(page: Page, url: string, expected: { recordId: string; revision: string; source: string; actor: string; login: string; name: string }, deleted: boolean) {
  await page.goto(url);
  await expect(page.getByText('正在查看只读历史资料', { exact: true })).toBeVisible();
  await expect(page.locator('.operation-summary')).toContainText('操作者：' + expected.login);
  const detail = await displayed(page), record = detail.selectedRecord;
  expect(record.recordId).toBe(expected.recordId); expect(Boolean(record.deleted)).toBe(deleted); expect(record.name).toBe(expected.name);
  expect(record.operationSummary).toMatchObject({ recordId: expected.recordId, revision: expected.revision, operationSourceVersion: expected.source, operator: { subjectId: expected.actor, login: expected.login } });
  if (deleted) { expect(record.revision).not.toBe(expected.revision); await expect(page.getByText('已删除记录原件，只读。下方保留删除前内容与期间。', { exact: true })).toBeVisible(); }
  else expect(record.revision).toBe(expected.revision);
}
async function deniedOriginals(page: Page, rights: Awaited<ReturnType<typeof accessFixture>>, urls: string[]) {
  rights.mutate(`DELETE FROM access.subject_grant WHERE ${rights.where} AND action='bu.history.read'`);
  for (const url of urls) {
    await page.goto(url); await expect(page.getByText('当前权限不允许读取历史', { exact: true })).toBeVisible();
    await expect(page.locator('.operation-summary')).toHaveCount(0); await expect(page.locator('.change-operation')).toHaveCount(0);
  }
  await page.getByRole('button', { name: '返回当前资料', exact: true }).click();
  await expect(page.locator('.operation-summary')).toBeVisible();
  await expect(page.getByRole('tab', { name: '变更日志', exact: true })).toHaveCount(0);
}

test('122复修 TL375/380 成功200收窄权限清除详情缓存，普通与历史读权独立', async ({ page }) => {
  await open(page); await create(page, '9998-11-10'); const rights = await accessFixture(page);
  try {
    await expect(basicEnglishName(page)).toBeVisible(); await logs(page);
    rights.mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'nameEn') WHERE ${rights.where} AND action='bu.history.read'`);
    const narrowedLog = page.waitForResponse(r => r.request().method() === 'GET' && new URL(r.url()).pathname === '/api/v1/business-units/' + new URL(page.url()).pathname.split('/').at(-1) + '/changes');
    await page.getByRole('button', { name: '刷新日志', exact: true }).click();
    const narrowedResponse = await narrowedLog; expect(narrowedResponse.status()).toBe(200); await narrowedResponse.finished();
    await expect(page.getByRole('region', { name: '业务单元详情正文' })).toHaveAttribute('data-page-scroll-ready', 'true');
    await expect(page.locator('.change-operation')).toHaveCount(1);
    await basic(page).click(); await expect(basicEnglishName(page)).toBeVisible();
    await logs(page);
    // The intersection is unchanged here; using only log.readFields would miss this revocation.
    rights.mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'nameEn') WHERE ${rights.where} AND action='bu.read'`);
    const reloaded = page.waitForResponse(r => r.request().method() === 'GET' && new URL(r.url()).pathname === '/api/v1/business-units/' + new URL(page.url()).pathname.split('/').at(-1));
    await page.getByRole('button', { name: '刷新日志', exact: true }).click(); expect((await reloaded).status()).toBe(200);
    await expect(page.locator('.change-operation')).toHaveCount(1);
    await expect(page.getByText('Alpha', { exact: true })).toHaveCount(0);
    await basic(page).click(); await expect(basic(page)).toHaveAttribute('aria-selected', 'true');
    await expect(page.getByText('英文名称', { exact: true })).toHaveCount(0); await expect(page.locator('.operation-summary')).toBeVisible();
    await logs(page); const old = await page.getByRole('link', { name: '查看当次资料', exact: true }).getAttribute('href');
    rights.restore(); await page.goto(old!); await expect(basicEnglishName(page)).toBeVisible(); await logs(page);
    rights.mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'nameEn') WHERE ${rights.where} AND action='bu.history.read'`);
    // A count of one can still be the pre-refresh row. Wait for the actual
    // narrowed historical detail and the subsequent log read to settle.
    const historyReloaded = page.waitForResponse(r => r.request().method() === 'GET' && new URL(r.url()).pathname === '/api/v1/business-units/' + new URL(page.url()).pathname.split('/').at(-1) && new URL(r.url()).searchParams.has('sourceVersion'));
    await page.getByRole('button', { name: '刷新日志', exact: true }).click(); expect((await historyReloaded).status()).toBe(200);
    await expect(page.getByRole('region', { name: '业务单元详情正文' })).toHaveAttribute('data-page-scroll-ready', 'true');
    await expect(page.locator('.change-operation')).toHaveCount(1);
    await basic(page).click(); await expect(basic(page)).toHaveAttribute('aria-selected', 'true'); await expect(page.getByText('Alpha', { exact: true })).toHaveCount(0);
    await page.getByRole('button', { name: '返回当前资料', exact: true }).click(); await expect(basicEnglishName(page)).toBeVisible();
  } finally { rights.restore(); }
});

test('122复修 TL376/382/383 键盘翻页、页大小、筛选、空结果及失败焦点', async ({ page }) => {
  await open(page); await create(page); const d = await displayed(page);
  await page.evaluate(async d => {
    const session = await (await fetch('/api/v1/session')).json(); let source = d.headSourceVersion, revision = d.selectedRecord.revision;
    for (let n = 2; n <= 45; n++) {
      const r = await fetch(`/api/v1/business-units/${d.id}/records/${d.selectedRecord.recordId}/correct`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': session.csrf, 'Idempotency-Key': crypto.randomUUID() }, body: JSON.stringify({ expectedSourceVersion: source, expectedRecordRevision: revision, changes: { name: '焦点样本' + n } }) });
      if (!r.ok) throw new Error('fixture ' + r.status); const saved = await r.json(); source = saved.sourceVersion; revision = saved.targetRevision;
    }
  }, d);
  await logs(page);
  const next = page.getByRole('button', { name: '下一页', exact: true }), previous = page.getByRole('button', { name: '上一页', exact: true });
  await next.focus(); await page.keyboard.press('Enter'); await expect(page.getByText('21–40 / 45 项操作', { exact: true })).toBeVisible(); await expect(next).toBeFocused();
  await page.keyboard.press('Enter'); await expect(page.getByText('41–45 / 45 项操作', { exact: true })).toBeVisible();
  // The last page disables Next; the toolbar is the reachable fallback.
  await expect(page.getByRole('button', { name: '列筛选…', exact: true })).toBeFocused();
  await previous.focus(); await page.keyboard.press('Enter'); await expect(page.getByText('21–40 / 45 项操作', { exact: true })).toBeVisible(); await expect(previous).toBeFocused();
  await page.keyboard.press('Enter'); await expect(page.getByText('1–20 / 45 项操作', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: '列筛选…', exact: true })).toBeFocused();
  const size = page.locator('.change-log').getByRole('combobox');
  await size.focus(); await page.keyboard.press('Enter'); await page.getByRole('option', { name: '50', exact: true }).press('Enter');
  await expect(page.getByText('1–45 / 45 项操作', { exact: true })).toBeVisible(); await expect(size).toBeFocused();
  const actorMenu = page.getByRole('button', { name: '操作者列菜单', exact: true });
  await actorMenu.focus(); await page.keyboard.press('Enter'); await page.getByRole('menuitem', { name: '筛选…', exact: true }).press('Enter');
  await page.getByRole('dialog').getByRole('textbox', { name: '值', exact: true }).fill('不存在的操作员');
  await page.getByRole('dialog').getByRole('button', { name: '应用', exact: true }).press('Enter');
  await expect(page.getByText('没有符合条件的变更日志', { exact: true })).toBeVisible(); await expect(actorMenu).toBeFocused();
  const filter = page.getByRole('button', { name: '列筛选…', exact: true });
  await filter.press('Enter'); await page.getByRole('dialog').getByRole('button', { name: '清除条件', exact: true }).press('Enter');
  await page.getByRole('dialog').getByRole('button', { name: '应用', exact: true }).press('Enter');
  await expect(page.getByText('1–45 / 45 项操作', { exact: true })).toBeVisible(); await expect(filter).toBeFocused();
  const pattern = '**/api/v1/business-units/*/changes?*';
  await page.route(pattern, route => route.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ code: 'DEPENDENCY_UNAVAILABLE', message: '合成读取失败' }) }));
  await page.getByRole('button', { name: '刷新日志', exact: true }).press('Enter');
  await expect(page.getByRole('button', { name: '重试', exact: true })).toBeFocused(); await expect(page.locator('.change-operation')).toHaveCount(0);
  await page.unroute(pattern); await page.getByRole('button', { name: '重试', exact: true }).press('Enter');
  await expect(page.getByText('1–45 / 45 项操作', { exact: true })).toBeVisible(); await expect(filter).toBeFocused();
  let release!: () => void, captured = false; const held = new Promise<void>(r => { release = r; });
  await page.route(pattern, async route => { const response = await route.fetch(); captured = true; await held; await route.fulfill({ response }); });
  await page.getByRole('button', { name: '刷新日志', exact: true }).press('Enter'); await expect.poll(() => captured).toBe(true);
  await basic(page).focus(); release(); await expect(page.getByText('1–45 / 45 项操作', { exact: true })).toBeVisible(); await expect(basic(page)).toBeFocused(); await page.unroute(pattern);
});

test('122复修 TL373/381 P固定两条不同操作者：前来源、墓碑原件、复制链接及撤权', async ({ page, browser }) => {
  page.setDefaultTimeout(15_000);
  await open(page); const code = await create(page); const context = await browser.newContext(), reader = await context.newPage();
  let actors: Awaited<ReturnType<typeof secondWriter>> | undefined;
  try {
    actors = await secondWriter(page, reader);
    await openMoreActions(page); await page.getByRole('menuitem', { name: '插入停用与重新启用', exact: true }).click();
    const dialog = page.getByRole('dialog'); await dialog.getByLabel('停用日期', { exact: true }).fill('2026-06-01'); await dialog.getByLabel('重新启用日期', { exact: true }).fill('2026-09-01');
    await choose(dialog, '停用原因', '机构撤并'); await expect(dialog.locator('#pair-reactivationReason')).toHaveText('业务恢复');
    const inserted = page.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith('/lifecycle-pairs'));
    await dialog.getByRole('button', { name: '保存', exact: true }).click(); const p2 = await (await inserted).json(); await expect(dialog).toHaveCount(0);
    const stop = (await read(page)).timeline.find((r: any) => r.effectiveDate === '2026-06-01');
    await openFromList(reader, code); await select(reader, '2026-09-01'); const p3 = await rename(reader, '恢复更正'); const resume = (await displayed(reader)).selectedRecord;
    await page.reload(); await select(page, '2026-06-01'); await openMoreActions(page); await page.getByRole('menuitem', { name: '删除停用与重新启用', exact: true }).click();
    const deletion = page.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith('/lifecycle-pairs/delete'));
    await page.getByRole('dialog').getByRole('button', { name: '删除两条记录', exact: true }).click(); const p4 = await (await deletion).json(); await expect(page.getByRole('dialog')).toHaveCount(0);
    await logs(page); const group = page.locator(`.change-operation[data-source-version="${p4.sourceVersion}"]`); await expect(group.locator('[data-label="操作者"]')).toHaveText('admin');
    const copied: string[] = [];
    for (const [record, source, actor, login] of [[stop, p2.sourceVersion, actors.a.session.subject, 'admin'], [resume, p3.sourceVersion, actors.b.session.subject, 'reader']] as const) {
      const link = group.locator(`a[id$="-${record.recordId}-before"]`); const href = (await link.getAttribute('href'))!; copied.push(href);
      await link.click(); await expect(page.locator('.operation-summary')).toContainText('操作者：' + login);
      const expected = { recordId: record.recordId, revision: record.revision, source, actor, login, name: record.name };
      await original(page, href, expected, false);
      const tombstone = new URL(page.url()); tombstone.searchParams.set('sourceVersion', p4.sourceVersion); copied.push(tombstone.href);
      await original(page, tombstone.href, expected, true);
      const value = (await displayed(page)).selectedRecord;
      for (const key of ['effectiveDate', 'endExclusive', 'operationType', 'operationReason', 'typeLabel', 'reasonLabel']) expect(value[key]).toEqual(record[key]);
      await page.getByRole('link', { name: '返回变更日志', exact: true }).click(); await expect(group).toBeVisible();
    }
    const after = group.getByRole('link', { name: '查看删除后版本', exact: true }).first(); const href = (await after.getAttribute('href'))!; copied.push(href); expect(href).not.toContain('selectedRecordId=');
    await after.click(); expect((await displayed(page)).timeline).toHaveLength(1);
    await deniedOriginals(page, actors.a, copied);
  } finally { actors?.a.restore(); actors?.b.restore(); await context.close().catch(() => {}); }
});

test('122复修 TL367/373/375/381 F未来首条删除后默认摘要、两类原件和缺失选择', async ({ page, browser }) => {
  page.setDefaultTimeout(15_000);
  await open(page); const code = await create(page, '9998-11-10'); await add(page, '9998-12-01', '未来乙'); const f2 = await displayed(page), r2 = f2.selectedRecord;
  const context = await browser.newContext(), reader = await context.newPage(); let actors: Awaited<ReturnType<typeof secondWriter>> | undefined;
  try {
    actors = await secondWriter(page, reader); await openFromList(reader, code); await select(reader, '9998-12-01'); const f3 = await remove(reader);
    await page.reload(); await logs(page); const group = page.locator(`.change-operation[data-source-version="${f3.sourceVersion}"]`);
    const before = (await group.getByRole('link', { name: '查看删除前原件', exact: true }).getAttribute('href'))!;
    const after = (await group.getByRole('link', { name: '查看删除后版本', exact: true }).getAttribute('href'))!;
    expect(after).not.toContain('selectedRecordId='); expect(after).not.toContain('businessAt');
    await group.getByRole('link', { name: '查看删除后版本', exact: true }).click();
    await expect(page.getByText('所选记录尚未生效。', { exact: true })).toBeVisible(); await expect(page.locator('.operation-summary')).toContainText('操作者：reader');
    const afterDetail = await displayed(page); expect(afterDetail.result).toBe('M02_NO_FACT'); expect(afterDetail.viewFact).toBeNull(); expect(afterDetail.timeline).toHaveLength(1);
    expect(afterDetail.selectedRecord.name).toBe('甲'); expect(afterDetail.selectedRecord.operationSummary).toMatchObject({ operationSourceVersion: f3.sourceVersion, operator: { subjectId: actors.b.session.subject } });
    const expected = { recordId: r2.recordId, revision: r2.revision, source: f2.headSourceVersion, actor: actors.a.session.subject, login: 'admin', name: '未来乙' };
    await original(page, before, expected, false); const tombstone = new URL(page.url()); tombstone.searchParams.set('sourceVersion', f3.sourceVersion); await original(page, tombstone.href, expected, true);
    const missing = new URL(page.url()); missing.searchParams.set('selectedRecordId', randomUUID()); await page.goto(missing.href);
    await expect(page.getByText('当前资料中没有所选记录，请重新选择。', { exact: true })).toBeVisible(); await expect(page.locator('.operation-summary')).toHaveCount(0); expect((await displayed(page)).selectedRecord).toBeNull();
    actors.a.mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'effectiveDate') WHERE ${actors.a.where} AND action='bu.read'`);
    for (const url of [after, '/settings/organization/business-units/' + f2.id]) {
      await page.goto(url); await expect(page.locator('.operation-summary')).toContainText('操作者：reader');
      await expect(page.getByText('所选记录尚未生效。', { exact: true })).toHaveCount(0); await expect(page.getByText('生效日期', { exact: true })).toHaveCount(0); await expect(page.getByText('适用期间', { exact: true })).toHaveCount(0);
    }
    await deniedOriginals(page, actors.a, [before, tombstone.href, after]);
  } finally { actors?.a.restore(); actors?.b.restore(); await context.close().catch(() => {}); }
});

test('122复修 TL373/381 S同日新身份不串两类旧原件，空变化不造记录', async ({ page, browser }) => {
  await open(page); const code = await create(page), current = new URL(page.url()); current.search = '';
  const context = await browser.newContext(), reader = await context.newPage(); let actors: Awaited<ReturnType<typeof secondWriter>> | undefined;
  try {
    actors = await secondWriter(page, reader); await openFromList(reader, code); await add(reader, '2026-06-01', '乙');
    await page.reload(); await add(page, '2026-10-01', '丙');
    await reader.reload(); await select(reader, '2026-10-01'); await rename(reader, '丙新');
    await page.reload(); await select(page, '2026-06-01'); const s5 = await rename(page, '乙新'), r2 = (await displayed(page)).selectedRecord;
    await reader.reload(); await select(reader, '2026-06-01'); const s6 = await remove(reader);
    await page.reload(); await logs(page); const group = page.locator(`.change-operation[data-source-version="${s6.sourceVersion}"]`);
    await expect(group.locator('[data-label="操作者"]')).toHaveText('reader');
    const before = (await group.getByRole('link', { name: '查看删除前原件', exact: true }).getAttribute('href'))!;
    const after = (await group.getByRole('link', { name: '查看删除后版本', exact: true }).getAttribute('href'))!;
    const expected = { recordId: r2.recordId, revision: r2.revision, source: s5.sourceVersion, actor: actors.a.session.subject, login: 'admin', name: '乙新' };
    await group.getByRole('link', { name: '查看删除前原件', exact: true }).click(); await original(page, before, expected, false);
    const tombstone = new URL(page.url()); tombstone.searchParams.set('sourceVersion', s6.sourceVersion); await original(page, tombstone.href, expected, true);
    await page.goto(after); await expect(page.getByText('正在查看只读历史资料', { exact: true })).toBeVisible(); expect((await displayed(page)).timeline).toHaveLength(2);
    await page.goto(current.href); await expect(page.locator('.operation-summary')).toBeVisible();
    const noChange = await page.evaluate(async () => {
      const path = '/api/v1/business-units/' + location.pathname.split('/').at(-1), d = await (await fetch(path)).json(), s = await (await fetch('/api/v1/session')).json();
      const response = await fetch(path + '/records', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': s.csrf, 'Idempotency-Key': crypto.randomUUID() }, body: JSON.stringify({ expectedSourceVersion: d.headSourceVersion, anchorRecordId: d.timeline[0].recordId, effectiveDate: '2026-06-01', operationType: 'CHANGE', operationReason: 'NAME_ADJUSTMENT', changes: {} }) });
      if (!response.ok) throw new Error('empty addition ' + response.status); return response.json();
    });
    expect(noChange.outcome).toBe('no_change'); expect((await displayed(page)).headSourceVersion).toBe(s6.sourceVersion); expect((await displayed(page)).timeline).toHaveLength(2);
    await add(page, '2026-06-01', '同日新登记'); const r4 = (await displayed(page)).selectedRecord; expect(r4.recordId).not.toBe(r2.recordId);
    await original(page, before, expected, false); await original(page, tombstone.href, expected, true);
    await deniedOriginals(page, actors.a, [before, tombstone.href, after]);
  } finally { actors?.a.restore(); actors?.b.restore(); await context.close().catch(() => {}); }
});

test('R4 历史日志切回基本资料后立即返回当前，保留最后选择的页签', async ({ page }) => {
  await open(page); await create(page, '9998-11-10'); await logs(page);
  const old = await page.getByRole('link', { name: '查看当次资料', exact: true }).getAttribute('href');
  await page.goto(old!); await expect(basicEnglishName(page)).toBeVisible(); await logs(page);
  // Two actions can arrive while React Router's transition has not rendered.
  // Use real DOM actions to make the navigation ordering deterministic.
  const observed = await page.evaluate(() => {
    const basic = [...document.querySelectorAll<HTMLElement>('[role="tab"]')].find(v => v.textContent === '基本资料')!;
    const current = [...document.querySelectorAll<HTMLButtonElement>('button')].find(v => v.textContent === '返回当前资料')!;
    basic.click(); const afterTab = location.search; current.click();
    return { afterTab, afterReturn: location.search };
  });
  expect(new URLSearchParams(observed.afterTab).get('tab')).toBe('basic');
  await expect(page).not.toHaveURL(/sourceVersion=/);
  await expect(basic(page)).toHaveAttribute('aria-selected', 'true');
  await expect(basicEnglishName(page)).toBeVisible();
  console.log(JSON.stringify({ event: 'r4_current_navigation', ...observed }));
});
