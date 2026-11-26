import { scenarioViewports } from '../e2e/desktop-viewports.mjs';
import { chromium, expect } from '@playwright/test';
import { execFileSync, spawn } from 'node:child_process';
import { readFileSync, writeFileSync, mkdirSync, mkdtempSync, chmodSync, rmSync, existsSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import assert from 'node:assert/strict';
import { captureViewport } from '../../docs/verification/data/100-ui-capture-fix/capture-viewport.mjs';

const root = process.cwd(), out = path.resolve(process.argv[2] ?? '.local/lifecycle-pair/viewport');
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
const code = 'PAIR109_' + Date.now().toString(36).toUpperCase();
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
  await expect(page.getByRole('heading',{name:'业务单元',exact:true})).toBeVisible();
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
  if(create){for(const [date,name,en] of [['2026-06-01','乙','Beta'],['2026-10-01','丙','Gamma']]){
    await page.getByRole('button',{name:'新增记录',exact:true}).click();await page.getByLabel('生效日期').fill(date);await page.getByRole('textbox',{name:'中文名称',exact:true}).fill(name);await page.getByRole('textbox',{name:'英文名称',exact:true}).fill(en);await page.getByRole('combobox',{name:'操作原因',exact:true}).click();await page.getByRole('option',{name:'名称调整',exact:true}).click();await page.getByRole('button',{name:'保存',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);await expect(page.getByRole('button',{name:'更多操作',exact:true})).toBeEnabled();
  }}
}
async function selectDate(date){
  await expect(page.getByRole('button',{name:'更多操作',exact:true})).toBeEnabled();
  const[y,m,d]=date.split('-'), name=`${y}-${m}-${d}`;
  if(await page.locator('.effective-date-compact').isVisible()){
    await page.locator('.effective-date-compact').click();
    await page.getByRole('menuitem',{name,exact:true}).click();
  }else if(await page.getByRole('button',{name,exact:true}).isVisible())await page.getByRole('button',{name,exact:true}).click();
  else {const first=await page.locator('.effective-date-strip [data-day]').first().getAttribute('data-day');await page.locator(`.effective-date-slot [data-date-menu="${first&&date<first?'before':'after'}"]`).click();await page.getByRole('menuitem',{name,exact:true}).click()}
  await expect(page.locator('.object-detail-field').filter({has:page.getByText('生效日期',{exact:true})})).toContainText(name);
  await expect(page.getByRole('button',{name:'更多操作',exact:true})).toBeEnabled();
}
async function detail(){return page.evaluate(async()=>{const q=new URLSearchParams();const selected=new URLSearchParams(location.search).get('selectedRecordId');if(selected)q.set('selectedRecordId',selected);return(await fetch('/api/v1/business-units/'+location.pathname.split('/').at(-1)+'?'+q)).json()})}
async function closeContext() {
  if (context) await context.close();
  if (browser?.isConnected()) await browser.close();
  if (profile) rmSync(profile, { recursive: true, force: true });
  context = undefined; browser = undefined; profile = undefined;
}
try {
  const displayName = ':197';
  if (existsSync('/tmp/.X197-lock')) throw new Error('Task X display is occupied');
  display = spawn('Xvfb', [displayName, '-screen', '0', '2560x1600x24', '-nolisten', 'tcp'], { stdio: 'ignore' });
  for (let i = 0; i < 50 && !existsSync('/tmp/.X11-unix/X197'); i++) await pause(100);
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
      const more=page.getByRole('button',{name:'更多操作',exact:true});
      result.stage=label+' fixed insert';save();await selectDate('2026-01-01');
      const openPair=async name=>{await expect(more).toBeEnabled();await more.focus();await expect(more).toBeFocused();await page.keyboard.press('Enter');await page.getByRole('menuitem',{name,exact:true}).press('Enter');return page.getByRole('dialog',{name,exact:true})};
      let dialog=await openPair('插入停用与重新启用');
      await expect(dialog.getByLabel('停用日期',{exact:true})).toHaveValue('2026-01-02');await expect(dialog.getByLabel('重新启用日期',{exact:true})).toHaveValue('2026-01-03');await shot('insert');
      await keyboardReach(dialog.getByRole('button',{name:'取消',exact:true}));await page.keyboard.press('Enter');await expect(more).toBeFocused();
      dialog=await openPair('插入停用与重新启用');let response=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/lifecycle-pairs'));
      await keyboardReach(dialog.getByRole('button',{name:'保存',exact:true}));await page.keyboard.press('Enter');assert.equal((await response).status(),400);await expect(dialog.getByRole('combobox',{name:'停用原因',exact:true})).toBeFocused();await shot('reason-error');
      for(const [field,name] of [['停用原因','业务终止'],['重新启用原因','业务恢复']]){await dialog.getByRole('combobox',{name:field,exact:true}).click();await page.getByRole('option',{name,exact:true}).press('Enter')}
      response=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/lifecycle-pairs'));await keyboardReach(dialog.getByRole('button',{name:'保存',exact:true}));await page.keyboard.press('Enter');assert.equal((await response).status(),200);await expect(dialog).toHaveCount(0);await expect(more).toBeEnabled();
      // Explicit ordinary correction only provisions a readable name-delta layout fixture.
      const status=await page.evaluate(async()=>{const id=location.pathname.split('/').at(-1),session=await(await fetch('/api/v1/session')).json(),d=await(await fetch('/api/v1/business-units/'+id)).json(),r=d.timeline.find(r=>r.effectiveDate==='2026-01-03');return(await fetch('/api/v1/business-units/'+id+'/records/'+r.recordId+'/correct',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':session.csrf,'Idempotency-Key':crypto.randomUUID()},body:JSON.stringify({expectedSourceVersion:d.headSourceVersion,expectedRecordRevision:r.revision,changes:{name:'恢复时采用的独立资料乙',nameEn:'Independent resumed snapshot'}})})).status});assert.equal(status,200);
      await page.reload();await selectDate('2026-01-02');result.stage=label+' fixed delete';save();dialog=await openPair('删除停用与重新启用');await expect(dialog.getByLabel('删除后的资料变化')).toContainText('恢复时采用的独立资料乙');await shot('delete-name-impact');
      const content=dialog.locator('.MuiDialogContent-root');const before=await content.evaluate(el=>({top:el.scrollTop,range:el.scrollHeight-el.clientHeight}));if(before.range>0){await content.hover();await page.mouse.wheel(0,500);await pause(150);assert.notEqual(await content.evaluate(el=>el.scrollTop),before.top)}
      response=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/lifecycle-pairs/delete'));await keyboardReach(dialog.getByRole('button',{name:'删除两条记录',exact:true}));await page.keyboard.press('Enter');assert.equal((await response).status(),200);await expect(dialog).toHaveCount(0);await expect(more).toBeEnabled();
      const data=await detail();assert.equal(data.timeline.length,3);assert.equal(data.timeline.find(r=>r.effectiveDate==='2026-06-01').name,'乙');assert.equal(data.selectedRecord.effectiveDate,'2026-01-01');
      assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
      row.checks.push('normal menu fixed insertion and pair deletion','default cancel and focus return','reason error focus','real wheel in overflowing dialog','readable name impact','keyboard actions reachable','actual two-record writes and successor retained','real native zoom');row.result='passed';save();
    } catch (error) {
      row.result = 'failed'; row.error = String(error).replaceAll(credentials.password, '[masked]').replaceAll(credentials.login, '[account]'); row.stage = result.stage;row.menu=await page.getByRole('menuitem').allTextContents();row.url=page.url();row.moreExpanded=await page.getByRole('button',{name:'更多操作',exact:true}).getAttribute('aria-expanded');
      try { await captureViewport(page, path.join(out, label + '-failure.png'), [page.locator('.app-account-name'), page.getByLabel('登录名'), page.getByLabel('密码')]); } catch {} save(); process.exitCode = 1; break;
    }
  }
} finally { await closeContext(); if (display) display.kill('SIGTERM'); save(); }
console.log(JSON.stringify(result.rows.map(({ label, result, error, dimensions }) => ({ label, result, error, dimensions })), null, 2));
