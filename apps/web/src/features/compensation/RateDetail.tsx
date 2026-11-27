import { useEffect, useRef, useState } from 'react';
import PaymentsOutlinedIcon from '@mui/icons-material/PaymentsOutlined';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import { Alert, Box, Button, IconButton, Menu, MenuItem, Tab, Tabs, Typography } from '@mui/material';
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom';
import { EffectiveDates, HistoryNotice, ObjectDetailField, ObjectDetailIdentity, ObjectDetailSection, ObjectDetailSheet,
  RecordBackButton, RecordControl, SourceExplanation, recordedTime } from '@chnhcm/ui';
import { useMaintenanceSearchParams, usePageTitle } from '../../shared/navigation';
import { useRead } from '../../shared/useRead';
import { paths as staffingPaths } from '../staffing/api';
import { RateEditor } from './RateEditor';
import { RateLog } from './RateLog';
import { actionLabel, amountKeys, amountLabels, createContext, formLabels, frequencies, gradeLabel, money, moneyUnit, rateDetail, rateHistory, ratePath,
  rateTitle, statusLabels, type Action, type Principal, type RateRecord, type Receipt } from './rates';

function RateFacts({ record, currency, frequency, precision }: { record: RateRecord; currency?: string; frequency?: string; precision?: number }) {
  const here = useLocation();
  const grade = record.grade, origin = record.adopted, f = frequencies[frequency ?? ''];
  return <><ObjectDetailSection title="职级与计量口径"><div className="object-detail-columns">
    {grade && <ObjectDetailField label="职级">{gradeLabel(record)}</ObjectDetailField>}
    {currency && <ObjectDetailField label="币种">{currency === 'CNY' ? '人民币（CNY）' : currency}</ObjectDetailField>}
    {f && <ObjectDetailField label="约定频率">{f.name}</ObjectDetailField>}
    {f && currency && <ObjectDetailField label="计量口径">{moneyUnit(currency,frequency)}</ObjectDetailField>}
    {precision !== undefined && <ObjectDetailField label="金额小数位数">{precision}</ObjectDetailField>}
  </div></ObjectDetailSection>
    <ObjectDetailSection title="标准金额"><div className="object-detail-columns">
      {record.form && <ObjectDetailField label="标准形式">{formLabels[record.form]}</ObjectDetailField>}
      {amountKeys.map(k => record[k] !== undefined && <ObjectDetailField key={k} label={amountLabels[k]}>{money(record[k])}</ObjectDetailField>)}
    </div></ObjectDetailSection>
    <ObjectDetailSection title="生效记录"><div className="object-detail-columns">
      {record.effectiveDate && <ObjectDetailField label="生效日期">{record.effectiveDate}</ObjectDetailField>}
      {record.status && <ObjectDetailField label="记录状态">{statusLabels[record.status]}</ObjectDetailField>}
      {record.operation && <ObjectDetailField label="操作类型">{actionLabel[record.operation as Action]}</ObjectDetailField>}
      {record.reason && <ObjectDetailField label="办理原因">{record.reason}</ObjectDetailField>}
    </div></ObjectDetailSection>
    {origin && grade && <SourceExplanation><Typography>保存时采用的职级为 {gradeLabel(record)}，业务日为 {origin.businessAt}。</Typography>
      {origin.open && <Link to={staffingPaths.grade + '/' + grade.id + '?' + new URLSearchParams({ sourceVersion: origin.sourceVersion, selectedRecordId: origin.recordId, back: here.pathname + here.search })}>查看当时的职级记录</Link>}</SourceExplanation>}</>;
}

