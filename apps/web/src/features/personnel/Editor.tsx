import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Alert, Box, Button, MenuItem } from '@mui/material';
import PersonIcon from '@mui/icons-material/Person';
import {
  ObjectForm,
  ObjectTextField,
  ObjectDetailSheet,
  ObjectDetailIdentity,
  ObjectDetailSection,
  ObjectDetailField,
  ObjectFormError,
  RecordControl,
  UnsavedChangesDialog,
  focusObjectField,
} from '@chnhcm/ui';
import { ApiError, type Models } from '@chnhcm/api-client';
import { useDraftExit } from '../../shared/dated/useDraftExit';
import * as person from './api';
import { actionLabel, kindLabel, documentLabels } from './Detail';
import { ProfileSections, type RowError } from './ProfileSections';
import {
  fromProfile,
  reconcileItem,
  requiredByGroup,
  isProfileItem,
  groups,
  type Group,
  command as profileCommand,
} from './profileDraft';
import { usePersonSubmission } from './Submission';

type Row = {
  key: string;
  itemId?: string;
  revision?: string;
  kind: string;
  value: string;
  removed?: boolean;
};
const blank = (): Row => ({ key: crypto.randomUUID(), kind: '', value: '' });
function initialRows(profile: person.Profile | undefined, kinds: string[]) {
  const rows: Row[] =
    profile?.contacts
      ?.filter((x) => !x.masked && kinds.includes(x.kind))
      .map((x) => ({ ...x, key: x.itemId })) || [];
  return rows.length || !kinds.length ? rows : [blank()];
}
type Command =
  | { kind: 'register'; input: Omit<Models['PersonRegister'], 'submissionId'> }
  | { kind: 'correct'; input: Omit<Models['PersonCorrect'], 'submissionId'> }
  | { kind: 'renumber'; input: Omit<Models['PersonRenumber'], 'submissionId'> };
