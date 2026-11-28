import { staffingPageSuite } from './staffing-page-suite';
import { matrixJourney309 } from './staffing309-journey';
staffingPageSuite(['grade'], '309', matrixJourney309);
import { test, expect } from './fixtures';
import { open } from './bu-support';
import { randomUUID } from 'node:crypto';
import { desktopViewports } from './desktop-viewports.mjs';
import { withPolicy } from './grants';
import { request } from './dated-object-support';
import { seed308, enter308, action308, click308, menu308 } from './staffing308-support';
test('309 grade 新建等待操作原因后一次保存', async ({ page }) => {
  const [width, height] = desktopViewports[0];
  await page.setViewportSize({ width, height });
  await open(page);
  await menu308(page, 'grade');
  const code = 'G309-' + randomUUID().slice(0, 8);
  const name = '延迟加载 ' + code;
  const save = page.getByRole('button', { name: '保存', exact: true });
  const reasonRequest = '**/api/v1/grades/operation-reasons?operationType=CREATE';
  let release!: () => void;
  const gate = new Promise<void>(resolve => { release = resolve; });
  await page.route(reasonRequest, async route => { await gate; await route.continue(); });
  try {
    await Promise.all([
      page.waitForRequest(reasonRequest),
      page.getByRole('link', { name: '新建', exact: true }).click(),
    ]);
    await page.getByRole('textbox', { name: '编码', exact: true }).fill(code);
    await page.getByRole('textbox', { name: '名称', exact: true }).fill(name);
    await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-01-01');
    await expect(save).toBeDisabled();
  } finally {
    release();
  }
  await expect(page.getByRole('combobox', { name: '操作原因', exact: true })).toHaveText('新设职级');
  const [created] = await Promise.all([
    page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === '/api/v1/grades'),
    save.click(),
  ]);
  expect(created.status()).toBe(201);
  await expect(page.getByRole('heading', { name, exact: true })).toBeVisible();
  const { objectId } = await created.json();
  expect((await request(page, 'GET', '/api/v1/grades/' + objectId + '/changes')).body.total).toBe(1);
});

  test(`309 grade 丢失保存响应自动查回且只写一份来源`, async ({ page }) => {
    await open(page);
    const o = { ...await seed308(page, 'grade'), path: '/api/v1/grades' };
    await enter308(page, o);
    await action308(page, '更正记录');
    await page.getByRole('textbox', { name: '名称', exact: true }).fill('已查回 ' + o.code);
    let writes = 0,
      resolves = 0;
    await page.route('**' + o.path + '/**', async (route) => {
      const r = route.request();
      if (r.method() === 'POST' && r.url().endsWith('/correct')) {
        writes++;
        await route.fetch();
        await route.abort('failed');
        return;
      }
      if (r.url().includes('/submissions/')) resolves++;
      await route.continue();
    });
    await click308(page, page.getByRole('button', { name: '保存', exact: true }));
    await expect(page.locator('.object-form')).toHaveCount(0);
    expect(writes).toBe(1);
    expect(resolves).toBe(1);
    const log = await request(page, 'GET', o.path + '/' + o.id + '/changes');
    expect(log.body.items).toHaveLength(2);
    await expect(page.getByText('已查回 ' + o.code, { exact: true }).first()).toBeVisible();
  });
  test(`309 grade 原提交安全关闭后保留草稿可重试`, async ({ page }) => {
    await open(page);
    const o = { ...await seed308(page, 'grade'), path: '/api/v1/grades' };
    await enter308(page, o);
    await action308(page, '更正记录');
    await page.getByRole('textbox', { name: '名称', exact: true }).fill('安全重试 ' + o.code);
    let first = true;
    await page.route('**' + o.path + '/**/correct', async (route) => {
      if (first) {
        first = false;
        await route.abort('failed');
      } else await route.continue();
    });
    await page.getByRole('button', { name: '保存', exact: true }).click();
    await expect(page.getByText('原提交未生效，请核对输入后重新保存。')).toBeVisible();
    await expect(page.getByRole('textbox', { name: '名称', exact: true })).toHaveValue('安全重试 ' + o.code);
    await click308(page, page.getByRole('button', { name: '保存', exact: true }));
    await expect(page.locator('.object-form')).toHaveCount(0);
    expect((await request(page, 'GET', o.path + '/' + o.id + '/changes')).body.items).toHaveLength(2);
  });

// Revoke actual grants while an existing form/history is already on screen.
test('309 grade 保存前撤权及历史页撤权清除旧资料', async ({ page }) => {
  await open(page);
  const o = await seed308(page, 'grade');
  await enter308(page, o);
  const path = '/api/v1/grades/' + o.id;
  const before = (await request(page, 'GET', path)).body;
  await action308(page, '更正记录');
  const englishName = page.getByRole('textbox', { name: '英文名称', exact: true });
  const allowedDraft = 'Draft ' + o.code;
  await englishName.fill(allowedDraft);
  await page.getByRole('textbox', { name: '名称', exact: true }).fill('不得写入');
  await withPolicy(page, async (where, mutate) => {
    mutate(`UPDATE access.subject_grant SET write_fields=array_remove(write_fields,'name') WHERE ${where} AND action='grade.record.correct'`);
    const refused = page.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith('/correct') && r.status() === 403);
    await click308(page, page.getByRole('button', { name: '保存', exact: true }));
    await refused;
    await expect(page.getByText('权限已变化，已重新读取当前可见资料。')).toBeVisible();
    await expect(englishName).toHaveValue(allowedDraft);
    await expect(page.getByRole('textbox', { name: '名称', exact: true })).toHaveCount(0);
    await expect(page.getByRole('tab', { name: '变更日志', exact: true })).toBeDisabled();
    await click308(page, page.getByRole('button', { name: '取消', exact: true }));
    await click308(page, page.getByRole('button', { name: '放弃修改', exact: true }));
    expect((await request(page, 'GET', path)).body.headSourceVersion).toBe(before.headSourceVersion);
  });
  await click308(page, page.getByRole('tab', { name: '变更日志', exact: true }));
  await click308(page, page.getByRole('link', { name: '查看当次资料', exact: true }));
  await expect(page.getByText('正在查看只读历史资料', { exact: true })).toBeVisible();
  await withPolicy(page, async (where, mutate) => {
    mutate(`DELETE FROM access.subject_grant WHERE ${where} AND action='grade.history.read'`);
    const denied = page.waitForResponse(r => r.url().includes('/grades/' + o.id + '?sourceVersion=') && r.status() === 403);
    await page.reload();
    await denied;
    await expect(page.getByRole('heading', { name: o.name, exact: true })).toHaveCount(0);
    await expect(page.locator('.object-detail-sheet')).toHaveCount(0);
    expect((await request(page, 'GET', path + '?sourceVersion=' + before.headSourceVersion)).status).toBe(403);
  });
});

