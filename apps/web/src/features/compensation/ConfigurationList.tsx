import { useLayoutEffect, useRef, useState } from 'react';
import { flushSync } from 'react-dom';
import {
  Alert,
  Box,
  Button,
  IconButton,
  Menu,
  MenuItem,
  Stack,
  Typography,
  useMediaQuery,
} from '@mui/material';
import FilterAlt from '@mui/icons-material/FilterAlt';
import MoreVert from '@mui/icons-material/MoreVert';
import {
  GridPreferencePanelsValue,
  useGridApiRef,
  type GridColDef,
  type GridFilterModel,
} from '@mui/x-data-grid';
import { Link, useLocation, useNavigationType } from 'react-router-dom';
import { DataGridPage, ListControl, ListPager, SearchControl } from '@chnhcm/ui';
import {
  ColumnFilterDialog,
  ColumnFilterEditor,
  columnFilterSlots,
  type FilterColumn,
  type FilterRequest,
} from '../../shared/dated/ColumnFilter';
import { useListReturn, useMaintenanceSearchParams, usePageTitle } from '../../shared/navigation';
import { useRead } from '../../shared/useRead';
import { statusLabels, type Principal } from './rates';

const exact = [{ value: 'equals', label: '等于' }];
export function filterColumn(field: string, label: string, labels?: Record<string, string>): FilterColumn {
  return {
    field,
    label,
    operators: exact,
    values: labels && Object.entries(labels).map(([value, name]) => ({ value, label: name })),
  };
}
export const statusFilter = filterColumn('status', '状态', statusLabels);
// The Community grid handles one filter item. M09 can combine several server conditions.
// Header indicators, the dialog and the applied summary all read the same URL conditions.
const serverFilters: GridFilterModel = { items: [] };
type ListResult<T> = { readable: string[]; items: T[]; total: number; create: boolean };
type Props<T extends { id: string }> = {
  session: Principal;
  path: string;
  title: string;
  searchLabel: string;
  searchFields: string[];
  filters: FilterColumn[];
  read: (query: {
    page: number;
    pageSize: number;
    q: string;
    [key: string]: string | number;
  }) => Promise<ListResult<T>>;
  columns: (can: (field: string) => boolean, link: (row: T) => React.ReactNode) => GridColDef<T>[];
  label: (row: T) => string;
  summary: (row: T) => string;
};

