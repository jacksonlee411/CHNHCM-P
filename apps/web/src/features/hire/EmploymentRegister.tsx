import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { Alert, Button, Stack, Typography } from '@mui/material';
import { Link, useLocation, useNavigationType, useSearchParams } from 'react-router-dom';
import { useGridApiRef, type GridColDef, type GridSortModel } from '@mui/x-data-grid';
import { api, ApiError, unwrap, sessionGuard, type Models } from '@chnhcm/api-client';
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
import { registerListParams } from './jobLinks';

export const registerPath = '/personnel/actions';
type Row = Models['EmploymentRegisterRow'];
type Result = Models['EmploymentRegister'];
const kinds = {
  Hire: '入职',
  DataChange: '数据更改',
  Transfer: '调动',
  AdditionalAssignment: '新增兼岗',
  ParallelEmployment: '并行关系聘任',
  ProbationStart: '试用开始',
  ConfirmEmployment: '转正',
  SetManager:'实际汇报',EndAssignment:'结束任职',Leave:'离职',Rehire:'再聘',
  GradeRecord:'补录实际职级',
  Promotion:'晋升',
} as const;
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
  const panel=useRef<HTMLDetailsElement>(null), trigger=useRef<HTMLElement>(null);
  const [open,setOpen]=useState(false);
  function close() {
    if(panel.current)panel.current.open=false;
    setOpen(false);
    trigger.current?.focus();
  }
  const summary=fields.flatMap(key=>{
    const value=params.get(key);
    if(!value)return [];
    if(key==='kind')return kinds[value as keyof typeof kinds] || [];
    if(key==='status')return statuses[value as keyof typeof statuses] || [];
    if(key in dates)return dates[key as keyof typeof dates]+' '+value;
    const option=options[key as keyof typeof options]?.find(x=>x.id===value);
    return option ? scopes[key as keyof typeof scopes]+' '+option.name : [];
  }).join('、');
  return (
    <details ref={panel} onToggle={e=>setOpen(e.currentTarget.open)} onKeyDown={e=>{
      if(e.key==='Escape' && open){e.preventDefault();close();}
    }}>
      <summary ref={trigger}>筛选条件{summary && ' · '+summary}</summary>
      {open && <Stack
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
          close();
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
        <Button onClick={()=>{apply(Object.fromEntries(fields.map(key=>[key,''])));close();}}>清除筛选</Button>
        <Button onClick={close}>取消</Button>
      </Stack>}
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

export function registerQuery(params: URLSearchParams) {
  const value = (name: string) => params.get(name) || undefined;
  const page = Number(value('page') ?? 1),
    pageSize = Number(value('pageSize') ?? 20) as 20 | 50 | 100;
  const ref = value('ref'),
    sort = (value('sort') ?? '-recordedAt') as Sort;
  return {
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
}

export function EmploymentRegister({ session }: { session: Models['Principal'] }) {
  usePageTitle('办理记录');
  const [params, setParams] = useSearchParams(),
    apiRef = useGridApiRef();
  const route = useLocation(), navigation = useNavigationType();
  const value = (name: string) => params.get(name) || undefined;
  const query = registerQuery(params);
  const {page, pageSize, ref, sort} = query;
  const listing = person.useRead(JSON.stringify([route.key, query]), async () =>
    unwrap(await api.GET('/api/v1/employment-actions', { params: { query } })),
  );
  const data = listing.data,
    [draft, setDraft] = useState(''),
    [error, setError] = useState('');
  // A name, type or date filter can legitimately have no intersection with
  // a selected department. Check the authorized scope on its own before
  // removing an applied selection or discarding its display name.
  const scopeFields = {department:'departments',fromDepartment:'fromDepartments',toDepartment:'toDepartments',operator:'operators'} as const;
  const missingScopes = (Object.keys(scopeFields) as (keyof typeof scopeFields)[]).filter(name =>
    data && params.has(name) && !data[scopeFields[name]]?.some(o=>o.id===params.get(name)));
  const scopeQuery = person.useRead(missingScopes.length ? JSON.stringify([route.key, query]) : undefined, () =>
    api.GET('/api/v1/employment-actions',{params:{query:{page:1,pageSize:20}}}).then(unwrap), 'fresh');
  const filterData = data && {...data};
  if (filterData && scopeQuery.data) {
    for (const name of missingScopes) {
      const field=scopeFields[name];
      const option=scopeQuery.data[field]?.find(o=>o.id===params.get(name));
      if (option) filterData[field]=[...(filterData[field]??[]),option];
    }
  }
  const unavailable = missingScopes.filter(name =>
    scopeQuery.data && !scopeQuery.data[scopeFields[name]]?.some(option=>option.id===params.get(name))).join(',');
  useEffect(()=>{
    if (!unavailable) return;
    const next=new URLSearchParams(params);
    for(const name of unavailable.split(',')) next.delete(name);
    next.delete('selected'); next.set('page','1');
    setParams(next,{replace:true});
  },[unavailable,params,setParams]);
  const countKey=JSON.stringify({...query,page:undefined});
  const lastCount=useRef<{key:string;total:number} | undefined>(undefined);
  useEffect(()=>{if(data)lastCount.current={key:countKey,total:data.total};},[countKey,data]);
  const knownTotal = data?.total ?? (!listing.error && lastCount.current?.key===countKey ? lastCount.current.total : undefined);
  const localSearch = useRef<string | undefined>(undefined);
  const request = useRef(0);
  const initializeDraft = useRef(true);
  const appliedName = useRef<{ref: string; keyword: string} | undefined>(undefined);
  const nameRef = ref ?? '';
  useLayoutEffect(() => {
    const local = navigation !== 'POP' && localSearch.current === route.search;
    localSearch.current = undefined;
    request.current++;
    setError('');
    if (!local) {
      initializeDraft.current = true;
      appliedName.current = undefined;
      setDraft('');
    }
  }, [route.key, route.search, navigation]);
  useLayoutEffect(() => {
    if (listing.status === 401 || listing.status === 403) {
      request.current++;
      initializeDraft.current = true;
      appliedName.current = undefined;
      lastCount.current = undefined;
      setDraft('');
      return;
    }
    if (!data) return;
    appliedName.current = {ref:nameRef, keyword:data.keyword};
    if (initializeDraft.current) {
      setDraft(data.keyword);
      initializeDraft.current = false;
    }
  }, [data, nameRef, listing.status]);
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
    const next = registerListParams(new URLSearchParams(location.search));
    next.delete('selected');
    for (const [key, v] of Object.entries(values)) {
      if (v) next.set(key, v);
      else next.delete(key);
    }
    if (next.toString() !== location.search.slice(1)) {
      localSearch.current = '?' + next;
      setParams(next);
    } else listing.refresh();
  }
  const denied = listing.status === 401 || listing.status === 403;
  async function apply(values: Record<string,string> = {}, newQuery = false) {
    if (denied) return;
    const path = location.pathname + location.search;
    const token = ++request.current, current = sessionGuard();
    const active = () => token === request.current && current() && location.pathname + location.search === path;
    try {
      const word = draft.trim();
      const applied = data?.keyword ?? (appliedName.current?.ref === nameRef ? appliedName.current.keyword : undefined);
      const carryApplied = initializeDraft.current || !newQuery && listing.status !== 404 && word === applied;
      let nextRef = nameRef;
      if (!carryApplied) {
        if (word) {
          const access = unwrap(await api.GET('/api/v1/employment-actions/context'));
          if (!active()) return;
          if (!access.canList) {
            initializeDraft.current = true;
            appliedName.current = undefined;
            setDraft('');
            listing.invalidate();
            return;
          }
          nextRef = (await person.queryRef(session, word)).ref;
        } else nextRef = '';
        if (!active()) return;
        appliedName.current = {ref:nextRef, keyword:word};
      }
      setError('');
      update({ ...values, ref: nextRef, page: '1' });
    } catch (e) {
      if (active()) {
        setError(person.message(e));
        if (e instanceof ApiError && (e.status === 401 || e.status === 403)) {
          initializeDraft.current = true;
          appliedName.current = undefined;
          setDraft('');
          listing.invalidate();
        }
      }
    }
  }
  const { columns, summary } = useMemo(() => {
    const link = (r: Row, compact = false) => {
      const text = compact ? kinds[r.kind] + ' · ' + r.name : kinds[r.kind];
      if (!r.canHistory) return text;
      const q = registerListParams(params);
      q.set('fromRegister', '1');
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
    const record = (r: Row) => r.relation ? [r.relation.employerName,'工作关系',r.relation.from].filter(Boolean).join(' · ') : r.recordNumber + ' 号' + (r.recordNumber === 0 ? '主岗' : '兼岗');
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
  const sortModel = useMemo<GridSortModel>(() => [
    {field:sort.replace('-', ''), sort:sort.startsWith('-') ? 'desc' : 'asc'},
  ], [sort]);
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
        rowCount={knownTotal}
        columns={columns}
        summaryColumn={summary}
        paginationModel={{ page: page - 1, pageSize }}
        onPaginationModelChange={(m) => {
          if (!data || (m.page===page-1 && m.pageSize===pageSize)) return;
          request.current++;
          update({page:String(m.pageSize===pageSize ? m.page+1 : 1),pageSize:String(m.pageSize)});
        }}
        sortModel={sortModel}
        onSortModelChange={(m) => {
          const v = m[0]?.sort ? (m[0].sort === 'desc' ? '-' : '') + m[0].field : '-recordedAt';
          if (v !== sort) void apply({sort:v});
        }}
        toolbar={
          <ListControl
            title="办理记录"
            pager={<ListPager />}
            search={!denied &&
              <SearchControl
                label="姓名或人员编号"
                value={draft}
                onChange={(v) => {
                  initializeDraft.current = false;
                  setDraft(v);
                }}
                onSearch={() => void apply({}, true)}
              />
            }
          >
            {!denied && <Filters params={params} data={filterData} apply={(values) => void apply(values)} />}
            {data && draft.trim() !== data.keyword && <Typography role="status" variant="body2">姓名或人员编号已修改，查询后生效。</Typography>}
            {data && <Typography>{data.total} 笔办理</Typography>}
          </ListControl>
        }
        messages={
          (error || listing.error) && (
            <Alert severity="error" action={<Button onClick={() => {setError(''); listing.refresh();}}>重试</Button>}>
              {error || listing.error}
            </Alert>
          )
        }
      />
    </div>
  );
}
