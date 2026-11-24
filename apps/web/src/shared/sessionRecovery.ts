import { createContext, useContext } from 'react';

export const SessionRecoveryContext = createContext(false);
export const useSessionRecovery = () => useContext(SessionRecoveryContext);
