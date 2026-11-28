import { randomUUID } from 'node:crypto';
import type { Page } from '@playwright/test';
import { test, expect } from './fixtures';
import { namedMenu308 } from './staffing308-support';
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

for (const mode of ['1440×900', '原生200%']) {
  test('317 地址类型配置与个人资料整页保存 ' + mode, async ({ page: standard }) => {
    test.setTimeout(180000);
    const zoom = mode === '原生200%' ? await nativeZoomBrowser() : undefined;
    const page = zoom?.page || standard;
    try {
      if (!zoom) await page.setViewportSize({ width: 1440, height: 900 });
      const { path, name } = await sample(page);
      if (zoom) await zoom.setZoom(2);
      const button = (label: string) => page.getByRole('button', { name: label, exact: true });
      const catalog = '/api/v1/person-address-types',
        title = '宿舍' + randomUUID().slice(0, 6);
      await namedMenu308(page, '地址类型', '人员配置', '/settings/personnel/address-types');
      const beforeCancel = (await request(page, 'GET', catalog)).body.items;
      await button('新增').click();
      await page.getByRole('dialog').getByLabel('名称').fill('取消样本');
      await button('取消').click();
      await button('放弃修改').click();
      expect((await request(page, 'GET', catalog)).body.items).toEqual(beforeCancel);
      await button('新增').click();
      await expect(page.getByRole('dialog').getByLabel('初始生效日')).toHaveValue('1900-01-01');
      await page
        .getByRole('dialog')
        .getByLabel('编码')
        .fill('DORM-' + randomUUID().slice(0, 8));
      await page.getByRole('dialog').getByLabel('名称').fill(title);
      await page.getByRole('dialog').getByLabel('排序').fill('5');
      await page.getByRole('dialog').press('Escape');
      await button('继续编辑').click();
      await button('保存').click();
      await expect(button(title)).toBeVisible();
      const created = (await request(page, 'GET', catalog)).body.items.find(
        (v: { name: string }) => v.name === title,
      );
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
        expectedRevision: created.revision,
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
      await button(title + '新名').click();
      await select(page, '选用状态', '已停选');
      await button('保存').click();
      await expect(page.getByRole('dialog')).toHaveCount(0);
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
      expect(
        await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1),
      ).toBeTruthy();
    } finally {
      await zoom?.close();
    }
  });
}
