import { useEffect, useRef } from 'react';
import { sessionGuard, unwrap } from '@chnhcm/api-client';
import type { ApiResponse } from './useSubmission';

/** Each query key owns its response only while mounted in the same session. */
export function useOwnedRead<T>(key: string, callbacks: {
  request: (signal: AbortSignal) => Promise<ApiResponse<T>>; start: () => void;
  accept: (value: T) => void; reject: (error: unknown) => void;
}) {
  const latest = useRef(callbacks); latest.current = callbacks;
  useEffect(() => {
    let active = true; const controller = new AbortController();
    const invalidate = () => { active = false; controller.abort(); };
    window.addEventListener('chnhcm-session-invalidated', invalidate);
    const same = sessionGuard(), current = latest.current;
    current.start();
    void (async () => {
      try { const response = await current.request(controller.signal); if (active && same()) current.accept(unwrap(response)); }
      catch (error) { if (active && same()) current.reject(error); }
    })();
    return () => { invalidate(); window.removeEventListener('chnhcm-session-invalidated', invalidate); };
  }, [key]);
}
