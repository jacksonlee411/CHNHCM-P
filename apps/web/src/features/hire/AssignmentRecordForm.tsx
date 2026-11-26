import { useId, useState } from 'react';
import { createPortal } from 'react-dom';
import { Autocomplete, Box, Button, MenuItem, Stack, TextField, Typography } from '@mui/material';
import { api, unwrap, type Models } from '@chnhcm/api-client';
import { ObjectDetailField, ObjectTextField } from '@chnhcm/ui';
import { useRead } from '../personnel/api';
import { ActualDutyFields } from './ActualDutyFields';
import {SuccessionFields} from './SuccessionFields';
import { ExchangeFields } from './ExchangeFields';
import { LifecycleFields } from './LifecycleFields';
import { EarlierAction, PromotionFields } from './PromotionFields';
import { ProbationFields } from './ProbationFields';
import type { Option } from './Target';

type Action = Models['EmploymentAction'];
export type JobDraft = {
  mode: 'add' | 'insert' | 'correct' | 'redate' | 'withdraw';
  kind: 'ActualDuty' | 'PositionExchange' | 'DataChange' | 'Transfer' | 'ProbationStart' | 'ConfirmEmployment' | 'GradeRecord' | 'Promotion' | 'PromotionTransfer' | 'SetManager' | 'EndAssignment' | 'Leave';
  purpose?: 'direct' | 'functional';
  manager?: Option;
  contracts?:Models['ContractRef'][];
  originalDuty?: Models['DutyEvidence'];
  observation?: Models['DutyEvidence'];
  paper?: Partial<Models['DutyObservation']>;
  handover?: boolean;
  incumbent?: Models['SuccessionOption'];
  peer?: Models['PositionExchangeOption'];
  exchange?: Models['PositionExchangeLink'];
  endReporting?: boolean;
  managerLastDay?: string;
  meaning?: 'last-active' | 'first-inactive';
  grade?: Models['AssignmentGrade'];
  originalGrade?: Models['AssignmentGrade'];
  successor?: Action;
  reason?: string;
  withdrawal?: string;
  basis?: Action;
  transferNotice?: string;
  occurrence?: Option;
  predecessor?: Pick<Action, 'id' | 'revision'>;
  action?: Action;
  day: string;
  source: string;
  value?: Models['EmploymentLocationRef'];
  position?: Option;
  department?: Option;
  job?: Option;
  notice: string;
  requires?: Models['EmploymentRequiredFact'];
};

const names = {
  ActualDuty:'补录实际履职', PositionExchange: '同日对调', Hire: '入职', SetManager: '实际汇报', EndAssignment:'结束任职', Leave:'离职', Rehire:'再聘',
  Promotion: '晋升',
  PromotionTransfer: '晋升并调动',
  GradeRecord: '补录实际职级',
  CancelHire: '撤销入职',
  DataChange: '数据更改',
  Transfer: '调动',
  AdditionalAssignment: '新增兼岗',
  ParallelEmployment: '并行关系聘任',
  ProbationStart: '试用开始',
  ConfirmEmployment: '转正',
};
export const relationAction = (kind: string) => kind === 'ProbationStart' || kind === 'ConfirmEmployment' || kind==='Leave';
export function availableRecordTypes(view: Models['AssignmentView']): JobDraft['kind'][] {
  const result: JobDraft['kind'][] = [];
  if (view.canChangeLocation) result.push('DataChange');
  if (view.canTransfer) result.push('Transfer');
  if (view.canTransfer && view.state?.position) result.push('PositionExchange');
  if (view.canRecordDuty) result.push('ActualDuty');
  if (view.canStartProbation) result.push('ProbationStart');
  if (view.canConfirm) result.push('ConfirmEmployment');
  if (view.canRecordGrade) result.push('GradeRecord');
  if (view.canPromote) result.push('Promotion');
  if (view.canPromote && view.canTransfer) result.push('PromotionTransfer');
  if (view.canSetManager) result.push('SetManager');
  if (view.canEnd) result.push('EndAssignment');
  if (view.canLeave) result.push('Leave');
  return result;
}
export const actionLabel = (action: Action) => names[action.kind as keyof typeof names] ?? action.kind;
const operations = { add: '新增记录', insert: '插入记录', correct: '更正记录', redate: '调整生效日期', withdraw:'撤销晋升' };

