import { useEffect, useMemo, useRef, useState } from 'react';
import { Link, useLocation, useNavigate, useSearchParams } from 'react-router-dom';
import { Alert, Button, MenuItem, Stack, Typography } from '@mui/material';
import { useGridApiRef, type GridColDef } from '@mui/x-data-grid';
import { DataGridPage, ListControl, ListPager, SearchControl, ObjectForm, ObjectTextField } from '@chnhcm/ui';
import { api, sessionGuard, unwrap, type Models } from '@chnhcm/api-client';
import { usePageTitle, useListReturn } from '../../shared/navigation';
import { QuotaDepartmentLookup } from './QuotaDepartmentLookup';
import { positionQuotaPath, quotaPath, quotaLabel, quotaTotals, type Position } from './quotaApi';

function showsDepartmentRows(list: Models['StaffingQuotaList'], departmentId: string, search: string) {
  return !departmentId && !search.trim() && !list.positionSummary && (list.departments.length > 0 || list.canBrowseDepartments);
}

export function QuotaList() {
  usePageTitle('部门编制统计');
  const [params, setParams] = useSearchParams(),
    key = params.toString(),
    grid = useGridApiRef(),
    navigate = useNavigate();
  const location = useLocation(),
    departmentLink = useRef<HTMLAnchorElement>(null),
    restored = useRef(''),
    dateField = useRef<HTMLInputElement>(null);
  const queryDay = params.get('businessAt') ?? '';
  const querySearch = params.get('q') ?? '';
  const [draftDay, setDraftDay] = useState<string>();
  const [draftSearch, setDraftSearch] = useState<string>();
  const lastTotal = useRef(0);
  const [data, setData] = useState<{ key: string; list: Models['StaffingQuotaList'] }>(),
    [error, setError] = useState(''),
    [retry, setRetry] = useState(0);
  const [findDepartment, setFindDepartment] = useState(false);
  const page = Number(params.get('page') ?? 1),
    pageSize = Number(params.get('pageSize') ?? 20),
    dep = params.get('departmentId') ?? '';
  const patch = (values: Record<string, string>) => {
    if (values.q !== undefined) setDraftSearch(undefined);
    if (values.businessAt !== undefined) setDraftDay(undefined);
    // A department menu can still be open as the preceding date navigation commits.
    const next = new URLSearchParams(window.location.search);
    for (const [k, v] of Object.entries(values)) {
      if (v) next.set(k, v);
      else next.delete(k);
    }
    if (next.toString() === window.location.search.slice(1)) return;
    setData(undefined);
    setParams(next, { flushSync: true });
  };
  useEffect(() => {
    const abort = new AbortController(),
      same = sessionGuard();
    setData(undefined);
    setError('');
    api
      .GET('/api/v1/staffing-quotas/statistics', {
        params: {
          query: {
            q: params.get('q') ?? '',
            businessAt: params.get('businessAt') ?? '',
            departmentId: dep,
            page,
            pageSize,
          },
        },
        signal: abort.signal,
      })
      .then(unwrap)
      .then((list) => {
        if (!abort.signal.aborted && same()) {
          setData({ key, list });
          lastTotal.current = showsDepartmentRows(list, dep, querySearch) ? list.departments.length : list.totalCount;
        }
      })
      .catch((e) => {
        if (!abort.signal.aborted && same()) setError(e.message);
      });
    return () => abort.abort();
  }, [key, retry]);
  const view = data?.key === key ? data.list : undefined,
    group = view?.departments.find((x) => x.departmentId === dep);
  const day = draftDay ?? (queryDay || view?.businessAt || '');
  const departmentQuery = new URLSearchParams(params);
  if (view) departmentQuery.set('businessAt', view.businessAt);
  useEffect(() => {
    if (group && location.state?.restoreDepartment === dep && restored.current !== location.key) {
      departmentLink.current?.focus();
      restored.current = location.key;
    }
  }, [group, location, dep]);
  const search = draftSearch ?? querySearch;
  type Row = Partial<Position> & { id: string; summary?: Models['StaffingQuotaDepartment'] };
  const grouped = view && showsDepartmentRows(view, dep, querySearch);
  const offset = (page - 1) * pageSize;
  // The response carries the complete authorized department totals for the selector.
  const rows: Row[] = grouped
    ? (view?.departments.slice(offset, offset + pageSize) ?? []).map((d) => ({
        id: d.departmentId,
        name: d.name,
        summary: d,
      }))
    : (view?.items ?? []).map((r) => ({ ...r, id: r.positionId }));
  const returned = useListReturn({
    key: 'staffing-quota-return',
    ready: !!view,
    apiRef: grid,
    selected: params.get('selected'),
    rows,
    pageOffset: (page - 1) * pageSize,
  });
  const link = (r: Row) => {
    const target = (query: URLSearchParams) => {
      const next = new URLSearchParams(query);
      if (r.summary) {
        next.set('departmentId', r.id);
        next.set('page', '1');
        next.delete('selected');
        return quotaPath + '?' + next;
      }
      next.set('selected', r.id);
      const target = new URLSearchParams({ back: quotaPath + '?' + next });
      if (r.recordId) target.set('selectedRecordId', r.recordId);
      return positionQuotaPath(r.id, target);
    };
    return (
      <Link
        data-object-link
        to={target(params)}
        onClick={(event) => {
          if (event.button || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
          event.preventDefault();
          returned.rememberRow(r.id, true);
          navigate(target(new URLSearchParams(window.location.search)));
        }}
      >
        {r.name ?? r.code ?? '查看岗位'}
      </Link>
    );
  };
  const scopes = useMemo(() => new Set(view?.departments.map((department) => department.scope)), [view]);
  const totalHeader = scopes.size > 1 ? '岗位核定编制' : quotaTotals[view?.departments[0]?.scope ?? 'complete'];
  const columns: GridColDef<Row>[] = grouped
    ? [
        { field: 'name', headerName: '部门', minWidth: 160, flex: 1, renderCell: (p) => link(p.row) },
        {
          field: 'scope',
          headerName: '统计范围',
          minWidth: 110,
          valueGetter: (_, r) => (r.summary?.scope === 'complete' ? '完整范围' : '可见范围'),
        },
        {
          field: 'total',
          headerName: totalHeader,
          minWidth: 200,
          valueGetter: (_, row) => row.summary?.total ?? '受限',
          renderCell: ({ row, value }) => scopes.size > 1 && row.summary
            ? quotaTotals[row.summary.scope] + ' ' + value
            : value,
        },
        {
          field: 'unsetCount',
          headerName: '未核编岗位数',
          minWidth: 135,
          valueGetter: (_, r) => r.summary?.unsetCount ?? '受限',
        },
        {
          field: 'quota',
          headerName: '部门总额',
          minWidth: 110,
          valueGetter: (_, r) => (r.summary?.scope === 'complete' ? quotaLabel(r.summary, 'department') : '受限'),
        },
        {
          field: 'remaining',
          headerName: '待分配',
          minWidth: 100,
          valueGetter: (_, r) => (r.summary?.scope === 'complete' ? (r.summary.remaining ?? '—') : '受限'),
        },
      ]
    : [
        { field: 'code', headerName: '岗位编码', minWidth: 140 },
        { field: 'name', headerName: '岗位名称', minWidth: 160, flex: 1, renderCell: (p) => link(p.row) },
        { field: 'departmentName', headerName: '所属部门', minWidth: 140 },
        { field: 'quota', headerName: '核定编制', minWidth: 100, renderCell: (p) => quotaLabel(p.row) },
        {
          field: 'active',
          headerName: '岗位状态',
          minWidth: 100,
          renderCell: (p) => (p.row.active === undefined ? '受限' : p.row.active ? '启用' : '停用'),
        },
      ];
  return (
    <div className="standard-page" aria-busy={!view && !error}>
      <DataGridPage
        rows={rows}
        selected={returned.selected}
        scrollerRef={returned.scroller}
        columns={columns.map((c) => ({ ...c, sortable: false, filterable: false }))}
        summaryColumn={{
          field: 'summary',
          headerName: grouped ? '部门' : '岗位',
          renderCell: (p) => (
            <Stack>
              {link(p.row)}
              <Typography variant="body2">
                {(p.row.summary
                  ? [
                      quotaTotals[p.row.summary.scope],
                      p.row.summary.total ?? '受限',
                      p.row.summary.unsetCount !== undefined
                        ? '未核编 ' + p.row.summary.unsetCount + ' 岗'
                        : '',
                    ]
                  : [p.row.code, p.row.departmentName, '核定编制 ' + quotaLabel(p.row)]
                )
                  .filter((v) => v !== undefined && v !== '')
                  .join(' · ')}
              </Typography>
            </Stack>
          ),
        }}
        label="部门编制统计"
        apiRef={grid}
        ready={!!view}
        loading={!view && !error}
        empty={grouped ? '没有可见的部门编制统计' : '没有匹配的岗位'}
        paginationModel={{ page: page - 1, pageSize }}
        rowCount={error ? 0 : lastTotal.current}
        onPaginationModelChange={(p) => {
          if (view && (page !== p.page + 1 || pageSize !== p.pageSize))
            patch({ page: String(p.page + 1), pageSize: String(p.pageSize) });
        }}
        sortModel={[]}
        onSortModelChange={() => {}}
        toolbar={
          <ListControl
            title="部门编制统计"
            search={
              <SearchControl
                value={search}
                onChange={setDraftSearch}
                label="搜索岗位编码或名称"
                onSearch={() => patch({ q: search, page: '1' })}
              />
            }
            pager={<ListPager />}
          >
            <ObjectForm>
              <Stack direction={{ xs: 'column', md: 'row' }} gap={1} alignItems={{ md: 'center' }}>
                <ObjectTextField
                  type="date"
                  inputRef={dateField}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') patch({ businessAt: dateField.current?.value ?? day, page: '1' });
                  }}
                  label="统计日期"
                  value={day}
                  onChange={(e) => setDraftDay(e.target.value)}
                />
                <Button onClick={() => patch({ businessAt: dateField.current?.value ?? day, page: '1' })}>
                  查询统计
                </Button>
                <ObjectTextField
                  select
                  label="所属部门"
                  value={dep}
                  onChange={(e) => {
                    if (e.target.value === 'search') setFindDepartment(true);
                    else patch({ departmentId: e.target.value, page: '1' });
                  }}
                >
                  <MenuItem value="">全部部门</MenuItem>
                  {view?.departments.map((d) => (
                    <MenuItem key={d.departmentId} value={d.departmentId}>
                      {d.name}
                    </MenuItem>
                  ))}
                  {view?.canBrowseDepartments && <MenuItem value="search">查找其他部门…</MenuItem>}
                </ObjectTextField>
                {findDepartment && view?.canBrowseDepartments && (
                  <QuotaDepartmentLookup
                    day={view.businessAt}
                    onClose={() => setFindDepartment(false)}
                    onSelect={(id) => {
                      setFindDepartment(false);
                      patch({ departmentId: id, page: '1' });
                    }}
                  />
                )}
                {group && (
                  <Stack direction="row" alignItems="center" flexWrap="wrap" gap={1}>
                    <Typography>
                      {group.scope === 'complete' && '部门总额 ' + quotaLabel(group, 'department') + ' · '}
                      {quotaTotals[group.scope]}{' '}
                      {group.total ?? '受限'}
                      {group.remaining !== undefined && ' · 待分配 ' + group.remaining}
                      {group.unsetCount !== undefined && ' · 未核编 ' + group.unsetCount + ' 岗'}
                    </Typography>
                    <Button
                      ref={departmentLink}
                      component={Link}
                      to={quotaPath + '/departments/' + group.departmentId + '?' + departmentQuery}
                    >
                      部门编制
                    </Button>
                  </Stack>
                )}
                {view?.positionSummary?.total !== undefined && (
                  <Typography>{quotaTotals.visible} {view.positionSummary.total}</Typography>
                )}
              </Stack>
            </ObjectForm>
          </ListControl>
        }
        messages={
          error && (
            <Alert severity="error" action={<Button onClick={() => setRetry((n) => n + 1)}>重试</Button>}>
              {error}
            </Alert>
          )
        }
      />
    </div>
  );
}
