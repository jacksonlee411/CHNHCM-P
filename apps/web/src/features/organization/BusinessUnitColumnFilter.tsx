import { createContext, useContext, useLayoutEffect, useRef, useState } from 'react';
import { Box, Button, Dialog, DialogActions, DialogContent, DialogTitle, IconButton, MenuItem, TextField, useMediaQuery } from '@mui/material';
import { DateField } from '@chnhcm/ui';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import {
  GridColumnMenu, GridFilterForm, getGridDateOperators, getGridSingleSelectOperators, getGridStringOperators,
  useGridApiContext, type GridColumnMenuItemProps, type GridColumnMenuProps, type GridFilterInputValueProps,
  type GridFilterItem, type GridFilterModel, type GridSlotsComponent,
} from '@mui/x-data-grid';

export const legalEntitySubjectTypes = [{ value: 'legal_person', label: '法人组织' }, { value: 'branch', label: '分支机构' }, { value: 'unincorporated', label: '非法人组织' }, { value: 'other', label: '其他组织' }];
export const columnStatuses = [{ value: 'active', label: '启用' }, { value: 'inactive', label: '停用' }, { value: 'not_effective', label: '未生效' }];
const fieldNames: Record<string, string> = { subjectType: '主体类别', registrationType: '登记类型', registrationNumber: '登记号码', establishingOrganization: '设立单位说明', nameEn: '英文名称', addressLine: '详细地址', postalCode: '邮政编码', shortName: '简称', managementBuId: '管理业务单元', departmentPath: '部门路径', code: '编码', name: '名称', status: '状态', effectiveDate: '生效日期' };
const operatorNames: Record<string, string> = { contains: '包含', equals: '等于', is: '等于', isAnyOf: '属于任一', before: '早于', after: '晚于', onOrBefore: '不晚于', onOrAfter: '不早于' };
export const filterKeys = ['filterField', 'filterOperator', 'filterValue'] as const;
export function columnFilterQuery(params: URLSearchParams) {
  // Preserve invalid direct-link input for Go to reject, including present-but-empty parameters.
  return {
    filterField: params.has('filterField') ? params.get('filterField') as 'code' | 'name' | 'shortName' | 'managementBuId' | 'departmentPath' | 'status' | 'effectiveDate' : undefined,
    filterOperator: params.has('filterOperator') ? params.get('filterOperator') as 'contains' | 'equals' | 'is' | 'isAnyOf' | 'before' | 'after' | 'onOrBefore' | 'onOrAfter' : undefined,
    filterValue: params.has('filterValue') ? params.get('filterValue')! : undefined,
  };
}
export function columnFilterModel(params: URLSearchParams): GridFilterModel {
  const { filterField: field, filterOperator: operator, filterValue: value } = columnFilterQuery(params);
  if (!field || !operator || value == null || !operatorsFor(field).some(op => op.value === operator)) return { items: [] };
  return { items: [{ field, operator, value: operator === 'isAnyOf' ? value.split(',') : value }] };
}
export function columnFilterSummary(item?: GridFilterItem, columns?: import('../../shared/dated/ColumnFilter').FilterColumn[]) {
  if (!item) return '';
  const values: string[] = Array.isArray(item.value) ? item.value : [item.value];
  const column = columns?.find(c => c.field === item.field);
  if (item.field === 'countryRegionId' || item.field === 'level') {
    const label = item.field === 'level' ? '层级' : '国家/地区';
    return `${label} ${operatorNames[item.operator]} ${column?.values?.find(option => option.value === item.value)?.label ?? '所选值不可用'}`;
  }
  return `${fieldNames[item.field]} ${operatorNames[item.operator]} ${values.map(value => item.field === 'managementBuId' ? '已选择的业务单元' : item.field === 'status' ? columnStatuses.find(s => s.value === value)?.label ?? value : item.field === 'subjectType' ? legalEntitySubjectTypes.find(s => s.value === value)?.label ?? value : value).join('、')}`;
}
// Synchronous text and pure-date input avoids the native debounce/Date-object conversion
// at an explicit Apply boundary. Field/operator selectors and enum inputs remain native.
function LiteralInput({ item, applyValue, focusElementRef }: GridFilterInputValueProps) {
  const Field = item.field === 'effectiveDate' ? DateField : TextField;
  return <Field size="small" fullWidth label="值"
    value={item.value ?? ''} onChange={e => applyValue({ ...item, value: e.target.value })} inputRef={focusElementRef}
    slotProps={{ inputLabel: { shrink: true } }} />;
}
export function operatorsFor(field: string) {
  if (field === 'countryRegionId' || field === 'level') {
    return getGridStringOperators().filter(op => op.value === 'equals').map(op => ({ ...op, label: operatorNames.equals }));
  }
  if (['status', 'subjectType', 'action', 'changedField'].includes(field)) return getGridSingleSelectOperators().filter(op => ['is', 'isAnyOf'].includes(op.value)).map(op => ({ ...op, label: operatorNames[op.value] }));
  const operators = ['effectiveDate', 'recordedAt'].includes(field) ? getGridDateOperators().filter(op => ['is', 'before', 'after', 'onOrBefore', 'onOrAfter'].includes(op.value))
    : ['code', 'name', 'nameEn', 'registrationType', 'registrationNumber', 'establishingOrganization', 'addressLine', 'postalCode', 'actor', 'shortName', 'departmentPath', 'managementBuId'].includes(field) ? getGridStringOperators().filter(op => (field === 'managementBuId' ? ['equals'] : ['contains', 'equals']).includes(op.value)) : [];
  return operators.map(op => ({ ...op, label: operatorNames[op.value], InputComponent: LiteralInput }));
}
import {ColumnFilterDialog as SharedColumnFilterDialog} from '../../shared/dated/ColumnFilter';
export const ColumnFilterDialog = (props: React.ComponentProps<typeof SharedColumnFilterDialog>) => <SharedColumnFilterDialog {...props} columns={props.columns?.map(c=>({...c,operators:c.operators ?? operatorsFor(c.field)}))}/>;
export { ColumnFilterEditor, columnFilterSlots, type FilterColumn, type FilterRequest } from '../../shared/dated/ColumnFilter';
