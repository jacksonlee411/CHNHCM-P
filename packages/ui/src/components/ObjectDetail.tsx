import { useEffect, useId, useState, type ReactNode } from 'react';
import { Box, IconButton, Paper, Popover, Typography, useMediaQuery } from '@mui/material';
import LinkIcon from '@mui/icons-material/Link';
import './object-detail.css';

/** Object-neutral detail sections; modules provide their own fields, dates and actions. */
export function ObjectDetailSheet({ identity, tabs, timeline, children, className = '' }: { identity: ReactNode; tabs?: ReactNode; timeline?: ReactNode; children: ReactNode; className?: string }) {
  return <Paper variant="outlined" className={`object-detail-sheet ${className}`}>
    <Box className="object-detail-identity">{identity}</Box>
    {tabs}
    {timeline && <Box className="object-detail-timeline">{timeline}</Box>}
    <Box className="object-detail-body">{children}</Box>
  </Paper>;
}

export function ObjectDetailIdentity({ icon, title, summary, related, compactRelatedLabel }: { icon: ReactNode; title: string; summary?: ReactNode; related?: ReactNode; compactRelatedLabel?: string }) {
  const narrow = useMediaQuery('(max-width:899px)'), relatedId = useId();
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);
  useEffect(() => { if (!narrow) setAnchor(null); }, [narrow]);
  return <Box className="object-identity-layout">
    <Box className="object-identity-icon" aria-hidden="true">{icon}</Box>
    <Box className="object-identity-name"><Typography component="h1">{title}</Typography><Box className="object-identity-summary">{summary}</Box></Box>
    {related && (narrow && compactRelatedLabel ? <>
      <IconButton className="object-identity-related-trigger" size="small" title={compactRelatedLabel} aria-label={compactRelatedLabel} aria-haspopup="dialog" aria-expanded={!!anchor} aria-controls={anchor ? relatedId : undefined} onClick={event => setAnchor(event.currentTarget)}><LinkIcon /></IconButton>
      <Popover open={!!anchor} anchorEl={anchor} onClose={() => setAnchor(null)} anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }} transformOrigin={{ vertical: 'top', horizontal: 'right' }} slotProps={{ paper: { id: relatedId, role: 'dialog', 'aria-label': compactRelatedLabel } }}>
        <Box className="object-identity-related-popup" onClick={event => { if ((event.target as HTMLElement).closest('button:not(:disabled)')) setAnchor(null); }}>{related}</Box>
      </Popover>
    </> : <Box className="object-identity-related">{related}</Box>)}
  </Box>;
}

export function ObjectDetailSection({ title, children }: { title?: string; children: ReactNode }) {
  return <Box component="section" className="object-detail-section">{title && <Typography component="h2" className="object-section-title">{title}</Typography>}{children}</Box>;
}

export function ObjectDetailField({ label, inputId, children, required, error, help }: { label: ReactNode; inputId?: string; children: ReactNode; required?: boolean; error?: string; help?: string }) {
  return <Box className="object-detail-field"><Typography component="label" id={inputId ? `${inputId}-label` : undefined} htmlFor={inputId} className="object-field-label">{label}{required && <span className="object-field-required" aria-hidden="true">*</span>}</Typography><Box className="object-field-value">{children}{error && <Typography id={inputId ? `${inputId}-error` : undefined} className="object-field-error">{error}</Typography>}{help && <Typography className="object-field-help">{help}</Typography>}</Box></Box>;
}
