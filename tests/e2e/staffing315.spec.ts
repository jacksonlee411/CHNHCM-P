import { desktopViewports } from './desktop-viewports.mjs';
import { execFileSync } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import { test, expect } from './fixtures';
import { open } from './bu-support';
import { click308, date308, namedMenu308, seed308 } from './staffing308-support';
import { seedPosition } from './staffing312-support';
import { request } from './dated-object-support';
import { withPolicy } from './grants';
import { staffingZoomCases } from './staffing-zoom-suite';

for (const [width, height] of desktopViewports)
  test('315 岗位核编与只读原件 ' + width + 'x' + height, async ({ page }) => {
    test.setTimeout(180000);
    await open(page);
    const family = await seed308(page, 'jobfamily'),
      sample = await seedPosition(page, 'position', family.id);
    const pos = await request(page, 'GET', '/api/v1/positions/' + sample.id);
    const { departmentId, jobId, locationId } = pos.body.selectedRecord;
    const supervisor = await request(
      page,
      'POST',
      '/api/v1/positions',
      {
        departmentId,
        jobId,
        locationId,
        name: '主管' + sample.code,
        code: 'M' + sample.code,
        effectiveDate: '2026-01-01',
        operationType: 'CREATE',
        operationReason: 'NEW_POSITION',
      },
      randomUUID(),
    );
    expect(supervisor.status).toBe(201);
    expect(
      (
        await request(
          page,
          'POST',
          '/api/v1/staffing-quotas',
          {
            positionId: supervisor.body.objectId,
            effectiveDate: '2027-01-01',
            quota: 1,
            reason: '核定主管岗',
          },
          randomUUID(),
        )
      ).status,
    ).toBe(200);
    await page.setViewportSize({ width, height });
    const menu = () =>
      namedMenu308(page, '部门编制分配', '职务配置', '/settings/staffing/quotas', '部门编制统计');
    const click = (name: string) => click308(page, page.getByRole('button', { name, exact: true }));
    const openPosition = async () => {
      await page.getByRole('textbox', { name: '搜索岗位编码或名称' }).fill(sample.code);
      await click('查询');
      await click308(page, page.getByRole('link', { name: sample.name, exact: true }));
      await expect(page.getByRole('heading', { name: sample.name, exact: true })).toBeVisible();
    };
    const setDay = async (d: string) => {
      await page.getByLabel('统计日期').and(page.locator('input')).fill(d);
      await click('查询统计');
    };
    const amount = page
      .locator('.object-detail-field')
      .filter({ has: page.locator('label').filter({ hasText: /^核定编制$/ }) })
      .locator('.object-field-value');
    const inputAmount = page.getByRole('spinbutton', { name: /核定编制/ }),
      date = page.getByLabel('生效日期').and(page.locator('input')),
      reason = page.getByLabel('调整原因').and(page.locator('input'));
    const save = async () => {
      await click('保存');
      await expect(page.getByRole('heading', { name: sample.name, exact: true })).toBeVisible();
    };
    const change = async (action: string, n: number, day: string) => {
      await click(action);
      await date.fill(day);
      await inputAmount.fill(String(n));
      await reason.fill(action + '依据');
      await save();
      await expect(amount).toHaveText(String(n));
    };
    await namedMenu308(page, '岗位', '职务配置', '/settings/staffing/positions');
    await expect(page.getByLabel(/查询日期|统计日期/)).toHaveCount(0);
    await page.getByRole('textbox', { name: '搜索编码或名称' }).fill(sample.code);
    await click('查询');
    await click308(page, page.getByRole('link', { name: sample.name, exact: true }));
    await expect(page.locator('.object-detail-columns section').first()).toBeVisible();
    const columns = () =>
      page
        .locator('.object-detail-columns > section')
        .evaluateAll((sections) => sections.map((el) => Math.round(el.getBoundingClientRect().left)));
    const referenceColumns = await columns();
    const definition = await request(page, 'GET', '/api/v1/positions/' + sample.id);
    await click308(page, page.getByRole('tab', { name: '岗位编制', exact: true }));
    await expect(page.getByLabel(/查询日期|统计日期/)).toHaveCount(0);
    const back = new URL(new URL(page.url()).searchParams.get('back')!, page.url());
    expect(back.searchParams.get('q')).toBe(sample.code);
    await expect(page.getByRole('navigation', { name: '对象导航' })).toContainText(/\d\s*\/\s*2$/);
    await expect(amount).toHaveText('未核编');
    if (width === 1440 && height === 900) {
      await click('上一个对象');
      await expect(page.getByRole('heading', { name: '主管' + sample.code, exact: true })).toBeVisible();
      await expect(amount).toHaveText('1');
      await click('下一个对象');
      await expect(page.getByRole('heading', { name: sample.name, exact: true })).toBeVisible();
      await expect(amount).toHaveText('未核编');
    }
    await click('核编');
    await expect(date).toHaveValue('1900-01-01');
    expect(await columns()).toEqual(referenceColumns);
    await inputAmount.fill('2');
    await date.fill('2027-01-01');
    await reason.fill('首次核编');
    expect(
      await inputAmount.evaluate((el) => [
        getComputedStyle(el).fontSize,
        el.closest('.MuiInputBase-root')!.getBoundingClientRect().height,
      ]),
    ).toEqual(['14px', 32]);
    await save();
    await expect(amount).toHaveText('2');
    await change('调编', 3, '2027-02-01');
    await change('调编', 1, '2027-04-01');
    await change('调编', 4, '2027-06-01');
    if (width === 1440 && height === 900) {
      await click('调编');
      await inputAmount.fill('5');
      await date.fill('2027-04-01');
      await reason.fill('四月定额重核');
      await expect(page.getByText('该日已有独立记录', { exact: true })).toBeVisible();
      await click('重读当前记录');
      await expect(inputAmount).toHaveValue('5');
      await expect(reason).toHaveValue('四月定额重核');
      await click('取消');
      await click('放弃修改');
    }
    await date308(page, '2027-04-01');
    await expect(amount).toHaveText('1');
    const correct = async () => {
      await click('更多操作');
      await click308(page, page.getByRole('menuitem', { name: '更正记录', exact: true }));
      await expect(inputAmount).toHaveValue('1');
      await inputAmount.fill('3');
    };
    await correct();
    await click('取消');
    await click('放弃修改');
    await expect(amount).toHaveText('1');
    await correct();
    await expect(page.getByText('2027-04-01 至 2027-05-31', { exact: true })).toBeVisible();
    if (width === 1440 && height === 900) {
      await page.route(
        '**/staffing-quotas/**/correct',
        async (route) => {
          const response = await route.fetch();
          expect(response.status()).toBe(200);
          await route.abort('failed');
        },
        { times: 1 },
      );
      await page.route('**/staffing-quotas/submissions/*/resolve', (route) => route.abort('failed'), {
        times: 1,
      });
      await click('保存');
      await expect(page.getByRole('button', { name: '查看保存结果' })).toBeEnabled();
      // Restore the pending reference as a browser last opened before 00028 stored it.
      await page.evaluate(() => {
        const currentKey = 'chnhcm-pending-staffing-quotas';
        const pending = localStorage.getItem(currentKey);
        if (!pending || !JSON.parse(pending).length) throw new Error('缺少待确认的核编提交号');
        localStorage.setItem('chnhcm-pending-quotas', pending);
        localStorage.removeItem(currentKey);
      });
      page.once('dialog', (dialog) => dialog.accept());
      await page.reload();
      const recovered = page.waitForResponse((r) =>
        r.request().method() === 'POST' && /\/staffing-quotas\/submissions\/[^/]+\/resolve$/.test(new URL(r.url()).pathname));
      await click('查看保存结果');
      expect((await recovered).status()).toBe(200);
      await expect(page.getByText('正在查看只读历史资料')).toBeVisible();
      await expect(page.getByRole('link', { name: '查看岗位核编原件' })).toBeVisible();
    } else await click('保存');
    await expect(amount).toHaveText('3');
    await expect(page.getByText('2027-04-01 至 2027-05-31', { exact: true })).toBeVisible();
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.context().clearCookies();
    await open(page, 'reader');
    await page.setViewportSize({ width, height });
    await menu();
    await setDay('2027-04-01');
    await click308(page, page.getByRole('combobox', { name: '所属部门' }));
    await click308(page, page.getByRole('option', { name: sample.department, exact: true }));
    await expect(page.getByText(/直属岗位核定合计 4/)).toBeVisible();
    await openPosition();
    await expect(amount).toHaveText('3');
    await expect(page.getByRole('button', { name: '调编', exact: true })).toHaveCount(0);
    await date308(page, '2027-01-01');
    await expect(amount).toHaveText('2');
    await date308(page, '2027-02-01');
    await expect(amount).toHaveText('3');
    await date308(page, '2027-06-01');
    await expect(amount).toHaveText('4');
    expect((await request(page, 'GET', '/api/v1/positions/' + sample.id)).body.headSourceVersion).toBe(
      definition.body.headSourceVersion,
    );
    await click308(page, page.getByRole('tab', { name: '变更日志', exact: true }));
    const operations = page.locator('tbody.change-operation');
    const audit = operations.filter({
      has: page.locator('[data-label="记录处理"]').filter({ hasText: /^更正记录$/ }),
    });
    await expect(audit.locator('[data-label="原值"]').first()).toHaveText('1');
    await expect(audit.locator('[data-label="新值"]').first()).toHaveText('3');
    const before = operations
      .filter({ has: page.locator('[data-label="记录处理"]').filter({ hasText: /^调编$/ }) })
      .filter({
        has: page.locator('tr:first-child [data-label="生效记录"]').filter({ hasText: '2027-04-01' }),
      });
    await click308(page, before.getByRole('link', { name: '查看当次资料' }).first());
    await expect(page.getByText('正在查看只读历史资料')).toBeVisible();
    await expect(amount).toHaveText('1');
    await click308(page, page.getByRole('link', { name: '返回变更日志' }));
    await expect(audit).toBeVisible();
    await click308(page, page.getByRole('link', { name: '返回列表' }));
    await expect(page.getByLabel('统计日期').and(page.locator('input'))).toHaveValue('2027-04-01');
    await expect(page.getByRole('textbox', { name: '搜索岗位编码或名称' })).toHaveValue(sample.code);
    if (width === 1440 && height === 900) {
      expect(sample.id).toMatch(/^[0-9a-f-]{36}$/);
      await withPolicy(page, async (where, mutate) => {
        const actions = where + " AND action LIKE 'staffingquota.%'";
        try {
          mutate(
            "UPDATE access.subject_grant SET scope_kind='position',scope_id='" +
              sample.id +
              "' WHERE " +
              actions,
          );
          await page.reload();
          await expect(page.getByText(/可见范围 · 岗位核定小计 3/)).toBeVisible();
          mutate(
            "UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'quota') WHERE " + actions,
          );
          await page.reload();
          await expect(page.getByText(/岗位核定小计 受限/)).toBeVisible();
          await openPosition();
          await expect(amount).toHaveText('受限');
        } finally {
          mutate('DELETE FROM access.subject_grant WHERE ' + actions);
        }
      });
    }
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
  });

