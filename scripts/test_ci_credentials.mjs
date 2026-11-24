import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, writeFileSync, readFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { execFileSync } from 'node:child_process';

test('discovery does not read credentials; real access loads once and preserves fields', () => {
  const dir = mkdtempSync(join(tmpdir(), 'chnhcm-credentials-test-'));
  try {
    writeFileSync(join(dir, 'fixture.json'), JSON.stringify({ login: 'fixture-only', instance: 'fixture-instance' }));
    writeFileSync(join(dir, 'python3'), '#!/bin/sh\nprintf x >> "$CI_CREDENTIAL_FIXTURE/count"\nprintf "%s/fixture.json\\n" "$CI_CREDENTIAL_FIXTURE"\n', { mode: 0o700 });
    const helper = new URL('../tests/e2e/credentials.mjs', import.meta.url).href;
    const script = `import { lazyCredentials } from ${JSON.stringify(helper)};
      import assert from 'node:assert/strict';
      import { existsSync } from 'node:fs';
      const value = lazyCredentials();
      assert.equal(existsSync(process.env.CI_CREDENTIAL_FIXTURE + '/count'), false);
      assert.equal(value.login, 'fixture-only');
      assert.deepEqual({...value}, {login: 'fixture-only', instance: 'fixture-instance'});`;
    const env = { ...process.env, PATH: dir + ':' + process.env.PATH, CI_CREDENTIAL_FIXTURE: dir };
    execFileSync(process.execPath, ['--input-type=module', '-e', script], { env, stdio: 'pipe' });
    assert.equal(readFileSync(join(dir, 'count'), 'utf8'), 'x');
    writeFileSync(join(dir, 'python3'), '#!/bin/sh\nexit 1\n', { mode: 0o700 });
    const failure = `import { lazyCredentials } from ${JSON.stringify(helper)};
      import assert from 'node:assert/strict'; const value = lazyCredentials(); assert.throws(() => value.login);`;
    execFileSync(process.execPath, ['--input-type=module', '-e', failure], { env, stdio: 'pipe' });
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});
