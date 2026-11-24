import { useEffect, useMemo, useRef, useState } from 'react';
import { Alert, Box, Button, Paper, Stack, Typography } from '@mui/material';
import { useGridApiRef, type GridColDef } from '@mui/x-data-grid';
import {
  Link,
  useNavigate,
  useParams,
  useSearchParams,
} from 'react-router-dom';
import { api, unwrap, type Models } from '@chnhcm/api-client';
import {
  DataGridPage,
  ListControl,
  ListPager,
  SearchControl,
  ObjectForm,
  ObjectTextField,
  ObjectDetailField,
  ObjectDetailSection,
  RecordBackButton,
  RecordControl,
  UnsavedChangesDialog,
  recordedTime,
} from '@chnhcm/ui';
import { useDraftExit } from '../../shared/dated/useDraftExit';
import { useListReturn, usePageTitle } from '../../shared/navigation';
import { useRead, path as people } from '../personnel/api';
import { Target, type Option } from '../hire/Target';
import { useRuleSubmission } from '../readiness/useRuleSubmission';

import { OfferHireHandoff } from './OfferHireHandoff';

export const offerPath = '/recruiting/offers';
type Session = Models['Principal'];
type Offer = Models['RecruitingOffer'];
type Change = Models['OfferChange'];
type Operation = Change['operation'];
type Key = 'department' | 'job' | 'employer' | 'position';
const operations: Record<Operation, string> = {
  issue: '发出录用',
  reschedule: '调整录用日期',
  accept: '登记接受',
  decline: '登记放弃',
  reserve: '重新预留',
  release: '释放预留',
};
const statuses: Record<string, string> = {
  issued: '已发出',
  accepted: '已接受',
  declined: '已放弃',
  converted: '已入职',
};
export const offerAccess = (person?: string) =>
  api
    .GET('/api/v1/recruiting/context', { params: { query: { person } } })
    .then(unwrap);
const detailURL = (id: string, params: URLSearchParams) =>
  offerPath + '/' + id + '?' + params.toString();

export function OfferList({ session }: { session: Session }) {
  usePageTitle('招聘录用');
  const [params, setParams] = useSearchParams(),
    apiRef = useGridApiRef();
  const person = params.get('person') || undefined,
    keyword = params.get('q') ?? '';
  const [text, setText] = useState(keyword);
  const data = useRead(JSON.stringify([session.subjectId, person]), () =>
    api
      .GET('/api/v1/recruiting/offers', { params: { query: { person } } })
      .then(unwrap),
  );
  const rows = useMemo(
    () =>
      (data.data?.offers ?? []).filter((v) =>
        [v.number, v.personName].some((s) => s?.includes(keyword)),
      ),
    [data.data, keyword],
  );
  const page = Math.max(0, Number(params.get('page') || 1) - 1),
    size = Number(params.get('size') || 20);
  const visible = rows.slice(page * size, (page + 1) * size);
  const returned = useListReturn({
    key: JSON.stringify(['offer-list', session.subjectId, person, keyword]),
    ready: !!data.data,
    apiRef,
    selected: params.get('selected'),
    rows: visible,
    pageOffset: page * size,
  });
  function update(changes: Record<string, string>) {
    const q = new URLSearchParams(params);
    q.delete('selected');
    for (const [k, v] of Object.entries(changes))
      if (v) q.set(k, v);
      else q.delete(k);
    setParams(q);
  }
  const link = (v: Offer) => {
    const q = new URLSearchParams(params);
    q.set('selected', v.id);
    return (
      <Link
        data-object-link
        to={detailURL(v.id, q)}
        onClick={() => returned.rememberRow(v.id)}
      >
        {v.number}
      </Link>
    );
  };
  const cols: GridColDef<Offer>[] = [
    {
      field: 'number',
      headerName: '录用编号',
      flex: 1,
      minWidth: 155,
      renderCell: ({ row }) => link(row),
    },
    { field: 'personName', headerName: '候选人', flex: 1, minWidth: 120 },
    { field: 'arrival', headerName: '约定到岗日', width: 132 },
    {
      field: 'status',
      headerName: '录用状态',
      width: 112,
      renderCell: ({ row }) => statuses[row.status],
    },
  ];
  return (
    <div className="standard-page">
      <DataGridPage<Offer>
        apiRef={apiRef}
        scrollerRef={returned.scroller}
        selected={returned.selected}
        label="招聘录用列表"
        loading={!data.data && !data.error}
        ready={!!data.data}
        empty="没有匹配的录用"
        rows={visible}
        rowCount={rows.length}
        columns={cols.map((c) => ({ ...c, sortable: false }))}
        summaryColumn={{
          field: 'summary',
          headerName: '录用',
          flex: 1,
          sortable: false,
          renderCell: ({ row }) => (
            <Stack>
              {link(row)}
              <small>
                {row.personName} · {row.arrival} · {statuses[row.status]}
              </small>
            </Stack>
          ),
        }}
        sortModel={[]}
        onSortModelChange={() => {}}
        paginationModel={{ page, pageSize: size }}
        onPaginationModelChange={(v) =>
          update({
            page: String(v.pageSize === size ? v.page + 1 : 1),
            size: String(v.pageSize),
          })
        }
        toolbar={
          <ListControl
            title="招聘录用"
            pager={<ListPager />}
            search={
              <SearchControl
                label="录用编号或姓名"
                value={text}
                onChange={setText}
                onSearch={() => update({ q: text.trim(), page: '1' })}
              />
            }
          >
            {data.data?.canIssue && (
              <Button
                component={Link}
                to={person ? detailURL('new', params) : people}
              >
                {person ? '发出录用' : '选择录用人员'}
              </Button>
            )}
          </ListControl>
        }
        messages={data.error && <Alert severity="error">{data.error}</Alert>}
      />
    </div>
  );
}

