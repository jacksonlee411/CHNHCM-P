import { useEffect, useState } from 'react';
import { Alert, Box, Button, Stack, Tabs, Tab } from '@mui/material';
import HomeWorkOutlined from '@mui/icons-material/HomeWorkOutlined';
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom';
import { useGridApiRef, type GridColDef } from '@mui/x-data-grid';
import { sessionGuard, type Models } from '@chnhcm/api-client';
import {
  DataGridPage,
  EffectiveDates,
  ListControl,
  ListPager,
  ObjectDetailField,
  ObjectDetailIdentity,
  ObjectDetailSection,
  ObjectDetailSheet,
  RecordBackButton,
  RecordControl,
  RecordPager,
  StatusChip,
  recordedTime,
} from '@chnhcm/ui';
import { AddressTypeEditor } from './AddressTypeEditor';
import * as person from './api';
import './profile.css';

type Item = Models['PersonAddressTypeView'];
const labels: Record<Models['PersonAddressTypeState'], string> = {
  active: '启用',
  inactive: '停用',
  not_effective: '未生效',
  unknown: '当时状态未留存',
};
const path = person.addressTypePath;
export function AddressTypes({ session }: { session: person.Principal }) {
  const { typeId } = useParams(),
    route = useLocation(),
    navigate = useNavigate();
  const owner = session.instanceId + ':' + session.subjectId,
    creating = typeId === 'new';
  const fromList = !!route.state?.fromList;
  const search = new URLSearchParams(route.search);
  const history = search.get('tab') === 'history';
  const back = typeId && !creating ? path + '?selected=' + encodeURIComponent(typeId) : path;
  const data = person.useRead(!typeId || creating || fromList ? owner : undefined, person.addressTypes);
  const detail = person.useRead(typeId && !creating ? owner + typeId : undefined, () =>
    person.addressTypeDetail(typeId!),
  );
  const denied = [401, 403].includes(data.status ?? 0) || [401, 403].includes(detail.status ?? 0);
  const info = denied ? undefined : detail.data,
    item = info?.item;
  const [edit, setEdit] = useState<{ id: string; stop: boolean }>();
  const [selected, setSelected] = useState<{ id: string; day: string }>();
  const [notice, setNotice] = useState('');
  const [pager, setPager] = useState({ page: 0, pageSize: 20 });
  const apiRef = useGridApiRef();
  const storageKey = 'chnhcm-address-type-submission:' + owner;
  const [pending, setPending] = useState(() => sessionStorage.getItem(storageKey) || '');
  const [recovering, setRecovering] = useState(false),
    [recoveryError, setRecoveryError] = useState('');
  useEffect(() => {
    document.title = (item?.name || '地址类型') + ' · CHNHCM';
  }, [item?.name]);
  useEffect(() => {
    setEdit(undefined);
    setSelected(undefined);
    setNotice('');
  }, [typeId]);
  useEffect(() => {
    if (
      detail.status === 403 ||
      detail.status === 401 ||
      data.status === 403 ||
      data.status === 401 ||
      (info && !info.canMaintain)
    ) {
      setEdit(undefined);
      setNotice('');
    }
  }, [detail.status, data.status, info?.canMaintain]);
  const refresh = () => {
    data.refresh();
    detail.refresh();
  };
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
        setEdit(undefined);
        setNotice(result.item ? '已确认保存“' + result.item.name + '”' : '已确认上次提交未执行');
        if (creating)
          navigate(result.item ? path + '/' + result.item.id : path, {
            replace: true,
            state: { fromList: true },
          });
        refresh();
      }
    } catch (e) {
      if (current()) setRecoveryError(person.message(e));
    } finally {
      if (current()) setRecovering(false);
    }
  }
  const errors = data.error || detail.error;
  const authorized = typeId && !creating ? !!info : !!data.data;
  const messages = (
    <>
      {errors && (
        <Alert severity="error" action={<Button onClick={refresh}>重试</Button>}>
          {errors}
        </Alert>
      )}
      {authorized && notice && (
        <Alert severity="success" onClose={() => setNotice('')}>
          {notice}
        </Alert>
      )}
      {authorized && pending && (
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
  );
  const rows = data.data?.items ?? [];
  const position = rows.findIndex((x) => x.id === typeId);
  const link = (x: Item) => (
    <Button
      size="small"
      component={Link}
      data-object-link
      to={path + '/' + x.id}
      state={{ fromList: true }}
    >
      {x.name}
    </Button>
  );
  const columns: GridColDef<Item>[] = [
    { field: 'name', headerName: '名称', flex: 1, minWidth: 140, renderCell: ({ row }) => link(row) },
    { field: 'code', headerName: '编码', flex: 1, minWidth: 120 },
    { field: 'sortOrder', headerName: '排序', width: 75 },
    { field: 'firstDay', headerName: '首次生效日', width: 128 },
    {
      field: 'stopDay',
      headerName: '停用生效日',
      width: 128,
      renderCell: ({ row }) => row.stopDay || (row.undatedStop ? '原日期未留存' : '—'),
    },
    { field: 'state', headerName: '今日状态', width: 100, renderCell: ({ row }) => labels[row.state] },
  ];
  if (!typeId)
    return (
      <div className="standard-page">
        <DataGridPage<Item>
          apiRef={apiRef}
          label="地址类型列表"
          rows={rows.slice(pager.page * pager.pageSize, (pager.page + 1) * pager.pageSize)}
          selected={search.get('selected') || route.state?.selected}
          rowCount={rows.length}
          columns={columns.map((c) => ({ ...c, sortable: false }))}
          summaryColumn={{
            field: 'summary',
            headerName: '地址类型',
            renderCell: ({ row }) => (
              <Stack>
                {link(row)}
                <small>
                  {row.code} · {labels[row.state]}
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
                data.data?.canMaintain && (
                  <Button
                    variant="contained"
                    component={Link}
                    to={path + '/new'}
                    state={{ fromList: true }}
                    disabled={!!pending}
                  >
                    新增
                  </Button>
                )
              }
            />
          }
          messages={messages}
        />
      </div>
    );
  const editing = !!(item && edit?.id === item.id && info?.canMaintain);
  const onList = () => navigate(path, { state: { selected: typeId } });
  if (
    (creating && data.data?.canMaintain && !pending) ||
    editing ||
    (creating && data.data?.canMaintain && edit?.id === 'new')
  ) {
    return (
      <AddressTypeEditor
        key={typeId + (edit?.stop ? 'stop' : 'edit')}
        session={session}
        item={item}
        firstDay={data.data?.defaultFirstDay ?? item?.firstDay ?? ''}
        defaultStopDay={info?.defaultStopDay}
        stopping={!!edit?.stop}
        back={back}
        cancel={creating ? onList : () => setEdit(undefined)}
        refresh={refresh}
        lostAccess={() => {
          setEdit(undefined);
          setNotice('');
          data.invalidate();
          detail.invalidate();
        }}
        remember={(id) => {
          if (creating && id) setEdit({ id: 'new', stop: false });
          remember(id);
        }}
        saved={(x) => {
          setEdit(undefined);
          setSelected(undefined);
          setNotice('已保存“' + x.name + '”');
          refresh();
          if (creating) navigate(path + '/' + x.id, { replace: true, state: { fromList: true } });
        }}
      />
    );
  }
  const point =
    info?.timeline.find((p) => p.day === (selected?.id === typeId ? selected?.day : undefined)) ??
    info?.timeline.find((p) => p.day === info.selectedDay);
  const move = (direction: 'previous' | 'next') => {
    const index = position + (direction === 'previous' ? -1 : 1),
      target = rows[index];
    if (target) {
      setPager((p) => ({ ...p, page: Math.floor(index / p.pageSize) }));
      navigate(path + '/' + target.id, { state: { fromList: true } });
    }
  };
  return (
    <Box className="standard-page object-page">
      <RecordControl
        identity={<RecordBackButton component={Link} to={back} />}
        actions={
          item &&
          info?.canMaintain && (
            <>
              <Button
                variant="contained"
                disabled={!!pending}
                onClick={() => setEdit({ id: item.id, stop: false })}
              >
                更正
              </Button>
              {!item.stopDay && !item.undatedStop && (
                <Button disabled={!!pending} onClick={() => setEdit({ id: item.id, stop: true })}>
                  停用
                </Button>
              )}
            </>
          )
        }
        navigation={
          fromList &&
          position >= 0 && (
            <RecordPager
              previous={position > 0}
              next={position < rows.length - 1}
              counter={{ position: position + 1, count: rows.length }}
              onChange={move}
            />
          )
        }
      />
      <Box className="page-scroll" data-page-scroll role="region" tabIndex={0} aria-label="地址类型详情">
        {messages}
        {creating && data.data && !data.data.canMaintain && (
          <Alert severity="warning">没有本次办理权限</Alert>
        )}
        {item && info && (
          <ObjectDetailSheet
            identity={
              <ObjectDetailIdentity
                icon={<HomeWorkOutlined />}
                title={item.name}
                summary={
                  <>
                    {item.code} · 今日状态{' '}
                    <StatusChip
                      label={labels[item.state]}
                      color={item.state === 'active' ? 'success' : 'default'}
                    />
                  </>
                }
              />
            }
            tabs={
              <Tabs
                value={history ? 'history' : 'basic'}
                aria-label="地址类型资料"
                onChange={(_, value) => {
                  if (value === 'history') search.set('tab', 'history');
                  else search.delete('tab');
                  navigate(
                    { pathname: route.pathname, search: search.toString() },
                    { replace: true, state: route.state },
                  );
                }}
              >
                <Tab value="basic" label="基本资料" />
                <Tab value="history" label="变更日志" />
              </Tabs>
            }
            timeline={
              !history && (
                <EffectiveDates
                  dates={info.timeline.map((p) => p.day)}
                  selected={point?.day}
                  onSelect={(day) => setSelected({ id: item.id, day })}
                />
              )
            }
          >
            {history ? (
              <AddressTypeHistory key={item.revision} item={item} refresh={detail.invalidate} />
            ) : (
              <div className="profile-grid">
                <ObjectDetailSection title="基本资料">
                  <ObjectDetailField label="名称">{item.name}</ObjectDetailField>
                  <ObjectDetailField label="编码">{item.code}</ObjectDetailField>
                  <ObjectDetailField label="排序">{item.sortOrder}</ObjectDetailField>
                </ObjectDetailSection>
                <ObjectDetailSection title="生效期间">
                  {point && (
                    <ObjectDetailField label="所选日期状态">
                      {point.day} · {labels[point.state]}
                    </ObjectDetailField>
                  )}
                  <ObjectDetailField label="首次生效日">{item.firstDay}</ObjectDetailField>
                  <ObjectDetailField label="停用生效日">
                    {item.stopDay || (item.undatedStop ? '已停用，原日期未留存' : '未安排')}
                  </ObjectDetailField>
                  {item.confirmedFrom && (
                    <ObjectDetailField label="状态可核实自">{item.confirmedFrom}</ObjectDetailField>
                  )}
                </ObjectDetailSection>
              </div>
            )}
          </ObjectDetailSheet>
        )}
      </Box>
    </Box>
  );
}

function AddressTypeHistory({ item, refresh }: { item: Item; refresh: () => void }) {
  const [revision, setRevision] = useState(item.revision);
  const read = person.useRead(item.id + revision, () => person.addressTypeSource(item.id, revision));
  useEffect(() => {
    if (read.status === 401 || read.status === 403) refresh();
  }, [read.status, refresh]);
  const value = read.data;
  return (
    <ObjectDetailSection title="配置修订">
      {read.error && (
        <Alert severity="error" action={<Button onClick={read.refresh}>重试</Button>}>
          {read.error}
        </Alert>
      )}
      {value && (
        <>
          <ObjectDetailField label="记录时刻">{recordedTime(value.recordedAt)}</ObjectDetailField>
          <ObjectDetailField label="名称">{value.name}</ObjectDetailField>
          <ObjectDetailField label="排序">{value.sortOrder}</ObjectDetailField>
          {value.dated ? (
            <ObjectDetailField label="停用生效日">
              {value.stopDay || (value.undatedStop ? '已停用，原日期未留存' : '未安排')}
            </ObjectDetailField>
          ) : (
            <ObjectDetailField label="当时选用标志">
              {value.selectable ? '可选' : '已停选'}
            </ObjectDetailField>
          )}
          {value.previousRevision && (
            <Button onClick={() => setRevision(value.previousRevision!)}>上一条修订</Button>
          )}
          {revision !== item.revision && (
            <Button onClick={() => setRevision(item.revision)}>返回当前修订</Button>
          )}
        </>
      )}
    </ObjectDetailSection>
  );
}
