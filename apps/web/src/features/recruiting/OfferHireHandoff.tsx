import { useEffect, useRef, useState } from 'react';
import { Alert, Button, Stack } from '@mui/material';
import { Link } from 'react-router-dom';
import { api, unwrap, type Models } from '@chnhcm/api-client';
import { useRead, path as people } from '../personnel/api';
import { useRuleSubmission } from '../readiness/useRuleSubmission';

const missingNames: Record<string, string> = {
  document: '适用的身份证件',
  expiryDate: '证件期限',
  inspection: '原件查验结论',
};

export function OfferHireHandoff({
  item,
  session,
  historical,
  refresh,
}: {
  item: Models['RecruitingOffer'];
  session: Models['Principal'];
  historical: boolean;
  refresh: () => void;
}) {
  const header = { 'X-CSRF-Token': session.csrf };
  const view = useRead(
    JSON.stringify([session.subjectId, item.id, item.version]),
    () =>
      api
        .GET('/api/v1/recruiting/offers/{id}/hire-handoff', {
          params: {
            path: { id: item.id },
            query: { sourceVersion: item.version },
          },
        })
        .then(unwrap),
  );
  const [error, setError] = useState(''),
    [notice, setNotice] = useState('');
  const failure = useRef<HTMLDivElement>(null);
  const send = useRuleSubmission<
    Models['OfferHireCommand'],
    Models['HireReceipt']
  >('offer-hire-' + item.id, session, {
    send: (body, key) =>
      api
        .POST('/api/v1/recruiting/offers/{id}/hire', {
          params: {
            path: { id: item.id },
            header: { ...header, 'Idempotency-Key': key },
          },
          body,
        })
        .then(unwrap),
    resolve: (sub) =>
      api
        .POST('/api/v1/employment-actions/submissions/{sub}/resolve', {
          params: { path: { sub }, header },
        })
        .then(unwrap),
    finished: (receipt) => {
      setError('');
      setNotice(
        receipt.outcome === 'accepted' ? '入职已办理。' : '原提交未办理。',
      );
      refresh();
      view.refresh();
    },
    failed: (cause) => {
      setError(cause instanceof Error ? cause.message : '暂时无法确认入职结果');
      view.refresh();
    },
  });
  useEffect(() => {
    if (error) failure.current?.focus();
  }, [error]);
  const state = view.data;
  return (
    <Stack alignItems="flex-start" spacing={1}>
      {(error || view.error || send.storageError) && (
        <Alert ref={failure} tabIndex={-1} severity="error">
          {error || view.error || '无法保存提交号，请恢复浏览器存储后办理。'}
        </Alert>
      )}
      {notice && <Alert severity="info">{notice}</Alert>}
      {!!state?.missing?.length && (
        <Alert severity="info">
          入职资料待补齐：
          {state.missing.map((k) => missingNames[k] ?? k).join('、')}。{' '}
          <Link to={people + '/' + item.person}>查看人员档案</Link>
        </Alert>
      )}
      {!historical && state?.canHire && state.binding && (
        <Button
          variant="contained"
          disabled={send.busy}
          onClick={() => {
            setNotice('');
            setError('');
            send.start();
            void send.submit(state.binding!);
          }}
        >
          按录用办理入职
        </Button>
      )}
      {state?.assignment && (
        <Button
          component={Link}
          to={
            people +
            '/' +
            item.person +
            '/assignments/' +
            state.assignment +
            '?businessAt=' +
            item.arrival +
            '&assignmentDay=' +
            item.arrival
          }
        >
          查看入职任职
        </Button>
      )}
      {send.pending.map((ref) => (
        <Button key={ref.submissionId} onClick={() => void send.recover(ref)}>
          查回入职提交
        </Button>
      ))}
    </Stack>
  );
}
