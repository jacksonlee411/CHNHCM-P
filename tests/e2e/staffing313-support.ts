import { expect, type Page } from '@playwright/test';
import { request } from './dated-object-support';
import { action308, chooseClassification308, click308, namedMenu308 } from './staffing308-support';

type Sample = { id: string; code: string; name: string };

export async function quotaAndJobClassification(page: Page, position: Sample, job: Sample, classification: Sample, legacy = false, snapshot?: (step: 'before' | 'quota' | 'job') => void) {
  const read = async (path: string) => {
    const response = await request(page, 'GET', path);
    expect(response.status).toBe(200);
    return response.body;
  };
  const openObject = async (label: string, resource: string, sample: Sample) => {
    await namedMenu308(page, label, '职务配置', '/settings/staffing/' + resource);
    await page.getByRole('textbox', { name: '搜索编码或名称' }).fill(sample.code);
    await page.getByRole('button', { name: '查询', exact: true }).click();
    await click308(page, page.getByRole('link', { name: sample.name, exact: true }));
    await expect(page.getByRole('heading', { level: 1, name: sample.name, exact: true })).toBeVisible();
  };
  const positionPath = '/api/v1/positions/' + position.id;
  const before = await read(positionPath);
  const original = await read(positionPath + '?sourceVersion=' + before.headSourceVersion);
  const adoption = original.selectedRecord.positionAdoptions.jobId;
  const jobPath = '/api/v1/jobs/' + job.id;
  const oldJob = await read(jobPath);
  if (legacy) expect(oldJob.selectedRecord.jobFunctionName).toBe('未登记职类');

  snapshot?.('before');
  await openObject('岗位', 'positions', position);
  await page.getByRole('tab', { name: '岗位编制' }).click();
  await page.getByRole('button', { name: '核编', exact: true }).click();
  await page.getByLabel('生效日期').and(page.locator('input')).fill('2027-01-01');
  await page.getByRole('spinbutton', { name: /核定编制/ }).fill('2');
  await page.getByLabel('调整原因').and(page.locator('input')).fill('设岗核定两名');
  const savedQuota = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === '/api/v1/staffing-quotas');
  await page.getByRole('button', { name: '保存', exact: true }).click();
  const quotaResponse = await savedQuota;
  expect(quotaResponse.status()).toBe(200);
  const receipt = await quotaResponse.json();
  await expect(page.getByText('设岗核定两名', { exact: true })).toBeVisible();
  const statistics = '/api/v1/staffing-quotas/statistics/positions/' + position.id + '?businessAt=2027-01-01';
  const quota = await read(statistics);
  expect(quota).toMatchObject({ quota: 2, sourceVersion: receipt.sourceVersion, recordId: receipt.selectedRecordId });
  expect((await read(jobPath)).headSourceVersion).toBe(oldJob.headSourceVersion);

  snapshot?.('quota');
  await openObject('职务', 'jobs', job);
  await action308(page, '更正记录');
  await chooseClassification308(page, 'jobFunctionId', classification.name);
  const savedJob = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname.endsWith('/correct'));
  await page.getByRole('button', { name: '保存', exact: true }).click();
  expect((await savedJob).status()).toBe(200);
  await expect(page.locator('.object-form')).toHaveCount(0);
  snapshot?.('job');
  const newJob = await read(jobPath);
  expect(newJob.headSourceVersion).not.toBe(oldJob.headSourceVersion);
  expect(newJob.selectedRecord).toMatchObject({ jobFunctionId: classification.id, jobFamilyId: oldJob.selectedRecord.jobFamilyId });

  await openObject('岗位', 'positions', position);
  await page.getByRole('tab', { name: '变更日志' }).click();
  await page.locator('.change-log-table a[href*="sourceVersion"]').first().click();
  await page.getByRole('button', { name: '来源说明', exact: true }).click();
  await expect(page.locator('.source-explanation')).toContainText(adoption.sourceVersion);
  expect((await read(positionPath)).headSourceVersion).toBe(before.headSourceVersion);
  expect((await read(positionPath + '?sourceVersion=' + before.headSourceVersion)).selectedRecord.positionAdoptions.jobId).toEqual(adoption);
  await page.getByRole('button', { name: '返回当前资料' }).click();
  await page.getByRole('tab', { name: '岗位编制' }).click();
  await expect(page.getByText('设岗核定两名', { exact: true })).toBeVisible();
  expect(await read(statistics)).toMatchObject({ quota: 2, recordId: quota.recordId, revision: quota.revision, sourceVersion: quota.sourceVersion });
}