export function OfferDetail({
  session,
  creating = false,
}: {
  session: Session;
  creating?: boolean;
}) {
  const { id = '' } = useParams(),
    [params] = useSearchParams(),
    navigate = useNavigate();
  const person = params.get('person') ?? '',
    header = { 'X-CSRF-Token': session.csrf };
  const [revision, setRevision] = useState(params.get('sourceVersion') ?? ''),
    [op, setOp] = useState<Operation | undefined>(
      creating ? 'issue' : undefined,
    );
  const [day, setDay] = useState(''),
    [basis, setBasis] = useState(''),
    [number, setNumber] = useState(''),
    [arrival, setArrival] = useState(''),
    [start, setStart] = useState(''),
    [last, setLast] = useState('');
  const [options, setOptions] = useState<Partial<Record<Key, Option>>>({});
  const [error, setError] = useState(''),
    [message, setMessage] = useState('');
  const saved = useRef(false),
    failure = useRef<HTMLDivElement>(null),
    opener = useRef<HTMLButtonElement | null>(null);
  const entry = useRead(creating ? person : undefined, () =>
    offerAccess(person),
  );
  const view = useRead(
    !creating ? JSON.stringify([id, revision]) : undefined,
    () =>
      api
        .GET('/api/v1/recruiting/offers/{id}', {
          params: {
            path: { id },
            query: { sourceVersion: revision || undefined },
          },
        })
        .then(unwrap),
  );
  const item = view.data;
  const readMode = (position?: string) =>
    api
      .POST('/api/v1/recruiting/arrangement', {
        params: { header },
        body: { person, arrival: start, lastDay: last || undefined, position },
      })
      .then(unwrap);
  const mode = useRead(
    creating && person && start
      ? JSON.stringify([person, start, last])
      : undefined,
    () => readMode(),
  );
  const m = mode.data;
  const derived = useRead(
    creating && m?.positionRequired && !m.jobRequired && options.position
      ? JSON.stringify([options.position.id, start, last])
      : undefined,
    () => readMode(options.position?.id),
  );
  const scheduleChanged =
    arrival !== item?.arrival ||
    start !== item?.reserveFrom ||
    last !== (item?.lastDay ?? '');
  const dirty = !!(
    day ||
    basis ||
    (op === 'reschedule'
      ? scheduleChanged
      : creating &&
        (number || arrival || start || last || Object.keys(options).length))
  );
  const close = () => {
    setOp(undefined);
    setDay('');
    setBasis('');
    if (!creating) requestAnimationFrame(() => opener.current?.focus());
  };
  const leave = useDraftExit(!!op && dirty, () => saved.current, close);
  const send = useRuleSubmission<Change, Models['OfferReceipt']>(
    'recruiting-offer-' + (creating ? person : id),
    session,
    {
      send: (body, key) =>
        creating
          ? api
              .POST('/api/v1/recruiting/offers', {
                params: { header: { ...header, 'Idempotency-Key': key } },
                body,
              })
              .then(unwrap)
          : api
              .POST('/api/v1/recruiting/offers/{id}/changes', {
                params: {
                  path: { id },
                  header: { ...header, 'Idempotency-Key': key },
                },
                body,
              })
              .then(unwrap),
      resolve: (sub) =>
        api
          .POST('/api/v1/recruiting/submissions/{sub}/resolve', {
            params: { path: { sub }, header },
            body: {},
          })
          .then(unwrap),
      finished: (r) => {
        if (r.outcome !== 'accepted' || !r.id) {
          setError('原提交未办理。');
          return;
        }
        saved.current = true;
        close();
        setRevision('');
        setError('');
        setMessage('已保存录用办理。');
        if (creating) navigate(detailURL(r.id, params), { replace: true });
        else view.refresh();
      },
      failed: (e) => {
        setError(e instanceof Error ? e.message : '保存失败');
        view.refresh();
      },
    },
  );
  const beginSubmission = send.start;
  useEffect(() => {
    if (creating) beginSubmission();
  }, [creating, beginSubmission]);
  useEffect(() => {
    if (error) failure.current?.focus();
  }, [error]);
  const title = creating ? '发出录用' : (item?.number ?? '招聘录用');
  usePageTitle(title);
  const backParams = new URLSearchParams(params);
  backParams.delete('sourceVersion');
  const back = offerPath + '?' + backParams.toString();
  const ready =
    !!op &&
    !!day &&
    !!basis.trim() &&
    (op !== 'reschedule' || (!!arrival && !!start && scheduleChanged)) &&
    (!creating ||
      (entry.data?.canIssue &&
        !!number.trim() &&
        !!arrival &&
        !!start &&
        !!options.employer &&
        !!m &&
        (!m.jobRequired || (!!options.department && !!options.job)) &&
        (!m.positionRequired ||
          (!!options.position &&
            (m.jobRequired || !!derived.data?.department)))));
  const input: Models['HireInput'] = {
    person,
    arrival: start,
    lastDay: last || undefined,
    employer: options.employer?.id ?? '',
    department: m?.jobRequired ? options.department?.id : undefined,
    job: m?.jobRequired ? options.job?.id : undefined,
  };
  function choose(kind: Key) {
    return (
      <Target
        recruiting
        key={kind}
        kind={kind}
        draft={input}
        session={session}
        choice={options[kind]}
        onChange={(v) => setOptions((old) => ({ ...old, [kind]: v }))}
      />
    );
  }
  function submit() {
    if (!ready || !op || send.busy) return;
    void send.submit({
      operation: op,
      businessDate: day,
      basis,
      expected: item?.version,
      dates:
        op === 'reschedule'
          ? { arrival, reserveFrom: start, lastDay: last || undefined }
          : undefined,
      terms: creating
        ? {
            number,
            person,
            employer: input.employer,
            department: input.department,
            job: input.job,
            position: m?.positionRequired ? options.position?.id : undefined,
            arrival,
            reserveFrom: start,
            lastDay: last || undefined,
          }
        : undefined,
    });
  }
  return (
    <Box className="standard-page object-page">
      <RecordControl
        navigation={null}
        identity={
          <>
            <RecordBackButton component={Link} to={back} />
            招聘录用
          </>
        }
        actions={null}
      />
      <Box
        className="page-scroll"
        data-page-scroll
        tabIndex={0}
        role="region"
        aria-label="录用详情"
      >
        <Paper sx={{ p: { xs: 1.5, sm: 3 }, maxWidth: 1000, mx: 'auto' }}>
          <Typography component="h1" variant="h5">
            {title}
          </Typography>
          {(error ||
            view.error ||
            entry.error ||
            mode.error ||
            derived.error) && (
            <Alert tabIndex={-1} ref={failure} severity="error">
              {error ||
                view.error ||
                entry.error ||
                mode.error ||
                derived.error}
            </Alert>
          )}
          {message && <Alert severity="success">{message}</Alert>}
          {send.pending.map((ref) => (
            <Button
              key={ref.submissionId}
              onClick={() => void send.recover(ref)}
            >
              查回原提交
            </Button>
          ))}
          {send.storageError && (
            <Alert severity="error">
              无法保存提交号，请恢复浏览器存储后办理。
            </Alert>
          )}
          {item && (
            <ObjectForm>
              <ObjectDetailField label="候选人">
                {item.personName}
              </ObjectDetailField>
              <ObjectDetailField label="用人单位">
                {item.employerName}
              </ObjectDetailField>
              <ObjectDetailField label="约定到岗日">
                {item.arrival}
              </ObjectDetailField>
              <ObjectDetailField label="录用状态">
                {statuses[item.status]}
              </ObjectDetailField>
              <ObjectDetailField label="办理依据">
                {item.basis}
              </ObjectDetailField>
              <ObjectDetailField label="登记时间">
                <time dateTime={item.recordedAt}>
                  {recordedTime(item.recordedAt)}
                </time>
              </ObjectDetailField>
              {!!item.segments?.length && (
                <ObjectDetailSection title="录用安排">
                  {item.segments.map((s) => (
                    <ObjectDetailField key={s.from} label={s.from + ' 起'}>
                      {[
                        s.departmentName,
                        s.jobName,
                        s.positionName,
                        s.seats ? '预留 1 编' : '免预留',
                      ]
                        .filter(Boolean)
                        .join(' · ')}
                      {s.until ? '（至 ' + s.until + ' 前）' : ''}
                    </ObjectDetailField>
                  ))}
                </ObjectDetailSection>
              )}
              {!op && (
                <OfferHireHandoff
                  item={item}
                  session={session}
                  historical={!!revision}
                  refresh={view.refresh}
                />
              )}
              {!op && !revision && (
                <Stack direction="row" gap={1} flexWrap="wrap">
                  {item.actions?.map((action) => (
                    <Button
                      key={action}
                      onClick={(e) => {
                        opener.current = e.currentTarget;
                        saved.current = false;
                        setOp(action as Operation);
                        if (action === 'reschedule') {
                          setArrival(item.arrival);
                          setStart(item.reserveFrom);
                          setLast(item.lastDay ?? '');
                        }
                        setDay('');
                        setBasis('');
                        setError('');
                        setMessage('');
                        send.start();
                      }}
                    >
                      {operations[action as Operation]}
                    </Button>
                  ))}
                </Stack>
              )}
              {!op && (
                <ObjectDetailSection title="办理历史">
                  {item.revisions?.map((r) => (
                    <Button
                      key={r.version}
                      aria-pressed={r.version === item.version}
                      onClick={() => setRevision(r.version)}
                    >
                      {r.businessDate} ·{' '}
                      {operations[r.operation as Operation] ??
                        (
                          {
                            convert: '转入职',
                            cancel_conversion: '撤销未来入职',
                          } as Record<string, string>
                        )[r.operation] ??
                        r.operation}
                    </Button>
                  ))}
                  {revision && (
                    <Button onClick={() => setRevision('')}>
                      返回当前录用
                    </Button>
                  )}
                </ObjectDetailSection>
              )}
            </ObjectForm>
          )}
          {op && (!creating || entry.data?.canIssue) && (
            <Box
              component="form"
              onSubmit={(e) => {
                e.preventDefault();
                submit();
              }}
            >
              <ObjectForm>
                {!creating && (
                  <Typography component="h2" variant="h6">
                    {operations[op]}
                  </Typography>
                )}
                {creating && (
                  <>
                    <ObjectTextField
                      required
                      label="录用编号"
                      value={number}
                      onChange={(e) => setNumber(e.target.value)}
                    />
                  </>
                )}
                {(creating || op === 'reschedule') && (
                  <>
                    <ObjectTextField
                      required
                      type="date"
                      label="约定到岗日"
                      value={arrival}
                      onChange={(e) => {
                        setArrival(e.target.value);
                        if (!start) setStart(e.target.value);
                      }}
                    />
                    <ObjectTextField
                      required
                      type="date"
                      label="安排起日"
                      value={start}
                      onChange={(e) => setStart(e.target.value)}
                    />
                    <ObjectTextField
                      type="date"
                      label="任期至"
                      value={last}
                      onChange={(e) => setLast(e.target.value)}
                    />
                  </>
                )}
                {creating && (
                  <>
                    {choose('employer')}
                    {m?.jobRequired && (
                      <>
                        {choose('department')}
                        {choose('job')}
                      </>
                    )}
                    {m?.positionRequired && choose('position')}
                    {derived.data?.department && (
                      <ObjectDetailField label="岗位所属部门">
                        {derived.data.department.name}
                      </ObjectDetailField>
                    )}
                    {derived.data?.job && (
                      <ObjectDetailField label="岗位对应职务">
                        {derived.data.job.name}
                      </ObjectDetailField>
                    )}
                  </>
                )}
                <ObjectTextField
                  required
                  type="date"
                  label="业务日期"
                  value={day}
                  onChange={(e) => setDay(e.target.value)}
                />
                <ObjectTextField
                  required
                  label="办理依据"
                  value={basis}
                  onChange={(e) => setBasis(e.target.value)}
                />
                <Stack direction="row" gap={1}>
                  <Button
                    type="submit"
                    variant="contained"
                    disabled={!ready || send.busy}
                  >
                    保存
                  </Button>
                  <Button
                    onClick={() =>
                      leave.request(() => (creating ? navigate(back) : close()))
                    }
                  >
                    取消
                  </Button>
                </Stack>
              </ObjectForm>
            </Box>
          )}
        </Paper>
      </Box>
      <UnsavedChangesDialog {...leave.dialog} />
    </Box>
  );
}
