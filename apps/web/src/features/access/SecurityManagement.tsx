import { useEffect, useMemo, useRef, useState } from 'react';
import { Alert, Box, Button, Stack, Typography } from '@mui/material';
import SecurityOutlined from '@mui/icons-material/SecurityOutlined';
import { useGridApiRef, type GridColDef } from '@mui/x-data-grid';
import {
  Link,
  useLocation,
  useNavigate,
  useParams,
  useSearchParams,
} from 'react-router-dom';
import { ApiError } from '@chnhcm/api-client';
import {
  DataGridPage,
  ListControl,
  ListPager,
  SearchControl,
  ObjectDetailSheet,
  ObjectDetailSection,
  ObjectDetailField,
  ObjectDetailIdentity,
  ObjectForm,
  ObjectTabPanel,
  RecordBackButton,
  RecordControl,
  UnsavedChangesDialog,
  recordedTime,
} from '@chnhcm/ui';
import { useRead } from '../personnel/api';
import { usePageTitle, useListReturn } from '../../shared/navigation';
import {
  ColumnFilterEditor,
  columnFilterSlots,
} from '../../shared/dated/ColumnFilter';
import { useDraftExit } from '../../shared/dated/useDraftExit';
import { useRuleSubmission } from '../readiness/useRuleSubmission';
import { ManagementFields, Pick } from './ManagementFields';
import {
  read,
  send,
  resolve,
  label,
  version,
  versionName,
  targetName,
  areaNames,
  accessPath,
  type Area,
  type Session,
  type View,
  type Command,
  type Definition,
} from './management';
import {
  AccountFilters,
  BusinessGrants,
  GrantFacts,
  GrantTabs,
  MandateFacts,
  VersionReferences,
  WorkspacePager,
} from './GrantWorkspace';
import './security.css';

type Row = {
  id: string;
  name: string;
  description: string;
  state: string;
  login?: string;
  count?: number;
};
type Result = Awaited<ReturnType<typeof send>>;
const blank = (area: Area, kind: string): Command =>
  area === 'accounts'
    ? { operation: 'account.create' }
    : { operation: 'version.publish', version: { kind, name: '' } };
