import { lazyCredentials } from './credentials.mjs';
import {readCurrentDate,  test, expect, openMoreActions, type Page } from './fixtures';
import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { randomUUID } from 'node:crypto';
import path from 'node:path';
import type { Locator } from '@playwright/test';

const root = path.resolve(import.meta.dirname, '../..');
const credentials = lazyCredentials();
const buPath = '/settings/organization/business-units';
async function fieldAppearance(field: Locator) {
  await field.waitFor({state:'visible'});
  // The new form is visible while its create authority is loading. Measure
  // editable controls only after that real response enables the form.
  await expect(field).toBeEnabled();
  // Measure the shared idle appearance, independently of the last pointer position.
  await field.page().mouse.move(0,0);
  await field.page().evaluate(async()=>{await document.fonts.ready;await new Promise<void>(resolve=>requestAnimationFrame(()=>resolve()))});
  return field.evaluate(element => {
    const style = getComputedStyle(element);
    const root = element.closest('.MuiInputBase-root');
    const underline = root && getComputedStyle(root, '::before');
    const label = element.id ? document.querySelector(`label[for="${CSS.escape(element.id)}"]`) : null;
    const inputBox = element.getBoundingClientRect(), labelBox = label?.getBoundingClientRect();
    return {
      fontSize: style.fontSize, lineHeight: style.lineHeight,
      height: Math.round(inputBox.height),
      labelBesideInput: !!labelBox && labelBox.right <= inputBox.left && Math.abs(labelBox.top - inputBox.top) < 8,
      underlineOnly: !!underline && underline.borderBottomWidth === '1px' && underline.borderBottomStyle === 'solid' && !root?.querySelector('fieldset'),
    };
  });
}
async function login(page: Page, reader = false) {
  await new Promise(resolve => setTimeout(resolve, 3100));
  await page.goto('/');
  await page.getByLabel('登录名').fill(reader ? credentials.reader_login : credentials.login);
  await page.getByLabel('密码').fill(reader ? credentials.reader_password : credentials.password);
  await page.getByRole('button', { name: '登录', exact: true }).click();
  await page.getByRole('button', { name: '切换应用', exact: true }).click();
  await page.getByRole('menuitem', { name: '设置', exact: true }).click();
  await page.getByRole('button', { name: '组织配置', exact: true }).click();
  await page.getByRole('menuitem', { name: '业务单元', exact: true }).click();
  expect(await readCurrentDate(page)).toMatch(/^\d{4}-\d{2}-\d{2}$/);
}
async function create(page: Page) {
  const today = await readCurrentDate(page);
  const code = 'MENU_' + randomUUID().slice(0, 8);
  await page.getByRole('link', { name: '新建', exact: true }).click();
  await page.getByRole('textbox', { name: '编码', exact: true }).fill(code);
  await page.getByRole('textbox', { name: '中文名称', exact: true }).fill('菜单验证单元');
  await page.getByLabel('生效日期').fill('2026-01-01');
  await page.getByRole('combobox', { name: '操作原因', exact: true }).click();
  await page.getByRole('option', { name: '新设组织', exact: true }).click();
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.getByRole('button', { name: '更多操作', exact: true })).toBeEnabled();
  return { today, code };
}


