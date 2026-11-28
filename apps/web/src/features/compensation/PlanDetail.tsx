import { useEffect, useLayoutEffect, useRef, useState } from 'react';
import { flushSync } from 'react-dom';
import PaymentsOutlinedIcon from '@mui/icons-material/PaymentsOutlined';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import { Alert, Box, Button, IconButton, Menu, MenuItem, Stack, Tab, Tabs, Typography } from '@mui/material';
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom';
import {
  DateField,
  EffectiveDates,
  HistoryNotice,
  ObjectDetailIdentity,
  ObjectDetailSheet,
  RecordBackButton,
  RecordControl,
  recordedTime,
} from '@chnhcm/ui';
import { useMaintenanceSearchParams, usePageTitle } from '../../shared/navigation';
import { useRead } from '../../shared/useRead';
import { PlanEditor, type PlanDraftBackup } from './PlanEditor';
import { useDraftRecovery } from './useDraftRecovery';
import { PlanFacts, PlanLog } from './PlanFacts';
import { planContext, planHistory, planPath, planTitle, readPlan } from './plans';
import { actionLabel, type Action, type Receipt, type Principal } from './rates';

export function PlanDetailPage({ session, creating = false }: { session: Principal; creating?: boolean }) {
  const { id = '' } = useParams(),
    navigate = useNavigate();
  const location = useLocation();
  const [leaving, setLeaving] = useState<string>();
  useLayoutEffect(() => setLeaving(undefined), [location.key]);
  const [params, setParams] = useMaintenanceSearchParams(planPath);
  const candidate = params.get('back') ?? planPath;
  const back = candidate.split('?')[0] === planPath ? candidate : planPath;
  const selected = params.get('selectedRecordId') || undefined,
    source = params.get('sourceVersion') || undefined;
  const referenceDate = params.get('referenceDate') || undefined;
  const [dateDraft, setDateDraft] = useState('');
  const [edit, setEdit] = useState<Action>(),
    [anchor, setAnchor] = useState<HTMLElement | null>(null);
  const [host, setHost] = useState<HTMLDivElement | null>(null),
    [notice, setNotice] = useState('');
  const opener = useRef<HTMLButtonElement | null>(null);
  const editing = creating ? 'CREATE' : edit;
  const context = useRead(creating ? 'create' : undefined, planContext);
  const read = useRead(!creating && id && leaving !== location.key ? JSON.stringify([id, selected, referenceDate]) : undefined, () =>
    readPlan(id, selected, selected ? undefined : referenceDate),
  );
  const data = read.data;
  const historyTab = !editing && params.get('tab') === 'history' && data?.history;
  const log = useRead(
    !editing && data?.history && (historyTab || source) ? JSON.stringify([id, source, data.sourceVersion]) : undefined,
    () => planHistory(id, source),
    'fresh',
  );
  const snapshot = source ? log.data?.items[0] : undefined;
  const current = source
    ? params.get('side') === 'before'
      ? snapshot?.before
      : (snapshot?.after ?? snapshot?.before)
    : data?.record;
  const visibleCode = source ? snapshot?.code : data?.code;
  const title = creating ? '新建' + planTitle : current?.name || planTitle;
  usePageTitle(title);
  const update = (values: Record<string, string | undefined>, replace = false) => {
    const next = new URLSearchParams(params);
    for (const [k, v] of Object.entries(values)) {
      if (v) next.set(k, v);
      else next.delete(k);
    }
    // Withdraw the previous day's source links before Router changes the address.
    if (next.get('referenceDate') !== params.get('referenceDate') ||
        next.get('selectedRecordId') !== params.get('selectedRecordId')) {
      flushSync(() => setLeaving(location.key));
    }
    setParams(next, { replace, flushSync: true });
  };
  const canEdit = !source && !!data && Object.keys(data.actions).length > 0;
  const start = (action: Action, button?: HTMLButtonElement) => {
    if (button) opener.current = button;
    setAnchor(null);
    setEdit(action);
    setNotice('');
  };
  const cancel = () => {
    if (creating) {
      navigate(back);
      return;
    }
    setEdit(undefined);
    requestAnimationFrame(() => opener.current?.focus());
  };
  const reload = () => {
    setEdit(undefined);
    read.invalidate();
    context.invalidate();
    setNotice('权限或资料已变更，已重新读取。');
  };
  const save = (r: Receipt) => {
    if (!r.objectId) {
      reload();
      return;
    }
    setEdit(undefined);
    const previous =
      data?.dates[
        data.dates.findIndex((node) => node.recordId === selected || node.recordId === data.record.recordId) - 1
      ]?.recordId;
    const target = editing === 'DELETE' ? previous : r.recordId;
    const next = new URLSearchParams({ back });
    if (target) next.set('selectedRecordId', target);
    setNotice(editing === 'DELETE' ? '已删除所选记录' : r.outcome === 'no_change' ? '资料没有变化' : '已保存');
    navigate(planPath + '/' + r.objectId + '?' + next, { replace: creating });
    read.invalidate();
  };
  const fields = (creating ? context.data?.writable : editing && data?.actions[editing]) || [];
  const readyToEdit = editing && (creating ? !!context.data : !!data?.actions[editing] && !source);
  const defaultDate = creating
    ? context.data?.defaultEffectiveDate
    : editing === 'DELETE'
      ? data?.record.effectiveDate
      : data?.windows[editing ?? '']?.default;
  const window = data?.windows.ADD,
    at = data?.dates.findIndex((n) => n.recordId === data.record.recordId) ?? -1;
  const addLabel = data && at >= 0 && at < data.dates.length - 1 ? '插入记录' : '新增记录';
  const errors =
    context.error || read.error || log.error || (source && data && !data.history ? '所选资料不可用' : undefined);
  const valid = creating ? !!context.data : !!data && (!source || !!snapshot);
  const recovery = useDraftRecovery<PlanDraftBackup>(
    JSON.stringify([id, selected, referenceDate, editing]),
    !!readyToEdit,
    !!errors,
  );
  const recoverDenied = (input: PlanDraftBackup) => {
    recovery.remember(input);
    read.invalidate();
    context.invalidate();
    log.invalidate();
    setNotice('办理权限已变更，已重新读取当前可维护的资料。');
  };
  useEffect(() => {
    setEdit(undefined);
    setAnchor(null);
  }, [id, selected, source]);
  useEffect(() => {
    if (edit && data && !data.actions[edit]) {
      setEdit(undefined);
      setNotice('办理权限已变更，已重新读取。');
    }
  }, [data, edit]);
  useEffect(() => {
    setDateDraft(referenceDate ?? data?.businessAt ?? '');
  }, [referenceDate, data?.businessAt]);
  const time = snapshot?.recordedAt;
  const dateQuery = (
    <Stack
      component="form"
      direction="row"
      alignItems="center"
      flexWrap="wrap"
      gap={1}
      sx={{ my: 1 }}
      onSubmit={(e) => {
        e.preventDefault();
        update({ referenceDate: dateDraft || undefined, selectedRecordId: undefined, focus: undefined });
      }}
    >
      <DateField label="适用日期" size="small" value={dateDraft} onChange={(e) => setDateDraft(e.target.value)} />
      <Button type="submit">查看该日</Button>
      {referenceDate && (
        <Button onClick={() => update({ referenceDate: undefined, selectedRecordId: undefined, focus: undefined })}>
          返回当前记录
        </Button>
      )}
    </Stack>
  );
  return (
    <Box className="standard-page object-page" aria-busy={!valid && !errors}>
      <RecordControl
        identity={
          <>
            <RecordBackButton component={Link} to={back} />
            {visibleCode && <Typography>{visibleCode}</Typography>}
            {canEdit && !editing && (
              <IconButton
                aria-label="更多操作"
                aria-haspopup="menu"
                aria-expanded={!!anchor}
                onClick={(e) => {
                  opener.current = e.currentTarget;
                  setAnchor(e.currentTarget);
                }}
              >
                <MoreVertIcon />
              </IconButton>
            )}
          </>
        }
        actions={
          editing ? (
            <Box ref={setHost} sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }} />
          ) : (
            canEdit &&
            data.actions.ADD && (
              <Button
                variant="contained"
                disabled={window?.available === false}
                onClick={(e) => start('ADD', e.currentTarget)}
              >
                {addLabel}
              </Button>
            )
          )
        }
        navigation={null}
      />
      <Menu open={!!anchor} anchorEl={anchor} onClose={() => setAnchor(null)}>
        {data &&
          Object.keys(data.actions)
            .filter((a) => a !== 'ADD')
            .map((a) => (
              <MenuItem key={a} disabled={data.windows[a]?.available === false} onClick={() => start(a as Action)}>
                {actionLabel[a as Action]}
              </MenuItem>
            ))}
      </Menu>
      <div
        className="page-scroll"
        data-page-scroll
        tabIndex={0}
        role="region"
        aria-label="薪酬计划资料"
        data-page-scroll-ready={valid ? 'true' : 'false'}
      >
        {notice && (
          <Alert severity="info" onClose={() => setNotice('')}>
            {notice}
          </Alert>
        )}
        {errors && (
          <Alert
            severity="error"
            action={
              <Button
                onClick={() => {
                  read.refresh();
                  log.refresh();
                  context.refresh();
                }}
              >
                重试
              </Button>
            }
          >
            {errors}
          </Alert>
        )}
        {read.error && referenceDate && dateQuery}
        {!errors && !valid && <Typography role="status">正在读取…</Typography>}
        {valid && (
          <ObjectDetailSheet
            identity={
              <ObjectDetailIdentity
                icon={<PaymentsOutlinedIcon />}
                title={title}
                summary={
                  creating
                    ? planTitle
                    : [
                        planTitle,
                        visibleCode,
                        current?.effectiveDate && data && current.effectiveDate > data.today ? '尚未生效' : undefined,
                      ]
                        .filter(Boolean)
                        .join(' · ')
                }
              />
            }
            tabs={
              !editing &&
              !source && (
                <Tabs
                  aria-label="薪酬计划详情"
                  value={historyTab ? 'history' : 'basic'}
                  onChange={(_, value) => update({ tab: value === 'history' ? value : undefined })}
                >
                  <Tab label="基本资料" value="basic" />
                  {data?.history && <Tab label="变更日志" value="history" />}
                </Tabs>
              )
            }
            timeline={
              !editing &&
              !source &&
              !historyTab &&
              data &&
              data.dates.length > 0 && (
                <EffectiveDates
                  dates={data.dates.map((d) => d.effectiveDate)}
                  selected={current?.effectiveDate}
                  onSelect={(day) =>
                    update({
                      selectedRecordId: data.dates.find((d) => d.effectiveDate === day)?.recordId,
                      referenceDate: undefined,
                      focus: undefined,
                    })
                  }
                />
              )
            }
          >
            {readyToEdit ? (
              <PlanEditor
                recovered={recovery.input}
                onDenied={recoverDenied}
                key={JSON.stringify([id, editing, data?.record.recordId])}
                session={session}
                detail={data}
                defaultDate={defaultDate ?? ''}
                fields={fields}
                action={editing}
                units={context.data?.units ?? data?.units ?? {}}
                scopeDimensions={context.data?.scopeDimensions ?? data?.scopeDimensions ?? []}
                reasons={creating ? (context.data?.reasons ?? []) : (data?.reasons[editing] ?? [])}
                host={host}
                onCancel={cancel}
                onReload={reload}
                onRefresh={read.refresh}
                onSaved={save}
              />
            ) : source ? (
              <>
                <HistoryNotice
                  label="返回变更日志"
                  onReturn={() => update({ sourceVersion: undefined, side: undefined, tab: 'history' })}
                />
                {time && (
                  <Typography component="p">
                    <time dateTime={time}>{recordedTime(time)}</time>
                  </Typography>
                )}
                {current && <PlanFacts record={current} code={visibleCode} historical />}
              </>
            ) : historyTab ? (
              <PlanLog
                items={log.data?.items ?? []}
                link={(version, side) => {
                  const next = new URLSearchParams(params);
                  next.set('sourceVersion', version);
                  if (side) next.set('side', side);
                  else next.delete('side');
                  return planPath + '/' + id + '?' + next;
                }}
              />
            ) : (
              current && (
                <>
                  {data?.readable.includes('effectiveDate') &&
                    current.components?.some((row) => row.method === 'GRADE_RATE') &&
                    dateQuery}
                  <PlanFacts record={current} code={visibleCode} />
                </>
              )
            )}
          </ObjectDetailSheet>
        )}
      </div>
    </Box>
  );
}
