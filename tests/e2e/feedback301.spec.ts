import { test, expect } from './fixtures';
import { withPolicy } from './grants';
import { open, create } from './bu-support';
import { objects, menu, fillNew, apiBU, apiDepartment, readObject } from './dated-object-support';

test.setTimeout(120000);

for (const kind of objects) {
  test(`301 ${kind.label}保存成功响应丢失后自动确认且不重复写入`, async ({ page }) => {
    await open(page);
    const bu = kind.segment === 'departments' ? await apiBU(page) : undefined;
    await menu(page, kind);
    await fillNew(page, kind, { bu: bu?.code, name: '自动确认验证' });
    let writes = 0, confirmations = 0;
    page.on('request', request => {
      if (request.method() === 'POST' && new URL(request.url()).pathname.endsWith('/resolve') && request.url().includes('/submissions/')) confirmations++;
    });
    await page.route(`**/api/v1/${kind.segment}`, async route => {
      if (route.request().method() !== 'POST') return route.continue();
      writes++;
      const response = await route.fetch();
      expect(response.status()).toBe(201);
      await route.abort('connectionreset');
    });
    await page.getByRole('button', { name: '保存', exact: true }).click();
    await expect(page.locator('.object-form')).toHaveCount(0);
    expect(writes).toBe(1); expect(confirmations).toBe(1);
    const detail = await readObject(page, kind);
    expect(detail.timeline).toHaveLength(1);
    await expect(page.getByText(/原标识|查回原提交|Unexpected Application Error/)).toHaveCount(0);
  });

  test(`301 ${kind.label}请求校验失败恢复会话但不自动保存`, async ({ page }) => {
    await open(page);
    const bu = kind.segment === 'departments' ? await apiBU(page) : undefined;
    await menu(page, kind);
    await fillNew(page, kind, { bu: bu?.code, name: '保留输入验证' });
    let writes = 0;
    await page.route(`**/api/v1/${kind.segment}`, async route => {
      if (route.request().method() !== 'POST') return route.continue();
      writes++;
      if (writes === 1) {
        const headers = { ...route.request().headers(), 'x-csrf-token': 'invalid' };
        const response = await route.fetch({ headers });
        expect(response.status()).toBe(403);
        expect((await response.json()).code).toBe('AUTH_REQUEST_VALIDATION_FAILED');
        return route.fulfill({ response });
      }
      return route.continue();
    });
    await page.getByRole('button', { name: '保存', exact: true }).click();
    await expect(page.getByRole('textbox', { name: kind.name, exact: true })).toHaveValue('保留输入验证');
    await expect(page.getByRole('button', { name: '保存', exact: true })).toBeEnabled();
    expect(writes).toBe(1);
    await page.getByRole('button', { name: '保存', exact: true }).click();
    await expect(page.locator('.object-form')).toHaveCount(0);
    expect(writes).toBe(2);
  });
}

test('301列表读取故障保留条件且不显示保存指令', async ({ page }) => {
  await open(page);
  await menu(page, objects[0]);
  let fail = true;
  await page.route('**/api/v1/business-units?**', route => fail
    ? route.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ code: 'M02_DEPENDENCY_UNAVAILABLE', message: '服务暂时不可用，请稍后重试' }) })
    : route.continue());
  await page.getByRole('textbox', { name: '搜索编码或名称', exact: true }).fill('保留条件');
  await page.getByRole('button', { name: '查询', exact: true }).click();
  await expect(page.getByRole('alert')).toContainText('服务暂时不可用');
  await expect(page.getByText(/已提交|原标识|查回/)).toHaveCount(0);
  fail = false;
  await page.getByRole('button', { name: '重试', exact: true }).click();
  await expect(page.getByText('没有符合条件的业务单元', { exact: true })).toBeVisible();
  expect(new URL(page.url()).searchParams.get('q')).toBe('保留条件');
});

