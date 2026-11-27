import { useEffect, useMemo, useRef, useState } from 'react';
import { Alert, Box, Button, CircularProgress } from '@mui/material';
import { api, ApiError, sessionGuard, unwrap, type Models } from '@chnhcm/api-client';
import { DepartmentTreePanel, type DepartmentTreeRow } from './DepartmentTreePanel';
import type { ListSidebarProps } from './dated/presentation';

type Node = Models['DepartmentTreeNode'];
type Page = Models['DepartmentTreePage'];
type Window = Page & { touched: number };
const ROOT = '';
// Return state contains identities only. Labels and permissions are always read
// again. Session invalidation discards this bounded, memory-only state.
let returnExpanded: string[] = [];
export function DepartmentTree(props: ListSidebarProps) {
  const latest = useRef(props); latest.current = props;
  const [windows, setWindows] = useState<Map<string, Window>>(new Map());
  const windowsRef = useRef(windows); windowsRef.current = windows;
  const [chain, setChain] = useState<Node[]>([]);
  const chainRef = useRef(chain); chainRef.current = chain;
  const relocated = useRef(false);
  const [reveal, setReveal] = useState<{id: string; sequence: number}>();
  const [expanded, setExpanded] = useState<Set<string>>(() => new Set(returnExpanded));
  const [pathClosed, setPathClosed] = useState<Set<string>>(new Set());
  const expandedRef = useRef(expanded); expandedRef.current = expanded;
  const [search, setSearch] = useState('');
  const [matches, setMatches] = useState<Page>();
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState<Set<string>>(new Set());
  const epoch = useRef(0), sequence = useRef(0), alive = useRef(true), dayBlocked = useRef(false);
  const requests = useRef(new Map<string, AbortController>());
  const parents = useRef(new Map<string, {parent: string | null; observed: number}>());
  const pruneParents = () => {
    const kept = new Set([...chainRef.current.map(n => n.id), ...[...windowsRef.current.values()].flatMap(w => w.items.map(n => n.id))]);
    for (const id of parents.current.keys()) if (!kept.has(id)) parents.current.delete(id);
  };
  const observe = (items: Node[], issued: number) => {
    if (items.some(n => { const known = parents.current.get(n.id); return known && known.observed > issued && known.parent !== n.parentId; })) return false;
    const next = new Map(windowsRef.current);
    for (const node of items) {
      const known = parents.current.get(node.id);
      if (known && known.parent !== node.parentId) { const key = known.parent ?? ROOT; next.delete(key); requests.current.get(`nodes:${key}`)?.abort(); }
      parents.current.set(node.id, { parent: node.parentId, observed: ++sequence.current });
    }
    windowsRef.current = next; setWindows(next);
    return true;
  };
  const active = useRef(0), queue = useRef<Array<() => void>>([]);
  const acquire = async (signal: AbortSignal) => {
    signal.throwIfAborted();
    if (active.current < 2) { active.current++; return; }
    await new Promise<void>((resolve, reject) => {
      const start = () => { signal.removeEventListener('abort', abort); if (signal.aborted) reject(new DOMException('已取消', 'AbortError')); else { active.current++; resolve(); } };
      const abort = () => { const index = queue.current.indexOf(start); if (index >= 0) queue.current.splice(index, 1); reject(new DOMException('已取消', 'AbortError')); };
      queue.current.push(start); signal.addEventListener('abort', abort, { once: true });
    });

  };
  const reset = () => {
    epoch.current++; requests.current.forEach(c => c.abort()); requests.current.clear();
    parents.current.clear(); windowsRef.current = new Map(); setWindows(new Map()); chainRef.current = []; setChain([]); setMatches(undefined); setErrors({}); setLoading(new Set()); latest.current.onLabel(undefined);
  };
  const read = async <T extends { businessAt: string }>(key: string, fetcher: (signal: AbortSignal) => Promise<T>, accept: (value: T, issued: number) => void, unavailable?: () => void) => {
    if (dayBlocked.current || !alive.current || !latest.current.day) return;
    requests.current.get(key)?.abort();
    const controller = new AbortController(), issued = ++sequence.current, generation = epoch.current, sameSession = sessionGuard(); requests.current.set(key, controller);
    setLoading(old => new Set(old).add(key)); setErrors(old => { const next = { ...old }; delete next[key]; return next; });
    let acquired = false;
    try {
      await acquire(controller.signal); acquired = true; controller.signal.throwIfAborted();
      const value = await fetcher(controller.signal);
      if (controller.signal.aborted || !alive.current || generation !== epoch.current || !sameSession()) return;
      if (latest.current.day && value.businessAt !== latest.current.day) { reset(); dayBlocked.current = true; latest.current.onDayMismatch(); return; }
      accept(value, issued);
    } catch (error) {
      if (controller.signal.aborted || !alive.current || generation !== epoch.current || !sameSession()) return;
      if (error instanceof ApiError && error.code === 'M02_TREE_SCOPE_UNAVAILABLE' && unavailable) { unavailable(); return; }
      if (error instanceof ApiError && ([401, 403].includes(error.status) || error.code === 'M02_TREE_SCOPE_UNAVAILABLE')) { reset(); setErrors({ location: error.message }); return; }
      setErrors(old => ({ ...old, [key]: error instanceof ApiError ? error.message : '部门树读取失败，请重试' }));
    } finally {
      if (acquired) { active.current--; while(active.current < 2 && queue.current.length) queue.current.shift()!(); }
      if (requests.current.get(key) === controller) { requests.current.delete(key); pruneParents(); if (alive.current) setLoading(old => { const next = new Set(old); next.delete(key); return next; }); }
    }
  };
  const store = (parent: string, page: Page, direction?: 'previous' | 'next') => {
    const next = new Map(windowsRef.current), previous = next.get(parent);
    let movedChain = page.items.some(row => chainRef.current.some(n => n.id === row.id && n.parentId !== row.parentId));
    for (const [key, window] of next) if (key !== parent && window.items.some(item => page.items.some(row => row.id === item.id && row.parentId !== item.parentId))) {
      next.delete(key); requests.current.get(`nodes:${key}`)?.abort();
      movedChain ||= window.items.some(item => chainRef.current.some(n => n.id === item.id));
    }
    const byID = new Map<string, Node>();
    const ordered = !direction ? page.items : direction === 'previous' ? [...page.items, ...(previous?.items ?? [])] : [...(previous?.items ?? []), ...page.items];
    ordered.forEach(n => byID.set(n.id, n));
    next.set(parent, { ...page, items: [...byID.values()], touched: ++sequence.current,
      hasPrevious: direction === 'next' && previous ? previous.hasPrevious : page.hasPrevious,
      previousCursor: direction === 'next' && previous ? previous.previousCursor : page.previousCursor,
      hasMore: direction === 'previous' && previous ? previous.hasMore : page.hasMore,
      nextCursor: direction === 'previous' && previous ? previous.nextCursor : page.nextCursor });
    const evicted = new Set<string>();
    while (next.size > 40 || [...next.values()].reduce((sum, w) => sum + w.items.length, 0) > 2000) {
      const candidate = [...next.entries()].filter(([key]) => key !== parent).sort((a,b) => a[1].touched-b[1].touched)[0];
      if (!candidate) { next.set(parent, { ...page, touched: sequence.current }); break; }
      next.delete(candidate[0]); evicted.add(candidate[0]);
    }
    windowsRef.current = next; setWindows(next);
    if (evicted.size) setExpanded(old => new Set([...old].filter(id => !evicted.has(id))));
    if (movedChain) { setChain([]); latest.current.onLabel(undefined); relocate(); }
  };
  const relocate = () => {
    const id = latest.current.scopeId;
    if (id && !relocated.current) { relocated.current = true; reset(); locate(id); nodes(); }
    else { reset(); setErrors({ location: '部门位置已变化，请刷新' }); }
  };
  const nodes = (parent = ROOT, cursor?: string, aroundId?: string) => void read(`nodes:${parent}`,
    async signal => unwrap(await api.GET('/api/v1/departments/tree/nodes', { params: { query: { parentId: parent || undefined, cursor, aroundId, pageSize: 50 } }, signal })),
    (page, issued) => { if (!observe(page.items, issued)) { setErrors(old => ({ ...old, [`nodes:${parent}`]: '部门位置已变化，请刷新' })); return; } store(parent, page, cursor ? cursor === windowsRef.current.get(parent)?.previousCursor ? 'previous' : 'next' : undefined); }, aroundId ? relocate : undefined);
  const locate = (id: string, select = false) => void read('location', async signal => unwrap(await api.GET('/api/v1/departments/{id}/tree-location', { params: { path: { id } }, signal })), (location, issued) => {
    if (!observe(location.items, issued)) { relocate(); return; }
    chainRef.current = location.items; setChain(location.items); setPathClosed(new Set()); setReveal({id, sequence: ++sequence.current}); setSearch(''); setMatches(undefined);
    const parent = location.items.at(-2), root = location.items[0];
    if (parent?.kind === 'node') nodes(parent.id, undefined, id);
    if (root?.kind === 'node' && !windowsRef.current.get(ROOT)?.items.some(n => n.id === root.id)) nodes(ROOT, undefined, root.id);
    const target = location.items.at(-1); latest.current.onLabel(target?.name);
    if (select) latest.current.onSelect(id);
  });
  useEffect(() => {
    alive.current = true;
    const invalidate = () => { returnExpanded = []; expandedRef.current = new Set(); setExpanded(new Set()); setSearch(''); reset(); };
    window.addEventListener('chnhcm-session-invalidated', invalidate);
    return () => { alive.current = false; returnExpanded = [...expandedRef.current].slice(-2000); epoch.current++; requests.current.forEach(c => c.abort()); window.removeEventListener('chnhcm-session-invalidated', invalidate); };
  }, []);
  useEffect(() => { dayBlocked.current = false; reset(); if (props.day) nodes(); }, [props.day]);
  useEffect(() => { requests.current.get('location')?.abort(); relocated.current = false; latest.current.onLabel(undefined); if (props.unavailable) { reset(); if (props.day) nodes(); } if (props.scopeId && props.day && !props.unavailable) { const target = chainRef.current.at(-1); if (target?.id === props.scopeId) latest.current.onLabel(target.name); else locate(props.scopeId); } else setChain([]); }, [props.scopeId, props.day, props.unavailable]);
  useEffect(() => {
    requests.current.get('search')?.abort(); setMatches(undefined);
    if (!props.day || !search.trim()) return;
    requests.current.get('location')?.abort();
    const timer = setTimeout(() => void read('search', async signal => unwrap(await api.GET('/api/v1/departments/tree/search', { params: { query: { q: search, pageSize: 20 } }, signal })), page => setMatches(page)), 200);
    return () => clearTimeout(timer);
  }, [search, props.day]);
  const actions = useRef(new Map<string, () => void>());
  const rows = useMemo(() => {
    actions.current = new Map(); const result: DepartmentTreeRow[] = [];
    const action = (key: string, name: string, depth: number, run: () => void) => { actions.current.set(key, run); result.push({ id: key, name, code: '', depth, context: false, hasChildren: false, expanded: false, action: true }); };
    if (search.trim()) {
      const searchPage = (cursor?: string) => void read('search', async signal => unwrap(await api.GET('/api/v1/departments/tree/search', { params: { query: { q: search, cursor, pageSize: 20 } }, signal })), page => setMatches(page));
      if (matches?.hasPrevious) action('search:previous', '加载较早定位结果', 0, () => searchPage(matches.previousCursor));
      matches?.items.forEach(n => result.push({ ...n, depth: 0, context: false, hasChildren: false, expanded: false }));
      if (matches?.hasMore) action('search:next', '加载更多定位结果', 0, () => searchPage(matches.nextCursor));
      return result;
    }
    const chainNodes = new Map(chain.map(n => [n.id, n]));
    const pathNext = new Map<string, Node>(); chain.forEach((n,i) => { if(i) pathNext.set(chain[i-1].id,n); });
    const rootItems = [...(windows.get(ROOT)?.items ?? [])]; if (chain[0] && !rootItems.some(n => n.id === chain[0].id)) rootItems.push(chain[0]);
    const seen = new Set<string>();
    const stack: Array<{ node?: Node; parent?: string; depth: number }> = [{parent:ROOT,depth:0}];
    while (stack.length) {
      const item = stack.pop()!;
      if (item.parent?.startsWith('@end:')) {
        const parent=item.parent.slice(5), win=windows.get(parent), depth=item.depth;
        if(errors[`nodes:${parent}`]) action(`retry:${parent}`, '读取失败，重试', depth, () => nodes(parent));
        else if(win?.hasMore) action(`next:${parent}`, '加载更多部门', depth, () => nodes(parent,win.nextCursor));
        continue;
      }
      if (item.parent !== undefined) {
        const parent=item.parent, win=windows.get(parent), depth=item.depth;
        if (parent === ROOT && !win && !loading.has('nodes:') && !errors['nodes:']) action('load:root', '加载根部门', depth, () => nodes());
        const children=parent===ROOT ? rootItems : [...(win?.items ?? [])]; const path=pathNext.get(parent);
        if (path && !children.some(n=>n.id===path.id)) children.push(path);
        if (path && !win && chainNodes.get(parent)?.kind === 'node' && !loading.has(`nodes:${parent}`) && !errors[`nodes:${parent}`]) action(`load:${parent}`, '加载本层部门', depth, () => nodes(parent, undefined, path.id));
        if (win?.hasPrevious) action(`previous:${parent}`, '加载较早部门', depth, () => nodes(parent,win.previousCursor));
        // A marker follows the children so pagination stays at its own level.
        stack.push({parent:`@end:${parent}`,depth});
        for(let i=children.length-1;i>=0;i--) stack.push({node:children[i],depth});
        continue;
      }
      const node=item.node!; if(seen.has(node.id)) continue; seen.add(node.id);
      const open=expanded.has(node.id)||(pathNext.has(node.id) && (node.kind === 'context' || !pathClosed.has(node.id)));
      result.push({...node,depth:item.depth,context:node.kind==='context',hasChildren:node.hasChildren===true,expanded:open,loading:loading.has(`nodes:${node.id}`)});
      if(open) stack.push({parent:node.id,depth:item.depth+1});
    }
    return result;
  }, [windows, chain, expanded, pathClosed, loading, search, matches, errors]);
  // Rehydrate only the explicitly expanded branches exposed by loaded rows.
  useEffect(() => { for (const row of rows) if (!row.context && !row.action && row.expanded && expanded.has(row.id) && !windows.has(row.id) && !loading.has(`nodes:${row.id}`) && !errors[`nodes:${row.id}`]) nodes(row.id, undefined, chain[chain.findIndex(n => n.id === row.id) + 1]?.parentId === row.id ? chain[chain.findIndex(n => n.id === row.id) + 1]?.id : undefined); }, [windows, expanded, chain]);
  const topError = errors['nodes:'] || errors.location || errors.search;
  return <DepartmentTreePanel reveal={reveal} rows={rows} search={search} onSearch={value => { requests.current.get('search')?.abort(); requests.current.get('location')?.abort(); setSearch(value); }} searching={Boolean(search.trim())} selectedId={props.scopeId} includeDescendants={props.includeDescendants}
    message={<>{topError && <Alert severity="error" action={<Button size="small" onClick={() => { reset(); if (search.trim()) setSearch(search+' '); else if (props.scopeId) locate(props.scopeId); else nodes(); }}>重试</Button>}>{topError}</Alert>}{loading.has('nodes:') || loading.has('location') || loading.has('search') ? <CircularProgress size={16} aria-label="正在读取部门树" /> : search.trim() && matches && !matches.items.length ? <Box role="status">没有匹配部门</Box> : null}</>}
    onIncludeDescendants={props.onInclude} onSelect={id => { if(id && actions.current.has(id)) {actions.current.get(id)!();return;} if(id && search.trim()) locate(id,true); else { requests.current.get('location')?.abort(); if (!id) { setSearch(''); setMatches(undefined); setChain([]); } props.onSelect(id); } }}
    onExpand={id => {
      const index = chain.findIndex(n => n.id === id), onPath = index >= 0 && index < chain.length-1;
      const opening = !(expanded.has(id) || onPath && !pathClosed.has(id));
      setExpanded(old => { const next=new Set(old); if(opening)next.add(id);else next.delete(id);return new Set([...next].slice(-2000)); });
      if(onPath) setPathClosed(old => {const next=new Set(old);if(opening)next.delete(id);else next.add(id);return next;});
      if(opening)nodes(id, undefined, onPath ? chain[index+1].id : undefined);
    }} />;
}