staffingZoomCases(
  ['position'] as const,
  '315',
  async (page, sample, _family, role) => {
    const button = (name: string) => page.getByRole('button', { name, exact: true });
    await namedMenu308(page, '部门编制分配', '职务配置', '/settings/staffing/quotas', '部门编制统计');
    const search = page.getByRole('textbox', { name: '搜索岗位编码或名称' });
    await search.fill(sample.code);
    await expect(search).toHaveValue(sample.code);
    if (role === 'reader') {
      await search.press('Tab');
      await expect(button('查询')).toBeFocused();
      await page.keyboard.press('Enter');
    } else {
      await click308(page, button('查询'));
    }
    await expect.poll(() => new URL(page.url()).searchParams.get('q')).toBe(sample.code);
    await click308(page, page.getByRole('link', { name: sample.name, exact: true }));
    await expect
      .poll(() => new URLSearchParams(new URL(page.url()).searchParams.get('back')?.split('?')[1]).get('q'), {
        message: '进入岗位后应保留返回统计的条件',
      })
      .toBe(sample.code);
    await expect(page.getByRole('heading', { name: sample.name, exact: true })).toBeVisible();
    if (role === 'admin') {
      await click308(page, button('调编'));
      await page.getByRole('spinbutton', { name: /核定编制/ }).fill('3');
      await page.getByLabel('生效日期').and(page.locator('input')).fill('2027-03-01');
      await page.getByLabel('调整原因').and(page.locator('input')).fill('岗位年度预算');
      await click308(page, button('保存'));
      await expect(page.getByText('自 2027-03-01 起', { exact: true })).toBeVisible();
    } else {
      await expect(button('调编')).toHaveCount(0);
      await click308(page, page.getByRole('tab', { name: '变更日志', exact: true }));
      await click308(page, page.getByRole('link', { name: '查看当次资料' }).first());
      await expect(page.getByText('正在查看只读历史资料')).toBeVisible();
      await click308(page, button('返回当前资料'));
    }
    await click308(page, page.getByRole('link', { name: '返回列表' }));
    await expect.poll(() => new URL(page.url()).searchParams.get('q')).toBe(sample.code);
    await expect(page.locator('.standard-page[aria-busy]')).toHaveAttribute('aria-busy', 'false');
    await expect(page.getByRole('textbox', { name: '搜索岗位编码或名称' })).toHaveValue(sample.code);
    await expect(page.getByRole('link', { name: sample.name, exact: true })).toBeFocused();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
  },
  async (page, kind, family) => {
    const sample = await seedPosition(page, kind, family);
    const receipt = await request(
      page,
      'POST',
      '/api/v1/staffing-quotas',
      {
        positionId: sample.id,
        effectiveDate: '2026-01-01',
        quota: 2,
        reason: '年度编制',
      },
      randomUUID(),
    );
    expect(receipt.status).toBe(200);
    return sample;
  },
);

