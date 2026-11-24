import { SubmissionError } from '../../shared/dated/submission';
import { selectedReason } from './dated/reasons';
import { usePathCursors } from './dated/usePathCursors';
import { useQuerySelection, ScopePageUnavailable } from './dated/useQuerySelection';
import { useOwnedRead } from './dated/useOwnedRead';
import { useDraftExit } from '../../shared/dated/useDraftExit';
import {
  useContext as useReactContext,
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import {
  Alert,
  Box,
  Tab,
  Tabs,
  Button,
  CircularProgress,
  FormControl,
  Divider,
  IconButton,
  ListSubheader,
  Menu,
  Tooltip,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  Typography,
} from "@mui/material";
import {
  Link,
  useLocation,
  useNavigate,
  useNavigationType,
  useParams,
  useSearchParams,
} from "react-router-dom";
import { ApiError, type Models } from "@chnhcm/api-client";

import BusinessIcon from '@mui/icons-material/Business';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import CloseIcon from '@mui/icons-material/Close';
import { DataGridPage, RecordControl, RecordBackButton, RecordPager, ObjectDetailSheet, ObjectDetailIdentity, ObjectDetailSection, ObjectDetailField, ObjectForm, ObjectTextField, ObjectFormError, ObjectTabPanel, EffectiveDates, StatusChip, UnsavedChangesDialog, HistoryNotice, businessDate, businessPeriod, focusObjectField } from '@chnhcm/ui';

import { useListReturn, usePageTitle } from '../../shared/navigation';
import { useObjectNavigation } from './dated/useObjectNavigation';
import { TimelineEditor } from './TimelineEditor';
import { LifecyclePairDialog } from './LifecyclePairDialog';
import { canReadStatus } from './businessUnitProjection';
import { useGridApiRef, GridPreferencePanelsValue, type GridColDef, type GridSortModel } from '@mui/x-data-grid';
import { BusinessUnitListControls } from './BusinessUnitListControls';
import { ColumnFilterEditor, columnFilterQuery, columnFilterModel, columnFilterSlots, columnStatuses, operatorsFor, type FilterRequest } from './BusinessUnitColumnFilter';

import { BusinessUnitChangeLog, OperationSummary, logKeys, logReturnHref } from './BusinessUnitChangeLog';

import { useDefinition, type DetailView, type ListView, type SummaryView, type ContextView, type CreateInput, type DataFields, type SelectionQuery } from './dated/presentation';
import { DataEditor, recordLabels as businessUnitFields } from './dated/DataEditor';
import { ListNavigation } from './dated/listNavigation';
import { safeListBack, selectedListBack } from './dated/object';

import { useSubmission } from './dated/useSubmission';

type Session = Models["Principal"];
const statusName = (s?: string | null) =>
  ({
    active: "启用",
    inactive: "停用",
    not_effective: "未生效",
    not_recorded: "尚未登记",
  })[s ?? ''] ?? s ?? '无读取权限';

function BusinessUnitNameLink({ id, name, tabIndex }: { id: string; name?: string | null; tabIndex: number }) {
  const navigation = useReactContext(ListNavigation)!;
  return <Link className="object-name-link" data-object-link tabIndex={tabIndex} to={navigation.href(id)} onClick={event => { if (!event.ctrlKey && !event.metaKey && !event.shiftKey && !event.altKey && event.button === 0) navigation.rememberScroll(id); }}>{name ?? '查看详情'}</Link>;
}
const businessUnitColumns: GridColDef<SummaryView>[] = [
    { field: 'code', headerName: '编码', width: 160, sortingOrder: ['asc'], filterOperators: operatorsFor('code'), renderCell: params => params.row.code },
    { field: 'name', headerName: '名称', flex: 1, minWidth: 180, sortingOrder: ['asc', 'desc', null],
      filterOperators: operatorsFor('name'),
      renderCell: params => <BusinessUnitNameLink id={params.row.id} name={params.row.name} tabIndex={params.tabIndex} /> },
    { field: 'status', headerName: '状态', width: 110, sortable: false,
      type: 'singleSelect', valueOptions: columnStatuses, filterOperators: operatorsFor('status'),
      renderCell: params => <StatusChip label={statusName(params.row.status)} color={params.row.status === 'active' ? 'success' : params.row.status === 'not_effective' ? 'info' : 'default'} /> },
    { field: 'effectiveDate', headerName: '生效日期', width: 170, sortable: false, filterOperators: operatorsFor('effectiveDate'), renderCell: params => params.row.effectiveDate ? businessDate(params.row.effectiveDate) : null },
];
const businessUnitSummary: GridColDef<SummaryView> = {
  field: 'summary', headerName: '', renderCell: params => <Box className="list-row-summary">
    <BusinessUnitNameLink id={params.row.id} name={params.row.name} tabIndex={params.tabIndex} />
    <Typography variant="body2" className="list-row-metadata">{[params.row.code, statusName(params.row.status), params.row.effectiveDate ? businessDate(params.row.effectiveDate) : null].filter(Boolean).join(' · ')}</Typography>
  </Box>,
};
const message = (e: unknown) =>
  e instanceof ApiError || e instanceof SubmissionError ? e.message : "请求失败，请重试";
function useContext() {
  const { api: businessUnitApi, kind } = useDefinition();
  const [value, setValue] = useState<ContextView>();
  const [error, setError] = useState("");
  const [denied, setDenied] = useState(false);
  const [revision, reload] = useState(0);
  useOwnedRead(`context/${kind}/${revision}`, {
    request: businessUnitApi.context,
    start: () => { setValue(undefined); setError(''); setDenied(false); },
    accept: setValue,
    reject: error => { setError(message(error)); setDenied(error instanceof ApiError && error.status === 403 && error.code !== 'AUTH_REQUEST_VALIDATION_FAILED'); },
  });
  return { value, error, denied, reload: () => reload((x) => x + 1) };
}
export function ObjectList() {
  const definition = useDefinition(), { api: businessUnitApi, selection: businessUnitSelection, basePath: BU_PATH } = definition;
  const selectedBack = (back: string | null, id: string) => selectedListBack(BU_PATH, back, id);
  const columns = useMemo(() => [...businessUnitColumns.slice(0, 2), ...(definition.additionalColumns ?? []), ...businessUnitColumns.slice(2)], [definition]);
  const summary = useMemo<GridColDef<SummaryView>>(() => ({ ...businessUnitSummary, renderCell: p => <Box sx={{width:'100%',minWidth:0}}>{businessUnitSummary.renderCell?.(p)}{definition.summary?.(p.row)}</Box> }), [definition]);
  usePageTitle(`${definition.label}列表`);
  const context = useContext();
  const [params, setParams] = useSearchParams();
  const location = useLocation(), navigation = useNavigationType();
  const [result, setResult] = useState<{ key: string; data?: ListView; error?: string; errorStatus?: number; scopeUnavailable?: boolean }>();
  const [retry, setRetry] = useState(0);
  const [search, setSearch] = useState(params.get("q") ?? "");
  const [scopeLabel, setScopeLabel] = useState<string>();
  const scopeId = definition.ListSidebar ? params.get("scopeDepartmentId") ?? undefined : undefined;
  const [includePreference, setIncludePreference] = useState(params.get("includeDescendants") !== "false");
  const includeDescendants = includePreference;
  useEffect(() => { if (scopeId) setIncludePreference(params.get("includeDescendants") !== "false"); }, [scopeId, params.get("includeDescendants")]);
  const localQuery = useRef<string | null>(null);
  const pathCursors = usePathCursors(BU_PATH, params);
  const pathMode = !!definition.pathCursorField && params.get("filterField") === definition.pathCursorField;
  useEffect(() => { if (pathMode) pathCursors.remember(Number(params.get("page") ?? 1), params.get("cursor") ?? ""); }, [pathMode, location.key]);
  const queryButton = useRef<HTMLButtonElement>(null);
  const gridApi = useGridApiRef();
  const [filterRequest, setFilterRequest] = useState<FilterRequest>();
  const filterModel = useMemo(() => columnFilterModel(params), [params.get('filterField'), params.get('filterOperator'), params.get('filterValue')]);
  const previousSearch = useRef(location.search);
  const selected = params.get("selected");
  const page = Number(params.get("page") ?? 1), size = Number(params.get("pageSize") ?? 20);
  const status = (params.get("status") ?? "all") as "all" | "active" | "inactive" | "not_effective";
  const sort = params.get("sort") ?? (definition.sortableFields ? "" : "code");
  // A quick navigation can return to the same query before React commits the
  // intermediate route. Its cleared result still needs a new owned read.
  const queryKey = `${definition.kind}:${location.key}:${params.toString()}&retry=${retry}`;
  const data = !context.denied && result?.key === queryKey ? result.data : undefined;
  const error = result?.key === queryKey ? result.error : undefined;
  const loading = !context.error && result?.key !== queryKey;
  useEffect(() => { if (context.denied) { setSearch(''); setFilterRequest(undefined); } }, [context.denied]);
  useLayoutEffect(() => {
    // Local query actions preserve newer input. History navigation reconstructs it.
    if (navigation === 'POP' || (previousSearch.current !== location.search && localQuery.current !== params.toString())) {
      setSearch(params.get('q') ?? '');
    }
    previousSearch.current = location.search;
    localQuery.current = null;
  }, [location.key]);
  const backContext = () => {
    const p = new URLSearchParams(params);
    p.delete("businessAt");
    p.set("page", String(data?.page ?? page));
    return location.pathname + "?" + p;
  };
  const scrollKey = () => {
    const q = new URLSearchParams(params);
    q.delete("businessAt");
    q.set("page", String(data?.page ?? page));
    q.delete("selected");
    return "bu-scroll:" + location.pathname + "?" + q;
  };
  const patch = (values: Record<string, string>, applyDraft = true) => {
    // A tree navigation may already have changed the URL while React is still
    // rendering its result. Merge the next action with that current query.
    const current = new URLSearchParams(window.location.search);
    const p = new URLSearchParams(current);
    p.delete('selected');
    const updates = applyDraft ? { q: search, page: '1', ...values } : values;
    Object.entries(updates).forEach(([k, v]) => (v ? p.set(k, v) : p.delete(k)));
    const apply = () => {
      clearReturn();
      selection.suspend(); setResult(undefined);
      if (p.toString() === current.toString()) { setRetry(x => x + 1); return; }
      localQuery.current = p.toString();
      setParams(p);
    };
    const changesQuery = ['q', 'status', 'filterField', 'filterOperator', 'filterValue', 'scopeDepartmentId', 'includeDescendants', 'countryRegionId'].some(key => (p.get(key) ?? '') !== (current.get(key) ?? ''));
    if (changesQuery || applyDraft || values.pageSize && Number(values.pageSize) !== size) { p.delete("cursor"); pathCursors.reset(); }
    if (!changesQuery && (values.page || values.pageSize || values.sort)) selection.reposition(apply);
    else apply();
  };
  useOwnedRead(queryKey, {
    request: signal => selection.requestPage(() => businessUnitApi.list!(params, undefined, signal)), start: () => setResult(undefined),
    accept: data => setResult({ key: queryKey, data }),
    reject: e => { if (e instanceof ScopePageUnavailable) { setResult({ key: queryKey, scopeUnavailable: true }); return; } if (e instanceof DOMException && e.name === 'AbortError') return; setResult({ key: queryKey, error: message(e), errorStatus: e instanceof ApiError && e.code !== 'AUTH_REQUEST_VALIDATION_FAILED' ? e.status : 0, scopeUnavailable: e instanceof ApiError && e.code === 'M02_TREE_SCOPE_UNAVAILABLE' }); if (e instanceof ApiError && [401, 403].includes(e.status) && e.code !== 'AUTH_REQUEST_VALIDATION_FAILED') {
        setSearch(''); setFilterRequest(undefined);
        if (definition.columnChoices && e.status === 403) context.reload();
      } },
  });
  const selectionQuery: SelectionQuery = { q: params.get('q') ?? '', status, ...columnFilterQuery(params), ...(params.get('countryRegionId') ? {countryRegionId: params.get('countryRegionId')!} : {}), ...(scopeId ? { scopeDepartmentId: scopeId, includeDescendants } : {}) };
  const selection = useQuerySelection(businessUnitSelection, data, selectionQuery, { page, pageSize: size, sort, cursor: params.get("cursor") ?? undefined }, Boolean(context.denied || error && [401,403,409].includes(result?.errorStatus ?? 0)), navigation === 'POP' ? location.key : null, () => { setResult(undefined); setRetry(value => value + 1); }, page => setResult({ key: queryKey, data: page }));
  useEffect(() => {
    if (definition.ListSidebar && data && context.value && data.businessAt !== context.value.currentDate) {
      selection.clear(); context.reload(); setRetry(value => value + 1);
    }
  }, [data?.businessAt, context.value?.currentDate]);
  const clearSelection = () => { selection.clear(); requestAnimationFrame(() => {
    const target = document.querySelector<HTMLInputElement>('.list-grid .MuiDataGrid-row input[type=checkbox], input[aria-label="搜索编码或名称"]');
    target?.focus();
  }); };
  const { scroller, remember: rememberScroll, rememberRow, clearReturn, selected: displayedSelection } = useListReturn({
    key: scrollKey(), ready: !loading && !!(data || error || context.error),
    apiRef: gridApi, selected, rows: data?.items ?? [],
    pageOffset: data ? (data.page - 1) * data.pageSize : 0,
  });
  const appliedBack = backContext();
  const selectScope = (id?: string) => {
    if (id === scopeId) return;
    selection.clear();
    patch({ scopeDepartmentId: id ?? '', includeDescendants: id ? String(includeDescendants) : '',
      selected: '', page: '1', cursor: '' }, false);
  };
  const filterAccess = context.denied || [401, 403].includes(result?.errorStatus ?? 0) ? undefined : context.value;
  const filterContext = { query: params, fields: filterAccess?.capabilities.readFields ?? [] };
  const filterColumns = useMemo(() => definition.columnChoices
    ? [...(definition.filterColumns ?? []), ...definition.columnChoices(filterAccess)]
    : definition.filterColumns, [definition, filterAccess]);
  const displayedColumns = useMemo(() => definition.columnChoices
    ? columns.map(column => ({ ...column, filterable: filterColumns?.some(filter => filter.field === column.field) ?? false }))
    : columns, [definition, columns, filterColumns]);
  const scopeText = result?.scopeUnavailable ? '当前范围不可用' : scopeLabel ? `当前范围：${scopeLabel}${includeDescendants ? '（含子部门）' : ''}` : '正在读取部门范围…';
  const href = useCallback((id: string) => `${BU_PATH}/${id}?${new URLSearchParams({ back: selectedBack(appliedBack, id) })}`, [appliedBack]);
  const sortModel = useMemo<GridSortModel>(() => sort ? [{field: sort.replace(/_desc$/, ''), sort: sort.endsWith('_desc') ? 'desc' : 'asc'}] : [], [sort]);

  return <ColumnFilterEditor.Provider value={setFilterRequest}><ListNavigation.Provider value={{ href, rowBack: id => selectedBack(appliedBack, id), rememberScroll: (id, childLink) => { selection.suspend(); rememberRow(id, childLink); } }}><Box className="standard-page" data-selection-state={selection.state}>
    {definition.ListFilters && <definition.ListFilters query={params} fields={context.value?.capabilities.readFields ?? []} onChange={patch} />}
    <DataGridPage apiRef={gridApi} scrollerRef={scroller}
      sidebar={definition.ListSidebar ? <definition.ListSidebar scopeId={scopeId} includeDescendants={includeDescendants} day={context.value?.currentDate} unavailable={Boolean(context.denied || result?.scopeUnavailable)} onLabel={setScopeLabel}
        onDayMismatch={() => { selection.clear(); context.reload(); setRetry(x => x + 1); }}
        onSelect={selectScope}
        onInclude={value => { if (!scopeId || value === includeDescendants) return; setIncludePreference(value); selection.clear(); patch({ includeDescendants: String(value), selected: '', page: '1', cursor: '' }, false); }} /> : undefined}
      selection={businessUnitSelection ? { checked: selection.checked, mixed: selection.mixed, ids: selection.selected, selectable: selection.selectable, onChange: selection.setPage, count: selection.summary?.selectedCount,
        total: data?.count ?? (data?.items.length ? -1 : 0), disabled: loading || !data, all: selection.all, clear: clearSelection } : undefined}
      label={`${definition.listLabel ?? definition.label}列表`} selected={displayedSelection} rows={data?.items ?? []} loading={loading} ready={Boolean(data)}
      empty={data ? definition.scopedEmpty?.(params) ?? `没有符合条件的${definition.label}` : '请重新查询'} columns={definition.sortableFields ? columns.map(c => ({...c, sortable: definition.sortableFields!.includes(c.field), sortingOrder: ['asc', 'desc', null]})) : displayedColumns} defaultHiddenColumns={definition.hiddenColumns}
      filterModel={filterModel} slots={columnFilterSlots}
      paginationModel={{ page: (data?.page ?? (Number.isSafeInteger(page) && page > 0 ? page : 1)) - 1,
        pageSize: data?.pageSize ?? ([20, 50, 100].includes(size) ? size : 20) }} rowCount={data?.count} hasNextPage={pathMode ? data?.hasMore : undefined}
      onPaginationModelChange={model => {
        const pageSize = model.pageSize;
        const nextPage = pageSize !== (data?.pageSize ?? size) ? 1 : model.page + 1;
        if (nextPage !== (data?.page ?? page) || pageSize !== (data?.pageSize ?? size))
          {
            if (pathMode) {
              if (data?.hasMore && data.nextCursor) pathCursors.remember(data.page + 1, data.nextCursor);
              const cursor = pageSize !== size ? '' : pathCursors.pages[nextPage];
              if (cursor === undefined) return;
              patch({ page: String(nextPage), pageSize: String(pageSize), cursor }, false);
            } else patch({ page: String(nextPage), pageSize: String(pageSize) }, false);
          }
      }}
      sortModel={sortModel} onSortModelChange={model => {
        const first = model[0];
        const next = definition.sortableFields ? (first?.sort ? first.field + (first.sort === 'desc' ? '_desc' : '') : '') : first?.field === 'name' ? first.sort === 'desc' ? 'name_desc' : first.sort === 'asc' ? 'name' : 'code' : 'code';
        if (next !== sort) patch({ sort: next });
      }}
      summaryColumn={summary}
      toolbar={<BusinessUnitListControls title={definition.listLabel ?? definition.label} columns={filterColumns} summaryRevision={queryKey}
        navigationScope={definition.ListScope && <definition.ListScope {...filterContext} />}
        scopeSummary={scopeId && <Box className="list-scope-summary" data-unavailable={Boolean(result?.scopeUnavailable)}>
          <Tooltip title={scopeText}><Box className="list-scope-label" role="status" aria-label={scopeText}>{scopeLabel && !result?.scopeUnavailable ? `${scopeLabel}${includeDescendants ? ' · 含子部门' : ''}` : scopeText}</Box></Tooltip>
          <Tooltip title="清除范围"><IconButton size="small" aria-label="清除范围" onClick={() => selectScope(undefined)}><CloseIcon sx={{fontSize: 16}} /></IconButton></Tooltip>
        </Box>}
        selectionBar={selection.active ? <Stack direction="row" gap={1} alignItems="center" flexWrap="wrap" role="toolbar" aria-label={`所选${definition.label}`}>
        <Typography role="status">{selection.pending ? selection.notice ? '选择待确认' : '正在核对选择…' : `已选择${selection.summary?.selectedCount ?? 0}项`}</Typography>
        <Button size="small" onClick={clearSelection}>取消选择</Button>
        {businessUnitSelection?.protocol === 'identity-draft' && <Button size="small" onClick={() => selection.reposition(() => { selection.suspend(); setResult(undefined); setRetry(value => value + 1); })}>刷新</Button>}
        <Tooltip title={!selection.pending && !selection.summary?.canExport ? '当前没有全部所选对象的导出权限' : ''}><span><Button size="small" disabled={selection.pending || !selection.summary?.canExport || selection.exporting} onClick={() => void selection.exportSelected()}>{selection.exporting ? '正在导出…' : '导出所选'}</Button></span></Tooltip>
      </Stack> : undefined} search={search} appliedSearch={params.get('q') ?? ''} onSearchChange={value => {
        setSearch(value);
        if (definition.kind === 'organization.department' && value === '' && params.get('q')) {
          selection.clear();
          patch({ q: '', selected: '', page: '1', cursor: '' }, false);
        }
      }}
        onSearch={() => patch({})} queryButton={queryButton}
        status={status} sort={sort} sortOptions={definition.sortableFields ? Object.fromEntries([['','默认顺序'], ...definition.sortableFields.flatMap(f=>{const label=definition.filterColumns?.find(c=>c.field===f)?.label??f;return [[f,label+'升序'],[f+'_desc',label+'降序']];})]) : undefined} onApply={patch} canCreate={Boolean(businessUnitApi.create && context.value?.capabilities.create)}
        filterModel={filterModel} filterRequest={filterRequest} onCloseFilter={() => setFilterRequest(undefined)}
        createHref={businessUnitApi.create ? `${BU_PATH}/new?${new URLSearchParams({ back: backContext() })}` : undefined} rememberScroll={() => { selection.suspend(); rememberScroll(); }} onManageColumns={() => gridApi.current?.showPreferences(GridPreferencePanelsValue.columns)} />}
      messages={<>
        {selection.notice && <Alert severity={selection.noticeSeverity} role="status" tabIndex={-1} data-selection-status action={selection.active && selection.pending ? <Button onClick={selection.retry}>重试</Button> : undefined}>{selection.notice}</Alert>}
        {(context.error || error) && <Alert severity="error" tabIndex={-1} data-list-return-fallback action={<Button onClick={() => { queryButton.current?.focus(); context.reload(); if (params.has('cursor')) { pathCursors.reset(); patch({ cursor: '', page: '1' }, false); } else setRetry(x => x + 1); }}>重试</Button>}>{context.error || error}</Alert>}
      </>} />
  </Box></ListNavigation.Provider></ColumnFilterEditor.Provider>;
}

export function ObjectNew({ session }: { session: Session }) {
  const definition = useDefinition(), { api: businessUnitApi, basePath: BU_PATH } = definition;
  const businessUnit = definition;
  const safeBack = (back: string | null) => safeListBack(BU_PATH, back);
  const selectedBack = (back: string | null, id: string) => selectedListBack(BU_PATH, back, id);
  usePageTitle(`新建${definition.label}`);
  const context = useContext();
  const [params] = useSearchParams();
  const navigate = useNavigate();
  const [dirty, setDirty] = useState(false);
  const [error, setError] = useState("");
  const [code, setCode] = useState("");
  const [values, setValues] = useState<DataFields>({ name: "" });
  const [date, setDate] = useState("");
  const dateInitialized = useRef(false);
  const [reason, setReason] = useState("");
  const [reasons, setReasons] = useState<Models["Reason"][]>([]);
  const [reasonRevision, retryReasons] = useState(0);
  const [reasonError, setReasonError] = useState('');
  const [fieldError, setFieldError] = useState("");
  const form = useRef<HTMLFormElement>(null);
  const { flow, sessionBlocked, pending: blockedByPending } = useSubmission({
    channel: businessUnit.submissions, owner: session, resolve: reference => businessUnit.resolve(session, reference),
    send: (body: CreateInput, id) => businessUnitApi.create!(body, { 'X-CSRF-Token': session.csrf, 'Idempotency-Key': id }),
    onResolved: result => { setError(''); if (result.outcome === 'accepted') { setDirty(false); navigate(`${BU_PATH}/${result.objectId}?${new URLSearchParams({ selectedRecordId: result.selectedRecordId ?? '', back: selectedBack(params.get('back'), result.objectId ?? ''), saved: '1' })}`, { replace: true }); } else context.reload(); },
    onDenied: reference => { flow.submittedId = reference.submissionId; flow.unknown = true; clearRestrictedInput(); context.reload(); },
  });
  const { inFlight: busy, unknown } = flow;
  const clearRestrictedInput = useCallback(() => {
    setCode("");
    setValues({ name: "" });
    setDate("");
    dateInitialized.current = false;
    setReason("");
    setDirty(false);
    flow.clearPayload();
  }, [flow]);
  useEffect(() => {
    if (context.denied) clearRestrictedInput();
  }, [context.denied, clearRestrictedInput]);
  useEffect(() => {
    if (fieldError && !busy) {
      focusObjectField(`${definition.prefix}-${fieldError}`, `${definition.prefix}-form-error`);
    }
  }, [fieldError, busy]);
  useEffect(() => {
    if (context.value && !dateInitialized.current) {
      dateInitialized.current = true;
      setDate(context.value.defaultEffectiveDate);
    }
  }, [context.value]);
  useOwnedRead(`create-reasons/${session.instanceId}/${session.subjectId}/${reasonRevision}`, {
    request: signal => businessUnitApi.reasons('CREATE', signal), start: () => setReasonError(''),
    accept: values => { setReasons(values); setReason(previous => selectedReason(previous, values)); }, reject: error => setReasonError(message(error)),
  });
  const leave = useDraftExit(dirty, () => flow.saved, () => {
    setCode(''); setValues({ name: '' }); setDate(context.value?.defaultEffectiveDate ?? ''); dateInitialized.current = !!context.value; setReason(selectedReason('', reasons)); setFieldError(''); flow.clearPayload(); setDirty(false);
  });
  const changed = (fn: (s: string) => void) => (e: React.SyntheticEvent) => {
    const value = (e.target as HTMLInputElement).value;
    fn(value);
    setDirty(true);
  };
  const submit = async (event?: React.FormEvent) => {
    event?.preventDefault();
    if (sessionBlocked || !context.value || (blockedByPending && !flow.attempt)) return;
    setError(''); setFieldError('');
    const body: CreateInput = { code, name: values.name ?? '', ...values, effectiveDate: date, operationType: 'CREATE', operationReason: reason };
    const outcome = await flow.submit(body);
    if (outcome.kind === 'saved') {
      setDirty(false); const result = outcome.result;
      navigate(`${BU_PATH}/${result.objectId}?${new URLSearchParams({ selectedRecordId: result.selectedRecordId ?? '', back: selectedBack(params.get('back'), result.objectId ?? ''), saved: '1' })}`, { replace: true });
    } else if (outcome.kind === 'failed') {
      const error = outcome.error; setError(flow.unknown ? '' : message(error));
      if (error instanceof ApiError) setFieldError(error.field ?? '');
      if (error instanceof ApiError && error.status === 403 && error.code !== 'AUTH_REQUEST_VALIDATION_FAILED') { clearRestrictedInput(); context.reload(); }
    }
  };
  const disabled = !context.value?.capabilities.create || busy || unknown || flow.saved;
  const saveHelp = busy || unknown || blockedByPending || context.error || reasonError ? ''
    : !context.value ? '正在加载…'
    : !reasons.length ? '正在加载操作原因…'
    : !context.value.capabilities.create ? '当前没有创建权限。'
    : !reason ? '请选择操作原因。'
    : '';
  return <Box className="standard-page object-page">
    <RecordControl identity={<><RecordBackButton component={Link} to={safeBack(params.get('back'))} /></>}
      actions={<><Button type="submit" form={`${definition.prefix}-create`} aria-describedby={context.error || reasonError ? `${definition.prefix}-form-error` : saveHelp ? `${definition.prefix}-save-help` : undefined} size="small" variant="contained" disabled={sessionBlocked || busy || unknown || flow.saved || blockedByPending || !context.value?.capabilities.create || !reason || !!reasonError}>{busy ? '保存中…' : '保存'}</Button><Button size="small" component={Link} to={safeBack(params.get('back'))}>取消</Button></>} navigation={null} />
    <Box className="page-scroll" data-page-scroll tabIndex={0} role="region" aria-label={`${definition.label}表单正文`}>
      <ObjectDetailSheet identity={<ObjectDetailIdentity icon={<BusinessIcon />} title={`新建${definition.label}`} summary={<Typography variant="body2">所属企业：{context.value?.enterprise?.name ?? '加载中…'}</Typography>} />}
>
        <ObjectForm><form id={`${definition.prefix}-create`} ref={form} onSubmit={event => void submit(event)}>
          <Box>
            {saveHelp && <Typography id={`${definition.prefix}-save-help`} className="object-form-hint">{saveHelp}</Typography>}
            <ObjectFormError id={`${definition.prefix}-form-error`} message={context.error || reasonError || error} issues={fieldError && !context.error && !reasonError ? [{ id: `${definition.prefix}-${fieldError}`, message: error }] : []}>
              {(context.error || reasonError) && <Button disabled={busy} onClick={() => { context.reload(); retryReasons(x => x + 1); }}>重试加载</Button>}
            </ObjectFormError>
            {unknown && flow.attempt && <Button disabled={sessionBlocked || busy} onClick={() => void submit()}>重试保存</Button>}
            <Box className="object-detail-columns">
              <ObjectDetailSection title="基本资料">
                <ObjectTextField id={`${definition.prefix}-code`} name="code" label={businessUnitFields.code.label} value={code} onChange={changed(setCode)} required disabled={disabled} error={fieldError === 'code'} helperText={fieldError === 'code' ? error : '1 至 40 位字母、数字、下划线或连字符，以字母或数字开头。'} />
                <DataEditor session={session} values={values} fields={context.value?.capabilities.writeFields ?? definition.fields.map(f => f.key)} writes={context.value?.capabilities.writeFields ?? []} disabled={disabled} prefix={definition.prefix} fieldError={fieldError} error={error} guidance={context.value?.nameGuidance}
                  onReadDenied={() => { clearRestrictedInput(); context.reload(); }} context={definition.createReferenceContext?.(date, values)} onChange={(field, value) => { setValues(v => ({ ...v, [field]: value })); setDirty(true); }} />
              </ObjectDetailSection>
              <ObjectDetailSection title="记录信息">
                <ObjectTextField id={`${definition.prefix}-effectiveDate`} name="effectiveDate" label={businessUnitFields.effectiveDate.label} type="date" value={date} onChange={changed(setDate)} required disabled={disabled} error={fieldError === 'effectiveDate'} helperText={fieldError === 'effectiveDate' ? error : undefined}  />
                <ObjectDetailField label={businessUnitFields.operationType.label}><Typography>新建</Typography></ObjectDetailField>
                <ObjectTextField id={`${definition.prefix}-operationReason`} name="operationReason" label={businessUnitFields.operationReason.label} select value={reason} onChange={changed(setReason)} required disabled={disabled} error={fieldError === 'operationReason'} helperText={fieldError === 'operationReason' ? error : undefined}>
                  <MenuItem value="">请选择</MenuItem>{reasons.map(v => <MenuItem key={v.code} value={v.code}>{v.name}</MenuItem>)}
                </ObjectTextField>
              </ObjectDetailSection>
            </Box>
          </Box>
        </form></ObjectForm>
      </ObjectDetailSheet>
    </Box>
    <UnsavedChangesDialog pending={unknown || blockedByPending} {...leave.dialog} />
  </Box>;
}

export function ObjectDetail({ session }: { session: Session }) {
  const definition = useDefinition(), { api: businessUnitApi, basePath: BU_PATH } = definition;
  const safeBack = (back: string | null) => safeListBack(BU_PATH, back);
  const [editing, setEditing] = useState<'ADD' | 'CORRECT' | 'DEACTIVATE' | 'REACTIVATE' | 'DELETE'>();
  const [pairMode,setPairMode]=useState<'insert'|'delete'|'maintain'>();
  const [moreAnchor, setMoreAnchor] = useState<HTMLElement | null>(null);
  const [editorActions, setEditorActions] = useState<HTMLDivElement | null>(null);
  const moreButton = useRef<HTMLButtonElement>(null);
  const restoreMore = useRef(false);
  const location = useLocation();
  const { id = "" } = useParams();
  const [params, setParams] = useSearchParams();
  const [response, setData] = useState<DetailView>();
  const [loadedKey, setLoadedKey] = useState('');
  const [error, setError] = useState("");
  const [retryableRead, setRetryableRead] = useState(false);
  const [retry, setRetry] = useState(0);
  const [logsReady, setLogsReady] = useState(false);
  const history = params.has("sourceVersion");
  const readParams = new URLSearchParams(params);
  for (const key of ['tab', 'back', 'saved', ...logKeys]) readParams.delete(key);
  const readKey = readParams.toString();
  const data = loadedKey === `${session.instanceId}/${session.subjectId}/${id}?${readKey}` ? response : undefined;
  useEffect(() => { setEditing(undefined);setPairMode(undefined); setMoreAnchor(null); }, [id, readKey]);
  useEffect(() => {
    if (data && !editing && !pairMode && restoreMore.current) {
      moreButton.current?.focus({ preventScroll: true });
      restoreMore.current = false;
    } else if (error && !data && restoreMore.current) {
      document.getElementById(`${definition.prefix}-detail-error`)?.focus();
    }
  }, [data, editing, pairMode, error]);
  useOwnedRead(`${session.instanceId}/${session.subjectId}/${id}?${readKey}&retry=${retry}`, {
    start: () => { setData(undefined); setError(''); setRetryableRead(false); },
    request: signal => {
      if (params.getAll('sourceVersion').length > 1 || params.getAll('selectedRecordId').length > 1) throw new ApiError('INVALID_QUERY', '历史资料链接无效，请返回当前资料重新选择。', 400);
      if (params.has('knownAt')) throw new ApiError('INVALID_QUERY', '无法打开此历史链接，请返回当前资料查看变更日志', 400);
      if (Array.from(readParams.keys()).some(key => !['sourceVersion', 'selectedRecordId'].includes(key))) throw new ApiError('INVALID_QUERY', '未知的查询条件，请返回当前资料重新选择。', 400);
      return businessUnitApi.detail(id, { sourceVersion: params.get('sourceVersion') ?? undefined, selectedRecordId: params.get('selectedRecordId') ?? undefined }, signal);
    },
    accept: value => { setData(value); setLoadedKey(`${session.instanceId}/${session.subjectId}/${id}?${readKey}`); },
    reject: error => { setError(message(error)); setRetryableRead(!(error instanceof ApiError) || [429, 500, 502, 503, 504].includes(error.status)); },
  });
  const returnCurrent = () => {
    // A preceding tab navigation updates the URL before React Router finishes
    // rendering. Preserve that latest choice, not this render's older query.
    const p = new URLSearchParams(window.location.search);
    p.delete("knownAt");
    for (const key of Array.from(p.keys())) if (!["sourceVersion", "selectedRecordId", "tab", "back", "saved", ...logKeys].includes(key)) p.delete(key);
    p.delete("sourceVersion");
    if(data?.selectedRecord && !data.selectedRecord.deleted) p.set("selectedRecordId",data.selectedRecord.recordId);
    else p.delete("selectedRecordId");
    p.delete("saved");
    p.delete("logThrough"); p.delete("logPage"); p.delete("logReturn");
    setParams(p);
  };
  const objectNavigation = useObjectNavigation(definition.navigation, BU_PATH, id, safeBack(params.get('back')), Boolean(data), businessUnitApi.list ? (query, signal) => businessUnitApi.list!(query, undefined, signal) : undefined, Boolean(error));
  const savedOutcome = params.get('saved');
  useEffect(() => {
    if (editing || pairMode) window.dispatchEvent(new Event('chnhcm-edit-started'));
    if ((editing || pairMode) && params.has('saved')) { const next = new URLSearchParams(params); next.delete('saved'); setParams(next, { replace: true }); }
  }, [editing, pairMode]);
  useEffect(() => {
    if (data && savedOutcome) document.getElementById(`${definition.prefix}-save-result`)?.focus({ preventScroll: true });
  }, [data, savedOutcome]);
  const record = data?.selectedRecord;
  const tab = !editing && params.get('tab') === 'history' && data?.capabilities.history ? 'history' : 'basic';
  const hasMoreActions = data && !history && (data.capabilities.correct || data.capabilities.delete || data.capabilities.redate || data.pairInsertion || data.pairDeletion || data.capabilities.deactivate || data.capabilities.reactivate);
  const title = data ? `${data.code ?? definition.label}${record?.name ? ` · ${record.name}` : ''}` : `${definition.label}详情`;
  usePageTitle(title);
  return <Box className="standard-page object-page">
    <RecordControl identity={<>
      {!definition.singleton && <RecordBackButton component={Link} to={safeBack(params.get('back'))} />}
      <Typography className="record-control-name">{data?.code ?? `${definition.label}详情`}</Typography>
      {hasMoreActions && <Tooltip title="更多操作" disableInteractive><span><IconButton ref={moreButton} size="small" aria-label="更多操作" aria-haspopup="menu" aria-expanded={!!moreAnchor} disabled={!data || !!editing} onClick={event => setMoreAnchor(event.currentTarget)}><MoreVertIcon /></IconButton></span></Tooltip>}
    </>}
      actions={editing ? <Box ref={setEditorActions} /> : data && !history && data.capabilities.add && data.additionContext ? <Button size="small" variant="contained" disabled={!data.additionContext.available} onClick={() => setEditing('ADD')}>{data.additionContext.mode==='INSERT'?'插入记录':'新增记录'}</Button> : null}
      navigation={objectNavigation.available && <RecordPager counter={objectNavigation.counter} focusOnMount={location.state?.recordPagerFocus} previous={objectNavigation.previous} next={objectNavigation.next} disabled={objectNavigation.busy || !!editing} onChange={direction => void objectNavigation.move(direction)} />} />
    <Menu anchorEl={moreAnchor} open={!!moreAnchor && !!data} onClose={() => setMoreAnchor(null)} slotProps={{ list: { 'aria-label': '更多操作' } }}>
      {data && !history && (data.capabilities.correct || data.capabilities.delete || data.capabilities.redate || data.pairInsertion || data.pairDeletion) && record && [
        <ListSubheader key="target">所选记录：{businessDate(record.effectiveDate)}</ListSubheader>,
        data.capabilities.correct && <MenuItem key="correct" onClick={() => { setMoreAnchor(null); setEditing('CORRECT'); }}>更正记录</MenuItem>,
        businessUnitApi.pair && data.pairInsertion && <MenuItem key="insert-pair" disabled={!data.pairInsertion.available} onClick={()=>{setMoreAnchor(null);setPairMode('insert')}}>插入停用与重新启用</MenuItem>,
        businessUnitApi.maintainPair && data.pairDeletion && <MenuItem key="maintain-pair" onClick={()=>{setMoreAnchor(null);setPairMode('maintain')}}>维护停用与重新启用</MenuItem>,
        businessUnitApi.pair && data.pairDeletion && <MenuItem key="delete-pair" onClick={()=>{setMoreAnchor(null);setPairMode('delete')}}>删除停用与重新启用</MenuItem>,
        data.capabilities.delete && <MenuItem key="delete" onClick={() => { setMoreAnchor(null); setEditing('DELETE'); }}>删除此记录</MenuItem>,
        <Divider key="divider" />,
      ]}
      {data && !history && (data.capabilities.deactivate || data.capabilities.reactivate) && [
        <ListSubheader key="lifecycle">{definition.label}</ListSubheader>,
        data.capabilities.deactivate && <MenuItem key="deactivate" onClick={() => { setMoreAnchor(null); setEditing('DEACTIVATE'); }}>停用{definition.label}</MenuItem>,
        data.capabilities.reactivate && <MenuItem key="reactivate" onClick={() => { setMoreAnchor(null); setEditing('REACTIVATE'); }}>重新启用{definition.label}</MenuItem>,
        <Divider key="lifecycle-divider" />,
      ]}
    </Menu>

    {businessUnitApi.pair && pairMode && data && !history && <LifecyclePairDialog key={`${id}-${pairMode}`} session={session} detail={data} mode={pairMode} onClose={()=>{setPairMode(undefined);restoreMore.current=true}} onReloaded={v=>{setLogsReady(false);setData(v)}} onDenied={()=>{setPairMode(undefined);setData(undefined);setLogsReady(false);setError('无法查看该资料，请重试')}} onSaved={result=>{const mode=pairMode;setPairMode(undefined);setData(undefined);const p=new URLSearchParams(params);const inserted=result.items?.filter(i=>i.afterRecord?.effectiveDate).sort((a,b)=>a.afterRecord!.effectiveDate!.localeCompare(b.afterRecord!.effectiveDate!))[0];if(mode==='insert'&&inserted?.recordId)p.set('selectedRecordId',inserted.recordId);if(mode==='delete'){const stopIndex=data.timeline.findIndex(r=>r.recordId===data.pairDeletion?.deactivation.recordId);const previous=data.timeline[stopIndex-1];if(previous)p.set('selectedRecordId',previous.recordId);else p.delete('selectedRecordId');}p.delete('logThrough');p.delete('logPage');p.set('tab','basic');p.set('saved',mode==='delete'?'deleted-pair':'1');setParams(p);setRetry(n=>n+1)}}/>}
    {history && data && <HistoryNotice onReturn={returnCurrent} />}
    {data?.capabilities.history && logReturnHref(params.get('logReturn'), location.pathname) && <Box sx={{ px: 2, py: 0.5 }}><Button size="small" component={Link} to={logReturnHref(params.get('logReturn'), location.pathname)!}>返回变更日志</Button></Box>}
    <Box className="page-scroll" data-page-scroll data-page-scroll-ready={!!error || (!!data && loadedKey === `${session.instanceId}/${session.subjectId}/${id}?${readKey}` && (tab !== 'history' || logsReady))} tabIndex={0} role="region" aria-label={`${definition.label}详情正文`}>
      <Stack className="page-messages" spacing={1}>
        {objectNavigation.counterError && !objectNavigation.error && <Alert severity="info" action={<Button onClick={objectNavigation.retry}>重试</Button>}>{objectNavigation.counterError}</Alert>}
        {objectNavigation.error && <Alert severity="info" action={<Button onClick={objectNavigation.retry}>重试</Button>}>{objectNavigation.error}</Alert>}
        {error && <Alert id={`${definition.prefix}-detail-error`} tabIndex={-1} severity="error" action={history && data ? undefined : !retryableRead && Boolean(readKey) ? <Button onClick={returnCurrent}>返回当前资料</Button> : <Button onClick={() => setRetry(x => x + 1)}>重试</Button>}>{error}</Alert>}
        {!data && !error && <CircularProgress aria-label="加载中" />}
        {data && !editing && !pairMode && params.get('saved') === 'no_change' && <Alert id={`${definition.prefix}-save-result`} severity="info" role="status" tabIndex={-1}>没有修改内容。</Alert>}
        {data && ['deleted','deleted-pair'].includes(params.get('saved')??'') && <Alert id={`${definition.prefix}-save-result`} severity="success" tabIndex={-1} role="status">已删除。</Alert>}

        {data && record?.deleted && <Alert severity="info">已删除记录原件，只读。下方保留删除前内容与期间。</Alert>}
        {data && !editing && !pairMode && params.get('saved') === '1' && <Alert id={`${definition.prefix}-save-result`} severity="success" tabIndex={-1} role="status">已保存。生效日期：{businessDate(record?.effectiveDate)}。</Alert>}
        {data?.selectedRecordIsFuture && <Alert severity="info">所选记录尚未生效。</Alert>}
        {data?.selectionMissing && <Alert severity="warning">当前资料中没有所选记录，请重新选择。</Alert>}
      </Stack>
      {data && <ObjectDetailSheet identity={<ObjectDetailIdentity icon={<BusinessIcon />} title={title} summary={data.enterprise && <Typography variant="body2">所属企业：{data.enterprise.name}</Typography>} />}
        tabs={<Tabs value={tab} variant="scrollable" allowScrollButtonsMobile onChange={(_, value: string) => { if (value === 'purposes') return; const p = new URLSearchParams(params); p.set('tab', value); setParams(p); }}>
          <Tab id={`${definition.prefix}-basic-tab`} value="basic" label="基本资料" aria-controls={`${definition.prefix}-basic-panel`} />
          {data.canReadPurposes && !editing && !pairMode && <Tab label="用途与绑定" component={Link} to={`${BU_PATH}/${id}/purposes${location.search}`} value="purposes" onClick={event => event.stopPropagation()} />}
          {data.capabilities.history && <Tab id={`${definition.prefix}-history-tab`} value="history" label="变更日志" disabled={!!editing} aria-controls={`${definition.prefix}-history-panel`} />}
        </Tabs>}
        timeline={!editing && tab === 'basic' && data.capabilities.readFields.includes('effectiveDate') && !!data.timeline.length && <EffectiveDates dates={data.timeline.flatMap(r => r.effectiveDate ? [r.effectiveDate] : [])} selected={record?.effectiveDate ?? undefined} onSelect={date => { const r = data.timeline.find(r => r.effectiveDate === date); if (r) { const p = new URLSearchParams(params); p.set('selectedRecordId', r.recordId); p.delete('saved'); setParams(p); } }} />}>
        <ObjectTabPanel prefix={definition.prefix} value="basic" selected={tab}>
          {editing && !history && <TimelineEditor key={`${id}-${editing}`} session={session} detail={data} action={editing as 'ADD'|'CORRECT'|'DEACTIVATE'|'REACTIVATE'|'DELETE'} actionsContainer={editorActions} onReloaded={value => {  setLogsReady(false); setData(value); }} onClose={() => { restoreMore.current = true; setEditing(undefined); }} onDenied={() => {  setEditing(undefined); setData(undefined); setLogsReady(false); setError('无法查看该资料，请重试'); }} onSaved={result => { setData(undefined); setEditing(undefined); const p = new URLSearchParams(params); if (result.selectedRecordId) p.set('selectedRecordId', result.selectedRecordId); if (result.outcome === 'accepted') { p.delete('logThrough'); p.delete('logPage'); } p.set('tab', 'basic'); p.set('saved', result.outcome === 'no_change' ? 'no_change' : result.action === 'DELETE' ? 'deleted' : '1'); setParams(p); setRetry(x => x + 1); }} />}

          {!editing && record && <>
            <Box className="object-detail-columns">
              <ObjectDetailSection title="基本资料">
                <ObjectDetailField label={businessUnitFields.code.label}>{data.code}</ObjectDetailField>
                {definition.fields.filter(f => data.capabilities.readFields.includes(f.key)).map(f => <ObjectDetailField key={f.key} label={f.label}>{record.values[f.key] ?? '未提供'}</ObjectDetailField>)}
                {data.basicFields?.(record, data.capabilities.readFields, !!data.relatedReferencesRestricted)}
              </ObjectDetailSection>
              <ObjectDetailSection title="记录信息">
                {data.capabilities.readFields.includes('effectiveDate') && record.effectiveDate && <>
                  <ObjectDetailField label={businessUnitFields.effectiveDate.label}>{businessDate(record.effectiveDate)}</ObjectDetailField>
                  <ObjectDetailField label={record.deleted ? "原适用期间" : "适用期间"}>{businessPeriod(record.effectiveDate, record.endExclusive, record.endInclusive)}</ObjectDetailField>
                </>}
                <ObjectDetailField label="状态">{statusName(canReadStatus(data.capabilities.readFields, definition.statusReadField) ? record.state : undefined)}</ObjectDetailField>
                <ObjectDetailField label={businessUnitFields.operationType.label}>{record.typeLabel}</ObjectDetailField>
                <ObjectDetailField label={businessUnitFields.operationReason.label}>{record.reasonLabel}</ObjectDetailField>
              </ObjectDetailSection>
            </Box>
          </>}
          {!editing && data.related?.(data.selectedRecord, data.capabilities.readFields, !!data.relatedReferencesRestricted)}
          {!editing && record?.operationSummary && <OperationSummary summary={record.operationSummary} />}
        </ObjectTabPanel>
        {data.capabilities.history && <ObjectTabPanel prefix={definition.prefix} value="history" selected={tab}>
          {tab === 'history' && <BusinessUnitChangeLog key={`${id}/${session.subjectId}/${data.resolvedSourceVersion}`} id={id} subjectKey={`${session.instanceId}/${session.subjectId}`} onReady={setLogsReady} onReadFields={fields => {
            const previous = data.capabilities.readFields;
            if (previous.length !== fields.length || previous.some(field => !fields.includes(field))) {
              // Discard the whole cached projection, including hidden panels,
              // titles and capabilities. Re-read under Detail's own authority.
              setData(undefined); setLogsReady(false); setRetry(n => n + 1);
            }
          }} onDenied={message => { setData(undefined); setLogsReady(false); setError(message); setRetryableRead(true); const p = new URLSearchParams(params); logKeys.forEach(k => p.delete(k)); p.set('tab', 'basic'); setParams(p, { replace: true }); }} />}
        </ObjectTabPanel>}
      </ObjectDetailSheet>}
    </Box>
  </Box>;
}
