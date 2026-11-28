import { Button, Table, TableCell, TableContainer, TableHead, TableRow } from '@mui/material';
import { Link } from 'react-router-dom';
import type { Models } from '@chnhcm/api-client';
import { DifferenceCells, OperationCells, OperationRows } from '../../shared/dated/OperationLog';
import { actionLabel, amountKeys, amountLabels, formLabels, money, statusLabels, type Action, type RateRecord } from './rates';

const labels = { effectiveDate: '生效日期', reason: '办理原因', status:'记录状态', form: '标准形式', ...amountLabels };
function values(r?: RateRecord): Record<string,string | undefined> {
  return { effectiveDate:r?.effectiveDate, reason:r?.reason, status:statusLabels[r?.status ?? ''], form:formLabels[r?.form ?? ''],
    ...Object.fromEntries(amountKeys.map(k => [k, r?.[k] === undefined ? undefined : money(r[k])])) };
}
export function RateLog({ items, link }: { items: Models['CompensationRateChange'][]; link: (version: string, side?: string) => string }) {
  return <TableContainer className="change-log-table"><Table size="small" aria-label="职级薪标变更日志"><TableHead><TableRow>
    {['记录时间','操作者','记录处理','变更字段','原值','新值','来源'].map(label => <TableCell key={label}>{label}</TableCell>)}
  </TableRow></TableHead>{items.map(c => {
    const before=values(c.before), after=values(c.after);
    const changed=Object.entries(labels).filter(([key]) => before[key] !== after[key]);
    const rows = changed.length ? changed : [['','']];
    return <OperationRows key={c.sourceVersion} source={c.sourceVersion} previous={c.previousSourceVersion} recordedAt={c.recordedAt} columns={7}
      details={<>{c.before && <Button component={Link} to={link(c.sourceVersion,'before')}>办理前资料</Button>}</>}>
      {toggle => rows.map(([key,label],i) => <TableRow key={key}>
        {!i && <OperationCells recordedAt={c.recordedAt} operator={c.operator} action={actionLabel[c.action as Action]} span={rows.length} />}
        <DifferenceCells difference={key ? {field:label,before:before[key] ?? '—',after:after[key] ?? '—'} : undefined}/>
        {!i && <TableCell rowSpan={rows.length} data-label="来源"><Button component={Link} to={link(c.sourceVersion)}>查看当次资料</Button>{toggle}</TableCell>}
      </TableRow>)}
    </OperationRows>;
  })}</Table></TableContainer>;
}
