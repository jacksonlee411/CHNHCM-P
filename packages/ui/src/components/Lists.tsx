import { createPortal } from 'react-dom';
import { createContext, useContext, useImperativeHandle, useLayoutEffect, useMemo, useRef, useState, type ReactNode, type Ref, type RefObject } from 'react';
import { Box, Button, Checkbox, Stack, TextField, Typography, useMediaQuery } from '@mui/material';
import {
  DataGrid, GridPagination, GridPreferencePanelsValue, GRID_CHECKBOX_SELECTION_COL_DEF, GRID_CHECKBOX_SELECTION_FIELD, type GridApi, type GridColDef, type GridColumnVisibilityModel,
  type GridPaginationModel, type GridSortModel, type GridFilterModel, type GridSlotsComponent,
} from '@mui/x-data-grid';

export function ListControl({ title, action, more, search, pager, children }: {
  title: string; action?: ReactNode; more?: ReactNode; search?: ReactNode; pager?: ReactNode; children?: ReactNode;
}) {
  return <Box className="list-control"><Box className="list-control-main">
    <Box className="list-control-heading">{action}<Typography component="h1" tabIndex={-1} data-list-heading>{title}</Typography>{more}</Box>
    {search && <Box className="list-control-search">{search}</Box>}
    {pager && <Box className="list-control-pager">{pager}</Box>}
  </Box>{children}</Box>;
}
export function SearchControl({ value, onChange, onSearch, label, buttonRef }: {
  value: string; onChange: (value: string) => void; onSearch: () => void; label: string; buttonRef?: Ref<HTMLButtonElement>;
}) {
  return <Stack component="form" direction="row" gap={1} onSubmit={event => { event.preventDefault(); onSearch(); }}>
    <TextField size="small" fullWidth placeholder={label} slotProps={{ htmlInput: { 'aria-label': label } }} value={value} onChange={event => onChange(event.target.value)} />
    <Button ref={buttonRef} type="submit" size="small" variant="outlined" sx={{ minWidth: 56, flexShrink: 0, whiteSpace: 'nowrap' }}>查询</Button>
  </Stack>;
}
/** Native pagination reads the grid's single controlled server pagination model. */
export function ListPager() {
  return <Box className="list-pager"><GridPagination /></Box>;
}

const ToolbarHost = createContext<HTMLElement | null>(null);
const ToolbarContent = createContext<ReactNode>(null);
// A stable slot type keeps search inputs, menus and the pager mounted during requests.
function ListToolbar() { const content = useContext(ToolbarContent), host = useContext(ToolbarHost); return host ? createPortal(content, host) : <>{content}</>; }
// When all business columns are hidden, their recovery control lives in the toolbar.
function NoColumnsInGridBody() { return null; }
type SelectionControl = { selectable?: (id: string) => boolean; ids: Set<string>; onChange: (ids: Set<string>) => void; count?: number; checked: boolean; mixed: boolean; total: number; disabled: boolean; all: () => void; clear: () => void };
const SelectionContext = createContext<SelectionControl | undefined>(undefined);
function FullSelectionHeader() {
  const selection = useContext(SelectionContext)!;
  const checked = selection.checked;
  const mixed = selection.mixed;
  return <Checkbox size="small" checked={checked} indeterminate={mixed} disabled={selection.disabled || !selection.total}
    slotProps={{ input: { 'aria-label': checked ? '取消全选查询结果' : '全选查询结果', 'aria-checked': mixed ? 'mixed' : checked } }}
    onChange={() => checked ? selection.clear() : selection.all()} />;
}
const automaticRowHeight = () => 'auto' as const;

