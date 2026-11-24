import { chromium, expect } from '@playwright/test';
const input = JSON.parse(await new Promise((resolve, reject) => { let value = ''; process.stdin.setEncoding('utf8'); process.stdin.on('data', part => value += part); process.stdin.on('end', () => resolve(value)); process.stdin.on('error', reject); }));
const browser = await chromium.launch();
try {
 const page = await browser.newPage({baseURL:process.argv[2]});
 await page.goto('/');await page.getByLabel('登录名').fill('admin');await page.getByLabel('密码').fill(input.password);await page.getByRole('button',{name:'登录',exact:true}).click();await expect(page.getByRole('button',{name:'切换应用',exact:true})).toBeVisible();
 for(const object of input.objects){
  await page.goto(`/settings/staffing/${object.segment}/${object.id}`);
  await expect(page.locator('.operation-summary')).toContainText('2026-01-15 08:02:03（北京时间 UTC+08:00）');
  await page.getByRole('tab',{name:'变更日志',exact:true}).click();
  await expect(page.locator('.change-operation time')).toHaveText('2026-01-15 08:02:03（北京时间 UTC+08:00）');
  await page.getByRole('button',{name:'来源明细',exact:true}).click();
  await expect(page.locator('.change-source-detail')).toContainText('2026-01-15T00:02:03.123456Z');
  await page.goto(`/settings/staffing/${object.segment}/new`);
  await expect(page.getByLabel('生效日期').and(page.locator('input'))).toHaveValue('1900-01-01');
  await page.getByRole('button',{name:'取消',exact:true}).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
 }
 console.log(JSON.stringify({objects:input.objects.map(o=>o.segment),display:'2026-01-15 08:02:03（北京时间 UTC+08:00）',raw:'2026-01-15T00:02:03.123456Z',enterpriseStart:'1984-01-01',initialFormDate:'1900-01-01',passed:true}));
}finally{await browser.close();}
