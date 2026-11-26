import { lazyCredentials } from './credentials.mjs';
import { deferAutomaticConfirmation } from './fixtures';
import { applyListOption,recordRange, readCurrentDate,  test, expect, expectCurrentDate, type Page } from "./fixtures";
import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import path from "node:path";
import { randomUUID } from "node:crypto";

const root = path.resolve(import.meta.dirname, "../..");
const credentials = lazyCredentials() as {
  login: string;
  password: string;
  reader_login: string;
  reader_password: string;
};
const buPath = "/settings/organization/business-units";
let lastLoginAt = 0;
async function submitLogin(page: Page) {
  // One worker shares the real account's 20/minute limit. Pace test traffic;
  // do not relax the product limit or retry rejected logins automatically.
  const delay = 3100 - (Date.now() - lastLoginAt);
  if (delay > 0) await new Promise(resolve => setTimeout(resolve, delay));
  lastLoginAt = Date.now();
  await page.getByRole("button", { name: "登录", exact: true }).click();
}
async function openBusinessUnits(page: Page) {
  await expect(page.locator('[data-navigation-mode]')).toBeVisible();
  await page.evaluate(() => document.fonts.ready);
  if (await page.getByRole('button', { name: '打开导航', exact: true }).isVisible()) {
    await page.getByRole('button', { name: '打开导航', exact: true }).click();
    await page.locator('#navigation-drawer').getByRole('button', { name: '业务单元', exact: true }).click();
  } else {
    await page.getByRole('button', { name: '切换应用', exact: true }).click();
    await page.getByRole('menuitem', { name: '设置', exact: true }).click();
    await page.getByRole('button', { name: '组织配置', exact: true }).click();
    await page.getByRole('menuitem', { name: '业务单元', exact: true }).click();
  }
  await expect(page).toHaveURL(new RegExp(`${buPath}$`));
}
async function login(page: Page, readonly = false) {
  await page.goto("/");
  await page
    .getByLabel("登录名")
    .fill(readonly ? credentials.reader_login : credentials.login);
  await page
    .getByLabel("密码")
    .fill(readonly ? credentials.reader_password : credentials.password);
  await submitLogin(page);
  await openBusinessUnits(page);
  await expect(
    page.getByRole("heading", { name: "业务单元", exact: true }),
  ).toBeVisible();
  expect(await readCurrentDate(page)).toMatch(/^\d{4}-\d{2}-\d{2}$/);
}
async function fillCreate(page: Page, code: string, date?: string) {
  await expect(
    page.getByRole("heading", { name: "新建业务单元", exact: true }),
  ).toBeVisible();
  await page.getByRole("textbox", { name: "编码", exact: true }).fill(code);
  await page.getByLabel("中文名称").fill(code + " 中文单元");
  await page.getByLabel("英文名称").fill(code + " English");
  if (date) await page.getByLabel("生效日期").fill(date);
  await page.getByRole("combobox", { name: "操作原因" }).click();
  await page.getByRole("option", { name: "新设组织" }).click();
}

test("设置菜单创建未来BU、字段错误、当日事实、只读历史与返回上下文", async ({
  page,
}) => {
  // This journey performs 48 API requests and repeated document navigations.
  // The captured 30s failure reached the final list reload with no pending API
  // request (slowest response 275ms). Keep individual assertions unchanged and
  // budget the whole multi-navigation journey independently of request latency.
  test.setTimeout(60000);
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await login(page);
  const date = await readCurrentDate(page);
  const future = new Date(date + "T00:00:00Z");
  future.setUTCDate(future.getUTCDate() + 2);
  const code = "E2E_" + randomUUID().slice(0, 8);
  const before = new Date(Date.now() - 10_000).toISOString();
  await page.getByLabel("搜索编码或名称").fill(code);
  const queried = page.waitForResponse(r =>
    new URL(r.url()).pathname === "/api/v1/business-units" &&
    new URL(r.url()).searchParams.get("q") === code);
  await page.getByRole("button", { name: "查询", exact: true }).click();
  expect((await queried).status()).toBe(200);
  await expect(page).toHaveURL(url => url.searchParams.get("q") === code);
  await expect(page.getByText("没有符合条件的业务单元")).toBeVisible();
  await page.getByRole("link", { name: "新建", exact: true }).click();
  await fillCreate(page, "_invalid", future.toISOString().slice(0, 10));
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect(
    page.getByRole("textbox", { name: "编码", exact: true }),
  ).toBeFocused();
  await expect(
    page.getByText("编码须为", { exact: false }).first(),
  ).toBeVisible();
  await page.getByRole("textbox", { name: "编码", exact: true }).fill(code);
  await page.getByLabel("中文名称").fill(code + " 中文单元");
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect(
    page.getByText("已保存。生效日期：", { exact: false }),
  ).toBeVisible();
  await expectCurrentDate(page, date);
  await expect(
    page.getByText("所选记录尚未生效。", { exact: false }),
  ).toBeVisible();
  await expect(page.locator(".operation-summary")).toContainText("操作者：admin");
  await expect(page.getByRole("button", { name: "来源说明" })).toHaveCount(0);
  let logFailed = false;
  await page.route("**/api/v1/business-units/*/changes?*", async route => {
    if (!logFailed) {
      await route.fulfill({ status: 503, json: { code: "M02_DEPENDENCY_UNAVAILABLE", message: "日志临时不可读取" } });
    } else await route.continue();
  });
  await page.getByRole("tab", { name: "变更日志", exact: true }).click();
  await expect(page.getByText("日志临时不可读取")).toBeVisible();
  logFailed = true;
  await page.getByRole("tab", { name: "基本资料", exact: true }).click();
  await expect(page.getByRole("tab", { name: "基本资料", exact: true })).toHaveAttribute("aria-selected", "true");
  await page.getByRole("tab", { name: "变更日志", exact: true }).click();
  await expect(page.getByRole("tab", { name: "变更日志", exact: true })).toHaveAttribute("aria-selected", "true");
  await expect(
    page.getByRole("link", { name: "查看当次资料" }),
  ).toBeVisible();
  await expect(page.getByText("日志临时不可读取")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "高级历史" })).toHaveCount(0);
  await expect(page.getByText("提交时间：", { exact: false })).toHaveCount(0);
  await page.getByRole("link", { name: "查看当次资料" }).click();
  await expect(page.getByText("正在查看只读历史资料")).toBeVisible();
  await expect(page).toHaveURL(/sourceVersion=/);
  await page.reload();
  await expect(page.getByText("正在查看只读历史资料")).toBeVisible();
  const historical = new URL(page.url());
  const invalid = new URL(historical);
  invalid.searchParams.set("sourceVersion", "");
  const rejected = page.waitForResponse(r =>
    new URL(r.url()).pathname.startsWith("/api/v1/business-units/") &&
    new URL(r.url()).searchParams.get("sourceVersion") === "");
  await page.goto(invalid.toString());
  expect((await rejected).status()).toBe(400);
  await expect(page.getByRole("heading", { name: "基本资料" })).toHaveCount(0);
  await expect(page.getByText("正在查看只读历史资料")).toHaveCount(0);
  await page.getByRole("button", { name: "返回当前资料" }).click();
  await expect(page.getByText("所选记录尚未生效。", { exact: false })).toBeVisible();
  invalid.searchParams.append("sourceVersion", historical.searchParams.get("sourceVersion")!);
  await page.goto(invalid.toString());
  await expect(page.getByText("历史资料链接无效", { exact: false })).toBeVisible();
  await expect(page.getByRole("heading", { name: "基本资料" })).toHaveCount(0);
  invalid.searchParams.delete("sourceVersion");
  invalid.searchParams.append("selectedRecordId", randomUUID());
  invalid.searchParams.append("selectedRecordId", randomUUID());
  await page.goto(invalid.toString());
  await expect(page.getByText("历史资料链接无效", { exact: false })).toBeVisible();
  await page.getByRole("button", { name: "返回当前资料" }).click();
  await expect(page.getByText("所选记录尚未生效。", { exact: false })).toBeVisible();
  const old = new URL(historical);
  old.searchParams.set("knownAt", before);
  await page.goto(old.toString());
  await expect(page.getByText("无法打开此历史链接，请返回当前资料查看变更日志", { exact: false })).toBeVisible();
  await expect(page.getByRole("heading", { name: "基本资料" })).toHaveCount(0);
  await page.getByRole("button", { name: "返回当前资料" }).click();
  await expect(page.getByText("所选记录尚未生效。", { exact: false })).toBeVisible();
  await page.getByRole("link", { name: "返回列表" }).click();
  await expect(page.getByLabel("搜索编码或名称")).toHaveValue(code);
  await expect(page.locator('.list-return-selected')).toContainText(code);
  await expect(
    page.getByRole("link", { name: code + " 中文单元" }),
  ).toBeFocused();
  await expectCurrentDate(page, date);
  await page.reload();
  await expect(page.locator('.list-return-selected')).toContainText(code);
  expect(errors).toEqual([]);
});

