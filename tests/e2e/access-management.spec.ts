import { execFileSync, spawn } from 'node:child_process';
import { existsSync } from 'node:fs';
import { randomUUID } from 'node:crypto';
import { chromium } from '@playwright/test';
import { test, expect, type Page } from './fixtures';
import { namedMenu308 } from './staffing308-support';
import { localConnection } from '../../scripts/local-runtime.mjs';
import { calibrateDesktopWindow } from './native-zoom-support';
import { apiBU, request } from './dated-object-support';
import { open as openBU } from './bu-support';

const loginName = 't04-security',
  devPassword = 'admin1234567',
  directory = '/settings/security/';
function requireTemporaryProfile() {
  const conn = localConnection();
  const profiles = JSON.parse(
    execFileSync('python3', ['scripts/local-bu.py', 'list'], {
      encoding: 'utf8',
    }),
  );
  if (
    !profiles.some(
      (p: { project: string; temporary: boolean }) =>
        p.project === conn.project && p.temporary === true,
    )
  )
    throw new Error('管理走查需要自有临时实例');
}
function bootstrap() {
  requireTemporaryProfile();
  execFileSync('python3', ['scripts/local-bu.py', 'access-admin'], {
    input: JSON.stringify({
      operation: 'bootstrap',
      operator: '浏览器验收运维',
      reason: '自有隔离环境首次引导',
      requestId: '02700000-0000-4000-8000-000000000004',
      login: loginName,
      password: devPassword,
      maxProxyDays: 7,
    }),
    stdio: ['pipe', 'pipe', 'pipe'],
  });
}
async function login(page: Page, user: string) {
  await page.goto('/');
  await page.getByLabel('登录名').fill(user);
  await page.getByLabel('密码').fill(devPassword);
  await page.getByRole('button', { name: '登录', exact: true }).click();
  await expect(
    page.getByRole('button', { name: '退出', exact: true }),
  ).toBeVisible();
}
const click = (p: Page, n: string) =>
  p.getByRole('button', { name: n, exact: true }).click();
async function more(p: Page, name: string) {
  await click(p, '更多操作');
  await p.getByRole('menuitem', { name, exact: true }).click();
}
const save = (p: Page) =>
  p.locator('button[type=submit][form=security-command-form]').click();
