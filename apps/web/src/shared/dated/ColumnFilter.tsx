import { createContext, useContext, useLayoutEffect, useRef, useState } from 'react';
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  MenuItem,
  TextField,
  useMediaQuery,
} from '@mui/material';
import { DateField } from '@chnhcm/ui';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import {
  GridColumnMenu,
  GridFilterForm,
  getGridDateOperators,
  getGridSingleSelectOperators,
  getGridStringOperators,
  useGridApiContext,
  type GridColumnMenuItemProps,
  type GridColumnMenuProps,
  type GridFilterInputValueProps,
  type GridFilterItem,
  type GridFilterModel,
  type GridSlotsComponent,
} from '@mui/x-data-grid';

export const operatorsFor = (field: string) => {
  const values = ['status', 'subjectType', 'action', 'changedField'].includes(field)
    ? ['is', 'isAnyOf']
    : ['effectiveDate', 'recordedAt'].includes(field)
      ? ['is', 'before', 'after', 'onOrBefore', 'onOrAfter']
      : ['contains', 'equals'];
  const labels: Record<string, string> = {
    contains: '包含',
    equals: '等于',
    is: '等于',
    isAnyOf: '属于任一',
    before: '早于',
    after: '晚于',
    onOrBefore: '不晚于',
    onOrAfter: '不早于',
  };
  return values.map((value) => ({ value, label: labels[value] }));
};
export type FilterRequest = { field?: string; initialValue?: string; source: HTMLElement };
export const ColumnFilterEditor = createContext<(request: FilterRequest) => void>(() => {});
function ColumnFilterMenuItem({ colDef, onClick }: GridColumnMenuItemProps) {
  const open = useContext(ColumnFilterEditor),
    api = useGridApiContext();
  if (colDef.filterable === false) return null;
  return (
    <MenuItem
      onClick={(event) => {
        const source = api.current.getColumnHeaderElement(colDef.field)!;
        onClick(event);
        open({ field: colDef.field, source });
      }}
    >
      <FilterAltIcon fontSize="small" sx={{ mr: 1 }} />
      筛选…
    </MenuItem>
  );
}
function ColumnMenu(props: GridColumnMenuProps) {
  return <GridColumnMenu {...props} slots={{ columnMenuFilterItem: ColumnFilterMenuItem }} />;
}
function FilterIcon({ field, counter }: { field: string; counter?: number }) {
  const open = useContext(ColumnFilterEditor);
  return counter ? (
    <IconButton
      size="small"
      aria-label="编辑列筛选"
      onClick={(event) => {
        event.stopPropagation();
        open({ field, source: event.currentTarget });
      }}
    >
      <FilterAltIcon fontSize="small" />
    </IconButton>
  ) : null;
}
export const columnFilterSlots: Partial<GridSlotsComponent> = {
  columnMenu: ColumnMenu,
  columnHeaderFilterIconButton: FilterIcon,
};

