import { useEffect, useRef, useState } from 'react';
import { Link, useLocation, useParams, useSearchParams } from 'react-router-dom';
import { Alert, Box, Button, Menu, MenuItem, Stack, Tab, Tabs, Typography } from '@mui/material';
import Business from '@mui/icons-material/Business';
import {
  EffectiveDates,
  HistoryNotice,
  ObjectDetailField,
  ObjectDetailIdentity,
  ObjectDetailSection,
  ObjectDetailSheet,
  ObjectForm,
  ObjectTextField,
  RecordControl,
  recordedTime,
} from '@chnhcm/ui';
import { ApiError, sessionGuard, type Models } from '@chnhcm/api-client';
import { usePageTitle } from '../../shared/navigation';
import { StaffingChangeLog, logReturnHref, logKeys } from './ChangeLog';
import { QuotaEditor, permittedQuotaDraft, type QuotaDraft } from './QuotaEditor';
import { AllocationEditor } from './AllocationEditor';
import {
  departmentRecordLabels,
  missingQuotaRecord,
  quotaTotals,
  quotaSavedNotice,
  type Action,
  type Result,
  period,
  positionQuotaPath,
  quotaLabel,
  quotaPath,
  readDepartment,
  type Department,
} from './quotaApi';

function DepartmentDates({
  day,
  dates = [],
  choose,
}: {
  day: string;
  dates?: string[];
  choose: (day: string, fromAxis: boolean) => void;
}) {
  const [open, setOpen] = useState(false);
  const [input, setInput] = useState(day);
  const trigger = useRef<HTMLButtonElement>(null);
  const selected = dates.includes(day) ? day : undefined;
  return (
    <Stack gap={0.5}>
      <Stack direction={{ xs: 'column', md: 'row' }} alignItems={{ md: 'center' }} gap={1}>
        {dates.length > 0 && (
          <Box sx={{ minWidth: 0, flex: 1 }}>
            <EffectiveDates dates={dates} selected={selected} onSelect={(date) => choose(date, true)} />
          </Box>
        )}
        <Stack direction="row" alignItems="center" flexWrap="wrap" gap={1}>
          {day && !selected && <Typography variant="body2">统计日期 <time dateTime={day}>{day}</time></Typography>}
          <Button
            size="small"
            ref={trigger}
            aria-expanded={open}
            onClick={() => {
              setInput(day);
              setOpen(!open);
            }}
          >
            查看指定日
          </Button>
        </Stack>
      </Stack>
      {open && (
        <Box
          component="form"
          onSubmit={(e) => {
            e.preventDefault();
            setOpen(false);
            trigger.current?.focus();
            choose(input, false);
          }}
        >
          <ObjectForm>
            <Stack direction="row" alignItems="center" flexWrap="wrap" gap={1}>
              <ObjectTextField
                type="date"
                label="统计日期"
                autoFocus
                required
                value={input}
                onChange={(e) => setInput(e.target.value)}
              />
              <Button type="submit">查询统计</Button>
            </Stack>
          </ObjectForm>
        </Box>
      )}
    </Stack>
  );
}

