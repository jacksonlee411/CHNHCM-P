import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { unwrap, ApiError, sessionGuard } from '@chnhcm/api-client';
import type { ListPage, NavigationPort, NavigationDirection, NavigationResult } from './list';
import type { ApiResponse } from './useSubmission';
import { usePathCursors } from './usePathCursors';

/** Resolve each direction against the current anchor; detail has its own read. */
export function useObjectNavigation(request: NavigationPort | undefined, base: string, id: string, back: string, ready = false, readPage?: (query: URLSearchParams, signal: AbortSignal) => Promise<ApiResponse<ListPage>>, detailFailed = false) {
  const navigate = useNavigate(), epoch = useRef(0), inFlight = useRef<AbortController | undefined>(undefined);
  const handoff = useRef<{ id: string; back: string; value: NavigationResult } | undefined>(undefined);
  const [counter, setCounter] = useState<{ position: number; count: number }>();
  const [counterError, setCounterError] = useState('');
  const [neighbors, setNeighbors] = useState<{ previous: boolean; next: boolean }>();
  const [retry, setRetry] = useState(0);
  const [busy, setBusy] = useState(false), [error, setError] = useState('');
  const query = new URL(back, window.location.origin).searchParams;
  const positions = usePathCursors(base, query);
  const pathMode = query.get('filterField') === 'departmentPath';
  const fail = (e: unknown) => { setCounter(undefined); setNeighbors(undefined); setError(e instanceof ApiError && e.status === 409 ? '当前记录不在查询结果中' : '暂时无法切换记录'); };
  useEffect(() => {
    const controller = new AbortController(); inFlight.current = controller;
    const invalidate = () => { epoch.current++; inFlight.current?.abort(); };
    const clearSession = () => { handoff.current = undefined; setCounter(undefined); setNeighbors(undefined); invalidate(); };
    window.addEventListener('chnhcm-session-invalidated', clearSession);
    const token = ++epoch.current, same = sessionGuard();
    if (detailFailed) handoff.current = undefined;
    setNeighbors(undefined); setCounter(undefined); setCounterError(''); setError(''); setBusy(false);
    const accept = (value: NavigationResult) => {
      if (epoch.current !== token || !same()) return;
      if (value.targetId !== id || !value.position || !value.count) throw new Error('导航位置不可用');
      setCounter({ position: value.position, count: value.count });
      if (!pathMode) setNeighbors({ previous: !!value.previousId, next: !!value.nextId });
    };
    if (request && ready && query.get('selected') === id) {
      if (!pathMode && handoff.current?.id === id && handoff.current.back === back) {
        accept(handoff.current.value); handoff.current = undefined;
      } else {
        // Path totals exhaust Go matching independently; arrows remain usable
        // while that auxiliary statistic is loading or unavailable.
        void request(query, id, 'current', controller.signal).then(value => accept(unwrap(value))).catch(e => {
          if (controller.signal.aborted || epoch.current !== token || !same()) return;
          if (pathMode) setCounterError('位置与总数暂时无法加载'); else fail(e);
        });
        if (pathMode) void Promise.all([
          request(query, id, 'previous', controller.signal), request(query, id, 'next', controller.signal),
        ]).then(([previous, next]) => { if (epoch.current === token && same()) setNeighbors({ previous: !!unwrap(previous).targetId, next: !!unwrap(next).targetId }); }).catch(e => { if (epoch.current === token && same()) fail(e); });
      }
    }
    return () => { invalidate(); window.removeEventListener('chnhcm-session-invalidated', clearSession); };
  }, [base, id, back, ready, retry, detailFailed]);
  const move = async (direction: NavigationDirection) => {
    if (!request || !neighbors?.[direction] || busy) return;
    inFlight.current?.abort();
    const controller = new AbortController(); inFlight.current = controller;
    const token = ++epoch.current, same = sessionGuard();
    setBusy(true); setError('');
    try {
      const result = await request(query, id, direction, controller.signal);
      if (epoch.current !== token || !same()) return;
      const value = unwrap(result), target = value.targetId;
      if (!target) { setNeighbors(current => current && ({ ...current, [direction]: false })); return; }
      const returned = new URLSearchParams(query);
      // Ordinary return pages use the server position. Path cursor pages
      // retain their bounded current/adjacent-page lookup.
      if (!pathMode && value.position) returned.set('page', String(Math.floor((value.position - 1) / Number(query.get('pageSize') ?? 20)) + 1));
      if (pathMode && readPage) try {
        const current = unwrap(await readPage(query, controller.signal));
        if (!current.items.some(row => row.id === target)) {
          const adjacent = current.page + (direction === 'next' ? 1 : -1);
          const cursor = direction === 'next' ? current.nextCursor : positions.pages[adjacent];
          if (adjacent >= 1 && (!pathMode || cursor !== undefined)) {
            const candidate = new URLSearchParams(query);
            candidate.set('page', String(adjacent));
            if (pathMode) { if (cursor) candidate.set('cursor', cursor); else candidate.delete('cursor'); }
            const page = unwrap(await readPage(candidate, controller.signal));
            if (page.items.some(row => row.id === target)) {
              returned.set('page', String(page.page));
              if (pathMode) {
                positions.remember(page.page, cursor ?? '');
                if (cursor) returned.set('cursor', cursor); else returned.delete('cursor');
              }
            }
          }
        }
      } catch { /* A failed return-position read cannot hide the resolved detail. */ }
      if (epoch.current !== token || !same()) return;
      returned.set('selected', target);
      if (!pathMode && value.position && value.count) handoff.current = { id: target, back: `${base}?${returned}`, value };
      navigate(`${base}/${target}?${new URLSearchParams({ back: `${base}?${returned}` })}`, { state: { recordPagerFocus: direction } });
    } catch (e) { if (epoch.current === token && same()) fail(e); }
    finally { if (epoch.current === token && same()) setBusy(false); }
  };
  return { counter, counterError, retry: () => setRetry(n => n + 1), available: !!neighbors, previous: !!neighbors?.previous, next: !!neighbors?.next, busy, error, move };
}
