import type { ApiResponse } from './useSubmission';
import type { Outcome, Owner, Pending, SubmissionChannel } from '../../../shared/dated/submission';

export type ResultView<K extends string> = Outcome & {
  kind: K; objectId?: string; sourceVersion?: string; selectedRecordId?: string; lines: string[];
};
export type FieldCatalog<T> = { readonly [Field in keyof T]-?: { readonly label: string } };
export type DatedObject<K extends string, R extends Outcome> = {
  label?: string; kind: K; basePath: string; submissions: SubmissionChannel<K, R>;
  resultView: (result: R) => ResultView<K>;
  resolve: (session: Owner & { csrf: string }, pending: Pending) => Promise<ApiResponse<R>>;
};
export function safeListBack(base: string, back: string | null, origin = window.location.origin) {
  try {
    const url = new URL(back ?? base, origin);
    url.searchParams.delete('businessAt');
    return url.origin === origin && url.pathname === base ? url.pathname + url.search : base;
  } catch { return base; }
}
export function selectedListBack(base: string, back: string | null, id: string, origin = window.location.origin) {
  const url = new URL(safeListBack(base, back, origin), origin);
  url.searchParams.set('selected', id); return url.pathname + url.search;
}
export function originalResultHref<K extends string>(definition: { kind: K; basePath: string }, view: ResultView<K>, back: string | null) {
  if (view.kind !== definition.kind || !view.objectId || !view.sourceVersion) return undefined;
  return `${definition.basePath}/${view.objectId}?${new URLSearchParams({ sourceVersion: view.sourceVersion,
    ...(view.selectedRecordId ? { selectedRecordId: view.selectedRecordId } : {}), back: selectedListBack(definition.basePath, back, view.objectId) })}`;
}
