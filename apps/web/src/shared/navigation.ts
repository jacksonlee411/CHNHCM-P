import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type RefObject } from 'react';
import type { GridApi } from '@mui/x-data-grid';
import { useLocation, useNavigationType, useSearchParams } from 'react-router-dom';

/** Ordinary maintenance links locate a record. Only a return to a report keeps its as-of date. */
export function useMaintenanceSearchParams(listPath: string) {
  const [params, setParams] = useSearchParams();
  const clean = useMemo(() => {
    const next = new URLSearchParams(params);
    next.delete('businessAt');
    next.delete('at');
    const back = next.get('back');
    if (back?.startsWith(listPath + '?')) {
      const query = new URLSearchParams(back.slice(listPath.length + 1));
      if (query.has('businessAt') || query.has('at')) {
        query.delete('businessAt');
        query.delete('at');
        next.set('back', query.size ? listPath + '?' + query : listPath);
      }
    }
    return next.toString() === params.toString() ? params : next;
  }, [params, listPath]);
  useEffect(() => {
    if (clean !== params) setParams(clean, { replace: true });
  }, [clean, params, setParams]);
  return [clean, setParams] as const;
}

/** The responsive layout has either a bounded body or one document scroll path. */
function scrollContainer(node: HTMLElement) {
  return ['auto', 'scroll'].includes(getComputedStyle(node).overflowY)
    ? node : document.scrollingElement as HTMLElement;
}

export function usePageTitle(title: string) {
  useEffect(() => {
    document.title = `${title} · CHNHCM`;
    return () => { document.title = 'CHNHCM'; };
  }, [title]);
}

