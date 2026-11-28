import {
  Box,
  Checkbox,
  FormControlLabel,
  Stack,
  Typography,
} from '@mui/material';
import { ObjectDetailField, ObjectTextField, recordedTime } from '@chnhcm/ui';
import {
  label,
  targetName,
  managementRightName,
  versionName,
  version,
  type View,
  type Command,
  type Definition,
} from './management';
import { useId } from 'react';
import { GrantFacts } from './GrantWorkspace';

type Choice = { id: string; name: string };
export function Pick({
  title,
  value,
  options,
  change,
  required = false,
}: {
  title: string;
  value?: string;
  options: Choice[];
  change: (v: string) => void;
  required?: boolean;
}) {
  const id = useId();
  return (
    <ObjectDetailField label={title} inputId={id} required={required}>
      <select
        className="access-select"
        id={id}
        required={required}
        value={value ?? ''}
        onChange={(e) => change(e.target.value)}
      >
        <option value="">请选择</option>
        {options.map((o) => (
          <option key={o.id} value={o.id}>
            {o.name}
          </option>
        ))}
      </select>
    </ObjectDetailField>
  );
}
function CheckList({
  title,
  values = [],
  options,
  change,
  group,
}: {
  group?: string;
  title: string;
  values?: string[];
  options: Choice[];
  change: (v: string[]) => void;
}) {
  return (
    <fieldset className="access-checklist">
      <legend>{title}</legend>
      <div className="security-check-options">
        {options.map((o) => (
          <FormControlLabel
            key={o.id}
            label={o.name}
            control={
              <Checkbox
                size="small"
                slotProps={{
                  input: {
                    'aria-label': group ? group + ' · ' + o.name : o.name,
                  },
                }}
                checked={values.includes(o.id)}
                onChange={(e) =>
                  change(
                    e.target.checked
                      ? [...values, o.id]
                      : values.filter((x) => x !== o.id),
                  )
                }
              />
            }
          />
        ))}
      </div>
    </fieldset>
  );
}
export function versionOptions(
  v: View,
  kind: string,
  right: string,
  definition?: string,
) {
  return (v.versions ?? [])
    .filter(
      (d) =>
        d.kind === kind &&
        !d.disabled &&
        d.id &&
        d.options?.includes(right) &&
        (!definition || d.definitionId === definition),
    )
    .map((d) => ({ id: d.id!, name: versionName(d) }));
}
const choices = (ids: string[]) => ids.map((id) => ({ id, name: label(id) }));
function TargetField({
  view,
  d,
  change,
}: {
  view: View;
  d: Definition;
  change: (v: Definition) => void;
}) {
  return (
    <Pick
      title="业务目标"
      required
      value={d.targetId}
      options={(view.targets ?? []).map((t) => ({
        id: t.id,
        name: label(t.kind) + ' · ' + targetName(view, t.id),
      }))}
      change={(id) =>
        change({
          ...d,
          targetId: id,
          targetKind: view.targets?.find((t) => t.id === id)?.kind,
        })
      }
    />
  );
}
function FieldPolicy({
  view,
  d,
  change,
}: {
  view: View;
  d: Definition;
  change: (v: Definition) => void;
}) {
  return (
    <>
      <CheckList
        title="可读字段"
        values={d.readFields}
        options={choices(view.readFields ?? [])}
        change={(readFields) => change({ ...d, readFields })}
      />
      <CheckList
        title="可写字段"
        values={d.writeFields}
        options={choices(view.writeFields ?? [])}
        change={(writeFields) => change({ ...d, writeFields })}
      />
    </>
  );
}
function PolicyChoice({
  kind,
  v,
  d,
  set,
}: {
  kind: 'scope' | 'fields';
  v: View;
  d: Command;
  set: (d: Command) => void;
}) {
  const key = kind === 'scope' ? 'newScope' : 'newFields',
    newPolicy = d[key],
    name = label(kind),
    role = d.version!;
  const choices = versionOptions(v, kind, 'role');
  if (v.rights?.includes('policy'))
    choices.unshift({ id: 'new', name: '新建' + name });
  const update = (p: Definition | undefined, ref?: string) =>
    set({ ...d, [key]: p, version: { ...role, [kind]: ref } });
  return (
    <>
      <Pick
        title={name}
        value={newPolicy ? 'new' : role[kind]}
        options={choices}
        required
        change={(id) =>
          id === 'new'
            ? update({ kind, name: role.name + name })
            : update(undefined, id)
        }
      />
      {newPolicy && (
        <>
          <ObjectTextField
            required
            label={name + '名称'}
            value={newPolicy.name}
            onChange={(e) => update({ ...newPolicy, name: e.target.value })}
          />
          {kind === 'scope' ? (
            <TargetField view={v} d={newPolicy} change={(p) => update(p)} />
          ) : (
            <FieldPolicy view={v} d={newPolicy} change={(p) => update(p)} />
          )}
        </>
      )}
    </>
  );
}
function MomentInput({
  title,
  value,
  change,
  required = false,
}: {
  title: string;
  value?: string;
  change: (v: string | undefined) => void;
  required?: boolean;
}) {
  const text = value ? recordedTime(value).slice(0, 19).replace(' ', 'T') : '';
  return (
    <ObjectTextField
      label={title + '（北京时间）'}
      type="datetime-local"
      required={required}
      slotProps={{ htmlInput: { step: 1 } }}
      value={text}
      onChange={(e) => {
        const t = e.target.value;
        change(t ? (t.length === 16 ? t + ':00' : t) + '+08:00' : undefined);
      }}
    />
  );
}
export function ManagementFields({
  v,
  d,
  set,
}: {
  v: View;
  d: Command;
  set: (d: Command) => void;
}) {
  const patch = (part: Partial<Command>) => set({ ...d, ...part });
  const op = d.operation;
  const ownerOptions = (right: string) =>
    (v.owners ?? [])
      .filter((t) => t.options?.includes(right))
      .map((t) => ({ id: t.id, name: targetName(v, t.id) }));
  if (op.startsWith('account.'))
    return (
      <>
        {op === 'account.create' && (
          <>
            <ObjectTextField
              label="登录名"
              required
              value={d.login ?? ''}
              autoComplete="off"
              onChange={(e) => patch({ login: e.target.value })}
            />
            <ObjectTextField
              label="显示名"
              value={d.displayName ?? ''}
              onChange={(e) => patch({ displayName: e.target.value })}
            />
          </>
        )}
        {(op === 'account.create' || op === 'account.move') && (
          <Pick
            title="管理归属"
            required
            options={ownerOptions('account')}
            value={d.owner}
            change={(owner) => patch({ owner })}
          />
        )}
        {(op === 'account.create' || op === 'account.reset') && (
          <ObjectTextField
            type="password"
            autoComplete="new-password"
            label={op === 'account.create' ? '初始凭据' : '新凭据'}
            required
            value={d.password ?? ''}
            onChange={(e) => patch({ password: e.target.value })}
          />
        )}
        {op === 'account.disable' && (
          <Typography>停用后，此账号的全部会话将结束。</Typography>
        )}
      </>
    );
  if (op === 'version.publish' && d.version) {
    const def = d.version,
      change = (version: Definition) => patch({ version });
    return (
      <>
        <ObjectTextField
          label={label(def.kind) + '名称'}
          required
          value={def.name}
          onChange={(e) => change({ ...def, name: e.target.value })}
        />
        {def.kind === 'scope' ? (
          <TargetField view={v} d={def} change={change} />
        ) : def.kind === 'fields' ? (
          <FieldPolicy view={v} d={def} change={change} />
        ) : (
          <>
            <div className="object-detail-columns security-policy-fields">
              <div>
                <PolicyChoice kind="scope" v={v} d={d} set={set} />
              </div>
              <div>
                <PolicyChoice kind="fields" v={v} d={d} set={set} />
              </div>
            </div>
            <h2 className="object-section-title">业务动作</h2>
            {Array.from(new Set(v.actions?.map((a) => a.menu))).map((menu) => (
              <CheckList
                key={menu}
                group={menu}
                title={menu}
                values={def.actions}
                options={(v.actions ?? [])
                  .filter((a) => a.menu === menu)
                  .map((a) => ({ id: a.key, name: a.name }))}
                change={(actions) => change({ ...def, actions })}
              />
            ))}
            <details>
              <summary>查看对应 API</summary>
              {(v.actions ?? [])
                .filter((a) => def.actions?.includes(a.key))
                .map((a) => (
                  <div key={a.key}>
                    {a.name} · {a.api}
                  </div>
                ))}
            </details>
          </>
        )}
      </>
    );
  }
  if (op === 'version.disable') return <></>;
  const mandate = op === 'mandate.issue',
    adopting = op === 'grant.adopt';
  if (mandate || op === 'grant.issue' || adopting) {
    const l = d.limit,
      grant = v.grants?.find((g) => g.id === d.id),
      bound = mandate ? l : d,
      right = mandate ? 'delegate' : 'assign';
    const selected = version(v, 'role', bound?.role);
    const selectBinding = (x: {
      role?: string;
      scope?: string;
      fields?: string;
    }) => (mandate && l ? patch({ limit: { ...l, ...x } }) : patch(x));
    const old = (kind: 'role' | 'scope' | 'fields') =>
      adopting ? version(v, kind, grant?.[kind])?.definitionId : undefined;
    const purposes = Array.from(
      new Set(selected?.actions?.map((a) => a.split('.')[0])),
    );
    return (
      <>
        <div className="grant-form-columns">
          <section>
            <h3>{mandate ? '委派内容' : '授予内容'}</h3>
            {mandate && l && (
              <>
                <Pick
                  title="获管账号归属"
                  required
                  value={l.owner}
                  options={ownerOptions('delegate')}
                  change={(owner) => patch({ limit: { ...l, owner } })}
                />
                <CheckList
                  title="可办理的管理事项"
                  values={l.rights}
                  options={(v.rights ?? []).map((id) => ({
                    id,
                    name: managementRightName(id),
                  }))}
                  change={(rights) => patch({ limit: { ...l, rights } })}
                />
              </>
            )}
            <Pick
              title={mandate ? '可委派业务的角色上限' : '角色版本'}
              required
              value={bound?.role}
              options={versionOptions(v, 'role', right, old('role'))}
              change={(role) => {
                const r = version(v, 'role', role);
                // A role can change its suggested policies. Adoption stays within
                // the definitions already assigned to this individual grant.
                const policy = (kind: 'scope' | 'fields') =>
                  !adopting ||
                  version(v, kind, r?.[kind])?.definitionId === old(kind)
                    ? r?.[kind]
                    : bound?.[kind];
                selectBinding({
                  role,
                  scope: policy('scope'),
                  fields: policy('fields'),
                });
              }}
            />
            {selected && (
              <>
                <Pick
                  title="范围版本"
                  required
                  value={bound?.scope}
                  options={versionOptions(v, 'scope', right, old('scope'))}
                  change={(scope) => selectBinding({ scope })}
                />
                <ObjectDetailField label="业务目标">
                  {targetName(
                    v,
                    version(v, 'scope', bound?.scope)?.targetId ?? '',
                  )}
                </ObjectDetailField>
                <Pick
                  title="字段版本"
                  required
                  value={bound?.fields}
                  options={versionOptions(v, 'fields', right, old('fields'))}
                  change={(fields) => selectBinding({ fields })}
                />
                <ObjectDetailField label="可读字段">
                  {version(v, 'fields', bound?.fields)
                    ?.readFields?.map(label)
                    .join('、') || '无'}
                </ObjectDetailField>
                <ObjectDetailField label="可写字段">
                  {version(v, 'fields', bound?.fields)
                    ?.writeFields?.map(label)
                    .join('、') || '无'}
                </ObjectDetailField>
              </>
            )}
            {selected && (
              <ObjectDetailField label="业务动作">
                {selected.actionNames?.join('、') || '原动作名称未登记'}
              </ObjectDetailField>
            )}
          </section>
          <section>
            <h3>{adopting ? '原授予' : mandate ? '委派期限' : '来源与期限'}</h3>
            {adopting && grant && <GrantFacts v={v} g={grant} />}
            {mandate && l ? (
              <>
                <ObjectTextField
                  label="代理期限上限（天，0 为无上限）"
                  type="number"
                  value={l.maxProxyDays}
                  onChange={(e) =>
                    patch({
                      limit: { ...l, maxProxyDays: Number(e.target.value) },
                    })
                  }
                />
                <MomentInput
                  title="管理职责到期"
                  value={l.until}
                  change={(until) => patch({ limit: { ...l, until } })}
                />
              </>
            ) : (
              !adopting && (
                <>
                  {purposes.length > 1 ? (
                    <Pick
                      title="用途"
                      required
                      options={choices(purposes)}
                      value={d.purpose}
                      change={(purpose) => patch({ purpose })}
                    />
                  ) : (
                    purposes.length === 1 && (
                      <ObjectDetailField label="用途">
                        {label(purposes[0])}
                      </ObjectDetailField>
                    )
                  )}
                  <Pick
                    title="授予来源"
                    value={d.source}
                    required
                    options={choices(['duty', 'proxy'])}
                    change={(source) => patch({ source })}
                  />
                  <MomentInput
                    title="开始，留空即生效"
                    value={d.from}
                    change={(from) => patch({ from })}
                  />
                  <MomentInput
                    title={
                      d.source === 'proxy' ? '代理到期' : '到期，留空为长期'
                    }
                    value={d.until}
                    required={d.source === 'proxy'}
                    change={(until) => patch({ until })}
                  />
                </>
              )
            )}
            {d.source === 'proxy' && !!v.proxyDays && (
              <p>本次可选职责的代理期限上限 {v.proxyDays} 天</p>
            )}
          </section>
        </div>
      </>
    );
  }
  return (
    <Box>
      <ObjectTextField
        label="办理事由（选填）"
        value={d.reason ?? ''}
        onChange={(e) => patch({ reason: e.target.value })}
      />
    </Box>
  );
}
