import { useEffect, useMemo } from 'react';

// Visited page positions only. Tokens are not authorization; every read is
// authenticated and the server binds each token to its session and business day.
export function usePathCursors(base: string, params: URLSearchParams) {
  const query = new URLSearchParams(params);
  for (const name of ['page', 'cursor', 'selected']) query.delete(name);
  query.set('pageSize', query.get('pageSize') ?? '20');
  query.sort();
  const key = 'department-page-positions:' + base + '?' + query;
  const pages = useMemo<Record<number, string>>(() => {
    try { return { 1: '', ...JSON.parse(sessionStorage.getItem(key) ?? '{}') }; }
    catch { return { 1: '' }; }
  }, [key]);
  useEffect(() => {
    const clear = () => {
      for (const name of Object.keys(sessionStorage)) if (name.startsWith('department-page-positions:')) sessionStorage.removeItem(name);
      for (const page of Object.keys(pages)) delete pages[Number(page)];
      pages[1] = '';
    };
    window.addEventListener('chnhcm-session-invalidated', clear);
    return () => window.removeEventListener('chnhcm-session-invalidated', clear);
  }, [pages]);
  const remember = (page: number, cursor: string) => {
    pages[page] = cursor;
    try { sessionStorage.setItem(key, JSON.stringify(pages)); } catch { /* A full browser store must not block current pagination. */ }
  };
  return { pages, remember, reset: () => { for (const page of Object.keys(pages)) delete pages[Number(page)]; remember(1, ''); } };
}
