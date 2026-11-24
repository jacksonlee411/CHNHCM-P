import { expect, test, type Page, type Locator } from './fixtures';
import { choose } from './bu-support';
import { request } from './dated-object-support';
import { matrixJourney308 } from './staffing308-journey';
import { seed308, enter308, action308, click308, date308, reach308, evidence308, menu308 } from './staffing308-support';

// Every Grade viewport inherits the actual BU comparison and the 308 page
// journey, then exercises committed date/lifecycle/history operations as well.
export async function matrixJourney309(...args: Parameters<typeof matrixJourney308>) {
  await matrixJourney308(...args);
  const [page, sample, , role] = args;
  await menu308(page, 'grade');
  await page.getByRole('textbox', { name: '搜索编码或名称', exact: true }).fill(sample.code);
  await click308(page, page.getByRole('button', { name: '查询', exact: true }));
  await expect(page.getByText('1–20 / 21', { exact: true })).toBeVisible();
  await click308(page, page.getByRole('button', { name: '更多操作', exact: true }));
  await click308(page, page.getByRole('menuitem', { name: '列筛选…', exact: true }));
  const reread = page.waitForRequest(r => r.method() === 'GET' && new URL(r.url()).pathname === '/api/v1/grades');
  await page.goBack();
  await reread;
  await expect(page.getByRole('dialog', { name: '列筛选', exact: true })).toHaveCount(0);
  await expect(page.locator('.standard-page[aria-busy]')).toHaveAttribute('aria-busy', 'false');
  await enter308(page, sample);
  await click308(page, page.getByRole('tab', { name: '变更日志', exact: true }));
  if (role === 'reader') {
    await expect(page.getByRole('button', { name: '更多操作', exact: true })).toHaveCount(0);
    await evidence308(page, 'grade-back-query-reader');
    return;
  }
  const before = (await request(page, 'GET', '/api/v1/grades/' + sample.id)).body.headSourceVersion;
  await action308(page, '更正记录');
  await expect(page.getByRole('tab', { name: '基本资料', exact: true })).toHaveAttribute('aria-selected', 'true');
  await click308(page, page.getByRole('button', { name: '取消', exact: true }));
  await expect(page.getByRole('tab', { name: '变更日志', exact: true })).toHaveAttribute('aria-selected', 'true');
  await expect(page.getByRole('button', { name: '更多操作', exact: true })).toBeFocused();
  expect((await request(page, 'GET', '/api/v1/grades/' + sample.id)).body.headSourceVersion).toBe(before);
  await action308(page, '更正记录');
  const description = page.getByRole('textbox', { name: '说明', exact: true });
  await reach308(page, description);
  await description.fill('日志直接维护后的说明');
  await click308(page, page.getByRole('button', { name: '保存', exact: true }));
  await expect(page.locator('.object-form')).toHaveCount(0);
  expect((await request(page, 'GET', '/api/v1/grades/' + sample.id + '/changes')).body.total).toBe(2);
  test.setTimeout(300000);
  const o = await seed308(page, 'grade', undefined, { date: '2020-01-01' });
  await enter308(page, o);
  const fill = async (target: Locator, value: string) => { await reach308(page, target); await target.fill(value); };
  const date = (label = '生效日期') => page.getByLabel(label).and(page.locator('input'));
  const save = async (label = '保存') => {
    await click308(page, page.getByRole('button', { name: label, exact: true }));
    await expect(page.locator('.object-form')).toHaveCount(0);
    await expect(page.locator('.object-detail-sheet')).toBeVisible();
  };
  const detail = async () => (await request(page, 'GET', '/api/v1/grades/' + o.id)).body;
  await action308(page, '新增记录');
  await fill(date(), '2035-01-01');
  await choose(page, '操作原因', '名称调整');
  await fill(page.getByRole('textbox', { name: '名称', exact: true }), '未来独立 ' + o.code);
  await save();
  await date308(page, '2020-01-01');
  await action308(page, '插入停用与重新启用');
  await fill(date('停用日期'), '2021-01-01');
  await fill(date('重新启用日期'), '2022-01-01');
  await expect(page.getByRole('combobox', { name: '停用原因', exact: true })).toHaveText('职级停用');
  await expect(page.getByRole('combobox', { name: '重新启用原因', exact: true })).toHaveText('职级恢复');
  await save();
  expect((await detail()).timeline).toHaveLength(4);
  await action308(page, '维护停用与重新启用');
  await fill(date('停用日期'), '2021-02-01');
  await save();
  await action308(page, '删除停用与重新启用');
  await save('确认删除');
  expect((await detail()).timeline).toHaveLength(2);
  await date308(page, '2035-01-01');
  await action308(page, '更正记录');
  await fill(date(), '2024-01-01');
  await save();
  await action308(page, '停用');
  await fill(date(), '2025-01-01');
  await expect(page.getByRole('combobox', { name: '操作原因', exact: true })).toHaveText('职级停用');
  await save();
  await action308(page, '重新启用');
  await fill(date(), '2026-01-01');
  await expect(page.getByRole('combobox', { name: '操作原因', exact: true })).toHaveText('职级恢复');
  await save();
  await date308(page, '2024-01-01');
  await action308(page, '删除记录');
  await save('确认删除');
  expect((await detail()).timeline).toHaveLength(3);
  // Inspect every committed link, including the render before the snapshot URL settles.
  const unpinnedReturn = await page.evaluateHandle(() => {
    const state = { seen: false };
    const observer = new MutationObserver(() => {
      for (const link of document.querySelectorAll<HTMLAnchorElement>('.change-operation a')) {
        const target = new URL(link.href).searchParams.get('logReturn');
        if (target && !new URL(target, location.origin).searchParams.has('logThrough')) state.seen = true;
      }
    });
    observer.observe(document.body, { subtree: true, childList: true, attributes: true, attributeFilter: ['href'] });
    return { state, observer };
  });
  await click308(page, page.getByRole('tab', { name: '变更日志', exact: true }));
  const original = page.getByRole('link', { name: '查看删除前原件', exact: true }).first();
  await click308(page, original);
  await reach308(page, page.getByText('正在查看只读历史资料', { exact: true }));
  await expect(page.getByRole('heading', { name: '未来独立 ' + o.code, exact: true })).toBeVisible();
  await click308(page, page.getByRole('link', { name: '返回变更日志', exact: true }));
  await expect(original).toBeFocused();
  expect(await unpinnedReturn.evaluate(({ state, observer }) => {
    observer.disconnect();
    return state.seen;
  })).toBe(false);
  await evidence308(page, 'grade-complete-maintenance');
}
