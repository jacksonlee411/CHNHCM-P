import { lazyCredentials } from './credentials.mjs';
import { deferAutomaticConfirmation } from './fixtures';
import { readRecord } from './fixtures';
import { test, expect, readCurrentDate, expectCurrentDate, openMoreActions, type Page } from "./fixtures";
const businessDate = (date: string) => date.replace(/^(\d{4})-(\d{2})-(\d{2})$/, '$1-$2-$3');
import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import path from "node:path";
import { randomUUID } from "node:crypto";
const root = path.resolve(import.meta.dirname, "../..");
const credentials = lazyCredentials();
const buPath = "/settings/organization/business-units";
async function open(page: Page) {
  await new Promise((resolve) => setTimeout(resolve, 3100));
  await page.goto("/");
  await page.getByLabel("登录名").fill(credentials.login);
  await page.getByLabel("密码").fill(credentials.password);
  await page.getByRole("button", { name: "登录", exact: true }).click();
  await page.getByRole("button", { name: "切换应用", exact: true }).click();
  await page.getByRole("menuitem", { name: "设置", exact: true }).click();
  await page.getByRole("button", { name: "组织配置", exact: true }).click();
  await page.getByRole("menuitem", { name: "业务单元", exact: true }).click();
  expect(await readCurrentDate(page)).toMatch(/^\d{4}-\d{2}-\d{2}$/);
}
async function create(page: Page) {
  const code = "TL_" + randomUUID().slice(0, 8);
  await page.getByRole("link", { name: "新建", exact: true }).click();
  await page.getByRole("textbox", { name: "编码", exact: true }).fill(code);
  await page.getByRole("textbox", { name: "中文名称", exact: true }).fill("甲");
  await page
    .getByRole("textbox", { name: "英文名称", exact: true })
    .fill("Alpha");
  await page
    .getByLabel("生效日期")
    .and(page.locator("input"))
    .fill("2026-01-01");
  await page.getByRole("combobox", { name: "操作原因", exact: true }).click();
  await page.getByRole("option", { name: "新设组织", exact: true }).click();
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect(
    page.getByText("已保存。生效日期：", { exact: false }),
  ).toBeVisible();
  return code;
}
async function add(page: Page, date: string, name: string, en?: string) {
 const d=await read(page), anchor=d.timeline.filter((r:{effectiveDate:string})=>r.effectiveDate<date).at(-1)??d.timeline[0];
 const [ay,am,ad]=anchor.effectiveDate.split('-');await page.getByRole('button',{name:`${ay}-${am}-${ad}`,exact:true}).click();await expect(page.getByRole('button',{name:`${ay}-${am}-${ad}`,exact:true})).toHaveAttribute('aria-current','date');
  await page.getByRole("button", { name: /^(新增记录|插入记录)$/, exact: true }).click();
  await page.getByLabel("生效日期").and(page.locator("input")).fill(date);
  await page.getByRole("combobox", { name: "操作原因", exact: true }).click();
  await page.getByRole("option", { name: "名称调整", exact: true }).click();
  await page.getByRole("textbox", { name: "中文名称", exact: true }).fill(name);
  if (en !== undefined)
    await page.getByRole("textbox", { name: "英文名称", exact: true }).fill(en);
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect(
    page.getByRole("button", { name: /^(新增记录|插入记录)$/, exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText(`已保存。生效日期：${businessDate(date)}`, { exact: false }),
  ).toBeVisible();
}
async function read(page: Page, date?: string, source?: string) { return readRecord(page, date, source); }
test("正常菜单追溯新增、更正、独立快照、原新来源及刷新", async ({ page }) => {
  await open(page);
  const viewing = await readCurrentDate(page);
  await create(page);
  await add(page, "2026-06-01", "乙", "Beta");
  await add(page, "2026-10-01", "丙", "Gamma");
  const old = (await read(page, "2026-03-01")).headSourceVersion;
  await add(page, "2026-03-01", "新");
  const middle = (await read(page, "2026-03-01")).headSourceVersion;
  await page.getByRole("button", { name: "2026-01-01", exact: true }).click();
  await expect(page.getByRole("button", { name: "2026-01-01", exact: true })).toHaveAttribute("aria-current", "date");
  await openMoreActions(page);
  await page.getByRole("menuitem", { name: "更正记录", exact: true }).click();
  await expect(
    page.locator(".object-form").getByText("操作类型", { exact: true }).locator('..'),
  ).toContainText('新建');
  await expect(
    page.getByRole("combobox", { name: "操作原因", exact: true }),
  ).toBeVisible();
  await page
    .getByRole("textbox", { name: "中文名称", exact: true })
    .fill("甲2");
  const correctedInput = page.waitForRequest(request => request.method() === 'POST' && request.url().endsWith('/correct'));
  await expect(page.locator('.object-form')).not.toContainText(/办理校验日|期间调整|M03\/M04|只提交明确修改字段/);
  await expect(page.getByRole('button', {name:'预览',exact:true})).toHaveCount(0);
  await page.getByRole("button", {name:"保存",exact:true}).click();
  expect((await correctedInput).postDataJSON().changes).toEqual({name:'甲2'});
  await expect(
    page.getByRole("button", { name: /^(新增记录|插入记录)$/, exact: true }),
  ).toBeVisible();
  for (const [date, name, en] of [
    ["2026-01-01", "甲2", "Alpha"],
    ["2026-03-01", "新", "Alpha"],
    ["2026-06-01", "乙", "Beta"],
    ["2026-10-01", "丙", "Gamma"],
  ]) {
    const r = await read(page, date);
    expect(r.selectedRecord.name).toBe(name);
    expect(r.selectedRecord.nameEn).toBe(en);
  }
  expect((await read(page, "2026-03-01", old)).selectedRecord.name).toBe("甲");
  expect((await read(page, "2026-03-01", old)).selectedRecord.endExclusive).toBe(
    "2026-06-01",
  );
  expect((await read(page, "2026-03-01", middle)).selectedRecord.nameEn).toBe(
    "Alpha",
  );
  execFileSync("python3", ["scripts/local-bu.py", "restart-api"], {
    cwd: root,
    stdio: "pipe",
    timeout: 60000,
  });
  await page.reload();
  await expectCurrentDate(page, viewing);
  expect((await read(page, "2026-03-01")).selectedRecord.name).toBe("新");
  await page.getByRole("tab", { name: "变更日志", exact: true }).click();
  await page
    .getByRole("link", { name: "查看当次资料", exact: true })
    .first()
    .click();
  await expect(page.getByText("只读", { exact: false }).first()).toBeVisible();
  await expect(
    page.getByRole("button", { name: /^(新增记录|插入记录)$/, exact: true }),
  ).toHaveCount(0);
});
test("新增未保存保护、同日错误及响应丢失后刷新查回", async ({ page }) => {
  await deferAutomaticConfirmation(page);
  await open(page);
  await create(page);
  await add(page, "2026-06-01", "乙", "Beta");
  await add(page, "2026-10-01", "丙", "Gamma");
  await page.getByRole("button",{name:"2026-01-01",exact:true}).click();
  await expect(page.getByRole("button",{name:"2026-01-01",exact:true})).toHaveAttribute("aria-current","date");
  await page.getByRole("button", { name: /^(新增记录|插入记录)$/, exact: true }).click();
  await page
    .getByLabel("生效日期")
    .and(page.locator("input"))
    .fill("2026-03-01");
  await page
    .getByRole("textbox", { name: "中文名称", exact: true })
    .fill("未保存");
  await page.getByRole("button", { name: "取消编辑", exact: true }).click();
  await expect(page.getByRole("dialog")).toBeVisible();
  await page.getByRole("button", { name: "继续编辑", exact: true }).click();
  await expect(
    page.getByRole("textbox", { name: "中文名称", exact: true }),
  ).toHaveValue("未保存");
  await page.getByRole("button", { name: "取消编辑", exact: true }).click();
  await page.getByRole("button", { name: /^(放弃修改|离开此页)$/, exact: true }).click();
  let source = "";
  let target = "";
  let key = "";
  await page.route("**/api/v1/business-units/*/records", async (route) => {
    const response = await route.fetch();
    expect(response.status()).toBe(201);
    const r = await response.json();
    source = r.sourceVersion;
    target = r.selectedRecordId;
    key = route.request().headers()["idempotency-key"];
    await route.abort("failed");
  });
  await page.getByRole("button", { name: /^(新增记录|插入记录)$/, exact: true }).click();
  await page
    .getByLabel("生效日期")
    .and(page.locator("input"))
    .fill("2026-03-01");
  await page.getByRole("combobox", { name: "操作原因", exact: true }).click();
  await page.getByRole("option", { name: "名称调整", exact: true }).click();
  await page
    .getByRole("textbox", { name: "中文名称", exact: true })
    .fill("已保存未响应");
  await expect(page.getByRole('button', {name:'预览',exact:true})).toHaveCount(0);
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect(
    page.getByText(/暂时无法确认保存结果/, { exact: true }),
  ).toBeVisible();
  await page.unroute("**/api/v1/business-units/*/records");
  page.once("dialog", dialog => void dialog.accept());
  await page.reload();
  await page.getByRole("button", { name: "查看保存结果", exact: true }).click();
  await expect(
    page.getByText("已保存", { exact: true }),
  ).toBeVisible();
  await page.getByRole("link", { name: "查看保存结果", exact: true }).click();
  expect(new URL(page.url()).searchParams.get("sourceVersion")).toBe(source);
  expect(new URL(page.url()).searchParams.get("selectedRecordId")).toBe(target);
  await expect(
    page.getByRole("heading", { name: /已保存未响应/ }),
  ).toBeVisible();
  expect((await read(page, "2026-03-01", source)).selectedRecord.name).toBe(
    "已保存未响应",
  );
  const refs = await page.evaluate(() =>
    localStorage.getItem("chnhcm-pending-submissions"),
  );
  await page.getByRole("button", { name: "返回当前资料", exact: true }).click();
  expect(new URL(page.url()).searchParams.get("selectedRecordId")).toBe(target);
  await expect(
    page.getByRole("heading", { name: /已保存未响应/ }),
  ).toBeVisible();
  expect(refs).not.toContain(key);
});

test("无变化、同日拒绝与真实并发后保留输入重新核对", async ({ page }) => {
  await open(page);
  await create(page);
  await page.getByRole("button", { name: /^(新增记录|插入记录)$/, exact: true }).click();
  await page
    .getByLabel("生效日期")
    .and(page.locator("input"))
    .fill("2026-03-01");
  await page.getByRole("combobox", { name: "操作原因", exact: true }).click();
  await page.getByRole("option", { name: "名称调整", exact: true }).click();
  await page.getByRole('textbox', { name: '中文名称', exact: true }).fill('临时输入');
  await page.getByRole("textbox", { name: "中文名称", exact: true }).fill("甲");

  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect(
    page.getByRole('status').filter({ hasText: '没有修改内容。' }),
  ).toBeVisible();
  expect((await read(page, "2026-03-01")).timeline).toHaveLength(1);
  await page.getByRole("button", { name: /^(新增记录|插入记录)$/, exact: true }).click();
  await page
    .getByLabel("生效日期")
    .and(page.locator("input"))
    .fill("2026-01-01");
  await page.getByRole("combobox", { name: "操作原因", exact: true }).click();
  await page.getByRole("option", { name: "名称调整", exact: true }).click();
  await page
    .getByRole("textbox", { name: "中文名称", exact: true })
    .fill("我的输入");
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect(
    page.getByRole('button', { name: "同一日期已有生效记录", exact: true }),
  ).toBeVisible();
  await expect(page.locator('#record-date')).toHaveAttribute('aria-invalid', 'true');
  await expect(page.locator("#record-date")).toBeFocused();
  await page
    .getByLabel("生效日期")
    .and(page.locator("input"))
    .fill("2026-03-01");
  await expect(page.getByRole('button', {name:'预览',exact:true})).toHaveCount(0);
  await expect(page.getByText("正在读取操作影响…", {exact:true})).toHaveCount(0);
  await expect(page.getByRole("button", {name:"保存",exact:true})).toBeEnabled();
  // Advance the source after automatic assistance has settled.
  const status = await page.evaluate(async () => {
    const s = await (await fetch("/api/v1/session")).json();
    const id = location.pathname.split("/").at(-1);
    const d = await (await fetch("/api/v1/business-units/" + id)).json();
    const first = d.timeline[0];
    return (
      await fetch(
        `/api/v1/business-units/${id}/records/${first.recordId}/correct`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "X-CSRF-Token": s.csrf,
            "Idempotency-Key": crypto.randomUUID(),
          },
          body: JSON.stringify({
            expectedSourceVersion: d.headSourceVersion,
            expectedRecordRevision: first.revision,
            changes: { nameEn: "Concurrent" },
          }),
        },
      )
    ).status;
  });
  expect(status).toBe(200);
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect(
    page.getByText("资料已更新，请核对最新内容后保存", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("textbox", { name: "中文名称", exact: true }),
  ).toHaveValue("我的输入");
  const detailPath = "/api/v1/business-units/" + new URL(page.url()).pathname.split("/").at(-1);
  const matchReload = (url: URL) => url.pathname === detailPath;
  let releaseReload!: () => void, reloadArrived!: () => void, reloadSent!: () => void;
  const reloadHold = new Promise<void>(r => releaseReload = r);
  const reloadReached = new Promise<void>(r => reloadArrived = r);
  const reloadDone = new Promise<void>(r => reloadSent = r);
  await page.route(matchReload, async route => {
    if (route.request().method() !== "GET") { await route.continue(); return; }
    const response = await route.fetch(); reloadArrived(); await reloadHold;
    await route.fulfill({ response }); reloadSent();
  });
  await page.getByRole("button", { name: "重新核对当前资料", exact: true }).click();
  await reloadReached;
  await expect(page.getByRole("button", { name: "保存", exact: true })).toBeDisabled();
  await expect(page.getByRole("button", { name: "重新核对当前资料", exact: true })).toBeDisabled();
  releaseReload(); await reloadDone; await page.unroute(matchReload);
  await expect(page.getByRole("button", { name: "保存", exact: true })).toBeEnabled();
  await expect(page.getByRole('button', {name:'预览',exact:true})).toHaveCount(0);
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect(
    page.getByRole("button", { name: /^(新增记录|插入记录)$/, exact: true }),
  ).toBeVisible();
  expect((await read(page, "2026-03-01")).selectedRecord.nameEn).toBe("Concurrent");
});
test("301维护请求结束后才能确认，确认后新草稿保持", async ({ page }) => {
  await deferAutomaticConfirmation(page);
  await open(page);
  await create(page);
  let release!: () => void;
  const held = new Promise<void>((r) => (release = r));
  let arrived!: () => void;
  const accepted = new Promise<void>((r) => (arrived = r));
  await page.route("**/api/v1/business-units/*/records", async (route) => {
    await route.fetch();
    arrived();
    await held;
    await route.abort("failed");
  });
  await page.getByRole("button", { name: /^(新增记录|插入记录)$/, exact: true }).click();
  await page
    .getByLabel("生效日期")
    .and(page.locator("input"))
    .fill("2026-03-01");
  await page.getByRole("combobox", { name: "操作原因", exact: true }).click();
  await page.getByRole("option", { name: "名称调整", exact: true }).click();
  await page
    .getByRole("textbox", { name: "中文名称", exact: true })
    .fill("先保存");
  await expect(page.getByRole('button', {name:'预览',exact:true})).toHaveCount(0);
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await accepted;
  await expect(page.getByRole("button", {name:"查看保存结果",exact:true})).toBeDisabled();
  const failed = page.waitForEvent("requestfailed", (r) =>
    r.url().endsWith("/records"),
  );
  release();
  await failed;
  await page.getByRole("button", { name: "查看保存结果", exact: true }).click();
  await expect(
    page.getByRole("button", { name: /^(新增记录|插入记录)$/, exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: /^(新增记录|插入记录)$/, exact: true }).click();
  await page
    .getByRole("textbox", { name: "中文名称", exact: true })
    .fill("下一次输入");
  await page.unroute("**/api/v1/business-units/*/records");
  await expect(
    page.getByRole("textbox", { name: "中文名称", exact: true }),
  ).toHaveValue("下一次输入");
  await expect(
    page.getByText(/暂时无法确认保存结果/, { exact: true }),
  ).toHaveCount(0);
  expect((await read(page, "2026-03-01")).timeline).toHaveLength(2);
});

test("维护名称200码点含扩展汉字、201拒绝与英文显式清空", async ({ page }) => {
  await open(page);
  await create(page);
  const original = (await read(page, "2026-01-01")).headSourceVersion;
  await openMoreActions(page);
  await page.getByRole("menuitem", { name: "更正记录", exact: true }).click();
  const valid = "中".repeat(198) + "𠮷文";
  const invalid = valid + "多";
  await page
    .getByRole("textbox", { name: "中文名称", exact: true })
    .fill(invalid);
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect(
    page.getByRole('button', { name: "名称应为 1 至 200 个字符", exact: true }),
  ).toBeVisible();
  await expect(page.getByRole('textbox', { name: '中文名称', exact: true })).toHaveAttribute('aria-invalid', 'true');
  await expect(
    page.getByRole("textbox", { name: "中文名称", exact: true }),
  ).toBeFocused();
  await expect(
    page.getByRole("textbox", { name: "中文名称", exact: true }),
  ).toHaveValue(invalid);
  await page
    .getByRole("textbox", { name: "中文名称", exact: true })
    .fill(valid);
  await page.getByRole("textbox", { name: "英文名称", exact: true }).fill("");
  await expect(page.getByRole('button', {name:'预览',exact:true})).toHaveCount(0);
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect(
    page.getByRole("button", { name: /^(新增记录|插入记录)$/, exact: true }),
  ).toBeVisible();
  const result = await read(page, "2026-01-01");
  expect(result.selectedRecord.name).toBe(valid);
  expect([...result.selectedRecord.name]).toHaveLength(200);
  expect(result.selectedRecord.nameEn).toBeUndefined();
  expect((await read(page, "2026-01-01", original)).selectedRecord.nameEn).toBe(
    "Alpha",
  );
});

test("正常菜单名称拒绝emoji及NFC创建新增更正", async ({ page }) => {
  await open(page);
  await page.getByRole("link", { name: "新建", exact: true }).click();
  const code = "NAME_" + randomUUID().slice(0, 8);
  await page.getByRole("textbox", { name: "编码", exact: true }).fill(code);
  const name = page.getByRole("textbox", { name: "中文名称", exact: true });
  await name.fill("华东😀事业部");
  await page.getByLabel("生效日期").and(page.locator("input")).fill("2026-01-01");
  await page.getByRole("combobox", { name: "操作原因", exact: true }).click();
  await page.getByRole("option", { name: "新设组织", exact: true }).click();
  await expect(page.getByText("名称最多200个字符", { exact: true }).first()).toBeVisible();
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect(name).toBeFocused();
  await expect(name).toHaveValue("华东😀事业部");
  await expect(page.getByText("名称请使用文字、数字和名称标点，不可包含表情、装饰图形或不合法的不可见字符", { exact: true }).first()).toBeVisible();
  await name.fill("𠮷祥Cafe\u0301研发（一期）");
  await page.getByRole("textbox", { name: "英文名称", exact: true }).fill("Socie\u0301te\u0301 R&D");
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect(page.getByText("已保存。生效日期：", { exact: false })).toBeVisible();
  const original = await read(page, "2026-01-01");
  expect(original.selectedRecord.name).toBe("𠮷祥Café研发（一期）");
  expect(original.selectedRecord.nameEn).toBe("Société R&D");
  await page.getByRole("button", { name: /^(新增记录|插入记录)$/, exact: true }).click();
  await page.getByLabel("生效日期").and(page.locator("input")).fill("2026-03-01");
  await page.getByRole("combobox", { name: "操作原因", exact: true }).click();
  await page.getByRole("option", { name: "名称调整", exact: true }).click();
  const editName = page.getByRole("textbox", { name: "中文名称", exact: true });
  await editName.fill("第1️⃣事业部");
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect(editName).toBeFocused();
  await expect(editName).toHaveValue("第1️⃣事业部");
  await expect(page.getByRole("button", { name: "保存", exact: true })).toBeEnabled();
  await editName.fill("第1研发事业部 / Cafe\u0301");
  await expect(page.getByRole('button', {name:'预览',exact:true})).toHaveCount(0);
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect(page.getByRole("button", { name: /^(新增记录|插入记录)$/, exact: true })).toBeVisible();
  const added = await read(page, "2026-03-01");
  expect(added.selectedRecord.name).toBe("第1研发事业部 / Café");
  await page.getByRole("button", { name: "2026-03-01", exact: true }).click();
  await expect(page.getByRole("button", { name: "2026-03-01", exact: true })).toHaveAttribute("aria-current", "date");
  await openMoreActions(page);
  await page.getByRole("menuitem", { name: "更正记录", exact: true }).click();
  await page.getByRole("textbox", { name: "英文名称", exact: true }).fill("Team 👩‍💻");
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect(page.getByRole("textbox", { name: "英文名称", exact: true })).toBeFocused();
  await page.getByRole("textbox", { name: "英文名称", exact: true }).fill("O’Connor R&D");
  await expect(page.getByRole('button', {name:'预览',exact:true})).toHaveCount(0);
  await page.getByRole("button", { name: "保存", exact: true }).click();
  await expect(page.getByRole("button", { name: /^(新增记录|插入记录)$/, exact: true })).toBeVisible();
  await page.reload();
  await expect(page.getByRole("button", { name: /^(新增记录|插入记录)$/, exact: true })).toBeVisible();
  expect((await read(page, "2026-03-01")).selectedRecord.nameEn).toBe("O’Connor R&D");
  expect((await read(page, "2026-03-01", added.headSourceVersion)).selectedRecord.nameEn).toBe("Société R&D");
  expect((await read(page, "2026-01-01", original.headSourceVersion)).selectedRecord.name).toBe("𠮷祥Café研发（一期）");
});
