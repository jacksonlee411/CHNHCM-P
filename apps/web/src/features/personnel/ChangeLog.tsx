import { useCallback, useEffect, useLayoutEffect, useRef, useState, type MouseEvent } from 'react';
import {
  Alert,
  Box,
  Button,
  IconButton,
  Menu,
  MenuItem,
  Stack,
  Table,
  TableCell,
  TableContainer,
  TableHead,
  TablePagination,
  TableRow,
  Typography,
} from '@mui/material';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import { Link, useLocation } from 'react-router-dom';
import { ApiError, sessionGuard, type Models } from '@chnhcm/api-client';
import { ColumnFilterDialog, type FilterColumn, type FilterRequest } from '../../shared/dated/ColumnFilter';
import { DifferenceCells, OperationCells, OperationRows } from '../../shared/dated/OperationLog';
import * as person from './api';

export const actionLabel: Record<string, string> = {
  register: '登记人员',
  correct: '更正资料',
  renumber: '改号',
};
const fields: Record<string, string> = {
  name: '姓名',
  number: '人员编号',
  document: '主要证件',
  phone: '电话',
  email: '电子邮箱',
};
type Query = Models['PersonChangeQuery'];
type Page = Models['PersonChangePage'];
type Group = Models['PersonChangeGroup'];
type Diff = Models['PersonDifference'];

export type LogPosition = { query: Query; focus: string; top: number; left: number };
function scroller() {
  const node = document.querySelector<HTMLElement>('[data-page-scroll]');
  return node && ['auto', 'scroll'].includes(getComputedStyle(node).overflowY)
    ? node
    : document.scrollingElement;
}

