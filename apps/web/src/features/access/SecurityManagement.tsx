import { useEffect, useMemo, useRef, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Stack,
  Tab,
  Tabs,
  Typography,
} from '@mui/material';
import SecurityOutlined from '@mui/icons-material/SecurityOutlined';
import PersonOutline from '@mui/icons-material/PersonOutline';
import BadgeOutlined from '@mui/icons-material/BadgeOutlined';
import {
  SecurityActionsMenu,
  type SecurityAction,
} from './SecurityActionsMenu';
import { SecurityHistory } from './SecurityHistory';
import { SecurityRecordPager } from './SecurityRecordPager';
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
  StatusChip,
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
  ColumnFilterDialog,
  type FilterRequest,
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
  BusinessGrants,
  GrantFacts,
  GrantTabs,
  MandateFacts,
  Mandates,
  VersionReferences,
  WorkspacePager,
} from './GrantWorkspace';
import './security.css';

type Row = {
  id: string;
  name: string;
  description: string;
  state: string;
  enabled: boolean;
  display?: string;
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
  const [filterRequest, setFilterRequest] = useState<FilterRequest>();
  const versionKind = params.get('versionKind') ?? '';
  const page = Math.max(0, Number(params.get('page')) || 0),
    pageSize = Number(params.get('size')) || 20;
  const query = {
    q: params.get('q') ?? '',
    owner: params.get('owner') ?? '',
    state: params.get('state') ?? '',
    ...(area === 'roles' && versionKind ? { versionKind } : {}),
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
            state: d.disabled ? '已停用' : '可用',
            enabled: !d.disabled,
          }))
        : (v?.accounts ?? []).map((a) => ({
            id: a.id,
            name: a.login,
            display: a.displayName,
            enabled: a.state === 'active',
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
    if (next.toString() === params.toString()) data.refresh();
    else set(next);
  };
  const details = (id: string) => {
    const next = new URLSearchParams(params);
    if (!id.startsWith('new-')) next.set('fromList', '1');
    return base + '/' + id + '?' + next;
  };
  const cols: GridColDef<Row>[] = [
    {
      field: 'name',
      filterable: false,
      headerName: area === 'roles' ? '版本名称' : '登录名',
      flex: 1.25,
      minWidth: 165,
      sortable: false,
      renderCell: (p) => (
        <Link
          className="object-name-link"
          tabIndex={p.tabIndex}
          data-object-link
          to={details(p.row.id)}
          onClick={() => restore.rememberRow(p.row.id)}
        >
          {p.row.name}
        </Link>
      ),
    },
    ...(area === 'roles'
      ? []
      : [
          {
            field: 'display',
            headerName: '显示名',
            flex: 1,
            minWidth: 125,
            sortable: false,
            filterable: false,
            renderCell: ({ row }: { row: Row }) => row.display || '—',
          },
        ]),
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
      filterable: area !== 'roles',
      renderCell: ({ row }) => (
        <StatusChip
          label={row.state}
          color={row.enabled ? 'success' : 'default'}
        />
      ),
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
  const create = (kind: string) => navigate(details('new-' + kind));
  const createKinds =
    area === 'roles'
      ? [
          ...(v?.rights?.includes('role') ? ['role'] : []),
          ...(v?.rights?.includes('policy') ? ['scope', 'fields'] : []),
        ]
      : [];
  const primaryKind = createKinds.includes(versionKind)
    ? versionKind
    : createKinds[0];
  const more: SecurityAction[] = createKinds
    .filter((k) => k !== primaryKind)
    .map((k) => ({ label: '新建' + label(k), run: () => create(k) }));
  const directoryFilters =
    area === 'roles'
      ? [
          {
            field: 'description',
            param: 'versionKind',
            label: '版本类型',
            values: ['role', 'scope', 'fields'].map((value) => ({
              value,
              label: label(value),
            })),
          },
        ]
      : [
          {
            field: 'state',
            param: 'state',
            label: '账号状态',
            values: ['active', 'disabled'].map((value) => ({
              value,
              label: label(value),
            })),
          },
          {
            field: 'description',
            param: 'owner',
            label: '管理归属',
            values: (v?.owners ?? []).map((o) => ({
              value: o.id,
              label: targetName(v!, o.id),
            })),
          },
        ];
  const filterItems = directoryFilters.flatMap((c) => {
    const value = params.get(c.param);
    return value ? [{ field: c.field, operator: 'equals', value }] : [];
  });
  const openFilter = (source: HTMLElement, field?: string) =>
    setFilterRequest({
      source,
      field: field ?? filterItems[0]?.field ?? directoryFilters[0].field,
    });
  more.push({ label: '列筛选…', run: (source) => openFilter(source) });
  const actions =
    area === 'accounts' && v?.rights?.includes('account') ? (
      <Button
        size="small"
        variant="contained"
        onClick={() => create('account')}
      >
        开户
      </Button>
    ) : (
      primaryKind && (
        <Button
          size="small"
          variant="contained"
          onClick={() => create(primaryKind)}
        >
          新建{label(primaryKind)}
        </Button>
      )
    );
  return (
    <div className="standard-page">
      <ColumnFilterEditor.Provider
        value={({ field, source }) => openFilter(source, field)}
      >
        <DataGridPage
          slots={columnFilterSlots}
          filterModel={{ items: filterItems.slice(0, 1) }}
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
                  className="object-name-link"
                  tabIndex={p.tabIndex}
                  data-object-link
                  to={details(p.row.id)}
                  onClick={() => restore.rememberRow(p.row.id)}
                >
                  {p.row.name}
                </Link>
                <div>
                  {p.row.display && p.row.display + ' · '}
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
              more={<SecurityActionsMenu actions={more} />}
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
              {input !== applied && (
                <Typography className="list-query-summary" role="status">
                  输入已修改，点击查询后生效。已应用关键词：{applied || '无'}
                </Typography>
              )}
              {filterItems.map((item) => {
                const c = directoryFilters.find((c) => c.field === item.field)!;
                return (
                  <Button
                    size="small"
                    key={c.field}
                    onClick={(e) => openFilter(e.currentTarget, c.field)}
                  >
                    {c.label} ·{' '}
                    {c.values.find((option) => option.value === item.value)
                      ?.label ?? item.value}
                  </Button>
                );
              })}
              {filterRequest && (
                <ColumnFilterDialog
                  request={filterRequest}
                  columns={directoryFilters.map((c) => ({
                    ...c,
                    operators: [{ value: 'equals', label: '等于' }],
                  }))}
                  applied={{ items: filterItems }}
                  onClose={() => {
                    const source = filterRequest.source;
                    setFilterRequest(undefined);
                    requestAnimationFrame(
                      () => source.isConnected && source.focus(),
                    );
                  }}
                  onApply={(f, field) => {
                    const c = directoryFilters.find((c) => c.field === field)!;
                    update({ [c.param]: f.filterValue, page: '0', q: input });
                  }}
                />
              )}
            </ListControl>
          }
        />
      </ColumnFilterEditor.Provider>
    </div>
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
  if (
    panel === 'events' &&
    !creating &&
    draft?.operation !== 'version.disable'
  ) {
    Object.assign(detailQuery, {
      panel: 'events',
      q: params.get('logValue') ?? '',
      logField: params.get('logField') ?? '',
      page: Number(params.get('gp')) || 0,
      pageSize: Number(params.get('gs')) || 20,
    });
  }
  const key =
    session.instanceId + session.subjectId + area + JSON.stringify(detailQuery);
  const data = useRead(key, () => read(area, detailQuery), 'fresh'),
    v = data.data;
  const account = v?.accounts?.find((a) => a.id === id),
    def = v?.versions?.find((d) => d.id === id);
  const origin = useRef(''),
    focusAfter = useRef('');
  const change = (values: Record<string, string>, focusId?: string) => {
    if (focusId) focusAfter.current = focusId;
    const next = new URLSearchParams(params);
    for (const [k, value] of Object.entries(values)) {
      if (value) next.set(k, value);
      else next.delete(k);
    }
    if (next.toString() === params.toString()) data.refresh();
    else setParams(next);
  };
  useEffect(() => {
    if (!v) return;
    if (draft) {
      formStart.current?.focus({ preventScroll: true });
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
      const target = document.getElementById(requested);
      const fallback = requested.startsWith('security-page-')
        ? 'security-page-count'
        : 'grant-panel-heading';
      (target && !target.matches(':disabled')
        ? target
        : document.getElementById(fallback)
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
    'fromList',
    'fromAccount',
    'fromVersion',
    'returnQuery',
    'via',
    'eventPage',
    'originEvent',
    'logField',
    'logValue',
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
        (r.operation === 'version.publish'
          ? '版本已保存'
          : label(r.operation) + '完成') +
        ' · ' +
        recordedTime(r.at) +
        (r.operation === 'account.create' ? '。开户时业务授予为空。' : '');
      setNotice(message);
      data.refresh();
      const target = r.subject ?? r.target,
        next = new URLSearchParams(params);
      if (r.operation === 'version.publish') {
        for (const k of ['tab', 'ref', 'gp', 'logField', 'logValue'])
          next.delete(k);
      }
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
  const open = (cmd: Command, triggerId?: string) => {
    origin.current =
      triggerId ?? (document.activeElement as HTMLElement)?.id ?? '';
    flow.start();
    saved.current = false;
    setDraft(cmd);
    setDirty(false);
    setError('');
    setNotice('');
    requestAnimationFrame(() =>
      formStart.current?.focus({ preventScroll: true }),
    );
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
      : '新建' + label(kind)
    : account?.displayName
      ? account.displayName + '（' + account.login + '）'
      : (account?.login ?? def?.name ?? areaNames[area]);
  usePageTitle(title);
  const button = (text: string, cmd: Command, primary = false) => (
    <Button
      key={text}
      size="small"
      variant={primary ? 'contained' : 'text'}
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
  const canEditVersion = def?.options?.includes(
    def.kind === 'role' ? 'role' : 'policy',
  );
  const showReferences =
    area === 'roles' &&
    v?.rights?.some((r) => r === 'assign' || r === 'delegate');
  const detailTab =
    panel === 'events'
      ? 'events'
      : panel === 'references' && showReferences
        ? 'references'
        : 'basic';
  const more: SecurityAction[] = [];
  const secondary = (text: string, cmd: Command) =>
    more.push({ label: text, run: (source) => open(cmd, source.id) });
  if (area === 'accounts' && choices.includes('account')) {
    secondary('变更管理归属', accountCmd('move'));
    secondary(
      account?.state === 'active' ? '停用账号' : '启用账号',
      accountCmd(account?.state === 'active' ? 'disable' : 'enable'),
    );
  }
  if (area === 'roles' && def && canEditVersion && !def.disabled) {
    secondary('停用版本', {
      operation: 'version.disable',
      version: { kind: def.kind, name: def.name },
      id,
      revision: def.number,
    });
  }
  const actions = !draft && v && (
    <>
      {area === 'accounts' &&
        choices.includes('credential') &&
        button('重置凭据', accountCmd('reset'), true)}
      {area === 'roles' &&
        def &&
        canEditVersion &&
        button('新增版本', editVersion(def), true)}
      {area === 'grants' &&
        v.panel === 'grants' &&
        choices.includes('assign') &&
        button(
          '授予权限',
          {
            operation: 'grant.issue',
            subject: id,
            source: 'duty',
          },
          true,
        )}
      {area === 'grants' &&
        v.panel === 'mandates' &&
        choices.includes('delegate') &&
        button(
          '委派管理职责',
          {
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
          },
          true,
        )}
    </>
  );
  return (
    <Box className="standard-page object-page">
      <RecordControl
        identity={
          <>
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
            {!creating && (
              <Typography className="record-control-name">
                {account?.login ?? (def && versionName(def))}
              </Typography>
            )}
            {!draft && (
              <SecurityActionsMenu
                actions={more}
                triggerId="security-record-more"
              />
            )}
          </>
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
        navigation={
          !creating &&
          !!v &&
          !fromAccount &&
          !fromVersion &&
          params.get('fromList') === '1' && (
            <SecurityRecordPager
              area={area}
              id={id}
              params={back}
              session={session}
              disabled={!!draft}
            />
          )
        }
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
            action={
              <>
                <Button onClick={data.refresh}>重试</Button>
                {panel === 'events' && params.has('logField') && (
                  <Button
                    onClick={() =>
                      change({ logField: '', logValue: '', gp: '' })
                    }
                  >
                    清除日志筛选
                  </Button>
                )}
              </>
            }
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
                    logField: '',
                    logValue: '',
                    originEvent: '',
                    grant: '',
                    gp: '',
                    inspect: '',
                  })
                }
              >
                查看该份变更
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
                icon={
                  area === 'accounts' ? (
                    <PersonOutline />
                  ) : area === 'roles' ? (
                    <BadgeOutlined />
                  ) : (
                    <SecurityOutlined />
                  )
                }
                related={
                  area === 'accounts' && account?.grantReadable && !draft ? (
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
                  ) : undefined
                }
                title={title}
                summary={
                  account ? (
                    <>
                      {area === 'grants' && (
                        <>管理归属 {targetName(v, account.owner)}</>
                      )}
                      <StatusChip
                        label={label(account.state)}
                        color={
                          account.state === 'active' ? 'success' : 'default'
                        }
                      />
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
                    <>
                      {label(def.kind)} · V{def.number}{' '}
                      <StatusChip
                        label={def.disabled ? '已停用' : '可用'}
                        color={def.disabled ? 'default' : 'success'}
                      />
                    </>
                  ) : undefined
                }
              />
            }
            tabs={
              !draft &&
              !creating &&
              (area === 'grants' ? (
                <GrantTabs v={v} change={change} />
              ) : (
                <Tabs
                  aria-label={areaNames[area] + '栏目'}
                  value={detailTab}
                  variant="scrollable"
                  scrollButtons="auto"
                  onChange={(_, tab: string) =>
                    change(
                      {
                        tab: tab === 'basic' ? '' : tab,
                        gp: '',
                        ref: '',
                        logField: '',
                        logValue: '',
                      },
                      'security-' + tab + '-tab',
                    )
                  }
                >
                  {[
                    ['basic', '基本资料'],
                    ...(showReferences ? [['references', '版本引用']] : []),
                    ['events', '变更日志'],
                  ].map(([tab, name]) => (
                    <Tab
                      key={tab}
                      id={'security-' + tab + '-tab'}
                      aria-controls={'security-' + tab + '-panel'}
                      value={tab}
                      label={name}
                    />
                  ))}
                </Tabs>
              ))
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
                  <ObjectDetailSection
                    title={creating ? undefined : label(draft.operation)}
                  >
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
                    {draft.operation === 'version.publish' && !creating && (
                      <Typography variant="body2" color="text.secondary">
                        保存后可为账号选择新版本，现有授予继续使用原版本。
                      </Typography>
                    )}
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
                {area !== 'grants' && (
                  <ObjectTabPanel
                    prefix="security"
                    value={detailTab}
                    selected={detailTab}
                  >
                    {detailTab === 'basic' &&
                      area === 'accounts' &&
                      account && (
                        <ObjectDetailSection title="账号资料">
                          <div className="object-detail-columns">
                            <div>
                              <ObjectDetailField label="登录名">
                                {account.login}
                              </ObjectDetailField>
                              <ObjectDetailField label="显示名">
                                {account.displayName || '未填写'}
                              </ObjectDetailField>
                            </div>
                            <div>
                              <ObjectDetailField label="管理归属">
                                {targetName(v, account.owner)}
                              </ObjectDetailField>
                            </div>
                          </div>
                        </ObjectDetailSection>
                      )}
                    {detailTab === 'basic' && area === 'roles' && def && (
                      <ObjectDetailSection title={label(def.kind) + '资料'}>
                        {def.kind === 'role' && (
                          <>
                            <div className="object-detail-columns">
                              <div>
                                <ObjectDetailField label="业务动作">
                                  {def.actions
                                    ?.map(
                                      (a) =>
                                        v.actions?.find((x) => x.key === a)
                                          ?.name ?? a,
                                    )
                                    .join('、')}
                                </ObjectDetailField>
                              </div>
                              <div>
                                <ObjectDetailField label="范围策略">
                                  {version(v, 'scope', def.scope) &&
                                    versionName(
                                      version(v, 'scope', def.scope)!,
                                    )}
                                </ObjectDetailField>
                                <ObjectDetailField label="字段策略">
                                  {version(v, 'fields', def.fields) &&
                                    versionName(
                                      version(v, 'fields', def.fields)!,
                                    )}
                                </ObjectDetailField>
                              </div>
                            </div>
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
                    {showReferences && detailTab === 'references' && (
                      <VersionReferences
                        v={v}
                        params={params}
                        change={change}
                        returnTo={id}
                      />
                    )}
                    {detailTab === 'events' && (
                      <SecurityHistory
                        area={area}
                        v={v}
                        def={def}
                        params={params}
                        change={change}
                      />
                    )}
                  </ObjectTabPanel>
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
                            events: '变更日志',
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
                            返回变更日志
                          </Button>
                        )}
                        <WorkspacePager v={v} params={params} change={change} />
                        <Mandates
                          v={v}
                          selected={params.get('grant')}
                          open={open}
                        />
                      </>
                    )}
                    {v.panel === 'events' && (
                      <SecurityHistory
                        area={area}
                        params={params}
                        change={change}
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
                    )}
                  </ObjectTabPanel>
                )}
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