const editVersion = (d: Definition): Command => ({
  operation: 'version.publish',
  version: {
    kind: d.kind,
    name: d.name,
    definitionId: d.definitionId,
    number: d.number,
    actions: d.actions,
    readFields: d.readFields,
    writeFields: d.writeFields,
    targetKind: d.targetKind,
    targetId: d.targetId,
    scope: d.scope,
    fields: d.fields,
  },
});
function ManagementList({ area, session }: { area: Area; session: Session }) {
  const [params, set] = useSearchParams(),
    navigate = useNavigate(),
    apiRef = useGridApiRef();
  const base = accessPath + '/' + area,
    title = areaNames[area];
  const [input, setInput] = useState(params.get('q') ?? '');
  const applied = params.get('q') ?? '';
  useEffect(() => setInput(applied), [applied]);
  const [filters, setFilters] = useState(false);
  const [filterField, setFilterField] = useState('state');
  const filterReturn = useRef<HTMLElement | null>(null);
  const page = Math.max(0, Number(params.get('page')) || 0),
    pageSize = Number(params.get('size')) || 20;
  const query = {
    q: params.get('q') ?? '',
    owner: params.get('owner') ?? '',
    state: params.get('state') ?? '',
    page,
    pageSize,
  };
  const data = useRead(
      session.subjectId + area + JSON.stringify(query),
      () => read(area, query),
      'fresh',
    ),
    v = data.data;
  const rows: Row[] = useMemo(
    () =>
      area === 'roles'
        ? (v?.versions ?? []).map((d) => ({
            id: d.id!,
            name: versionName(d),
            description: label(d.kind),
            state: d.disabled ? '停用' : '已发布',
          }))
        : (v?.accounts ?? []).map((a) => ({
            id: a.id,
            name: a.displayName || a.login,
            login: a.displayName ? a.login : undefined,
            count: a.activeGrants,
            description: targetName(v!, a.owner),
            state: label(a.state),
          })),
    [v, area],
  );
  const selected = params.get('selected');
  const restore = useListReturn({
    key: 'x03-list-' + area + '-' + session.subjectId,
    ready: !!v,
    apiRef,
    selected,
    rows,
    pageOffset: page * pageSize,
  });
  const update = (changes: Record<string, string>) => {
    const next = new URLSearchParams(params);
    next.delete('selected');
    for (const [k, value] of Object.entries(changes)) {
      if (value) next.set(k, value);
      else next.delete(k);
    }
    set(next);
  };
  const details = (id: string) => base + '/' + id + '?' + params;
  const cols: GridColDef<Row>[] = [
    {
      field: 'name',
      filterable: false,
      headerName: area === 'roles' ? '版本名称' : '账号',
      flex: 2,
      minWidth: 165,
      sortable: false,
      renderCell: (p) => (
        <Link
          data-object-link
          to={details(p.row.id)}
          onClick={() => restore.rememberRow(p.row.id)}
        >
          {p.row.name}
          {p.row.login && <small className="grant-login">{p.row.login}</small>}
        </Link>
      ),
    },
    {
      field: 'description',
      headerName: area === 'roles' ? '版本类型' : '管理归属',
      flex: 1,
      minWidth: 145,
      sortable: false,
    },
    {
      field: 'state',
      headerName: area === 'roles' ? '版本状态' : '账号状态',
      width: 118,
      sortable: false,
    },
    ...(area === 'grants' && v?.rights?.includes('assign')
      ? [
          {
            field: 'count',
            filterable: false,
            headerName: '生效授予（可见）',
            width: 160,
            sortable: false,
            renderCell: ({ row }: { row: Row }) => row.count ?? '—',
          },
        ]
      : []),
  ];
  usePageTitle(title);
  const add = (kind: string, text: string) => (
    <Button
      key={kind}
      variant="contained"
      onClick={() => navigate(details('new-' + kind))}
    >
      {text}
    </Button>
  );
  const actions = (
    <>
      {area === 'accounts' &&
        v?.rights?.includes('account') &&
        add('account', '开户')}
      {area === 'roles' &&
        v?.rights?.includes('role') &&
        add('role', '发布角色')}
      {area === 'roles' && v?.rights?.includes('policy') && (
        <>
          {add('scope', '发布范围策略')}
          {add('fields', '发布字段策略')}
        </>
      )}
    </>
  );
  return (
    <div className="standard-page">
      <ColumnFilterEditor.Provider
        value={({ field, source }) => {
          filterReturn.current = source;
          setFilterField(field ?? 'state');
          setFilters(true);
        }}
      >
        <DataGridPage
          slots={area === 'roles' ? undefined : columnFilterSlots}
          filterModel={area === 'roles' ? undefined : { items: [] }}
          rows={rows}
          columns={cols}
          summaryColumn={{
            field: 'summary',
            headerName: title,
            flex: 1,
            sortable: false,
            renderCell: (p) => (
              <div>
                <Link
                  data-object-link
                  to={details(p.row.id)}
                  onClick={() => restore.rememberRow(p.row.id)}
                >
                  {p.row.name}
                  {p.row.login && (
                    <small className="grant-login">{p.row.login}</small>
                  )}
                </Link>
                <div>
                  {p.row.description} · {p.row.state}
                  {p.row.count !== undefined &&
                    ' · 生效授予 ' + p.row.count + ' 份'}
                </div>
              </div>
            ),
          }}
          label={title}
          selected={selected}
          apiRef={apiRef}
          scrollerRef={restore.scroller}
          ready={!!v}
          loading={!v && !data.error}
          empty="没有符合条件的记录"
          sortModel={[]}
          onSortModelChange={() => {}}
          paginationModel={{ page, pageSize }}
          rowCount={v?.total ?? 0}
          onPaginationModelChange={(m) =>
            update({ page: String(m.page), size: String(m.pageSize) })
          }
          messages={
            data.error && (
              <Alert
                severity="error"
                action={<Button onClick={data.refresh}>重试</Button>}
              >
                {data.error}
              </Alert>
            )
          }
          toolbar={
            <ListControl
              title={title}
              action={actions}
              more={
                area !== 'roles' && (
                  <Button
                    onClick={(e) => {
                      filterReturn.current = e.currentTarget;
                      setFilterField('state');
                      setFilters(true);
                    }}
                  >
                    筛选
                  </Button>
                )
              }
              pager={<ListPager />}
              search={
                <SearchControl
                  label={
                    area === 'roles' ? '搜索版本名称' : '搜索登录名或显示名'
                  }
                  value={input}
                  onChange={setInput}
                  onSearch={() => update({ q: input, page: '0' })}
                />
              }
            >
              {area !== 'roles' && (query.state || query.owner) && (
                <Button size="small" onClick={() => setFilters(true)}>
                  {[
                    query.state && label(query.state),
                    query.owner && v && targetName(v, query.owner),
                  ]
                    .filter(Boolean)
                    .join(' · ')}
                </Button>
              )}
              <AccountFilters
                open={filters}
                field={filterField}
                onClose={() => {
                  setFilters(false);
                  requestAnimationFrame(() => filterReturn.current?.focus());
                }}
                state={query.state}
                owner={query.owner}
                v={v}
                apply={(state, owner) =>
                  update({ state, owner, q: input, page: '0' })
                }
              />
            </ListControl>
          }
        />
      </ColumnFilterEditor.Provider>
    </div>
  );
}