async function menu(p: Page, name: string, part: string) {
  await namedMenu308(p, name, '安全管理', directory + part);
}
async function journey(p: Page, adopt = false) {
  p.setDefaultTimeout(15000);
  const tag = randomUUID().slice(0, 8),
    user = 'clerk-' + tag,
    role = '部门名单 ' + tag;
  await menu(p, '账号', 'accounts');
  await click(p, '开户');
  await p
    .locator('.object-form')
    .getByLabel('登录名')
    .fill(user + 'cancel');
  await p.locator('.object-form').getByLabel('登录名').press('Tab');
  await expect(p.getByLabel('显示名', { exact: true })).toBeFocused();
  await click(p, '取消');
  await click(p, '放弃修改');
  await expect(p).toHaveURL(/security\/accounts\??$/);
  await click(p, '开户');
  await p.locator('.object-form').getByLabel('登录名').fill(user);
  await p.getByLabel('管理归属').selectOption({ label: '实例直管' });
  await p.getByLabel('初始凭据').fill(devPassword);
  // Drop the acknowledged HTTP response. The original request must resolve to the same account exactly once.
  await p.route(
    '**/api/v1/access/commands',
    async (route) => {
      await route.fetch();
      await route.abort('failed');
    },
    { times: 1 },
  );
  await p.getByLabel('初始凭据').press('Enter');
  await expect(
    p.getByRole('heading', { name: user, exact: true }),
  ).toBeVisible();
  const context = await p
    .context()
    .browser()!
    .newContext({ baseURL: localConnection().webOrigin });
  try {
    const clerk = await context.newPage();
    await login(clerk, user);
    expect((await clerk.request.get('/api/v1/departments')).status()).toBe(403);
    await menu(p, '角色', 'roles');
    await click(p, '新建角色');
    await expect(p.locator('.object-form')).not.toContainText('发布');
    const nameField = p.getByLabel('角色名称');
    await nameField.fill(role);
    await nameField.press('Tab');
    await expect(p.locator('.object-form select').first()).toBeFocused();
    await p
      .getByRole('checkbox', { name: '部门 · 查看部门', exact: true })
      .check();
    await p
      .getByLabel('范围策略', { exact: false })
      .first()
      .selectOption('new');
    await p
      .getByLabel('业务目标')
      .selectOption({ label: '企业 · LOCAL · 本地学习企业' });
    await p
      .getByLabel('字段策略', { exact: false })
      .first()
      .selectOption('new');
    const fields = p.getByRole('group', { name: '可读字段', exact: true });
    await fields.getByRole('checkbox', { name: '编码', exact: true }).check();
    await fields.getByRole('checkbox', { name: '名称', exact: true }).check();
    await save(p);
    await expect(
      p.getByRole('heading', { name: role, exact: true }),
    ).toBeVisible();
    await menu(p, '授权', 'grants');
    await p.getByLabel('搜索登录名或显示名').fill(user);
    await more(p, '列筛选…');
    await p.getByRole('dialog').getByRole('button', { name: '取消' }).click();
    await expect(p.getByLabel('搜索登录名或显示名')).toHaveValue(user);
    await more(p, '列筛选…');
    await p.getByRole('dialog').getByRole('combobox', { name: '值' }).click();
    await p.getByRole('option', { name: '启用', exact: true }).click();
    await p
      .getByRole('dialog')
      .getByRole('button', { name: '应用' })
      .press('Enter');
    await expect(p).toHaveURL(new RegExp('q=' + user));
    await p.getByRole('link', { name: user, exact: true }).click();
    async function grant(proxy: boolean) {
      await click(p, '授予权限');
      await p.getByLabel('角色版本').selectOption({ label: role + ' · V1' });
      if (proxy) {
        await p.getByLabel('授予来源').selectOption('proxy');
        const time = new Date(Date.now() + 9 * 3600e3)
          .toISOString()
          .slice(0, 16);
        await p.getByLabel('代理到期').fill(time);
      }
      await save(p);
      await expect(p.locator('.object-form')).toHaveCount(0);
    }
    await grant(false);
    await grant(true);
    const grants = p.locator('.access-grant');
    await expect(grants).toHaveCount(2);
    // State is returned by Go at read time. Query again when a recorded start is still ahead of that read.
    if (await grants.filter({ hasText: '待生效' }).count()) {
      await expect
        .poll(
          async () => {
            const response = p.waitForResponse(
              (r) =>
                r.request().method() === 'GET' &&
                r.url().includes('/api/v1/access/grants?subject=') &&
                r.ok(),
            );
            await click(p, '查询');
            await response;
            await expect(grants).toHaveCount(2);
            return grants.filter({ hasText: '生效中' }).count();
          },
          { timeout: 12000 },
        )
        .toBe(2);
    }
    await expect(grants.filter({ hasText: '生效中' })).toHaveCount(2);
    expect((await clerk.request.get('/api/v1/departments')).status()).toBe(200);
    await expect(
      grants.getByRole('button', { name: '采用新版本' }),
    ).toHaveCount(0);
    if (adopt) {
      await menu(p, '角色', 'roles');
      await p.getByLabel('搜索版本名称').fill(role);
      await click(p, '查询');
      await p.getByRole('link', { name: role + ' · V1', exact: true }).click();
      await p.getByRole('tab', { name: '版本引用' }).click();
      const references = p.getByRole('link', { name: user, exact: true });
      await expect(references).toHaveCount(2);
      const referenceId = await references.first().getAttribute('id');
      await references.first().click();
      await expect(p.getByRole('heading', { name: '授予明细' })).toBeVisible();
      await p.getByRole('link', { name: '返回版本引用' }).click();
      await expect(p.locator('#' + referenceId)).toBeFocused();
      await click(p, '新增版本');
      await p
        .getByRole('checkbox', { name: '部门 · 查看部门历史', exact: true })
        .check();
      await save(p);
      await expect(p.locator('.object-form')).toHaveCount(0);
      await menu(p, '授权', 'grants');
      await p.getByLabel('搜索登录名或显示名').fill(user);
      await click(p, '查询');
      await p.getByRole('link', { name: user, exact: true }).click();
      await expect(grants.filter({ hasText: role + ' · V1' })).toHaveCount(2);
      await grants
        .filter({ hasText: '本职' })
        .getByRole('button', { name: '采用新版本' })
        .click();
      await p.getByLabel('角色版本').selectOption({ label: role + ' · V2' });
      await save(p);
      await expect(grants.filter({ hasText: '本职' })).toContainText(
        role + ' · V2',
      );
      await expect(grants.filter({ hasText: '限期代理' })).toContainText(
        role + ' · V1',
      );
    }
    await grants
      .filter({ hasText: '限期代理' })
      .getByRole('button', { name: '撤销此份授予' })
      .click();
    const confirm = p.locator('#security-command-form');
    await expect(confirm).toContainText('限期代理');
    await expect(confirm).toContainText('授予编号');
    await expect(confirm).toContainText(role);
    await click(p, '取消');
    const revokeProxy = grants
      .filter({ hasText: '限期代理' })
      .getByRole('button', { name: '撤销此份授予' });
    await expect(revokeProxy).toBeFocused();
    await revokeProxy.press('Enter');
    await save(p);
    await expect(grants.filter({ hasText: '限期代理' })).toContainText(
      '已撤销',
    );
    await expect(grants.filter({ hasText: '本职' })).toContainText('生效中');
    expect((await clerk.request.get('/api/v1/departments')).status()).toBe(200);
    await grants
      .filter({ hasText: '本职' })
      .getByRole('button', { name: '撤销此份授予' })
      .click();
    await save(p);
    await expect(grants.filter({ hasText: '已撤销' })).toHaveCount(2);
    expect((await clerk.request.get('/api/v1/departments')).status()).toBe(403);
    await p.getByRole('tab', { name: '变更日志' }).click();
    if (adopt) {
      await click(p, '操作者列筛选');
      await p
        .getByRole('dialog')
        .getByLabel('值', { exact: true })
        .fill('missing-user');
      await p.getByRole('dialog').getByRole('button', { name: '取消' }).click();
      await click(p, '记录处理列筛选');
      await p.getByRole('dialog').getByRole('combobox', { name: '值' }).click();
      await p.getByRole('option', { name: '撤销授予' }).click();
      await p.getByRole('dialog').getByRole('button', { name: '应用' }).click();
      await expect(p.locator('.change-operation')).toHaveCount(2);
      await p.reload();
      await expect(p.locator('.change-operation')).toHaveCount(2);
    }
    const log = p.getByRole('table', { name: '变更日志', exact: true });
    await expect(log.locator('th').filter({ hasText: '原值' })).toBeAttached();
    const event = log.locator('tbody').first();
    await expect(event.locator('[data-label="变更内容"]')).toContainText(
      '本职',
    );
    await expect(event.locator('[data-label="原值"]')).toContainText(role);
    await expect(event.locator('[data-label="新值"]')).toHaveText('已撤销');
    await event.getByRole('button', { name: '来源明细' }).click();
    await expect(event.locator('.change-source-detail')).toContainText(
      '提交编号',
    );
    await event.getByRole('button', { name: '查看这份授予' }).click();
    await expect(p.getByRole('heading', { name: '授予明细' })).toBeVisible();
    await click(p, '返回变更日志');
    await expect(
      p.getByRole('button', { name: '查看这份授予' }).first(),
    ).toBeFocused();
    if (adopt) await expect(p.locator('.change-operation')).toHaveCount(2);
    await p.getByRole('tab', { name: '业务授予' }).click();
    await p.reload();
    await expect(grants.filter({ hasText: '已撤销' })).toHaveCount(2);
    await expect
      .poll(() =>
        p.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth + 1,
        ),
      )
      .toBe(true);
  } finally {
    await context.close();
  }
}
test('028 内置 admin 查看长期管理职责并从页面办理授权', async ({ page }) => {
  test.setTimeout(120000);
  requireTemporaryProfile();
  await page.setViewportSize({ width: 1440, height: 540 });
  await login(page, 'admin');
  await menu(page, '授权', 'grants');
  await page.getByLabel('搜索登录名或显示名').fill('admin');
  await click(page, '查询');
  await page.getByRole('link', { name: /^admin(?: ·|$)/ }).click();
  await expect(
    page.getByText('全部安全管理权限 · 长期', { exact: true }),
  ).toBeInViewport();
  await expect(page.getByRole('button', { name: '撤销管理职责' })).toHaveCount(0);
  await journey(page, true);
});