test('315 两位管理员从同一未核编表单分别保存', async ({ page, browser, baseURL }) => {
  test.setTimeout(180000);
  execFileSync('python3', ['docs/verification/data/315-staffing-quota-administration/second-admin.py']);
  const second = await browser.newContext({ baseURL });
  try {
    const colleague = await second.newPage();
    for (const operator of [page, colleague]) {
      operator.setDefaultTimeout(15000);
      operator.setDefaultNavigationTimeout(60000);
    }
    await open(page);
    await open(colleague, 'quota-315-admin');
    const family = await seed308(page, 'jobfamily');
    for (const [first, next] of [
      [page, colleague],
      [colleague, page],
    ]) {
      const sample = await seedPosition(page, 'position', family.id);
      for (const [index, operator] of [page, colleague].entries()) {
        await namedMenu308(operator, '部门编制分配', '职务配置', '/settings/staffing/quotas', '部门编制统计');
        await operator.getByRole('textbox', { name: '搜索岗位编码或名称' }).fill(sample.code);
        await click308(operator, operator.getByRole('button', { name: '查询', exact: true }));
        await click308(operator, operator.getByRole('link', { name: sample.name, exact: true }));
        await click308(operator, operator.getByRole('button', { name: '核编', exact: true }));
        await operator.getByLabel('生效日期').and(operator.locator('input')).fill('2026-01-01');
        await operator.getByRole('spinbutton', { name: /核定编制/ }).fill(String(2 + index));
        await operator.getByLabel('调整原因').and(operator.locator('input')).fill('年度核编');
      }
      for (const [operator, expected] of [
        [first, 200],
        [next, 409],
      ] as const) {
        const saved = operator.waitForResponse(
          (r) => r.request().method() === 'POST' && r.url().endsWith('/api/v1/staffing-quotas'),
        );
        await click308(operator, operator.getByRole('button', { name: '保存', exact: true }));
        expect((await saved).status()).toBe(expected);
      }
      await expect(next.getByText('岗位已有核编，请刷新后调编', { exact: true })).toBeVisible();
      await click308(next, next.getByRole('button', { name: '重读当前记录' }));
      await expect(next.getByRole('button', { name: '调编', exact: true })).toBeVisible();
    }
  } finally {
    if (browser.isConnected()) await second.close();
  }
});

