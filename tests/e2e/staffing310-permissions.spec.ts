import { randomUUID } from 'node:crypto';
import { execFileSync } from 'node:child_process';
import { test, expect } from './fixtures';
import { open } from './bu-support';
import { withPolicy } from './grants';
import { request } from './dated-object-support';
import { seed308, enter308, action308, click308, menu308, base308 } from './staffing308-support';

for (const kind of ['job', 'gradeladder'] as const) {
  test(`310 ${kind} 精确Grant：整集合、历史与保留获权草稿`, async ({ page }) => {
    test.setTimeout(120000); await open(page);
    const family = await seed308(page, 'jobfamily'), grade = await seed308(page, 'grade', undefined, { date: '1900-01-01' });
    const sample = await seed308(page, kind, family.id, { gradeIds: [grade.id], gradeNames: [grade.name] });
    await enter308(page, sample); const path = base308[kind] + '/' + sample.id;
    await withPolicy(page, async (where, mutate) => {
      mutate(`UPDATE access.subject_grant SET write_fields=array_remove(write_fields,'gradeIds') WHERE ${where} AND action='${kind}.record.correct'`);
      await page.reload(); await action308(page, '更正记录');
      await expect(page.getByRole('button', { name: '选择职级', exact: true })).toHaveCount(0);
      await page.getByRole('textbox', { name: '名称', exact: true }).fill('仅资料 ' + sample.code);
      await click308(page, page.getByRole('button', { name: '保存', exact: true })); await expect(page.locator('.object-form')).toHaveCount(0);
      expect((await request(page, 'GET', path)).body.selectedRecord.gradeIds).toEqual([grade.id]);
    });
    const extra = await seed308(page, 'grade', undefined, { date: '1900-01-01' });
    const sourceBeforeRevoke = (await request(page, 'GET', path)).body.headSourceVersion;
    await page.reload(); await action308(page, '更正记录');
    await click308(page, page.getByRole('button', { name: '选择职级', exact: true }));
    const picker = page.getByRole('dialog', { name: '选择职级', exact: true });
    await picker.getByRole('textbox', { name: '查找职级', exact: true }).fill(extra.code);
    await click308(page, picker.getByRole('checkbox', { name: new RegExp(extra.code, 'i') }));
    await click308(page, picker.getByRole('button', { name: '完成选择', exact: true }));
    const draftName = '保存时保留名称 ' + sample.code;
    await page.getByRole('textbox', { name: '英文名称', exact: true }).fill('withdrawn draft');
    const refreshed = page.waitForResponse(r => r.url().endsWith('/grade-reference-context') &&
      r.request().postDataJSON()?.changes?.name === draftName);
    await page.getByRole('textbox', { name: '名称', exact: true }).fill(draftName);
    expect((await refreshed).status()).toBe(200);
    await expect(page.locator('.object-form')).toContainText(extra.name);
    await withPolicy(page, async (where, mutate) => {
      mutate(`UPDATE access.subject_grant SET write_fields=array_remove(write_fields,'gradeIds') WHERE ${where} AND action='${kind}.record.correct'`);
      mutate(`UPDATE access.subject_grant SET write_fields=array_remove(write_fields,'nameEn') WHERE ${where} AND action='${kind}.record.correct'`);
      const rejected = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname.startsWith(path + '/records/') && r.url().endsWith('/correct'));
      await click308(page, page.getByRole('button', { name: '保存', exact: true }));
      expect((await rejected).status()).toBe(403);
      await expect(page.getByText('权限已变化，已重新读取当前可见资料。', { exact: true })).toBeVisible();
      await expect(page.getByRole('textbox', { name: '名称', exact: true })).toHaveValue(draftName);
      await expect(page.getByRole('textbox', { name: '英文名称', exact: true })).toHaveCount(0);
      await expect(page.getByRole('button', { name: '选择职级', exact: true })).toHaveCount(0);
      const fresh = (await request(page, 'GET', path)).body;
      expect(fresh.headSourceVersion).toBe(sourceBeforeRevoke);
      expect(fresh.selectedRecord.gradeIds).toEqual([grade.id]);
      await expect(page.locator('.object-detail-sheet')).not.toContainText(extra.name);
      const saved = page.waitForResponse(r => r.url().endsWith('/correct'));
      await click308(page, page.getByRole('button', { name: '保存', exact: true }));
      const response = await saved;
      expect(response.status()).toBe(200);
      expect(response.request().postDataJSON().changes).toEqual({ name: draftName });
      await expect(page.locator('.object-form')).toHaveCount(0);
      const updated = (await request(page, 'GET', path)).body;
      expect(updated.selectedRecord.name).toBe(draftName);
      expect(updated.selectedRecord.nameEn).toBe(fresh.selectedRecord.nameEn);
      expect(updated.selectedRecord.gradeIds).toEqual([grade.id]);
      await test.info().attach('membership-write-revoked-before-save', { body: JSON.stringify({ kind, response: 403, sourceUnchanged: true, originalMembersUnchanged: true }), contentType: 'application/json' });
    });
    await page.reload(); await action308(page, '更正记录');
    await expect(page.locator('.object-form')).toContainText(grade.name);
    await page.getByRole('textbox', { name: '名称', exact: true }).fill('保留仍获权输入');
    await withPolicy(page, async (where, mutate) => {
      mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE ${where} AND action='grade.read'`);
      await page.getByLabel('生效日期').and(page.locator('input')).fill('2026-02-01');
      await expect(page.locator('.object-form')).toContainText('职级资料受限');
      await expect(page.locator('.object-form')).not.toContainText(grade.name);
      await expect(page.getByRole('button', { name: '移除第1个职级', exact: true })).toHaveCount(0);
      await expect(page.getByRole('textbox', { name: '名称', exact: true })).toHaveValue('保留仍获权输入');
      await expect(page.getByRole('button', { name: '选择职级', exact: true })).toHaveCount(0);
      await click308(page, page.getByRole('button', { name: '取消', exact: true })); await click308(page, page.getByRole('button', { name: '放弃修改', exact: true }));
      await expect(page.locator('.object-form')).toHaveCount(0);
      await expect(page.locator('.object-detail-sheet')).not.toContainText(grade.name);
      await expect(page.getByText('职级资料受限', { exact: true })).toBeVisible();
    });
    await page.reload(); await click308(page, page.getByRole('tab', { name: '变更日志', exact: true }));
    await click308(page, page.getByRole('link', { name: '查看当次资料', exact: true }).first());
    await withPolicy(page, async (where, mutate) => {
      mutate(`DELETE FROM access.subject_grant WHERE ${where} AND action='grade.history.read'`);
      await page.reload(); await expect(page.getByText('职级资料受限', { exact: true })).toBeVisible();
      await expect(page.locator('.object-detail-sheet')).not.toContainText(grade.name);
      await expect(page.getByRole('button', { name: '更多操作', exact: true })).toHaveCount(0);
    });
  });

  test(`310 ${kind} 候选日期、迟到响应和503不覆盖输入`, async ({ page }) => {
    test.setTimeout(90000); await open(page);
    const family = await seed308(page, 'jobfamily'), grade = await seed308(page, 'grade', undefined, { date: '1900-01-01', name: '原职级名称 ' + randomUUID().slice(0, 8) });
    const g = (await request(page, 'GET', base308.grade + '/' + grade.id)).body;
    expect(g.timeline[0].name).toBe(grade.name);
    await test.info().attach('candidate-fixture', { body: JSON.stringify({ beforeName: grade.name, afterName: '未来职级 ' + grade.code, beforeDate: '1900-01-01', afterDate: '2030-01-01' }), contentType: 'application/json' });
    expect((await request(page, 'POST', base308.grade + '/' + grade.id + '/records', { expectedSourceVersion: g.headSourceVersion, anchorRecordId: g.selectedRecord.recordId, effectiveDate: '2030-01-01', operationType: 'CHANGE', operationReason: 'NAME_ADJUSTMENT', changes: { name: '未来职级 ' + grade.code } }, randomUUID())).status).toBe(200);
    const sample = await seed308(page, kind, family.id, { gradeIds: [grade.id], gradeNames: [grade.name] });
    await enter308(page, sample);
    let release!: () => void, entered!: () => void;
    const gate = new Promise<void>(resolve => release = resolve), started = new Promise<void>(resolve => entered = resolve);
    let first = true;
    const pattern = '**' + base308[kind] + '/grade-reference-context';
    await page.route(pattern, async route => {
      if (first) { first = false; const response = await route.fetch(); entered(); await gate; await route.fulfill({ response }).catch(() => {}); }
      else await route.continue();
    });
    try {
      await action308(page, '更正记录'); await started;
      await page.getByRole('textbox', { name: '名称', exact: true }).fill('独立输入保留');
      await page.getByLabel('生效日期').and(page.locator('input')).fill('2035-01-01');
      await expect(page.locator('.object-form')).toContainText('未来职级 ' + grade.code);
      release(); await expect(page.locator('.object-form')).not.toContainText(grade.name);
      await expect(page.getByRole('textbox', { name: '名称', exact: true })).toHaveValue('独立输入保留');
    } finally { release(); await page.unroute(pattern); }
    await page.route(pattern, route => route.fulfill({ status: 503, json: { code: 'M03_DEPENDENCY_UNAVAILABLE', message: '候选读取故障310' } }));
    await page.getByLabel('生效日期').and(page.locator('input')).fill('2036-01-01');
    await expect(page.getByText('候选读取故障310', { exact: true })).toBeVisible();
    await expect(page.locator('.object-form')).not.toContainText('未来职级 ' + grade.code);
    await expect(page.getByRole('textbox', { name: '名称', exact: true })).toHaveValue('独立输入保留');
    await page.unroute(pattern); await click308(page, page.getByRole('button', { name: '重试', exact: true }));
    await expect(page.locator('.object-form')).toContainText('未来职级 ' + grade.code);
    await click308(page, page.getByRole('button', { name: '取消', exact: true })); await click308(page, page.getByRole('button', { name: '放弃修改', exact: true }));
    expect((await request(page, 'GET', base308[kind] + '/' + sample.id)).body.timeline).toHaveLength(1);
  });

  test(`310 ${kind} 集合保存丢失响应、未知离开和换账号隔离`, async ({ page }) => {
    test.setTimeout(90000); await open(page);
    const family = await seed308(page, 'jobfamily'), a = await seed308(page, 'grade', undefined, { date: '1900-01-01' }), b = await seed308(page, 'grade', undefined, { date: '1900-01-01' });
    const sample = await seed308(page, kind, family.id, { gradeIds: [a.id, b.id], gradeNames: [a.name, b.name] });
    await enter308(page, sample); await action308(page, '更正记录');
    await click308(page, page.getByRole('button', { name: '移除第2个职级', exact: true }));
    let writes = 0;
    await page.route('**' + base308[kind] + '/**/correct', async route => { writes++; await route.fetch(); await route.abort('failed'); });
    const resolvePattern = '**' + base308[kind] + '/submissions/*/resolve';
    await page.route(resolvePattern, route => route.fulfill({ status: 202, json: { kind, submissionId: new URL(route.request().url()).pathname.split('/').at(-2), outcome: 'pending' } }));
    await click308(page, page.getByRole('button', { name: '保存', exact: true }));
    await expect(page.getByText('保存结果尚未确认，请使用“查看保存结果”。')).toBeVisible();
    await click308(page, page.getByRole('button', { name: '取消', exact: true }));
    const unknown = page.getByRole('dialog', { name: '保存结果尚未确认', exact: true });
    await expect(unknown).toContainText('离开不会撤销已提交的保存操作');
    await click308(page, unknown.getByRole('button', { name: '离开此页', exact: true }));
    await page.reload(); await expect(page.getByRole('button', { name: '查看保存结果', exact: true })).toBeVisible();
    await page.context().clearCookies(); await open(page, 'reader');
    await enter308(page, sample); await expect(page.getByRole('button', { name: '查看保存结果', exact: true })).toHaveCount(0);
    await page.context().clearCookies(); await open(page);
    await page.unroute(resolvePattern); await enter308(page, sample);
    await click308(page, page.getByRole('button', { name: '查看保存结果', exact: true }));
    await expect(page.getByRole('button', { name: '查看保存结果', exact: true })).toHaveCount(0);
    expect(writes).toBe(1);
    const detail = (await request(page, 'GET', base308[kind] + '/' + sample.id)).body;
    expect(detail.selectedRecord.gradeIds).toHaveLength(1);
    expect((await request(page, 'GET', base308[kind] + '/' + sample.id + '/changes')).body.items).toHaveLength(2);
  });
}

test('310 gradeladder 固定选择跨页、成员CSV与源撤权', async ({ page }) => {
  test.setTimeout(120000); await open(page);
  const tag = 'CSV310-' + randomUUID().slice(0, 7);
  const a = await seed308(page, 'grade', undefined, { date: '1900-01-01', name: '初级 ' + tag });
  const b = await seed308(page, 'grade', undefined, { date: '1900-01-01', name: '中级,含逗号 ' + tag });
  const options = { gradeIds: [b.id, a.id], gradeNames: [b.name, a.name] };
  const objects = [];
  for (let n = 1; n <= 21; n++) objects.push(await seed308(page, 'gradeladder', undefined, { ...options, code: `${tag}-${String(n).padStart(3, '0')}` }));
  await menu308(page, 'gradeladder');
  await page.getByRole('textbox', { name: '搜索编码或名称', exact: true }).fill(tag);
  await click308(page, page.getByRole('button', { name: '查询', exact: true }));
  await expect(page.getByText('1–20 / 21', { exact: true })).toBeVisible();
  await click308(page, page.getByRole('checkbox', { name: '全选查询结果', exact: true }));
  const row = (name: string) => page.getByRole('row').filter({ has: page.getByRole('link', { name, exact: true }) });
  await click308(page, row(objects[0].name).getByRole('checkbox'));
  const later = await seed308(page, 'gradeladder', undefined, { ...options, code: tag + '-022' });
  await click308(page, page.getByRole('button', { name: '下一页', exact: true }));
  await expect(page.getByText('21–22 / 22', { exact: true })).toBeVisible();
  await expect(row(objects[20].name).getByRole('checkbox')).toBeChecked();
  await expect(row(later.name).getByRole('checkbox')).not.toBeChecked();
  await click308(page, row(objects[20].name).getByRole('checkbox'));
  await click308(page, page.getByRole('button', { name: '更多操作', exact: true }));
  const download = page.waitForEvent('download');
  await click308(page, page.getByRole('menuitem', { name: '导出所选（19）', exact: true }));
  let csv = '';
  for await (const chunk of (await (await download).createReadStream())!) csv += chunk;
  const rows = JSON.parse(execFileSync('python3', ['-c', 'import sys,csv,json; print(json.dumps(list(csv.reader(sys.stdin))))'], { input: csv.replace(/^\ufeff/, ''), encoding: 'utf8' }));
  expect(rows[0]).toEqual(['编码', '名称', '英文名称', '说明', '状态', '生效日期', '职级成员']);
  expect(rows).toHaveLength(20);
  for (let i = 1; i <= 19; i++) {
    const expected = objects[i];
    expect(rows[i].slice(0, 6)).toEqual([expected.code.toUpperCase(), expected.name, 'English ' + expected.code, '说明第一行\n说明第二行', '启用', '2026-01-01']);
    expect(JSON.parse(rows[i][6])).toEqual([[b.code.toUpperCase(), b.name], [a.code.toUpperCase(), a.name]]);
  }
  await withPolicy(page, async (where, mutate) => {
    mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'code') WHERE ${where} AND action='grade.read'`);
    await click308(page, page.getByRole('button', { name: '更多操作', exact: true }));
    const rejected = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === base308.gradeladder + '/export');
    await click308(page, page.getByRole('menuitem', { name: '导出所选（19）', exact: true }));
    expect((await rejected).status()).toBe(403);
    await expect(page.getByText('职级编码不可读，无法生成完整文件', { exact: true })).toBeVisible();
  });
});

