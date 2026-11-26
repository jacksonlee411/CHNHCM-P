import { desktopViewports, nativeZoomCases } from './desktop-viewports.mjs';
import { action308, date308, family308, chooseClassification308, jobFunction308, menu308, reach308 } from './staffing308-support';
import { randomUUID } from 'node:crypto';
import { test, expect, type Page } from './fixtures';
import { open, choose } from './bu-support';
import { request } from './dated-object-support';
const families = '/settings/staffing/job-families',
  jobs = '/settings/staffing/jobs';
async function createFamily(page: Page) {
  await menu308(page, 'jobfamily');
  await page.getByRole('link', { name: '新建', exact: true }).click();
  const code = 'F307-' + randomUUID().slice(0, 8),
    name = '研发系列 ' + code;
  await page.getByRole('textbox', { name: '编码', exact: true }).fill(code);
  await page.getByRole('textbox', { name: '名称', exact: true }).fill(name);
  await page.getByLabel('生效日期').and(page.locator('input')).fill('2020-01-01');
  await page.getByRole('textbox', { name: '说明', exact: true }).fill('主要专业方向\n完整保留换行');
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.locator('.object-form')).toHaveCount(0);
  return { code, name, id: page.url().split('?')[0].split('/').at(-1)! };
}
test.describe.configure({ timeout: 120000 });
test.use({ actionTimeout: 15000 });
test.beforeEach(async ({ page }) => {
  page.on('response', async (r) => {
    if (r.url().includes('/api/v1/job') && r.status() >= 400) {
      const v = await r.json().catch(() => ({}));
      console.log(
        JSON.stringify({
          event: 'staffing-http-refusal',
          route: new URL(r.url()).pathname.replace(/[0-9a-f-]{36}/g, ':id'),
          status: r.status(),
          code: v.code,
          field: v.field,
          fields: Object.keys(r.request().postDataJSON() ?? {}),
        }),
      );
    }
  });
});
test('Staffing307 正常菜单完整创建职务系列和职务', async ({ page }) => {
  await open(page);
  const f = await createFamily(page);
  const jobFunction = await jobFunction308(page);
  await menu308(page, 'job');
  await page.getByRole('link', { name: '新建', exact: true }).click();
  const code = 'J307-' + randomUUID().slice(0, 8);
  await page.getByRole('textbox', { name: '编码', exact: true }).fill(code);
  await page.getByRole('textbox', { name: '名称', exact: true }).fill('研发工程师 ' + code);
  await page.getByLabel('生效日期').and(page.locator('input')).fill('2020-01-01');
  await expect(page.locator('#staffing-jobFunctionId')).toHaveText('请选择职类');
  await expect(page.locator('#staffing-jobFamilyId')).toHaveText('请选择职务系列');
  await chooseClassification308(page, 'jobFunctionId', jobFunction.name);
  await family308(page, f.name);
  await page.getByRole('textbox', { name: '主要职责', exact: true }).fill('系统设计\n实现与验证');
  await page.getByRole('textbox', { name: '任职要求', exact: true }).fill('熟悉产品开发');
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.locator('.object-form')).toHaveCount(0);
  await expect(page.getByText(f.name, { exact: true })).toBeVisible();
  await expect(page.getByText(jobFunction.name, { exact: true })).toBeVisible();
  await action308(page, '更正记录');
  await page.getByRole('textbox', { name: '任职要求', exact: true }).fill('');
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.locator('.object-form')).toHaveCount(0);
  await page.getByRole('tab', { name: '变更日志', exact: true }).click();
  await expect(page.getByRole('link', { name: '查看当次资料' })).toHaveCount(2);
});
test('Staffing307 系列记录维护及取消草稿', async ({ page }) => {
  await open(page);
  await createFamily(page);
  await action308(page, '新增记录');
  await page.getByRole('textbox', { name: '名称', exact: true }).fill('后续系列名称');
  await page.getByLabel('生效日期').and(page.locator('input')).fill('2021-01-01');
  await choose(page, '操作原因', '名称调整');
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.locator('.object-form')).toHaveCount(0);
  await action308(page, '更正记录');
  await page.getByRole('textbox', { name: '说明', exact: true }).fill('未保存草稿');
  await page.getByRole('button', { name: '取消', exact: true }).click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.getByRole('button', { name: '继续编辑', exact: true }).click();
  await expect(page.getByRole('textbox', { name: '说明', exact: true })).toHaveValue('未保存草稿');
});
async function apiObject(
  page: Page,
  kind: 'jobfamily' | 'job',
  options: {
    family?: string;
    name?: string;
    date?: string;
    code?: string;
  } = {},
) {
  const code = options.code ?? `T307-${randomUUID().slice(0, 8)}`,
    name = options.name ?? code;
  const jobFunction = kind === 'job' ? await jobFunction308(page) : undefined;
  const path = kind === 'job' ? '/api/v1/jobs' : '/api/v1/job-families';
  const r = await request(
    page,
    'POST',
    path,
    {
      code,
      name,
      effectiveDate: options.date ?? '2020-01-01',
      operationType: 'CREATE',
      operationReason: kind === 'job' ? 'NEW_JOB' : 'NEW_JOB_FAMILY',
      ...(kind === 'job'
        ? {
            jobFamilyId: options.family,
            jobFunctionId: jobFunction!.id,
            responsibilities: '第一行职责\n第二行职责',
            requirements: '任职要求',
          }
        : { description: '第一行说明\n第二行说明' }),
    },
    randomUUID(),
  );
  expect(r.status).toBe(201);
  return { id: r.body.objectId as string, code, name, path, jobFunction, web: kind === 'job' ? jobs : families };
}
async function detail(page: Page, o: Awaited<ReturnType<typeof apiObject>>) {
  const r = await request(page, 'GET', o.path + '/' + o.id);
  expect(r.status).toBe(200);
  return r.body;
}
async function saved(page: Page) {
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.locator('.object-form')).toHaveCount(0);
}
async function selectDate(page: Page, date: string) {
  await date308(page, date);
}
for (const kind of ['jobfamily', 'job'] as const) {
  test(`Staffing307 ${kind} 完整启停组合改日删除和原件`, async ({ page }) => {
    await open(page);
    const f = await apiObject(page, 'jobfamily');
    const o = kind === 'jobfamily' ? f : await apiObject(page, 'job', { family: f.id });
    await menu308(page, kind);
    await page.getByRole('textbox', { name: '搜索编码或名称', exact: true }).fill(o.code);
    await page.getByRole('button', { name: '查询', exact: true }).click();
    await page.getByRole('link', { name: o.name, exact: true }).first().click();
    await action308(page, '新增记录');
    await page.getByLabel('生效日期').and(page.locator('input')).fill('2025-01-01');
    await choose(page, '操作原因', '名称调整');
    await page.getByRole('textbox', { name: '名称', exact: true }).fill('独立后项 ' + o.code);
    await saved(page);
    await selectDate(page, '2020-01-01');
    await action308(page, '插入停用与重新启用');
    await page.getByLabel('停用日期').and(page.locator('input')).fill('2021-01-01');
    await page.getByLabel('重新启用日期').and(page.locator('input')).fill('2022-01-01');
    await saved(page);
    expect((await detail(page, o)).timeline).toHaveLength(4);
    await action308(page, '维护停用与重新启用');
    await page.getByLabel('停用日期').and(page.locator('input')).fill('2021-02-01');
    await saved(page);
    await action308(page, '删除停用与重新启用');
    await expect(page.getByRole('button', { name: '确认删除', exact: true })).toBeEnabled();
    await page.getByRole('button', { name: '确认删除', exact: true }).click();
    await expect(page.locator('.object-form')).toHaveCount(0);
    expect((await detail(page, o)).timeline).toHaveLength(2);
    await selectDate(page, '2025-01-01');
    await action308(page, '更正记录');
    await page.getByLabel('生效日期').and(page.locator('input')).fill('2024-01-01');
    await saved(page);
    await action308(page, '停用');
    await page.getByLabel('生效日期').and(page.locator('input')).fill('2025-01-01');
    await saved(page);
    await action308(page, '重新启用');
    await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-01-01');
    await saved(page);
    await selectDate(page, '2024-01-01');
    await action308(page, '删除记录');
    await page.getByRole('button', { name: '确认删除', exact: true }).click();
    await expect(page.locator('.object-form')).toHaveCount(0);
    expect((await detail(page, o)).timeline).toHaveLength(3);
    await page.getByRole('tab', { name: '变更日志', exact: true }).click();
    const deleted = page.getByRole('link', { name: '查看删除前原件', exact: true }).first();
    await reach308(page, deleted);
    await deleted.click();
    await expect(page.getByText('正在查看只读历史资料')).toBeVisible();
    await expect(
      page.getByRole('heading', { name: '独立后项 ' + o.code, level: 1, exact: true }),
    ).toBeVisible();
    await page.getByRole('link', { name: '返回变更日志', exact: true }).click();
    await expect(deleted).toBeFocused();
    await page.getByRole('link', { name: '查看当次资料', exact: true }).last().click();
    await expect(page.getByText('正在查看只读历史资料')).toBeVisible();
    await expect(page.getByRole('button', { name: '更多操作', exact: true })).toHaveCount(0);
    await expect(page.getByRole('button', { name: '返回当前资料' })).toBeVisible();
  });
  test(`Staffing307 ${kind} 查询者正常菜单只读及直接新建拒绝`, async ({ page, browser }) => {
    await open(page);
    const f = await apiObject(page, 'jobfamily');
    const o = kind === 'jobfamily' ? f : await apiObject(page, 'job', { family: f.id });
    const c = await browser.newContext({ baseURL: new URL(page.url()).origin });
    const p = await c.newPage();
    try {
      await open(p, 'reader');
      await menu308(p, kind);
      await expect(p.getByRole('link', { name: '新建', exact: true })).toHaveCount(0);
      await p.getByRole('textbox', { name: '搜索编码或名称' }).fill(o.code);
      await p.getByRole('button', { name: '查询', exact: true }).click();
      await p.getByRole('link', { name: o.name, exact: true }).first().click();
      await expect(p.getByRole('button', { name: '更多操作', exact: true })).toHaveCount(0);
      await p.getByRole('tab', { name: '变更日志' }).click();
      await expect(p.getByRole('link', { name: '查看当次资料' })).toHaveCount(1);
      await p.goto(o.web + '/new');
      await expect(p.getByText('当前没有新建权限，请返回列表查看获权资料。')).toBeVisible();
      await expect(p.locator('.object-form')).toHaveCount(0);
      const r = await request(
        p,
        'POST',
        o.path,
        {
          code: 'DENY-' + randomUUID().slice(0, 8),
          name: '拒绝',
          operationType: 'CREATE',
          operationReason: kind === 'job' ? 'NEW_JOB' : 'NEW_JOB_FAMILY',
          ...(kind === 'job' ? { jobFamilyId: f.id, jobFunctionId: o.jobFunction!.id } : {}),
        },
        randomUUID(),
      );
      expect(r.status).toBe(403);
    } finally {
      await c.close();
    }
  });
  test(`Staffing307 ${kind} 丢失保存响应自动查回且只写一份来源`, async ({ page }) => {
    await open(page);
    const f = await apiObject(page, 'jobfamily');
    const o = kind === 'jobfamily' ? f : await apiObject(page, 'job', { family: f.id });
    await page.goto(o.web + '/' + o.id);
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
    await saved(page);
    expect(writes).toBe(1);
    expect(resolves).toBe(1);
    const log = await request(page, 'GET', o.path + '/' + o.id + '/changes');
    expect(log.body.items).toHaveLength(2);
    await expect(page.getByText('已查回 ' + o.code, { exact: true }).first()).toBeVisible();
  });
  test(`Staffing307 ${kind} 原提交安全关闭后保留草稿可重试`, async ({ page }) => {
    await open(page);
    const f = await apiObject(page, 'jobfamily');
    const o = kind === 'jobfamily' ? f : await apiObject(page, 'job', { family: f.id });
    await page.goto(o.web + '/' + o.id);
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
    await saved(page);
    expect((await request(page, 'GET', o.path + '/' + o.id + '/changes')).body.items).toHaveLength(2);
  });
  test(`Staffing307 ${kind} 固定查询选择跨页新增排除CSV与未知建立`, async ({ page }) => {
    await open(page);
    const f = await apiObject(page, 'jobfamily');
    const prefix = 'SEL-' + randomUUID().slice(0, 8);
    const objects = [];
    for (let i = 0; i < 21; i++)
      objects.push(
        await apiObject(page, kind, { family: f.id, code: prefix + '-' + String(i).padStart(2, '0') }),
      );
    const o = objects[0];
    await menu308(page, kind);
    await page.getByRole('textbox', { name: '搜索编码或名称' }).fill(prefix);
    const applied = page.waitForResponse(
      (r) =>
        r.request().method() === 'GET' &&
        new URL(r.url()).pathname === o.path &&
        new URL(r.url()).searchParams.get('q') === prefix,
    );
    await page.getByRole('button', { name: '查询', exact: true }).click();
    await (await applied).finished();
    await expect(page).toHaveURL(new RegExp('q=' + prefix));
    await expect(page.locator('.standard-page[aria-busy]')).toHaveAttribute('aria-busy', 'false');
    await expect(page.getByText('1–20 / 21', { exact: true })).toBeVisible();
    await expect(page.getByRole('link', { name: prefix + '-00', exact: true }).first()).toBeVisible();
    let starts = 0;
    await page.route('**' + o.path + '/selection/start', async (route) => {
      starts++;
      await route.fetch();
      await route.abort('failed');
    });
    await page.getByRole('checkbox', { name: '全选查询结果', exact: true }).click();
    await page.getByRole('button', { name: '更多操作', exact: true }).click();
    await expect(page.getByRole('menuitem', { name: /导出所选/, exact: true })).toBeDisabled();
    await page.keyboard.press('Escape');
    await expect(page.getByText('网络连接失败，请重试', { exact: true })).toBeVisible();
    await page.getByRole('checkbox', { name: '全选查询结果', exact: true }).click();
    await page.getByRole('button', { name: '更多操作', exact: true }).click();
    await expect(page.getByRole('menuitem', { name: '导出所选（21）', exact: true })).toBeEnabled();
    await page.keyboard.press('Escape');
    expect(starts).toBe(1);
    await apiObject(page, kind, { family: f.id, code: prefix + '-99' });
    await page.getByRole('button', { name: '下一页', exact: true }).click();
    await expect(page).toHaveURL(/page=2/);
    await expect(page.getByRole('link', { name: prefix + '-20', exact: true }).first()).toBeVisible();
    await page
      .getByRole('checkbox', { name: `取消选择${kind === 'job' ? '职务' : '职务系列'}`, exact: true })
      .last()
      .click();
    await page.getByRole('button', { name: '更多操作', exact: true }).click();
    await expect(page.getByRole('menuitem', { name: '导出所选（20）', exact: true })).toBeEnabled();
    await page.keyboard.press('Escape');
    const download = page.waitForEvent('download');
    await page.getByRole('button', { name: '更多操作', exact: true }).click();
    await page.getByRole('menuitem', { name: '导出所选（20）' }).click();
    const d = await download;
    const stream = await d.createReadStream();
    const chunks: Buffer[] = [];
    for await (const chunk of stream!) chunks.push(Buffer.from(chunk));
    const csv = Buffer.concat(chunks).toString('utf8');
    expect(csv).toContain(o.code);
    expect(csv).not.toContain(prefix + '-99');
    expect(csv).not.toContain(prefix + '-20');
    expect(csv).toContain(kind === 'job' ? '第一行职责\n第二行职责' : '第一行说明\n第二行说明');
  });
}
test('Staffing307 完整条件导航局部重试和返回高亮', async ({ page }) => {
  await open(page);
  const prefix = 'NAV-' + randomUUID().slice(0, 8);
  for (let i = 0; i < 3; i++) await apiObject(page, 'jobfamily', { code: prefix + '-' + i });
  await menu308(page, 'jobfamily');
  await page.getByRole('textbox', { name: '搜索编码或名称' }).fill(prefix);
  const applied = page.waitForResponse(
    (r) =>
      r.request().method() === 'GET' &&
      new URL(r.url()).pathname === '/api/v1/job-families' &&
      new URL(r.url()).searchParams.get('q') === prefix,
  );
  await page.getByRole('button', { name: '查询', exact: true }).click();
  await (await applied).finished();
  await expect(page).toHaveURL(new RegExp('q=' + prefix));
  await expect(page.getByText('1–3 / 3', { exact: true })).toBeVisible();
  await page.route('**/api/v1/job-families/*/navigation?*', (route) =>
    route.fulfill({ status: 503, json: { code: 'M03_DEPENDENCY_UNAVAILABLE', message: '导航测试故障' } }),
  );
  await page
    .getByRole('link', { name: prefix + '-1', exact: true })
    .first()
    .click();
  expect(new URLSearchParams(new URL(page.url()).searchParams.get('back')!.split('?')[1]).get('q')).toBe(prefix);
  await expect(page.getByText('对象导航暂不可用，当前详情仍可查看。')).toBeVisible();
  await expect(page.getByRole('navigation', { name: '生效日期' })).toBeVisible();
  await page.unroute('**/api/v1/job-families/*/navigation?*');
  await page.getByRole('button', { name: '重试导航', exact: true }).click();
  await expect(page.getByText('2 / 3', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: '下一个对象', exact: true }).click();
  await expect(page.getByText('3 / 3', { exact: true })).toBeVisible();
  await page.getByRole('link', { name: '返回列表', exact: true }).click();
  await expect(page.getByRole('textbox', { name: '搜索编码或名称' })).toHaveValue(prefix);
  await expect(page.locator('.list-return-selected')).toHaveCount(1);
});
test('Staffing307 候选迟到和失败不覆盖新日期及已有输入', async ({ page }) => {
  await open(page);
  const f = await apiObject(page, 'jobfamily', {
    name: '候选期间 ' + randomUUID().slice(0, 8),
    date: '2025-01-01',
  });
  await menu308(page, 'job');
  await page.getByRole('link', { name: '新建', exact: true }).click();
  await page.getByRole('textbox', { name: '名称', exact: true }).fill('保留职责草稿');
  await page.getByRole('textbox', { name: '主要职责', exact: true }).fill('其他输入保持');
  let release!: () => void;
  const hold = new Promise<void>((r) => (release = r));
  let held = false;
  await page.route('**/api/v1/jobs/reference-context', async (route) => {
    const body = route.request().postDataJSON();
    if (body.effectiveDate === '2026-01-01') {
      held = true;
      const response = await route.fetch();
      await hold;
      await route.fulfill({ response }).catch(() => {});
      return;
    }
    await route.continue();
  });
  await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-01-01');
  await expect.poll(() => held).toBe(true);
  await page.getByLabel('生效日期').and(page.locator('input')).fill('2020-01-01');
  await expect(page.getByText('正在校验可用职务系列…')).toHaveCount(0);
  release();
  await page.locator('#staffing-jobFamilyId').click();
  await page.getByRole('textbox', { name: '查找职务系列', exact: true }).fill(f.name);
  await expect(page.getByRole('dialog').getByRole('button', { name: f.name, exact: true })).toHaveCount(0);
  await page.keyboard.press('Escape');
  await expect(page.getByRole('textbox', { name: '主要职责', exact: true })).toHaveValue('其他输入保持');
});
import { nativeZoomBrowser } from './native-zoom-support';
import { mkdirSync, writeFileSync } from 'node:fs';
for (const role of ['admin', 'reader']) {
  test(`Staffing307 ${role} 双对象桌面和短屏布局及取消`, async ({ page, browser }) => {
    await open(page);
    const f = await apiObject(page, 'jobfamily'),
      j = await apiObject(page, 'job', { family: f.id });
    let p = page,
      c: Awaited<ReturnType<typeof browser.newContext>> | undefined;
    if (role === 'reader') {
      c = await browser.newContext({ baseURL: new URL(page.url()).origin });
      p = await c.newPage();
      await open(p, 'reader');
    }
    try {
      await p.setViewportSize({ width: 1440, height: 540 });
      await menu308(p, 'jobfamily');
      await menu308(p, 'job');
      for (const o of [f, j])
        for (const [width, height] of desktopViewports) {
          await p.setViewportSize({ width, height });
          await p.goto(o.web + '?q=' + o.code);
          const link = p.getByRole('link', { name: o.name, exact: true }).first();
          await reach308(p, link);
          await expect(link).toBeInViewport();
          await expect
            .poll(() => p.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1))
            .toBe(true);
          await link.click();
          await expect(p.getByRole('navigation', { name: '生效日期' })).toBeVisible();
          if (role === 'admin') {
            await action308(p, '更正记录');
            const field = p.getByRole('textbox', { name: o === f ? '说明' : '主要职责', exact: true });
            await reach308(p, field);
            await expect(field).toBeInViewport();
            await field.focus();
            await p.keyboard.press('Tab');
            await reach308(p, p.getByRole('button', { name: '保存', exact: true }));
            await expect(p.getByRole('button', { name: '保存', exact: true })).toBeInViewport();
            await expect
              .poll(() => p.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1))
              .toBe(true);
            await p.getByRole('button', { name: '取消', exact: true }).click();
            await expect(p.locator('.object-form')).toHaveCount(0);
            await expect(p.getByRole('dialog')).toHaveCount(0);
          }
        }
    } finally {
      await c?.close();
    }
  });
  test(`Staffing307 ${role} 原生缩放双对象正常旅程`, async ({ page }) => {
    test.setTimeout(180000);
    await open(page);
    const f = await apiObject(page, 'jobfamily'),
      j = await apiObject(page, 'job', { family: f.id });
    const native = await nativeZoomBrowser(),
      p = native.page;
    const matrix = [];
    try {
      await open(p, role);
      for (const [, factor] of nativeZoomCases()) {
        await p.goto(f.web);
        await native.setZoom(factor);
        for (const o of [f, j]) {
          await p.goto(o.web + '?q=' + o.code);
          const link = p.getByRole('link', { name: o.name, exact: true }).first();
          await reach308(p, link);
          await expect(link).toBeInViewport();
          await link.click();
          await expect(p.getByRole('navigation', { name: '生效日期' })).toBeVisible();
          if (role === 'admin') {
            await action308(p, '更正记录');
            const field = p.getByRole('textbox', { name: o === f ? '说明' : '主要职责', exact: true });
            await reach308(p, field);
            await expect(field).toBeInViewport();
            await reach308(p, p.getByRole('button', { name: '保存', exact: true }));
            await expect(p.getByRole('button', { name: '保存', exact: true })).toBeInViewport();
          }
          await expect
            .poll(() => p.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1))
            .toBe(true);
          matrix.push({
            role,
            object: o === f ? 'jobfamily' : 'job',
            ...(await p.evaluate(() => ({ width: innerWidth, height: innerHeight, dpr: devicePixelRatio }))),
            passed: true,
          });
          if (role === 'admin') {
            await p.getByRole('button', { name: '取消', exact: true }).click();
            await expect(p.locator('.object-form')).toHaveCount(0);
          }
        }
      }
      const dir = '.local/staffing307-native';
      mkdirSync(dir, { recursive: true });
      writeFileSync(dir + '/native-' + role + '.json', JSON.stringify(matrix, null, 2) + '\n');
    } finally {
      await native.close();
    }
  });
}
test('Staffing307 未知结果离开刷新和换账号隔离后查回原提交', async ({ page }) => {
  await open(page);
  const f = await apiObject(page, 'jobfamily');
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
        kind: 'jobfamily',
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
test('Staffing307 当前系列名称准确旧采用和撤权标签清理', async ({ page }) => {
  const { withPolicy } = await import('./grants');
  await open(page);
  const f = await apiObject(page, 'jobfamily', { name: '原采用系列 ' + randomUUID().slice(0, 8) }),
    j = await apiObject(page, 'job', { family: f.id });
  const before = await detail(page, j),
    fd = await detail(page, f),
    renamed = '当前系列 ' + f.code;
  expect(
    (
      await request(
        page,
        'POST',
        f.path + '/' + f.id + '/records/' + fd.selectedRecord.recordId + '/correct',
        {
          expectedSourceVersion: fd.headSourceVersion,
          expectedRecordRevision: fd.selectedRecord.revision,
          changes: { name: renamed },
        },
        randomUUID(),
      )
    ).status,
  ).toBe(200);
  await page.goto(j.web + '/' + j.id);
  await expect(page.getByText(renamed, { exact: true })).toBeVisible();
  await page.goto(j.web + '/' + j.id + '?sourceVersion=' + before.headSourceVersion);
  await expect(page.getByText(f.name, { exact: true })).toBeVisible();
  await expect(page.getByText('正在查看只读历史资料')).toBeVisible();
  await withPolicy(page, async (where, mutate) => {
    mutate(`DELETE FROM access.subject_grant WHERE ${where} AND action='jobfamily.history.read'`);
    await page.reload();
    await expect(page.getByText('受限', { exact: true })).toBeVisible();
    await expect(page.getByText(f.name, { exact: true })).toHaveCount(0);
    await expect(page.getByText(renamed, { exact: true })).toHaveCount(0);
  });
  await page.goto(j.web + '/' + j.id);
  await expect(page.getByText(renamed, { exact: true })).toBeVisible();
  await action308(page, '更正记录');
  await expect(page.locator('#staffing-jobFamilyId')).toHaveText(renamed);
  await withPolicy(page, async (where, mutate) => {
    mutate(`DELETE FROM access.subject_grant WHERE ${where} AND action='jobfamily.read'`);
    await page.locator('#staffing-jobFamilyId').click();
    const denied = page.waitForResponse(
      (r) => r.url().endsWith('/api/v1/jobs/reference-context') && r.status() === 403,
    );
    await page.getByRole('textbox', { name: '查找职务系列', exact: true }).fill(renamed);
    await denied;
    await expect(page.locator('#staffing-jobFamilyId')).not.toHaveText(renamed);
    await page.route('**/api/v1/jobs/reference-context', (route) =>
      route.fulfill({ status: 503, json: { code: 'M03_DEPENDENCY_UNAVAILABLE', message: '候选故障' } }),
    );
    await page.getByRole('textbox', { name: '查找职务系列', exact: true }).fill('另一次搜索');
    await expect(page.getByText('候选故障', { exact: true })).toBeVisible();
    await expect(page.locator('#staffing-jobFamilyId')).not.toHaveText(renamed);
    await page.getByRole('dialog').getByRole('button', { name: '取消', exact: true }).click();
    await page.getByRole('textbox', { name: '主要职责', exact: true }).fill('仅改职责，引用保持');
    await saved(page);
    await expect(page.getByText(renamed, { exact: true })).toHaveCount(0);
    expect((await detail(page, j)).selectedRecord.responsibilities).toBe('仅改职责，引用保持');
  });
});
test('Staffing307 未来引用阻止系列停用且有限系列不进入开放新建候选', async ({ page }) => {
  await open(page);
  const f = await apiObject(page, 'jobfamily', { name: '未来引用系列 ' + randomUUID().slice(0, 8) }),
    j = await apiObject(page, 'job', { family: f.id, date: '2030-01-01' });
  await page.goto(f.web + '/' + f.id);
  await action308(page, '停用');
  await page.getByLabel('生效日期').and(page.locator('input')).fill('2029-01-01');
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.getByText('现有职务的启用期间超出候选系列范围，请先处理相关职务')).toBeVisible();
  expect((await detail(page, j)).timeline).toHaveLength(1);
  expect((await detail(page, f)).timeline).toHaveLength(1);
  const limited = await apiObject(page, 'jobfamily', { name: '有限系列 ' + randomUUID().slice(0, 8) }),
    v = await detail(page, limited);
  expect(
    (
      await request(
        page,
        'POST',
        limited.path + '/' + limited.id + '/deactivate',
        {
          expectedSourceVersion: v.headSourceVersion,
          anchorRecordId: v.selectedRecord.recordId,
          effectiveDate: '2029-01-01',
          operationType: 'DEACTIVATE',
          operationReason: 'FAMILY_CLOSURE',
        },
        randomUUID(),
      )
    ).status,
  ).toBe(200);
  await page.getByRole('button', { name: '取消', exact: true }).click();
  await page.getByRole('button', { name: '放弃修改', exact: true }).click();
  await menu308(page, 'job');
  await page.getByRole('link', { name: '新建', exact: true }).click();
  await page.getByLabel('生效日期').and(page.locator('input')).fill('2025-01-01');
  await page.locator('#staffing-jobFamilyId').click();
  await page.getByRole('textbox', { name: '查找职务系列', exact: true }).fill(limited.name);
  await expect(page.getByText('没有可用系列，请调整日期或搜索内容。')).toBeVisible();
  await expect(page.getByRole('dialog').getByRole('button', { name: limited.name, exact: true })).toHaveCount(
    0,
  );
});

test('Staffing307 不完整或跨类型确认保留原身份直到确切终态', async ({ page }) => {
  await open(page);
  const f = await apiObject(page, 'jobfamily');
  await page.goto(f.web + '/' + f.id);
  await action308(page, '更正记录');
  await page.getByRole('textbox', { name: '名称', exact: true }).fill('核对原身份 ' + f.code);
  await page.route('**' + f.path + '/**/correct', (r) => r.abort('failed'));
  let confirmations = 0;
  await page.route('**' + f.path + '/submissions/*/resolve', (route) => {
    confirmations++;
    const id = new URL(route.request().url()).pathname.split('/').at(-2);
    return route.fulfill({
      status: 200,
      json: {
        submissionId: confirmations === 1 ? randomUUID() : id,
        kind: confirmations === 1 ? 'jobfamily' : 'job',
        outcome: 'not_applied',
      },
    });
  });
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.getByRole('button', { name: '查看保存结果', exact: true })).toBeEnabled();
  expect(confirmations).toBe(1);
  await page.getByRole('button', { name: '查看保存结果', exact: true }).click();
  await expect(page.getByRole('button', { name: '查看保存结果', exact: true })).toBeEnabled();
  expect(confirmations).toBe(2);
  await expect(page.getByRole('textbox', { name: '名称', exact: true })).toHaveValue('核对原身份 ' + f.code);
  await page.unroute('**' + f.path + '/submissions/*/resolve');
  await page.getByRole('button', { name: '查看保存结果', exact: true }).click();
  await expect(page.getByText('原提交未生效，请核对输入后重新保存。')).toBeVisible();
  await page.unroute('**' + f.path + '/**/correct');
  await saved(page);
  expect((await request(page, 'GET', f.path + '/' + f.id + '/changes')).body.items).toHaveLength(2);
});

test.beforeEach(async ({ page }) => page.setDefaultTimeout(15000));
