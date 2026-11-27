import { useCallback, useEffect, useRef, useState } from 'react';
import { ApiError, sessionGuard } from '@chnhcm/api-client';

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