for (const height of [900, 540])
  test(
    '027 T04 管理员开户、发布和逐份撤销 · 1440×' + height,
    async ({ page }) => {
      test.setTimeout(120000);
      bootstrap();
      await page.setViewportSize({ width: 1440, height });
      await login(page, loginName);
      await journey(page, height === 900);
    },
  );
test('027 T04 管理员关键流程原生 200%', async () => {
  test.setTimeout(150000);
  bootstrap();
  let n = 3900 + (process.pid % 1000);
  while (
    existsSync('/tmp/.X11-unix/X' + n) ||
    existsSync('/tmp/.X' + n + '-lock')
  )
    n++;
  const env = { ...process.env, DISPLAY: ':' + n };
  const x = spawn(
    'Xvfb',
    [
      env.DISPLAY,
      '-screen',
      '0',
      '1920x1200x24',
      '-nolisten',
      'tcp',
      '-nolisten',
      'unix',
    ],
    { env, detached: true, stdio: 'ignore' },
  );
  let b;
  try {
    await expect
      .poll(() => {
        try {
          execFileSync('xdpyinfo', [], { env, stdio: 'ignore', timeout: 1000 });
          return true;
        } catch {
          return false;
        }
      })
      .toBe(true);
    b = await chromium.launch({
      headless: false,
      env,
      args: [
        '--window-size=1280,1039',
        '--window-position=0,0',
        '--ozone-platform=x11',
      ],
    });
    const c = await b.newContext({
      baseURL: localConnection().webOrigin,
      viewport: null,
      deviceScaleFactor: undefined,
    });
    const p = await c.newPage();
    await login(p, loginName);
    const zoom = (steps: number) =>
      execFileSync(
        'python3',
        [
          'docs/verification/data/100-ui-baseline/native-zoom.py',
          String(steps),
        ],
        { env, stdio: 'pipe' },
      );
    zoom(0);
    await calibrateDesktopWindow(p);
    zoom(5);
    await expect
      .poll(() => p.evaluate(() => [innerWidth, innerHeight, devicePixelRatio]))
      .toEqual([640, 450, 2]);
    await journey(p);
  } finally {
    await b?.close();
    if (x.pid && x.exitCode === null) process.kill(-x.pid, 'SIGTERM');
  }
});

