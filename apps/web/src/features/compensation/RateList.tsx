import { type GridColDef } from '@mui/x-data-grid';
import { ConfigurationList, filterColumn, statusFilter } from './ConfigurationList';
import {
  amountKeys,
  amountLabels,
  formLabels,
  frequencies,
  gradeLabel,
  money,
  rateList,
  ratePath,
  rateTitle,
  statusLabels,
  type Principal,
  type Rate,
} from './rates';

const filters = [
  filterColumn('currency', '币种'),
  filterColumn(
    'frequency',
    '约定频率',
    Object.fromEntries(Object.entries(frequencies).map(([key, v]) => [key, v.name])),
  ),
  filterColumn('form', '标准形式', formLabels),
  statusFilter,
];
export function RateList({ session }: { session: Principal }) {
  return (
    <ConfigurationList<Rate>
      session={session}
      path={ratePath}
      title={rateTitle}
      read={rateList}
      searchLabel="搜索职级编码或名称"
      searchFields={['gradeId']}
      filters={filters}
      label={(row) => gradeLabel(row.record) || '查看标准'}
      columns={(can, link) => {
        const columns: GridColDef<Rate>[] = [
          {
            field: can('gradeId') ? 'gradeId' : 'detail',
            headerName: can('gradeId') ? '职级' : '标准',
            flex: 1.5,
            minWidth: 132,
            renderCell: ({ row }) => link(row),
          },
        ];
        function add(field: string, headerName: string, value: (row: Rate) => string | undefined) {
          if (can(field))
            columns.push({
              field,
              headerName,
              flex: 1,
              minWidth: 90,
              valueGetter: (_, row) => value(row) || '—',
            });
        }
        add('currency', '币种', (r) => r.currency);
        add('frequency', '约定频率', (r) => frequencies[r.frequency ?? '']?.name);
        add('form', '标准形式', (r) => formLabels[r.record.form ?? '']);
        amountKeys.forEach((key) => add(key, amountLabels[key], (r) => money(r.record[key])));
        add('effectiveDate', '生效日期', (r) => r.record.effectiveDate);
        add('status', '状态', (r) => statusLabels[r.record.status ?? '']);
        return columns;
      }}
      summary={({ record, currency, frequency }) =>
        [
          currency,
          frequencies[frequency ?? '']?.unitName,
          formLabels[record.form ?? ''],
          ...amountKeys
            .filter((key) => record[key] !== undefined)
            .map((key) => amountLabels[key] + ' ' + money(record[key])),
          record.effectiveDate,
          statusLabels[record.status ?? ''],
        ]
          .filter(Boolean)
          .join(' · ')
      }
    />
  );
}
