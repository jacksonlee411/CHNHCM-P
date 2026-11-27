import { type GridColDef } from '@mui/x-data-grid';
import { ConfigurationList, statusFilter } from './ConfigurationList';
import { type Principal, statusLabels } from './rates';
import { listPlans, planPath, planTitle, scopeLabel, type Plan } from './plans';

export function PlanList({ session }: { session: Principal }) {
  return (
    <ConfigurationList<Plan>
      session={session}
      path={planPath}
      title={planTitle}
      read={listPlans}
      searchLabel="搜索计划编码或名称"
      searchFields={['name', 'code']}
      filters={[statusFilter]}
      label={(row) => row.record.name || row.code || '查看计划'}
      columns={(can, link) => {
        const columns: GridColDef<Plan>[] = [
          {
            field: can('name') ? 'name' : 'detail',
            headerName: can('name') ? '计划名称' : '计划',
            flex: 1.5,
            minWidth: 160,
            renderCell: ({ row }) => link(row),
          },
        ];
        if (can('code')) columns.push({ field: 'code', headerName: '计划编码', flex: 1, minWidth: 120 });
        if (can('scope'))
          columns.push({
            field: 'scope',
            headerName: '适用对象',
            flex: 2,
            minWidth: 165,
            valueGetter: (_, row) => scopeLabel(row.record.scope) || '—',
          });
        if (can('effectiveDate'))
          columns.push({
            field: 'effectiveDate',
            headerName: '生效日期',
            flex: 1,
            minWidth: 115,
            valueGetter: (_, row) => row.record.effectiveDate,
          });
        if (can('status'))
          columns.push({
            field: 'status',
            headerName: '状态',
            flex: 1,
            minWidth: 90,
            valueGetter: (_, row) => statusLabels[row.record.status ?? ''],
          });
        return columns;
      }}
      summary={(row) =>
        [
          row.code,
          scopeLabel(row.record.scope),
          row.record.effectiveDate,
          statusLabels[row.record.status ?? ''],
        ]
          .filter(Boolean)
          .join(' · ')
      }
    />
  );
}
