// Verifies the app's async navigation adapter, not the vendor grid or business API.
import { chromium, expect } from '@playwright/test';
import { createRequire } from 'node:module';
import { createServer } from 'node:http';
import { mkdir, readFile, stat, writeFile } from 'node:fs/promises';
import path from 'node:path';
import assert from 'node:assert/strict';
const out = path.resolve(process.argv[2] ?? '.local/111-community-t01-t02/grid-return');
await mkdir(out, { recursive: true });
const require = createRequire(path.resolve('apps/web/package.json'));
const { build } = await import(require.resolve('vite'));
const dist = path.join(out, 'dist');
await build({ root: path.resolve('apps/web'), configFile: path.resolve('apps/web/vite.config.ts'), build: {
  outDir: dist, emptyOutDir: true, rollupOptions: { input: path.resolve('apps/web/test-fixtures/datagrid-community.html') },
} });
const server = createServer(async (req, res) => {
  try {
    let file = path.resolve(dist, '.' + new URL(req.url, 'http://localhost').pathname);
    if (!file.startsWith(dist + '/')) { res.writeHead(403).end(); return; }
    try { if (!(await stat(file)).isFile()) throw new Error(); }
    catch { file = path.join(dist, 'test-fixtures/datagrid-community.html'); }
    res.writeHead(200, { 'content-type': { '.html': 'text/html', '.js': 'text/javascript', '.css': 'text/css', '.otf': 'font/otf' }[path.extname(file)] ?? 'application/octet-stream' });
    res.end(await readFile(file));
  } catch { res.writeHead(500).end(); }
});
await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
const origin = `http://127.0.0.1:${server.address().port}`, list = `${origin}/test-fixtures/datagrid-community.html?selected=row-58`;
const result = { scope: 'actual source-built Community + app return hook with synthetic authorized pages; no Go/API/DB acceptance', checks: [], errors: [], warnings: [], stopped: false };
let browser, page;
try {
  browser = await chromium.launch({ headless: true });
  page = await browser.newPage({ viewport: { width: 1440, height: 740 } });
  page.on('pageerror', error => result.errors.push(error.message));
  page.on('console', message => { if (['warning', 'error'].includes(message.type())) result.warnings.push(message.text()); });
  const target = page.getByRole('link', { name: '目标对象', exact: true });
  const input = page.getByRole('textbox', { name: '等待期间输入' });
  await page.goto(list);
  await expect(target).toBeFocused(); await expect(target).toBeInViewport();
  assert(await page.locator('.MuiDataGrid-virtualScroller').evaluate(node => node.scrollTop) > 500);
  result.checks.push('real virtual row 58 outside initial DOM is scrolled and focused after current page arrives');
  await target.click(); await page.getByRole('link', { name: '返回列表', exact: true }).click();
  await expect(target).toBeFocused(); await expect(target).toBeInViewport();
  result.checks.push('real grid explicit return restores selected virtual row');
  await target.click(); await page.goBack(); await expect(target).toBeFocused();
  result.checks.push('real grid browser back restores current row');
  await page.goto(list + '&page=3&pageSize=20');
  await expect(target).toBeFocused(); await expect(target).toBeInViewport();
  await expect(page.getByText('41–60 / 73', { exact: true })).toBeVisible();
  result.checks.push('third server page uses page offset and current controlled range');
  await page.goto(`${origin}/test-fixtures/datagrid-community.html?page=3&pageSize=20`);
  await expect(page.locator('[data-fixture-ready=true]')).toBeVisible();
  await page.locator('.MuiDataGrid-virtualScroller').hover(); await page.mouse.wheel(0, 900);
  await target.click(); await page.goBack(); await expect(target).toBeFocused(); await expect(target).toBeInViewport();
  result.checks.push('first browser back restores the clicked object when the original URL has no selected ID');
  await page.goto(list + '&slow=1'); await expect(target).toBeFocused();
  result.checks.push('late current-page response still restores actual grid row');
  await page.goto(list + '&slow=1'); await input.fill('用户输入优先');
  await expect(page.locator('[data-fixture-ready=true]')).toBeVisible();
  await expect(input).toBeFocused();
  assert.equal(await page.locator('.MuiDataGrid-virtualScroller').evaluate(node => node.scrollTop), 0);
  result.checks.push('typing before page arrives prevents late real-grid scroll/focus takeover');
  await page.goto(`${origin}/test-fixtures/datagrid-community.html`);
  await expect(page.locator('[data-fixture-ready=true]')).toBeVisible();
  await page.getByRole('button', { name: '列设置', exact: true }).click();
  await page.getByRole('checkbox', { name: '名称', exact: true }).uncheck(); await page.keyboard.press('Escape');
  await expect(page.getByRole('columnheader', { name: /名称/ })).toHaveCount(0);
  await input.fill('跨布局草稿');
  await page.screenshot({ path: path.join(out, 'desktop.png'), fullPage: true });
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(input).toHaveValue('跨布局草稿'); await expect(input).toBeFocused();
  await expect(page.getByRole('columnheader')).toHaveCount(0);
  await expect(page.getByRole('link', { name: '合成对象 1', exact: true })).toBeVisible();
  await expect(page.getByRole('combobox', { name: '每页条数', exact: true })).toBeVisible();
  await page.screenshot({ path: path.join(out, 'summary.png'), fullPage: false });
  await page.setViewportSize({ width: 1440, height: 740 });
  await expect(page.getByRole('columnheader', { name: /名称/ })).toHaveCount(0);
  await expect(input).toHaveValue('跨布局草稿'); await expect(input).toBeFocused();
  result.checks.push('summary ignores desktop hidden-name preference, then restores it without remounting input');
  await page.goto(`${origin}/test-fixtures/datagrid-community.html?selected=missing`);
  await expect(page.getByText('目标不在当前授权页', { exact: true })).toBeFocused();
  result.checks.push('missing target focuses current fallback');
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto(list);
  await expect(target).toBeFocused(); await expect(target).toBeInViewport();
  assert(await page.evaluate(() => document.scrollingElement.scrollTop) > 500);
  result.checks.push('natural document scrolling restores actual mobile summary link');
  await page.setViewportSize({ width: 1440, height: 740 });
  await expect(target).toBeFocused(); await expect(target).toBeInViewport();
  await page.setViewportSize({ width: 700, height: 740 });
  await expect(target).toBeFocused(); await expect(target).toBeInViewport();
  result.checks.push('same-object link focus survives mobile, desktop and bounded tablet transitions');
  assert.deepEqual(result.errors, []); assert.deepEqual(result.warnings, []);
  result.status = 'passed';
} catch (error) {
  result.status = 'failed'; result.failure = error.stack; result.url = page?.url();
  result.body = await page?.locator('body').innerText();
  result.rows = await page?.locator('.MuiDataGrid-row').evaluateAll(nodes => nodes.map(n => n.getAttribute('data-id')));
  await page?.screenshot({ path: path.join(out, 'failure.png'), fullPage: true }); throw error;
} finally {
  await browser?.close(); await new Promise(resolve => server.close(resolve)); result.stopped = true;
  await writeFile(path.join(out, 'results.json'), JSON.stringify(result, null, 2) + '\n');
}
console.log(JSON.stringify({ status: result.status, checks: result.checks.length, scope: result.scope, stopped: result.stopped }));
