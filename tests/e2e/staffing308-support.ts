import { test, expect, type Page, type Locator } from './fixtures';
import { randomUUID } from 'node:crypto';
import { request } from './dated-object-support';
export type Kind308 = 'jobfamily' | 'jobfunction' | 'job' | 'grade' | 'gradeladder';
export const labels308 = { jobfunction: '职类', jobfamily: '职务系列', job: '职务', grade: '职级', gradeladder: '职级阶梯' };
export const web308 = { jobfunction: '/settings/staffing/job-functions', jobfamily: '/settings/staffing/job-families', job: '/settings/staffing/jobs', grade: '/settings/staffing/grades', gradeladder: '/settings/staffing/grade-ladders' };
export const base308 = { jobfunction: '/api/v1/job-functions', jobfamily: '/api/v1/job-families', job: '/api/v1/jobs', grade: '/api/v1/grades', gradeladder: '/api/v1/grade-ladders' };
const traces308 = new WeakMap<Page, unknown[]>();
export async function evidence308(page: Page, label: string) {
  await test.info().attach(label, {
    body: JSON.stringify({
      steps: traces308.get(page) ?? [],
      environment: await page.evaluate(() => ({
        userAgent: navigator.userAgent,
        font: getComputedStyle(document.body).fontFamily,
        width: innerWidth,
        height: innerHeight,
        dpr: devicePixelRatio,
      })),
    }),
    contentType: 'application/json',
  });
  traces308.set(page, []);
}
// Read geometry, then use actual wheel input. Never scrollIntoView, evaluate a
// scroll mutation, or rely on Locator.click's auto-scroll as reachability proof.
export async function reach308(page: Page, target: Locator) {
  await expect(target).toBeVisible();
  for (let n = 0; n < 45; n++) {
    const state = await target.evaluate((el) => {
      const r = el.getBoundingClientRect();
      let top = 0,
        bottom = innerHeight,
        left = 0,
        right = innerWidth,
        scroll: Element | null = null;
      for (let p = el.parentElement; p; p = p.parentElement) {
        const style = getComputedStyle(p),
          b = p.getBoundingClientRect();
        if (/auto|scroll/.test(style.overflowY) && p.scrollHeight > p.clientHeight + 1) {
          top = Math.max(top, b.top);
          bottom = Math.min(bottom, b.bottom);
          left = Math.max(left, b.left);
          right = Math.min(right, b.right);
          scroll ??= p;
        }
      }
      const s = scroll?.getBoundingClientRect();
      return {
        r: { top: r.top, bottom: r.bottom, left: r.left, right: r.right, height: r.height },
        top,
        bottom,
        left,
        right,
        x: s ? Math.max(3, Math.min(innerWidth - 5, s.right - 18)) : innerWidth - 18,
        y: s
          ? Math.max(5, Math.min(innerHeight - 5, (Math.max(0, s.top) + Math.min(innerHeight, s.bottom)) / 2))
          : innerHeight / 2,
      };
    });
    const b = state.r,
      visible = Math.min(b.bottom, state.bottom) - Math.max(b.top, state.top);
    if (visible >= Math.min(b.height, 30) - 1 && b.left >= -1 && b.right <= state.right + 1) {
      const trace = traces308.get(page) ?? [];
      trace.push({
        target: await target.evaluate(
          (el) =>
            el.getAttribute('aria-label') ?? el.getAttribute('id') ?? el.textContent?.trim().slice(0, 80),
        ),
        wheels: n,
        bounds: b,
      });
      traces308.set(page, trace);
      return {
        bounds: b,
        viewport: await page.evaluate(() => ({
          width: innerWidth,
          height: innerHeight,
          dpr: devicePixelRatio,
        })),
      };
    }
    await page.mouse.move(state.x, state.y);
    await page.mouse.wheel(
      0,
      b.top < state.top
        ? -Math.max(100, state.top - b.top + 30)
        : Math.max(100, b.bottom - state.bottom + 30),
    );
    await page.waitForTimeout(70);
  }
  throw Error('普通滚轮无法到达目标');
}
export async function click308(page: Page, target: Locator) {
  await reach308(page, target);
  await target.click();
}
export async function namedMenu308(page: Page, label: string, category: string, path: string, heading = label) {
  // Resize and native zoom update matchMedia before React replaces the old menu.
  // Choose the expected layout, then wait for that layout's actual entry.
  if (await page.evaluate(() => matchMedia('(max-width:899px)').matches)) {
    await expect(page.getByRole('button', { name: '打开导航', exact: true })).toBeVisible();
    await click308(page, page.getByRole('button', { name: '打开导航', exact: true }));
    const drawer = page.locator('#navigation-drawer');
    const settings = drawer.getByRole('button', { name: '设置', exact: true });
    if ((await settings.getAttribute('aria-expanded')) !== 'true') await click308(page, settings);
    const section = drawer.getByRole('button', { name: category, exact: true });
    if ((await section.getAttribute('aria-expanded')) !== 'true') await click308(page, section);
    await click308(page, drawer.getByRole('button', { name: label, exact: true }));
  } else {
    await click308(page, page.getByRole('button', { name: '切换应用', exact: true }));
    await click308(page, page.getByRole('menuitem', { name: '设置', exact: true }));
    await click308(page, page.getByRole('button', { name: category, exact: true }));
    await click308(page, page.getByRole('menuitem', { name: label, exact: true }));
  }
  await expect(page).toHaveURL(new RegExp(path + '$'));
  await expect(page.getByRole('heading', { level: 1, name: heading, exact: true })).toBeVisible();
  if (path.includes('/staffing/'))
    await expect(page.locator('.standard-page[aria-busy]')).toHaveAttribute('aria-busy', 'false');
}
export async function menu308(page: Page, kind: Kind308) {
  await namedMenu308(page, labels308[kind], '职务配置', web308[kind]);
}
export async function reference308(page: Page, role: string) {
  await namedMenu308(page, '业务单元', '组织配置', '/settings/organization/business-units');
  await expect(page.getByRole('heading', { level: 1 })).toHaveCount(1);
  await reach308(page, page.getByRole('textbox', { name: '搜索编码或名称', exact: true }));
  if (role === 'admin') {
    await click308(page, page.getByRole('link', { name: '新建', exact: true }));
    const name = page.getByRole('textbox', { name: '中文名称', exact: true });
    const measured = await name.evaluate((el) => ({
      label: getComputedStyle(document.querySelector('.object-form .object-field-label')!).fontSize,
      value: getComputedStyle(el).fontSize,
      height: el.closest('.MuiInputBase-root')!.getBoundingClientRect().height,
    }));
    expect(measured).toEqual({ label: '13px', value: '14px', height: 32 });
    await test
      .info()
      .attach('reference-form', { body: JSON.stringify(measured), contentType: 'application/json' });
    await click308(page, page.getByRole('link', { name: '取消', exact: true }));
    await expect(page.locator('.object-form')).toHaveCount(0);
  }
}
export async function action308(page: Page, action: string) {
  if (['新增记录', '插入记录'].includes(action)) {
    await click308(page, page.getByRole('button', { name: /^(新增|插入)记录$/ }));
    return;
  }
  await click308(page, page.getByRole('button', { name: '更多操作', exact: true }));
  const item = page.getByRole('menuitem', { name: action, exact: true });
  await reach308(page, item);
  await item.hover();
  await contrast308(page);
  await click308(page, item);
}
export async function date308(page: Page, date: string) {
  await expect(page.getByRole('navigation', { name: '生效日期', exact: true })).toBeVisible();
  const direct = page.getByRole('button', { name: date, exact: true });
  const compact = page.getByRole('button', { name: /^选择生效日期/ });
  // Saving removes the editor before the refreshed timeline is rendered. Wait
  // for the actual responsive date control before choosing its interaction.
  await expect.poll(async () => (await direct.isVisible()) || (await compact.isVisible())).toBe(true);
  if (await direct.isVisible()) await click308(page, direct);
  else {
    await click308(page, compact);
    await click308(page, page.getByRole('menuitem', { name: date, exact: true }));
  }
  // The closing popup still contains its former selected menu item during the
  // MUI exit transition. Assert the refreshed date navigation, not that popup.
  await expect(page.getByRole('navigation', { name: '生效日期', exact: true }).locator('[aria-current="date"]')).toContainText(date);
}
export async function chooseClassification308(page: Page, field: 'jobFamilyId' | 'jobFunctionId', name: string) {
  await click308(page, page.locator('#staffing-' + field));
  const dialog = page.getByRole('dialog', { name: field === 'jobFamilyId' ? '选择职务系列' : '选择职类', exact: true });
  await dialog.getByRole('textbox').fill(name);
  await click308(page, dialog.getByRole('button', { name, exact: true }));
}
export async function family308(page: Page, name: string) {
  await chooseClassification308(page, 'jobFamilyId', name);
}
const jobFunctions308 = new WeakMap<Page, Promise<{ id: string; name: string }>>();
export function jobFunction308(page: Page) {
  let result = jobFunctions308.get(page);
  if (!result) {
    result = seed308(page, 'jobfunction', undefined, { date: '1900-01-01' });
    jobFunctions308.set(page, result);
  }
  return result;
}
export async function seed308(
  page: Page,
  kind: Kind308,
  family?: string,
  options: { code?: string; name?: string; date?: string; gradeIds?: string[]; gradeNames?: string[] } = {},
): Promise<{ id: string; code: string; name: string; kind: Kind308; jobFunction?: { id: string; name: string }; gradeIds?: string[]; gradeNames?: string[] }> {
  const code = options.code ?? 'P308-' + randomUUID().slice(0, 8),
    name = options.name ?? code;
  let gradeIds = options.gradeIds, gradeNames = options.gradeNames;
  if (kind === 'gradeladder' && !gradeIds) { const grade = await seed308(page, 'grade', undefined, { date: '1900-01-01' }); gradeIds = [grade.id]; gradeNames = [grade.name]; }
  const jobFunction = kind === 'job' ? await jobFunction308(page) : undefined;
  const result = await request(
    page,
    'POST',
    base308[kind],
    {
      code,
      name,
      nameEn: 'English ' + code,
      effectiveDate: options.date ?? '2026-01-01',
      operationType: 'CREATE',
      ...(gradeIds ? { gradeIds } : {}),
      operationReason: kind === 'jobfunction' ? 'NEW_JOB_FUNCTION' : kind === 'gradeladder' ? 'NEW_GRADE_LADDER' : kind === 'job' ? 'NEW_JOB' : kind === 'grade' ? 'NEW_GRADE' : 'NEW_JOB_FAMILY',
      ...(kind === 'job'
        ? { jobFamilyId: family, jobFunctionId: jobFunction!.id, responsibilities: '职责第一行\n职责第二行', requirements: '任职要求' }
        : { description: '说明第一行\n说明第二行' }),
    },
    randomUUID(),
  );
  expect(result.status).toBe(201);
  return { id: result.body.objectId as string, code, name, kind, jobFunction, gradeIds, gradeNames };
}
export async function enter308(page: Page, o: Awaited<ReturnType<typeof seed308>>) {
  await menu308(page, o.kind);
  await page.getByRole('textbox', { name: '搜索编码或名称', exact: true }).fill(o.code);
  await click308(page, page.getByRole('button', { name: '查询', exact: true }));
  await click308(page, page.getByRole('link', { name: o.name, exact: true }).first());
  await expect(page.getByRole('heading', { name: o.name, level: 1, exact: true })).toBeVisible();
}

