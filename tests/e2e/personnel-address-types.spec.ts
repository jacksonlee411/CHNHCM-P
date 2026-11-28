import { randomUUID } from 'node:crypto';
import type { Page } from '@playwright/test';
import { test, expect } from './fixtures';
import { namedMenu308, reference308, reach308 } from './staffing308-support';
import { openPersonnel as menu } from './app-navigation';
import { request } from './dated-object-support';
import { nativeZoomBrowser } from './native-zoom-support';

async function search(page: Page, name: string) {
  await page.getByLabel('姓名或人员编号').fill(name);
  await page.getByLabel('姓名或人员编号').press('Enter');
  await expect(page.getByRole('link', { name, exact: true })).toBeVisible();
}
async function select(page: Page, name: string, option: string) {
  await page.getByRole('combobox', { name, exact: true }).click();
  await page.getByRole('option', { name: option, exact: true }).click();
}
async function sample(page: Page) {
  await page.goto('/');
  await page.getByLabel('登录名').fill('admin');
  await page.getByLabel('密码').fill('admin1234567');
  await page.getByRole('button', { name: '登录', exact: true }).click();
  await expect(page.getByRole('button', { name: '退出' })).toBeVisible();
  const name = '配置试点' + randomUUID().slice(0, 8);
  const person = await request(page, 'POST', '/api/v1/persons', { submissionId: randomUUID(), name });
  expect(person.status).toBe(200);
  const path = '/api/v1/persons/' + person.body.id;
  const source = await request(page, 'POST', path + '/corrections', {
    submissionId: randomUUID(),
    expectedSourceVersion: person.body.sourceVersion,
    addresses: [{ fields: { addressType: '居住地址', addressCountry: '合成地区', street: '一号' } }],
  });
  expect(source.status).toBe(200);
  return { name, path };
}

