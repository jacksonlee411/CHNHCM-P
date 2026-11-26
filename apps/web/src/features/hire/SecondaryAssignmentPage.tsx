import { useEffect, useRef, useState } from 'react';
import { Alert, Autocomplete, Box, Button, MenuItem, Stack, TextField, Typography } from '@mui/material';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { api, unwrap, type Models } from '@chnhcm/api-client';
import {
  ObjectDetailField,
  ObjectDetailSection,
  ObjectForm,
  ObjectTextField,
  RecordBackButton,
  RecordControl,
  UnsavedChangesDialog,
} from '@chnhcm/ui';
import { useDraftExit } from '../../shared/dated/useDraftExit';
import { usePageTitle } from '../../shared/navigation';
import { message, path, useRead } from '../personnel/api';
import { useRuleSubmission } from '../readiness/useRuleSubmission';
import { labels, type Option } from './Target';

type Input = Models['AdditionalAssignmentInput'];
type Search = Models['AdditionalAssignmentOptions'];
type Principal = Models['Principal'];
type Kind = Search['kind'];

function SecondaryTarget({
  kind,
  request,
  assignment,
  session,
  selected,
  choose,
}: {
  kind: Kind;
  request: Omit<Search, 'kind' | 'search' | 'page'>;
  assignment: string;
  session: Principal;
  selected?: Option;
  choose: (value?: Option) => void;
}) {
  const [search, setSearch] = useState('');
  const body = { ...request, kind, search, page: 1 };
  const choices = useRead(request.from ? JSON.stringify([assignment, body]) : undefined, async () =>
    unwrap(
      await api.POST('/api/v1/assignments/{id}/additional-options', {
        params: { path: { id: assignment }, header: { 'X-CSRF-Token': session.csrf } },
        body,
      }),
    ),
  );
  const required = kind !== 'bu' && kind !== 'location' && kind !== 'grade';
  return (
    <ObjectDetailField label={labels[kind]} inputId={'secondary-' + kind} required={required}>
      <Autocomplete
        id={'secondary-' + kind}
        value={selected ?? null}
        options={choices.data ?? []}
        filterOptions={(v) => v}
        size="small"
        fullWidth
        getOptionLabel={(v) => v.name}
        isOptionEqualToValue={(a, b) => a.id === b.id}
        onInputChange={(_, v, why) => {
          if (why === 'input' || why === 'clear') setSearch(v);
        }}
        onChange={(_, v) => choose(v ?? undefined)}
        noOptionsText={choices.error ?? '没有匹配记录'}
        renderInput={(props) => (
          <TextField
            {...props}
            variant="standard"
            required={required}
            error={!!choices.error}
            helperText={choices.error}
          />
        )}
      />
    </ObjectDetailField>
  );
}

