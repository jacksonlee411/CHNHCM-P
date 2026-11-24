import { test, expect, openMoreActions, type Page } from './fixtures';
import { open, create, add, life, readyToSave, read, select } from './bu-support';
import { mkdirSync, writeFileSync } from 'node:fs';

async function correct(page: Page, date: string, field: '中文名称' | '英文名称', value: string, successors = false) {
  await select(page, date);
  await openMoreActions(page);
  await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
  await page.getByRole('textbox', { name: field, exact: true }).fill(value);
  await readyToSave(page);
  if (successors) await expect(page.getByText('后续独立资料记录保留原值；如需修改，请明确更正相应记录。', { exact: true })).toHaveCount(0);
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.locator('.object-form')).toHaveCount(0);
}

function evidence(name: string, value: unknown) {
  mkdirSync('.local/verification/t07/business', { recursive: true });
  writeFileSync(`.local/verification/t07/business/${name}.json`, JSON.stringify(value, null, 2));
}

test('TL050 F11英文纠错保持曾部分输入的后继，直接保存保护', async ({ page }) => {
  await open(page); const code = await create(page);
  await correct(page, '2026-01-01', '英文名称', 'Nroth');
  await add(page, '2026-06-01', '乙'); // English deliberately omitted: persisted independent default.
  await life(page, 'DEACTIVATE', '2026-07-01');
  await life(page, 'REACTIVATE', '2026-08-01'); // Confirm inherited default, without editing it.
  const original = await read(page);
  await correct(page, '2026-01-01', '英文名称', 'North', true);
  const current = await read(page);
  expect(current.timeline.map((r: any) => [r.effectiveDate, r.name, r.nameEn, r.state])).toEqual([
    ['2026-01-01', '甲', 'North', 'active'],
    ['2026-06-01', '乙', 'Nroth', 'active'],
    ['2026-07-01', '乙', 'Nroth', 'inactive'],
    ['2026-08-01', '乙', 'Nroth', 'active'],
  ]);
  for (const date of ['2026-01-01', '2026-05-31']) expect((await read(page, date)).selectedRecord.nameEn).toBe('North');
  expect((await read(page, '2026-01-01', original.headSourceVersion)).timeline).toEqual(original.timeline);
  await page.reload();
  await select(page, '2026-06-01');
  await expect(page.getByText('Nroth', { exact: true }).first()).toBeVisible();
  evidence('TL050', { code, buId: current.id, original, current });
});

test('TL249正常菜单完成创建启停更正改日删除，逐份来源保持', async ({ page }) => {
  await open(page); const code = await create(page); const history: any[] = [];
  const keep = async () => { const v = await read(page); history.push(v); return v; };
  await keep();
  await life(page, 'DEACTIVATE', '2026-04-01'); await keep();
  await life(page, 'REACTIVATE', '2026-05-01', '乙', 'Beta'); await keep();
  await add(page, '2026-06-01', '丙', 'Gamma'); await keep();
  await correct(page, '2026-05-01', '中文名称', '乙2', true); await keep();
  await select(page, '2026-05-01'); await openMoreActions(page);
  await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
  await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-05-15');
  await readyToSave(page); await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.locator('.object-form')).toHaveCount(0); await keep();
  await select(page, '2026-06-01'); await openMoreActions(page);
  await page.getByRole('menuitem', { name: '删除此记录', exact: true }).click();
  await readyToSave(page); await page.getByRole('button', { name: '确认删除', exact: true }).click();
  await expect(page.locator('.object-form')).toHaveCount(0); const final = await keep();
  expect(new Set(history.map(v => v.headSourceVersion)).size).toBe(7);
  for (const [date, name, en, state] of [
    ['2026-03-31', '甲', 'Alpha', 'active'],
    ['2026-04-01', '甲', 'Alpha', 'inactive'],
    ['2026-05-14', '甲', 'Alpha', 'inactive'],
    ['2026-05-15', '乙2', 'Beta', 'active'],
    ['2026-06-01', '乙2', 'Beta', 'active'],
  ]) {
    const v = (await read(page, date)).selectedRecord;
    expect([v.name, v.nameEn, v.state]).toEqual([name, en, state]);
  }
  for (const old of history) expect((await read(page, '2026-11-01', old.headSourceVersion)).timeline).toEqual(old.timeline);
  await page.reload(); await select(page, '2026-05-15');
  await expect(page.getByText('乙2', { exact: true }).first()).toBeVisible();
  evidence('TL249', { code, buId: final.id, history, final });
});
