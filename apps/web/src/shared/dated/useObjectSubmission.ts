import { useEffect, useReducer, useRef } from 'react';
import { sessionGuard } from '@chnhcm/api-client';
import { useSessionRecovery } from '../sessionRecovery';
import { SubmissionFlow, ownedReference, type Owner, type Outcome, type Pending, type SubmissionChannel } from './submission';

export function useObjectSubmission<C,R extends Outcome>(session:Owner, channel:SubmissionChannel<string,R>,
    send:(body:C,id:string)=>Promise<R>, resolve:(ref:Pending)=>Promise<R>,
    onSaved:(result:R)=>void, definitive:(error:unknown)=>boolean){
    const [,render]=useReducer(n=>n+1,0), mounted=useRef(false);
    const latest=useRef({send,resolve,onSaved,definitive}); latest.current={send,resolve,onSaved,definitive};
    const flow=useRef<SubmissionFlow<C,R,R> | undefined>(undefined);
    if (!flow.current) flow.current=new SubmissionFlow({owner:session,channel,uuid:()=>crypto.randomUUID(),changed:render,
      current:()=>{const same=sessionGuard();return ()=>mounted.current && same()},
      send:(input,id)=>latest.current.send(input,id),resolve:ref=>latest.current.resolve(ref),unwrap:r=>r,definitive:e=>latest.current.definitive(e)});
    const f = flow.current;
    useEffect(() => { mounted.current = true; return () => { mounted.current = false; f.clearPayload(); }; }, [f]);
    useEffect(() => channel.watch(render), [channel]);
    useEffect(() => channel.subscribe(notice => { if (f.resolved(notice.reference, notice.result)) {
        latest.current.onSaved(notice.result); return true; } }, () => render()), [channel, f]);
    let pending = true;
    try {
        pending = channel.read().some(p => ownedReference(p, session));
    }
    catch { /* fail closed */ }
    return { flow: f, pending, sessionBlocked: useSessionRecovery() };
}
