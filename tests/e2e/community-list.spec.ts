import { test, expect, readCurrentDate } from './fixtures';
import { login, seed, query, panel, option, isList, policy, database, out, credentials, connection } from './community-list-support.mjs';
import { appendFileSync } from 'node:fs';

let fixture: Awaited<ReturnType<typeof seed>>;
test.beforeAll(async ({ browser }) => {
  const context = await browser.newContext();
  try { const page = await context.newPage(); await login(page); fixture = await seed(page); }
  finally { await context.close(); }
});
test.beforeEach(async ({ page }) => { await login(page); await query(page, fixture.prefix); });
test.afterEach(async ({}, info) => {
  appendFileSync(out + '/business-results.jsonl', JSON.stringify({ title: info.title, result: info.status, durationMs: info.duration, prefix: fixture?.prefix }) + '\n');
});

test('TL323/324/325 正常菜单、独立跨页排序和0/1/20/21/73条原生分页', async ({ page }) => {
  test.setTimeout(120000);
  await expect(page.getByRole('link', { name: '新建', exact: true })).toBeEnabled();
  await expect(page.getByLabel('查看日期')).toHaveCount(0);
  expect(await page.locator('.app-header').evaluate(el => getComputedStyle(el).backgroundColor)).toBe('rgb(6, 107, 104)');
  for (const [q, count] of [[fixture.prefix + 'ABSENT', 0], [fixture.items[0].code, 1], [fixture.prefix + 'G20', 20], [fixture.prefix + 'G21', 21], [fixture.prefix, 73]] as const) {
    const data = await query(page, q); expect(data.count).toBe(count);
    if (count <= 20) await expect(page.getByRole('button', { name: '下一页', exact: true })).toBeDisabled(); else await expect(page.getByRole('button', { name: '下一页', exact: true })).toBeEnabled();
  }
  for (const size of [50, 100, 20]) {
    const pending = page.waitForResponse(isList); await option(page, '每页条数', String(size));
    const data = await (await pending).json(); expect(data.pageSize).toBe(size); expect(data.page).toBe(1);
  }
  const byName = [...fixture.items].sort((a, b) => a.name < b.name ? -1 : a.name > b.name ? 1 : a.code < b.code ? -1 : 1);
  for (const [sort, expected] of [['name', byName], ['name_desc', [...byName].reverse()], ['code', fixture.items]] as const) {
    const pending = page.waitForResponse(isList); await page.getByRole('columnheader', { name: /名称/ }).click();
    let data = await (await pending).json(); await expect(page).toHaveURL(u => u.searchParams.get('sort') === sort);
    const codes = data.items.map((x: any) => x.code);
    for (let p = 2; p <= 4; p++) {
      const next = page.waitForResponse(isList); await page.getByRole('button', { name: '下一页', exact: true }).press('Enter');
      data = await (await next).json(); expect(data.page).toBe(p); codes.push(...data.items.map((x: any) => x.code));
      await expect(page.getByText(`${(p - 1) * 20 + 1}–${Math.min(p * 20, 73)} / 73`, { exact: true })).toBeVisible();
    }
    expect(codes).toEqual(expected.map(x => x.code));
  }
});

