import { useEffect, useMemo, useRef, useState } from 'react';
import { Alert, Autocomplete, Box, Button, CircularProgress, TextField, Typography } from '@mui/material';
import { api, ApiError, sessionGuard, unwrap, type Models } from '@chnhcm/api-client';
import { ObjectDetailField, ObjectDetailSection, businessDate } from '@chnhcm/ui';
import { useOwnedRead } from './dated/useOwnedRead';
import type { DataEditorProps, RecordView } from './dated/presentation';

type Choice = { id: string; code?: string | null; name?: string | null };
export const referenceLabel = (v?: Choice | null) => v ? [v.code, v.name].filter(Boolean).join(' · ') || '名称无读取权限' : '';
function ReferenceField({ kind, props }: { kind: 'bu' | 'parent'; props: DataEditorProps }) {
  const requestContext = props.context && !('kind' in props.context) ? props.context : undefined;
  const key = kind === 'bu' ? 'managementBuId' : 'parentDepartmentId';
  const label = kind === 'bu' ? '管理业务单元' : '行政上级';
  const [open, setOpen] = useState(false), [search, setSearch] = useState(''), [page, setPage] = useState(1), [retry, setRetry] = useState(0);
  const [selected, setSelected] = useState<Choice>(), [response, setResponse] = useState<{ key: string; value?: Models['DepartmentReferences'] | Models['DepartmentParents']; error?: string }>();
  const cursors = useRef<string[]>(['']);
  const denied = useRef<{ context: string; error: string } | undefined>(undefined);
  const context = JSON.stringify([props.session.instanceId, props.session.subjectId, props.context, kind, retry]);
  const identity = JSON.stringify([props.session.instanceId, props.session.subjectId, props.context, kind, open, search, page, retry, props.values[key]]);
  const current = useRef(identity); current.current = identity;
  const value = props.values[key] ?? '';
  const original = kind === 'bu' ? props.baseline?.references?.managementBuId : props.baseline?.references?.parentDepartmentId;
  const resolved = response?.key === identity && response.value?.selectedReference?.id === value ? response.value.selectedReference : undefined;
  const selection = value ? resolved ?? (selected?.id === value ? selected : original?.id === value ? original : { id: value }) : null;
  // Equal server projections must not reset the autocomplete's search input.
  const chosen = useMemo<Choice | null>(() => selection ? { id: selection.id, code: selection.code, name: selection.name } : null,
    [selection?.id, selection?.code, selection?.name]);
  const data = response?.key === identity ? response.value : undefined, error = response?.key === identity ? response.error : undefined;
  const writable = props.writes.includes(key), readable = props.fields.includes(key);
  const filter = requestContext?.mode === 'filter';
  const enabled = readable && writable && !props.disabled && !!requestContext;
  useEffect(() => { cursors.current = ['']; setPage(1); }, [search, JSON.stringify(props.context), retry]);
  useEffect(() => { if (!readable) { setSelected(undefined); setResponse(undefined); setSearch(''); setOpen(false); } }, [readable]);
  useEffect(() => {
    if ((!open && !value) || !enabled) return;
    // A detail refresh toggles enabled. Keep its existing authorization rejection
    // until the user retries or the query context changes, rather than looping.
    if (denied.current?.context === context) { setResponse({ key: identity, error: denied.current.error }); return; }
    const abort = new AbortController(), same = sessionGuard();
    let active = true;
    setResponse(undefined);
    const timer = setTimeout(() => {
      const body = { context: requestContext!, selectedId: value || undefined, q: open ? search : value, page: open ? page : 1, pageSize: 20 };
      const request = kind === 'bu'
        ? api.POST('/api/v1/departments/references/business-units', { params: { header: { 'X-CSRF-Token': props.session.csrf } }, body, signal: abort.signal })
        : api.POST('/api/v1/departments/references/parents', { params: { header: { 'X-CSRF-Token': props.session.csrf } }, body: { context: body.context, selectedId: body.selectedId, q: body.q, pageSize: 20, cursor: open ? cursors.current[page - 1] || undefined : undefined }, signal: abort.signal });
      void request.then(value => unwrap<Models['DepartmentReferences'] | Models['DepartmentParents']>(value)).then(result => {
        if (!active || !same() || current.current !== identity) return;
        denied.current = undefined;
        if ('hasMore' in result && result.hasMore && result.nextCursor) cursors.current[page] = result.nextCursor;
        setResponse({ key: identity, value: result });
        const reference = result.selectedReference;
        if (reference?.id === value) {
          setSelected(reference);
          if (selection && (selection.name && !reference.name || selection.code && !reference.code)) props.onReferenceRestricted?.();
        }
      }).catch(e => {
        if (!active || !same() || current.current !== identity) return;
        setResponse({ key: identity, error: e instanceof ApiError ? e.message : '候选读取失败' });
        if (e instanceof ApiError && [401, 403, 404].includes(e.status) && e.code !== 'AUTH_REQUEST_VALIDATION_FAILED') { denied.current = { context, error: e.message }; setSelected(undefined); props.onReadDenied?.(); }
      });
    }, 200);
    return () => { active = false; clearTimeout(timer); abort.abort(); };
  }, [identity, open, enabled]);
  if (!readable) return <ObjectDetailField label={label}>无读取权限</ObjectDetailField>;
  const control = <Box sx={{ minWidth: 0 }}>
    <Autocomplete<Choice> id={`${props.prefix}-${key}`} open={open && enabled} onOpen={() => { cursors.current = ['']; setSearch(''); setPage(1); setOpen(true); }} onClose={() => setOpen(false)}
      disabled={!enabled} options={data?.items ?? []} value={chosen} filterOptions={x => x} getOptionLabel={referenceLabel} isOptionEqualToValue={(a, b) => a.id === b.id}
      onInputChange={(_, text, reason) => { if (reason === 'input') { setSearch(text); setPage(1); } }}
      onChange={(_, option) => { setSelected(option ?? undefined); props.onChange(key, option?.id ?? ''); }}
      loading={open && !data && !error} loadingText="正在读取候选…"
      noOptionsText={error ? <Box>{error}<Button size="small" onMouseDown={e => e.preventDefault()} onClick={() => setRetry(n => n + 1)}>重试读取候选</Button></Box> : (data?.applicable === false ? '此操作不适用关系选择' : '没有符合条件的候选')}
      clearText={kind === 'parent' ? '清除行政上级，设为企业根部门' : '清除管理业务单元'} openText="选择" closeText="关闭候选"
      renderInput={params => <TextField {...params} label={filter ? label : undefined} size="small" required={!filter && kind === 'bu'}
        error={props.fieldError.replace('changes.', '') === key || !!error}
        helperText={props.fieldError.replace('changes.', '') === key ? props.error : error || (kind === 'parent' && !value ? '企业根部门（无行政上级）' : undefined)}
        slotProps={{ input: { ...params.InputProps, endAdornment: <>{open && !data && !error && <CircularProgress size={16} />}{params.InputProps.endAdornment}</> } }} />}
    />
    {open && data && ('count' in data ? data.count > 20 : data.hasMore || page > 1) && <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
      <Button size="small" disabled={page <= 1} onMouseDown={e => e.preventDefault()} onClick={() => setPage(p => p - 1)}>上一页候选</Button>
      <Typography variant="caption">{'count' in data ? `第${data.page}页，共${data.count}项` : `第${page}页`}</Typography>
      <Button size="small" disabled={'count' in data ? page * 20 >= data.count : !data.hasMore} onMouseDown={e => e.preventDefault()} onClick={() => setPage(p => p + 1)}>下一页候选</Button>
    </Box>}

  </Box>;
  return filter ? control : <ObjectDetailField label={label} inputId={`${props.prefix}-${key}`} required={kind === 'bu'}>{control}</ObjectDetailField>;
}
export function DepartmentRelations(props: DataEditorProps) {
  return <><ReferenceField kind="bu" props={props} /><ReferenceField kind="parent" props={props} /></>;
}
function Reference({ value }: { value?: Models['DepartmentReference'] | null }) {
  if (!value) return <>无读取权限</>;
  return <>{referenceLabel(value)}</>;
}
export function DepartmentPath({ path }: { path?: Models['DepartmentPath'] }) {
  if (!path || path.pathMode === 'restricted') return <>无读取权限</>;
  if (path.pathMode === 'no_fact') return <>暂无适用路径</>;
  const segments = path.pathSegments;
  return <Box className="department-path" sx={{ overflowWrap: 'anywhere' }}>
    {segments?.length ? segments.map((s, index) => <Box key={`${s.start}-${index}`} sx={{ mb: segments.length > 1 ? 1 : 0 }}>
      {segments.length > 1 && <Typography variant="body2" color="text.secondary">{s.endInclusive ? `${businessDate(s.start)} 至 ${businessDate(s.endInclusive)}` : `自 ${businessDate(s.start)} 起`}</Typography>}
      <span>{s.path}</span>
    </Box>) : path.departmentPath}
  </Box>;
}
export function DepartmentRelationsView({ record: r, readFields, path }: { record: RecordView | null; readFields: string[]; path: Models['DepartmentPath'] }) {
  if (!r) return null;
  return <ObjectDetailSection title="组织关系">
    <ObjectDetailField label="管理业务单元">{readFields.includes('managementBuId') ? <Reference value={r.references?.managementBuId} /> : '无读取权限'}</ObjectDetailField>
    <ObjectDetailField label="行政上级">{!readFields.includes('parentDepartmentId') ? '无读取权限' : !r.values.parentDepartmentId ? '企业根部门（无行政上级）' : <Reference value={r.references?.parentDepartmentId} />}</ObjectDetailField>
    <ObjectDetailField label="部门路径"><DepartmentPath path={path} /></ObjectDetailField>
  </ObjectDetailSection>;
}

export function DepartmentBUFilter({ value, onChange }: { value: string; onChange: (value: string) => void }) {
  const [retry, setRetry] = useState(0);
  const [session, setSession] = useState<Models['Principal']>(), [error, setError] = useState('');
  useOwnedRead(`department-filter-session:${retry}`, { request: signal => api.GET('/api/v1/session', { signal }), start: () => { setSession(undefined); setError(''); }, accept: setSession, reject: e => setError(e instanceof ApiError ? e.message : '会话读取失败') });
  if (error) return <Alert severity="error" action={<Button onClick={() => setRetry(n => n + 1)}>重试</Button>}>{error}</Alert>;
  if (!session) return <CircularProgress size={20} />;
  return <Box sx={{ flex: 1, minWidth: 0 }}><ReferenceField kind="bu" props={{ session, values: { managementBuId: value }, fields: ['managementBuId'], writes: ['managementBuId'], disabled: false, prefix: 'department-filter', fieldError: '', error: '', context: { mode: 'filter' }, onChange: (_, v) => onChange(v) }} /></Box>;
}
