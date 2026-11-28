import {OfferList, OfferDetail, offerAccess,offerPath} from '../features/recruiting/Offers';
import { AssignmentPolicyPage, PurposePage, policyPath } from '../features/readiness/RulesPages';
import { SecondaryAssignmentPage } from '../features/hire/SecondaryAssignmentPage';
import { AssignmentPage } from '../features/hire/AssignmentPage';
import {EmploymentRegister,registerPath} from '../features/hire/EmploymentRegister';
import {RegisterActionPage} from '../features/hire/RegisterActionPage';
import { HirePage } from "../features/hire/HirePage";
import { AddressTypes } from '../features/personnel/AddressTypes';
import { PersonList } from '../features/personnel/List';
import { PersonDetail } from "../features/personnel/Detail";
import { PersonRecovery } from "../features/personnel/Submission";
import * as person from "../features/personnel/api";
import { StaffingList } from '../features/staffing/List';
import { PositionReplenishment } from '../features/staffing/Replenishment';
import { StaffingDetail } from '../features/staffing/Detail';
import { HierarchyPage, hierarchyPath } from '../features/staffing/Hierarchy';
import { StaffingRecovery } from '../features/staffing/useSubmission';
import { kinds as staffingKinds, paths as staffingPaths } from '../features/staffing/api';
import { LegalEntityList, NewLegalEntity, LegalEntityDetail, LegalEntityRecovery, LEGAL_ENTITY_PATH } from '../features/organization/LegalEntities';
import { LocationList, NewLocation, LocationDetail, LocationRecovery, LOCATION_PATH } from '../features/organization/Locations';
import { useCallback, useEffect, useRef, useState } from "react";
import {
  Alert,
  Button,
  Paper,
  Stack,
  TextField,
  Typography,
  useMediaQuery,
} from "@mui/material";
import { Link, Navigate, Route, Routes, useNavigate, useLocation } from "react-router-dom";
import { AppShell, AppNavigation, type NavigationApp } from "@chnhcm/ui";
import { api, ApiError, unwrap, sessionChanged, sessionChangeStarted, sessionGuard, isCurrentSessionResponse, type Models } from "@chnhcm/api-client";
import {
  BusinessUnitList,
  BusinessUnitDetail,
  NewBusinessUnit,
  Recovery,
  BU_PATH,
} from "../features/organization/BusinessUnits";

import { GeoAreaList, GeoAreaDetail, GeoAreaRecovery, GEOAREA_PATH } from '../features/organization/GeoAreas';
import { EnterpriseEntry, EnterpriseDetail, EnterpriseRecovery, ENTERPRISE_PATH } from '../features/organization/Enterprise';
import { CountryList, CountryDetail, CountryRecovery, COUNTRY_PATH } from '../features/organization/Countries';
import { DepartmentList, NewDepartment, DepartmentDetail, DepartmentRecovery, DEPARTMENT_PATH } from '../features/organization/Departments';

import { SessionRecoveryContext } from '../shared/sessionRecovery';

import { usePageScrollRestoration } from '../shared/navigation';

import {QuotaList} from '../features/staffing/QuotaList';
import {QuotaDepartment} from '../features/staffing/QuotaDepartment';
import {QuotaDetail} from '../features/staffing/QuotaDetail';
import {QuotaRecovery} from '../features/staffing/QuotaRecovery';
import {quotaPath} from '../features/staffing/quotaApi';

const departmentApp: NavigationApp = { id: "departments", label: "部门", path: DEPARTMENT_PATH, sections: [] };
const baseSettings: NavigationApp = { id: "settings", label: "设置", sections: [
  { label: "组织配置", items: [{ label: "业务单元", path: BU_PATH }, { label: "企业", path: ENTERPRISE_PATH }, { label: "法律实体", path: LEGAL_ENTITY_PATH }, { label: "地点", path: LOCATION_PATH }] },
  { label: "职务配置", items: [{ label: "职类", path: staffingPaths.jobfunction }, { label: "职务系列", path: staffingPaths.jobfamily }, { label: "职务", path: staffingPaths.job }, { label: "职级", path: staffingPaths.grade }, { label: "职级阶梯", path: staffingPaths.gradeladder }, { label: "岗位", path: staffingPaths.position }, { label: "部门编制分配", path: quotaPath }, {label:"任职与占编规则",path:policyPath}] },
  { label: "基础资料", items: [{ label: "国家/地区", path: COUNTRY_PATH }, { label: "行政区划", path: GEOAREA_PATH }] },
] };



