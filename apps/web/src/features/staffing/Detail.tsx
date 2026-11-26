import { useEffect, useRef, useState } from 'react';
import { Alert, Box, Button, IconButton, Menu, MenuItem, Stack, Tabs, Tab, Typography } from '@mui/material';
import WorkOutlineIcon from '@mui/icons-material/WorkOutline';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import { Link, useNavigate, useParams } from 'react-router-dom';
import {
  ObjectDetailSheet,
  ObjectDetailIdentity,
  ObjectDetailSection,
  ObjectDetailField,
  ObjectTabPanel,
  RecordControl,
  RecordBackButton,
  RecordPager,
  EffectiveDates,
  HistoryNotice,
  SourceExplanation,
  businessDate,
} from '@chnhcm/ui';
import { ApiError, sessionGuard, type Models } from '@chnhcm/api-client';
import { useMaintenanceSearchParams, usePageTitle } from '../../shared/navigation';
import { OperationSummary } from '../../shared/dated/OperationLog';
import { StaffingEditor, permittedDraft, type StaffingDraft } from './Editor';
import { GradeMemberList } from './GradeMembers';
import { StaffingChangeLog, logKeys, logReturnHref } from './ChangeLog';
import { positionQuotaPath } from './quotaApi';
import {Arrangement} from './Arrangement';
import { ReplenishmentLink } from './Replenishment';
import {StructureControls} from './Hierarchy';
import {structurePath,structureQuery,nodeLabel,useStructure} from './structure';
import * as staff from './api';
export function StaffingDetail({
  kind: k,
  session,
  creating = false,
}: {
  kind: staff.Kind;
  session: staff.Principal;
  creating?: boolean;
}) {
  const { id = '' } = useParams(),
    [params, setParams] = useMaintenanceSearchParams(staff.paths[k]),
    navigate = useNavigate();
  const proposed = params.get('back') ?? staff.paths[k],
    back = (proposed.split('?')[0] === staff.paths[k] || (k==='position' && proposed.split('?')[0]===structurePath)) ? proposed : staff.paths[k];
  const returnToCurrentStructure=()=>{
    const q=new URLSearchParams(fromStructure?back.split('?')[1]:undefined);
    if(structureDate)q.set('businessAt',structureDate);
    q.delete('sourceVersion');q.set('focus',id);
    navigate(structurePath+'?'+q);
  };
  const source = params.get('sourceVersion') ?? '',
    selected = params.get('selectedRecordId') ?? '';
  const structureDate=params.get('structureDate')||undefined;
  const structureVersion=k==='position'?params.get('structureVersion')||undefined:undefined;
  const fromStructure=back.split('?')[0]===structurePath;
  const [arranging,setArranging]=useState(false),[childrenPage,setChildrenPage]=useState(1),[relationRetry,setRelationRetry]=useState(0);
  const [context, setContext] = useState<Models['StaffingContext']>(),
    [data, setData] = useState<staff.Detail>(),
    [error, setError] = useState(''),
    [notice, setNotice] = useState(''),
    [loading, setLoading] = useState(true),
    [retry, setRetry] = useState(0),
    [editing, setEditing] = useState<string>();
  const identity = JSON.stringify([session.instanceId, session.subjectId, k, id, creating, source, selected,structureVersion,structureVersion?structureDate:null]);
  const pendingDraft = useRef<{
    identity: string;
    action: string;
    head?: string;
    record?: string;
    draft: StaffingDraft;
  } | undefined>(undefined);
  const [draft, setDraft] = useState<StaffingDraft>();
  const [nav, setNav] = useState<Models['StaffingNavigation']>(),
    [navError, setNavError] = useState(''),
    [navRetry, setNavRetry] = useState(0);
  const [anchor, setAnchor] = useState<HTMLElement | null>(null),
    [editorActions, setEditorActions] = useState<HTMLDivElement | null>(null),
    more = useRef<HTMLButtonElement>(null);
  const current = data?.selectedRecord,
    actions = data?.capabilities.actions ?? {},
    formActive = creating || !!editing,
    // Match the standard dated page: show the editor while retaining the view tab for cancellation.
    tab = !formActive && params.get('tab') === 'relations' && k==='position' ? 'relations' : !formActive && !structureVersion && params.get('tab')==='history' ? 'history' : 'basic';
  const relations=useStructure({mode:'relation',businessAt:structureDate,sourceVersion:structureVersion,nodeId:id,page:childrenPage},
    k==='position' && !creating && (tab==='relations'||arranging) && (!source || !!structureVersion),relationRetry);
  const openAdjustment=!source && !structureVersion && !!context?.capabilities.actions.adjustSuperior && data && !data.readOnly;
  const startView=useStructure({mode:'nodes',businessAt:structureDate,pageSize:1},
    !!openAdjustment && arranging && relations?.status===404,relationRetry);
  const initialArrangement=relations?.data ?? startView?.data;
  useEffect(()=>setChildrenPage(1),[id,structureDate,structureVersion]);
  usePageTitle(creating ? `新建${staff.labels[k]}` : (current?.name ?? staff.labels[k]));
  useEffect(() => {
    const a = new AbortController(),
      same = sessionGuard();
    if (pendingDraft.current?.identity !== identity) pendingDraft.current = undefined;
    setDraft(undefined);
    setArranging(false);
    setLoading(true);
    setError('');
    setData(undefined);
    setContext(undefined);
    setEditing(undefined);
    void (async () => {
      const ctx = await staff.context(k, a.signal);
      let positionSource=source, record=selected;
      if (structureVersion) {
        const sealed=await structureQuery({mode:'node',businessAt:structureDate,nodeId:id,sourceVersion:structureVersion},a.signal);
        if(!sealed.selected)throw new Error('该历史结构内没有可读的岗位资料');
        positionSource=sealed.selected.positionSource;
        record=sealed.selected.recordId;
      }
      const detail = creating
          ? undefined
          : await staff.detail(
              k,
              id,
              { sourceVersion: positionSource || undefined, selectedRecordId: record || undefined },
              a.signal,
            );
      if (!a.signal.aborted && same()) {
        const pending = pendingDraft.current;
        pendingDraft.current = undefined;
        const caps = detail?.capabilities ?? ctx.capabilities;
        if (
          pending && pending.identity === identity && pending.head === detail?.headSourceVersion &&
          pending.record === detail?.selectedRecord.recordId && !detail?.readOnly && caps.actions[pending.action]
        ) {
          setDraft(permittedDraft(pending.draft, caps.actionFields[pending.action] ?? []));
          if (!creating) setEditing(pending.action);
        }
        setContext(ctx);
        setData(detail);
      }
    })()
      .catch((e) => {
        if (!a.signal.aborted && same()) {
          if (e instanceof ApiError && [401, 403].includes(e.status)) pendingDraft.current = undefined;
          setError(e instanceof Error ? e.message : '资料暂不可读');
        }
      })
      .finally(() => {
        if (!a.signal.aborted && same()) setLoading(false);
      });
    return () => a.abort();
  }, [identity, retry]);
  useEffect(() => {
    setNav(undefined);
    setNavError('');
    if (creating || fromStructure || !params.has('back')) return;
    const a = new AbortController(),
      same = sessionGuard();
    void staff
      .navigation(k, id, staff.queryFrom(new URLSearchParams(back.split('?')[1] ?? '')), a.signal)
      .then((v) => {
        if (!a.signal.aborted && same()) setNav(v);
      })
      .catch(() => {
        if (!a.signal.aborted && same()) setNavError('对象导航暂不可用，当前详情仍可查看。');
      });
    return () => a.abort();
  }, [k, id, back, creating, navRetry]);
  useEffect(() => {
    const clear = () => {
      pendingDraft.current = undefined;
      setDraft(undefined);
      setData(undefined);
      setContext(undefined);
      setEditing(undefined);
    };
    window.addEventListener('chnhcm-session-invalidated', clear);
    return () => window.removeEventListener('chnhcm-session-invalidated', clear);
  }, []);
  useEffect(() => {
    if (data && tab === 'history' && !data.capabilities.actions['history.read']) {
      const next = new URLSearchParams(params);
      next.set('tab', 'basic');
      logKeys.forEach((key) => next.delete(key));
      setParams(next, { replace: true });
    }
  }, [data, tab]);
  const close = () => {
    setDraft(undefined);
    setEditing(undefined);
    requestAnimationFrame(() => more.current?.focus());
  };
  const saved = (r: staff.Result) => {
    setDraft(undefined);
    setNotice(r.outcome === 'no_change' ? '输入没有资料变化。' : '已保存。');
    setEditing(undefined);
    if (creating) {
      navigate(
        `${staff.paths[k]}/${r.objectId}?back=${encodeURIComponent(back)}&selectedRecordId=${r.selectedRecordId ?? ''}`,
        { replace: true },
      );
    } else {
      const next = new URLSearchParams(params);
      next.delete('sourceVersion');
      logKeys.forEach((key) => next.delete(key));
      next.set('tab', 'basic');
      if (r.selectedRecordId) next.set('selectedRecordId', r.selectedRecordId);
      setParams(next, { replace: true });
      setRetry((n) => n + 1);
    }
  };
  const operations = [
    'CORRECT',
    ...(!actions.CORRECT && actions.REDATE ? ['REDATE'] : []),
    'DELETE',
    'DEACTIVATE',
    'REACTIVATE',
    'INSERT_LIFECYCLE_PAIR',
    'MAINTAIN_LIFECYCLE_PAIR',
    'DELETE_LIFECYCLE_PAIR',
  ].filter(
    (a) =>
      actions[a] &&
      (a !== 'DELETE' || current?.recordId !== data?.timeline[0].recordId) &&
      (!['DEACTIVATE', 'REACTIVATE'].includes(a) || data?.lifecycleContext?.operationType === a) &&
      (a !== 'INSERT_LIFECYCLE_PAIR' || data?.pairInsertion) &&
      (!a.includes('LIFECYCLE_PAIR') || a === 'INSERT_LIFECYCLE_PAIR' || data?.pairContext),
  );
  const move = (dir: 'previous' | 'next') => {
    const target = dir === 'previous' ? nav?.previousId : nav?.nextId;
    if (target && nav) {
      const q = new URLSearchParams(back.split('?')[1] ?? '');
      q.set('selected', target);
      q.set(
        'page',
        String(Math.max(1, Math.ceil((nav.position + (dir === 'next' ? 1 : -1)) / Number(q.get('pageSize') ?? 20)))),
      );
      navigate(`${staff.paths[k]}/${target}?back=${encodeURIComponent(staff.paths[k] + '?' + q)}`);
    }
  };
  const returnLog = logReturnHref(params.get('logReturn'), `${staff.paths[k]}/${id}`);
  const read = data?.capabilities.readFields ?? [];
  const edit =
    context && (creating ? context.capabilities.actions.CREATE : data) && formActive ? (
      <StaffingEditor
        key={`${id}:${editing ?? 'CREATE'}:${data?.headSourceVersion ?? ''}`}
        kind={k}
        draft={draft}
        session={session}
        action={creating ? 'CREATE' : editing!}
        detail={data}
        context={context}
        actionsContainer={editorActions}
        onClose={() => (creating ? navigate(back) : close())}
        onSaved={saved}
        onGradesRestricted={() => {
          // Preserve the authorized scalar draft, but also clear the parent's
          // snapshots so cancellation or a read-only form cannot reveal old members.
          const restrict = (record: staff.RecordView): staff.RecordView => {
            const { gradeIds: _ids, gradeMembers: _members, ...visible } = record;
            return { ...visible, gradesRestricted: true };
          };
          setData(value => value && ({ ...value, selectedRecord: restrict(value.selectedRecord),
            timeline: value.timeline.map(restrict), pairContext: value.pairContext && {
              deactivation: restrict(value.pairContext.deactivation), reactivation: restrict(value.pairContext.reactivation),
            } }));
        }}
        onAccessChanged={(cause, input) => {
          pendingDraft.current = input ? {
            identity, action: creating ? 'CREATE' : editing!, head: data?.headSourceVersion,
            record: data?.selectedRecord.recordId, draft: input,
          } : undefined;
          setDraft(undefined);
          setData(undefined);
          setContext(undefined);
          setEditing(undefined);
          setNotice(cause === 'denied' ? '权限已变化，已重新读取当前可见资料。' : '');
          setRetry((n) => n + 1);
        }}
      />
    ) : null;
  if (arranging && openAdjustment && initialArrangement?.canPublish) {
    return <Arrangement key={id} session={session} id={id} initial={initialArrangement}
      onClose={()=>{setArranging(false);requestAnimationFrame(()=>more.current?.focus())}}
      onSaved={(day,changed)=>{
        setArranging(false);setRelationRetry(n=>n+1);
        const next=new URLSearchParams(params);next.set('tab','relations');next.set('structureDate',day);setParams(next,{replace:true});
        setNotice(changed?'已保存，'+day+' 起生效。':'岗位关系没有变化。');
      }}/>
  }
  return (
    <Box className="standard-page object-page" aria-busy={loading || (tab==='relations' && !relations)}>
      <RecordControl
        identity={
          <>
            <RecordBackButton component={Link} label={fromStructure?"返回岗位上下级":"返回列表"} to={back} />
            {data?.code && <Typography>{data.code}</Typography>}
            {!creating && !data?.readOnly && (!!operations.length || !!openAdjustment) && (
              <IconButton
                ref={more}
                aria-label="更多操作"
                aria-haspopup="menu"
                aria-expanded={!!anchor}
                disabled={formActive}
                onClick={(e) => setAnchor(e.currentTarget)}
              >
                <MoreVertIcon />
              </IconButton>
            )}
          </>
        }
        actions={
          formActive ? (
            <Box ref={setEditorActions} sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }} />
          ) : !data?.readOnly && actions.ADD && data?.additionContext ? (
            <Button variant="contained" disabled={!data.additionContext.available} onClick={() => setEditing('ADD')}>
              {data.additionContext.mode === 'INSERT' ? '插入记录' : '新增记录'}
            </Button>
          ) : null
        }
        navigation={
          !creating && nav ? (
            <RecordPager
              previous={!!nav.previousId}
              next={!!nav.nextId}
              counter={nav.position ? { position: nav.position, count: nav.totalCount } : undefined}
              disabled={formActive}
              onChange={move}
            />
          ) : null
        }
      />
      <Menu anchorEl={anchor} open={!!anchor} onClose={() => setAnchor(null)}>
        {openAdjustment && <MenuItem onClick={()=>{
          setAnchor(null);setArranging(true);
        }}>调整上级</MenuItem>}
        {operations.map((op) => (
          <MenuItem
            key={op}
            onClick={() => {
              setAnchor(null);
              setEditing(op);
            }}
          >
            {staff.operationLabel[op]}
          </MenuItem>
        ))}
      </Menu>
      {structureVersion && <HistoryNotice label="返回当前结构" onReturn={returnToCurrentStructure}/>}
      {data?.readOnly && !structureVersion && (
        <HistoryNotice
          onReturn={() => {
            const p = new URLSearchParams(params);
            p.delete('sourceVersion');
            p.delete('selectedRecordId');
            logKeys.forEach((key) => p.delete(key));
            p.set('tab', 'basic');
            setParams(p);
          }}
        />
      )}
      {returnLog && (
        <Box>
          <Button component={Link} to={returnLog}>
            返回变更日志
          </Button>
        </Box>
      )}
      <Box className="page-scroll" data-page-scroll tabIndex={0} role="region" aria-label={`${staff.labels[k]}资料`}>
        <Stack spacing={1}>
          {notice && (
            <Alert severity="success" onClose={() => setNotice('')}>
              {notice}
            </Alert>
          )}
          {error && (
            <Alert severity="error" action={<Button onClick={() => setRetry((n) => n + 1)}>重试</Button>}>
              {error}
            </Alert>
          )}
          {arranging && relations?.error && <Alert severity="warning" action={<Button onClick={()=>{setArranging(false);const q=new URLSearchParams(params);q.set('tab','relations');setParams(q)}}>查看岗位关系</Button>}>{relations.error}</Alert>}
          {navError && (
            <Alert severity="info" action={<Button onClick={() => setNavRetry((n) => n + 1)}>重试导航</Button>}>
              {navError}
            </Alert>
          )}
          {nav && nav.position === 0 && <Alert severity="info">当前资料不在原查询结果中。</Alert>}
          {loading && <Typography role="status">正在读取…</Typography>}
          {data?.selectionMissing && <Alert severity="info">原定位记录已不在当前时间线，已显示当前可读记录。</Alert>}
          {data?.selectedRecordIsFuture && <Alert severity="info">所选记录尚未生效。</Alert>}
        </Stack>
        {creating && context && !context.capabilities.actions.CREATE && (
          <Alert severity="warning">当前没有新建权限，请返回列表查看获权资料。</Alert>
        )}
        {context && (creating ? context.capabilities.actions.CREATE : data) && (
          <ObjectDetailSheet
            identity={
              <ObjectDetailIdentity
                icon={<WorkOutlineIcon />}
                title={creating ? `新建${staff.labels[k]}` : (current?.name ?? staff.labels[k])}
              />
            }
            tabs={
              !creating && (
                <Tabs
                  value={tab}
                  variant="scrollable"
                  allowScrollButtonsMobile
                  sx={k === 'position' ? {
                    '@media (max-width:599px)': {
                      '&& .MuiTabs-scrollButtons': { display: 'none' },
                      '&& .MuiTab-root': { flex: '1 1 0', minWidth: 0, paddingInline: '4px' },
                    },
                  } : undefined}
                  onChange={(_, value) => {
                    if (value === 'quota') {
                      const dest = new URLSearchParams({ back });
                      if (selected) dest.set('positionRecordId', selected);
                      navigate(positionQuotaPath(id, dest));
                      return;
                    }
                    const p = new URLSearchParams(params);
                    p.set('tab', value);
                    setParams(p);
                  }}
                >
                  <Tab value="basic" id="staffing-basic-tab" aria-controls="staffing-basic-panel" label="基本资料" />
                  {data?.canReadQuota && (
                    <Tab value="quota" label="岗位编制" disabled={formActive} />
                  )}
                  {k==='position' && (!data?.readOnly || structureVersion) && context?.capabilities.actions.positionRelations && <Tab value="relations" id="staffing-relations-tab" aria-controls="staffing-relations-panel" label="岗位关系" disabled={formActive}/>}
                  {!structureVersion && actions['history.read'] && (
                    <Tab
                      value="history"
                      id="staffing-history-tab"
                      aria-controls="staffing-history-panel"
                      label="变更日志"
                      disabled={formActive}
                    />
                  )}
                </Tabs>
              )
            }
            timeline={
              !formActive &&
              tab === 'basic' && !structureVersion &&
              data &&
              read.includes('effectiveDate') && (
                <EffectiveDates
                  dates={data.timeline.flatMap((r) => (r.effectiveDate ? [r.effectiveDate] : []))}
                  selected={current?.effectiveDate}
                  onSelect={(date) => {
                    const r = data.timeline.find((r) => r.effectiveDate === date);
                    if (r) {
                      const p = new URLSearchParams(params);
                      p.set('selectedRecordId', r.recordId);
                      setParams(p);
                    }
                  }}
                />
              )
            }
          >
            <ObjectTabPanel prefix="staffing" value="basic" selected={creating ? 'basic' : tab}>
              {edit}
              {!formActive && current && (
                <>
                  <Box className="object-detail-columns">
                    <ObjectDetailSection title="基本资料">
                      {data?.code && <ObjectDetailField label="编码">{data.code}</ObjectDetailField>}
                      {staff
                        .dataFields(k)
                        .filter((f) => read.includes(f))
                        .map((f) => (
                          <ObjectDetailField key={f} label={staff.fieldLabels[f]}>
                            <Typography sx={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>
                              {staff.isPositionField(f) ? staff.positionName(current,f) ?? '未指定' : staff.isJobClassification(f)
                                ? (staff.classificationName(current, f) ?? '受限')
                                : String(current[f as keyof staff.RecordView] ?? '未提供')}
                            </Typography>
                          </ObjectDetailField>
                        ))}
                      {(k === 'job' || k === 'gradeladder') && read.includes('gradeIds') && <ObjectDetailField label={k === 'job' ? '可用职级' : '职级成员（低至高）'}><GradeMemberList members={current.gradeMembers} restricted={current.gradesRestricted} ordered={k === 'gradeladder'} /></ObjectDetailField>}
                    </ObjectDetailSection>
                    <ObjectDetailSection title="记录信息">
                      {current.effectiveDate && (
                        <>
                          <ObjectDetailField label="生效日期">{businessDate(current.effectiveDate)}</ObjectDetailField>
                          <ObjectDetailField label="适用期间">
                            {current.endInclusive
                              ? `${businessDate(current.effectiveDate)} 至 ${businessDate(current.endInclusive)}`
                              : `自 ${businessDate(current.effectiveDate)} 起`}
                          </ObjectDetailField>
                        </>
                      )}
                      {current.state && (
                        <ObjectDetailField label="状态">{staff.statusLabel(current.state)}</ObjectDetailField>
                      )}
                      {current.operationType && (
                        <ObjectDetailField label="操作类型">
                          {staff.operationLabel[current.operationType]}
                        </ObjectDetailField>
                      )}
                      {current.reasonLabel && (
                        <ObjectDetailField label="操作原因">{current.reasonLabel}</ObjectDetailField>
                      )}
                    </ObjectDetailSection>
                  </Box>
                  {k==='position'&&!data?.readOnly&&!source&&!structureVersion&&<ReplenishmentLink id={id} day={current.effectiveDate ?? ''} back={staff.paths.position+'/'+id+'?'+params.toString()}/>}
                  {data?.readOnly && current.positionAdoptions && <SourceExplanation>{Object.entries(current.positionAdoptions).map(([field,ref]) => <Typography key={field} sx={{overflowWrap:'anywhere'}}>{staff.fieldLabels[field]}：{ref.name}；采用日期：{businessDate(ref.businessAt)}；来源：{ref.sourceVersion}；记录：{ref.recordId}；修订：{ref.revision}</Typography>)}</SourceExplanation>}
                  {data?.readOnly && (current.adoption || current.jobFunctionAdoption) && (
                    <SourceExplanation>
                      {([['职务系列', current.adoption], ['职类', current.jobFunctionAdoption]] as const).map(([label, ref]) =>
                        ref && <Box key={label} sx={{ overflowWrap: 'anywhere' }}>
                          <Typography>{label}采用日期：{businessDate(ref.date)}</Typography>
                          <Typography>{label === '职务系列' ? '系列' : label}来源：{ref.sourceVersion}</Typography>
                          <Typography>采用记录：{ref.recordId}；修订：{ref.revision}</Typography>
                        </Box>
                      )}
                    </SourceExplanation>
                  )}
                  {current.operationSummary && (
                    <OperationSummary
                      summary={{
                        recordedAt: current.operationSummary.recordedAt,
                        operator: {
                          subjectId: current.operationSummary.subjectId ?? '',
                          displayName: current.operationSummary.displayName,
                          login: current.operationSummary.login,
                          kind: current.operationSummary.identityKind ?? 'human',
                          identityEvidence: 'captured',
                        },
                      }}
                    />
                  )}
                </>
              )}
            </ObjectTabPanel>
            {tab==='relations' && <ObjectTabPanel prefix="staffing" value="relations" selected={tab}>
              <StructureControls key={JSON.stringify([id,structureVersion])} day={structureDate||context?.currentDate||''} change={day=>{
                const q=new URLSearchParams(params);
                q.set('structureDate',day);
                setParams(q)
              }}/>
              {relations?.error && <Alert severity="warning" action={<Button onClick={()=>setRelationRetry(n=>n+1)}>重试</Button>}>{relations.error}</Alert>}
              {relations?.data && <ObjectDetailSection title="岗位关系">
                <ObjectDetailField label="上级岗位">{relations.data.parent?nodeLabel(relations.data.parent):'无上级'}</ObjectDetailField>
                {relations.data.endInclusive && <ObjectDetailField label="适用至">{relations.data.endInclusive}</ObjectDetailField>}
                {relations.data.nextStart && <ObjectDetailField label="下次安排">{relations.data.nextStart} 起</ObjectDetailField>}
                <ObjectDetailField label="直接下级">{relations.data.nodes.length ? relations.data.nodes.map(n=><Typography key={n.id}>{nodeLabel(n)}</Typography>) : '无直接下级'}</ObjectDetailField>
                {(childrenPage>1||relations.data.more) && <Stack direction="row">
                  <Button disabled={childrenPage===1} onClick={()=>setChildrenPage(p=>p-1)}>上一页</Button>
                  <Button disabled={!relations.data.more} onClick={()=>setChildrenPage(p=>p+1)}>下一页</Button>
                </Stack>}
              </ObjectDetailSection>}
            </ObjectTabPanel>}
            {actions['history.read'] && tab === 'history' && (
              <ObjectTabPanel prefix="staffing" value="history" selected={tab}>
                <StaffingChangeLog
                  kind={k}
                  id={id}
                  onReadFields={(fields) => {
                    if (fields.length !== read.length || fields.some((f) => !read.includes(f))) {
                      setData(undefined);
                      setRetry((n) => n + 1);
                    }
                  }}
                  onDenied={() => {
                    setData(undefined);
                    setRetry((n) => n + 1);
                  }}
                />
              </ObjectTabPanel>
            )}
          </ObjectDetailSheet>
        )}
      </Box>
    </Box>
  );
}
