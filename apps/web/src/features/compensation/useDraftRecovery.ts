import { useEffect, useRef } from 'react';

/** Carry a rejected form only across the authorized reload of that same form. */
export function useDraftRecovery<T>(identity: string, ready: boolean, failed: boolean) {
  const pending = useRef<{ identity: string; input: T } | undefined>(undefined);
  const input = ready && pending.current?.identity === identity ? pending.current.input : undefined;
  useEffect(() => {
    if (ready || failed || pending.current?.identity !== identity) pending.current = undefined;
  }, [identity, ready, failed]);
  return {
    input,
    remember: (input: T) => {
      pending.current = { identity, input };
    },
  };
}