test('309 grade 详情503清除旧资料且可原位重试', async ({ page }) => {
  await open(page);
  const o = await seed308(page, 'grade');
  await enter308(page, o);
  const pattern = '**/api/v1/grades/' + o.id;
  await page.route(pattern, route => route.fulfill({status: 503, json: {code: 'M03_DEPENDENCY_UNAVAILABLE', message: '职级读取测试故障'}}));
  await page.reload();
  await expect(page.getByText('职级读取测试故障', {exact: true})).toBeVisible();
  await expect(page.getByRole('heading', { name: o.name, exact: true })).toHaveCount(0);
  await page.unroute(pattern);
  await click308(page, page.getByRole('button', {name: '重试', exact: true}));
  await expect(page.getByRole('heading', { name: o.name, exact: true })).toBeVisible();
});

test('309 grade 未知结果离开刷新和换账号隔离后查回原提交', async ({ page }) => {
  await open(page);
  const f = { ...await seed308(page, 'grade'), web: '/settings/staffing/grades', path: '/api/v1/grades' };
  await page.goto(f.web + '/' + f.id);
  await action308(page, '更正记录');
  await page.getByRole('textbox', { name: '名称', exact: true }).fill('原提交恢复 ' + f.code);
  let writes = 0,
    resolves = 0;
  await page.route('**' + f.path + '/**/correct', async (route) => {
    writes++;
    await route.fetch();
    await route.abort('failed');
  });
  await page.route('**' + f.path + '/submissions/*/resolve', async (route) => {
    resolves++;
    await route.fulfill({
      status: 202,
      json: {
        kind: 'grade',
        submissionId: new URL(route.request().url()).pathname.split('/').at(-2),
        outcome: 'pending',
      },
    });
  });
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.getByText('保存结果尚未确认，请使用“查看保存结果”。')).toBeVisible();
  expect(writes).toBe(1);
  expect(resolves).toBe(1);
  await page.getByRole('button', { name: '退出', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: '保存结果尚未确认', exact: true });
  await expect(dialog).toContainText('离开不会撤销已提交的保存操作');
  await dialog.getByRole('button', { name: '留在此页', exact: true }).click();
  await page.getByRole('button', { name: '取消', exact: true }).click();
  await dialog.getByRole('button', { name: '离开此页', exact: true }).click();
  await page.reload();
  await expect(page.getByRole('button', { name: '查看保存结果', exact: true })).toBeEnabled();
  expect(resolves).toBe(1);
  await page.getByRole('button', { name: '退出', exact: true }).click();
  await open(page, 'reader');
  await expect(page.getByRole('button', { name: '查看保存结果', exact: true })).toHaveCount(0);
  await page.getByRole('button', { name: '退出', exact: true }).click();
  await open(page);
  await expect(page.getByRole('button', { name: '查看保存结果', exact: true })).toBeEnabled();
  await page.unroute('**' + f.path + '/submissions/*/resolve');
  await page.getByRole('button', { name: '查看保存结果', exact: true }).click();
  await expect(page.getByText('原提交已保存。')).toBeVisible();
  expect(writes).toBe(1);
  expect((await request(page, 'GET', f.path + '/' + f.id + '/changes')).body.items).toHaveLength(2);
});


test('309 grade 迟到详情不覆盖新对象', async ({ page }) => {
  await open(page);
  const a = await seed308(page, 'grade'), b = await seed308(page, 'grade');
  let release!: () => void, captured!: () => void, finished!: () => void;
  const gate = new Promise<void>(resolve => { release = resolve; });
  const ready = new Promise<void>(resolve => { captured = resolve; });
  const done = new Promise<void>(resolve => { finished = resolve; });
  const pattern = '**/api/v1/grades/' + a.id;
  await page.route(pattern, async route => {
    const response = await route.fetch();
    captured();
    await gate;
    try { await route.fulfill({ response }); }
    finally { finished(); }
  });
  try {
    await menu308(page, 'grade');
    await page.getByRole('textbox', { name: '搜索编码或名称', exact: true }).fill(a.code);
    await click308(page, page.getByRole('button', { name: '查询', exact: true }));
    await click308(page, page.getByRole('link', { name: a.name, exact: true }));
    await ready;
    await enter308(page, b);
    release();
    await done;
    await expect(page.getByRole('heading', { name: b.name, exact: true })).toBeVisible();
    await expect(page.getByRole('heading', { name: a.name, exact: true })).toHaveCount(0);
  } finally { release(); await page.unroute(pattern); }
});
