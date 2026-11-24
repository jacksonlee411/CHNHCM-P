import { chromium } from '@playwright/test';
import { execFileSync, spawn } from 'node:child_process';
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import { test, expect } from './fixtures';
import { open } from './bu-support';
import { chooseApplication } from './app-navigation';

test('303 一级菜单原生200%和400%缩放、顺序与直接进入', async () => {
  test.setTimeout(120000);
  let number = 1700 + process.pid % 1000;
  while (existsSync(`/tmp/.X11-unix/X${number}`) || existsSync(`/tmp/.X${number}-lock`)) number++;
  const env = { ...process.env, DISPLAY: `:${number}` };
  const xvfb = spawn('Xvfb', [env.DISPLAY, '-screen', '0', '1920x1200x24', '-nolisten', 'tcp', '-nolisten', 'unix'], { stdio: 'ignore' });
  let browser: Awaited<ReturnType<typeof chromium.launch>> | undefined;
  try {
    await expect.poll(() => { try { execFileSync('xdpyinfo', [], { env, stdio: 'ignore', timeout: 1000 }); return true; } catch { return false; } }).toBe(true);
    browser = await chromium.launch({ headless: false, env, args: ['--window-size=1440,1039', '--window-position=0,0', '--ozone-platform=x11'] });
    const connection = JSON.parse(execFileSync('python3', ['scripts/local-bu.py', 'connection'], { encoding: 'utf8' }));
    const context = await browser.newContext({ viewport: null, deviceScaleFactor: undefined, baseURL: connection.webOrigin });
    const page = await context.newPage();
    await open(page);
    const zoom = (steps: number) => execFileSync('python3', ['docs/verification/data/100-ui-baseline/native-zoom.py', String(steps)], { env, stdio: 'pipe' });
    const observations = [];
    for (const [steps, dpr] of [[5, 2], [8, 4]]) {
      zoom(steps);
      await expect.poll(() => page.evaluate(() => devicePixelRatio)).toBe(dpr);
      await page.getByRole('button', { name: '打开导航', exact: true }).click();
      const drawer = page.locator('#navigation-drawer');
      await expect(drawer.getByRole('button', { name: /^(部门|人员|设置)$/ })).toHaveText(['部门', '人员', '设置']);
      await expect(drawer.getByRole('button', { name: '部门', exact: true })).toBeInViewport();
      await drawer.getByRole('button', { name: '部门', exact: true }).click();
      await expect(drawer).not.toBeVisible();
      await expect(page.getByRole('heading', { level: 1, name: '部门', exact: true })).toBeVisible();
      await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
      observations.push({ ...await page.evaluate(() => ({ width: innerWidth, height: innerHeight, dpr: devicePixelRatio })), passed: true });
      await chooseApplication(page, '设置');
      await expect(page.getByRole('heading', { level: 1, name: '业务单元', exact: true })).toBeVisible();
    }
    zoom(0);
    const directory = '.local/primary303-native-zoom';
    mkdirSync(directory, { recursive: true });
    writeFileSync(`${directory}/${process.env.CHNHCM_LOCAL_PROFILE}.json`, JSON.stringify(observations, null, 2) + '\n');
  } finally {
    await browser?.close();
    xvfb.kill('SIGTERM');
    await new Promise<void>(resolve => { if (xvfb.exitCode !== null) resolve(); else xvfb.once('exit', () => resolve()); });
  }
});
