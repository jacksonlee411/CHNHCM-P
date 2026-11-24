import { useEffect, useMemo, useRef, useState } from 'react';
import { Alert, Button, Stack, Typography } from '@mui/material';
import { Link, useSearchParams } from 'react-router-dom';
import { useGridApiRef, type GridColDef } from '@mui/x-data-grid';
import { api, unwrap, sessionGuard, type Models } from '@chnhcm/api-client';
import {
  DataGridPage,
  ListControl,
  ListPager,
  ObjectTextField,
  SearchControl,
  recordedTime,
} from '@chnhcm/ui';
import { useListReturn, usePageTitle } from '../../shared/navigation';
import * as person from '../personnel/api';

export const registerPath = '/personnel/actions';
type Row = Models['EmploymentRegisterRow'];
type Result = Models['EmploymentRegister'];
const kinds = { Hire: '入职', DataChange: '数据更改', Transfer: '调动' } as const;
const statuses = { effective: '已生效', future: '尚未生效', cancelled: '已撤销', deleted: '已删除' } as const;
const dates = {
  businessFrom: '生效起日',
  businessTo: '生效止日',
  recordedFrom: '办理起日',
  recordedTo: '办理止日',
};
const scopes = {
  department: '发生部门',
  fromDepartment: '调出部门',
  toDepartment: '调入部门',
  operator: '原经办人',
};
const fields = ['kind', 'status', ...Object.keys(dates), ...Object.keys(scopes)];
type Sort = 'recordedAt' | '-recordedAt' | 'businessDate' | '-businessDate';

function SelectFilter({
  name,
  label,
  choices,
  initial,
}: {
  name: string;
  label: string;
  choices: { id: string; name: string }[];
  initial: string;
}) {
  const [value, setValue] = useState(initial);
  return (
    <ObjectTextField
      key={name}
      select
      label={label}
      name={name}
      value={value}
      onChange={(e) => setValue(e.target.value)}
      slotProps={{ select: { native: true }, inputLabel: { shrink: true } }}
    >
      <option value="">全部</option>
      {choices.map((v) => (
        <option key={v.id} value={v.id}>
          {v.name}
        </option>
      ))}
    </ObjectTextField>
  );
}

function Filters({
  params,
  data,
  apply,
}: {
  params: URLSearchParams;
  data?: Result;
  apply: (values: Record<string, string>) => void;
}) {
  function select(name: string, label: string, choices: { id: string; name: string }[]) {
    return (
      <SelectFilter key={name} name={name} label={label} choices={choices} initial={params.get(name) ?? ''} />
    );
  }
  const options={department:data?.departments,fromDepartment:data?.fromDepartments,toDepartment:data?.toDepartments,operator:data?.operators};
  const named = (record: Record<string, string>) =>
    Object.entries(record).map(([id, name]) => ({ id, name }));
  return (
    <details>
      <summary>筛选条件</summary>
      <Stack
        key={params.toString()}
        component="form"
        direction="row"
        flexWrap="wrap"
        gap={1.5}
        sx={{ p: 1 }}
        onSubmit={(e) => {
          e.preventDefault();
          const values = new FormData(e.currentTarget);
          apply(Object.fromEntries(fields.map((name) => [name, String(values.get(name) ?? '')])));
        }}
      >
        {select('kind', '办理类型', named(kinds))}
        {select('status', '办理状态', named(statuses))}
        {Object.entries(dates).map(([name, label]) => (
          <RegisterDate key={name} name={name} label={label} initial={params.get(name) ?? ''} />
        ))}
        {Object.entries(scopes).map(([name, label]) =>
          select(name, label, options[name as keyof typeof options] ?? []),
        )}
        <Button type="submit">应用筛选</Button>
        <Button onClick={() => apply(Object.fromEntries(fields.map((key) => [key, ''])))}>清除筛选</Button>
      </Stack>
    </details>
  );
}

function RegisterDate({ name, label, initial }: { name: string; label: string; initial: string }) {
  const [value, setValue] = useState(initial);
  return (
    <ObjectTextField
      type="date"
      name={name}
      label={label}
      value={value}
      onChange={(e) => setValue(e.target.value)}
    />
  );
}

