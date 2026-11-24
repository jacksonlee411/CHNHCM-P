import { test } from 'node:test';
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { copyFileSync, existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const scripts = fileURLToPath(new URL('.', import.meta.url));
function fixture(t) {
  const directory = mkdtempSync(join(tmpdir(), 'chnhcm-comparison-test-'));
  t.after(() => rmSync(directory, { recursive: true, force: true }));
  const checkouts = {};
  mkdirSync(join(directory, 'private'), { mode: 0o700 });
  for (const owner of ['a', 'b']) {
    const root = join(directory, owner);
    mkdirSync(join(root, 'scripts'), { recursive: true });
    const credentialPath = join(directory, 'private', owner + '.json');
    // Synthetic markers only. No real instance or account is contacted.
    writeFileSync(credentialPath, JSON.stringify({ reviewOwner: owner }), { mode: 0o600 });
    writeFileSync(join(root, 'fixture.json'), JSON.stringify({
      webOrigin: owner === 'a' ? 'http://127.0.0.1:49380' : 'http://127.0.0.1:49381', credentialPath,
    }));
    writeFileSync(join(root, 'scripts/local-bu.py'), `
from pathlib import Path
import json, sys
root = Path(__file__).resolve().parents[1]
data = json.loads((root / 'fixture.json').read_text())
command = sys.argv[1]
with (root / 'calls.txt').open('a') as log:
    log.write(command + '\\n')
if command == 'connection':
    print(json.dumps({'webOrigin': data['webOrigin']}))
elif command == 'verify':
    if (root / 'reject').exists() or sys.argv[2:] != ['--origin', data['webOrigin']]:
        raise SystemExit('Synthetic identity mismatch')
elif command == 'credentials-path':
    print(data['credentialPath'])
else:
    raise SystemExit('Unexpected fixture command')
`);
    const options = { cwd: root, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] };
    execFileSync('git', ['init', '-q'], options);
    execFileSync('git', ['-c', 'user.name=T01 Test', '-c', 'user.email=t01@example.invalid',
      '-c', 'commit.gpgsign=false', '-c', 'core.hooksPath=/dev/null', 'commit', '--allow-empty', '-qm', owner], options);
    checkouts[owner] = { root, head: execFileSync('git', ['rev-parse', 'HEAD'], options).trim() };
  }
  for (const name of ['comparison-context.mjs', 'local-runtime.mjs']) {
    copyFileSync(join(scripts, name), join(checkouts.b.root, 'scripts', name));
  }
  return { directory, ...checkouts };
}

function prepareFromForeignDirectory(f, output) {
  const moduleURL = pathToFileURL(join(f.b.root, 'scripts/comparison-context.mjs')).href;
  const code = `
    const { comparisonContext } = await import(${JSON.stringify(moduleURL)});
    try { console.log(JSON.stringify({ context: await comparisonContext(), cwd: process.cwd() })); }
    catch (error) { console.log(JSON.stringify({ error: error.message })); }
  `;
  const env = { ...process.env, CHNHCM_REFERENCE_URL: 'http://127.0.0.1:49382' };
  delete env.CHNHCM_COMPARISON_OUTPUT;
  if (output !== undefined) env.CHNHCM_COMPARISON_OUTPUT = output;
  return JSON.parse(execFileSync(process.execPath, ['--input-type=module', '-e', code], {
    cwd: f.a.root, encoding: 'utf8', env, stdio: ['ignore', 'pipe', 'pipe'],
  }));
}

test('cross-worktree invocation keeps target, credential owner, Git baseline and default output together', t => {
  const f = fixture(t);
  const { context, cwd } = prepareFromForeignDirectory(f);
  assert.equal(context.targetURL, 'http://127.0.0.1:49381');
  assert.deepEqual(context.credentials, { reviewOwner: 'b' });
  assert.equal(context.targetBase, f.b.head);
  assert.notEqual(context.targetBase, f.a.head);
  assert.equal(context.output, join(f.b.root, '.local/ui-migration-019/comparison'));
  assert.equal(cwd, f.a.root);
  assert.equal(existsSync(join(f.a.root, 'calls.txt')), false);
  assert.equal(existsSync(join(f.a.root, '.local')), false);
  assert.equal(readFileSync(join(f.b.root, 'calls.txt'), 'utf8'), 'connection\nverify\ncredentials-path\n');
});

test('relative comparison output is rooted in the script checkout', t => {
  const f = fixture(t);
  assert.equal(prepareFromForeignDirectory(f, 'evidence/new').context.output, resolve(f.b.root, 'evidence/new'));
});

test('explicit absolute output is preserved', t => {
  const f = fixture(t), output = join(f.directory, 'chosen-output');
  assert.equal(prepareFromForeignDirectory(f, output).context.output, output);
});

test('a rejected target stops before credential lookup or output creation', t => {
  const f = fixture(t);
  writeFileSync(join(f.b.root, 'reject'), '');
  const result = prepareFromForeignDirectory(f);
  assert.match(result.error, /Synthetic identity mismatch/);
  assert.equal(result.context, undefined);
  assert.equal(readFileSync(join(f.b.root, 'calls.txt'), 'utf8'), 'connection\nverify\n');
  assert.equal(existsSync(join(f.a.root, 'calls.txt')), false);
  assert.equal(existsSync(join(f.b.root, '.local')), false);
});
