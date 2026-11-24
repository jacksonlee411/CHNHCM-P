// Full-background, real-menu observations. Never intercepts business responses
// or stores a browser session, trace, credential or request header.
import { chromium, expect as baseExpect } from "@playwright/test";
import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, writeFileSync } from "node:fs";
import { dirname } from "node:path";
import { localConnection, verifyLocalTarget } from "../../scripts/local-runtime.mjs";

if (process.env.CHNHCM_LOCAL_PROFILE !== "r6-capacity-b-1106") throw new Error("dedicated temporary capacity profile required");
const output = process.argv[2];
if (!output || existsSync(output)) throw new Error("new evidence path required");
const connection = localConnection();
verifyLocalTarget();
const expect = baseExpect.configure({ timeout: 20_000 });
mkdirSync(dirname(output), { recursive: true });
for (const key of ["HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"]) delete process.env[key];
const browser = await chromium.launch(process.env.CHNHCM_BROWSER_EXECUTABLE ? { executablePath: process.env.CHNHCM_BROWSER_EXECUTABLE } : {});
const result = { scope: "two roles; real department application menu; four individual observations, not a page p95", connection, observations: [], passed: true };
try {
  for (const login of ["admin", "reader"]) {
    for (const cache of ["cold", "warm"]) {
      const context = await browser.newContext({ viewport: { width: 1440, height: 900 }, timezoneId: "Asia/Shanghai" });
      const page = await context.newPage();
      page.setDefaultTimeout(20_000);
      const row = { login, cache, requests: [], passed: false };
      result.observations.push(row);
      try {
        await page.goto(connection.webOrigin);
        await page.getByLabel("登录名").fill(login);
        await page.getByLabel("密码").fill("admin1234567");
        await page.getByRole("button", { name: "登录", exact: true }).click();
        await page.getByRole("button", { name: "切换应用", exact: true }).click();
        await page.getByRole("menuitem", { name: "设置", exact: true }).click();
        await page.getByRole("button", { name: "切换应用", exact: true }).click();
        if (cache === "warm") {
          // Prime through the same natural page entry; preserve its result
          // separately. A failed entry is not silently retried until green.
          const prepStart = performance.now();
          const listResponse = page.waitForResponse(r => new URL(r.url()).pathname === "/api/v1/departments");
          const contextResponse = page.waitForResponse(r => new URL(r.url()).pathname === "/api/v1/departments/context");
          await page.getByRole("menuitem", { name: "部门", exact: true }).click();
          const prep = await Promise.all([listResponse, contextResponse]);
          row.warmup = { menuToResponsesMs: performance.now() - prepStart, requests: await Promise.all(prep.map(async r => ({ path: new URL(r.url()).pathname, status: r.status(), result: await r.json() }))) };
          if (prep.some(r => r.status() !== 200)) throw new Error("natural menu warm-up failed");
          await expect(page.getByRole("row").filter({ hasText: /R6_/ }).first()).toBeVisible();
          await page.getByRole("button", { name: "切换应用", exact: true }).click();
          await page.getByRole("menuitem", { name: "设置", exact: true }).click();
          await expect(page.getByRole("heading", { name: "业务单元", exact: true })).toBeVisible();
          await expect(page.getByRole("row").filter({ hasText: /R6_/ }).first()).toBeVisible();
          await page.getByRole("button", { name: "切换应用", exact: true }).click();
        }
        if (cache === "cold") {
          row.coldReset = JSON.parse(execFileSync("python3", ["scripts/r6-cold-cache.py"], { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] }));
          if (!row.coldReset.passed) throw new Error("cold reset failed");
        }
        // Login and menu preparation precede the cold reset. No department
        // context/list/detail request may precede this click after the reset.
        const start = performance.now();
        const times = new WeakMap();
        const bodies = [];
        page.on("request", request => {
          if (new URL(request.url()).pathname.startsWith("/api/v1/")) times.set(request, performance.now());
        });
        page.on("response", response => {
          const request = response.request();
          const path = new URL(response.url()).pathname;
          if (!times.has(request)) return;
          const item = { path, method: request.method(), startMs: times.get(request) - start, responseMs: performance.now() - start, status: response.status() };
          row.requests.push(item);
          if (path === "/api/v1/departments" || path === "/api/v1/departments/context") {
            bodies.push(response.json().then(value => {
              item.result = value;
            }).catch(() => { item.decodeError = true; }));
          }
        });
        await page.getByRole("menuitem", { name: "部门", exact: true }).click();
        await expect(page.getByRole("heading", { name: "部门", exact: true })).toBeVisible();
        await expect(page.getByRole("row").filter({ hasText: /R6_/ }).first()).toBeVisible();
        await expect(page.locator(".MuiAlert-standardError")).toHaveCount(0);
        row.menuToUsableMs = performance.now() - start;
        await Promise.all(bodies);
        const list = row.requests.find(v => v.path === "/api/v1/departments")?.result;
        const entry = row.requests.find(v => v.path === "/api/v1/departments/context")?.result;
        if (!list || list.count < 22_000 || list.items.length !== 20 || !entry || entry.capabilities.create !== (login === "admin")) throw new Error("full-background rows/context capability mismatch");
        if (row.requests.some(v => v.status >= 400 || v.decodeError)) throw new Error("business request failed");
        if (login === "admin") {
          await page.getByRole("link", { name: "新建", exact: true }).click();
          await expect(page.getByRole("textbox", { name: "部门名称", exact: true })).toBeVisible();
          await expect(page.getByRole("combobox", { name: "管理业务单元", exact: true })).toBeVisible();
          await expect(page.locator("#department-effectiveDate")).toHaveValue(entry.currentDate);
          row.createForm = { date: await page.locator("#department-effectiveDate").inputValue(), nameVisible: true, managementBUVisible: true };
        } else {
          await expect(page.getByRole("link", { name: "新建", exact: true })).toHaveCount(0);
        }
        await Promise.all(bodies);
        await page.screenshot({ path: output.replace(/\.json$/, `-${login}-${cache}.png`), fullPage: true, mask: [page.locator('input[type="password"]')] });
        row.passed = true;
      } catch (error) {
        row.error = String(error.message).split("Call log:")[0].slice(0, 1200);
        result.passed = false;
        await page.screenshot({ path: output.replace(/\.json$/, `-${login}-${cache}-failed.png`), fullPage: true, mask: [page.locator('input[type="password"]')] }).catch(() => {});
      } finally {
        await context.close();
        writeFileSync(output, JSON.stringify(result, null, 2) + "\n");
      }
    }
  }
} finally {
  await browser.close();
  writeFileSync(output, JSON.stringify(result, null, 2) + "\n");
}
if (!result.passed) process.exitCode = 1;
