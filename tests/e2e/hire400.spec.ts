import { chromium } from '@playwright/test';
import { randomUUID } from 'node:crypto';
import { existsSync } from 'node:fs';
import { execFileSync, spawn } from 'node:child_process';
import { localConnection } from '../../scripts/local-runtime.mjs';
import { test, expect, type Page } from './fixtures';
import { open } from './bu-support';
import { namedMenu308 } from './staffing308-support';
import { create, root } from './legal-entity206-support';

const policyPath = '/settings/staffing/assignment-policies';
const policyMenu = (page: Page) => namedMenu308(page, '任职与占编规则', '职务配置', policyPath);
const button = (page: Page, name: string) => page.getByRole('button', { name, exact: true });
const field = (page: Page, name: string) => page.getByRole('textbox', { name: new RegExp('^' + name) });
const date = (page: Page, name: string) => page.getByRole('textbox', { name: new RegExp('^' + name) });
async function layout(page: Page) {
  await expect
    .poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1))
    .toBe(true);
}
async function purposeMenu(page: Page, code: string) {
  await namedMenu308(page, '法律实体', '组织配置', root);
  await page.getByRole('textbox', { name: '搜索编码或名称' }).fill(code);
  await button(page, '查询').click();
  await page.getByRole('link', { name: code + '主体', exact: true }).click();
  await page.getByRole('tab', { name: '用途与绑定' }).click();
  await expect(page.getByRole('heading', { name: '用途与绑定' })).toBeVisible();
}
async function publish(page: Page, category: string) {
  await button(page, '新增').click();
  await field(page, '人员类别').fill(category);
  await field(page, '用途').fill('hire');
  await date(page, '生效日期').fill('2027-04-01');
  await date(page, '首个不适用日').fill('2027-06-01');
  await date(page, '首个不适用日').press('Tab');
  await page.keyboard.press('Tab');
  await expect(button(page, '保存')).toBeFocused();
  await page.keyboard.press('Enter');
  await expect(page.getByText('已保存。', { exact: true })).toBeVisible();
  await expect(page.getByRole('heading', { name: category + ' · hire' })).toBeVisible();
  await layout(page);
}
for (const [width, height] of [
  [1440, 900],
  [1440, 540],
  [390, 844],
  [320, 900],
  [320, 256],
]) {
  test(`400 用途与任职政策 ${width}×${height}`, async ({ page }) => {
    test.setTimeout(150000);
    await open(page);
    const code = await create(page);
    const category = '员工' + randomUUID().slice(0, 8);
    await page.setViewportSize({ width, height });
    await purposeMenu(page, code);
    await button(page, '新增').click();
    await expect(date(page, '生效日期')).toHaveValue('1900-01-01');
    await date(page, '生效日期').fill('2027-04-01');
    await date(page, '首个不适用日').fill('2027-05-01');
    await field(page, '资格或绑定依据').fill('组织授权决定');
    await layout(page);
    await button(page, '保存').click();
    await expect(page.getByText('组织授权决定', { exact: true })).toBeVisible();
    await button(page, '更正期间').click();
    await expect(field(page, '资格或绑定依据')).toHaveValue('组织授权决定');
    await date(page, '首个不适用日').fill('');
    await button(page, '保存').click();
    await expect(page.getByText('2027-04-01 起', { exact: true })).toBeVisible();
    await policyMenu(page);
    await publish(page, category);
    if (width === 1440 && height === 900) {
      await field(page, '筛选规则').fill(category);
      await button(page, '插入后续期间').click();
      await date(page, '生效日期').fill('2027-05-01');
      await page.getByRole('combobox', { name: '任职模式' }).click();
      await page.getByRole('option', { name: '按岗位免占编' }).click();
      await button(page, '保存').click();
      await expect(page.getByText('2027-04-01 起，2027-05-01 起不再适用')).toBeVisible();
      await expect(page.getByText('按岗位免占编', { exact: true })).toBeVisible();
    }
    await page.context().clearCookies();
    await page.setViewportSize({ width: 1440, height: 900 });
    await open(page, 'reader');
    await page.setViewportSize({ width, height });
    await purposeMenu(page, code);
    await expect(button(page, '新增')).toHaveCount(0);
    await expect(button(page, '更正期间')).toHaveCount(0);
    await page.getByRole('combobox', { name: '资料版本' }).click();
    await page.getByRole('option', { name: /第 1 版/ }).click();
    await expect(page.getByText('2027-04-01 起，2027-05-01 起不再适用')).toBeVisible();
    await button(page, '返回当前资料').click();
    await layout(page);
    await policyMenu(page);
    await field(page, '筛选规则').fill(category);
    await expect(page.getByText('仅按职务', { exact: true })).toBeVisible();
    await expect(button(page, '新增')).toHaveCount(0);
    await layout(page);
  });
}
test('400 原生 200% 规则维护、保存及返回', async () => {
  test.setTimeout(140000);
  let display = 5800 + (process.pid % 1000);
  while (existsSync('/tmp/.X11-unix/X' + display) || existsSync('/tmp/.X' + display + '-lock')) display++;
  const env = { ...process.env, DISPLAY: ':' + display };
  const xvfb = spawn(
    'Xvfb',
    [env.DISPLAY, '-screen', '0', '1920x1200x24', '-nolisten', 'tcp', '-nolisten', 'unix'],
    { env, stdio: ['ignore', 'ignore', 'pipe'] },
  );
  xvfb.stderr?.on('data', (data) => process.stderr.write(data));
  let browser: Awaited<ReturnType<typeof chromium.launch>> | undefined;
  try {
    await expect
      .poll(
        () => {
          try {
            execFileSync('xdpyinfo', [], { env, stdio: 'ignore', timeout: 1000 });
            return true;
          } catch {
            return false;
          }
        },
        { timeout: 15000 },
      )
      .toBe(true);
    browser = await chromium.launch({
      headless: false,
      env,
      args: ['--window-size=1280,1039', '--ozone-platform=x11'],
    });
    const ctx = await browser.newContext({
      viewport: null,
      deviceScaleFactor: undefined,
      baseURL: localConnection().webOrigin,
    });
    const page = await ctx.newPage();
    await open(page);
    const code = await create(page);
    const cdp = await ctx.newCDPSession(page),
      win = await cdp.send('Browser.getWindowForTarget');
    const size = await page.evaluate(() => ({ w: innerWidth, h: innerHeight }));
    await cdp.send('Browser.setWindowBounds', {
      windowId: win.windowId,
      bounds: { width: win.bounds.width! + 1280 - size.w, height: win.bounds.height! + 900 - size.h },
    });
    const dims = () => page.evaluate(() => [innerWidth, innerHeight, devicePixelRatio]);
    await expect.poll(dims).toEqual([1280, 900, 1]);
    const zoom = (n: number) =>
      execFileSync('python3', ['docs/verification/data/100-ui-baseline/native-zoom.py', String(n)], {
        env,
        stdio: 'pipe',
      });
    zoom(5);
    await expect.poll(dims).toEqual([640, 450, 2]);
    await purposeMenu(page, code);
    await button(page, '新增').click();
    await field(page, '资格或绑定依据').fill('原生缩放授权');
    await date(page, '生效日期').fill('2027-04-01');
    await button(page, '保存').click();
    await expect(page.getByText('原生缩放授权')).toBeVisible();
    await button(page, '更正期间').click();
    await date(page, '首个不适用日').fill('2027-03-31');
    await button(page, '保存').click();
    await expect(page.getByRole('alert').filter({ hasText: '有效期间' })).toBeFocused();
    await button(page, '取消').click();
    await button(page, '继续编辑').click();
    await expect(date(page, '首个不适用日')).toHaveValue('2027-03-31');
    await button(page, '取消').click();
    await button(page, '放弃修改').click();
    await expect(button(page, '更正期间')).toBeFocused();
    await page.getByRole('link', { name: /返回/ }).click();
    await expect(page.getByRole('tab', { name: '用途与绑定' })).toBeVisible();
    await policyMenu(page);
    await publish(page, '原生' + randomUUID().slice(0, 8));
    console.log(
      JSON.stringify(
        await page.evaluate(() => ({
          viewport: [innerWidth, innerHeight],
          outer: [outerWidth, outerHeight],
          zoom: devicePixelRatio,
          navigator: navigator.userAgent,
        })),
      ),
    );
    zoom(0);
    await expect.poll(dims).toEqual([1280, 900, 1]);
  } finally {
    await browser?.close();
    xvfb.kill();
  }
});

