import { useLayoutEffect, useRef } from 'react';
import {
  Button,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import { Link, useLocation } from 'react-router-dom';
import { ObjectDetailField, ObjectDetailSection, SourceExplanation } from '@chnhcm/ui';
import { DifferenceCells, OperationCells, OperationRows } from '../../shared/dated/OperationLog';
import { dimensionNames, methods, natures, scopeLabel, type PlanChange, type PlanRecord } from './plans';
import { actionLabel, money, moneyUnit, rateAmounts, ratePath, statusLabels, type Action } from './rates';
import { valueSummary } from './PlanComponents';

export function PlanFacts({
  record,
  code,
  historical = false,
}: {
  record: PlanRecord;
  code?: string;
  historical?: boolean;
}) {
  const location = useLocation(),
    rows = record.components ?? [],
    restored = useRef(false);
  const back = (key: string) => {
    const params = new URLSearchParams(location.search);
    params.set('focus', key);
    return location.pathname + '?' + params;
  };
  useLayoutEffect(() => {
    if (restored.current) return;
    const key = new URLSearchParams(location.search).get('focus');
    const target = key && document.getElementById(key);
    if (target && target.matches('[data-plan-source]')) {
      restored.current = true;
      target.focus({ preventScroll: true });
      target.scrollIntoView({ block: 'nearest' });
    }
  }, [location.search, record]);
  return (
    <>
      <ObjectDetailSection title="计划资料">
        <div className="object-detail-columns">
          {code && <ObjectDetailField label="计划编码">{code}</ObjectDetailField>}
          {record.name && <ObjectDetailField label="计划名称">{record.name}</ObjectDetailField>}
          {record.description && <ObjectDetailField label="说明">{record.description}</ObjectDetailField>}
          {record.effectiveDate && <ObjectDetailField label="生效日期">{record.effectiveDate}</ObjectDetailField>}
          {record.status && <ObjectDetailField label="记录状态">{statusLabels[record.status]}</ObjectDetailField>}
          {record.reason && <ObjectDetailField label="办理原因">{record.reason}</ObjectDetailField>}
        </div>
      </ObjectDetailSection>
      {record.scope && (
        <ObjectDetailSection title="适用对象">
          {record.scope.mode === 'ENTERPRISE' ? (
            <Typography>企业通用</Typography>
          ) : (
            <>
              <div className="object-detail-columns">
                {Object.entries(dimensionNames).flatMap(([k, v]) =>
                  [false, true].map((exclude) => {
                    const members =
                      record.scope?.members?.filter((m) => m.dimension === k && !!m.exclude === exclude) ?? [];
                    return (
                      members.length > 0 && (
                        <ObjectDetailField key={k + exclude} label={(exclude ? '排除' : '适用') + v}>
                          {members.map((m) => [m.code, m.name].filter(Boolean).join(' ')).join('、')}
                        </ObjectDetailField>
                      )
                    );
                  }),
                )}
              </div>
              {record.scope.restricted && <Typography>部分适用对象受限</Typography>}
            </>
          )}
        </ObjectDetailSection>
      )}
      {record.components && (
        <ObjectDetailSection title="薪酬组成">
          <TableContainer>
            <Table size="small" aria-label="薪酬组成">
              <TableHead>
                <TableRow>
                  {['组成项', '性质', '取值方式', '制度约定'].map((k) => (
                    <TableCell key={k}>{k}</TableCell>
                  ))}
                </TableRow>
              </TableHead>
              <TableBody>
                {rows.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell>
                      {row.name}
                      <Typography variant="body2">{row.code}</Typography>
                    </TableCell>
                    <TableCell>{natures[row.nature]}</TableCell>
                    <TableCell>{methods[row.method]}</TableCell>
                    <TableCell>
                      {row.method !== 'GRADE_RATE' ? (
                        <>
                          {valueSummary(row, rows)}
                          {row.targets?.map((t) => (
                            <Typography variant="body2" key={t.gradeId ?? ''}>
                              {[
                                rows.flatMap((r) => r.rates ?? []).find((r) => r.gradeId === t.gradeId)?.code,
                                '参考额 ' + money(t.amount),
                                moneyUnit(row.currency, row.frequency),
                              ]
                                .filter(Boolean)
                                .join(' ')}
                            </Typography>
                          ))}
                        </>
                      ) : (
                        (row.rates ?? []).map((r) => {
                          const name = [r.code, r.name, moneyUnit(row.currency, row.frequency)]
                              .filter(Boolean)
                              .join(' '),
                            key = 'plan-source-' + row.id + '-' + r.rateId;
                          const current = new URLSearchParams({ back: back(key) }),
                            original = new URLSearchParams({ back: back(key + '-original') });
                          if (r.record?.recordId) current.set('selectedRecordId', r.record.recordId);
                          if (r.adopted) original.set('sourceVersion', r.adopted.sourceVersion);
                          return (
                            <Stack key={r.rateId} alignItems="flex-start">
                              {r.open && (!historical || r.adopted?.open) ? (
                                <Link
                                  id={key}
                                  data-plan-source
                                  to={
                                    ratePath +
                                    '/' +
                                    r.rateId +
                                    '?' +
                                    (historical
                                      ? new URLSearchParams({ ...Object.fromEntries(original), back: back(key) })
                                      : current)
                                  }
                                >
                                  {name}
                                </Link>
                              ) : (
                                <span>{name}</span>
                              )}
                              {r.record && (
                                <span>
                                  {[
                                    rateAmounts(r.record),
                                    r.record.effectiveDate && '自 ' + r.record.effectiveDate + ' 起',
                                  ]
                                    .filter(Boolean)
                                    .join(' · ')}
                                </span>
                              )}
                              {!historical && r.adopted?.open && (
                                <SourceExplanation
                                  defaultExpanded={
                                    new URLSearchParams(location.search).get('focus') === key + '-original'
                                  }
                                >
                                  <Link
                                    id={key + '-original'}
                                    data-plan-source
                                    to={ratePath + '/' + r.rateId + '?' + original}
                                  >
                                    查看保存时依据（{r.adopted.businessAt}）
                                  </Link>
                                </SourceExplanation>
                              )}
                            </Stack>
                          );
                        })
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </ObjectDetailSection>
      )}
    </>
  );
}

function logFields(record?: PlanRecord) {
  const values = new Map<string, string>();
  if (!record) return values;
  for (const [label, v] of [
    ['计划名称', record.name],
    ['说明', record.description],
    ['生效日期', record.effectiveDate],
    ['记录状态', statusLabels[record.status ?? '']],
    ['办理原因', record.reason],
    ['适用对象', scopeLabel(record.scope)],
  ])
    if (v) values.set(label!, v);
  const rows = record.components ?? [];
  if (rows.length) values.set('组成顺序', rows.map((r) => r.code).join('、'));
  for (const row of rows) {
    const prefix = row.code + ' · ';
    values.set(prefix + '名称', row.name);
    values.set(prefix + '性质', natures[row.nature]);
    values.set(prefix + '取值方式', methods[row.method]);
    values.set(prefix + '计量口径', moneyUnit(row.currency, row.frequency));
    const v = valueSummary(row, rows);
    if (v) values.set(prefix + '约定', v);
  }
  return values;
}
export function PlanLog({ items, link }: { items: PlanChange[]; link: (version: string, side?: string) => string }) {
  return (
    <TableContainer className="change-log-table">
      <Table size="small" aria-label="薪酬计划变更日志">
        <TableHead>
          <TableRow>
            {['记录时间', '操作者', '记录处理', '变更字段', '原值', '新值', '来源'].map((k) => (
              <TableCell key={k}>{k}</TableCell>
            ))}
          </TableRow>
        </TableHead>
        {items.map((c) => {
          const before = logFields(c.before),
            after = logFields(c.after),
            fields = [...new Set([...before.keys(), ...after.keys()])].filter((k) => before.get(k) !== after.get(k));
          if (!fields.length) fields.push('');
          return (
            <OperationRows
              key={c.sourceVersion}
              source={c.sourceVersion}
              recordedAt={c.recordedAt}
              columns={7}
              details={
                c.before && (
                  <Button component={Link} to={link(c.sourceVersion, 'before')}>
                    办理前资料
                  </Button>
                )
              }
            >
              {(toggle) =>
                fields.map((k, i) => (
                  <TableRow key={k}>
                    {!i && (
                      <OperationCells
                        recordedAt={c.recordedAt}
                        operator={c.operator}
                        action={actionLabel[c.action as Action]}
                        span={fields.length}
                      />
                    )}
                    <DifferenceCells
                      difference={
                        k ? { field: k, before: before.get(k) ?? '—', after: after.get(k) ?? '—' } : undefined
                      }
                    />
                    {!i && (
                      <TableCell rowSpan={fields.length} data-label="来源">
                        <Button component={Link} to={link(c.sourceVersion)}>
                          查看当次资料
                        </Button>
                        {toggle}
                      </TableCell>
                    )}
                  </TableRow>
                ))
              }
            </OperationRows>
          );
        })}
      </Table>
    </TableContainer>
  );
}
