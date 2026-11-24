import { useEffect, useRef, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  IconButton,
  Menu,
  MenuItem,
  Table,
  TableCell,
  TableContainer,
  TableHead,
  TablePagination,
  TableRow,
  Typography,
} from '@mui/material';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import { useLocation, useSearchParams } from 'react-router-dom';
import { api, unwrap, ApiError, sessionGuard, type Models } from '@chnhcm/api-client';
import { ColumnFilterDialog, operatorsFor, type FilterColumn, type FilterRequest } from '../../shared/dated/ColumnFilter';
import { OperationGroup } from '../../shared/dated/OperationLog';
import { labels as quotaLabels, quotaAmountFields } from './quotaApi';
import * as staff from './api';
export const logKeys = [
  'logPage',
  'logPageSize',
  'logThrough',
  'logFilterField',
  'logFilterOperator',
  'logFilterValue',
  'logReturn',
];
export function logReturnHref(value: string | null, pathname: string) {
  if (!value) return;
  try {
    const u = new URL(value, window.location.origin);
    if (
      u.origin === window.location.origin &&
      u.pathname === pathname &&
      u.searchParams.get('tab') === 'history'
    )
      return u.pathname + u.search;
  } catch {
    /* invalid external return */
  }
}
const positions = new Map<string, { id: string; top: number; left: number }>();
function scrollNode() {
  const node = document.querySelector<HTMLElement>('[data-page-scroll]');
  return node && getComputedStyle(node).overflowY === 'auto' ? node : document.scrollingElement;
}
export function StaffingChangeLog({
  kind: k,
  quotaTarget = 'position',
  id,
  onReadFields,
  onDenied,
}: {
  kind: staff.Kind | 'staffingquota';
  quotaTarget?: keyof typeof quotaAmountFields;
  id: string;
  onReadFields: (fields: string[]) => void;
  onDenied: () => void;
}) {
  const [params, setParams] = useSearchParams(),
    location = useLocation(),
    [result, setResult] = useState<{ url: string; log: Models['StaffingLog'] }>(),
    [error, setError] = useState(''),
    [loading, setLoading] = useState(true),
    [retry, setRetry] = useState(0),
    [menu, setMenu] = useState<HTMLElement | null>(null),
    [filter, setFilter] = useState<FilterRequest>();
  const callbacks = useRef({ onReadFields, onDenied });
  callbacks.current = { onReadFields, onDenied };
  const query: Models['StaffingLogQuery'] = {
    page: Number(params.get('logPage') ?? 1),
    pageSize: Number(params.get('logPageSize') ?? 20),
    contextSourceVersion: params.get('sourceVersion') ?? undefined,
    throughSourceVersion: params.get('logThrough') ?? undefined,
    filterField: params.get('logFilterField') ?? undefined,
    filterOperator: params.get('logFilterOperator') ?? undefined,
    filterValue: params.get('logFilterValue') ?? undefined,
  };
  const url = location.pathname + location.search;
  const data = result?.url === url ? result.log : undefined;
  const patch = (values: Record<string, string>) => {
    const next = new URLSearchParams(params);
    for (const [key, v] of Object.entries(values))
      if (v) next.set(key, v);
      else next.delete(key);
    setParams(next, { replace: true });
  };
  useEffect(() => {
    const a = new AbortController(),
      same = sessionGuard();
    const ownsLocation = () =>
      !a.signal.aborted && same() && window.location.pathname + window.location.search === url;
    setResult(undefined);
    setLoading(true);
    setError('');
    const request =
      k === 'staffingquota'
        ? api
            .GET('/api/v1/staffing-quotas/history/{id}/changes', {
              params: { path: { id }, query },
              signal: a.signal,
            })
            .then(unwrap)
        : staff.log(k, id, query, a.signal);
    void request
      .then((v) => {
        if (!ownsLocation()) return;
        callbacks.current.onReadFields(v.detailReadFields);
        if (!query.throughSourceVersion) {
          patch({ logThrough: v.throughSourceVersion });
          return;
        }
        setResult({ url, log: v });
        setLoading(false);
      })
      .catch((e) => {
        if (!ownsLocation()) return;
        setError(e instanceof Error ? e.message : '日志暂不可读');
        setLoading(false);
        if (e instanceof ApiError && [401, 403, 404].includes(e.status)) callbacks.current.onDenied();
      });
    return () => a.abort();
  }, [k, id, url, retry]);
  useEffect(() => {
    if (!data) return;
    const saved = positions.get(url);
    if (!saved) return;
    const frame = requestAnimationFrame(() => {
      const target = document.getElementById(saved.id);
      if (
        target &&
        (document.activeElement === document.body || document.activeElement?.matches('[data-page-scroll]'))
      ) {
        target.focus({ preventScroll: true });
        const scroll = scrollNode();
        if (scroll) scroll.scrollTop = saved.top;
        const table = document.querySelector('.change-log-table');
        if (table) table.scrollLeft = saved.left;
      }
      positions.delete(url);
    });
    return () => cancelAnimationFrame(frame);
  }, [data, url]);
  const quota = k === 'staffingquota';
  const actionLabels = quota ? { ...quotaLabels, allocate: '同日调剂' } : staff.operationLabel;
  const audit = !quota || data?.readFields.includes('audit') === true;
  const labels: Record<string, string> = quota
    ? { quota: quotaAmountFields[quotaTarget], reason: '调整原因', effectiveDate: '生效日期', period: '适用期间' }
    : { ...staff.fieldLabels, period: '适用期间', operationType: '操作类型', operationReason: '操作原因' };
  const columns: FilterColumn[] = [
    ...(audit
      ? [
          { field: 'recordedAt', label: '记录时间（按日期）' },
          { field: 'actor', label: '操作者' },
        ]
      : []),
    {
      field: 'action',
      label: '记录处理',
      values: Object.entries(actionLabels)
        .filter(([key]) => key !== 'CHANGE')
        .map(([value, label]) => ({ value, label })),
    },
    ...(data?.readFields.includes('effectiveDate') ? [{ field: 'effectiveDate', label: '生效记录' }] : []),
    {
      field: 'changedField',
      label: '变更字段',
      values: Object.entries(labels)
        .filter(([field]) => data?.readFields.includes(field === 'period' ? 'effectiveDate' : field))
        .map(([value, label]) => ({ value, label })),
    },
  ];
  const appliedColumn = columns.find((column) => column.field === query.filterField);
  const operator = appliedColumn && operatorsFor(appliedColumn.field).find((op) => op.value === query.filterOperator);
  const values = query.filterOperator === 'isAnyOf' ? query.filterValue?.split(',') : [query.filterValue];
  const displayValues = values?.map((value) => appliedColumn?.values
    ? appliedColumn.values.find((option) => option.value === value)?.label
    : value);
  const href = (source: string, record?: string) => {
    const p = new URLSearchParams(params);
    logKeys.forEach((key) => p.delete(key));
    p.set('tab', 'basic');
    p.set('sourceVersion', source);
    if (record) p.set('selectedRecordId', record);
    else p.delete('selectedRecordId');
    p.set('logReturn', url);
    return location.pathname + '?' + p;
  };
  const remember = (focus: string, e: React.MouseEvent) => {
    if (e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
    positions.set(url, {
      id: focus,
      top: scrollNode()?.scrollTop ?? 0,
      left: document.querySelector('.change-log-table')?.scrollLeft ?? 0,
    });
  };
  const close = () => {
    const source = filter?.source;
    setFilter(undefined);
    requestAnimationFrame(() => source?.focus());
  };
  return (
    <Box className="change-log">
      <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', flexWrap: 'wrap' }}>
        <Typography component="h2" variant="subtitle2">
          变更日志
        </Typography>
        <IconButton aria-label="日志更多操作" onClick={(e) => setMenu(e.currentTarget)}>
          <MoreVertIcon />
        </IconButton>
        {data && appliedColumn && operator && displayValues?.length && displayValues.every(Boolean) && (
          <Typography variant="body2">
            {appliedColumn.label} {operator.label} {displayValues.join('、')}
          </Typography>
        )}
      </Box>
      <Menu anchorEl={menu} open={!!menu} onClose={() => setMenu(null)}>
        <MenuItem
          disabled={!data}
          onClick={() => {
            setFilter({ source: menu! });
            setMenu(null);
          }}
        >
          列筛选…
        </MenuItem>
        <MenuItem
          onClick={() => {
            patch({ logThrough: '', logPage: '1' });
            setRetry((n) => n + 1);
            setMenu(null);
          }}
        >
          刷新日志
        </MenuItem>
      </Menu>
      {error && (
        <Alert severity="error" action={<Button onClick={() => setRetry((n) => n + 1)}>重试</Button>}>
          {error}
        </Alert>
      )}
      {loading && <Typography role="status">正在读取日志…</Typography>}
      {data && (
        <>
          <TableContainer className="change-log-table" tabIndex={0} aria-label="变更日志表格">
            <Table size="small">
              <TableHead>
                <TableRow>
                  {[
                    ...(audit ? ['记录时间', '操作者'] : []),
                    '记录处理',
                    '生效记录',
                    '变更字段',
                    '原值',
                    '新值',
                    '查看',
                  ].map((label) => (
                    <TableCell key={label}>{label}</TableCell>
                  ))}
                </TableRow>
              </TableHead>
              {data.items.map((change) => (
                <OperationGroup
                  key={change.sourceVersion}
                  change={change}
                  showAudit={audit}
                  fieldLabels={labels}
                  actionLabels={actionLabels}
                  href={href}
                  remember={remember}
                />
              ))}
            </Table>
          </TableContainer>
          {!data.items.length && (
            <Typography role="status">
              {query.filterField ? '没有符合条件的变更日志' : '没有可读取的变更日志'}
            </Typography>
          )}
          <TablePagination
            component="div"
            count={data.total}
            page={data.page - 1}
            rowsPerPage={data.pageSize}
            rowsPerPageOptions={[20, 50, 100]}
            labelRowsPerPage="每页操作"
            labelDisplayedRows={({ from, to, count }) => `${from}–${to} / ${count}`}
            getItemAriaLabel={(type) => (type === 'next' ? '下一页日志' : '上一页日志')}
            onPageChange={(_, page) => patch({ logPage: String(page + 1) })}
            onRowsPerPageChange={(e) => patch({ logPage: '1', logPageSize: e.target.value })}
          />
        </>
      )}
      {filter && (
        <ColumnFilterDialog
          request={filter}
          columns={columns}
          applied={{
            items: query.filterField
              ? [
                  {
                    field: query.filterField,
                    operator: query.filterOperator ?? '',
                    value:
                      query.filterOperator === 'isAnyOf' ? query.filterValue?.split(',') : query.filterValue,
                  },
                ]
              : [],
          }}
          onClose={close}
          onApply={(v) =>
            patch({
              logFilterField: v.filterField,
              logFilterOperator: v.filterOperator,
              logFilterValue: v.filterValue,
              logPage: '1',
            })
          }
        />
      )}
    </Box>
  );
}