test("响应丢失后刷新查回原结果，并阻止新标识重复提交", async ({ page }) => {
  await deferAutomaticConfirmation(page);
  await login(page);
  const code = "LOST_" + randomUUID().slice(0, 8);
  const date = await readCurrentDate(page);
  await page.getByLabel("搜索编码或名称").fill(code);
  await page.getByRole("button", { name: "查询", exact: true }).click();
  await applyListOption(page, '状态', '启用');
  await applyListOption(page, '排序', '名称降序');
  await page.getByRole("link", { name: "新建", exact: true }).click();
  const originalBack = new URL(new URL(page.url()).searchParams.get("back")!, page.url());
  await fillCreate(page, code);
  let committed = false;
  await page.route("**/api/v1/business-units", async (route) => {
    if (route.request().method() === "POST" && !committed) {
      const response = await route.fetch();
      expect(response.status()).toBe(201);
      committed = true;
      await route.abort("failed");
    } else await route.continue();
  });
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect(
    page.getByText(/暂时无法确认保存结果/, { exact: false }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "保存", exact: true }),
  ).toBeDisabled();
  const refs = await page.evaluate(() =>
    JSON.parse(localStorage.getItem("chnhcm-pending-submissions") ?? "[]"),
  );
  expect(refs).toHaveLength(1);
  expect(Object.keys(refs[0]).sort()).toEqual([
    "instanceRef",
    "subjectRef",
    "submissionId",
  ]);
  page.on("dialog", (dialog) => void dialog.accept());
  await page.reload();
  await expect(
    page.getByText(/暂时无法确认保存结果/),
  ).toBeVisible();
  await fillCreate(page, "DUPLICATE");
  await expect(page.getByRole("button", { name: "查看保存结果" })).toHaveCount(1);
  await expect(
    page.getByRole("button", { name: "保存", exact: true }),
  ).toBeDisabled();
  await page.getByRole("button", { name: "查看保存结果" }).click();
  await expect(
    page.getByText("已保存", { exact: true }),
  ).toBeVisible();
  const recovered = new URL((await page.getByRole("link", { name: "查看保存结果" }).getAttribute("href"))!, page.url());
  const recoveredBack = new URL(recovered.searchParams.get("back")!, page.url());
  for (const [key, value] of originalBack.searchParams)
    expect(recoveredBack.searchParams.get(key)).toBe(value);
  expect(recoveredBack.searchParams.get("selected")).toBe(recovered.pathname.split("/").at(-1));
  await page.getByRole("link", { name: "查看保存结果" }).click();
  await page.getByRole("button", { name: /^(放弃修改|离开此页)$/ }).click();
  await expect(page.getByText("正在查看只读历史资料")).toBeVisible();
  await page.getByRole("button", { name: "返回当前资料" }).click();
  await page.getByRole("link", { name: "返回列表" }).click();
  await expect(page.getByLabel("搜索编码或名称")).toHaveValue(code);
  await expect(page.getByRole("button", { name: "状态：启用", exact: true })).toBeVisible();
  await expect(page.getByRole("columnheader", { name: /名称/ })).toHaveAttribute("aria-sort", "descending");
  await expect(page.getByText(recordRange(page, 1, 1))).toBeVisible();
  await expect(page.locator('.list-return-selected')).toContainText(code);
  await expect(page.getByRole("link", { name: code + " 中文单元", exact: true })).toBeFocused();
  await expectCurrentDate(page, date);
});

test("未保存输入保护、正常取消、只读账号与短屏退出", async ({ page }) => {
  await login(page);
  await page.getByRole("link", { name: "新建", exact: true }).click();
  await page
    .getByRole("textbox", { name: "编码", exact: true })
    .fill("DISCARD");
  await page.getByRole("button", { name: "退出", exact: true }).click();
  await page.getByRole("button", { name: "继续编辑" }).click();
  await page.getByRole("link", { name: "取消", exact: true }).click();
  await page.getByRole("button", { name: "继续编辑" }).click();
  await expect(
    page.getByRole("textbox", { name: "编码", exact: true }),
  ).toHaveValue("DISCARD");
  await page.getByRole("link", { name: "取消", exact: true }).click();
  await page.getByRole("button", { name: "放弃修改" }).click();
  await page.getByRole("button", { name: "退出", exact: true }).click();
  await expect(page.getByRole("heading", { name: "登录" })).toBeVisible();
  await page.setViewportSize({ width: 1440, height: 540 });
  await login(page, true);
  await expect(
    page.getByRole("link", { name: "新建", exact: true }),
  ).toHaveAttribute("aria-disabled", "true");
  await page.getByRole("button", { name: "退出", exact: true }).click();
  await expect(page.getByRole("heading", { name: "登录" })).toBeVisible();
});

