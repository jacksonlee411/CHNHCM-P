import { createContext, forwardRef, useContext, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { Alert, Autocomplete, Box, Breadcrumbs, Button, CircularProgress, Paper, TextField, Typography, type PaperProps } from '@mui/material';
import { ChevronRight } from '@mui/icons-material';
import { api, ApiError, sessionGuard, unwrap, type Models } from '@chnhcm/api-client';

type Candidate = Models['GeoAreaCandidate'];
type Path = Candidate['path'];
type Panel = {
  country: string; path: Path; searching: boolean; loading: boolean; more: boolean; error?: string;
  panelRef: React.RefObject<HTMLDivElement | null>;
  browse: (depth: number) => void; loadMore: () => void; retry: () => void; close: () => void;
};
const PanelContext = createContext<Panel | null>(null);

// A stable paper component keeps the listbox, focus and scroll position while
// candidate pages arrive. Navigation lives inside the popup, beside its list.
const RegionPaper = forwardRef<HTMLDivElement, PaperProps>(function RegionPaper(props, ref) {
  const panel = useContext(PanelContext)!;
  return <Paper {...props} ref={ref}>
    <Box ref={panel.panelRef} onKeyDown={event => { if (event.key === 'Escape') { event.stopPropagation(); panel.close(); } }}>
      <Box sx={{ p: 1, borderBottom: 1, borderColor: 'divider' }} onMouseDown={event => event.preventDefault()}>
        {panel.searching ? <Typography variant="body2">在{panel.country}内搜索</Typography> :
          <Breadcrumbs aria-label="地区选择路径" sx={{ '& .MuiBreadcrumbs-ol': { flexWrap: 'wrap' }, '& .MuiBreadcrumbs-li': { minWidth: 0 }, '& button': { minWidth: 0, overflowWrap: 'anywhere' } }}>
            <Button size="small" onClick={() => panel.browse(0)}>{panel.country}</Button>
            {panel.path.map((node, index) => <Button key={node.id} size="small" onClick={() => panel.browse(index + 1)}>{node.name}</Button>)}
          </Breadcrumbs>}
        {(panel.searching || panel.path.length > 0) && <Button size="small" onClick={() => panel.browse(panel.searching ? panel.path.length : panel.path.length - 1)}>{panel.searching ? '返回逐级选择' : '返回上一级'}</Button>}
      </Box>
      {props.children}
      {panel.error && <Alert severity="error" sx={{ m: 1 }} action={<Button onMouseDown={event => event.preventDefault()} onClick={panel.retry}>重试</Button>}>{panel.error}</Alert>}
      {panel.more && <Box sx={{ p: 1 }} onMouseDown={event => event.preventDefault()}><Button fullWidth size="small" disabled={panel.loading} onClick={panel.loadMore}>{panel.loading ? '正在加载…' : '加载更多地区'}</Button></Box>}
    </Box>
  </Paper>;
});

export function LocationRegionPicker(props: {
  id: string; identity: string; countryId: string; countryName: string;
  adoptedDate?: string; areaId: string; label: string; path?: Path;
  disabled: boolean; error?: string; onSelect: (candidate: Candidate | null) => void;
  onDenied: () => void;
}) {
  const [open, setOpen] = useState(false), [search, setSearch] = useState(''), [path, setPath] = useState<Path>([]);
  const [retry, setRetry] = useState(0), [next, setNext] = useState<{ key: string; cursor: string }>();
  const [page, setPage] = useState<{ key: string; items: Candidate[]; more: boolean; cursor?: string; error?: string; loading: boolean }>();
  const panelRef = useRef<HTMLDivElement>(null), inputRef = useRef<HTMLInputElement>(null);
  const lookup = useRef<AbortController | null>(null);
  const q = search.trim(), parentId = q ? undefined : path.at(-1)?.id;
  const key = JSON.stringify([props.identity, props.countryId, props.adoptedDate, parentId, q, retry]);
  const cursor = next?.key === key ? next.cursor : undefined;
  const current = useRef(key); current.current = key;
  const available = page?.key === key ? page : undefined;
  const selected: Candidate | null = props.areaId ? {
    id: props.areaId, name: props.label, code: '', level: '', terminal: true, path: props.path ?? [],
  } : null;
  // The parent identity covers action/date/record/country changes. Do not carry
  // an open candidate list or search text into a different editing context.
  useEffect(() => { setOpen(false); setSearch(''); setPath([]); setPage(undefined); setNext(undefined); }, [props.identity, props.countryId]);
  useEffect(() => { if (props.disabled) setOpen(false); }, [props.disabled]);
  useLayoutEffect(() => () => lookup.current?.abort(), [props.identity, props.countryId, props.adoptedDate, props.disabled]);
  useEffect(() => {
    if (!open || props.disabled || !props.adoptedDate) return;
    lookup.current?.abort();
    const abort = new AbortController(), same = sessionGuard();
    lookup.current = abort;
    let visible = true;
    setPage(previous => ({ key, items: cursor && previous?.key === key ? previous.items : [], more: cursor && previous?.key === key ? previous.more : false, loading: true }));
    const timer = setTimeout(() => {
      void api.GET('/api/v1/geo-areas/candidates', {
        params: { query: { countryRegionId: props.countryId, adoptedDate: props.adoptedDate!, parentAreaId: parentId, q: q || undefined, cursor, pageSize: 50 } }, signal: abort.signal,
      }).then(unwrap).then(result => {
        if (!visible || abort.signal.aborted || !same() || current.current !== key) return;
        setPage(previous => {
          const items = cursor && previous?.key === key ? [...previous.items] : [];
          const ids = new Set(items.map(item => item.id));
          for (const item of result.items) if (!ids.has(item.id)) { items.push(item); ids.add(item.id); }
          return { key, items, more: result.hasMore, cursor: result.cursor ?? undefined, loading: false };
        });
      }).catch(error => {
        if (abort.signal.aborted || !same()) return;
        const denied = error instanceof ApiError && error.status === 403;
        // A definitive denial also invalidates the saved label after the popup closes.
        if (denied) { setPath([]); props.onDenied(); }
        if (!visible || current.current !== key) return;
        if (cursor && error instanceof ApiError && error.status === 409 && error.code === 'M02_CURSOR_CHANGED') {
          setPage(undefined);
          setNext(undefined);
          setRetry(previous => previous + 1);
          return;
        }
        setPage(previous => ({ key, items: !denied && cursor && previous?.key === key ? previous.items : [], more: false, error: error instanceof Error ? error.message : '地区读取失败', loading: false }));
      });
    }, q ? 180 : 0);
    return () => { clearTimeout(timer); visible = false; };
  }, [open, key, cursor, props.disabled]);

  const focus = () => inputRef.current?.focus();
  const close = () => { setOpen(false); setSearch(''); };
  const browse = (depth: number) => { setPath(previous => previous.slice(0, depth)); setSearch(''); setNext(undefined); focus(); };
  const loadMore = () => { if (available?.more && available.cursor && !available.loading) setNext({ key, cursor: available.cursor }); };
  const label = (candidate: Candidate) => candidate.id === props.areaId ? props.label : candidate.path.map(node => node.name).join(' / ') || candidate.name;
  const panel: Panel = { country: props.countryName, path, searching: !!q, panelRef, browse, close: () => { close(); focus(); },
    loading: available?.loading ?? true, more: available?.more ?? false, error: available?.error, loadMore, retry: () => setRetry(previous => previous + 1) };

  return <PanelContext.Provider value={panel}><Autocomplete
    id={props.id} open={open} disableCloseOnSelect clearOnBlur={false} selectOnFocus handleHomeEndKeys
    disabled={props.disabled} options={available?.items ?? []} value={selected}
    inputValue={open ? search : props.label} filterOptions={options => options}
    isOptionEqualToValue={(a, b) => a.id === b.id} getOptionLabel={label}
    onOpen={() => { setPath((props.path ?? []).filter(node => node.id !== props.countryId && node.id !== props.areaId)); setNext(undefined); setOpen(true); }}
    onClose={(_, reason) => { if (reason === 'blur' && panelRef.current?.contains(document.activeElement)) return; close(); }}
    onInputChange={(_, value, reason) => { if (reason === 'input') { setSearch(value); setNext(undefined); setOpen(true); } }}
    onChange={(event, candidate, reason) => {
      // Emptying a search is not an instruction to discard the saved address.
      // The clear button remains an explicit draft change.
      if (reason === 'clear' && event.type !== 'click') return;
      if (!candidate || candidate.terminal) { props.onSelect(candidate); close(); }
      else { setPath(candidate.path.filter(node => node.id !== props.countryId)); setSearch(''); setNext(undefined); }
    }}
    loading={!available || available.loading} loadingText="正在加载地区…"
    noOptionsText={available?.error ? '地区暂时无法读取' : q ? '没有匹配地区，请调整搜索词' : '没有可选地区'}
    slots={{ paper: RegionPaper }}
    slotProps={{ listbox: { sx: { maxHeight: 'min(320px, 40vh)', '& li': { overflowWrap: 'anywhere' } }, onScroll: event => {
      const list = event.currentTarget;
      if (list.scrollHeight - list.scrollTop - list.clientHeight < 48) loadMore();
    } } }}
    renderOption={(attributes, candidate) => {
      const { key: optionKey, ...rest } = attributes;
      return <li key={optionKey} {...rest} aria-label={candidate.path.map(node => node.name).join(' / ')}>
        <Box sx={{ flex: 1, minWidth: 0 }}>{q ? candidate.path.map(node => node.name).join(' / ') : candidate.name}</Box>
        {!candidate.terminal && <ChevronRight fontSize="small" aria-hidden="true" sx={{ ml: 1, flexShrink: 0 }} />}
      </li>;
    }}
    renderInput={params => <TextField {...params} inputRef={inputRef} label="行政区划" error={!!props.error}
      placeholder="请选择行政区划"
      helperText={props.error || '可逐级选择，或输入地区名称、组合地名或编码搜索。'}
      slotProps={{ input: { ...params.InputProps, endAdornment: <>{available?.loading && open && <CircularProgress size={16} />}{params.InputProps.endAdornment}</> } }} />}
  /></PanelContext.Provider>;
}