export function draftChanged(draft: JobDraft, initial: JobDraft) {
  return (
    JSON.stringify(draft.contracts)!==JSON.stringify(initial.contracts) ||
    draft.observation?.version !== initial.observation?.version || JSON.stringify(draft.paper)!==JSON.stringify(initial.paper) ||
    draft.day !== initial.day ||
    draft.handover !== initial.handover || draft.incumbent?.assignment!==initial.incumbent?.assignment || draft.kind !== initial.kind || draft.peer?.assignment!==initial.peer?.assignment || draft.manager?.id!==initial.manager?.id ||
    draft.purpose!==initial.purpose || draft.endReporting!==initial.endReporting ||
    draft.meaning!==initial.meaning || draft.managerLastDay!==initial.managerLastDay ||
    draft.grade?.id !== initial.grade?.id ||
    draft.reason !== initial.reason ||
    draft.withdrawal !== initial.withdrawal || draft.basis?.id!==initial.basis?.id ||
    draft.transferNotice !== initial.transferNotice ||
    draft.notice !== initial.notice ||
    draft.occurrence?.id !== initial.occurrence?.id ||
    draft.predecessor?.id !== initial.predecessor?.id || draft.predecessor?.revision !== initial.predecessor?.revision ||
    draft.value?.id !== initial.value?.id ||
    draft.position?.id !== initial.position?.id ||
    draft.department?.id !== initial.department?.id ||
    draft.job?.id !== initial.job?.id ||
    draft.requires?.action !== initial.requires?.action
  );
}

export function TransferTarget({
  kind, mixed,
  draft,
  assignment,
  csrf,
  change,
}: {
  kind: 'department' | 'job' | 'position';
  mixed?: boolean;
  draft: JobDraft;
  assignment: string;
  csrf: string;
  change: (value?: Option) => void;
}) {
  const [search, setSearch] = useState('');
  const query = {
    kind,
    businessDate: draft.day,
    department: kind === 'job' || mixed ? draft.department?.id : undefined,
    job: mixed ? draft.job?.id : undefined,
    search,
    page: 1,
  };
  const candidates = useRead(draft.day ? JSON.stringify([assignment, query]) : undefined, async () =>
    unwrap(
      await api.POST('/api/v1/assignments/{id}/transfer-options', {
        params: { path: { id: assignment }, header: { 'X-CSRF-Token': csrf } },
        body: query,
      }),
    ),
  );
  const label = kind === 'department' ? '调入部门' : kind === 'position' ? '调入岗位' : '拟任职务';
  return (
    <ObjectDetailField label={label} inputId={'transfer-' + kind} required>
      <Autocomplete
        id={'transfer-' + kind}
        fullWidth
        size="small"
        options={candidates.data ?? []}
        value={draft[kind] ?? null}
        filterOptions={(v) => v}
        getOptionLabel={(v) => v.name}
        isOptionEqualToValue={(a, b) => a.id === b.id}
        onInputChange={(_, text, reason) => {
          if (reason === 'input' || reason === 'clear') setSearch(text);
        }}
        onChange={(_, v) => change(v ?? undefined)}
        noOptionsText={candidates.error ?? '没有匹配记录'}
        renderInput={(props) => (
          <TextField
            {...props}
            variant="standard"
            required
            error={!!candidates.error}
            helperText={candidates.error}
          />
        )}
      />
    </ObjectDetailField>
  );
}