test('TL329 搜索A/D、加载、大小、失败重试、排序和历史重建', async ({ page }) => {
  const input = page.getByLabel('搜索编码或名称');
  await input.fill(fixture.items[0].code);
  let release!: () => void; const held = new Promise<void>(r => { release = r; });
  await page.route('**/api/v1/business-units?*', async route => { const response = await route.fetch(); await held; await route.fulfill({ response }); });
  await page.getByRole('button', { name: '下一页', exact: true }).press('Enter');
  await expect(page.getByText('正在查询…', { exact: true })).toBeVisible();
  await expect(input).toHaveValue(fixture.items[0].code); release();
  await expect(page.getByText('21–40 / 73', { exact: true })).toBeVisible(); await page.unrouteAll({ behavior: 'wait' });
  expect(new URL(page.url()).searchParams.get('q')).toBe(fixture.prefix);
  await option(page, '每页条数', '50'); await expect(page.getByText('1–50 / 73', { exact: true })).toBeVisible();
  await expect(input).toHaveValue(fixture.items[0].code);
  await page.route('**/api/v1/business-units?*', route => route.fulfill({ status: 503, json: { code: 'UNAVAILABLE', message: '受控网络读取中断' } }));
  await page.getByRole('button', { name: '下一页', exact: true }).click(); await expect(page.getByText('受控网络读取中断')).toBeVisible();
  await expect(page.getByText('1–50 / 73', { exact: true })).toHaveCount(0);
  await input.fill('较新的草稿'); await page.unrouteAll({ behavior: 'wait' });
  await page.getByRole('button', { name: '重试', exact: true }).click(); await expect(page.getByText('51–73 / 73', { exact: true })).toBeVisible();
  await expect(input).toHaveValue('较新的草稿');
  await input.fill(''); await page.getByRole('columnheader', { name: /名称/ }).click();
  await expect(page).toHaveURL(u => !u.searchParams.has('q') && u.searchParams.get('page') === '1');
  await query(page, fixture.prefix); await input.fill('刷新丢弃'); await page.reload(); await expect(input).toHaveValue(fixture.prefix);
  await query(page, fixture.items[0].code); await page.goBack(); await expect(input).toHaveValue(fixture.prefix);
  await page.goForward(); await expect(input).toHaveValue(fixture.items[0].code);
});

test('TL330 状态局部草稿和三种真实状态，无通用查看日期', async ({page}) => {
 const input=page.getByLabel('搜索编码或名称'); let requests=0;
 page.on('request',r=>{if(new URL(r.url()).pathname==='/api/v1/business-units')requests++});
 await input.fill('未应用草稿');const before=requests;
 await panel(page,'状态筛选…');await page.getByRole('button',{name:'取消',exact:true}).click();
 expect(requests).toBe(before);await expect(input).toHaveValue('未应用草稿');await expect(page.getByRole('button',{name:'更多操作',exact:true})).toBeFocused();
 await input.fill(fixture.prefix);
 for(const [label,count] of [['启用',71],['停用',1],['未生效',1],['全部',73]] as const){
  await panel(page,'状态筛选…');await option(page,'状态',label);const pending=page.waitForResponse(isList);await page.getByRole('button',{name:'应用',exact:true}).click();expect((await(await pending).json()).count).toBe(count);
 }
 expect(new URL(page.url()).searchParams.has('businessAt')).toBe(false);expect(await readCurrentDate(page)).toBe(fixture.currentDate);
});