test("真实滚轮与列表滚动、焦点返回，详情日志可以滚动查看", async ({ page }) => {
  await login(page);
  const date = await readCurrentDate(page);
  await page.setViewportSize({ width: 1440, height: 540 });
  const prefix = "SCROLL_" + randomUUID().slice(0, 6);
  const session = await (await page.request.get("/api/v1/session")).json();
  // Independent fixture rows, saved through the same real Go command and database.
  for (let i = 0; i < 20; i++) {
    const code = prefix + "_" + String(i).padStart(2, "0");
    const r = await page.request.post("/api/v1/business-units", {
      headers: {
        Origin: new URL(page.url()).origin,
        "X-CSRF-Token": session.csrf,
        "Idempotency-Key": randomUUID(),
      },
      data: {
        code,
        name: code + " 单元",
        effectiveDate: date,
        operationType: "CREATE",
        operationReason: "NEW_ORGANIZATION",
      },
    });
    expect(r.status()).toBe(201);
  }
  await page.getByLabel("搜索编码或名称").fill(prefix);
  await page.getByRole("button", { name: "查询", exact: true }).click();
  await expect(page.getByText(recordRange(page, 20, 1))).toBeVisible();
  const scroller = page.locator(".MuiDataGrid-virtualScroller");
  await scroller.hover();
  await page.mouse.wheel(0, 650);
  await expect
    .poll(() => scroller.evaluate(e => (['auto', 'scroll'].includes(getComputedStyle(e).overflowY) ? e : document.scrollingElement!).scrollTop))
    .toBeGreaterThan(0);
  for (const dirty of [false, true]) {
    const createLink = page.getByRole("link", { name: "新建", exact: true });
    await createLink.focus();
    // Focusing the toolbar can scroll the short page before the link is activated.
    const departure = await createLink.evaluateHandle(link => {
      const position = { top: null as number | null };
      link.addEventListener('click', () => { position.top = document.scrollingElement!.scrollTop; }, { once: true });
      return position;
    });
    await createLink.press('Enter');
    const leaveScroll = await departure.evaluate(position => position.top);
    await departure.dispose();
    expect(leaveScroll).not.toBeNull();
    await expect(page.getByRole("heading", { name: "新建业务单元", exact: true })).toBeVisible();
    if (dirty) await page.getByRole("textbox", { name: "编码", exact: true }).fill("SCROLL_CANCEL");
    await page.getByRole("link", { name: "取消", exact: true }).click();
    if (dirty) await page.getByRole("button", { name: "放弃修改" }).click();
    await expect(page.getByText(recordRange(page, 20, 1))).toBeVisible();
    await expect.poll(() => page.evaluate(() => document.scrollingElement!.scrollTop)).toBe(leaveScroll);
  }
  const target = page.getByRole("link", {
    name: prefix + "_12 单元",
    exact: true,
  });
  await target.click();
  await expect(
    page.getByRole("heading", {
      name: prefix + "_12 · " + prefix + "_12 单元",
    }),
  ).toBeVisible();
  await page.getByRole("tab", { name: "变更日志", exact: true }).click();
  // The previous tab can still report ready before the log's first effect runs.
  // Wait for actual log content so its restoration cannot overwrite the wheel.
  await expect(page.getByRole("table", { name: "变更日志", exact: true })).toBeVisible();
  await expect(page.getByRole("region",{name:"业务单元详情正文"})).toHaveAttribute("data-page-scroll-ready","true");
  await page.setViewportSize({ width: 1440, height: 360 });
  // Use a real log control rather than fixed coordinates or the tab strip.
  await page.getByRole("button", { name: "刷新日志", exact: true }).hover();
  const beforeLogWheel = await page.getByRole("region", { name: "业务单元详情正文" }).evaluate(e => (['auto', 'scroll'].includes(getComputedStyle(e).overflowY) ? e : document.scrollingElement!).scrollTop);
  await page.mouse.wheel(0, 850);
  await expect
    .poll(() => page.getByRole("region", { name: "业务单元详情正文" }).evaluate(e => (['auto', 'scroll'].includes(getComputedStyle(e).overflowY) ? e : document.scrollingElement!).scrollTop))
    .toBeGreaterThan(beforeLogWheel);
  await expect(
    page.getByRole("link", { name: "查看当次资料" }),
  ).toBeVisible();
  const viewSource = page.getByRole('link', { name: '查看当次资料', exact: true });
  // Clicks can first scroll a partially visible button; remember the departure
  // position only after that required scrolling has finished.
  await viewSource.scrollIntoViewIfNeeded();
  await expect(viewSource).toBeInViewport({ ratio: 1 });
  const detailScroll = await page.getByRole('region', { name: '业务单元详情正文' }).evaluate(e => (['auto', 'scroll'].includes(getComputedStyle(e).overflowY) ? e : document.scrollingElement!).scrollTop);
  await viewSource.click();
  await expect(page.getByText('正在查看只读历史资料')).toBeVisible();
  await page.goBack();
  await expect(page.getByRole('tab', { name: '变更日志', exact: true })).toHaveAttribute('aria-selected', 'true');
  await expect(page.getByRole('link', { name: '查看当次资料', exact: true })).toBeVisible();
  await expect.poll(() => page.getByRole('region', { name: '业务单元详情正文' }).evaluate(e => (['auto', 'scroll'].includes(getComputedStyle(e).overflowY) ? e : document.scrollingElement!).scrollTop)).toBe(detailScroll);
  await page.setViewportSize({ width: 1440, height: 540 });
  await page.getByRole("link", { name: "返回列表" }).click();
  await expect(page.locator('.list-return-selected')).toContainText(
    prefix + "_12",
  );
  await expect
    .poll(() => page.locator(".MuiDataGrid-virtualScroller").evaluate(e => (['auto', 'scroll'].includes(getComputedStyle(e).overflowY) ? e : document.scrollingElement!).scrollTop))
    .toBeGreaterThan(0);
  await expect(
    page.getByRole("link", { name: prefix + "_12 单元", exact: true }),
  ).toBeFocused();
});

test("依赖加载可原地重试，退出失败后放弃输入不会锁死表单", async ({ page }) => {
  await login(page);
  // Transport failure injection only; successful reads still use the real API.
  let contextFailing = true;
  await page.route("**/api/v1/organization/context", async (route) => {
    if (contextFailing) {
      await route.abort("failed");
    } else await route.continue();
  });
  await page.getByRole("link", { name: "新建", exact: true }).click();
  await expect(page.getByRole("button", { name: "重试加载" })).toBeVisible();
  await expect(
    page.getByRole("textbox", { name: "编码", exact: true }),
  ).toBeDisabled();
  contextFailing = false;
  await page.getByRole("button", { name: "重试加载" }).click();
  await fillCreate(page, "EXIT_" + randomUUID().slice(0, 8));
  await page.route("**/api/v1/session", async (route) => {
    if (route.request().method() === "DELETE")
      await route.fulfill({
        status: 503,
        json: { code: "SERVICE_ERROR", message: "测试传输失败" },
      });
    else await route.continue();
  });
  await page.getByRole("button", { name: "退出", exact: true }).click();
  await page.getByRole("button", { name: "放弃修改" }).click();
  await expect(page.getByText("退出未完成，请重试")).toBeVisible();
  await expect(
    page.getByRole("textbox", { name: "编码", exact: true }),
  ).toHaveValue("");
  await expect(
    page.getByRole("textbox", { name: "编码", exact: true }),
  ).toBeEnabled();
  await fillCreate(page, "CONTINUE_" + randomUUID().slice(0, 8));
  await expect(
    page.getByRole("button", { name: "保存", exact: true }),
  ).toBeEnabled();
});