test('新建与记录维护使用一致的字段样式、标签和响应式布局', async ({ page }) => {
  await login(page);
  await page.getByRole('link', { name: '新建', exact: true }).click();
  const nameStyle = await fieldAppearance(page.getByRole('textbox', { name: '中文名称', exact: true }));
  const dateStyle = await fieldAppearance(page.getByLabel('生效日期'));
  const reasonStyle = await fieldAppearance(page.getByRole('combobox', { name: '操作原因', exact: true }));
  const saveHeight = await page.getByRole('button', { name: '保存', exact: true }).evaluate(e => Math.round(e.getBoundingClientRect().height));
  console.log(JSON.stringify({ event: 'form_idle_appearance', nameStyle, dateStyle, reasonStyle, saveHeight }));
  expect(nameStyle).toMatchObject({ fontSize: '14px', height: 32, labelBesideInput: true, underlineOnly: true });
  await page.getByRole('link', { name: '取消', exact: true }).click();
  await create(page);
  for (const action of ['ADD', 'CORRECT', 'REACTIVATE'] as const) {
    if (action === 'REACTIVATE') {
      await openMoreActions(page);
      await page.getByRole('menuitem', { name: '停用业务单元', exact: true }).click();
      await expect(page.getByRole('textbox', { name: '中文名称', exact: true })).toHaveCount(0);
      expect(await fieldAppearance(page.getByLabel('生效日期'))).toEqual(dateStyle);
      expect(await fieldAppearance(page.getByRole('combobox', { name: '操作原因', exact: true }))).toEqual(reasonStyle);
      await expect(page.locator('.record-control').getByRole('button', { name: '确认停用', exact: true })).toBeEnabled();
      await page.getByRole('button', { name: '取消编辑', exact: true }).click();
      await expect(page.getByRole('button', { name: '更多操作', exact: true })).toBeFocused();
      await openMoreActions(page);
      await page.getByRole('menuitem', { name: '停用业务单元', exact: true }).click();
      await page.getByLabel('生效日期').fill('2026-02-01');
      await page.getByRole('combobox', { name: '操作原因', exact: true }).click();
      await page.getByRole('option', { name: '业务终止', exact: true }).click();
      await page.getByRole('button', { name: '确认停用', exact: true }).click();
      await expect(page.locator('.object-form')).toHaveCount(0);
      await expect(page.getByRole('button', { name: '更多操作', exact: true })).toBeEnabled();
    }
    if (action === 'ADD') await page.getByRole('button', { name: /^(新增记录|插入记录)$/, exact: true }).click();
    else {
      await openMoreActions(page);
      await page.getByRole('menuitem', { name: action === 'CORRECT' ? '更正记录' : '重新启用业务单元', exact: true }).click();
    }
    const name = page.getByRole('textbox', { name: '中文名称', exact: true });
    expect(await fieldAppearance(name)).toEqual(nameStyle);
    const columns = page.locator('.object-detail-columns:visible');
    expect(await columns.evaluate(e => getComputedStyle(e).gridTemplateColumns.split(' ').length)).toBe(2);
    expect(await page.getByRole('button', { name: action === 'REACTIVATE' ? '确认重新启用' : '保存', exact: true }).evaluate(e => Math.round(e.getBoundingClientRect().height))).toBe(saveHeight);
    if (action !== 'CORRECT') {
      expect(await fieldAppearance(page.getByLabel('生效日期'))).toEqual(dateStyle);
      expect(await fieldAppearance(page.getByRole('combobox', { name: '操作原因', exact: true }))).toEqual(reasonStyle);
      await expect(page.getByLabel('生效日期')).toHaveAttribute('required', '');
    }
    await expect(page.getByRole('checkbox', { name: /明确修改/ })).toHaveCount(0);
    await expect(page.locator('.object-form')).not.toContainText(/已有后继资料保持独立快照|不填写更正原因|旧来源及原采用保持|只提交明确修改字段/);
    await page.setViewportSize({ width: 899, height: 900 });
    expect(await fieldAppearance(name)).toEqual(nameStyle);
    expect(await columns.evaluate(e => getComputedStyle(e).gridTemplateColumns.split(' ').length)).toBe(1);
    await name.fill('重排后输入保持');
    await page.setViewportSize({ width: 1440, height: 900 });
    await expect(name).toHaveValue('重排后输入保持');
    await page.getByRole('button', { name: '取消编辑', exact: true }).click();
    await page.getByRole('button', { name: '放弃修改', exact: true }).click();
    await expect(page.getByRole('button', { name: '更多操作', exact: true })).toBeFocused();
  }
});

test('更多操作移除查看日期，维护入口及键盘焦点保持', async ({ page }) => {
  await login(page); await create(page);
  const more = page.getByRole('button', { name: '更多操作', exact: true });
  await expect(page.getByRole('button', { name: '更正记录', exact: true })).toHaveCount(0);
  await more.focus(); await page.keyboard.press('Enter');
  await expect(page.getByRole('menuitem', { name: '更正记录', exact: true })).toBeFocused();
  await expect(page.getByText('所选记录：2026-01-01', { exact: true })).toBeVisible();
  await page.keyboard.press('Escape'); await expect(more).toBeFocused();
  await more.click();
  await expect(page.getByRole('menuitem',{name:'查看日期…',exact:true})).toHaveCount(0);
  await page.keyboard.press('Escape'); await expect(more).toBeFocused();
  expect(new URL(page.url()).searchParams.has('businessAt')).toBe(false);
});

test('只读来源保持准确记录且不提供空操作菜单', async ({page}) => {
 await login(page); await create(page);
 await page.getByRole('tab',{name:'变更日志',exact:true}).click(); await page.getByRole('link',{name:'查看当次资料',exact:true}).click();
 await expect(page.getByText('正在查看只读历史资料',{exact:true})).toBeVisible();
 const source=new URL(page.url()).searchParams.get('sourceVersion'); await page.reload();
 await expect(page.getByRole('button',{name:'更多操作',exact:true})).toHaveCount(0);
 await expect(page.getByRole('button',{name:'新增记录',exact:true})).toHaveCount(0);
 expect(new URL(page.url()).searchParams.get('sourceVersion')).toBe(source);
 await expect(page.getByRole('heading',{name:/菜单验证单元/})).toBeVisible();
});