function History({
  v,
  follow,
}: {
  v: View;
  follow?: (target: string, mandate: boolean, event: string) => void;
}) {
  return (
    <ObjectDetailSection title={follow ? undefined : '管理变更'}>
      {v.events?.length ? (
        <ol
          className={
            follow ? 'access-history grant-event-list' : 'access-history'
          }
        >
          {v.events.map((e) => (
            <li key={e.id}>
              <div className={follow ? 'grant-event-heading' : undefined}>
                <strong>{label(e.operation)}</strong>
                {e.summary && <span>{e.summary}</span>}
                <span>{e.actor}</span>
                <time dateTime={e.at}>{recordedTime(e.at)}</time>
              </div>
              {follow && (
                <Button
                  size="small"
                  id={'event-' + e.id}
                  onClick={() =>
                    follow(e.target, e.operation.startsWith('mandate.'), e.id)
                  }
                >
                  查看
                  {e.operation.startsWith('mandate.') ? '这份职责' : '这份授予'}
                </Button>
              )}
              {e.previousOwner && (
                <div>原管理归属 · {targetName(v, e.previousOwner)}</div>
              )}
              {e.assignedOwner && (
                <div>管理归属 · {targetName(v, e.assignedOwner)}</div>
              )}
              {!follow && e.reason && <div>{e.reason}</div>}
              <details>
                <summary>{follow ? '变更明细' : '提交身份'}</summary>
                {e.before && <div>变更前 · {e.before}</div>}
                {e.after && <div>变更后 · {e.after}</div>}
                {follow && e.reason && <div>办理事由 · {e.reason}</div>}
                {follow && (e.before || e.after) && (
                  <small>
                    业务范围名称按当前可见资料显示，版本取自当时的修订。
                  </small>
                )}
                提交编号 {e.request}
                {follow && <div>办理目标 {e.target}</div>}
              </details>
            </li>
          ))}
        </ol>
      ) : (
        <Typography>暂无管理变更</Typography>
      )}
    </ObjectDetailSection>
  );
}
function ManagementDetail({
  id,
  area,
  session,
}: {
  id: string;
  area: Area;
  session: Session;
}) {
  const base = accessPath + '/' + area,
    creating = id.startsWith('new-'),
    kind = id.slice(4);
  const navigate = useNavigate(),
    [params, setParams] = useSearchParams(),
    location = useLocation();
  const [draft, setDraft] = useState<Command>(),
    [dirty, setDirty] = useState(false),
    [error, setError] = useState(''),
    [notice, setNotice] = useState<string>(location.state?.notice ?? '');
  const [resultTarget, setResultTarget] = useState('');
  const choicesNeeded =
    !!draft &&
    ['grant.issue', 'grant.adopt', 'mandate.issue'].includes(draft.operation);
  const panel = params.get('tab') ?? '';
  const detailQuery: Record<string, string | number> =
    creating || area === 'roles' ? { object: id } : { subject: id };
  if (
    area === 'grants' ||
    (area === 'roles' &&
      (panel === 'references' || draft?.operation === 'version.disable'))
  ) {
    Object.assign(detailQuery, {
      panel: area === 'roles' ? 'references' : panel,
      q:
        area === 'grants' && (!panel || panel === 'grants')
          ? (params.get('gq') ?? '')
          : '',
      state:
        draft?.operation === 'version.disable'
          ? 'impact'
          : (params.get('gstate') ?? ''),
      source: params.get('gsource') ?? '',
      version: params.get('gversion') ?? '',
      record:
        panel === 'events'
          ? (params.get('eventFor') ?? '')
          : (draft?.id ?? params.get('grant') ?? params.get('ref') ?? ''),
      page: Number(params.get('gp')) || 0,
      pageSize: Number(params.get('gs')) || 20,
      choices: choicesNeeded ? '1' : '',
    });
  }
  const key =
    session.instanceId + session.subjectId + area + JSON.stringify(detailQuery);
  const data = useRead(key, () => read(area, detailQuery), 'fresh'),
    v = data.data;
  const account = v?.accounts?.find((a) => a.id === id),
    def = v?.versions?.find((d) => d.id === id);
  const change = (values: Record<string, string>) => {
    const next = new URLSearchParams(params);
    for (const [k, value] of Object.entries(values)) {
      if (value) next.set(k, value);
      else next.delete(k);
    }
    setParams(next);
  };
  const origin = useRef(''),
    focusAfter = useRef('');
  useEffect(() => {
    if (!v) return;
    if (draft) {
      formStart.current?.focus();
      return;
    }
    const requested =
      focusAfter.current ||
      (params.get('via') === 'grants'
        ? 'account-grants'
        : panel === 'events' && params.get('originEvent')
          ? 'event-' + params.get('originEvent')
          : params.get('inspect') === '1'
            ? 'grant-detail-heading'
            : params.get('grant') && !params.get('inspect')
              ? 'row-' + params.get('grant')
              : params.get('ref')
                ? 'ref-' + params.get('ref')
                : '');
    if (requested) {
      (
        document.getElementById(requested) ??
        document.getElementById('grant-panel-heading')
      )?.focus();
      focusAfter.current = '';
    }
  }, [v, draft?.operation, params]);
  const conflict = useRef(false),
    saved = useRef(false),
    formStart = useRef<HTMLElement>(null);
  const close = () => {
    saved.current = true;
    setDraft(undefined);
    setDirty(false);
  };
  const back = new URLSearchParams(params);
  for (const k of [
    'grant',
    'tab',
    'gq',
    'gstate',
    'gsource',
    'gversion',
    'gp',
    'gs',
    'ref',
    'inspect',
    'eventFor',
    'eventOrigin',
    'fromAccount',
    'fromVersion',
    'returnQuery',
    'via',
    'eventPage',
    'originEvent',
  ])
    back.delete(k);
  if (!creating) back.set('selected', id);
  const fromAccount = area === 'grants' && params.get('fromAccount') === '1';
  const fromVersion =
    area === 'grants' &&
    /^[0-9a-f-]{36}$/i.test(params.get('fromVersion') ?? '')
      ? params.get('fromVersion')
      : null;
  const returnParams = new URLSearchParams(params.get('returnQuery') ?? '');
  if (fromAccount) returnParams.set('via', 'grants');
  if (fromVersion) {
    returnParams.set('tab', 'references');
    if (params.get('grant')) returnParams.set('ref', params.get('grant')!);
  }
  const backTo = fromAccount
    ? accessPath + '/accounts/' + id + '?' + returnParams
    : fromVersion
      ? accessPath + '/roles/' + fromVersion + '?' + returnParams
      : base + '?' + back;
  const leave = useDraftExit(dirty, () => saved.current, close);
  const flow = useRuleSubmission<Command, Result>('access-' + area, session, {
    send: (c, k) => send(session, c, k),
    resolve: (k) => resolve(session, k),
    finished: (r) => {
      saved.current = true;
      close();
      setError('');
      if (r.outcome === 'not_applied') {
        setNotice('原提交未办理。');
        return;
      }
      const message =
        label(r.operation) +
        '完成 · ' +
        recordedTime(r.at) +
        (r.operation === 'account.create' ? '。开户时业务授予为空。' : '');
      setNotice(message);
      data.refresh();
      const target = r.subject ?? r.target,
        next = new URLSearchParams(params);
      if (area === 'grants') {
        next.set('grant', r.target);
        next.set(
          'tab',
          r.operation.startsWith('mandate.') ? 'mandates' : 'grants',
        );
        next.delete('inspect');
        setResultTarget(r.target);
        focusAfter.current = 'row-' + r.target;
      }
      if (target !== id || area === 'grants')
        navigate(base + '/' + target + '?' + next, {
          replace: creating,
          state: { notice: message },
        });
    },
    failed: (e) => {
      setError(e instanceof Error ? e.message : '暂时无法确认办理结果');
      if (
        e instanceof ApiError &&
        e.status === 403 &&
        e.code !== 'AUTH_REQUEST_VALIDATION_FAILED'
      ) {
        close();
        data.invalidate();
      }
      if (e instanceof ApiError && e.status === 409) {
        conflict.current = true;
        data.refresh();
      }
    },
  });
  const open = (cmd: Command) => {
    origin.current = (document.activeElement as HTMLElement)?.id ?? '';
    flow.start();
    saved.current = false;
    setDraft(cmd);
    setDirty(false);
    setError('');
    setNotice('');
    requestAnimationFrame(() => formStart.current?.focus());
  };
  const initial = useRef(false);
  useEffect(() => {
    if (creating && v && !initial.current) {
      initial.current = true;
      if (
        v.rights?.includes(
          kind === 'account' ? 'account' : kind === 'role' ? 'role' : 'policy',
        )
      )
        open(blank(area, kind));
    }
  }, [creating, v]);
  const right =
    draft?.operation === 'account.reset'
      ? 'credential'
      : draft?.operation.startsWith('account.')
        ? 'account'
        : draft?.operation.startsWith('grant.')
          ? 'assign'
          : draft?.operation.startsWith('mandate.')
            ? 'delegate'
            : draft?.version?.kind === 'role'
              ? 'role'
              : 'policy';
  const candidateLost = (d: Command, current: View) => {
    if (
      d.owner &&
      !current.owners?.some(
        (t) => t.id === d.owner && t.options?.includes(right),
      )
    )
      return true;
    if (
      d.limit?.owner &&
      !current.owners?.some(
        (t) => t.id === d.limit?.owner && t.options?.includes('delegate'),
      )
    )
      return true;
    const binding = d.limit ?? d;
    if (
      (['role', 'scope', 'fields'] as const).some(
        (k) =>
          binding[k] &&
          !version(current, k, binding[k])?.options?.includes(right),
      )
    )
      return true;
    if (
      d.version?.definitionId &&
      !current.versions?.some(
        (p) =>
          p.definitionId === d.version?.definitionId &&
          p.options?.includes(right),
      )
    )
      return true;
    const policies = [d.version, d.newScope, d.newFields].filter(
      (p): p is Definition => !!p,
    );
    return policies.some(
      (p) =>
        (p.targetId && !current.targets?.some((t) => t.id === p.targetId)) ||
        p.actions?.some(
          (a) => !current.actions?.some((allowed) => allowed.key === a),
        ) ||
        p.readFields?.some((f) => !current.readFields?.includes(f)) ||
        p.writeFields?.some((f) => !current.writeFields?.includes(f)),
    );
  };
  const lost =
    !!draft &&
    !!v &&
    (candidateLost(draft, v) ||
      !v.rights?.includes(right) ||
      (!creating && area !== 'roles' && !account?.options?.includes(right)) ||
      (draft.operation === 'grant.revoke' &&
        !v.grants?.some((g) => g.id === draft.id && g.canRevoke)) ||
      (draft.operation === 'grant.adopt' &&
        !v.grants?.some((g) => g.id === draft.id && g.adoption)) ||
      (draft.operation === 'mandate.revoke' &&
        !v.mandates?.some((m) => m.id === draft.id && m.canRevoke)));
  useEffect(() => {
    if (lost || data.status === 403) {
      close();
      setError('当前管理权限不足');
    }
  }, [lost, data.status]);
  useEffect(() => {
    if (!v || !draft || lost || !conflict.current) return;
    conflict.current = false;
    if (draft.operation === 'version.disable')
      setDraft({ ...draft, impactToken: v.impact?.token });
    if (area === 'accounts' && account)
      setDraft({ ...draft, revision: account.revision });
    if (area === 'grants') {
      const g = v.grants?.find((g) => g.id === draft.id);
      if (draft.id && !g && draft.operation.startsWith('grant.')) close();
      else if (g && ['expired', 'revoked'].includes(g.state)) close();
      else if (g) setDraft({ ...draft, revision: g.revision });
    }
    if (draft.operation === 'version.publish' && draft.version?.definitionId) {
      const related =
        v.versions?.filter(
          (p) =>
            p.definitionId === draft.version?.definitionId &&
            p.kind === draft.version?.kind,
        ) ?? [];
      setDraft({
        ...draft,
        version: {
          ...draft.version,
          number: Math.max(...related.map((p) => p.number ?? 0)),
        },
      });
    }
  }, [v]);
  useEffect(() => {
    if (
      draft?.operation === 'version.disable' &&
      !draft.impactToken &&
      v?.impact?.token
    )
      setDraft({ ...draft, impactToken: v.impact.token });
  }, [draft?.operation, draft?.impactToken, v?.impact?.token]);
  const title = creating
    ? kind === 'account'
      ? '开户'
      : '发布' + label(kind)
    : account?.displayName
      ? account.displayName + '（' + account.login + '）'
      : (account?.login ?? def?.name ?? areaNames[area]);
  usePageTitle(title);
  const button = (text: string, cmd: Command) => (
    <Button
      key={text}
      id={cmd.operation + '-' + (cmd.id ?? id)}
      onClick={() => open(cmd)}
    >
      {text}
    </Button>
  );
  const accountCmd = (op: string) => ({
    operation: 'account.' + op,
    subject: id,
    revision: account?.revision,
  });
  const choices = account?.options ?? [];
  const actions = !draft && v && (
    <>
      {area === 'accounts' && (
        <>
          {choices.includes('account') && (
            <>
              {button(
                account?.state === 'active' ? '停用账号' : '启用账号',
                accountCmd(account?.state === 'active' ? 'disable' : 'enable'),
              )}
              {button('变更管理归属', accountCmd('move'))}
            </>
          )}
          {choices.includes('credential') &&
            button('重置凭据', accountCmd('reset'))}
        </>
      )}
      {area === 'roles' &&
        def?.options?.includes(def.kind === 'role' ? 'role' : 'policy') && (
          <>
            {button('发布新版本', editVersion(def))}
            {!def.disabled &&
              button('停用版本', {
                operation: 'version.disable',
                version: { kind: def.kind, name: def.name },
                id,
                revision: def.number,
              })}
          </>
        )}
      {area === 'grants' &&
        v.panel === 'grants' &&
        choices.includes('assign') &&
        button('授予权限', {
          operation: 'grant.issue',
          subject: id,
          source: 'duty',
        })}
      {area === 'grants' &&
        v.panel === 'mandates' &&
        choices.includes('delegate') &&
        button('委派管理职责', {
          operation: 'mandate.issue',
          subject: id,
          limit: {
            rights: [],
            owner: '',
            role: '',
            scope: '',
            fields: '',
            maxProxyDays: v?.suggestedProxyDays ?? 0,
          },
        })}
    </>
  );
  return (
    <Box className="standard-page object-page">
      <RecordControl
        identity={
          <RecordBackButton
            component={Link}
            to={backTo}
            label={
              fromAccount
                ? '返回账号资料'
                : fromVersion
                  ? '返回版本引用'
                  : '返回列表'
            }
          />
        }
        actions={
          draft && v && !lost ? (
            <Stack direction="row" spacing={1}>
              <Button
                type="submit"
                form="security-command-form"
                variant="contained"
                disabled={
                  flow.busy ||
                  (draft.operation === 'version.disable' &&
                    (!draft.impactToken || !v.impact?.token))
                }
              >
                {(
                  {
                    'grant.revoke': '撤销此份授予',
                    'grant.issue': '授予权限',
                    'grant.adopt': '采用新版本',
                    'mandate.issue': '委派管理职责',
                    'mandate.revoke': '撤销此份职责',
                    'version.disable': '停用此版本',
                  } as Record<string, string>
                )[draft.operation] ?? '保存'}
              </Button>
              <Button
                onClick={() =>
                  leave.request(() => {
                    focusAfter.current = origin.current;
                    close();
                    if (creating) navigate(backTo);
                  })
                }
              >
                取消
              </Button>
            </Stack>
          ) : (
            actions
          )
        }
        navigation={null}
      />
      <div
        className="page-scroll"
        data-page-scroll
        tabIndex={0}
        role="region"
        aria-label={areaNames[area] + '详情'}
      >
        {!v && !data.error && <p role="status">正在读取资料…</p>}
        {data.error && (
          <Alert
            severity="error"
            action={<Button onClick={data.refresh}>重试</Button>}
          >
            {data.error}
          </Alert>
        )}
        {error && <Alert severity="error">{error}</Alert>}
        {notice && (
          <Alert severity="success">
            {notice}
            {area === 'grants' && resultTarget && (
              <Button
                size="small"
                onClick={() =>
                  change({
                    tab: 'events',
                    eventFor: resultTarget,
                    grant: '',
                    gp: '',
                    inspect: '',
                  })
                }
              >
                查看这次变更
              </Button>
            )}
          </Alert>
        )}
        {flow.pending.map((p) => (
          <Button key={p.submissionId} onClick={() => void flow.recover(p)}>
            查回本次提交
          </Button>
        ))}
        {v && (
          <ObjectDetailSheet
            identity={
              <ObjectDetailIdentity
                icon={<SecurityOutlined />}
                title={title}
                summary={
                  account ? (
                    <>
                      管理归属 {targetName(v, account.owner)} · 账号状态{' '}
                      {label(account.state)}
                      {area === 'grants' &&
                        'builtinSecurityAdmin' in account &&
                        account.builtinSecurityAdmin === true && (
                          <div>
                            内置管理员 ·{' '}
                            <strong>全部安全管理权限 · 长期</strong>
                          </div>
                        )}
                    </>
                  ) : def ? (
                    versionName(def)
                  ) : undefined
                }
              />
            }
            tabs={
              area === 'grants' && !draft && <GrantTabs v={v} change={change} />
            }
          >
            {draft && !lost ? (
              <ObjectForm>
                <form
                  id="security-command-form"
                  ref={(n) => {
                    formStart.current = n;
                  }}
                  tabIndex={-1}
                  onSubmit={(e) => {
                    e.preventDefault();
                    if (
                      !flow.busy &&
                      (draft.operation !== 'version.disable' || v.impact?.token)
                    )
                      void flow.submit(draft);
                  }}
                >
                  <ObjectDetailSection title={label(draft.operation)}>
                    {draft.operation === 'version.disable' && (
                      <VersionReferences
                        v={v}
                        params={params}
                        change={change}
                        returnTo={id}
                        confirm
                      />
                    )}
                    {draft.operation === 'grant.revoke' &&
                      v.grants
                        ?.filter((g) => g.id === draft.id)
                        .map((g) => (
                          <GrantFacts key={g.id} v={v} g={g} compact />
                        ))}
                    {draft.operation === 'mandate.revoke' &&
                      v.mandates
                        ?.filter((m) => m.id === draft.id)
                        .map((m) => <MandateFacts key={m.id} v={v} m={m} />)}
                    <ManagementFields
                      d={draft}
                      v={v}
                      set={(d) => {
                        setDraft(d);
                        setDirty(true);
                      }}
                    />
                  </ObjectDetailSection>
                </form>
              </ObjectForm>
            ) : (
              <>
                {area === 'accounts' && account && (
                  <ObjectDetailSection title="账号资料">
                    <ObjectDetailField label="显示名">
                      {account.displayName || '未填写'}
                    </ObjectDetailField>
                    <ObjectDetailField label="管理归属">
                      {targetName(v, account.owner)}
                    </ObjectDetailField>
                    {account.grantReadable && (
                      <Button
                        component={Link}
                        id="account-grants"
                        to={
                          accessPath +
                          '/grants/' +
                          id +
                          '?' +
                          new URLSearchParams({
                            fromAccount: '1',
                            returnQuery: params.toString(),
                          })
                        }
                      >
                        查看账号授权
                      </Button>
                    )}
                  </ObjectDetailSection>
                )}
                {area === 'roles' && def && (
                  <ObjectDetailSection title={label(def.kind) + '资料'}>
                    <ObjectDetailField label="版本状态">
                      {def.disabled ? '已停用' : '已发布'}
                    </ObjectDetailField>
                    {def.kind === 'role' && (
                      <>
                        <ObjectDetailField label="业务动作">
                          {def.actions
                            ?.map(
                              (a) =>
                                v.actions?.find((x) => x.key === a)?.name ?? a,
                            )
                            .join('、')}
                        </ObjectDetailField>
                        <ObjectDetailField label="范围策略">
                          {version(v, 'scope', def.scope)?.name}
                        </ObjectDetailField>
                        <ObjectDetailField label="字段策略">
                          {version(v, 'fields', def.fields)?.name}
                        </ObjectDetailField>
                      </>
                    )}
                    {def.kind === 'scope' && (
                      <ObjectDetailField label="目标">
                        {targetName(v, def.targetId ?? '')}
                      </ObjectDetailField>
                    )}
                    {def.kind === 'fields' && (
                      <>
                        <ObjectDetailField label="可读字段">
                          {def.readFields?.map(label).join('、') || '无'}
                        </ObjectDetailField>
                        <ObjectDetailField label="可写字段">
                          {def.writeFields?.map(label).join('、') || '无'}
                        </ObjectDetailField>
                      </>
                    )}
                  </ObjectDetailSection>
                )}
                {area === 'roles' &&
                  def &&
                  v.rights?.some((r) => r === 'assign' || r === 'delegate') && (
                    <Button
                      id="version-refs"
                      onClick={() =>
                        change({
                          tab: panel === 'references' ? '' : 'references',
                          gp: '',
                          ref: '',
                        })
                      }
                    >
                      {panel === 'references'
                        ? '收起版本引用'
                        : '查看引用此版本的授予'}
                    </Button>
                  )}
                {area === 'roles' && panel === 'references' && (
                  <VersionReferences
                    v={v}
                    params={params}
                    change={change}
                    returnTo={id}
                  />
                )}
                {area === 'grants' && v.panel && (
                  <ObjectTabPanel
                    prefix="grant"
                    value={v.panel}
                    selected={v.panel}
                  >
                    <h2
                      className="object-section-title grant-panel-heading"
                      id="grant-panel-heading"
                      tabIndex={-1}
                    >
                      {
                        (
                          {
                            grants: '业务授予',
                            mandates: '管理职责',
                            events: '管理变更',
                          } as Record<string, string>
                        )[v.panel]
                      }
                    </h2>
                    {v.panel === 'grants' && (
                      <BusinessGrants
                        v={v}
                        params={params}
                        change={change}
                        open={open}
                      />
                    )}
                    {v.panel === 'mandates' && (
                      <>
                        {params.get('eventOrigin') === '1' && (
                          <Button
                            onClick={() =>
                              change({
                                tab: 'events',
                                grant: '',
                                eventOrigin: '',
                                gp: params.get('eventPage') ?? '',
                              })
                            }
                          >
                            返回管理变更
                          </Button>
                        )}
                        <WorkspacePager v={v} params={params} change={change} />
                        {!v.mandates?.length && <p>暂无可见管理职责</p>}
                        {v.mandates?.map((m) => (
                          <section
                            className="access-grant"
                            tabIndex={-1}
                            key={m.id}
                            id={'row-' + m.id}
                          >
                            <MandateFacts v={v} m={m} />
                            {m.canRevoke &&
                              button('撤销管理职责', {
                                operation: 'mandate.revoke',
                                subject: id,
                                id: m.id,
                              })}
                          </section>
                        ))}
                      </>
                    )}
                    {v.panel === 'events' && (
                      <>
                        <WorkspacePager v={v} params={params} change={change} />
                        {params.get('eventFor') && (
                          <Button
                            onClick={() => change({ eventFor: '', gp: '' })}
                          >
                            查看全部变更
                          </Button>
                        )}
                        <History
                          v={v}
                          follow={(target, mandate, event) =>
                            change({
                              eventPage: String(v.page ?? 0),
                              originEvent: event,
                              tab: mandate ? 'mandates' : 'grants',
                              grant: target,
                              inspect: mandate ? '' : '1',
                              eventOrigin: '1',
                              gstate: '',
                              gq: '',
                              gsource: '',
                              gversion: '',
                              gp: '',
                            })
                          }
                        />
                      </>
                    )}
                  </ObjectTabPanel>
                )}
                {!creating && area !== 'grants' && <History v={v} />}
              </>
            )}
          </ObjectDetailSheet>
        )}
      </div>
      <UnsavedChangesDialog {...leave.dialog} />
    </Box>
  );
}
export function SecurityManagement({
  area,
  session,
}: {
  area: Area;
  session: Session;
}) {
  const { id } = useParams();
  return id ? (
    <ManagementDetail key={area + id} id={id} area={area} session={session} />
  ) : (
    <ManagementList key={area} area={area} session={session} />
  );
}
