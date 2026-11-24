import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { expect, type Page } from './fixtures';

// Export only this synthetic test subject's grant shape. Credentials, session
// tokens and database URLs remain in the existing protected runtime directory.
export async function grants308(page: Page, role: string) {
  const path = execFileSync('python3', ['scripts/local-bu.py', 'credentials-path'], { encoding: 'utf8' }).trim();
  const config = JSON.parse(readFileSync(path, 'utf8'));
  const connection = JSON.parse(execFileSync('python3', ['scripts/local-bu.py', 'connection'], { encoding: 'utf8' }));
  const container = execFileSync(
    'docker',
    [
      'ps',
      '-q',
      '--filter',
      'label=com.docker.compose.project=' + connection.project,
      '--filter',
      'label=com.docker.compose.service=postgres',
    ],
    { encoding: 'utf8' },
  ).trim();
  expect(container).toMatch(/^[0-9a-f]+$/);
  const subject = await page.evaluate(async () => (await (await fetch('/api/v1/session')).json()).subjectId);
  for (const id of [subject, config.instance]) expect(id).toMatch(/^[0-9a-f-]{36}$/);
  const input = `SELECT coalesce(json_agg(json_build_object('scopeKind',scope_kind,'action',action,'readFields',read_fields,'writeFields',write_fields) ORDER BY action),'[]') FROM access.subject_grant WHERE instance_id='${config.instance}' AND principal_id='${subject}' AND (action LIKE 'job.%' OR action LIKE 'jobfunction.%' OR action LIKE 'jobfamily.%' OR action LIKE 'grade.%' OR action LIKE 'gradeladder.%');`;
  const result = execFileSync(
    'docker',
    ['exec', '-i', container, 'psql', '-U', 'postgres', '-d', 'chnhcm', '-X', '-Atq', '-v', 'ON_ERROR_STOP=1'],
    { input, encoding: 'utf8' },
  ).trim();
  return { role, grants: JSON.parse(result) };
}
