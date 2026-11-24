export interface NavigationLeaf { label: string; path: string; group?: string; placeholder?: boolean }
export interface NavigationApp {
  id: string;
  label: string;
  /** An application with a single destination opens directly, without sections. */
  path?: string;
  sections: { label: string; direct?: boolean; items: NavigationLeaf[] }[];
}