export function App() {
  usePageScrollRestoration();
  const navigate = useNavigate();
  const location = useLocation();
  const departmentPage = location.pathname === DEPARTMENT_PATH || location.pathname.startsWith(`${DEPARTMENT_PATH}/`);
  const onRegister=location.pathname===registerPath || location.pathname.startsWith(registerPath+"/");
  const onOffer=location.pathname===offerPath||location.pathname.startsWith(offerPath+"/");
  const personPage =onOffer ||onRegister ||  location.pathname === person.path || location.pathname.startsWith(person.path + "/");
  const desktop = useMediaQuery("(min-width:900px)");
  const [session, setSession] = useState<Models["Principal"] | null>(null);
  const personAccess = person.useRead(session ? session.instanceId + session.subjectId : undefined, person.entry);
  const registerAccess=person.useRead(session ? session.instanceId+session.subjectId : undefined,async()=>unwrap(await api.GET('/api/v1/employment-actions/context')));
  const recruiting=person.useRead(session?session.instanceId+session.subjectId:undefined,()=>offerAccess());
  const canRegister=!!registerAccess.data?.canList;
  const canArchive=!!(personAccess.data?.canRead||personAccess.data?.canRegister);
  const sections:NavigationApp['sections']=[];
  if(canArchive)sections.push({label:'人员',direct:true,items:[{label:'人员',path:person.path}]});
  if(canRegister)sections.push({label:'人事事务',items:[{label:'办理记录',path:registerPath}]});
  if(recruiting.data?.canRead)sections.push({label:'招聘',items:[{label:'招聘录用',path:offerPath}]});
  const personnelApp:NavigationApp={id:'personnel',label:'人员',path:sections.length?undefined:person.path,sections};
  const onAddressTypes = location.pathname === person.addressTypePath || location.pathname.startsWith(person.addressTypePath + '/');
  const settingsApp = personAccess.data?.canReadAddressTypes ? { ...baseSettings, sections: [...baseSettings.sections, { label: '人员配置', items: [{ label: '地址类型', path: person.addressTypePath }] }] } : baseSettings;
  const activeApp = personPage ? personnelApp : departmentPage ? departmentApp : settingsApp;
  const activeSection = onAddressTypes ? '人员配置' : onOffer ? '招聘' : onRegister ? '人事事务' : personPage ? '人员' : location.pathname.startsWith("/settings/staffing/") ? "职务配置" : departmentPage ? undefined : (location.pathname.startsWith(GEOAREA_PATH) || location.pathname.startsWith(COUNTRY_PATH)) ? "基础资料" : "组织配置";
  const apps = canArchive || canRegister || recruiting.data?.canRead ? [departmentApp, personnelApp, settingsApp] : [departmentApp, settingsApp];
  useEffect(() => {
    if (!session) document.title = "登录 · CHNHCM";
    else if (!onAddressTypes && !personPage && !location.pathname.startsWith("/settings/staffing/") && !location.pathname.startsWith(LEGAL_ENTITY_PATH) && !location.pathname.startsWith(LOCATION_PATH) && !location.pathname.startsWith(BU_PATH) && !location.pathname.startsWith(DEPARTMENT_PATH) && !location.pathname.startsWith(COUNTRY_PATH) && !location.pathname.startsWith(GEOAREA_PATH) && !location.pathname.startsWith(ENTERPRISE_PATH) && location.pathname !== "/") document.title = "页面不存在 · CHNHCM";
  }, [session, location.pathname]);
  const sessionRef = useRef(session); sessionRef.current = session;
  const [loading, setLoading] = useState(true);
  const [login, setLogin] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [sessionError, setSessionError] = useState("");
  const refreshEpoch = useRef(0);
  const refresh = useCallback(async () => {
    const same = sessionGuard(), own = ++refreshEpoch.current;
    const current = () => own === refreshEpoch.current && same();
    setLoading(true);
    try {
      const result = await api.GET("/api/v1/session");
      if (current()) {
        const principal = unwrap(result);
        const previous = sessionRef.current;
        if (previous && (previous.instanceId !== principal.instanceId || previous.subjectId !== principal.subjectId)) sessionChanged();
        setSession(principal);
        setSessionError("");
        setLoading(false);
      }
    } catch {
      if (current()) setSessionError("暂时无法确认登录状态，请重试。");
    } finally {
      if (current()) setLoading(false);
    }
  }, []);
  useEffect(() => {
    void refresh();
    const lost = () => {
      sessionChanged();
      setSession(null);
      setSessionError("");
      setLoading(false);
    };
    const changedElsewhere = () => {
      setSession(null);
      setError('');
      void refresh();
    };
    window.addEventListener("chnhcm-auth-lost", lost);
    const validationFailed = () => { void refresh(); };
    window.addEventListener("chnhcm-request-validation-failed", validationFailed);
    window.addEventListener('chnhcm-session-changed-elsewhere', changedElsewhere);
    return () => {
      window.removeEventListener("chnhcm-auth-lost", lost);
      window.removeEventListener("chnhcm-request-validation-failed", validationFailed);
      window.removeEventListener('chnhcm-session-changed-elsewhere', changedElsewhere);
    };
  }, [refresh]);
  const enter = async (event: React.FormEvent) => {
    event.preventDefault();
    setError("");
    setLoading(true);
    try {
      const principal = unwrap(
        await api.POST("/api/v1/session/login", {
          body: { login, password },
        }),
      );
      sessionChanged(true);
      setSession(principal);
      setPassword("");
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "登录失败");
    } finally {
      setLoading(false);
    }
  };
  const performExit = async () => {
    if (!session) return;
    // Stop current async consumers when the authorized logout starts, even if
    // its response is delayed. The existing draft-exit gate runs before this.
    // The same invalidation boundary applies to other tabs sharing the cookie.
    sessionChangeStarted();
    const current = sessionGuard();
    setError("");
    try {
      const result = await api.DELETE("/api/v1/session", {
        params: { header: { "X-CSRF-Token": session.csrf } },
      });
      if (!isCurrentSessionResponse(result.response)) return;
      if (result.response.ok || result.response.status === 401) {
        sessionChanged(true);
        setSession(null);
      } else setError("退出未完成，请重试");
    } catch {
      if (current()) setError("退出结果尚未确认，请重试");
    }
  };
  const exit = () => {
    if (
      window.dispatchEvent(
        new CustomEvent("chnhcm-before-logout", {
          cancelable: true,
          detail: { proceed: () => void performExit() },
        }),
      )
    )
      void performExit();
  };
  const invalidHierarchyLink = new URLSearchParams(location.search).has('purpose');
  const missingPage = (
    <Stack spacing={2}>
      <Typography component="h1">页面不存在</Typography>
      <Button component={Link} to={BU_PATH}>返回业务单元</Button>
    </Stack>
  );
  return (
    <SessionRecoveryContext.Provider value={Boolean(session && (loading || sessionError))}>
    <AppShell
      navigation={session ? <AppNavigation apps={apps} activeApp={activeApp} activeSection={activeSection} activePath={onAddressTypes?person.addressTypePath:onOffer?offerPath:onRegister?registerPath:personPage?person.path:location.pathname===policyPath?policyPath:location.pathname.startsWith(quotaPath)?quotaPath:location.pathname===hierarchyPath||location.pathname.startsWith(staffingPaths.position)?staffingPaths.position:location.pathname.startsWith(staffingPaths.gradeladder)?staffingPaths.gradeladder:location.pathname.startsWith(staffingPaths.grade)?staffingPaths.grade:location.pathname.startsWith(staffingPaths.jobfunction)?staffingPaths.jobfunction:location.pathname.startsWith(staffingPaths.jobfamily)?staffingPaths.jobfamily:location.pathname.startsWith(staffingPaths.job)?staffingPaths.job:location.pathname.startsWith(LEGAL_ENTITY_PATH)?LEGAL_ENTITY_PATH:location.pathname.startsWith(ENTERPRISE_PATH)?ENTERPRISE_PATH:location.pathname.startsWith(LOCATION_PATH)?LOCATION_PATH:location.pathname.startsWith(GEOAREA_PATH)?GEOAREA_PATH:location.pathname.startsWith(COUNTRY_PATH) ? COUNTRY_PATH : departmentPage ? DEPARTMENT_PATH : BU_PATH} desktop={desktop} go={navigate} /> : undefined}
      account={session && <><Typography className="app-account-name">{session.login}</Typography><Button color="inherit" onClick={() => void exit()}>退出</Button></>}
      notices={session && <>{loading && <Alert severity="info">正在恢复登录状态…</Alert>}{sessionError && <Alert severity="error" action={<Button disabled={loading} onClick={() => void refresh()}>重试</Button>}>{sessionError}</Alert>}{error && <Alert severity="error">{error}</Alert>}<QuotaRecovery key={session.instanceId+session.subjectId} session={session}/><StaffingRecovery session={session}/><Recovery key={`recovery:${session.instanceId}:${session.subjectId}`} session={session} /><GeoAreaRecovery key={`geoarea-recovery:${session.instanceId}:${session.subjectId}`} session={session} /><LegalEntityRecovery key={`legalentity-recovery:${session.instanceId}:${session.subjectId}`} session={session} /><EnterpriseRecovery key={`enterprise-recovery:${session.instanceId}:${session.subjectId}`} session={session} /><LocationRecovery key={`location-recovery:${session.instanceId}:${session.subjectId}`} session={session} /><CountryRecovery key={`country-recovery:${session.instanceId}:${session.subjectId}`} session={session} /><PersonRecovery key={`person-recovery:${session.instanceId}:${session.subjectId}`} session={session} /><DepartmentRecovery key={`department-recovery:${session.instanceId}:${session.subjectId}`} session={session} /></>}
    >
      {sessionError && !session ? (
        <Alert
          severity="error"
          action={<Button disabled={loading} onClick={() => void refresh()}>重试</Button>}
        >
          {sessionError}
        </Alert>
      ) : !session ? (
        <Paper
          variant="outlined"
          className="login-panel"
        >
          <Stack component="form" spacing={2} onSubmit={enter}>
            <Typography component="h1" variant="h5">
              登录
            </Typography>
            <TextField
              label="登录名"
              autoComplete="username"
              value={login}
              onChange={(e) => setLogin(e.target.value)}
              required
            />
            <TextField
              label="密码"
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
            {error && <Alert severity="error">{error}</Alert>}
            <Button type="submit" variant="contained" disabled={loading}>
              {loading ? "正在验证…" : "登录"}
            </Button>
          </Stack>
        </Paper>
      ) : (
        <>
          <Routes key={`routes:${session.instanceId}:${session.subjectId}`}>
            <Route path={offerPath} element={<OfferList session={session}/>}/>
            <Route path={offerPath+'/new'} element={<OfferDetail key={location.pathname+location.search} creating session={session}/>}/>
            <Route path={offerPath+'/:id'} element={<OfferDetail key={location.pathname} session={session}/>}/>
            <Route path={registerPath} element={<EmploymentRegister session={session}/>}/>
            <Route path={registerPath+'/:actionId'} element={<RegisterActionPage key={location.pathname} session={session}/>}/>
            <Route path={registerPath+'/:actionId/cancel'} element={<RegisterActionPage key={location.pathname} session={session} cancel/>}/>
            <Route path={person.path+"/:id/assignments/:assignment/additional"} element={<SecondaryAssignmentPage key={location.pathname} session={session}/>}/>
            <Route path={person.path+"/:id/assignments/:assignment"} element={<AssignmentPage session={session}/>}/>
            <Route path={person.path} element={<PersonList session={session} />}/>
            <Route path={person.path+"/new"} element={<PersonDetail session={session} creating />}/>
            <Route path={person.path+"/:id/employment"} element={<HirePage session={session}/>}/>
            <Route path={person.path+"/:id"} element={<PersonDetail session={session} />}/>
            <Route path={staffingPaths.position+'/:id/replenishment'} element={<PositionReplenishment session={session}/>}/><Route path={quotaPath} element={<QuotaList/>}/><Route path={quotaPath+'/departments/:id'} element={<QuotaDepartment session={session}/>}/><Route path={staffingPaths.position+'/:id/quota'} element={<QuotaDetail session={session}/>}/>
            <Route path={hierarchyPath} element={invalidHierarchyLink ? missingPage : <HierarchyPage session={session}/>}/>
            {staffingKinds.map(kind=><Route key={kind} path={staffingPaths[kind]} element={<StaffingList key={kind} kind={kind} session={session}/>}/>)}
            {staffingKinds.map(kind=><Route key={`${kind}-new`} path={`${staffingPaths[kind]}/new`} element={<StaffingDetail key={`${kind}-new`} kind={kind} session={session} creating/>}/>)}
            {staffingKinds.map(kind=><Route key={`${kind}-detail`} path={`${staffingPaths[kind]}/:id`} element={kind === "position" && invalidHierarchyLink ? missingPage : <StaffingDetail key={kind} kind={kind} session={session}/>}/>)}
            <Route path="/" element={<Navigate to={BU_PATH} replace />} />
            <Route path={person.addressTypePath + "/:typeId?"} element={<AddressTypes key={session.instanceId+session.subjectId} session={session}/>} />
            <Route path={BU_PATH} element={<BusinessUnitList />} />
            <Route path={GEOAREA_PATH} element={<GeoAreaList />} /><Route path={`${GEOAREA_PATH}/:id`} element={<GeoAreaDetail session={session} />} /><Route path={ENTERPRISE_PATH} element={<EnterpriseEntry />} /><Route path={`${ENTERPRISE_PATH}/:id`} element={<EnterpriseDetail session={session} />} /><Route path={COUNTRY_PATH} element={<CountryList />} />
            <Route path={`${COUNTRY_PATH}/:id`} element={<CountryDetail session={session} />} />
            <Route path={policyPath} element={<AssignmentPolicyPage session={session}/>}/><Route path={`${LEGAL_ENTITY_PATH}/:id/purposes`} element={<PurposePage session={session}/>}/><Route path={LEGAL_ENTITY_PATH} element={<LegalEntityList/>}/><Route path={`${LEGAL_ENTITY_PATH}/new`} element={<NewLegalEntity session={session}/>}/><Route path={`${LEGAL_ENTITY_PATH}/:id`} element={<LegalEntityDetail session={session}/>}/><Route path={LOCATION_PATH} element={<LocationList/>}/><Route path={`${LOCATION_PATH}/new`} element={<NewLocation session={session}/>}/><Route path={`${LOCATION_PATH}/:id`} element={<LocationDetail session={session}/>}/><Route path={DEPARTMENT_PATH} element={<DepartmentList />} />
            <Route path={`${DEPARTMENT_PATH}/new`} element={<NewDepartment session={session} />} />
            <Route path={`${DEPARTMENT_PATH}/:id`} element={<DepartmentDetail session={session} />} />
            <Route
              path={`${BU_PATH}/new`}
              element={<NewBusinessUnit session={session} />}
            />
            <Route path={`${BU_PATH}/:id`} element={<BusinessUnitDetail session={session} />} />
            <Route path="*" element={missingPage}/>
          </Routes>
        </>
      )}
    </AppShell>
    </SessionRecoveryContext.Provider>
  );
}
