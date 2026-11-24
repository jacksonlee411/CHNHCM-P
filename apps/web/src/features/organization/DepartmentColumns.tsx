import { Box } from '@mui/material';
import type { Definition } from './dated/presentation';
import { operatorsFor, columnStatuses } from './BusinessUnitColumnFilter';
import { DepartmentBUFilter, DepartmentPath, referenceLabel } from './DepartmentRelations';
export const departmentColumns: Pick<Definition, 'hiddenColumns' | 'additionalColumns' | 'filterColumns' | 'summary'> = {
  hiddenColumns: ['shortName'],
  additionalColumns: [
    { field: 'managementBuId', headerName: '管理业务单元', width: 200, sortable: false, filterOperators: operatorsFor('managementBuId'), renderCell: p => referenceLabel(p.row.references?.managementBuId) || '无读取权限' },
    { field: 'departmentPath', headerName: '部门路径', flex: 1.5, minWidth: 230, sortable: false, filterOperators: operatorsFor('departmentPath'), renderCell: p => p.row.secondary },
    { field: 'shortName', headerName: '简称', width: 120, sortable: false, valueGetter: (_, row) => row.values?.shortName, filterOperators: operatorsFor('shortName') },
  ],
  filterColumns: [
    { field: 'code', label: '编码' }, { field: 'name', label: '名称' }, { field: 'shortName', label: '简称' },
    { field: 'managementBuId', label: '管理业务单元', Input: DepartmentBUFilter }, { field: 'departmentPath', label: '部门路径' },
    { field: 'status', label: '状态', values: columnStatuses }, { field: 'effectiveDate', label: '生效日期' },
  ],
  summary: row => <Box sx={{ whiteSpace: 'normal', overflowWrap: 'anywhere', px: 1, pb: 1, fontSize: 13 }}><div>{referenceLabel(row.references?.managementBuId) || '管理业务单元无读取权限'}</div>{row.secondary}</Box>,
};
