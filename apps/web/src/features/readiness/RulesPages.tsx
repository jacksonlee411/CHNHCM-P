import { useEffect, useState } from 'react';
import { Checkbox, FormControlLabel, MenuItem, Typography } from '@mui/material';
import { useLocation, useParams } from 'react-router-dom';
import { ObjectTextField } from '@chnhcm/ui';
import { api, sessionGuard, unwrap, type Models } from '@chnhcm/api-client';
import { DatedRules } from './DatedRules';
import { LEGAL_ENTITY_PATH } from '../organization/LegalEntities';

export const policyPath = '/settings/staffing/assignment-policies';
type Purpose = Models['LegalPurpose'];
type Policy = Models['AssignmentPolicy'];
type Session = { session: Models['Principal'] };
const roles = {
  employer: '用人单位',
  receiver: '派遣接收',
  signatory: '受托签约',
  payroll: '薪资责任',
  withholding: '扣缴',
  contribution: '参缴',
  payment: '付款',
  cost: '成本',
};
const kinds = { declaration: '资格声明', bu: '业务单元绑定', location: '地点绑定' };
const modes = { JobOnly: '仅按职务', PositionExempt: '按岗位免占编', PositionCounted: '按岗位占编' };
const options = (values: Record<string, string>) =>
  Object.entries(values).map(([key, label]) => (
    <MenuItem key={key} value={key}>
      {label}
    </MenuItem>
  ));

