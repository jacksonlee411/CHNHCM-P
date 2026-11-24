import { useLayoutEffect, useRef, useState, type InputHTMLAttributes } from 'react';
import { IconButton, InputAdornment, TextField, type TextFieldProps } from '@mui/material';
import CalendarMonthIcon from '@mui/icons-material/CalendarMonth';
import { useForkRef } from '@mui/material/utils';

/** ISO text is visible; the browser owns calendar picking and calendar validity.
 * Bounds are supplied by the server. No business date arithmetic lives here.
 */
export function DateField({ type: _type, inputRef, inputProps, slotProps, onChange, ...props }: TextFieldProps) {
  const field = useRef<HTMLInputElement>(null), picker = useRef<HTMLInputElement>(null);
  const combinedRef = useForkRef(field, inputRef);
  const [validation, setValidation] = useState('');
  const html = { ...inputProps, ...(typeof slotProps?.htmlInput === 'object' ? slotProps.htmlInput : {}) } as InputHTMLAttributes<HTMLInputElement>;
  const input = typeof slotProps?.input === 'object' ? slotProps.input : {};
  const value = String(props.value ?? '');
  const min = String(html.min ?? '0001-01-01'), max = String(html.max ?? '9999-12-31');
  const readOnly = Boolean(html.readOnly || input.readOnly || props.InputProps?.readOnly);
  const validate = (next: string) => {
    const calendar = picker.current;
    if (!calendar) return '';
    calendar.value = next;
    if (next && (!/^\d{4}-\d{2}-\d{2}$/.test(next) || calendar.value !== next)) return '请输入 YYYY-MM-DD 格式的有效日期';
    if (next && calendar.validity.rangeUnderflow) return `日期不得早于 ${min}`;
    if (next && calendar.validity.rangeOverflow) return `日期不得晚于 ${max}`;
    return '';
  };
  useLayoutEffect(() => {
    const message = validate(value);
    field.current?.setCustomValidity(message);
    setValidation(message);
  }, [value, min, max]);
  return <TextField {...props} type="text" inputRef={combinedRef} placeholder="YYYY-MM-DD"
    error={props.error || Boolean(validation)} helperText={props.error ? props.helperText : validation || props.helperText}
    onChange={event => {
      event.target.setCustomValidity(validate(event.target.value));
      onChange?.(event);
    }}
    slotProps={{ ...slotProps,
      htmlInput: { ...html, readOnly, pattern: '[0-9]{4}-[0-9]{2}-[0-9]{2}', maxLength: 10 },
      input: { ...props.InputProps, ...input, endAdornment: <>{input.endAdornment ?? props.InputProps?.endAdornment}<InputAdornment position="end" sx={{ position: 'relative' }}>
        <IconButton size="small" aria-label="选择日期" disabled={props.disabled || readOnly}
          onClick={() => picker.current?.showPicker()}><CalendarMonthIcon fontSize="small" /></IconButton>
        <input ref={picker} type="date" aria-hidden="true" tabIndex={-1} disabled={props.disabled || readOnly}
          min={min} max={max} value={value} onChange={event => { onChange?.(event); field.current?.focus(); }}
          style={{ position: 'absolute', width: 1, height: 1, opacity: 0, pointerEvents: 'none', bottom: 0, right: 0 }} />
      </InputAdornment></> },
    }} />;
}
