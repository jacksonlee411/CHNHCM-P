// Optional, explicit reference comparison. Product builds/tests do not need CubeBox.
import { chromium, expect } from '@playwright/test';
import { mkdir, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { randomUUID } from 'node:crypto';
import { comparisonContext } from './comparison-context.mjs';

const { referenceURL, targetURL, output, credentials, targetBase } = await comparisonContext();
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true });
const target = await browser.newPage(), source = await browser.newPage();
const errors = [];
target.on('pageerror', error => errors.push(error.message));
const evidence = { sourceCommit: '2deaf9276086f68e07aaa0c2cf7f35025046a9ee', targetBase, pages: [], checks: [], limitations: ['Fixed CubeBox prototype reference, not external business acceptance.', 'Target uses real Go/PostgreSQL for create/query only.'] };
const sizes = [{ name: 'desktop', width: 1440, height: 900 }, { name: 'short', width: 1440, height: 540 }, { name: 'narrow', width: 390, height: 844 }];
async function capture(page, side, screen) {
  for (const size of sizes) {
    await page.setViewportSize(size);
    await page.evaluate(() => document.fonts.ready);
    await page.locator('[data-page-scroll], .list-grid').evaluateAll(nodes => nodes.forEach(node => node.scrollTop = 0));
    await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
    await page.locator('.object-detail-sheet, .list-grid, .MuiDataGrid-root').first().waitFor();
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
    const file = `${side}-${screen}-${size.name}.png`;
    await page.screenshot({ path: join(output, file), animations: 'disabled', mask: side === 'target' ? [page.locator('.app-account-name')] : [], maskColor: '#09a7a3' });
    const metrics = await page.evaluate(() => {
      const rect = selector => { const element = document.querySelector(selector); if (!element) return null; const r = element.getBoundingClientRect(), s = getComputedStyle(element); return { x: r.x, y: r.y, width: r.width, height: r.height, font: s.fontSize, color: s.color, background: s.backgroundColor, overflow: s.overflowY, columns: s.gridTemplateColumns }; };
      return { header: rect('header'), control: rect('.list-control, .staff-control, .record-control'), sheet: rect('.object-detail-sheet'), identity: rect('.object-detail-identity'), field: rect('.object-detail-field'), input: rect('.object-form input'), scroll: rect('[data-page-scroll], .list-grid'), windowScrollY: scrollY };
    });
    expect(metrics.header.height).toBeGreaterThanOrEqual(64);
    expect(metrics.header.height).toBeLessThanOrEqual(65);
    expect(metrics.control.y).toBe(64);
    expect(metrics.windowScrollY).toBe(0);
    if (side === 'target' && screen === 'form') {
      expect(metrics.input.font).toBe('14px');
      await expect(page.getByRole('button', { name: '保存', exact: true })).toBeInViewport();
      await expect(page.getByRole('link', { name: '取消', exact: true })).toBeInViewport();
    }
    evidence.pages.push({ side, screen, viewport: size, screenshot: file, metrics });
  }
  await page.setViewportSize(sizes[0]);
}
try {
  await target.setViewportSize(sizes[0]);
  await target.goto(targetURL);
  await target.getByLabel('登录名').fill(credentials.login);
  await target.getByLabel('密码').fill(credentials.password);
  await target.getByRole('button', { name: '登录', exact: true }).click();
  await target.getByRole('button', { name: '组织配置', exact: true }).click();
  await target.getByRole('menuitem', { name: '业务单元', exact: true }).click();
  await expect(target.getByRole('heading', { name: '业务单元', exact: true })).toBeVisible();
  const contextResponse = await target.request.get(targetURL + '/api/v1/organization/context');
  expect(contextResponse.status()).toBe(200);
  const currentDate = (await contextResponse.json()).currentDate;
  const prefix = 'VIS019_' + randomUUID().slice(0, 6);
  const session = await (await target.request.get(targetURL + '/api/v1/session')).json();
  for (let index = 1; index <= 8; index++) {
    const response = await target.request.post(targetURL + '/api/v1/business-units', { headers: { Origin: targetURL, 'X-CSRF-Token': session.csrf, 'Idempotency-Key': randomUUID() }, data: { code: `${prefix}_${index}`, name: `青禾研发业务单元 ${index}`, effectiveDate: currentDate, operationType: 'CREATE', operationReason: 'NEW_ORGANIZATION' } });
    expect(response.status()).toBe(201);
  }
  await target.getByLabel('搜索编码或名称').fill(prefix);
  await target.getByRole('button', { name: '查询', exact: true }).click();
  await expect(target.getByText('共 8 条 · 第 1 页')).toBeVisible();
  await target.getByRole('link', { name: '新建', exact: true }).click();
  await target.getByRole('textbox', { name: '编码', exact: true }).fill(`${prefix}_0`);
  await target.getByLabel('中文名称').fill('青禾研发业务单元');
  await target.getByLabel('英文名称').fill('Qinghe Research');
  await target.getByRole('combobox', { name: '操作原因', exact: true }).click();
  await target.getByRole('option', { name: '新设组织', exact: true }).click();
  await expect(target.getByRole('listbox')).toHaveCount(0);
  await capture(target, 'target', 'form');
  await target.getByRole('button', { name: '保存', exact: true }).click();
  await expect(target.getByText('已保存。生效日期：', { exact: false })).toBeVisible();
  const savedURL = new URL(target.url()); savedURL.searchParams.delete('saved');
  await target.goto(savedURL.toString());
  await expect(target.getByRole('heading', { name: /青禾研发业务单元/ })).toBeVisible();
  await capture(target, 'target', 'detail');
  await target.getByRole('button', { name: '来源说明', exact: true }).click();
  await expect(target.getByText('来源版本：', { exact: false })).toBeVisible();
  await target.getByRole('tab', { name: '变更日志', exact: true }).click();
  await target.getByRole('button', { name: '查看当次来源', exact: true }).click();
  await expect(target.getByText('正在查看只读历史资料')).toBeVisible();
  await capture(target, 'target', 'history');
  await target.getByRole('button', { name: '返回当前资料', exact: true }).click();
  await target.getByRole('link', { name: '返回列表', exact: true }).click();
  await expect(target.getByLabel('搜索编码或名称')).toHaveValue(prefix);
  await expect(target.getByText('共 9 条 · 第 1 页')).toBeVisible();
  await capture(target, 'target', 'list');
  evidence.checks.push('Real create, reload, source explanation, historical read and return context passed.');

  await source.setViewportSize(sizes[0]);
  await source.goto(referenceURL); await source.evaluate(() => document.fonts.ready);
  await source.getByRole('button', { name: '原型工具', exact: true }).click();
  await source.getByRole('tab', { name: '工具设置', exact: true }).click();
  await source.getByRole('combobox', { name: '演示角色', exact: true }).click();
  await source.getByRole('option', { name: '人事专员', exact: true }).click();
  await source.getByRole('dialog', { name: '原型工具', exact: true }).getByRole('button', { name: '关闭', exact: true }).click();
  await source.getByRole('button', { name: '切换应用', exact: true }).click();
  await source.getByRole('menuitem', { name: '设置', exact: true }).click();
  await source.getByRole('button', { name: '组织配置', exact: true }).click();
  await source.getByRole('menuitem', { name: '业务单位', exact: true }).click();
  await expect(source.getByRole('heading', { name: '业务单位', exact: true })).toBeVisible();
  await capture(source, 'source', 'list');
  await source.getByRole('button', { name: '青禾研发业务单位', exact: true }).click();
  await expect(source.getByRole('heading', { name: '青禾研发业务单位', exact: true })).toBeVisible();
  await capture(source, 'source', 'detail');
  await source.getByRole('button', { name: '返回列表', exact: true }).click();
  await source.getByRole('button', { name: '新建', exact: true }).click();
  await capture(source, 'source', 'form');
  await source.getByRole('button', { name: '取消', exact: true }).click();
  // The shared compact form standard is also exercised by CubeBox's geography sample.
  await source.getByRole('button', { name: '地理区域', exact: true }).click();
  await source.getByRole('menuitem', { name: '国家/地区组', exact: true }).click();
  await expect(source.getByRole('heading', { name: '国家/地区组', exact: true })).toBeVisible();
  await source.getByRole('button', { name: '新建配置记录', exact: true }).click();
  await capture(source, 'source', 'compact-form');
  expect(errors).toEqual([]);
  evidence.checks.push('Three viewport captures have a 64px header, adjoining controls and no horizontal document overflow.');
  await writeFile(join(output, 'metrics.json'), JSON.stringify(evidence, null, 2) + '\n');
  console.log(`Comparison captured: ${evidence.pages.length} pages in ${output}. Visual review is still required.`);
} finally { await browser.close(); }
