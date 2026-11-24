// Verifies the app's async navigation adapter, not the vendor grid or business API.
import { chromium, expect } from '@playwright/test';
import { createRequire } from 'node:module';
import { createServer } from 'node:http';
import { mkdir, readFile, stat, writeFile } from 'node:fs/promises';
import path from 'node:path';
import assert from 'node:assert/strict';
const out = path.resolve(process.argv[2] ?? '.local/111-t02/return');
await mkdir(out, { recursive: true });
const require = createRequire(path.resolve('apps/web/package.json'));
const { build } = await import(require.resolve('vite'));
const dist = path.join(out, 'dist');
await build({ root: path.resolve('apps/web'), configFile: path.resolve('apps/web/vite.config.ts'), build: {
  outDir: dist, emptyOutDir: true, rollupOptions: { input: path.resolve('apps/web/test-fixtures/list-return.html') },
} });
const server = createServer(async (req, res) => {
  try {
    let file = path.resolve(dist, '.' + new URL(req.url, 'http://localhost').pathname);
    if (!file.startsWith(dist + '/')) { res.writeHead(403).end(); return; }
    try { if (!(await stat(file)).isFile()) throw new Error(); }
    catch { file = path.join(dist, 'test-fixtures/list-return.html'); }
    res.writeHead(200, { 'content-type': { '.html': 'text/html', '.js': 'text/javascript', '.css': 'text/css', '.otf': 'font/otf' }[path.extname(file)] ?? 'application/octet-stream' });
    res.end(await readFile(file));
  } catch { res.writeHead(500).end(); }
});
await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
const origin = `http://127.0.0.1:${server.address().port}`, list = `${origin}/test-fixtures/list-return.html?selected=target`;
const result = { scope: 'app return hook with a synthetic delayed row renderer; not actual DataGrid/DB acceptance', checks: [], errors: [], warnings: [], stopped: false };
let browser;
try {
  browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
  page.on('pageerror', error => result.errors.push(error.message));
  page.on('console', message => { if (['warning', 'error'].includes(message.type())) result.warnings.push(message.text()); });
  await page.goto(list);
  const target = page.getByRole('link', { name: '目标对象' });
  await expect(target).toBeFocused();
  await expect(target).toBeInViewport();
  const indexes = (await page.locator('[data-fixture-indexes]').getAttribute('data-fixture-indexes')).split(',').map(Number);
  assert(indexes.length > 0 && indexes.every(index => index === 47));
  result.checks.push('third-page index 40+7; wait for virtual target then focus real link');
  await target.click();
  await page.getByRole('link', { name: '返回列表' }).click();
  await expect(target).toBeFocused();
  result.checks.push('explicit return restores target');
  await target.click(); await page.goBack();
  await expect(target).toBeFocused();
  result.checks.push('browser back restores target');

  await page.goto(list + '&slowRow=1');
  await expect(target).toBeFocused();
  result.checks.push('slow row render beyond 60 frames still restores target');

  await page.goto(list);
  const input = page.getByRole('textbox', { name: '等待期间输入' });
  await input.fill('用户输入优先');
  await expect(page.locator('[data-fixture-ready=true]')).toBeVisible();
  await expect(input).toBeFocused();
  await expect(page.locator('[data-fixture-indexes]')).toHaveAttribute('data-fixture-indexes', '');
  result.checks.push('input before authorized response cancels delayed restoration');
  await page.goto(list);
  await page.mouse.wheel(0, 10);
  await expect(page.locator('[data-fixture-ready=true]')).toBeVisible();
  await expect(page.locator('[data-fixture-indexes]')).toHaveAttribute('data-fixture-indexes', '');
  result.checks.push('wheel input cancels delayed restoration');

  await page.goto(`${origin}/test-fixtures/list-return.html?selected=missing`);
  await expect(page.getByText('目标不在当前授权页', { exact: true })).toBeFocused();
  result.checks.push('unavailable target focuses current fallback');
  assert.deepEqual(result.errors, []); assert.deepEqual(result.warnings, []);
  result.status = 'passed';
} catch (error) { result.status = 'failed'; result.failure = error.stack; throw error; }
finally {
  await browser?.close(); await new Promise(resolve => server.close(resolve)); result.stopped = true;
  await writeFile(path.join(out, 'results.json'), JSON.stringify(result, null, 2) + '\n');
}
console.log(JSON.stringify({ status: result.status, checks: result.checks.length, scope: result.scope, stopped: result.stopped }));