export function QuotaDepartment({ session }: { session: Models['Principal'] }) {
  const { id = '' } = useParams(),
    location = useLocation(),
    [params, setParams] = useSearchParams();
  const day = params.get('at') ?? params.get('businessAt') ?? '',
    version = params.get('sourceVersion') ?? '',
    record = params.get('selectedRecordId') ?? '';
  const [data, setData] = useState<{ key: string; value: Department }>(),
    [error, setError] = useState(''),
    [edit, setEdit] = useState<Action | 'group'>(),
    [menu, setMenu] = useState<HTMLElement | null>(null),
    [reload, setReload] = useState(0);
  const key = JSON.stringify([id, day, record, version, reload, session.csrf]);
  const dateControls = useRef<HTMLDivElement>(null);
  const focusDate = useRef<'axis' | 'specified' | undefined>(undefined);
  const [draft, setDraft] = useState<QuotaDraft>();
  const pending = useRef<
    | { id: string; day: string; selection: string; record?: string; action: Action; draft: QuotaDraft }
    | undefined
  >(undefined);
  const [notice, setNotice] = useState('');
  const view = data?.key === key ? data.value : undefined;
  usePageTitle(view?.name ?? '部门编制');
  const patch = (values: Record<string, string>) => {
    const next = new URLSearchParams(window.location.search);
    for (const [k, v] of Object.entries(values)) {
      if (v) next.set(k, v);
      else next.delete(k);
    }
    setParams(next);
  };
  useEffect(() => {
    const abort = new AbortController(),
      same = sessionGuard();
    setError('');
    setEdit(undefined);
    setDraft(undefined);
    setMenu(null);
    readDepartment(id, day, version, abort.signal, record || undefined)
      .then((value) => {
        if (!abort.signal.aborted && same()) {
          const attempt = pending.current;
          pending.current = undefined;
          if (
            attempt &&
            attempt.id === id &&
            attempt.day === day &&
            attempt.selection === record &&
            value.actions?.includes(attempt.action) &&
            value.recordId === attempt.record
          ) {
            setDraft(permittedQuotaDraft(attempt.draft, attempt.action, value));
            setEdit(attempt.action);
          }
          setData({ key, value });
        }
      })
      .catch((e) => {
        if (!abort.signal.aborted && same()) {
          if (e instanceof ApiError && [401, 403, 404].includes(e.status)) pending.current = undefined;
          setError(e.message);
        }
      });
    return () => abort.abort();
  }, [key]);
  useEffect(() => {
    if (!view || !focusDate.current) return;
    const selector = focusDate.current === 'axis' ? '[aria-current="date"]' : 'button[aria-expanded]:not([aria-haspopup])';
    dateControls.current?.querySelector<HTMLElement>(selector)?.focus({ preventScroll: true });
    focusDate.current = undefined;
  }, [view]);
  useEffect(() => {
    const clear = () => {
      pending.current = undefined;
      setDraft(undefined);
      setEdit(undefined);
      setData(undefined);
    };
    window.addEventListener('chnhcm-session-invalidated', clear);
    return () => window.removeEventListener('chnhcm-session-invalidated', clear);
  }, []);
  const back = new URLSearchParams(params);
  for (const k of ['at', 'sourceVersion', 'selectedRecordId', 'tab', ...logKeys]) back.delete(k);
  const selected = view?.selectionMissing ? undefined : view;
  const snapshot = !version && !!selected;
  function chooseDay(date: string, fromAxis: boolean) {
    if (date === view?.businessAt && !record) return;
    focusDate.current = fromAxis ? 'axis' : 'specified';
    patch({ at: date, selectedRecordId: '', tab: '', logReturn: '' });
  }
  const finish = (result: Result, date: string) => {
    setNotice(quotaSavedNotice(result));
    setEdit(undefined);
    setReload((n) => n + 1);
    patch({ at: date, tab: '', sourceVersion: '', selectedRecordId: '' });
  };
  if (edit && edit !== 'group' && view && selected)
    return (
      <QuotaEditor
        view={{ ...view, targetKind: 'department' }}
        record={selected}
        session={session}
        action={edit}
        close={() => setEdit(undefined)}
        draft={draft}
        reread={(input, denied) => {
          pending.current = {
            id,
            day,
            selection: record,
            record: selected.recordId,
            action: edit,
            draft: input,
          };
          setData(undefined);
          setDraft(undefined);
          setEdit(undefined);
          setReload((n) => n + 1);
          setNotice(denied ? '权限已变化，已重新读取可见资料。' : '已重读记录，请核对输入。');
        }}
        done={(r: Result, date: string) => {
          setEdit(undefined);
          setReload((n) => n + 1);
          setDraft(undefined);
          setNotice(quotaSavedNotice(r));
          patch({
            at: date,
            tab: '',
            sourceVersion: r.originalSourceVersion ?? (r.replayed ? (r.sourceVersion ?? '') : ''),
            selectedRecordId: r.selectedRecordId ?? '',
          });
        }}
      />
    );
  if (edit === 'group' && view)
    return <AllocationEditor view={view} session={session} close={() => setEdit(undefined)} done={finish} />;
  const log = params.get('tab') === 'history' && view?.historyReadFields;
  const returnLog = logReturnHref(params.get('logReturn'), quotaPath + '/departments/' + id);
  type Operation = keyof typeof departmentRecordLabels | 'group';
  const operations: Operation[] = [];
  if (snapshot && (view?.canSetTotal || view?.members.some((m) => m.canAllocate))) operations.push('group');
  if (snapshot && selected?.recordId) {
    for (const op of view?.actions ?? [])
      if (op === 'redate' || op === 'correct' || op === 'delete') operations.push(op);
  }
  const label = (op: Operation) => (op === 'group' ? '调整编制' : departmentRecordLabels[op]);
  return (
    <Box className="standard-page object-page">
      <RecordControl
        identity={view?.name ?? '部门编制'}
        navigation={
          <Button component={Link} state={{ restoreDepartment: id }} to={quotaPath + '?' + back}>
            返回列表
          </Button>
        }
        actions={
          !log &&
          !version &&
          operations.length > 0 && (
            <>
              <Button variant="contained" onClick={() => setEdit(operations[0])}>
                {label(operations[0])}
              </Button>
              {operations.length > 1 && (
                <Button aria-haspopup="menu" onClick={(e) => setMenu(e.currentTarget)}>
                  更多操作
                </Button>
              )}
              <Menu anchorEl={menu} open={!!menu} onClose={() => setMenu(null)}>
                {operations.slice(1).map((op) => (
                  <MenuItem
                    key={op}
                    onClick={() => {
                      setMenu(null);
                      setEdit(op);
                    }}
                  >
                    {label(op)}
                  </MenuItem>
                ))}
              </Menu>
            </>
          )
        }
      />
      {version && (
        <HistoryNotice
          onReturn={() => patch({ sourceVersion: '', selectedRecordId: record, tab: 'basic', logReturn: '' })}
        />
      )}
      {returnLog && (
        <Box>
          <Button component={Link} to={returnLog}>
            返回变更日志
          </Button>
        </Box>
      )}
      <Box className="page-scroll" data-page-scroll role="region" aria-label="部门编制资料" tabIndex={0}>
        {notice && <Alert onClose={() => setNotice('')}>{notice}</Alert>}
        {view && record && !selected && (
          <Alert
            severity="warning"
            action={<Button onClick={() => patch({ selectedRecordId: '' })}>查看默认记录</Button>}
          >
            {missingQuotaRecord}
          </Alert>
        )}
        {error && (
          <>
            <Alert severity="error" action={<Button onClick={() => setReload((n) => n + 1)}>重试</Button>}>{error}</Alert>
            {!version && <DepartmentDates day={day} choose={chooseDay} />}
          </>
        )}
        {!error && !view && <Typography role="status">正在读取…</Typography>}
        {view && selected && (
          <ObjectDetailSheet
            identity={<ObjectDetailIdentity icon={<Business />} title={view.name} summary="部门编制" />}
            tabs={
              view.historyReadFields && (
                <Tabs
                  value={log ? 'history' : 'basic'}
                  variant="scrollable"
                  allowScrollButtonsMobile
                  onChange={(_, v) => patch({ tab: v })}
                >
                  <Tab id="allocation-basic-tab" aria-controls="allocation-basic-panel" value="basic" label="部门分配" />
                  <Tab id="allocation-history-tab" aria-controls="allocation-history-panel" value="history" label="变更日志" />
                </Tabs>
              )
            }
            timeline={!version && !log && (
              <Box ref={dateControls}>
                <DepartmentDates day={view.businessAt} dates={view.allocationDates} choose={chooseDay} />
              </Box>
            )}
          >
            <Box role={view.historyReadFields ? 'tabpanel' : undefined}
              id={log ? 'allocation-history-panel' : 'allocation-basic-panel'}
              aria-labelledby={view.historyReadFields ? (log ? 'allocation-history-tab' : 'allocation-basic-tab') : undefined}>
              {log ? (
                <StaffingChangeLog
                  kind="staffingquota"
                  quotaTarget="department"
                  id={view.quotaId!}
                  onReadFields={(fields) => {
                    if (
                      fields.length !== view.historyReadFields?.length ||
                      fields.some((f) => !view.historyReadFields?.includes(f))
                    ) {
                      setData(undefined);
                      setReload((n) => n + 1);
                    }
                  }}
                  onDenied={() => {
                    setData(undefined);
                    setReload((n) => n + 1);
                  }}
                />
              ) : (
                <>
                  <ObjectDetailSection title="部门额度">
                    {view.scope === 'complete' && (
                      <ObjectDetailField label="部门总额">
                        {quotaLabel(selected, 'department')}
                      </ObjectDetailField>
                    )}
                    {snapshot && (
                      <ObjectDetailField
                        label={quotaTotals[view.scope]}
                      >
                        {view.total ?? '受限'}
                      </ObjectDetailField>
                    )}
                    {snapshot && view.unsetCount !== undefined && (
                      <ObjectDetailField label="未核编岗位数">{view.unsetCount}</ObjectDetailField>
                    )}
                    {snapshot && view.remaining !== undefined && (
                      <ObjectDetailField label="待分配">{view.remaining}</ObjectDetailField>
                    )}
                    {selected.effectiveDate && (
                      <ObjectDetailField label="总额适用期间">{period(selected)}</ObjectDetailField>
                    )}
                    {selected.reason && (
                      <ObjectDetailField label="调整原因">{selected.reason}</ObjectDetailField>
                    )}
                    {selected.operator && (
                      <ObjectDetailField label="操作者">{selected.operator}</ObjectDetailField>
                    )}
                    {selected.recordedAt && (
                      <ObjectDetailField label="操作时间">
                        <time dateTime={selected.recordedAt}>{recordedTime(selected.recordedAt)}</time>
                      </ObjectDetailField>
                    )}
                  </ObjectDetailSection>
                  {snapshot && (
                    <ObjectDetailSection title="直属启用岗位">
                      {view.members.length === 0 ? (
                        <Typography>当日没有可见的启用岗位</Typography>
                      ) : (
                        view.members.map((m) => {
                          const parent = new URLSearchParams(params);
                          parent.set('at', view.businessAt);
                          const dest = new URLSearchParams({ back: location.pathname + '?' + parent });
                          if (m.recordId) dest.set('selectedRecordId', m.recordId);
                          const name = m.name ?? m.code ?? '岗位';
                          return (
                            <ObjectDetailField key={m.positionId} label={
                              <Link to={positionQuotaPath(m.positionId, dest)}>{name}</Link>
                            }>
                              {quotaLabel(m)}
                            </ObjectDetailField>
                          );
                        })
                      )}
                    </ObjectDetailSection>
                  )}
                </>
              )}
            </Box>
          </ObjectDetailSheet>
        )}
      </Box>
    </Box>
  );
}