test("未知提交遇拒绝后清空资料，查回未保存终态可重新办理", async ({ page }) => {
  await deferAutomaticConfirmation(page);
  await login(page);
  await page.getByRole("link", { name: "新建", exact: true }).click();
  await fillCreate(page, "DENIED_" + randomUUID().slice(0, 8));
  let attempts = 0;
  let denyContext = false;
  // Inject denial envelopes to exercise UI recovery; real authorization races are in DB tests.
  await page.route("**/api/v1/organization/context", async (route) => {
    if (denyContext)
      await route.fulfill({
        status: 403,
        json: { code: "ACCESS_DENIED", message: "当前无访问权限" },
      });
    else await route.continue();
  });
  await page.route("**/api/v1/business-units", async (route) => {
    if (route.request().method() !== "POST") return route.continue();
    if (++attempts === 1) return route.abort("failed");
    denyContext = true;
    await route.fulfill({
      status: 403,
      json: { code: "ACCESS_DENIED", message: "当前无访问权限" },
    });
  });
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await page.getByRole("button", { name: "重试保存" }).click();
  await expect(
    page.getByRole("textbox", { name: "编码", exact: true }),
  ).toHaveValue("");
  await expect(page.getByRole("textbox", { name: "中文名称" })).toHaveValue("");
  await expect(page.getByText("所属企业：本地学习企业")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "重试保存" })).toHaveCount(
    0,
  );
  denyContext = false;
  await page.getByRole("button", { name: "重试加载" }).click();
  // Resolve really seals this never-received request as not_applied in PostgreSQL.
  await page.getByRole("button", { name: "查看保存结果" }).click();
  await expect(
    page.getByText("未保存"),
  ).toBeVisible();
  await expect(page.getByText(/暂时无法确认保存结果/, { exact: false })).toHaveCount(
    0,
  );
  await expect(
    page.getByRole("textbox", { name: "编码", exact: true }),
  ).toBeEnabled();
  expect(
    await page.evaluate(() =>
      JSON.parse(localStorage.getItem("chnhcm-pending-submissions") ?? "[]"),
    ),
  ).toEqual([]);
  await page.unroute("**/api/v1/business-units");
  await fillCreate(page, "RESOLVED_" + randomUUID().slice(0, 8));
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect(
    page.getByText("已保存。生效日期：", { exact: false }),
  ).toBeVisible();
});

test("旧会话迟到401保留新会话输入，当前会话401仍要求登录", async ({ page }) => {
  // This race deliberately delivers an old response even when the new
  // request ownership code correctly aborts it. Cancellation has separate tests.
  await page.addInitScript(() => {
    const original = window.fetch;
    window.fetch = (input, init) => {
      const request = new Request(input, init);
      return original(request.url.includes('/api/v1/business-units?')
        ? new Request(request, { signal: new AbortController().signal }) : request);
    };
  });
  let release!: () => void;
  let oldStarted = false;
  const held = new Promise<void>(resolve => { release = resolve; });
  await page.route("**/api/v1/business-units?*", async route => {
    if (!oldStarted && route.request().method() === "GET") {
      oldStarted = true;
      await held;
      await route.fulfill({ status: 401, json: { code: "AUTH_REQUIRED", message: "旧会话已失效" } });
    } else await route.continue();
  });
  await login(page);
  await expect.poll(() => oldStarted).toBe(true);
  await page.getByRole("button", { name: "退出", exact: true }).click();
  await expect(page.getByLabel("登录名")).toBeVisible();
  // Stay in the same SPA: a full navigation would cancel the held old response.
  await page.getByLabel("登录名").fill(credentials.login);
  await page.getByLabel("密码").fill(credentials.password);
  await submitLogin(page);
  await page.getByRole("link", { name: "新建", exact: true }).click();
  await fillCreate(page, "NEW_SESSION_INPUT");
  const lateResponse = page.waitForResponse(r => r.status() === 401 && r.url().includes("/api/v1/business-units?"));
  release();
  await (await lateResponse).finished();
  // A new authenticated read after delivery also lets the old promise settle.
  await expect.poll(async () => (await page.request.get("/api/v1/session")).status()).toBe(200);
  await expect(page.getByRole("textbox", { name: "编码", exact: true })).toHaveValue("NEW_SESSION_INPUT");
  await expect(page.getByLabel("登录名")).toHaveCount(0);
  await page.getByRole("link", { name: "取消", exact: true }).click();
  await page.getByRole("button", { name: "放弃修改" }).click();
  await page.route("**/api/v1/business-units?*", async route => {
    if (new URL(route.request().url()).searchParams.get("q") === "EXPIRED")
      await route.fulfill({ status: 401, json: { code: "AUTH_REQUIRED", message: "当前会话已失效" } });
    else await route.continue();
  });
  await page.getByLabel("搜索编码或名称").fill("EXPIRED");
  await page.getByRole("button", { name: "查询", exact: true }).click();
  await expect(page.getByLabel("登录名")).toBeVisible();
});

for (const deniedStatus of [403, 404]) {
  test(`查回${deniedStatus}清空正文和重试内容，保留引用直到确认终态`, async ({ page }) => {
  await deferAutomaticConfirmation(page);
    await login(page);
    await page.getByRole("link", { name: "新建", exact: true }).click();
    await fillCreate(page, "PENDING_" + deniedStatus);
    // Abort before the server receives the command; only the resolve envelope is simulated.
    await page.route("**/api/v1/business-units", async route => {
      if (route.request().method() === "POST") await route.abort("connectionreset");
      else await route.continue();
    });
    await page.getByRole("button", { name: "保存", exact: true }).click();
    await expect(page.getByRole("button", { name: "重试保存" })).toBeVisible();
    const pending = await page.evaluate(() => localStorage.getItem("chnhcm-pending-submissions"));
    if (deniedStatus === 404) {
      page.on("dialog", dialog => void dialog.accept());
      await page.reload();
      await fillCreate(page, "AFTER_REFRESH");
    }
    await page.route("**/api/v1/organization/submissions/*/resolve", route => route.fulfill({
      status: deniedStatus, json: { code: "ACCESS_DENIED", message: "查回暂不可访问" },
    }));
    await page.getByRole("button", { name: "查看保存结果" }).click();
    await expect(page.getByText("查回暂不可访问")).toBeVisible();
    for (const label of ["编码", "中文名称", "英文名称"])
      await expect(page.getByRole("textbox", { name: label, exact: true })).toHaveValue("");
    await expect(page.getByRole("combobox", { name: "操作原因" })).not.toHaveText("新设组织");
    await expect(page.getByRole("button", { name: "重试保存" })).toHaveCount(0);
    await expect(page.getByRole("button", { name: "保存", exact: true })).toBeDisabled();
    expect(await page.evaluate(() => localStorage.getItem("chnhcm-pending-submissions"))).toBe(pending);
    await page.unroute("**/api/v1/organization/submissions/*/resolve");
    await page.getByRole("button", { name: "查看保存结果" }).click();
    await expect(page.getByText("未保存")).toBeVisible();
    expect(await page.evaluate(() => JSON.parse(localStorage.getItem("chnhcm-pending-submissions") ?? "[]"))).toEqual([]);
    await expect(page.getByRole("textbox", { name: "编码", exact: true })).toBeEnabled();
  });
}

