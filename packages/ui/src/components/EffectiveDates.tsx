import { useLayoutEffect, useRef, useState } from 'react';
import { Box, ButtonBase, Menu, MenuItem, Typography, useMediaQuery } from '@mui/material';
import ArrowDropDownIcon from '@mui/icons-material/ArrowDropDown';
import { businessDate } from '../date-display';
import { uiText as t } from '../text';
import './effective-dates.css';

export function EffectiveDates({ dates, selected, onSelect }: { dates: string[]; selected?: string; onSelect: (day: string) => void }) {
  const locale = 'zh';
  const slot = useRef<HTMLDivElement>(null), measure = useRef<HTMLDivElement>(null);
  const [bounds, setBounds] = useState({ start: 0, end: dates.length - 1, compact: true });
  const boundsRef = useRef(bounds); boundsRef.current = bounds;
  const [menu, setMenu] = useState<{ anchor: HTMLElement; side: 'before' | 'after' | 'all' } | null>(null);
  const menuRef = useRef(menu); menuRef.current = menu;
  const mobile = useMediaQuery('(max-width:767px)');
  const label = businessDate;
  const identity = dates.join(',');
  useLayoutEffect(() => {
    const container = slot.current, measurements = measure.current;
    if (!container || !measurements) return;
    let active = true;
    const layout = () => {
      if (!active) return;
      const widths = [...measurements.children].map(node => node.getBoundingClientRect().width + 24);
      const index = dates.indexOf(selected ?? '');
      let start = 0, end = dates.length - 1, compact = mobile;
      const width = () => widths.slice(start, end + 1).reduce((sum, value) => sum + value, 0) + (start > 0 ? 36 : 0) + (end < dates.length - 1 ? 36 : 0) + 6;
      while (!compact && width() > container.clientWidth) {
        if (index < 0) compact = true;
        else if (start < Math.max(0, index - 1)) start++;
        else if (end > Math.min(dates.length - 1, index + 1)) end--;
        else compact = true;
      }
      if (boundsRef.current.start !== start || boundsRef.current.end !== end || boundsRef.current.compact !== compact) {
        boundsRef.current = { start, end, compact };
        setBounds(boundsRef.current);
        if (menuRef.current) {
          requestAnimationFrame(() => slot.current?.querySelector<HTMLElement>('[aria-current="date"], [data-date-menu="all"], button')?.focus({ preventScroll: true }));
          setMenu(null);
        }
      }
    };
    const observer = new ResizeObserver(layout); observer.observe(container); layout();
    void document.fonts.ready.then(layout);
    return () => { active = false; observer.disconnect(); };
  }, [identity, selected, locale, mobile]);
  const close = () => { const anchor = menu?.anchor; setMenu(null); requestAnimationFrame(() => anchor?.isConnected ? anchor.focus() : slot.current?.querySelector<HTMLElement>('[aria-current="date"], [data-date-menu="all"]')?.focus()); };
  const choose = (date: string) => {
    const source = document.activeElement;
    const anchor = menu?.anchor;
    setMenu(null); onSelect(date);
    requestAnimationFrame(() => requestAnimationFrame(() => {
      const active = document.activeElement;
      if (active === source || active === anchor || active === document.body) {
        slot.current?.querySelector<HTMLElement>('[aria-current="date"], [data-date-menu="all"]')?.focus({ preventScroll: true });
      }
    }));
  };
  const open = (anchor: HTMLElement, side: 'before' | 'after' | 'all') => setMenu(menu?.anchor === anchor ? null : { anchor, side });
  const menuDates = menu?.side === 'before' ? dates.slice(0, bounds.start) : menu?.side === 'after' ? dates.slice(bounds.end + 1) : dates;
  const trigger = (side: 'before' | 'after', name: string) => <ButtonBase className="effective-date-piece" data-date-menu={side} aria-label={t(name)} aria-haspopup="menu" aria-expanded={menu?.side === side} onClick={event => open(event.currentTarget, side)}>…</ButtonBase>;
  return <Box component="nav" aria-label={t('Effective Dates')} className="effective-dates">
    <Typography component="h2" variant="subtitle2">{t('Effective Dates')}</Typography>
    <Box ref={slot} className="effective-date-slot" data-mode={!dates.length ? 'empty' : bounds.compact ? 'compact' : 'strip'}>
      <Box ref={measure} aria-hidden="true" className="effective-date-measure">{dates.map(date => <span key={date}>{label(date)}</span>)}</Box>
      {!!dates.length && (bounds.compact ? <ButtonBase className="effective-date-compact" data-date-menu="all" aria-label={t('Select effective date') + (selected ? ': ' + label(selected) : '')} aria-current={selected ? 'date' : undefined} aria-haspopup="menu" aria-expanded={menu?.side === 'all'} onClick={event => open(event.currentTarget, 'all')}><span>{selected ? label(selected) : t('Select effective date')}</span><ArrowDropDownIcon fontSize="small" /></ButtonBase> : <Box className="effective-date-strip" onKeyDown={event => {
        if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
        const buttons = [...event.currentTarget.querySelectorAll<HTMLButtonElement>('button')];
        const index = buttons.indexOf(document.activeElement as HTMLButtonElement);
        event.preventDefault(); buttons[event.key === 'Home' ? 0 : event.key === 'End' ? buttons.length - 1 : Math.max(0, Math.min(buttons.length - 1, index + (event.key === 'ArrowLeft' ? -1 : 1)))]?.focus();
      }}>
        {bounds.start > 0 && trigger('before', 'Earlier effective dates')}
        {dates.slice(bounds.start, bounds.end + 1).map(date => <ButtonBase key={date} className={'effective-date-piece' + (date === selected ? ' selected' : '')} data-day={date} aria-current={date === selected ? 'date' : undefined} onClick={() => choose(date)}>{label(date)}</ButtonBase>)}
        {bounds.end < dates.length - 1 && trigger('after', 'Later effective dates')}
      </Box>)}
    </Box>
    <Menu anchorEl={menu?.anchor} open={!!menu} onClose={close} disableRestoreFocus slotProps={{ paper: { sx: { maxHeight: 320 } }, list: { 'aria-label': t('Effective Dates') } }}>
      {menuDates.map(date => <MenuItem key={date} selected={date === selected} aria-current={date === selected ? 'date' : undefined} data-day={date} onClick={() => choose(date)} sx={{ fontSize: 14, '&.Mui-selected, &.Mui-selected:hover, &.Mui-selected.Mui-focusVisible': { bgcolor: 'var(--chnhcm-action-color)', color: 'var(--chnhcm-on-primary)' } }}>{label(date)}</MenuItem>)}
    </Menu>
  </Box>;
}
