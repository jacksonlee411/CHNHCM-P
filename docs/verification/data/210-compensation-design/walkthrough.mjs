// Run from the repository with Node and the installed Playwright browsers.
import assert from 'node:assert/strict';
import { chromium, expect } from '@playwright/test';

const prototype = new URL('../../../specs/m09/prototype/compensation-configuration.html', import.meta.url);
const issues = [];
const routes = [];
const metrics = [];
const browser = await chromium.launch();

async function walk(page, checkFrame) {
  page.on('pageerror', error => issues.push(error.message));
  const main = page.getByRole('main');
  const heading = name => expect(main.getByRole('heading', { level: 1 })).toHaveText(name);
  const follow = async anchor => {
    const destination = (await anchor.getAttribute('href')).slice(1);
    await anchor.click();
    await expect(main).toHaveAttribute('data-route', destination);
  };
  const open = async name => {
    await page.getByText('设置', { exact: true }).click();
    await follow(page.getByRole('navigation', { name: '设置菜单' }).getByRole('link', { name, exact: true }));
    await heading(name);
  };
  const press = name => follow(main.getByRole('link', { name, exact: true }));
  const sample = async name => {
    await page.getByText('设计样例', { exact: true }).click();
    await follow(page.getByRole('navigation', { name: '独立设计样例' }).getByRole('link', { name, exact: true }));
  };
  const enterJulyRate = async () => {
    for (const [label,original,next] of [
      ['下限','10,000.00','11,000.00'],['指导值','12,500.00','13,500.00'],['上限','15,000.00','16,000.00']
    ]) {
      const input = main.getByLabel(label,{exact:true});
      await expect(input).toHaveValue(original);
      await input.fill(next);
    }
  };
  await page.goto(prototype.href);
  await open('薪酬计划');
  const search = main.getByRole('searchbox').or(main.getByRole('textbox', { name: '搜索' }));
  await search.fill('PRO-M');
  await main.getByRole('button', { name: '查询' }).click();
  await expect(main.getByText('共 1 条')).toBeVisible();
  await press('专业人员月薪计划');
  await expect(main.getByText('300.00 元/月')).toBeVisible();
  await press('P5 · 月薪薪带');
  await expect(main.getByText('15,000.00', { exact: true })).toBeVisible();
  await press('返回专业人员月薪计划');
  await expect(main.getByRole('link', { name: 'P5 · 月薪薪带', exact: true })).toBeFocused();
  await press('返回列表');
  await expect(search).toHaveValue('PRO-M');
  await expect(main.getByText('共 1 条')).toBeVisible();
  await search.fill('');
  await main.getByRole('button', { name: '查询' }).click();
  const sharedSources = [];
  for (const plan of ['专业人员月薪计划','运营人员月薪计划']) {
    await press(plan);
    if (plan==='运营人员月薪计划') await expect(main.getByText('岗位津贴')).toHaveCount(0);
    await press('P5 · 月薪薪带');
    const sourcePaths = [new URL(page.url()).hash.split('?')[0]];
    await expect(main.getByText('15,000.00',{exact:true})).toBeVisible();
    await press('新增记录');
    await enterJulyRate();
    await press('查看保存结果样例');
    await expect(main.getByText('16,000.00',{exact:true})).toBeVisible();
    sourcePaths.push(new URL(page.url()).hash.split('?')[0]);
    await press('返回'+plan);
    await press('查看保存时引用的 P5 标准');
    await expect(main.getByText('15,000.00',{exact:true})).toBeVisible();
    sourcePaths.push(new URL(page.url()).hash.split('?')[0]);
    sharedSources.push(sourcePaths);
    await press('返回原计划');
    await heading(plan);
    await press('返回列表');
  }
  assert.deepEqual(sharedSources[0],['#monthly','#july','#origin-monthly']);
  assert.deepEqual(sharedSources[0],sharedSources[1]);
  await sample('两计划同名津贴');
  await press('专业人员月薪计划');
  await press('新增记录');
  await heading('新增计划生效记录');
  await expect(main.getByLabel('生效日期')).toHaveValue('2027-10-01');
  const allowance = main.getByLabel('金额', { exact:true });
  await expect(allowance).toHaveValue('300.00');
  await allowance.fill('350.00');
  page.once('dialog', dialog => dialog.accept());
  await press('取消');
  await expect(main.getByText('300.00 元/月')).toBeVisible();
  await expect(main.getByText('2027-10-01')).toHaveCount(0);
  await press('新增记录');
  await allowance.fill('350.00');
  await press('查看保存结果样例');
  await expect(main.getByText('尚未生效')).toBeVisible();
  await expect(main.getByText('350.00 元/月')).toBeVisible();
  await press('更正记录');
  await expect(main.getByLabel('办理原因')).toHaveValue('岗位津贴调整');
  await expect(allowance).toHaveValue('350.00');
  await press('取消');
  await press('P5 · 月薪薪带');
  await expect(main.getByText('16,000.00',{exact:true})).toBeVisible();
  await press('返回专业人员月薪计划');
  await expect(main.getByText('350.00 元/月')).toBeVisible();
  await press('2027-01-01');
  await expect(main.getByText('300.00 元/月')).toBeVisible();
  await expect(main.getByText('2027-01-01 至 2027-09-30')).toBeVisible();
  await press('2027-10-01');
  await press('变更日志');
  await press('查看十月保存的计划');
  await press('查看保存时引用的 P5 标准');
  await expect(main.getByText('16,000.00',{exact:true})).toBeVisible();
  await press('返回计划原件');
  await press('返回变更日志');
  await press('查看保存时的计划');
  await expect(main.getByText('300.00 元/月')).toBeVisible();
  await press('查看保存时引用的 P5 标准');
  await expect(main.getByText('15,000.00',{exact:true})).toBeVisible();
  await press('返回计划原件');
  await press('返回变更日志');
  await expect(main.getByRole('link',{name:'查看保存时的计划',exact:true})).toBeFocused();
  await press('返回基本资料');
  await expect(main.getByText('350.00 元/月')).toBeVisible();
  await press('返回列表');
  await press('运营人员月薪计划');
  await expect(main.getByText('200.00 元/月')).toBeVisible();
  await expect(main.getByText('自 2027-01-01 起')).toBeVisible();
  await expect(main.getByText('350.00 元/月')).toHaveCount(0);
  await sample('一份计划覆盖两个职级');
  await main.getByLabel('适用范围').selectOption('企业通用');
  await expect(main.getByLabel('法律实体')).toBeHidden();
  await main.getByRole('button',{name:'添加职级标准'}).click();
  const gradeRates = main.getByLabel('职级及引用标准');
  await expect(gradeRates.nth(1)).toBeFocused();
  await gradeRates.nth(1).selectOption('P4 · 月薪薪带');
  await expect(gradeRates.first()).toHaveValue('P5 · 月薪薪带');
  await expect(gradeRates.nth(1)).toHaveValue('P4 · 月薪薪带');
  await main.getByRole('button',{name:'移除此职级'}).nth(1).click();
  await expect(gradeRates).toHaveCount(1);
  const basic = main.locator('.composition').first();
  await basic.getByLabel('约定频率').selectOption('年');
  await expect(gradeRates.locator('option:checked')).toHaveText(/P5 · 月薪薪带.*频率不符/);
  await gradeRates.selectOption('P5 · 年薪定额');
  await expect(gradeRates).toHaveValue('P5 · 年薪定额');
  page.once('dialog', dialog => dialog.accept());
  await press('取消');
  await open('薪酬计划');
  await press('新建');
  await expect(main.getByLabel('生效日期')).toHaveValue('1900-01-01');
  await expect(main.getByLabel('法律实体')).toBeHidden();
  await main.getByLabel('适用范围').selectOption('企业通用');
  await main.getByRole('button',{name:'添加固定额组成项'}).click();
  const first = main.locator('.composition').first();
  await first.getByLabel('名称').fill('固定津贴');
  await first.getByLabel('项编码').fill('ALLOW');
  await first.getByLabel('金额').fill('300.00');
  await expect(main.getByLabel('职级及引用标准')).toHaveCount(0);
  // An empty percentage row must not require a second confirmation or invent a base.
  await main.getByRole('button',{name:'添加固定额组成项'}).click();
  const addedTarget = main.locator('.composition').last();
  await addedTarget.getByLabel('取值方式').selectOption('目标比例');
  const base = addedTarget.getByLabel('基准项');
  await expect(base.locator('option')).toHaveText(['','ALLOW · 固定津贴']);
  await expect(base).toHaveValue('');
  await addedTarget.getByLabel('取值方式').selectOption('固定额');
  await expect(addedTarget.getByLabel('金额')).toBeVisible();
  await addedTarget.getByLabel('取值方式').selectOption('目标比例');
  await base.selectOption({label:'ALLOW · 固定津贴'});
  await first.getByLabel('名称').fill('岗位津贴');
  await first.getByLabel('名称').press('Tab');
  await expect(base.locator('option:checked')).toHaveText('ALLOW · 岗位津贴');
  await first.getByRole('button',{name:'下移'}).click();
  const targetAfterMove = main.locator('.composition').first().getByLabel('基准项');
  await expect(targetAfterMove.locator('option:checked')).toHaveText('ALLOW · 岗位津贴');
  await main.locator('.composition').last().getByRole('button',{name:'移除此项'}).click();
  await expect(targetAfterMove.locator('option')).toHaveText(['']);
  await expect(targetAfterMove).toHaveValue('');
  page.once('dialog', dialog => dialog.accept());
  await press('取消');
  await press('新建');
  await press('月薪计划填写样例');
  await expect(main.getByLabel('法律实体')).toHaveValue('甲公司');
  await press('取消');
  await press('新建');
  await press('年薪计划填写样例');
  await expect(main.getByLabel('目标比例（%）')).toHaveValue('12.50');
  await expect(main.getByLabel('基准项').locator('option:checked')).toHaveText('BASIC · 基本工资');
  await press('查看保存结果样例');
  await expect(main.getByText(/22,500.00 元\/年/)).toBeVisible();
  await press('P5 · 年薪定额');
  await expect(main.getByText('180,000.00', { exact: true })).toBeVisible();
  await press('返回管理人员年薪计划');
  await press('更正记录');
  await expect(main.getByLabel('办理原因')).toHaveValue('建立制度');
  await main.getByRole('button', { name: '添加固定额组成项' }).click();
  const added = main.locator('.composition').filter({ has: page.getByRole('heading', { name: '新增组成项' }) });
  await expect(added.getByLabel('名称')).toBeFocused();
  await added.getByRole('button', { name: '上移' }).click();
  await expect(main.locator('.composition h3')).toHaveText(['基本工资','新增组成项','目标绩效奖金']);
  await added.getByLabel('金额').fill('400.00');
  await added.getByLabel('取值方式').focus();
  page.once('dialog', dialog => dialog.dismiss());
  await added.getByLabel('取值方式').selectOption('目标比例');
  await expect(added.getByLabel('金额')).toHaveValue('400.00');
  page.once('dialog', dialog => dialog.accept());
  await added.getByLabel('取值方式').selectOption('目标比例');
  await expect(added.getByLabel('目标比例（%）')).toBeVisible();
  page.once('dialog', dialog => dialog.accept());
  await press('取消');
  await expect(main.getByText('新增组成项')).toHaveCount(0);
  await press('查看保存时引用的 P5 标准');
  await expect(main.getByText('只读历史')).toBeVisible();
  await press('返回原计划');
  await heading('管理人员年薪计划');
  await press('变更日志');
  await press('查看保存时的计划');
  await press('查看保存时引用的 P5 标准');
  await press('返回计划原件');
  await press('返回变更日志');
  await heading('管理人员年薪计划 · 变更日志');

  await open('职级薪酬标准');
  await expect(main.getByRole('link',{name:'P5 月薪标准',exact:true})).toHaveCount(1);
  await sample('重复建立 P5 月薪带');
  await expect(main.getByRole('alert')).toContainText('已有 P5 人民币月薪带');
  await expect(main.getByText('12,000.00–18,000.00 元/月')).toBeVisible();
  await press('打开已有 P5 月薪标准');
  await expect(main.getByText('15,000.00',{exact:true})).toBeVisible();
  await press('返回列表');
  await press('新建');
  await expect(main.getByLabel('生效日期')).toHaveValue('1900-01-01');
  await press('年薪定额填写样例');
  await expect(main.getByLabel('金额', { exact: true })).toHaveValue('180,000.00');
  await press('查看保存结果样例');
  await press('更正记录');
  await expect(main.getByLabel('办理原因')).toHaveValue('建立标准');
  await expect(main.getByLabel('金额', { exact: true })).toHaveValue('180,000.00');
  await press('取消');
  await press('返回列表');
  await press('新建');
  await press('月薪薪带填写样例');
  await press('查看保存结果样例');
  await heading('P5 月薪标准');
  await press('新增记录');
  await expect(main.getByLabel('生效日期')).toHaveValue('2027-07-01');
  await press('取消');
  await expect(main.getByText('2027-07-01', { exact: true })).toHaveCount(0);
  await press('新增记录');
  await enterJulyRate();
  await press('查看保存结果样例');
  await expect(main.getByText('16,000.00', { exact: true })).toBeVisible();
  await press('2027-01-01');
  await expect(main.getByText('2027-01-01 至 2027-06-30')).toBeVisible();
  await expect(main.getByText('15,000.00', { exact: true })).toBeVisible();
  await sample('七月上限误录');
  await press('更正记录');
  await expect(main.getByLabel('办理原因')).toHaveValue('薪酬制度调整');
  const upper = main.getByLabel('上限', { exact: true });
  await expect(upper).toHaveValue('16,500.00');
  await upper.fill('16,000.00');
  await press('查看保存结果样例');
  await press('变更日志');
  await expect(main.getByText('上限 16,500.00 → 16,000.00')).toBeVisible();
  await press('更正前原件');
  await expect(main.getByText('16,500.00', { exact: true })).toBeVisible();
  await press('返回变更日志');
  await press('返回基本资料');
  await press('2027-01-01');
  await press('2027-07-01');
  await press('变更日志');
  await expect(main.getByText('上限 16,500.00 → 16,000.00')).toBeVisible();
  await sample('指导值超出薪带');
  await expect(main.getByRole('alert')).toContainText('16,000.00');
  await expect(main.getByLabel('指导值')).toHaveValue('17,000.00');
  await expect(main.getByLabel('指导值')).toHaveAccessibleDescription(/11,000.00/);
  await sample('薪酬只读');
  await expect(main.getByRole('link', { name: /更正记录|新增记录/ })).toHaveCount(0);
  await press('返回列表');
  await expect(main.getByRole('link',{name:'新建'})).toHaveCount(0);
  await press('P5 年薪标准');
  await expect(main.getByText('180,000.00',{exact:true})).toBeVisible();
  await expect(main.getByRole('link',{name:'更正记录'})).toHaveCount(0);
  await open('职级薪酬标准');
  await page.goto(prototype.href+'#annual-edit');
  await heading('资料不可用');
  await page.reload();
  await heading('资料不可用');
  await page.goBack();
  await heading('职级薪酬标准');
  await expect(main.getByRole('link',{name:'新建'})).toHaveCount(0);
  await sample('仅有职级目录权');
  await heading('资料不可用');
  await expect(main.getByText(/15,000|P5|元\/月/)).toHaveCount(0);
  await page.goto(prototype.href+'#origin-monthly');
  await heading('资料不可用');
  await expect(page.locator('#settings')).toBeHidden();
  await expect(main.getByText(/15,000|P5|元\/月/)).toHaveCount(0);
  await sample('新增七月薪带');
  await main.getByRole('link', { name: '查看保存结果样例' }).focus();
  await page.keyboard.press('Tab');
  await expect(main.getByRole('link', { name: '取消', exact: true })).toBeFocused();
  await page.keyboard.press('Enter');
  await heading('P5 月薪标准');
  const dimensions = await page.evaluate(() => ({
    width: innerWidth, height: innerHeight, clientWidth: document.documentElement.clientWidth,
    contentWidth: document.documentElement.scrollWidth
  }));
  assert.equal(dimensions.contentWidth, dimensions.clientWidth);
  assert.ok(dimensions.clientWidth <= dimensions.width);
  checkFrame(dimensions);
  metrics.push(dimensions);
}