test("初始会话检查的迟到401不能清空已登录的新表单", async ({ page }) => {
  let release!: () => void;
  let heldFirst = false;
  const held = new Promise<void>(resolve => { release = resolve; });
  // StrictMode starts two bootstrap reads in the local development app.
  await page.route("**/api/v1/session", async route => {
    if (route.request().method() === "GET" && !heldFirst) {
      heldFirst = true;
      await held;
      await route.fulfill({ status: 401, json: { code: "AUTH_REQUIRED", message: "旧启动检查" } });
    } else await route.continue();
  });
  await login(page);
  await page.getByRole("link", { name: "新建", exact: true }).click();
  await fillCreate(page, "BOOTSTRAP_INPUT");
  const delivered = page.waitForResponse(r => r.status() === 401 && new URL(r.url()).pathname === "/api/v1/session");
  release();
  await (await delivered).finished();
  expect((await page.request.get("/api/v1/session")).status()).toBe(200);
  await expect(page.getByRole("textbox", { name: "编码", exact: true })).toHaveValue("BOOTSTRAP_INPUT");
  await expect(page.getByLabel("登录名")).toHaveCount(0);
});

test("旧会话的迟到查回拒绝不清空重新登录后的正文", async ({ page }) => {
  await deferAutomaticConfirmation(page);
  await login(page);
  await page.getByRole("link", { name: "新建", exact: true }).click();
  await fillCreate(page, "OLD_RESOLVE");
  await page.route("**/api/v1/business-units", async route => {
    if (route.request().method() === "POST") await route.abort("connectionreset");
    else await route.continue();
  });
  await page.getByRole("button", { name: "保存", exact: true }).click();
  let release!: () => void;
  let started = false;
  const held = new Promise<void>(resolve => { release = resolve; });
  await page.route("**/api/v1/organization/submissions/*/resolve", async route => {
    started = true;
    await held;
    await route.fulfill({ status: 403, json: { code: "ACCESS_DENIED", message: "旧查回拒绝" } });
  });
  await page.getByRole("button", { name: "查看保存结果" }).click();
  await expect.poll(() => started).toBe(true);
  await page.getByRole("button", { name: "退出", exact: true }).click();
  await page.getByRole("button", { name: /^(放弃修改|离开此页)$/ }).click();
  await page.getByLabel("登录名").fill(credentials.login);
  await page.getByLabel("密码").fill(credentials.password);
  await submitLogin(page);
  await fillCreate(page, "NEW_RESOLVE_INPUT");
  const delivered = page.waitForResponse(r => r.status() === 403 && r.url().endsWith("/resolve"));
  release();
  await (await delivered).finished();
  expect((await page.request.get("/api/v1/session")).status()).toBe(200);
  await expect(page.getByRole("textbox", { name: "编码", exact: true })).toHaveValue("NEW_RESOLVE_INPUT");
  await expect(page.getByText("旧查回拒绝")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "查看保存结果" })).toHaveCount(1);
});

test("离开表单后的迟到保存结果保留待确认引用，不跳转新会话", async ({ page }) => {
  await deferAutomaticConfirmation(page);
  await login(page);
  await page.getByRole("link", { name: "新建", exact: true }).click();
  await fillCreate(page, "LATE_SAVE_" + randomUUID().slice(0, 8));
  let release!: () => void;
  let committed = false;
  const held = new Promise<void>(resolve => { release = resolve; });
  await page.route("**/api/v1/business-units", async route => {
    if (route.request().method() !== "POST") return route.continue();
    const response = await route.fetch();
    expect(response.status()).toBe(201);
    committed = true;
    await held;
    await route.fulfill({ response });
  });
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect.poll(() => committed).toBe(true);
  await page.getByRole("link", { name: "取消", exact: true }).click();
  await page.getByRole("button", { name: /^(放弃修改|离开此页)$/ }).click();
  await page.getByRole("button", { name: "退出", exact: true }).click();
  await page.getByLabel("登录名").fill(credentials.login);
  await page.getByLabel("密码").fill(credentials.password);
  await submitLogin(page);
  await expect(page.getByRole("heading", { name: "业务单元", exact: true })).toBeVisible();
  const delivered = page.waitForResponse(r => r.status() === 201 && r.request().method() === "POST");
  release();
  await (await delivered).finished();
  expect((await page.request.get("/api/v1/session")).status()).toBe(200);
  await expect(page.getByRole("heading", { name: "业务单元", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "查看保存结果" })).toHaveCount(1);
  await page.getByRole("button", { name: "查看保存结果" }).click();
  await expect(page.getByText("已保存", { exact: true })).toBeVisible();
});

test("旧退出请求的迟到网络错误不写入新会话", async ({ page }) => {
  await login(page);
  let release!: () => void;
  let started = false;
  const held = new Promise<void>(resolve => { release = resolve; });
  await page.route("**/api/v1/session", async route => {
    if (route.request().method() !== "DELETE") return route.continue();
    started = true;
    await held;
    await route.abort("connectionreset");
  });
  await page.getByRole("button", { name: "退出", exact: true }).click();
  await expect.poll(() => started).toBe(true);
  await page.route("**/api/v1/business-units?*", async route => {
    if (new URL(route.request().url()).searchParams.get("q") === "REAUTH")
      await route.fulfill({ status: 401, json: { code: "AUTH_REQUIRED", message: "需重新登录" } });
    else await route.continue();
  });
  await page.getByLabel("搜索编码或名称").fill("REAUTH");
  await page.getByRole("button", { name: "查询", exact: true }).click();
  await expect(page.getByLabel("登录名")).toBeVisible();
  await page.unroute("**/api/v1/business-units?*");
  await page.getByLabel("登录名").fill(credentials.login);
  await page.getByLabel("密码").fill(credentials.password);
  await submitLogin(page);
  await page.getByRole("link", { name: "新建", exact: true }).click();
  await fillCreate(page, "NEW_AFTER_EXIT");
  const failed = page.waitForEvent("requestfailed", request => request.method() === "DELETE");
  release();
  await failed;
  expect((await page.request.get("/api/v1/session")).status()).toBe(200);
  await expect(page.getByRole("textbox", { name: "编码", exact: true })).toHaveValue("NEW_AFTER_EXIT");
  await expect(page.getByText("退出结果尚未确认，请重试")).toHaveCount(0);
});

test("301原内容重试期间禁止确认，完成后只保存一次", async ({ page }) => {
  await deferAutomaticConfirmation(page);
  await login(page);
  await page.getByRole("link", { name: "新建", exact: true }).click();
  await fillCreate(page, "RESOLVE_RACE_" + randomUUID().slice(0, 8));
  let calls = 0;
  let started = false;
  let release!: () => void;
  const held = new Promise<void>(resolve => { release = resolve; });
  await page.route("**/api/v1/business-units", async route => {
    if (route.request().method() !== "POST") return route.continue();
    if (++calls === 1) return route.abort("connectionreset");
    if (calls === 2) {
      started = true;
      await held;
    }
    await route.continue();
  });
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await page.getByRole("button", { name: "重试保存" }).click();
  await expect.poll(() => started).toBe(true);
  await expect(page.getByRole("button", { name: "查看保存结果" })).toBeDisabled();
  const response = page.waitForResponse(r => r.request().method() === "POST" && r.url().endsWith("/business-units"));
  release();
  const saved = await response; expect(saved.status()).toBe(201); await saved.finished();
  await expect(page.getByText("已保存。生效日期：", { exact: false })).toBeVisible();
  expect(calls).toBe(2);
  expect(await page.evaluate(() => JSON.parse(localStorage.getItem("chnhcm-pending-submissions") ?? "[]"))).toEqual([]);
});

