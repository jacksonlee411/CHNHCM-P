// Read the installed fixture through the real menu. Writes only query references.
const { chromium, expect } = require('@playwright/test');
const { execFileSync } = require('node:child_process');
const fs = require('node:fs'),
  path = require('node:path');
const [origin, output, mode = 'desktop', ...extra] = process.argv.slice(2);
if (!origin || !output || extra.length || !['desktop', '200'].includes(mode)) {
  console.error('用法：verify.cjs <本机地址> <输出目录> [模式]。模式须为 desktop 或 200。');
  process.exit(1);
}
if (new URL(origin).hostname !== '127.0.0.1') throw Error('请使用本机验收地址');
fs.mkdirSync(output, { recursive: true });
const zoom = mode === '200';
(async () => {
  const browser = await chromium.launch(
    zoom ? { headless: false, args: ['--window-size=1280,1039', '--ozone-platform=x11'] } : {},
  );
  let p;
  try {
    const ctx = await browser.newContext({
      viewport: zoom ? null : { width: 1440, height: 900 },
    });
    p = await ctx.newPage();
    p.setDefaultTimeout(12000);
    const button = (name) => p.getByRole('button', { name, exact: true });
    const cell = (name) => p.getByRole('gridcell', { name, exact: true });
    const input = p.getByRole('textbox', { name: '姓名或人员编号' });
    const tree = (name) => p.getByRole('treeitem', { name, exact: true });
    const capture = await ctx.newCDPSession(p);
    const snap = async (name) => {
      const photo = await capture.send('Page.captureScreenshot', {
        format: 'png',
        fromSurface: true,
        captureBeyondViewport: false,
      });
      fs.writeFileSync(path.join(output, mode + '-' + name + '.png'), Buffer.from(photo.data, 'base64'));
    };
    async function login(login) {
      await p.goto(origin);
      await p.getByLabel('登录名').fill(login);
      await p.getByLabel('密码').fill('admin1234567');
      await button('登录').click();
      await expect(button('退出')).toBeVisible();
    }
    async function menu() {
      if (await p.evaluate(() => innerWidth < 900)) {
        await button('打开导航').click();
        const app = p.locator('#navigation-drawer').getByRole('button', { name: '人员', exact: true });
        if ((await app.first().getAttribute('aria-expanded')) !== 'true') await app.first().click();
        await app.last().click();
      } else {
        await button('切换应用').click();
        await p.getByRole('menuitem', { name: '人员', exact: true }).click();
      }
      await expect(input).toBeVisible();
    }
    await login('admin');
    if (zoom) {
      const cdp = await ctx.newCDPSession(p),
        win = await cdp.send('Browser.getWindowForTarget'),
        size = await p.evaluate(() => [innerWidth, innerHeight]);
      await cdp.send('Browser.setWindowBounds', {
        windowId: win.windowId,
        bounds: { width: win.bounds.width + 1280 - size[0], height: win.bounds.height + 900 - size[1] },
      });
      await expect.poll(() => p.evaluate(() => [innerWidth, innerHeight])).toEqual([1280, 900]);
      execFileSync('python3', ['docs/verification/data/100-ui-baseline/native-zoom.py', '5']);
      await expect
        .poll(() => p.evaluate(() => [innerWidth, innerHeight, devicePixelRatio]))
        .toEqual([640, 450, 2]);
    }
    await menu();
    await expect(p.getByRole('link', { name: /示例陈舒/ }).first()).toBeVisible();
    await expect(p.getByRole('combobox', { name: '主兼岗' })).toHaveCount(0);
    const dims = await p.evaluate(() => ({
      viewport: [innerWidth, innerHeight, devicePixelRatio],
      control: document.querySelector('.list-control').getBoundingClientRect().height,
      sidebar: document.querySelector('.department-tree-panel')?.getBoundingClientRect().width ?? 0,
    }));
    expect(dims.viewport).toEqual(zoom ? [640, 450, 2] : [1440, 900, 1]);
    expect(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
    if (!zoom) {
      expect(dims.control).toBe(64);
      expect(dims.sidebar).toBe(220);
      expect(await p.getByRole('columnheader').first().getAttribute('data-field')).toBe('__check__');
      for (const name of ['职务', '岗位'])
        await expect(p.getByRole('columnheader', { name, exact: true })).toHaveCount(1);
    }
    await snap('list');
    await input.fill('示例陈舒');
    await input.press('Enter');
    await expect(p.getByText('3 份任职，1 人', { exact: true })).toBeVisible();
    if (zoom) await button('打开部门树').press('Enter');
    for (const [i, n] of ['示例运营总部', '华东区域', '客户服务中心', '服务交付部', '一线服务组'].entries())
      await expect(tree(n)).toHaveAttribute('aria-level', String(i + 1));
    await tree('示例运营总部').focus();
    await p.keyboard.press('ArrowLeft');
    await expect(tree('一线服务组')).toHaveCount(0);
    await p.keyboard.press('ArrowRight');
    await expect(tree('一线服务组')).toBeVisible();
    await tree('服务交付部').click();
    await expect.poll(() => new URL(p.url()).searchParams.has('department')).toBe(true);
    await expect(p.getByText('3 份任职，1 人', { exact: true })).toBeVisible();
    if (zoom) await button('打开部门树').click();
    await p.getByRole('checkbox', { name: '含子部门' }).click();
    await expect(p.getByText('1 份任职，1 人', { exact: true })).toBeVisible();
    await p.getByRole('checkbox', { name: '含子部门' }).click();
    await expect(p.getByText('3 份任职，1 人', { exact: true })).toBeVisible();
    if (zoom) await button('关闭部门树').click();
    if (!zoom) {
      await expect(cell('服务值班岗')).toBeVisible();
      await expect(cell('服务质检岗')).toBeVisible();
      await expect(cell('客户服务专员')).toHaveCount(3);
    }
    const hdr = p.locator('[role=columnheader][data-field=primaryJob]');
    const openFilter = async (header) => {
      if (header) {
        await hdr.hover();
        await hdr.locator('.MuiDataGrid-menuIconButton').click();
        await p.getByRole('menuitem', { name: '筛选…' }).click();
      } else {
        await button('更多操作').click();
        await p.getByRole('menuitem', { name: '列筛选…' }).click();
      }
    };
    let reads = 0;
    p.on('request', (r) => {
      if (r.url().endsWith('/personnel/directory/query')) reads++;
    });
    await openFilter(!zoom);
    const dialog = p.getByRole('dialog', { name: '列筛选' });
    const selectSecondary = async () => {
      await dialog.getByRole('combobox', { name: '值' }).click();
      await p.getByRole('option', { name: '兼岗', exact: true }).click();
    };
    await selectSecondary();
    const pending = reads;
    await dialog.getByRole('button', { name: '取消' }).click();
    expect(reads).toBe(pending);
    await expect(zoom ? button('更多操作') : hdr).toBeFocused();
    await openFilter(false);
    await selectSecondary();
    await dialog.getByRole('button', { name: '应用' }).click();
    await expect(p.getByText('2 份任职，1 人', { exact: true })).toBeVisible();
    await expect(button('更多操作')).toBeFocused();
    const all = p.getByRole('checkbox', { name: '全选查询结果', exact: true });
    await all.click();
    await expect(p.getByText('已选择2项', { exact: true })).toBeVisible();
    await p.getByRole('checkbox', { name: '取消选择人员', exact: true }).first().click();
    await expect(p.getByText('已选择1项', { exact: true })).toBeVisible();
    await button('取消选择').click();
    await expect(input).toBeFocused();
    await input.fill('示例林青');
    await expect(p.getByRole('status').filter({ hasText: '输入已修改' })).toContainText('示例陈舒');
    await snap('filter');
    let link = p.getByRole('link', { name: /示例陈舒/ }).last();
    await link.focus();
    await link.press('Enter');
    await expect(p.locator('.object-detail-identity')).toContainText('2 号兼岗');
    await p.getByRole('link', { name: '返回人员', exact: true }).click();
    await expect(link).toBeFocused();
    await expect(input).toHaveValue('示例陈舒');
    const expiredAddress = p.url();
    if (!zoom) {
      await input.fill('甲'.repeat(201));
      await input.press('Enter');
      await expect(p.getByRole('alert')).toBeVisible();
      await expect(p.locator('a[data-object-link]')).toHaveCount(0);
      await expect(p.getByText('没有匹配的人员', { exact: true })).toHaveCount(0);
      await input.fill('示例陈舒');
      await input.press('Enter');
      await expect(link).toBeVisible();
      let attempts = 0;
      await p.route('**/api/v1/persons/queries', async (r) => {
        attempts++;
        if (attempts === 1)
          await r.fulfill({
            status: 503,
            contentType: 'application/json',
            body: JSON.stringify({ message: '临时读取失败', code: 'TEMP' }),
          });
        else await r.continue();
      });
      await input.fill('示例林青');
      await input.press('Enter');
      await expect(p.getByRole('alert')).toContainText('临时读取失败');
      await button('重试').click();
      await expect(p.getByText('没有匹配的人员', { exact: true })).toBeVisible();
      expect(attempts).toBe(2);
      await button('更多操作').click();
      await p.getByRole('menuitem', { name: '列设置…' }).click();
      await expect(p.getByRole('checkbox', { name: '显示/隐藏全部' })).toBeChecked();
      await p.getByRole('checkbox', { name: '显示/隐藏全部' }).click();
      await p.keyboard.press('Escape');
      await expect(p.getByText('没有显示的列', {exact:true})).toBeVisible();
      await expect(p.getByRole('checkbox', {name:'全选查询结果'})).toBeVisible();
      await button('管理列').click();
      await p.getByRole('button', { name: '重置', exact: true }).click();
      await p.keyboard.press('Escape');
      await expect(p.getByRole('columnheader', { name: '姓名', exact: true })).toBeVisible();
    }
    await button('退出').click();
    await expect(p.getByLabel('登录名')).toBeVisible();
    await login('reader');
    await menu();
    await expect(p.getByRole('link', { name: '示例陈舒', exact: true })).toBeVisible();
    await expect(p.getByRole('columnheader', { name: '主兼岗' })).toHaveCount(0);
    await expect(p.getByRole('tree')).toHaveCount(0);
    await expect(button('打开部门树')).toHaveCount(0);
    await p.goto(expiredAddress);
    await expect(p.getByRole('alert')).toContainText('原查询已失效');
    await expect(p.getByText('没有匹配的人员', { exact: true })).toHaveCount(0);
    await button('重新查询').click();
    await button('清除任职条件').click();
    await expect(input).toBeVisible();
    await input.fill('示例陈舒');
    await input.press('Enter');
    await expect(p.getByText('0 份任职，1 人', { exact: true })).toBeVisible();
    await snap('reader');
    expect(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
    if (zoom) {
      execFileSync('python3', ['docs/verification/data/100-ui-baseline/native-zoom.py', '0']);
      await expect.poll(() => p.evaluate(() => [innerWidth, innerHeight, devicePixelRatio])).toEqual([1280, 900, 1]);
    }
    fs.writeFileSync(path.join(output, mode + '.json'), JSON.stringify(dims));
    console.log(mode, JSON.stringify(dims), '业务验证通过');
  } catch (error) {
    if (p)
      console.error(
        '最后视图',
        new URL(p.url()).pathname,
        [...new URL(p.url()).searchParams.keys()],
        (await p.locator('.list-control, [role=columnheader], .MuiDataGrid-overlay').allTextContents()).join(
          ' | ',
        ),
      );
    throw error;
  } finally {
    await browser.close();
  }
})().catch((e) => {
  console.error(e);
  process.exitCode = 1;
});
