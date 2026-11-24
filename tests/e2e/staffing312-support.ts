import { randomUUID } from 'node:crypto';
import { expect, test, type Page } from './fixtures';
import { choose } from './bu-support';
import { request } from './dated-object-support';
import { action308, click308, date308, evidence308, namedMenu308, reach308, reference308, seed308 } from './staffing308-support';

const menu = (page: Page) => namedMenu308(page, '岗位', '职务配置', '/settings/staffing/positions');
export async function seedPosition(page: Page, _kind: 'position', family: string) {
  const suffix = randomUUID().slice(0, 8), department = '部门' + suffix, location = '地点' + suffix;
  const create = async (path: string, code: string, name: string, reason: string, extra = {}) => {
    const result = await request(page, 'POST', '/api/v1/' + path, { code, name, operationType: 'CREATE', operationReason: reason, effectiveDate: '2026-01-01', ...extra }, randomUUID());
    expect(result.status, `${path}: ${result.body.message ?? ''}`).toBe(201);
    return result.body;
  };
  const company = await create('business-units', 'BP' + suffix, '管理单元' + suffix, 'NEW_ORGANIZATION');
  const dep = await create('departments', 'DP' + suffix, department, 'NEW_ORGANIZATION', { managementBuId: company.buId });
  const countries = await request(page, 'GET', '/api/v1/country-regions?q=MO');
  const country = countries.body.items.find((item: {code:string}) => item.code === 'MO');
  expect(country).toBeTruthy();
  const loc = await create('locations', 'LP' + suffix, location, 'NEW_LOCATION', { countryRegionId: country.id, addressLine: '测试街一号' });
  const grade = await seed308(page, 'grade'), job = await seed308(page, 'job', family, { gradeIds: [grade.id] });
  const refs = { departmentId: dep.departmentId, locationId: loc.locationId, jobId: job.id };
  const code = 'RP' + suffix, name = '留空岗位' + suffix;
  const sample = await create('positions', code, name, 'NEW_POSITION', refs);
  return { code, name, id: sample.objectId as string, department, location, job, grade };
}

async function pick(page: Page, field: string, label: string, name: string) {
  await click308(page, page.locator('#staffing-' + field));
  const dialog = page.getByRole('dialog', { name: '选择' + label, exact: true });
  await dialog.getByRole('textbox', { name: `搜索${label}编码或名称` }).fill(name);
  const option = dialog.getByRole('button', { name: new RegExp('^' + name + '（') });
  if (field === 'gradeId') {
    await reach308(page, option);
    await dialog.getByRole('textbox').press('Tab');
    await expect(option).toBeFocused();
    await page.keyboard.press('Enter');
  } else await click308(page, option);
  await expect(dialog).toHaveCount(0);
}

export async function positionJourney(page: Page, sample: Awaited<ReturnType<typeof seedPosition>>, _family: unknown, role: string) {
  await reference308(page, role);
  await menu(page);
  const save = () => click308(page, page.getByRole('button', { name: '保存', exact: true }));
  const name = page.getByRole('textbox', { name: '名称', exact: true });
  const date = page.getByLabel('生效日期').and(page.locator('input'));
  if (role === 'admin') {
    await click308(page, page.getByRole('link', { name: '新建', exact: true }));
    await expect(date).toHaveValue('1900-01-01');
    await expect(page.locator('#staffing-gradeId')).toHaveText('未指定职级');
    await name.fill(sample.name + '新');
    await page.getByRole('textbox', { name: '编码', exact: true }).fill('N' + sample.code);
    await date.fill('2026-01-01');
    await pick(page, 'departmentId', '所属部门', sample.department);
    await pick(page, 'jobId', '职务', sample.job.name);
    await save();
    await expect(page.locator('#staffing-error')).toContainText('工作地点');
    await expect(page.locator('#staffing-locationId')).toBeFocused();
    await expect(page.locator('#staffing-locationId')).toHaveAccessibleDescription(/工作地点/);
    await pick(page, 'locationId', '工作地点', sample.location);
    await expect(page.locator('#staffing-error')).toHaveCount(0);
    const viewport = await page.evaluate(() => [innerWidth,innerHeight].join('x'));
    await page.screenshot({path:test.info().outputPath(`form-${viewport}.png`),mask:[page.locator('.app-account-name')]});
    await save();
    await expect(page.getByRole('heading', {level: 1, name: sample.name + '新', exact: true})).toBeVisible();
    await action308(page, '新增记录');
    await date.fill('2027-02-01');
    await pick(page, 'gradeId', '职级', sample.grade.name);
    await choose(page, '操作原因', '岗位归属调整');
    await save();
    await date308(page, '2027-02-01');
    await action308(page, '更正记录');
    await expect(page.locator('#staffing-gradeId')).toHaveText(sample.grade.name);
    await name.fill(sample.name + '放弃');
    await click308(page, page.getByRole('button', {name:'取消', exact:true}));
    await click308(page, page.getByRole('button', {name:'放弃修改', exact:true}));
    await action308(page, '更正记录');
    await expect(name).toHaveValue(sample.name + '新');
    await name.fill(sample.name + '更正');
    await save();
    await date308(page, '2027-02-01');
    await action308(page, '停用');
    await date.fill('2027-05-01');
    await save();
    await date308(page, '2027-05-01');
  } else {
    await expect(page.getByRole('link', {name:'新建', exact:true})).toHaveCount(0);
    await page.getByRole('textbox', {name:'搜索编码或名称', exact:true}).fill(sample.code);
    await click308(page, page.getByRole('button', {name:'查询', exact:true}));
    await click308(page, page.getByRole('link', {name:sample.name, exact:true}));
    await expect(page.getByRole('button', {name:'更多操作', exact:true})).toHaveCount(0);
  }
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
  await click308(page, page.getByRole('tab', {name:'变更日志', exact:true}));
  await click308(page, page.getByRole('link', {name:'查看当次资料', exact:true}).last());
  await reach308(page, page.getByText('正在查看只读历史资料', {exact:true}));
  await expect(page.getByText(sample.department, {exact:true}).first()).toBeVisible();
  await click308(page, page.getByRole('link', {name:'返回变更日志', exact:true}));
  await expect(page.getByRole('tab', {name:'变更日志', exact:true})).toHaveAttribute('aria-selected','true');
  await evidence308(page, 'position-journey');
}
