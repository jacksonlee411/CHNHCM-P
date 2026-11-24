import MoreVertIcon from '@mui/icons-material/MoreVert';
import {
  ColumnFilterEditor,
  ColumnFilterDialog,
  columnFilterSlots,
  type FilterColumn,
  type FilterRequest,
} from '../../shared/dated/ColumnFilter';
import { useEffect, useMemo, useRef, useState } from 'react';
import {
  Alert,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  IconButton,
  Menu,
  MenuItem,
  Stack,
  TextField,
  Typography,
  useMediaQuery,
} from '@mui/material';
import { Link, useNavigate } from 'react-router-dom';
import { useGridApiRef, type GridSortModel, type GridColDef } from '@mui/x-data-grid';
import { DataGridPage, ListControl, ListPager, SearchControl, StatusChip, businessDate } from '@chnhcm/ui';
import { ApiError, sessionGuard, type Models } from '@chnhcm/api-client';
import { useListReturn, useMaintenanceSearchParams, usePageTitle } from '../../shared/navigation';
import {structurePath} from './structure';
import { DepartmentTree } from '../organization/DepartmentTree';
import * as staff from './api';
export function StaffingList({ kind: k, session }: { kind: staff.Kind; session: staff.Principal }) {
  usePageTitle(staff.labels[k]);
  const [params, setParams] = useMaintenanceSearchParams(staff.paths[k]),
    query = staff.queryFrom(params),
    queryKey = JSON.stringify(query);
  const [dataKey, setDataKey] = useState('');
  const pendingQuery = useRef<string | undefined>(undefined);
  const [menu, setMenu] = useState<HTMLElement | null>(null),
    [filter, setFilter] = useState<FilterRequest>();
  const more = useRef<HTMLButtonElement>(null);
  const narrow = useMediaQuery('(max-width:899px)');
  const [sorting, setSorting] = useState(false),
    [draftSort, setDraftSort] = useState('');
  const previousWidth = useRef(narrow);
  const closeSort = () => {
    setSorting(false);
    requestAnimationFrame(() => more.current?.focus());
  };
  useEffect(() => {
    if (previousWidth.current !== narrow && (menu || sorting)) {
      setMenu(null);
      closeSort();
    }
    previousWidth.current = narrow;
  }, [narrow, menu, sorting]);
  const total = useRef(0);
  const [data, setData] = useState<Models['StaffingList']>(),
    [error, setError] = useState(''),
    [loading, setLoading] = useState(true),
    [retry, setRetry] = useState(0),
    [search, setSearch] = useState(query.q ?? '');
  const [selection, setSelection] = useState<staff.Selection>(),
    [selectionBusy, setSelectionBusy] = useState(false),
    [notice, setNotice] = useState('');
  const sortModel = useMemo<GridSortModel>(
    () =>
      query.sort
        ? [{ field: query.sort.replace(/^-/, ''), sort: query.sort.startsWith('-') ? 'desc' : 'asc' }]
        : [],
    [query.sort],
  );
  const identity = JSON.stringify({ ...query, page: undefined, pageSize: undefined, sort: undefined });
  const previous = useRef(identity);
  const key = `chnhcm-staffing-selection:${session.instanceId}:${session.subjectId}:${k}`;
  const controller = useRef<AbortController | undefined>(undefined),
    choosing = useRef(false);
  const patch = (v: Record<string, string>) => {
    // The address can already contain a newly applied department while the
    // toolbar that initiated the next action still belongs to the prior render.
    const next = new URLSearchParams(window.location.search);
    for (const [key, value] of Object.entries(v)) {
      if (value) next.set(key, value);
      else next.delete(key);
    }
    const nextKey = JSON.stringify(staff.queryFrom(next));
    if (nextKey !== queryKey) {
      pendingQuery.current = nextKey;
      setDataKey('');
      setData(undefined);
      setLoading(true);
    }
    setParams(next, { flushSync: true });
  };
  useEffect(() => {
    const abort = new AbortController(),
      same = sessionGuard();
    setLoading(true);
    setError('');
    setData(undefined);
    setFilter(undefined);
    staff
      .list(k, query, abort.signal)
      .then((v) => {
        if (!abort.signal.aborted && same() && queryKey === JSON.stringify(staff.queryFrom(new URLSearchParams(window.location.search)))) {
          pendingQuery.current = undefined;
          setData(v);
          setDataKey(queryKey);
          total.current = v.totalCount;
        }
      })
      .catch((e) => {
        if (!abort.signal.aborted && same()) {
          if (e instanceof ApiError && [401, 403].includes(e.status)) clear();
          setError(e instanceof Error ? e.message : '查询失败');
        }
      })
      .finally(() => {
        if (!abort.signal.aborted && same()) setLoading(false);
      });
    return () => abort.abort();
  }, [k, queryKey, retry]);
  const [scopeContext,setScopeContext] = useState<Awaited<ReturnType<typeof staff.context>>>();
  const [scopeLabel,setScopeLabel] = useState<string>();
  useEffect(() => {
    if (k !== 'position') return;
    const abort = new AbortController(), same = sessionGuard();
    setScopeContext(undefined);
    void staff.context(k,abort.signal).then(value => {
      if (!abort.signal.aborted && same()) setScopeContext(value);
    }).catch(() => { /* 岗位列表独立读取，侧栏须先取得权限。 */ });
    return () => abort.abort();
  }, [k,session.csrf,retry]);
  const selectScope = (id?:string) => {
    if (id === query.scopeDepartmentId) return;
    patch({scopeDepartmentId:id ?? '',includeDescendants:id ? String(query.includeDescendants ?? true) : '',selected:'',page:'1'});
  };
  const persist = (v: staff.Selection) => {
    if (v.state !== 'active') {
      clear();
      setNotice('原选择未生效或已取消，请重新选择。');
      return;
    }
    sessionStorage.setItem(
      key,
      JSON.stringify({
        draftId: v.draftId,
        expectedRevision: v.revision,
        query: { ...query, page: undefined, pageSize: undefined, sort: undefined },
      }),
    );
    setSelection(v);
  };
  const clear = () => {
    controller.current?.abort();
    setSelection(undefined);
    sessionStorage.removeItem(key);
  };
  useEffect(() => {
    if (previous.current !== identity) {
      previous.current = identity;
      const hadSelection = !!selection || !!sessionStorage.getItem(key);
      clear();
      setNotice(hadSelection ? '筛选条件已变化，请重新选择。' : '');
    }
  }, [identity]);
  useEffect(() => {
    const abort = new AbortController(),
      same = sessionGuard();
    try {
      const raw = sessionStorage.getItem(key);
      if (raw) {
        const input = JSON.parse(raw) as Models['StaffingSelectionInput'];
        if (
          JSON.stringify({ ...input.query, page: undefined, pageSize: undefined, sort: undefined }) !==
          identity
        ) {
          sessionStorage.removeItem(key);
        } else {
          controller.current = abort;
          void staff
            .select(
              k,
              session,
              'resolve',
              { ...input, initialIds: undefined, expectedRevision: undefined, excludedIds: undefined },
              abort.signal,
            )
            .then((v) => {
              if (!abort.signal.aborted && same()) persist(v);
            })
            .catch((e) => {
              if (!abort.signal.aborted && same()) {
                if (e instanceof ApiError && [401, 403, 409].includes(e.status)) {
                  clear();
                  setNotice('上次固定选择已失效，请重新选择。');
                } else setNotice('原选择结果尚未确认，请再次点击全选以查回原选择。');
              }
            });
        }
      }
    } catch {
      setNotice('固定选择暂不可读，请重新选择。');
    }
    return () => {
      abort.abort();
      controller.current?.abort();
    };
  }, [k, session.instanceId, session.subjectId]);
  useEffect(() => {
    const invalidate = () => clear();
    window.addEventListener('chnhcm-session-invalidated', invalidate);
    return () => window.removeEventListener('chnhcm-session-invalidated', invalidate);
  }, [key]);
  const choose = async (excluded?: string[], initialIds?: string[]) => {
    if (choosing.current || pendingQuery.current || dataKey !== queryKey) return;
    choosing.current = true;
    controller.current?.abort();
    const abort = new AbortController();
    controller.current = abort;
    const same = sessionGuard();
    setSelectionBusy(true);
    try {
      let input: Models['StaffingSelectionInput'] = {
        draftId: selection?.draftId ?? crypto.randomUUID(),
        expectedRevision: selection?.revision,
        query,
        excludedIds: excluded,
        initialIds,
      };
      let op: 'start' | 'resolve' = selection ? 'resolve' : 'start';
      if (!selection) {
        const pending = sessionStorage.getItem(key);
        if (pending) {
          const original = JSON.parse(pending) as Models['StaffingSelectionInput'];
          if (
            JSON.stringify({ ...original.query, page: undefined, pageSize: undefined, sort: undefined }) ===
            identity
          ) {
            input = { draftId: original.draftId, query };
            op = 'resolve';
          }
        }
        sessionStorage.setItem(key, JSON.stringify(input));
      }
      sessionStorage.setItem(key, JSON.stringify(input));
      if (op !== 'start') delete input.initialIds;
      const v = await staff.select(k, session, op, input, abort.signal);
      if (!abort.signal.aborted && same()) persist(v);
    } catch (e) {
      if (!abort.signal.aborted && same()) {
        setSelection(undefined);
        if (e instanceof ApiError && [401, 403, 409].includes(e.status)) clear();
        setNotice(e instanceof Error ? e.message : '选择结果尚未确认，请刷新查询。');
      }
    } finally {
      choosing.current = false;
      if (same()) setSelectionBusy(false);
    }
  };
  const download = async () => {
    if (!selection) return;
    const abort = new AbortController();
    controller.current = abort;
    const same = sessionGuard();
    setSelectionBusy(true);
    try {
      await staff.exportCSV(
        k,
        session,
        { draftId: selection.draftId, expectedRevision: selection.revision, query, sort: query.sort },
        abort.signal,
      );
    } catch (e) {
      if (same()) {
        if (e instanceof ApiError && e.code === 'M03_SELECTION_UNAVAILABLE') clear();
        setNotice(e instanceof Error ? e.message : '导出失败');
      }
    } finally {
      if (same()) setSelectionBusy(false);
    }
  };
  useEffect(() => {
    if (selection && data?.businessAt && selection.businessAt !== data.businessAt) {
      clear();
      setNotice('业务日期已变化，请重新选择。');
    }
  }, [data?.businessAt, selection?.businessAt]);
  const grid = useGridApiRef();
  const returned = useListReturn({
    key: `staffing-return:${k}`,
    ready: !!data && dataKey === queryKey,
    apiRef: grid,
    selected: params.get('selected'),
    rows: data?.items ?? [],
    pageOffset: ((query.page ?? 1) - 1) * (query.pageSize ?? 20),
  });
  const nameLink = (r: staff.Row) => (
    <StaffingNameLink kind={k} row={r} params={params} remember={() => returned.rememberRow(r.id)} />
  );
  const read = data?.capabilities.readFields ?? [];
  const columns: GridColDef<staff.Row>[] = [
    'code',
    'name',
    'nameEn',
    ...(k === 'job' ? ['jobFamilyId', 'jobFunctionId'] : []),
    'status',
    'effectiveDate',
  ]
    .filter((f) => read.includes(f))
    .map((field) => ({
      field,
      headerName: staff.fieldLabels[field],
      width: field === 'name' ? 220 : 170,
      flex: field === 'name' ? 1 : undefined,
      minWidth: 140,
      filterable: true,
      filterOperators: filterOperators(field).map((op) => ({ ...op, getApplyFilterFn: () => null })),
      sortable: (k === 'grade' || k === 'gradeladder') ? ['code', 'name', 'nameEn'].includes(field) : !staff.isJobClassification(field),
      sortingOrder: (k === 'grade' || k === 'gradeladder') && field === 'code' ? ['asc', null] : undefined,
      renderCell: ({ row }) =>
        field === 'name' ? (
          nameLink(row)
        ) : staff.isJobClassification(field) ? (
          field === 'jobFamilyId' ? row.jobFamilyName : row.jobFunctionName
        ) : field === 'status' ? (
          <StatusChip
            label={staff.statusLabel(row.status)}
            color={row.status === 'active' ? 'success' : 'default'}
          />
        ) : field === 'effectiveDate' && row.effectiveDate ? (
          businessDate(row.effectiveDate)
        ) : (
          String(row[field as keyof staff.Row] ?? '')
        ),
    }));
  const ids = selection?.members.filter((id) => !selection.excludedIds.includes(id)) ?? [];
  const filterFields = read.filter((f) =>
    [
      'code',
      'name',
      'nameEn',
      'description',
      'responsibilities',
      'requirements',
      'status',
      'effectiveDate',
      ...(k === 'job' ? ['jobFamilyId', 'jobFunctionId'] : []),
    ].includes(f),
  );
  return (
    <ColumnFilterEditor.Provider value={setFilter}>
      <div className="standard-page" aria-busy={selectionBusy || (!error && (loading || dataKey !== queryKey))}>
        <DataGridPage
          sidebar={k === 'position' ? <>{scopeContext?.capabilities.actions.departmentRange &&
            <DepartmentTree scopeId={query.scopeDepartmentId} includeDescendants={query.includeDescendants ?? true}
              day={scopeContext.currentDate} unavailable={!!error} onLabel={setScopeLabel}
              onDayMismatch={() => {clear();setRetry(n=>n+1)}} onSelect={selectScope}
              onInclude={value => {if(query.scopeDepartmentId) patch({includeDescendants:String(value),selected:'',page:'1'})}}/>
            }</> : undefined}
          apiRef={grid}
          slots={columnFilterSlots}
          filterModel={{
            items: query.filterField
              ? [{ field: query.filterField, operator: query.filterOperator ?? '', value: query.filterValue }]
              : [],
          }}
          scrollerRef={returned.scroller}
          rows={dataKey === queryKey ? (data?.items ?? []) : []}
          columns={columns}
          summaryColumn={{
            field: 'summary',
            headerName: '',
            renderCell: ({ row }) => (
              <Stack>
                {nameLink(row)}
                <Typography variant="body2">
                  {[
                    row.code,
                    staff.statusLabel(row.status),
                    row.jobFamilyName,
                    row.jobFunctionName,
                    row.effectiveDate ? businessDate(row.effectiveDate) : '',
                  ]
                    .filter(Boolean)
                    .join(' · ')}
                </Typography>
              </Stack>
            ),
          }}
          label={`${staff.labels[k]}列表`}
          selected={params.get('selected')}
          loading={loading || dataKey !== queryKey}
          ready={!!data && dataKey === queryKey}
          empty={`没有符合条件的${staff.labels[k]}`}
          paginationModel={{ page: (query.page ?? 1) - 1, pageSize: query.pageSize ?? 20 }}
          rowCount={error ? 0 : total.current}
          onPaginationModelChange={(m) => {
            if (
              !loading &&
              data &&
              dataKey === queryKey &&
              (m.page + 1 !== (query.page ?? 1) || m.pageSize !== (query.pageSize ?? 20))
            )
              patch({ page: String(m.page + 1), pageSize: String(m.pageSize) });
          }}
          sortModel={sortModel}
          onSortModelChange={(m) => {
            const next = m[0] ? `${m[0].sort === 'desc' ? '-' : ''}${m[0].field}` : '';
            if (!loading && dataKey === queryKey && next !== (query.sort ?? ''))
              patch({ sort: next, q: search, page: '1' });
          }}
          selection={{
            checked: !!selection && selection.count === selection.members.length,
            mixed: !!selection && selection.count > 0 && selection.count < selection.members.length,
            ids: new Set(ids),
            selectable: (id) => selection?.members.includes(id) ?? true,
            onChange: (selected) =>
              void choose(
                selection?.members.filter((id) => !selected.has(id)),
                selection ? undefined : [...selected],
              ),
            count: selection?.count,
            total: data?.totalCount ?? 0,
            // Keep the focused checkbox mounted and focusable while the
            // server confirms selection. choose() serializes in-flight updates.
            disabled: loading || !!pendingQuery.current || !data || dataKey !== queryKey,
            all: () => void choose([]),
            clear,
          }}
          messages={
            <>
              {error && (
                <Alert
                  severity="error"
                  action={
                    <>
                      <Button onClick={() => setRetry((n) => n + 1)}>重试</Button>
                      {(error.includes('筛选') || error.includes('排序')) && (
                        (query.jobFamilyId || query.jobFunctionId || query.jobFunctionMissing) ? <>
                          {query.jobFamilyId && <Button onClick={() => patch({jobFamilyId: '', selected: '', page: '1'})}>移除职务系列筛选</Button>}
                          {(query.jobFunctionId || query.jobFunctionMissing) && <Button onClick={() => patch({jobFunctionId: '', jobFunctionMissing: '', selected: '', page: '1'})}>移除职类筛选</Button>}
                        </> : <Button
                          onClick={() => {
                            setParams(new URLSearchParams());
                            setSearch('');
                            setNotice('已清除查询条件。');
                          }}
                        >
                          清除条件并查询
                        </Button>
                      )}
                    </>
                  }
                >
                  {error}
                </Alert>
              )}
              {notice && (
                <Alert severity="info" onClose={() => setNotice('')}>
                  {notice}
                </Alert>
              )}
            </>
          }
          toolbar={
            <ListControl
              title={staff.labels[k]}
              action={
                data?.capabilities.actions.CREATE && (
                  <Button
                    component={Link}
                    to={`${staff.paths[k]}/new?back=${encodeURIComponent(staff.paths[k] + '?' + params)}`}
                    variant="contained"
                  >
                    新建
                  </Button>
                )
              }
              search={
                <SearchControl
                  value={search}
                  onChange={setSearch}
                  onSearch={() => patch({ q: search, page: '1' })}
                  label="搜索编码或名称"
                />
              }
              pager={<ListPager />}
              more={
                <>
                  <IconButton
                    ref={more}
                    aria-label="更多操作"
                    aria-haspopup="menu"
                    aria-expanded={!!menu}
                    disabled={loading && !data && !scopeContext?.capabilities.actions.positionRelations}
                    onClick={(e) => setMenu(e.currentTarget)}
                  >
                    <MoreVertIcon />
                  </IconButton>
                  <Menu anchorEl={menu} open={!!menu} onClose={() => setMenu(null)}>
                    {k === 'position' && scopeContext?.capabilities.actions.positionRelations && (
                      <MenuItem
                        component={Link}
                        to={params.size ? structurePath + '?' + new URLSearchParams({ listBack: staff.paths.position + '?' + params }) : structurePath}
                        onClick={() => {
                          returned.rememberRow(returned.selected ?? undefined);
                          setMenu(null);
                        }}
                      >
                        查看岗位上下级
                      </MenuItem>
                    )}
                    <MenuItem
                      disabled={!filterFields.length}
                      onClick={() => {
                        setFilter({ source: more.current! });
                        setMenu(null);
                      }}
                    >
                      列筛选…
                    </MenuItem>
                    {narrow && (
                      <MenuItem
                        onClick={() => {
                          setDraftSort(query.sort ?? '');
                          setSorting(true);
                          setMenu(null);
                        }}
                      >
                        排序…
                      </MenuItem>
                    )}
                    <MenuItem
                      disabled={!selection?.count || selectionBusy || !(data?.capabilities.actions.export ?? scopeContext?.capabilities.actions.export)}
                      onClick={() => {
                        setMenu(null);
                        void download();
                      }}
                    >
                      导出所选{selection ? `（${selection.count}）` : ''}
                    </MenuItem>
                    {selection && (
                      <MenuItem
                        onClick={() => {
                          clear();
                          setMenu(null);
                        }}
                      >
                        取消选择
                      </MenuItem>
                    )}
                  </Menu>
                </>
              }
            >
              {k === 'position' && query.scopeDepartmentId && <Stack direction="row" alignItems="center">
                <Typography variant="body2" role="status">{error ? '当前部门范围不可用' : scopeLabel ? scopeLabel + (query.includeDescendants ? ' · 含子部门' : '') : '所选部门'}</Typography>
                <Button size="small" onClick={()=>selectScope()}>清除范围</Button>
              </Stack>}
              {search !== (query.q ?? '') && (
                <Typography className="list-query-summary" role="status">
                  输入已修改，点击查询后生效。已应用关键词：{query.q || '无'}
                </Typography>
              )}
              {sorting && (
                <Dialog
                  open
                  disableRestoreFocus
                  fullWidth
                  maxWidth="xs"
                  onClose={closeSort}
                  aria-labelledby="staffing-sort-title"
                >
                  <form
                    onSubmit={(e) => {
                      e.preventDefault();
                      patch({ sort: draftSort, q: search, page: '1' });
                      closeSort();
                    }}
                  >
                    <DialogTitle id="staffing-sort-title">排序</DialogTitle>
                    <DialogContent>
                      <TextField
                        select
                        fullWidth
                        size="small"
                        label="排序方式"
                        slotProps={{
                          select: {
                            SelectDisplayProps: { 'aria-label': '排序方式', 'aria-labelledby': undefined },
                          },
                        }}
                        value={draftSort}
                        onChange={(e) => setDraftSort(e.target.value)}
                        sx={{ mt: 1 }}
                      >
                        <MenuItem value="">默认顺序</MenuItem>
                        {columns
                          .filter((c) => c.sortable !== false)
                          .flatMap((c) => [
                            <MenuItem key={c.field} value={c.field}>
                              {c.headerName}升序
                            </MenuItem>,
                            (k === 'grade' || k === 'gradeladder') && c.field === 'code' ? null : <MenuItem key={'-' + c.field} value={'-' + c.field}>
                              {c.headerName}降序
                            </MenuItem>,
                          ])}
                      </TextField>
                    </DialogContent>
                    <DialogActions>
                      <Button onClick={closeSort}>取消</Button>
                      <Button type="submit" variant="contained">
                        应用
                      </Button>
                    </DialogActions>
                  </form>
                </Dialog>
              )}
              {(query.filterField || query.jobFamilyId || query.jobFunctionId || query.jobFunctionMissing || (query.status && query.status !== 'all')) && (
                <Typography variant="body2" sx={{ py: 1 }}>
                  {query.jobFamilyId || query.jobFunctionId || query.jobFunctionMissing
                    ? [query.jobFamilyId && '职务系列：' + (data?.jobFamilyFilterName ?? '已选择'), query.jobFunctionId && '职类：' + (data?.jobFunctionFilterName ?? '已选择'), query.jobFunctionMissing && '职类：未登记'].filter(Boolean).join(' · ')
                    : `${staff.fieldLabels[query.filterField ?? 'status']}：${query.filterValue ?? staff.statusLabel(query.status)}`}
                </Typography>
              )}
              {filter && filterFields.length > 0 && (
                <ColumnFilterDialog
                  request={filter}
                  applied={{
                    items: query.jobFamilyId || query.jobFunctionId || query.jobFunctionMissing
                      ? [
                        ...(query.jobFamilyId ? [{field: 'jobFamilyId', operator: 'equals', value: query.jobFamilyId}] : []),
                        ...(query.jobFunctionId || query.jobFunctionMissing ? [{field: 'jobFunctionId', operator: query.jobFunctionMissing ? 'isEmpty' : 'equals', value: query.jobFunctionId ?? ''}] : []),
                      ]
                      : query.filterField
                        ? [
                            {
                              field: query.filterField,
                              operator: query.filterOperator ?? '',
                              value: query.filterValue,
                            },
                          ]
                        : query.status && query.status !== 'all'
                          ? [{ field: 'status', operator: 'equals', value: query.status }]
                          : [],
                  }}
                  columns={filterFields.map(
                    (field) =>
                      ({
                        field,
                        label: staff.fieldLabels[field],
                        operators: filterOperators(field),
                        ...(field === 'status'
                          ? {
                              values: ['active', 'inactive', 'not_effective'].map((value) => ({
                                value,
                                label: staff.statusLabel(value),
                              })),
                            }
                          : field === 'jobFamilyId'
                            ? { Input: FamilyFilter }
                            : field === 'jobFunctionId' ? { Input: FunctionFilter } : {}),
                      }) satisfies FilterColumn,
                  )}
                  onClose={() => {
                    const source = filter.source;
                    setFilter(undefined);
                    requestAnimationFrame(() =>
                      (source.isConnected && source.getClientRects().length ? source : more.current)?.focus(),
                    );
                  }}
                  onApply={(v, editedField) => {
                    const common = { q: search, page: '1', selected: '' };
                    const resetColumn = { status: '', filterField: '', filterOperator: '', filterValue: '' };
                    if (editedField === 'jobFamilyId') {
                      if (!v.filterField && !query.jobFamilyId) return;
                      patch({ ...common, ...resetColumn, jobFamilyId: v.filterValue });
                    } else if (editedField === 'jobFunctionId') {
                      if (!v.filterField && !query.jobFunctionId && !query.jobFunctionMissing) return;
                      const missing = v.filterOperator === 'isEmpty';
                      patch({ ...common, ...resetColumn, jobFunctionId: missing ? '' : v.filterValue,
                        jobFunctionMissing: missing ? 'true' : '' });
                    } else {
                      if (!v.filterField && (query.jobFamilyId || query.jobFunctionId || query.jobFunctionMissing)) return;
                      patch({ ...common, ...v, status: '', jobFamilyId: '', jobFunctionId: '', jobFunctionMissing: '' });
                    }
                  }}
                />
              )}
            </ListControl>
          }
        />
      </div>
    </ColumnFilterEditor.Provider>
  );
}
function filterOperators(field: string) {
  const labels: Record<string, string> = {
    contains: '包含',
    equals: '等于',
    startsWith: '开头是',
    endsWith: '结尾是',
    isEmpty: '为空',
    isNotEmpty: '不为空',
    before: '早于',
    after: '晚于',
  };
  return (
    field === 'jobFunctionId' ? ['equals', 'isEmpty'] : field === 'jobFamilyId' || field === 'status'
      ? ['equals']
      : field === 'effectiveDate'
        ? ['equals', 'before', 'after']
        : ['contains', 'equals', 'startsWith', 'endsWith', 'isEmpty', 'isNotEmpty']
  ).map((value) => ({ value, label: field === 'jobFunctionId' && value === 'isEmpty' ? '未登记职类' : labels[value] }));
}

