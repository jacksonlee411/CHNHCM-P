import { useCallback, useEffect, useRef, useState } from 'react';
import { api, ApiError, sessionGuard, unwrap, type Models } from '@chnhcm/api-client';
import { submissionChannel } from '../../shared/dated/submission';

export const path = '/personnel/employees';
export type Principal = Models['Principal'];
export type Profile = Models['PersonDetail'];
export type Entry = Models['PersonEntry'];
export type Result = Models['PersonResult'];
export const channel = submissionChannel<'person', Result>('person', 'chnhcm-person-submissions');
const base = '/api/v1/persons';
const headers = (s: Principal) => ({ 'X-CSRF-Token': s.csrf });
export const entry = async () => unwrap(await api.GET('/api/v1/persons/context'));
export const list = async (query: DirectoryQuery) => unwrap(await api.GET(base, { params: { query } }));
export const queryRef = async (s: Principal, keyword: string) =>
  unwrap(await api.POST('/api/v1/persons/queries', { params: { header: headers(s) }, body: { keyword } }));
export type DirectoryQuery = { ref?: string; page: number; pageSize: number; sort: string };
export function parse(params: URLSearchParams): DirectoryQuery {
  return {
    ref: params.get('ref') || undefined,
    page: Number(params.get('page') || 1),
    pageSize: Number(params.get('pageSize') || 20),
    sort: params.get('sort') || 'number',
  };
}
export function directoryParams(params: URLSearchParams) {
  const result = new URLSearchParams();
  for (const key of ['ref', 'department', 'primary', 'sort', 'selected', 'page', 'pageSize']) {
    const value = params.get(key);
    if (value) result.set(key, value);
  }
  return result;
}
export function directoryQuery(params: URLSearchParams): Models['PersonnelDirectoryQuery'] & {sort:string} {
  const query = parse(params);
  const primary = params.get('primary');
  return { ...query, pageSize: query.pageSize as 20 | 50 | 100,
    department: params.get('department') || undefined,
    primary: primary === 'primary' || primary === 'secondary' ? primary : undefined };
}
export async function directory(s: Principal, params: URLSearchParams, anchor?: string) {
  return unwrap(await api.POST('/api/v1/personnel/directory/query', {
    params: { header: headers(s) }, body: { ...directoryQuery(params), anchor },
  }));
}
export function directoryTarget(params: URLSearchParams, personId: string, assignment?: string) {
  const next = directoryParams(params);
  next.set('from', 'directory');
  next.set('selected', assignment || personId);
  return path + '/' + personId + (assignment ? '/assignments/' + assignment : '') + '?' + next;
}
export const profile = async (id: string, version?: string, view?: string) =>
  version
    ? unwrap(
        await api.GET('/api/v1/persons/{id}/sources/{version}', {
          params: { path: { id, version }, query: { view: view === 'fullDocument' ? view : undefined } },
        }),
      )
    : unwrap(await api.GET('/api/v1/persons/{id}', { params: { path: { id }, query: { view } } }));
export const changes = async (
  session: Principal,
  id: string,
  body: Models['PersonChangeQuery'],
  signal: AbortSignal,
) =>
  unwrap(
    await api.POST('/api/v1/persons/{id}/changes', {
      params: { path: { id }, header: headers(session) },
      body,
      signal,
    }),
  );
export const navigation = async (id: string, query: DirectoryQuery) =>
  unwrap(await api.GET('/api/v1/persons/{id}/navigation', { params: { path: { id }, query } }));
export const register = async (s: Principal, body: Models['PersonRegister']) =>
  unwrap(await api.POST(base, { params: { header: headers(s) }, body }));
export const correct = async (s: Principal, id: string, body: Models['PersonCorrect']) =>
  unwrap(
    await api.POST('/api/v1/persons/{id}/corrections', {
      params: { path: { id }, header: headers(s) },
      body,
    }),
  );
export const renumber = async (s: Principal, id: string, body: Models['PersonRenumber']) =>
  unwrap(
    await api.POST('/api/v1/persons/{id}/renumber', { params: { path: { id }, header: headers(s) }, body }),
  );
export const resolve = async (s: Principal, submission: string) =>
  unwrap(
    await api.POST('/api/v1/persons/submissions/{submission}/resolve', {
      params: { path: { submission }, header: headers(s) },
    }),
  );
export function message(e: unknown) {
  return e instanceof Error ? e.message : '读取失败，请重试';
}
export function useRead<T>(
  key: string | undefined,
  read: () => Promise<T>,
  mode: 'retain' | 'fresh' = 'retain',
) {
  const [epoch, setEpoch] = useState(0);
  const refresh = useCallback(() => setEpoch((i) => i + 1), []);
  const identity = JSON.stringify([key, epoch]);
  const resultKey = mode === 'fresh' ? identity : key;
  const [result, set] = useState<{ key: string; data?: T; error?: string; status?: number }>();
  const generation = useRef(0);
  const invalidate = useCallback(() => {
    generation.current++;
    set(undefined);
    refresh();
  }, [refresh]);
  useEffect(() => {
    if (key === undefined || resultKey === undefined) {
      set(undefined);
      return;
    }
    let active = true;
    const current = sessionGuard(),
      issued = generation.current;
    read()
      .then((data) => {
        if (active && issued === generation.current && current()) set({ key: resultKey, data });
      })
      .catch((e) => {
        if (active && issued === generation.current && current())
          set({ key: resultKey, error: message(e), status: e instanceof ApiError ? e.status : undefined });
      });
    return () => {
      active = false;
    };
  }, [identity, resultKey]);
  useEffect(() => {
    const visible = () => {
      if (document.visibilityState === 'visible') refresh();
    };
    window.addEventListener('focus', refresh);
    window.addEventListener('pageshow', refresh);
    document.addEventListener('visibilitychange', visible);
    return () => {
      window.removeEventListener('focus', refresh);
      window.removeEventListener('pageshow', refresh);
      document.removeEventListener('visibilitychange', visible);
    };
  }, [refresh]);
  return { ...(key !== undefined && result?.key === resultKey ? result : {}), refresh, invalidate };
}

export const differences = async (id: string, version: string, signal?: AbortSignal) =>
  unwrap(
    await api.GET('/api/v1/persons/{id}/changes/{version}', { params: { path: { id, version } }, signal }),
  );
