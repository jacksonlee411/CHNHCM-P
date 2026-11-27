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
  RecordBackButton,
  RecordControl,
  UnsavedChangesDialog,
  recordedTime,
} from '@chnhcm/ui';
import { useRead } from '../personnel/api';
import { usePageTitle, useListReturn } from '../../shared/navigation';
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
import './security.css';

type Row = { id: string; name: string; description: string; state: string };
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
  useEffect(() => setInput(params.get('q') ?? ''), [params]);
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
            name: a.login + (a.displayName ? ' · ' + a.displayName : ''),
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
      headerName: area === 'roles' ? '版本名称' : '登录账号',
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
    { field: 'state', headerName: '状态', width: 105, sortable: false },
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
      <DataGridPage
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
              </Link>
              <div>
                {p.row.description} · {p.row.state}
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
        messages={data.error && <Alert severity="error">{data.error}</Alert>}
        toolbar={
          <ListControl
            title={title}
            action={actions}
            pager={<ListPager />}
            search={
              <SearchControl
                label={area === 'roles' ? '搜索版本名称' : '搜索登录名或显示名'}
                value={input}
                onChange={setInput}
                onSearch={() => update({ q: input, page: '0' })}
              />
            }
          >
            {area !== 'roles' && v && (
              <div className="security-filters">
                <label>
                  状态{' '}
                  <select
                    value={query.state}
                    onChange={(e) =>
                      update({ state: e.target.value, page: '0' })
                    }
                  >
                    <option value="">全部</option>
                    <option value="active">启用</option>
                    <option value="disabled">停用</option>
                  </select>
                </label>
                <label>
                  管理归属{' '}
                  <select
                    value={query.owner}
                    onChange={(e) =>
                      update({ owner: e.target.value, page: '0' })
                    }
                  >
                    <option value="">全部</option>
                    {v.owners?.map((t) => (
                      <option key={t.id} value={t.id}>
                        {targetName(v, t.id)}
                      </option>
                    ))}
                  </select>
                </label>
              </div>
            )}
          </ListControl>
        }
      />
    </div>
  );
}

