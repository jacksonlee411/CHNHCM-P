import { useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { Alert, Button, MenuItem } from '@mui/material';
import { ApiError, sessionGuard } from '@chnhcm/api-client';
import {
  ObjectDetailField,
  ObjectDetailSection,
  ObjectForm,
  ObjectFormError,
  ObjectTextField,
  UnsavedChangesDialog,
  focusObjectField,
} from '@chnhcm/ui';
import { useObjectSubmission } from '../../shared/dated/useObjectSubmission';
import { useDraftExit } from '../../shared/dated/useDraftExit';
import { message, useRead } from '../../shared/useRead';
import { ComponentEditor } from './PlanComponents';
import { PlanScopeEditor } from './PlanScope';
import { initialDraft, planChanges, rebasePlan, type Draft } from './planDraft';
import { planChannel, readPlan, resolvePlan, savePlan, scopeLabel, type PlanDetail, type PlanInput } from './plans';
import { type Principal, type Action, type Receipt } from './rates';

export type PlanDraftBackup = { original: Draft; draft: Draft };
type Props = {
  recovered?: PlanDraftBackup;
  onDenied: (input: PlanDraftBackup) => void;
  session: Principal;
  detail?: PlanDetail;
  action: Action;
  fields: string[];
  reasons: string[];
  units: Record<string, string>;
  scopeDimensions: PlanDetail['scopeDimensions'];
  defaultDate: string;
  host: HTMLElement | null;
  onCancel: () => void;
  onSaved: (r: Receipt) => void;
  onReload: () => void;
  onRefresh: () => void;
};
export function PlanEditor(props: Props) {
  const { session, detail, action, fields, onSaved, onReload } = props;
  const creating = action === 'CREATE',
    correct = action === 'CORRECT',
    deleting = action === 'DELETE';
  const values = !deleting && action !== 'DEACTIVATE' && !(correct && detail?.record.status === 'INACTIVE');
  const [original, setOriginal] = useState(() => initialDraft(detail, action, props.defaultDate, fields)),
    [draft, setDraft] = useState(() => {
      const saved = props.recovered;
      if (!saved) return original;
      const next = rebasePlan(saved.draft, saved.original, original, fields);
      next.scope = {
        ...next.scope,
        choices: next.scope.choices.filter((c) => props.scopeDimensions.includes(c.dimension)),
      };
      return next;
    });
  const [basis, setBasis] = useState({ version: detail?.sourceVersion, revision: detail?.record.revision });
  const [issue, setIssue] = useState<{ field?: string; text: string }>(),
    [conflict, setConflict] = useState(false),
    [rereading, setRereading] = useState(false);
  const capabilityKey = JSON.stringify([fields, props.scopeDimensions]);
  const previousFields = useRef(capabilityKey),
    saved = useRef(false);
  const [rowDirty, setRowDirty] = useState(false);
  const stale = conflict || basis.version !== detail?.sourceVersion;
  const receive = (r: Receipt) => {
    if (r.outcome === 'accepted' || r.outcome === 'no_change') {
      saved.current = true;
      onSaved(r);
    } else
      setIssue({ text: r.message ?? (r.outcome === 'not_applied' ? '原提交未执行，可重新保存。' : '原提交被拒绝。') });
  };
  const { flow, pending, sessionBlocked } = useObjectSubmission(
    session,
    planChannel,
    (c: PlanInput, sub) => savePlan(session, detail?.id ?? '', c, sub),
    (ref) => resolvePlan(session, ref),
    receive,
    (e) => e instanceof ApiError && e.status >= 400 && e.status < 500,
  );
  const changed = planChanges(draft, original, fields);
  const dirty =
    draft.code !== original.code ||
    draft.reason !== original.reason ||
    draft.effectiveDate !== original.effectiveDate ||
    Object.keys(changed).length > 0;
  const exit = useDraftExit(
    dirty || rowDirty,
    () => saved.current || flow.saved,
    () => {
      saved.current = true;
      flow.clearPayload();
    },
  );
  const writable = (k: string) => fields.includes(k);
  const set = <K extends keyof Draft>(k: K, v: Draft[K]) => {
    setDraft((prev) => ({ ...prev, [k]: v }));
    setIssue(undefined);
  };
  const refreshFrom = (fresh: PlanDetail | undefined, allowed: string[], dimensions: PlanDetail['scopeDimensions']) => {
    const baseline = initialDraft(
      fresh,
      action,
      fresh?.windows[action]?.default ?? fresh?.record.effectiveDate ?? props.defaultDate,
      allowed,
    );
    setDraft((v) => {
      const next = rebasePlan(v, original, baseline, allowed);
      next.scope = { ...next.scope, choices: next.scope.choices.filter((c) => dimensions.includes(c.dimension)) };
      return next;
    });
    setOriginal(baseline);
    setBasis({ version: fresh?.sourceVersion, revision: fresh?.record.revision });
    setIssue(undefined);
  };
  useEffect(() => {
    if (previousFields.current !== capabilityKey) {
      refreshFrom(detail, fields, props.scopeDimensions);
      flow.clearPayload();
      previousFields.current = capabilityKey;
    }
  }, [capabilityKey, detail]);
  const reread = async () => {
    if (!detail || rereading) return;
    const current = sessionGuard();
    setRereading(true);
    try {
      const fresh = await readPlan(detail.id, detail.record.recordId);
      if (!current()) return;
      if (!fresh.actions[action]) {
        onReload();
        return;
      }
      refreshFrom(fresh, fresh.actions[action], fresh.scopeDimensions);
      setConflict(false);
      props.onRefresh();
    } catch (e) {
      if (current()) {
        if (e instanceof ApiError && [401, 403, 404].includes(e.status)) onReload();
        else setIssue({ text: message(e) });
      }
    } finally {
      if (current()) setRereading(false);
    }
  };
  useEffect(() => {
    if (!issue?.field) return;
    const row = /^components\.(\d+)/.exec(issue.field);
    focusObjectField(row ? 'plan-components-' + row[1] : 'plan-' + issue.field, 'plan-error');
  }, [issue]);
  const busy = pending || sessionBlocked || flow.inFlight || flow.unknown;
  const save = async (event: React.FormEvent) => {
    event.preventDefault();
    if (busy || stale || rereading) return;
    const c: PlanInput = { action };
    if (creating) c.code = draft.code;
    else
      Object.assign(c, {
        recordId: detail?.record.recordId,
        expectedVersion: basis.version,
        expectedRevision: basis.revision,
      });
    if (!deleting) {
      if (writable('effectiveDate') && (!correct || draft.effectiveDate !== original.effectiveDate))
        c.effectiveDate = draft.effectiveDate;
      if (writable('reason') && (!correct || draft.reason !== original.reason)) c.reason = draft.reason;
      if (values) c.changes = planChanges(draft, original, fields, creating);
    }
    setIssue(undefined);
    const result = await flow.submit(c);
    if (result.kind === 'saved') receive(result.result);
    if (result.kind === 'failed') {
      const e = result.error;
      if (e instanceof ApiError && e.status === 403) {
        flow.clearPayload();
        props.onDenied({ original, draft });
        return;
      }
      if (e instanceof ApiError && [401, 404].includes(e.status)) {
        saved.current = true;
        onReload();
        return;
      }
      if (e instanceof ApiError && e.code === 'M09_VERSION_CONFLICT') {
        setConflict(true);
        return;
      }
      setIssue({ text: message(e), field: e instanceof ApiError ? e.field : undefined });
    }
  };
  const controls = (
    <>
      <Button disabled={flow.inFlight} onClick={() => exit.request(props.onCancel)}>
        取消
      </Button>
      <Button
        variant="contained"
        type="submit"
        form="compensation-plan-form"
        disabled={busy || stale || rereading}
        color={deleting ? 'error' : 'primary'}
      >
        {deleting
          ? '确认删除'
          : action === 'DEACTIVATE'
            ? '确认停用'
            : action === 'REACTIVATE'
              ? '确认重新启用'
              : '保存'}
      </Button>
    </>
  );
  const record = detail?.record,
    window = detail?.windows[action];
  const previous = detail?.dates[(detail?.dates.findIndex((d) => d.recordId === record?.recordId) ?? 0) - 1];
  const restored = useRead(
    deleting && previous ? JSON.stringify([detail?.id, detail?.sourceVersion, previous.recordId]) : undefined,
    () => readPlan(detail!.id, previous!.recordId),
    'fresh',
  );
  const input = (k: 'code' | 'name' | 'description', label: string, required = false) => (
    <ObjectTextField
      id={'plan-' + k}
      label={label}
      required={required}
      value={draft[k]}
      onChange={(e) => set(k, e.target.value)}
      error={issue?.field === k}
      helperText={issue?.field === k ? issue.text : undefined}
    />
  );
  return (
    <ObjectForm>
      <form id="compensation-plan-form" onSubmit={save} noValidate>
        <fieldset disabled={busy} style={{ border: 0, padding: 0, margin: 0, minWidth: 0 }}>
          {stale && (
            <Alert
              severity="warning"
              action={
                <Button disabled={rereading} onClick={() => void reread()}>
                  重新读取
                </Button>
              }
            >
              这份计划已变更，请重新读取并核对输入。
            </Alert>
          )}
          {issue && <ObjectFormError id="plan-error" message={issue.text} />}
          {deleting ? (
            <ObjectDetailSection title="撤去这条记录">
              <p>
                删除 {record?.effectiveDate} 的记录后，从该日起沿用前一条计划记录
                {record?.endExclusive ? '，直至 ' + record.endExclusive + ' 的后续记录生效' : ''}。
              </p>
              {restored.data && (
                <ObjectDetailField label="恢复适用的计划">
                  {restored.data.record.name} · {restored.data.record.effectiveDate}
                </ObjectDetailField>
              )}
              {restored.error && <Alert severity="error">{restored.error}</Alert>}
            </ObjectDetailSection>
          ) : (
            <>
              <ObjectDetailSection title="生效记录">
                <div className="object-detail-columns">
                  {writable('effectiveDate') ? (
                    <ObjectTextField
                      type="date"
                      label="生效日期"
                      id="plan-effectiveDate"
                      required
                      value={draft.effectiveDate}
                      onChange={(e) => set('effectiveDate', e.target.value)}
                      error={issue?.field === 'effectiveDate'}
                      helperText={issue?.field === 'effectiveDate' ? issue.text : undefined}
                      slotProps={{ htmlInput: { min: window?.minimum, max: window?.maximum } }}
                    />
                  ) : (
                    record?.effectiveDate && (
                      <ObjectDetailField label="生效日期">{record.effectiveDate}</ObjectDetailField>
                    )
                  )}
                  {writable('reason') && (
                    <ObjectTextField
                      id="plan-reason"
                      select
                      label="办理原因"
                      required
                      value={draft.reason}
                      onChange={(e) => set('reason', e.target.value)}
                    >
                      {[...new Set([original.reason, ...props.reasons].filter(Boolean))].map((r) => (
                        <MenuItem key={r} value={r}>
                          {r}
                        </MenuItem>
                      ))}
                    </ObjectTextField>
                  )}
                </div>
              </ObjectDetailSection>
              <ObjectDetailSection title="计划资料">
                <div className="object-detail-columns">
                  {creating && writable('code')
                    ? input('code', '计划编码', true)
                    : detail?.code && <ObjectDetailField label="计划编码">{detail.code}</ObjectDetailField>}
                  {values && writable('name')
                    ? input('name', '计划名称', true)
                    : record?.name && <ObjectDetailField label="计划名称">{record.name}</ObjectDetailField>}
                  {values && writable('description') && input('description', '说明')}
                </div>
              </ObjectDetailSection>
              {values && (writable('scope') || record?.scope) && (
                <ObjectDetailSection title="适用对象">
                  {writable('scope') ? (
                    <PlanScopeEditor
                      dimensions={props.scopeDimensions}
                      value={draft.scope}
                      onChange={(v) => set('scope', v)}
                      error={issue?.field === 'scope' ? issue.text : undefined}
                    />
                  ) : (
                    scopeLabel(record?.scope)
                  )}
                </ObjectDetailSection>
              )}
              {values && writable('components') && (
                <ObjectDetailSection title="薪酬组成">
                  <ComponentEditor
                    key={capabilityKey}
                    rows={draft.components}
                    fields={fields}
                    units={props.units}
                    date={draft.effectiveDate}
                    onScratchChange={setRowDirty}
                    onChange={(v) => set('components', v)}
                    issue={issue}
                  />
                </ObjectDetailSection>
              )}
            </>
          )}
        </fieldset>
        {props.host ? createPortal(controls, props.host) : controls}
      </form>
      <UnsavedChangesDialog {...exit.dialog} pending={flow.unknown} />
    </ObjectForm>
  );
}
