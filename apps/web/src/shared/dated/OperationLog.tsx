import { useState, type ReactNode } from 'react';
import { Box, Button, Stack, TableBody, TableCell, TableRow, Typography } from '@mui/material';
import { Link } from 'react-router-dom';
import { recordedTime } from '@chnhcm/ui';
import './operation-log.css';
export type Operator = {
  subjectId: string;
  displayName?: string | null;
  login?: string | null;
  kind: string;
  identityEvidence: string;
};
export type Value = {
  members?: { gradeId: string; name: string; code?: string; position?: number }[];
  kind: string;
  text?: string | null;
  label?: string | null;
  code?: string | null;
  start?: string | null;
  endInclusive?: string | null;
};
export type Operation = {
  sourceVersion: string;
  previousSourceVersion?: string | null;
  recordedAt?: string;
  operator?: Operator;
  action: string;
  items: {
    recordId: string;
    changeKind: string;
    beforeRef?: { sourceVersion: string; revision: string } | null;
    afterRef?: { sourceVersion: string; revision: string } | null;
    beforeContext?: {
      effectiveDate?: string | null;
      name?: string | null;
      typeLabel?: string | null;
      reasonLabel?: string | null;
    } | null;
    afterContext?: {
      effectiveDate?: string | null;
      name?: string | null;
      typeLabel?: string | null;
      reasonLabel?: string | null;
    } | null;
    diffs: { field: string; before: Value; after: Value }[];
  }[];
};
const statusLabels: Record<string, string> = { active: '启用', inactive: '停用' };
export function operatorText(operator: Operator) {
  if (operator.identityEvidence === 'subject_id_only') return operator.subjectId;
  const identity = operator.displayName ? `${operator.displayName}（${operator.login}）` : operator.login;
  return `${identity}${operator.kind === 'service' ? '（服务主体）' : ''}`;
}
export function OperationSummary({ summary }: { summary: { operator: Operator | string; recordedAt: string } }) {
  const { operator, recordedAt } = summary;
  return (
    <Typography className="operation-summary" variant="body2" color="text.secondary">
      <span>
        {typeof operator !== 'string' && operator.identityEvidence === 'subject_id_only' ? '操作者标识' : '操作者'}：
        {typeof operator === 'string' ? operator : operatorText(operator)}
      </span>
      <span>操作时间：<time dateTime={recordedAt}>{recordedTime(recordedAt)}</time></span>
    </Typography>
  );
}
function valueText(v: Value, emptyReferenceLabel: string) {
  if (v.kind === 'members') {
    if (!v.members?.length) return '未配置可用职级';
    return v.members
      .map((m) => {
        const label = m.code ? `${m.code} · ${m.name}` : m.name;
        return m.position ? `${m.position}. ${label}` : label;
      })
      .join('；');
  }
  if (v.kind === 'missing') return '未登记';
  if (v.kind === 'empty') return '未提供';
  if (v.kind === 'period') return v.endInclusive ? `${v.start} 至 ${v.endInclusive}` : `自 ${v.start} 起`;
  if (v.kind === 'classification') return v.label;
  if (v.kind === 'reference') return v.label || (v.text ? '名称无读取权限' : emptyReferenceLabel);
  if (v.kind === 'state') return statusLabels[v.code ?? ''] ?? v.code;
  return v.text;
}
export function OperationCells({
  recordedAt,
  operator,
  action,
  span,
  operatorLabel = '操作者',
  actionLabel = '记录处理',
}: {
  recordedAt: string;
  operator: ReactNode;
  action: string;
  span: number;
  operatorLabel?: string;
  actionLabel?: string;
}) {
  return (
    <>
      <TableCell rowSpan={span} data-label="记录时间">
        <time dateTime={recordedAt}>{recordedTime(recordedAt)}</time>
      </TableCell>
      <TableCell rowSpan={span} data-label={operatorLabel}>
        {operator}
      </TableCell>
      <TableCell rowSpan={span} data-label={actionLabel}>
        {action}
      </TableCell>
    </>
  );
}

export function DifferenceCells({
  difference,
  fieldLabel = '变更字段',
}: {
  fieldLabel?: string;
  difference?: { field: ReactNode; before: ReactNode; after: ReactNode };
}) {
  return difference ? (
    <>
      <TableCell data-label={fieldLabel}>{difference.field}</TableCell>
      <TableCell data-label="原值">{difference.before}</TableCell>
      <TableCell data-label="新值">{difference.after}</TableCell>
    </>
  ) : (
    <TableCell colSpan={3}>无可展示的字段变化</TableCell>
  );
}

export function OperationRows({
  source,
  previous,
  sourceIdentity,
  recordedAt,
  columns,
  details,
  children,
}: {
  source: string;
  previous?: string | null;
  sourceIdentity?: ReactNode;
  recordedAt: string;
  columns: number;
  details?: ReactNode;
  children: (sourceButton: ReactNode) => ReactNode;
}) {
  const [expanded, setExpanded] = useState(false);
  const panelId = `source-${source}`;
  const button = (
    <Button
      size="small"
      aria-expanded={expanded}
      aria-controls={panelId}
      onClick={() => setExpanded((v) => !v)}
    >
      来源明细
    </Button>
  );
  return (
    <TableBody className="change-operation" data-source-version={source}>
      {children(button)}
      {expanded && (
        <TableRow>
          <TableCell colSpan={columns}>
            <Box id={panelId} className="change-source-detail">
              {sourceIdentity ?? <>
                <div>来源版本：{source}</div>
                <div>前一来源：{previous ?? '无'}</div>
              </>}
              <div>原始时刻：{recordedAt}</div>
              {details}
            </Box>
          </TableCell>
        </TableRow>
      )}
    </TableBody>
  );
}

