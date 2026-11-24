import { execFileSync } from 'node:child_process';
import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { localRoot, localConnection, verifyLocalTarget } from './local-runtime.mjs';

// Keep the target, credentials, evidence baseline and output in one checkout,
// even when invoked by absolute path from another worktree or an IDE directory.
export async function comparisonContext() {
  const referenceURL = process.env.CHNHCM_REFERENCE_URL;
  if (!referenceURL) throw new Error('Set CHNHCM_REFERENCE_URL to the pinned CubeBox preview.');
  const targetURL = localConnection().webOrigin;
  verifyLocalTarget(targetURL);
  const options = { cwd: localRoot, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] };
  const credentialPath = execFileSync('python3', ['scripts/local-bu.py', 'credentials-path'], options).trim();
  const credentials = JSON.parse(await readFile(credentialPath, 'utf8'));
  const targetBase = execFileSync('git', ['rev-parse', 'HEAD'], options).trim();
  const output = resolve(localRoot, process.env.CHNHCM_COMPARISON_OUTPUT ?? '.local/ui-migration-019/comparison');
  return { referenceURL, targetURL, output, credentials, targetBase };
}
