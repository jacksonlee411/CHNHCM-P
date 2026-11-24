import { lazyCredentials } from './credentials.mjs';
import { readRecord } from './fixtures';
import { expect, openMoreActions, readCurrentDate, type Page } from './fixtures';
import { readFileSync } from 'node:fs';
import { execFileSync } from 'node:child_process';
import { randomUUID } from 'node:crypto';
const credentials= lazyCredentials();
export async function open(page: Page, login = credentials.login) { await new Promise(r => setTimeout(r, 3100)); await page.goto('/'); await page.getByLabel('登录名').fill(login); await page.getByLabel('密码').fill(credentials.password); await page.getByRole('button', { name: '登录', exact: true }).click(); await page.getByRole('button', { name: '切换应用', exact: true }).click(); await page.getByRole('menuitem', { name: '设置', exact: true }).click(); await page.getByRole('button', { name: '组织配置', exact: true }).click(); await page.getByRole('menuitem', { name: '业务单元', exact: true }).click(); return readCurrentDate(page); }
export async function create(page: Page, date = '2026-01-01', englishName = 'Alpha') {
    const code = 'JOINT_' + randomUUID().slice(0, 8);
    await page.getByRole('link', { name: '新建', exact: true }).click();
    await page.getByRole('textbox', { name: '编码', exact: true }).fill(code);
    await page.getByRole('textbox', { name: '中文名称', exact: true }).fill('甲');
    await page.getByRole('textbox', { name: '英文名称', exact: true }).fill(englishName);
    await page.getByLabel('生效日期').and(page.locator('input')).fill(date);
    await choose(page, '操作原因', '新设组织');
    await page.getByRole('button', { name: '保存', exact: true }).click();
    await expect(page.locator('.object-form')).toHaveCount(0);
    return code;
}
export async function choose(page: Page | ReturnType<Page['locator']>, label: string, name: string) { await page.getByRole('combobox', { name: label, exact: true }).click(); const owner = 'page' in page ? page.page() : page; await owner.getByRole('option', { name, exact: true }).click(); }
export async function add(page: Page, date: string, name: string, en?: string) {
    const d=await read(page), anchor=d.timeline.filter((r:{effectiveDate:string})=>r.effectiveDate<date).at(-1)??d.timeline[0];
    await select(page,anchor.effectiveDate);
    await page.getByRole('button', { name: /^(新增记录|插入记录)$/, exact: true }).click();
    await page.getByLabel('生效日期').and(page.locator('input')).fill(date);
    await choose(page, '操作原因', '名称调整');
    await page.getByRole('textbox', { name: '中文名称', exact: true }).fill(name);
    if (en)
        await page.getByRole('textbox', { name: '英文名称', exact: true }).fill(en);
    await readyToSave(page);
    await page.getByRole('button', { name: '保存', exact: true }).click();
    await expect(page.locator('.object-form')).toHaveCount(0);
}
export async function life(page: Page, type: 'DEACTIVATE' | 'REACTIVATE', date: string, name?: string, en?: string) {
    await openMoreActions(page);
    await page.getByRole('menuitem', { name: type === 'DEACTIVATE' ? '停用业务单元' : '重新启用业务单元', exact: true }).click();
    await page.getByLabel('生效日期').and(page.locator('input')).fill(date);
    await choose(page, '操作原因', type === 'DEACTIVATE' ? '业务终止' : '业务恢复');
    if (name)
        await page.getByRole('textbox', { name: '中文名称', exact: true }).fill(name);
    if (en)
        await page.getByRole('textbox', { name: '英文名称', exact: true }).fill(en);
    await readyToSave(page);
    await page.getByRole('button', { name: type === 'DEACTIVATE' ? '确认停用' : '确认重新启用', exact: true }).click();
    await expect(page.locator('.object-form')).toHaveCount(0);
}
export async function readyToSave(page: Page) { await expect(page.getByRole('button', { name: '预览', exact: true })).toHaveCount(0); await expect(page.getByText('正在读取操作影响…', {exact:true})).toHaveCount(0); }
export async function enter(page: Page, name:string) { await openMoreActions(page); await page.getByRole('menuitem', { name, exact: true }).click(); await expect(page.getByRole('heading', {name: name === '更正记录' ? /^更正记录：/ : name,exact:true})).toBeVisible(); }
export async function expectSingleClassification(page: Page, readable = ['操作类型', '操作原因']) {
    const form = page.locator('.object-form');
    for (const label of ['操作类型', '操作原因'])
        await expect(form.locator('.object-field-label').filter({hasText: new RegExp(`^${label}$`)})).toHaveCount(readable.includes(label) ? 1 : 0);
    for (const label of ['原操作类型', '原操作原因', '新操作类型', '新操作原因'])
        await expect(form.getByText(label, {exact: true})).toHaveCount(0);
}
export async function editClassification(page: Page) { await enter(page, '更正记录'); await expect(page.getByRole('combobox', {name:'操作原因',exact:true})).toBeVisible(); }
export async function save(page:Page){const response=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/correct'));await page.getByRole('button',{name:'保存',exact:true}).click();const r=await response;expect(r.status()).toBe(200);await expect(page.locator('.object-form')).toHaveCount(0);return r.json()}
export async function read(page: Page, date?: string, source?: string) { return readRecord(page, date, source); }
export async function select(page: Page, date: string) { const [y, m, d] = date.split('-'); await page.getByRole('button', { name: `${y}-${m}-${d}`, exact: true }).click(); await expect(page.getByRole('button', { name: `${y}-${m}-${d}`, exact: true })).toHaveAttribute('aria-current', 'date'); }