test('310 gradeladder 新建时从无权历史日改为有权当日', async ({ page }) => {
  await open(page);
  const grade = await seed308(page, 'grade', undefined, { date: '1900-01-01' });
  const base = base308.gradeladder;
  const today = (await request(page, 'GET', base + '/context')).body.currentDate;
  const code = 'DATE-' + randomUUID().slice(0, 8);
  await withPolicy(page, async (where, mutate) => {
    mutate(`DELETE FROM access.subject_grant WHERE ${where} AND action='grade.history.read'`);
    await menu308(page, 'gradeladder');
    await click308(page, page.getByRole('link', { name: '新建', exact: true }));
    await expect(page.getByText('职级资料受限', { exact: true })).toBeVisible();
    await page.getByRole('textbox', { name: '编码', exact: true }).fill(code);
    await page.getByRole('textbox', { name: '名称', exact: true }).fill('当日阶梯');
    const allowed = page.waitForResponse(r => r.url().endsWith('/grade-reference-context') &&
      r.request().postDataJSON()?.effectiveDate === today);
    await page.getByLabel('生效日期').and(page.locator('input')).fill(today);
    expect((await allowed).status()).toBe(200);
    await click308(page, page.getByRole('button', { name: '选择职级', exact: true }));
    const picker = page.getByRole('dialog', { name: '选择职级', exact: true });
    await picker.getByRole('textbox', { name: '查找职级', exact: true }).fill(grade.code);
    await click308(page, picker.getByRole('checkbox', { name: new RegExp(grade.code, 'i') }));
    await click308(page, picker.getByRole('button', { name: '完成选择', exact: true }));
    const save = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === base);
    await click308(page, page.getByRole('button', { name: '保存', exact: true }));
    const response = await save;
    expect(response.status()).toBe(201);
    const result = (await request(page, 'GET', base + '/' + (await response.json()).objectId)).body;
    expect(result.selectedRecord).toMatchObject({ name: '当日阶梯', effectiveDate: today, gradeIds: [grade.id] });
  });
});

