import { directoryReturnJourney, geoField, openGeoFilter } from './geoarea-directory-support';
import { chromium } from '@playwright/test';
import { spawn, execFileSync } from 'node:child_process';
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import { test, expect, openMoreActions } from './fixtures';
import { open } from './bu-support';
import { captureViewport } from '../../docs/verification/data/100-ui-capture-fix/capture-viewport.mjs';
test('区划203 原生Chromium200%和400%布局及键盘', async () => {
    test.setTimeout(180000);
    const evidence = `.local/g3-implementation/layout/${Date.now()}-${process.pid}`;
    mkdirSync(evidence, { recursive: true });
    let number = 2700 + process.pid % 1000;
    while (existsSync(`/tmp/.X11-unix/X${number}`) || existsSync(`/tmp/.X${number}-lock`))
        number++;
    const env = { ...process.env, DISPLAY: `:${number}` };
    const xvfb = spawn('Xvfb', [env.DISPLAY, '-screen', '0', '1920x1200x24', '-nolisten', 'tcp', '-nolisten', 'unix'], { env, stdio: 'ignore' });
    let browser: Awaited<ReturnType<typeof chromium.launch>> | undefined;
    try {
        await expect.poll(() => { try {
            execFileSync('xdpyinfo', [], { env, stdio: 'ignore', timeout: 1000 });
            return true;
        }
        catch {
            return false;
        } }).toBe(true);
        browser = await chromium.launch({ headless: false, env, args: ['--window-size=1440,1039', '--window-position=0,0', '--ozone-platform=x11'] });
        const connection = JSON.parse(execFileSync('python3', ['scripts/local-bu.py', 'connection'], { encoding: 'utf8' }));
        const actual = await browser.newContext({ viewport: null, deviceScaleFactor: undefined, baseURL: connection.webOrigin });
        const page = await actual.newPage();
        page.setDefaultTimeout(15000);
        await open(page);
        const zoom = (steps: number) => execFileSync('python3', ['docs/verification/data/100-ui-baseline/native-zoom.py', String(steps)], { env, stdio: 'pipe' });
        const dims = () => page.evaluate(() => ({ width: innerWidth, height: innerHeight, dpr: devicePixelRatio, outer: [outerWidth, outerHeight] }));
        const cdp = await actual.newCDPSession(page);
        const { windowId } = await cdp.send('Browser.getWindowForTarget');
        const first = await dims();
        await cdp.send('Browser.setWindowBounds', { windowId, bounds: { width: first.outer[0] + 1280 - first.width, height: first.outer[1] + 900 - first.height } });
        const matrix = [];
        for (const [steps, dpr] of [[5, 2], [8, 4]]) {
            zoom(steps);
            await expect.poll(async () => (await dims()).dpr).toBe(dpr);
            if (await page.getByRole('button', { name: '打开导航', exact: true }).isVisible()) {
                await page.getByRole('button', { name: '打开导航', exact: true }).click();
                const drawer = page.locator('#navigation-drawer');
                if (!await drawer.getByRole('button', { name: '行政区划', exact: true }).isVisible())
                    await drawer.getByRole('button', { name: '基础资料', exact: true }).click();
                await drawer.getByRole('button', { name: '行政区划', exact: true }).click();
            }
            else {
                await page.getByRole('button', { name: '基础资料', exact: true }).click();
                await page.getByRole('menuitem', { name: '行政区划', exact: true }).click();
            }
            await expect(page).toHaveURL(/\/settings\/basic-data\/geo-areas$/);
            await expect(page.getByRole('heading', { name: '行政区划', exact: true })).toBeVisible();
            await directoryReturnJourney(page);
            await page.getByRole('textbox', { name: '搜索编码或名称' }).fill('110114');
            const dialog = await openGeoFilter(page,'层级');
            await dialog.getByRole('button',{name:'清除条件'}).click();
            await dialog.getByRole('button', { name: '应用' }).click();
            await expect(page).toHaveURL(/q=110114/);
            await expect(page.getByRole('link', { name: '昌平区', exact: true })).toBeVisible();
            await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
            await captureViewport(page, `${evidence}/native-${dpr * 100}-list.png`, [page.locator('.app-account-name')]);
            await page.getByRole('link', { name: '昌平区', exact: true }).scrollIntoViewIfNeeded();
            await expect(page.getByRole('link', { name: '昌平区', exact: true })).toBeInViewport();
            await captureViewport(page, `${evidence}/native-${dpr * 100}-list-row.png`, [page.locator('.app-account-name')]);
            await page.getByRole('link', { name: '昌平区', exact: true }).click();
            await geoField(page, '代码方案').filter({hasText: 'CN-MCA'}).scrollIntoViewIfNeeded();
            await expect(geoField(page, '代码方案').filter({hasText: 'CN-MCA'})).toBeInViewport();
            await expect(page.getByText('对象导航暂时不可用，请返回列表重新查询。', { exact: true })).toHaveCount(0);
            await captureViewport(page, `${evidence}/native-${dpr * 100}-detail.png`, [page.locator('.app-account-name')]);
            await openMoreActions(page);
            await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
            const english = page.getByRole('textbox', { name: '英文名称', exact: true });
            await english.scrollIntoViewIfNeeded();
            await page.getByRole('textbox', { name: '显示名称', exact: true }).focus();
            await page.keyboard.press('Tab');
            await expect(english).toBeFocused();
            await expect(english).toBeInViewport();
            await captureViewport(page, `${evidence}/native-${dpr * 100}-fields.png`, [page.locator('.app-account-name')]);
            await page.getByRole('button', { name: '保存', exact: true }).scrollIntoViewIfNeeded();
            await expect(page.getByRole('button', { name: '保存', exact: true })).toBeInViewport();
            await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
            await captureViewport(page, `${evidence}/native-${dpr * 100}-form.png`, [page.locator('.app-account-name')]);
            matrix.push({ ...await dims(), profile: process.env.CHNHCM_LOCAL_PROFILE, result: 'passed' });
            await page.getByRole('button', { name: '取消编辑', exact: true }).click();
        }
        writeFileSync(`${evidence}/native-zoom.json`, JSON.stringify(matrix, null, 2) + '\n');
        console.log(JSON.stringify({ event: 'G3-native-layout', profile: process.env.CHNHCM_LOCAL_PROFILE, evidence, matrix }));
        zoom(0);
    }
    finally {
        await browser?.close();
        xvfb.kill('SIGTERM');
        await new Promise<void>(resolve => { if (xvfb.exitCode !== null)
            resolve();
        else
            xvfb.once('exit', () => resolve()); });
    }
});
