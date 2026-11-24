import { useEffect, useRef } from 'react';
import { Box, Button, IconButton, useMediaQuery } from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import type { ElementType } from 'react';

export function RecordBackButton({ to, component, label = '返回列表' }: { to: string; component: ElementType; label?: string }) {
  const narrow = useMediaQuery('(max-width:899px)');
  return <Button component={component} to={to} aria-label={label} title={label} size="small" startIcon={<ArrowBackIcon />} sx={{ minWidth: 0, '& .MuiButton-startIcon': { mr: narrow ? 0 : 1 } }}>{narrow ? null : label}</Button>;
}
export function RecordPager({ previous, next, counter, onChange, disabled = false, focusOnMount }: {
  counter?: { position: number; count: number }; previous: boolean; next: boolean; onChange: (direction: 'previous' | 'next') => void; disabled?: boolean; focusOnMount?: string;
}) {
  const before = useRef<HTMLButtonElement>(null), after = useRef<HTMLButtonElement>(null);
  useEffect(() => {
    if (!focusOnMount) return;
    const requested = focusOnMount === 'previous' ? before.current : after.current;
    (requested && !requested.disabled ? requested : before.current && !before.current.disabled ? before.current : after.current && !after.current.disabled ? after.current : null)?.focus({ preventScroll: true });
  }, [focusOnMount, previous, next]);
  return <Box component="nav" className="record-pager" aria-label="对象导航">
    {counter && <Box component="span" className="record-pager-counter" aria-label={`第${counter.position}个，共${counter.count}个匹配对象`}>{counter.position} / {counter.count}</Box>}
    <Box className="record-pager-buttons"><IconButton ref={before} size="small" disabled={disabled || !previous} aria-label="上一个对象" onClick={() => onChange('previous')}><ChevronLeftIcon /></IconButton><IconButton ref={after} size="small" disabled={disabled || !next} aria-label="下一个对象" onClick={() => onChange('next')}><ChevronRightIcon /></IconButton></Box>
  </Box>;
}