function BindingTarget({
  kind,
  value,
  choose,
}: {
  kind: 'bu' | 'location';
  value: string;
  choose: (v: string) => void;
}) {
  const [query, setQuery] = useState(''),
    [items, setItems] = useState<{ id: string; name?: string | null; code?: string | null }[]>([]),
    [error, setError] = useState('');
  useEffect(() => {
    const stop = new AbortController(),
      same = sessionGuard(),
      params = { query: { q: query, page: 1, pageSize: 20 as const } };
    const request =
      kind === 'bu'
        ? api.GET('/api/v1/business-units', { params, signal: stop.signal }).then((r) => unwrap(r).items)
        : api.GET('/api/v1/locations', { params, signal: stop.signal }).then((r) => unwrap(r).items);
    setError('');
    setItems([]);
    request
      .then((rows) => {
        if (!stop.signal.aborted && same()) setItems(rows);
      })
      .catch((e) => {
        if (!stop.signal.aborted && same()) setError(e.message);
      });
    return () => stop.abort();
  }, [kind, query]);
  return (
    <>
      <ObjectTextField
        label="搜索绑定目标"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        error={!!error}
        helperText={error}
      />
      <ObjectTextField
        select
        required
        label={kind === 'bu' ? '业务单元' : '地点'}
        value={value}
        onChange={(e) => choose(e.target.value)}
      >
        {value && !items.some((r) => r.id === value) && (
          <MenuItem value={value}>
            <TargetLabel kind={kind} id={value} />
          </MenuItem>
        )}
        {items.map((r) => (
          <MenuItem key={r.id} value={r.id}>
            {r.name ?? r.code}
          </MenuItem>
        ))}
      </ObjectTextField>
    </>
  );
}
export function PurposePage({ session }: Session) {
  const { id = '' } = useParams(),
    location = useLocation();
  return (
    <DatedRules<Purpose>
      key={id + session.instanceId + session.subjectId}
      session={session}
      title="用途与绑定"
      scope="legal-purpose"
      resolve={async (sub) =>
        unwrap(
          await api.POST('/api/v1/legal-entities/purpose-submissions/{sub}/resolve', {
            params: { path: { sub }, header: { 'X-CSRF-Token': session.csrf } },
          }),
        )
      }
      back={LEGAL_ENTITY_PATH + '/' + id + location.search}
      load={async (version, signal) =>
        unwrap(
          await api.GET('/api/v1/legal-entities/{id}/purposes', {
            params: { path: { id }, query: { sourceVersion: version || undefined } },
            signal,
          }),
        )
      }
      save={async (expectedVersion, row, key, insertAfter) =>
        unwrap(
          await api.POST('/api/v1/legal-entities/{id}/purposes', {
            params: { path: { id }, header: { 'X-CSRF-Token': session.csrf, 'Idempotency-Key': key } },
            body: { expectedVersion, row, insertAfter },
          }),
        )
      }
      blank={(from) => ({ id: crypto.randomUUID(), from, kind: 'declaration', role: 'employer', basis: '' })}
      label={(r) => roles[r.role] + ' · ' + kinds[r.kind]}
      summary={(r) => (
        <>
          <Typography>{r.basis}</Typography>
          {r.requireBU && <Typography>限定业务单元</Typography>}
          {r.requireLocation && <Typography>限定地点</Typography>}
          {r.target && <TargetLabel kind={r.kind} id={r.target} />}
        </>
      )}
      fields={(r, update) => (
        <>
          <ObjectTextField
            select
            label="主体用途"
            value={r.role}
            onChange={(e) => update({ role: e.target.value as Purpose['role'] })}
          >
            {options(roles)}
          </ObjectTextField>
          <ObjectTextField
            select
            label="记录类型"
            value={r.kind}
            onChange={(e) =>
              update({
                kind: e.target.value as Purpose['kind'],
                target: undefined,
                requireBU: false,
                requireLocation: false,
              })
            }
          >
            {options(kinds)}
          </ObjectTextField>
          {r.kind === 'declaration' ? (
            <>
              <FormControlLabel
                label="限定业务单元"
                control={
                  <Checkbox
                    checked={!!r.requireBU}
                    onChange={(e) => update({ requireBU: e.target.checked })}
                  />
                }
              />
              <FormControlLabel
                label="限定地点"
                control={
                  <Checkbox
                    checked={!!r.requireLocation}
                    onChange={(e) => update({ requireLocation: e.target.checked })}
                  />
                }
              />
            </>
          ) : (
            <BindingTarget kind={r.kind} value={r.target ?? ''} choose={(target) => update({ target })} />
          )}
          <ObjectTextField
            label="资格或绑定依据"
            required
            value={r.basis}
            onChange={(e) => update({ basis: e.target.value })}
          />
        </>
      )}
    />
  );
}
function TargetLabel({ kind, id }: { kind: Purpose['kind']; id: string }) {
  const [name, setName] = useState('');
  useEffect(() => {
    const stop = new AbortController(),
      same = sessionGuard(),
      params = { path: { id } };
    const req =
      kind === 'bu'
        ? api
            .GET('/api/v1/business-units/{id}', { params, signal: stop.signal })
            .then((r) => unwrap(r).selectedRecord?.name)
        : api
            .GET('/api/v1/locations/{id}', { params, signal: stop.signal })
            .then((r) => unwrap(r).selectedRecord?.name);
    setName('');
    req
      .then((n) => {
        if (same() && !stop.signal.aborted) setName(n ?? '名称不可用');
      })
      .catch(() => {
        if (same() && !stop.signal.aborted) setName('名称不可用');
      });
    return () => stop.abort();
  }, [kind, id]);
  return <Typography>{name}</Typography>;
}
export function AssignmentPolicyPage({ session }: Session) {
  return (
    <DatedRules<Policy>
      key={session.instanceId + session.subjectId}
      title="任职与占编规则"
      scope="assignment-policy"
      resolve={async (sub) =>
        unwrap(
          await api.POST('/api/v1/assignment-policies/submissions/{sub}/resolve', {
            params: { path: { sub }, header: { 'X-CSRF-Token': session.csrf } },
          }),
        )
      }
      session={session}
      load={async (version, signal) =>
        unwrap(
          await api.GET('/api/v1/assignment-policies', {
            params: { query: { sourceVersion: version || undefined } },
            signal,
          }),
        )
      }
      save={async (expectedVersion, row, key, insertAfter) =>
        unwrap(
          await api.POST('/api/v1/assignment-policies', {
            params: { header: { 'X-CSRF-Token': session.csrf, 'Idempotency-Key': key } },
            body: { expectedVersion, row, insertAfter },
          }),
        )
      }
      blank={(from) => ({ id: crypto.randomUUID(), from, category: '', purpose: '', mode: 'JobOnly' })}
      label={(r) => r.category + ' · ' + r.purpose}
      summary={(r) => <Typography>{modes[r.mode]}</Typography>}
      fields={(r, update) => (
        <>
          <ObjectTextField
            required
            label="人员类别"
            value={r.category}
            onChange={(e) => update({ category: e.target.value })}
          />
          <ObjectTextField
            required
            label="用途"
            value={r.purpose}
            onChange={(e) => update({ purpose: e.target.value })}
          />
          <ObjectTextField
            select
            label="任职模式"
            value={r.mode}
            onChange={(e) => update({ mode: e.target.value as Policy['mode'] })}
          >
            {options(modes)}
          </ObjectTextField>
          <Typography>{r.mode === 'PositionCounted' ? '每份任职占一编' : '占编数为零'}</Typography>
        </>
      )}
    />
  );
}
