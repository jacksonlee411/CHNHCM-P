import { ManagementBUSummary } from './ManagementBUSummary';
import { useContext, useEffect, useLayoutEffect, useRef, useState, type ReactNode, type RefObject } from 'react';
import {
  Box, Button, Dialog, DialogActions, DialogContent, DialogTitle, FormControl,
  IconButton, InputLabel, Menu, MenuItem, Select, Tooltip, Typography, useMediaQuery,
} from '@mui/material';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import { ListControl, ListPager, SearchControl } from '@chnhcm/ui';
import { Link } from 'react-router-dom';
import type { GridFilterModel } from '@mui/x-data-grid';
import { ColumnFilterEditor, ColumnFilterDialog, columnFilterSummary, type FilterRequest, type FilterColumn } from './BusinessUnitColumnFilter';

export type ListStatus = 'all' | 'active' | 'inactive' | 'not_effective';
export type ListSort = 'code' | 'name' | 'name_desc';
const statuses: Record<ListStatus, string> = { all: '全部', active: '启用', inactive: '停用', not_effective: '未生效' };
const sorts: Record<ListSort, string> = { code: '编码升序', name: '名称升序', name_desc: '名称降序' };

export function BusinessUnitListControls({
  title, columns, search, appliedSearch, onSearchChange, onSearch, queryButton,
  status, sort, onApply, createHref, canCreate, rememberScroll, onManageColumns,
  filterModel, filterRequest, onCloseFilter, selectionBar, scopeSummary, navigationScope, summaryRevision, sortOptions,
}: {
  title: string; summaryRevision: string; columns?: FilterColumn[]; search: string; appliedSearch: string; onSearchChange: (value: string) => void; onSearch: () => void;
  queryButton: RefObject<HTMLButtonElement | null>;
  status: ListStatus; sort: string; sortOptions?: Record<string,string>; onApply: (values: Record<string, string>) => void;
  createHref?: string; canCreate: boolean; rememberScroll: () => void; onManageColumns: () => void;
  navigationScope?: ReactNode; selectionBar?: ReactNode; scopeSummary?: ReactNode; filterModel: GridFilterModel; filterRequest?: FilterRequest; onCloseFilter: () => void;
}) {
  const options: Record<string,string> = sortOptions ?? sorts;
  const narrow = useMediaQuery('(max-width:899px)');
  const more = useRef<HTMLButtonElement>(null);
  const openFilter = useContext(ColumnFilterEditor);
  const priorFilterRequest = useRef<FilterRequest | undefined>(undefined);
  useLayoutEffect(() => {
    if (priorFilterRequest.current && !filterRequest) {
      const source = priorFilterRequest.current.source;
      (source.isConnected ? source : more.current)?.focus();
    }
    priorFilterRequest.current = filterRequest;
  }, [filterRequest]);
  const trigger = useRef<HTMLElement | null>(null);
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);
  const [panel, setPanel] = useState<'status' | 'sort' | null>(null);
  const [draftStatus, setDraftStatus] = useState(status);
  const [draftSort, setDraftSort] = useState(sort);
  const restorePending = useRef(false);
  const awaitingApplied = useRef<Record<string, string> | null>(null);
  const restoreFocus = () => { restorePending.current = true; };
  useLayoutEffect(() => {
    if (anchor || panel || !restorePending.current) return;
    const applied: Record<string, string> = { status, sort };
    if (awaitingApplied.current && Object.entries(awaitingApplied.current).some(([key, value]) => applied[key] !== value)) return;
    awaitingApplied.current = null;
    restorePending.current = false;
    const target = trigger.current?.isConnected ? trigger.current : more.current;
    target?.focus();
  }, [anchor, panel, status, sort]);
  const closePanel = () => { setPanel(null); restoreFocus(); };
  const closeMenu = () => { setAnchor(null); trigger.current = more.current; restoreFocus(); };
  const open = (next: 'status' | 'sort', source: HTMLElement) => {
    trigger.current = source;
    setDraftStatus(Object.hasOwn(statuses, status) ? status : 'all');
    setDraftSort(Object.hasOwn(options, sort) ? sort : 'code'); setAnchor(null); setPanel(next);
  };
  const previousWidth = useRef(narrow);
  useEffect(() => {
    if (previousWidth.current === narrow) return;
    previousWidth.current = narrow;
    if (anchor || panel) { setAnchor(null); setPanel(null); trigger.current = more.current; restoreFocus(); }
  }, [narrow, anchor, panel]);
  const apply = (values: Record<string, string>) => { awaitingApplied.current = values; onApply(values); closePanel(); };
  return <ListControl title={title}
    action={createHref && <Button size="small" variant="contained" disabled={!canCreate} component={Link} onClick={event => { if (!event.ctrlKey && !event.metaKey && !event.shiftKey && !event.altKey && event.button === 0) rememberScroll(); }} to={createHref}>新建</Button>}
    more={<>
      <Tooltip title="更多操作"><IconButton ref={more} size="small" aria-label="更多操作" aria-haspopup="menu"
        aria-controls={anchor ? 'bu-list-menu' : undefined} aria-expanded={Boolean(anchor)}
        onClick={event => setAnchor(event.currentTarget)}><MoreVertIcon /></IconButton></Tooltip>
      <Menu id="bu-list-menu" anchorEl={anchor} open={Boolean(anchor)} onClose={closeMenu} disableRestoreFocus>
        <MenuItem onClick={() => open('status', more.current!)}>状态筛选…</MenuItem>
        <MenuItem onClick={() => { setAnchor(null); openFilter({ source: more.current! }); }}>列筛选…</MenuItem>
        {!narrow && <MenuItem onClick={() => { setAnchor(null); onManageColumns(); }}>列设置…</MenuItem>}
        {narrow && <MenuItem onClick={() => open('sort', more.current!)}>排序：{options[sort] ?? sort}</MenuItem>}
      </Menu>
    </>}
    search={<Box className="list-search-controls">{scopeSummary}{selectionBar ?? <SearchControl buttonRef={queryButton} label="搜索编码或名称" value={search} onChange={onSearchChange} onSearch={onSearch} />}</Box>}
    pager={<ListPager />}>
    {navigationScope}
    {(status !== 'all' || filterModel.items.length > 0) && <Box className="list-applied-summary">
      {status !== 'all' && <Button size="small" onClick={event => open('status', event.currentTarget)}>状态：{statuses[status] ?? status}</Button>}
      {filterModel.items[0] && <Button size="small" onClick={event => openFilter({ source: event.currentTarget })}>{filterModel.items[0].field === 'managementBuId' ? <ManagementBUSummary key={`${filterModel.items[0].value}:${summaryRevision}`} id={String(filterModel.items[0].value)} /> : columnFilterSummary(filterModel.items[0], columns)}</Button>}
    </Box>}
    {search !== appliedSearch && <Typography className="list-query-summary" role="status">输入已修改，点击查询后生效。已应用关键词：{appliedSearch || '无'}</Typography>}
    {filterRequest && <ColumnFilterDialog columns={columns} request={filterRequest} applied={filterModel} onApply={onApply} onClose={onCloseFilter} />}
    {(panel === 'status' || panel === 'sort') && <Dialog open onClose={closePanel} disableRestoreFocus fullWidth maxWidth="xs" aria-labelledby="bu-list-panel-title">
      <Box component="form" onSubmit={event => { event.preventDefault(); apply(panel === 'status' ? { status: draftStatus } : { sort: draftSort }); }}>
        <DialogTitle id="bu-list-panel-title">{panel === 'status' ? '状态筛选' : '排序'}</DialogTitle>
        <DialogContent>
          <FormControl fullWidth size="small" sx={{ mt: 1 }}>
            <InputLabel id="bu-list-option-label">{panel === 'status' ? '状态' : '排序方式'}</InputLabel>
            {panel === 'status' ? <Select autoFocus labelId="bu-list-option-label" label="状态" value={draftStatus} onChange={event => setDraftStatus(event.target.value)}>
              {Object.entries(statuses).map(([value, label]) => <MenuItem key={value} value={value}>{label}</MenuItem>)}
            </Select> : <Select autoFocus labelId="bu-list-option-label" label="排序方式" value={draftSort} onChange={event => setDraftSort(event.target.value)}>
              {Object.entries(options).map(([value, label]) => <MenuItem key={value} value={value}>{label}</MenuItem>)}
            </Select>}
          </FormControl>
        </DialogContent>
        <DialogActions><Button onClick={closePanel}>取消</Button><Button type="submit" variant="contained">应用</Button></DialogActions>
      </Box>
    </Dialog>}
  </ListControl>;
}
