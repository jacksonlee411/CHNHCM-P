import { useEffect, useState } from 'react';
import { Alert, Button, Dialog, DialogActions, DialogContent, DialogTitle, MenuItem, Stack, TextField } from '@mui/material';
import { useGridApiRef, type GridColDef } from '@mui/x-data-grid';
import { Link } from 'react-router-dom';
import { DataGridPage, ListControl, ListPager, SearchControl } from '@chnhcm/ui';
import { useListReturn, useMaintenanceSearchParams, usePageTitle } from '../../shared/navigation';
import { useRead } from '../../shared/useRead';
import { amountKeys, formLabels, frequencies, gradeLabel, rateAmounts, rateList, ratePath, rateTitle, statusLabels, type Principal, type Rate } from './rates';

const filters = [
  { key: 'currency', label: '币种' },
  { key: 'frequency', label: '约定频率', values: Object.entries(frequencies).map(([id, f]) => [id, f.name]) },
  { key: 'form', label: '标准形式', values: Object.entries(formLabels) },
  { key: 'status', label: '状态', values: Object.entries(statusLabels) },
] as const;
export function RateList({ session }: { session: Principal }) {
  usePageTitle(rateTitle);
  const [params, setParams] = useMaintenanceSearchParams(ratePath);
  const size = Number(params.get('pageSize')), pageNumber = Number(params.get('page'));
  const pageSize = [20,50,100].includes(size) ? size : 20;
  const page = Number.isSafeInteger(pageNumber) && pageNumber > 0 ? pageNumber : 1;
  const q = params.get('q') ?? '';
  const currentFilters = Object.fromEntries(filters.map(({ key }) => [key, params.get(key) ?? '']));
  const query = { q, ...currentFilters, page, pageSize };
  const list = useRead(JSON.stringify(query), () => rateList(query), 'fresh');
  const [search, setSearch] = useState(q);
  const [editing, setEditing] = useState<Record<string, string>>();
  useEffect(() => setSearch(q), [q]);
  const update = (changes: Record<string, string>) => {
    const next = new URLSearchParams(params);
    next.delete('selected');
    for (const [key, v] of Object.entries(changes)) { if (v) next.set(key, v); else next.delete(key); }
    setParams(next);
  };
  const grid = useGridApiRef(), rows = list.data?.items ?? [];
  const returned = useListReturn({ key: 'compensation-rates-return-' + session.instanceId + session.subjectId,
    ready: !!list.data, apiRef: grid, selected: params.get('selected'), rows, pageOffset: (page - 1) * pageSize });
  const open = (id: string) => {
    const back = new URLSearchParams(params);
    if (id !== 'new') back.set('selected', id);
    return ratePath + '/' + id + '?back=' + encodeURIComponent(ratePath + (back.size ? '?' + back : ''));
  };
  const link = (r: Rate) => <Link data-object-link className="object-name-link" to={open(r.id)} onClick={() => returned.rememberRow(r.id)}>{gradeLabel(r.record) || '查看标准'}</Link>;
  const can = (field: string) => !!list.data?.readable.includes(field);
  const columns: GridColDef<Rate>[] = [
    { field: 'gradeId', headerName: '职级', flex: 1, minWidth: 150, renderCell: ({ row }) => link(row) },
  ];
  const column = (field: string, headerName: string, value: (r: Rate) => string | undefined) => {
    columns.push({ field, headerName, minWidth: 100, flex: 1, valueGetter: (_, row) => value(row) ?? '' });
  };
  if (can('currency')) column('currency', '币种', r => r.currency);
  if (can('frequency')) column('frequency', '约定频率', r => frequencies[r.frequency ?? '']?.name);
  if (can('form')) column('form', '标准形式', r => formLabels[r.record.form ?? '']);
  if (amountKeys.some(can)) column('amounts', '金额或区间', r => rateAmounts(r.record));
  if (can('effectiveDate')) column('effectiveDate', '生效日期', r => r.record.effectiveDate);
  if (can('status')) column('status', '状态', r => statusLabels[r.record.status ?? '']);
  const available = filters.filter(f => can(f.key));
  const active = available.filter(({ key }) => !!currentFilters[key]).length;
  return <div className="standard-page" aria-busy={!list.data && !list.error}>
    <DataGridPage apiRef={grid} scrollerRef={returned.scroller} rows={rows} selected={returned.selected} label={rateTitle}
      columns={columns.map(c => ({ ...c, sortable: false, filterable: false }))}
      summaryColumn={{ field: 'summary', headerName: rateTitle, renderCell: ({ row }) => <Stack>{link(row)}
        <span>{[row.currency, frequencies[row.frequency ?? '']?.unitName, rateAmounts(row.record), row.record.effectiveDate, statusLabels[row.record.status ?? '']].filter(Boolean).join(' · ')}</span></Stack> }}
      paginationModel={{ page: page - 1, pageSize }} rowCount={list.data?.total ?? (list.error ? 0 : undefined)} hasNextPage={list.data ? page * pageSize < list.data.total : undefined} onPaginationModelChange={m => update({ page: String(m.page + 1), pageSize: String(m.pageSize) })}
      sortModel={[]} onSortModelChange={() => {}} loading={!list.data && !list.error} ready={!!list.data} empty="没有符合条件的职级薪酬标准"
      messages={list.error && <Alert severity="error" action={<Button onClick={list.refresh}>重试</Button>}>{list.error}</Alert>}
      toolbar={<ListControl title={rateTitle} action={list.data?.create && <Button component={Link} to={open('new')} variant="contained" onClick={returned.remember}>新建</Button>}
        search={can('gradeId') && <SearchControl value={search} onChange={setSearch} onSearch={() => update({ q: search.trim(), page: '1' })} label="搜索职级编码或名称" />}
        more={available.length > 0 && <Button onClick={() => setEditing(currentFilters)}>筛选{active ? '（' + active + '）' : ''}</Button>} pager={<ListPager />} />} />
    {editing && <Dialog open onClose={() => setEditing(undefined)} fullWidth maxWidth="xs"><DialogTitle>筛选职级薪酬标准</DialogTitle>
      <DialogContent><Stack spacing={2} sx={{ pt: 1 }}>{available.map(f => <TextField key={f.key} label={f.label} select={'values' in f}
        value={editing[f.key] ?? ''} onChange={e => setEditing({ ...editing, [f.key]: e.target.value })}>
        {'values' in f ? [<MenuItem key="" value="">全部</MenuItem>, ...f.values.map(([id, name]) => <MenuItem key={id} value={id}>{name}</MenuItem>)] : undefined}
      </TextField>)}</Stack></DialogContent><DialogActions><Button onClick={() => setEditing({})}>清空</Button>
        <Button onClick={() => setEditing(undefined)}>取消</Button><Button variant="contained" onClick={() => {
          update({ ...Object.fromEntries(available.map(f => [f.key, editing[f.key] ?? ''])), page: '1' }); setEditing(undefined);
        }}>应用</Button></DialogActions></Dialog>}
  </div>;
}