test('310 gradeladder 同名职级换序按当前编码读取权显示', async ({ page }) => {
  await open(page);
  const name = '同名职级';
  const a = await seed308(page, 'grade', undefined, { date: '1900-01-01', name });
  const b = await seed308(page, 'grade', undefined, { date: '1900-01-01', name });
  const sample = await seed308(page, 'gradeladder', undefined, { gradeIds: [a.id, b.id], gradeNames: [name, name] });
  await enter308(page, sample);
  await action308(page, '更正记录');
  await click308(page, page.getByRole('button', { name: '上移第2个职级', exact: true }));
  await click308(page, page.getByRole('button', { name: '保存', exact: true }));
  await expect(page.locator('.object-form')).toHaveCount(0);
  await click308(page, page.getByRole('tab', { name: '变更日志', exact: true }));
  const row = page.locator('.change-operation').first().getByRole('row').filter({
    has: page.getByRole('cell', { name: '职级成员', exact: true }),
  });
  const label = (code: string) => code.toUpperCase() + ' · ' + name;
  await expect(row.locator('[data-label="原值"]')).toHaveText(`1. ${label(a.code)}；2. ${label(b.code)}`);
  await expect(row.locator('[data-label="新值"]')).toHaveText(`1. ${label(b.code)}；2. ${label(a.code)}`);
  await withPolicy(page, async (where, mutate) => {
    mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'code') WHERE ${where} AND action='grade.read'`);
    await page.reload();
    await expect(row.locator('[data-label="原值"]')).toHaveText(`1. ${name}；2. ${name}`);
    await expect(row.locator('[data-label="新值"]')).toHaveText(`1. ${name}；2. ${name}`);
  });
});
