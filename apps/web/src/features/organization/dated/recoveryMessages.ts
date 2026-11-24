import type { Models } from '@chnhcm/api-client';

const reasons: Record<string, string> = {
  M02_SOURCE_CONFLICT: '资料已更新，请核对最新内容后保存',
  M02_RECORD_CONFLICT: '记录已更新，请核对最新内容后保存',
  M02_TIMELINE_INVALID: '当前修改无法保存',
  M02_INVALID_INPUT: '输入内容不符合要求',
  M02_CODE_EXISTS: '编码已存在',
  ACCESS_DENIED: '没有操作权限',
};
const fields: Record<string, string> = {
  code: '编码', name: '名称', nameEn: '英文名称', shortName: '简称',
  effectiveDate: '生效日期', newEffectiveDate: '生效日期',
  operationType: '操作类型', operationReason: '操作原因',
  managementBuId: '管理业务单元', parentDepartmentId: '行政上级',
  deactivationDate: '停用日期', reactivationDate: '重新启用日期',
};

// Translate the authorized, stored result; never revalidate against today's data.
export function recoveryMessages(details?: Models['FailureDetails'] | null) {
  return [...new Set((details?.issues ?? []).map(issue => {
    const labels = [...new Set(issue.locations.flatMap(location => {
      const key = location.fieldPath?.replace(/^changes\./, '');
      return key && fields[key] ? [fields[key]] : [];
    }))];
    return `${labels.length ? labels.join('、') + '：' : ''}${reasons[issue.code] ?? '当前修改无法保存'}`;
  }))];
}