export function OperationGroup({
  change: c,
  href,
  remember,
  fieldLabels,
  actionLabels,
  emptyReferenceLabel = '未提供',
  showAudit = true,
}: {
  change: Operation;
  href: (source: string, record?: string) => string;
  remember: (id: string, event: React.MouseEvent) => void;
  fieldLabels: Record<string, string>;
  actionLabels: Record<string, string>;
  emptyReferenceLabel?: string;
  showAudit?: boolean;
}) {
  const [expanded, setExpanded] = useState(false);
  const span = c.items.reduce((n, item) => n + Math.max(1, item.diffs.length), 0) || 1;
  return (
    <TableBody className="change-operation" data-source-version={c.sourceVersion}>
      {c.items.map((item, itemIndex) => {
        const context = item.afterContext ?? item.beforeContext;
        const rows = item.diffs.length ? item.diffs : [null];
        return rows.map((diff, index) => (
          <TableRow key={`${item.recordId}-${diff?.field ?? 'none'}`}>
            {itemIndex === 0 && index === 0 && (
              <>
                {showAudit && (
                  <>
                    <TableCell rowSpan={span} data-label="记录时间">
                      {c.recordedAt && <time dateTime={c.recordedAt}>{recordedTime(c.recordedAt)}</time>}
                    </TableCell>
                    <TableCell
                      rowSpan={span}
                      data-label={
                        c.operator?.identityEvidence === 'subject_id_only' ? '操作者标识' : '操作者'
                      }
                    >
                      {c.operator?.identityEvidence === 'subject_id_only' && (
                        <span>
                          操作者标识：
                          <br />
                        </span>
                      )}
                      {c.operator && operatorText(c.operator)}
                    </TableCell>
                  </>
                )}
                <TableCell rowSpan={span} data-label="记录处理">
                  {actionLabels[c.action]}
                </TableCell>
              </>
            )}
            {index === 0 && (
              <TableCell rowSpan={rows.length} data-label="生效记录">
                <Stack spacing={0.5}>
                  {context?.effectiveDate && <span>{context.effectiveDate}</span>}
                  {context?.name && <span>{context.name}</span>}
                  {context?.typeLabel && (
                    <span>
                      {context.typeLabel}
                      {context.reasonLabel ? ` · ${context.reasonLabel}` : ''}
                    </span>
                  )}
                  {!context?.typeLabel && context?.reasonLabel && <span>{context.reasonLabel}</span>}
                  <Typography variant="caption" color="text.secondary">
                    {item.changeKind === 'derived' ? '随期间变化' : '直接修改'}
                  </Typography>
                </Stack>
              </TableCell>
            )}
            {diff ? (
              <>
                <TableCell data-label="变更字段">{fieldLabels[diff.field]}</TableCell>
                <TableCell data-label="原值">{valueText(diff.before, emptyReferenceLabel)}</TableCell>
                <TableCell data-label="新值">{valueText(diff.after, emptyReferenceLabel)}</TableCell>
              </>
            ) : (
              <TableCell colSpan={3}>无可展示的字段变化</TableCell>
            )}
            {index === 0 && (
              <TableCell rowSpan={rows.length} data-label="查看">
                <Stack alignItems="flex-start" spacing={1}>
                  {item.afterRef ? (
                    <Link
                      id={`change-${c.sourceVersion}-${item.recordId}-after`}
                      to={href(item.afterRef.sourceVersion, item.recordId)}
                      onClick={(e) => remember(e.currentTarget.id, e)}
                    >
                      查看当次资料
                    </Link>
                  ) : (
                    item.beforeRef && (
                      <>
                        <Link
                          id={`change-${c.sourceVersion}-${item.recordId}-before`}
                          to={href(item.beforeRef.sourceVersion, item.recordId)}
                          onClick={(e) => remember(e.currentTarget.id, e)}
                        >
                          查看删除前原件
                        </Link>
                        <Link
                          id={`change-${c.sourceVersion}-${item.recordId}-deleted`}
                          to={href(c.sourceVersion)}
                          onClick={(e) => remember(e.currentTarget.id, e)}
                        >
                          查看删除后版本
                        </Link>
                      </>
                    )
                  )}
                  {itemIndex === 0 && (
                    <Button
                      size="small"
                      aria-expanded={expanded}
                      aria-controls={`source-${c.sourceVersion}`}
                      onClick={() => setExpanded((v) => !v)}
                    >
                      来源明细
                    </Button>
                  )}
                </Stack>
              </TableCell>
            )}
          </TableRow>
        ));
      })}
      {!c.items.length && (
        <TableRow>
          {showAudit && (
            <>
              <TableCell>{c.recordedAt && recordedTime(c.recordedAt)}</TableCell>
              <TableCell>{c.operator && operatorText(c.operator)}</TableCell>
            </>
          )}
          <TableCell>{actionLabels[c.action]}</TableCell>
          <TableCell colSpan={5}>无可展示的字段变化</TableCell>
        </TableRow>
      )}
      {expanded && (
        <TableRow>
          <TableCell colSpan={showAudit ? 8 : 6}>
            <Box id={`source-${c.sourceVersion}`} className="change-source-detail">
              <div>来源版本：{c.sourceVersion}</div>
              <div>前一来源：{c.previousSourceVersion ?? '无'}</div>
              {c.recordedAt && <div>原始时刻：{c.recordedAt}</div>}
              {c.operator && <div>操作者标识：{c.operator.subjectId}</div>}
              {c.items.map((item) => (
                <div key={item.recordId}>
                  记录标识：{item.recordId}；前修订：{item.beforeRef?.revision ?? '无'}；后修订：
                  {item.afterRef?.revision ?? '无'}
                </div>
              ))}
            </Box>
          </TableCell>
        </TableRow>
      )}
    </TableBody>
  );
}