test('027 T04 委派管理员仅见获管归属，撤去职责后旧表单和直接请求均拒绝', async ({
  page,
  browser,
}) => {
  test.setTimeout(120000);
  bootstrap();
  const tag = randomUUID().slice(0, 8);
  await openBU(page);
  const a = await apiBU(page, '管理甲' + tag);
  const b = await apiBU(page, '管理乙' + tag);
  const referenceTop = new Map<number, number>();
  for (const h of [900, 540]) {
    await page.setViewportSize({ width: 1440, height: h });
    const bar = page.locator('.list-control-main');
    await expect(bar).toBeInViewport();
    referenceTop.set(h, (await bar.boundingBox())!.y);
  }
  const rootContext = await browser.newContext({
    baseURL: localConnection().webOrigin,
  });
  const limitedContext = await browser.newContext({
    baseURL: localConnection().webOrigin,
  });
  try {
    const root = await rootContext.newPage();
    await login(root, loginName);
    const instanceId = (await request(root, 'GET', '/api/v1/session')).body
      .instanceId;
    const command = (p: Page, cmd: object) =>
      request(p, 'POST', '/api/v1/access/commands', {
        ...cmd,
        requestId: randomUUID(),
        instanceId,
      });
    const rootCommand = async (cmd: object) => {
      const response = await command(root, cmd);
      expect(response.status).toBe(200);
      return response.body.target as string;
    };
    const loginId = 'limited-' + tag;
    const subject = await rootCommand({
      operation: 'account.create',
      login: loginId,
      owner: a.id,
      password: devPassword,
    });
    const role = await rootCommand({
      operation: 'version.publish',
      version: {
        kind: 'role',
        name: '甲办事上限' + tag,
        actions: ['department.read'],
      },
      newScope: {
        kind: 'scope',
        name: '甲范围' + tag,
        targetKind: 'bu',
        targetId: a.id,
      },
      newFields: { kind: 'fields', name: '编码' + tag, readFields: ['code'] },
    });
    const def = (
      await request(root, 'GET', '/api/v1/access/roles?object=' + role)
    ).body.versions.find((d: { id: string }) => d.id === role);
    const mandate = await rootCommand({
      operation: 'mandate.issue',
      subject,
      limit: {
        owner: a.id,
        rights: ['account'],
        role,
        scope: def.scope,
        fields: def.fields,
        maxProxyDays: 1,
      },
    });
    const limited = await limitedContext.newPage();
    await login(limited, loginId);
    await menu(limited, '账号', 'accounts');
    for (const h of [900, 540]) {
      await limited.setViewportSize({ width: 1440, height: h });
      const bar = limited.locator('.list-control-main');
      await expect(bar).toBeInViewport();
      expect(
        Math.abs((await bar.boundingBox())!.y - referenceTop.get(h)!),
      ).toBeLessThanOrEqual(2);
    }
    expect((await request(limited, 'GET', '/api/v1/departments')).status).toBe(
      403,
    );
    expect((await request(limited, 'GET', '/api/v1/access/roles')).status).toBe(
      403,
    );
    await click(limited, '开户');
    const owner = limited.getByLabel('管理归属');
    await expect(owner.locator('option')).toHaveCount(2);
    await expect(owner.locator('option', { hasText: a.name })).toHaveCount(1);
    await expect(owner.locator('option', { hasText: b.name })).toHaveCount(0);
    const draft = {
      operation: 'account.create',
      login: 'pending-' + tag,
      owner: b.id,
      password: devPassword,
    };
    expect((await command(limited, draft)).status).toBe(403);
    await limited
      .locator('.object-form')
      .getByLabel('登录名')
      .fill(draft.login);
    await owner.selectOption(a.id);
    await limited.getByLabel('初始凭据').fill(devPassword);
    await rootCommand({ operation: 'mandate.revoke', subject, id: mandate });
    await save(limited);
    await expect(limited.locator('input[type=password]')).toHaveCount(0);
    await expect(limited.locator('.object-form')).toHaveCount(0);
    expect((await command(limited, { ...draft, owner: a.id })).status).toBe(
      403,
    );
    const accounts = (
      await request(root, 'GET', '/api/v1/access/accounts?q=' + draft.login)
    ).body;
    expect(accounts.total).toBe(0);
  } finally {
    await limitedContext.close();
    await rootContext.close();
  }
});

