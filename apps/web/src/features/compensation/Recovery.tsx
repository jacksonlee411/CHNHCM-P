import { useEffect, useState } from 'react';
import { Alert, Button, Stack } from '@mui/material';
import { sessionGuard } from '@chnhcm/api-client';
import { ownedReference, type Pending } from '../../shared/dated/submission';
import { useSessionRecovery } from '../../shared/sessionRecovery';
import { message } from '../../shared/useRead';
import { planChannel, planTitle, resolvePlan } from './plans';
import { rateChannel, rateTitle, resolveRate, type Principal } from './rates';

export function CompensationRecovery({ session, kind }: { session: Principal; kind:'rate'|'plan' }) {
  const channel = kind === 'rate' ? rateChannel : planChannel, resolveReceipt = kind === 'rate' ? resolveRate : resolvePlan;
  const [pending, setPending] = useState<Pending[]>([]);
  const [notice, setNotice] = useState('');
  const blocked = useSessionRecovery();
  useEffect(() => {
    const update = () => {
      try { setPending(channel.read().filter(ref => ownedReference(ref, session))); }
      catch { setNotice('暂时无法读取保存进度'); }
    };
    update();
    return channel.watch(update);
  }, [session.instanceId, session.subjectId, channel]);
  const resolve = async (ref: Pending) => {
    if (blocked || !channel.lock(ref)) return;
    const current = sessionGuard();
    try {
      const r = await resolveReceipt(session, ref);
      if (!current()) return;
      channel.forget(ref);
      if (!channel.publish(ref, r)) setNotice(r.message ?? ({ accepted: '原提交已保存', no_change: '资料没有变化', rejected: '原提交被拒绝', not_applied: '原提交未执行' }[r.outcome]));
    } catch (err) { if (current()) setNotice(message(err)); }
    finally { channel.unlock(ref); }
  };
  return <Stack spacing={1}>{pending.map(ref => <Alert severity="warning" key={ref.submissionId}
    action={<Button disabled={blocked || channel.busy(ref)} onClick={() => void resolve(ref)}>查看保存结果</Button>}>{kind === 'rate' ? rateTitle : planTitle}有一笔保存结果尚未确认。</Alert>)}
    {notice && <Alert severity="info" onClose={() => setNotice('')}>{notice}</Alert>}</Stack>;
}