/** WCAG text contrast against the rendered ancestor layers, including alpha. */
export async function contrast308(page: Page) {
  const result = await page.evaluate(() => {
    const color = (value: string) => {
      const parts = value.match(/[\d.]+/g)?.map(Number) ?? [];
      return { r: parts[0] ?? 0, g: parts[1] ?? 0, b: parts[2] ?? 0, a: parts[3] ?? 1 };
    };
    const over = (fg: ReturnType<typeof color>, bg: ReturnType<typeof color>) => ({
      r: fg.r * fg.a + bg.r * (1 - fg.a),
      g: fg.g * fg.a + bg.g * (1 - fg.a),
      b: fg.b * fg.a + bg.b * (1 - fg.a),
      a: 1,
    });
    const light = (v: ReturnType<typeof color>) =>
      [v.r, v.g, v.b].reduce((sum, channel, index) => {
        const s = channel / 255;
        return (
          sum + (s <= 0.04045 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4) * [0.2126, 0.7152, 0.0722][index]
        );
      }, 0);
    const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
    const samples: { text: string; ratio: number }[] = [];
    while (walker.nextNode()) {
      const node = walker.currentNode,
        el = node.parentElement;
      if (
        !node.textContent?.trim() ||
        !el ||
        el.closest('[aria-hidden="true"], [hidden], .Mui-disabled, script, style')
      )
        continue;
      const range = document.createRange();
      range.selectNodeContents(node);
      if (
        ![...range.getClientRects()].some(
          (r) => r.width > 0 && r.height > 0 && r.bottom > 0 && r.top < innerHeight,
        )
      )
        continue;
      const style = getComputedStyle(el);
      if (style.visibility !== 'visible' || Number(style.opacity) === 0) continue;
      const layers: Element[] = [];
      for (let parent: Element | null = el; parent; parent = parent.parentElement) layers.unshift(parent);
      let bg = { r: 255, g: 255, b: 255, a: 1 };
      for (const layer of layers) {
        bg = over(color(getComputedStyle(layer).backgroundColor), bg);
        // Effective-date arrows paint the text background in a full-size
        // ::before layer. Include it, not the transparent button backing.
        for (const pseudo of ['::before', '::after']) {
          const style = getComputedStyle(layer, pseudo);
          if (
            style.content !== 'none' &&
            style.position === 'absolute' &&
            style.top === '0px' &&
            style.bottom === '0px'
          )
            bg = over(color(style.backgroundColor), bg);
        }
      }
      const a = light(over(color(style.color), bg)),
        b = light(bg);
      samples.push({
        text: node.textContent.trim().slice(0, 40),
        ratio: (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05),
      });
    }
    return {
      count: samples.length,
      minimum: Math.min(...samples.map((v) => v.ratio)),
      failures: samples.filter((v) => v.ratio < 4.5),
    };
  });
  expect(result.count).toBeGreaterThan(0);
  expect(result.failures).toEqual([]);
  return result;
}

export async function matrixSeed308(page: Page, kind: Kind308, family: string) {
  const o = await seed308(page, kind, family);
  for (let i = 1; i <= 20; i++)
    await seed308(page, kind, family, { code: o.code + '-' + String(i).padStart(2, '0') });
  return o;
}
