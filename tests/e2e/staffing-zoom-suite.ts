import { grants308 } from './staffing308-grants';
import { chromium } from '@playwright/test';
import { execFileSync, spawn } from 'node:child_process';
import { existsSync, writeFileSync } from 'node:fs';
import { test, expect, type Page } from './fixtures';
import { open } from './bu-support';
import { localConnection } from '../../scripts/local-runtime.mjs';
import { seed308, matrixSeed308, type Kind308 } from './staffing308-support';
import { matrixJourney308 } from './staffing308-journey';
export function staffingZoomSuite(kinds: readonly Kind308[], batch: string, journey = matrixJourney308, seed = matrixSeed308) {
  staffingZoomCases(kinds, batch, journey, seed);
}
export function staffingZoomCases<Kind extends string, Sample>(kinds: readonly Kind[], batch: string,
  journey: (page: Page, sample: Sample, family: Awaited<ReturnType<typeof seed308>>, role: string) => Promise<void>,
  seed: (page: Page, kind: Kind, family: string) => Promise<Sample>) {
for (const kind of kinds)
  for (const role of ['admin', 'reader'])
    test(`${batch} ${kind} ${role} V06/V07 原生200%与400%`, async () => {
      test.setTimeout(300000);
      let display = 2700 + (process.pid % 1000);
      while (existsSync(`/tmp/.X11-unix/X${display}`) || existsSync(`/tmp/.X${display}-lock`)) display++;
      const env = { ...process.env, DISPLAY: `:${display}` },
        xvfb = spawn('Xvfb', [env.DISPLAY, '-screen', '0', '1920x1200x24', '-nolisten', 'tcp', '-nolisten', 'unix'], {
          stdio: 'ignore',
          detached: true,
        });
      let browser: Awaited<ReturnType<typeof chromium.launch>> | undefined;
      const zoom = (steps: number) =>
        execFileSync('python3', ['docs/verification/data/100-ui-baseline/native-zoom.py', String(steps)], {
          env,
          stdio: 'pipe',
        });
      try {
        await expect
          .poll(() => {
            try {
              execFileSync('xdpyinfo', [], { env, stdio: 'ignore', timeout: 1000 });
              return true;
            } catch {
              return false;
            }
          })
          .toBe(true);
        browser = await chromium.launch({
          headless: false,
          env,
          args: ['--window-size=1280,1039', '--window-position=0,0', '--ozone-platform=x11'],
        });
        const context = await browser.newContext({
          viewport: null,
          deviceScaleFactor: undefined,
          baseURL: localConnection().webOrigin,
        });
        const page = await context.newPage();
        page.setDefaultTimeout(15000);
        await open(page);
        const family = await seed308(page, 'jobfamily');
        const samples = [await seed(page, kind, family.id), await seed(page, kind, family.id)];
        if (role === 'reader') {
          await context.clearCookies();
          await open(page, 'reader');
        }
        await test
          .info()
          .attach('grants', { body: JSON.stringify(await grants308(page, role)), contentType: 'application/json' });
        const cdp = await context.newCDPSession(page);
        const window = await cdp.send('Browser.getWindowForTarget');
        const size = await page.evaluate(() => ({ width: innerWidth, height: innerHeight }));
        await cdp.send('Browser.setWindowBounds', {
          windowId: window.windowId,
          bounds: {
            width: window.bounds.width! + 1280 - size.width,
            height: window.bounds.height! + 900 - size.height,
          },
        });
        await expect
          .poll(() => page.evaluate(() => ({ width: innerWidth, height: innerHeight, dpr: devicePixelRatio })))
          .toEqual({ width: 1280, height: 900, dpr: 1 });
        const evidence = [];
        for (const [index, steps, factor] of [
          [0, 5, 2],
          [1, 8, 4],
        ]) {
          zoom(steps);
          await expect.poll(() => page.evaluate(() => devicePixelRatio)).toBe(factor);
          const actual = await page.evaluate(() => ({
            width: innerWidth,
            height: innerHeight,
            dpr: devicePixelRatio,
            outerWidth,
            outerHeight,
          }));
          expect(actual.width).toBe(1280 / factor);
          expect(actual.height).toBe(900 / factor);
          try {
            await journey(page, samples[index], family, role);
          } catch (error) {
            await page.screenshot({
              path: test.info().outputPath('native-failure.png'),
              mask: [page.locator('.app-account-name')],
            });
            throw error;
          }
          const mask = await page.addStyleTag({ content: '.app-account-name{visibility:hidden!important}' });
          const screenshot = await cdp.send('Page.captureScreenshot', {
            format: 'png',
            fromSurface: false,
            captureBeyondViewport: false,
          });
          writeFileSync(test.info().outputPath(`native-${factor}.png`), Buffer.from(screenshot.data, 'base64'));
          await mask.evaluate((el) => el.remove());
          evidence.push({ kind, role, factor, ...actual, passed: true });
        }
        zoom(0);
        await expect.poll(() => page.evaluate(() => devicePixelRatio)).toBe(1);
        await test.info().attach('native-zoom', { body: JSON.stringify(evidence), contentType: 'application/json' });
        await cdp.detach();
      } finally {
        if (browser) {
          try {
            zoom(0);
          } catch {
            /* process may already have stopped */
          }
          await browser.close();
        }
        if (xvfb.pid && xvfb.exitCode === null) process.kill(-xvfb.pid, 'SIGTERM');
        await new Promise<void>((resolve) => {
          if (xvfb.exitCode !== null) resolve();
          else xvfb.once('exit', () => resolve());
        });
      }
    });

}
