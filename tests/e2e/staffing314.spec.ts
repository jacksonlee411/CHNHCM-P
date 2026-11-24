import { randomUUID } from 'node:crypto';
import { test, expect } from './fixtures';
import { open, choose } from './bu-support';
import { request } from './dated-object-support';
import { staffingPageSuite } from './staffing-page-suite';
import { withPolicy } from './grants';
import { action308, chooseClassification308, menu308, click308, base308, seed308 } from './staffing308-support';

staffingPageSuite(['jobfunction'], '314');

test('314 AC082 从职类菜单建立管理类和专业类，只读者直接提交被拒', async ({ page }) => {
  await open(page);
  const code = 'T02-' + randomUUID().slice(0, 8);
  await seed308(page, 'jobfamily', undefined, { code });
  const formLayout = () => page.locator('.object-form').evaluate(form => ({
    columns: [...form.querySelectorAll('.object-detail-columns > section')].map(el => el.getBoundingClientRect().left),
    fields: [...form.querySelectorAll('input[id^="staffing-"],textarea[id^="staffing-"]')].map(el => ({
      id: el.id, height: el.getBoundingClientRect().height, font: getComputedStyle(el).fontSize,
    })),
    label: getComputedStyle(form.querySelector('#staffing-name-label')!).fontSize,
  }));
  await menu308(page, 'jobfamily');
  await click308(page, page.getByRole('link', { name: '新建', exact: true }));
  await expect(page.getByRole('textbox', { name: '名称', exact: true })).toBeVisible();
  const familyLayout = await formLayout();
  await click308(page, page.getByRole('button', { name: '取消', exact: true }));
  for (const [suffix, name] of [['', '管理类'], ['-P', '专业类']]) {
    await menu308(page, 'jobfunction');
    await click308(page, page.getByRole('link', { name: '新建', exact: true }));
    await expect(page.getByLabel('生效日期').and(page.locator('input'))).toHaveValue('1900-01-01');
    expect(await formLayout()).toEqual(familyLayout);
    await page.getByRole('textbox', { name: '编码', exact: true }).fill(code);
    await page.getByRole('textbox', { name: '名称', exact: true }).fill(name);
    await click308(page, page.getByRole('button', { name: '保存', exact: true }));
    if (suffix) {
      const input = page.getByRole('textbox', { name: '编码', exact: true });
      await expect(input).toBeFocused();
      await expect(input).toHaveAttribute('aria-invalid', 'true');
      await expect(page.getByRole('textbox', { name: '名称', exact: true })).toHaveValue(name);
      await input.fill(code + suffix);
      await click308(page, page.getByRole('button', { name: '保存', exact: true }));
    }
    await expect(page.getByRole('heading', { level: 1, name, exact: true })).toBeVisible();
  }
  await page.context().clearCookies();
  await open(page, 'reader');
  await menu308(page, 'jobfunction');
  await expect(page.getByRole('link', { name: '新建', exact: true })).toHaveCount(0);
  await page.getByRole('textbox', { name: '搜索编码或名称', exact: true }).fill(code);
  await click308(page, page.getByRole('button', { name: '查询', exact: true }));
  await expect(page.getByText('1–2 / 2', { exact: true })).toBeVisible();
  await click308(page, page.getByRole('checkbox', { name: '全选查询结果', exact: true }));
  await click308(page, page.getByRole('button', { name: '更多操作', exact: true }));
  const downloaded = page.waitForEvent('download');
  await click308(page, page.getByRole('menuitem', { name: '导出所选（2）', exact: true }));
  const file = await (await downloaded).createReadStream();
  const chunks: Buffer[] = [];
  for await (const chunk of file!) chunks.push(chunk);
  expect(Buffer.concat(chunks).toString('utf8').trim().split(/\r?\n/)).toEqual([
    '编码,名称,英文名称,说明,状态,生效日期',
    code.toUpperCase() + ',管理类,,,启用,1900-01-01',
    code.toUpperCase() + '-P,专业类,,,启用,1900-01-01',
  ]);
  const write = await request(page, 'POST', base308.jobfunction, {
    code: code + '-R', name: '只读者提交', operationType: 'CREATE', operationReason: 'NEW_JOB_FUNCTION',
  }, randomUUID());
  expect(write.status).toBe(403);
});