for (const type of ['政策', '用途'] as const) {
  test(`400 ${type}已保存但响应中断，刷新后由原号查回`, async ({ page }) => {
    test.setTimeout(120000);
    await open(page);
    const policy = type === '政策';
    const label = '恢复' + randomUUID().slice(0, 8);
    const entity = policy ? '' : await create(page);
    const other = policy ? '' : await create(page);
    if (policy) await policyMenu(page);
    else await purposeMenu(page, entity);
    await button(page, '新增').click();
    if (policy) {
      await field(page, '人员类别').fill(label);
      await field(page, '用途').fill('hire');
    } else await field(page, '资格或绑定依据').fill(label);
    await date(page, '生效日期').fill('2027-04-01');
    // Keep another entity's unsent draft open while the first tab loses its response.
    const recoveryPage = policy ? page : await page.context().newPage();
    if (!policy) {
      await recoveryPage.goto('/');
      await purposeMenu(recoveryPage, other);
      await button(recoveryPage, '新增').click();
      await field(recoveryPage, '资格或绑定依据').fill('独立的未保存草稿');
    }
    let saved = '',
      writes = 0;
    const endpoint = policy ? '**/api/v1/assignment-policies' : '**/api/v1/legal-entities/*/purposes';
    await page.route(endpoint, async (route) => {
      if (route.request().method() !== 'POST') return route.continue();
      writes++;
      const res = await route.fetch();
      expect(res.status()).toBe(200);
      saved = (await res.json()).submissionId;
      await route.abort();
    });
    const recovery = policy
      ? '**/api/v1/assignment-policies/submissions/*/resolve'
      : '**/api/v1/legal-entities/purpose-submissions/*/resolve';
    await page.route(recovery, (route) => route.abort());
    await button(page, '保存').click();
    await expect(page.getByText('有一笔保存结果待确认。')).toBeVisible();
    await expect.poll(() => saved).not.toBe('');
    const key = 'chnhcm-pending-' + (policy ? 'assignment-policy' : 'legal-purpose');
    const refs = await page.evaluate((key) => JSON.parse(localStorage.getItem(key) ?? '[]'), key);
    expect(refs).toHaveLength(1);
    expect(Object.keys(refs[0]).sort()).toEqual(['instanceRef', 'subjectRef', 'submissionId']);
    expect(refs[0].submissionId).toBe(saved);
    await page.reload();
    await expect(button(page, '新增')).toBeDisabled();
    await page.unroute(recovery);
    const acknowledged = recoveryPage.waitForResponse(
      (r) => r.url().endsWith('/' + saved + '/resolve') && r.status() === 200,
    );
    await button(recoveryPage, '查看保存结果').click();
    await acknowledged;
    await expect(recoveryPage.getByText('原提交已保存。', { exact: true })).toBeVisible();
    if (policy) {
      await field(page, '筛选规则').fill(label);
      await expect(page.getByRole('heading', { name: label + ' · hire' })).toHaveCount(1);
    } else {
      await expect(field(recoveryPage, '资格或绑定依据')).toHaveValue('独立的未保存草稿');
      await button(recoveryPage, '取消').click();
      await button(recoveryPage, '放弃修改').click();
      await expect(recoveryPage.getByText(label, { exact: true })).toHaveCount(0);
      await purposeMenu(recoveryPage, entity);
      await expect(recoveryPage.getByText(label, { exact: true })).toHaveCount(1);
      await recoveryPage.close();
    }
    expect(writes).toBe(1);
  });
}
