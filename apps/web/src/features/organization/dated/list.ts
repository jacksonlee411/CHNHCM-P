import type { ApiResponse } from './useSubmission';

export type ListPage = { items: { id: string }[]; count?: number; hasMore?: boolean; nextCursor?: string; page: number; pageSize: number; queryKey?: string; businessAt?: string };
export type Selection = { mode: 'explicit'; ids: string[] } | { mode: 'allMatching'; excludedIds: string[] };
export type SelectionSummary = { queryKey: string; selectionVersion: string; totalCount: number; selectedCount: number; canExport: boolean };
export type SelectionPosition<S extends string> = { page: number; pageSize: number; sort: S; cursor?: string };
export type SelectionPage<P extends ListPage> = { page: P; summary: SelectionSummary };
export type DraftReference = { draftId: string; revision: number };
export type DraftEdit = { addIds?: string[]; removeIds?: string[]; restoreIds?: string[] };
export type DraftSummary = DraftReference & { queryKey: string; selectedCount: number; canExport: boolean };
export type DraftPage<P extends ListPage> = { pageState: 'available'; businessAt: string; page: P; summary: DraftSummary; selectedIds: string[]; memberIds: string[] } | { pageState: 'scope_unavailable'; businessAt: string; summary: DraftSummary };
type CommonPort<Q, S extends string, P extends ListPage> = {
  fallbackFilename: (page: P) => string;
  toolbarSelector: string;
  returnFocusSelector: string;
};
export type SelectionPort<Q, S extends string, P extends ListPage = ListPage> = CommonPort<Q, S, P> & (
  | { protocol: 'list-version'; version: (page: P) => string;
      export: (query: Q, choice: Selection, version: string, sort: S, signal: AbortSignal) => Promise<ApiResponse<Blob>>;
      resolve: (query: Q, choice: Selection, version: string, signal: AbortSignal) => Promise<ApiResponse<SelectionSummary>> }
  | { protocol: 'identity-draft';
      start: (query: Q, choice: Selection, position: SelectionPosition<S>, signal: AbortSignal) => Promise<ApiResponse<DraftPage<P>>>;
      resolvePage: (query: Q, draft: DraftReference, edit: DraftEdit, position: SelectionPosition<S>, signal: AbortSignal) => Promise<ApiResponse<DraftPage<P>>>;
      cancel: (draft: DraftReference) => Promise<unknown>;
      export: (query: Q, draft: DraftReference, sort: S, signal: AbortSignal) => Promise<ApiResponse<Blob>> }
);

export type NavigationDirection = 'previous' | 'next';
export type NavigationResult = { targetId: string | null; previousId?: string; nextId?: string; position?: number; count?: number };
export type NavigationPort = (query: URLSearchParams, anchorId: string, direction: NavigationDirection | 'current', signal: AbortSignal) => Promise<ApiResponse<NavigationResult>>;
