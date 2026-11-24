import { useState, type ReactNode } from 'react';
import { Box } from '@mui/material';
import { focusObjectField } from './ObjectForm';

/** Editing tabs belong to one draft. Only view navigation changes the route. */
export function useObjectTabs(routeTab: string, editing: boolean, navigate: (tab: string) => void, retainedTab = routeTab) {
  const [draftTab, setDraftTab] = useState(retainedTab);
  const tab = editing ? draftTab : routeTab;
  const select = (value: string) => editing ? setDraftTab(value) : navigate(value);
  const fieldTab = (id: string) => document.getElementById(id)?.closest<HTMLElement>('[data-object-tab]')?.dataset.objectTab;
  const focusField = (id: string, fallbackId: string) => {
    const targetTab = fieldTab(id);
    if (targetTab) select(targetTab);
    focusObjectField(id, fallbackId);
  };
  return { tab, select, start: setDraftTab, fieldTab, focusField };
}

/** Keep panel contents mounted so local disclosure/input state survives tab changes. */
export function ObjectTabPanel({ prefix, value, selected, children }: { prefix: string; value: string; selected: string; children: ReactNode }) {
  return <Box role="tabpanel" id={`${prefix}-${value}-panel`} aria-labelledby={`${prefix}-${value}-tab`} data-object-tab={value} hidden={value !== selected}>
    {children}
  </Box>;
}
