import { useState } from 'react';
import {
  Button,
  IconButton,
  Stack,
  Table,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import FilterAltOutlined from '@mui/icons-material/FilterAltOutlined';
import {
  OperationRows,
  OperationCells,
  DifferenceCells,
} from '../../shared/dated/OperationLog';
import {
  ColumnFilterDialog,
  type FilterColumn,
  type FilterRequest,
} from '../../shared/dated/ColumnFilter';
import { WorkspacePager, type Change } from './GrantWorkspace';
import {
  label,
  targetName,
  type Area,
  type Definition,
  type View,
} from './management';

const operations = {
  accounts: [
    'account.create',
    'account.enable',
    'account.disable',
    'account.move',
    'account.reset',
    'admin.bootstrap',
    'admin.recover',
    'admin.handover',
    'admin.handover.end',
  ],
  roles: ['version.publish', 'version.disable'],
  grants: [
    'grant.issue',
    'grant.adopt',
    'grant.revoke',
    'mandate.issue',
    'mandate.revoke',
  ],
};

export function SecurityHistory({
  area,
  v,
  def,
  params,
  change,
  follow,
}: {
  area: Area;
  v: View;
  def?: Definition;
  params: URLSearchParams;
  change: Change;
  follow?: (target: string, mandate: boolean, event: string) => void;
}) {
  const [request, setRequest] = useState<FilterRequest>();
  const actionName = (op: string) =>
    op === 'version.publish' && def?.number === 1
      ? '新建' + label(def.kind)
      : label(op);
  const columns: FilterColumn[] = [
    {
      field: 'recordedAt',
      label: '记录时间',
      operators: [{ value: 'is', label: '日期等于' }],
    },
    {
      field: 'actor',
      label: '操作者',
      operators: [{ value: 'contains', label: '包含' }],
    },
    {
      field: 'action',
      label: '记录处理',
      operators: [{ value: 'is', label: '等于' }],
      values: operations[area].map((value) => ({
        value,
        label: actionName(value),
      })),
    },
  ];
  const field = params.get('logField'),
    value = params.get('logValue') ?? '';
  const column = columns.find((c) => c.field === field);
  const model = {
    items: column
      ? [{ field: column.field, operator: column.operators![0].value, value }]
      : [],
  };
  return (
    <div className="change-log">
      <Stack
        direction="row"
        useFlexGap
        flexWrap="wrap"
        alignItems="center"
        spacing={1}
      >
        <Button
          size="small"
          id="security-log-filter"
          onClick={(e) => setRequest({ source: e.currentTarget })}
        >
          列筛选…
        </Button>
        {column && (
          <Button
            size="small"
            onClick={(e) =>
              setRequest({ field: column.field, source: e.currentTarget })
            }
          >
            {column.label} ·{' '}
            {column.field === 'action' ? actionName(value) : value}
          </Button>
        )}
        {params.get('eventFor') && (
          <Button size="small" onClick={() => change({ eventFor: '', gp: '' })}>
            查看全部变更
          </Button>
        )}
      </Stack>
      <TableContainer
        className="change-log-table"
        tabIndex={0}
        aria-label="变更日志表格"
      >
        <Table size="small" aria-label="变更日志">
          <TableHead>
            <TableRow>
              {columns.map((c) => (
                <TableCell scope="col" key={c.field}>
                  {c.label}
                  <IconButton
                    size="small"
                    id={'security-log-' + c.field}
                    aria-label={c.label + '列筛选'}
                    aria-haspopup="dialog"
                    onClick={(e) =>
                      setRequest({ field: c.field, source: e.currentTarget })
                    }
                  >
                    <FilterAltOutlined fontSize="inherit" />
                  </IconButton>
                </TableCell>
              ))}
              <TableCell scope="col">变更内容</TableCell>
              <TableCell scope="col">原值</TableCell>
              <TableCell scope="col">新值</TableCell>
              <TableCell scope="col">查看</TableCell>
            </TableRow>
          </TableHead>
          {v.events?.map((e) => (
            <OperationRows
              key={e.id}
              source={e.id}
              columns={7}
              recordedAt={e.at}
              sourceIdentity={
                <>
                  <div>事件编号：{e.id}</div>
                  <div>提交编号：{e.request}</div>
                </>
              }
              details={
                <>
                  <div>办理目标：{e.target}</div>
                  {e.reason && <div>办理事由：{e.reason}</div>}
                  {follow && (e.before || e.after) && (
                    <div>
                      业务范围名称按当前可见资料显示，版本取自当时的修订。
                    </div>
                  )}
                </>
              }
            >
              {(source) => (
                <TableRow>
                  <OperationCells
                    recordedAt={e.at}
                    operator={e.actor || '未登记'}
                    action={actionName(e.operation)}
                    span={1}
                  />
                  <DifferenceCells
                    fieldLabel="变更内容"
                    difference={
                      e.before || e.after || e.assignedOwner || e.previousOwner
                        ? {
                            field: e.summary || '管理归属',
                            before:
                              e.before ||
                              (e.previousOwner &&
                                targetName(v, e.previousOwner)) ||
                              '—',
                            after:
                              e.after ||
                              (e.assignedOwner &&
                                targetName(v, e.assignedOwner)) ||
                              '—',
                          }
                        : undefined
                    }
                  />
                  <TableCell data-label="查看">
                    {source}
                    {follow && (
                      <Button
                        size="small"
                        id={'event-' + e.id}
                        onClick={() =>
                          follow(
                            e.target,
                            e.operation.startsWith('mandate.'),
                            e.id,
                          )
                        }
                      >
                        查看
                        {e.operation.startsWith('mandate.')
                          ? '这份职责'
                          : '这份授予'}
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              )}
            </OperationRows>
          ))}
        </Table>
      </TableContainer>
      {!v.events?.length && (
        <Typography role="status" sx={{ py: 2 }}>
          {column ? '没有符合条件的变更日志' : '暂无可见变更日志'}
        </Typography>
      )}
      <WorkspacePager v={v} params={params} change={change} unit="操作" />
      {request && (
        <ColumnFilterDialog
          request={request}
          columns={columns}
          applied={model}
          onClose={() => {
            const source = request.source;
            setRequest(undefined);
            requestAnimationFrame(() => source.isConnected && source.focus());
          }}
          onApply={(f) =>
            change(
              {
                logField: f.filterField,
                logValue: f.filterValue,
                gp: '',
                originEvent: '',
              },
              request.source.id || 'security-log-filter',
            )
          }
        />
      )}
    </div>
  );
}
