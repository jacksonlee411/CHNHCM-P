import { api, unwrap, type Models } from '@chnhcm/api-client';

export type Area = 'accounts' | 'roles' | 'grants';
export type View = Models['AccessManagementView'];
export type Definition = Models['AccessDefinition'];
export type Command = Omit<
  Models['AccessManagementCommand'],
  'instanceId' | 'requestId'
>;
export type Grant = Models['AccessManagedGrant'];
export type Session = Models['Principal'];
export const accessPath = '/settings/security';
export const areaNames: Record<Area, string> = {
  accounts: '账号',
  roles: '角色',
  grants: '授权',
};
export const captions: Record<string, string> = {
  account: '账号管理',
  credential: '凭据重置',
  role: '角色',
  policy: '策略管理',
  assign: '业务授权',
  delegate: '管理职责委派',
  scope: '范围策略',
  fields: '字段策略',
  active: '启用',
  disabled: '停用',
  future: '待生效',
  expired: '已到期',
  revoked: '已撤销',
  duty: '本职',
  proxy: '限期代理',
  development: '开发样本',
  migration: '存量迁入',
  bu: '业务单元',
  department: '部门',
  staffingquota: '岗位核编',
  position: '岗位',
  enterprise: '企业',
  instance: '实例直管',
  code: '编码',
  name: '名称',
  nameEn: '英文名称',
  shortName: '简称',
  managementBuId: '管理业务单元',
  parentDepartmentId: '上级部门',
  departmentPath: '部门路径',
  status: '状态',
  effectiveDate: '生效日期',
  operationType: '业务类型',
  operationReason: '业务原因',
  departmentId: '所属部门',
  quota: '编制额度',
  reason: '原因',
  audit: '办理来源',
  usageState: '占用状态',
  sourceVersion: '来源版本',
  'account.create': '开户',
  'account.reset': '重置凭据',
  'account.enable': '启用账号',
  'account.disable': '停用账号',
  'account.move': '变更管理归属',
  'version.publish': '发布版本',
  'version.disable': '停用版本',
  'grant.issue': '新增授予',
  'grant.revoke': '撤销授予',
  'grant.adopt': '采用新版本',
  'mandate.issue': '委派管理职责',
  'mandate.revoke': '撤销管理职责',
  'admin.bootstrap': '首次引导',
  'admin.recover': '恢复管理员',
  'admin.handover': '交接管理员',
  'admin.handover.end': '结束原管理资格',
};
export const label = (value: string) => captions[value] ?? value;
export const pages = (rights: string[] = []) =>
  (Object.keys(areaNames) as Area[]).filter((area) =>
    ({
      accounts: ['account', 'credential'],
      roles: ['role', 'policy'],
      grants: ['assign', 'delegate'],
    })[area].some((r) => rights.includes(r)),
  );
export const targetName = (v: View, id: string) =>
  [...(v.targets ?? []), ...(v.owners ?? [])]
    .filter((t) => t.id === id)
    .map((t) => [t.code, t.name].filter(Boolean).join(' · '))[0] ||
  (id ? '原范围 ' + id : '待指定归属');
export const versionName = (v: Definition) => v.name + ' · V' + v.number;
export const version = (v: View, kind: string, id?: string) =>
  v.versions?.find((d) => d.id === id && d.kind === kind);
export const read = (
  area: Area | 'context',
  query: Record<string, string | number> = {},
) =>
  api
    .GET('/api/v1/access/{area}', { params: { path: { area }, query } })
    .then(unwrap);
const receipt = (r: Models['AccessManagementResult']) => ({
  ...r,
  submissionId: r.requestId,
  outcome:
    r.operation === 'submission.closed'
      ? ('not_applied' as const)
      : ('accepted' as const),
});
export const send = (s: Session, d: Command, id: string) =>
  api
    .POST('/api/v1/access/commands', {
      params: { header: { 'X-CSRF-Token': s.csrf } },
      body: { ...d, instanceId: s.instanceId, requestId: id },
    })
    .then(unwrap)
    .then(receipt);
export const resolve = (s: Session, id: string) =>
  api
    .POST('/api/v1/access/submissions/{id}/resolve', {
      params: { path: { id }, header: { 'X-CSRF-Token': s.csrf } },
    })
    .then(unwrap)
    .then(receipt);
