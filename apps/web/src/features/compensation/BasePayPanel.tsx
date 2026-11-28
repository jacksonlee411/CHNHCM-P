import { useEffect, useLayoutEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { useSearchParams } from 'react-router-dom';
import {
  Pagination,
  Alert,
  Button,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import {
  ObjectDetailSection,
  ObjectDetailField,
  ObjectTextField,
  ObjectForm,
  HistoryNotice,
  recordedTime,
} from '@chnhcm/ui';
import type { Models } from '@chnhcm/api-client';
import { useRead } from '../../shared/useRead';
import { awardLabels, awardOriginal, payDetail, type Award } from './basePay';
import { money } from './rates';
import { PayBasisFacts } from './BasePayFacts';
import { BasePayEditor } from './BasePayEditor';

export function BasePayPanel({
  assignment,
  day,
  session,
  host,
  onDenied,
}: {
  assignment: string;
  day: string;
  session: Models['Principal'];
  host: HTMLElement | null;
  onDenied: () => void;
}) {
  const [params, setParams] = useSearchParams();
  const selectedDay = params.get('payDay') || day || undefined;
  const page = Number(params.get('candidatePage') || 1);
  const view = useRead(JSON.stringify([session.subjectId, assignment, selectedDay, page]), () =>
    payDetail(assignment, selectedDay, page),
  );
  const data = view.data;
  const originalId = params.get('award') ?? '',
    rev = params.get('awardRevision') ?? '';
  const original = useRead(
    data && originalId && rev ? JSON.stringify([assignment, originalId, rev]) : undefined,
    () => awardOriginal(originalId, rev),
    'fresh',
  );
  const [query, setQuery] = useState(selectedDay ?? '');
  const [edit, setEdit] = useState<{ action: string; award?: Award }>();
  const [notice, setNotice] = useState('');
  const opener = useRef<string>('');
  useEffect(() => {
    if (data) setQuery(data.businessAt);
  }, [data?.businessAt]);
  useEffect(() => {
    if (view.status === 403 || view.status === 404) {
      setEdit(undefined);
      onDenied();
    }
  }, [view.status]);
  const sourceFocus = params.get('payFocus');
  const seenFocus = useRef('');
  useEffect(() => {
    if (!sourceFocus || !data) return;
    const key = location.href;
    if (seenFocus.current === key) return;
    const link = document.querySelector<HTMLElement>(
      '[id^="pay-basis-' + (sourceFocus === 'plan' ? 'plan' : 'rate') + '-"]',
    );
    if (link) {
      link.focus();
      seenFocus.current = key;
    }
  }, [sourceFocus, data, original.data]);
  const setQueryParams = (change: Record<string, string | undefined>) =>
    setParams((previous) => {
      const next = new URLSearchParams(previous);
      for (const [k, v] of Object.entries(change)) {
        if (v) next.set(k, v);
        else next.delete(k);
      }
      return next;
    });
  const [focusTarget,setFocusTarget]=useState('');
  useLayoutEffect(()=>{
    if(!focusTarget||edit||originalId||!data)return;
    const target=document.getElementById(focusTarget);
    if(target){target.focus();setFocusTarget('')}
  },[focusTarget,edit,originalId,data]);
  const focus=()=>setFocusTarget(opener.current);
  const start = (action: string, el: HTMLElement, award?: Award) => {
    opener.current = el.id;
    setNotice('');
    setEdit({ action, award });
  };
  if (view.error)
    return (
      <Alert severity="error" action={<Button onClick={view.refresh}>重试</Button>}>
        {view.error}
      </Alert>
    );
  if (!data) return <Typography role="status">正在读取个人基本工资…</Typography>;
  if (edit && data.actions.includes(edit.action))
    return (
      <BasePayEditor
        key={edit.action + (edit.award?.revision ?? '')}
        session={session}
        detail={data}
        action={edit.action}
        award={edit.award}
        initialDay={edit.award?.effectiveDate ?? (selectedDay || '')}
        host={host}
        onDenied={() => {
          setEdit(undefined);
          view.invalidate();
          onDenied();
        }}
        onReload={view.refresh}
        onCancel={() => {
          setEdit(undefined);
          focus();
        }}
        onSaved={(receipt) => {
          setEdit(undefined);
          opener.current = 'award-' + receipt.award.revision;
          setNotice('已保存 · 申请号 ' + receipt.submissionId);
          setQueryParams({ payDay: receipt.award.effectiveDate, award: undefined, awardRevision: undefined });
          view.refresh();
          focus();
        }}
      />
    );
  if (originalId && rev) {
    const record = original.data?.assignmentId === assignment ? original.data : undefined;
    return (
      <>
        <HistoryNotice
          label="返回核定记录"
          onReturn={() => {
            setQueryParams({ award: undefined, awardRevision: undefined, payFocus: undefined });
            setFocusTarget('award-'+rev);
          }}
        />
        {original.error && <Alert severity="error">{original.error}</Alert>}
        {record && (
          <>
            <ObjectDetailSection title={awardLabels[record.action] + '原件'}>
              <ObjectDetailField label="生效日期">{record.effectiveDate}</ObjectDetailField>
              <ObjectDetailField label="个人基本工资">{money(record.amount)} 元/月</ObjectDetailField>
              {record.reason && <ObjectDetailField label="业务原因">{record.reason}</ObjectDetailField>}
              {record.note && <ObjectDetailField label="本次缘由">{record.note}</ObjectDetailField>}
              <ObjectDetailField label="记录时刻">
                <time dateTime={record.recordedAt}>{recordedTime(record.recordedAt)}</time>
              </ObjectDetailField>
              {record.operator && <ObjectDetailField label="办理人">{record.operator}</ObjectDetailField>}
            </ObjectDetailSection>
            {record.basis && (
              <PayBasisFacts basis={record.basis} title="核定时依据" day={record.effectiveDate} original />
            )}
          </>
        )}
      </>
    );
  }
  return (
    <ObjectForm>
      {notice && (
        <Alert severity="success" onClose={() => setNotice('')}>
          {notice}
        </Alert>
      )}
      {host &&
        createPortal(
          <>
            {['FIRST', 'ADJUST']
              .filter((a) => data.actions.includes(a))
              .map((a) => (
                <Button
                  id={'pay-action-' + a}
                  key={a}
                  variant="contained"
                  onClick={(e) => start(a, e.currentTarget)}
                >
                  {awardLabels[a]}
                </Button>
              ))}
          </>,
          host,
        )}
      <ObjectDetailSection title="个人基本工资">
        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (query) setQueryParams({ payDay: query, candidatePage: undefined });
          }}
        >
          <Stack direction="row" alignItems="center" spacing={1}>
            <ObjectTextField
              type="date"
              label="工资查询日"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
            <Button type="submit">查询</Button>
          </Stack>
        </form>
        {data.current ? (
          <>
            <ObjectDetailField label="基本月薪">{money(data.current.amount)} 元/月</ObjectDetailField>
            <ObjectDetailField label="生效日期">{data.current.effectiveDate}</ObjectDetailField>
          </>
        ) : (
          data.active && <Typography>尚未定薪</Typography>
        )}
      </ObjectDetailSection>
      {data.policy && <PayBasisFacts basis={data.policy} title="当日制度" day={data.businessAt} />}
      {!data.policy &&
        data.candidates.map((b) => (
          <PayBasisFacts key={b.planId} basis={b} title="可用定薪依据" day={data.businessAt} />
        ))}
      {!data.policy && data.candidateCount > data.candidatePageSize && (
        <Pagination
          aria-label="薪酬计划候选分页"
          count={Math.ceil(data.candidateCount / data.candidatePageSize)}
          page={data.candidatePage}
          onChange={(_, page) => setQueryParams({ candidatePage: String(page) })}
        />
      )}
      {data.message && <Alert severity="info">{data.message}</Alert>}
      {data.items.length > 0 && (
        <ObjectDetailSection title="按日核定记录">
          <TableContainer>
            <Table size="small" aria-label="按日核定记录">
              <TableHead>
                <TableRow>
                  {['生效日期', '办理', '基本月薪（元）', '操作'].map((h) => (
                    <TableCell key={h}>{h}</TableCell>
                  ))}
                </TableRow>
              </TableHead>
              <TableBody>
                {data.items.map((r) => (
                  <TableRow key={r.revision}>
                    <TableCell>{r.effectiveDate}</TableCell>
                    <TableCell>{awardLabels[r.action]}</TableCell>
                    <TableCell>{money(r.amount)}</TableCell>
                    <TableCell>
                      <Button
                        id={'award-' + r.revision}
                        onClick={() =>
                          setQueryParams({ award: r.awardId, awardRevision: r.revision, payFocus: undefined })
                        }
                      >
                        查看原件
                      </Button>
                      {r.current &&
                        ['CORRECT', 'WITHDRAW']
                          .filter((a) => data.actions.includes(a))
                          .map((a) => (
                            <Button
                              id={'pay-action-' + a + '-' + r.revision}
                              key={a}
                              onClick={(e) => start(a, e.currentTarget, r)}
                            >
                              {awardLabels[a]}
                            </Button>
                          ))}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </ObjectDetailSection>
      )}
    </ObjectForm>
  );
}
