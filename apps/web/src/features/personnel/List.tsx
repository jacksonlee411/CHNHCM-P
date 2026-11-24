import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import { Link, useLocation, useNavigate, useSearchParams } from 'react-router-dom';
import { Alert, Button, Stack, IconButton, Menu, MenuItem, useMediaQuery } from '@mui/material';
import { useGridApiRef, type GridColDef } from '@mui/x-data-grid';
import { DataGridPage, ListControl, ListPager, SearchControl } from '@chnhcm/ui';
import { sessionGuard, type Models } from '@chnhcm/api-client';
import { useListReturn, usePageTitle } from '../../shared/navigation';
import * as person from './api';
type Row = Models['PersonSummary'];
export function PersonList({ session }: { session: person.Principal }) {
  usePageTitle('员工');
  const [params, setParams] = useSearchParams(),
    query = person.parse(params),
    navigate = useNavigate();
  const location = useLocation();
  const access = person.useRead('entry', person.entry);
  const listing = person.useRead(access.data?.canRead ? JSON.stringify(query) : undefined, () =>
    person.list(query),
  );
  const data = listing.data,
    apiRef = useGridApiRef();
  const [draft, setDraft] = useState(''),
    [error, setError] = useState(''),
    request = useRef(0),
    draftRevision = useRef(0);
  const [initialized, setInitialized] = useState<string>();
  useEffect(() => {
    if (data && initialized !== (query.ref || '')) {
      setDraft(data.keyword);
      setInitialized(query.ref || '');
    }
  }, [data, query.ref, initialized]);
  useEffect(() => {
    if ((access.data && !access.data.canRead) || listing.status === 403 || listing.status === 404) {
      setDraft('');
      setInitialized(undefined);
    }
  }, [access.data, listing.status]);
  useEffect(
    () => () => {
      request.current++;
    },
    [],
  );
  const returned = useListReturn({
    key: 'person-list:' + session.instanceId + ':' + session.subjectId + ':' + (query.ref || ''),
    ready: !!data,
    apiRef,
    rows: data?.rows || [],
    selected: params.get('selected'),
    pageOffset: (query.page - 1) * query.pageSize,
  });
  function patch(values: Record<string, string>) {
    if (window.location.pathname !== location.pathname) return;
    const next = new URLSearchParams(params);
    next.delete('selected');
    for (const [key, value] of Object.entries(values)) {
      if (value) next.set(key, value);
      else next.delete(key);
    }
    setParams(next);
  }
  async function search() {
    const token = ++request.current,
      revision = draftRevision.current,
      current = sessionGuard();
    try {
      const { ref } = await person.queryRef(session, draft);
      if (token !== request.current || !current()) return;
      setError('');
      if (revision !== draftRevision.current) setInitialized(ref);
      patch({ ref, page: '1' });
    } catch (e) {
      if (token === request.current && current()) setError(person.message(e));
    }
  }
  const [sortMenu, setSortMenu] = useState<HTMLElement | null>(null),
    compact = useMediaQuery('(max-width:899px)');
  const link = useCallback(
    (row: Row) => {
      const next = new URLSearchParams(params);
      next.set('from', 'list');
      next.set('selected', row.id);
      return (
        <Link
          data-object-link
          className="object-name-link"
          to={person.path + '/' + row.id + '?' + next}
          onClick={() => {
            request.current++;
            returned.rememberRow(row.id);
          }}
        >
          {row.name}
        </Link>
      );
    },
    [params, returned.rememberRow],
  );
  const columns = useMemo<GridColDef<Row>[]>(
    () => [
      {
        field: 'number',
        headerName: '人员编号',
        minWidth: 170,
        renderCell: ({ row }) => (
          <>
            {row.number}
            {row.historical && '（历史编号命中）'}
          </>
        ),
      },
      { field: 'name', headerName: '姓名', flex: 1, renderCell: ({ row }) => link(row) },
    ],
    [link],
  );
  const summary = useMemo<GridColDef<Row>>(
    () => ({
      field: 'summary',
      headerName: '人员',
      renderCell: ({ row }) => (
        <Stack>
          {link(row)}
          <small>
            {row.number}
            {row.historical && ' · 历史编号'}
          </small>
        </Stack>
      ),
    }),
    [link],
  );
  const pagination = useMemo(
    () => ({ page: query.page - 1, pageSize: query.pageSize }),
    [query.page, query.pageSize],
  );
  const sortModel = useMemo(
    () => [
      {
        field: query.sort.replace(/^-/, ''),
        sort: query.sort.startsWith('-') ? ('desc' as const) : ('asc' as const),
      },
    ],
    [query.sort],
  );
  const button = access.data?.canRegister && (
    <Button
      variant="contained"
      onClick={() => {
        request.current++;
        navigate(person.path + '/new?' + params);
      }}
    >
      登记人员
    </Button>
  );
  return (
    <div className="standard-page">
      {location.state?.registered && access.data?.canRegister && (
        <Alert severity="success">人员登记已保存</Alert>
      )}
      {access.error && (
        <Alert severity="error" action={<Button onClick={access.refresh}>重试</Button>}>
          {access.error}
        </Alert>
      )}
      {access.data && !access.data.canRead && (
        <ListControl title="员工" action={button}>
          {!button && '没有人员名录权限'}
        </ListControl>
      )}
      {access.data?.canRead && (
        <DataGridPage<Row>
          label="员工名录"
          apiRef={apiRef}
          scrollerRef={returned.scroller}
          selected={returned.selected}
          loading={!data && !listing.error}
          ready={!!data}
          empty="没有匹配的人员"
          columns={columns}
          summaryColumn={summary}
          rows={data?.rows || []}
          rowCount={data?.total}
          paginationModel={pagination}
          onPaginationModelChange={(m) =>
            patch({
              page: String(m.pageSize === query.pageSize ? m.page + 1 : 1),
              pageSize: String(m.pageSize),
            })
          }
          sortModel={sortModel}
          onSortModelChange={(m) => {
            const sort = (m[0]?.sort === 'desc' ? '-' : '') + (m[0]?.field || 'number');
            if (sort !== query.sort) patch({ page: '1', sort });
          }}
          messages={
            (listing.error || error) && (
              <Alert
                severity="error"
                action={
                  <>
                    <Button onClick={listing.refresh}>重试</Button>
                    <Button
                      onClick={() => {
                        setDraft('');
                        setError('');
                        setParams({});
                      }}
                    >
                      重新查询
                    </Button>
                  </>
                }
              >
                {listing.error || error}
              </Alert>
            )
          }
          toolbar={
            <ListControl
              title="员工"
              action={button}
              more={
                compact && (
                  <>
                    <IconButton aria-label="名录排序" onClick={(e) => setSortMenu(e.currentTarget)}>
                      <MoreVertIcon />
                    </IconButton>
                    <Menu open={!!sortMenu} anchorEl={sortMenu} onClose={() => setSortMenu(null)}>
                      {Object.entries({
                        number: '编号升序',
                        '-number': '编号降序',
                        name: '姓名升序',
                        '-name': '姓名降序',
                      }).map(([value, label]) => (
                        <MenuItem
                          key={value}
                          selected={query.sort === value}
                          onClick={() => {
                            setSortMenu(null);
                            patch({ sort: value, page: '1' });
                          }}
                        >
                          {label}
                        </MenuItem>
                      ))}
                    </Menu>
                  </>
                )
              }
              pager={<ListPager />}
              search={
                <SearchControl
                  label="姓名或人员编号"
                  value={draft}
                  onChange={(v) => {
                    draftRevision.current++;
                    setInitialized(query.ref || '');
                    setDraft(v);
                  }}
                  onSearch={search}
                />
              }
            >
              {data && draft !== data.keyword && '输入已修改，点击查询后生效'}
            </ListControl>
          }
        />
      )}
    </div>
  );
}
