import { Fragment, useEffect, useState } from 'react';
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Tab,
  Tabs,
  TablePagination,
} from '@mui/material';
import { Link } from 'react-router-dom';
import {
  ObjectDetailField,
  ObjectDetailSection,
  SearchControl,
  StatusChip,
  recordedTime,
} from '@chnhcm/ui';
import { useRead } from '../personnel/api';
import {
  accessPath,
  read,
  label,
  targetName,
  managementRightName,
  version,
  versionName,
  type Command,
  type Grant,
  type View,
} from './management';

export type Change = (values: Record<string, string>, focusId?: string) => void;
type Open = (command: Command) => void;
const panels: Record<string, string> = {
  grants: '业务授予',
  mandates: '管理职责',
  events: '变更日志',
};
export const grantState = (state: string) =>
  state === 'active'
    ? '生效中'
    : state === 'disabled'
      ? '引用版本已停用'
      : label(state);
const term = (g: Grant) =>
  (g.from ? recordedTime(g.from) : '原始起点未登记') +
  ' 至 ' +
  (g.until ? recordedTime(g.until) : '长期');
export function GrantTabs({ v, change }: { v: View; change: Change }) {
  if ((v.panels?.length ?? 0) < 2) return null;
  return (
    <Tabs
      aria-label="账号授权栏目"
      value={v.panel}
      onChange={(_, tab: string) =>
        change(
          {
            tab,
            gp: '',
            logField: '',
            logValue: '',
            grant: '',
            inspect: '',
            eventFor: '',
            eventOrigin: '',
            originEvent: '',
            eventPage: '',
          },
          'grant-' + tab + '-tab',
        )
      }
      variant="scrollable"
      scrollButtons="auto"
    >
      {v.panels?.map((tab) => (
        <Tab
          key={tab}
          value={tab}
          id={'grant-' + tab + '-tab'}
          aria-controls={'grant-' + tab + '-panel'}
          label={panels[tab]}
        />
      ))}
    </Tabs>
  );
}
export function GrantFacts({
  v,
  g,
  compact = false,
}: {
  v: View;
  g: Grant;
  compact?: boolean;
}) {
  const r = version(v, 'role', g.role),
    s = version(v, 'scope', g.scope),
    f = version(v, 'fields', g.fields);
  return (
    <div className="grant-facts">
      <ObjectDetailField label="角色版本">
        {r && versionName(r)}
      </ObjectDetailField>
      <ObjectDetailField label="业务范围">
        {targetName(v, s?.targetId ?? '')}
      </ObjectDetailField>
      <ObjectDetailField label="授予来源">{label(g.source)}</ObjectDetailField>
      <ObjectDetailField label="有效期（北京时间）">
        {term(g)}
      </ObjectDetailField>
      <ObjectDetailField label="授予状态">
        <StatusChip
          label={grantState(g.state)}
          color={g.state === 'active' ? 'success' : 'default'}
        />
      </ObjectDetailField>
      <ObjectDetailField label="授予编号">{g.id}</ObjectDetailField>
      {!compact && (
        <>
          <ObjectDetailField label="范围及字段版本">
            {[s, f].map((d) => d && versionName(d)).join(' / ')}
          </ObjectDetailField>
          <ObjectDetailField label="用途">{label(g.purpose)}</ObjectDetailField>
          <ObjectDetailField label="业务动作">
            {r?.actionNames?.join('、') || '原动作名称未登记'}
          </ObjectDetailField>
          <details>
            <summary>动作来源</summary>
            {r?.actions?.join('、')}
          </details>
          <ObjectDetailField label="可读字段">
            {f?.readFields?.map(label).join('、') || '无'}
          </ObjectDetailField>
          <ObjectDetailField label="可写字段">
            {f?.writeFields?.map(label).join('、') || '无'}
          </ObjectDetailField>
        </>
      )}
    </div>
  );
}
export function WorkspacePager({
  v,
  params,
  change,
  unit = '条数',
}: {
  v: View;
  params: URLSearchParams;
  change: Change;
  unit?: string;
}) {
  return (
    <TablePagination
      className="grant-pager"
      component="div"
      count={v.total}
      page={v.page ?? 0}
      rowsPerPage={Number(params.get('gs')) || 20}
      rowsPerPageOptions={[20, 50, 100]}
      slotProps={{
        select: { SelectDisplayProps: { id: 'security-page-size' } },
        displayedRows: {
          id: 'security-page-count',
          tabIndex: -1,
          role: 'status',
        },
        actions: {
          previousButton: { id: 'security-page-prev' },
          nextButton: { id: 'security-page-next' },
        },
      }}
      labelRowsPerPage={'每页' + unit}
      labelDisplayedRows={({ from, to, count }) =>
        from + '–' + to + ' / ' + count
      }
      getItemAriaLabel={(type) =>
        ({
          next: '下一页',
          previous: '上一页',
          first: '第一页',
          last: '最后一页',
        })[type]
      }
      onPageChange={(event, page) =>
        change({ gp: String(page), grant: '' }, event?.currentTarget.id)
      }
      onRowsPerPageChange={(e) =>
        change({ gs: e.target.value, gp: '', grant: '' }, 'security-page-size')
      }
    />
  );
}
export function GrantQuery({
  params,
  change,
  v,
}: {
  params: URLSearchParams;
  change: Change;
  v: View;
}) {
  const applied = params.get('gq') ?? '',
    state = params.get('gstate') ?? '',
    source = params.get('gsource') ?? '',
    ver = params.get('gversion') ?? '';
  const [input, setInput] = useState(applied),
    [filter, setFilter] = useState(false),
    [s, S] = useState(''),
    [o, O] = useState(''),
    [d, D] = useState('');
  useEffect(() => setInput(applied), [applied]);
  const subject = v.accounts?.[0]?.id;
  const options = useRead(
    filter && subject ? 'grant-filters-' + subject : undefined,
    () =>
      read('grants', {
        subject: subject!,
        choices: 'filters',
        panel: 'grants',
      }),
    'fresh',
  );
  const open = () => {
    S(state);
    O(source);
    D(ver);
    setFilter(true);
  };
  return (
    <>
      <div className="grant-query" id="grant-query" role="search" aria-label="业务授予查询" tabIndex={-1}>
        <SearchControl
          label="搜索角色或业务范围"
          value={input}
          onChange={setInput}
          onSearch={() => change({ gq: input, gp: '', grant: '' }, 'grant-query')}
        />
        <Button onClick={open}>筛选授予</Button>
        <WorkspacePager v={v} params={params} change={change} />
      </div>
      {(state || source || ver) && (
        <Button size="small" onClick={open}>
          {[
            state && grantState(state),
            source && label(source),
            ver && '指定版本',
          ]
            .filter(Boolean)
            .join(' · ')}
        </Button>
      )}
      <Dialog
        open={filter}
        onClose={() => setFilter(false)}
        fullWidth
        maxWidth="xs"
        aria-labelledby="grant-filter-title"
      >
        <form
          onSubmit={(e) => {
            e.preventDefault();
            change({
              gq: input,
              gstate: s,
              gsource: o,
              gversion: d,
              grant: '',
              gp: '',
            });
            setFilter(false);
          }}
        >
          <DialogTitle id="grant-filter-title">筛选授予</DialogTitle>
          <DialogContent className="grant-filter-dialog">
            <label>
              授予状态{' '}
              <select autoFocus value={s} onChange={(e) => S(e.target.value)}>
                <option value="">全部</option>
                {['active', 'future', 'expired', 'revoked', 'disabled'].map(
                  (x) => (
                    <option key={x} value={x}>
                      {grantState(x)}
                    </option>
                  ),
                )}
              </select>
            </label>
            <label>
              授予来源{' '}
              <select value={o} onChange={(e) => O(e.target.value)}>
                <option value="">全部</option>
                {['duty', 'proxy', 'migration', 'development'].map((x) => (
                  <option key={x} value={x}>
                    {label(x)}
                  </option>
                ))}
              </select>
            </label>
            <label>
              角色或策略版本{' '}
              {options.error && <span role="alert">{options.error}</span>}
              <select value={d} onChange={(e) => D(e.target.value)}>
                <option value="">全部</option>
                {options.data?.versions?.map((x) => (
                  <option key={x.kind + x.id} value={x.id}>
                    {label(x.kind)} · {versionName(x)}
                  </option>
                ))}
              </select>
            </label>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setFilter(false)}>取消</Button>
            <Button type="submit" variant="contained">
              应用
            </Button>
          </DialogActions>
        </form>
      </Dialog>
    </>
  );
}
export function BusinessGrants({
  v,
  params,
  change,
  open,
}: {
  v: View;
  params: URLSearchParams;
  change: Change;
  open: Open;
}) {
  const selected = params.get('grant'),
    inspect = params.get('inspect') === '1',
    rows = v.grants ?? [];
  const operations = (g: Grant) => (
    <>
      {g.adoption && (
        <Button
          size="small"
          id={'adopt-' + g.id}
          onClick={() =>
            open({
              operation: 'grant.adopt',
              id: g.id,
              subject: g.subject,
              revision: g.revision,
              ...g.adoption,
            })
          }
        >
          采用新版本
        </Button>
      )}
      {g.canRevoke && (
        <Button
          size="small"
          id={'revoke-' + g.id}
          onClick={() =>
            open({
              operation: 'grant.revoke',
              id: g.id,
              subject: g.subject,
              revision: g.revision,
            })
          }
        >
          撤销此份授予
        </Button>
      )}
    </>
  );
  const href = (id: string) => {
    const next = new URLSearchParams(params);
    next.set('grant', id);
    next.set('inspect', '1');
    return '?' + next;
  };
  const canAct = rows.some((g) => g.canRevoke || g.adoption);
  const g = rows.find((row) => row.id === selected);
  if (inspect && g)
    return (
      <ObjectDetailSection>
        <h2
          tabIndex={-1}
          id="grant-detail-heading"
          className="object-section-title"
        >
          授予明细
        </h2>
        <Button
          onClick={() =>
            change(
              params.get('eventOrigin') === '1'
                ? {
                    tab: 'events',
                    grant: '',
                    inspect: '',
                    eventOrigin: '',
                    gp: params.get('eventPage') ?? '',
                  }
                : { inspect: '' },
            )
          }
        >
          {params.get('eventOrigin') === '1' ? '返回变更日志' : '返回业务授予'}
        </Button>
        {operations(g)}
        <GrantFacts v={v} g={g} />
        <Button
          onClick={() =>
            change({
              tab: 'events',
              eventFor: g.id,
              logField: '',
              logValue: '',
              originEvent: '',
              grant: '',
              inspect: '',
              gp: '',
            })
          }
        >
          查看该份变更
        </Button>
      </ObjectDetailSection>
    );
  return (
    <>
      <GrantQuery params={params} change={change} v={v} />
      {rows.length ? (
        <table className="grant-table business-grants">
          <colgroup>
            <col />
            <col style={{ width: '20%' }} />
            <col style={{ width: '10%' }} />
            <col style={{ width: '24%' }} />
            <col style={{ width: '12%' }} />
            {canAct && <col style={{ width: '13%' }} />}
          </colgroup>
          <thead>
            <tr>
              {[
                '角色版本',
                '业务范围',
                '来源',
                '有效期（北京时间）',
                '授予状态',
                ...(canAct ? ['操作'] : []),
              ].map((h) => (
                <th scope="col" key={h}>
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((g) => {
              const r = version(v, 'role', g.role),
                s = version(v, 'scope', g.scope);
              return (
                <tr
                  key={g.id}
                  className="access-grant"
                  id={'row-' + g.id}
                  tabIndex={-1}
                  data-selected={g.id === selected}
                >
                  <td data-title="角色版本">
                    <Link
                      className="object-name-link"
                      id={'detail-' + g.id}
                      to={href(g.id)}
                    >
                      {r ? versionName(r) : '原角色'}
                    </Link>
                    {g.distinguish && <small>授予编号 {g.id}</small>}
                  </td>
                  <td data-title="业务范围">
                    {targetName(v, s?.targetId ?? '')}
                  </td>
                  <td data-title="来源">{label(g.source)}</td>
                  <td data-title="有效期">{term(g)}</td>
                  <td data-title="授予状态">
                    <StatusChip
                      label={grantState(g.state)}
                      color={
                        g.state === 'active'
                          ? 'success'
                          : g.state === 'future'
                            ? 'info'
                            : 'default'
                      }
                    />
                  </td>
                  {canAct && <td data-title="操作">{operations(g)}</td>}
                </tr>
              );
            })}
          </tbody>
        </table>
      ) : (
        <p>没有符合条件的业务授予</p>
      )}
    </>
  );
}
export function MandateFacts({
  v,
  m,
}: {
  v: View;
  m: NonNullable<View['mandates']>[number];
}) {
  const l = m.limit;
  return (
    <div className="grant-facts">
      <ObjectDetailField label="管理职责">
        {l.rights.map(managementRightName).join('、')}
      </ObjectDetailField>
      <ObjectDetailField label="管理归属">
        {targetName(v, l.owner)}
      </ObjectDetailField>
      {(['role', 'scope', 'fields'] as const).map((k) => {
        const d = version(v, k, l[k]);
        return (
          d && (
            <ObjectDetailField key={k} label={label(k)}>
              {versionName(d)}
            </ObjectDetailField>
          )
        );
      })}
      <ObjectDetailField label="职责到期">
        {l.until ? recordedTime(l.until) : '长期'}
      </ObjectDetailField>
      <ObjectDetailField label="代理期限上限">
        {l.maxProxyDays ? l.maxProxyDays + ' 天' : '无限制'}
      </ObjectDetailField>
      <ObjectDetailField label="委派编号">{m.id}</ObjectDetailField>
    </div>
  );
}
export function Mandates({
  v,
  selected,
  open,
}: {
  v: View;
  selected: string | null;
  open: Open;
}) {
  const [expanded, setExpanded] = useState(selected);
  useEffect(() => setExpanded(selected), [selected]);
  if (!v.mandates?.length) return <p role="status">暂无可见管理职责</p>;
  return (
    <table className="grant-table">
      <thead>
        <tr>
          {['管理职责', '管理归属', '职责到期', '状态', '操作'].map((col) => (
            <th scope="col" key={col}>
              {col}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {v.mandates.map((m) => (
          <Fragment key={m.id}>
            <tr
              id={'row-' + m.id}
              className="access-grant"
              tabIndex={-1}
              data-selected={selected === m.id}
            >
              <td data-title="管理职责">
                {m.limit.rights.map(managementRightName).join('、')}
              </td>
              <td data-title="管理归属">{targetName(v, m.limit.owner)}</td>
              <td data-title="职责到期">
                {m.limit.until ? recordedTime(m.limit.until) : '长期'}
              </td>
              <td data-title="状态">
                <StatusChip
                  label={m.state ? grantState(m.state) : '未登记'}
                  color={m.state === 'active' ? 'success' : 'default'}
                />
              </td>
              <td data-title="操作">
                <Button
                  size="small"
                  aria-expanded={expanded === m.id}
                  aria-controls={'mandate-' + m.id}
                  onClick={() => setExpanded(expanded === m.id ? null : m.id)}
                >
                  职责明细
                </Button>
                {m.canRevoke && (
                  <Button
                    size="small"
                    id={'mandate-revoke-' + m.id}
                    onClick={() =>
                      open({
                        operation: 'mandate.revoke',
                        subject: v.accounts?.[0]?.id,
                        id: m.id,
                      })
                    }
                  >
                    撤销管理职责
                  </Button>
                )}
              </td>
            </tr>
            {expanded === m.id && (
              <tr>
                <td
                  colSpan={5}
                  className="security-mandate-facts"
                  id={'mandate-' + m.id}
                >
                  <MandateFacts v={v} m={m} />
                </td>
              </tr>
            )}
          </Fragment>
        ))}
      </tbody>
    </table>
  );
}
export function VersionReferences({
  v,
  params,
  change,
  returnTo,
  confirm = false,
}: {
  v: View;
  params: URLSearchParams;
  change: Change;
  returnTo: string;
  confirm?: boolean;
}) {
  const impact = v.impact;
  if (!impact) return null;
  return (
    <ObjectDetailSection
      title={confirm ? '停用影响' : '引用此版本的授予与管理职责'}
    >
      <p>
        可见影响涉及 {impact.accounts} 个账号、{impact.grants} 份业务授予、
        {impact.mandates} 份管理职责。
      </p>
      {!impact.token &&
        !v.versions?.find((x) => x.id === returnTo)?.disabled && (
          <p>当前权限不足以停用此版本。</p>
        )}
      <WorkspacePager v={v} change={change} params={params} />
      {impact.references?.length ? (
        <table className="grant-table">
          <thead>
            <tr>
              <th scope="col">账号</th>
              <th scope="col">引用类型</th>
              <th scope="col">业务内容</th>
              <th scope="col">状态</th>
            </tr>
          </thead>
          <tbody>
            {impact.references.map((r) => (
              <tr key={r.id}>
                <td data-title="账号">
                  {confirm ? (
                    r.account
                  ) : (
                    <Link
                      id={'ref-' + r.id}
                      to={
                        accessPath +
                        '/grants/' +
                        r.subject +
                        '?' +
                        new URLSearchParams({
                          tab: r.kind === 'mandate' ? 'mandates' : 'grants',
                          grant: r.id,
                          inspect: r.kind === 'grant' ? '1' : '',
                          fromVersion: returnTo,
                          returnQuery: params.toString(),
                        })
                      }
                    >
                      {r.account}
                    </Link>
                  )}
                </td>
                <td data-title="引用类型">
                  {r.kind === 'mandate' ? '管理职责' : '业务授予'}
                </td>
                <td data-title="业务内容">{r.summary}</td>
                <td data-title="状态">{grantState(r.state)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : (
        <p>没有可见引用</p>
      )}
    </ObjectDetailSection>
  );
}