test('TL327/328 名称原生链接、跨页对象、浏览器返回及缺失目标', async ({ page, context }) => {
  test.setTimeout(60000); // Multiple native tabs and complete return journeys share this test budget.
  await option(page, '每页条数', '50'); await expect(page.getByText('1–50 / 73', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: '下一页', exact: true }).press('Enter'); await expect(page.getByText('51–73 / 73', { exact: true })).toBeVisible();
  await page.locator('.MuiDataGrid-virtualScroller').hover(); await page.mouse.wheel(0, 1000);
  const row = fixture.items[68], link = page.getByRole('link', { name: row.name, exact: true });
  await expect(link).toBeInViewport(); const listURL = page.url();
  await page.getByLabel('搜索编码或名称').fill('未应用秘密草稿');
  const href = await link.getAttribute('href'); expect(href).not.toContain('秘密'); expect(new URL(href!, page.url()).pathname).toContain(row.id);
  for (const click of [{ modifiers: ['Control'] as ('Control')[] }, { button: 'middle' as const }]) {
    const waiting = context.waitForEvent('page'); await link.click(click); const popup = await waiting;
    await expect(popup.getByRole('heading', { level: 1 })).toContainText(row.code); await popup.close(); await page.bringToFront();
    await expect(page.getByLabel('搜索编码或名称')).toHaveValue('未应用秘密草稿');
  }
  const copied = await context.newPage(); await copied.goto(new URL(href!, page.url()).href); await expect(copied.getByRole('heading', { level: 1 })).toContainText(row.code); await copied.close();
  await page.getByRole('gridcell', { name: row.code, exact: true }).click(); await page.keyboard.press('Enter'); await expect(page).toHaveURL(listURL);
  for (const back of ['link', 'browser']) {
    await link.press('Enter'); await expect(page.getByRole('heading', { level: 1 })).toContainText(row.code);
    if (back === 'link') await page.getByRole('link', { name: '返回列表', exact: true }).click(); else await page.goBack();
    await expect(link).toBeFocused(); await expect(link).toBeInViewport();
  }
  await link.click(); await expect(page.getByRole('button', { name: '下一个对象', exact: true })).toBeEnabled();
  let release!: () => void; const held = new Promise<void>(r => { release = r; });
  await page.route('**/api/v1/business-units?*', async route => { const response = await route.fetch(); await held; await route.fulfill({ response }); });
  const returned = page.waitForResponse(isList);
  await page.getByRole('link', { name: '返回列表', exact: true }).click();
  await page.getByLabel('搜索编码或名称').fill('返回期间用户接管'); release(); await (await returned).finished(); await page.unroute('**/api/v1/business-units?*');
  await expect(page.getByLabel('搜索编码或名称')).toBeFocused(); await expect(page.getByLabel('搜索编码或名称')).toHaveValue('返回期间用户接管');
  await query(page, fixture.prefix); await option(page, '每页条数', '20'); await expect(page.getByText('1–20 / 73', { exact: true })).toBeVisible();
  const last = page.getByRole('link', { name: fixture.items[19].name, exact: true }); await last.click();
  await page.getByRole('button', { name: '下一个对象', exact: true }).click(); await expect(page.getByRole('heading', { level: 1 })).toContainText(fixture.items[20].code);
  await page.getByRole('link', { name: '返回列表', exact: true }).click(); await expect(page.getByText('21–40 / 73', { exact: true })).toBeVisible();
  const target = page.getByRole('link', { name: fixture.items[20].name, exact: true }); await expect(target).toBeFocused(); await target.click();
  await policy(page, async (where, mutate) => {
    mutate(`UPDATE access.subject_grant SET scope_kind='bu',scope_id='${fixture.items[0].id}' WHERE ${where} AND action='bu.read'`);
    await page.getByRole('link', { name: '返回列表', exact: true }).click();
    await expect(page.locator('[data-list-heading]')).toBeFocused();
    await expect(page.getByText('原选中对象不在当前页或筛选结果中，查询条件已保留。')).toHaveCount(0);
    await expect(page.getByRole('link', { name: '查看原对象', exact: true })).toHaveCount(0);
    await expect(target).toHaveCount(0);
    expect(new URL(page.url()).searchParams.get('q')).toBe(fixture.prefix);
  });
});

