import { Button } from '@mui/material';
import { RecordPager } from '@chnhcm/ui';
import { useNavigate } from 'react-router-dom';
import { useRead } from '../personnel/api';
import {
  accessPath,
  read,
  type Area,
  type Session,
  type View,
} from './management';

function ids(area: Area, value?: View) {
  return area === 'roles'
    ? (value?.versions?.flatMap((d) => (d.id ? [d.id] : [])) ?? [])
    : (value?.accounts?.map((a) => a.id) ?? []);
}

export function SecurityRecordPager({
  area,
  id,
  params,
  session,
  disabled,
}: {
  area: Area;
  id: string;
  params: URLSearchParams;
  session: Session;
  disabled: boolean;
}) {
  const navigate = useNavigate();
  const size = Number(params.get('size')) || 20;
  const query = {
    q: params.get('q') ?? '',
    owner: params.get('owner') ?? '',
    state: params.get('state') ?? '',
    ...(area === 'roles' && params.get('versionKind')
      ? { versionKind: params.get('versionKind')! }
      : {}),
    pageSize: size,
  };
  const key =
    session.instanceId + session.subjectId + area + JSON.stringify(query);
  const page = Number(params.get('page')) || 0;
  const load = (n: number) => read(area, { ...query, page: n });
  const current = useRead(key + ':' + page, () => load(page), 'fresh');
  const rows = ids(area, current.data),
    pos = rows.indexOf(id),
    actual = current.data?.page ?? page,
    total = current.data?.total ?? 0;
  const previous = useRead(
    pos === 0 && actual > 0 ? key + ':' + (actual - 1) : undefined,
    () => load(actual - 1),
    'fresh',
  );
  const next = useRead(
    pos >= 0 && pos === rows.length - 1 && (actual + 1) * size < total
      ? key + ':' + (actual + 1)
      : undefined,
    () => load(actual + 1),
    'fresh',
  );
  const before = pos === 0 ? ids(area, previous.data).at(-1) : rows[pos - 1];
  const after =
    pos === rows.length - 1 ? ids(area, next.data)[0] : rows[pos + 1];
  const failed = [current, previous, next].find((result) => result.error);
  if (failed)
    return (
      <Button size="small" onClick={failed.refresh}>
        重试对象导航
      </Button>
    );
  if (pos < 0) return null;
  return (
    <RecordPager
      previous={!!before}
      next={!!after}
      counter={{ position: actual * size + pos + 1, count: total }}
      disabled={disabled}
      onChange={(direction) => {
        const previous = direction === 'previous',
          target = previous ? before : after;
        if (!target || disabled) return;
        const list = new URLSearchParams(params);
        list.delete('selected');
        list.set('fromList', '1');
        list.set(
          'page',
          String(
            actual +
              (previous && pos === 0
                ? -1
                : !previous && pos === rows.length - 1
                  ? 1
                  : 0),
          ),
        );
        navigate(accessPath + '/' + area + '/' + target + '?' + list);
      }}
    />
  );
}
