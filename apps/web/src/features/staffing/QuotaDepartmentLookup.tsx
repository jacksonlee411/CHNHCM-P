import { useEffect, useState } from 'react';
import { Alert, Button, Dialog, DialogActions, DialogContent, DialogTitle, Stack, TextField, Typography } from '@mui/material';
import { api, sessionGuard, unwrap, type Models } from '@chnhcm/api-client';

type Props = { day: string; onClose: () => void; onSelect: (id: string) => void };

export function QuotaDepartmentLookup({ day, onClose, onSelect }: Props) {
  const [query, setQuery] = useState(''), [page, setPage] = useState(1), [retry, setRetry] = useState(0);
  const [response, setResponse] = useState<{ key: string; data?: Models['StaffingQuotaDepartmentOptions']; error?: string }>();
  const key = JSON.stringify([day, query, page, retry]);
  const current = response?.key === key ? response : undefined;
  useEffect(() => {
    const abort = new AbortController(), same = sessionGuard();
    const timer = setTimeout(() => {
      api.GET('/api/v1/staffing-quotas/department-options', {
        params: { query: { businessAt: day, q: query, page, pageSize: 20 } }, signal: abort.signal,
      }).then(unwrap).then(data => {
        if (!abort.signal.aborted && same()) setResponse({ key, data });
      }).catch(error => {
        if (!abort.signal.aborted && same()) setResponse({ key, error: error.message });
      });
    }, 120);
    return () => { clearTimeout(timer); abort.abort(); };
  }, [key, day, query, page]);
  return <Dialog open onClose={onClose} fullWidth maxWidth="sm" aria-labelledby="quota-department-lookup">
    <DialogTitle id="quota-department-lookup">查找部门</DialogTitle>
    <DialogContent dividers><Stack spacing={1}>
      <TextField autoFocus label="搜索部门编码或名称" value={query} onChange={e => { setQuery(e.target.value); setPage(1); }} />
      {current?.error ? <Alert severity="error" action={<Button onClick={() => setRetry(n => n + 1)}>重试</Button>}>{current.error}</Alert>
        : !current?.data ? <Typography role="status">正在查询…</Typography>
        : current.data.items.length ? current.data.items.map(d => <Button key={d.id} sx={{ justifyContent: 'flex-start', overflowWrap: 'anywhere' }}
          onClick={() => onSelect(d.id)}>{d.name}{d.code && '（' + d.code + '）'}</Button>)
        : <Typography>没有匹配的部门</Typography>}
      {current?.data && (page > 1 || current.data.hasMore) && <Stack direction="row" spacing={1} alignItems="center">
        <Button disabled={page === 1} onClick={() => setPage(n => n - 1)}>上一页</Button>
        <Typography>第 {page} 页</Typography>
        <Button disabled={!current.data.hasMore} onClick={() => setPage(n => n + 1)}>下一页</Button>
      </Stack>}
    </Stack></DialogContent>
    <DialogActions><Button onClick={onClose}>取消</Button></DialogActions>
  </Dialog>;
}
