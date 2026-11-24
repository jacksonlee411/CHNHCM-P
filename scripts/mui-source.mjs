import { createHash } from 'node:crypto';
import { readFileSync, readdirSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { resolve, relative } from 'node:path';

const root = fileURLToPath(new URL('../', import.meta.url));
const sourceRoot = resolve(root, 'packages/mui-x-data-grid');

/** Explicit provenance review; ordinary development and production builds do not call it. */
export function verifyMuiSource() {
  const manifest = JSON.parse(readFileSync(resolve(sourceRoot, 'upstream.json'), 'utf8'));
  const digest = path => createHash('sha256').update(readFileSync(path)).digest('hex');
  if (digest(resolve(root, manifest.archive)) !== manifest.archive_sha256) {
    throw new Error('MUI Community source archive differs from its recorded upstream input');
  }
  if (digest(resolve(root, manifest.types_archive)) !== manifest.types_archive_sha256) {
    throw new Error('MUI Community declaration archive differs from its recorded release input');
  }
  for (const [file, expected] of Object.entries({ ...manifest.files_sha256, ...manifest.declarations_sha256 })) {
    if (digest(resolve(sourceRoot, file)) !== expected) {
      throw new Error(`MUI Community upstream file changed: ${file}`);
    }
  }
  const actualSource = readdirSync(resolve(sourceRoot, 'src'), { recursive: true, withFileTypes: true })
    .filter(entry => entry.isFile()).map(entry => relative(sourceRoot, resolve(entry.parentPath, entry.name)));
  if (actualSource.some(file => !Object.hasOwn(manifest.files_sha256, file))) {
    throw new Error('MUI Community source includes an unrecorded upstream file');
  }
  return { version: manifest.version, commit: manifest.commit, sourceFiles: Object.keys(manifest.files_sha256).length, declarationFiles: Object.keys(manifest.declarations_sha256).length, license: manifest.license };
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  console.log(JSON.stringify(verifyMuiSource()));
}
