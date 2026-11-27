import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Alert, Button, Stack } from '@mui/material';
import { ApiError, sessionGuard } from '@chnhcm/api-client';
import { useSessionRecovery } from '../../shared/sessionRecovery';
import { useObjectSubmission } from '../../shared/dated/useObjectSubmission';
import { ownedReference, type Pending } from '../../shared/dated/submission';
import * as staff from './api';
import {arrangementChannel,resolveArrangement,structurePath} from './structure';
export function useStaffingSubmission(k: staff.Kind, session: staff.Principal, id: string, onSaved: (r: staff.Result) => void) {
    return useObjectSubmission(session, staff.channels[k], (body:staff.Input,sub:string)=>staff.send(k,session,id,body,sub), ref=>staff.resolve(k,session,ref), onSaved, e=>e instanceof staff.RejectedSubmission);
}
export { useObjectSubmission } from '../../shared/dated/useObjectSubmission';
export function StaffingRecovery({ session }: {
    session: staff.Principal;
}) {
    return <>{([...staff.kinds, 'positionhierarchy'] as const).map(k => <RecoveryItem key={`${k}:${session.instanceId}:${session.subjectId}`} kind={k} session={session}/>)}</>;
}
function RecoveryItem({ kind: k, session }: {
    kind: staff.Kind|'positionhierarchy';
    session: staff.Principal;
}) {
    const channel=k==='positionhierarchy'?arrangementChannel:staff.channels[k];
    const navigate = useNavigate();
    const sessionBlocked = useSessionRecovery();
    const [pending, setPending] = useState<Pending[]>([]), [message, setMessage] = useState(''), [busy, setBusy] = useState(false);
    useEffect(() => { const update = () => { try {
        setPending(channel.read().filter(p => ownedReference(p, session)));
    }
    catch {
        setMessage('保存进度暂不可读，请重试。');
    } }; update(); return channel.watch(update); }, [k, session.instanceId, session.subjectId]);
    const confirm = async (ref: Pending) => { if (sessionBlocked || !channel.lock(ref))
        return; const same = sessionGuard(), from = window.location.href; setBusy(true); try {
        const r = k==='positionhierarchy' ? await resolveArrangement(session,ref) : await staff.resolve(k, session, ref);
        if (!same())
            return;
        if (r.outcome === 'pending') {
            setMessage('保存结果尚未确认，请稍后再查。');
            return;
        }
        channel.forget(ref);
        if (channel.publish(ref,r) && k==='positionhierarchy') {setMessage('');return}
        setMessage(r.outcome === 'accepted' ? '原提交已保存。' : r.outcome === 'no_change' ? '原提交没有资料变化。' : r.outcome === 'not_applied' ? '原提交未生效，可核对输入后重新保存。' : k==='positionhierarchy' ? '原提交已拒绝。' : r.failure?.message ?? '原提交已拒绝。');
        if (k === 'positionhierarchy' && window.location.pathname !== structurePath) {
            // A recovered receipt carries no verified child or date.
            // Keep its confirmed outcome visible even if the structure cannot be read.
            const access = await staff.context('position').catch(() => undefined);
            if (same() && window.location.href === from && access?.capabilities.actions.positionRelations) {
                navigate(structurePath);
            }
        }
    }
    catch (e) {
        if (same())
            setMessage(e instanceof ApiError ? e.message : '暂时无法确认保存结果。');
    }
    finally {
        channel.unlock(ref);
        if (same())
            setBusy(false);
    } };
    if (!pending.length && !message)
        return null;
    return <Stack spacing={1}>{pending.map(ref => <Alert key={ref.submissionId} severity="warning" action={<Button disabled={sessionBlocked || busy || channel.busy(ref)} onClick={() => void confirm(ref)}>查看保存结果</Button>}>{k==='positionhierarchy'?'岗位关系':staff.labels[k]}有一笔保存结果尚未确认。</Alert>)}{message && <Alert onClose={() => setMessage('')} severity="info">{message}</Alert>}</Stack>;
}