test('TL326/331 查询者、真实字段裁剪、撤权及迟到真实响应', async ({ page, browser }) => {
  const readContext = await browser.newContext();
  try {
    const reader = await readContext.newPage(); await login(reader, true); await query(reader, fixture.prefix);
    await expect(reader.getByRole('link', { name: '新建', exact: true })).toBeDisabled();
    await reader.setViewportSize({ width: 390, height: 844 }); await query(reader, fixture.items[72].code); await expect(reader.locator('.list-row-metadata')).toContainText('未生效');
    await query(reader, fixture.items[22].code); expect(await reader.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true); await expect(reader.getByRole('link', { name: fixture.items[22].name, exact: true })).toBeVisible();
    await policy(reader, async (where, mutate) => {
      mutate(`UPDATE access.subject_grant SET read_fields=ARRAY['code'] WHERE ${where} AND action='bu.read'`);
      await reader.setViewportSize({ width: 390, height: 844 }); const actual = await query(reader, fixture.items[0].code);
      expect(actual.items[0].name).toBeUndefined(); expect(actual.items[0].effectiveDate).toBeUndefined();
      await expect(reader.getByRole('link', { name: '查看详情', exact: true })).toBeVisible();
      expect(await reader.locator('.MuiDataGrid-root').innerText()).not.toContain(fixture.items[0].name);
      expect(await reader.locator('.MuiDataGrid-root').innerText()).not.toContain('PrivateEnglish');
    });
  } finally { await readContext.close(); }
  await policy(page, async (where, mutate) => {
    let release!: () => void, arrived!: () => void; const held = new Promise<void>(r => { release = r; }), fetched = new Promise<void>(r => { arrived = r; });
    await page.route('**/api/v1/business-units?*', async route => {
      const response = await route.fetch(); if (new URL(route.request().url()).searchParams.get('q') === fixture.items[0].code) { arrived(); await held; }
      await route.fulfill({ response });
    });
    await page.getByLabel('搜索编码或名称').fill(fixture.items[0].code); await page.getByRole('button', { name: '查询', exact: true }).click(); await fetched;
    mutate(`DELETE FROM access.subject_grant WHERE ${where} AND action='bu.read'`);
    await page.getByLabel('搜索编码或名称').fill('撤权后草稿'); await page.getByRole('button', { name: '查询', exact: true }).click();
    await expect(page.getByLabel('搜索编码或名称')).toHaveValue(''); await expect(page.getByRole('link', { name: fixture.items[0].name, exact: true })).toHaveCount(0);
    release(); await page.unrouteAll({ behavior: 'wait' });
    await expect(page.getByText('1–1 / 1', { exact: true })).toHaveCount(0); await expect(page.getByLabel('搜索编码或名称')).toHaveValue('');
  });
});

test('TL333 原生显隐列宽及恢复仅改变展示，A/D与摘要保持', async ({ page }) => {
  let reads = 0, writes = 0; page.on('request', r => { if (r.url().includes('/api/v1/business-units')) { if (r.method() === 'GET') reads++; else writes++; } });
  const menu = async (name: string) => { await page.getByRole('columnheader', { name: new RegExp(name) }).hover(); await page.getByRole('button', { name: name + '列操作', exact: true }).click(); };
  await menu('编码'); await expect(page.getByRole('menuitem', { name: /降序|固定|导出/ })).toHaveCount(0); await expect(page.getByRole('menuitem', { name: '筛选…', exact: true })).toBeVisible(); await page.keyboard.press('Escape'); await expect(page.getByRole('menu')).toHaveCount(0);
  const codeHeader = page.getByRole('columnheader', { name: /编码/ }), width = await codeHeader.evaluate(e => e.getBoundingClientRect().width);
  await codeHeader.hover(); const grip = (await codeHeader.locator('.MuiDataGrid-columnSeparator').boundingBox())!;
  await page.mouse.move(grip.x + grip.width / 2, grip.y + grip.height / 2); await page.mouse.down(); await page.mouse.move(grip.x + 55, grip.y + grip.height / 2, { steps: 6 }); await page.mouse.up();
  await expect.poll(() => codeHeader.evaluate(e => e.getBoundingClientRect().width)).toBeGreaterThan(width + 20);
  const resized = await codeHeader.evaluate(e => e.getBoundingClientRect().width);
  await menu('名称'); await page.getByRole('menuitem', { name: '隐藏列', exact: true }).click();
  await panel(page, '列设置…'); await page.getByRole('checkbox', { name: '显示/隐藏全部', exact: true }).check(); await page.getByRole('checkbox', { name: '显示/隐藏全部', exact: true }).uncheck(); await page.keyboard.press('Escape'); await expect(page.getByRole('columnheader').filter({ hasNot: page.getByRole('checkbox') })).toHaveCount(0); await expect(page.getByRole('checkbox', { name: /全选查询结果/ })).toBeVisible();
  await panel(page, '列设置…'); await page.getByRole('button', { name: '重置', exact: true }).click(); await page.keyboard.press('Escape'); await expect(page.getByRole('columnheader').filter({ hasNot: page.getByRole('checkbox') })).toHaveCount(4);
  expect(reads).toBe(0); expect(writes).toBe(0);
  await page.getByLabel('搜索编码或名称').fill('保留草稿'); await expect.poll(() => codeHeader.evaluate(e => e.getBoundingClientRect().width)).toBe(resized);
  await query(page, fixture.items[0].code); await expect.poll(() => codeHeader.evaluate(e => e.getBoundingClientRect().width)).toBe(resized);
  await menu('名称'); await page.getByRole('menuitem', { name: '隐藏列', exact: true }).click();
  await page.getByLabel('搜索编码或名称').fill('断点草稿'); await page.setViewportSize({ width: 700, height: 900 });
  await expect(page.getByRole('link', { name: fixture.items[0].name, exact: true })).toBeVisible(); await expect(page.getByLabel('搜索编码或名称')).toHaveValue('断点草稿');
  await page.setViewportSize({ width: 1440, height: 900 }); await expect(page.getByRole('columnheader', { name: /名称/ })).toHaveCount(0);
});

