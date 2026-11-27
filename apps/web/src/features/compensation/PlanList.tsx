import { useEffect, useState } from 'react';
import { Alert, Button, MenuItem, Stack, TextField } from '@mui/material';
import { useGridApiRef, type GridColDef } from '@mui/x-data-grid';
import { Link } from 'react-router-dom';
import { DataGridPage, ListControl, ListPager, SearchControl } from '@chnhcm/ui';
import { useListReturn, useMaintenanceSearchParams, usePageTitle } from '../../shared/navigation';
import { useRead } from '../../shared/useRead';
import { type Principal, statusLabels } from './rates';
import { listPlans, planPath, planTitle, scopeLabel, type Plan } from './plans';

export function PlanList({ session }: { session: Principal }) {
  usePageTitle(planTitle);
  const [params, setParams] = useMaintenanceSearchParams(planPath);
  const q = params.get('q') ?? '',
    status = params.get('status') ?? '';
  const [draft, setDraft] = useState(q);
  useEffect(() => setDraft(q), [q]);
  const size = Number(params.get('pageSize')),
    rawPage = Number(params.get('page'));
  const pageSize = [20, 50, 100].includes(size) ? size : 20,
    page = Number.isSafeInteger(rawPage) && rawPage > 0 ? rawPage : 1;
  const list = useRead(
    JSON.stringify([q, status, page, pageSize]),
    () => listPlans({ q, status, page, pageSize }),
    'fresh',
  );
  const rows = list.data?.items ?? [],
    grid = useGridApiRef();
  const returned = useListReturn({
    key: 'compensation-plans-return-' + session.instanceId + session.subjectId,
    ready: !!list.data,
    apiRef: grid,
    selected: params.get('selected'),
    rows,
    pageOffset: (page - 1) * pageSize,
  });
  const change = (fields: Record<string, string>) => {
    const next = new URLSearchParams(params);
    next.delete('selected');
    for (const [k, v] of Object.entries(fields)) {
      if (v) next.set(k, v);
      else next.delete(k);
    }
    setParams(next);
  };
  const to = (id: string) => {
    const back = new URLSearchParams(params);
    if (id !== 'new') back.set('selected', id);
    return planPath + '/' + id + '?back=' + encodeURIComponent(planPath + (back.size ? '?' + back : ''));
  };
  const link = (p: Plan) => (
    <Link data-object-link className="object-name-link" to={to(p.id)} onClick={() => returned.rememberRow(p.id)}>
      {p.record.name || p.code || '查看计划'}
    </Link>
  );
  const can = (k: string) => !!list.data?.readable.includes(k);
  const cols: GridColDef<Plan>[] = [
    { field: 'name', headerName: '计划名称', minWidth: 160, flex: 1.5, renderCell: ({ row }) => link(row) },
  ];
  if (can('code')) cols.push({ field: 'code', headerName: '计划编码', flex: 1, minWidth: 120 });
  if (can('scope'))
    cols.push({
      field: 'scope',
      headerName: '适用对象',
      flex: 2,
      minWidth: 165,
      valueGetter: (_, row) => scopeLabel(row.record.scope),
    });
  if (can('effectiveDate'))
    cols.push({
      field: 'effectiveDate',
      headerName: '生效日期',
      flex: 1,
      minWidth: 115,
      valueGetter: (_, row) => row.record.effectiveDate,
    });
  if (can('status'))
    cols.push({
      field: 'status',
      headerName: '状态',
      flex: 1,
      minWidth: 90,
      valueGetter: (_, row) => statusLabels[row.record.status ?? ''],
    });
  return (
    <div className="standard-page">
      <DataGridPage
        apiRef={grid}
        scrollerRef={returned.scroller}
        label={planTitle}
        selected={returned.selected}
        rows={rows}
        columns={cols.map((v) => ({ ...v, sortable: false }))}
        summaryColumn={{
          field: 'plan',
          headerName: planTitle,
          flex: 1,
          renderCell: ({ row }) => (
            <Stack>
              {link(row)}
              <span>
                {[row.code, scopeLabel(row.record.scope), row.record.effectiveDate].filter(Boolean).join(' · ')}
              </span>
            </Stack>
          ),
        }}
        paginationModel={{ page: page - 1, pageSize }}
        rowCount={list.data?.total ?? (list.error ? 0 : undefined)}
        hasNextPage={list.data ? page * pageSize < list.data.total : undefined}
        onPaginationModelChange={(m) => change({ page: String(m.page + 1), pageSize: String(m.pageSize) })}
        sortModel={[]}
        onSortModelChange={() => {}}
        loading={!list.data && !list.error}
        ready={!!list.data}
        empty="没有符合条件的薪酬计划"
        messages={
          list.error && (
            <Alert severity="error" action={<Button onClick={list.refresh}>重试</Button>}>
              {list.error}
            </Alert>
          )
        }
        toolbar={
          <ListControl
            title={planTitle}
            action={
              list.data?.create && (
                <Button component={Link} to={to('new')} onClick={returned.remember} variant="contained">
                  新建
                </Button>
              )
            }
            search={
              (can('name') || can('code')) && (
                <SearchControl
                  label="搜索计划编码或名称"
                  value={draft}
                  onChange={setDraft}
                  onSearch={() => change({ q: draft.trim(), page: '1' })}
                />
              )
            }
            more={
              can('status') && (
                <TextField
                  select
                  size="small"
                  label="状态"
                  value={status}
                  sx={{ minWidth: 102 }}
                  onChange={(e) => change({ status: e.target.value, page: '1' })}
                >
                  <MenuItem value="">全部</MenuItem>
                  {Object.entries(statusLabels).map(([k, v]) => (
                    <MenuItem key={k} value={k}>
                      {v}
                    </MenuItem>
                  ))}
                </TextField>
              )
            }
            pager={<ListPager />}
          />
        }
      />
    </div>
  );
}