try {
  for (const height of [900, 540]) {
    const page = await browser.newPage({ viewport: { width: 1440, height } });
    await walk(page, d => assert.deepEqual([d.width, d.height], [1440, height]));
    // Every fragment exposed by any example should name a real prototype screen.
    if (height === 900) {
      const queue = ['plans'];
      const seen = new Set();
      while (queue.length) {
        const route = queue.shift();
        if (seen.has(route)) continue;
        seen.add(route);
        // Role samples have their own tab. Their grants persist on ordinary links and reload.
        const roleSample = ['reader','unavailable'].includes(route);
        const routePage = roleSample ? await browser.newPage({viewport:page.viewportSize()}) : page;
        await routePage.goto(prototype.href + '#' + route);
        await routePage.waitForFunction(hash => document.querySelector('main').dataset.route === hash, route);
        if (route.split('?')[0] !== 'unavailable') {
          assert.notEqual(await routePage.locator('main h1').textContent(), '资料不可用', route);
        }
        const targets = await routePage.locator('a[href^="#"]').evaluateAll(links => links.map(link => link.hash.slice(1)));
        queue.push(...targets.filter(target => !seen.has(target)));
        if (roleSample) await routePage.close();
      }
      routes.push(...seen);
    }
    await page.close();
  }
  if (process.argv.includes('--native-zoom')) {
    const profile = await chromium.launchPersistentContext('', {
      headless: false, viewport: null, args: ['--window-size=1320,1000', '--lang=en-US']
    });
    try {
      const page = profile.pages()[0];
      await page.goto(prototype.href);
      const session = await profile.newCDPSession(page);
      const { windowId } = await session.send('Browser.getWindowForTarget');
      // The window manager can finish installing the browser frame after navigation.
      for (let attempt = 0; attempt < 4; attempt++) {
        const { bounds } = await session.send('Browser.getWindowBounds', { windowId });
        const inner = await page.evaluate(() => [innerWidth, innerHeight]);
        if (inner[0] === 1280 && inner[1] === 900) break;
        await session.send('Browser.setWindowBounds', { windowId, bounds: {
          width: bounds.width + 1280 - inner[0], height: bounds.height + 900 - inner[1]
        } });
        try {
          await page.waitForFunction(() => innerWidth === 1280 && innerHeight === 900, null, {timeout: 1000});
          break;
        } catch (error) {
          if (error.name !== 'TimeoutError') throw error;
        }
      }
      await expect.poll(() => page.evaluate(() => [innerWidth, innerHeight])).toEqual([1280, 900]);
      const frame = await page.evaluate(() => [outerWidth, outerHeight]);
      const settings = await profile.newPage();
      await settings.goto('chrome://settings/appearance');
      const zoom = settings.locator('select#zoomLevel');
      await zoom.selectOption('2');
      await page.bringToFront();
      await expect.poll(() => page.evaluate(() => [innerWidth, innerHeight])).toEqual([640, 450]);
      await walk(page, d => assert.deepEqual([d.width, d.height], [640, 450]));
      assert.deepEqual(await page.evaluate(() => [outerWidth, outerHeight]), frame);
      await zoom.selectOption('1');
      await expect.poll(() => page.evaluate(() => [innerWidth, innerHeight])).toEqual([1280, 900]);
    } finally {
      await profile.close();
    }
  }
  assert.deepEqual(issues, []);
  console.log(JSON.stringify({ browser: browser.version(), metrics, sampleRoutes: routes.length, issues }, null, 2));
} finally {
  await browser.close();
}
