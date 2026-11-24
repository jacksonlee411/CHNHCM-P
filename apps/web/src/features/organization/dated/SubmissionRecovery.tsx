import { useCallback, useEffect, useRef, useState } from 'react';
import { Alert, Button, Stack, Typography } from '@mui/material';
import { Link, useLocation } from 'react-router-dom';
import { ApiError, sessionGuard, unwrap } from '@chnhcm/api-client';
import { useSessionRecovery } from '../../../shared/sessionRecovery';
import { terminalOutcome, ownedReference, type Outcome, type Owner, type Pending } from '../../../shared/dated/submission';
import { originalResultHref, type DatedObject } from './object';

export function SubmissionRecovery<K extends string, R extends Outcome>({ definition, session }: { definition: DatedObject<K, R>; session: Owner & { csrf: string } }) {
  const sessionBlocked = useSessionRecovery();
  const channel = definition.submissions, location = useLocation(), epoch = useRef(0);
  const [pending, setPending] = useState<Pending[]>([]), [result, setResult] = useState<R>(), [error, setError] = useState('');
  const refresh = useCallback(() => {
    try { const entries = channel.read().filter(p => ownedReference(p, session)); setPending([...entries]); if (!entries.length) setError(''); }
    catch { setError('暂时无法读取上次保存进度'); }
  }, [channel, session.instanceId, session.subjectId]);
  useEffect(() => { refresh(); return channel.watch(refresh); }, [refresh]);
  useEffect(() => { epoch.current++; setResult(undefined); return () => { epoch.current++; }; }, [definition, session.instanceId, session.subjectId, location.pathname]);
  useEffect(() => channel.subscribe(notice => {
    if (ownedReference(notice.reference, session)) { setResult(notice.result); setError(''); }
  }, () => {}), [channel, session.instanceId, session.subjectId]);
  useEffect(() => {
    const dismiss = () => setResult(undefined);
    window.addEventListener('chnhcm-edit-started', dismiss);
    return () => window.removeEventListener('chnhcm-edit-started', dismiss);
  }, []);
  const resolve = async (reference: Pending) => {
    if (sessionBlocked || !channel.lock(reference)) return;
    const own = ++epoch.current, same = sessionGuard();
    const current = () => own === epoch.current && same() && channel.has(reference);
    setError('');
    try {
      const response = await definition.resolve(session, reference);
      if (!current()) return;
      const value = unwrap(response);
      if (value.submissionId !== reference.submissionId || (value.outcome !== 'pending' && !terminalOutcome(value.outcome))) throw new Error('暂时无法确认保存结果');
      if (value.outcome !== 'pending') { channel.forget(reference); channel.publish(reference, value); }
      setResult(value);
    } catch (error) {
      if (!same() || own !== epoch.current) return;
      // A concurrent exact retry can have settled and removed this reference.
      // Its late failed resolve must not clear the next form or restore an error.
      try { if (!channel.has(reference)) return; }
      catch { setError('暂时无法读取上次保存进度'); return; }
      setError(error instanceof ApiError ? error.message : '请求失败，请重试');
      if (error instanceof ApiError && [403, 404].includes(error.status) && error.code !== 'AUTH_REQUEST_VALIDATION_FAILED') { setResult(undefined); channel.deny(reference); }
    } finally { channel.unlock(reference); }
  };
  if (!pending.length && !result && !error) return null;
  const view = result && definition.resultView(result);
  const href = view && originalResultHref(definition, view, location.pathname === definition.basePath ? location.pathname + location.search : new URLSearchParams(location.search).get('back'));
  return <Stack spacing={1} sx={{ mb: 2 }} role="region" aria-label={`${definition.label ?? '对象'}提交结果`}>
    {pending.map(p => <Alert key={p.submissionId} severity={error ? "error" : channel.busy(p) ? "info" : "warning"} action={<Button disabled={sessionBlocked || channel.busy(p)} onClick={() => void resolve(p)}>查看保存结果</Button>}>{definition.label ? `${definition.label}：` : ''}{error || (channel.phase(p) === 'saving' ? '正在保存…' : channel.busy(p) ? '正在确认保存结果…' : '暂时无法确认保存结果')}</Alert>)}
    {view && view.outcome !== 'pending' && <Alert severity={view.outcome === 'accepted' ? 'success' : view.outcome === 'no_change' ? 'info' : 'warning'}>
      <Typography component="span" variant="body2">{view.outcome === 'accepted' ? '已保存' : view.outcome === 'no_change' ? '没有修改内容' : '未保存'}</Typography>
      {view.lines.map((line, i) => <Typography key={i}>{line}</Typography>)}
      {href && <Button component={Link} to={href}>查看保存结果</Button>}
    </Alert>}
    {error && !pending.length && <Alert severity="error">{error}</Alert>}
  </Stack>;
}
