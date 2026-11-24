import { useEffect, useRef, useState } from 'react';
import { Alert, Box, Button, Checkbox, Dialog, DialogActions, DialogContent, DialogTitle, FormControlLabel, Stack, TextField, Typography } from '@mui/material';
import { ObjectDetailField, businessDate } from '@chnhcm/ui';
import { ApiError, sessionGuard, type Models } from '@chnhcm/api-client';
import * as staff from './api';

type Member = Models['StaffingGradeMember'];
export function GradeMemberList({ members, restricted, ordered }: { members?: Member[]; restricted?: boolean; ordered: boolean }) {
  if (restricted) return <Typography>职级资料受限</Typography>;
  if (!members?.length) return <Typography>{ordered ? '暂无可读职级成员' : '未配置可用职级'}</Typography>;
  return <Box component={ordered ? 'ol' : 'ul'} sx={{ m: 0, pl: 2.5, overflowWrap: 'anywhere' }}>
    {members.map(m => <li key={m.gradeId}>
      {m.code ? `${m.code} · ` : ''}{m.name}
      {m.adopted && <Typography component="span" display="block" variant="body2" color="text.secondary">采用日期：{businessDate(m.adopted.businessAt)}</Typography>}
    </li>)}
  </Box>;
}
export function GradeMembersEditor({ kind, session, body, ids, onChange, onRestricted, disabled, error, restricted }: {
  kind: 'job' | 'gradeladder'; session: staff.Principal; body: staff.Input; ids: string[];
  onChange: (ids: string[]) => void; onRestricted: () => void;
  disabled: boolean; error?: string; restricted?: boolean;
}) {
  const [open, setOpen] = useState(false), [query, setQuery] = useState(''), [page, setPage] = useState(1);
  const [data, setData] = useState<Models['StaffingGradeReferenceContext']>(), [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState(''), [retry, setRetry] = useState(0);
  const [members, setMembers] = useState<Member[]>([]);
  const generation = useRef(0);
  const signature = JSON.stringify(body);
  const change = useRef(onRestricted); change.current = onRestricted;
  useEffect(() => setPage(1), [signature, query]);
  useEffect(() => {
    if (restricted) { setData(undefined); setMembers([]); return; }
    const a = new AbortController(), same = sessionGuard(), own = ++generation.current;
    setBusy(true); setFailure(''); setData(undefined); setMembers([]);
    const timer = setTimeout(() => {
      void staff.gradeReferences(kind, session, { ...body, referenceQ: query, referencePage: page }, a.signal).then(v => {
        if (a.signal.aborted || !same() || own !== generation.current) return;
        setData(v); setMembers(v.selected);
      }).catch(e => {
        if (a.signal.aborted || !same() || own !== generation.current) return;
        setFailure(e instanceof Error ? e.message : '职级候选暂不可用');
        if (e instanceof ApiError && [401, 403].includes(e.status)) { setOpen(false); change.current(); }
      }).finally(() => { if (!a.signal.aborted && same() && own === generation.current) setBusy(false); });
    }, 120);
    return () => { clearTimeout(timer); a.abort(); };
  }, [kind, session.csrf, signature, query, page, retry, restricted]);
  const move = (index: number, delta: number) => { const next = [...ids]; [next[index], next[index + delta]] = [next[index + delta], next[index]]; onChange(next); };
  return <ObjectDetailField label={kind === 'job' ? '可用职级' : '职级成员（低至高）'} inputId="staffing-gradeIds" required={kind === 'gradeladder'} error={error}>
    <Stack id="staffing-gradeIds" tabIndex={-1} spacing={1}>
      {restricted ? <Typography>职级资料受限</Typography> : <>
        {!ids.length && <Typography>{kind === 'job' ? '未配置可用职级' : '请选择至少一个职级'}</Typography>}
        <Box component={kind === 'gradeladder' ? 'ol' : 'ul'} sx={{ m: 0, pl: 2.5 }}>
          {ids.map((id, index) => { const member = members.find(m => m.gradeId === id); return <Box component="li" key={id} sx={{ mb: 1, overflowWrap: 'anywhere' }}>
            <Typography sx={{ fontSize: 'inherit', lineHeight: 'inherit' }}>{member ? `${member.code ? `${member.code} · ` : ''}${member.name}` : busy ? '正在读取职级…' : '职级资料暂不可读'}</Typography>
            <Stack direction="row" useFlexGap flexWrap="wrap" gap={0.5}>
              {kind === 'gradeladder' && <><Button size="small" disabled={disabled || index === 0} aria-label={`上移第${index + 1}个职级`} onClick={() => move(index, -1)}>上移</Button><Button size="small" disabled={disabled || index === ids.length - 1} aria-label={`下移第${index + 1}个职级`} onClick={() => move(index, 1)}>下移</Button></>}
              <Button size="small" disabled={disabled} aria-label={`移除第${index + 1}个职级`} onClick={() => onChange(ids.filter(x => x !== id))}>移除</Button>
            </Stack>
          </Box>; })}
        </Box>
        <Button disabled={disabled} aria-haspopup="dialog" onClick={() => setOpen(true)}>选择职级</Button>
        {failure && <Alert severity="error" action={<Button onClick={() => setRetry(x => x + 1)}>重试</Button>}>{failure}</Alert>}
        {data?.selectedUnavailable && <Alert severity="warning">部分已选职级在当前期间不可用，请调整日期或重新选择。</Alert>}
      </>}
    </Stack>
    <Dialog open={open} onClose={() => setOpen(false)} fullWidth maxWidth="sm" aria-labelledby="staffing-grades-title">
      <DialogTitle id="staffing-grades-title">选择职级</DialogTitle>
      <DialogContent>
        <TextField label="查找职级" size="small" fullWidth value={query} onChange={e => setQuery(e.target.value)} sx={{ my: 1 }} />
        {failure && <Alert severity="error" action={<Button onClick={() => setRetry(x => x + 1)}>重试</Button>}>{failure}</Alert>}
        {busy ? <Typography role="status">正在读取…</Typography> : <Stack>{data?.items.map(v => <FormControlLabel key={v.id} sx={{ m: 0, overflowWrap: 'anywhere' }} label={`${v.code ? `${v.code} · ` : ''}${v.name}`} control={<Checkbox checked={ids.includes(v.id)} onChange={(_, checked) => onChange(checked ? [...ids, v.id] : ids.filter(id => id !== v.id))} />} />)}</Stack>}
        {!busy && !failure && !data?.items.length && <Typography>没有可用职级，请调整日期或搜索内容。</Typography>}
        <Stack direction="row" useFlexGap flexWrap="wrap" alignItems="center">
          <Button disabled={busy || page <= 1} onClick={() => setPage(x => x - 1)}>上一页职级</Button>
          <Typography>共 {data?.totalCount ?? 0} 项</Typography>
          <Button disabled={busy || page * 20 >= (data?.totalCount ?? 0)} onClick={() => setPage(x => x + 1)}>下一页职级</Button>
        </Stack>
      </DialogContent>
      <DialogActions><Button onClick={() => setOpen(false)}>完成选择</Button></DialogActions>
    </Dialog>
  </ObjectDetailField>;
}
