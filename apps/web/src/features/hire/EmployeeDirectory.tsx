import { useEffect, useLayoutEffect, useMemo, useRef, useState, type MouseEvent } from 'react';
import {
  Alert,
  Box,
  Button,
  IconButton,
  Menu,
  MenuItem,
  Stack,
  Tooltip,
  Typography,
  useMediaQuery,
} from '@mui/material';
import { Link, useLocation, useNavigationType, useSearchParams } from 'react-router-dom';
import {
  GridPreferencePanelsValue,
  useGridApiRef,
  type GridColDef,
  type GridSortModel,
} from '@mui/x-data-grid';
import { ApiError, sessionGuard, type Models } from '@chnhcm/api-client';
import { DataGridPage, ListControl, ListPager, SearchControl } from '@chnhcm/ui';
import { useListReturn } from '../../shared/navigation';
import MoreVert from '@mui/icons-material/MoreVert';
import {
  ColumnFilterDialog,
  ColumnFilterEditor,
  columnFilterSlots,
  type FilterRequest,
} from '../../shared/dated/ColumnFilter';
import { PersonnelDepartmentTree } from './PersonnelDepartmentTree';
import { usePersonnelDirectory } from './usePersonnelDirectory';
import * as person from '../personnel/api';

type Row = Models['PersonnelDirectoryRow'];
const primaryOptions = [
  { value: 'primary', label: '主岗' },
  { value: 'secondary', label: '兼岗' },
];
const filterColumns = [
  {
    field: 'primaryJob',
    label: '主兼岗',
    operators: [{ value: 'is', label: '等于' }],
    values: primaryOptions,
  },
];
const sorts = { number: '人员编号升序', '-number': '人员编号降序', name: '姓名升序', '-name': '姓名降序' };