for (const width of [1440, 390]) {
  test(`314 T04 双维查询、返回与固定选择 ${width}px`, async ({ page }) => {
    test.setTimeout(120000);
    await open(page);
    const tag = 'T04-' + randomUUID().slice(0, 6);
    const catalog = async (kind: 'jobfamily' | 'jobfunction', name: string) =>
      seed308(page, kind, undefined, { name: name + tag, date: '1900-01-01' });
    const hr = await catalog('jobfamily', '人力资源'), finance = await catalog('jobfamily', '财务');
    const management = await catalog('jobfunction', '管理类'), professional = await catalog('jobfunction', '专业类');
    const jobs = [
      ['人力资源经理', hr, management],
      ['人力资源专员', hr, professional],
      ['财务经理', finance, management],
    ] as const;
    for (const [index, [name, family, fn]] of jobs.entries()) {
      const created = await request(page, 'POST', base308.job, {code: tag + index, name: name + tag,
        effectiveDate: '2026-01-01', operationType: 'CREATE', operationReason: 'NEW_JOB',
        jobFamilyId: family.id, jobFunctionId: fn.id}, randomUUID());
      expect(created.status).toBe(201);
    }
    await page.setViewportSize({width, height: 844});
    await menu308(page, 'job');
    const listMore = page.locator('.list-control').getByRole('button', {name:'更多操作',exact:true});
    const openFilter = async (label: string) => {
      if (width > 900) {
        const header = page.getByRole('columnheader', {name: new RegExp(label)});
        await header.hover();
        await header.getByLabel('列操作').click();
        await page.getByRole('menuitem', {name:'筛选…', exact:true}).click();
      } else {
        await listMore.click();
        await page.getByRole('menuitem', {name:'列筛选…',exact:true}).click();
        await choose(page, '列', label);
      }
    };
    const button = (name: string) => page.getByRole('button', {name, exact:true});
    const count = (n: number) => expect(page.getByText(`1–${n} / ${n}`, {exact:true})).toBeVisible();
    await openFilter('职类');
    await choose(page,'职类',management.name);
    await button('应用').click();
    await count(2);
    await openFilter('职务系列');
    await choose(page,'职务系列',hr.name);
    await button('应用').click();
    await count(1);
    const manager = page.getByRole('link',{name:'人力资源经理'+tag,exact:true});
    await expect(manager).toBeVisible();
    await expect(page.getByText('职务系列：'+hr.name+' · 职类：'+management.name,{exact:true})).toBeVisible();
    await page.screenshot({path: test.info().outputPath('classification-intersection.png'),
      mask: [page.locator('.app-account-name')]});
    await openFilter('职务系列');
    await choose(page,'职务系列',finance.name);
    await button('取消').click();
    await count(1);
    await manager.click();
    await expect(page.getByText('1 / 1',{exact:true})).toBeVisible();
    await page.getByRole('link',{name:'返回列表',exact:true}).click();
    await expect(page.locator('.list-return-selected')).toContainText('人力资源经理'+tag);
    await openFilter('职类');
    await button('清除条件').click();
    await button('应用').click();
    await count(2);
    expect(new URL(page.url()).searchParams.get('jobFamilyId')).toBe(hr.id);
    await manager.click();
    await button('下一个对象').click();
    await expect(page.getByRole('heading',{level:1, name:'人力资源专员'+tag})).toBeVisible();
    await button('上一个对象').click();
    await expect(page.getByRole('heading',{level:1, name:'人力资源经理'+tag})).toBeVisible();
    let releaseList!:()=>void;
    const pendingList = new Promise<void>(resolve=>{releaseList=resolve});
    const listRequest = '**/api/v1/jobs?**';
    await page.route(listRequest, async route=>{await pendingList;await route.fallback()});
    try {
      await page.getByRole('link',{name:'返回列表',exact:true}).click();
      await expect(listMore).toBeDisabled();
    } finally {
      releaseList();
    }
    await expect(manager).toBeFocused();
    await openFilter('职类');
    await choose(page,'职类',management.name);
    await button('应用').click();
    await count(1);
    await page.getByRole('checkbox',{name:'全选查询结果',exact:true}).click();
    await button('更多操作').click();
    const download = page.waitForEvent('download');
    await page.getByRole('menuitem',{name:'导出所选（1）',exact:true}).click();
    const stream = await (await download).createReadStream();
    const chunks = [];
    for await (const chunk of stream!) chunks.push(chunk);
    expect(Buffer.concat(chunks).toString('utf8').trim().split(/\r?\n/)).toEqual([
      '编码,名称,英文名称,职务系列,主要职责,任职要求,状态,生效日期,职类',
      [tag.toUpperCase()+'0','人力资源经理'+tag,'',hr.name,'','','启用','2026-01-01',management.name].join(','),
    ]);
    await withPolicy(page, async (where, mutate) => {
      mutate(`DELETE FROM access.subject_grant WHERE ${where} AND action='jobfunction.read'`);
      await page.reload();
      await expect(page.getByRole('alert').filter({hasText:'职类筛选权限已变化'})).toBeVisible();
      await button('移除职类筛选').click();
      await count(2);
      await expect(page.getByText(management.name,{exact:true})).toHaveCount(0);
      await openFilter('职类');
      await choose(page, '条件', '未登记职类');
      await button('应用').click();
      await expect(page.getByText('没有符合条件的职务',{exact:true})).toBeVisible();
      expect(new URL(page.url()).searchParams.get('jobFamilyId')).toBe(hr.id);
    });
  });
}