test('301未知保存只确认一次，离开和刷新后由用户查看结果', async ({ page }) => {
  await open(page); await menu(page, objects[0]); await fillNew(page, objects[0]);
  let confirmations = 0, writes = 0;
  await page.route('**/api/v1/business-units', route => {
    if (route.request().method() !== 'POST') return route.continue();
    writes++; return route.abort('connectionreset');
  });
  await page.route('**/api/v1/organization/submissions/*/resolve', route => {
    confirmations++;
    const id = new URL(route.request().url()).pathname.split('/').at(-2);
    return route.fulfill({ status: 202, json: { submissionId: id, outcome: 'pending' } });
  });
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.getByRole('button', { name: '查看保存结果', exact: true })).toBeEnabled();
  expect(confirmations).toBe(1); expect(writes).toBe(1);
  await page.getByRole('link', { name: '取消', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: '保存结果尚未确认', exact: true });
  await expect(dialog).toContainText('离开不会撤销已提交的保存操作');
  await dialog.getByRole('button', { name: '留在此页', exact: true }).click();
  await page.getByRole('link', { name: '取消', exact: true }).click();
  await dialog.getByRole('button', { name: '离开此页', exact: true }).click();
  await page.reload();
  await expect(page.getByRole('button', { name: '查看保存结果', exact: true })).toBeEnabled();
  expect(confirmations).toBe(1); expect(writes).toBe(1);
  await page.unroute('**/api/v1/organization/submissions/*/resolve');
  await page.getByRole('button', { name: '查看保存结果', exact: true }).click();
  await expect(page.getByText('未保存', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: '查看保存结果', exact: true })).toHaveCount(0);
});

test('301重复日志返回参数可定向重置，保留其他条件', async ({ page }) => {
  await open(page); await create(page);
  const url = new URL(page.url());
  url.searchParams.set('tab', 'history');
  url.searchParams.set('logPageSize', '50');
  url.searchParams.append('logReturn', 'bad'); url.searchParams.append('logReturn', 'duplicate');
  await page.goto(url.toString());
  await page.getByRole('tab', { name: '变更日志', exact: true }).click();
  await expect(page.getByText('日志查询条件重复，请重置条件', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: '重置日志条件', exact: true }).click();
  await expect(page.locator('.change-operation').first()).toBeVisible();
  const current = new URL(page.url());
  expect(current.searchParams.has('logReturn')).toBe(false);
  expect(current.searchParams.get('logPageSize')).toBe('50');
});

test('301辅助导航失败可局部重试，保留当前详情', async ({ page }) => {
  await open(page);
  const bu = await apiBU(page, '导航详情保留验证'); await menu(page, objects[0]);
  await page.getByRole('textbox', { name: '搜索编码或名称', exact: true }).fill(bu.code);
  await page.getByRole('button', { name: '查询', exact: true }).click();
  await page.route('**/api/v1/business-units/navigation', route => route.fulfill({ status: 503, json: { code: 'SERVICE_ERROR', message: '导航暂不可用' } }));
  await page.getByRole('link', { name: '导航详情保留验证', exact: true }).click();
  await expect(page.getByText('暂时无法切换记录', { exact: true })).toBeVisible();
  await expect(page.locator('.object-detail-sheet')).toContainText('导航详情保留验证');
  await page.unroute('**/api/v1/business-units/navigation');
  await page.getByRole('button', { name: '重试', exact: true }).click();
  await expect(page.getByText('暂时无法切换记录', { exact: true })).toHaveCount(0);
  await expect(page.locator('.object-detail-sheet')).toContainText('导航详情保留验证');
});

for (const kind of objects) test(`301查询者从正常菜单查询${kind.label}`, async ({ page }) => {
  await open(page, 'reader'); await menu(page, kind);
  await expect(page.getByRole('grid', { name: `${kind.label}列表`, exact: true })).toBeVisible();
  await page.getByRole('textbox', { name: '搜索编码或名称', exact: true }).fill('没有匹配记录301');
  await expect(page.getByText(/输入已修改，点击查询后生效/)).toBeVisible();
  await page.getByRole('button', { name: '查询', exact: true }).click();
  await expect(page.getByText(`没有符合条件的${kind.label}`, { exact: true })).toBeVisible();
  await expect(page.getByText(/原标识|查回|刷新后不保留/)).toHaveCount(0);
});

