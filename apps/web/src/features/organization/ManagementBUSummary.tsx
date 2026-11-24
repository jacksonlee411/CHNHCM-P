import { useState } from 'react';
import { api, unwrap } from '@chnhcm/api-client';
import { useOwnedRead } from './dated/useOwnedRead';

export function ManagementBUSummary({ id }: { id: string }) {
  const [label, setLabel] = useState('正在读取名称…');
  useOwnedRead(`management-bu-summary:${id}`, {
    start: () => setLabel('正在读取名称…'),
    request: async signal => {
      const session = unwrap(await api.GET('/api/v1/session', { signal }));
      return api.POST('/api/v1/departments/references/business-units', {
        params: { header: { 'X-CSRF-Token': session.csrf } }, signal,
        body: { context: { mode: 'filter' }, selectedId: id, q: id, page: 1, pageSize: 20 },
      });
    },
    accept: data => {
      const reference = data.selectedReference;
      setLabel(reference?.id === id ? [reference.code, reference.name].filter(Boolean).join(' · ') || '名称无读取权限' : '无法查看该业务单元');
    },
    reject: () => setLabel('名称暂时无法加载'),
  });
  return <>管理业务单元：{label}</>;
}
