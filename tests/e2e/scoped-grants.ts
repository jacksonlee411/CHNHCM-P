import { randomUUID } from 'node:crypto';
import { expect, type Page } from './fixtures';
import { withPolicy } from './grants';

type Binding = {
  scope: 'position' | 'bu' | 'department';
  target: string;
  actions: string[];
  read: string[];
  write?: string[];
};
type Validity = { from: Date; until: Date };
type FixtureRole = { issue: (source?: 'duty' | 'proxy', validity?: Validity) => () => void };
type Publish = (binding: Binding) => FixtureRole;

// The fixture owns only the temporary actor's test assignments. Browser requests
// and the Go business transactions still use the real versions, validity and revisions.
export async function withScopedGrants(page: Page, run: (publish: Publish) => Promise<void>) {
  await withPolicy(page, async (where, mutate) => {
    const active = new Set<() => void>();
    const actor = `FROM access.policy_guard WHERE ${where}`;
    const words = (values: string[]) => {
      for (const value of values) expect(value).toMatch(/^[a-zA-Z][a-zA-Z0-9.]*$/);
      return `ARRAY[${values.map(value => `'${value}'`).join(', ')}]::text[]`;
    };
    const publish: Publish = ({ scope, target, actions, read, write = [] }) => {
      expect(target).toMatch(/^[0-9a-f-]{36}$/);
      expect(actions.length).toBeGreaterThan(0);
      const version = randomUUID();
      mutate(`
        INSERT INTO access.policy_version(instance_id, kind, version_id, definition_id, version_no, actions)
          SELECT instance_id, 'role', '${version}', '${version}', 1, ${words(actions)} ${actor};
        INSERT INTO access.policy_version(instance_id, kind, version_id, definition_id, version_no, scope_kind, scope_id)
          SELECT instance_id, 'scope', '${version}', '${version}', 1, '${scope}', '${target}' ${actor};
        INSERT INTO access.policy_version(instance_id, kind, version_id, definition_id, version_no, read_fields, write_fields)
          SELECT instance_id, 'fields', '${version}', '${version}', 1, ${words(read)}, ${words(write)} ${actor}
      `);
      return {
        issue(source = 'duty', validity) {
          const id = randomUUID();
          const start = validity ? `'${validity.from.toISOString()}'::timestamptz` : 'NULL';
          const end = validity ? `'${validity.until.toISOString()}'::timestamptz` : 'NULL';
          const purpose = actions[0].split('.')[0];
          mutate(`
            INSERT INTO access.grant_assignment(instance_id, grant_id, principal_id, source, purpose, valid_from, valid_until, revision)
              SELECT instance_id, '${id}', principal_id, '${source}', '${purpose}', ${start}, ${end}, 1 ${actor};
            INSERT INTO access.grant_revision(instance_id, grant_id, revision, role_id, scope_id, fields_id, operation)
              SELECT instance_id, '${id}', 1, '${version}', '${version}', '${version}', 'issue' ${actor}
          `);
          const revoke = () => {
            // A role can expose several actions through the fixture view. End the assignment once.
            mutate(`DELETE FROM access.subject_grant WHERE ${where} AND grant_id='${id}' AND action='${actions[0]}'`);
            active.delete(revoke);
          };
          active.add(revoke);
          return revoke;
        },
      };
    };
    try {
      mutate('DELETE FROM access.subject_grant WHERE ' + where);
      await run(publish);
    } finally {
      for (const revoke of active) revoke();
    }
  });
}
