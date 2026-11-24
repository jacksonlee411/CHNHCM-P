import { useEffect, useState } from 'react';
import { Alert, Button, Dialog, DialogActions, DialogContent, DialogTitle, Stack, TextField, Typography } from '@mui/material';
import { ApiError, sessionGuard, type Models } from '@chnhcm/api-client';
import { ObjectDetailField } from '@chnhcm/ui';
import * as staff from './api';

type Choice = Models['PositionReferenceContext']['items'][number];
type Props = {
  field: staff.PositionField;
  body: staff.Input;
  session: staff.Principal;
  id: string;
  original?: staff.RecordView;
  disabled: boolean;
  error?: string;
  onChange: (id: string) => void;
  onDenied: () => void;
};

const pageSize = 20;

export function PositionReference({ field, body, session, id, original, disabled, error, onChange, onDenied }: Props) {
  const label = staff.fieldLabels[field];
  const [open, setOpen] = useState(false), [query, setQuery] = useState(''), [page, setPage] = useState(1);
  const [picked, setPicked] = useState<Choice>(), [failure, setFailure] = useState(''), [retry, setRetry] = useState(0);
  const [result, setResult] = useState<Models['PositionReferenceContext']>();
  const signature = JSON.stringify({ ...body, referenceQ: query, referencePage: page, referencePageSize: pageSize });
  useEffect(() => {
    if (!open) return;
    const abort = new AbortController(), same = sessionGuard();
    setResult(undefined);
    setFailure('');
    const timer = setTimeout(() => {
      void staff.positionReferences(session, field, JSON.parse(signature), abort.signal).then(data => {
        if (!abort.signal.aborted && same()) setResult(data);
      }).catch(err => {
        if (abort.signal.aborted || !same()) return;
        if (err instanceof ApiError && (err.status === 401 || err.status === 403)) {
          setOpen(false);
          setPicked(undefined);
          onDenied();
        } else setFailure(err instanceof Error ? err.message : '候选暂不可用');
      });
    }, 120);
    return () => { clearTimeout(timer); abort.abort(); };
  }, [open, signature, session.csrf, field, retry]);
  const select = (choice?: Choice) => {
    setPicked(choice);
    onChange(choice?.id ?? '');
    setOpen(false);
  };
  const originalName = staff.positionName(original, field);
  const selected = picked?.id === id ? picked.name : original?.[field] === id ? originalName : undefined;
  const empty = originalName === '受限' ? '受限' : field === 'gradeId' ? '未指定职级' : `请选择${label}`;
  return <ObjectDetailField label={label} inputId={`staffing-${field}`} required={field !== 'gradeId'} error={error}>
    <Button id={`staffing-${field}`} disabled={disabled} aria-haspopup="dialog" aria-invalid={error ? true : undefined}
      aria-describedby={error ? `staffing-${field}-error` : undefined} onClick={() => {
      setQuery(''); setPage(1); setFailure(''); setResult(undefined); setOpen(true);
    }}>{id ? selected ?? `已选${label}` : empty}</Button>
    <Dialog open={open} onClose={() => setOpen(false)} aria-labelledby={`position-${field}-title`} fullWidth maxWidth="sm">
      <DialogTitle id={`position-${field}-title`}>选择{label}</DialogTitle>
      <DialogContent dividers><Stack spacing={1}>
        <TextField label={`搜索${label}编码或名称`} value={query} onChange={event => { setQuery(event.target.value); setPage(1); }} />
        {failure ? <Alert severity="error" action={<Button onClick={() => setRetry(count => count + 1)}>重试</Button>}>{failure}</Alert>
          : !result ? <Typography role="status">正在查询…</Typography>
          : result.items.length ? result.items.map(choice => <Button key={choice.id} sx={{justifyContent: 'flex-start', overflowWrap: 'anywhere'}} onClick={() => select(choice)}>
            {choice.name}{choice.code && `（${choice.code}）`}
          </Button>) : <Typography>{field === 'gradeId' ? '当前职务在拟定期间没有匹配的可用职级' : '拟定期间没有匹配的资料'}</Typography>}
        {result && (page > 1 || result.hasMore) && <Stack direction="row" spacing={1}>
          <Button disabled={page === 1} onClick={() => setPage(page - 1)}>上一页</Button>
          <Typography>第 {page} 页</Typography>
          <Button disabled={!result.hasMore} onClick={() => setPage(page + 1)}>下一页</Button>
        </Stack>}
      </Stack></DialogContent>
      <DialogActions>
        {field === 'gradeId' && <Button onClick={() => select()}>不指定职级</Button>}
        <Button onClick={() => setOpen(false)}>取消</Button>
      </DialogActions>
    </Dialog>
  </ObjectDetailField>;
}