export function SecondaryAssignmentPage({ session }: { session: Principal }) {
  const { id = '', assignment = '' } = useParams(),
    [params] = useSearchParams(),
    navigate = useNavigate();
  const initialDay = params.get('assignmentDay') ?? '';
  const [from, setFrom] = useState(initialDay),
    [last, setLast] = useState(''),
    [notice, setNotice] = useState('');
  const [selected, setSelected] = useState<Partial<Record<Kind, Option>>>({});
  const [mode, setMode] = useState<'existing' | 'parallel'>();
  const [error, setError] = useState('');
  const saved = useRef(false),
    errorRef = useRef<HTMLDivElement>(null);
  const header = { 'X-CSRF-Token': session.csrf };
  const view = useRead(JSON.stringify([assignment, from]), async () =>
    unwrap(
      await api.GET('/api/v1/assignments/{id}', {
        params: { path: { id: assignment }, query: { businessAt: from || undefined } },
      }),
    ),
  );
  const data = view.data?.person === id ? view.data : undefined;
  const parallel =
    mode === 'parallel' || (mode === undefined && !!data?.canAddParallel && !data.canAddSecondary);
  const available = parallel ? data?.canAddParallel : data?.canAddSecondary;
  const back = path + '/' + id + '/assignments/' + assignment + '?' + params;
  const dirty =
    !!notice || !!last || from !== initialDay || Object.values(selected).some(Boolean) || mode !== undefined;
  const exit = useDraftExit(
    dirty,
    () => saved.current,
    () => {
      saved.current = true;
    },
  );
  usePageTitle('新增兼岗');
  const command = useRuleSubmission<Input, Models['HireReceipt']>('secondary-assignment', session, {
    send: async (body, key) =>
      unwrap(
        await api.POST('/api/v1/assignments/{id}/additional-assignments', {
          params: { path: { id: assignment }, header: { ...header, 'Idempotency-Key': key } },
          body,
        }),
      ),
    resolve: async (sub) =>
      unwrap(
        await api.POST('/api/v1/hires/submissions/{sub}/resolve', { params: { path: { sub }, header } }),
      ),
    finished: (r, recovery) => {
      if (recovery === 'other') return;
      if (r.outcome !== 'accepted') {
        setError(r.outcome === 'not_applied' ? '原号已确认未执行。' : '原提交已拒绝。');
        return;
      }
      saved.current = true;
      const q = new URLSearchParams(params);
      if (from) q.set('assignmentDay', from);
      navigate(r.assignment ? path + '/' + id + '/assignments/' + r.assignment + '?' + q : back, {
        replace: true,
      });
    },
    failed: (e) => {
      setError(message(e));
      view.refresh();
    },
  });
  useEffect(() => {
    command.start();
  }, [command.start]);
  useEffect(() => {
    if (error) errorRef.current?.focus();
  }, [error]);
  const effective = from || data?.businessAt || '';
  const arrangementRequest = {from:effective, lastDay:last||undefined, parallel, employer:parallel?selected.employer?.id:undefined,position:selected.position?.id};
  const arrangement = useRead(available && effective ? JSON.stringify([assignment, arrangementRequest]) : undefined,()=>api.POST('/api/v1/assignments/{id}/additional-arrangement',{
    params:{path:{id:assignment},header},body:arrangementRequest,
  }).then(unwrap));
  const arrangementMode=arrangement.data;
  const query = {
    from: effective,
    lastDay: last || undefined,
    parallel,
    employer: parallel ? selected.employer?.id : undefined,
    department: arrangementMode?.jobRequired ? selected.department?.id : undefined,
    job: arrangementMode?.jobRequired ? selected.job?.id : arrangementMode?.job?.id,
    bu: selected.bu?.id,
    location: selected.location?.id,
  };
  const valid = !!(
    available &&
    effective &&
    arrangementMode &&
    (!arrangementMode.jobRequired || (!!selected.department && !!selected.job)) &&
    (!arrangementMode.positionRequired || !!selected.position) &&
    (!parallel || selected.employer) &&
    notice.trim()
  );
  const target = (kind: Kind) => (
    <SecondaryTarget
      key={kind}
      kind={kind}
      request={query}
      assignment={assignment}
      session={session}
      selected={selected[kind]}
      choose={(v) => setSelected({ ...selected, ...(kind==='job'||kind==='position'?{grade:undefined}:{}), [kind]: v })}
    />
  );
  function save() {
    if (!valid || !data || command.busy) return;
    setError('');
    void command.submit({
      ...query,
      expectedSourceVersion: data.sourceVersion,
      position: arrangementMode?.positionRequired ? selected.position?.id:undefined,
      department: arrangementMode?.jobRequired ? selected.department?.id:undefined,
      job: arrangementMode?.jobRequired ? selected.job?.id:undefined,
      grade: selected.grade?.id,
      notice,
    });
  }
  return (
    <Box className="standard-page object-page">
      <RecordControl
        identity={
          <>
            <RecordBackButton component={Link} to={back} label="返回任职" />
            {data?.name} {data?.number}
          </>
        }
        navigation={null}
        actions={
          available && (
            <Stack direction="row" gap={1}>
              <Button
                variant="contained"
                type="submit"
                form="secondary-form"
                disabled={!valid || command.busy}
              >
                保存
              </Button>
              <Button component={Link} to={back}>
                取消
              </Button>
            </Stack>
          )
        }
      />
      <Box className="page-scroll" data-page-scroll role="region" aria-label="新增兼岗" tabIndex={0}>
        <Typography component="h1" variant="h5">
          新增兼岗
        </Typography>
        {view.error && (
          <Alert severity="error" action={<Button onClick={view.refresh}>重试</Button>}>
            {view.error}
          </Alert>
        )}
        {data && !available && <Alert severity="info">该日无可办理的兼岗。</Alert>}
        <Box
          component="form"
          id="secondary-form"
          onSubmit={(e) => {
            e.preventDefault();
            save();
          }}
        >
          <ObjectForm>
            <ObjectDetailSection title="任职安排">
              <ObjectTextField
                type="date"
                label="开始日期"
                required
                value={effective}
                onChange={(e) => setFrom(e.target.value)}
              />
              {data && available && (
                <>
                  {data.canAddParallel && data.canAddSecondary && (
                    <ObjectTextField
                      select
                      label="所属关系"
                      value={parallel ? 'parallel' : 'existing'}
                      onChange={(e) => setMode(e.target.value as 'parallel' | 'existing')}
                    >
                      <MenuItem value="existing">当前工作关系</MenuItem>
                      <MenuItem value="parallel">新增并行工作关系</MenuItem>
                    </ObjectTextField>
                  )}
                  {parallel ? (
                    target('employer')
                  ) : (
                    <ObjectDetailField label="用人单位">{data.employerName}</ObjectDetailField>
                  )}
                  {arrangementMode?.jobRequired && <>{target('department')}{target('job')}</>}
                  {(arrangementMode?.positionRequired || selected.position) && target('position')}
                  {arrangementMode?.department && <ObjectDetailField label="岗位所属部门">{arrangementMode.department.name}</ObjectDetailField>}
                  {arrangementMode?.job && <ObjectDetailField label="岗位对应职务">{arrangementMode.job.name}</ObjectDetailField>}
                  {query.job && target('grade')}
                  {arrangement.error && <Alert severity="error">{arrangement.error}</Alert>}
                  <ObjectTextField
                    type="date"
                    label="最后任职日（含）"
                    value={last}
                    onChange={(e) => setLast(e.target.value)}
                  />
                  {target('bu')}
                  {target('location')}
                  <ObjectTextField
                    label="任用依据"
                    required
                    value={notice}
                    onChange={(e) => setNotice(e.target.value)}
                  />
                </>
              )}
            </ObjectDetailSection>
          </ObjectForm>
        </Box>
        {error && (
          <Alert severity="error" ref={errorRef} tabIndex={-1}>
            {error}
          </Alert>
        )}
        {command.pending.map((ref) => (
          <Button key={ref.submissionId} onClick={() => void command.recover(ref)}>
            查回兼岗提交
          </Button>
        ))}
        {command.storageError && <Alert severity="error">无法保存提交号，请恢复浏览器存储后办理。</Alert>}
      </Box>
      <UnsavedChangesDialog {...exit.dialog} />
    </Box>
  );
}
