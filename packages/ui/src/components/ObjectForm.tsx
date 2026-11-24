import { createContext, useContext, useId, useMemo, useState, type ReactNode } from 'react';
import { Alert, Box, Button, TextField, ThemeProvider, Typography, createTheme, useTheme, type TextFieldProps } from '@mui/material';
import { ObjectDetailField } from './ObjectDetail';
import { DateField } from './DateField';
import './object-form.css';

const ObjectFormContext = createContext(false);
export const useObjectForm = () => useContext(ObjectFormContext);

/** Appearance and field relationships only; drafts and validation stay with the object. */
export function ObjectForm({ children, active = true }: { children: ReactNode; active?: boolean }) {
  const parent = useTheme();
  const theme = useMemo(() => createTheme(parent, { components: { MuiTextField: { defaultProps: { variant: 'standard', size: 'small' } }, MuiCheckbox: { defaultProps: { size: 'small' }, styleOverrides: { root: { '&.Mui-checked:not(.Mui-disabled)': { color: '#087f7b' } } } }, MuiMenuItem: { styleOverrides: { root: { whiteSpace: 'normal', overflowWrap: 'anywhere' } } } } }), [parent]);
  if (!active) return <>{children}</>;
  return <ObjectFormContext.Provider value><ThemeProvider theme={theme}><Box className="object-form">{children}</Box></ThemeProvider></ObjectFormContext.Provider>;
}

/** Shared editors retain their original appearance outside an object form. */
export function ObjectTextField(props: TextFieldProps) {
  const compact = useObjectForm(), generatedId = useId();
  const Field = props.type === 'date' ? DateField : TextField;
  if (!compact || !props.label) return <Field {...props} />;
  const { label, id = generatedId, helperText, ...rest } = props;
  return <ObjectDetailField label={label} inputId={id} required={props.required}>
    <Field {...rest} id={id} label={undefined} variant="standard" fullWidth helperText={helperText}
      slotProps={{ ...props.slotProps, ...(props.select ? { select: { ...(typeof props.slotProps?.select === 'object' ? props.slotProps.select : {}), SelectDisplayProps: { ...(typeof props.slotProps?.select === 'object' ? props.slotProps.select.SelectDisplayProps : {}), 'aria-labelledby': `${id}-label` } } } : {}) }} />
  </ObjectDetailField>;
}

export function ObjectFormSection({ title, summary, children, defaultOpen = false, id }: { title: string; summary?: string; children: ReactNode; defaultOpen?: boolean; id?: string }) {
  const [open, setOpen] = useState(defaultOpen);
  return <Box component="details" id={id} className="object-form-disclosure" open={open} onToggle={event => setOpen(event.currentTarget.open)}>
    <Box component="summary"><Typography component="span">{title}</Typography>{summary && <Typography component="span" className="object-form-section-summary">{summary}</Typography>}</Box>
    <Box className="object-form-section-content">{children}</Box>
  </Box>;
}

export function focusObjectField(id: string, fallbackId: string) {
  requestAnimationFrame(() => {
    const target = document.getElementById(id) ?? document.getElementById(fallbackId);
    for (let parent = target?.parentElement; parent; parent = parent.parentElement) if (parent instanceof HTMLDetailsElement) parent.open = true;
    requestAnimationFrame(() => { target?.focus({ preventScroll: true }); target?.scrollIntoView({ block: 'nearest' }); });
  });
}

export function ObjectFormError({ id, message, issues = [], children, onLocate }: { id: string; message: string; issues?: { id: string; message: string }[]; children?: ReactNode; onLocate?: (id: string) => void }) {
  if (!message) return null;
  return <Alert id={id} tabIndex={-1} severity="error" role="alert" className="object-form-error">
    {!(issues.length === 1 && issues[0]?.message === message) && <Typography variant="body2">{message}</Typography>}
    {issues.length > 0 && <Box component="ul">{issues.map(issue => <li key={issue.id}><Button onClick={() => onLocate ? onLocate(issue.id) : focusObjectField(issue.id, id)}>{issue.message}</Button></li>)}</Box>}
    {children}
  </Alert>;
}
