import { useCallback, useEffect, useMemo, useReducer, useRef, useState } from 'react';
import { ApiError, sessionGuard, type Models } from '@chnhcm/api-client';
import {
  ownedReference,
  submissionChannel,
  terminalOutcome,
  SubmissionFlow,
  type Pending,
  type Outcome,
} from '../../shared/dated/submission';
import { useSessionRecovery } from '../../shared/sessionRecovery';

type Receipt = Models['RuleSaved'];
type Ports<C, R extends Outcome> = {
  send: (cmd: C, id: string) => Promise<R>;
  resolve: (id: string) => Promise<R>;
  finished: (result: R, recovery?: 'current' | 'other') => void;
  failed: (error: unknown) => void;
};

export function useRuleSubmission<C, R extends Outcome = Receipt>(
  scope: string,
  session: Models['Principal'],
  ports: Ports<C, R>,
) {
  const channel = useMemo(() => submissionChannel<string, R>(scope, 'chnhcm-pending-' + scope), [scope]);
  const latest = useRef(ports);
  latest.current = ports;
  const mounted = useRef(false);
  const flow = useRef<SubmissionFlow<C, R, R> | undefined>(undefined);
  const [pending, setPending] = useState<Pending[]>([]);
  const [storageError, setStorageError] = useState(false);
  const [, render] = useReducer((n: number) => n + 1, 0);
  const sessionBlocked = useSessionRecovery();
  useEffect(() => {
    mounted.current = true;
    const read = () => {
      try {
        setPending(channel.read().filter((ref) => ownedReference(ref, session)));
        setStorageError(false);
      } catch {
        setStorageError(true);
      }
    };
    read();
    const watch = channel.watch(read);
    const sub = channel.subscribe(
      (notice) => {
        if (!ownedReference(notice.reference, session)) return false;
        const current = flow.current?.resolved(notice.reference, notice.result);
        latest.current.finished(notice.result, current ? 'current' : 'other');
        return true;
      },
      () => {},
    );
    return () => {
      mounted.current = false;
      watch();
      sub();
      flow.current?.clearPayload();
    };
  }, [channel, session.instanceId, session.subjectId]);

  const recover = async (ref: Pending) => {
    if (sessionBlocked || !channel.lock(ref)) return;
    const same = sessionGuard();
    try {
      const result = await latest.current.resolve(ref.submissionId);
      if (!same() || !mounted.current) return;
      if (result.submissionId !== ref.submissionId || !terminalOutcome(result.outcome))
        throw new Error('暂时无法确认原提交结果');
      channel.forget(ref);
      channel.publish(ref, result);
    } catch (e) {
      if (same() && mounted.current) latest.current.failed(e);
    } finally {
      channel.unlock(ref);
    }
  };
  const start = useCallback(() => {
    flow.current = new SubmissionFlow({
      owner: session,
      channel,
      uuid: () => crypto.randomUUID(),
      changed: render,
      current: () => {
        const same = sessionGuard();
        return () => mounted.current && same();
      },
      send: (cmd, id) => latest.current.send(cmd, id),
      unwrap: (r) => r,
      resolve: (ref) => latest.current.resolve(ref.submissionId),
      definitive: (e) => e instanceof ApiError && [400, 401, 403, 404, 409].includes(e.status),
    });
  }, [session, channel]);
  const submit = async (cmd: C) => {
    if (!flow.current || sessionBlocked || storageError) return;
    const result = await flow.current.submit(cmd);
    if (result.kind === 'saved') latest.current.finished(result.result);
    if (result.kind === 'failed') latest.current.failed(result.error);
  };
  return {
    start,
    submit,
    recover,
    pending,
    storageError,
    busy: sessionBlocked || storageError || !!flow.current?.inFlight || pending.length > 0,
  };
}
