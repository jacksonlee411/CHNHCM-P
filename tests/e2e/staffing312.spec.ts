import { expect, test } from './fixtures';
import { open } from './bu-support';
import { seed308 } from './staffing308-support';
import { positionJourney, seedPosition } from './staffing312-support';

for (const role of ['admin', 'reader'])
  for (const [cell, width, height] of [['V01',1440,900],['V02',1440,540],['V03',390,844],['V04',320,900],['V05',320,256]] as const)
    test(`312 position ${role} ${cell} 岗位维护与历史`, async ({page}) => {
      test.setTimeout(180000);
      await open(page);
      const family = await seed308(page, 'jobfamily'), sample = await seedPosition(page, 'position', family.id);
      if (role === 'reader') { await page.context().clearCookies(); await open(page, role); }
      await page.setViewportSize({width,height});
      await positionJourney(page, sample, family, role);
      expect(await page.evaluate(() => [innerWidth,innerHeight,devicePixelRatio])).toEqual([width,height,1]);
      await page.screenshot({path:test.info().outputPath('position.png'),mask:[page.locator('.app-account-name')]});
    });
