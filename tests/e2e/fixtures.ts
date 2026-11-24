import { test as base, expect, type Page, type Request } from '@playwright/test';
import { verifyLocalTarget } from '../../scripts/local-runtime.mjs';
import { writeFile } from 'node:fs/promises';

const formAssistance = new WeakMap<Page, { request?: object; value?: any }>();
const test = base.extend({
  page: async ({page, baseURL}, use, info) => {
    verifyLocalTarget(baseURL);
    const state: {request?: object; value?: any} = {};
    const network: { method: string; route: string; ms: number; status?: number; failed?: boolean }[] = [];
    const pending = new Map<Request, number>();
    const diagnostics: { kind: string; route?: string; status?: number; name?: string }[] = [];
    page.on('pageerror', error => diagnostics.push({ kind: 'pageerror', name: error.name }));
    page.on('requestfailed', request => {
      if (['script', 'stylesheet', 'document'].includes(request.resourceType())) diagnostics.push({ kind: 'asset-failed', route: new URL(request.url()).pathname });
    });
    page.on('response', response => {
      if (response.status() >= 400 && ['script', 'stylesheet', 'document'].includes(response.request().resourceType())) diagnostics.push({ kind: 'asset-response', route: new URL(response.url()).pathname, status: response.status() });
    });
    const started = performance.now();
    const timed = process.env.CHNHCM_BROWSER_TIMINGS === '1';
    if (timed) {
      page.on('request', request => { if (new URL(request.url()).pathname.startsWith('/api/')) pending.set(request, performance.now()); });
      const finish = async (request: Request, failed: boolean) => {
        const start = pending.get(request);
        if (start === undefined || network.length >= 2048) return;
        pending.delete(request);
        const response = await request.response().catch(() => null);
        network.push({ method: request.method(), route: new URL(request.url()).pathname.replace(/[0-9a-f]{8}-[0-9a-f-]{27}/gi, ':id'), ms: performance.now() - start, status: response?.status(), failed });
      };
      page.on('requestfinished', request => { void finish(request, false); });
      page.on('requestfailed', request => { void finish(request, true); });
    }
    formAssistance.set(page, state);
    page.on('request', request => {
      if (request.method() === 'POST' && request.url().endsWith('/preview')) {
        state.request = request; state.value = undefined;
      }
    });
    page.on('response', async response => {
      if (response.request() !== state.request) return;
      const request = response.request();
      const value = await response.json().catch(() => undefined);
      if (request === state.request) state.value = value;
    });
    await use(page);
    if (timed) {
      const path = info.outputPath('request-timings.json');
      await writeFile(path, JSON.stringify({ elapsedMS: performance.now() - started, network, pendingRequests: pending.size }), { mode: 0o600 });
      await info.attach('request-timings', { path, contentType: 'application/json' });
    }
    if (info.status !== info.expectedStatus) {
      const diagnosticPath = info.outputPath('browser-diagnostics.json');
      await writeFile(diagnosticPath, JSON.stringify(diagnostics), { mode: 0o600 });
      await info.attach('browser-diagnostics', { path: diagnosticPath, contentType: 'application/json' });
      // Keep failure locations even if the job is cancelled before the final report.
      // Never print assertion values or request/session contents.
      const locations = info.errors.flatMap(error => Array.from(
        (error.stack ?? '').matchAll(/tests\/e2e\/[\w./-]+:\d+:\d+/g), match => match[0],
      ));
      console.error(JSON.stringify({ event: 'browser_failure_location', title: info.title,
        status: info.status, locations: [...new Set(locations)] }));
    }
    if (info.status === info.expectedStatus || page.isClosed()) return;
    try {
      const path = info.outputPath('failure-masked.png');
      await page.screenshot({path, mask:[page.locator('.app-account-name'), page.getByLabel('登录名'), page.getByLabel('密码')]});
      await info.attach('masked failure',{path,contentType:'image/png'});
    } catch { /* Preserve the original failure even if the page has already closed. */ }
  },
});
export {test, expect};
export type {Page} from '@playwright/test';

// Manual recovery scenarios deliberately keep the first automatic lookup
// pending. Subsequent lookups and writes still reach the real API. The 301
// suite separately verifies automatic confirmation without this fault.
export async function deferAutomaticConfirmation(page: Page) {
  await page.addInitScript(() => {
    const original = window.fetch;
    const uncertain = new Set<string>();
    const attempted = new Set<string>();
    window.fetch = async (input, init) => {
      const request = new Request(input, init), url = new URL(request.url);
      const id = request.headers.get('Idempotency-Key');
      // Object types deliberately allow the same submission ID. Match
      // the production channels' object boundary when injecting uncertainty.
      const segment = url.pathname.match(/^\/api\/v1\/(business-units|organization|departments|locations|enterprise|legal-entities)(?:\/|$)/)?.[1];
      const kind = segment === 'organization' ? 'business-units' : segment;
      const identity = id && kind ? `${kind}:${id}` : undefined;
      const first = Boolean(identity && !attempted.has(identity));
      if (identity) attempted.add(identity);
      const resolving = url.pathname.match(/\/submissions\/([^/]+)\/resolve$/)?.[1];
      if (resolving && kind && uncertain.delete(`${kind}:${resolving}`)) {
        return new Response(JSON.stringify({ submissionId: resolving, outcome: 'pending' }), {
          status: 202, headers: { 'Content-Type': 'application/json' },
        });
      }
      try {
        const response = await original(request);
        if (identity && first && response.status >= 500) uncertain.add(identity);
        return response;
      } catch (error) {
        if (identity && first) uncertain.add(identity);
        throw error;
      }
    };
  });
}

