import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { Alert, Button } from '@mui/material';
import { sessionGuard, type Models } from '@chnhcm/api-client';
import { ownedReference, type Pending } from '../../shared/dated/submission';
import { useSessionRecovery } from '../../shared/sessionRecovery';
import { quotaResultLinks, quotaSavedNotice, receipts, recoverStoredQuotaReferences, resolveQuota } from './quotaApi';

export function QuotaRecovery({ session }: { session: Models['Principal'] }) {
  const [pending, setPending] = useState<Pending[]>([]),
    [notice, setNotice] = useState(''),
    [links, setLinks] = useState<ReturnType<typeof quotaResultLinks>>([]),
    blocked = useSessionRecovery();
  useEffect(() => {
    const read = () => {
      try {
        setPending(receipts.read().filter((r) => ownedReference(r, session)));
      } catch {
        setNotice('暂时无法读取核编保存进度');
      }
    };
    try {
      recoverStoredQuotaReferences();
    } catch {
      setNotice('暂时无法读取原核编保存进度');
    }
    read();
    return receipts.watch(read);
  }, [session]);
  async function confirm(ref: Pending) {
    if (!receipts.lock(ref)) return;
    const same = sessionGuard();
    try {
      const r = await resolveQuota(session.csrf, ref);
      if (!same()) return;
      receipts.forget(ref);
      receipts.publish(ref, r);
      setLinks(quotaResultLinks(r));
      setNotice(
        r.outcome === 'accepted' || r.outcome === 'no_change'
          ? quotaSavedNotice(r)
          : r.outcome === 'not_applied'
            ? '原提交未生效，可重新办理'
            : (r.failure?.message ?? '原提交已拒绝'),
      );
    } catch (e) {
      if (same()) {
        setLinks([]);
        setNotice(e instanceof Error ? e.message : '暂时无法查回');
      }
    } finally {
      receipts.unlock(ref);
    }
  }
  return (
    <>
      {pending
        .filter((r) => receipts.phase(r) !== 'saving')
        .map((r) => (
          <Alert
            key={r.submissionId}
            severity="warning"
            action={
              <Button disabled={blocked || receipts.busy(r)} onClick={() => void confirm(r)}>
                查看保存结果
              </Button>
            }
          >
            有一笔核编保存结果尚未确认。
          </Alert>
        ))}
      {notice && (
        <Alert
          onClose={() => {
            setNotice('');
            setLinks([]);
          }}
        >
          {notice}{' '}
          {links.map((link) => (
            <Button key={link.href} component={Link} to={link.href}>
              {link.label}
            </Button>
          ))}
        </Alert>
      )}
    </>
  );
}
