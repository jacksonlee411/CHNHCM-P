import { createContext } from 'react';

export const ListNavigation = createContext<{
  href: (id: string) => string;
  rowBack: (id: string) => string;
  rememberScroll: (id: string, childLink?: boolean) => void;
} | null>(null);
