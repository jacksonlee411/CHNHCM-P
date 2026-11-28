import { useEffect, useState } from 'react';
import {
  Autocomplete,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  MenuItem,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
} from '@mui/material';
import { ObjectDetailField, ObjectForm, ObjectTextField } from '@chnhcm/ui';
import { useRead } from '../../shared/useRead';
import { methods, natures, rateCandidates, type Component } from './plans';
import { frequencies, money, moneyUnit } from './rates';
import { same } from './planDraft';

type RateRef = NonNullable<Component['rates']>[number] & { effectiveDate?: string; endExclusive?: string };
type Method = Component['method'];
export const methodField: Record<Method, string> = { FIXED: 'amount', GRADE_RATE: 'rates', PERCENT: 'percentage' };
function emptyRow(method: Method, units: Record<string, string>): Component {
  return {
    id: crypto.randomUUID(),
    name: '',
    code: '',
    nature: 'BASIC',
    currency: 'CNY',
    frequency: 'MONTH',
    unit: units.MONTH,
    precision: 2,
    method,
    ...(method === 'PERCENT' ? { rounding: 'HALF_UP' } : {}),
  };
}
export function valueSummary(row: Component, rows: Component[]) {
  if (row.method === 'FIXED')
    return row.amount === undefined
      ? ''
      : [money(row.amount), moneyUnit(row.currency, row.frequency)].filter(Boolean).join(' ');
  if (row.method === 'GRADE_RATE')
    return (row.rates ?? []).map((r) => [r.code, r.name].filter(Boolean).join(' ')).join('、');
  if (!row.percent) return '';
  const base = rows.find((other) => other.id === row.baseId);
  return (base?.name || '基准待补') + ' × ' + row.percent + '%';
}
function RatePicker({ row, day, update }: { row: Component; day: string; update: (rates: RateRef[]) => void }) {
  const [query, setQuery] = useState('');
  const list = useRead(
    day && row.currency && row.frequency ? JSON.stringify([day, query, row.currency, row.frequency]) : undefined,
    () => rateCandidates(day, query, row.currency, row.frequency),
    'fresh',
  );
  return (
    <ObjectDetailField label="职级及引用标准" inputId="plan-row-rates" required>
      <Autocomplete<RateRef, true>
        id="plan-row-rates"
        multiple
        filterSelectedOptions
        filterOptions={(v) => v}
        options={list.data?.items ?? []}
        value={row.rates ?? []}
        isOptionEqualToValue={(a, b) => a.rateId === b.rateId}
        renderOption={({ key, ...props }, r) => (
          <li key={key} {...props}>
            {[
              r.code,
              r.name,
              moneyUnit(row.currency, row.frequency),
              r.effectiveDate && '自 ' + r.effectiveDate + ' 起',
              r.endExclusive && '至 ' + r.endExclusive + ' 前',
            ]
              .filter(Boolean)
              .join(' ')}
          </li>
        )}
        getOptionKey={(r) => r.rateId}
        getOptionLabel={(r) => [r.code, r.name, moneyUnit(row.currency, row.frequency)].filter(Boolean).join(' ')}
        loading={!!day && !list.data && !list.error}
        noOptionsText="该日期和口径下没有符合条件的标准"
        loadingText="正在查询"
        onChange={(_, v) => update(v)}
        onInputChange={(_, v, why) => {
          if (why === 'input' || why === 'clear') setQuery(v);
        }}
        renderInput={(p) => (
          <TextField
            {...p}
            variant="standard"
            error={!!list.error}
            helperText={list.error || (day ? '适用日期 ' + day : '请先填写计划生效日期')}
          />
        )}
      />
    </ObjectDetailField>
  );
}
function ComponentDialog({
  initial,
  rows,
  fields,
  day,
  units,
  cancel,
  accept,
  onScratchChange,
}: {
  initial: Component;
  rows: Component[];
  fields: string[];
  day: string;
  units: Record<string, string>;
  cancel: () => void;
  accept: (r: Component) => void;
  onScratchChange: (changed: boolean) => void;
}) {
  const [row, setRow] = useState(initial),
    [nextMethod, setNextMethod] = useState<Method>();
  useEffect(() => {
    onScratchChange(!same(row, initial));
    return () => onScratchChange(false);
  }, [row, initial, onScratchChange]);
  const input = (key: 'code' | 'name' | 'currency' | 'amount' | 'percent', label: string) => (
    <ObjectTextField
      id={'plan-row-' + key}
      label={label}
      required
      value={row[key] ?? ''}
      onChange={(e) => setRow({ ...row, [key]: e.target.value })}
      slotProps={{ htmlInput: { inputMode: key === 'percent' || key === 'amount' ? 'decimal' : undefined } }}
    />
  );
  const switchTo = (method: Method) => {
    const { amount, rates, baseId, percent, rounding, targets, ...meta } = row;
    setRow({ ...meta, method, ...(method === 'PERCENT' ? { rounding: 'HALF_UP' } : {}) });
    setNextMethod(undefined);
  };
  const chooseMethod = (v: Method) => {
    if (row.method === v) return;
    if ((row.amount !== undefined && row.amount !== '') || row.rates?.length || row.baseId || row.percent)
      setNextMethod(v);
    else switchTo(v);
  };
  const choices = (labels: Record<string, string>) =>
    Object.entries(labels).map(([k, v]) => (
      <MenuItem key={k} value={k}>
        {v}
      </MenuItem>
    ));
  const bases = rows.filter((r) => r.id !== row.id),
    missingBase = !!row.baseId && !bases.some((b) => b.id === row.baseId);
  const methodsAllowed = Object.entries(methods).filter(
    ([k]) => fields.includes(methodField[k as Method]) || row.method === k,
  );
  return (
    <>
      <Dialog open onClose={cancel} fullWidth maxWidth="sm" aria-labelledby="plan-row-title">
        <DialogTitle id="plan-row-title">维护组成项</DialogTitle>
        <DialogContent dividers>
          <ObjectForm>
            <div className="object-detail-columns">
              {input('code', '项编码')}
              {input('name', '项名称')}
              <ObjectTextField
                id="plan-row-nature"
                select
                required
                label="性质"
                value={row.nature}
                onChange={(e) => setRow({ ...row, nature: e.target.value })}
              >
                {choices(natures)}
              </ObjectTextField>
              <ObjectTextField
                id="plan-row-method"
                select
                required
                label="取值方式"
                value={row.method}
                disabled={!fields.includes(methodField[row.method])}
                onChange={(e) => chooseMethod(e.target.value as Method)}
              >
                {methodsAllowed.map(([k, v]) => (
                  <MenuItem key={k} value={k}>
                    {v}
                  </MenuItem>
                ))}
              </ObjectTextField>
              {input('currency', '币种')}
              <ObjectTextField
                id="plan-row-frequency"
                select
                required
                label="约定频率"
                value={row.frequency}
                onChange={(e) => setRow({ ...row, frequency: e.target.value, unit: units[e.target.value] })}
              >
                {Object.keys(units).map((k) => (
                  <MenuItem value={k} key={k}>
                    {frequencies[k]?.name ?? k}
                  </MenuItem>
                ))}
              </ObjectTextField>
              <ObjectDetailField label="计量单位">{moneyUnit(row.currency, row.frequency)}</ObjectDetailField>
              <ObjectTextField
                id="plan-row-precision"
                label="金额小数位数"
                type="number"
                value={row.precision}
                onChange={(e) => setRow({ ...row, precision: Number(e.target.value) })}
              />
              {row.method === 'FIXED' && fields.includes('amount') && input('amount', '固定金额')}
              {row.method === 'GRADE_RATE' && fields.includes('rates') && (
                <RatePicker row={row} day={day} update={(rates) => setRow({ ...row, rates })} />
              )}
              {row.method === 'PERCENT' && fields.includes('percentage') && (
                <>
                  <ObjectTextField
                    select
                    required
                    id="plan-row-baseId"
                    label="基准项"
                    value={missingBase ? '' : (row.baseId ?? '')}
                    error={missingBase}
                    helperText={missingBase ? '原基准项已移除，请重新选择。' : undefined}
                    onChange={(e) => setRow({ ...row, baseId: e.target.value })}
                  >
                    {bases.map((r) => (
                      <MenuItem key={r.id} value={r.id}>
                        {[r.code, r.name].filter(Boolean).join(' ')}
                      </MenuItem>
                    ))}
                  </ObjectTextField>
                  {input('percent', '目标比例（%）')}
                  <ObjectDetailField label="舍入方式">最终金额四舍五入</ObjectDetailField>
                </>
              )}
            </div>
          </ObjectForm>
        </DialogContent>
        <DialogActions>
          <Button onClick={cancel}>取消</Button>
          <Button variant="contained" onClick={() => accept(row)}>
            确定
          </Button>
        </DialogActions>
      </Dialog>
      <Dialog open={!!nextMethod} onClose={() => setNextMethod(undefined)}>
        <DialogTitle>切换取值方式</DialogTitle>
        <DialogContent>切换会清除本行原取值方式已填的约定。</DialogContent>
        <DialogActions>
          <Button onClick={() => setNextMethod(undefined)}>保留当前方式</Button>
          <Button onClick={() => nextMethod && switchTo(nextMethod)}>切换</Button>
        </DialogActions>
      </Dialog>
    </>
  );
}