test('027 T04 从账号页启停及重置，两个旧会话均失效', async ({
  page,
  playwright,
}) => {
  test.setTimeout(120000);
  bootstrap();
  await login(page, loginName);
  const user = 'sessions-' + randomUUID().slice(0, 8);
  await menu(page, '账号', 'accounts');
  await click(page, '开户');
  await page.locator('.object-form').getByLabel('登录名').fill(user);
  await page.getByLabel('管理归属').selectOption({ label: '实例直管' });
  await page.getByLabel('初始凭据').fill(devPassword);
  await save(page);
  await expect(
    page.getByRole('heading', { name: user, exact: true }),
  ).toBeVisible();

  const origin = localConnection().webOrigin;
  const sessions = await Promise.all(
    [0, 1].map(() =>
      playwright.request.newContext({
        baseURL: origin,
        extraHTTPHeaders: { Origin: origin },
      }),
    ),
  );
  const signIn = async (client: (typeof sessions)[number], password: string) =>
    (
      await client.post('/api/v1/session/login', {
        data: { login: user, password },
      })
    ).status();
  const statuses = () =>
    Promise.all(
      sessions.map(async (c) => (await c.get('/api/v1/session')).status()),
    );
  try {
    for (const client of sessions)
      expect(await signIn(client, devPassword)).toBe(200);
    expect(await statuses()).toEqual([200, 200]);
    await more(page, '停用账号');
    await save(page);
    await expect(
      page.locator('.object-identity-summary .MuiChip-label'),
    ).toHaveText('停用');
    expect(await statuses()).toEqual([401, 401]);
    expect(await signIn(sessions[0], devPassword)).toBe(401);

    await more(page, '启用账号');
    await save(page);
    await expect(
      page.locator('.object-identity-summary .MuiChip-label'),
    ).toHaveText('启用');
    expect(await statuses()).toEqual([401, 401]);
    for (const client of sessions)
      expect(await signIn(client, devPassword)).toBe(200);
    // Reopen after the two replacement logins, which can advance the account revision.
    await page.reload();
    const changed = randomUUID();
    await click(page, '重置凭据');
    await page.getByLabel('新凭据').fill(changed);
    await save(page);
    await page.getByRole('tab', { name: '变更日志' }).click();
    await expect(page.locator('.change-operation').first()).toContainText(
      '重置凭据',
    );
    expect(await statuses()).toEqual([401, 401]);
    expect(await signIn(sessions[0], devPassword)).toBe(401);
    expect(await signIn(sessions[1], changed)).toBe(200);
    expect((await sessions[1].get('/api/v1/departments')).status()).toBe(403);
    await expect(
      page.getByRole('table', { name: '变更日志', exact: true }),
    ).not.toContainText(changed);
  } finally {
    await Promise.all(sessions.map((c) => c.dispose()));
  }
});

