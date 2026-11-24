import type { Page } from './fixtures';

// Click the real application entry without waiting for navigation: callers
// exercising an unsaved-draft dialog must be able to answer it first.
export async function chooseApplication(page: Page, label: '部门' | '设置') {
  if (await page.getByRole('button', { name: '打开导航', exact: true }).isVisible()) {
    await page.getByRole('button', { name: '打开导航', exact: true }).click();
    const drawer = page.locator('#navigation-drawer');
    const app = drawer.getByRole('button', { name: label, exact: true });
    if (label === '部门') { await app.click(); return; }
    if (await app.getAttribute('aria-expanded') !== 'true') await app.click();
    const section = drawer.getByRole('button', { name: '组织配置', exact: true });
    if (await section.getAttribute('aria-expanded') !== 'true') await section.click();
    await drawer.getByRole('button', { name: '业务单元', exact: true }).click();
  } else {
    await page.getByRole('button', { name: '切换应用', exact: true }).click();
    await page.getByRole('menuitem', { name: label, exact: true }).click();
  }
}
