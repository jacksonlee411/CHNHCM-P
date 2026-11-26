import { scenarioViewports } from '../e2e/desktop-viewports.mjs';
import { chromium, expect } from '@playwright/test';
import { execFileSync, spawn } from 'node:child_process';
import { readFileSync, writeFileSync, mkdirSync, mkdtempSync, chmodSync, rmSync, existsSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import assert from 'node:assert/strict';
import { captureViewport } from '../../docs/verification/data/100-ui-capture-fix/capture-viewport.mjs';

const root = process.cwd(), out = path.resolve(process.argv[2] ?? '.local/unified-correction/matrix');
mkdirSync(out, { recursive: true });
const local = (...args) => execFileSync('python3', ['scripts/local-bu.py', ...args], { cwd: root, encoding: 'utf8', stdio: 'pipe' }).trim();
local('verify');
const connection = JSON.parse(local('connection'));
const credentials = JSON.parse(readFileSync(local('credentials-path'), 'utf8'));
const native = path.join(root, 'docs/verification/data/100-ui-baseline/native-zoom.py');
const xshot = path.join(root, 'docs/verification/data/100-ui-baseline/x11-shot.py');
const result = { base: execFileSync('git', ['rev-parse', 'HEAD'], { encoding: 'utf8' }).trim(), rows: [] };
const save = () => writeFileSync(path.join(out, 'results.json'), JSON.stringify(result, null, 2) + '\n');
const conditions = scenarioViewports();
const code = 'UC104_' + Date.now().toString(36).toUpperCase();
result.code = code;
let created = false;
let browser, context, page, profile, display;
const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
const dimensions = () => page.evaluate(() => ({ width: innerWidth, height: innerHeight, dpr: devicePixelRatio, outer: [outerWidth, outerHeight], visualScale: visualViewport.scale }));
const zoom = n => execFileSync('python3', [native, String(n)], { stdio: 'pipe' });
async function keyboardReach(locator) {
  for (let i = 0; i < 90; i++) {
    if (await locator.evaluate(el => el === document.activeElement)) { await expect(locator).toBeInViewport(); return; }
    await page.keyboard.press('Tab');
  }
  throw new Error('Keyboard target unreachable: ' + locator.toString());
}
async function compactField(locator, expected) {
  const field = await locator.evaluate(element => {
    const style = getComputedStyle(element), box = element.getBoundingClientRect();
    const label = element.id ? document.querySelector(`label[for="${CSS.escape(element.id)}"]`) : null;
    const labelBox = label?.getBoundingClientRect();
    const input = element.closest('.MuiInputBase-root'), underline = input && getComputedStyle(input, '::before');
    return { fontSize: style.fontSize, height: Math.round(box.height), labelBesideInput: !!labelBox && labelBox.right <= box.left && Math.abs(labelBox.top - box.top) < 8, underlineOnly: !!underline && underline.borderBottomWidth === '1px' && underline.borderBottomStyle === 'solid' && !input?.querySelector('fieldset') };
  });
  assert.equal(field.fontSize, '14px'); assert.equal(field.labelBesideInput, true); assert.equal(field.underlineOnly, true);
  if (expected) assert.deepEqual(field, expected);
  return field;
}
async function openList() {
  await page.locator('[data-navigation-mode]').waitFor({ state: 'visible' });
  await page.evaluate(() => document.fonts.ready);
  if (await page.getByRole('button', { name: '打开导航', exact: true }).isVisible()) {
    await page.getByRole('button', { name: '打开导航', exact: true }).click();
    await page.locator('#navigation-drawer').getByRole('button', { name: '业务单元', exact: true }).click();
  } else {
    await page.getByRole('button', { name: '组织配置', exact: true }).click();
    await page.getByRole('menuitem', { name: '业务单元', exact: true }).click();
  }
  await expect(page.getByRole('heading', { name: '业务单元', exact: true })).toBeVisible();
}
async function startContext(steps) {
  if (steps) {
    profile = mkdtempSync(path.join(tmpdir(), 'chnhcm-ui012-form-')); chmodSync(profile, 0o700);
    context = await chromium.launchPersistentContext(profile, { headless: false, viewport: null, locale: 'zh-CN', timezoneId: 'Asia/Shanghai', args: ['--window-size=1280,1039', '--window-position=0,0', '--ozone-platform=x11'] });
    browser = context.browser(); page = context.pages()[0];
  } else {
    browser = await chromium.launch({ headless: true }); context = await browser.newContext({ viewport: { width: 1440, height: 900 }, locale: 'zh-CN', timezoneId: 'Asia/Shanghai' }); page = await context.newPage();
  }
  page.setDefaultTimeout(10000);
  await page.goto(connection.webOrigin);
}
async function loginAndOpen(create) {
  result.stage = 'login fields'; save();
  await page.getByLabel('登录名').fill(credentials.login); await page.getByLabel('密码').fill(credentials.password);
  await pause(3100); result.stage = 'login submit'; save(); await page.getByRole('button', { name: '登录', exact: true }).click(); result.stage = 'normal menu'; save(); await openList();
  result.stage = create ? 'create fixture' : 'search fixture'; save();
  if (create) {
    await page.getByRole('link', { name: '新建', exact: true }).click();
    await page.getByRole('textbox', { name: '编码', exact: true }).fill(code);
    await page.getByRole('textbox', { name: '中文名称', exact: true }).fill(code + ' 矩阵');
    await page.getByLabel('生效日期').fill('2026-01-01');
    await page.getByRole('combobox', { name: '操作原因', exact: true }).click(); await page.getByRole('option', { name: '新设组织', exact: true }).click();
    await page.getByRole('button', { name: '保存', exact: true }).click();
  } else {
    await page.getByLabel('搜索编码或名称').fill(code); await page.getByRole('button', { name: '查询', exact: true }).click();
    await page.getByRole('row').filter({ hasText: code }).getByRole('link').click();
  }
  await expect(page.getByRole('heading', { level: 1 })).toContainText(code);
  await expect(page.getByRole('tab', { name: '基本资料', exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: '更多操作', exact: true })).toBeEnabled();
  if (create) {
    await page.getByRole('button', { name: '新增记录', exact: true }).click();
    await page.getByLabel('生效日期').fill('2026-06-01');
    await page.getByRole('textbox', { name: '中文名称', exact: true }).fill(code + ' 基准');
    await page.getByRole('combobox', { name: '操作原因', exact: true }).click();
    await page.getByRole('option', { name: '名称调整', exact: true }).click();
    await page.getByRole('button', { name: '保存', exact: true }).click();
    await expect(page.locator('.object-form')).toHaveCount(0);
    await expect(page.getByRole('button', { name: '更多操作', exact: true })).toBeEnabled();
  }
}
async function closeContext() {
  if (context) await context.close();
  if (browser?.isConnected()) await browser.close();
  if (profile) rmSync(profile, { recursive: true, force: true });
  context = undefined; browser = undefined; profile = undefined;
}
try {
  const displayName = ':198';
  if (existsSync('/tmp/.X198-lock')) throw new Error('Task X display is occupied');
  display = spawn('Xvfb', [displayName, '-screen', '0', '2560x1600x24', '-nolisten', 'tcp'], { stdio: 'ignore' });
  for (let i = 0; i < 50 && !existsSync('/tmp/.X11-unix/X198'); i++) await pause(100);
  process.env.DISPLAY = displayName;
  for (const [label, width, height, steps] of conditions) {
    const row = { label, result: 'not_run', checks: [], captures: [] }; result.rows.push(row); save();
    try {
      if (!context || steps) {
        await closeContext(); await startContext(steps);
        if (steps) {
          const cdp = await context.newCDPSession(page), { windowId } = await cdp.send('Browser.getWindowForTarget');
          zoom(0); await pause(150);
          for (let i = 0; i < 3; i++) {
            const d = await dimensions(); if (d.width === 1280 && d.height === 900) break;
            await cdp.send('Browser.setWindowBounds', { windowId, bounds: { width: d.outer[0] + 1280 - d.width, height: d.outer[1] + 900 - d.height } }); await pause(150);
          }
          row.calibration = await dimensions(); assert.equal(row.calibration.width, 1280); assert.equal(row.calibration.height, 900);
          zoom(steps); await pause(200); row.dimensions = await dimensions();
          assert.equal(row.dimensions.dpr, steps === 5 ? 2 : 4); assert.deepEqual(row.dimensions.outer, row.calibration.outer); assert.equal(row.dimensions.visualScale, 1);
          execFileSync('python3', [xshot, path.join(out, label + '-browser.png'), String(row.dimensions.outer[0]), String(row.dimensions.outer[1])]);
        }
        await loginAndOpen(!created); created = true;
      }
      if (!steps) { await page.setViewportSize({ width, height }); row.dimensions = await dimensions(); }
      row.browser = browser.version();
      const shot = async name => {
        await pause(250);
        const file = label + '-' + name + '.png';
        const capture = await captureViewport(page, path.join(out, file), [page.locator('.app-account-name'), page.getByLabel('登录名'), page.getByLabel('密码')]);
        row.captures.push({ file, ...capture }); save();
      };
      result.stage = label + ' detail menu'; save();
      result.stage = label + ' canonical new form'; save();
      await openList(); await page.getByRole('link', { name: '新建', exact: true }).click();
      const canonicalName = page.getByRole('textbox', { name: '中文名称', exact: true }); await expect(canonicalName).toBeEnabled();
      row.canonicalFields = {
        name: await compactField(canonicalName),
        date: await compactField(page.getByLabel('生效日期')),
        reason: await compactField(page.getByRole('combobox', { name: '操作原因', exact: true })),
      };
      assert.equal(row.canonicalFields.name.height, 32); assert.equal(row.canonicalFields.reason.height, 32);
      await shot('create'); await page.getByRole('link', { name: '取消', exact: true }).click();
      await page.getByLabel('搜索编码或名称').fill(code); await page.getByRole('button', { name: '查询', exact: true }).click();
      await page.getByRole('row').filter({ hasText: code }).getByRole('link').click();
      await expect(page.getByRole('heading', { level: 1 })).toContainText(code);
      await expect(page.getByRole('tab', { name: '基本资料', exact: true })).toBeVisible();
      const more = page.getByRole('button', { name: '更多操作', exact: true });
      await expect(more).toBeEnabled(); result.stage = label + ' detail menu'; save();
      await more.press('Enter'); await expect(page.getByRole('menuitem', { name: '更正记录', exact: true })).toBeFocused();
      await shot('menu');
      await expect(page.getByRole('menuitem', { name: '查看日期…', exact: true })).toHaveCount(0);
      result.stage = label + ' correct form'; save();
      await page.getByRole('menuitem', { name: '更正记录', exact: true }).press('Enter');
      const form = page.locator('.object-form');
      for (const label of ['操作类型', '操作原因']) await expect(form.locator('.object-field-label').filter({hasText:new RegExp(`^${label}$`)})).toHaveCount(1);
      for (const label of ['原操作类型', '原操作原因', '新操作类型', '新操作原因']) await expect(form.getByText(label,{exact:true})).toHaveCount(0);
      const input = page.getByRole('textbox', { name: '中文名称', exact: true });
      const index=conditions.findIndex(x=>x[0]===label), nextDate=`2026-02-${String(index+1).padStart(2,'0')}`;
      const date=page.getByLabel('生效日期');await keyboardReach(date);await date.fill(nextDate);
      await expect(page.getByRole('button',{name:'业务分类',exact:true})).toHaveCount(0);
      const reason=page.getByRole('combobox',{name:'操作原因',exact:true});await keyboardReach(reason);await page.keyboard.press('Enter');await page.getByRole('option',{name:index%2?'名称调整':'其他数据调整',exact:true}).press('Enter');
      row.dateField=await compactField(date,row.canonicalFields.date);row.reasonField=await compactField(reason,row.canonicalFields.reason);
      await keyboardReach(input);await input.fill('非法😀');
      const refused=page.waitForResponse(r=>r.url().endsWith('/correct')&&r.request().method()==='POST');await keyboardReach(page.getByRole('button',{name:'保存',exact:true}));await page.keyboard.press('Enter');assert.equal((await refused).status(),400);await expect(input).toBeFocused();await shot('error');
      await input.fill(code + ' ' + label);
      await expect(page.getByRole('button', { name: '预览', exact: true })).toHaveCount(0); row.correctField = await compactField(input, row.canonicalFields.name); await keyboardReach(input); await shot('edit');
      await keyboardReach(page.getByRole('button', { name: '保存', exact: true })); await page.keyboard.press('Enter');
      await expect(page.getByRole('status').filter({ hasText: '已保存' })).toBeVisible();
      const fact = await page.evaluate(async () => { const selected = new URLSearchParams(location.search).get('selectedRecordId'); return (await fetch('/api/v1/business-units/' + location.pathname.split('/').at(-1) + '?' + new URLSearchParams(selected ? {selectedRecordId:selected} : {}))).json(); });
      assert.equal(fact.selectedRecord.name, code + ' ' + label); assert.equal(fact.selectedRecord.effectiveDate, nextDate); assert.equal(fact.selectedRecord.operationReason,index%2?'NAME_ADJUSTMENT':'OTHER_DATA_ADJUSTMENT');
      const d = await dimensions(); await page.mouse.move(Math.max(10, d.width / 2), d.height - 20);
      const positions = () => page.evaluate(() => [document.scrollingElement, ...document.querySelectorAll('[data-page-scroll]')].filter(Boolean).map(el => ({ top: el.scrollTop, range: el.scrollHeight - el.clientHeight })));
      await page.mouse.wheel(0, -10000); await pause(150); const beforeWheel = await positions();
      await page.mouse.wheel(0, 800); await pause(200); const afterWheel = await positions();
      if (beforeWheel.some(x => x.range > 1)) assert.ok(afterWheel.some((x, i) => x.top !== beforeWheel[i].top), 'Real wheel did not move an overflowing body');
      row.wheel = { before: beforeWheel, after: afterWheel };
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
      await expect(page.getByLabel('查看日期', {exact:true})).toHaveCount(0);
      row.checks.push('compact fields match canonical creation by control type, with side labels and underline', 'combined name/date/classification with field error and single save', 'normal-menu target', 'keyboard menu/date/classification/save', 'real API fact and selected record', 'server-current maintenance with actual selected record', 'native/viewport surface with masks', 'wheel and no horizontal overflow'); row.result = 'passed'; save();
    } catch (error) {
      row.result = 'failed'; row.error = String(error).replaceAll(credentials.password, '[masked]').replaceAll(credentials.login, '[account]'); row.stage = result.stage;
      try { await captureViewport(page, path.join(out, label + '-failure.png'), [page.locator('.app-account-name'), page.getByLabel('登录名'), page.getByLabel('密码')]); } catch {} save(); process.exitCode = 1; break;
    }
  }
} finally { await closeContext(); if (display) display.kill('SIGTERM'); save(); }
console.log(JSON.stringify(result.rows.map(({ label, result, error, dimensions }) => ({ label, result, error, dimensions })), null, 2));