export function AssignmentRecordForm({
  draft,
  update,
  view,
  csrf,
  busy,
  dirty,
  close,
  save,
  actionSlot,
}: {
  draft: JobDraft;
  update: (draft: JobDraft) => void;
  view: Models['AssignmentView'];
  csrf: string;
  busy: boolean;
  dirty: boolean;
  close: () => void;
  save: (input: JobDraft) => void;
  actionSlot: HTMLElement | null;
}) {
  const formId = useId();
  const [search, setSearch] = useState('');
  const leaveImpact = useRead(draft.kind === 'Leave' && draft.day ? JSON.stringify([view.assignment, draft.day, draft.meaning]) : undefined,
    () => api.GET('/api/v1/assignments/{id}/leave-impact', {params:{path:{id:view.assignment},query:{date:draft.day,meaning:draft.meaning ?? 'last-active'}}}).then(unwrap));
  const requiredContracts = leaveImpact.data?.contracts ?? [];
  const contractsConfirmed = !!leaveImpact.data?.canManageContracts && requiredContracts.length === draft.contracts?.length &&
    requiredContracts.every(contract => draft.contracts?.some(ref => ref.id === contract.id && ref.version === contract.version));
  const location = draft.kind === 'DataChange' && draft.mode !== 'redate';
  const choices = useRead(
    location && draft.day ? JSON.stringify([view.assignment, draft.day, search]) : undefined,
    async () =>
      unwrap(
        await api.POST('/api/v1/assignments/{id}/location-options', {
          params: { path: { id: view.assignment }, header: { 'X-CSRF-Token': csrf } },
          body: { businessDate: draft.day, search, page: 1 },
        }),
      ),
  );
  const candidates = view.actions.filter(
    (a) =>
      a.canHistory &&
      a.operation !== 'Delete' &&
      (a.kind === 'DataChange' || a.kind === 'Transfer') &&
      a.businessDate < draft.day &&
      a.id !== draft.action?.id,
  );
  const prior = draft.requires;
  const basis = useRead(
    prior && draft.mode !== 'redate' ? JSON.stringify([prior.action, prior.revision]) : undefined,
    async () =>
      unwrap(
        await api.GET('/api/v1/employment-actions/{id}', {
          params: { path: { id: prior!.action }, query: { adoptedRef: prior!.revision } },
        }),
      ),
  );
  const relation = relationAction(draft.kind);
  const lifecycle = ['Leave','EndAssignment','SetManager'].includes(draft.kind);
  const grade = ['GradeRecord','Promotion','PromotionTransfer'].includes(draft.kind);
  const transfer = draft.kind === 'Transfer' && draft.mode !== 'redate';
  const readMode = (position?: string) => api.POST('/api/v1/assignments/{id}/transfer-arrangement', {
    params: {path: {id: view.assignment}, header: {'X-CSRF-Token': csrf}},
    body: {businessDate: draft.day, position},
  }).then(unwrap);
  const promotionMove = (draft.kind === 'Promotion' || draft.kind === 'PromotionTransfer') && draft.mode !== 'withdraw' && view.canTransfer;
  const mode = useRead((transfer || promotionMove) && draft.day ? JSON.stringify([view.assignment, draft.day]) : undefined, () => readMode());
  const targets = mode.data;
  const derived = useRead((transfer || promotionMove) && !draft.handover && targets?.positionRequired && !targets.jobRequired && draft.position ?
    JSON.stringify([view.assignment,draft.day,draft.position.id]) : undefined, () => readMode(draft.position?.id));
  const targetReady = !!targets && (!targets.positionRequired || !!draft.position) &&
    (!targets.jobRequired || !!draft.department && !!draft.job) &&
    (draft.handover ? !!draft.incumbent : targets.jobRequired || !!derived.data?.department && !!derived.data.job);
  const options = availableRecordTypes(view);
  const duty=draft.kind==='ActualDuty';
  const proof=draft.paper;
  const actualReady=!!draft.observation || !!(proof?.dispatch && proof.from && proof.observedThrough && proof.issuer?.trim() && proof.number?.trim());
  const valid =
    !!draft.day && (draft.kind !== 'Leave' || !!leaveImpact.data && (!requiredContracts.length || contractsConfirmed)) &&
    (duty ? actualReady : draft.kind==='PositionExchange' ? (draft.mode==='redate' ? !!draft.exchange?.peerSourceVersion : !!draft.peer && !!draft.notice.trim()) : lifecycle ? (draft.kind === 'SetManager' ? !!draft.manager || !!draft.endReporting : !!draft.notice.trim()) : draft.mode === 'withdraw' ? !!draft.basis || !!draft.withdrawal?.trim() : draft.mode === 'redate' ||
      ((grade ? !!draft.grade && !!draft.notice.trim() && (draft.kind!=='PromotionTransfer'||targetReady) : relation
        ? !!draft.notice.trim()
        : location
          ? !!draft.value
          : targetReady && !!draft.notice.trim()) &&
        (!prior || !!basis.data)));
  function submit() {
    if (!valid || !dirty || busy) return;
    if (draft.kind === 'Leave') {
      save({...draft, contracts: contractsConfirmed ? draft.contracts : undefined});
      return;
    }
    const record = basis.data;
    let requires = prior;
    if (prior && record && !prior.location && !prior.department && !prior.job && !prior.position) {
      requires = {
        ...prior,
        location: record.locationChange?.after.id,
        department: record.transfer?.after.department,
        job: record.transfer?.after.job,
        position: record.transfer?.after.position,
      };
    }
    if ((transfer || promotionMove && (draft.kind==='PromotionTransfer' || draft.position?.id !== view.state?.position)) && targets) {
      save({...draft, requires, position: targets.positionRequired ? draft.position : undefined,
        department: targets.jobRequired ? draft.department : undefined, job: targets.jobRequired ? draft.job : undefined});
    } else save({ ...draft, requires });
  }
  const controls = <Stack direction="row" gap={1}>
    <Button form={formId} variant="contained" type="submit" disabled={!valid || !dirty || busy}>保存</Button>
    <Button onClick={close}>取消</Button>
  </Stack>;
  return (
    <Box
      component="form"
      id={formId}
      onSubmit={(e) => {
        e.preventDefault();
        submit();
      }}
    >
        <Typography component="h3">
          {operations[draft.mode]} · {names[draft.kind]}
        </Typography>
      <Box className="object-detail-columns">
        <Stack spacing={1}>
        {!draft.action && !draft.successor && options.length > 1 && (
          <ObjectTextField
            select
            label="办理类型"
            value={draft.kind}
            onChange={(e) => update({ ...draft, kind: e.target.value as JobDraft['kind'], handover:undefined, incumbent:undefined, paper:undefined, observation:undefined, predecessor:undefined, occurrence:undefined, contracts:undefined })}
          >
            {options.map((kind) => (
              <MenuItem key={kind} value={kind}>
                {names[kind]}
              </MenuItem>
            ))}
          </ObjectTextField>
        )}
        {!duty && <ObjectTextField
          type="date"
          label={draft.kind==='Leave'||draft.kind==='EndAssignment' ? '原通知日期' : '生效日期'}
          value={draft.day}
          required
          disabled={draft.mode === 'correct' || draft.mode==='withdraw' || !!draft.successor}
          onChange={(e) => update({ ...draft, day: e.target.value, peer:undefined, incumbent:undefined,contracts:undefined, predecessor:undefined, occurrence:undefined })}
        />}
        </Stack>
        <Stack spacing={1}>
        {draft.mode !== 'redate' &&
          (duty ? <ActualDutyFields draft={draft} update={update} view={view}/> : draft.kind==='PositionExchange' ? <ExchangeFields draft={draft} update={update} assignment={view.assignment} csrf={csrf}/> : lifecycle ? <LifecycleFields view={view} draft={draft} csrf={csrf} update={update} impact={leaveImpact} contractsConfirmed={contractsConfirmed}/> : grade ? <PromotionFields view={view} draft={draft} csrf={csrf} update={update} mode={targets} derived={derived.data} arrangementError={derived.error} /> : relation ? (
            <ProbationFields view={view} draft={draft} csrf={csrf} update={update} />
          ) : location ? (
            <>
              <ObjectDetailField label="业务原因">工作地点变更</ObjectDetailField>
              <ObjectDetailField label="工作地点" inputId="assignment-location" required>
                <Autocomplete
                  id="assignment-location"
                  fullWidth
                  size="small"
                  value={draft.value ?? null}
                  options={choices.data ?? []}
                  filterOptions={(v) => v}
                  isOptionEqualToValue={(a, b) => a.id === b.id}
                  getOptionLabel={(v) => v.name ?? v.id}
                  onInputChange={(_, text, reason) => {
                    if (reason === 'input' || reason === 'clear') setSearch(text);
                  }}
                  onChange={(_, v) => update({ ...draft, value: v ?? undefined })}
                  noOptionsText={choices.error ?? '没有匹配的地点'}
                  renderInput={(props) => (
                    <TextField
                      {...props}
                      variant="standard"
                      required
                      error={!!choices.error}
                      helperText={choices.error}
                    />
                  )}
                />
              </ObjectDetailField>
            </>
          ) : (
            <>
              {view.state && (
                <ObjectDetailField label="查看日任职">
                  {view.state.department} · {view.state.job}
                </ObjectDetailField>
              )}
              {targets?.jobRequired && <>
              <TransferTarget
                kind="department"
                draft={draft}
                assignment={view.assignment}
                csrf={csrf}
                change={(v) => update({ ...draft, department: v })}
              />
              <TransferTarget
                kind="job"
                draft={draft}
                assignment={view.assignment}
                csrf={csrf}
                change={(v) => update({ ...draft, job: v })}
              />
              </>}
              {targets?.positionRequired && view.canHandover && <ObjectTextField select label="调动方式" value={draft.handover?'handover':'regular'}
                onChange={e=>update({...draft,handover:e.target.value==='handover',incumbent:undefined})}>
                <MenuItem value="regular">普通调入</MenuItem><MenuItem value="handover">内部接任</MenuItem>
              </ObjectTextField>}
              {targets?.positionRequired && <TransferTarget kind="position" mixed={targets.jobRequired} draft={draft} assignment={view.assignment} csrf={csrf}
                change={(position) => update({ ...draft, position,incumbent:undefined })}/>}
              {draft.handover && draft.position && <SuccessionFields draft={draft} assignment={view.assignment} csrf={csrf} update={update}/>}
              {derived.data?.department && <ObjectDetailField label="岗位所属部门">{derived.data.department.name}</ObjectDetailField>}
              {derived.data?.job && <ObjectDetailField label="岗位对应职务">{derived.data.job.name}</ObjectDetailField>}
              {(mode.error || derived.error) && <Typography role="alert" color="error">{mode.error || derived.error}</Typography>}
              <ObjectTextField
                label="调动依据"
                value={draft.notice}
                required
                onChange={(e) => update({ ...draft, notice: e.target.value })}
              />
              <EarlierAction draft={draft} view={view} update={update}/>
              {(candidates.length > 0 || prior) && (
                <ObjectTextField
                  select
                  label="通知要求的先行办理"
                  value={prior?.action ?? ''}
                  onChange={(e) => {
                    const a = candidates.find((v) => v.id === e.target.value);
                    update({ ...draft, requires: a ? { action: a.id, revision: a.revision } : undefined });
                  }}
                >
                  <MenuItem value="">无</MenuItem>
                  {candidates.map((a) => (
                    <MenuItem key={a.id} value={a.id}>
                      {a.businessDate} · {actionLabel(a)}
                    </MenuItem>
                  ))}
                </ObjectTextField>
              )}
            </>
          ))}
        </Stack>
      </Box>
      {actionSlot ? createPortal(controls, actionSlot) : controls}
    </Box>
  );
}
