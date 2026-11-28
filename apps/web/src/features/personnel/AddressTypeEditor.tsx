import { useRef, useState } from 'react';
import { Alert, Box, Button } from '@mui/material';
import HomeWorkOutlined from '@mui/icons-material/HomeWorkOutlined';
import { Link } from 'react-router-dom';
import { ApiError, sessionGuard, type Models } from '@chnhcm/api-client';
import {
  ObjectDetailField,
  ObjectDetailIdentity,
  ObjectDetailSection,
  ObjectDetailSheet,
  ObjectForm,
  ObjectTextField,
  RecordBackButton,
  RecordControl,
  UnsavedChangesDialog,
} from '@chnhcm/ui';
import { useDraftExit } from '../../shared/dated/useDraftExit';
import * as person from './api';

type Item = Models['PersonAddressTypeView'];
type Form = {
  code: string;
  name: string;
  sortOrder: string;
  firstDay: string;
  stopDay: string;
  undatedStop: boolean;
};
function fields(firstDay: string, x?: Item): Form {
  return {
    code: x?.code ?? '',
    name: x?.name ?? '',
    sortOrder: String(x?.sortOrder ?? 0),
    firstDay: x?.firstDay ?? firstDay,
    stopDay: x?.stopDay ?? '',
    undatedStop: !!x?.undatedStop,
  };
}