test("TL202 重试结束仍未知时可手动确认未保存，随后开始新操作", async ({ page }) => {
  await deferAutomaticConfirmation(page);
  await login(page);
  await page.getByRole("link", { name: "新建", exact: true }).click();
  await fillCreate(page, "RESOLVE_RACE_" + randomUUID().slice(0, 8));
  let calls = 0;
  let started = false;
  let release!: () => void;
  const held = new Promise<void>(resolve => { release = resolve; });
  await page.route("**/api/v1/business-units", async route => {
    if (route.request().method() !== "POST") return route.continue();
    if (++calls === 1) return route.abort("connectionreset");
    if (calls === 2) {
      started = true;
      await held;
      return route.abort("connectionreset");
    }
    await route.continue();
  });
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await page.getByRole("button", { name: "重试保存" }).click();
  await expect.poll(() => started).toBe(true);
  await expect(page.getByRole("button", { name: "查看保存结果" })).toBeDisabled();
  const response = page.waitForEvent("requestfailed", r => r.method() === "POST" && r.url().endsWith("/business-units"));
  release();
  await response;
  await page.getByRole("button", { name: "查看保存结果" }).click();
  await expect(page.getByText("未保存", {exact:true})).toBeVisible();
  await expect(page.getByRole("button", { name: "查看保存结果" })).toHaveCount(0);
  expect(await page.evaluate(() => JSON.parse(localStorage.getItem("chnhcm-pending-submissions") ?? "[]"))).toEqual([]);
  await expect(page.getByRole("textbox", { name: "编码", exact: true })).toBeEnabled();
  await expect(page.getByRole("button", { name: "保存", exact: true })).toBeEnabled();
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect(page.getByText("已保存。生效日期：", { exact: false })).toBeVisible();
});

test("旧退出204迟到后，新登录仍可访问且输入保留", async ({ page }) => {
  await login(page);
  let started = false;
  let release!: () => void;
  const held = new Promise<void>(resolve => { release = resolve; });
  let releaseRead!: () => void;
  let capturedRead!: () => void;
  let holdRefresh = true;
  const readHeld = new Promise<void>(resolve => { releaseRead = resolve; });
  const readCaptured = new Promise<void>(resolve => { capturedRead = resolve; });
  await page.route(url => url.pathname === '/api/v1/business-units', async route => {
    if (route.request().method() !== 'GET' || !holdRefresh) return route.continue();
    capturedRead();
    await readHeld;
    const response = await route.fetch();
    expect(response.status()).toBe(401);
    await route.fulfill({ response });
  });
  await page.route("**/api/v1/session", async route => {
    if (route.request().method() !== "DELETE") return route.continue();
    const response = await route.fetch();
    expect(response.status()).toBe(204);
    started = true;
    await held;
    await route.fulfill({ response });
  });
  await page.getByRole("button", { name: "退出", exact: true }).click();
  await expect.poll(() => started).toBe(true);
  // Logout invalidation itself refreshes the list. Hold that actual request
  // until the server has revoked the session; never race a click against an
  // already displayed login page to provoke the same 401.
  await readCaptured;
  holdRefresh = false;
  releaseRead();
  await expect(page.getByLabel("登录名")).toBeVisible();
  await page.getByLabel("登录名").fill(credentials.login);
  await page.getByLabel("密码").fill(credentials.password);
  await submitLogin(page);
  await page.getByRole("link", { name: "新建", exact: true }).click();
  await fillCreate(page, "AFTER_LOGOUT_RESPONSE");
  const delivered = page.waitForResponse(r => r.request().method() === "DELETE");
  release();
  expect((await delivered).status()).toBe(204);
  expect((await page.request.get("/api/v1/session")).status()).toBe(200);
  await expect(page.getByRole("textbox", { name: "编码", exact: true })).toHaveValue("AFTER_LOGOUT_RESPONSE");
});

test("未知详情条件拒绝，可返回当前资料", async ({ page }) => {
  await login(page);
  await page.getByRole("link", { name: "新建", exact: true }).click();
  await fillCreate(page, "ADOPT_LINK_" + randomUUID().slice(0, 8));
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect(page.getByText("已保存。生效日期：", { exact: false })).toBeVisible();
  const target = new URL(page.url());
  target.searchParams.set("adoptedRef", randomUUID());
  expect((await page.request.get(`/api/v1/business-units/${target.pathname.split('/').pop()}?adoptedRef=unsupported`)).status()).toBe(400);
  await page.goto(target.toString());
  await expect(page.getByText("未知的查询条件，请返回当前资料重新选择。")).toBeVisible();
  await expect(page.getByRole("heading", { name: "基本资料" })).toHaveCount(0);
  await page.getByRole("button", { name: "返回当前资料" }).click();
  await expect(page.getByRole("heading", { name: "基本资料" })).toBeVisible();
  expect(new URL(page.url()).searchParams.has("adoptedRef")).toBe(false);
});

for (const outcome of ["rejected", "unknown"]) {
  test(`上下文403清理${outcome}表单正文，待确认引用单独保留`, async ({ page }) => {
  await deferAutomaticConfirmation(page);
    await login(page);
    await page.getByRole("link", { name: "新建", exact: true }).click();
    await fillCreate(page, outcome === "rejected" ? "_invalid" : "CONTEXT_DENIED");
    if (outcome === "unknown") {
      await page.route("**/api/v1/business-units", async route => {
        if (route.request().method() === "POST") await route.abort("connectionreset");
        else await route.continue();
      });
    }
    await page.getByRole("button", { name: "保存", exact: true }).click();
    await expect(outcome === 'unknown' ? page.getByRole('button', { name: '重试保存' }) : page.getByText('编码须为', { exact: false }).first()).toBeVisible();
    const pending = await page.evaluate(() => localStorage.getItem("chnhcm-pending-submissions"));
    let contextCalls = 0;
    await page.route("**/api/v1/organization/context", async route => {
      contextCalls++;
      await route.fulfill({ status: 403, json: { code: "ACCESS_DENIED", message: "组织资料权限已收回" } });
    });
    await page.route("**/api/v1/business-units", route => route.request().method() === 'POST' ? route.fulfill({ status: 403, json: { code: 'ACCESS_DENIED', message: '当前办理权限已收回' } }) : route.continue());
    if (outcome === 'rejected') {
      await page.getByRole('textbox', { name: '编码', exact: true }).fill('CONTEXT_RETRY');
      await page.getByRole('button', { name: '保存', exact: true }).click();
    } else await page.getByRole('button', { name: '重试保存' }).click();
    await expect(page.getByText("组织资料权限已收回")).toBeVisible();
    for (const label of ["编码", "中文名称"])
      await expect(page.getByRole("textbox", { name: label, exact: true })).toHaveValue("");
    await expect(page.getByLabel("生效日期")).toHaveValue("");
    await expect(page.getByRole("combobox", { name: "操作原因" })).not.toHaveText("新设组织");
    await expect(page.getByRole("button", { name: "重试保存" })).toHaveCount(0);
    await expect(page.getByRole("button", { name: "保存", exact: true })).toBeDisabled();
    expect(await page.evaluate(() => localStorage.getItem("chnhcm-pending-submissions"))).toBe(pending);
    expect(contextCalls).toBe(1);
    await page.unroute("**/api/v1/organization/context");
    await page.getByRole("button", { name: "重试加载" }).click();
    await expect(page.getByLabel("英文名称")).toHaveValue("");
    if (outcome === "unknown") {
      await page.getByRole("button", { name: "查看保存结果" }).click();
      await expect(page.getByText("未保存")).toBeVisible();
    }
    await expect(page.getByRole("textbox", { name: "编码", exact: true })).toBeEnabled();
  });
}