test('更正入口绑定所选记录，编辑操作集中于控制栏且未保存保护保持', async ({ page }) => {
  await login(page); await create(page);
  await page.getByRole('tab', { name: '变更日志', exact: true }).click();
  await openMoreActions(page);
  await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
  await expect(page.getByRole('heading', { name: '更正记录：2026-01-01', exact: true })).toBeFocused();
  await expect(page.getByRole('tab', { name: '基本资料', exact: true })).toHaveAttribute('aria-selected', 'true');
  await expect(page.getByRole('button', { name: '更多操作', exact: true })).toBeDisabled();
  await expect(page.locator('.record-control').getByRole('button', { name: '预览', exact: true })).toHaveCount(0);
  await expect(page.locator('.record-control').getByRole('button', { name: '保存', exact: true })).toBeEnabled();
  await page.getByRole('textbox', { name: '中文名称', exact: true }).fill('尚未保存');
  await page.getByRole('link', { name: '返回列表', exact: true }).click();
  await expect(page.getByRole('dialog', { name: '尚有未保存修改', exact: true })).toBeVisible();
  await page.getByRole('button', { name: '继续编辑', exact: true }).click();
  await expect(page.getByRole('textbox', { name: '中文名称', exact: true })).toHaveValue('尚未保存');
  await page.getByRole('button', { name: '取消编辑', exact: true }).click();
  await page.getByRole('button', { name: '放弃修改', exact: true }).click();
  await expect(page.getByRole('button', { name: '更多操作', exact: true })).toBeFocused();
  await expect(page.getByRole('tab', { name: '变更日志', exact: true })).toHaveAttribute('aria-selected', 'true');
  await page.getByRole('button', { name: /^(新增记录|插入记录)$/, exact: true }).click();
  await expect(page.getByRole('heading', { name: /^(新增记录|插入记录)$/, exact: true })).toBeFocused();
  await expect(page.getByRole('tab', { name: '基本资料', exact: true })).toHaveAttribute('aria-selected', 'true');
  await page.getByRole('button', { name: '取消编辑', exact: true }).click();
  await expect(page.getByRole('heading', { name: /菜单验证单元/ })).toBeVisible();
  await openMoreActions(page);
  await page.getByRole('menuitem', { name: '更正记录', exact: true }).click();
  await page.getByRole('textbox', { name: '中文名称', exact: true }).fill('日志入口已更正');
  await expect(page.getByRole('button', {name:'预览',exact:true})).toHaveCount(0);
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.getByRole('status').filter({ hasText: '已保存' })).toBeVisible();
  await expect(page.getByRole('tab', { name: '基本资料', exact: true })).toHaveAttribute('aria-selected', 'true');
  await expect(page.getByRole('heading', { name: /日志入口已更正/ })).toBeVisible();
});

test('只读主体没有空更多菜单，仍能读取生效记录', async ({page}) => {
 await login(page); const {code}=await create(page); await page.getByRole('button',{name:'退出',exact:true}).click(); await login(page,true);
 await page.getByLabel('搜索编码或名称').fill(code); await page.getByRole('button',{name:'查询',exact:true}).click();
 await page.getByRole('row').filter({hasText:code}).getByRole('link').click();
 await expect(page.getByRole('heading',{name:/菜单验证单元/})).toBeVisible();
 await expect(page.getByRole('button',{name:'更多操作',exact:true})).toHaveCount(0);
 await expect(page.getByRole('button',{name:'新增记录',exact:true})).toHaveCount(0);
 await expect(page.locator('.effective-dates')).toBeVisible();
});

test('详情更多操作在桌面可达且无查看日期', async ({page}) => {
 await login(page);await create(page);
 const more=page.getByRole('button',{name:'更多操作',exact:true});await more.focus();await expect(more).toBeInViewport();await more.click();
  await expect(page.getByRole('menuitem',{name:'更正记录',exact:true})).toBeInViewport();await expect(page.getByRole('menuitem',{name:'查看日期…',exact:true})).toHaveCount(0);
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await page.keyboard.press('Escape');await expect(more).toBeFocused();
});

test('记录读取失败清理旧资料，重试恢复准确记录', async ({page}) => {
 await login(page);await create(page);const target=new URL(page.url()).searchParams.get('selectedRecordId');
 const pattern='**/api/v1/business-units/*?*';await page.route(pattern,route=>route.fulfill({status:503,json:{code:'M02_DEPENDENCY_UNAVAILABLE',message:'记录读取临时失败'}}));
 await page.reload();await expect(page.getByText('记录读取临时失败',{exact:true})).toBeVisible();await expect(page.getByRole('heading',{name:/菜单验证单元/})).toHaveCount(0);
 await page.unroute(pattern);await page.getByRole('button',{name:'重试',exact:true}).click();await expect(page.getByRole('heading',{name:/菜单验证单元/})).toBeVisible();
 expect(new URL(page.url()).searchParams.get('selectedRecordId')).toBe(target);
});
