// Real single-session interactions. Fixture identities stay in the private
// input; reports contain timings, counts and request sequence numbers only.
import { chromium, expect } from '@playwright/test';
import { readFileSync, writeFileSync, existsSync } from 'node:fs';
import { performance } from 'node:perf_hooks';

const [inputFile, output] = process.argv.slice(2);
if (existsSync(output)) throw Error('preserve previous measurements');
const input = JSON.parse(readFileSync(inputFile, 'utf8'));
if (!['127.0.0.1', 'localhost'].includes(new URL(input.base).hostname)) throw Error('loopback only');
const report = { version: 1, completed: false, warmupsPerRole: 2, formalPerRole: 400, formalPerCase: 200, groups: [] };
const save = () => writeFileSync(output, JSON.stringify(report, null, 2) + '\n');
writeFileSync(output.replace(/\.json$/, '.matrix.json'), JSON.stringify({ roles: ['admin', 'reader'], sessions: 1, warmupsPerRole: 2, formalPerRole: 400, formalPerCase: 200, targetVisibleMS: 1500, cases: ['deepest', 'wide-last'], fixtureDigest: input.graphSHA256, inputAndDebounce: 'separately measured', siblingCompletion: 'separately measured' }, null, 2) + '\n');
const browser = await chromium.launch();
try {
  for (const role of ['admin', 'reader']) {
    const context = await browser.newContext({ baseURL: input.base, viewport: { width: 1440, height: 900 } });
    const page = await context.newPage();
    page.setDefaultTimeout(15000);
    await page.goto('/');
    await page.getByLabel('登录名').fill(role);
    await page.getByLabel('密码').fill('admin1234567');
    await page.getByRole('button', { name: '登录', exact: true }).click();
    await page.getByRole('button', { name: '切换应用', exact: true }).click();
    await page.getByRole('menuitem', { name: '设置', exact: true }).click();
    await page.getByRole('button', { name: '切换应用', exact: true }).click();
    await page.getByRole('menuitem', { name: '部门', exact: true }).click();
    const search = page.getByRole('textbox', { name: '定位树中部门', exact: true });
    await expect(search).toBeVisible();
    const pending = new Set(); let peak = 0, calls = [], failed = [];
    const isTree = request => /\/departments\/(?:tree\/|[^/]+\/tree-location)/.test(new URL(request.url()).pathname);
    page.on('request', request => { if (isTree(request)) { pending.add(request); peak = Math.max(peak, pending.size); } });
    page.on('requestfinished', request => pending.delete(request));
    page.on('requestfailed', request => { if (pending.delete(request)) failed.push('transport'); });
    page.on('response', response => { if (isTree(response.request())) { calls.push({ status: response.status(), capacityRequest: response.headers()['x-capacity-request'] }); } });
    const group = { role, attempts: [], warmups: [] }; report.groups.push(group); save();
    const metrics = await context.newCDPSession(page);
    await metrics.send('Performance.enable');
    let stopMetrics = false;
    group.memory = { samplingIntervalMS: 100, samples: 0, peakJSHeapUsedBytes: 0, errors: 0, scope: 'sampled renderer JS heap; includes warmups; not whole-browser resident memory' };
    const memoryObservation = (async () => {
      while (!stopMetrics) {
        try {
          const value = await metrics.send('Performance.getMetrics');
          const heap = value.metrics.find(metric => metric.name === 'JSHeapUsedSize');
          if (heap) { group.memory.samples++; group.memory.peakJSHeapUsedBytes = Math.max(group.memory.peakJSHeapUsedBytes, heap.value); }
        } catch { group.memory.errors++; break; }
        await new Promise(resolve => setTimeout(resolve, 100));
      }
    })();
    for (let index = -2; index < 400; index++) {
      const target = input.targets[(index + 2) % input.targets.length];
      const sample = { case: target.case, passed: false };
      const start = performance.now(); peak = pending.size; calls = []; failed = [];
      try {
        await search.fill(target.code);
        const option = page.getByRole('listbox', { name: '部门定位结果' }).locator(`[data-tree-id="${target.id}"]`);
        await expect(option).toContainText(target.name);
        sample.searchInputToResultMS = performance.now() - start;
        const confirmed = performance.now();
        await option.click();
        // Visibility includes intersection with both scroll axes. Merely
        // existing in the virtual DOM is insufficient for deep paths.
        await page.waitForFunction(id => {
          const tree = document.querySelector('[role="tree"][aria-label="部门树"]');
          const node = tree?.querySelector(`[data-tree-id="${id}"][aria-selected="true"]`);
          if (!tree || !node) return false;
          const a = tree.getBoundingClientRect(), b = node.getBoundingClientRect();
          return b.height > 0 && b.bottom > a.top && b.top < a.bottom && b.right > a.left && b.left < a.right;
        }, target.id);
        sample.confirmToVisibleMS = performance.now() - confirmed;
        expect(new URL(page.url()).searchParams.get('scopeDepartmentId')).toBe(target.id);
        await expect(page.locator('[role="tree"]').locator(`[data-tree-id="${target.id}"]`)).toContainText(target.name);
        await expect.poll(() => pending.size, { timeout: 15000 }).toBe(0);
        sample.confirmToSiblingCompleteMS = performance.now() - confirmed;
        sample.treeDOM = await page.getByRole('treeitem').count();
        sample.peakTreeRequests = peak; sample.requests = calls; sample.transportFailures = failed.length;
        sample.heap = await page.evaluate(() => performance.memory ? { usedBytes: performance.memory.usedJSHeapSize, totalBytes: performance.memory.totalJSHeapSize } : null);
        expect(sample.treeDOM).toBeLessThanOrEqual(200);
        expect(peak).toBeLessThanOrEqual(2); expect(failed).toHaveLength(0);
        expect(calls.every(call => call.status === 200)).toBe(true);
        sample.passed = true;
      } catch (error) { sample.error = error.name; sample.elapsedMS = performance.now() - start; }
      (index < 0 ? group.warmups : group.attempts).push(sample); save();
    }
    stopMetrics = true; await memoryObservation; await metrics.detach();
    const successes = group.attempts.filter(v => v.passed).map(v => v.confirmToVisibleMS).sort((a, b) => a - b);
    group.failures = group.attempts.length - successes.length;
    group.successP95MS = successes.length ? successes[Math.ceil(successes.length * .95) - 1] : null;
    group.cases = input.targets.map(target => {
      const attempts = group.attempts.filter(sample => sample.case === target.case);
      const successful = attempts.filter(sample => sample.passed).map(sample => sample.confirmToVisibleMS).sort((a, b) => a - b);
      const p95 = successful.length ? successful[Math.ceil(successful.length * .95) - 1] : null;
      return { case: target.case, attempts: attempts.length, failures: attempts.length - successful.length, successP95MS: p95, passed: attempts.length === 200 && successful.length === 200 && p95 <= 1500 };
    });
    group.passed = group.cases.every(sample => sample.passed) && group.memory.samples > 0 && group.memory.errors === 0;
    save(); await context.close();
  }
  report.completed = true; report.passed = report.groups.every(g => g.passed); save();
} finally { await browser.close(); }
