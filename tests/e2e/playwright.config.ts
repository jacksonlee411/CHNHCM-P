import { defineConfig, devices } from "@playwright/test";
import { localConnection } from "../../scripts/local-runtime.mjs";

// Playwright 1.55's automatic error context includes password input values.
// Keep masked failure screenshots, but never persist that textual snapshot.
process.env.PLAYWRIGHT_NO_COPY_PROMPT = "1";

// This suite only visits loopback. A proxy's HTTP 400 must not be mistaken
// for an already running local server by Playwright's availability probe.
for (const key of [
  "HTTP_PROXY",
  "HTTPS_PROXY",
  "ALL_PROXY",
  "http_proxy",
  "https_proxy",
  "all_proxy",
]) {
  delete process.env[key];
}

const connection = localConnection();

export default defineConfig({
  globalSetup: "./runtime-setup.ts",
  testDir: ".",
  testMatch: ["access-management.spec.ts", "personnel.spec.ts", "hire400.spec.ts", "first-hire.spec.ts", "staffing313.spec.ts", "staffing315.spec.ts", "staffing315-allocation.spec.ts", "staffing314.spec.ts", "staffing314-zoom.spec.ts", "staffing312-hierarchy.spec.ts", "staffing312.spec.ts", "staffing312-zoom.spec.ts", "staffing310-permissions.spec.ts", "staffing310.spec.ts", "staffing310-zoom.spec.ts", "staffing309.spec.ts", "staffing309-zoom.spec.ts", "staffing308.spec.ts", "staffing308-zoom.spec.ts", "staffing307.spec.ts", "legal-entity206.spec.ts", "legal-entity206-layout.spec.ts", "enterprise304.spec.ts", "enterprise304-zoom.spec.ts", "primary303.spec.ts", "primary303-zoom.spec.ts", "location205-navigation.spec.ts", "navigation302.spec.ts", "geography205-joint.spec.ts", "location204-layout.spec.ts", "location204.spec.ts", "feedback301.spec.ts", "department-tree300.spec.ts", "geoarea203-layout.spec.ts", "geoarea203.spec.ts", "country200.spec.ts", "default-date202.spec.ts", "bu-coverage133.spec.ts", "department-path133.spec.ts", "department-selection-menu.spec.ts", "department-selection-races.spec.ts", "dated-assistance-focus.spec.ts", "department-reference-projection.spec.ts", "department-authorization-followup.spec.ts", "department-review-fixes.spec.ts", "department-layout.spec.ts", "department-selection.spec.ts", "department-acceptance.spec.ts", "department-journeys.spec.ts", "dated-workflow.spec.ts", "operation-review-fixes.spec.ts", "operation-permissions.spec.ts", "operation-summary.spec.ts", "current-maintenance.spec.ts", "query-selection.spec.ts", "date-format.spec.ts", "column-filter.spec.ts", "community-list.spec.ts", "first-record-maintenance.spec.ts", "correction-coverage.spec.ts", "positioned-addition.spec.ts", "unified-correction.spec.ts", "direct-save.spec.ts", "bu.spec.ts", "experience.spec.ts", "ui-improvements.spec.ts", "timeline.spec.ts", "detail-actions.spec.ts", "lifecycle.spec.ts", "delete.spec.ts", "redate.spec.ts", "lifecycle-pair.spec.ts", "t07-journeys.spec.ts", "t07-query.spec.ts", "list-return-notice.spec.ts"],
  // The singleton lifecycle scenario needs an empty organization. Run it before
  // dependent suites create BU/department/location references, including CI shards.
  projects: [
    { name: "enterprise-maintenance", testMatch: "enterprise304.spec.ts" },
    { name: "chromium", testIgnore: "enterprise304.spec.ts", dependencies: ["enterprise-maintenance"] },
  ],
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: "list",
  outputDir: "../../.local/playwright-results",
  use: {
    ...devices["Desktop Chrome"],
    viewport: { width: 1440, height: 900 },
    baseURL: connection.webOrigin,
    launchOptions: process.env.CHNHCM_BROWSER_EXECUTABLE
      ? { executablePath: process.env.CHNHCM_BROWSER_EXECUTABLE }
      : {},
    // fixtures.ts captures failures with login fields and the account name masked.
    screenshot: "off",
    // Authentication cookies and CSRF values must not be persisted inside this repository.
    trace: "off",
  },
  webServer: {
    command: "corepack pnpm --filter @chnhcm/web dev",
    cwd: "../..",
    url: connection.webOrigin,
    reuseExistingServer: true,
    timeout: 30_000,
  },
});
