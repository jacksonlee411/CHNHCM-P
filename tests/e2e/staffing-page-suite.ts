import { grants308 } from './staffing308-grants';
import { randomUUID } from 'node:crypto';
import { execFileSync } from 'node:child_process';
import { test, expect } from './fixtures';
import { open, choose } from './bu-support';
import { request } from './dated-object-support';
import { withPolicy } from './grants';
import {
  action308,
  labels308,
  base308,
  click308,
  date308,
  enter308,
  family308,
  menu308,
  matrixSeed308,
  reach308,
  seed308,
  web308,
  type Kind308,
} from './staffing308-support';
import { matrixJourney308 } from './staffing308-journey';
export function staffingPageSuite(kinds: readonly Kind308[], batch: string, journey = matrixJourney308) {
test.describe.configure({ timeout: 180000 });
test.beforeEach(async ({ page }) => page.setDefaultTimeout(15000));
for (const kind of kinds) {
  test(`${batch} ${kind} AC018/019 筛选打开时后退重查`, async ({ page }) => {
    await open(page);
    const family = await seed308(page, 'jobfamily'),
      o = await seed308(page, kind, family.id);
    await menu308(page, kind);
    await page.getByRole('textbox', { name: '搜索编码或名称', exact: true }).fill(o.code);
    await click308(page, page.getByRole('button', { name: '查询', exact: true }));
    await expect(page.getByText('1–1 / 1', { exact: true })).toBeVisible();
    await click308(page, page.getByRole('button', { name: '更多操作', exact: true }));
    await click308(page, page.getByRole('menuitem', { name: '列筛选…', exact: true }));
    await expect(page.getByRole('dialog', { name: '列筛选', exact: true })).toBeVisible();
    let release!: () => void;
    const gate = new Promise<void>((resolve) => { release = resolve; });
    const routePattern = '**' + base308[kind] + '?*';
    await page.route(routePattern, async (route) => { await gate; await route.continue(); });
    try {
      const reread = page.waitForRequest((r) => r.method() === 'GET' && new URL(r.url()).pathname === base308[kind]);
      await page.goBack();
      await reread;
      await expect(page.getByRole('heading', { name: labels308[kind], exact: true })).toBeVisible();
      await expect(page.getByRole('dialog', { name: '列筛选', exact: true })).toHaveCount(0);
    } finally {
      release();
    }
    await expect(page.locator('.standard-page[aria-busy]')).toHaveAttribute('aria-busy', 'false');
    await page.unroute(routePattern);
    await click308(page, page.getByRole('button', { name: '更多操作', exact: true }));
    await click308(page, page.getByRole('menuitem', { name: '列筛选…', exact: true }));
    await choose(page, '列', '编码');
    await choose(page, '条件', '等于');
    await page.getByRole('textbox', { name: '值', exact: true }).fill(o.code);
    await click308(page, page.getByRole('button', { name: '应用', exact: true }));
    await expect(page.getByText('1–1 / 1', { exact: true })).toBeVisible();
    await expect(page.getByRole('link', { name: o.name, exact: true })).toBeVisible();
  });
  for (const [width, height] of [[1440, 900], [320, 256]])
    test(`${batch} ${kind} AC021/022 从日志直接维护 ${width}x${height}`, async ({ page }) => {
      await open(page);
      const family = await seed308(page, 'jobfamily'),
        o = await seed308(page, kind, family.id),
        path = base308[kind] + '/' + o.id;
      await page.setViewportSize({ width, height });
      await enter308(page, o);
      await click308(page, page.getByRole('tab', { name: '变更日志', exact: true }));
      await expect(page.getByRole('tab', { name: '变更日志', exact: true })).toHaveAttribute('aria-selected', 'true');
      await expect(page.locator('.change-operation')).toHaveCount(1);
      const before = (await request(page, 'GET', path)).body.headSourceVersion;
      await action308(page, '更正记录');
      const basic = page.getByRole('tab', { name: '基本资料', exact: true }),
        history = page.getByRole('tab', { name: '变更日志', exact: true }),
        name = page.getByRole('textbox', { name: '名称', exact: true });
      await expect(basic).toHaveAttribute('aria-selected', 'true');
      await expect(history).toBeDisabled();
      await reach308(page, name);
      await expect(name).toHaveValue(o.name);
      await click308(page, page.getByRole('button', { name: '取消', exact: true }));
      await expect(page.locator('.object-form')).toHaveCount(0);
      await expect(history).toHaveAttribute('aria-selected', 'true');
      await expect(page.getByRole('button', { name: '更多操作', exact: true })).toBeFocused();
      await action308(page, '更正记录');
      await reach308(page, name);
      await name.fill(o.name + '草稿');
      await click308(page, page.getByRole('button', { name: '取消', exact: true }));
      await expect(page.getByRole('dialog', { name: '尚有未保存修改', exact: true })).toBeVisible();
      await click308(page, page.getByRole('button', { name: '继续编辑', exact: true }));
      await expect(name).toHaveValue(o.name + '草稿');
      await click308(page, page.getByRole('button', { name: '取消', exact: true }));
      await click308(page, page.getByRole('button', { name: '放弃修改', exact: true }));
      await expect(history).toHaveAttribute('aria-selected', 'true');
      expect((await request(page, 'GET', path)).body.headSourceVersion).toBe(before);
      await action308(page, '更正记录');
      await reach308(page, name);
      await name.fill(o.name + '已更正');
      await click308(page, page.getByRole('button', { name: '保存', exact: true }));
      await expect(page.locator('.object-form')).toHaveCount(0);
      await expect(basic).toHaveAttribute('aria-selected', 'true');
      await expect(page.getByRole('heading', { name: o.name + '已更正', exact: true })).toBeVisible();
      await click308(page, history);
      await expect(page.locator('.change-operation')).toHaveCount(2);
      await click308(page, page.getByRole('button', { name: '新增记录', exact: true }));
      await expect(basic).toHaveAttribute('aria-selected', 'true');
      await reach308(page, name);
      await expect(name).toHaveValue(o.name + '已更正');
      await click308(page, page.getByRole('button', { name: '取消', exact: true }));
      await expect(history).toHaveAttribute('aria-selected', 'true');
      await expect(page.locator('.change-operation')).toHaveCount(2);
    });
  test(`${batch} ${kind} AC018/019/027 查询草稿、41项固定选择与完整返回`, async ({ page }) => {
    await open(page);
    const tag = 'Q308-' + randomUUID().slice(0, 6),
      ref = await seed308(page, 'jobfamily');
    const objects = [],
      references = [];
    for (let n = 1; n <= 41; n++) {
      const reference =
        kind === 'job'
          ? await seed308(page, 'jobfamily', undefined, {
              code: `${tag}-F${String(n).padStart(3, '0')}`,
              name: `分类样本${String(n).padStart(3, '0')} ${tag}`,
            })
          : ref;
      references.push(reference);
      objects.push(
        await seed308(page, kind, reference.id, {
          code: `${tag}-${String(n).padStart(3, '0')}`,
          name: `${n <= 25 ? '华东' : '华南'}样本${String(n).padStart(3, '0')} ${tag}`,
        }),
      );
    }
    await test
      .info()
      .attach('query-samples', {
        body: JSON.stringify({ objects, references }),
        contentType: 'application/json',
      });
    await menu308(page, kind);
    const search = page.getByRole('textbox', { name: '搜索编码或名称', exact: true });
    await search.fill(tag);
    const appliedResponse = page.waitForResponse(
      (r) =>
        r.request().method() === 'GET' &&
        new URL(r.url()).pathname === base308[kind] &&
        new URL(r.url()).searchParams.get('q') === tag,
    );
    await click308(page, page.getByRole('button', { name: '查询', exact: true }));
    await (await appliedResponse).finished();
    await expect(page.getByText('1–20 / 41', { exact: true })).toBeVisible();
    await expect(page.getByRole('link', { name: objects[0].name, exact: true })).toBeVisible();
    const requests: string[] = [];
    page.on('request', (r) => {
      if (r.method() === 'GET' && new URL(r.url()).pathname === base308[kind])
        requests.push(new URL(r.url()).search);
    });
    const applied = page.url();
    await search.fill('华东');
    await click308(page, page.getByRole('button', { name: '更多操作', exact: true }));
    await click308(page, page.getByRole('menuitem', { name: '列筛选…', exact: true }));
    await choose(page, '列', '名称');
    await page.getByRole('textbox', { name: '值', exact: true }).fill(tag);
    await page.setViewportSize({ width: 1200, height: 540 });
    await expect(page.getByRole('textbox', { name: '值', exact: true })).toHaveValue(tag);
    // 012 §3.1.5 closes the local panel draft across 900px. A and search D remain.
    await page.setViewportSize({ width: 390, height: 844 });
    await expect(page.getByRole('dialog')).toHaveCount(0);
    await expect(page.getByRole('button', { name: '更多操作', exact: true })).toBeFocused();
    await expect(search).toHaveValue('华东');
    await page.setViewportSize({ width: 1440, height: 900 });
    await click308(page, page.getByRole('button', { name: '更多操作', exact: true }));
    await click308(page, page.getByRole('menuitem', { name: '列筛选…', exact: true }));
    await choose(page, '列', '名称');
    await page.getByRole('textbox', { name: '值', exact: true }).fill(tag);
    await click308(page, page.getByRole('button', { name: '取消', exact: true }));
    expect(page.url()).toBe(applied);
    expect(requests).toEqual([]);
    await expect(search).toHaveValue('华东');
    await click308(page, page.getByRole('button', { name: '更多操作', exact: true }));
    await click308(page, page.getByRole('menuitem', { name: '列筛选…', exact: true }));
    await choose(page, '列', '名称');
    await page.getByRole('textbox', { name: '值', exact: true }).fill(tag);
    await click308(page, page.getByRole('button', { name: '应用', exact: true }));
    await expect.poll(() => new URL(page.url()).searchParams.get('q')).toBe('华东');
    await expect.poll(() => requests.length).toBeGreaterThan(0);
    expect(new URLSearchParams(requests.at(-1)).get('filterValue')).toBe(tag);
    await menu308(page, kind);
    await search.fill(tag);
    const reappliedResponse = page.waitForResponse(
      (r) =>
        r.request().method() === 'GET' &&
        new URL(r.url()).pathname === base308[kind] &&
        new URL(r.url()).searchParams.get('q') === tag,
    );
    await click308(page, page.getByRole('button', { name: '查询', exact: true }));
    await (await reappliedResponse).finished();
    await expect(page.getByText('1–20 / 41', { exact: true })).toBeVisible();
    await expect(page.locator('.standard-page[aria-busy]')).toHaveAttribute('aria-busy', 'false');
    const row = page
      .getByRole('row')
      .filter({ has: page.getByRole('link', { name: objects[0].name, exact: true }) });
    await click308(page, row.getByRole('checkbox'));
    await expect(row.getByRole('checkbox')).toBeChecked();
    await expect(row.getByRole('checkbox')).toBeFocused();
    await page.keyboard.press('Space');
    await expect(row.getByRole('checkbox')).not.toBeChecked();
    await expect(page.getByRole('checkbox', { name: '全选查询结果', exact: true })).toHaveAttribute(
      'aria-checked',
      'false',
    );
    await page.keyboard.press('Space');
    await expect(row.getByRole('checkbox')).toBeChecked();
    await click308(page, page.getByRole('button', { name: '更多操作', exact: true }));
    await expect(page.getByRole('menuitem', { name: '导出所选（1）', exact: true })).toBeEnabled();
    await page.keyboard.press('Escape');
    await page.setViewportSize({ width: 390, height: 844 });
    await expect(row.getByRole('checkbox')).toBeChecked();
    await expect(row).toContainText('2026-01-01');
    await expect(search).toHaveValue(tag);
    await click308(page, page.getByRole('button', { name: '更多操作', exact: true }));
    await click308(page, page.getByRole('menuitem', { name: '排序…', exact: true }));
    await choose(page, '排序方式', '名称降序');
    await click308(page, page.getByRole('button', { name: '应用', exact: true }));
    await expect.poll(() => new URL(page.url()).searchParams.get('sort')).toBe('-name');
    await click308(page, page.getByRole('button', { name: '更多操作', exact: true }));
    await expect(page.getByRole('menuitem', { name: '导出所选（1）', exact: true })).toBeEnabled();
    await click308(page, page.getByRole('menuitem', { name: '排序…', exact: true }));
    await choose(page, '排序方式', '默认顺序');
    await click308(page, page.getByRole('button', { name: '应用', exact: true }));
    await page.setViewportSize({ width: 1440, height: 900 });
    await click308(page, page.getByRole('button', { name: '更多操作', exact: true }));
    await click308(page, page.getByRole('menuitem', { name: '取消选择', exact: true }));
    await click308(page, page.getByRole('checkbox', { name: '全选查询结果', exact: true }));
    await click308(page, page.getByRole('button', { name: '更多操作', exact: true }));
    await expect(page.getByRole('menuitem', { name: '导出所选（41）', exact: true })).toBeEnabled();
    await page.keyboard.press('Escape');
    await seed308(page, kind, ref.id, { code: `${tag}-042`, name: '后来成员 ' + tag });
    await click308(page, row.getByRole('checkbox'));
    await click308(page, page.getByRole('button', { name: '更多操作', exact: true }));
    const download = page.waitForEvent('download');
    await click308(page, page.getByRole('menuitem', { name: '导出所选（40）', exact: true }));
    const file = await (await download).createReadStream();
    let csv = '';
    for await (const chunk of file!) csv += chunk;
    expect(csv).not.toContain(`${tag}-001`);
    expect(csv).not.toContain(`${tag}-042`);
    for (let n = 2; n <= 41; n++) expect(csv).toContain(`${tag}-${String(n).padStart(3, '0')}`);
    expect(csv.split('\n')[0].replace('\ufeff', '').trim()).toBe(
      kind !== 'job'
        ? '编码,名称,英文名称,说明,状态,生效日期'
        : '编码,名称,英文名称,职务系列,主要职责,任职要求,状态,生效日期,职类',
    );
    const csvRows = JSON.parse(
      execFileSync('python3', ['-c', 'import sys,csv,json; print(json.dumps(list(csv.reader(sys.stdin))))'], {
        input: csv.replace(/^\ufeff/, ''),
        encoding: 'utf8',
      }),
    );
    expect(csvRows).toHaveLength(41);
    for (let i = 1; i <= 40; i++) {
      const object = objects[i];
      expect(csvRows[i]).toEqual([
        object.code.toUpperCase(),
        object.name,
        'English ' + object.code,
        ...(kind === 'job'
          ? [references[i].name, '职责第一行\n职责第二行', '任职要求']
          : ['说明第一行\n说明第二行']),
        '启用',
        '2026-01-01',
        ...(kind === 'job' ? [object.jobFunction!.name] : []),
      ]);
    }
    await search.fill('未应用的下一次查询');
    const failedPattern = '**' + base308[kind] + '?*';
    await page.route(failedPattern, (route) => route.abort('failed'), { times: 1 });
    await click308(page, page.getByRole('button', { name: '下一页', exact: true }));
    await expect(page.getByText('网络连接失败，请重试', { exact: true })).toBeVisible();
    const retryURL = page.url();
    await click308(page, page.getByRole('button', { name: '重试', exact: true }));
    await expect(page.getByRole('link', { name: objects[20].name, exact: true })).toBeVisible();
    expect(page.url()).toBe(retryURL);
    expect(new URL(page.url()).searchParams.get('q')).toBe(tag);
    await expect(search).toHaveValue('未应用的下一次查询');
    await click308(page, page.getByRole('link', { name: objects[20].name, exact: true }));
    await expect(page.getByLabel('第21个，共42个匹配对象', { exact: true })).toBeVisible();
    await click308(page, page.getByRole('link', { name: /返回/ }).first());
    await expect(page.getByRole('link', { name: objects[20].name, exact: true })).toBeFocused();
    expect(new URL(page.url()).searchParams.get('page')).toBe('2');
    await test
      .info()
      .attach('query-trace', {
        body: JSON.stringify({ requests, url: page.url() }),
        contentType: 'application/json',
      });
    if (kind === 'job') {
      await menu308(page, kind);
      await search.fill(tag);
      await click308(page, page.getByRole('button', { name: '更多操作', exact: true }));
      await click308(page, page.getByRole('menuitem', { name: '列筛选…', exact: true }));
      // Keep the fault active until observed. Development StrictMode may cancel
      // and reissue a mount request, which must not consume the only failure.
      await page.route('**/api/v1/job-families?*', (route) => route.abort('failed'));
      await choose(page, '列', '职务系列');
      await expect(page.getByRole('dialog').getByText('网络连接失败，请重试', { exact: true })).toBeVisible();
      await page.unroute('**/api/v1/job-families?*');
      await click308(page, page.getByRole('dialog').getByRole('button', { name: '重试', exact: true }));
      await expect(page.getByRole('combobox', { name: '职务系列', exact: true })).toBeEnabled();
      await choose(page, '职务系列', references[20].name);
      await click308(page, page.getByRole('button', { name: '应用', exact: true }));
      await expect(page.getByText('1–1 / 1', { exact: true })).toBeVisible();
      expect(new URL(page.url()).searchParams.get('jobFamilyId')).toBe(references[20].id);
      expect(new URL(page.url()).searchParams.get('q')).toBe(tag);
      await expect(page.getByRole('link', { name: objects[20].name, exact: true })).toBeVisible();
      const familyQuery = page.url(), requestCount = requests.length;
      await click308(page, page.getByRole('columnheader', { name: /职务系列/ }));
      expect(page.url()).toBe(familyQuery);
      expect(requests).toHaveLength(requestCount);
    }
  });
  test(`${batch} ${kind} AC020/026 首条、未来记录与准确旧源`, async ({ page }) => {
    await open(page);
    const family = await seed308(page, 'jobfamily');
    const future = await seed308(page, kind, family.id, { date: '9998-03-01' });
    await enter308(page, future);
    await expect(page.getByText('所选记录尚未生效。', { exact: true })).toBeVisible();
    await click308(page, page.getByRole('button', { name: '更多操作', exact: true }));
    await expect(page.getByRole('menuitem', { name: '删除记录', exact: true })).toHaveCount(0);
    await page.keyboard.press('Escape');
    const o = await seed308(page, kind, family.id);
    const path = base308[kind] + '/' + o.id,
      original = (await request(page, 'GET', path)).body;
    await enter308(page, o);
    await action308(page, '新增记录');
    await page.getByLabel('生效日期').and(page.locator('input')).fill('9998-03-01');
    await page.getByRole('textbox', { name: '名称', exact: true }).fill('独立后继 ' + o.code);
    await choose(page, '操作原因', '名称调整');
    await click308(page, page.getByRole('button', { name: '保存', exact: true }));
    await expect(page.locator('.object-form')).toHaveCount(0);
    await date308(page, '2026-01-01');
    await action308(page, '更正记录');
    await page.getByRole('textbox', { name: '名称', exact: true }).fill('更正首条 ' + o.code);
    await click308(page, page.getByRole('button', { name: '保存', exact: true }));
    await expect(page.locator('.object-form')).toHaveCount(0);
    await date308(page, '9998-03-01');
    await expect(
      page.getByRole('heading', { name: '独立后继 ' + o.code, exact: true, level: 1 }),
    ).toBeVisible();
    await expect(page.getByText('所选记录尚未生效。', { exact: true })).toBeVisible();
    await click308(page, page.getByRole('tab', { name: '变更日志', exact: true }));
    const first = page.getByRole('link', { name: '查看当次资料', exact: true }).last();
    await expect(first).toHaveAttribute('href', new RegExp('sourceVersion=' + original.headSourceVersion));
    await click308(page, first);
    await expect(page.getByRole('heading', { name: o.name, level: 1, exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: '更多操作', exact: true })).toHaveCount(0);
    await expect(page.getByText('正在查看只读历史资料', { exact: true })).toBeVisible();
  });
  test(`${batch} ${kind} AC024/025/026 恢复资料、操作分组和准确原件`, async ({ page }) => {
    await open(page);
    const a = await seed308(page, 'jobfamily'),
      b = await seed308(page, 'jobfamily');
    const o = await seed308(page, kind, a.id);
    const path = base308[kind] + '/' + o.id;
    await enter308(page, o);
    const unchanged = (await request(page, 'GET', path)).body.headSourceVersion;
    await action308(page, '更正记录');
    await click308(page, page.getByRole('button', { name: '保存', exact: true }));
    await expect(page.locator('.object-form')).toHaveCount(0);
    expect((await request(page, 'GET', path)).body.headSourceVersion).toBe(unchanged);
    await action308(page, '更正记录');
    const edit = page.getByRole('textbox', { name: '名称', exact: true });
    await edit.fill('尚未保存的名称');
    await expect(page.getByRole('navigation', { name: '生效日期', exact: true })).toHaveCount(0);
    await expect(page.getByRole('tab', { name: '变更日志', exact: true })).toBeDisabled();
    await click308(page, page.getByRole('link', { name: '返回列表', exact: true }));
    await click308(page, page.getByRole('button', { name: '继续编辑', exact: true }));
    await expect(edit).toHaveValue('尚未保存的名称');
    await page.goBack();
    await click308(page, page.getByRole('button', { name: '继续编辑', exact: true }));
    await expect(edit).toHaveValue('尚未保存的名称');
    await click308(page, page.getByRole('button', { name: '职务配置', exact: true }));
    await click308(
      page,
      page.getByRole('menuitem', { name: kind === 'job' ? '职务系列' : '职务', exact: true }),
    );
    await click308(page, page.getByRole('button', { name: '继续编辑', exact: true }));
    await expect(edit).toHaveValue('尚未保存的名称');
    await click308(page, page.getByRole('button', { name: '取消', exact: true }));
    await click308(page, page.getByRole('button', { name: '放弃修改', exact: true }));
    expect((await request(page, 'GET', path)).body.headSourceVersion).toBe(unchanged);
    await action308(page, '插入停用与重新启用');
    const dialog = page.getByRole('dialog');
    await expect(dialog.getByRole('heading', { name: '重新启用资料', exact: true })).toBeVisible();
    await page.getByLabel('停用日期').and(page.locator('input')).fill('2026-02-15');
    await page.getByLabel('重新启用日期').and(page.locator('input')).fill('2026-04-01');
    if (kind === 'job') await family308(page, b.name);
    else await page.getByRole('textbox', { name: '说明', exact: true }).fill('恢复后说明');
    await click308(page, page.getByRole('button', { name: '保存', exact: true }));
    await expect(page.locator('.object-form')).toHaveCount(0);
    const pair = (await request(page, 'GET', path)).body;
    expect(new URL(page.url()).searchParams.get('selectedRecordId')).toBe(pair.timeline[2].recordId);
    expect(pair.timeline[1][kind === 'job' ? 'jobFamilyId' : 'description']).toBe(
      kind === 'job' ? a.id : '说明第一行\n说明第二行',
    );
    expect(pair.timeline[2][kind === 'job' ? 'jobFamilyId' : 'description']).toBe(
      kind === 'job' ? b.id : '恢复后说明',
    );
    // 25 operations including CREATE, one pair and 23 independent corrections.
    for (let i = 1; i <= 23; i++) {
      const d = (await request(page, 'GET', path)).body;
      const r = await request(
        page,
        'POST',
        path + '/records/' + d.selectedRecord.recordId + '/correct',
        {
          expectedSourceVersion: d.headSourceVersion,
          expectedRecordRevision: d.selectedRecord.revision,
          changes: { name: `原件${i}`, nameEn: `Revision ${i}` },
        },
        randomUUID(),
      );
      expect(r.status).toBe(200);
    }
    await page.reload();
    await click308(page, page.getByRole('tab', { name: '变更日志', exact: true }));
    await expect(page.locator('.change-operation')).toHaveCount(20);
    await expect(page.locator('.change-operation').first()).toContainText('原件22');
    await expect(page.locator('.change-operation').first()).toContainText('原件23');
    await expect(
      page.locator('.change-operation').first().getByText('英文名称', { exact: true }),
    ).toBeVisible();
    const upper = new URL(page.url()).searchParams.get('logThrough');
    expect(upper).toBeTruthy();
    await click308(page, page.getByRole('button', { name: '下一页日志', exact: true }));
    await expect(page.locator('.change-operation')).toHaveCount(5);
    const originalURL = page.url();
    // Explicit Apply/Cancel/Clear is the same standard boundary on log columns.
    await click308(page, page.getByRole('button', { name: '日志更多操作', exact: true }));
    await click308(page, page.getByRole('menuitem', { name: '列筛选…', exact: true }));
    await choose(page, '列', '记录处理');
    await choose(page, '值', '更正记录');
    await click308(page, page.getByRole('button', { name: '取消', exact: true }));
    expect(page.url()).toBe(originalURL);
    await click308(page, page.getByRole('button', { name: '日志更多操作', exact: true }));
    await click308(page, page.getByRole('menuitem', { name: '列筛选…', exact: true }));
    await choose(page, '列', '记录处理');
    await choose(page, '值', '更正记录');
    await click308(page, page.getByRole('button', { name: '应用', exact: true }));
    await expect(page.getByText('1–20 / 23', { exact: true })).toBeVisible();
    expect(new URL(page.url()).searchParams.get('logThrough')).toBe(upper);
    await click308(page, page.getByRole('button', { name: '日志更多操作', exact: true }));
    await click308(page, page.getByRole('menuitem', { name: '列筛选…', exact: true }));
    await click308(page, page.getByRole('button', { name: '清除条件', exact: true }));
    expect(new URL(page.url()).searchParams.get('logFilterField')).toBe('action');
    await click308(page, page.getByRole('button', { name: '应用', exact: true }));
    await expect(page.getByText('1–20 / 25', { exact: true })).toBeVisible();
    await click308(page, page.getByRole('button', { name: '下一页日志', exact: true }));
    await expect(page.locator('.change-operation')).toHaveCount(5);
    expect(page.url()).toBe(originalURL);
    await click308(page, page.getByRole('link', { name: '查看当次资料', exact: true }).first());
    await expect(page.getByText('正在查看只读历史资料', { exact: true })).toBeVisible();
    await click308(page, page.getByRole('link', { name: '返回变更日志', exact: true }));
    expect(page.url()).toBe(originalURL);
    await expect(page.locator('.change-operation')).toHaveCount(5);
    expect(new URL(page.url()).searchParams.get('logThrough')).toBe(upper);
  });
  for (const permission of ['R0', 'RF', 'W0'] as const)
    test(`${batch} ${kind} AC026/030 ${permission} 权限边界`, async ({ page }) => {
      await open(page);
      const ref = await seed308(page, 'jobfamily'),
        o = await seed308(page, kind, ref.id, { name: '权限名称 ' + randomUUID().slice(0, 8) });
      await withPolicy(page, async (where, mutate) => {
        if (permission === 'R0')
          mutate(
            `DELETE FROM access.subject_grant WHERE ${where} AND action IN ('${kind}.history.read','${kind}.export')`,
          );
        if (permission === 'RF')
          mutate(
            `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE ${where} AND action IN ('${kind}.read','${kind}.history.read')`,
          );
        if (permission === 'W0')
          mutate(
            `DELETE FROM access.subject_grant WHERE ${where} AND action LIKE '${kind}.%' AND action NOT IN ('${kind}.read','${kind}.history.read','${kind}.export')`,
          );
        await test.info().attach('grants', {
          body: JSON.stringify(await grants308(page, permission)),
          contentType: 'application/json',
        });
        await page.goto(web308[kind] + '/' + o.id);
        await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
        if (permission === 'R0') {
          await expect(page.getByRole('tab', { name: '变更日志' })).toHaveCount(0);
          await expect(page.getByText(/操作时间：/)).toBeVisible();
        }
        if (permission === 'RF') {
          await expect(page.getByText(o.name, { exact: true })).toHaveCount(0);
          await click308(page, page.getByRole('tab', { name: '变更日志' }));
          await expect(page.locator('.change-log')).not.toContainText(o.name);
        }
        if (permission === 'W0') {
          await expect(page.getByRole('button', { name: '更多操作', exact: true })).toHaveCount(0);
          await expect(page.getByRole('button', { name: '新增记录', exact: true })).toHaveCount(0);
          await expect(page.getByRole('heading', { name: o.name, level: 1, exact: true })).toBeVisible();
        }
      });
    });
  for (const role of ['admin', 'reader'])
    for (const [code, width, height] of [
      ['V01', 1440, 900],
      ['V02', 1440, 540],
      ['V03', 390, 844],
      ['V04', 320, 900],
      ['V05', 320, 256],
    ] as const)
      test(`${batch} ${kind} ${role} ${code} AC017/020/021/022/023/028/030`, async ({ page }) => {
        await open(page);
        const ref = await seed308(page, 'jobfamily'),
          o = await matrixSeed308(page, kind, ref.id);
        if (role === 'reader') {
          await page.context().clearCookies();
          await open(page, 'reader');
        }
        await test
          .info()
          .attach('grants', {
            body: JSON.stringify(await grants308(page, role)),
            contentType: 'application/json',
          });
        await page.setViewportSize({ width, height });
        await journey(page, o, ref, role);
        await test.info().attach('viewport', {
          body: JSON.stringify({
            code,
            kind,
            role,
            ...(await page.evaluate(() => ({
              width: innerWidth,
              height: innerHeight,
              dpr: devicePixelRatio,
            }))),
          }),
          contentType: 'application/json',
        });
        await page.screenshot({
          path: test.info().outputPath('viewport.png'),
          mask: [page.locator('.app-account-name')],
        });
      });
}

}
