import { useEffect, useState } from 'react';
import { Alert, Button, Dialog, DialogActions, DialogContent, DialogTitle, Stack, TextField, Typography } from '@mui/material';
import { ApiError, sessionGuard, type Models } from '@chnhcm/api-client';
import { ObjectDetailField } from '@chnhcm/ui';
import * as staff from './api';

type Props = {
  field: staff.JobClassificationField;
  session: staff.Principal;
  body: staff.Input;
  value: string;
  originalLabel?: string;
  required: boolean;
  disabled: boolean;
  error?: string;
  onChange: (id: string) => void;
};

export function JobClassificationReference({ field, session, body, value, originalLabel, required, disabled, error, onChange }: Props) {
  const label = staff.fieldLabels[field];
  const short = field === 'jobFamilyId' ? '系列' : '职类';
  const inputId = 'staffing-' + field;
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const [retry, setRetry] = useState(0);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState('');
  const [result, setResult] = useState<Models['StaffingReferenceContext']>();
  const [selectedLabel, setSelectedLabel] = useState(originalLabel);

  // Only the proposed timeline affects eligibility. Unrelated form fields and
  // the other classification can change without restarting this search.
  const { action, objectId, expectedSourceVersion, recordId, expectedRecordRevision, anchorRecordId,
    effectiveDate, newEffectiveDate, operationType, classification, deactivationRecordId, deactivationRevision,
    reactivationRecordId, reactivationRevision, deactivationDate, reactivationDate } = body;
  const reference = { action, objectId, expectedSourceVersion, recordId, expectedRecordRevision, anchorRecordId,
    effectiveDate, newEffectiveDate, operationType, classification, deactivationRecordId, deactivationRevision,
    reactivationRecordId, reactivationRevision, deactivationDate, reactivationDate };
  const context = JSON.stringify(reference);
  useEffect(() => setPage(1), [context, search]);
  useEffect(() => {
    const controller = new AbortController();
    const sameSession = sessionGuard();
    const current = () => !controller.signal.aborted && sameSession();
    setBusy(true);
    setFailure('');
    setResult(undefined);
    const timer = setTimeout(async () => {
      try {
        const data = await staff.classificationReferences(field, session, {
          ...reference, [field]: value || undefined, referenceQ: search, referencePage: page,
        }, controller.signal);
        if (current()) {
          setResult(data);
          if (value) setSelectedLabel(data.selectedName);
        }
      } catch (e) {
        if (current()) {
          setFailure(e instanceof Error ? e.message : '候选暂不可用，请重试。');
          if (e instanceof ApiError && [401, 403].includes(e.status)) setSelectedLabel(undefined);
        }
      } finally {
        if (current()) setBusy(false);
      }
    }, 120);
    return () => { clearTimeout(timer); controller.abort(); };
  }, [context, field, search, page, open, retry, session.csrf, value]);

  return (
    <>
      <ObjectDetailField inputId={inputId} label={label} required={required} error={error ?? (result?.selectedAvailable === false ? '该日期下所选' + label + '不可用' : undefined)}>
        <Button id={inputId} disabled={disabled} onClick={() => setOpen(true)} aria-haspopup="dialog"
          sx={{ justifyContent: 'flex-start', textAlign: 'left', overflowWrap: 'anywhere' }}>
          {selectedLabel ?? (value ? '已选' + label : '请选择' + label)}
        </Button>
      </ObjectDetailField>
      <Dialog open={open} fullWidth maxWidth="sm" onClose={() => setOpen(false)} aria-labelledby={inputId + '-title'}>
        <DialogTitle id={inputId + '-title'}>选择{label}</DialogTitle>
        <DialogContent>
          <TextField fullWidth size="small" label={'查找' + label} value={search} onChange={e => setSearch(e.target.value)} sx={{ my: 1 }} />
          {failure && <Alert severity="error" action={<Button onClick={() => setRetry(n => n + 1)}>重试</Button>}>{failure}</Alert>}
          {busy ? <Typography role="status">正在读取…</Typography> : !result?.items.length && !failure ? (
            <Typography>没有可用{short}，请调整日期或搜索内容。</Typography>
          ) : (
            <Stack>
              {result?.items.map(item => (
                <Button key={item.id} sx={{ justifyContent: 'flex-start', textAlign: 'left' }} onClick={() => {
                  setSelectedLabel(item.name);
                  onChange(item.id);
                  setOpen(false);
                }}>{item.name}</Button>
              ))}
            </Stack>
          )}
          <Stack direction="row" useFlexGap flexWrap="wrap">
            <Button disabled={busy || page <= 1} onClick={() => setPage(n => n - 1)}>上一页{short}</Button>
            <Typography sx={{ alignSelf: 'center' }}>共 {result?.totalCount ?? 0} 项</Typography>
            <Button disabled={busy || page * 20 >= (result?.totalCount ?? 0)} onClick={() => setPage(n => n + 1)}>下一页{short}</Button>
          </Stack>
        </DialogContent>
        <DialogActions><Button onClick={() => setOpen(false)}>取消</Button></DialogActions>
      </Dialog>
    </>
  );
}