for (const kind of objects) {
  test(`301 ${kind.label}登录状态恢复失败时保留输入并暂停保存`, async ({ page }) => {
    await open(page);
    const bu = kind.segment === 'departments' ? await apiBU(page) : undefined;
    await menu(page, kind); await fillNew(page, kind, { bu: bu?.code, name: '恢复失败仍保留输入' });
    let writes = 0, failSession = true;
    await page.route('**/api/v1/session', route => failSession
      ? route.fulfill({ status: 503, json: { code: 'SERVICE_ERROR', message: '登录状态读取失败' } }) : route.continue());
    await page.route(`**/api/v1/${kind.segment}`, async route => {
      if (route.request().method() !== 'POST') return route.continue();
      if (++writes !== 1) return route.continue();
      const response = await route.fetch({ headers: { ...route.request().headers(), 'x-csrf-token': 'invalid' } });
      expect(response.status()).toBe(403); return route.fulfill({ response });
    });
    await page.getByRole('button', { name: '保存', exact: true }).click();
    await expect(page.getByText('暂时无法确认登录状态，请重试。', { exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: '保存', exact: true })).toBeDisabled();
    await expect(page.getByRole('textbox', { name: kind.name, exact: true })).toHaveValue('恢复失败仍保留输入');
    await page.locator(`#${kind.prefix}-create`).evaluate((form: HTMLFormElement) => form.requestSubmit());
    expect(writes).toBe(1);
    failSession = false;
    await page.getByRole('button', { name: '重试', exact: true }).click();
    await expect(page.getByRole('button', { name: '保存', exact: true })).toBeEnabled();
    expect(writes).toBe(1);
    await page.getByRole('button', { name: '保存', exact: true }).click();
    await expect(page.locator('.object-form')).toHaveCount(0); expect(writes).toBe(2);
  });

  test(`301 ${kind.label}发送前存储失败显示原因且不写入`, async ({ page }) => {
    await open(page);
    const bu = kind.segment === 'departments' ? await apiBU(page) : undefined;
    await menu(page, kind); await fillNew(page, kind, { bu: bu?.code, name: '发送前失败' });
    let writes = 0;
    page.on('request', r => { if (r.method() === 'POST' && new URL(r.url()).pathname === `/api/v1/${kind.segment}`) writes++; });
    await page.evaluate(() => {
      const original = Storage.prototype.setItem;
      Storage.prototype.setItem = function(key, value) {
        if (key.startsWith('chnhcm-pending-')) throw new Error('injected storage failure');
        return original.call(this, key, value);
      };
    });
    await page.getByRole('button', { name: '保存', exact: true }).click();
    await expect(page.getByText('暂时无法开始保存，请重试', { exact: true })).toBeVisible();
    await expect(page.getByRole('textbox', { name: kind.name, exact: true })).toHaveValue('发送前失败');
    await expect(page.getByRole('button', { name: '查看保存结果', exact: true })).toHaveCount(0);
    expect(writes).toBe(0);
  });
}

test('301不完整确认结果保留原请求，手动确认后才显示未保存', async ({ page }) => {
  await open(page); await menu(page, objects[0]); await fillNew(page, objects[0]);
  let writes = 0, resolves = 0;
  await page.route('**/api/v1/business-units', route => {
    if (route.request().method() !== 'POST') return route.continue();
    writes++; return route.abort('connectionreset');
  });
  await page.route('**/api/v1/organization/submissions/*/resolve', route => {
    resolves++; const submissionId = new URL(route.request().url()).pathname.split('/').at(-2);
    return route.fulfill({ status: 200, json: { submissionId, outcome: 'unrecognized' } });
  });
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.getByRole('button', { name: '查看保存结果', exact: true })).toBeEnabled();
  expect(resolves).toBe(1); expect(writes).toBe(1);
  await expect(page.getByText('未保存', { exact: true })).toHaveCount(0);
  const retained = () => page.evaluate(() => JSON.parse(localStorage.getItem('chnhcm-pending-submissions') ?? '[]'));
  expect(await retained()).toHaveLength(1);
  await page.getByRole('button', { name: '查看保存结果', exact: true }).click();
  await expect(page.getByText('业务单元：请求失败，请重试', { exact: true })).toBeVisible();
  expect(await retained()).toHaveLength(1); expect(resolves).toBe(2);
  await page.unroute('**/api/v1/organization/submissions/*/resolve');
  await page.getByRole('button', { name: '查看保存结果', exact: true }).click();
  await expect(page.getByText('未保存', { exact: true })).toBeVisible();
  expect(await retained()).toHaveLength(0); expect(writes).toBe(1);
});