export function EmployeeDirectory({
  session,
  register,
}: {
  session: Models['Principal'];
  register: React.ReactNode;
}) {
  const compact = useMediaQuery('(max-width:899px)');
  const [searching, setSearching] = useState(false);
  const [params, setParams] = useSearchParams(),
    route = useLocation(),
    navigation = useNavigationType();
  const query = person.directoryQuery(params);
  const {
    listing,
    data: pageData,
    selection,
    selecting,
    notice,
    dismiss,
  } = usePersonnelDirectory(session, params, route.key);
  const [failure, setFailure] = useState<{ text: string; status?: number; keyword: string }>();
  const data = searching || failure ? undefined : pageData;
  const expired = listing.status === 404 && !!query.ref;
  const forbidden = listing.status === 403 || listing.status === 401;
  const failed = !!failure || !!listing.error;
  const [menu, setMenu] = useState<HTMLElement | null>(null);
  const more = useRef<HTMLButtonElement>(null),
    queryButton = useRef<HTMLButtonElement>(null);
  const [filter, setFilter] = useState<FilterRequest>();
  const filterModel = useMemo(
    () => ({ items: query.primary ? [{ field: 'primaryJob', operator: 'is', value: query.primary }] : [] }),
    [query.primary],
  );
  const lastDirectory = useRef<Models['PersonnelDirectory'] | undefined>(undefined);
  useLayoutEffect(() => {
    if (failed) lastDirectory.current = undefined;
    else if (data) lastDirectory.current = data;
  }, [data, failed]);
  const display = data ?? (!failed ? lastDirectory.current : undefined);
  const staffing = !!display?.canListAssignments;
  const staleScope =
    data &&
    ((!staffing && !!(query.primary || query.department)) ||
      (query.department && !data.departments.some((d) => d.id === query.department)));
  function closeFilter() {
    const source = filter?.source;
    setFilter(undefined);
    requestAnimationFrame(() =>
      (source?.isConnected && source.getClientRects().length ? source : more.current)?.focus(),
    );
  }
  const countKey = JSON.stringify({ ...query, page: undefined });
  const lastCount = useRef<{ key: string; total: number } | undefined>(undefined);
  useLayoutEffect(() => {
    if (listing.error) lastCount.current = undefined;
    else if (data) lastCount.current = { key: countKey, total: data.total };
  }, [countKey, data, listing.error]);
  const rowCount =
    data?.total ??
    (!failed && !searching && lastCount.current?.key === countKey ? lastCount.current.total : undefined);
  const sortModel = useMemo<GridSortModel>(
    () => [{ field: query.sort.replace(/^-/, ''), sort: query.sort.startsWith('-') ? 'desc' : 'asc' }],
    [query.sort],
  );
  const apiRef = useGridApiRef(),
    [draft, setDraft] = useState('');
  const appliedParams = useRef(params);
  useLayoutEffect(() => {
    appliedParams.current = params;
  }, [params]);
  const readyRef = useRef<string | undefined>(undefined),
    request = useRef(0),
    edit = useRef(0);
  const waitingForRef = useRef<string | undefined>(undefined);
  const localSearch = useRef<string | undefined>(undefined);
  useLayoutEffect(() => {
    const local = navigation !== 'POP' && localSearch.current === route.search;
    localSearch.current = undefined;
    if (!local) {
      request.current++;
      waitingForRef.current = undefined;
      readyRef.current = undefined;
      setSearching(false);
      setDraft('');
      setFailure(undefined);
    }
  }, [route.key, route.search, navigation]);
  useLayoutEffect(() => {
    if (waitingForRef.current && waitingForRef.current === query.ref && (listing.data || listing.error)) {
      waitingForRef.current = undefined;
      setSearching(false);
    }
    if (listing.status === 401 || listing.status === 403) {
      request.current++;
      readyRef.current = undefined;
      setSearching(false);
      setDraft('');
      return;
    }
    if (data && readyRef.current !== (query.ref ?? '')) {
      setDraft(data.keyword);
      readyRef.current = query.ref ?? '';
    }
  }, [data, query.ref, listing.data, listing.error, listing.status]);
  useEffect(
    () => () => {
      request.current++;
    },
    [],
  );
  const returned = useListReturn({
    key: [
      'personnel',
      session.instanceId,
      session.subjectId,
      query.ref,
      query.department,
      query.primary,
      query.directOnly,
    ].join(':'),
    ready: !!data,
    apiRef,
    rows: data?.rows ?? [],
    selected: params.get('selected'),
    pageOffset: (query.page - 1) * query.pageSize,
  });
  function update(fields: Record<string, string>) {
    if (window.location.pathname !== route.pathname) return;
    // A search reply can arrive after another filter has already navigated.
    const next = new URLSearchParams(appliedParams.current);
    next.delete('selected');
    for (const [k, v] of Object.entries(fields)) {
      if (v) next.set(k, v);
      else next.delete(k);
    }
    if (next.toString() === appliedParams.current.toString()) return;
    appliedParams.current = next;
    localSearch.current = '?' + next;
    setParams(next);
  }
  async function search(keyword = draft) {
    const token = ++request.current,
      revision = edit.current,
      current = sessionGuard();
    setSearching(true);
    waitingForRef.current = undefined;
    try {
      const { ref } = await person.queryRef(session, keyword);
      if (token === request.current && current()) {
        setFailure(undefined);
        if (revision !== edit.current || keyword !== draft) readyRef.current = ref;
        waitingForRef.current = ref;
        update({ ref, page: '1' });
      }
    } catch (e) {
      if (token === request.current && current())
        setFailure({
          text: person.message(e),
          status: e instanceof ApiError ? e.status : undefined,
          keyword,
        });
    } finally {
      if (token === request.current && current() && !waitingForRef.current) setSearching(false);
    }
  }
  const visibleRows = useRef<Set<string>>(new Set());
  useLayoutEffect(() => {
    visibleRows.current = new Set(data?.rows.map((row) => row.id));
  }, [data]);
  function openRow(event: MouseEvent<HTMLAnchorElement>, row: Row) {
    if (
      new URLSearchParams(window.location.search).toString() !== params.toString() ||
      !visibleRows.current.has(row.id)
    ) {
      event.preventDefault();
      return;
    }
    returned.rememberRow(row.id);
  }
  function rowLink(row: Row, compactRow = false) {
    const term = row.assignment;
    return (
      <Link
        data-object-link
        className="object-name-link"
        onClick={(e) => openRow(e, row)}
        to={person.directoryTarget(params, row.person, term?.id)}
      >
        {row.name}
        {compactRow && term && ` · ${term.recordNumber} 号${term.primaryJob ? '主岗' : '兼岗'}`}
      </Link>
    );
  }
  const columns = useMemo<GridColDef<Row>[]>(() => {
    const fields: GridColDef<Row>[] = [
      { field: 'name', headerName: '姓名', flex: 1, minWidth: 120, renderCell: ({ row }) => rowLink(row) },
      {
        field: 'number',
        headerName: '人员编号',
        width: 146,
        renderCell: ({ row }) => row.number + (row.historical ? '（历史编号命中）' : ''),
      },
      {
        field: 'recordNumber',
        headerName: '雇用记录编号',
        width: 118,
        sortable: false,
        renderCell: ({ row }) => row.assignment?.recordNumber ?? '—',
      },
      {
        field: 'primaryJob',
        type: 'singleSelect',
        valueOptions: primaryOptions,
        filterable: true,
        headerName: '主兼岗',
        width: 82,
        sortable: false,
        renderCell: ({ row }) => (row.assignment ? (row.assignment.primaryJob ? '主岗' : '兼岗') : '—'),
      },
      {
        field: 'department',
        headerName: '部门',
        flex: 1,
        minWidth: 105,
        sortable: false,
        renderCell: ({ row }) => row.assignment?.departmentName || '—',
      },
      {
        field: 'job',
        headerName: '职务',
        flex: 1,
        minWidth: 100,
        sortable: false,
        renderCell: ({ row }) => row.assignment?.jobName || '—',
      },
      {
        field: 'position',
        headerName: '岗位',
        flex: 1,
        minWidth: 100,
        sortable: false,
        renderCell: ({ row }) => row.assignment?.positionName || '—',
      },
      {
        field: 'period',
        headerName: '任职期间',
        width: 185,
        sortable: false,
        renderCell: ({ row }) =>
          row.assignment ? (
            <Stack>
              <span>{row.assignment.from} 起</span>
              {row.assignment.until && <span>{row.assignment.until} 起不在任</span>}
            </Stack>
          ) : (
            '—'
          ),
      },
      {
        field: 'status',
        headerName: '状态',
        width: 95,
        sortable: false,
        renderCell: ({ row }) => (row.assignment ? (row.future ? '尚未生效' : '在任') : '—'),
      },
    ];
    return (staffing ? fields : fields.slice(0, 2)).map((column) => ({ filterable: false, ...column }));
  }, [params, returned.rememberRow, staffing]);
  const summary = useMemo<GridColDef<Row>>(
    () => ({
      field: 'summary',
      headerName: '人员',
      renderCell: ({ row }) => (
        <Stack>
          {rowLink(row, true)}
          <small>
            {[
              row.number,
              row.historical ? '历史编号命中' : '',
              row.assignment?.departmentName,
              row.assignment && '职务 ' + row.assignment.jobName,
              row.assignment?.positionName && '岗位 ' + row.assignment.positionName,
              row.future ? '尚未生效' : '',
            ]
              .filter(Boolean)
              .join(' · ')}
          </small>
        </Stack>
      ),
    }),
    [params, returned.rememberRow],
  );
  function scope(id?: string, directOnly?: boolean) {
    update({ department: id ?? '', directOnly: id && directOnly ? '1' : '', page: '1' });
  }
  const restoreSearch = () =>
    requestAnimationFrame(() =>
      document.querySelector<HTMLInputElement>('input[aria-label="姓名或人员编号"]')?.focus(),
    );
  return (
    <ColumnFilterEditor.Provider value={setFilter}>
      <div className="standard-page">
        <DataGridPage<Row>
          label="人员"
          apiRef={apiRef}
          scrollerRef={returned.scroller}
          selected={returned.selected}
          rows={data?.rows ?? []}
          ready={!!data}
          loading={!data && !failed}
          empty={data && !staleScope ? '没有匹配的人员' : ''}
          filterModel={staffing ? filterModel : undefined}
          slots={columnFilterSlots}
          selection={{ ...selection, disabled: !data || selection.disabled }}
          columns={columns}
          summaryColumn={summary}
          paginationModel={{ page: query.page - 1, pageSize: query.pageSize }}
          rowCount={rowCount}
          onPaginationModelChange={(m) => {
            if (!data || (m.page === query.page - 1 && m.pageSize === query.pageSize)) return;
            update({
              page: String(m.pageSize === query.pageSize ? m.page + 1 : 1),
              pageSize: String(m.pageSize),
            });
          }}
          sortModel={sortModel}
          onSortModelChange={(m) => {
            const sort = (m[0]?.sort === 'desc' ? '-' : '') + (m[0]?.field ?? 'number');
            if (sort !== query.sort) update({ sort, page: '1' });
          }}
          toolbar={
            <ListControl
              title="人员"
              action={register}
              more={
                <>
                  <Tooltip title="更多操作">
                    <IconButton
                      ref={more}
                      size="small"
                      aria-label="更多操作"
                      aria-haspopup="menu"
                      aria-expanded={!!menu}
                      onClick={(e) => setMenu(e.currentTarget)}
                    >
                      <MoreVert />
                    </IconButton>
                  </Tooltip>
                  <Menu anchorEl={menu} open={!!menu} onClose={() => setMenu(null)}>
                    {staffing && (
                      <MenuItem
                        onClick={() => {
                          setMenu(null);
                          setFilter({ field: 'primaryJob', source: more.current! });
                        }}
                      >
                        列筛选…
                      </MenuItem>
                    )}
                    {!compact && (
                      <MenuItem
                        onClick={() => {
                          setMenu(null);
                          apiRef.current?.showPreferences(GridPreferencePanelsValue.columns);
                        }}
                      >
                        列设置…
                      </MenuItem>
                    )}
                    {compact &&
                      Object.entries(sorts).map(([value, label]) => (
                        <MenuItem
                          key={value}
                          selected={value === query.sort}
                          onClick={() => {
                            setMenu(null);
                            update({ sort: value, page: '1' });
                          }}
                        >
                          排序 · {label}
                        </MenuItem>
                      ))}
                  </Menu>
                </>
              }
              pager={<ListPager />}
              search={
                <Box className="list-search-controls">
                  {data && (
                    <Typography component="span" variant="body2" sx={{ whiteSpace: 'nowrap' }}>
                      {data.assignments} 份任职，{data.people} 人
                    </Typography>
                  )}
                  {selecting && !failed ? (
                    <Stack direction="row" alignItems="center" gap={1}>
                      <Typography role="status">
                        {listing.data?.selection ? `已选择${selection.count}项` : '正在核对选择…'}
                      </Typography>
                      <Button
                        size="small"
                        onClick={() => {
                          selection.clear();
                          restoreSearch();
                        }}
                      >
                        取消选择
                      </Button>
                    </Stack>
                  ) : (
                    <SearchControl
                      buttonRef={queryButton}
                      value={draft}
                      onChange={(value) => {
                        edit.current++;
                        readyRef.current = query.ref ?? '';
                        setDraft(value);
                      }}
                      onSearch={() => search()}
                      label="姓名或人员编号"
                    />
                  )}
                </Box>
              }
            >
              {data && !staleScope && (query.primary || query.department) && (
                <Box className="list-applied-summary">
                  {query.department && (
                    <Button size="small" onClick={() => scope()}>
                      清除部门范围 · {data.departments.find((d) => d.id === query.department)?.name}
                    </Button>
                  )}
                  {query.primary && (
                    <Button size="small" onClick={(e) => setFilter({ source: e.currentTarget })}>
                      主兼岗：{primaryOptions.find((o) => o.value === query.primary)?.label}
                    </Button>
                  )}
                </Box>
              )}
              {data && draft !== data.keyword && (
                <Typography className="list-query-summary" role="status">
                  输入已修改，点击查询后生效。已应用关键词：{data.keyword || '无'}
                </Typography>
              )}
              {filter && staffing && (
                <ColumnFilterDialog
                  columns={filterColumns}
                  request={filter}
                  applied={filterModel}
                  onApply={(values) => update({ primary: values.filterValue, page: '1' })}
                  onClose={closeFilter}
                />
              )}
            </ListControl>
          }
          sidebar={
            <>
              {staffing && !!display?.departments.length && (
                <PersonnelDepartmentTree
                  nodes={display.departments}
                  selected={query.department}
                  directOnly={query.directOnly}
                  onScope={scope}
                />
              )}
            </>
          }
          messages={
            <>
              {failed && (
                <Alert
                  severity="error"
                  action={
                    expired && !failure ? (
                      <Button
                        onClick={() => {
                          setDraft('');
                          update({ ref: '', page: '1' });
                          restoreSearch();
                        }}
                      >
                        重新查询
                      </Button>
                    ) : failure && [400, 422].includes(failure.status ?? 0) ? (
                      <Button onClick={restoreSearch}>修改输入</Button>
                    ) : (
                      !forbidden && (
                        <Button onClick={() => (failure ? search(failure.keyword) : listing.refresh())}>
                          重试
                        </Button>
                      )
                    )
                  }
                >
                  {failure?.text || (expired ? '原查询已失效，请重新输入姓名或人员编号。' : listing.error)}
                </Alert>
              )}
              {staleScope && (
                <Alert
                  severity="info"
                  action={
                    <Button
                      onClick={() => update({ primary: '', department: '', directOnly: '', page: '1' })}
                    >
                      清除任职条件
                    </Button>
                  }
                >
                  当前权限下无法使用此任职条件。
                </Alert>
              )}
              {notice && (
                <Alert severity="info" onClose={dismiss}>
                  {notice}
                </Alert>
              )}
            </>
          }
        />
      </div>
    </ColumnFilterEditor.Provider>
  );
}