test('027 T04 从角色页单独发布范围、字段策略并停用指定版本', async ({
  page,
}) => {
  test.setTimeout(120000);
  bootstrap();
  await login(page, loginName);
  const tag = randomUUID().slice(0, 8);
  for (const kind of ['范围策略', '字段策略']) {
    await menu(page, '角色', 'roles');
    await more(page, '新建' + kind);
    const name = kind + tag;
    await page.getByLabel(kind + '名称').fill(name);
    if (kind === '范围策略') {
      await page
        .getByLabel('业务目标')
        .selectOption({ label: '企业 · LOCAL · 本地学习企业' });
    } else {
      await page
        .getByRole('group', { name: '可读字段' })
        .getByRole('checkbox', { name: '名称', exact: true })
        .check();
    }
    await save(page);
    await expect(
      page.getByRole('heading', { name, exact: true }),
    ).toBeVisible();
    await click(page, '新增版本');
    await save(page);
    await expect(page.locator('.object-form')).toHaveCount(0);
    await menu(page, '角色', 'roles');
    await more(page, '列筛选…');
    await page
      .getByRole('dialog')
      .getByRole('combobox', { name: '值' })
      .click();
    await page.getByRole('option', { name: kind, exact: true }).click();
    await page
      .getByRole('dialog')
      .getByRole('button', { name: '应用' })
      .click();
    await expect(
      page.getByRole('button', { name: '新建' + kind, exact: true }),
    ).toBeVisible();
    await page.getByLabel('搜索版本名称').fill(name);
    await click(page, '查询');
    await page.getByRole('link', { name: name + ' · V1', exact: true }).click();
    await page.getByRole('button', { name: '上一个对象' }).click();
    await expect(page.locator('.record-control-name')).toContainText('V2');
    await page.getByRole('button', { name: '下一个对象' }).click();
    await expect(page.locator('.record-control-name')).toContainText('V1');
    await more(page, '停用版本');
    await save(page);
    await expect(page.getByText('已停用', { exact: true })).toBeVisible();
    await page.getByRole('tab', { name: '变更日志' }).click();
    await expect(page.locator('.change-operation').first()).toContainText(
      '停用版本',
    );
    const changes = page.getByRole('table', { name: '变更日志', exact: true });
    await expect(
      changes.locator('tbody').first().locator('[data-label="原值"]'),
    ).toHaveText('可用');
    await expect(
      changes.locator('tbody').first().locator('[data-label="新值"]'),
    ).toHaveText('已停用');
    await expect(
      changes
        .locator('tbody')
        .filter({ hasText: '新建' + kind })
        .locator('[data-label="新值"]'),
    ).toHaveText(name + ' · V1');
    await menu(page, '角色', 'roles');
    await page.getByLabel('搜索版本名称').fill(name);
    await click(page, '查询');
    await page.getByRole('link', { name: name + ' · V2', exact: true }).click();
    await expect(page.getByText('可用', { exact: true })).toBeVisible();
  }
});

