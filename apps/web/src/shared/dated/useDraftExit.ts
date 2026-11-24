import { useEffect, useRef, useState } from 'react';
import { useBlocker } from 'react-router-dom';

/** Shared route, logout, browser-close and editor-cancel protection. */
export function useDraftExit(dirty: boolean, saved: () => boolean, discard: () => void) {
  const callbacks = useRef({ saved, discard }); callbacks.current = { saved, discard };
  const next = useRef<(() => void) | undefined>(undefined);
  const [asked, setAsked] = useState(false);
  const shouldBlock = () => dirty && !callbacks.current.saved();
  const blocker = useBlocker(shouldBlock);
  const request = (proceed: () => void) => { if (shouldBlock()) { next.current = proceed; setAsked(true); } else proceed(); };
  useEffect(() => {
    const unload = (event: BeforeUnloadEvent) => { if (shouldBlock()) { event.preventDefault(); event.returnValue = ''; } };
    const logout = (event: Event) => {
      if (shouldBlock()) { event.preventDefault(); next.current = (event as CustomEvent<{ proceed: () => void }>).detail.proceed; setAsked(true); }
    };
    window.addEventListener('beforeunload', unload); window.addEventListener('chnhcm-before-logout', logout);
    return () => { window.removeEventListener('beforeunload', unload); window.removeEventListener('chnhcm-before-logout', logout); };
  }, [dirty]);
  return { request, dialog: {
    open: asked || blocker.state === 'blocked',
    onContinue() { next.current = undefined; setAsked(false); if (blocker.state === 'blocked') blocker.reset(); },
    onDiscard() {
      callbacks.current.discard(); setAsked(false);
      const proceed = next.current; next.current = undefined;
      // Proceed only with this confirmed action. If it fails and the editor
      // stays mounted, subsequent drafts must still be protected.
      if (proceed) proceed(); else if (blocker.state === 'blocked') blocker.proceed();
    },
  } };
}