export type FilterColumn = {
  field: string;
  label: string;
  Input?: (props: { value: string; onChange: (value: string) => void }) => React.ReactNode;
  operators?: { value: string; label: string }[];
  values?: { value: string; label: string }[];
};
// The table and grid share one draft/apply/cancel boundary. Only the field catalog
// adapter differs: GridFilterForm needs a grid provider, a semantic table does not.
function CatalogFilterForm({
  columns,
  item,
  onChange,
}: {
  columns: FilterColumn[];
  item: GridFilterItem;
  onChange: (v: GridFilterItem) => void;
}) {
  const column = columns.find((c) => c.field === item.field)!;
  const Field = ['effectiveDate', 'recordedAt'].includes(item.field) ? DateField : TextField;
  return (
    <Box
      sx={{
        display: 'grid',
        gridTemplateColumns: { xs: 'minmax(0, 1fr)', md: 'minmax(0, 1fr) minmax(0, 0.8fr) minmax(0, 1.5fr)' },
        gap: 1.5,
        pt: 1,
      }}
    >
      <TextField
        select
        slotProps={{ select: { SelectDisplayProps: { 'aria-label': '列', 'aria-labelledby': undefined } } }}
        label="列"
        size="small"
        fullWidth
        value={item.field}
        onChange={(e) =>
          onChange({
            field: e.target.value,
            operator: (columns.find((c) => c.field === e.target.value)?.operators ?? operatorsFor(e.target.value))[0]
              .value,
            value: '',
          })
        }
      >
        {columns.map((c) => (
          <MenuItem key={c.field} value={c.field}>
            {c.label}
          </MenuItem>
        ))}
      </TextField>
      <TextField
        select
        slotProps={{ select: { SelectDisplayProps: { 'aria-label': '条件', 'aria-labelledby': undefined } } }}
        label="条件"
        size="small"
        fullWidth
        value={item.operator}
        onChange={(e) => onChange({ ...item, operator: e.target.value, value: e.target.value === 'isAnyOf' ? [] : '' })}
      >
        {(column.operators ?? operatorsFor(item.field)).map((op) => (
          <MenuItem key={op.value} value={op.value}>
            {op.label}
          </MenuItem>
        ))}
      </TextField>
      {['isEmpty', 'isNotEmpty'].includes(item.operator) ? null : column.Input ? (
        <column.Input value={item.value ?? ''} onChange={(value) => onChange({ ...item, value })} />
      ) : column.values ? (
        <TextField
          select
          label="值"
          size="small"
          fullWidth
          value={item.value || (item.operator === 'isAnyOf' ? [] : '')}
          slotProps={{
            select: {
              multiple: item.operator === 'isAnyOf',
              SelectDisplayProps: { 'aria-label': '值', 'aria-labelledby': undefined },
            },
          }}
          onChange={(e) => onChange({ ...item, value: e.target.value })}
        >
          {column.values.map((v) => (
            <MenuItem key={v.value} value={v.value}>
              {v.label}
            </MenuItem>
          ))}
        </TextField>
      ) : (
        <Field
          label="值"
          size="small"
          fullWidth
          value={item.value ?? ''}
          onChange={(e) => onChange({ ...item, value: e.target.value })}
          slotProps={{ inputLabel: { shrink: true } }}
        />
      )}
    </Box>
  );
}
export function ColumnFilterDialog({
  request,
  applied,
  onApply,
  onClose,
  columns,
}: {
  request: FilterRequest;
  applied: GridFilterModel;
  onApply: (values: Record<string, string>, editedField: string) => void;
  onClose: () => void;
  columns?: FilterColumn[];
}) {
  const cancelButton = useRef<HTMLButtonElement>(null);
  const initial = applied.items.find(item => item.field === request.field) ?? applied.items[0];
  const requested = request.field ?? initial?.field ?? columns?.[0]?.field ?? 'code';
  const field = columns && !columns.some((c) => c.field === requested) ? columns[0].field : requested;
  const [draft, setDraft] = useState<GridFilterItem | null>(
    initial?.field === field
      ? { ...initial }
      : {
          field,
          operator: (columns?.find((c) => c.field === field)?.operators ?? operatorsFor(field))[0].value,
          value: field === request.field ? request.initialValue ?? '' : '',
        },
  );
  const narrow = useMediaQuery('(max-width:899px)'),
    previousWidth = useRef(narrow);
  useLayoutEffect(() => {
    if (previousWidth.current !== narrow) onClose();
    previousWidth.current = narrow;
  }, [narrow, onClose]);
  const clearedField = useRef(field);
  const valid =
    !draft ||
    ['isEmpty', 'isNotEmpty'].includes(draft.operator) ||
    (Array.isArray(draft.value)
      ? draft.value.length > 0
      : typeof draft.value === 'string' && draft.value.trim().length > 0);
  const apply = () => {
    if (!valid) return;
    onApply(
      draft
        ? {
            filterField: draft.field,
            filterOperator: draft.operator,
            filterValue: Array.isArray(draft.value) ? draft.value.join(',') : draft.value,
          }
        : { filterField: '', filterOperator: '', filterValue: '' },
      draft?.field ?? clearedField.current,
    );
    onClose();
  };
  return (
    <Dialog open disableRestoreFocus fullWidth maxWidth="sm" onClose={onClose} aria-labelledby="column-filter-title">
      <Box
        component="form"
        onSubmit={(event) => {
          event.preventDefault();
          apply();
        }}
      >
        <DialogTitle id="column-filter-title">列筛选</DialogTitle>
        <DialogContent>
          {draft && columns ? (
            <CatalogFilterForm columns={columns} item={draft} onChange={next => {
              const previous = applied.items.find(item => item.field === next.field);
              setDraft(next.field !== draft.field && previous ? { ...previous } : next);
            }} />
          ) : draft ? (
            <Box
              sx={{
                '& .MuiDataGrid-filterForm': {
                  flexDirection: { xs: 'column', md: 'row' },
                  alignItems: 'stretch',
                  gap: 1.5,
                  p: 0,
                  pt: 1,
                },
              }}
            >
              <GridFilterForm
                item={draft}
                hasMultipleFilters={false}
                applyFilterChanges={setDraft}
                applyMultiFilterOperatorChanges={() => {}}
                deleteFilter={() => setDraft(null)}
                logicOperatorInputProps={{ style: { display: 'none' } }}
                columnInputProps={{
                  style: { flex: 1, minWidth: 0 },
                  sx: { width: '100%' },
                  material: { SelectDisplayProps: { 'aria-label': '列', 'aria-labelledby': undefined } },
                }}
                operatorInputProps={{
                  style: { flex: 1, minWidth: 0 },
                  sx: { width: '100%' },
                  material: { SelectDisplayProps: { 'aria-label': '条件', 'aria-labelledby': undefined } },
                }}
                valueInputProps={{ style: { flex: 1, minWidth: 0, width: 'auto' } }}
                deleteIconProps={{ style: { display: 'none' } }}
              />
            </Box>
          ) : (
            <Button
              onClick={() =>
                setDraft({
                  field,
                  operator: (columns?.find((c) => c.field === field)?.operators ?? operatorsFor(field))[0].value,
                  value: '',
                })
              }
            >
              添加条件
            </Button>
          )}
        </DialogContent>
        <DialogActions sx={{ flexWrap: 'wrap' }}>
          <Button
            onClick={() => {
              clearedField.current = draft?.field ?? field;
              setDraft(null);
              cancelButton.current?.focus();
            }}
            disabled={!draft}
          >
            清除条件
          </Button>
          <Box sx={{ flex: 1 }} />
          <Button ref={cancelButton} onClick={onClose}>
            取消
          </Button>
          <Button type="submit" variant="contained" disabled={!valid}>
            应用
          </Button>
        </DialogActions>
      </Box>
    </Dialog>
  );
}