/** Only presentation positions are retained. No object facts or authorization are cached. */
export function useListReturn({ key, ready, apiRef, selected, rows, pageOffset }: {
  key: string; ready: boolean; apiRef: RefObject<GridApi | null>;
  selected: string | null; rows: { id: string }[]; pageOffset: number;
}) {
  const location = useLocation(), navigation = useNavigationType();
  const scroller = useRef<HTMLDivElement>(null);
  const previous = useRef(location.key), restore = useRef(true), interrupted = useRef(false);
  const cancelRestore = useRef<() => void>(() => {});
  const focusing = useRef(false);
  const [clearedAt, setClearedAt] = useState<string>();
  const saved = useMemo(() => {
    try { return JSON.parse(sessionStorage.getItem(key) ?? 'null') as {
      gridTop?: number; documentTop?: number; rowId?: string; childLink?: boolean; locationKey?: string; selectedAtClick?: string | null;
    } | null; } catch { return null; }
  }, [key, location.key]);
  const targetId = clearedAt === location.key ? null
    : (saved?.locationKey === location.key && saved.selectedAtClick === selected ? saved.rowId : undefined) ?? selected;
  const clearReturn = () => {
    cancelRestore.current();
    restore.current = false;
    setClearedAt(location.key);
    try { sessionStorage.removeItem(key); } catch { /* Local state still ends return positioning. */ }
  };
  useLayoutEffect(() => {
    if (previous.current !== location.key) {
      restore.current = navigation === 'POP' || location.state?.restoreChildLink === true;
      interrupted.current = false;
      previous.current = location.key;
    }
    const interrupt = (event: Event) => {
      if (event.type === 'focusin' && focusing.current) return;
      interrupted.current = true; cancelRestore.current();
    };
    const events = ['pointerdown', 'keydown', 'wheel', 'touchstart', 'focusin'];
    events.forEach(name => document.addEventListener(name, interrupt, true));
    return () => events.forEach(name => document.removeEventListener(name, interrupt, true));
  }, [location.key, navigation]);
  useLayoutEffect(() => {
    const node = scroller.current;
    if (!ready || !node || !restore.current || interrupted.current) return;
    restore.current = false;
    if ((!targetId && saved === null) || document.activeElement?.matches('input, textarea, [role=combobox]')) return;
    const page = node.closest('.standard-page');
    const fallback = () => {
      const target = page?.querySelector<HTMLElement>('[data-list-return-fallback]') ?? page?.querySelector<HTMLElement>('[data-list-heading]');
      target?.focus({ preventScroll: true });
      if (target?.hasAttribute('data-list-return-fallback')) target.scrollIntoView({ block: 'nearest' });
    };
    const api = apiRef.current;
    if (!api?.scroll) { fallback(); return; }
    const documentScroll = document.scrollingElement as HTMLElement;
    const natural = matchMedia('(max-width:599px), (max-height:600px)').matches;
    const restoreTop = () => {
      if (natural) documentScroll.scrollTop = Math.max(0, saved?.documentTop ?? 0);
      else api.scroll({ top: Math.max(0, saved?.gridTop ?? 0) });
    };
    restoreTop();
    const index = rows.findIndex(row => row.id === targetId);
    if (index < 0) { fallback(); return; }
    let frame = 0, focused = false;
    let unsubscribe: (() => void) | undefined;
    const stop = () => { observer.disconnect(); unsubscribe?.(); cancelAnimationFrame(frame); };
    const focus = () => {
      if (interrupted.current || !node.isConnected) { stop(); return; }
      const row = api.getRowElement(targetId!);
      const childLink = location.state?.restoreChildLink || (saved?.locationKey === location.key && saved.childLink);
      const link = (childLink ? row?.querySelector<HTMLElement>('a[data-child-link]') : undefined) ?? row?.querySelector<HTMLElement>('a[data-object-link]');
      if (link) {
        if (!focused) {
          // Loading a virtual row can move the viewport after the first scroll.
          // Restore its original position once the target link actually exists.
          if (saved?.rowId === targetId) restoreTop();
          focused = true; focusing.current = true;
          try { link.focus({ preventScroll: true }); }
          finally { focusing.current = false; }
        }
        const rect = link.getBoundingClientRect();
        const bounds = natural ? { top: 0, bottom: innerHeight } : node.querySelector('.MuiDataGrid-virtualScroller')?.getBoundingClientRect();
        if (rect.top < Math.max(0, bounds?.top ?? 0) || rect.bottom > Math.min(innerHeight, bounds?.bottom ?? innerHeight)) link.scrollIntoView({ block: 'nearest' });
        return;
      }
      if (row) { stop(); fallback(); return; }
      // MUI 8.29.3 subtracts page * pageSize internally, also in server mode.
      api.scrollToIndexes({ rowIndex: pageOffset + index });
    };
    const schedule = () => {
      cancelAnimationFrame(frame);
      frame = requestAnimationFrame(focus);
    };
    const observer = new MutationObserver(schedule);
    observer.observe(node, { childList: true, subtree: true });
    // Auto row measurements can shift the target after its link first receives focus.
    unsubscribe = api.subscribeEvent('stateChange', schedule);
    cancelRestore.current = stop;
    frame = requestAnimationFrame(focus);
    return () => { stop(); cancelRestore.current = () => {}; };
  }, [ready, key, location.key, targetId, saved]);
  const rememberRow = useCallback((rowId?: string, childLink = false) => {
    try {
      if (scroller.current) sessionStorage.setItem(key, JSON.stringify({
        gridTop: apiRef.current?.getScrollPosition?.().top ?? 0,
        documentTop: document.scrollingElement?.scrollTop ?? 0,
        rowId, childLink, locationKey: location.key, selectedAtClick: selected,
      }));
    } catch { /* Navigation remains available. */ }
  }, [key, location.key, selected]);
  const remember = useCallback(() => rememberRow(), [rememberRow]);
  return { scroller, remember, rememberRow, clearReturn, selected: targetId };
}

export function usePageScrollRestoration() {
  const location = useLocation(), navigation = useNavigationType();
  const positions = useRef(new Map<string, number>());
  useLayoutEffect(() => {
    const target = navigation === 'POP' ? positions.current.get(location.key) ?? 0 : 0;
    let node: HTMLElement | null = null, restored = false;
    // Wait for the authorized content; a loading placeholder would clamp the offset.
    const restore = () => {
      node = document.querySelector<HTMLElement>('[data-page-scroll]');
      if (restored || !node || node.dataset.pageScrollReady === 'false') return;
      restored = true;
      scrollContainer(node).scrollTop = target;
      observer.disconnect();
    };
    const observer = new MutationObserver(restore);
    observer.observe(document.getElementById('root')!, { childList: true, subtree: true, attributes: true, attributeFilter: ['data-page-scroll-ready'] });
    const frame = requestAnimationFrame(restore);
    const remember = (event: Event) => {
      if (!restored || !node) return;
      const container = scrollContainer(node);
      if (event.target === container || (container === document.scrollingElement && event.target === document)) positions.current.set(location.key, container.scrollTop);
    };
    document.addEventListener('scroll', remember, true);
    return () => { cancelAnimationFrame(frame); observer.disconnect(); document.removeEventListener('scroll', remember, true); };
  }, [location.key, navigation]);
}
