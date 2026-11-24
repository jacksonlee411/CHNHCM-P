import { useEffect, useReducer, useRef, useState } from 'react';
import { Alert, Button } from '@mui/material';
import { ApiError, sessionGuard } from '@chnhcm/api-client';
import { SubmissionFlow, ownedReference, type Pending } from '../../shared/dated/submission';
import { useSessionRecovery } from '../../shared/sessionRecovery';
import { channel, resolve, message, type Principal, type Result } from './api';

export function usePersonSubmission<T>(
  session: Principal,
  send: (v: T, id: string) => Promise<Result>,
  done: (r: Result) => void,
) {
  const [, render] = useReducer((v) => v + 1, 0),
    mounted = useRef(false),
    latest = useRef({ session, send, done });
  latest.current = { session, send, done };
  const [flow] = useState(
    () =>
      new SubmissionFlow<T, Result, Result>({
        owner: session,
        channel,
        uuid: () => crypto.randomUUID(),
        changed: render,
        current: () => {
          const same = sessionGuard();
          return () => same() && mounted.current;
        },
        send: (v, id) => latest.current.send(v, id),
        unwrap: (r) => r,
        definitive: (e) => e instanceof ApiError && e.status >= 400 && e.status < 500,
        resolve: (ref) => resolve(latest.current.session, ref.submissionId),
      }),
  );
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      flow.clearPayload();
    };
  }, [flow]);
  useEffect(
    () =>
      channel.subscribe((n) => {
        if (flow.resolved(n.reference, n.result)) latest.current.done(n.result);
      }, render),
    [flow],
  );
  return flow;
}
export function PersonRecovery({ session }: { session: Principal }) {
  const [pending, setPending] = useState<Pending[]>([]),
    [notice, setNotice] = useState(''),
    [, render] = useReducer((v) => v + 1, 0);
  const blocked = useSessionRecovery();
  useEffect(() => {
    const read = () => {
      try {
        setPending(channel.read().filter((p) => ownedReference(p, session)));
      } catch {
        setNotice('保存进度暂不可读');
      }
    };
    read();
    return channel.watch(read);
  }, [session]);
  async function confirm(ref: Pending) {
    if (!channel.lock(ref)) return;
    render();
    const current = sessionGuard();
    try {
      const r = await resolve(session, ref.submissionId);
      if (!current()) return;
      if (r.outcome === 'pending') {
        setNotice('保存结果尚未确认');
        return;
      }
      channel.forget(ref);
      channel.publish(ref, r);
      setNotice(
        {
          accepted: '原提交已保存',
          not_applied: '原提交未执行',
          rejected: '原提交已拒绝',
          no_change: '没有修改内容',
        }[r.outcome],
      );
    } catch (e) {
      if (current()) setNotice(message(e));
    } finally {
      channel.unlock(ref);
      if (current()) render();
    }
  }
  return (
    <>
      {pending.map((ref) => (
        <Alert
          key={ref.submissionId}
          severity="warning"
          action={
            <Button disabled={blocked || channel.busy(ref)} onClick={() => confirm(ref)}>
              查看保存结果
            </Button>
          }
        >
          人员办理结果待确认
        </Alert>
      ))}
      {notice && <Alert onClose={() => setNotice('')}>{notice}</Alert>}
    </>
  );
}
