import { useLayoutEffect, useRef, useState } from 'react';
import type { Models } from '@chnhcm/api-client';
import * as person from '../personnel/api';

type Selection = NonNullable<Models['PersonnelDirectoryQuery']['selection']>;
type Intent = { scope: string; all: boolean; ids: string[] };
export function usePersonnelDirectory(
  session: person.Principal,
  params: URLSearchParams,
  locationKey: string,
) {
  const query = person.directoryQuery(params);
  const scope = JSON.stringify([
    session.instanceId,
    session.subjectId,
    query.ref,
    query.primary,
    query.department,
    query.directOnly,
  ]);
  const page = JSON.stringify([scope, locationKey, query.page, query.pageSize, query.sort]);
  const [intent, setIntent] = useState<Intent>();
  const [notice, setNotice] = useState('');
  const current = intent?.scope === scope ? intent : undefined;
  const version = useRef('');
  const previous = useRef<{ page: string; data: Models['PersonnelDirectory'] } | undefined>(undefined);
  const input: Selection | undefined = current && {
    all: current.all,
    ids: current.ids,
    version: version.current,
  };
  const listing = person.useRead(
    JSON.stringify([page, current]),
    () => person.directory(session, params, undefined, input),
    'fresh',
  );
  const checked = listing.data?.selection;
  useLayoutEffect(() => {
    if (intent && !current) {
      setIntent(undefined);
      version.current = '';
    }
    if (listing.error) {
      previous.current = undefined;
    }
    if (listing.data) previous.current = { page, data: listing.data };
    if (checked && current) {
      version.current = checked.version;
      if (checked.changed || !checked.count) {
        setIntent(undefined);
        version.current = '';
        if (checked.changed) setNotice('查询结果已变化，请重新选择。');
      }
    }
  }, [page, scope, current, intent, listing.data, listing.error, checked]);
  useLayoutEffect(() => {
    setNotice('');
  }, [scope]);
  // Retain the grid for a checkbox interaction on the same page. Only the
  // response to this selection can confirm the count or enable further edits.
  const data =
    listing.data ??
    (current && previous.current?.page === page && !listing.error ? previous.current.data : undefined);
  const clear = () => {
    setIntent(undefined);
    version.current = '';
    setNotice('');
  };
  const ids = new Set<string>();
  const stored = new Set(current?.ids);
  for (const row of data?.rows ?? []) if (current && stored.has(row.id) !== current.all) ids.add(row.id);
  const total = data?.total ?? 0,
    count = checked?.count ?? 0;
  return {
    listing,
    data,
    notice,
    dismiss: () => setNotice(''),
    selecting: !!current,
    selection: {
      ids,
      count,
      checked: total > 0 && count === total,
      mixed: count > 0 && count < total,
      total,
      disabled: !listing.data || !!checked?.changed,
      clear,
      all: () => {
        setNotice('');
        setIntent({ scope, all: true, ids: [] });
      },
      onChange(next: Set<string>) {
        const all = current?.all ?? false;
        for (const row of data?.rows ?? []) {
          if (next.has(row.id) !== all) stored.add(row.id);
          else stored.delete(row.id);
        }
        if (!all && stored.size === 0) {
          clear();
          return;
        }
        setNotice('');
        setIntent({ scope, all, ids: [...stored] });
      },
    },
  };
}
