import { useState } from 'react';
import { Navigate } from 'react-router-dom';
import { Alert, Button } from '@mui/material';
import { api, type Models } from '@chnhcm/api-client';
import { ObjectDefinition } from './dated/presentation';
import { useOwnedRead } from './dated/useOwnedRead';
import { ObjectDetail } from './DatedPages';
import { SubmissionRecovery } from './dated/SubmissionRecovery';
import { enterprise, ENTERPRISE_PATH } from './enterpriseAdapter';
export { ENTERPRISE_PATH } from './enterpriseAdapter';
export function EnterpriseEntry() {
  const [id, setId] = useState(''), [error, setError] = useState(''), [retry, setRetry] = useState(0);
  useOwnedRead(`enterprise-entry/${retry}`, {
    request: signal => api.GET('/api/v1/enterprise/context', { signal }),
    start: () => { setId(''); setError(''); }, accept: value => setId(value.id),
    reject: error => setError(error instanceof Error ? error.message : '暂时无法加载企业'),
  });
  if (id) return <Navigate replace to={`${ENTERPRISE_PATH}/${id}`} />;
  return <Alert severity={error ? 'error' : 'info'} action={error && <Button onClick={() => setRetry(n => n + 1)}>重试</Button>}>{error || '正在加载企业…'}</Alert>;
}
export function EnterpriseDetail({ session }: { session: Models['Principal'] }) { return <ObjectDefinition value={enterprise}><ObjectDetail session={session} /></ObjectDefinition>; }
export function EnterpriseRecovery({ session }: { session: Models['Principal'] }) { return <SubmissionRecovery definition={enterprise} session={session} />; }