test("301保存响应未结束时不允许确认，成功后不再发确认", async ({ page }) => {
  await deferAutomaticConfirmation(page);
  await login(page);
  await page.getByRole("link", { name: "新建", exact: true }).click();
  await fillCreate(page, "PENDING_LATE_" + randomUUID().slice(0, 8));
  let releaseSave!: () => void;
  let committed = false;
  let resolving = false;
  const heldSave = new Promise<void>(resolve => { releaseSave = resolve; });
  await page.route("**/api/v1/business-units", async route => {
    if (route.request().method() !== "POST") return route.continue();
    const response = await route.fetch();
    expect(response.status()).toBe(201);
    committed = true;
    await heldSave;
    await route.fulfill({ response });
  });
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect.poll(() => committed).toBe(true);
  await page.route("**/api/v1/organization/submissions/*/resolve", async route => {
    resolving = true;
    const submissionId = new URL(route.request().url()).pathname.split("/").at(-2)!;
    await route.fulfill({ status: 202, json: { submissionId, outcome: "pending" } });
  });
  await expect(page.getByRole("button", { name: "查看保存结果" })).toBeDisabled();
  await expect(page.getByRole("region", { name: "业务单元提交结果", exact: true }).getByRole("alert")).toHaveClass(/MuiAlert-standardInfo/);
  await expect(page.getByText("正在保存，请等待结果。", { exact: true })).toHaveCount(0);
  expect(resolving).toBe(false);
  releaseSave();
  await expect(page.getByText("已保存。生效日期：", { exact: false })).toBeVisible();
  expect(resolving).toBe(false);
  await expect(page.getByRole("button", { name: "查看保存结果" })).toHaveCount(0);
});

test("默认列表查回未来首条，准确定位原保存记录", async ({ page }) => {
  await deferAutomaticConfirmation(page);
  await login(page);
  const date = await readCurrentDate(page);
  const future = new Date(date + "T00:00:00Z");
  future.setUTCDate(future.getUTCDate() + 2);
  await page.getByRole("link", { name: "新建", exact: true }).click();
  await fillCreate(page, "DEFAULT_RECOVERY_" + randomUUID().slice(0, 8), future.toISOString().slice(0, 10));
  await page.route("**/api/v1/business-units", async route => {
    if (route.request().method() !== "POST") return route.continue();
    const response = await route.fetch();
    expect(response.status()).toBe(201);
    await route.abort("connectionreset");
  });
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect(page.getByText(/暂时无法确认保存结果/, { exact: false })).toBeVisible();
  await page.getByRole('button', { name: '组织配置', exact: true }).click();
  await page.getByRole('menuitem', { name: '业务单元', exact: true }).click();
  await expect(page.getByRole('dialog', { name: /^(尚有未保存修改|保存结果尚未确认)$/, exact: true })).toBeVisible();
  await page.getByRole("button", { name: /^(放弃修改|离开此页)$/ }).click();
  await expect(page.getByRole("heading", { name: "业务单元", exact: true })).toBeVisible();
  expect(new URL(page.url()).searchParams.has("businessAt")).toBe(false);
  await expectCurrentDate(page, date);
  await page.getByRole("button", { name: "查看保存结果" }).click();
  await page.getByRole("link", { name: "查看保存结果" }).click();
  await expect(page.getByText("正在查看只读历史资料")).toBeVisible();
  await expectCurrentDate(page, date);
  await expect(page.getByText("所选记录尚未生效。", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "返回当前资料" }).click();
  await expectCurrentDate(page, date);
  await page.getByRole("link", { name: "返回列表" }).click();
  await expectCurrentDate(page, date);
});

test("刷新会话临时失败可原地重试，真实退出仍要求登录", async ({ page }) => {
  await login(page);
  await page.route("**/api/v1/session", async route => {
    if (route.request().method() === "GET")
      await route.fulfill({ status: 503, json: { code: "M02_DEPENDENCY_UNAVAILABLE", message: "暂时无法读取登录状态" } });
    else await route.continue();
  });
  await page.reload();
  await expect(page.getByText("暂时无法确认登录状态，请重试。")).toBeVisible();
  await expect(page.getByLabel("密码")).toHaveCount(0);
  await page.unroute("**/api/v1/session");
  await page.getByRole("button", { name: "重试", exact: true }).click();
  await expect(page.getByRole("heading", { name: "业务单元", exact: true })).toBeVisible();
  expect((await page.request.get("/api/v1/session")).status()).toBe(200);
  const exited = page.waitForResponse(r =>
    r.request().method() === "DELETE" && new URL(r.url()).pathname === "/api/v1/session");
  await page.getByRole("button", { name: "退出", exact: true }).click();
  expect((await exited).status()).toBe(204);
  await expect(page.getByLabel("登录名")).toBeVisible();
  await page.reload();
  await expect(page.getByLabel("登录名")).toBeVisible();
  await expect(page.getByRole("button", { name: "登录", exact: true })).toBeEnabled();
  await expect(page.getByText("暂时无法确认登录状态，请重试。")).toHaveCount(0);
});

test("会话已恢复后，迟到启动503不卸载新建表单", async ({ page }) => {
  await login(page);
  let release!: () => void;
  let heldFirst = false;
  const held = new Promise<void>(resolve => { release = resolve; });
  await page.route("**/api/v1/session", async route => {
    if (route.request().method() === "GET" && !heldFirst) {
      heldFirst = true;
      await held;
      await route.fulfill({ status: 503, json: { code: "M02_DEPENDENCY_UNAVAILABLE", message: "迟到启动失败" } });
    } else await route.continue();
  });
  await page.reload();
  await expect(page.getByRole("heading", { name: "业务单元", exact: true })).toBeVisible();
  await page.getByRole("link", { name: "新建", exact: true }).click();
  await fillCreate(page, "RESTORED_SESSION_INPUT");
  const delivered = page.waitForResponse(r => r.status() === 503 && new URL(r.url()).pathname === "/api/v1/session");
  release();
  await (await delivered).finished();
  expect((await page.request.get("/api/v1/session")).status()).toBe(200);
  await expect(page.getByRole("textbox", { name: "编码", exact: true })).toHaveValue("RESTORED_SESSION_INPUT");
  await expect(page.getByText("暂时无法确认登录状态，请重试。")).toHaveCount(0);
  await page.getByRole("button", { name: "退出", exact: true }).click();
  await page.getByRole("button", { name: "放弃修改" }).click();
  await expect(page.getByRole("button", { name: "登录", exact: true })).toBeEnabled();
});