export function ConfigurationList<T extends { id: string }>(props: Props<T>) {
  const { path, title, session, filters } = props;
  usePageTitle(title);
  const [params, setParams] = useMaintenanceSearchParams(path);
  const location = useLocation(),
    navigation = useNavigationType();
  const q = params.get('q') ?? '',
    size = Number(params.get('pageSize')),
    number = Number(params.get('page'));
  const pageSize = [20, 50, 100].includes(size) ? size : 20;
  const page = Number.isSafeInteger(number) && number > 0 ? number : 1;
  const conditions = Object.fromEntries(filters.map(({ field }) => [field, params.get(field) || '']));
  const query = { ...conditions, q, page, pageSize };
  const [leaving, setLeaving] = useState<string>();
  useLayoutEffect(() => setLeaving(undefined), [location.key]);
  const list = useRead(leaving === location.key ? undefined : JSON.stringify(query), () => props.read(query), 'fresh');
  const [draft, setDraft] = useState(q);
  const previousFields = useRef<string[]>([]);
  const forbidden = list.status === 401 || list.status === 403 || list.status === 404;
  const readable = forbidden ? [] : (list.data?.readable ?? previousFields.current);
  const can = (field: string) => readable.includes(field);
  const canSearch = props.searchFields.some(can);
  useLayoutEffect(() => {
    if (forbidden) previousFields.current = [];
    else if (list.data) previousFields.current = list.data.readable;
    if (forbidden || (list.data && !canSearch)) setDraft('');
  }, [list.data, forbidden, canSearch]);
  useLayoutEffect(() => {
    if (navigation === 'POP') setDraft(q);
  }, [location.key, navigation, q]);

  const grid = useGridApiRef(),
    rows = list.data?.items ?? [];
  const returned = useListReturn({
    key: path + session.instanceId + session.subjectId,
    ready: !!list.data,
    apiRef: grid,
    selected: params.get('selected'),
    rows,
    pageOffset: (page - 1) * pageSize,
  });
  function update(fields: Record<string, string>) {
    const applied = new URLSearchParams(window.location.search);
    const next = new URLSearchParams(applied);
    next.delete('selected');
    for (const [field, value] of Object.entries(fields)) {
      if (value) next.set(field, value);
      else next.delete(field);
    }
    if (next.toString() === applied.toString()) {
      list.refresh();
      return;
    }
    flushSync(() => setLeaving(location.key));
    setParams(next, { flushSync: true });
  }
  function apply(fields: Record<string, string>) {
    const keyword = canSearch ? draft.trim() : '';
    setDraft(keyword);
    update({ ...fields, q: keyword, page: '1' });
  }
  function target(id: string) {
    const back = new URLSearchParams(params);
    if (id !== 'new') back.set('selected', id);
    return path + '/' + id + '?back=' + encodeURIComponent(path + (back.size ? '?' + back : ''));
  }
  const link = (row: T) => (
    <Link
      data-object-link
      className="object-name-link"
      to={target(row.id)}
      onClick={(event) => {
        if (new URLSearchParams(window.location.search).toString() !== params.toString()) {
          event.preventDefault();
          return;
        }
        returned.rememberRow(row.id);
      }}
    >
      {props.label(row)}
    </Link>
  );
  const more = useRef<HTMLButtonElement>(null);
  const [menu, setMenu] = useState<HTMLElement | null>(null);
  const [filter, setFilter] = useState<FilterRequest>();
  const compact = useMediaQuery('(max-width:899px)');
  const available = filters.filter((f) => can(f.field));
  const active = available.filter((f) => conditions[f.field]);
  useLayoutEffect(() => {
    if (forbidden || (list.data && filter?.field && !list.data.readable.includes(filter.field)))
      setFilter(undefined);
  }, [forbidden, list.data, filter]);
  function closeFilter() {
    const source = filter?.source;
    setFilter(undefined);
    requestAnimationFrame(() =>
      (source?.isConnected && source.getClientRects().length ? source : more.current)?.focus(),
    );
  }
  const columns = props.columns(can, link).map((column) => {
    const field = column.field;
    const applied = active.some((f) => f.field === field);
    return {
      ...column,
      sortable: false,
      filterable: available.some((f) => f.field === field),
      ...(applied && {
        renderHeader: () => (
          <>
            {column.headerName}
            <IconButton
              size="small"
              aria-label={'编辑' + column.headerName + '筛选'}
              onClick={(e) => {
                e.stopPropagation();
                setFilter({ field, source: e.currentTarget });
              }}
            >
              <FilterAlt fontSize="small" />
            </IconButton>
          </>
        ),
      }),
    };
  });
  const hasQuery = !!q || Object.values(conditions).some(Boolean);
  return (
    <ColumnFilterEditor.Provider value={setFilter}>
      <div className="standard-page" aria-busy={!list.data && !list.error}>
        <DataGridPage
          apiRef={grid}
          scrollerRef={returned.scroller}
          selected={returned.selected}
          label={title}
          rows={rows}
          columns={columns}
          slots={columnFilterSlots}
          filterModel={serverFilters}
          summaryColumn={{
            field: 'summary',
            headerName: title,
            renderCell: ({ row }) => (
              <Stack>
                {link(row)}
                <span>{props.summary(row)}</span>
              </Stack>
            ),
          }}
          paginationModel={{ page: page - 1, pageSize }}
          rowCount={list.data?.total}
          hasNextPage={list.data ? page * pageSize < list.data.total : undefined}
          onPaginationModelChange={(m) => {
            if (list.data && (m.page + 1 !== page || m.pageSize !== pageSize))
              update({
                page: String(m.pageSize === pageSize ? m.page + 1 : 1),
                pageSize: String(m.pageSize),
              });
          }}
          sortModel={[]}
          onSortModelChange={() => {}}
          loading={!list.data && !list.error}
          ready={!!list.data}
          empty={list.data ? '没有符合条件的' + title : ''}
          messages={
            list.error && (
              <Alert
                severity="error"
                action={
                  <>
                    {!forbidden && <Button onClick={list.refresh}>重试</Button>}
                    {hasQuery && (
                      <Button
                        onClick={() => {
                          setDraft('');
                          setFilter(undefined);
                          setParams(new URLSearchParams(), { flushSync: true });
                        }}
                      >
                        清除查询条件
                      </Button>
                    )}
                  </>
                }
              >
                {list.error}
              </Alert>
            )
          }
          toolbar={
            <ListControl
              title={title}
              action={
                list.data?.create && (
                  <Button component={Link} to={target('new')} onClick={returned.remember} variant="contained">
                    新建
                  </Button>
                )
              }
              search={
                canSearch && (
                  <SearchControl
                    value={draft}
                    onChange={setDraft}
                    onSearch={() => apply({})}
                    label={props.searchLabel}
                  />
                )
              }
              more={
                readable.length > 0 &&
                (!compact || available.length > 0) && (
                  <>
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
                    <Menu anchorEl={menu} open={!!menu} onClose={() => setMenu(null)}>
                      {available.length > 0 && (
                        <MenuItem
                          onClick={() => {
                            setMenu(null);
                            setFilter({ source: more.current! });
                          }}
                        >
                          列筛选…
                        </MenuItem>
                      )}
                      {!compact && (
                        <MenuItem
                          onClick={() => {
                            setMenu(null);
                            grid.current?.showPreferences(GridPreferencePanelsValue.columns);
                          }}
                        >
                          列设置…
                        </MenuItem>
                      )}
                    </Menu>
                  </>
                )
              }
              pager={<ListPager />}
            >
              {active.length > 0 && (
                <Box className="list-applied-summary">
                  {active.map((f) => (
                    <Button
                      size="small"
                      key={f.field}
                      onClick={(e) => setFilter({ field: f.field, source: e.currentTarget })}
                    >
                      {f.label}：
                      {f.values?.find((v) => v.value === conditions[f.field])?.label ?? conditions[f.field]}
                    </Button>
                  ))}
                </Box>
              )}
              {canSearch && draft !== q && (
                <Typography className="list-query-summary" role="status">
                  输入已修改，点击查询后生效。已应用关键词：{q || '无'}
                </Typography>
              )}
              {filter && available.length > 0 && (!filter.field || can(filter.field)) && (
                <ColumnFilterDialog
                  key={available.map((f) => f.field).join(',')}
                  request={filter}
                  columns={available}
                  applied={{
                    items: active.map((f) => ({
                      field: f.field,
                      operator: 'equals',
                      value: conditions[f.field],
                    })),
                  }}
                  onApply={(values, field) => apply({ [field]: values.filterValue.trim() })}
                  onClose={closeFilter}
                />
              )}
            </ListControl>
          }
        />
      </div>
    </ColumnFilterEditor.Provider>
  );
}
