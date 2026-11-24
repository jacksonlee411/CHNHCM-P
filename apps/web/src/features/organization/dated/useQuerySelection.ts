import { useEffect, useLayoutEffect, useRef, useState } from 'react';
import { unwrap, ApiError, sessionGuard } from '@chnhcm/api-client';
import type { ApiResponse } from './useSubmission';
import type { ListPage, Selection, SelectionPort, SelectionPosition, SelectionSummary, DraftReference, DraftSummary, DraftEdit } from './list';

export class ScopePageUnavailable extends Error { constructor() { super('当前范围不可用'); } }
type Saved = { key: string; version: string; choice: Selection; query: string; total?: number; draft?: DraftReference; day?: string; pageIDs?: string[]; memberIDs?: string[] };
export function useQuerySelection<Q, S extends string, P extends ListPage>(port: SelectionPort<Q, S, P> | undefined, data: P | undefined, query: Q, position: SelectionPosition<S>, failed: boolean, historyKey: string | null, reload: () => void, acceptPage: (page: P) => void) {
  const [saved, setSaved] = useState<Saved>();
  const savedRef = useRef<Saved | undefined>(undefined);
  const [pendingChoice, setPendingChoice] = useState<Selection>();
  const intent = useRef<Selection | undefined>(undefined);
  const positionIntent = useRef<(() => void) | undefined>(undefined);
  const [summary, setSummary] = useState<SelectionSummary | DraftSummary>();
  const [notice, setNotice] = useState('');
  const [noticeSeverity, setNoticeSeverity] = useState<'error' | 'success'>('error');
  const failNotice = (value: string) => { setNoticeSeverity('error'); setNotice(value); };
  const [exporting, setExporting] = useState(false);
  const epoch = useRef(0), download = useRef<AbortController | undefined>(undefined), command = useRef<AbortController | undefined>(undefined), pageRead = useRef<AbortController | undefined>(undefined);
  const previousHistory = useRef(historyKey);
  const queryJSON = JSON.stringify(query);
  const remember = (value: Saved | undefined) => { savedRef.current = value; setSaved(value); };
  const cancel = () => { epoch.current++; command.current?.abort(); command.current = undefined; pageRead.current?.abort(); pageRead.current = undefined; download.current?.abort(); intent.current = undefined; positionIntent.current = undefined; setExporting(false); setPendingChoice(undefined); setSummary(undefined); };
  const clear = () => {
    const restore = Boolean(port && document.activeElement?.closest(port.toolbarSelector));
    const previous = savedRef.current;
    cancel(); remember(undefined); setNotice(''); setNoticeSeverity('error');
    if (port?.protocol === 'identity-draft' && previous?.draft) void port.cancel(previous.draft).catch(() => {});
    if (restore && port) requestAnimationFrame(() => { if (document.activeElement === document.body) document.querySelector<HTMLElement>(port.returnFocusSelector)?.focus(); });
  };
  useLayoutEffect(() => { clear(); }, [port?.protocol]);
  useLayoutEffect(() => {
    if (historyKey && historyKey !== previousHistory.current) clear();
    previousHistory.current = historyKey;
  }, [historyKey]);
  useLayoutEffect(() => {
    cancel();
    if (savedRef.current && savedRef.current.query !== queryJSON) clear();
  }, [queryJSON, position.page, position.pageSize, position.sort, position.cursor]);
  useLayoutEffect(() => {
    if (failed) { clear(); return; }
    if (!saved || !data) return;
    if (saved.day !== undefined && saved.day !== data.businessAt || saved.key !== data.queryKey || (port?.protocol === 'list-version' && saved.version !== port.version(data))) {
      const sameQuery = saved.key === data.queryKey; clear();
      if (sameQuery) failNotice('选择已失效，请重新选择');
    }
  }, [data, failed]);
  useLayoutEffect(() => () => { epoch.current++; command.current?.abort(); pageRead.current?.abort(); download.current?.abort(); const current = savedRef.current; if (port?.protocol === 'identity-draft' && current?.draft) void port.cancel(current.draft).catch(() => {}); }, []);
  useEffect(() => {
    const invalidate = () => { clear(); reload(); };
    window.addEventListener('chnhcm-session-invalidated', invalidate);
    return () => window.removeEventListener('chnhcm-session-invalidated', invalidate);
  }, []);
  const valid = port && saved?.query === queryJSON && (port?.protocol === 'identity-draft' ? Boolean(saved.draft && summary?.queryKey === saved.key) : Boolean(data && saved.key === data.queryKey && saved.version === port.version(data))) ? saved : undefined;
  useEffect(() => {
    if (port?.protocol !== 'list-version') return;
    const token = ++epoch.current, controller = new AbortController(), sameSession = sessionGuard();
    download.current?.abort(); setExporting(false); setSummary(undefined);
    if (valid && data) void (async () => {
      const response = await port.resolve(query, valid.choice, valid.version, controller.signal);
      if (controller.signal.aborted || epoch.current !== token || !sameSession()) return;
      setSummary(unwrap(response)); setNotice('');
    })().catch(error => {
      if (controller.signal.aborted || epoch.current !== token || !sameSession()) return;
      if (error instanceof ApiError && [401,403,409].includes(error.status) && error.code !== 'AUTH_REQUEST_VALIDATION_FAILED') { clear(); reload(); failNotice('选择已失效，请重新选择'); } else failNotice(error instanceof ApiError && [413,429].includes(error.status) ? error.message : '暂时无法核对选择，请重试');
    });
    return () => { controller.abort(); epoch.current++; download.current?.abort(); };
  }, [saved, data, queryJSON, position.sort]);

  // While selected, the page and summary come from one resolve request. The
  // caller's owned-read guard still rejects results for an obsolete route key.
  const requestPage = async (load: () => Promise<ApiResponse<P>>): Promise<ApiResponse<P>> => {
    const current = savedRef.current;
    if (port?.protocol !== 'identity-draft') return load();
    if (!current || current.query !== queryJSON) {
      const token = epoch.current, sameSession = sessionGuard();
      const response = await load();
      if (token !== epoch.current || !sameSession()) throw new DOMException('过期的列表读取', 'AbortError');
      return response;
    }
    const controller = new AbortController(), token = ++epoch.current, sameSession = sessionGuard();
    pageRead.current?.abort(); pageRead.current = controller;
    setSummary(undefined);
    try {
      const response = await port.resolvePage(query, current.draft!, {}, position, controller.signal);
      controller.signal.throwIfAborted();
      if (!sameSession() || token !== epoch.current) throw new DOMException('过期的选择读取', 'AbortError');
      const result = unwrap(response);
      remember({ ...current, draft: result.summary, day: result.businessAt, pageIDs: result.pageState === 'available' ? result.selectedIds : [], memberIDs: result.pageState === 'available' ? result.memberIds : [] });
      setSummary(result.summary); setNotice('');
      if (result.pageState === 'scope_unavailable') throw new ScopePageUnavailable();
      return { ...response, data: result.page, error: undefined };
    } catch (error) {
      if (error instanceof ScopePageUnavailable) throw error;
      if (!controller.signal.aborted && token === epoch.current && sameSession()) {
        if (error instanceof ApiError && [401,403,409].includes(error.status) && error.code !== 'AUTH_REQUEST_VALIDATION_FAILED') clear(); else setSummary(undefined);
        failNotice(error instanceof ApiError ? error.message : '选择核对失败，请重试');
        if (error instanceof ApiError && [401,403,409].includes(error.status) && error.code !== 'AUTH_REQUEST_VALIDATION_FAILED') reload();
      }
      throw error;
    } finally { if (pageRead.current === controller) pageRead.current = undefined; }
  };
  const change = (choice: Selection) => {
    if (!data || !port) return;
    setNotice('');
    if (choice.mode === 'explicit' && !choice.ids.length || choice.mode === 'allMatching' && choice.excludedIds.length === savedRef.current?.total) {
      clear(); if (port?.protocol === 'identity-draft') reload(); return;
    }
    if (port?.protocol === 'list-version') { cancel(); remember({ key: data.queryKey ?? '', version: port.version(data), choice, query: queryJSON, total: data.count }); return; }
    // One establishment owns the baseline. Later clicks update its intent;
    // they must not silently capture a newer baseline with another start.
    if (command.current) { intent.current = choice; setPendingChoice(choice); return; }
    cancel(); intent.current = choice;
    const current = savedRef.current, controller = new AbortController(), token = epoch.current, sameSession = sessionGuard();
    command.current = controller; setPendingChoice(choice);
    void (async () => {
      let draft = current?.draft, confirmed = current?.choice;
      while (intent.current) {
        const requested = intent.current;
        let response;
        if (!draft || confirmed?.mode !== requested.mode) {
          const first = requested.mode === 'explicit' ? { mode: 'explicit' as const, ids: requested.ids.slice(0, 1024) } : requested;
          response = await port.start(query, first, position, controller.signal);
          confirmed = first;
        } else {
          const previous = new Set(confirmed.mode === 'explicit' ? confirmed.ids : confirmed.excludedIds);
          const next = new Set(requested.mode === 'explicit' ? requested.ids : requested.excludedIds);
          const delta: { field: keyof DraftEdit; id: string }[] = [];
          for (const id of next) if (!previous.has(id)) delta.push({ field: requested.mode === 'explicit' ? 'addIds' : 'removeIds', id });
          for (const id of previous) if (!next.has(id)) delta.push({ field: requested.mode === 'explicit' ? 'removeIds' : 'restoreIds', id });
          const edit: DraftEdit = {};
          for (const { field, id } of delta.slice(0, 1024)) { (edit[field] ??= []).push(id); if (next.has(id)) previous.add(id); else previous.delete(id); }
          response = await port.resolvePage(query, draft, edit, position, controller.signal);
          confirmed = requested.mode === 'explicit' ? { mode: 'explicit', ids: [...previous] } : { mode: 'allMatching', excludedIds: [...previous] };
        }
        const result = unwrap(response);
        if (controller.signal.aborted || token !== epoch.current || !sameSession()) { void port.cancel(result.summary).catch(() => {}); return; }
        if (result.pageState !== 'available') throw new ScopePageUnavailable();
        draft = result.summary;
        remember({ key: result.summary.queryKey, version: '', draft, choice: confirmed, query: queryJSON, day: result.page.businessAt, pageIDs: result.selectedIds, memberIDs: result.memberIds,
          total: result.summary.selectedCount + (confirmed.mode === 'allMatching' ? confirmed.excludedIds.length : 0) });
        const done = confirmed.mode === requested.mode && (confirmed.mode === 'explicit' && requested.mode === 'explicit'
          ? confirmed.ids.length === requested.ids.length && confirmed.ids.every(id => requested.ids.includes(id))
          : confirmed.mode === 'allMatching' && requested.mode === 'allMatching' && confirmed.excludedIds.length === requested.excludedIds.length && confirmed.excludedIds.every(id => requested.excludedIds.includes(id)));
        if (!done) continue;
        if (requested !== intent.current) continue;
        if (result.summary.selectedCount === 0) { clear(); reload(); return; }
        intent.current = undefined;
        setPendingChoice(undefined); setSummary(result.summary); acceptPage(result.page);
      }
    })().catch(async error => {
      if (controller.signal.aborted || token !== epoch.current || !sameSession()) return;
      const prior = savedRef.current;
      // Resolve only the known revision with no edits. A committed lost edit
      // conflicts; an unapplied edit leaves the old selection intact.
      if (prior?.draft && !(error instanceof ApiError && [401,403,409].includes(error.status) && error.code !== 'AUTH_REQUEST_VALIDATION_FAILED')) {
        try {
          const response = await port.resolvePage(query, prior.draft, {}, position, controller.signal);
          if (controller.signal.aborted || token !== epoch.current || !sameSession()) return;
          const result = unwrap(response);
          intent.current = undefined; setPendingChoice(undefined);
          remember({ ...prior, draft: result.summary, pageIDs: result.pageState === 'available' ? result.selectedIds : [], memberIDs: result.pageState === 'available' ? result.memberIds : [] });
          setSummary(result.summary);
          if (result.pageState === 'available') acceptPage(result.page);
          failNotice(error instanceof ApiError && [413,429].includes(error.status) ? error.message : '本次选择修改未完成，请重新选择要增减的项目');
          return;
        } catch (confirmationError) {
          if (controller.signal.aborted || token !== epoch.current || !sameSession()) return;
          if (!(confirmationError instanceof ApiError && [401,403,409].includes(confirmationError.status) && confirmationError.code !== 'AUTH_REQUEST_VALIDATION_FAILED')) {
            intent.current = undefined; setPendingChoice(undefined); setSummary(undefined);
            failNotice('暂时无法核对选择，请重试'); return;
          }
        }
      }
      clear(); if (error instanceof ApiError && [401,403,409].includes(error.status) && error.code !== 'AUTH_REQUEST_VALIDATION_FAILED') reload(); failNotice(error instanceof ApiError && [413,429].includes(error.status) ? error.message : '选择未完成，请重新选择');
    }).finally(() => {
      if (command.current !== controller) return;
      command.current = undefined;
      const applyPosition = positionIntent.current;
      positionIntent.current = undefined;
      if (!controller.signal.aborted && token === epoch.current && sameSession()) applyPosition?.();
    });
  };
  // Keep the original establishment/version and latest checked IDs when a
  // position change arrives during selection validation. Only the latest
  // position is queued; cancellation and query/session changes discard it.
  const reposition = (apply: () => void) => {
    if (port?.protocol === 'identity-draft' && command.current) positionIntent.current = apply;
    else apply();
  };
  const visualChoice = pendingChoice ?? valid?.choice;
  const selected = new Set(data?.items.filter(row => {
    if (port?.protocol === 'list-version' || !valid?.draft) return visualChoice?.mode === 'allMatching' ? !visualChoice.excludedIds.includes(row.id) : visualChoice?.ids.includes(row.id);
    if (!pendingChoice) return valid.pageIDs?.includes(row.id);
    if (pendingChoice.mode === 'explicit') return pendingChoice.ids.includes(row.id);
    return !pendingChoice.excludedIds.includes(row.id) && (valid.pageIDs?.includes(row.id) || valid.choice.mode === 'allMatching' && valid.choice.excludedIds.includes(row.id));
  }).map(row => row.id));
  const setPage = (next: Set<string>) => {
    if (!data || !port) return;
    const choice = visualChoice ?? { mode: 'explicit' as const, ids: [] };
    const ids = new Set(choice.mode === 'explicit' ? choice.ids : choice.excludedIds);
    for (const row of data.items) { if (choice.mode === 'allMatching' && valid?.draft && !valid.memberIDs?.includes(row.id)) continue; if (choice.mode === 'explicit' ? next.has(row.id) : !next.has(row.id)) ids.add(row.id); else ids.delete(row.id); }
    change(choice.mode === 'explicit' ? { mode: 'explicit', ids: [...ids] } : { mode: 'allMatching', excludedIds: [...ids] });
  };
  const exportSelected = async () => {
    if (!port || !valid || pendingChoice || !summary?.canExport || exporting || (port?.protocol === 'list-version' && !data)) return;
    const token = epoch.current, sameSession = sessionGuard(), controller = new AbortController();
    download.current = controller; setExporting(true); setNotice(''); setNoticeSeverity('error');
    try {
      const result = port?.protocol === 'identity-draft'
        ? await port.export(query, valid.draft!, position.sort, controller.signal)
        : await port.export(query, valid.choice, valid.version, position.sort, controller.signal);
      if (!sameSession() || token !== epoch.current || controller.signal.aborted) return;
      const blob = unwrap(result);
      const url = URL.createObjectURL(blob), link = document.createElement('a');
      link.href = url; link.download = result.response.headers.get('Content-Disposition')?.match(/filename="([^"]+)"/)?.[1] ?? (data ? port.fallbackFilename(data) : `chnhcm-selection-${valid.day}.csv`); link.click();
      setTimeout(() => URL.revokeObjectURL(url), 0); setNoticeSeverity('success'); setNotice(`已导出${summary.selectedCount}项`);
    } catch (error) {
      if (!sameSession() || token !== epoch.current || controller.signal.aborted) return;
      if (error instanceof ApiError && [401,403,409].includes(error.status) && error.code !== 'AUTH_REQUEST_VALIDATION_FAILED') { clear(); reload(); }
      failNotice(error instanceof ApiError ? error.message : '导出失败，请重试');
    } finally { if (token === epoch.current) setExporting(false); }
  };
  const visualCount = visualChoice?.mode === 'allMatching' ? (valid?.total ?? data?.count ?? 0) - visualChoice.excludedIds.length : visualChoice?.ids.length ?? 0;
  const pending = Boolean(pendingChoice || saved && (!summary || (port?.protocol === 'list-version' && !data)));
  const state = exporting ? 'exporting' : pending ? saved ? 'checking' : 'establishing' : saved ? 'ready' : 'idle';
  const checked = visualChoice?.mode === 'allMatching' ? visualChoice.excludedIds.length === 0 : Boolean(data?.count && visualCount === data.count);
  return { state, requestPage, reposition, suspend: cancel, checked, mixed: !checked && visualCount > 0, active: Boolean(saved || pendingChoice), pending, summary, selected,
    selectable: (id: string) => !valid?.draft || valid.choice.mode !== 'allMatching' || !!valid.memberIDs?.includes(id),
    setPage, all: () => { if (valid?.draft && valid.choice.mode === 'allMatching') clear(); change({ mode: 'allMatching', excludedIds: [] }); }, clear: () => { const refresh = port?.protocol === 'identity-draft' && Boolean(savedRef.current || intent.current || pageRead.current); clear(); if (refresh) reload(); }, exporting, exportSelected, notice, noticeSeverity, retry: reload };
}