test('301页面异常使用中文入口，待确认请求仍可恢复', async ({ page }) => {
  await open(page); await menu(page, objects[0]); await fillNew(page, objects[0]);
  await page.route('**/api/v1/business-units', route => route.request().method() === 'POST' ? route.abort('connectionreset') : route.continue());
  await page.route('**/api/v1/organization/submissions/*/resolve', route => route.fulfill({ status: 202, json: {
    submissionId: new URL(route.request().url()).pathname.split('/').at(-2), outcome: 'pending',
  } }));
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.getByRole('button', { name: '查看保存结果', exact: true })).toBeEnabled();
  const original = await page.evaluate(() => localStorage.getItem('chnhcm-pending-submissions'));
  await page.route('**/api/v1/business-units?**', async route => {
    const response = await route.fetch();
    return route.fulfill({ response, json: { ...await response.json(), items: null } });
  });
  await page.getByRole('link', { name: '取消', exact: true }).click();
  await page.getByRole('dialog').getByRole('button', { name: '离开此页', exact: true }).click();
  await expect(page.getByRole('heading', { name: '页面暂时无法显示', exact: true })).toBeVisible();
  await expect(page.getByText(/Unexpected Application Error|TypeError|developer/i)).toHaveCount(0);
  expect(await page.evaluate(() => localStorage.getItem('chnhcm-pending-submissions'))).toBe(original);
  await page.unroute('**/api/v1/business-units?**');
  await page.unroute('**/api/v1/organization/submissions/*/resolve');
  await page.getByRole('link', { name: '返回首页', exact: true }).click();
  await page.getByRole('button', { name: '查看保存结果', exact: true }).click();
  await expect(page.getByText('未保存', { exact: true })).toBeVisible();
});


test('301失效分页重试保留筛选和未应用输入，重新建立首页位置', async ({ page }) => {
  await open(page);
  const bu = await apiBU(page), department = await apiDepartment(page, bu.id, { name: '分页恢复301' });
  await menu(page, objects[1]);
  const url = new URL(page.url());
  const values = { q: department.code, status: 'active', filterField: 'departmentPath', filterOperator: 'contains', filterValue: '分页恢复301', sort: 'name', pageSize: '50', page: '2', cursor: 'expired' };
  for (const [key, value] of Object.entries(values)) url.searchParams.set(key, value);
  await page.goto(url.toString());
  await expect(page.locator('[data-list-return-fallback]')).toBeVisible();
  const input = page.getByRole('textbox', { name: '搜索编码或名称', exact: true });
  await input.fill('尚未应用的新输入');
  await page.locator('[data-list-return-fallback]').getByRole('button', { name: '重试', exact: true }).click();
  await expect(page.getByRole('grid', { name: '部门列表', exact: true })).toContainText(department.code);
  await expect(input).toHaveValue('尚未应用的新输入');
  const result = new URL(page.url());
  expect(result.searchParams.has('cursor')).toBe(false); expect(result.searchParams.get('page')).toBe('1');
  for (const [key, value] of Object.entries(values)) if (!['page','cursor'].includes(key)) expect(result.searchParams.get(key)).toBe(value);
});


for (const kind of objects) for (const invalid of ['logPageSize', 'logThrough', 'logFilterValue']) test(`301 ${kind.label}日志重置${invalid}保留其他合法条件`, async ({ page }) => {
  await open(page);
  const bu = await apiBU(page, '日志条件保留301');
  const object = kind.segment === 'departments' ? await apiDepartment(page, bu.id) : bu;
  await menu(page, kind);
  const url = new URL(page.url() + '/' + object.id);
  url.searchParams.set('tab', 'history');
  url.searchParams.set('logPageSize', '50');
  url.searchParams.set('logFilterField', 'action');
  url.searchParams.set('logFilterOperator', 'is');
  url.searchParams.set('logFilterValue', 'CREATE');
  if (invalid === 'logFilterValue') url.searchParams.append(invalid, 'CORRECT');
  else url.searchParams.set(invalid, invalid === 'logPageSize' ? '10' : 'invalid');
  await page.goto(url.toString());
  await page.getByRole('tab', { name: '变更日志', exact: true }).click();
  await page.getByRole('button', { name: '重置日志条件', exact: true }).click();
  await expect(page.locator('.change-operation')).toHaveCount(1);
  const result = new URL(page.url());
  for (const key of ['logFilterField','logFilterOperator','logFilterValue']) expect(result.searchParams.get(key)).toBe(invalid === 'logFilterValue' ? null : url.searchParams.get(key));
  if (invalid !== 'logPageSize') expect(result.searchParams.get('logPageSize')).toBe('50');
  else expect(result.searchParams.has('logPageSize')).toBe(false);
  expect(result.searchParams.get('logThrough')).not.toBe('invalid');
});


