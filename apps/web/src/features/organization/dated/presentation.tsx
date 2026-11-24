import type { GridColDef } from '@mui/x-data-grid';
import type { FilterColumn } from '../BusinessUnitColumnFilter';
import { createContext, useContext, type ReactNode } from 'react';
import type { Models } from '@chnhcm/api-client';
import type { DatedObject } from './object';
import type { ApiResponse } from './useSubmission';
import type { NavigationPort, SelectionPort } from './list';

// Presentation ports only. Object adapters retain their own DTOs and command
// serialization; this layer neither computes eligibility nor business dates.
// Values are a UI draft/projection only; each adapter enumerates its real DTO fields.
export type DataFields = Record<string, string | null | undefined>;
export type ListFilterContext = { query: URLSearchParams; fields: string[] };
export type DataKey = string;
export type RecordMeta = {
  recordId: string; revision: string; deleted: boolean; name?: string | null;
  effectiveDate?: string | null; endExclusive?: string | null; endInclusive?: string | null;
  state?: string | null; operationType?: string | null; operationReason?: string | null;
  typeLabel?: string | null; reasonLabel?: string | null; operationSummary?: Models['OperationSummary'] | null;
};
export type ReferenceView = { id: string; code?: string | null; name?: string | null; status?: string; effectiveDate?: string; path?: Models['GeoAreaPathNode'][] };
export type RecordView = RecordMeta & { values: DataFields; references?: Record<string, ReferenceView | null | undefined> };
export type DetailView = Pick<Models['Detail'], 'id' | 'currentDate' | 'headSourceVersion' | 'resolvedSourceVersion' | 'readOnly' | 'selectionMissing' | 'selectedRecordIsFuture' | 'capabilities'> & {
    canReadPurposes?: boolean;
  enterprise?: Models['Enterprise']; code?: string | null; status?: string | null; nameGuidance?: string;
  selectedRecord: RecordView | null; timeline: RecordView[]; viewFact?: RecordView | null; related?: (record: RecordView | null, readFields: string[], referencesRestricted: boolean) => ReactNode; relatedReferencesRestricted?: boolean;
  basicFields?: (record: RecordView, readFields: string[], referencesRestricted: boolean) => ReactNode;
  additionContext?: Omit<Models['AdditionContext'], 'defaults' | 'mode'> & { mode: string; defaults?: Partial<RecordView> | null } | null;
  lifecycleContext?: Omit<Models['LifecycleContext'], 'defaults' | 'operationType'> & { operationType: string; defaults?: Partial<RecordView> | null } | null;
  pairInsertion?: Models['PairInsertionContext'] | null;
  pairDeletion?: { deactivation: RecordView; reactivation: RecordView; changes: { start?: string; endExclusive?: string | null; endInclusive?: string | null; before: Partial<RecordView>; after: Partial<RecordView> }[] } | null;
};
export type SummaryView = { id: string; code?: string | null; name?: string | null; status?: string | null; effectiveDate?: string | null; values?: DataFields; references?: Record<string, ReferenceView | null | undefined>; secondary?: ReactNode };
export type ListView = { items: SummaryView[]; count?: number; hasMore?: boolean; nextCursor?: string; page: number; pageSize: number; currentDate?: string; businessAt?: string; queryKey?: string; selectionVersion?: string };
export type ContextView = Omit<Models['Context'], 'timezone' | 'enterprise' | 'codeMaxLength'> & { filterCountries?: Models['GeoAreaContext']['filterCountries']; timezone: string; enterprise?: Models['Enterprise']; codeMaxLength?: number };
export type Result = {
  submissionId: string; outcome: 'accepted' | 'rejected' | 'no_change' | 'not_applied' | 'pending';
  objectId?: string | null; sourceVersion?: string | null; observedSourceVersion?: string | null; selectedRecordId?: string | null;
  targetRecordId?: string | null; targetRevision?: string | null; effectiveDate?: string | null; recordedAt?: string | null;
  action?: string; errorCode?: string | null; currentChanged: boolean; replayed: boolean;
  failureDetails?: Models['FailureDetails'] | null; items?: { recordId?: string | null; outcome: string; effectiveDate?: string | null; afterRecord?: { name?: string | null; effectiveDate?: string | null } | null }[];
};
export type Preview = Pick<Models['Preview'], 'readFields'> & Partial<Pick<Models['Preview'], 'previousCurrentStatus' | 'currentStatus' | 'newNoFactRange'>> & { afterRecord?: RecordView; impact?: Models['TimelineImpact'] | null };
export type PreviewInput = Omit<Models['PreviewInput'], 'changes' | 'operationReason'> & { changes?: DataFields; operationReason?: string };
export type CorrectInput = Omit<Models['CorrectInput'], 'changes'> & { changes?: DataFields };
export type RecordCommand =
  | { action: 'ADD'; body: Omit<Models['AddInput'], 'changes' | 'operationReason'> & { changes: DataFields; operationReason: string } }
  | { action: 'CORRECT'; recordId: string; body: CorrectInput }
  | { action: 'DELETE'; recordId: string; body: Models['DeleteInput'] }
  | { action: 'DEACTIVATE'; body: Omit<Models['DeactivateInput'], 'operationReason'> & { operationReason: string } }
  | { action: 'REACTIVATE'; body: Omit<Models['ReactivateInput'], 'changes' | 'operationReason'> & { changes?: DataFields; operationReason: string } };