// Current day is server metadata, not an editable view-date control.
export async function openMoreActions(page: Page) {
  await test.step('展开更多操作并确认菜单状态', async () => {
    const trigger = page.getByRole('button', { name: '更多操作', exact: true });
    await expect(trigger).toBeEnabled(); await trigger.click();
    // MUI's modal menu hides the page from the accessibility tree while open.
    await expect(page.getByRole('button', { name: '更多操作', exact: true, includeHidden: true })).toHaveAttribute('aria-expanded', 'true');
    await expect(page.getByRole('menu')).toBeVisible();
  });
}
export async function readCurrentDate(page: Page): Promise<string> {
  return page.evaluate(async () => {
    const response = await fetch('/api/v1/organization/context');
    if (!response.ok) throw new Error('context read failed: ' + response.status);
    return (await response.json()).currentDate;
  });
}
export async function expectCurrentDate(page: Page, expected: string) {
  await expect(page.getByLabel('查看日期', {exact: true})).toHaveCount(0);
  expect(await readCurrentDate(page)).toBe(expected);
}
// Test-only record discovery from the public timeline, followed by an actual
// selectedRecordId request. Assertions inspect the server's selectedRecord;
// this does not synthesize viewFact or expose a production as-of endpoint.
export async function readRecord(page: Page, date?: string, source?: string) {
  return page.evaluate(async ({date, source}) => {
    const base = '/api/v1/business-units/' + location.pathname.split('/').at(-1);
    const query = new URLSearchParams(source ? {sourceVersion: source} : {});
    const get = async () => { const response = await fetch(base + '?' + query); if (!response.ok) throw new Error('record read: ' + response.status); return response.json(); };
    const detail = await get();
    if (!date) return detail;
    const record = detail.timeline.find((r: {effectiveDate: string; endExclusive?: string}) => r.effectiveDate <= date && (!r.endExclusive || date < r.endExclusive));
    if (!record) throw new Error('test probe has no applicable record: ' + date);
    query.set('selectedRecordId', record.recordId);
    return get();
  }, {date, source});
}

export async function readFormAssistance(page: Page, status: string) {
  // Allow the specified 300ms debounce to enqueue the latest draft.
  await page.waitForTimeout(350);
  await expect.poll(() => formAssistance.get(page)?.value?.candidateStatus ?? formAssistance.get(page)?.value?.code).toBe(status);
  return formAssistance.get(page)!.value;
}

export async function applyListOption(page: Page, label: '状态' | '排序', value: string) {
  if (label === '排序' && await page.getByRole('columnheader', { name: /名称/ }).isVisible()) {
    const header = page.getByRole('columnheader', { name: value === '编码升序' ? /编码/ : /名称/ });
    const direction = value === '名称降序' ? 'descending' : 'ascending';
    for (let attempt = 0; attempt < 3; attempt++) {
      const previous = await header.getAttribute('aria-sort');
      if (previous === direction) break;
      await header.click();
      await expect.poll(() => header.getAttribute('aria-sort')).not.toBe(previous);
    }
    await expect(header).toHaveAttribute('aria-sort', direction);
    return;
  }
  await openMoreActions(page);
  await page.getByRole('menuitem', { name: label === '状态' ? '状态筛选…' : /^排序：/ }).click();
  const dialog = page.getByRole('dialog');
  await dialog.getByRole('combobox', { name: label === '状态' ? label : '排序方式', exact: true }).click();
  await page.getByRole('option', { name: value, exact: true }).click();
  await dialog.getByRole('button', { name: '应用', exact: true }).click();
  await expect(dialog).toBeHidden();
}
export function recordRange(page: Page, count: number, currentPage: number) {
  const size = Number(new URL(page.url()).searchParams.get('pageSize') ?? 20);
  return [count ? (currentPage - 1) * size + 1 : 0, '–', Math.min(currentPage * size, count), ' / ', count].join('');
}

export async function expectLogChange(page: Page, field: string, before: string, after: string) {
  const row = page.locator('.change-operation').first().getByRole('row').filter({has: page.getByRole('cell', {name: field, exact: true})});
  await expect(row.locator('[data-label="原值"]')).toHaveText(before);
  await expect(row.locator('[data-label="新值"]')).toHaveText(after);
}
