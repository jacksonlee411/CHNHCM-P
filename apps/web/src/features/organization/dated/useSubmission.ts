import { useSessionRecovery } from '../../../shared/sessionRecovery';
import { useEffect, useReducer, useRef, useState } from 'react';
import { ApiError, sessionGuard, unwrap } from '@chnhcm/api-client';
import { SubmissionFlow, ownedReference, type Outcome, type Owner, type Pending, type SubmissionChannel } from '../../../shared/dated/submission';

export type ApiResponse<R> = Parameters<typeof unwrap<R>>[0];
export function usePending<K extends string, R extends Outcome>(channel: SubmissionChannel<K, R>, owner: Owner) {
  const [blocked, setBlocked] = useState(true);
  useEffect(() => {
    const refresh = () => { try { setBlocked(channel.read().some(p => ownedReference(p, owner))); } catch { setBlocked(true); } };
    refresh(); return channel.watch(refresh);
  }, [channel, owner.instanceId, owner.subjectId]);
  return blocked;
}
export function useSubmission<K extends string, C, R extends Outcome>(options: {
  channel: SubmissionChannel<K, R>; owner: Owner;
  send: (command: C, id: string) => Promise<ApiResponse<R>>;
  resolve: (reference: Pending) => Promise<ApiResponse<R>>;
  onResolved: (result: R) => void; onDenied: (reference: Pending) => void;
}) {
  const optionsRef = useRef(options); optionsRef.current = options;
  const [, render] = useReducer(n => n + 1, 0);
  const mounted = useRef(false);
  const identity = `${options.channel.kind}/${options.owner.instanceId}/${options.owner.subjectId}`;
  const identityRef = useRef(identity); identityRef.current = identity;
  const holder = useRef<{ identity: string; flow: SubmissionFlow<C, R, ApiResponse<R>> } | undefined>(undefined);
  if (!holder.current || holder.current.identity !== identity) holder.current = { identity, flow: new SubmissionFlow({
    owner: options.owner, channel: options.channel, uuid: () => crypto.randomUUID(), changed: render,
    current: () => { const same = sessionGuard(); return () => mounted.current && identityRef.current === identity && same(); },
    send: (command, id) => optionsRef.current.send(command, id), unwrap,
    resolve: async reference => {
      const same = sessionGuard();
      try { return unwrap(await optionsRef.current.resolve(reference)); }
      catch (error) {
        if (mounted.current && identityRef.current === identity && same() && error instanceof ApiError && [403, 404].includes(error.status) && error.code !== 'AUTH_REQUEST_VALIDATION_FAILED') optionsRef.current.onDenied(reference);
        throw error;
      }
    },
    definitive: error => error instanceof ApiError && error.status >= 400 && error.status < 500,
  }) };
  const flow = holder.current.flow;
  useEffect(() => { mounted.current = true; return () => { mounted.current = false; }; }, []);
  useEffect(() => {
    const same = sessionGuard();
    return options.channel.subscribe(notice => {
      if (mounted.current && same() && flow.resolved(notice.reference, notice.result)) optionsRef.current.onResolved(notice.result);
    }, reference => {
      if (mounted.current && same() && ownedReference(reference, options.owner) && (!flow.submittedId || flow.submittedId === reference.submissionId)) optionsRef.current.onDenied(reference);
    });
  }, [flow, identity]);
  useEffect(() => options.channel.watch(() => render()), [options.channel]);
  return { flow, pending: usePending(options.channel, options.owner), sessionBlocked: useSessionRecovery() };
}