test('315 所选记录删除、字段撤权与岗位定义独立授权', async ({ page }) => {
  test.setTimeout(90000);
  await open(page);
  const family = await seed308(page, 'jobfamily'),
    sample = await seedPosition(page, 'position', family.id);
  const created = await request(
    page,
    'POST',
    '/api/v1/staffing-quotas',
    { positionId: sample.id, effectiveDate: '2026-01-01', quota: 2, reason: '首次定额' },
    randomUUID(),
  );
  const base = '/api/v1/staffing-quotas/' + created.body.objectId;
  const apr = await request(
    page,
    'POST',
    base + '/adjust',
    {
      expectedSourceVersion: created.body.sourceVersion,
      effectiveDate: '2027-04-01',
      quota: 1,
      reason: '四月原始原因',
    },
    randomUUID(),
  );
  const jun = await request(
    page,
    'POST',
    base + '/adjust',
    {
      expectedSourceVersion: apr.body.sourceVersion,
      effectiveDate: '2027-06-01',
      quota: 4,
      reason: '六月独立定额',
    },
    randomUUID(),
  );
  expect(jun.status).toBe(200);
  const click = (name: string) => click308(page, page.getByRole('button', { name, exact: true }));
  const action = async (name: string) => {
    await click('更多操作');
    await click308(page, page.getByRole('menuitem', { name, exact: true }));
  };
  await namedMenu308(page, '岗位', '职务配置', '/settings/staffing/positions');
  const listBookmark = new URL(page.url());
  listBookmark.searchParams.set('businessAt', '1900-01-01');
  await page.goto(listBookmark.href);
  await expect.poll(() => new URL(page.url()).searchParams.has('businessAt')).toBe(false);
  await page.getByRole('textbox', { name: '搜索编码或名称' }).fill(sample.code);
  await click('查询');
  await click308(page, page.getByRole('link', { name: sample.name, exact: true }));
  await click308(page, page.getByRole('tab', { name: '岗位编制' }));
  await date308(page, '2027-04-01');
  const bookmark = new URL(page.url());
  bookmark.searchParams.set('at', '1900-01-01');
  bookmark.searchParams.set('back', bookmark.searchParams.get('back') + '&businessAt=1900-01-01');
  await page.goto(bookmark.href);
  await expect.poll(() => new URL(page.url()).searchParams.has('at')).toBe(false);
  expect(new URL(page.url()).searchParams.get('back')).not.toContain('businessAt');
  const stale = page.url();
  const navigation = '**/api/v1/positions/*/navigation?*';
  await page.route(navigation, (route) => route.abort('failed'));
  await page.reload();
  const unavailable = page.getByRole('alert').filter({ hasText: '暂时无法切换记录' });
  await expect(unavailable).toBeVisible();
  await expect(page.getByRole('tab', { name: '岗位编制', exact: true })).toBeVisible();
  await page.unroute(navigation);
  await Promise.all([
    page.waitForResponse((response) => response.ok() && new URL(response.url()).pathname.endsWith('/navigation')),
    click308(page, unavailable.getByRole('button', { name: '重试', exact: true })),
  ]);
  await expect(unavailable).toHaveCount(0);
  await action('更正记录');
  const amount = page.getByRole('spinbutton', { name: /核定编制/ }),
    reason = page.getByRole('textbox', { name: /调整原因/ });
  await withPolicy(page, async (where, mutate) => {
    mutate(
      "UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'reason'),write_fields=array_remove(write_fields,'reason') WHERE " +
        where +
        " AND action LIKE 'staffingquota.%'",
    );
    await amount.fill('3');
    await reason.fill('四月草稿不再可见');
    await expect(reason).toHaveCount(0);
    await expect(page.getByText('四月原始原因')).toHaveCount(0);
    await expect(amount).toHaveValue('3');
    await click('按当前权限预览');
    await click('保存');
    await expect(page.getByText('已保存。', { exact: true })).toBeVisible();
  });
  await page.reload();
  await action('删除记录');
  await expect(page.getByText('2027-04-01 至 2027-05-31', { exact: true })).toBeVisible();
  await expect(page.locator('.object-detail-field').filter({ hasText: '删除后核定编制' })).toContainText('2');
  await click('确认删除');
  await expect(page.getByText('已删除。', { exact: true })).toBeVisible();
  await expect(page.getByText('正在查看只读历史资料')).toBeVisible();
  await page.goto(stale);
  await expect(page.getByText('当前资料中没有所选记录，请重新选择。', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: '更多操作' })).toHaveCount(0);
  await click('查看默认记录');
  await expect(page.getByRole('button', { name: '调编' })).toBeVisible();
  await withPolicy(page, async (where, mutate) => {
    mutate('DELETE FROM access.subject_grant WHERE ' + where + " AND action='position.read'");
    await namedMenu308(page, '部门编制分配', '职务配置', '/settings/staffing/quotas', '部门编制统计');
    await expect(page.getByRole('columnheader', { name: '直属岗位核定合计' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: '未核编岗位数' })).toBeVisible();
    await page.getByRole('textbox', { name: '搜索岗位编码或名称' }).fill(sample.code);
    await click('查询');
    await click308(page, page.getByRole('link', { name: sample.name, exact: true }));
    await expect(page.getByRole('button', { name: '调编' })).toBeVisible();
    await expect(page.getByRole('tab', { name: '基本资料' })).toHaveCount(0);
  });
  await withPolicy(page, async (where, mutate) => {
    mutate('DELETE FROM access.subject_grant WHERE ' + where + " AND action='staffingquota.read'");
    await namedMenu308(page, '岗位', '职务配置', '/settings/staffing/positions');
    await page.getByRole('textbox', { name: '搜索编码或名称' }).fill(sample.code);
    await click('查询');
    await click308(page, page.getByRole('link', { name: sample.name, exact: true }));
    await expect(page.getByRole('tab', { name: '基本资料' })).toBeVisible();
    await expect(page.getByRole('tab', { name: '岗位编制' })).toHaveCount(0);
  });
});