test("查回临时失败后原内容重试成功，清除旧错误提示", async ({ page }) => {
  await deferAutomaticConfirmation(page);
  await login(page);
  await page.getByRole("link", { name: "新建", exact: true }).click();
  await fillCreate(page, "RETRY_RECOVERY_" + randomUUID().slice(0, 8));
  let first = true;
  await page.route("**/api/v1/business-units", async route => {
    if (route.request().method() === "POST" && first) {
      first = false;
      const response = await route.fetch();
      expect(response.status()).toBe(201);
      await route.abort("connectionreset");
    } else await route.continue();
  });
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect(page.getByRole("button", { name: "重试保存" })).toBeVisible();
  await page.route("**/api/v1/organization/submissions/*/resolve", route => route.fulfill({
    status: 503, json: { code: "M02_DEPENDENCY_UNAVAILABLE", message: "暂时无法查回" },
  }));
  await page.getByRole("button", { name: "查看保存结果" }).click();
  await expect(page.getByText("暂时无法查回")).toBeVisible();
  await page.getByRole("button", { name: "重试保存" }).click();
  await expect(page.getByText("已保存。生效日期：", { exact: false })).toBeVisible();
  await expect(page.getByText("暂时无法查回")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "查看保存结果" })).toHaveCount(0);
  expect(await page.evaluate(() => JSON.parse(localStorage.getItem("chnhcm-pending-submissions") ?? "[]"))).toEqual([]);
});


test('对象导航跨服务端分页、返回定位，并在当前授权拒绝时清空详情', async ({ page }) => {
  await login(page);
  const date = await readCurrentDate(page);
  const prefix = 'NAV019_' + randomUUID().slice(0, 6);
  const principal = await (await page.request.get('/api/v1/session')).json();
  for (let index = 0; index < 21; index++) {
    const code = prefix + '_' + String(index).padStart(2, '0');
    const result = await page.request.post('/api/v1/business-units', {
      headers: { Origin: new URL(page.url()).origin, 'X-CSRF-Token': principal.csrf, 'Idempotency-Key': randomUUID() },
      data: { code, name: code + ' 单元', effectiveDate: date, operationType: 'CREATE', operationReason: 'NEW_ORGANIZATION' },
    });
    expect(result.status()).toBe(201);
  }
  await page.getByLabel('搜索编码或名称').fill(prefix);
  await page.getByRole('button', { name: '查询', exact: true }).click();
  await expect(page.getByText(recordRange(page, 21, 1))).toBeVisible();
  await page.getByRole('link', { name: prefix + '_19 单元', exact: true }).click();
  await expect(page.getByRole('button', { name: '下一个对象', exact: true })).toBeEnabled();
  await page.getByRole('button', { name: '下一个对象', exact: true }).click();
  await expect(page.getByRole('heading', { name: prefix + '_20 · ' + prefix + '_20 单元' })).toBeVisible();
  await expect(page.getByRole('button', { name: '上一个对象', exact: true })).toBeFocused();
  await expect(page.getByRole('button', { name: '下一个对象', exact: true })).toBeDisabled();
  await page.getByRole('link', { name: '返回列表', exact: true }).click();
  await expect(page.getByText(recordRange(page, 21, 2))).toBeVisible();
  await expect(page.locator('.list-return-selected')).toContainText(prefix + '_20');
  await page.getByRole('link', { name: prefix + '_20 单元', exact: true }).click();
  await expect(page.getByRole('button', { name: '上一个对象', exact: true })).toBeEnabled();
  await page.route('**/api/v1/business-units/navigation', route => route.fulfill({ status: 503, json: { code: 'M02_DEPENDENCY_UNAVAILABLE', message: '受控辅助失败' } }));
  await page.getByRole('button', { name: '上一个对象', exact: true }).click();
  await expect(page.getByText('暂时无法切换记录')).toBeVisible();
  await expect(page.getByRole('heading', { name: '基本资料', exact: true })).toBeVisible();
  await page.unroute('**/api/v1/business-units/navigation');
  await page.reload();
  await expect(page.getByRole('button', { name: '上一个对象', exact: true })).toBeEnabled();
  await page.route('**/api/v1/business-units/*', route => route.request().method() === 'GET'
    ? route.fulfill({ status: 403, json: { code: 'M02_FORBIDDEN', message: '当前权限不足，请返回列表。' } }) : route.continue());
  await page.getByRole('button', { name: '上一个对象', exact: true }).click();
  await expect(page.getByText('当前权限不足，请返回列表。')).toBeVisible();
  await expect(page.getByRole('heading', { name: '基本资料', exact: true })).toHaveCount(0);
  await expect(page.getByRole('button', { name: '来源说明', exact: true })).toHaveCount(0);
});


test('公共导航桌面键盘关闭和导航断点切换恢复焦点', async ({ page }) => {
  await login(page);
  const section = page.getByRole('button', { name: '组织配置', exact: true });
  await section.focus(); await page.keyboard.press('Enter');
  await expect(page.getByRole('menuitem', { name: '业务单元', exact: true })).toBeFocused();
  await page.keyboard.press('Escape'); await expect(section).toBeFocused();
  await page.setViewportSize({ width: 899, height: 900 });
  await page.getByRole('button', { name: '打开导航', exact: true }).click();
  await expect(page.locator('#navigation-drawer').getByRole('button', { name: '业务单元', exact: true })).toBeVisible();
  await page.getByRole('button', { name: '关闭导航', exact: true }).click();
  await expect(page.getByRole('button', { name: '打开导航', exact: true })).toBeFocused();
  await page.getByRole('button', { name: '打开导航', exact: true }).click();
  await page.setViewportSize({ width: 1440, height: 900 });
  await expect(page.locator('#navigation-drawer')).not.toBeVisible();
  await expect(page.getByRole('button', { name: '切换应用', exact: true })).toBeFocused();
  await expect(page.getByRole('heading', { name: '业务单元', exact: true })).toBeVisible();
});


test('唯一原因自动呈现，确认保存后的本地清理失败不变成未知', async ({ page }) => {
  await deferAutomaticConfirmation(page);
  await login(page);
  await page.getByRole('link', { name: '新建', exact: true }).click();
  await expect(page.getByRole('combobox', { name: '操作原因' })).toHaveText('新设组织');
  // A program default alone must not create a dirty form.
  await page.getByRole('link', { name: '取消', exact: true }).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await page.getByRole('link', { name: '新建', exact: true }).click();
  const code = 'DEFAULT_' + randomUUID().slice(0, 8);
  await page.getByRole('textbox', { name: '编码', exact: true }).fill(code);
  await page.getByLabel('中文名称').fill('默认原因保存');
  await page.evaluate(() => {
    const original = Storage.prototype.setItem;
    Storage.prototype.setItem = function(key, value) {
      if (key === 'chnhcm-pending-submissions' && value === '[]') throw new Error('test cleanup unavailable');
      return original.call(this, key, value);
    };
  });
  let writes = 0;
  page.on('request', r => { if (new URL(r.url()).pathname === '/api/v1/business-units' && r.method() === 'POST') writes++; });
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.getByText('已保存。生效日期：', { exact: false })).toBeVisible();
  await expect(page.getByRole('button', { name: '保存', exact: true })).toHaveCount(0);
  await expect(page.getByText('有一笔提交结果待确认', { exact: false })).toHaveCount(0);
  await expect(page.getByRole('button', { name: '重试保存' })).toHaveCount(0);
  expect(writes).toBe(1);
  expect(await page.evaluate(() => JSON.parse(localStorage.getItem('chnhcm-pending-submissions') ?? '[]').length)).toBe(1);
});
