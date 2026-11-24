import { useEffect, useRef, useState, type ReactNode } from 'react';
import { Box, Checkbox, Drawer, FormControlLabel, IconButton, TextField, Tooltip, Typography, useMediaQuery } from '@mui/material';
import ChevronRight from '@mui/icons-material/ChevronRight';
import ExpandMore from '@mui/icons-material/ExpandMore';
import AccountTree from '@mui/icons-material/AccountTree';
import ChevronLeft from '@mui/icons-material/ChevronLeft';

// Presentation only: the department adapter supplies authorized visible rows,
// expansion state and loading actions. This component never derives membership.
export type DepartmentTreeRow = {
  id: string; name: string; code: string; depth: number;
  action?: boolean; context: boolean; hasChildren: boolean; expanded: boolean; loading?: boolean;
};
type Props = {
  reveal?: {id: string; sequence: number}; message?: ReactNode; searching?: boolean; rows: DepartmentTreeRow[]; selectedId?: string; includeDescendants: boolean;
  search: string; onSearch: (value: string) => void;
  onSelect: (id?: string) => void; onExpand: (id: string) => void;
  onIncludeDescendants: (value: boolean) => void;
};
const rowHeight = 28;
let returnView = { collapsed: false, offset: 0 };
export function DepartmentTreePanel(props: Props) {
  const narrow = useMediaQuery('(max-width:899px)');
  const [open, setOpen] = useState(false), [collapsed, setCollapsed] = useState(returnView.collapsed);
  const [offset, setOffset] = useState(returnView.offset);
  const [focusId, setFocusId] = useState<string>();
  const scroll = useRef<HTMLDivElement>(null);
  const view = useRef({collapsed, offset}); view.current = {collapsed, offset};
  useEffect(() => {
    const invalidate = () => { returnView = {collapsed: false, offset: 0}; view.current = returnView; setCollapsed(false); setOffset(0); };
    window.addEventListener('chnhcm-session-invalidated', invalidate);
    return () => { returnView = view.current; window.removeEventListener('chnhcm-session-invalidated', invalidate); };
  }, []);
  useEffect(() => { if (scroll.current && offset && !scroll.current.scrollTop) scroll.current.scrollTop = offset; }, [collapsed, open]);
  const trigger = useRef<HTMLButtonElement>(null);
  const close = () => { setOpen(false); requestAnimationFrame(() => trigger.current?.focus()); };
  const select = (id?: string) => { props.onSelect(id); if (narrow) close(); };
  // Render a bounded row window, including overscan. Keep focus inside that
  // window before moving DOM focus; long paths need no deep nested DOM.
  const start = Math.max(0, Math.floor(offset / rowHeight) - 10);
  const visible = props.rows.slice(start, start + 200);
  const contentWidth = props.rows.reduce((width, row) => Math.max(width, row.depth * 16 + 80), 80);
  const move = (index: number) => {
    const row = props.rows[Math.max(0, Math.min(props.rows.length - 1, index))];
    if (!row) return;
    setFocusId(row.id);
    const top = Math.max(0, index) * rowHeight;
    if (scroll.current && (top < scroll.current.scrollTop || top + rowHeight > scroll.current.scrollTop + scroll.current.clientHeight)) { scroll.current.scrollTop = top; setOffset(scroll.current.scrollTop); }
    if (scroll.current) scroll.current.scrollLeft = Math.max(0, row.depth * 16 + 80 - scroll.current.clientWidth);
    requestAnimationFrame(() => scroll.current?.querySelector<HTMLElement>(`[data-tree-id="${CSS.escape(row.id)}"]`)?.focus());
  };
  useEffect(() => { if (scroll.current) scroll.current.scrollTop = 0; setOffset(0); setFocusId(undefined); }, [props.search]);
  const revealed = useRef<number | undefined>(undefined);
  useEffect(() => {
    if (!props.reveal || props.searching || revealed.current === props.reveal.sequence) return;
    const index = props.rows.findIndex(row => row.id === props.reveal?.id);
    if (index < 0 || !scroll.current) return;
    scroll.current.scrollTop = index * rowHeight; scroll.current.scrollLeft = Math.max(0, props.rows[index].depth * 16 + 80 - scroll.current.clientWidth); setOffset(scroll.current.scrollTop);
    setFocusId(props.reveal.id); revealed.current = props.reveal.sequence;
  }, [props.reveal, props.rows, props.searching, open, collapsed]);
  const contents = <Box className="department-tree-panel" sx={{ display: 'flex', flexDirection: 'column', height: '100%', minHeight: 0, flexShrink: 0, width: narrow ? 260 : 220, maxWidth: 'calc(100vw - 48px)', py: 1, pl: 2, pr: narrow ? 2 : 0, boxSizing: 'border-box', bgcolor: 'transparent' }}>
    <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
      <Typography fontWeight={600} fontSize={14}>部门</Typography>
      <IconButton size="small" aria-label={narrow ? '关闭部门树' : '折叠部门树'} onClick={() => narrow ? close() : setCollapsed(true)}><ChevronLeft fontSize="small" /></IconButton>
    </Box>
    <TextField size="small" label="定位树中部门" value={props.search} onChange={e => props.onSearch(e.target.value)} sx={{ my: 1 }} />
    {props.message}
    <Box component="button" type="button" onClick={() => select()} aria-pressed={!props.selectedId} sx={{ border: 0, borderRadius: '6px', minHeight: 28, textAlign: 'left', background: !props.selectedId ? '#e6f2f3' : 'transparent', color: !props.selectedId ? '#066b68' : 'inherit', font: 'inherit', fontSize: 14, cursor: 'pointer' }}>全部</Box>
    <Box ref={scroll} role={props.searching ? "listbox" : "tree"} aria-label={props.searching ? "部门定位结果" : "部门树"} onScroll={e => setOffset(e.currentTarget.scrollTop)} sx={{ flex: 1, minHeight: 0, overflow: 'auto', mt: .5 }}>
      <Box sx={{ height: props.rows.length * rowHeight, width: contentWidth, minWidth: '100%', position: 'relative' }}>
        {visible.map((row, i) => <Box key={row.id} role={props.searching ? "option" : "treeitem"} data-tree-id={row.id} aria-label={row.action ? row.name : undefined} aria-level={props.searching ? undefined : row.depth + 1} aria-disabled={row.context || undefined} aria-selected={!row.context && row.id === props.selectedId} aria-expanded={!row.context && row.hasChildren ? row.expanded : undefined} aria-busy={row.loading || undefined}
          tabIndex={row.id === (visible.some(item => item.id === focusId) ? focusId : visible[0]?.id) ? 0 : -1}
          onFocus={() => setFocusId(row.id)}
          onClick={() => { if (!row.context) select(row.id); }}
          onKeyDown={e => {
            const index = start + i;
            if (e.key === 'ArrowDown' || e.key === 'ArrowUp') { e.preventDefault(); move(index + (e.key === 'ArrowDown' ? 1 : -1)); }
            else if (!row.context && row.hasChildren && ((e.key === 'ArrowRight' && !row.expanded) || (e.key === 'ArrowLeft' && row.expanded))) { e.preventDefault(); props.onExpand(row.id); }
            else if (e.key === 'Enter' && !row.context) { e.preventDefault(); select(row.id); }
          }}
          sx={{ position: 'absolute', top: (start + i) * rowHeight, height: rowHeight, left: row.depth * 16, right: 0, minWidth: 80, display: 'flex', alignItems: 'center', borderLeft: row.depth ? '1px solid #e0e6e8' : undefined, borderRadius: '6px', bgcolor: row.id === props.selectedId ? '#e6f2f3' : undefined, color: row.id === props.selectedId ? '#066b68' : row.context ? 'text.secondary' : 'inherit', '&:hover': { bgcolor: row.id === props.selectedId ? '#e6f2f3' : 'rgba(0, 0, 0, .04)' }, '&:focus-visible': { outline: '2px solid #066b68', outlineOffset: -2 } }}>
          <Box sx={{ width: 24, flexShrink: 0 }}>{!row.context && row.hasChildren && <IconButton tabIndex={-1} size="small" disabled={row.loading} aria-label={`${row.expanded ? '收起' : '展开'}${row.name}`} onClick={e => { e.stopPropagation(); props.onExpand(row.id); }} sx={{ p: .5 }}>{row.expanded ? <ExpandMore sx={{ fontSize: 16 }} /> : <ChevronRight sx={{ fontSize: 16 }} />}</IconButton>}</Box>
          <Tooltip disableInteractive title={`${row.name}（${row.code}）${row.context ? ' · 不可选择' : ''}`}>
            <Box component="span" sx={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontSize: 14, lineHeight: '20px', fontWeight: row.id === props.selectedId ? 600 : 400, cursor: row.context ? 'default' : 'pointer' }}>{row.name}{props.searching && row.code ? `（${row.code}）` : ''}</Box>
          </Tooltip>
        </Box>)}
      </Box>
    </Box>
    <FormControlLabel sx={{ m: 0 }} control={<Checkbox size="small" checked={props.includeDescendants} disabled={!props.selectedId} onChange={(_, checked) => props.onIncludeDescendants(checked)} />} label={<Typography fontSize={14}>含子部门</Typography>} />
  </Box>;
  if (narrow) return <><Tooltip title="部门树"><IconButton ref={trigger} sx={{alignSelf:'flex-start',flexShrink:0}} aria-label="打开部门树" onClick={() => setOpen(true)}><AccountTree /></IconButton></Tooltip><Drawer open={open} onClose={close} slotProps={{paper: {sx: {bgcolor: '#f8fafb'}}}}>{contents}</Drawer></>;
  if (collapsed) return <Box sx={{ width: 32, flexShrink: 0 }}><Tooltip title="展开部门树"><IconButton size="small" aria-label="展开部门树" onClick={() => setCollapsed(false)}><AccountTree fontSize="small" /></IconButton></Tooltip></Box>;
  return contents;
}