export function AddressTypeEditor({
  session,
  item,
  firstDay,
  defaultStopDay,
  stopping,
  back,
  cancel,
  saved,
  remember,
  refresh,
  lostAccess,
}: {
  session: person.Principal;
  item?: Item;
  firstDay: string;
  defaultStopDay?: string;
  stopping: boolean;
  back: string;
  cancel: () => void;
  saved: (x: Models['PersonAddressType']) => void;
  remember: (id: string) => void;
  refresh: () => void;
  lostAccess: () => void;
}) {
  const [basis, setBasis] = useState(item);
  const [draft, setDraft] = useState(() => ({
    ...fields(firstDay, item),
    ...(stopping ? { stopDay: defaultStopDay ?? '' } : {}),
  }));
  const [error, setError] = useState(''),
    [field, setField] = useState(''),
    [stale, setStale] = useState(false);
  const [busy, setBusy] = useState(false),
    [uncertain, setUncertain] = useState<Models['PersonAddressTypeCommand']>();
  const base = fields(firstDay, basis);
  const dirty = JSON.stringify(draft) !== JSON.stringify(base);
  const unsavedInput = stopping
    ? JSON.stringify(draft) !== JSON.stringify({ ...base, stopDay: defaultStopDay ?? '' })
    : dirty;
  const leaving = useRef(false);
  const exit = useDraftExit(
    unsavedInput,
    () => leaving.current,
    () => {
      leaving.current = true;
    },
  );
  const change = (key: keyof Form, value: string) => {
    setDraft((d) => ({ ...d, [key]: value, ...(key === 'stopDay' ? { undatedStop: false } : {}) }));
    setError('');
    setField('');
  };
  const stopChanged = draft.stopDay !== base.stopDay || draft.undatedStop !== base.undatedStop;
  async function reload() {
    if (!basis) return;
    const current = sessionGuard();
    try {
      const latest = await person.addressTypeDetail(basis.id);
      if (!current()) return;
      if (!latest.canMaintain) {
        lostAccess();
        return;
      }
      const next = fields(firstDay, latest.item);
      setDraft({
        ...next,
        name: draft.name === base.name ? next.name : draft.name,
        sortOrder: draft.sortOrder === base.sortOrder ? next.sortOrder : draft.sortOrder,
        ...(stopChanged ? { stopDay: draft.stopDay, undatedStop: draft.undatedStop } : {}),
      });
      setBasis(latest.item);
      setError('');
      setStale(false);
      refresh();
    } catch (e) {
      if (current()) {
        setError(person.message(e));
        refresh();
      }
    }
  }
  async function submit() {
    if (busy || stale || (!dirty && !uncertain)) return;
    const command = uncertain ?? {
      submissionId: crypto.randomUUID(),
      ...(basis
        ? { id: basis.id, expectedRevision: basis.revision }
        : { code: draft.code, firstDay: draft.firstDay }),
      name: draft.name,
      sortOrder: Number(draft.sortOrder),
      ...(stopChanged ? { stopDay: draft.stopDay } : {}),
    };
    const current = sessionGuard();
    setBusy(true);
    setError('');
    setField('');
    remember(command.submissionId);
    try {
      const result = await person.saveAddressType(session, command);
      if (current()) {
        leaving.current = true;
        remember('');
        saved(result);
      }
    } catch (e) {
      if (!current()) return;
      if (e instanceof ApiError && e.status < 500) {
        remember('');
        setUncertain(undefined);
        setError(e.message);
        setField(e.field || '');
        setStale(e.code === 'ADDRESS_TYPE_CONFLICT');
        if (e.status === 403 || e.status === 401) lostAccess();
      } else {
        setUncertain(command);
        setError('保存结果待确认，可按原提交重试。');
      }
    } finally {
      if (current()) setBusy(false);
    }
  }
  const disabled = busy || !!uncertain;
  const title = stopping ? '停用地址类型' : basis ? '更正地址类型' : '新增地址类型';
  const dateField = (
    <ObjectTextField
      id="type-stop"
      label="停用生效日"
      type="date"
      value={draft.stopDay}
      required={stopping}
      disabled={disabled}
      error={field === 'stopDay'}
      onChange={(e) => change('stopDay', e.target.value)}
    />
  );
  return (
    <Box className="standard-page object-page">
      <RecordControl
        identity={<RecordBackButton component={Link} to={back} />}
        navigation={null}
        actions={
          <>
            <Button
              type="submit"
              form="address-type-form"
              variant="contained"
              disabled={busy || stale || (!dirty && !uncertain)}
            >
              {uncertain ? '重试保存' : '保存'}
            </Button>
            <Button
              disabled={busy}
              onClick={() =>
                exit.request(() => {
                  leaving.current = true;
                  cancel();
                })
              }
            >
              取消
            </Button>
          </>
        }
      />
      <Box className="page-scroll" data-page-scroll role="region" tabIndex={0} aria-label={title}>
        <form
          id="address-type-form"
          onSubmit={(e) => {
            e.preventDefault();
            void submit();
          }}
        >
          <ObjectDetailSheet
            identity={
              <ObjectDetailIdentity icon={<HomeWorkOutlined />} title={title} summary={basis?.name} />
            }
          >
            <ObjectForm>
              <div className="profile-grid">
                {!stopping && (
                  <ObjectDetailSection title="基本资料">
                    {basis ? (
                      <ObjectDetailField label="编码">{basis.code}</ObjectDetailField>
                    ) : (
                      <ObjectTextField
                        id="type-code"
                        label="编码"
                        required
                        value={draft.code}
                        disabled={disabled}
                        error={field === 'code'}
                        onChange={(e) => change('code', e.target.value)}
                      />
                    )}
                    <ObjectTextField
                      id="type-name"
                      label="名称"
                      required
                      value={draft.name}
                      disabled={disabled}
                      error={field === 'name'}
                      onChange={(e) => change('name', e.target.value)}
                    />
                    <ObjectTextField
                      id="type-order"
                      label="排序"
                      required
                      type="number"
                      value={draft.sortOrder}
                      disabled={disabled}
                      error={field === 'sortOrder'}
                      onChange={(e) => change('sortOrder', e.target.value)}
                    />
                  </ObjectDetailSection>
                )}
                <ObjectDetailSection title="生效期间">
                  {basis ? (
                    <ObjectDetailField label="首次生效日">{basis.firstDay}</ObjectDetailField>
                  ) : (
                    <ObjectTextField
                      id="type-first"
                      label="首次生效日"
                      required
                      type="date"
                      value={draft.firstDay}
                      disabled={disabled}
                      error={field === 'firstDay'}
                      onChange={(e) => change('firstDay', e.target.value)}
                    />
                  )}
                  {dateField}
                  {draft.undatedStop && (
                    <ObjectDetailField label="既有状态">已停用，原日期未留存</ObjectDetailField>
                  )}
                  {!stopping && basis && (draft.stopDay || draft.undatedStop) && (
                    <Button size="small" disabled={disabled} onClick={() => change('stopDay', '')}>
                      更正为启用
                    </Button>
                  )}
                </ObjectDetailSection>
              </div>
            </ObjectForm>
            {error && (
              <Alert severity="error" action={stale && <Button onClick={reload}>核对当前修订</Button>}>
                {error}
              </Alert>
            )}
          </ObjectDetailSheet>
        </form>
      </Box>
      <UnsavedChangesDialog {...exit.dialog} />
    </Box>
  );
}
