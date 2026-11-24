import { useState } from 'react';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { Alert, Box, Button, Stack, Typography } from '@mui/material';
import { api, unwrap, type Models } from '@chnhcm/api-client';
import { ObjectForm, RecordBackButton, RecordControl } from '@chnhcm/ui';
import { usePageTitle } from '../../shared/navigation';
import * as person from '../personnel/api';
import { EmployeeHire, employment } from './HirePage';
import { actionLabel, relationAction } from './AssignmentRecordForm';
import { JobInformation, OriginalEvidence } from './JobInformation';
import { registerPath } from './EmploymentRegister';

export function RegisterActionPage({
  session,
  cancel = false,
}: {
  session: Models['Principal'];
  cancel?: boolean;
}) {
  const { actionId = '' } = useParams(),
    [params] = useSearchParams(),
    navigate = useNavigate();
  const original = person.useRead(actionId, async () =>
    unwrap(await api.GET('/api/v1/employment-actions/{id}', { params: { path: { id: actionId } } })),
  );
  const a = original.data?.action;
  const visibleAssignment = person.useRead(
    a ? JSON.stringify([a.assignment, a.businessDate, a.revision]) : undefined,
    async () =>
      unwrap(
        await api.GET('/api/v1/assignments/{id}', {
          params: { path: { id: a!.assignment }, query: { businessAt: a!.businessDate } },
        }),
      ),
  );
  const [revision, setRevision] = useState('');
  const archived = person.useRead(revision || undefined, async () =>
    unwrap(
      await api.GET('/api/v1/employment-actions/{id}', {
        params: { path: { id: actionId }, query: { adoptedRef: revision } },
      }),
    ),
  );
  const overview = person.useRead(a?.person, () => employment(a!.person));
  const current = overview.data?.actions.find((v) => v.id === actionId);
  const related = overview.data?.actions.find((v) => v.original === actionId && v.canHistory);
  const [reader, setReader] = useState('');
  const canReturn = person.useRead(session.subjectId, async () =>
    unwrap(await api.GET('/api/v1/employment-actions/context')),
  );
  const query = new URLSearchParams(params);
  if (!query.has('selected')) query.set('selected', actionId);
  const details = registerPath + '/' + actionId + '?' + query;
  const assignmentQuery = new URLSearchParams(query);
  assignmentQuery.set('returnAction', actionId);
  if (a) assignmentQuery.set('assignmentDay', a.businessDate);
  const entered = reader || actionId;
  usePageTitle(cancel ? '撤销未来入职' : '办理明细');
  if (cancel && a && current?.canCancel)
    return (
      <EmployeeHire
        id={a.person}
        session={session}
        cancellation={{
          action: current,
          returnTo: details,
          done: () => navigate(details, { replace: true }),
        }}
      />
    );
  return (
    <Box className="standard-page object-page">
      <RecordControl
        navigation={null}
        identity={
          <>
            {canReturn.data?.canList && (
              <RecordBackButton component={Link} to={registerPath + '?' + query} label="返回办理记录" />
            )}
            办理记录
          </>
        }
        actions={
          a && (
            <Stack direction="row" gap={1}>
              {current?.canCancel && (
                <Button component={Link} to={registerPath + '/' + actionId + '/cancel?' + query}>
                  撤销未来入职
                </Button>
              )}
              {visibleAssignment.data && (
                <Button
                  component={Link}
                  to={person.path + '/' + a.person + '/assignments/' + a.assignment + '?' + assignmentQuery}
                >
                  查看任职
                </Button>
              )}
            </Stack>
          )
        }
      />
      <Box className="page-scroll" role="region" aria-label="办理明细" tabIndex={0} data-page-scroll>
        {original.error && (
          <Alert severity="error" action={<Button onClick={original.refresh}>重试</Button>}>
            {original.error}
          </Alert>
        )}
        {a && (
          <>
            <Typography component="h1" variant="h5">
              {actionLabel(a)} ·{' '}
              {relationAction(a.kind)
                ? '工作关系'
                : a.recordNumber + ' 号' + (a.primaryJob ? '主岗' : '兼岗')}
            </Typography>
            {cancel && overview.data && !current?.canCancel && (
              <Alert severity="info">当前不可撤销这笔办理。</Alert>
            )}
            {related && (
              <Button onClick={() => setReader(reader ? '' : related.id)}>
                {reader ? '返回入职原件' : '查看撤销依据'}
              </Button>
            )}
            <ObjectForm>
              {visibleAssignment.data && (
                <JobInformation
                  key={entered}
                  session={session}
                  assignment={a.assignment}
                  day={a.businessDate}
                  selectedAction={entered}
                  onDay={() => {}}
                  actionSlot={null}
                  onSave={() => {
                    original.refresh();
                    overview.refresh();
                  }}
                />
              )}
              {(visibleAssignment.status === 403 || visibleAssignment.status === 404) &&
                (revision ? archived.data : original.data) && (
                  <OriginalEvidence
                    original={(revision ? archived.data : original.data)!}
                    setRevision={setRevision}
                  />
                )}
              {archived.error && <Alert severity="error">{archived.error}</Alert>}
            </ObjectForm>
          </>
        )}
      </Box>
    </Box>
  );
}