type ClassificationFilterProps = { value: string; onChange: (id: string) => void };
function FamilyFilter(props: ClassificationFilterProps) {
  return <ClassificationFilter {...props} kind="jobfamily" />;
}
function FunctionFilter(props: ClassificationFilterProps) {
  return <ClassificationFilter {...props} kind="jobfunction" />;
}
function ClassificationFilter({ value, onChange, kind }: ClassificationFilterProps & {kind: 'jobfamily' | 'jobfunction'}) {
  const label = staff.labels[kind];
  const [search, setSearch] = useState('');
  const [items, setItems] = useState<staff.Row[]>([]);
  const [loading, setLoading] = useState(true),
    [error, setError] = useState(''),
    [retry, setRetry] = useState(0);
  useEffect(() => {
    const a = new AbortController();
    setItems([]);
    setLoading(true);
    setError('');
    void (async () => {
      const rows: staff.Row[] = [];
      for (let page = 1; ; page++) {
        const r = await staff.list(kind, { q: search, page, pageSize: 100 }, a.signal);
        rows.push(...r.items);
        if (rows.length >= r.totalCount) break;
      }
      if (!a.signal.aborted) setItems(rows);
    })()
      .catch((e) => {
        if (!a.signal.aborted) setError(e instanceof Error ? e.message : label + '暂不可读');
      })
      .finally(() => {
        if (!a.signal.aborted) setLoading(false);
      });
    return () => a.abort();
  }, [retry, kind, search]);
  return (
    <Stack spacing={1}>
      <TextField size="small" label={'查找' + label} value={search} onChange={e => setSearch(e.target.value)} />
      <TextField
        select
        size="small"
        label={label}
        slotProps={{
          select: { SelectDisplayProps: { 'aria-label': label, 'aria-labelledby': undefined } },
        }}
        value={value}
        disabled={loading || !!error}
        sx={{ minWidth: 160 }}
        onChange={(e) => onChange(e.target.value)}
      >
        <MenuItem value="">全部{label}</MenuItem>
        {value && !items.some((r) => r.id === value) && <MenuItem value={value}>已选{label}</MenuItem>}
        {items.map((r) => (
          <MenuItem key={r.id} value={r.id}>
            {r.name ?? '受限'}
          </MenuItem>
        ))}
      </TextField>
      {loading && <Typography role="status">正在读取{label}…</Typography>}
      {error && (
        <Alert severity="error" action={<Button onClick={() => setRetry((n) => n + 1)}>重试</Button>}>
          {error}
        </Alert>
      )}
      {!loading && !error && !items.length && <Typography>当前没有可读的{label}。</Typography>}
    </Stack>
  );
}
function StaffingNameLink({
  kind,
  row,
  params,
  remember,
}: {
  kind: staff.Kind;
  row: staff.Row;
  params: URLSearchParams;
  remember: () => void;
}) {
  const navigate = useNavigate();
  const href = (query: URLSearchParams) => {
    const back = new URLSearchParams(query);
    back.delete('businessAt');
    back.delete('at');
    back.set('selected', row.id);
    return `${staff.paths[kind]}/${row.id}?back=${encodeURIComponent(staff.paths[kind] + '?' + back)}`;
  };
  return (
    <Link
      data-object-link
      className="object-name-link"
      to={href(params)}
      onClick={(event) => {
        if (event.button || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
        event.preventDefault();
        remember();
        navigate(href(new URLSearchParams(window.location.search)));
      }}
    >
      {row.name ?? '查看详情'}
    </Link>
  );
}
