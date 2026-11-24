import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { localRoot } from '../../scripts/local-runtime.mjs';

// Listing tests must not require a database or read any runtime credential.
// The first actual use keeps the same profile lookup and fail-closed behavior.
export function lazyCredentials() {
  let value;
  const load = () => value ??= JSON.parse(readFileSync(execFileSync('python3',
    ['scripts/local-bu.py', 'credentials-path'],
    { cwd: localRoot, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).trim(), 'utf8'));
  return new Proxy({}, {
    get: (_, key) => load()[key],
    ownKeys: () => Reflect.ownKeys(load()),
    getOwnPropertyDescriptor: (_, key) => Object.getOwnPropertyDescriptor(load(), key),
  });
}