/** Presentation only: one authorized server page, never local sorting/filtering. */
export function DataGridPage<T extends { id: string }>({
  rows, columns, summaryColumn, selected, label, loading, ready, empty, defaultHiddenColumns,
  scrollerRef, apiRef, toolbar, messages, paginationModel, rowCount, hasNextPage,
  onPaginationModelChange, sortModel, onSortModelChange, filterModel, slots, selection, sidebar,
}: {
  defaultHiddenColumns?: string[]; rows: T[]; columns: GridColDef<T>[]; summaryColumn: GridColDef<T>;
  selected?: string | null; label: string; loading: boolean; ready: boolean; empty: string;
  scrollerRef?: Ref<HTMLDivElement>; apiRef: RefObject<GridApi | null>;
  toolbar: ReactNode; messages?: ReactNode; sidebar?: ReactNode;
  paginationModel: GridPaginationModel; rowCount?: number; hasNextPage?: boolean; sortModel: GridSortModel;
  onPaginationModelChange: (model: GridPaginationModel) => void;
  onSortModelChange: (model: GridSortModel) => void;
  filterModel?: GridFilterModel; slots?: Partial<GridSlotsComponent>; selection?: SelectionControl;
}) {
  const [toolbarHost, setToolbarHost] = useState<HTMLDivElement | null>(null);
  const compact = useMediaQuery('(max-width:899px)');
  const naturalHeight = useMediaQuery('(max-width:640px)');
  const [desktopVisibility, setDesktopVisibility] = useState<GridColumnVisibilityModel>(() => Object.fromEntries((defaultHiddenColumns ?? []).map(field => [field, false])));
  // Keep one grid and all field identities mounted. Responsive visibility never overwrites
  // desktop preferences, native widths, the toolbar input, or the authorized query state.
  const gridColumns = useMemo(() => [...(selection ? [{ ...GRID_CHECKBOX_SELECTION_COL_DEF, hideable: false, renderHeader: FullSelectionHeader }] : []), ...columns, {
    ...summaryColumn, sortable: false, filterable: false, hideable: false,
    disableColumnMenu: true, resizable: false, flex: 1, minWidth: 0,
  }], [columns, summaryColumn, Boolean(selection)]);
  const visibility = useMemo(() => compact
    ? Object.fromEntries(gridColumns.map(column => [column.field, column.field === summaryColumn.field || column.field === GRID_CHECKBOX_SELECTION_FIELD]))
    : { ...desktopVisibility, [summaryColumn.field]: false }, [compact, gridColumns, summaryColumn.field, desktopVisibility]);
  const allBusinessColumnsHidden = !compact && columns.length > 0 && columns.every(column => desktopVisibility[column.field] === false);
  const root = useRef<HTMLDivElement>(null);
  const pagerFocus = useRef<HTMLElement | null>(null);
  const focusedRow = useRef<{ id: string; link: boolean } | null>(null);
  const layout = `${compact}:${naturalHeight}`;
  const previousLayout = useRef(layout);
  useImperativeHandle(scrollerRef, () => root.current!, []);
  useLayoutEffect(() => {
    if (previousLayout.current === layout) return;
    previousLayout.current = layout;
    const target = focusedRow.current, node = root.current, api = apiRef.current;
    if (!target || !node || !api || !ready || !rows.some(row => row.id === target.id)) return;
    let frame = 0, focused = false;
    let unsubscribe: (() => void) | undefined;
    const interactions = ['pointerdown', 'keydown', 'wheel', 'touchstart'];
    const stop = () => {
      observer.disconnect(); unsubscribe?.(); cancelAnimationFrame(frame);
      interactions.forEach(name => document.removeEventListener(name, stop, true));
    };
    const restore = () => {
      if (focusedRow.current?.id !== target.id || !node.isConnected) { stop(); return; }
      const active = document.activeElement;
      if (active && active !== document.body && !node.contains(active)) { stop(); return; }
      const field = compact ? summaryColumn.field : columns.find(column => column.field === 'name' && desktopVisibility[column.field] !== false)?.field
        ?? columns.find(column => desktopVisibility[column.field] !== false)?.field;
      const cell = field ? api.getCellElement(target.id, field) : null;
      if (cell) {
        const destination = target.link ? cell.querySelector<HTMLElement>('a[data-object-link]') ?? cell : cell;
        if (!focused) {
          api.setCellFocus(target.id, field!);
          destination.focus({ preventScroll: true });
          focused = true;
        }
        const rect = destination.getBoundingClientRect();
        const bounds = naturalHeight ? { top: 0, bottom: innerHeight } : node.querySelector('.MuiDataGrid-virtualScroller')?.getBoundingClientRect();
        if (rect.top < Math.max(0, bounds?.top ?? 0) || rect.bottom > Math.min(innerHeight, bounds?.bottom ?? innerHeight)) {
          api.scrollToIndexes({ rowIndex: paginationModel.page * paginationModel.pageSize + rows.findIndex(row => row.id === target.id) });
          destination.scrollIntoView({ block: 'nearest', inline: 'nearest' });
        }
      } else {
        api.scrollToIndexes({ rowIndex: paginationModel.page * paginationModel.pageSize + rows.findIndex(row => row.id === target.id) });
      }
    };
    const schedule = () => { cancelAnimationFrame(frame); frame = requestAnimationFrame(restore); };
    const observer = new MutationObserver(schedule);
    observer.observe(node, { childList: true, subtree: true });
    // Dynamic row measurements can move a focused virtual row after the first render.
    // Keep it visible through those events; any user input immediately takes over.
    unsubscribe = api.subscribeEvent('stateChange', schedule);
    interactions.forEach(name => document.addEventListener(name, stop, { capture: true, passive: true }));
    frame = requestAnimationFrame(restore);
    return stop;
  }, [compact, naturalHeight, ready, apiRef, rows, columns, desktopVisibility, summaryColumn.field, paginationModel.page, paginationModel.pageSize]);
  useLayoutEffect(() => {
    const target = pagerFocus.current;
    if (target && (target.matches(':disabled') || !target.isConnected) &&
      (document.activeElement === document.body || document.activeElement === target)) {
      root.current?.querySelector<HTMLElement>('.list-pager [role=status]')?.focus({ preventScroll: true });
    }
  }, [loading, ready, paginationModel.page, paginationModel.pageSize, rowCount]);
  const grid = <DataGrid<T>
      className={`list-grid${compact ? ' list-grid--summary' : ''}${selection ? ' list-grid--selection' : ''}`} ref={root} apiRef={apiRef} aria-label={label}
      rows={rows} columns={gridColumns} columnVisibilityModel={visibility}
      onColumnVisibilityModelChange={model => {
        if (!compact) setDesktopVisibility(Object.fromEntries(columns.map(column => [column.field, model[column.field] ?? true])));
      }}
      autoHeight={naturalHeight} getRowHeight={automaticRowHeight}
      getEstimatedRowHeight={() => compact ? 76 : 39} columnHeaderHeight={compact ? selection ? 36 : 0 : 36}
      loading={loading} pagination paginationMode="server" sortingMode="server"
      paginationModel={paginationModel} rowCount={rowCount ?? -1} paginationMeta={hasNextPage === undefined ? undefined : { hasNextPage }} pageSizeOptions={[20, 50, 100]}
      onPaginationModelChange={model => { if (ready && !loading) onPaginationModelChange(model); }}
      sortModel={sortModel} onSortModelChange={onSortModelChange}
      rowSelectionModel={{ type: 'include', ids: selection?.ids ?? new Set() }}
      onRowSelectionModelChange={(model, details) => { if (selection && !selection.disabled && details.reason) selection.onChange(new Set([...model.ids].map(String))); }}
      getRowClassName={params => params.id === selected ? 'list-return-selected' : ''}
      checkboxSelection={Boolean(selection)} disableRowSelectionExcludeModel keepNonExistentRowsSelected
      isRowSelectable={p => !selection?.disabled && (selection?.selectable?.(String(p.id)) ?? true)}
      onCellKeyDown={(_, event) => { if (selection && (event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'a' && !(event.target as HTMLElement).closest('input:not([type=checkbox]), textarea, [contenteditable=true]')) { event.defaultMuiPrevented = true; event.preventDefault(); if (!selection.disabled) selection.all(); } }}
      disableRowSelectionOnClick
      disableColumnFilter={!filterModel} filterMode="server" filterModel={filterModel}
      hideFooter showToolbar slots={{ ...(allBusinessColumnsHidden && { noColumnsOverlay: NoColumnsInGridBody }), ...slots, toolbar: ListToolbar }}
      localeText={{
        noRowsLabel: empty, noColumnsOverlayLabel: '没有显示的列', noColumnsOverlayManageColumns: '管理列',
        checkboxSelectionHeaderName: '选择', checkboxSelectionSelectRow: `选择${label.replace(/列表$/, '')}`, checkboxSelectionUnselectRow: `取消选择${label.replace(/列表$/, '')}`,
        noResultsOverlayLabel: empty,
        columnHeaderSortIconLabel: '排序',
        filterPanelColumns: '列', filterPanelOperator: '条件', filterPanelInputLabel: '值', filterPanelInputPlaceholder: '筛选值',
        filterPanelDeleteIconLabel: '清除条件', filterValueAny: '请选择',
        columnMenuLabel: '列操作',
        columnMenuAriaLabel: name => `${name}列操作`,
        columnMenuManageColumns: '管理列', columnMenuShowColumns: '显示列', columnMenuHideColumn: '隐藏列',
        columnMenuSortAsc: '升序', columnMenuSortDesc: '降序', columnMenuUnsort: '清除排序',
        columnsManagementNoColumns: '没有匹配的列', columnsManagementSearchTitle: '搜索列', columnsManagementShowHideAllText: '显示/隐藏全部', columnsManagementReset: '重置',
        paginationRowsPerPage: '每页',
        paginationItemAriaLabel: type => ({ first: '第一页', last: '最后一页', next: '下一页', previous: '上一页' })[type],
        paginationDisplayedRows: ({ from, to, count }) => loading ? '正在查询…' : !ready ? '尚无查询结果' : rowCount === undefined ? `第${paginationModel.page + 1}页 · ${from}–${to}` : `${from}–${to} / ${count}`,
      }}
      slotProps={{ root: { onKeyDownCapture: event => {
        const target = event.target as HTMLElement;
        if (selection && (event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'a' && !target.closest('input:not([type=checkbox]), textarea, [contenteditable=true]')) {
          event.preventDefault(); event.stopPropagation(); if (!selection.disabled) selection.all();
        }
      }, onFocusCapture: event => {
        const target = event.target as HTMLElement;
        pagerFocus.current = target.closest('.list-pager') ? target : null;
        const id = target.closest('.MuiDataGrid-row')?.getAttribute('data-id');
        focusedRow.current = id ? { id, link: target.matches('a[data-object-link]') } : null;
      } }, columnsManagement: {
        getTogglableColumns: columns => columns.filter(column => column.field !== summaryColumn.field && column.field !== GRID_CHECKBOX_SELECTION_FIELD).map(column => column.field),
      }, basePagination: { disabled: loading || !ready, material: {
        slotProps: {
          select: { inputProps: { 'aria-label': '每页条数' }, SelectDisplayProps: { 'aria-labelledby': undefined } },
          displayedRows: { role: 'status', 'aria-live': 'polite', tabIndex: -1 },
        },
      } } }}
      sx={{ border: 0, borderRadius: 0, minHeight: 160, '--DataGrid-overlayHeight': '100px', '--DataGrid-containerBackground': '#fff' }}
    />;
  return <SelectionContext.Provider value={selection}><ToolbarHost.Provider value={toolbarHost}><ToolbarContent.Provider value={<>{toolbar}{messages}{allBusinessColumnsHidden && <Box className="list-applied-summary" role="status" sx={{px: 2}}>
    <Typography component="span" variant="body2">没有显示的列</Typography>
    <Button size="small" onClick={() => apiRef.current?.showPreferences(GridPreferencePanelsValue.columns)}>管理列</Button>
  </Box>}</>}>
    {sidebar ? <Box className="list-with-sidebar" sx={{display:'flex',flexDirection:'column',flex:1,minWidth:0,minHeight:0}}>
      <Box ref={setToolbarHost} />
      <Box sx={{display:'flex',flex:1,minWidth:0,minHeight:160}}>{sidebar}<Box sx={{display:'flex',flexDirection:'column',flex:1,minWidth:0,minHeight:0}}>{grid}</Box></Box>
    </Box> : grid}
  </ToolbarContent.Provider></ToolbarHost.Provider></SelectionContext.Provider>;
}
