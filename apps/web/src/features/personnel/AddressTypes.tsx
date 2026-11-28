import { useEffect, useState } from 'react';
import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  MenuItem,
  Stack,
} from '@mui/material';
import { useGridApiRef, type GridColDef } from '@mui/x-data-grid';
import { ApiError, sessionGuard, type Models } from '@chnhcm/api-client';
import {
  DataGridPage,
  ListControl,
  ListPager,
  ObjectDetailField,
  ObjectForm,
  ObjectTextField,
  UnsavedChangesDialog,
} from '@chnhcm/ui';
import { useDraftExit } from '../../shared/dated/useDraftExit';
import * as person from './api';
import './profile.css';

type Item = Models['PersonAddressType'];
type Form = { code: string; name: string; sortOrder: string; firstDay: string; selectable: boolean };
const defaults = (firstDay: string, item?: Item): Form => ({
  code: item?.code || '',
  name: item?.name || '',
  sortOrder: String(item?.sortOrder ?? 0),
  firstDay: item?.firstDay || firstDay,
  selectable: item?.selectable ?? true,
});

export function AddressTypes({ session }: { session: person.Principal }) {
  const owner = session.instanceId + ':' + session.subjectId;
  const data = person.useRead(owner, person.addressTypes);
  const [editing, setEditing] = useState<Item | 'new'>();
  const [pager, setPager] = useState({ page: 0, pageSize: 20 });
  const [notice, setNotice] = useState('');
  const apiRef = useGridApiRef();
  const storageKey = 'chnhcm-address-type-submission:' + owner;
  const [pending, setPending] = useState(() => sessionStorage.getItem(storageKey) || '');
  const [recovering, setRecovering] = useState(false);
  const [recoveryError, setRecoveryError] = useState('');
  useEffect(() => {
    document.title = '地址类型 · CHNHCM';
  }, []);
  useEffect(() => {
    if (data.status === 403 || data.status === 401) {
      setEditing(undefined);
      setNotice('');
    }
  }, [data.status]);
  function remember(id: string) {
    if (id) sessionStorage.setItem(storageKey, id);
    else sessionStorage.removeItem(storageKey);
    setPending(id);
    setRecoveryError('');
  }
  async function recover() {
    const current = sessionGuard();
    setRecovering(true);
    try {
      const result = await person.addressTypeReceipt(session, pending);
      if (current()) {
        remember('');
        setEditing(undefined);
        setNotice(result.item ? '已确认保存“' + result.item.name + '”' : '已确认上次提交未执行');
        data.refresh();
      }
    } catch (e) {
      if (current()) {
        setRecoveryError(person.message(e));
      }
    } finally {
      if (current()) setRecovering(false);
    }
  }
  const rows = data.data?.items || [];
  const canWrite = !!data.data?.canMaintain;
  const open = (item: Item) => (
    <Button size="small" data-object-link disabled={!!pending} onClick={() => setEditing(item)}>
      {item.name}
    </Button>
  );
  const columns: GridColDef<Item>[] = [
    { field: 'name', headerName: '名称', flex: 1, minWidth: 140, renderCell: ({ row }) => open(row) },
    { field: 'code', headerName: '编码', flex: 1, minWidth: 130 },
    { field: 'sortOrder', headerName: '排序', width: 90 },
    { field: 'firstDay', headerName: '初始生效日', width: 136 },
    {
      field: 'selectable',
      headerName: '选用状态',
      width: 110,
      renderCell: ({ row }) => (row.selectable ? '可选' : '已停选'),
    },
  ];
  return (
    <div className="standard-page">
      <DataGridPage<Item>
        apiRef={apiRef}
        label="地址类型列表"
        rows={rows.slice(pager.page * pager.pageSize, (pager.page + 1) * pager.pageSize)}
        rowCount={rows.length}
        columns={columns.map((c) => ({ ...c, sortable: false }))}
        summaryColumn={{
          field: 'summary',
          headerName: '地址类型',
          renderCell: ({ row }) => (
            <Stack>
              {open(row)}
              <small>
                {row.code} · {row.selectable ? '可选' : '已停选'}
              </small>
            </Stack>
          ),
        }}
        ready={!!data.data}
        loading={!data.data && !data.error}
        empty="没有地址类型"
        sortModel={[]}
        onSortModelChange={() => {}}
        paginationModel={pager}
        onPaginationModelChange={setPager}
        toolbar={
          <ListControl
            title="地址类型"
            pager={<ListPager />}
            action={
              canWrite && (
                <Button variant="contained" disabled={!!pending} onClick={() => setEditing('new')}>
                  新增
                </Button>
              )
            }
          />
        }
        messages={
          <>
            {data.error && (
              <Alert severity="error" action={<Button onClick={data.refresh}>重试</Button>}>
                {data.error}
              </Alert>
            )}
            {notice && data.data && (
              <Alert severity="success" onClose={() => setNotice('')}>
                {notice}
              </Alert>
            )}
            {pending && !editing && data.data && (
              <Alert
                severity="info"
                action={
                  <Button disabled={recovering} onClick={recover}>
                    查回结果
                  </Button>
                }
              >
                {recoveryError || '上次保存结果待确认'}
              </Alert>
            )}
          </>
        }
      />
      {editing && data.data && (
        <TypeDialog
          key={editing === 'new' ? 'new' : editing.id}
          session={session}
          item={editing === 'new' ? undefined : editing}
          firstDay={data.data.defaultFirstDay}
          writable={canWrite}
          close={() => setEditing(undefined)}
          refresh={data.refresh}
          remember={remember}
          saved={(value) => {
            setEditing(undefined);
            setNotice('已保存“' + value.name + '”');
            data.refresh();
          }}
        />
      )}
    </div>
  );
}