export function ComponentEditor({
  rows,
  fields,
  date,
  units,
  onChange,
  onScratchChange,
  issue,
}: {
  rows: Component[];
  fields: string[];
  date: string;
  units: Record<string, string>;
  onChange: (rows: Component[]) => void;
  onScratchChange: (changed: boolean) => void;
  issue?: { field?: string; text: string };
}) {
  const [editing, setEditing] = useState<Component>();
  const method = (Object.keys(methodField) as Method[]).find((m) => fields.includes(methodField[m]));
  const move = (i: number, step: number) => {
    const next = rows.slice();
    [next[i], next[i + step]] = [next[i + step], next[i]];
    onChange(next);
  };
  return (
    <>
      <TableContainer>
        <Table size="small" aria-label="计划组成草稿">
          <TableHead>
            <TableRow>
              {['项名称', '性质', '取值方式', '约定', '操作'].map((v) => (
                <TableCell key={v}>{v}</TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {rows.map((r, i) => (
              <TableRow key={r.id}>
                <TableCell>{r.name || r.code}</TableCell>
                <TableCell>{natures[r.nature]}</TableCell>
                <TableCell>{methods[r.method]}</TableCell>
                <TableCell>{valueSummary(r, rows)}</TableCell>
                <TableCell>
                  <Stack direction="row" flexWrap="wrap">
                    <Button id={'plan-components-' + i} aria-label={'编辑 ' + r.name} onClick={() => setEditing(r)}>
                      编辑
                    </Button>
                    <Button disabled={!i} aria-label={'上移 ' + r.name} onClick={() => move(i, -1)}>
                      上移
                    </Button>
                    <Button disabled={i === rows.length - 1} aria-label={'下移 ' + r.name} onClick={() => move(i, 1)}>
                      下移
                    </Button>
                    <Button
                      aria-label={'移除 ' + r.name}
                      onClick={() => onChange(rows.filter((other) => other.id !== r.id))}
                    >
                      移除
                    </Button>
                  </Stack>
                  {issue?.field?.startsWith('components.' + i + '.') && <span role="alert">{issue.text}</span>}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
      {method && (
        <Button id="plan-components" onClick={() => setEditing(emptyRow(method, units))}>
          添加组成项
        </Button>
      )}
      {editing && (
        <ComponentDialog
          onScratchChange={onScratchChange}
          initial={editing}
          rows={rows}
          fields={fields}
          day={date}
          units={units}
          cancel={() => setEditing(undefined)}
          accept={(row) => {
            onChange(rows.some((r) => r.id === row.id) ? rows.map((r) => (r.id === row.id ? row : r)) : [...rows, row]);
            setEditing(undefined);
          }}
        />
      )}
    </>
  );
}
