import { actionLabel } from './AssignmentRecordForm';
import { useEffect, useRef, useState } from 'react';
import { Alert, Box, Button, MenuItem, Paper, Stack, Typography } from '@mui/material';
import { Link, useLocation, useParams } from 'react-router-dom';
import { api, ApiError, unwrap, type Models } from '@chnhcm/api-client';
import {
  ObjectDetailField,
  ObjectDetailSection,
  ObjectForm,
  ObjectTextField,
  RecordBackButton,
  RecordControl,
  UnsavedChangesDialog,
  recordedTime,
} from '@chnhcm/ui';
import { useDraftExit } from '../../shared/dated/useDraftExit';
import { usePageTitle } from '../../shared/navigation';
import * as person from '../personnel/api';
import { useRuleSubmission } from '../readiness/useRuleSubmission';
import { Target, labels, type TargetKind, type Option } from './Target';

export const employment = async (id: string, businessAt?: string) =>
  unwrap(
    await api.GET('/api/v1/persons/{id}/employment', {
      params: { path: { id }, query: { businessAt: businessAt || undefined } },
    }),
  );
const empty = (id: string): Models['HireInput'] => ({
  person: id,
  employer: '',
  department: '',
  job: '',
  arrival: '',
  notice: '',
});
type Command = { hire: Models['HireInput'] } | { cancel: string; revision: string; reason: string };
export function HirePage({ session }: { session: Models['Principal'] }) {
  const { id = '' } = useParams();
  return <EmployeeHire key={session.instanceId + session.subjectId + id} id={id} session={session} />;
}
export function EmployeeHire({ id, session, cancellation }: { id: string; session: Models['Principal']; cancellation?: {action:Models['EmploymentAction']; returnTo:string; done:()=>void} }) {
  const route = useLocation(),
    back = cancellation?.returnTo ?? person.path + '/' + id + route.search;
  const [day, setDay] = useState(''),
    [mode, setMode] = useState<'hire' | 'cancel' | ''>(cancellation ? 'cancel' : route.state?.hire ? 'hire' : '');
  const [draft, setDraft] = useState(() => empty(id)),
    [choices, setChoices] = useState<Partial<Record<TargetKind, Option>>>({});
  const [reason, setReason] = useState(''),
    [action, setAction] = useState<Models['EmploymentAction'] | undefined>(cancellation?.action),
    [original, setOriginal] = useState('');
  const [notice, setNotice] = useState(''),
    [error, setError] = useState(''),
    [generation, newPreview] = useState(0);
  const failure = useRef<HTMLDivElement>(null),
    saved = useRef(false);
  const details = person.useRead(id, () => person.profile(id));
  const overview = person.useRead(JSON.stringify([id, day]), () => employment(id, day));
  const decisions = person.useRead(mode === 'hire' && overview.data?.canHire ? id : undefined, async () =>
    unwrap(await api.GET('/api/v1/persons/{id}/appointment-decisions', { params: { path: { id } } })),
  );
  const basis = decisions.data?.find(
    (v) => v.ref.source === draft.decision?.source && v.ref.version === draft.decision.version,
  );
  const source = person.useRead(original || undefined, async () =>
    unwrap(await api.GET('/api/v1/employment-actions/{id}', { params: { path: { id: original } } })),
  );
  const header = { 'X-CSRF-Token': session.csrf };
  const ready =
    mode === 'hire' &&
    overview.data?.canHire &&
    draft.arrival &&
    draft.employer &&
    draft.department &&
    draft.job &&
    (draft.decision ? !!basis : draft.notice?.trim());
  const previewKey = JSON.stringify([draft, generation]);
  const selected = useRef<{ key: string; adoption: Models['HireAdoption'] } | undefined>(undefined);
  const preview = person.useRead(ready ? previewKey : undefined, async () => {
    const expected = selected.current?.key === previewKey ? selected.current.adoption : undefined;
    return unwrap(
      await api.POST('/api/v1/hires/preview', { params: { header }, body: { ...draft, expected } }),
    );
  });
  useEffect(() => {
    if (preview.data?.adoption) selected.current = { key: previewKey, adoption: preview.data.adoption };
  }, [previewKey, preview.data]);
  const close = () => {
    selected.current = undefined;
    setMode('');
    if (cancellation) cancellation.done();
    setDraft(empty(id));
    setChoices({});
    setReason('');
    setAction(undefined);
    setError('');
  };
  const leave = useDraftExit(
    !!mode && (JSON.stringify(draft) !== JSON.stringify(empty(id)) || !!reason),
    () => saved.current,
    close,
  );
  const send = useRuleSubmission<Command, Models['HireReceipt']>('employee-hire', session, {
    send: async (cmd, key) =>
      'hire' in cmd
        ? unwrap(
            await api.POST('/api/v1/hires', {
              params: { header: { ...header, 'Idempotency-Key': key } },
              body: cmd.hire,
            }),
          )
        : unwrap(
            await api.POST('/api/v1/employment-actions/{id}/cancel', {
              params: { path: { id: cmd.cancel }, header: { ...header, 'Idempotency-Key': key } },
              body: { revision: cmd.revision, reason: cmd.reason },
            }),
          ),
    resolve: async (sub) =>
      unwrap(
        await api.POST('/api/v1/hires/submissions/{sub}/resolve', { params: { path: { sub }, header } }),
      ),
    finished: (r, recovered) => {
      if (r.outcome === 'accepted') {
        if (recovered !== 'other') {
          saved.current = true;
          if (draft.arrival) setDay(draft.arrival);
          close();
        }
        overview.invalidate();
        setNotice(recovered ? '已查回原办理。' : '已保存。');
      } else {
        setError(r.outcome === 'rejected' ? '原提交已拒绝。' : '原号已确认未执行。');
      }
    },
    failed: (e) => {
      if (e instanceof ApiError && [401, 403, 404].includes(e.status)) {
        close();
        overview.invalidate();
      }
      if (e instanceof ApiError && e.code === 'APPOINTMENT_CHANGED') decisions.refresh();
      setError(person.message(e));
    },
  });
  usePageTitle(mode === 'hire' ? '办理入职' : '工作关系与任职');
  useEffect(() => {
    send.start();
  }, [send.start]);
  useEffect(() => {
    if (error) failure.current?.focus();
  }, [error]);
  useEffect(() => {
    if (
      details.status === 403 ||
      details.status === 404 ||
      overview.status === 403 ||
      overview.status === 404 ||
      (mode === 'hire' && overview.data && !overview.data.canHire) ||
      (mode === 'cancel' &&
        overview.data &&
        !overview.data.actions.some(
          (a) => a.id === action?.id && a.revision === action.revision && a.canCancel,
        ))
    ) {
      close();
      setOriginal('');
    }
  }, [details.status, overview.status, overview.data, mode, action]);
  const update = (patch: Partial<Models['HireInput']>) => {
    selected.current = undefined;
    setDraft((v) => ({ ...v, ...patch }));
    setError('');
  };
  const selectDecision = (source: string) => {
    const item = decisions.data?.find((v) => v.ref.source === source);
    selected.current = undefined;
    setDraft({
      ...empty(id),
      decision: item?.ref,
      arrival: item?.arrival ?? '',
      lastDay: item?.lastDay,
      department: item?.department?.id ?? '',
      job: item?.job?.id ?? '',
      bu: item?.bu?.id,
      location: item?.location?.id,
      employer: item?.employer?.id ?? '',
    });
    setChoices({});
    setError('');
  };
  useEffect(() => {
    if (
      draft.decision &&
      (decisions.status === 403 || decisions.status === 404 || (decisions.data && !basis))
    ) {
      setDraft(empty(id));
      setChoices({});
      selected.current = undefined;
      setError('原任用决定已变化或不可用，请重新选择依据。');
    }
  }, [decisions.status, decisions.data, draft.decision, basis, id]);
  const changeTarget = (kind: TargetKind, v?: Option) => {
    selected.current = undefined;
    setChoices((old) => ({
      ...old,
      [kind]: v,
      ...(['bu', 'location', 'department'].includes(kind) ? { employer: undefined } : {}),
    }));
    update({
      [kind]: v?.id ?? '',
      ...(!basis?.employer && ['bu', 'location', 'department'].includes(kind) ? { employer: '' } : {}),
    });
  };
  const begin = (next: 'hire' | 'cancel', row?: Models['EmploymentAction']) => {
    saved.current = false;
    setAction(row);
    setReason('');
    setMode(next);
    setNotice('');
    setError('');
    send.start();
  };
  const submit = async () => {
    setError('');
    if (mode === 'cancel' && action) {
      await send.submit({ cancel: action.id, revision: action.revision, reason });
    } else if (preview.data?.adoption) {
      await send.submit({ hire: { ...draft, expected: preview.data.adoption } });
    }
  };
  const missingNames: Record<string, string> = {
    document: '适用的身份证件',
    expiryDate: '证件期限',
    inspection: '原件查验结论',
  };
  const editable = details.data?.editable ?? [];
  const canComplete = preview.data?.missing?.some((field) =>
    field === 'document'
      ? editable.includes('documentKind') && editable.includes('documentNumber')
      : field === 'expiryDate' && (editable.includes('expiryDate') || editable.includes('longTerm')),
  );
  const targets = (k: TargetKind) => {
    const fixed = basis?.[k];
    if (fixed)
      return (
        <ObjectDetailField key={k} label={labels[k]}>
          {fixed.name}
        </ObjectDetailField>
      );
    return (
      <Target
        key={k}
        kind={k}
        draft={draft}
        session={session}
        choice={choices[k]}
        onChange={(v) => changeTarget(k, v)}
      />
    );
  };
  return (
    <Box className="standard-page object-page">
      <RecordControl
        navigation={null}
        identity={
          <>
            <RecordBackButton component={Link} to={back} />
            {details.data?.name}
          </>
        }
        actions={
          !mode && overview.data?.canHire ? (
            <Button variant="contained" onClick={() => begin('hire')}>
              办理入职
            </Button>
          ) : null
        }
      />
      <Box className="page-scroll" data-page-scroll role="region" tabIndex={0} aria-label="入职与任职">
        <Paper sx={{ p: { xs: 1.5, sm: 3 }, maxWidth: 1000, mx: 'auto' }}>
          <Typography variant="h5" component="h1">
            {mode === 'hire' ? '办理入职' : mode === 'cancel' ? '撤销未来入职' : '工作关系与任职'}
          </Typography>
          {details.data && (
            <Typography>
              {details.data.name} · {details.data.number}
            </Typography>
          )}
          {(error || (mode==='hire' && details.error) || overview.error) && (
            <Alert ref={failure} tabIndex={-1} severity="error">
              {error || (mode==='hire' && details.error) || overview.error}
            </Alert>
          )}
          {notice && <Alert severity="success">{notice}</Alert>}
          {send.pending.map((ref) => (
            <Button key={ref.submissionId} onClick={() => void send.recover(ref)}>
              查回原提交
            </Button>
          ))}
          {send.storageError && <Alert severity="error">无法保存提交号，请恢复浏览器存储后办理。</Alert>}
          {mode && details.data && overview.data && (
            <Box
              component="form"
              onSubmit={(e) => {
                e.preventDefault();
                void submit();
              }}
            >
              <ObjectForm>
                {mode === 'hire' && overview.data.canHire && (
                  <ObjectDetailSection title="拟任安排">
                    {!!decisions.data?.length && (
                      <ObjectTextField
                        select
                        label="任用依据"
                        slotProps={{ select: { displayEmpty: true } }}
                        value={basis?.ref.source ?? ''}
                        onChange={(e) => selectDecision(e.target.value)}
                      >
                        <MenuItem value="">无编号任用通知</MenuItem>
                        {decisions.data.map((v) => (
                          <MenuItem key={v.ref.source} value={v.ref.source}>
                            {v.number}
                          </MenuItem>
                        ))}
                      </ObjectTextField>
                    )}
                    {decisions.error && <Alert severity="error">{decisions.error}</Alert>}
                    {basis?.arrival ? (
                      <ObjectDetailField label="到岗日">{basis.arrival}</ObjectDetailField>
                    ) : (
                      <ObjectTextField
                        type="date"
                        required
                        label="到岗日"
                        value={draft.arrival}
                        onChange={(e) => update({ arrival: e.target.value })}
                      />
                    )}
                    {basis?.lastDay ? (
                      <ObjectDetailField label="任期至（含当日）">{basis.lastDay}</ObjectDetailField>
                    ) : (
                      <ObjectTextField
                        type="date"
                        label="任期至（含当日）"
                        value={draft.lastDay ?? ''}
                        onChange={(e) => update({ lastDay: e.target.value || undefined })}
                      />
                    )}
                    {(['department', 'job', 'bu', 'location', 'employer'] as const).map(targets)}
                    {!draft.decision && (
                      <ObjectTextField
                        required
                        label="任用通知说明"
                        multiline
                        value={draft.notice ?? ''}
                        onChange={(e) => update({ notice: e.target.value })}
                      />
                    )}
                  </ObjectDetailSection>
                )}
                {mode === 'cancel' && action && (
                  <ObjectDetailSection title="撤销依据">
                    <Typography>
                      {action.from} 到岗 · {action.notice || '任用决定'}
                    </Typography>
                    <ObjectTextField
                      required
                      label="撤销说明"
                      value={reason}
                      onChange={(e) => setReason(e.target.value)}
                    />
                  </ObjectDetailSection>
                )}
                {preview.data?.missing?.length && (
                  <Alert severity="warning">
                    待补齐：{preview.data.missing.map((k) => missingNames[k]).join('、')}。
                    {canComplete && (
                      <Button
                        component="a"
                        target="_blank"
                        rel="noopener"
                        href={person.path + '/' + id + '?action=correct'}
                      >
                        在新窗口补充原档
                      </Button>
                    )}
                  </Alert>
                )}
                {preview.error && <Alert severity="error">{preview.error}</Alert>}
                <Stack direction="row" gap={1} flexWrap="wrap">
                  <Button
                    type="submit"
                    variant="contained"
                    disabled={send.busy || (mode === 'hire' && !preview.data?.adoption)}
                  >
                    保存
                  </Button>
                  <Button onClick={() => leave.request(close)}>取消</Button>
                  {(error || decisions.error || preview.error || preview.data?.missing?.length) &&
                    mode === 'hire' && (
                      <Button
                        onClick={() => {
                          setError('');
                          selected.current = undefined;
                          if (draft.decision || decisions.error) decisions.refresh();
                          newPreview((v) => v + 1);
                        }}
                      >
                        重新核对
                      </Button>
                    )}
                </Stack>
              </ObjectForm>
            </Box>
          )}
          {!mode && overview.data && (
            <ObjectForm>
              <ObjectDetailSection title="该日任职">
                {overview.data && !overview.data.terms.length && <Typography>暂无可查看的任职</Typography>}
                <ObjectTextField
                  type="date"
                  label="查询日期"
                  value={day}
                  onChange={(e) => setDay(e.target.value)}
                />
                {overview.data && (
                  <Typography>
                    可见在职 {overview.data.people} 人，任职 {overview.data.assignments} 份，占编{' '}
                    {overview.data.seats}
                  </Typography>
                )}
              </ObjectDetailSection>
              <ObjectDetailSection title="办理记录">
                {overview.data?.actions.map((a) => (
                  <Box key={a.id} sx={{ mb: 1 }}>
                    <Typography>
                      {a.businessDate} ·{' '}
                      {a.kind==='Hire'&&a.cancelled ? '入职（已撤销）' : actionLabel(a)} ·{' '}
                      {a.notice || (a.kind==='DataChange'?'工作地点变更':'任用决定')}
                    </Typography>
                    <Typography variant="body2">
                      雇用记录 {a.recordNumber} 号 · {a.primaryJob ? '主岗' : '兼岗'}
                    </Typography>
                    <Typography variant="body2">{recordedTime(a.recordedAt)}</Typography>
                    {a.canHistory && (
                      <Button onClick={() => setOriginal(original === a.id ? '' : a.id)}>查看原办理</Button>
                    )}
                    {a.canCancel && <Button onClick={() => begin('cancel', a)}>撤销未来入职</Button>}
                    {a.canHistory && original === a.id && source.error && (
                      <Alert severity="error">{source.error}</Alert>
                    )}
                    {a.canHistory && original === a.id && source.data && (
                      <Box sx={{ overflowWrap: 'anywhere' }}>
                        <Typography>工作关系 {a.employment}</Typography>
                        <Typography>首任职 {a.assignment}</Typography>
                        {source.data.locationChange&&<Typography>工作地点 {source.data.locationChange.before?.name??'—'} → {source.data.locationChange.after.name}</Typography>}
                        {source.data.sources && (
                          <>
                            {source.data.sources.legal.map((v) => (
                              <Typography key={v.from}>
                                {v.from} 起 · {v.entityName}
                              </Typography>
                            ))}
                            {source.data.sources.segments.map((v) => (
                              <Typography key={v.from}>
                                {v.from} 起 · {v.departmentName} · {v.jobName}
                              </Typography>
                            ))}
                            {source.data.appointment && (
                              <>
                                <Typography>任用决定 {source.data.appointment.number}</Typography>
                                <Typography>
                                  原单 {source.data.appointment.ref.source}，修订{' '}
                                  {source.data.appointment.ref.version}
                                </Typography>
                                {source.data.appointment.ref.line && (
                                  <Typography>
                                    名单行 {source.data.appointment.ref.line}，修订{' '}
                                    {source.data.appointment.ref.lineRevision}
                                  </Typography>
                                )}
                              </>
                            )}
                            <Typography>采用规则 {source.data.sources.rule}</Typography>
                            <Typography>人员来源 {source.data.sources.identity.personVersion}</Typography>
                            <Typography>
                              证件行 {source.data.sources.identity.documentId}，修订{' '}
                              {source.data.sources.identity.documentRevision}
                            </Typography>
                          </>
                        )}
                      </Box>
                    )}
                  </Box>
                ))}
              </ObjectDetailSection>
            </ObjectForm>
          )}
        </Paper>
      </Box>
      <UnsavedChangesDialog {...leave.dialog} />
    </Box>
  );
}
