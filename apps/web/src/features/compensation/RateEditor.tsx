import { useEffect, useLayoutEffect, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { createPortal } from 'react-dom';
import { Alert, Autocomplete, Button, Dialog, DialogActions, DialogContent, DialogTitle, InputAdornment, MenuItem, TextField } from '@mui/material';
import { api, ApiError, sessionGuard, unwrap, type Models } from '@chnhcm/api-client';
import { ObjectDetailField, ObjectDetailSection, ObjectForm, ObjectFormError, ObjectTextField, UnsavedChangesDialog, focusObjectField } from '@chnhcm/ui';
import { useDraftExit } from '../../shared/dated/useDraftExit';
import { useObjectSubmission } from '../../shared/dated/useObjectSubmission';
import { message, useRead } from '../../shared/useRead';
import { actionLabel, amountKeys, amountLabels, formLabels, frequencies, gradeLabel, moneyUnit, rateAmounts, rateChannel, rateDetail, rateList, ratePath, statusLabels, resolveRate, saveRate,
  type Action, type AmountKey, type Command, type Detail, type Principal, type Receipt } from './rates';

type Grade = Pick<Models['GradeSummary'], 'id' | 'code' | 'name' | 'effectiveDate'>;
type Draft = { gradeId: string; currency: string; frequency: string; precision: string; form: string; effectiveDate: string; reason: string } & Record<AmountKey, string>;
export type RateDraftBackup = { original: Draft; draft: Draft; grade?: Grade };
function rebaseRate(draft: Draft, original: Draft, fresh: Draft, fields: string[]): Draft {
  const next = { ...fresh };
  for (const key of Object.keys(draft) as (keyof Draft)[]) {
    if (fields.includes(key) && draft[key] !== original[key]) next[key] = draft[key];
  }
  return next;
}
type Props = { recovered?: RateDraftBackup; onDenied: (input: RateDraftBackup) => void; returnTo: string; session: Principal; detail?: Detail; action: Action; fields: string[]; reasons: string[]; units?: Record<string,string>; defaultDate: string;
  host: HTMLElement | null; onCancel: () => void; onSaved: (r: Receipt) => void; onReload: () => void; onRefresh: () => void };
function defaults({ detail, action, defaultDate, fields }: Props): Draft {
  const r = detail?.record;
  const field = (key: AmountKey) => fields.includes(key) ? r?.[key] ?? '' : '';
  return { gradeId: r?.grade?.id ?? '', currency: detail?.currency ?? 'CNY', frequency: detail?.frequency ?? 'MONTH', precision: String(detail?.precision ?? 2),
    form: r?.form ?? (action === 'CREATE' ? 'FIXED' : ''), amount: field('amount'), minimum: field('minimum'), guidance: field('guidance'), maximum: field('maximum'),
    effectiveDate: defaultDate, reason: action === 'CORRECT' || action === 'DELETE' ? r?.reason ?? '' : '' };
}

function GradeInput({ value, onChange, error }: { value?: Grade; onChange: (grade?: Grade) => void; error?: string }) {
  const [input, setInput] = useState(''), [search, setSearch] = useState('');
  const list = useRead(search, () => api.GET('/api/v1/grades', { params: { query: { q: search || undefined, page: 1, pageSize: 50 } } }).then(unwrap), 'fresh');
  const label = (g: Grade) => [g.code, g.name].filter(Boolean).join(' ');
  const options = list.data?.items.filter(row => row.name !== undefined || row.code !== undefined) ?? [];
  return <ObjectDetailField label="职级" inputId="rate-gradeId" required>
    <Autocomplete<Grade> id="rate-gradeId" options={options} loading={!list.data && !list.error} loadingText="正在查询"
      noOptionsText="没有符合条件的职级" filterOptions={v => v} getOptionLabel={label} getOptionKey={g => g.id} isOptionEqualToValue={(a, b) => a.id === b.id}
      value={value ?? null} inputValue={input} onInputChange={(_, text, reason) => { setInput(text); if (reason === 'input' || reason === 'clear') setSearch(text); }}
      onChange={(_, g) => onChange(g ?? undefined)} renderInput={props => <TextField {...props} variant="standard" placeholder="输入职级编码或名称"
        error={!!error || !!list.error} helperText={error || list.error || (value?.effectiveDate && '所选目录记录生效日 ' + value.effectiveDate)} />} />
  </ObjectDetailField>;
}
export function RateEditor(props: Props) {
  const { session, detail, action, fields, host, onCancel, onSaved, onReload } = props;
  const creating = action === 'CREATE', deleting = action === 'DELETE', correct = action === 'CORRECT';
  const [original, setOriginal] = useState(() => defaults(props));
  const [basis,setBasis] = useState({ version: detail?.sourceVersion, revision: detail?.record.revision });
  const [conflict,setConflict] = useState(false), [rereading,setRereading] = useState(false);
  const stale = conflict || basis.version !== detail?.sourceVersion;
  const [draft, setDraft] = useState(() => props.recovered ? rebaseRate(props.recovered.draft, props.recovered.original, original, fields) : original);
  const [grade, setGrade] = useState<Grade | undefined>(() => fields.includes('gradeId') ? props.recovered?.grade : undefined);
  const [issue, setIssue] = useState<{ text: string; field?: string; duplicate?: boolean }>(), [nextForm, setNextForm] = useState('');
  const matching = useRead(issue?.duplicate ? JSON.stringify([draft.gradeId, draft.currency, draft.frequency]) : undefined,
    () => rateList({ gradeId: draft.gradeId, currency: draft.currency, frequency: draft.frequency, page: 1, pageSize: 1 }), 'fresh');
  const existing = matching.data?.items[0];
  const dirty = (Object.keys(original) as (keyof Draft)[]).some(k => draft[k] !== original[k]);
  const saved = useRef(false);
  const onResult = (r: Receipt) => {
    if (r.outcome === 'accepted' || r.outcome === 'no_change') { saved.current = true; onSaved(r); }
    else setIssue({ text: r.message ?? (r.outcome === 'not_applied' ? '原提交未执行，可重新保存。' : '原提交被拒绝。') });
  };
  const { flow, pending, sessionBlocked } = useObjectSubmission(session, rateChannel,
    (c: Command, sub) => saveRate(session, detail?.id ?? '', c, sub), ref => resolveRate(session, ref), onResult,
    e => e instanceof ApiError && e.status >= 400 && e.status < 500);
  const exit = useDraftExit(dirty, () => saved.current || flow.saved, () => { saved.current = true; flow.clearPayload(); });
  const writable = (key: string) => fields.includes(key);
  const fieldKey = JSON.stringify(fields), priorFields = useRef(fieldKey);
  useLayoutEffect(() => {
    if (priorFields.current === fieldKey) return;
    priorFields.current = fieldKey;
    const baseline = defaults(props);
    setDraft(value => rebaseRate(value, original, baseline, fields));
    setOriginal(baseline);
    setBasis({version: detail?.sourceVersion, revision: detail?.record.revision});
    if (!fields.includes('gradeId')) setGrade(undefined);
    setIssue(undefined); setNextForm(''); flow.clearPayload();
  }, [fieldKey, detail]);
  const reread = async () => {
    if (!detail || rereading) return;
    const current = sessionGuard();
    setRereading(true);
    try {
      const fresh = await rateDetail(detail.id, detail.record.recordId);
      if (!current()) return;
      const allowed = fresh.actions[action];
      if (!allowed) { onReload(); return; }
      const baseline = defaults({ ...props, fields: allowed, detail: fresh, defaultDate: fresh.windows[action]?.default ?? fresh.record.effectiveDate ?? '' });
      setDraft(rebaseRate(draft, original, baseline, allowed)); setOriginal(baseline); setBasis({version:fresh.sourceVersion, revision:fresh.record.revision});
      setConflict(false); setIssue(undefined); props.onRefresh();
    } catch (e) {
      if (current()) { if (e instanceof ApiError && [401,403,404].includes(e.status)) onReload(); else setIssue({text:message(e)}); }
    } finally { if (current()) setRereading(false); }
  };
  const valueFields = action !== 'DEACTIVATE' && !deleting && !(correct && detail?.record.status === 'INACTIVE');
  const visibleKeys: AmountKey[] = draft.form === 'FIXED' ? ['amount'] : draft.form === 'RANGE' ? ['minimum', 'guidance', 'maximum'] : amountKeys;
  const set = (key: keyof Draft, value: string) => { setDraft(d => ({ ...d, [key]: value })); setIssue(undefined); };
  const replaceForm = (form: string) => { setDraft(d => ({ ...d, form, amount: '', minimum: '', guidance: '', maximum: '' })); setNextForm(''); setIssue(undefined); };
  const setForm = (form: string) => {
    if (draft.form === form) return;
    if (visibleKeys.some(k => draft[k] !== '')) setNextForm(form); else replaceForm(form);
  };
  const window = detail?.windows[action], busy = sessionBlocked || flow.inFlight || flow.unknown || pending;
  useEffect(() => {
    if (issue?.field) focusObjectField('rate-' + issue.field, 'rate-error');
  }, [issue]);
  const save = async (event: React.FormEvent) => {
    event.preventDefault();
    if (busy || stale || rereading) return;
    const c: Command = { action };
    if (creating) Object.assign(c, { gradeId: draft.gradeId, currency: draft.currency, frequency: draft.frequency,
      unit: props.units?.[draft.frequency], precision: draft.precision === '' ? undefined : Number(draft.precision) });
    else Object.assign(c, { recordId: detail?.record.recordId, expectedRevision: basis.revision, expectedVersion: basis.version });
    if (!deleting) {
      if (writable('effectiveDate') && (!correct || draft.effectiveDate !== original.effectiveDate)) c.effectiveDate = draft.effectiveDate;
      if (writable('reason') && (!correct || draft.reason !== original.reason)) c.reason = draft.reason;
      if (valueFields) {
        const changes: NonNullable<Command['changes']> = {};
        const switched = draft.form !== original.form;
        if (writable('form') && (creating || switched)) changes.form = draft.form;
        for (const key of visibleKeys) {
          if (writable(key) && (creating || switched || draft[key] !== original[key]) && !(key === 'guidance' && draft[key] === '' && (creating || switched))) changes[key] = draft[key] === '' ? null : draft[key];
        }
        c.changes = changes;
      }
    }
    setIssue(undefined);
    const result = await flow.submit(c);
    if (result.kind === 'saved') onResult(result.result);
    if (result.kind === 'failed') {
      const e = result.error;
      if (e instanceof ApiError && e.status === 403) { flow.clearPayload(); props.onDenied({ draft, original, grade }); return; }
      if (e instanceof ApiError && [401, 404].includes(e.status)) { saved.current = true; onReload(); return; }
      if (e instanceof ApiError && e.code === 'M09_VERSION_CONFLICT') { setConflict(true); return; }
      setIssue({ text: message(e), field: e instanceof ApiError ? e.field : undefined, duplicate: e instanceof ApiError && e.code === 'M09_DUPLICATE_RATE' });
    }
  };
  const controls = <><Button onClick={() => exit.request(onCancel)} disabled={flow.inFlight}>取消</Button>
    <Button variant="contained" color={deleting ? 'error' : 'primary'} type="submit" form="compensation-rate-form"
      disabled={busy || stale || rereading}>{deleting ? '确认删除' : action === 'DEACTIVATE' ? '确认停用' : action === 'REACTIVATE' ? '确认重新启用' : '保存'}</Button></>;
  const unitLabel = creating ? moneyUnit(draft.currency, draft.frequency) : moneyUnit(detail?.currency,detail?.frequency);
  const text = (key: keyof Draft, label: string, required = false, type = 'text') => <ObjectTextField key={key} id={'rate-' + key} label={label} required={required}
    value={draft[key]} type={type} onChange={e => set(key, e.target.value)} error={issue?.field === key} helperText={issue?.field === key ? issue.text : undefined}
    sx={amountKeys.includes(key as AmountKey) ? { '& input': { textAlign:'right' } } : undefined}
    slotProps={{
      input: amountKeys.includes(key as AmountKey) && unitLabel ? { endAdornment:<InputAdornment position="end">{unitLabel}</InputAdornment> } : undefined,
      htmlInput: key === 'effectiveDate' ? { min: window?.minimum, max: window?.maximum } : { inputMode: amountKeys.includes(key as AmountKey) ? 'decimal' : undefined }
    }} />;
  const state = detail?.record;
  const previousIndex = (detail?.dates.findIndex(d => d.recordId === state?.recordId) ?? -1) - 1;
  const previous = detail?.dates[previousIndex];
  const restored = useRead(deleting && previous ? JSON.stringify([detail?.id, detail?.sourceVersion, previous.recordId]) : undefined,
    () => rateDetail(detail!.id, previous!.recordId), 'fresh');
  const options = (entries: Record<string, string>) => Object.entries(entries).map(([id, label]) => <MenuItem key={id} value={id}>{label}</MenuItem>);
  return <ObjectForm><form id="compensation-rate-form" onSubmit={save} noValidate><fieldset disabled={busy} style={{ border: 0, padding: 0, margin: 0, minWidth: 0 }}>
    {stale && <Alert severity="warning" action={<Button disabled={rereading} onClick={() => void reread()}>重新读取</Button>}>这份标准已变更，请重新读取并核对输入。</Alert>}
    {issue && <ObjectFormError id="rate-error" message={issue.text} />}
    {issue?.duplicate && existing && <Link to={ratePath + '/' + existing.id + '?' + new URLSearchParams({ back: props.returnTo })}>打开已有职级薪酬标准</Link>}
    {deleting ? <ObjectDetailSection title="撤去这条记录"><p>删除 {state?.effectiveDate} 的记录后，从该日起将沿用前一条记录{state?.endExclusive ? '，直至 ' + state.endExclusive + ' 的后续记录生效' : ''}。</p>
      {restored.data && <ObjectDetailField label="恢复适用的标准">{[statusLabels[restored.data.record.status ?? ''], rateAmounts(restored.data.record)].filter(Boolean).join(' · ')}</ObjectDetailField>}
      {restored.error && <Alert severity="error">{restored.error}</Alert>}
      <ObjectDetailField label="办理原因">{state?.reason}</ObjectDetailField></ObjectDetailSection> : <>
      <ObjectDetailSection title="生效记录"><div className="object-detail-columns">
        {writable('effectiveDate') ? text('effectiveDate', '生效日期', true, 'date') : state?.effectiveDate && <ObjectDetailField label="生效日期">{state.effectiveDate}</ObjectDetailField>}
        {writable('reason') && <ObjectTextField select required id="rate-reason" label="办理原因" value={draft.reason} onChange={e => set('reason', e.target.value)}
          error={issue?.field === 'reason'} helperText={issue?.field === 'reason' ? issue.text : undefined}>
          {[...new Set([original.reason, ...props.reasons].filter(Boolean))].map(r => <MenuItem key={r} value={r}>{r}</MenuItem>)}</ObjectTextField>}
      </div></ObjectDetailSection>
      <ObjectDetailSection title="职级与计量口径"><div className="object-detail-columns">
        {creating ? <>{writable('gradeId') && <GradeInput value={grade} onChange={g => { setGrade(g); set('gradeId', g?.id ?? ''); }} error={issue?.field === 'gradeId' ? issue.text : undefined} />}
          {writable('currency') && text('currency', '币种', true)}
          {writable('frequency') && <ObjectTextField id="rate-frequency" select label="约定频率" required value={draft.frequency} onChange={e => set('frequency', e.target.value)}>{Object.keys(props.units ?? {}).map(id => <MenuItem key={id} value={id}>{frequencies[id]?.name ?? id}</MenuItem>)}</ObjectTextField>}
          {writable('unit') && <ObjectDetailField label="计量单位">{frequencies[draft.frequency]?.unitName}</ObjectDetailField>}
          {writable('precision') && (draft.currency !== 'CNY' || draft.frequency === 'HOUR') && text('precision', '金额小数位数', true)}</>
          : <>{state?.grade && <ObjectDetailField label="职级">{gradeLabel(state)}</ObjectDetailField>}
            {detail?.unit && <ObjectDetailField label="计量口径">{[detail.currency, frequencies[detail.frequency ?? '']?.unitName].filter(Boolean).join(' / ')}</ObjectDetailField>}</>}
      </div></ObjectDetailSection>
      <ObjectDetailSection title="标准金额"><div className="object-detail-columns">{valueFields ? <>
        {writable('form') && <ObjectTextField id="rate-form" label="标准形式" select required value={draft.form} onChange={e => setForm(e.target.value)}>
          {options(formLabels)}</ObjectTextField>}
        {visibleKeys.filter(writable).map(k => text(k, amountLabels[k], k !== 'guidance', 'text'))}
      </> : state && <ObjectDetailField label="原标准">{rateAmounts(state)}</ObjectDetailField>}</div></ObjectDetailSection>
    </>}</fieldset>{host ? createPortal(controls, host) : controls}</form>
    <UnsavedChangesDialog {...exit.dialog} pending={flow.unknown} /><Dialog open={!!nextForm} onClose={() => setNextForm('')}><DialogTitle>切换标准形式</DialogTitle>
      <DialogContent>切换后需重新填写该形式的金额。</DialogContent><DialogActions><Button onClick={() => setNextForm('')}>保留当前形式</Button>
        <Button onClick={() => replaceForm(nextForm)}>切换</Button></DialogActions></Dialog></ObjectForm>;
}
