import { expect, test } from './fixtures';
import { open } from './bu-support';
import { request } from './dated-object-support';
import { seed308 } from './staffing308-support';
import { seedPosition } from './staffing312-support';
import { quotaAndJobClassification } from './staffing313-support';

test('313 岗位核编后只更正职类，再从岗位核对原采用与额度', async ({ page }) => {
  test.setTimeout(120000);
  await open(page);
  const family = await seed308(page, 'jobfamily');
  const position = await seedPosition(page, 'position', family.id);
  const job = (await request(page, 'GET', '/api/v1/jobs/' + position.job.id)).body;
  const classification = await seed308(page, 'jobfunction', undefined, { date: '1900-01-01' });
  expect(job.selectedRecord.jobFunctionId).not.toBe(classification.id);
  await quotaAndJobClassification(page, position, { id: position.job.id, code: job.code, name: job.selectedRecord.name }, classification);
});
