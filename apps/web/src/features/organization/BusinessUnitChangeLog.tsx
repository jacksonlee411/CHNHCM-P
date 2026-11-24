import { useDefinition, type Change, type Changes } from './dated/presentation';
import { useEffect, useLayoutEffect, useRef, useState } from 'react';
import { Alert, Box, Button, CircularProgress, IconButton, Menu, MenuItem, Stack, Table, TableBody, TableCell, TableContainer, TableHead, TablePagination, TableRow, Typography } from '@mui/material';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import { Link, useLocation, useSearchParams } from 'react-router-dom';
import { ApiError, sessionGuard, unwrap, type Models } from '@chnhcm/api-client';
import { recordedTime } from '@chnhcm/ui';
import { ColumnFilterDialog, columnFilterModel, operatorsFor, type FilterColumn, type FilterRequest } from './BusinessUnitColumnFilter';
import { canReadStatus } from './businessUnitProjection';
import { OperationGroup, OperationSummary, operatorText } from '../../shared/dated/OperationLog';
export {OperationSummary, operatorText};

export const logKeys = ['logPage', 'logPageSize', 'logThrough', 'logFilterField', 'logFilterOperator', 'logFilterValue', 'logReturn'];
export const changeActions: Record<string, string> = { CREATE: '新建', ADD: '新增', CORRECT: '更正记录', REDATE: '改日', DELETE: '删除', DEACTIVATE: '停用', REACTIVATE: '重新启用', REVISE_TIMELINE: '联合修订' };
const baseFieldLabels: Record<string, string> = { name: '中文名称', nameEn: '英文名称', effectiveDate: '生效日期', period: '适用期间', state: '状态', operationType: '操作类型', operationReason: '操作原因' };
const returnPositions = new Map<string, { focus: string; top: number; left: number }>();
function scrollNode() {
  const node = document.querySelector<HTMLElement>('[data-page-scroll]');
  return node && getComputedStyle(node).overflowY === 'auto' ? node : document.scrollingElement;
}
export function logReturnHref(value: string | null, pathname: string) {
  if (!value) return undefined;
  try {
    const u = new URL(value, window.location.origin);
    return u.origin === window.location.origin && u.pathname === pathname && u.searchParams.get('tab') === 'history' ? u.pathname + u.search : undefined;
  } catch { return undefined; }
}
export function BusinessUnitChangeLog({ id, subjectKey, onReady, onReadFields, onDenied }: { id: string; subjectKey: string; onReady: (ready: boolean) => void; onReadFields: (fields: string[]) => void; onDenied: (message: string) => void }) {
  const [params, setParams] = useSearchParams(), location = useLocation();
  const definition = useDefinition();
  const fieldLabels: Record<string, string> = { ...baseFieldLabels, shortName: '简称', managementBuId: '管理业务单元', parentDepartmentId: '行政上级', ...Object.fromEntries((definition.summaryFields??definition.fields).map(f => [f.key, f.label])) };
  if (definition.logStateField === 'status') { delete fieldLabels.state; fieldLabels.status = '状态'; }
  const [result, setResult] = useState<{ key: string; value: Changes }>();
  const [invalid, setInvalid] = useState(false);
  const [invalidKeys, setInvalidKeys] = useState<string[]>([]);
  const [error, setError] = useState(''), [retry, setRetry] = useState(0);
  const [request, setRequest] = useState<FilterRequest>();
  const [menu, setMenu] = useState<{ field: string; source: HTMLElement }>();
  const fallback = useRef<HTMLButtonElement>(null);
  const root = useRef<HTMLDivElement>(null);
  const pendingFocus = useRef<{ source: HTMLElement; label: string | null; select: boolean; started: boolean } | undefined>(undefined);
  const settledKey = useRef('');
  const callbacks = useRef({ onReady, onReadFields, onDenied }); callbacks.current = { onReady, onReadFields, onDenied };
  const filters = new URLSearchParams();
  for (const key of ['Field', 'Operator', 'Value']) if (params.has(`logFilter${key}`)) filters.set(`filter${key}`, params.get(`logFilter${key}`)!);
  const model = columnFilterModel(filters);
  const query = {
    page: params.has('logPage') ? Number(params.get('logPage')) : 1,
    pageSize: (params.has('logPageSize') ? Number(params.get('logPageSize')) : 20) as 20 | 50 | 100,
    contextSourceVersion: params.get('sourceVersion') ?? undefined,
    throughSourceVersion: params.get('logThrough') ?? undefined,
    filterField: filters.get('filterField') as 'recordedAt' | 'actor' | 'action' | 'effectiveDate' | 'changedField' | undefined ?? undefined,
    filterOperator: filters.get('filterOperator') as 'contains' | 'equals' | 'is' | 'isAnyOf' | 'before' | 'after' | 'onOrBefore' | 'onOrAfter' | undefined ?? undefined,
    filterValue: filters.get('filterValue') ?? undefined,
  };
  const key = `${id}/${subjectKey}/${JSON.stringify(query)}/${retry}`;
  const data = result?.key === key ? result.value : undefined;
  const fields = data?.readFields ?? [];
  const columns: FilterColumn[] = [
    { field: 'recordedAt', label: '记录时间（按日期）' }, { field: 'actor', label: '操作者' },
    { field: 'action', label: '记录处理', values: Object.entries(changeActions).map(([value, label]) => ({ value, label })) },
    ...(fields.includes('effectiveDate') ? [{ field: 'effectiveDate', label: '生效记录' }] : []),
    { field: 'changedField', label: '变更字段', values: Object.entries(fieldLabels).filter(([field]) => field === (definition.logStateField ?? 'state') ? canReadStatus(fields, definition.statusReadField) : fields.includes(field === 'period' ? 'effectiveDate' : field)).map(([value, label]) => ({ value, label })) },
  ];
  const currentURL = location.pathname + location.search;
  const positionKey = `${subjectKey}:${currentURL}`;
  useEffect(() => {
    let active = true; const abort = new AbortController(), sameSession = sessionGuard();
    // Navigation can update the browser URL before this effect is cleaned up.
    // A late response must not restore its old tab, history source or filters.
    const ownsLocation = () => active && sameSession() && window.location.pathname + window.location.search === currentURL;
    if (pendingFocus.current) pendingFocus.current.started = true;
    setResult(undefined); setError(''); setInvalid(false); callbacks.current.onReady(false);
    void (async () => {
      try {
        if (logKeys.some(k => params.getAll(k).length > 1)) throw new ApiError('INVALID_QUERY', '日志查询条件重复，请重置条件', 400);
        const value = unwrap(await definition.api.changes(id, query, abort.signal));
        if (!ownsLocation()) return;
        callbacks.current.onReadFields(value.detailReadFields);
        // Persist the server's snapshot and clamped page in the same URL used by
        // pagination, filters, history links and browser back.
        if (!params.has('logThrough') || value.page !== query.page) {
          const p = new URLSearchParams(params); p.set('logThrough', value.throughSourceVersion); p.set('logPage', String(value.page)); setParams(p, { replace: true });
        } else { settledKey.current = key; setResult({ key, value }); callbacks.current.onReady(true); }
      } catch (e) {
        if (!ownsLocation()) return;
        settledKey.current = key; setInvalid(e instanceof ApiError && e.status === 400);
        const duplicates = logKeys.filter(k => params.getAll(k).length > 1).flatMap(k => k.startsWith('logFilter') ? ['logFilterField', 'logFilterOperator', 'logFilterValue'] : [k]);
        const fields: Record<string, string[]> = { page: ['logPage'], pageSize: ['logPageSize'], throughSourceVersion: ['logThrough'], filter: ['logFilterField', 'logFilterOperator', 'logFilterValue'] };
        setInvalidKeys(duplicates.length ? duplicates : e instanceof ApiError && e.code === 'M02_INVALID_LOG_RANGE' ? ['logThrough'] : e instanceof ApiError && e.field && fields[e.field] ? fields[e.field] : ['logFilterField', 'logFilterOperator', 'logFilterValue']); setError(e instanceof ApiError ? e.message : '读取变更日志失败'); callbacks.current.onReady(true);
        if (e instanceof ApiError && [401, 403, 404].includes(e.status) && e.code !== 'AUTH_REQUEST_VALIDATION_FAILED') callbacks.current.onDenied(e.message);
      }
    })();
    return () => { active = false; abort.abort(); };
  }, [key, currentURL]);
  useLayoutEffect(() => {
    if (!data) return;
    const saved = returnPositions.get(positionKey);
    if (saved) {
      const frame = requestAnimationFrame(() => {
        const node = scrollNode(); if (node) node.scrollTop = saved.top;
        const table = document.querySelector('.change-log-table'); if (table) table.scrollLeft = saved.left;
        (document.getElementById(saved.focus) ?? fallback.current)?.focus({ preventScroll: true });
        returnPositions.delete(positionKey);
      });
      return () => cancelAnimationFrame(frame);
    }
  }, [data, positionKey]);
  useLayoutEffect(() => {
    if (!pendingFocus.current?.started || settledKey.current !== key || (!data && !error)) return;
    const frame = requestAnimationFrame(() => {
      const pending = pendingFocus.current;
      if (!pending) return;
      pendingFocus.current = undefined;
      const active = document.activeElement;
      // A user who moved focus while the request was pending keeps control.
      if (active && active !== document.body && active !== pending.source && active !== fallback.current) return;
      const replacement = pending.select ? root.current?.querySelector<HTMLElement>('[role="combobox"]')
        : [...root.current?.querySelectorAll<HTMLElement>('[aria-label]') ?? []].find(node => node.getAttribute('aria-label') === pending.label);
      const target = error ? root.current?.querySelector<HTMLElement>('[data-log-retry]')
        : pending.source.isConnected ? pending.source : replacement;
      (target && !target.matches(':disabled, [aria-disabled="true"]') ? target : fallback.current)?.focus({ preventScroll: true });
    });
    return () => cancelAnimationFrame(frame);
  });
  const update = (values: Record<string, string>, refresh = false, origin?: HTMLElement | null) => {
    const p = new URLSearchParams(params);
    for (const [k, v] of Object.entries(values)) v ? p.set(k, v) : p.delete(k);
    if (refresh) p.delete('logThrough');
    const source = origin ?? request?.source ?? document.activeElement;
    if (p.toString() !== params.toString() && source instanceof HTMLElement && root.current?.contains(source)) {
      pendingFocus.current = { source, label: source.getAttribute('aria-label'), select: source.getAttribute('role') === 'combobox', started: false };
    }
    setParams(p);
  };
  const href = (source: string, record?: string) => {
    const p = new URLSearchParams(); p.set('sourceVersion', source); if (record) p.set('selectedRecordId', record);
    if (params.has('back')) p.set('back', params.get('back')!);
    p.set('logReturn', currentURL); return `${location.pathname}?${p}`;
  };
  const remember = (focus: string, event: React.MouseEvent) => {
    if (event.button !== 0 || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) return;
    if (returnPositions.size >= 100) returnPositions.clear();
    returnPositions.set(positionKey, { focus, top: scrollNode()?.scrollTop ?? 0, left: document.querySelector('.change-log-table')?.scrollLeft ?? 0 });
  };
  const closeFilter = () => { const source = request?.source; setRequest(undefined); if (!pendingFocus.current) requestAnimationFrame(() => (source?.isConnected ? source : fallback.current)?.focus()); };
  const applied = model.items[0], appliedColumn = columns.find(c => c.field === applied?.field);
  const appliedValues = applied ? (Array.isArray(applied.value) ? applied.value : [applied.value]) : [];
  return <Box ref={root} className="change-log">
    <Stack direction="row" flexWrap="wrap" gap={1} alignItems="center" sx={{ mb: 1 }}>
      <Button ref={fallback} disabled={!data} size="small" onClick={e => setRequest({ source: e.currentTarget })}>列筛选…</Button>
      <Button size="small" onClick={() => update({ logPage: '1' }, true)}>刷新日志</Button>
      {applied && <Button size="small" onClick={e => setRequest({ source: e.currentTarget })}>{appliedColumn?.label} {operatorsFor(applied.field).find(o => o.value === applied.operator)?.label} {appliedValues.map(v => appliedColumn?.values?.find(o => o.value === v)?.label ?? v).join('、')}</Button>}
    </Stack>
    {error && <Alert severity="error" action={<Stack direction="row">{invalid && <Button onClick={() => update(Object.fromEntries(invalidKeys.map(key => [key, key === 'logPage' ? '1' : ''])), true)}>重置日志条件</Button>}<Button data-log-retry onClick={event => { pendingFocus.current = { source: event.currentTarget, label: null, select: false, started: false }; setRetry(n => n + 1); }}>重试</Button></Stack>}>{error}</Alert>}
    {!data && !error && <CircularProgress size={24} aria-label="加载变更日志" />}
    {data && <>
      <TableContainer className="change-log-table" tabIndex={0} aria-label="变更日志表格">
        <Table size="small" aria-label="变更日志">
          <TableHead><TableRow>{[['recordedAt', '记录时间'], ['actor', '操作者'], ['action', '记录处理'], ['effectiveDate', '生效记录'], ['changedField', '变更字段'], ['', '原值'], ['', '新值'], ['', '查看']].map(([field, label]) => <TableCell key={label} scope="col">{label}{columns.some(c => c.field === field) && <IconButton size="small" aria-label={`${label}列菜单`} aria-haspopup="menu" onClick={e => setMenu({ field, source: e.currentTarget })}><MoreVertIcon fontSize="small" /></IconButton>}</TableCell>)}</TableRow></TableHead>
          {data.items.map(c => <OperationGroup fieldLabels={fieldLabels} actionLabels={changeActions} emptyReferenceLabel="企业根部门" key={c.sourceVersion} change={c} href={href} remember={remember} />)}
        </Table>
      </TableContainer>
      {!data.items.length && <Typography role="status" sx={{ py: 3 }}>{applied ? '没有符合条件的变更日志' : '没有可读取的变更日志'}</Typography>}
      <TablePagination component="div" count={data.total} page={data.page - 1} rowsPerPage={data.pageSize} rowsPerPageOptions={[20, 50, 100]}
        labelRowsPerPage="每页操作" labelDisplayedRows={({ from, to, count }) => `${from}–${to} / ${count} 项操作`}
        getItemAriaLabel={type => ({ first: '第一页', last: '最后一页', next: '下一页', previous: '上一页' })[type]}
        onPageChange={(event, page) => update({ logPage: String(page + 1) }, false, event?.currentTarget)} onRowsPerPageChange={e => update({ logPage: '1', logPageSize: e.target.value }, false, root.current?.querySelector<HTMLElement>('[role="combobox"]'))} />
    </>}
    <Menu anchorEl={menu?.source} open={!!menu} onClose={() => setMenu(undefined)}>
      <MenuItem onClick={() => { if (menu) setRequest(menu); setMenu(undefined); }}>筛选…</MenuItem>
    </Menu>
    {request && <ColumnFilterDialog request={request} applied={model} columns={columns} onClose={closeFilter} onApply={values => update({ logPage: '1', ...Object.fromEntries(Object.entries(values).map(([k, v]) => [`log${k[0].toUpperCase()}${k.slice(1)}`, v])) })} />}
  </Box>;
}