function History({ v }: { v: View }) {
  return (
    <ObjectDetailSection title="管理变更">
      {v.events?.length ? (
        <ol className="access-history">
          {v.events.map((e) => (
            <li key={e.id}>
              <strong>{label(e.operation)}</strong> · {e.actor} ·{' '}
              <time dateTime={e.at}>{recordedTime(e.at)}</time>
              {e.previousOwner && (
                <div>原管理归属 · {targetName(v, e.previousOwner)}</div>
              )}
              {e.assignedOwner && (
                <div>管理归属 · {targetName(v, e.assignedOwner)}</div>
              )}
              {e.reason && <div>{e.reason}</div>}
              <details>
                <summary>提交身份</summary>
                {e.request}
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
    [params] = useSearchParams(),
    location = useLocation();
  const key = session.instanceId + session.subjectId + area + id;
  const data = useRead(
      key,
      () =>
        read(
          area,
          creating || area === 'roles' ? { object: id } : { subject: id },
        ),
      'fresh',
    ),
    v = data.data;
  const account = v?.accounts?.find((a) => a.id === id),
    def = v?.versions?.find((d) => d.id === id);
  const [draft, setDraft] = useState<Command>(),
    [dirty, setDirty] = useState(false),
    [error, setError] = useState(''),
    [notice, setNotice] = useState<string>(location.state?.notice ?? ''),
    [focus, setFocus] = useState(params.get('grant') ?? '');
  const conflict = useRef(false),
    saved = useRef(false),
    formStart = useRef<HTMLElement>(null);
  const close = () => {
    saved.current = true;
    setDraft(undefined);
    setDirty(false);
  };
  const back = new URLSearchParams(params);
  back.delete('grant');
  if (!creating) back.set('selected', id);
  const backTo = base + '?' + back;
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
        setFocus(r.target);
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
      (!creating && area !== 'roles' && !account?.options?.includes(right)));
  useEffect(() => {
    if (lost || data.status === 403) {
      close();
      setError('当前管理权限不足');
    }
  }, [lost, data.status]);
  useEffect(() => {
    if (!v || !draft || lost || !conflict.current) return;
    conflict.current = false;
    if (area === 'accounts' && account)
      setDraft({ ...draft, revision: account.revision });
    if (area === 'grants') {
      const g = v.grants?.find((g) => g.id === draft.id);
      if (g && ['expired', 'revoked'].includes(g.state)) close();
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
  const title = creating
    ? kind === 'account'
      ? '开户'
      : '发布' + label(kind)
    : (account?.login ?? def?.name ?? areaNames[area]);
  usePageTitle(title);
  const button = (text: string, cmd: Command) => (
    <Button key={text} onClick={() => open(cmd)}>
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
        choices.includes('assign') &&
        button('新增授予', {
          operation: 'grant.issue',
          subject: id,
          source: 'duty',
        })}
      {area === 'grants' &&
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
        identity={<RecordBackButton component={Link} to={backTo} />}
        actions={
          draft && v && !lost ? (
            <Stack direction="row" spacing={1}>
              <Button
                type="submit"
                form="security-command-form"
                variant="contained"
                disabled={flow.busy}
              >
                保存
              </Button>
              <Button
                onClick={() =>
                  leave.request(() => {
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
        {data.error && <Alert severity="error">{data.error}</Alert>}
        {error && <Alert severity="error">{error}</Alert>}
        {notice && <Alert severity="success">{notice}</Alert>}
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
                  account
                    ? targetName(v, account.owner) +
                      ' · ' +
                      label(account.state)
                    : def
                      ? versionName(def)
                      : undefined
                }
              />
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
                    if (!flow.busy) void flow.submit(draft);
                  }}
                >
                  <ObjectDetailSection title={label(draft.operation)}>
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
                    {v.rights?.some(
                      (r) => r === 'assign' || r === 'delegate',
                    ) && (
                      <Button
                        component={Link}
                        to={accessPath + '/grants/' + id}
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
                {area === 'grants' && (
                  <>
                    <ObjectDetailSection title="业务授予">
                      {!v.grants?.length && (
                        <Typography>暂无可见业务授予</Typography>
                      )}
                      {v.grants?.map((g) => {
                        const r = version(v, 'role', g.role),
                          s = version(v, 'scope', g.scope),
                          f = version(v, 'fields', g.fields);
                        return (
                          <div
                            className="access-grant"
                            key={g.id}
                            data-selected={focus === g.id}
                            tabIndex={-1}
                            ref={(n) => {
                              if (n && focus === g.id && !draft)
                                n.focus({ preventScroll: true });
                            }}
                          >
                            <strong>{r ? versionName(r) : g.role}</strong> ·{' '}
                            {label(g.source)} ·{' '}
                            {g.state === 'active' ? '生效中' : label(g.state)}
                            <div>
                              {targetName(v, s?.targetId ?? '')} ·{' '}
                              {s ? versionName(s) : ''} ·{' '}
                              {f ? versionName(f) : ''} · {label(g.purpose)}
                            </div>
                            <div>
                              {g.from ? recordedTime(g.from) : '原始起点未登记'}{' '}
                              至 {g.until ? recordedTime(g.until) : '长期'}
                            </div>
                            <details>
                              <summary>授予身份及字段</summary>
                              {g.id} · 可读{' '}
                              {f?.readFields?.map(label).join('、') || '无'} ·
                              可写{' '}
                              {f?.writeFields?.map(label).join('、') || '无'}
                            </details>
                            {choices.includes('assign') &&
                              !['revoked', 'expired'].includes(g.state) && (
                                <>
                                  {r?.options?.includes('assign') &&
                                    button('采用新版本', {
                                      operation: 'grant.adopt',
                                      id: g.id,
                                      subject: id,
                                      revision: g.revision,
                                      role: g.role,
                                      scope: g.scope,
                                      fields: g.fields,
                                    })}
                                  {button('撤销此份授予', {
                                    operation: 'grant.revoke',
                                    id: g.id,
                                    subject: id,
                                    revision: g.revision,
                                  })}
                                </>
                              )}
                          </div>
                        );
                      })}
                    </ObjectDetailSection>
                    {(choices.includes('delegate') || v.mandates?.length) && (
                      <ObjectDetailSection title="管理职责">
                        {v.mandates?.map((m) => (
                          <div key={m.id} className="access-grant">
                            {m.limit.rights.map(label).join('、')} ·{' '}
                            {targetName(v, m.limit.owner)} ·{' '}
                            {m.revoked
                              ? '已撤销'
                              : m.limit.until
                                ? recordedTime(m.limit.until)
                                : '长期'}{' '}
                            {!m.revoked &&
                              button('撤销管理职责', {
                                operation: 'mandate.revoke',
                                subject: id,
                                id: m.id,
                              })}
                          </div>
                        ))}
                      </ObjectDetailSection>
                    )}
                  </>
                )}
                {!creating && <History v={v} />}
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
