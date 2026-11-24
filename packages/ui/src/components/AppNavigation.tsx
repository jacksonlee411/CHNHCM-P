import { Fragment, useEffect, useLayoutEffect, useRef, useState, type MouseEvent, type KeyboardEvent } from 'react';
import { Box, Button, Chip, ClickAwayListener, Collapse, Divider, Drawer, IconButton, List, ListItemButton, ListItemText, ListSubheader, MenuItem, MenuList, Paper, Popper, Stack, Typography } from '@mui/material';
import AppsIcon from '@mui/icons-material/Apps';
import MenuIcon from '@mui/icons-material/Menu';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import CloseIcon from '@mui/icons-material/Close';
import type { NavigationApp, NavigationLeaf } from '../navigation';
import { uiText } from '../text';

interface AppNavigationProps {
  apps: NavigationApp[];
  activeApp: NavigationApp;
  activeSection?: string;
  activePath?: string;
  desktop: boolean;
  t?: (key: string) => string;
  go: (path: string) => void;
}

export function AppNavigation({ apps, activeApp, activeSection, activePath, desktop, t = uiText, go }: AppNavigationProps) {
  const holder = useRef<HTMLDivElement>(null), desktopBar = useRef<HTMLDivElement>(null);
  const appButton = useRef<HTMLButtonElement>(null), mobileButton = useRef<HTMLButtonElement>(null);
  const drawerFocus = useRef(false), [fits, setFits] = useState(true);
  const fullNavigation = desktop && fits;
  const [appAnchor, setAppAnchor] = useState<HTMLElement | null>(null);
  const [sectionAnchor, setSectionAnchor] = useState<HTMLElement | null>(null);
  const [sectionLabel, setSectionLabel] = useState<string | null>(null);
  const [mobileOpen, setMobileOpen] = useState(false);
  const [mobileApp, setMobileApp] = useState(activeApp.id);
  const [mobileSection, setMobileSection] = useState<string | null>(activeSection ?? null);

  // The same bar remains measurable while hidden. Its intrinsic width is independent of drawer mode.
  useLayoutEffect(() => {
    const container = holder.current, bar = desktopBar.current; if (!container || !bar) return;
    const measure = () => setFits(Math.ceil(bar.getBoundingClientRect().width) <= Math.floor(container.getBoundingClientRect().width));
    measure(); const observer = new ResizeObserver(measure); observer.observe(container); observer.observe(bar);
    return () => observer.disconnect();
  }, [activeApp, desktop, t]);
  const focusTrigger = (target?: HTMLElement | null) => requestAnimationFrame(() => {
    const usable = target?.isConnected && target.getClientRects().length && !target.closest('[inert]');
    (usable ? target : holder.current?.dataset.navigationMode === 'desktop' ? appButton.current : mobileButton.current)?.focus();
  });
  useEffect(() => {
    const focused = document.activeElement;
    const wasInNavigation = focused && (desktopBar.current?.contains(focused) || focused === mobileButton.current || !!appAnchor || !!sectionAnchor);
    setAppAnchor(null); setSectionAnchor(null); setSectionLabel(null);
    if (fullNavigation && mobileOpen) { drawerFocus.current = true; setMobileOpen(false); }
    if (wasInNavigation) focusTrigger();
  }, [fullNavigation]);
  useEffect(() => {
    setAppAnchor(null); setSectionAnchor(null); setSectionLabel(null); setMobileOpen(false);
    setMobileApp(activeApp.id); setMobileSection(activeSection ?? null);
  }, [activeApp.id, activePath, activeSection]);

  const navigate = (path: string) => {
    setAppAnchor(null); setSectionAnchor(null); setSectionLabel(null); drawerFocus.current = false; setMobileOpen(false);
    go(path);
  };
  const chooseApp = (app: NavigationApp) => navigate(app.path ?? app.sections.flatMap(section => section.items).find(item => !item.placeholder)?.path ?? app.sections[0]!.items[0]!.path);
  const closeSection = () => { setSectionAnchor(null); setSectionLabel(null); };
  const openSection = (event: MouseEvent<HTMLElement>, label: string, toggle = true) => {
    setAppAnchor(null);
    if (toggle && sectionAnchor && sectionLabel === label) { closeSection(); return; }
    setSectionAnchor(event.currentTarget); setSectionLabel(label);
  };
  const openMobile = () => { setMobileApp(activeApp.id); setMobileSection(activeSection ?? null); drawerFocus.current = false; setMobileOpen(true); };
  const closeMobile = () => { drawerFocus.current = true; setMobileOpen(false); };
  const currentSection = activeApp.sections.find(section => section.label === sectionLabel);
  const menuItem = (item: NavigationLeaf) => <MenuItem key={item.path} selected={activePath === item.path} aria-current={activePath === item.path ? 'page' : undefined} onClick={() => navigate(item.path)} sx={{ minWidth: 190, gap: 1.5, justifyContent: 'space-between' }}>
    <span>{t(item.label)}</span>{item.placeholder && <Chip label={t('To be designed')} size="small" variant="outlined" sx={{ height: 20 }} />}
  </MenuItem>;
  const menuKeys = (event: KeyboardEvent, trigger: HTMLElement | null, close: () => void) => {
    if (!['Escape', 'Tab'].includes(event.key)) return;
    event.preventDefault(); event.stopPropagation(); close();
    if (event.key === 'Escape' || event.shiftKey) { focusTrigger(trigger); return; }
    const header = holder.current?.closest('header');
    const controls = [...(header?.querySelectorAll<HTMLElement>('button, [role="combobox"], a[href], input') ?? [])].filter(element => element.tabIndex >= 0 && !element.matches(':disabled') && element.getClientRects().length && !element.closest('[inert]'));
    focusTrigger(controls[controls.indexOf(trigger!) + 1] ?? trigger);
  };
  const arrow = (expanded: boolean) => <ExpandMoreIcon fontSize="small" sx={{ transform: expanded ? 'rotate(180deg)' : undefined, flexShrink: 0 }} />;
  const sectionKey = (appId: string, index: number) => `nav-${appId}-${index}`;

  return <>
    <Box ref={holder} sx={{ flex: '1 1 0', minWidth: 40, position: 'relative', overflow: 'hidden', display: 'flex', alignItems: 'center' }} data-navigation-mode={fullNavigation ? 'desktop' : 'drawer'}>
      <Box ref={desktopBar} aria-hidden={!fullNavigation} inert={!fullNavigation} sx={{ display: 'flex', alignItems: 'center', width: 'max-content', flexShrink: 0, gap: 0.5, ...(!fullNavigation ? { position: 'absolute', visibility: 'hidden', pointerEvents: 'none' } : {}) }}>
        <Button ref={appButton} color="inherit" startIcon={<AppsIcon />} aria-label={t('Switch application')} aria-haspopup="menu" aria-controls={appAnchor ? 'navigation-apps-menu' : undefined} aria-expanded={!!appAnchor} onClick={event => { closeSection(); setAppAnchor(appAnchor ? null : event.currentTarget); }} sx={{ flexShrink: 0, fontWeight: 700, whiteSpace: 'nowrap', px: 1.25, textTransform: 'none' }}>{t(activeApp.label)}</Button>
        {activeApp.sections.length > 0 && <Box component="nav" aria-label={t('Application sections')} sx={{ display: 'flex', alignItems: 'center', flexShrink: 0 }}>
          {activeApp.sections.map((section, index) => <Button id={sectionKey(activeApp.id, index)} key={section.label} color="inherit" aria-label={t(section.label)} endIcon={section.direct ? undefined : arrow(sectionLabel === section.label && !!sectionAnchor)} aria-haspopup={section.direct ? undefined : 'menu'} aria-controls={!section.direct && sectionLabel === section.label && sectionAnchor ? 'navigation-section-menu' : undefined} aria-expanded={section.direct ? undefined : sectionLabel === section.label && !!sectionAnchor} aria-current={activeSection === section.label ? section.direct ? 'page' : 'true' : undefined} onClick={event => section.direct ? navigate(section.items[0]!.path) : openSection(event, section.label)} onMouseEnter={event => { if (!sectionAnchor || sectionLabel === section.label) return; if (section.direct) closeSection(); else openSection(event, section.label, false); }} sx={{ flexShrink: 0, minWidth: 0, px: 1.25, whiteSpace: 'nowrap', textTransform: 'none', fontWeight: activeSection === section.label ? 700 : 400, bgcolor: activeSection === section.label ? 'rgba(255,255,255,.15)' : undefined }}>{t(section.label)}</Button>)}
        </Box>}
      </Box>
      {!fullNavigation && <><IconButton ref={mobileButton} sx={{ flexShrink: 0 }} color="inherit" aria-label={t('Open navigation')} aria-expanded={mobileOpen} aria-controls={mobileOpen ? 'navigation-drawer' : undefined} onClick={openMobile}><MenuIcon /></IconButton><Typography variant="subtitle1" title={t(activeApp.label)} sx={{ fontWeight: 700, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{t(activeApp.label)}</Typography></>}
    </Box>

    <Popper anchorEl={appAnchor} open={fullNavigation && !!appAnchor} placement="bottom-start" sx={{ zIndex: 1400 }}>
      <ClickAwayListener onClickAway={() => setAppAnchor(null)}><Paper elevation={4} sx={{ maxHeight: 'min(75vh, 680px)', minWidth: 220, maxWidth: 'calc(100vw - 24px)', overflowY: 'auto' }}>
        <MenuList id="navigation-apps-menu" autoFocusItem aria-label={t('Applications')} onKeyDown={event => menuKeys(event, appAnchor, () => setAppAnchor(null))}>
          {apps.map(app => <MenuItem key={app.id} selected={activeApp.id === app.id} onClick={() => chooseApp(app)}>{t(app.label)}</MenuItem>)}
        </MenuList>
      </Paper></ClickAwayListener>
    </Popper>
    <Popper anchorEl={sectionAnchor} open={fullNavigation && !!(sectionAnchor && currentSection && !currentSection.direct)} placement="bottom-start" sx={{ zIndex: 1400 }}>
      <ClickAwayListener onClickAway={closeSection}><Paper elevation={4} sx={{ maxHeight: 'min(75vh, 680px)', minWidth: 220, maxWidth: 'calc(100vw - 24px)', overflowY: 'auto' }}>
        <MenuList id="navigation-section-menu" autoFocusItem aria-label={t(sectionLabel ?? 'Application sections')} onKeyDown={event => menuKeys(event, sectionAnchor, closeSection)}>{currentSection?.items.flatMap((item, index, items) => [item.group && item.group !== items[index - 1]?.group ? <ListSubheader key={`group:${item.group}`} sx={{ position: 'static', lineHeight: '30px', fontSize: 12 }}>{t(item.group)}</ListSubheader> : null, menuItem(item)])}</MenuList>
      </Paper></ClickAwayListener>
    </Popper>

    <Drawer anchor="left" open={mobileOpen} onClose={closeMobile} sx={{ zIndex: theme => theme.zIndex.drawer + 2 }} ModalProps={{ disableRestoreFocus: true }} slotProps={{ paper: { id: 'navigation-drawer', sx: { width: 'min(88vw, 360px)' } }, transition: { onExited: () => { if (drawerFocus.current) { drawerFocus.current = false; focusTrigger(); } } } }}>
      <Box sx={{ p: 2 }}><Stack direction="row" justifyContent="space-between" alignItems="center"><Typography variant="h6">{t('Applications')}</Typography><IconButton aria-label={t('Close navigation')} onClick={closeMobile}><CloseIcon /></IconButton></Stack><Typography variant="body2" color="text.secondary">{t('Choose an application and task.')}</Typography></Box>
      <Divider />
      <List aria-label={t('Applications')} sx={{ overflowY: 'auto' }}>
        {apps.map(app => <Box key={app.id}>
          <ListItemButton selected={activeApp.id === app.id} aria-current={activeApp.id === app.id ? 'true' : undefined} aria-expanded={app.path ? undefined : mobileApp === app.id} onClick={() => { if (app.path) { navigate(app.path); return; } setMobileApp(mobileApp === app.id ? '' : app.id); setMobileSection(app.id === activeApp.id ? activeSection ?? null : app.sections[0]?.label ?? null); }}><ListItemText primary={t(app.label)} primaryTypographyProps={{ fontWeight: activeApp.id === app.id ? 700 : 500 }} />{!app.path && arrow(mobileApp === app.id)}</ListItemButton>
          {!app.path && <Collapse in={mobileApp === app.id} timeout="auto" unmountOnExit>
            <List component="div" role="group" disablePadding aria-label={t(app.label)}>{app.sections.map((section, index) => <Box key={section.label}>
              <ListItemButton sx={{ pl: 4 }} selected={activeApp.id === app.id && activeSection === section.label} aria-current={section.direct && activePath === section.items[0]?.path ? 'page' : undefined} aria-expanded={section.direct ? undefined : mobileSection === section.label} aria-controls={!section.direct && mobileSection === section.label ? sectionKey(app.id, index) + '-children' : undefined} onClick={() => section.direct ? navigate(section.items[0]!.path) : setMobileSection(mobileSection === section.label ? null : section.label)}><ListItemText primary={t(section.label)} secondary={section.direct && section.items[0]?.placeholder ? t('To be designed') : undefined} />{!section.direct && arrow(mobileSection === section.label)}</ListItemButton>
              {!section.direct && <Collapse in={mobileSection === section.label} timeout="auto" unmountOnExit><List id={sectionKey(app.id, index) + '-children'} component="div" role="group" disablePadding aria-label={t(section.label)}>{section.items.map((item, itemIndex, items) => <Fragment key={item.path}>{item.group && item.group !== items[itemIndex - 1]?.group && <ListSubheader sx={{ position: 'static', pl: 6 }}>{t(item.group)}</ListSubheader>}<ListItemButton key={item.path} selected={activePath === item.path} aria-current={activePath === item.path ? 'page' : undefined} sx={{ pl: 6 }} onClick={() => navigate(item.path)}><ListItemText primary={t(item.label)} secondary={item.placeholder ? t('To be designed') : undefined} /></ListItemButton></Fragment>)}</List></Collapse>}
            </Box>)}</List>
          </Collapse>}
        </Box>)}
      </List>
    </Drawer>
  </>;
}