test('TL334 查询只读，不产生身份、来源、修订或提交结果', async ({ page }) => {
  const sql = database();
  const snapshot = () => sql("SELECT json_build_array((SELECT md5(coalesce(string_agg(to_jsonb(t)::text,',' ORDER BY bu_id),'')) FROM organization.business_unit t),(SELECT md5(coalesce(string_agg(to_jsonb(t)::text,',' ORDER BY object_id,chunk_no),'')) FROM organization.bu_history t),(SELECT md5(coalesce(string_agg(to_jsonb(t)::text,',' ORDER BY subject_id,submission_id),'')) FROM organization.bu_terminal t),(SELECT count(*) FROM organization.bu_shape),(SELECT count(*) FROM organization.bu_dictionary))");
  const before = snapshot(); let writes = 0; page.on('request', r => { if (r.method() === 'POST' && r.url().includes('/api/v1/business-units')) writes++; });
  await query(page, fixture.prefix); await page.getByRole('button', { name: '下一页', exact: true }).click();
  await expect(page.getByText('21–40 / 73', { exact: true })).toBeVisible();
  await panel(page, '状态筛选…'); await option(page, '状态', '停用'); await page.getByRole('button', { name: '应用', exact: true }).click(); await expect(page.getByText('1–1 / 1', { exact: true })).toBeVisible();
  expect(snapshot()).toBe(before); expect(writes).toBe(0);
});

test('TL331 旧会话真实响应不覆盖新草稿，当前会话失效要求登录', async ({ page }) => {
  let release!: () => void, arrived!: () => void, captured = false;
  const held = new Promise<void>(r => { release = r; }), fetched = new Promise<void>(r => { arrived = r; });
  await page.route('**/api/v1/business-units?*', async route => {
    const response = await route.fetch();
    if (!captured) { captured = true; arrived(); await held; }
    await route.fulfill({ response });
  });
  try {
    await page.getByLabel('搜索编码或名称').fill(fixture.items[0].code);
    await page.getByRole('button', { name: '查询', exact: true }).click(); await fetched;
    await page.getByRole('button', { name: '退出', exact: true }).click(); await expect(page.getByLabel('登录名')).toBeVisible();
    await page.getByLabel('登录名').fill(credentials.login); await page.getByLabel('密码').fill(credentials.password);
    await page.waitForTimeout(3100); await page.getByRole('button', { name: '登录', exact: true }).click();
    await expect(page.getByRole('grid', { name: '业务单元列表' })).toBeVisible();
    await query(page, fixture.prefix); await page.getByLabel('搜索编码或名称').fill('新会话未应用草稿');
    release(); await page.unrouteAll({ behavior: 'wait' });
    await expect(page.getByLabel('搜索编码或名称')).toHaveValue('新会话未应用草稿'); await expect(page.getByText('1–20 / 73', { exact: true })).toBeVisible();
    const session = await (await page.request.get('/api/v1/session')).json();
    const expired = await page.request.delete('/api/v1/session', { headers: { Origin: connection.webOrigin, 'X-CSRF-Token': session.csrf } }); expect(expired.status()).toBe(204);
    await page.getByRole('button', { name: '查询', exact: true }).click(); await expect(page.getByLabel('登录名')).toBeVisible();
  } finally { release(); await page.unrouteAll({ behavior: 'ignoreErrors' }); }
});

