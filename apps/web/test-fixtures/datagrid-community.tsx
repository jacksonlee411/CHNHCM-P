// Source-built grid + real app return hook. Synthetic pages, not Go/API/DB acceptance.
import { createContext, useContext, useEffect, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { createBrowserRouter, Link, RouterProvider, useSearchParams } from 'react-router-dom';
import { Box, Button, Typography } from '@mui/material';
import { GridPreferencePanelsValue, useGridApiRef, type GridColDef, type GridSortModel } from '@mui/x-data-grid';
import { AppShell, DataGridPage, ListControl, ListPager, SearchControl } from '@chnhcm/ui';
import { useListReturn } from '../src/shared/navigation';

const listPath = '/test-fixtures/datagrid-community.html';
type Row = { id: string; code: string; name: string; status: string; effectiveDate: string };
const allRows: Row[] = Array.from({ length: 73 }, (_, index) => ({
  id: `row-${index + 1}`, code: `UI-${String(index + 1).padStart(3, '0')}`,
  name: index === 57 ? '目标对象' : `合成对象 ${index + 1}`, status: '启用', effectiveDate: '2026年11月03日',
}));
const Navigation = createContext<{ href: (id: string) => string; remember: (id: string) => void }>({ href: () => '', remember: () => {} });
function Name({ row, tabIndex }: { row: Row; tabIndex: number }) {
  const navigation = useContext(Navigation);
  return <Link className="object-name-link" data-object-link tabIndex={tabIndex} to={navigation.href(row.id)} onClick={() => navigation.remember(row.id)}>{row.name}</Link>;
}
const columns: GridColDef<Row>[] = [
  { field: 'code', headerName: '编码', width: 160, sortingOrder: ['asc'] },
  { field: 'name', headerName: '名称', minWidth: 180, flex: 1, renderCell: p => <Name row={p.row} tabIndex={p.tabIndex} /> },
  { field: 'status', headerName: '状态', width: 110, sortable: false },
  { field: 'effectiveDate', headerName: '生效日期', width: 170, sortable: false },
];
const summaryColumn: GridColDef<Row> = {
  field: 'summary', headerName: '', renderCell: p => <Box className="list-row-summary">
    <Name row={p.row} tabIndex={p.tabIndex} />
    <Typography className="list-row-metadata">{p.row.code} · {p.row.status} · {p.row.effectiveDate}</Typography>
  </Box>,
};
const sortModel: GridSortModel = [{ field: 'code', sort: 'asc' }];
function Fixture() {
  const [params, setParams] = useSearchParams();
  const page = Number(params.get('page') ?? 1), pageSize = Number(params.get('pageSize') ?? 100);
  const selected = params.get('selected');
  const [ready, setReady] = useState(false), [draft, setDraft] = useState('');
  const rows = ready ? allRows.slice((page - 1) * pageSize, page * pageSize) : [];
  const apiRef = useGridApiRef();
  const { scroller, rememberRow: remember } = useListReturn({ key: 'community-return-fixture', ready, apiRef, selected, rows, pageOffset: (page - 1) * pageSize });
  useEffect(() => {
    const timer = setTimeout(() => setReady(true), params.get('slow') === '1' ? 1500 : 250);
    return () => clearTimeout(timer);
  }, []);
  const href = (id: string) => {
    const back = `${listPath}?${new URLSearchParams({ page: String(page), pageSize: String(pageSize), selected: id })}`;
    return `/test-fixtures/community-detail?${new URLSearchParams({ back })}`;
  };
  return <AppShell><Navigation.Provider value={{ href, remember }}><Box className="standard-page" data-fixture-ready={ready}>
    <DataGridPage label="Community组件接线夹具" rows={rows} columns={columns} summaryColumn={summaryColumn}
      selected={selected} apiRef={apiRef} scrollerRef={scroller} loading={!ready} ready={ready}
      rowCount={ready ? allRows.length : undefined} empty="合成页无结果"
      paginationModel={{ page: page - 1, pageSize }} onPaginationModelChange={model => {
        const next = new URLSearchParams(params); next.set('page', String(model.page + 1)); next.set('pageSize', String(model.pageSize)); setParams(next);
      }} sortModel={sortModel} onSortModelChange={() => {}}
      toolbar={<ListControl title="组件接线夹具"
        more={<Button onClick={() => apiRef.current?.showPreferences(GridPreferencePanelsValue.columns)}>列设置</Button>}
        search={<SearchControl label="等待期间输入" value={draft} onChange={setDraft} onSearch={() => {}} />}
        pager={<ListPager />} />}
      messages={ready && selected && !rows.some(row => row.id === selected)
        ? <div data-list-return-fallback tabIndex={-1}>目标不在当前授权页</div> : undefined} />
  </Box></Navigation.Provider></AppShell>;
}
function Detail() {
  const [params] = useSearchParams();
  return <AppShell><Link to={params.get('back') ?? listPath}>返回列表</Link></AppShell>;
}
createRoot(document.getElementById('root')!).render(<RouterProvider router={createBrowserRouter([
  { path: listPath, element: <Fixture /> }, { path: '/test-fixtures/community-detail', element: <Detail /> },
])} />);