function TypeDialog({
  session,
  item,
  firstDay,
  writable,
  close,
  saved,
  refresh,
  remember,
}: {
  session: person.Principal;
  item?: Item;
  firstDay: string;
  writable: boolean;
  close: () => void;
  saved: (item: Item) => void;
  refresh: () => void;
  remember: (id: string) => void;
}) {
  const [basis, setBasis] = useState(item);
  const [draft, setDraft] = useState(() => defaults(firstDay, item));
  const [error, setError] = useState('');
  const [field, setField] = useState('');
  const [stale, setStale] = useState(false);
  const [busy, setBusy] = useState(false);
  const [uncertain, setUncertain] = useState<Models['PersonAddressTypeCommand']>();
  const dirty = writable && JSON.stringify(draft) !== JSON.stringify(defaults(firstDay, basis));
  const exit = useDraftExit(
    dirty,
    () => false,
    () => {},
  );
  const cancel = () => {
    if (!busy) exit.request(close);
  };
  const change = (key: keyof Form, value: string | boolean) => {
    setDraft((d) => ({ ...d, [key]: value }));
    setError('');
    setField('');
  };
  async function reload() {
    const current = sessionGuard();
    try {
      const list = await person.addressTypes();
      if (!current()) return;
      const latest = list.items.find((x) => x.id === basis?.id);
      if (!latest || !list.canMaintain) {
        refresh();
        close();
        return;
      }
      const previous = defaults(firstDay, basis),
        next = defaults(firstDay, latest);
      setDraft({
        ...next,
        name: draft.name === previous.name ? next.name : draft.name,
        sortOrder: draft.sortOrder === previous.sortOrder ? next.sortOrder : draft.sortOrder,
        selectable: draft.selectable === previous.selectable ? next.selectable : draft.selectable,
      });
      setBasis(latest);
      setStale(false);
      setError('');
      refresh();
    } catch (e) {
      if (current()) {
        setError(person.message(e));
        refresh();
      }
    }
  }
  async function submit() {
    if (busy || !writable || stale) return;
    const pending = uncertain || {
      submissionId: crypto.randomUUID(),
      ...(basis
        ? { id: basis.id, expectedRevision: basis.revision }
        : { code: draft.code, firstDay: draft.firstDay }),
      name: draft.name,
      sortOrder: Number(draft.sortOrder),
      selectable: draft.selectable,
    };
    const current = sessionGuard();
    setBusy(true);
    setError('');
    setField('');
    remember(pending.submissionId);
    try {
      const result = await person.saveAddressType(session, pending);
      if (current()) {
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
        if (e.code === 'ADDRESS_TYPE_CONFLICT') setStale(true);
        if (e.status === 403 || e.status === 401) {
          close();
          refresh();
        }
      } else {
        setUncertain(pending);
        setError('保存结果待确认，可按原提交重试。');
      }
    } finally {
      if (current()) setBusy(false);
    }
  }
  const disabled = busy || !writable || !!uncertain;
  return (
    <>
      <Dialog open fullWidth maxWidth="sm" onClose={cancel} aria-labelledby="address-type-title">
        <DialogTitle id="address-type-title">
          {basis ? (writable ? '编辑地址类型' : '地址类型详情') : '新增地址类型'}
        </DialogTitle>
        <form
          onSubmit={(event) => {
            event.preventDefault();
            void submit();
          }}
        >
          <DialogContent dividers>
            {writable ? (
              <ObjectForm>
                <div className="profile-grid">
                  <ObjectTextField
                    id="type-code"
                    label="编码"
                    required
                    value={draft.code}
                    disabled={disabled || !!basis}
                    error={field === 'code'}
                    onChange={(e) => change('code', e.target.value)}
                  />
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
                  <ObjectTextField
                    id="type-day"
                    label="初始生效日"
                    required
                    type="date"
                    value={draft.firstDay}
                    disabled={disabled || !!basis}
                    error={field === 'firstDay'}
                    onChange={(e) => change('firstDay', e.target.value)}
                  />
                  <ObjectTextField
                    id="type-selectable"
                    label="选用状态"
                    select
                    value={draft.selectable ? 'on' : 'off'}
                    disabled={disabled}
                    onChange={(e) => change('selectable', e.target.value === 'on')}
                  >
                    <MenuItem value="on">可选</MenuItem>
                    <MenuItem value="off">已停选</MenuItem>
                  </ObjectTextField>
                </div>
              </ObjectForm>
            ) : (
              basis && (
                <div className="profile-grid">
                  <ObjectDetailField label="编码">{basis.code}</ObjectDetailField>
                  <ObjectDetailField label="名称">{basis.name}</ObjectDetailField>
                  <ObjectDetailField label="排序">{basis.sortOrder}</ObjectDetailField>
                  <ObjectDetailField label="初始生效日">{basis.firstDay}</ObjectDetailField>
                  <ObjectDetailField label="选用状态">
                    {basis.selectable ? '可选' : '已停选'}
                  </ObjectDetailField>
                </div>
              )
            )}
            {error && (
              <Alert severity="error" action={stale && <Button onClick={reload}>核对当前修订</Button>}>
                {error}
              </Alert>
            )}
          </DialogContent>
          <DialogActions>
            <Button disabled={busy} onClick={cancel}>
              {writable ? '取消' : '关闭'}
            </Button>
            {writable && (
              <Button type="submit" variant="contained" disabled={busy || stale || !dirty}>
                {uncertain ? '重试保存' : '保存'}
              </Button>
            )}
          </DialogActions>
        </form>
      </Dialog>
      <UnsavedChangesDialog {...exit.dialog} />
    </>
  );
}