function GroupRows({
  group,
  id,
  href,
  visit,
  remember,
  recheck,
}: {
  group: Group;
  id: string;
  href: string;
  visit: string;
  remember: (e: MouseEvent<HTMLAnchorElement>) => void;
  recheck: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [full, setFull] = useState<Diff[]>();
  const [failure, setFailure] = useState('');
  const [retry, setRetry] = useState(0);
  useEffect(() => {
    if (!open) return;
    const controller = new AbortController(),
      same = sessionGuard();
    const current = () => !controller.signal.aborted && same();
    person
      .differences(id, group.sourceVersion, controller.signal)
      .then((diffs) => {
        if (!current()) return;
        // An explicit read can observe a field revocation before the page is refreshed.
        if (
          diffs.length !== group.diffs.length ||
          diffs.some(
            (diff, i) =>
              diff.fieldKey !== group.diffs[i].fieldKey ||
              diff.itemId !== group.diffs[i].itemId ||
              diff.canReveal !== group.diffs[i].canReveal,
          )
        ) {
          recheck();
        } else setFull(diffs);
      })
      .catch((e) => {
        if (!current()) return;
        if (e instanceof ApiError && [401, 403, 404].includes(e.status)) recheck();
        else setFailure(person.message(e));
      });
    return () => controller.abort();
  }, [open, id, group, recheck, retry]);
  const shown = open && full ? full : group.diffs;
  const rows: (Diff | undefined)[] = shown.length ? shown : [undefined];
  const source = group.sourceVersion;
  const canReveal = group.diffs.some((diff) => diff.canReveal);
  return (
    <OperationRows
      source={source}
      previous={group.previousSourceVersion}
      recordedAt={group.recordedAt}
      columns={7}
    >
      {(sourceButton) => (
        <>
          {rows.map((diff, index) => (
            <TableRow key={diff ? diff.fieldKey + (diff.itemId ?? '') : 'empty'}>
              {index === 0 && (
                <OperationCells
                  span={rows.length}
                  actionLabel="办理动作"
                  recordedAt={group.recordedAt}
                  operator={group.operator}
                  action={actionLabel[group.action]}
                />
              )}
              <DifferenceCells
                difference={
                  diff && {
                    field: diff.field,
                    before: diff.before || '未登记',
                    after: diff.after || '已移除',
                  }
                }
              />
              {index === 0 && (
                <TableCell rowSpan={rows.length} data-label="查看">
                  <Stack alignItems="flex-start" spacing={1}>
                    <Link id={'change-' + source} to={href} state={{ logVisit: visit }} onClick={remember}>
                      查看当次资料
                    </Link>
                    {sourceButton}
                    {canReveal && (
                      <Button
                        size="small"
                        aria-expanded={open}
                        onClick={() => {
                          setFull(undefined);
                          setFailure('');
                          setOpen((v) => !v);
                        }}
                      >
                        {open ? '收起敏感差异' : '查看敏感差异'}
                      </Button>
                    )}
                    {open &&
                      !full &&
                      (failure ? (
                        <Alert
                          severity="error"
                          action={
                            <Button
                              onClick={() => {
                                setFailure('');
                                setRetry((v) => v + 1);
                              }}
                            >
                              重试
                            </Button>
                          }
                        >
                          {failure}
                        </Alert>
                      ) : (
                        <Typography role="status">正在读取敏感差异…</Typography>
                      ))}
                  </Stack>
                </TableCell>
              )}
            </TableRow>
          ))}
        </>
      )}
    </OperationRows>
  );
}

export function PersonChangeLog({
  id,
  session,
  params,
  positions,
  returning,
  recheckProfile,
}: {
  id: string;
  session: person.Principal;
  params: URLSearchParams;
  positions: Map<string, LogPosition>;
  returning: boolean;
  recheckProfile: () => void;
}) {
  const route = useLocation();
  const visit = route.key;
  const initialPosition = useRef(positions.get(visit) ?? positions.get(route.state?.logVisit));
  const [query, setQuery] = useState<Query>(() => initialPosition.current?.query ?? {});
  const through = useRef(query.throughSourceVersion);
  const [attempt, setAttempt] = useState(0);
  const [page, setPage] = useState<Page>();
  const [error, setError] = useState('');
  const [menu, setMenu] = useState<HTMLElement | null>(null);
  const [filter, setFilter] = useState<FilterRequest>();
  const table = useRef<HTMLDivElement>(null);
  const title = useRef<HTMLHeadingElement>(null);
  const restore = useRef(returning || !!initialPosition.current);
  const change = (next: Query) => {
    positions.delete(visit);
    setPage(undefined);
    setError('');
    setQuery(next);
  };
  const refresh = useCallback(() => {
    setPage(undefined);
    setMenu(null);
    setFilter(undefined);
    setError('');
    setAttempt((n) => n + 1);
  }, []);
  useEffect(() => {
    function visible() {
      if (document.visibilityState === 'visible') refresh();
    }
    window.addEventListener('focus', refresh);
    window.addEventListener('pageshow', refresh);
    document.addEventListener('visibilitychange', visible);
    return () => {
      window.removeEventListener('focus', refresh);
      window.removeEventListener('pageshow', refresh);
      document.removeEventListener('visibilitychange', visible);
    };
  }, [refresh]);
  useEffect(() => {
    const controller = new AbortController(),
      same = sessionGuard();
    const current = () => !controller.signal.aborted && same();
    async function load() {
      if (!current()) return;
      try {
        const value = await person.changes(
          session, id, { ...query, throughSourceVersion: through.current }, controller.signal,
        );
        if (!current()) return;
        through.current = value.throughSourceVersion;
        setPage(value);
      } catch (e) {
        if (!current()) return;
        setPage(undefined);
        if (e instanceof ApiError && e.field === 'filterValue' && query.filterField === 'changedField') {
          positions.clear();
          initialPosition.current = undefined;
          setQuery({ page: 1, pageSize: query.pageSize, throughSourceVersion: query.throughSourceVersion });
          return;
        }
        if (e instanceof ApiError && [401, 403, 404].includes(e.status)) {
          positions.clear();
          initialPosition.current = undefined;
          recheckProfile();
        }
        setError(person.message(e));
      }
    }
    // A StrictMode mount can be cleaned up before its first read should reach the API.
    queueMicrotask(load);
    return () => controller.abort();
  }, [session.csrf, id, query, attempt, positions, recheckProfile]);
  // The original may finish loading after the operator has started another
  // navigation. Restore the saved position only while that return is pending.
  useLayoutEffect(() => {
    if (!restore.current) return;
    const cancel = () => { restore.current = false; };
    const events = ['pointerdown', 'keydown', 'wheel', 'touchstart', 'focusin'];
    if (document.activeElement?.closest('a, button, input, select, textarea')) cancel();
    events.forEach(event => document.addEventListener(event, cancel, true));
    return () => events.forEach(event => document.removeEventListener(event, cancel, true));
  }, []);
  useLayoutEffect(() => {
    if (!page || !restore.current) return;
    const saved = initialPosition.current;
    initialPosition.current = undefined;
    const target = saved && document.getElementById(saved.focus);
    restore.current = false;
    if (target && saved) {
      target.focus({ preventScroll: true });
      const body = scroller();
      if (body) body.scrollTop = saved.top;
      if (table.current) table.current.scrollLeft = saved.left;
    } else title.current?.focus();
  }, [page]);
  const applied: Query = {
    ...query,
    page: page?.page ?? query.page,
    pageSize: page?.pageSize ?? query.pageSize,
    throughSourceVersion: page?.throughSourceVersion ?? query.throughSourceVersion,
  };
  useLayoutEffect(() => {
    if (!page) return;
    // Browser history can jump between two visits without opening an original
    // in the current visit. Retain its authorized query independently.
    const previous = positions.get(visit);
    positions.set(visit, {
      query: {
        ...query,
        page: page.page,
        pageSize: page.pageSize,
        throughSourceVersion: page.throughSourceVersion,
      },
      focus: previous?.focus ?? '',
      top: previous?.top ?? 0,
      left: previous?.left ?? 0,
    });
  }, [page, positions, query, visit]);
  const remember = (event: MouseEvent<HTMLAnchorElement>) => {
    if (event.button || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
    positions.set(visit, {
      query: applied,
      focus: event.currentTarget.id,
      top: scroller()?.scrollTop ?? 0,
      left: table.current?.scrollLeft ?? 0,
    });
  };
  const close = () => {
    const trigger = filter?.source;
    setFilter(undefined);
    requestAnimationFrame(() => trigger?.focus());
  };
  const columns: FilterColumn[] = [
    { field: 'recordedAt', label: '记录时间（按日期）' },
    { field: 'actor', label: '操作者' },
    {
      field: 'action',
      label: '办理动作',
      values: Object.entries(actionLabel).map(([value, label]) => ({ value, label })),
    },
    {
      field: 'changedField',
      label: '变更字段',
      values: (page?.readFields ?? []).map((value) => ({ value, label: fields[value] })),
    },
  ];
  const filterColumn = columns.find((c) => c.field === query.filterField);
  const filterSummary = query.filterValue
    ?.split(',')
    .map((value) => filterColumn?.values?.find((o) => o.value === value)?.label ?? value)
    .join('、');
  return (
    <Box className="change-log">
      <Stack direction="row" alignItems="center" useFlexGap flexWrap="wrap" gap={1}>
        <Typography ref={title} component="h2" tabIndex={-1} variant="subtitle2">
          变更日志
        </Typography>
        <IconButton aria-label="日志更多操作" onClick={(e) => setMenu(e.currentTarget)}>
          <MoreVertIcon />
        </IconButton>
        {page && filterColumn && (
          <Typography variant="body2">
            {filterColumn.label}：{filterSummary}
          </Typography>
        )}
      </Stack>
      <Menu anchorEl={menu} open={!!menu} onClose={() => setMenu(null)}>
        <MenuItem
          disabled={!page}
          onClick={() => {
            if (menu) setFilter({ source: menu });
            setMenu(null);
          }}
        >
          列筛选…
        </MenuItem>
        <MenuItem
          onClick={() => {
            setMenu(null);
            through.current = undefined;
            change({ ...applied, page: 1, throughSourceVersion: undefined });
          }}
        >
          刷新日志
        </MenuItem>
      </Menu>
      {!page &&
        (error ? (
          <Alert
            severity="error"
            action={
              <>
                {query.filterField && (
                  <Button
                    onClick={() =>
                      change({ page: 1, pageSize: query.pageSize, throughSourceVersion: through.current })
                    }
                  >
                    清除条件
                  </Button>
                )}
                <Button onClick={refresh}>重试</Button>
              </>
            }
          >
            {error}
          </Alert>
        ) : (
          <Typography role="status">正在读取日志…</Typography>
        ))}
      {page && (
        <>
          <TableContainer ref={table} tabIndex={0} className="change-log-table" aria-label="变更日志表格">
            <Table size="small" aria-label="变更日志">
              <TableHead>
                <TableRow>
                  {['记录时间', '操作者', '办理动作', '变更字段', '原值', '新值', '查看'].map((label) => (
                    <TableCell key={label}>{label}</TableCell>
                  ))}
                </TableRow>
              </TableHead>
              {page.items.map((group) => {
                const next = new URLSearchParams(params);
                next.set('source', group.sourceVersion);
                return (
                  <GroupRows
                    key={group.sourceVersion}
                    id={id}
                    group={group}
                    visit={visit}
                    href={person.path + '/' + id + '?' + next}
                    remember={remember}
                    recheck={refresh}
                  />
                );
              })}
            </Table>
          </TableContainer>
          {!page.items.length && (
            <Typography>{query.filterField ? '没有符合条件的办理记录。' : '暂无办理记录。'}</Typography>
          )}
          <TablePagination
            component="div"
            count={page.total}
            page={page.page - 1}
            rowsPerPage={page.pageSize}
            rowsPerPageOptions={[20, 50, 100]}
            labelRowsPerPage="每页操作"
            labelDisplayedRows={({ from, to, count }) => `${from}–${to} / ${count}`}
            getItemAriaLabel={(direction) => (direction === 'next' ? '下一页日志' : '上一页日志')}
            onPageChange={(_, index) => change({ ...applied, page: index + 1 })}
            onRowsPerPageChange={(e) =>
              change({ ...applied, page: 1, pageSize: Number(e.target.value) as Query['pageSize'] })
            }
          />
        </>
      )}
      {page && filter && (
        <ColumnFilterDialog
          request={filter}
          columns={columns}
          onClose={close}
          applied={{
            items: query.filterField
              ? [
                  {
                    field: query.filterField,
                    operator: query.filterOperator!,
                    value:
                      query.filterOperator === 'isAnyOf' ? query.filterValue?.split(',') : query.filterValue,
                  },
                ]
              : [],
          }}
          onApply={(v) =>
            change({
              page: 1,
              pageSize: applied.pageSize,
              throughSourceVersion: applied.throughSourceVersion,
              ...(v.filterField && {
                filterField: v.filterField as Query['filterField'],
                filterOperator: v.filterOperator,
                filterValue: v.filterValue,
              }),
            })
          }
        />
      )}
    </Box>
  );
}
