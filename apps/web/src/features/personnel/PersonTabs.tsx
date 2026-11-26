import { useLayoutEffect, useRef } from 'react';
import { Tab, Tabs } from '@mui/material';
import { useLocation, useNavigate, useNavigationType } from 'react-router-dom';

export type PersonTab = 'profile' | 'job' | 'history';
const labels: Record<PersonTab, string> = { profile: '个人资料', job: '职务信息', history: '变更日志' };

export function PersonTabs({
  selected,
  targets,
}: {
  selected: PersonTab;
  targets: Partial<Record<PersonTab, string>>;
}) {
  const navigate = useNavigate(),
    route = useLocation(),
    navigation = useNavigationType(),
    focused = useRef('');
  useLayoutEffect(() => {
    if (navigation !== 'PUSH' || route.state?.focusPersonTab !== selected || focused.current === route.key) return;
    const target = document.getElementById('person-' + selected + '-tab');
    if (target) {
      target.focus({ preventScroll: true });
      focused.current = route.key;
    }
  }, [navigation, selected, route.key, route.state]);
  return (
    <Tabs
      aria-label="人员详情"
      value={selected}
      variant="scrollable"
      allowScrollButtonsMobile
      onChange={(_, tab: PersonTab) => {
        if (targets[tab]) navigate(targets[tab]!, { state: { focusPersonTab: tab } });
      }}
    >
      {(['profile', 'job', 'history'] as const)
        .filter((tab) => targets[tab])
        .map((tab) => (
          <Tab
            key={tab}
            value={tab}
            id={'person-' + tab + '-tab'}
            aria-controls={'person-' + tab + '-panel'}
            label={labels[tab]}
          />
        ))}
    </Tabs>
  );
}
