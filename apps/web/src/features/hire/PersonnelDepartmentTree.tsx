import { useMemo, useState } from 'react';
import { Typography } from '@mui/material';
import type { Models } from '@chnhcm/api-client';
import { DepartmentTreePanel, type DepartmentTreeRow } from '../../shared/DepartmentTreePanel';

type Department = Models['PersonnelDirectory']['departments'][number];
export function PersonnelDepartmentTree({
  nodes,
  selected,
  directOnly,
  onScope,
}: {
  nodes: Department[];
  selected?: string;
  directOnly?: boolean;
  onScope: (id?: string, directOnly?: boolean) => void;
}) {
  const [search, setSearch] = useState('');
  const [closed, setClosed] = useState(new Set<string>());
  const term = search.trim();
  const rows = useMemo(() => {
    const children = new Map<string, Department[]>();
    for (const node of nodes) {
      const parent = node.parent ?? '';
      const siblings = children.get(parent) ?? [];
      siblings.push(node);
      children.set(parent, siblings);
    }
    const rows: DepartmentTreeRow[] = [];
    if (term)
      return nodes
        .filter((node) => node.name.includes(term))
        .map((node) => ({
          ...node,
          code: '',
          depth: 0,
          hasChildren: false,
          expanded: false,
          context: false,
          action: true,
        }));
    const visit = (parent: string, depth: number) => {
      for (const node of children.get(parent) ?? []) {
        const expanded = !closed.has(node.id);
        rows.push({
          ...node,
          code: '',
          depth,
          hasChildren: children.has(node.id),
          expanded,
          context: false,
          action: true,
        });
        if (expanded) visit(node.id, depth + 1);
      }
    };
    visit('', 0);
    return rows;
  }, [nodes, term, closed]);
  return (
    <DepartmentTreePanel
      viewKey="personnel"
      rows={rows}
      search={search}
      onSearch={setSearch}
      searching={!!term}
      selectedId={selected}
      includeDescendants={!directOnly}
      message={
        term &&
        !rows.length && (
          <Typography variant="body2" role="status">
            没有匹配的部门
          </Typography>
        )
      }
      onSelect={(id) => {
        if (id && term) {
          const parents = new Map(nodes.map((node) => [node.id, node.parent]));
          setClosed((previous) => {
            const next = new Set(previous);
            for (let parent = parents.get(id); parent; parent = parents.get(parent)) next.delete(parent);
            return next;
          });
        }
        onScope(id, id ? directOnly : false);
        setSearch('');
      }}
      onIncludeDescendants={(include) => onScope(selected, !include)}
      onExpand={(id) =>
        setClosed((previous) => {
          const next = new Set(previous);
          if (next.has(id)) next.delete(id);
          else next.add(id);
          return next;
        })
      }
    />
  );
}
