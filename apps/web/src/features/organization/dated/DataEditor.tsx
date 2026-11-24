import { ObjectTextField } from '@chnhcm/ui';
import { useDefinition, type DataEditorProps } from './presentation';
export function DataEditor(props: DataEditorProps) {
  const definition = useDefinition();
  return <>{definition.fields.filter(f => props.fields.includes(f.key)).map(f => <ObjectTextField key={f.key}
    id={`${props.prefix}-${f.key}`} name={f.key} label={f.label} value={props.values[f.key] ?? ''} required={f.required}
    disabled={props.disabled || !props.writes.includes(f.key)} onChange={e => props.onChange(f.key, e.target.value)}
    error={props.fieldError.replace('changes.', '') === f.key} helperText={props.fieldError.replace('changes.', '') === f.key ? props.error : undefined} />)}
    {props.guidance && <p className="object-form-hint">{props.guidance}</p>}
    {definition.Relations && <definition.Relations {...props} />}
  </>;
}
export const recordLabels = { code: { label: '编码' }, effectiveDate: { label: '生效日期' }, operationType: { label: '操作类型' }, operationReason: { label: '操作原因' } };