export function Editor({
  mode,
  session,
  profile,
  entry,
  back,
  recheck,
  saved,
}: {
  mode: string;
  session: person.Principal;
  profile?: person.Profile;
  entry?: person.Entry;
  back: string;
  recheck: () => void;
  saved: (r: person.Result, section?: string) => void;
}) {
  const creating = mode === 'register',
    renumber = mode === 'renumber',
    navigate = useNavigate();
  const [basis, setBasis] = useState(profile),
    fields = profile?.editable || [],
    kinds = entry?.contactKinds || fields.filter((x) => ['phone', 'email'].includes(x)),
    canDocument = creating ? !!entry?.canRegisterDocument : fields.includes('document'),
    canEvidence = creating
      ? !!entry?.canRegisterEvidence
      : fields.includes('evidence') && !!profile?.evidence;
  const [name, setName] = useState(profile?.name || ''),
    [kind, setKind] = useState(profile?.evidence?.kind || 'form'),
    [ref, setRef] = useState(profile?.evidence?.reference || ''),
    [documentKind, setDocumentKind] = useState(canDocument ? profile?.primaryDocument?.kind || '' : ''),
    [documentNumber, setDocumentNumber] = useState(canDocument ? profile?.primaryDocument?.number || '' : '');
  const [rows, setRows] = useState(() => initialRows(profile, kinds)),
    [newNumber, setNumber] = useState(''),
    [reason, setReason] = useState(''),
    [decision, setDecision] = useState(''),
    [until, setUntil] = useState('');
  const [draft, setDraft] = useState(() => fromProfile(profile)),
    [lineDirty, setLineDirty] = useState(false);
  const patch = profileCommand(basis, draft);
  const [rowError, setRowError] = useState<RowError>();
  const personal = !!profile && mode === 'correct';
  const legacyDocument = personal && !profile?.readable?.includes('documentKind');
  const [notice, setNotice] = useState<{ message: string; field?: string; code?: string }>();
  const error = notice?.message ?? '';
  const field = notice?.field;
  const [conflict, setConflict] = useState(false),
    [existingId, setExistingId] = useState('');
  const contactEdits: Models['PersonEdit'][] = [];
  for (const x of rows) {
    if (!x.itemId) {
      if (!x.removed && (x.kind || x.value)) contactEdits.push({ kind: x.kind, value: x.value });
      continue;
    }
    const original = basis?.contacts?.find((i) => i.itemId === x.itemId);
    if (x.removed) contactEdits.push({ itemId: x.itemId, expectedRevision: x.revision, remove: true });
    else if (x.value !== original?.value || x.kind !== original?.kind)
      contactEdits.push({ itemId: x.itemId, expectedRevision: x.revision, value: x.value, kind: x.kind });
  }
  const nameChanged = name !== (basis?.name || ''),
    evidenceChanged =
      canEvidence &&
      (kind !== (basis?.evidence?.kind || 'form') || ref !== (basis?.evidence?.reference || '')),
    documentChanged =
      canDocument &&
      (documentKind !== (basis?.primaryDocument?.kind || '') ||
        documentNumber !== (basis?.primaryDocument?.number || ''));
  const dirty = renumber
    ? !!(newNumber || reason || decision || until)
    : personal
      ? nameChanged ||
        (legacyDocument && documentChanged) ||
        !!(
          patch.nationalityChoice ||
          Object.keys(patch.basic).length ||
          patch.contacts.length ||
          patch.documents.length ||
          patch.addresses.length ||
          patch.family.length ||
          patch.education.length ||
          patch.credentials.length ||
          patch.bankAccounts.length ||
          patch.healthSummary !== undefined ||
          lineDirty
        )
      : nameChanged || documentChanged || evidenceChanged || contactEdits.length > 0;
  const leaving = useRef(false);
  const flow = usePersonSubmission<Command>(
    session,
    (v, submissionId) => {
      if (v.kind === 'register') return person.register(session, { ...v.input, submissionId });
      if (v.kind === 'correct') return person.correct(session, profile!.id, { ...v.input, submissionId });
      return person.renumber(session, profile!.id, { ...v.input, submissionId });
    },
    finish,
  );
  function finish(r: person.Result) {
    if (r.outcome === 'accepted' || r.outcome === 'no_change') {
      leaving.current = true;
      saved(
        r,
        personal
          ? groups.find((group) => patch[group].length) ||
              (patch.healthSummary !== undefined ? 'health' : 'basic')
          : undefined,
      );
    } else {
      setExistingId(r.existingPersonId || '');
      setNotice({ message: r.outcome === 'not_applied' ? '原提交未执行，可重新保存' : '原提交已拒绝，请重新核对' });
    }
  }
  const exit = useDraftExit(
    dirty,
    () => leaving.current || flow.saved,
    () => {
      leaving.current = true;
      flow.clearPayload();
    },
  );
  function fail(msg: string, id = '', code?: string) {
    setNotice({ message: msg, field: id, code });
    if (!/^(documents|contacts|addresses|family|education|credentials|bankAccounts)\.\d+\./.test(id))
      focusObjectField(id === 'healthSummary' ? id : 'person-error', 'person-error');
  }
  const patchRow = (key: string, change: Partial<Row>) =>
    setRows((v) => v.map((x) => (x.key === key ? { ...x, ...change } : x)));
  async function save() {
    if (!creating && !dirty) {
      fail('没有修改内容');
      return;
    }
    let command: Command;
    const primaryDocument =
      documentKind || documentNumber ? { kind: documentKind, number: documentNumber } : undefined;
    if (canDocument && primaryDocument && (!documentKind || !documentNumber)) {
      fail(
        !documentKind ? '请选择主要证件类型' : '请填写主要证件号码',
        !documentKind ? 'documentType' : 'documentNumber',
      );
      return;
    }
    if (creating) {
      if (!name.trim()) {
        fail('请填写姓名', 'name');
        return;
      }
      command = {
        kind: 'register',
        input: {
          name,
          primaryDocument: canDocument ? primaryDocument : undefined,
          evidence: canEvidence && ref ? { kind: 'form', reference: ref } : undefined,
          contacts: contactEdits.map((x) => ({ kind: x.kind!, value: x.value! })),
        },
      };
    } else if (renumber) {
      command = {
        kind: 'renumber',
        input: {
          expectedSourceVersion: basis!.sourceVersion,
          oldNumber: basis!.number!,
          newNumber,
          reason,
          decisionRef: decision,
          validUntil: until || undefined,
        },
      };
    } else {
      command = {
        kind: 'correct',
        input: {
          expectedSourceVersion: basis!.sourceVersion,
          ...(nameChanged ? { name } : {}),
          ...patch,
          ...(legacyDocument && documentChanged ? { primaryDocument: primaryDocument ?? null } : {}),
        },
      };
    }
    setNotice(undefined);
    setExistingId('');
    const result = await flow.submit(command);
    if (result.kind === 'saved') finish(result.result);
    if (result.kind === 'failed') {
      const e = result.error;
      fail(
        result.definitive ? person.message(e) : '结果待确认，请查看上次保存结果',
        e instanceof ApiError ? e.field : '',
        e instanceof ApiError ? e.code : undefined,
      );
      if (e instanceof ApiError) {
        const match =
          /^(documents|contacts|addresses|family|education|credentials|bankAccounts)\.(\d+)\.(.+)$/.exec(
            e.field || '',
          );
        if (match) {
          const group = match[1] as Group,
            index = Number(match[2]),
            edit = patch[group][index];
          const additions = draft[group].filter((r) => !r.revision && !r.removed);
          const key =
            edit?.itemId || additions[patch[group].slice(0, index).filter((x) => !x.itemId).length]?.key;
          if (e.code === 'PERSON_PLACE_DENIED' && group === 'addresses') {
            const original = fromProfile(basis);
            const affected =
              match[3] === 'addressCountry' ? ['addressCountry', 'province', 'area'] : ['province', 'area'];
            setDraft((current) => ({
              ...current,
              addresses: current.addresses.map((row) => {
                const old = original.addresses.find((x) => x.key === row.key);
                const places = { ...row.places },
                  values = { ...row.fields };
                for (const field of affected) {
                  if (
                    !places[field] ||
                    (places[field]?.id === old?.places?.[field]?.id &&
                      places[field]?.sourceVersion === old?.places?.[field]?.sourceVersion)
                  )
                    continue;
                  places[field] = old?.places?.[field];
                  values[field] = old?.fields[field] || '';
                }
                return { ...row, places, fields: values };
              }),
            }));
          }
          if (key) setRowError({ group, key, field: match[3], message: person.message(e) });
        }
        setExistingId(e.existingPersonId || '');
        if (e.status === 403 || e.status === 404) recheck();
        if (e.code === 'PERSON_CONFLICT' && !match && !creating) {
          setConflict(true);
          recheck();
        }
      }
    }
  }
  // Drop a revoked value before painting the new projection, preserving unrelated edits.
  const scope = JSON.stringify([
    fields,
    profile?.readable,
    ...[
      profile?.addresses,
      profile?.family,
      profile?.education,
      profile?.credentials,
      profile?.bankAccounts,
    ].map((rows) => rows?.map((x) => [x.itemId, x.locked, x.hidden, x.masked, x.canRemove])),
    profile?.documents?.map((x) => [x.itemId, x.locked]),
    kinds,
    canDocument,
    canEvidence,
    !!profile?.evidence,
    profile?.contacts?.map((x) => [x.itemId, x.kind, x.masked, x.locked]),
  ]);
  const [previousScope, setPreviousScope] = useState(scope);
  if (previousScope !== scope) {
    setPreviousScope(scope);
    reconcile(true);
  }
  const submissionScope = useRef(scope);
  useEffect(() => {
    if (submissionScope.current !== scope && flow.attempt) flow.clearPayload();
    submissionScope.current = scope;
  }, [scope, flow]);
  function reconcile(scopeChanged: boolean) {
    const denial = notice?.code === 'PERSON_DENIED' ? { message: notice.message, code: notice.code } : undefined;
    setExistingId('');
    if (!canDocument || !documentChanged) {
      setDocumentKind(canDocument ? profile?.primaryDocument?.kind || '' : '');
      setDocumentNumber(canDocument ? profile?.primaryDocument?.number || '' : '');
    }
    if (!profile) {
      if (!canEvidence) {
        setKind('form');
        setRef('');
      }
      setRows((current) => current.filter((row) => !row.kind || kinds.includes(row.kind)));
      setNotice(denial);
      return;
    }
    // Keep unrelated authorized drafts while dropping revoked values and row snapshots.
    const clean = fromProfile(profile);
    if (fields.includes('healthSummary') && patch.healthSummary !== undefined)
      clean.healthSummary = patch.healthSummary;
    if (fields.includes('nationality') && (patch.nationalityChoice || 'nationality' in patch.basic))
      clean.nationality = draft.nationality;
    for (const [key, value] of Object.entries(patch.basic))
      if (fields.includes(key)) clean.basic[key] = value;
    for (const group of groups) {
      const existing = clean[group];
      for (const row of draft[group]) {
        const latest = existing.find((x) => x.key === row.key);
        const old = fromProfile(basis)[group].find((x) => x.key === row.key);
        if (row.revision && (!latest || !old)) continue;
        if (isProfileItem(group)) {
          if (!latest && !requiredByGroup[group].every((k) => fields.includes(k))) continue;
          const next = reconcileItem(old, row, latest || { key: row.key, fields: {} }, fields);
          if (latest) existing[existing.indexOf(latest)] = next;
          else existing.push(next);
          continue;
        }
        const writable =
          group === 'documents'
            ? fields.includes('documentKind') && fields.includes('documentNumber')
            : kinds.includes(row.fields.kind) && (!latest || kinds.includes(latest.fields.kind));
        if (!latest && !writable) continue;
        const target = latest || { key: row.key, fields: {} as Record<string, string> };
        for (const [key, value] of Object.entries(row.fields)) {
          const can = key === 'value' || key === 'kind' ? writable : fields.includes(key);
          if (can && !latest?.locked?.includes(key) && value !== (old?.fields[key] || ''))
            target.fields[key] = value;
        }
        if (
          group === 'documents' &&
          fields.includes('issuingCountry') &&
          (row.fields.issuingCountry !== old?.fields.issuingCountry || row.country?.id !== old?.country?.id)
        )
          target.country = row.country;
        if (row.removed && writable && !latest?.locked?.length) target.removed = true;
        if (!latest) existing.push(target);
      }
    }
    setDraft(clean);
    setRowError(undefined);
    setRows((previous) => {
      const latest = initialRows(profile, kinds);
      for (const edit of previous) {
        if (!edit.itemId) {
          if ((!edit.kind || kinds.includes(edit.kind)) && (edit.kind || edit.value)) latest.push(edit);
          continue;
        }
        const original = basis?.contacts?.find((x) => x.itemId === edit.itemId);
        if (original && !edit.removed && original.value === edit.value && original.kind === edit.kind)
          continue;
        const index = latest.findIndex((x) => x.itemId === edit.itemId);
        if (index >= 0 && kinds.includes(edit.kind))
          latest[index] = { ...edit, revision: scopeChanged ? edit.revision : latest[index].revision };
      }
      return latest;
    });
    if (!nameChanged || !fields.includes('name')) setName(profile.name || '');
    if (!evidenceChanged || !fields.includes('evidence')) {
      setKind(profile.evidence?.kind || 'form');
      setRef(profile.evidence?.reference || '');
    }
    const stale = scopeChanged && basis && basis.sourceVersion !== profile.sourceVersion;
    setBasis(stale ? { ...profile, sourceVersion: basis.sourceVersion } : profile);
    setConflict(!!stale);
    if (scopeChanged) {
      // Keep the definitive denial while discarding stale field names and values.
      setNotice(denial);
    } else {
      setNotice({ message: '已载入当前资料，请核对后保存' });
    }
  }
  const disabled = flow.inFlight || flow.unknown;
  const input = (
    id: string,
    label: string,
    value: string,
    set: (v: string) => void,
    options?: Record<string, string>,
  ) => (
    <ObjectTextField
      id={id}
      required={id === 'name'}
      label={label}
      value={value}
      onChange={(e) => set(e.target.value)}
      disabled={disabled}
      select={!!options}
      error={field === id}
      helperText={field === id ? error : undefined}
    >
      {options &&
        Object.entries(options).map(([v, l]) => (
          <MenuItem key={v} value={v}>
            {l}
          </MenuItem>
        ))}
    </ObjectTextField>
  );
  return (
    <Box className="standard-page object-page">
      <RecordControl
        identity={basis?.number || '员工'}
        navigation={null}
        actions={
          <>
            <Button variant="contained" disabled={disabled || conflict} onClick={save}>
              保存
            </Button>
            <Button
              disabled={flow.inFlight}
              onClick={() =>
                exit.request(() => {
                  leaving.current = true;
                  navigate(back);
                })
              }
            >
              取消
            </Button>
          </>
        }
      />
      <Box className="page-scroll" data-page-scroll role="region" tabIndex={0} aria-label={actionLabel[mode]}>
        <ObjectDetailSheet
          identity={
            <ObjectDetailIdentity icon={<PersonIcon />} title={actionLabel[mode]} summary={basis?.name} />
          }
        >
          <ObjectForm>
            <ObjectFormError
              id="person-error"
              message={error}
              issues={field ? [{ id: field, message: error }] : []}
            />
            {existingId && (
              <Button onClick={() => exit.request(() => navigate(person.path + '/' + existingId))}>
                查看已有档案
              </Button>
            )}
            {conflict && (
              <Alert action={<Button onClick={() => reconcile(false)}>重新核对当前资料</Button>}>
                当前姓名 {profile?.name}，编号 {profile?.number}。
                {profile?.contacts?.map((x) => kindLabel[x.kind] + ' ' + x.value).join('，')}
              </Alert>
            )}
            {renumber ? (
              <Box className="object-detail-columns">
                <ObjectDetailSection title="人员编号">
                  <ObjectDetailField label="当前号码">{basis?.number}</ObjectDetailField>
                  {input('newNumber', '新号', newNumber, setNumber)}
                </ObjectDetailSection>
                <ObjectDetailSection title="改号依据">
                  {input('reason', '原因', reason, setReason, {
                    correction: '编号纠错',
                    migration: '制度迁移',
                  })}
                  {input('decisionRef', '客户决定文号', decision, setDecision)}
                  <ObjectTextField
                    id="validUntil"
                    type="date"
                    label="有效截止日"
                    value={until}
                    onChange={(e) => setUntil(e.target.value)}
                  />
                </ObjectDetailSection>
              </Box>
            ) : personal && profile ? (
              <>
                <ProfileSections
                  data={profile}
                  recheck={recheck}
                  editing={{
                    scope, rowError, draft, setDraft, name, setName, disabled, setLineDirty,
                    healthError: field === 'healthSummary' ? error : undefined,
                  }}
                />
                {legacyDocument && canDocument && (
                  <ObjectDetailSection title="主要证件">
                    {input('documentType', '证件类型', documentKind, setDocumentKind, {
                      '': '未填写',
                      ...documentLabels,
                    })}
                    {input('documentNumber', '证件号码', documentNumber, setDocumentNumber)}
                  </ObjectDetailSection>
                )}
              </>
            ) : (
              <>
                <Box className="object-detail-columns">
                  <ObjectDetailSection title="基本资料">
                    <ObjectDetailField label="人员编号">{basis?.number || '自动分配'}</ObjectDetailField>
                    {creating || fields.includes('name') ? (
                      input('name', '姓名', name, setName)
                    ) : (
                      <ObjectDetailField label="姓名">{name}</ObjectDetailField>
                    )}
                  </ObjectDetailSection>
                  {canDocument && (
                    <ObjectDetailSection title="主要证件">
                      {input(
                        'documentType',
                        '证件类型',
                        documentKind,
                        (value) => {
                          setExistingId('');
                          setDocumentKind(value);
                        },
                        { '': '未填写', ...documentLabels },
                      )}
                      {input('documentNumber', '证件号码', documentNumber, (value) => {
                        setExistingId('');
                        setDocumentNumber(value);
                      })}
                    </ObjectDetailSection>
                  )}
                </Box>
                {canEvidence && (
                  <ObjectDetailSection title="登记来源">
                    {!creating &&
                      input('evidenceType', '材料类型', kind, setKind, {
                        form: kindLabel.form,
                        personal: kindLabel.personal,
                      })}
                    {input('reference', creating ? '登记单号（选填）' : '材料编号', ref, setRef)}
                  </ObjectDetailSection>
                )}
                {basis?.contacts
                  ?.filter((x) => x.masked || !kinds.includes(x.kind))
                  .map((x) => (
                    <ObjectDetailField key={x.itemId} label={kindLabel[x.kind]}>
                      {x.value}
                    </ObjectDetailField>
                  ))}
                {!!kinds.length && (
                  <ObjectDetailSection title="联系方式">
                    {rows
                      .filter((x) => !x.removed)
                      .map((x, i) => (
                        <Box key={x.key} className="object-detail-columns">
                          {input(
                            'kind-' + x.key,
                            '联系类型 ' + (i + 1),
                            x.kind,
                            (v) => patchRow(x.key, { kind: v }),
                            Object.fromEntries(kinds.map((k) => [k, kindLabel[k]])),
                          )}
                          <Box>
                            {input(
                              i === 0 ? 'contacts' : 'value-' + x.key,
                              '联系值 ' + (i + 1),
                              x.value,
                              (v) => patchRow(x.key, { value: v }),
                            )}
                            <Button disabled={disabled} onClick={() => patchRow(x.key, { removed: true })}>
                              移除
                            </Button>
                          </Box>
                        </Box>
                      ))}
                    <Button disabled={disabled} onClick={() => setRows((v) => [...v, blank()])}>
                      添加联系方式
                    </Button>
                  </ObjectDetailSection>
                )}
              </>
            )}
          </ObjectForm>
        </ObjectDetailSheet>
      </Box>
      <UnsavedChangesDialog {...exit.dialog} />
    </Box>
  );
}
