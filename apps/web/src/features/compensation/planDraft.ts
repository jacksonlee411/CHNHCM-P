import type { Models } from '@chnhcm/api-client';
import type { Component, PlanDetail, PlanInput } from './plans';
import type { ScopeDraft } from './PlanScope';
import type { Action } from './rates';

export type Draft = {
  code: string;
  name: string;
  description: string;
  reason: string;
  effectiveDate: string;
  scope: ScopeDraft;
  components: Component[];
};
type RowInput = Models['CompensationPlanComponentInput'];
type Changes = NonNullable<PlanInput['changes']>;
const common = ['code', 'name', 'nature', 'currency', 'frequency', 'unit', 'precision', 'method'] as const;
export const same = (a: unknown, b: unknown) => JSON.stringify(a) === JSON.stringify(b);
function rowInput(row: Component, fields: string[]): RowInput {
  const visible: RowInput = Object.fromEntries(common.map((k) => [k, row[k]]));
  visible.id = row.id;
  if (row.method === 'FIXED' && fields.includes('amount') && row.amount !== undefined) visible.amount = row.amount;
  if (row.method === 'GRADE_RATE' && fields.includes('rates') && row.rates !== undefined)
    visible.rates = row.rates.map((r) => ({ gradeId: r.gradeId, rateId: r.rateId }));
  if (row.method === 'PERCENT' && fields.includes('percentage')) {
    if (row.percent !== undefined) visible.percent = row.percent;
    if (row.baseId !== undefined) visible.baseId = row.baseId;
    if (row.rounding !== undefined) visible.rounding = row.rounding;
  }
  return visible;
}
function rowPatch(row: Component, old: Component | undefined, fields: string[]): RowInput {
  const next = rowInput(row, fields),
    prior = old && rowInput(old, fields);
  return prior
    ? {
        id: row.id,
        ...Object.fromEntries(Object.entries(next).filter(([k, v]) => !same(prior[k as keyof RowInput], v))),
      }
    : next;
}
function scopeWire(s: ScopeDraft): Changes['scope'] {
  if (!s.mode) return undefined;
  if (s.mode === 'ENTERPRISE') return { mode: s.mode };
  return {
    mode: s.mode,
    choices: s.choices
      .map((c) => ({ dimension: c.dimension, id: c.id, exclude: !!c.exclude }))
      .sort((a, b) => (a.dimension + a.id + a.exclude).localeCompare(b.dimension + b.id + b.exclude)),
  };
}
export function initialDraft(detail: PlanDetail | undefined, action: Action, date: string, fields: string[]): Draft {
  const r = detail?.record;
  return {
    code: detail?.code ?? '',
    name: r?.name ?? '',
    description: r?.description ?? '',
    effectiveDate: date,
    reason: action === 'CORRECT' || action === 'DELETE' ? (r?.reason ?? '') : '',
    scope: { mode: r?.scope?.mode ?? '', choices: r?.scope?.members?.map((m) => ({ ...m, name: m.name ?? '' })) ?? [] },
    components: (r?.components ?? []).map((item) => {
      const { amount, rates, baseId, rounding, percent, targets, ...meta } = item;
      return {
        ...meta,
        ...(fields.includes('amount') && { amount }),
        ...(fields.includes('rates') && {
          rates: rates?.map(({ gradeId, rateId, code, name }) => ({ gradeId, rateId, code, name })),
        }),
        ...(fields.includes('percentage') && { baseId, rounding, percent }),
      };
    }),
  };
}
export function planChanges(draft: Draft, original: Draft, fields: string[], creating = false): Changes {
  const out: Changes = {};
  for (const k of ['name', 'description'] as const)
    if (fields.includes(k) && (creating || draft[k] !== original[k])) out[k] = draft[k];
  if (fields.includes('scope') && (creating || !same(scopeWire(draft.scope), scopeWire(original.scope))))
    out.scope = scopeWire(draft.scope);
  if (fields.includes('components')) {
    const rows = draft.components.map((row) =>
      rowPatch(row, creating ? undefined : original.components.find((o) => o.id === row.id), fields),
    );
    if (
      creating ||
      rows.some((r) => Object.keys(r).length > 1) ||
      !same(
        rows.map((r) => r.id),
        original.components.map((r) => r.id),
      )
    )
      out.components = rows;
  }
  return out;
}
// Reapply only the local edits the current administrator may still make, preserving
// other authors' updated values and the fresh record/revision used for submission.
export function rebasePlan(local: Draft, old: Draft, fresh: Draft, fields: string[]): Draft {
  const next = { ...fresh };
  for (const k of ['code', 'name', 'description', 'reason', 'effectiveDate'] as const)
    if (fields.includes(k) && local[k] !== old[k]) next[k] = local[k];
  const changes = planChanges(local, old, fields);
  if (changes.scope) next.scope = local.scope;
  if (changes.components) {
    const oldById = new Map(old.components.map((r) => [r.id, r])),
      freshById = new Map(fresh.components.map((r) => [r.id, r]));
    const localById = new Map(local.components.map((r) => [r.id, r]));
    const localOrder = local.components.map((r) => r.id),
      oldOrder = old.components.map((r) => r.id);
    const ordered = same(localOrder, oldOrder)
      ? fresh.components.map((r) => r.id)
      : [...localOrder, ...fresh.components.map((r) => r.id).filter((id) => !oldById.has(id) && !localById.has(id))];
    next.components = ordered.flatMap((id) => {
      const row = localById.get(id),
        current = freshById.get(id),
        prior = oldById.get(id);
      if (!row) return current && !prior ? [current] : [];
      if (!current && prior) return [];
      const allowed = rowInput(row, fields);
      const patch = rowPatch(row, prior, fields);
      const copy = { ...(current ?? {}), ...patch } as Component;
      if (patch.rates && allowed.rates) copy.rates = row.rates;
      return [copy];
    });
  }
  return next;
}