test('TL327 窄屏链接支持浏览器操作，直达不伪造对象总数', async ({ page, context }) => {
  await page.setViewportSize({ width: 390, height: 844 }); await query(page, fixture.items[0].code);
  const link = page.getByRole('link', { name: fixture.items[0].name, exact: true });
  await page.getByLabel('搜索编码或名称').fill('手机未应用草稿'); const href = await link.getAttribute('href'); expect(href).not.toContain('手机未应用草稿');
  for (const click of [{ modifiers: ['Control'] as ('Control')[] }, { button: 'middle' as const }]) {
    const waiting = context.waitForEvent('page'); await link.click(click); const popup = await waiting;
    await expect(popup.getByRole('heading', { level: 1 })).toContainText(fixture.items[0].code); await popup.close(); await page.bringToFront();
    await expect(page.getByLabel('搜索编码或名称')).toHaveValue('手机未应用草稿');
  }
  const copied = await context.newPage(); await copied.goto(new URL(href!, page.url()).href); await expect(copied.getByRole('heading', { level: 1 })).toContainText(fixture.items[0].code);
  const direct = new URL(href!, page.url()); direct.search = ''; await copied.goto(direct.href);
  await expect(copied.getByRole('heading', { level: 1 })).toContainText(fixture.items[0].code); await expect(copied.getByRole('button', { name: '下一个对象', exact: true })).toHaveCount(0); await copied.close();
  await page.locator('.list-row-metadata').click(); await page.keyboard.press('Enter'); await expect(link).toBeVisible();
  await link.press('Enter'); await expect(page.getByRole('heading', { level: 1 })).toContainText(fixture.items[0].code); await page.goBack(); await expect(link).toBeFocused(); await expect(link).toBeInViewport(); await expect(page.getByRole('row').filter({ has: link })).toHaveClass(/list-return-selected/); await expect(page.getByRole('row').filter({ has: link }).getByRole('checkbox')).not.toBeChecked();
});

test('TL328/334 更正名称后移出原筛选，当前资料与旧来源仍可读', async ({ page }) => {
  const item = fixture.items[20]; await query(page, item.name); await page.getByRole('link', { name: item.name, exact: true }).click();
  const before = await (await page.request.get('/api/v1/business-units/' + item.id)).json();
  await panel(page, '更正记录'); await page.getByRole('textbox', { name: '中文名称', exact: true }).fill('更正后脱离原筛选');
  await page.getByRole('button', { name: '保存', exact: true }).click(); await expect(page.locator('.object-form')).toHaveCount(0);
  const after = await (await page.request.get('/api/v1/business-units/' + item.id)).json(); expect(after.headSourceVersion).not.toBe(before.headSourceVersion);
  const original = await (await page.request.get('/api/v1/business-units/' + item.id + '?sourceVersion=' + before.headSourceVersion)).json(); expect(original.timeline).toEqual(before.timeline);
  await page.getByRole('link', { name: '返回列表', exact: true }).click(); await expect(page.getByText('0–0 / 0', { exact: true })).toBeVisible();
  await expect(page.locator('[data-list-heading]')).toBeFocused();
  await expect(page.getByText('原选中对象不在当前页或筛选结果中，查询条件已保留。')).toHaveCount(0);
  await expect(page.getByRole('link', { name: '查看原对象', exact: true })).toHaveCount(0);
  expect(new URL(page.url()).searchParams.get('q')).toBe(item.name);
});