export function EmploymentRegister({ session }: { session: Models['Principal'] }) {
  usePageTitle('办理记录');
  const [params, setParams] = useSearchParams(),
    apiRef = useGridApiRef();
  const value = (name: string) => params.get(name) || undefined;
  const page = Number(value('page') ?? 1),
    pageSize = Number(value('pageSize') ?? 20) as 20 | 50 | 100;
  const ref = value('ref'),
    sort = (value('sort') ?? '-recordedAt') as Sort;
  const query = {
    ref,
    sort,
    page,
    pageSize,
    kind: value('kind') as Row['kind'] | undefined,
    status: value('status') as Row['status'] | undefined,
    department: value('department'),
    fromDepartment: value('fromDepartment'),
    toDepartment: value('toDepartment'),
    operator: value('operator'),
    businessFrom: value('businessFrom'),
    businessTo: value('businessTo'),
    recordedFrom: value('recordedFrom'),
    recordedTo: value('recordedTo'),
  };
  const listing = person.useRead(JSON.stringify(query), async () =>
    unwrap(await api.GET('/api/v1/employment-actions', { params: { query } })),
  );
  const data = listing.data,
    [draft, setDraft] = useState(''),
    [error, setError] = useState('');
  const initialized = useRef<string | undefined>(undefined),
    request = useRef(0),
    edit = useRef(0);
  useEffect(() => {
    if (data && initialized.current !== (ref ?? '')) {
      setDraft(data.keyword);
      initialized.current = ref ?? '';
    }
  }, [data, ref]);
  useEffect(
    () => () => {
      request.current++;
    },
    [],
  );
  const returned = useListReturn({
    key: ['employment-register', session.subjectId, session.instanceId, JSON.stringify(query)].join(':'),
    ready: !!data,
    apiRef,
    selected: value('selected') ?? null,
    rows: data?.rows ?? [],
    pageOffset: (page - 1) * pageSize,
  });
  function update(values: Record<string, string>) {
    if (location.pathname !== registerPath) return;
    const next = new URLSearchParams(location.search);
    next.delete('selected');
    for (const [key, v] of Object.entries(values)) {
      if (v) next.set(key, v);
      else next.delete(key);
    }
    if (next.toString() !== location.search.slice(1)) setParams(next);
  }
  async function search() {
    const token = ++request.current,
      rev = edit.current,
      current = sessionGuard();
    try {
      const saved = await person.queryRef(session, draft);
      if (token !== request.current || !current()) return;
      if (edit.current !== rev) initialized.current = saved.ref;
      setError('');
      update({ ref: saved.ref, page: '1' });
    } catch (e) {
      if (token === request.current && current()) setError(person.message(e));
    }
  }
  const { columns, summary } = useMemo(() => {
    const link = (r: Row, compact = false) => {
      const text = compact ? kinds[r.kind] + ' · ' + r.name : kinds[r.kind];
      if (!r.canHistory) return text;
      const q = new URLSearchParams(params);
      q.set('selected', r.id);
      return (
        <Link
          data-object-link
          to={registerPath + '/' + r.id + '?' + q}
          onClick={() => returned.rememberRow(r.id)}
        >
          {text}
        </Link>
      );
    };
    const record = (r: Row) => r.recordNumber + ' 号' + (r.recordNumber === 0 ? '主岗' : '兼岗');
    const columns: GridColDef<Row>[] = [
      {
        field: 'kind',
        headerName: '办理类型',
        minWidth: 105,
        sortable: false,
        renderCell: ({ row }) => link(row),
      },
      {
        field: 'name',
        headerName: '员工',
        flex: 1,
        minWidth: 130,
        sortable: false,
        renderCell: ({ row }) => (
          <Stack>
            {row.name}
            <small>{row.number}</small>
          </Stack>
        ),
      },
      {
        field: 'recordNumber',
        headerName: '办理对象',
        minWidth: 108,
        sortable: false,
        renderCell: ({ row }) => record(row),
      },
      { field: 'businessDate', headerName: '生效日期', width: 125 },
      { field: 'departmentName', headerName: '发生部门', minWidth: 110, flex: 1, sortable: false, renderCell:({row})=>row.kind==='Transfer'?[row.fromDepartmentName && '调出 '+row.fromDepartmentName,row.toDepartmentName && '调入 '+row.toDepartmentName].filter(Boolean).join(' → '):row.departmentName },
      { field: 'operator', headerName: '经办人', minWidth: 100, sortable: false },
      {
        field: 'recordedAt',
        headerName: '首次办理时间',
        width: 182,
        renderCell: ({ row }) => recordedTime(row.recordedAt),
      },
      {
        field: 'status',
        headerName: '办理状态',
        width: 103,
        sortable: false,
        renderCell: ({ row }) => statuses[row.status],
      },
    ];
    const summary: GridColDef<Row> = {
      field: 'summary',
      headerName: '办理记录',
      renderCell: ({ row }) => (
        <Stack>
          {link(row, true)}
          <small>
            {row.businessDate} · {record(row)} · {statuses[row.status]}
          </small>
        </Stack>
      ),
    };
    return { columns, summary };
  }, [params, returned.rememberRow]);
  return (
    <div className="standard-page">
      <DataGridPage<Row>
        label="办理记录列表"
        apiRef={apiRef}
        scrollerRef={returned.scroller}
        selected={returned.selected}
        ready={!!data}
        loading={!data && !listing.error}
        rows={data?.rows ?? []}
        empty="没有匹配的办理记录"
        rowCount={data?.total ?? 0}
        columns={columns}
        summaryColumn={summary}
        paginationModel={{ page: page - 1, pageSize }}
        onPaginationModelChange={(m) =>
          update({ page: String(m.pageSize === pageSize ? m.page + 1 : 1), pageSize: String(m.pageSize) })
        }
        sortModel={[{ field: sort.replace('-', ''), sort: sort.startsWith('-') ? 'desc' : 'asc' }]}
        onSortModelChange={(m) => {
          const v = m[0]?.sort ? (m[0].sort === 'desc' ? '-' : '') + m[0].field : '-recordedAt';
          if (v !== sort) update({ sort: v, page: '1' });
        }}
        toolbar={
          <ListControl
            title="办理记录"
            pager={<ListPager />}
            search={
              <SearchControl
                label="姓名或人员编号"
                value={draft}
                onChange={(v) => {
                  edit.current++;
                  initialized.current = ref ?? '';
                  setDraft(v);
                }}
                onSearch={search}
              />
            }
          >
            <Filters params={params} data={data} apply={(values) => update({ ...values, page: '1' })} />
            {data && <Typography>{data.total} 笔办理</Typography>}
          </ListControl>
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