test('314 T04 系列先选、单维更正及固定恢复端迟到响应', async ({page}) => {
  test.setTimeout(120000);
  await open(page);
  const tag = 'T04-' + randomUUID().slice(0,8);
  const family = await seed308(page,'jobfamily',undefined,{date:'1900-01-01'});
  const first = await seed308(page,'jobfunction',undefined,{date:'1900-01-01'});
  const second = await seed308(page,'jobfunction',undefined,{date:'1900-01-01'});
  const future = await seed308(page,'jobfunction',undefined,{date:'2031-01-01'});
  await menu308(page,'job');
  await page.getByRole('link',{name:'新建',exact:true}).click();
  await page.getByRole('textbox',{name:'编码',exact:true}).fill(tag);
  await page.getByRole('textbox',{name:'名称',exact:true}).fill('职务'+tag);
  await chooseClassification308(page,'jobFamilyId',family.name);
  await chooseClassification308(page,'jobFunctionId',first.name);
  const save = page.getByRole('button',{name:'保存',exact:true});
  await save.click();
  await expect(page.getByRole('heading',{level:1,name:'职务'+tag})).toBeVisible();
  const path = base308.job + '/' + new URL(page.url()).pathname.split('/').at(-1);
  const view = async (source?: string) => (await request(page,'GET',path + (source ? '?sourceVersion='+source : ''))).body;
  const original = await view();
  const historical = (await view(original.headSourceVersion)).selectedRecord;
  await action308(page,'更正记录');
  await withPolicy(page, async (where, mutate) => {
    mutate(`DELETE FROM access.subject_grant WHERE ${where} AND action='jobfamily.read'`);
    await chooseClassification308(page,'jobFunctionId',second.name);
    await save.click();
    await expect(page.locator('.object-form')).toHaveCount(0);
  });
  const changed = await view();
  const adopted = (await view(changed.headSourceVersion)).selectedRecord;
  expect(adopted.jobFunctionId).toBe(second.id);
  expect(adopted.adoption).toEqual(historical.adoption);
  const nextFamily = await seed308(page,'jobfamily',undefined,{date:'1900-01-01'});
  await action308(page,'更正记录');
  await chooseClassification308(page,'jobFamilyId',nextFamily.name);
  await save.click();
  await expect(page.locator('.object-form')).toHaveCount(0);
  const beforePair = await view();
  expect((await view(beforePair.headSourceVersion)).selectedRecord.jobFunctionAdoption).toEqual(adopted.jobFunctionAdoption);
  await action308(page,'插入停用与重新启用');
  await page.getByLabel('停用日期').and(page.locator('input')).fill('2029-01-01');
  const resumeDate = page.getByLabel('重新启用日期').and(page.locator('input'));
  await resumeDate.fill('2032-01-01');
  await chooseClassification308(page,'jobFunctionId',future.name);
  await expect(page.getByRole('dialog',{name:'选择职类'})).toHaveCount(0);
  let release!: () => void;
  let handled!: () => void;
  const oldResponseHandled = new Promise<void>(done => { handled = done; });
  const wait = new Promise<void>(done => { release=done; });
  let held = false;
  await page.route('**/api/v1/jobs/function-reference-context',async route => {
    if (route.request().postDataJSON().reactivationDate !== '2033-01-01') return route.continue();
    const response = await route.fetch();
    const eligible = await response.json();
    held = eligible.selectedName === future.name && eligible.items.some((item: {id:string}) => item.id === future.id);
    await wait;
    await route.fulfill({response}).catch(() => {});
    handled();
  });
  try {
    await resumeDate.fill('2033-01-01');
    await expect.poll(() => held).toBe(true);
    await resumeDate.fill('2030-01-01');
    await expect(page.getByText('该日期下所选职类不可用',{exact:true})).toBeVisible();
    await page.locator('#staffing-jobFunctionId').click();
    const dialog = page.getByRole('dialog',{name:'选择职类'});
    await dialog.getByRole('textbox').fill(future.name);
    await expect(dialog.getByText('没有可用职类，请调整日期或搜索内容。')).toBeVisible();
    release();
    await oldResponseHandled;
    await expect(dialog.getByRole('button',{name:future.name,exact:true})).toHaveCount(0);
    await dialog.getByRole('button',{name:'取消',exact:true}).click();
  } finally { release(); }
  await expect(page.getByRole('dialog',{name:'选择职类'})).toHaveCount(0);
  await expect(page.locator('#staffing-jobFamilyId')).toHaveText(nextFamily.name);
  await page.getByRole('dialog',{name:'插入停用与重新启用',exact:true}).getByRole('button',{name:'取消',exact:true}).click();
  await page.getByRole('button',{name:'放弃修改',exact:true}).click();
  expect((await view()).headSourceVersion).toBe(beforePair.headSourceVersion);
  await page.getByRole('tab',{name:'变更日志',exact:true}).click();
  await page.getByRole('link',{name:'查看当次资料',exact:true}).last().click();
  await expect(page.getByText('正在查看只读历史资料',{exact:true})).toBeVisible();
  await expect(page.getByText(first.name,{exact:true})).toBeVisible();
  await withPolicy(page, async (where, mutate) => {
    mutate(`DELETE FROM access.subject_grant WHERE ${where} AND action='jobfunction.history.read'`);
    await page.reload();
    await expect(page.getByText('受限',{exact:true})).toBeVisible();
    await expect(page.getByText(first.name,{exact:true})).toHaveCount(0);
    await page.getByRole('link',{name:'返回变更日志',exact:true}).click();
    await expect(page.locator('.change-operation')).toHaveCount(3);
    for (const name of [first.name, second.name]) {
      await expect(page.locator('.change-operation').filter({hasText:name})).toHaveCount(0);
    }
  });
});

