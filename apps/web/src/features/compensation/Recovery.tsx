import { useEffect, useState } from 'react';
import { Alert, Button, Stack } from '@mui/material';
import { sessionGuard } from '@chnhcm/api-client';
import { ownedReference, type Pending } from '../../shared/dated/submission';
import { useSessionRecovery } from '../../shared/sessionRecovery';
import { message } from '../../shared/useRead';
import { rateChannel, resolveRate, type Principal } from './rates';

export function RateRecovery({ session }: { session: Principal }) {
  const [pending, setPending] = useState<Pending[]>([]);
  const [notice, setNotice] = useState('');
  const blocked = useSessionRecovery();
  useEffect(() => {
    const update = () => {
      try { setPending(rateChannel.read().filter(ref => ownedReference(ref, session))); }
      catch { setNotice('暂时无法读取保存进度'); }
    };
    update();
    return rateChannel.watch(update);
  }, [session.instanceId, session.subjectId]);
  const resolve = async (ref: Pending) => {
    if (blocked || !rateChannel.lock(ref)) return;
    const current = sessionGuard();
    try {
      const r = await resolveRate(session, ref);
      if (!current()) return;
      rateChannel.forget(ref);
      if (!rateChannel.publish(ref, r)) setNotice(r.message ?? ({ accepted: '原提交已保存', no_change: '资料没有变化', rejected: '原提交被拒绝', not_applied: '原提交未执行' }[r.outcome]));
    } catch (err) { if (current()) setNotice(message(err)); }
    finally { rateChannel.unlock(ref); }
  };
  return <Stack spacing={1}>{pending.map(ref => <Alert severity="warning" key={ref.submissionId}
    action={<Button disabled={blocked || rateChannel.busy(ref)} onClick={() => void resolve(ref)}>查看保存结果</Button>}>职级薪酬标准有一笔保存结果尚未确认。</Alert>)}
    {notice && <Alert severity="info" onClose={() => setNotice('')}>{notice}</Alert>}</Stack>;
}
