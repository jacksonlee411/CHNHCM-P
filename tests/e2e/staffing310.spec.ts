import { test, expect } from './fixtures';
import { open } from './bu-support';
import { grants308 } from './staffing308-grants';
import { seed308 } from './staffing308-support';
import { matrixSeed310, matrixJourney310 } from './staffing310-support';

for (const kind of ['gradeladder', 'job'] as const)
  for (const role of ['admin', 'reader'])
    for (const [code, width, height] of [['V01', 1440, 900], ['V02', 1440, 540], ['V03', 390, 844], ['V04', 320, 900], ['V05', 320, 256]] as const)
      test(`310 ${kind} ${role} ${code} 集合、日期与标准页面`, async ({ page }) => {
        test.setTimeout(360000); page.setDefaultTimeout(15000);
        await open(page);
        const family = await seed308(page, 'jobfamily'), sample = await matrixSeed310(page, kind, family.id);
        if (role === 'reader') { await page.context().clearCookies(); await open(page, 'reader'); }
        await test.info().attach('grants', { body: JSON.stringify(await grants308(page, role)), contentType: 'application/json' });
        await page.setViewportSize({ width, height });
        await matrixJourney310(page, sample, family, role);
        const actual = await page.evaluate(() => ({ width: innerWidth, height: innerHeight, dpr: devicePixelRatio }));
        expect(actual.width).toBe(width); expect(actual.height).toBe(height);
        await test.info().attach('viewport', { body: JSON.stringify({ code, kind, role, ...actual }), contentType: 'application/json' });
        await page.screenshot({ path: test.info().outputPath('viewport.png'), mask: [page.locator('.app-account-name')] });
      });
