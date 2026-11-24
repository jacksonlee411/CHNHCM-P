import { useEffect, useLayoutEffect, useState } from 'react';
import {
  Button,
  Checkbox,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  IconButton,
  MenuItem,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import {
  ObjectDetailField,
  ObjectDetailSection,
  ObjectForm,
  ObjectTextField,
  UnsavedChangesDialog,
} from '@chnhcm/ui';
import * as person from './api';
import {
  applyRow,
  basicColumns,
  groups,
  labelsByGroup,
  requiredByGroup,
  reconcileItem,
  itemOptions,
  type Places,
  fromProfile,
  kinds,
  type Draft,
  type Row,
  type Group,
} from './profileDraft';
import './profile.css';
import { ProfileCountry } from './ProfileCountry';
import { ProfileCollection } from './ProfileCollection';

const titles = {
  documents: '身份证件',
  contacts: '联系方式',
  addresses: '地址',
  family: '家庭成员与紧急联系人',
};
const isFlag = (k: string) => ['longTerm', 'contactPrimary', 'addressPrimary', 'emergency'].includes(k);
export type RowError = { group: Group; key: string; field: string; message: string };
type Editing = {
  scope: string;
  rowError?: RowError;
  draft: Draft;
  setDraft: (v: Draft) => void;
  name: string;
  setName: (v: string) => void;
  disabled: boolean;
  setLineDirty: (v: boolean) => void;
};
type RowMode = '新增记录' | '记录详情' | '编辑记录';
type Selection = { group: Group; key: string; mode: RowMode; newRow?: Row; error?: RowError };
type ActiveSelection = { group: Group; row: Row; mode: RowMode; error?: RowError };

function RowDialog({
  selection: s,
  data,
  apply,
  close,
  dirtyChanged,
  scope,
}: {
  scope?: string;
  selection: ActiveSelection;
  data: person.Profile;
  apply: (r: Row) => void;
  close: () => void;
  dirtyChanged?: (v: boolean) => void;
}) {
  const view = s.mode === '记录详情';
  const [inputs, set] = useState(view ? {} : s.row.fields),
    [discard, setDiscard] = useState(false),
    [error, setError] = useState(s.error?.field || ''),
    [reveal, setReveal] = useState(false);
  const labels = labelsByGroup[s.group];
  const fields = view ? s.row.fields : inputs;
  const read = data.readable || [],
    write = (data.editable || []).filter((k) => !s.row.locked?.includes(k));
  const [country, setCountry] = useState(view ? undefined : s.row.country);
  const [places, setPlaces] = useState<Places>(view ? {} : s.row.places || {});
  const [basis, setBasis] = useState({ scope, row: view ? undefined : s.row });
  if (basis.scope !== scope) {
    setBasis({ scope, row: view ? undefined : s.row });
    if (basis.row && !view && (s.group === 'addresses' || s.group === 'family')) {
      const current = reconcileItem(basis.row, { ...basis.row, fields: inputs, places }, s.row, write);
      set(current.fields);
      setPlaces(current.places || {});
      setError('');
      setDiscard(false);
    }
  }
  const dirty =
    !view &&
    (JSON.stringify(places) !== JSON.stringify(s.row.places || {}) ||
      JSON.stringify(fields) !== JSON.stringify(s.row.fields) ||
      country?.id !== s.row.country?.id ||
      country?.sourceVersion !== s.row.country?.sourceVersion);
  useEffect(() => {
    dirtyChanged?.(dirty);
    return () => dirtyChanged?.(false);
  }, [dirty, dirtyChanged]);
  const full = person.useRead(reveal ? data.id + s.row.key : undefined, () =>
    person.profile(data.id, undefined, 'document-' + s.row.key),
  );
  const number =
    full.data?.sourceVersion === data.sourceVersion
      ? full.data?.documents?.find((x) => x.itemId === s.row.key)?.fields.documentNumber
      : undefined;
  function update(k: string, v: string) {
    setError('');
    set((old) => ({
      ...old,
      [k]: v,
      ...(k === 'kind' && old.kind !== v ? { value: '', contactPrimary: '' } : {}),
      ...(k === 'longTerm' && v ? { expiryDate: '' } : {}),
    }));
  }
  function cancel() {
    if (dirty) setDiscard(true);
    else close();
  }
  function submit() {
    const key = requiredByGroup[s.group].find(
      (k) => !fields[k] && (!s.row.revision || write.includes(k) || k === 'value' || k === 'kind'),
    );
    if (key) {
      setError(key);
      requestAnimationFrame(() => document.getElementById('row-' + key)?.focus());
      return;
    }
    apply({ ...s.row, fields, country, places });
  }
  return (
    <>
      <Dialog
        open
        onClose={cancel}
        slotProps={{
          transition: {
            onEntered: () => {
              if (s.error) document.getElementById('row-' + s.error.field)?.focus();
            },
          },
        }}
        aria-labelledby="profile-row-title"
        maxWidth={false}
        className="profile-dialog"
      >
        <DialogTitle id="profile-row-title">
          {titles[s.group]} · {s.mode}
          <IconButton aria-label="关闭" onClick={cancel}>
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent tabIndex={0}>
          <ObjectForm>
            <div className="profile-grid">
              {Object.keys(labels).map((k) => {
                if (
                  (k === 'expiryDate' && fields.longTerm) ||
                  s.row.hidden?.includes(k) ||
                  !(
                    k === 'kind' ||
                    k === 'value' ||
                    read.includes(k) ||
                    (k === 'documentNumber' && read.includes('documentNumber.masked'))
                  )
                )
                  return null;
                const label = k === 'value' ? kinds[fields.kind] || '电话' : labels[k],
                  v = fields[k] || '';
                const can =
                  !view && (k === 'kind' || k === 'value' ? write.includes(fields.kind) : write.includes(k));
                if (!can)
                  return (
                    <ObjectDetailField label={label} key={k}>
                      {(k === 'documentNumber' ? number : undefined) ||
                        (isFlag(k)
                          ? v
                            ? '是'
                            : '否'
                          : k === 'kind' || k === 'documentKind'
                            ? kinds[v]
                            : v) || <span className="profile-empty">未填写</span>}
                      {view && k === 'documentNumber' && read.includes(k) && (
                        <Button size="small" onClick={() => setReveal(!reveal)}>
                          {reveal ? '隐藏号码' : '查看完整号码'}
                        </Button>
                      )}
                      {k === 'documentNumber' && full.error && <span role="alert">{full.error}</span>}
                    </ObjectDetailField>
                  );
                if (k === 'issuingCountry' || ['addressCountry', 'province', 'area'].includes(k))
                  return (
                    <ProfileCountry
                      key={k}
                      id={'row-' + k}
                      label={label}
                      value={v}
                      required={k === 'addressCountry'}
                      fieldError={error === k ? s.error?.message || '请填写' + label : undefined}
                      region={
                        k === 'province' || k === 'area'
                          ? {
                              country: places.addressCountry?.id,
                              parent: k === 'area' ? places.province?.id : undefined,
                              province: k === 'province',
                            }
                          : undefined
                      }
                      onChange={(text, choice) => {
                        if (k === 'issuingCountry') {
                          update(k, text);
                          setCountry(choice);
                          return;
                        }
                        const next = { ...places, [k]: choice };
                        const values = { ...fields, [k]: text };
                        for (const child of k === 'addressCountry'
                          ? ['province', 'area']
                          : k === 'province'
                            ? ['area']
                            : []) {
                          if (places[child] && places[k]?.id !== choice?.id && write.includes(child)) {
                            delete next[child];
                            values[child] = '';
                          }
                        }
                        setPlaces(next);
                        set(values);
                        setError('');
                      }}
                    />
                  );
                if (isFlag(k))
                  return (
                    <ObjectDetailField label={label} key={k} inputId={'row-' + k}>
                      <Checkbox
                        id={'row-' + k}
                        checked={!!v}
                        onChange={(e) => update(k, e.target.checked ? 'yes' : '')}
                      />
                      {error === k && <span role="alert">{s.error?.message}</span>}
                    </ObjectDetailField>
                  );
                const choices =
                  k === 'kind'
                    ? ['phone', 'email'].filter((x) => write.includes(x))
                    : k === 'documentKind'
                      ? ['resident_id', 'passport', 'other']
                      : itemOptions[k];
                return (
                  <ObjectTextField
                    key={k}
                    id={'row-' + k}
                    label={label}
                    value={v}
                    onChange={(e) => update(k, e.target.value)}
                    required={requiredByGroup[s.group].includes(k)}
                    select={!!choices}
                    type={
                      k === 'issueDate' || k === 'expiryDate' || k === 'familyBirthDate' ? 'date' : 'text'
                    }
                    error={error === k}
                    helperText={error === k ? s.error?.message || '请填写' + label : undefined}
                  >
                    {choices?.map((x) => (
                      <MenuItem key={x} value={x}>
                        {kinds[x] || x}
                      </MenuItem>
                    ))}
                  </ObjectTextField>
                );
              })}
            </div>
          </ObjectForm>
        </DialogContent>
        <DialogActions>
          {!view && <span>应用到本页草稿后，请在页顶保存。</span>}
          <div>
            {view ? (
              <Button onClick={close}>关闭</Button>
            ) : (
              <>
                <Button onClick={close}>取消</Button>
                <Button variant="contained" onClick={submit}>
                  应用
                </Button>
              </>
            )}
          </div>
        </DialogActions>
      </Dialog>
      <UnsavedChangesDialog open={discard} onContinue={() => setDiscard(false)} onDiscard={close} />
    </>
  );
}

export function ProfileSections({ data, editing }: { data: person.Profile; editing?: Editing }) {
  const draft = editing?.draft || fromProfile(data),
    read = data.readable || [],
    write = data.editable || [];
  const [selection, select] = useState<Selection>();
  // A view follows the latest authorized projection, including changes to a single open row.
  const item = selection?.group === 'addresses' || selection?.group === 'family';
  const scope =
    editing?.scope ??
    JSON.stringify([
      data.sourceVersion,
      data.readable,
      data.editable,
      data.contacts?.map((x) => [x.itemId, x.masked]),
    ]);
  const [previousScope, setPreviousScope] = useState(scope);
  if (previousScope !== scope) {
    setPreviousScope(scope);
    if (!item) select(undefined);
  }
  let active: ActiveSelection | undefined;
  if (selection) {
    const row = selection.newRow || draft[selection.group].find((r) => r.key === selection.key && !r.removed);
    if (
      row &&
      !(item && selection.newRow && !requiredByGroup[selection.group].every((k) => write.includes(k)))
    ) {
      const writable = Object.keys(labelsByGroup[selection.group]).some(
        (key) => write.includes(key) && !row.locked?.includes(key) && !row.hidden?.includes(key),
      );
      const mode = item && selection.mode === '编辑记录' && !writable ? '记录详情' : selection.mode;
      active = { group: selection.group, mode, row, error: selection.error };
    }
  }
  const missingGroup = selection && !active ? selection.group : undefined;
  useLayoutEffect(() => {
    if (!missingGroup) return;
    select(undefined);
    requestAnimationFrame(() => {
      (
        document.getElementById('profile-' + missingGroup) || document.getElementById('profile-basic')
      )?.focus();
    });
  }, [missingGroup]);
  const issue = editing?.rowError;
  useEffect(() => {
    if (!issue) return;
    const row = draft[issue.group].find((x) => x.key === issue.key && !x.removed);
    if (row) select({ group: issue.group, key: row.key, mode: '编辑记录', error: issue });
  }, [issue]);
  function apply(group: Group, row: Row) {
    editing?.setDraft(applyRow(draft, group, row));
    select(undefined);
    requestAnimationFrame(() => {
      const section = document.getElementById('profile-' + group);
      const target =
        document.getElementById('profile-row-' + row.key) ||
        section?.querySelector<HTMLElement>('button') ||
        section;
      target?.focus();
    });
  }
  return (
    <>
      <ObjectDetailSection title="基本信息">
        <div id="profile-basic" tabIndex={-1} className="profile-basic">
          {basicColumns.map((column) => {
            const visible = Object.entries(column).filter(
              ([k]) => k === 'name' || k === 'number' || read.includes(k),
            );
            if (!visible.length) return null;
            return (
              <div className="profile-basic-column" key={visible[0][0]}>
                {visible.map(([k, label]) => {
                  const value =
                    k === 'number'
                      ? data.number
                      : k === 'name'
                        ? (editing?.name ?? data.name)
                        : draft.basic[k];
                  if (!editing || !write.includes(k) || k === 'number') {
                    return (
                      <ObjectDetailField key={k} label={label}>
                        {value || <span className="profile-empty">未填写</span>}
                      </ObjectDetailField>
                    );
                  }
                  if (k === 'nationality') {
                    return (
                      <ProfileCountry
                        key={k}
                        id={k}
                        label={label}
                        value={value || ''}
                        disabled={editing.disabled}
                        onChange={(text, choice) =>
                          editing.setDraft({
                            ...draft,
                            basic: { ...draft.basic, [k]: text },
                            nationality: choice,
                          })
                        }
                      />
                    );
                  }
                  return (
                    <ObjectTextField
                      key={k}
                      id={k}
                      label={label}
                      required={k === 'name'}
                      type={k === 'birthDate' ? 'date' : 'text'}
                      disabled={editing.disabled}
                      value={value || ''}
                      onChange={(e) =>
                        k === 'name'
                          ? editing.setName(e.target.value)
                          : editing.setDraft({ ...draft, basic: { ...draft.basic, [k]: e.target.value } })
                      }
                    />
                  );
                })}
              </div>
            );
          })}
        </div>
      </ObjectDetailSection>
      {groups.map((group) => {
        const allowed =
          group === 'contacts'
            ? read.includes('phone') || read.includes('email')
            : read.includes(requiredByGroup[group][0]);
        if (!allowed) return null;
        return (
          <ProfileCollection
            key={group}
            group={group}
            title={titles[group]}
            rows={draft[group].filter((row) => !row.removed)}
            read={read}
            write={write}
            editing={!!editing}
            disabled={editing?.disabled}
            open={(row, mode) =>
              select({ group, key: row.key, newRow: mode === '新增记录' ? row : undefined, mode })
            }
            apply={(row) => apply(group, row)}
          />
        );
      })}
      {active && (
        <RowDialog
          key={active.group + active.row.key + active.mode}
          selection={active}
          scope={editing?.scope}
          data={data}
          close={() => select(undefined)}
          apply={(r) => apply(active.group, r)}
          dirtyChanged={editing?.setLineDirty}
        />
      )}
    </>
  );
}
