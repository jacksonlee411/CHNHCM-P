import type { ReactNode } from 'react';
import { Accordion, AccordionDetails, AccordionSummary, Alert, Box, Button, Dialog, DialogActions, DialogContent, DialogTitle, Stack, Typography } from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';

export function UnsavedChangesDialog({ open, onContinue, onDiscard, pending = false }: { open: boolean; onContinue: () => void; onDiscard: () => void; pending?: boolean }) {
  return <Dialog open={open} onClose={onContinue} aria-labelledby="unsaved-title">
    <DialogTitle id="unsaved-title">{pending ? '保存结果尚未确认' : '尚有未保存修改'}</DialogTitle>
    <DialogContent>{pending ? '离开不会撤销已提交的保存操作。' : '离开将放弃尚未保存的修改。'}</DialogContent>
    <DialogActions sx={{ flexWrap: 'wrap', gap: 1 }}><Button onClick={onContinue}>{pending ? '留在此页' : '继续编辑'}</Button><Button color="error" onClick={onDiscard}>{pending ? '离开此页' : '放弃修改'}</Button></DialogActions>
  </Dialog>;
}
export function SourceExplanation({ children, defaultExpanded = false }: { children: ReactNode; defaultExpanded?: boolean }) {
  return <Accordion className="source-explanation" defaultExpanded={defaultExpanded}><AccordionSummary expandIcon={<ExpandMoreIcon />}>来源说明</AccordionSummary><AccordionDetails><Stack spacing={1}>{children}</Stack></AccordionDetails></Accordion>;
}
export function HistoryNotice({ onReturn, label = '返回当前资料' }: { onReturn: () => void; label?:string }) {
  return <Box className="history-notice" role="status"><span>正在查看只读历史资料</span><Button size="small" onClick={onReturn}>{label}</Button></Box>;
}
export function ChangeLogEntry({ summary, children, onViewSource }: { summary: ReactNode; children: ReactNode; onViewSource: () => void }) {
  return <Stack className="log-entry" spacing={1}><Typography>{summary}</Typography>{children}<Button sx={{ alignSelf: 'flex-start' }} onClick={onViewSource}>查看当次来源</Button></Stack>;
}