for (const mode of ['1440×900', '1440×540', '原生200%']) {
  test('317 地址类型配置与个人资料整页保存 ' + mode, async ({ page: standard }) => {
    test.setTimeout(180000);
    const zoom = mode === '原生200%' ? await nativeZoomBrowser() : undefined;
    const page = zoom?.page || standard;
    try {
      if (!zoom) await page.setViewportSize({ width: 1440, height: mode === '1440×540' ? 540 : 900 });
      const { path, name } = await sample(page);
      if (zoom) await zoom.setZoom(2);
      await reference308(page, 'admin');
      const button = (label: string) => page.getByRole('button', { name: label, exact: true });
      const catalog = '/api/v1/person-address-types',
        title = '宿舍' + randomUUID().slice(0, 6);
      await namedMenu308(page, '地址类型', '人员配置', '/settings/personnel/address-types');
      const beforeCancel = (await request(page, 'GET', catalog)).body.items;
      await page.getByRole('link', { name: '新增' }).click();
      await expect(page.getByRole('heading', { name: '新增地址类型', exact: true })).toBeVisible();
      const nameField = page.getByRole('textbox', { name: '名称', exact: true });
      await reach308(page, nameField);
      expect(
        await nameField.evaluate((el) => ({
          label: getComputedStyle(document.querySelector('.object-form .object-field-label')!).fontSize,
          value: getComputedStyle(el).fontSize,
          height: el.closest('.MuiInputBase-root')!.getBoundingClientRect().height,
        })),
      ).toEqual({ label: '13px', value: '14px', height: 32 });
      await page.getByRole('textbox', { name: '名称', exact: true }).fill('取消样本');
      await button('取消').click();
      await button('放弃修改').click();
      expect((await request(page, 'GET', catalog)).body.items).toEqual(beforeCancel);
      await page.getByRole('link', { name: '新增' }).click();
      await expect(page.getByRole('heading', { name: '新增地址类型', exact: true })).toBeVisible();
      await expect(page.getByRole('textbox', { name: '首次生效日', exact: true })).toHaveValue(
        '1900-01-01',
      );
      await page
        .getByRole('textbox', { name: '编码', exact: true })
        .fill('DORM-' + randomUUID().slice(0, 8));
      await page.getByRole('textbox', { name: '名称', exact: true }).fill(title);
      await page.getByRole('spinbutton', { name: '排序', exact: true }).fill('5');
      await button('取消').click();
      await button('继续编辑').click();
      await button('保存').click();
      await expect(page.getByRole('heading', { name: title, exact: true })).toBeVisible();
      const created = (await request(page, 'GET', catalog)).body.items.find(
        (v: { name: string }) => v.name === title,
      );
      const selectDay = async (day: string) => {
        const timeline = page.getByRole('navigation', { name: '生效日期' });
        const compact = timeline.locator('[data-date-menu="all"]');
        if (await compact.isVisible()) {
          await compact.click();
          await page.getByRole('menuitem', { name: day, exact: true }).click();
        } else {
          await timeline.getByRole('button', { name: day, exact: true }).press('Enter');
        }
      };
      const detailPage = page.getByRole('region', { name: '地址类型详情' });
      await button('停用').click();
      await page.getByRole('textbox', { name: '停用生效日', exact: true }).fill('2099-06-01');
      await button('保存').click();
      await expect(detailPage).toContainText('2099-06-01');
      await selectDay('2099-06-01');
      await expect(detailPage).toContainText('2099-06-01 · 停用');
      await selectDay('1900-01-01');
      await expect(detailPage).toContainText('1900-01-01 · 启用');
      const scheduled = (await request(page, 'GET', catalog + '/' + created.id)).body.item;
      await menu(page);
      await search(page, name);
      await page.getByRole('link', { name, exact: true }).click();
      await button('更正资料').click();
      await button('编辑 · 居住地址').click();
      await select(page, '地址类型', title);
      await page.getByLabel('详细地址').fill('合成宿舍三号');
      await button('应用').click();
      const rename = await request(page, 'POST', catalog, {
        id: created.id,
        expectedRevision: scheduled.revision,
        submissionId: randomUUID(),
        name: title + '新名',
        sortOrder: 5,
        selectable: true,
      });
      expect(rename.status).toBe(200);
      await button('保存').click();
      const dialog = page.getByRole('dialog');
      await expect(dialog).toBeVisible();
      await expect(dialog.getByLabel('详细地址')).toHaveValue('合成宿舍三号');
      await expect(dialog).toContainText('重新选择');
      await select(page, '地址类型', title + '新名');
      await button('应用').click();
      await button('保存').click();
      await expect(button('更正资料')).toBeVisible();
      const prior = (await request(page, 'GET', path)).body.addresses[0];
      expect(prior.addressTypeAdoption.revision).toBe(rename.body.revision);
      await namedMenu308(page, '地址类型', '人员配置', '/settings/personnel/address-types');
      await page.getByRole('link', { name: title + '新名', exact: true }).click();
      await button('更正').click();
      await expect(page.getByRole('textbox', { name: '停用生效日', exact: true })).toHaveValue(
        '2099-06-01',
      );
      await page.getByRole('textbox', { name: '停用生效日', exact: true }).fill('2001-01-01');
      await button('保存').click();
      await expect(detailPage.locator('.object-identity-summary')).toContainText('停用');
      await menu(page);
      await search(page, name);
      await page.getByRole('link', { name, exact: true }).click();
      await button('更正资料').click();
      await button('编辑 · ' + title + '新名').click();
      await page.getByLabel('详细地址').fill('合成宿舍五号');
      await button('应用').click();
      await button('保存').click();
      await expect(button('更正资料')).toBeVisible();
      const after = (await request(page, 'GET', path)).body.addresses[0];
      expect(after.addressTypeAdoption).toEqual(prior.addressTypeAdoption);
      expect(after.fields.street).toBe('合成宿舍五号');
      await namedMenu308(page, '地址类型', '人员配置', '/settings/personnel/address-types');
      await page.getByRole('link', { name: title + '新名', exact: true }).click();
      await button('更正').click();
      await button('更正为启用').click();
      await button('保存').click();
      await expect(detailPage.locator('.object-identity-summary')).toContainText('启用');
      await expect(detailPage).toContainText('未安排');
      await expect(button('重新启用')).toHaveCount(0);
      await page.getByRole('tab', { name: '变更日志' }).click();
      await button('上一条修订').click();
      await expect(detailPage).toContainText('2001-01-01');
      await button('返回当前修订').click();
      await expect(detailPage).toContainText('未安排');
      expect(
        await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1),
      ).toBeTruthy();
    } finally {
      await zoom?.close();
    }
  });
}