test('029 从账号资料检索存量授予并恢复返回位置', async ({ page }) => {
  test.setTimeout(100000);
  bootstrap();
  await login(page, loginName);
  await page.setViewportSize({ width: 1440, height: 540 });
  await menu(page, '账号', 'accounts');
  await page.getByLabel('搜索登录名或显示名').fill('admin');
  await click(page, '查询');
  await page.getByRole('link', { name: /^admin(?: ·|$)/ }).click();
  await page.getByRole('link', { name: '查看账号授权' }).click();
  const table = page.locator('.grant-table tbody tr');
  await expect(table).toHaveCount(20);
  const fresh = page.waitForResponse(
    (r) =>
      r.request().method() === 'GET' &&
      r.url().includes('/api/v1/access/grants?subject=') &&
      r.ok(),
  );
  await click(page, '查询');
  await fresh;
  await expect(table).toHaveCount(20);
  await click(page, '下一页');
  await expect(page.locator('.grant-pager')).toContainText('21–40');
  await page.locator('.grant-pager').getByRole('combobox').click();
  await page.getByRole('option', { name: '50', exact: true }).click();
  await expect(table).toHaveCount(50);
  await page.getByLabel('搜索角色或业务范围').fill('岗位');
  await click(page, '筛选授予');
  await page
    .getByRole('dialog')
    .getByLabel('授予来源')
    .selectOption('development');
  await page.getByRole('dialog').getByRole('button', { name: '应用' }).click();
  await expect(table.first()).toContainText('岗位');
  await table.first().getByRole('link').click();
  await expect(page.getByRole('heading', { name: '授予明细' })).toBeVisible();
  await page.reload();
  await expect(page.getByRole('heading', { name: '授予明细' })).toBeVisible();
  await click(page, '返回业务授予');
  await expect(page.getByLabel('搜索角色或业务范围')).toHaveValue('岗位');
  await page.getByRole('link', { name: '返回账号资料' }).click();
  await expect(page.getByRole('link', { name: '查看账号授权' })).toBeFocused();
  await page.getByRole('link', { name: '返回列表' }).click();
  await expect(page.getByLabel('搜索登录名或显示名')).toHaveValue('admin');
});