test('301同一BU筛选重新查询后只显示当前可读名称，条件不变且可清除', async ({ page }) => {
  await open(page);
  const bu = await apiBU(page, '筛选摘要需保护名称301');
  await apiDepartment(page, bu.id);
  await menu(page, objects[1]);
  const params = new URLSearchParams({ filterField: 'managementBuId', filterOperator: 'equals', filterValue: bu.id });
  const initialReference = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/departments/references/business-units' && r.status() === 200);
  await page.goto(new URL(page.url()).origin + '/settings/organization/departments?' + params);
  expect((await (await initialReference).json()).selectedReference.name).toBe(bu.name);
  const summary = page.locator('.list-applied-summary').getByRole('button', { name: /^管理业务单元：/ });
  await expect(summary).toContainText(bu.name);
  await withPolicy(page, async (where, mutate) => {
    mutate(`UPDATE access.subject_grant SET read_fields=ARRAY['code']::text[] WHERE ${where} AND action='bu.reference.read'`);
    const loaded = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/departments/references/business-units');
    await page.getByRole('button', { name: '查询', exact: true }).click();
    const response = await loaded; expect(response.status()).toBe(200);
    const reference = (await response.json()).selectedReference;
    expect(reference.id).toBe(bu.id); expect(reference).not.toHaveProperty('name');
    await expect(summary).toContainText(bu.code);
    await expect(summary).not.toContainText(bu.name);
    expect(new URL(page.url()).searchParams.get('filterValue')).toBe(bu.id);
    await summary.click();
    const dialog = page.getByRole('dialog');
    await dialog.getByRole('button', { name: '清除条件', exact: true }).click();
    await dialog.getByRole('button', { name: '应用', exact: true }).click();
    expect(new URL(page.url()).searchParams.has('filterValue')).toBe(false);
    await expect(summary).toHaveCount(0);
  });
});

test('301树范围更新后立即查询和清空关键词不覆盖新范围', async ({ page }) => {
  await open(page);
  const bu = await apiBU(page), leaf = await apiDepartment(page, bu.id, { name: '连续查询末级部门' });
  const filters = { status: 'active', filterField: 'code', filterOperator: 'equals', filterValue: leaf.code, sort: 'name_desc', pageSize: '50', q: '旧' };
  await page.goto('/settings/organization/departments?' + new URLSearchParams(filters));
  const query = page.getByRole('textbox', { name: '搜索编码或名称', exact: true });
  await query.fill('无匹配结果');
  await page.getByRole('textbox', { name: '定位树中部门', exact: true }).fill(leaf.code);
  // Exercise the interval after history changes but before React commits the
  // new query. A user action on the still-visible controls must keep its scope.
  await page.evaluate(id => {
    const original = history.pushState;
    history.pushState = function(...args) {
      original.apply(this, args);
      if (new URL(location.href).searchParams.get('scopeDepartmentId') === id) {
        history.pushState = original;
        const button = [...document.querySelectorAll('button')].find(element => element.textContent === '查询');
        button?.click();
      }
    };
  }, leaf.id);
  await page.getByRole('listbox', { name: '部门定位结果', exact: true }).getByRole('option').click();
  await expect(page).toHaveURL(url => url.searchParams.get('q') === '无匹配结果' && url.searchParams.get('scopeDepartmentId') === leaf.id);
  await expect(page.getByText('没有符合条件的部门', { exact: true })).toBeVisible();
  await query.fill('');
  await expect(page.getByRole('grid', { name: '部门列表', exact: true }).getByRole('link', { name: '连续查询末级部门', exact: true })).toBeVisible();
  const current = new URL(page.url()).searchParams;
  expect(current.get('scopeDepartmentId')).toBe(leaf.id);
  expect(current.get('q')).toBeNull();
  for (const [key, value] of Object.entries(filters)) if (key !== 'q') expect(current.get(key)).toBe(value);
});
