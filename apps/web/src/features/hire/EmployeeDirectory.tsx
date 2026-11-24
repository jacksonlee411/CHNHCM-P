import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { Alert, Box, Button, MenuItem, Stack, Typography, useMediaQuery } from '@mui/material';
import { Link, useLocation, useSearchParams } from 'react-router-dom';
import { useGridApiRef, type GridColDef } from '@mui/x-data-grid';
import { api, sessionGuard, unwrap, type Models } from '@chnhcm/api-client';
import { DataGridPage, ListControl, ListPager, ObjectTextField, SearchControl } from '@chnhcm/ui';
import { useListReturn, usePageTitle } from '../../shared/navigation';
import * as person from '../personnel/api';

export const employees = '/personnel/assignments';
type Row = Models['EmployeeAssignmentRow'];
type Branch = Models['EmployeeAssignmentDirectory']['departments'][number];

export function EmployeeDirectory({ session }: { session: Models['Principal'] }) {
  usePageTitle('员工');
  const compact = useMediaQuery('(max-width:899px)');
  const [searching, setSearching] = useState(false);
  const [params, setParams] = useSearchParams(),
    route = useLocation();
  const query = {
    ...person.parse(params),
    businessAt: params.get('businessAt') ?? undefined,
    department: params.get('department') ?? undefined,
    person: params.get('person') ?? undefined,
  };
  const listing = person.useRead(JSON.stringify(query), async () =>
    unwrap(
      await api.POST('/api/v1/assignments/query', {
        params: { header: { 'X-CSRF-Token': session.csrf } },
        body: { ...query, pageSize: query.pageSize as 20 | 50 | 100 },
      }),
    ),
  );
  const data = searching ? undefined : listing.data;
  const apiRef = useGridApiRef(),
    [draft, setDraft] = useState(''),
    [error, setError] = useState('');
  const [dateEdit, setDateEdit] = useState<{ base?: string; value: string }>();
  const [dateAtFocus, setDateAtFocus] = useState<string>();
  const dateField = useRef<HTMLInputElement>(null);
  const dateInput = dateEdit && dateEdit.base === query.businessAt ? dateEdit.value : undefined;
  const appliedParams = useRef(params);
  useLayoutEffect(() => {
    appliedParams.current = params;
  }, [params]);
  const readyRef = useRef<string | undefined>(undefined),
    request = useRef(0),
    edit = useRef(0);
  useEffect(() => {
    if (data && readyRef.current !== (query.ref ?? '')) {
      setDraft(data.keyword);
      readyRef.current = query.ref ?? '';
    }
  }, [data, query.ref]);
  useEffect(
    () => () => {
      request.current++;
    },
    [],
  );
  const returned = useListReturn({
    key: [
      'employees',
      session.instanceId,
      session.subjectId,
      query.ref,
      query.department,
      query.person,
      query.businessAt,
    ].join(':'),
    ready: !!data,
    apiRef,
    rows: data?.rows ?? [],
    selected: params.get('selected'),
    pageOffset: (query.page - 1) * query.pageSize,
  });
  function update(fields: Record<string, string>) {
    if (window.location.pathname !== route.pathname) return;
    // A search reply can arrive after another filter has already navigated.
    const next = new URLSearchParams(appliedParams.current);
    next.delete('selected');
    for (const [k, v] of Object.entries(fields)) {
      if (v) next.set(k, v);
      else next.delete(k);
    }
    appliedParams.current = next;
    setParams(next);
  }
  async function search() {
    const token = ++request.current,
      revision = edit.current,
      current = sessionGuard();
    setSearching(true);
    try {
      const { ref } = await person.queryRef(session, draft);
      if (token === request.current && current()) {
        setError('');
        if (revision !== edit.current) readyRef.current = ref;
        update({ ref, page: '1' });
      }
    } catch (e) {
      if (token === request.current && current()) setError(person.message(e));
    } finally {
      if (token === request.current && current()) setSearching(false);
    }
  }
  const columns = useMemo<GridColDef<Row>[]>(() => {
    const link = (r: Row) => {
      const next = new URLSearchParams(params);
      next.set('selected', r.id);
      next.set('from', 'assignments');
      if (!next.has('businessAt') && data) next.set('businessAt', data.businessAt);
      return (
        <Link
          data-object-link
          className="object-name-link"
          onClick={() => returned.rememberRow(r.id)}
          to={person.path + '/' + r.person + '/assignments/' + r.id + '?' + next}
        >
          {r.name}
        </Link>
      );
    };
    return [
      { field: 'name', headerName: '姓名', flex: 1, minWidth: 120, renderCell: ({ row }) => link(row) },
      { field: 'number', headerName: '人员编号', width: 124 },
      { field: 'recordNumber', headerName: '雇用记录编号', width: 118, sortable: false },
      {
        field: 'primaryJob',
        headerName: '主兼岗',
        width: 82,
        sortable: false,
        renderCell: ({ row }) => (row.primaryJob ? '主岗' : '兼岗'),
      },
      { field: 'departmentName', headerName: '部门', flex: 1, minWidth: 115, sortable: false },
      {
        field: 'jobName',
        headerName: '职务 / 岗位',
        flex: 1,
        minWidth: 110,
        sortable: false,
        renderCell: ({ row }) => (
          <Stack>
            {row.jobName}
            <small>{row.positionName || '—'}</small>
          </Stack>
        ),
      },
      {
        field: 'from',
        headerName: '任职期间',
        minWidth: 170,
        sortable: false,
        renderCell: ({ row }) =>
          row.until ? `${row.from} 起，${row.until} 起不在任` : '自 ' + row.from + ' 起',
      },
      { field: 'state', headerName: '状态', width: 75, sortable: false, renderCell: () => '在任' },
    ];
  }, [params, data?.businessAt, returned.rememberRow]);
  const summary = useMemo<GridColDef<Row>>(
    () => ({
      field: 'summary',
      headerName: '员工任职',
      renderCell: ({ row }) => {
        const q = new URLSearchParams(params);
        q.set('from', 'assignments');
        q.set('selected', row.id);
        if (data) q.set('businessAt', data.businessAt);
        return (
          <Stack>
            <Link
              data-object-link
              to={person.path + '/' + row.person + '/assignments/' + row.id + '?' + q}
              onClick={() => returned.rememberRow(row.id)}
            >
              {row.name} · {row.recordNumber} 号{row.primaryJob ? '主岗' : '兼岗'}
            </Link>
            <small>
              {[row.number, row.departmentName, row.jobName, row.positionName].filter(Boolean).join(' · ')}
            </small>
          </Stack>
        );
      },
    }),
    [params, data?.businessAt, returned.rememberRow],
  );
  function branch(nodes: Branch[], parent = ''): React.ReactNode {
    return nodes
      .filter((n) => (n.parent ?? '') === parent)
      .map((n) => (
        <Box key={n.id} sx={{ pl: parent ? 1.5 : 0 }}>
          <Button
            aria-pressed={query.department === n.id}
            onClick={() => update({ department: n.id, page: '1' })}
          >
            {n.name}
          </Button>
          {branch(nodes, n.id)}
        </Box>
      ));
  }
  return (
    <div className="standard-page">
      <DataGridPage<Row>
        label="员工任职列表"
        apiRef={apiRef}
        scrollerRef={returned.scroller}
        selected={returned.selected}
        rows={data?.rows ?? []}
        ready={!!data}
        loading={!data && !listing.error}
        empty="该日没有可查看的任职"
        columns={columns}
        summaryColumn={summary}
        paginationModel={{ page: query.page - 1, pageSize: query.pageSize }}
        rowCount={data?.total ?? 0}
        onPaginationModelChange={(m) =>
          update({
            page: String(m.pageSize === query.pageSize ? m.page + 1 : 1),
            pageSize: String(m.pageSize),
          })
        }
        sortModel={[
          { field: query.sort.replace(/^-/, ''), sort: query.sort.startsWith('-') ? 'desc' : 'asc' },
        ]}
        onSortModelChange={(m) => {
          const sort = (m[0]?.sort === 'desc' ? '-' : '') + (m[0]?.field ?? 'number');
          if (sort !== query.sort) update({ sort, page: '1' });
        }}
        toolbar={
          <ListControl
            title="员工"
            action={
              <Button component={Link} to={person.path}>
                人员档案
              </Button>
            }
            pager={<ListPager />}
            search={
              <SearchControl
                value={draft}
                onChange={(value) => {
                  edit.current++;
                  readyRef.current = query.ref ?? '';
                  setDraft(value);
                }}
                onSearch={search}
                label="姓名或人员编号"
              />
            }
          >
            <Stack direction="row" flexWrap="wrap" gap={2} alignItems="center">
              {query.person && (
                <Button onClick={() => update({ person: '', page: '1', selected: '' })}>查看全部员工</Button>
              )}
              <ObjectTextField
                inputRef={dateField}
                type="date"
                label="业务日期"
                value={dateInput ?? query.businessAt ?? dateAtFocus ?? data?.businessAt ?? ''}
                onFocus={(e) => setDateAtFocus(e.target.value)}
                onChange={(e) => {
                  const value = e.target.value;
                  setDateEdit({ base: query.businessAt, value });
                  if (value && e.target.validity.valid) update({ businessAt: value, page: '1' });
                }}
                onBlur={() => {
                  setDateAtFocus(undefined);
                  const field = dateField.current;
                  if (field && dateInput !== undefined && field.validity.valid && field.value !== (appliedParams.current.get('businessAt') ?? '')) {
                    update({ businessAt: field.value, page: '1' });
                  }
                }}
              />
              {compact && (
                <ObjectTextField
                  select
                  label="任职部门"
                  value={query.department ?? ''}
                  onChange={(e) => update({ department: e.target.value, page: '1' })}
                  slotProps={{ select: { displayEmpty: true } }}
                >
                  <MenuItem value="">全部部门</MenuItem>
                  {data?.departments.map((d) => (
                    <MenuItem key={d.id} value={d.id}>
                      {d.name}
                    </MenuItem>
                  ))}
                </ObjectTextField>
              )}
              {data && (
                <Typography>
                  {data.total} 份任职，{data.people} 人
                </Typography>
              )}
            </Stack>
          </ListControl>
        }
        sidebar={
          <Box
            component="nav"
            aria-label="任职部门"
            sx={{ display: compact ? 'none' : 'block', width: 204, flexShrink: 0, overflow: 'auto' }}
          >
            <Button aria-pressed={!query.department} onClick={() => update({ department: '', page: '1' })}>
              全部部门
            </Button>
            {branch(data?.departments ?? [])}
          </Box>
        }
        messages={
          (error || listing.error) && (
            <Alert severity="error" action={<Button onClick={listing.refresh}>重试</Button>}>
              {error || listing.error}
            </Alert>
          )
        }
      />
    </div>
  );
}