test('314 T05 AC090 仅308从职类目录到职务双维保存与查询', async ({ page }) => {
  await open(page);
  const tag = 'T05-' + randomUUID().slice(0, 8);
  const family = await seed308(page, 'jobfamily', undefined, {date: '1900-01-01', name: '人力资源' + tag});
  const functionName = '管理类' + tag;
  const jobName = '人力资源经理' + tag;
  await menu308(page, 'jobfunction');
  await page.getByRole('link', {name: '新建', exact: true}).click();
  await page.getByRole('textbox', {name: '编码', exact: true}).fill(tag);
  await page.getByRole('textbox', {name: '名称', exact: true}).fill(functionName);
  await page.getByRole('button', {name: '保存', exact: true}).click();
  await expect(page.getByRole('heading', {level: 1, name: functionName})).toBeVisible();
  const functionId = new URL(page.url()).pathname.split('/').at(-1);

  await menu308(page, 'job');
  await page.getByRole('link', {name: '新建', exact: true}).click();
  await page.getByRole('textbox', {name: '编码', exact: true}).fill(tag);
  await page.getByRole('textbox', {name: '名称', exact: true}).fill(jobName);
  await chooseClassification308(page, 'jobFunctionId', functionName);
  await chooseClassification308(page, 'jobFamilyId', family.name);
  await page.getByRole('button', {name: '保存', exact: true}).click();
  await expect(page.getByRole('heading', {level: 1, name: jobName})).toBeVisible();
  const jobId = new URL(page.url()).pathname.split('/').at(-1);
  const saved = await request(page, 'GET', base308.job + '/' + jobId);
  expect(saved.status).toBe(200);
  expect(saved.body.selectedRecord).toMatchObject({jobFunctionId: functionId, jobFamilyId: family.id});

  await seed308(page, 'job', family.id, {name: '人力资源专员' + tag});
  await page.context().clearCookies();
  await open(page, 'reader');
  await menu308(page, 'job');
  for (const [label, selectedName, count] of [
    ['职务系列', family.name, 2], ['职类', functionName, 1],
  ] as const) {
    const header = page.getByRole('columnheader', {name: new RegExp(label)});
    await header.hover();
    await header.getByLabel('列操作').click();
    await page.getByRole('menuitem', {name: '筛选…', exact: true}).click();
    await choose(page, label, selectedName);
    await page.getByRole('button', {name: '应用', exact: true}).click();
    await expect(page.getByText(`1–${count} / ${count}`, {exact: true})).toBeVisible();
  }
  const query = new URL(page.url()).searchParams;
  expect(query.get('jobFamilyId')).toBe(family.id);
  expect(query.get('jobFunctionId')).toBe(functionId);
  const matches = await request(page, 'GET', base308.job + '?' + query);
  expect(matches.status).toBe(200);
  expect(matches.body.items.map((item: {id: string}) => item.id)).toEqual([jobId]);
  await page.getByRole('link', {name: jobName, exact: true}).click();
  await expect(page.getByText('1 / 1', {exact: true})).toBeVisible();
  await page.getByRole('link', {name: '返回列表', exact: true}).click();
  await expect(page.locator('.list-return-selected')).toContainText(jobName);
  await expect(page.getByRole('link', {name: jobName, exact: true})).toBeFocused();
  await page.screenshot({path: test.info().outputPath('reader-return.png'), mask: [page.locator('.app-account-name')]});
});
