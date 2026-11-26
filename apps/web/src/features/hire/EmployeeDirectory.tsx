import { useEffect, useLayoutEffect, useMemo, useRef, useState, type MouseEvent } from 'react';
import { Alert, Box, Button, MenuItem, Stack, Typography, useMediaQuery } from '@mui/material';
import { Link, useLocation, useNavigationType, useSearchParams } from 'react-router-dom';
import { useGridApiRef, type GridColDef, type GridSortModel } from '@mui/x-data-grid';
import { sessionGuard, type Models } from '@chnhcm/api-client';
import { DataGridPage, ListControl, ListPager, ObjectTextField, SearchControl } from '@chnhcm/ui';
import { useListReturn } from '../../shared/navigation';
import * as person from '../personnel/api';

type Row = Models['PersonnelDirectoryRow'];
type Branch = Models['EmployeeAssignmentDirectory']['departments'][number];

export function EmployeeDirectory({ session, register }: { session: Models['Principal']; register: React.ReactNode }) {
  const compact = useMediaQuery('(max-width:899px)');
  const [searching, setSearching] = useState(false);
  const [params, setParams] = useSearchParams(),
    route = useLocation(), navigation = useNavigationType();
  const query = person.directoryQuery(params);
  const listing = person.useRead(JSON.stringify([route.key, query]),
    () => person.directory(session, params));
  const data = searching ? undefined : listing.data;
  const countKey = JSON.stringify({ ...query, page: undefined });
  const lastCount = useRef<{ key: string; total: number } | undefined>(undefined);
  useLayoutEffect(() => {
    if (listing.error) lastCount.current = undefined;
    else if (data) lastCount.current = {key: countKey, total: data.total};
  }, [countKey, data, listing.error]);
  const rowCount = data?.total ?? (!listing.error && lastCount.current?.key === countKey ? lastCount.current.total : undefined);
  const sortModel = useMemo<GridSortModel>(() => [
    { field: query.sort.replace(/^-/, ''), sort: query.sort.startsWith('-') ? 'desc' : 'asc' },
  ], [query.sort]);
  const apiRef = useGridApiRef(),
    [draft, setDraft] = useState(''),
    [error, setError] = useState('');
  const appliedParams = useRef(params);
  useLayoutEffect(() => {
    appliedParams.current = params;
  }, [params]);
  const readyRef = useRef<string | undefined>(undefined),
    request = useRef(0),
    edit = useRef(0);
  const waitingForRef = useRef<string | undefined>(undefined);
  const localSearch = useRef<string | undefined>(undefined);
  useLayoutEffect(() => {
    const local = navigation !== 'POP' && localSearch.current === route.search;
    localSearch.current = undefined;
    if (!local) {
      request.current++;
      waitingForRef.current = undefined;
      readyRef.current = undefined;
      setSearching(false);
      setDraft('');
      setError('');
    }
  }, [route.key, route.search, navigation]);
  useLayoutEffect(() => {
    if (waitingForRef.current && waitingForRef.current === query.ref && (listing.data || listing.error)) {
      waitingForRef.current = undefined;
      setSearching(false);
    }
    if (listing.status === 401 || listing.status === 403) {
      request.current++;
      readyRef.current = undefined;
      setSearching(false);
      setDraft('');
      return;
    }
    if (data && readyRef.current !== (query.ref ?? '')) {
      setDraft(data.keyword);
      readyRef.current = query.ref ?? '';
    }
  }, [data, query.ref, listing.data, listing.error, listing.status]);
  useEffect(
    () => () => {
      request.current++;
    },
    [],
  );
  const returned = useListReturn({
    key: [
      'personnel',
      session.instanceId,
      session.subjectId,
      query.ref,
      query.department,
      query.primary,
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
    if (next.toString() === appliedParams.current.toString()) return;
    appliedParams.current = next;
    localSearch.current = '?' + next;
    setParams(next);
  }
  async function search() {
    const token = ++request.current,
      revision = edit.current,
      current = sessionGuard();
    setSearching(true);
    waitingForRef.current = undefined;
    try {
      const { ref } = await person.queryRef(session, draft);
      if (token === request.current && current()) {
        setError('');
        if (revision !== edit.current) readyRef.current = ref;
        waitingForRef.current = ref;
        update({ ref, page: '1' });
      }
    } catch (e) {
      if (token === request.current && current()) setError(person.message(e));
    } finally {
      if (token === request.current && current() && !waitingForRef.current) setSearching(false);
    }
  }
  const visibleRows = useRef<Set<string>>(new Set());
  useLayoutEffect(() => {
    visibleRows.current = new Set(data?.rows.map(row => row.id));
  }, [data]);
  function openRow(event: MouseEvent<HTMLAnchorElement>, row: Row) {
    if (new URLSearchParams(window.location.search).toString() !== params.toString() || !visibleRows.current.has(row.id)) {
      event.preventDefault();
      return;
    }
    returned.rememberRow(row.id);
  }
  function rowLink(row: Row, compactRow = false) {
    const term = row.assignment;
    return <Link data-object-link className="object-name-link" onClick={e => openRow(e, row)}
      to={person.directoryTarget(params, row.person, term?.id)}>{row.name}
      {compactRow && term && ` · ${term.recordNumber} 号${term.primaryJob ? '主岗' : '兼岗'}`}
    </Link>;
  }
  const columns = useMemo<GridColDef<Row>[]>(() => [
    {field:'name',headerName:'姓名',flex:1,minWidth:120,renderCell:({row})=>rowLink(row)},
    {field:'number',headerName:'人员编号',width:146,renderCell:({row})=>row.number+(row.historical?'（历史编号命中）':'')},
    {field:'recordNumber',headerName:'雇用记录编号',width:118,sortable:false,renderCell:({row})=>row.assignment?.recordNumber ?? '—'},
    {field:'primaryJob',headerName:'主兼岗',width:82,sortable:false,renderCell:({row})=>row.assignment ? row.assignment.primaryJob ? '主岗':'兼岗' : '—'},
    {field:'department',headerName:'部门',flex:1,minWidth:105,sortable:false,renderCell:({row})=>row.assignment?.departmentName || '—'},
    {field:'job',headerName:'职务 / 岗位',flex:1,minWidth:110,sortable:false,renderCell:({row})=>row.assignment ?
      <Stack>{row.assignment.jobName}<small>{row.assignment.positionName || '—'}</small></Stack>:'—'},
    {field:'period',headerName:'任职期间',width:185,sortable:false,renderCell:({row})=>row.assignment ?
      row.assignment.from+' 起'+(row.assignment.until?'，'+row.assignment.until+' 起不在任':''):'—'},
    {field:'status',headerName:'状态',width:95,sortable:false,renderCell:({row})=>row.assignment ? row.future?'尚未生效':'在任' : '—'},
  ], [params, returned.rememberRow]);
  const summary = useMemo<GridColDef<Row>>(() => ({
    field:'summary',headerName:'人员',renderCell:({row})=><Stack>
      {rowLink(row,true)}<small>{[row.number,row.historical?'历史编号命中':'',row.assignment?.departmentName,row.assignment?.jobName,
        row.future?'尚未生效':''].filter(Boolean).join(' · ')}</small>
    </Stack>,
  }),[params,returned.rememberRow]);
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
        label="人员"
        apiRef={apiRef}
        scrollerRef={returned.scroller}
        selected={returned.selected}
        rows={data?.rows ?? []}
        ready={!!data}
        loading={!data && !listing.error}
        empty="没有匹配的人员"
        columns={columns}
        summaryColumn={summary}
        paginationModel={{ page: query.page - 1, pageSize: query.pageSize }}
        rowCount={rowCount}
        onPaginationModelChange={(m) => {
          if (!data || (m.page === query.page - 1 && m.pageSize === query.pageSize)) return;
          update({
            page: String(m.pageSize === query.pageSize ? m.page + 1 : 1),
            pageSize: String(m.pageSize),
          });
        }}
        sortModel={sortModel}
        onSortModelChange={(m) => {
          const sort = (m[0]?.sort === 'desc' ? '-' : '') + (m[0]?.field ?? 'number');
          if (sort !== query.sort) update({ sort, page: '1' });
        }}
        toolbar={
          <ListControl
            title="人员"
            action={register}
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
              <ObjectTextField select label="主兼岗" value={query.primary || ''} slotProps={{select:{displayEmpty:true}}}
                onChange={e=>update({primary:e.target.value,page:'1'})}>
                <MenuItem value="">全部</MenuItem><MenuItem value="primary">主岗</MenuItem><MenuItem value="secondary">兼岗</MenuItem>
              </ObjectTextField>
              {compact && <ObjectTextField select label="排序" value={query.sort} onChange={e=>update({sort:e.target.value,page:'1'})}>
                <MenuItem value="number">人员编号升序</MenuItem><MenuItem value="-number">人员编号降序</MenuItem>
                <MenuItem value="name">姓名升序</MenuItem><MenuItem value="-name">姓名降序</MenuItem>
              </ObjectTextField>}
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
                  {data.assignments} 份任职，{data.people} 人
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