export type MaintainPairInput = Models['DeleteLifecyclePairInput'] & { deactivationDate: string; reactivationDate: string; deactivationReason: string; reactivationReason: string };
export type PairCommand = { action: 'insert'; body: Models['InsertLifecyclePairInput'] } | { action: 'delete'; body: Models['DeleteLifecyclePairInput'] };
export type CreateInput = { code: string; name: string; effectiveDate: string; operationType: 'CREATE'; operationReason: string } & DataFields;
export type SelectionQuery = Omit<Models['DepartmentSelectionQuery'], 'filterField'> & { filterField?: string; countryRegionId?: string };
export type Change = Omit<Models['Change'], 'items' | 'action'> & { action: string; items: (Omit<Models['ChangeItem'], 'beforeContext' | 'afterContext' | 'changeKind'> & { changeKind: string } & { beforeContext: Partial<RecordView> | null; afterContext: Partial<RecordView> | null })[] };
export type Changes = Omit<Models['Changes'], 'items' | 'pageSize'> & { items: Change[]; pageSize: number };
export type ChangeQuery = { page: number; pageSize: 20 | 50 | 100; contextSourceVersion?: string; throughSourceVersion?: string; filterField?: 'recordedAt' | 'actor' | 'action' | 'effectiveDate' | 'changedField'; filterOperator?: 'contains' | 'equals' | 'is' | 'isAnyOf' | 'before' | 'after' | 'onOrBefore' | 'onOrAfter'; filterValue?: string };
export type Header = { 'X-CSRF-Token': string; 'Idempotency-Key': string };
export type ReferenceContext = Models['DepartmentReferenceContext'] | { kind: 'location'; input: Models['LocationReferenceInput'] } | { kind: 'legalentity'; input: Models['LegalEntityReferenceInput'] };
export type DataEditorProps = {
  session: Models['Principal']; values: DataFields; baseline?: Partial<RecordView> | null; fields: string[]; writes: string[];
  disabled: boolean; prefix: string; fieldError: string; error: string; guidance?: string;
  context?: ReferenceContext; onReadDenied?: () => void; onReferenceRestricted?: () => void; onChange: (field: DataKey, value: string) => void;
};
export type ListSidebarProps = {
 scopeId?: string; includeDescendants: boolean; day?: string; unavailable: boolean;
 onSelect: (id?: string) => void; onInclude: (value: boolean) => void;
 onLabel: (label?: string) => void; onDayMismatch: () => void;
};
export type Definition = DatedObject<string, Result> & {
  ListSidebar?: (props: ListSidebarProps) => ReactNode;
  singleton?: boolean;
  label: string; listLabel?: string; prefix: string; fields: { key: DataKey; label: string; required?: boolean }[];
  statusReadField?: string;
  logStateField?: "state" | "status";
  sortableFields?: string[];
  api: {
    context: (signal?: AbortSignal) => Promise<ApiResponse<ContextView>>;
    reasons: (type: Models['Classification']['operationType'], signal?: AbortSignal) => Promise<ApiResponse<Models['Reason'][]>>;
    list?: (query: URLSearchParams, page?: number, signal?: AbortSignal) => Promise<ApiResponse<ListView>>;
    detail: (id: string, query: { sourceVersion?: string; selectedRecordId?: string }, signal?: AbortSignal) => Promise<ApiResponse<DetailView>>;
    create?: (body: CreateInput, header: Header) => Promise<ApiResponse<Result>>;
    preview: (id: string, csrf: string, body: PreviewInput) => Promise<ApiResponse<Preview>>;
    record: (id: string, command: RecordCommand, header: Header) => Promise<ApiResponse<Result>>;
    maintainPair?: (id: string, body: MaintainPairInput, header: Header) => Promise<ApiResponse<Result>>;
    pair?: (id: string, command: PairCommand, header: Header) => Promise<ApiResponse<Result>>;
    changes: (id: string, query: ChangeQuery, signal: AbortSignal) => Promise<ApiResponse<Changes>>;
  };
  summaryFields?: { key: DataKey; label: string; render?: (record: Partial<RecordView>) => string }[];
  hiddenColumns?: string[];
  additionalColumns?: GridColDef<SummaryView>[]; filterColumns?: FilterColumn[]; summary?: (row: SummaryView) => ReactNode;
  navigation?: NavigationPort;
  selection?: SelectionPort<SelectionQuery, string, ListView>;
  referenceContext?: (id: string, input: PreviewInput) => ReferenceContext | undefined;
  createReferenceContext?: (date: string, values: DataFields) => ReferenceContext;
  Relations?: (props: DataEditorProps) => ReactNode;
  pathCursorField?: string;
  columnChoices?: (context?: ContextView) => FilterColumn[];
  ListScope?: (props: ListFilterContext) => ReactNode;
  scopedEmpty?: (query: URLSearchParams) => string | undefined;
  ListFilters?: (props: { query: URLSearchParams; fields: string[]; onChange: (values: Record<string,string>) => void }) => ReactNode;
};
const Context = createContext<Definition | null>(null);
export const ObjectDefinition = Context.Provider;
export function useDefinition() { const value = useContext(Context); if (!value) throw new Error('缺少对象页面定义'); return value; }
export function inputValues(record?: Partial<RecordView> | null): DataFields { return { ...record?.values }; }
export function recordView<T extends RecordMeta>(record: T, values: DataFields): RecordView { return { ...record, values }; }

export async function mapResponse<A, B>(request: Promise<ApiResponse<A>>, map: (a: A) => B): Promise<ApiResponse<B>> {
  const r = await request;
  return r.data !== undefined ? { ...r, data: map(r.data), error: undefined } : { ...r, data: undefined };
}
