import { randomUUID } from 'node:crypto';
import { type Page, expect, test } from './fixtures';
import { request } from './dated-object-support';
import { choose } from './bu-support';
import { seed308, menu308, enter308, click308, action308, date308, reach308, evidence308, base308, type Kind308 } from './staffing308-support';
import { matrixJourney308 } from './staffing308-journey';

export async function matrixSeed310(page: Page, kind: Kind308, family: string) {
  const tag = 'G2-' + randomUUID().slice(0, 7), gradeIds: string[] = [], gradeNames: string[] = [];
  for (let n = 1; n <= 20; n++) {
    const grade = await seed308(page, 'grade', undefined, { code: `${tag}-G${String(n).padStart(2, '0')}`, name: `${tag}职级${String(n).padStart(2, '0')} 长名称用于检验成员换行与末项可达`, date: '1900-01-01' });
    gradeIds.push(grade.id); gradeNames.push(grade.name);
  }
  const sample = await seed308(page, kind, family, { code: tag, gradeIds, gradeNames });
  for (let n = 1; n <= 20; n++) await seed308(page, kind, family, { code: `${tag}-${String(n).padStart(2, '0')}`, gradeIds, gradeNames });
  return sample;
}
export async function matrixJourney310(...args: Parameters<typeof matrixJourney308>) {
  test.setTimeout(360000);
  await matrixJourney308(...args);
  const [page, sample, family, role] = args, kind = sample.kind, path = base308[kind] + '/' + sample.id;
  await menu308(page, kind);
  await page.getByRole('textbox', { name: '搜索编码或名称', exact: true }).fill(sample.code);
  await click308(page, page.getByRole('button', { name: '查询', exact: true }));
  await expect(page.getByText('1–20 / 21', { exact: true })).toBeVisible();
  await click308(page, page.getByRole('button', { name: '更多操作', exact: true }));
  await click308(page, page.getByRole('menuitem', { name: '列筛选…', exact: true }));
  const read = page.waitForRequest(r => r.method() === 'GET' && new URL(r.url()).pathname === base308[kind]);
  await page.goBack(); await read;
  await expect(page.getByRole('dialog', { name: '列筛选', exact: true })).toHaveCount(0);
  await enter308(page, sample);
  const memberRows = () => page.locator('.object-detail-sheet li').filter({ hasText: '长名称用于检验成员换行与末项可达' });
  await expect(memberRows()).toHaveCount(20);
  // Job is a set: its server order need not equal the fixture's insertion order.
  // Reach the actual final rendered member for both objects and both read modes.
  await reach308(page, memberRows().last());
  await click308(page, page.getByRole('tab', { name: '变更日志', exact: true }));
  if (role === 'reader') {
    await expect(page.getByRole('button', { name: '更多操作', exact: true })).toHaveCount(0);
    await click308(page, page.getByRole('link', { name: '查看当次资料', exact: true }));
    await expect(memberRows()).toHaveCount(20);
    const last = memberRows().last();
    await reach308(page, last);
    await test.info().attach('actual-final-member', { body: JSON.stringify({ kind, count: 20, text: await last.innerText() }), contentType: 'application/json' });
    await evidence308(page, 'G2-members-reader'); return;
  }
  const original = (await request(page, 'GET', path)).body.headSourceVersion;
  await action308(page, '更正记录');
  await expect(page.getByRole('tab', { name: '基本资料', exact: true })).toHaveAttribute('aria-selected', 'true');
  const memberStyle = await page.locator('#staffing-gradeIds li .MuiTypography-root').first().evaluate(el => ({
    family: getComputedStyle(el).fontFamily, size: getComputedStyle(el).fontSize,
    referenceFamily: getComputedStyle(document.querySelector('#staffing-name')!).fontFamily,
    referenceSize: getComputedStyle(document.querySelector('#staffing-name')!).fontSize,
  }));
  expect(memberStyle.family).toBe(memberStyle.referenceFamily);
  expect(memberStyle.size).toBe(memberStyle.referenceSize);
  await test.info().attach('member-form-style', { body: JSON.stringify(memberStyle), contentType: 'application/json' });
  const lastRemove = page.getByRole('button', { name: '移除第20个职级', exact: true });
  await reach308(page, lastRemove);
  if (kind === 'gradeladder') {
    const up = page.getByRole('button', { name: '上移第20个职级', exact: true });
    await click308(page, up); await expect(page.getByRole('button', { name: '上移第19个职级', exact: true })).toBeFocused();
    await page.keyboard.press('Enter');
  } else await click308(page, lastRemove);
  await click308(page, page.getByRole('button', { name: '取消', exact: true }));
  await expect(page.getByRole('dialog', { name: '尚有未保存修改', exact: true })).toBeVisible();
  await click308(page, page.getByRole('button', { name: '继续编辑', exact: true }));
  await click308(page, page.getByRole('button', { name: '取消', exact: true }));
  await click308(page, page.getByRole('button', { name: '放弃修改', exact: true }));
  await expect(page.getByRole('tab', { name: '变更日志', exact: true })).toHaveAttribute('aria-selected', 'true');
  await expect(page.getByRole('button', { name: '更多操作', exact: true })).toBeFocused();
  expect((await request(page, 'GET', path)).body.headSourceVersion).toBe(original);
  await action308(page, '更正记录');
  if (kind === 'gradeladder') await click308(page, page.getByRole('button', { name: '上移第20个职级', exact: true }));
  else await click308(page, page.getByRole('button', { name: '移除第20个职级', exact: true }));
  await click308(page, page.getByRole('button', { name: '保存', exact: true }));
  await expect(page.locator('.object-form')).toHaveCount(0);
  const updated = (await request(page, 'GET', path)).body;
  expect(updated.headSourceVersion).not.toBe(original);
  if (kind === 'gradeladder') expect(updated.selectedRecord.gradeIds).toEqual([...sample.gradeIds!.slice(0, 18), sample.gradeIds![19], sample.gradeIds![18]]);
  else expect(updated.selectedRecord.gradeIds).toHaveLength(19);
  await click308(page, page.getByRole('tab', { name: '变更日志', exact: true }));
  await click308(page, page.getByRole('button', { name: '新增记录', exact: true }));
  await expect(page.locator('.object-form')).toHaveCount(1);
  await click308(page, page.getByRole('button', { name: '取消', exact: true }));
  await expect(page.getByRole('tab', { name: '变更日志', exact: true })).toHaveAttribute('aria-selected', 'true');
  // Commit a future independent membership and a fixed inactive interval.
  await click308(page, page.getByRole('button', { name: '新增记录', exact: true }));
  const date = page.getByLabel('生效日期').and(page.locator('input'));
  await reach308(page, date); await date.fill('2035-01-01');
  await choose(page, '操作原因', '名称调整');
  const name = page.getByRole('textbox', { name: '名称', exact: true }); await reach308(page, name); await name.fill('未来 ' + sample.code);
  await click308(page, page.getByRole('button', { name: '保存', exact: true })); await expect(page.locator('.object-form')).toHaveCount(0);
  await date308(page, '2026-01-01'); await action308(page, '插入停用与重新启用');
  for (const [label, value] of [['停用日期', '2027-01-01'], ['重新启用日期', '2028-01-01']]) { const target = page.getByLabel(label).and(page.locator('input')); await reach308(page, target); await target.fill(value); }
  await reach308(page, page.getByRole('button', { name: kind === 'job' ? '移除第19个职级' : '移除第20个职级', exact: true }));
  await click308(page, page.getByRole('button', { name: '保存', exact: true })); await expect(page.locator('.object-form')).toHaveCount(0);
  expect((await request(page, 'GET', path)).body.timeline).toHaveLength(4);
  await action308(page, '维护停用与重新启用');
  for (const [label, value] of [['停用日期', '2027-02-01'], ['重新启用日期', '2028-02-01']]) { const target = page.getByLabel(label).and(page.locator('input')); await reach308(page, target); await target.fill(value); }
  await click308(page, page.getByRole('button', { name: '保存', exact: true })); await expect(page.locator('.object-form')).toHaveCount(0);
  expect((await request(page, 'GET', path)).body.timeline.map((r: { effectiveDate: string }) => r.effectiveDate)).toEqual(['2026-01-01', '2027-02-01', '2028-02-01', '2035-01-01']);
  await action308(page, '删除停用与重新启用'); await click308(page, page.getByRole('button', { name: '确认删除', exact: true })); await expect(page.locator('.object-form')).toHaveCount(0);
  await expect(page.locator('.object-detail-sheet')).toBeVisible();
  const settled = page.getByRole('heading', { name: sample.name, level: 1, exact: true });
  await expect(settled).toBeVisible();
  expect((await request(page, 'GET', path)).body.timeline).toHaveLength(2);
  // The same normal menu covers redating, a single deletion and standalone
  // lifecycle forms in every administrator viewport, including native zoom.
  await date308(page, '2035-01-01'); await action308(page, '更正记录');
  const futureDate = page.getByLabel('生效日期').and(page.locator('input'));
  await reach308(page, futureDate); await futureDate.fill('2036-01-01');
  await click308(page, page.getByRole('button', { name: '保存', exact: true })); await expect(page.locator('.object-form')).toHaveCount(0);
  expect((await request(page, 'GET', path)).body.timeline.map((r: { effectiveDate: string }) => r.effectiveDate)).toEqual(['2026-01-01', '2036-01-01']);
  await action308(page, '删除记录'); await click308(page, page.getByRole('button', { name: '确认删除', exact: true })); await expect(page.locator('.object-form')).toHaveCount(0);
  for (const [action, day, reason] of [['停用', '2027-01-01', kind === 'job' ? '职务停用' : '职级阶梯停用'], ['重新启用', '2028-01-01', kind === 'job' ? '职务恢复' : '职级阶梯恢复']]) {
    await action308(page, action);
    const field = page.getByLabel('生效日期').and(page.locator('input')); await reach308(page, field); await field.fill(day);
    await choose(page, '操作原因', reason);
    await click308(page, page.getByRole('button', { name: '保存', exact: true })); await expect(page.locator('.object-form')).toHaveCount(0);
  }
  expect((await request(page, 'GET', path)).body.timeline).toHaveLength(3);
  for (const day of ['2028-01-01', '2027-01-01']) {
    await date308(page, day); await action308(page, '删除记录');
    await click308(page, page.getByRole('button', { name: '确认删除', exact: true })); await expect(page.locator('.object-form')).toHaveCount(0);
  }
  expect((await request(page, 'GET', path)).body.timeline).toHaveLength(1);
  await expect(settled).toBeVisible();
  await reach308(page, settled);
  await evidence308(page, 'G2-members-and-dates');
  void family;
}
