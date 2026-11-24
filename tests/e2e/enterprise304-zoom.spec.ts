import { chromium } from '@playwright/test';
import { execFileSync, spawn } from 'node:child_process';
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import { test, expect, openMoreActions } from './fixtures';
import { open } from './bu-support';

test('304 企业详情和更正表单原生200%和400%缩放', async () => {
  test.setTimeout(120000);
  let number = 1700 + process.pid % 1000;
  while (existsSync(`/tmp/.X11-unix/X${number}`) || existsSync(`/tmp/.X${number}-lock`)) number++;
  const env = { ...process.env, DISPLAY: `:${number}` };
  const xvfb = spawn('Xvfb', [env.DISPLAY, '-screen', '0', '1920x1200x24', '-nolisten', 'tcp', '-nolisten', 'unix'], { stdio: 'ignore', detached: true });
  let browser: Awaited<ReturnType<typeof chromium.launch>> | undefined;
  try {
    await expect.poll(() => { try { execFileSync('xdpyinfo', [], { env, stdio: 'ignore', timeout: 1000 }); return true; } catch { return false; } }).toBe(true);
    browser = await chromium.launch({ headless: false, env, args: ['--window-size=1440,1039', '--window-position=0,0', '--ozone-platform=x11'] });
    const connection = JSON.parse(execFileSync('python3', ['scripts/local-bu.py', 'connection'], { encoding: 'utf8' }));
    const context = await browser.newContext({ viewport: null, deviceScaleFactor: undefined, baseURL: connection.webOrigin });
    const page = await context.newPage();
    await open(page);
    await page.getByRole('button', {name:'组织配置',exact:true}).click();
    await page.getByRole('menuitem', {name:'企业',exact:true}).click();
    await expect(page).toHaveURL(/settings\/organization\/enterprise\/[0-9a-f-]+$/);
    const zoom = (steps: number) => execFileSync('python3', ['docs/verification/data/100-ui-baseline/native-zoom.py', String(steps)], { env, stdio: 'pipe' });
    const directory = '.local/enterprise304-native-zoom';
    mkdirSync(directory, { recursive: true });
    const observations = [];
    for (const [steps, dpr] of [[5, 2], [8, 4]]) {
      zoom(steps);
      await expect.poll(() => page.evaluate(() => devicePixelRatio)).toBe(dpr);
      await expect(page.getByRole('tab', {name:'基本资料',exact:true})).toBeVisible();
      await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
      await openMoreActions(page);await page.getByRole('menuitem',{name:'更正记录',exact:true}).click();
      const name = page.getByRole('textbox',{name:'中文名称',exact:true});
      await expect(name).toBeVisible();
      await expect(page.getByRole('menuitem',{name:'更正记录',exact:true})).toHaveCount(0);
      await name.scrollIntoViewIfNeeded();
      await expect(name).toBeInViewport();
      await name.click();
      await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
      // Capture the native viewport: the ordinary CSS-sized screenshot clips
      // Chromium's image when browser zoom changes the physical pixel ratio.
      const mask = await page.addStyleTag({content:'.app-account-name { visibility: hidden !important; }'});
      const cdp = await context.newCDPSession(page);
      try {
        const screenshot = await cdp.send('Page.captureScreenshot', {format:'png', fromSurface:false, captureBeyondViewport:false});
        writeFileSync(`${directory}/zoom-${dpr}.png`, Buffer.from(screenshot.data, 'base64'));
      } finally { await cdp.detach(); await mask.evaluate(node => node.remove()); }
      await page.getByRole('button',{name:'取消编辑',exact:true}).click();
      await expect(page.locator('.object-form')).toHaveCount(0);
      observations.push({ ...await page.evaluate(() => ({ width: innerWidth, height: innerHeight, dpr: devicePixelRatio })), passed: true });
    }
    zoom(0);
    writeFileSync(`${directory}/${process.env.CHNHCM_LOCAL_PROFILE}.json`, JSON.stringify(observations, null, 2) + '\n');
  } finally {
    await browser?.close();
    if (xvfb.pid && xvfb.exitCode === null) process.kill(-xvfb.pid, 'SIGTERM');
    await new Promise<void>(resolve => { if (xvfb.exitCode !== null) resolve(); else xvfb.once('exit', () => resolve()); });
  }
});