export function RateDetailPage({ session, creating = false }: { session: Principal; creating?: boolean }) {
  const { id = '' } = useParams(), navigate = useNavigate();
  const [params, setParams] = useMaintenanceSearchParams(ratePath);
  const candidate = params.get('back') ?? ratePath;
  const back = candidate.split('?')[0] === ratePath ? candidate : ratePath;
  const selected = params.get('selectedRecordId') || undefined, source = params.get('sourceVersion') || undefined;
  const [edit, setEdit] = useState<Action>(), [anchor, setAnchor] = useState<HTMLElement | null>(null);
  const [host, setHost] = useState<HTMLDivElement | null>(null), [notice, setNotice] = useState('');
  const opener = useRef<HTMLButtonElement | null>(null);
  const editing = creating ? 'CREATE' : edit;
  const context = useRead(creating ? 'create' : undefined, createContext);
  const read = useRead(!creating && id ? JSON.stringify([id, selected]) : undefined, () => rateDetail(id, selected));
  const data = read.data;
  const historyTab = !editing && params.get('tab') === 'history' && data?.history;
  const log = useRead(!editing && data?.history && (historyTab || source) ? JSON.stringify([id, source, data.sourceVersion]) : undefined, () => rateHistory(id, source), 'fresh');
  const snapshot = source ? log.data?.items[0] : undefined;
  const current = source ? params.get('side') === 'before' ? snapshot?.before : snapshot?.after ?? snapshot?.before : data?.record;
  const title = creating ? '新建' + rateTitle : current && gradeLabel(current) || rateTitle;
  usePageTitle(title);
  const update = (values: Record<string,string | undefined>, replace = false) => {
    const next = new URLSearchParams(params);
    for (const [k, v] of Object.entries(values)) { if (v) next.set(k, v); else next.delete(k); }
    setParams(next, { replace });
  };
  const canEdit = !source && !!data && Object.keys(data.actions).length > 0;
  const start = (action: Action, button?: HTMLButtonElement) => {
    if (button) opener.current = button;
    setAnchor(null); setEdit(action); setNotice('');
  };
  const cancel = () => {
    if (creating) { navigate(back); return; }
    setEdit(undefined);
    requestAnimationFrame(() => opener.current?.focus());
  };
  const reload = () => { setEdit(undefined); read.invalidate(); context.invalidate(); setNotice('权限或资料已变更，已重新读取。'); };
  const save = (r: Receipt) => {
    if (!r.objectId) { reload(); return; }
    setEdit(undefined);
    const previous = data?.dates[data.dates.findIndex(node => node.recordId === selected || node.recordId === data.record.recordId) - 1]?.recordId;
    const target = editing === 'DELETE' ? previous : r.recordId;
    const next = new URLSearchParams({ back });
    if (target) next.set('selectedRecordId', target);
    setNotice(editing === 'DELETE' ? '已删除所选记录' : r.outcome === 'no_change' ? '资料没有变化' : '已保存');
    navigate(ratePath + '/' + r.objectId + '?' + next, { replace: creating });
    read.invalidate();
  };
  const fields = (creating ? context.data?.writable : editing && data?.actions[editing]) || [];
  const readyToEdit = editing && (creating ? !!context.data : !!data?.actions[editing] && !source);
  const defaultDate = creating ? context.data?.defaultEffectiveDate : editing === 'DELETE' ? data?.record.effectiveDate : data?.windows[editing ?? '']?.default;
  const window = data?.windows.ADD, at = data?.dates.findIndex(n => n.recordId === data.record.recordId) ?? -1;
  const addLabel = data && at >= 0 && at < data.dates.length - 1 ? '插入记录' : '新增记录';
  const errors = context.error || read.error || log.error || (source && data && !data.history ? '所选资料不可用' : undefined);
  const valid = creating ? !!context.data : !!data && (!source || !!snapshot);
  useEffect(() => { setEdit(undefined); setAnchor(null); }, [id, selected, source]);
  useEffect(() => {
    if (edit && data && !data.actions[edit]) {
      setEdit(undefined); setNotice('办理权限已变更，已重新读取。');
    }
  }, [data, edit]);
  const time = snapshot?.recordedAt;
  return <Box className="standard-page object-page" aria-busy={!valid && !errors}>
    <RecordControl identity={<><RecordBackButton component={Link} to={back} />{current?.grade?.code && <Typography>{current.grade.code}</Typography>}
      {canEdit && !editing && <IconButton aria-label="更多操作" aria-haspopup="menu" aria-expanded={!!anchor} onClick={e => { opener.current = e.currentTarget; setAnchor(e.currentTarget); }}><MoreVertIcon /></IconButton>}</>}
      actions={editing ? <Box ref={setHost} sx={{ display:'flex', gap:1, flexWrap:'wrap' }} /> :
        canEdit && data.actions.ADD && <Button variant="contained" disabled={window?.available === false} onClick={e => start('ADD', e.currentTarget)}>{addLabel}</Button>}
      navigation={null}/>
    <Menu open={!!anchor} anchorEl={anchor} onClose={() => setAnchor(null)}>{data && Object.keys(data.actions).filter(a => a !== 'ADD').map(a =>
      <MenuItem key={a} disabled={data.windows[a]?.available === false} onClick={() => start(a as Action)}>{actionLabel[a as Action]}</MenuItem>)}</Menu>
    <div className="page-scroll" data-page-scroll tabIndex={0} role="region" aria-label="职级薪酬标准资料" data-page-scroll-ready={valid ? 'true':'false'}>
      {notice && <Alert severity="info" onClose={() => setNotice('')}>{notice}</Alert>}
      {errors && <Alert severity="error" action={<Button onClick={() => { read.refresh(); log.refresh(); context.refresh(); }}>重试</Button>}>{errors}</Alert>}
      {!errors && !valid && <Typography role="status">正在读取…</Typography>}
      {valid && <ObjectDetailSheet identity={<ObjectDetailIdentity icon={<PaymentsOutlinedIcon />} title={title}
        summary={creating ? rateTitle : [rateTitle, data?.currency, frequencies[data?.frequency ?? '']?.unitName,
          current?.effectiveDate && data && current.effectiveDate > data.today ? '尚未生效' : undefined].filter(Boolean).join(' · ')} />}
        tabs={!editing && !source && <Tabs aria-label="职级薪酬标准详情" value={historyTab ? 'history' : 'basic'} onChange={(_, value) => update({ tab: value === 'history' ? value : undefined })}>
          <Tab label="基本资料" value="basic" />{data?.history && <Tab label="变更日志" value="history" />}</Tabs>}
        timeline={!editing && !source && !historyTab && data && data.dates.length > 0 && <EffectiveDates dates={data.dates.map(d => d.effectiveDate)} selected={current?.effectiveDate}
          onSelect={day => update({ selectedRecordId: data.dates.find(d => d.effectiveDate === day)?.recordId })}/>}
      >
        {readyToEdit ? <RateEditor key={JSON.stringify([id, editing, data?.record.recordId, fields])}
          session={session} detail={data} defaultDate={defaultDate ?? ''} fields={fields} action={editing} units={context.data?.units}
          reasons={creating ? context.data?.reasons ?? [] : data?.reasons[editing] ?? []}
          host={host} onCancel={cancel} onReload={reload} onRefresh={read.refresh} onSaved={save}/> :
          source ? <><HistoryNotice label="返回变更日志" onReturn={() => update({ sourceVersion: undefined, side: undefined, tab: 'history' })} />
            {time && <Typography component="p"><time dateTime={time}>{recordedTime(time)}</time></Typography>}
            {current && <RateFacts record={current} currency={snapshot?.currency} frequency={snapshot?.frequency} precision={snapshot?.precision}/>}</> :
          historyTab ? <RateLog items={log.data?.items ?? []} link={(version, side) => {
            const next = new URLSearchParams(params);
            next.set('sourceVersion',version);
            if (side) next.set('side',side); else next.delete('side');
            return ratePath + '/' + id + '?' + next;
          }}/> : current && <RateFacts record={current} currency={data?.currency} frequency={data?.frequency} precision={data?.precision}/>}
      </ObjectDetailSheet>}
    </div></Box>;
}
