import type { Models } from '@chnhcm/api-client';
import type { Profile } from './api';

export const basicColumns = [
  {
    number: '工号',
    name: '姓名',
    formerName: '曾用名',
    englishName: '英文名',
    birthDate: '出生日期',
    gender: '性别',
    maritalStatus: '婚姻状况',
  },
  {
    nationality: '国籍',
    ethnicity: '民族',
    nativePlace: '籍贯',
    birthPlace: '出生地',
    householdCategory: '户籍类别',
    householdNature: '户口性质',
    politicalStatus: '政治面貌',
  },
] as const;
export const documentLabels: Record<string, string> = {
  documentKind: '证件类型',
  documentNumber: '证件号码',
  documentName: '证件姓名',
  issuingCountry: '签发国家/地区',
  issuer: '签发机关',
  issueDate: '签发日期',
  expiryDate: '到期日期',
  longTerm: '长期有效',
};
export const contactLabels: Record<string, string> = {
  kind: '联系渠道',
  contactPurpose: '联系用途',
  value: '电话',
  contactPrimary: '主要联系方式',
};
export const kinds: Record<string, string> = {
  resident_id: '居民身份证',
  passport: '护照',
  other: '其他身份证件',
  phone: '电话',
  email: '电子邮箱',
};
type Choice = Models['PersonCountryChoice'];
export const addressLabels = {
  addressType: '地址类型',
  addressCountry: '国家/地区',
  province: '省/州',
  area: '地址地区',
  street: '详细地址',
  postalCode: '邮政编码',
  addressPrimary: '主要地址',
};
export const familyLabels = {
  familyName: '联系人姓名',
  relationship: '与员工关系',
  familyBirthDate: '出生日期',
  employer: '工作单位',
  familyJob: '职务',
  familyPhone: '电话',
  familyEmail: '电子邮箱',
  emergency: '紧急联系人',
};
export const educationLabels = {
  educationType: '教育记录类型',
  school: '学校/授予院校',
  major: '所学专业',
  educationLevel: '学历',
  degree: '学位',
  studyMode: '学习形式',
  educationPath: '教育途径',
  startMonth: '入学年月',
  endMonth: '毕业/结束年月',
  completion: '教育完成状态',
  awardDate: '学位授予日期',
  highestEducation: '最高学历',
  highestDegree: '最高学位',
};
export const credentialLabels = {
  credentialKind: '凭证类别',
  credentialName: '职称/资格名称',
  credentialLevel: '等级',
  specialty: '专业方向',
  certificateNumber: '证书编号',
  awardingBody: '颁发机构',
  issuedOn: '签发日期',
  validFrom: '有效起始日期',
  validUntil: '有效截止日期',
  indefinite: '长期有效',
};
const educationLayout: Record<string, string[]> = {
  学习经历: [
    'educationType',
    'school',
    'major',
    'educationLevel',
    'degree',
    'studyMode',
    'educationPath',
    'startMonth',
    'endMonth',
    'completion',
    'highestEducation',
    'highestDegree',
  ],
  已取得学历: [
    'educationType',
    'school',
    'major',
    'educationLevel',
    'studyMode',
    'educationPath',
    'endMonth',
    'highestEducation',
  ],
  学位授予: ['educationType', 'school', 'major', 'degree', 'awardDate', 'highestDegree'],
};
export function rowFields(group: Group, fields: Fields): string[] {
  if (group === 'education') return educationLayout[fields.educationType] || Object.keys(educationLabels);
  return Object.keys(labelsByGroup[group]).filter(
    (k) => !((k === 'expiryDate' && fields.longTerm) || (k === 'validUntil' && fields.indefinite)),
  );
}
export function isProfileItem(group?: Group) {
  return group !== undefined && group !== 'documents' && group !== 'contacts';
}
export const groups = ['documents', 'addresses', 'contacts', 'family', 'education', 'credentials'] as const;
export type Group = (typeof groups)[number];
export const labelsByGroup: Record<Group, Record<string, string>> = {
  documents: documentLabels,
  contacts: contactLabels,
  addresses: addressLabels,
  family: familyLabels,
  education: educationLabels,
  credentials: credentialLabels,
};
export const requiredByGroup: Record<Group, string[]> = {
  documents: ['documentKind', 'documentNumber'],
  contacts: ['kind', 'value'],
  addresses: ['addressType', 'addressCountry', 'street'],
  family: ['familyName', 'relationship'],
  education: ['school', 'educationType'],
  credentials: ['credentialName', 'credentialKind'],
};
export const itemOptions: Record<string, string[]> = {
  educationType: Object.keys(educationLayout),
  credentialKind: ['职称', '职业资格', '职业技能等级', '执业许可', '专业认证'],
  completion: ['在读', '已毕业', '结业', '肄业'],
  addressType: ['居住地址', '户籍地址', '通讯地址', '其他地址'],
  relationship: [
    '配偶',
    '父母',
    '父亲',
    '母亲',
    '子女',
    '子',
    '女',
    '兄弟姐妹',
    '祖父母',
    '其他亲属',
    '朋友',
    '其他',
  ],
};
export type Places = Record<
  string,
  Models['PersonPlaceChoice'] | Models['PersonCountryAdoption'] | undefined
>;
export type Fields = Record<string, string>;
export type Row = {
  key: string;
  revision?: string;
  locked?: string[];
  hidden?: string[];
  masked?: string[];
  canRemove?: boolean;
  places?: Places;
  country?: Choice;
  fields: Fields;
  removed?: boolean;
};
export type Draft = {
  basic: Fields;
  nationality?: Choice;
  documents: Row[];
  contacts: Row[];
  addresses: Row[];
  family: Row[];
  education: Row[];
  credentials: Row[];
};
export function fromProfile(p?: Profile): Draft {
  return {
    basic: { ...p?.basic },
    nationality: p?.nationalityAdoption,
    addresses: (p?.addresses || []).map(fromItem),
    family: (p?.family || []).map(fromItem),
    education: (p?.education || []).map(fromItem),
    credentials: (p?.credentials || []).map(fromItem),
    documents: (p?.documents || []).map((x) => ({
      key: x.itemId,
      revision: x.revision,
      locked: x.locked,
      country: x.issuingCountryAdoption,
      fields: { ...x.fields, longTerm: x.longTerm ? 'yes' : '', documentPrimary: x.primary ? 'yes' : '' },
    })),
    contacts: (p?.contacts || []).map((x) => ({
      key: x.itemId,
      revision: x.revision,
      locked: x.locked,
      fields: {
        kind: x.kind,
        value: x.value,
        contactPurpose: x.purpose || '',
        contactPrimary: x.primary ? 'yes' : '',
      },
    })),
  };
}
function fromItem(x: Models['PersonProfileItem']): Row {
  return {
    key: x.itemId,
    revision: x.revision,
    locked: x.locked,
    hidden: x.hidden,
    masked: x.masked,
    canRemove: x.canRemove,
    fields: { ...x.fields },
    places: x.adoptions,
  };
}
// Carry forward only edits that still have full read and write permission on this row.
export function reconcileItem(old: Row | undefined, edited: Row, current: Row, writable: string[]): Row {
  const fields = { ...current.fields },
    places = { ...current.places };
  const canWrite = (key: string) =>
    writable.includes(key) && !current.locked?.includes(key) && !current.hidden?.includes(key);
  for (const [key, value] of Object.entries(edited.fields)) {
    if (canWrite(key) && value !== (old?.fields[key] || '')) fields[key] = value;
  }
  for (const key of ['addressCountry', 'province', 'area']) {
    if (canWrite(key) && JSON.stringify(edited.places?.[key]) !== JSON.stringify(old?.places?.[key])) {
      places[key] = edited.places?.[key];
      fields[key] = edited.fields[key] || '';
    }
  }
  return { ...current, fields, places, removed: !!edited.removed && !!current.canRemove };
}
function itemCommand(rows: Row[], original: Row[]) {
  const edits: Models['PersonProfileItemEdit'][] = [];
  for (const row of rows) {
    const old = original.find((x) => x.key === row.key);
    const item = old ? { itemId: old.key, expectedRevision: old.revision } : {};
    if (row.removed) {
      if (old) edits.push({ ...item, remove: true });
      continue;
    }
    const fields = changed(old?.fields || {}, row.fields);
    const choices: Record<string, Models['PersonPlaceChoice']> = {};
    for (const key of ['addressCountry', 'province', 'area']) {
      const prev = old?.places?.[key],
        next = row.places?.[key];
      // Saved adoptions carry their date. A fresh picker choice also needs its ancestors rechecked.
      if (next && !('date' in next)) {
        choices[key] = { id: next.id, sourceVersion: next.sourceVersion, ancestors: next.ancestors };
        delete fields[key];
      }
      if (prev && !next) fields[key] = row.fields[key] || '';
    }
    if (Object.keys(choices).length || Object.keys(fields).length) edits.push({ ...item, fields, choices });
  }
  return edits;
}
function changed(before: Fields, after: Fields) {
  return Object.fromEntries(Object.entries(after).filter(([k, v]) => v !== (before[k] || '')));
}
export function command(p: Profile | undefined, draft: Draft) {
  const base = fromProfile(p);
  const basic = changed(base.basic, draft.basic);
  const documents: Models['PersonDocumentEdit'][] = [],
    contacts: Models['PersonEdit'][] = [];
  for (const group of ['documents', 'contacts'] as const) {
    for (const row of draft[group]) {
      const old = base[group].find((x) => x.key === row.key);
      const item = old ? { itemId: old.key, expectedRevision: old.revision } : {};
      if (row.removed) {
        if (old) (group === 'documents' ? documents : contacts).push({ ...item, remove: true });
        continue;
      }
      const diff = changed(old?.fields || {}, row.fields);
      const choice = changedChoice(old?.country, row.country);
      if (old?.country && !row.country) diff.issuingCountry = row.fields.issuingCountry || '';
      if (!Object.keys(diff).length && !choice) continue;
      if (group === 'documents') {
        const { longTerm, documentPrimary, ...fields } = diff;
        documents.push({
          ...item,
          fields,
          issuingCountryChoice: choice,
          ...(longTerm !== undefined ? { longTerm: !!longTerm } : {}),
          ...(documentPrimary !== undefined ? { primary: !!documentPrimary } : {}),
        });
      } else {
        contacts.push({
          ...item,
          ...(diff.kind !== undefined || diff.value !== undefined
            ? { kind: row.fields.kind, value: row.fields.value }
            : {}),
          ...(diff.contactPurpose !== undefined ? { purpose: diff.contactPurpose } : {}),
          ...(diff.contactPrimary !== undefined ? { primary: !!diff.contactPrimary } : {}),
        });
      }
    }
  }
  if (base.nationality && !draft.nationality) basic.nationality = draft.basic.nationality || '';
  return {
    basic,
    documents,
    contacts,
    addresses: itemCommand(draft.addresses, base.addresses),
    family: itemCommand(draft.family, base.family),
    education: itemCommand(draft.education, base.education),
    credentials: itemCommand(draft.credentials, base.credentials),
    nationalityChoice: changedChoice(base.nationality, draft.nationality),
  };
}
export function applyRow(draft: Draft, group: Group, row: Row): Draft {
  const flags =
    group === 'education'
      ? ['highestEducation', 'highestDegree']
      : group === 'addresses'
        ? ['addressPrimary']
        : group === 'documents'
          ? ['documentPrimary']
          : group === 'contacts'
            ? ['contactPrimary']
            : [];
  const next = draft[group]
    .filter((x) => x.key !== row.key || !row.removed || x.revision)
    .map((x) => {
      if (x.key === row.key) return row;
      if (row.removed || (group === 'contacts' && row.fields.kind !== x.fields.kind)) return x;
      const cleared = flags.filter((k) => row.fields[k] && x.fields[k] && !x.locked?.includes(k));
      return cleared.length
        ? { ...x, fields: { ...x.fields, ...Object.fromEntries(cleared.map((k) => [k, ''])) } }
        : x;
    });
  if (row.removed && !row.revision) return { ...draft, [group]: next };
  if (!next.some((x) => x.key === row.key)) next.push(row);
  return { ...draft, [group]: next };
}

function changedChoice(old: Choice | undefined, next: Choice | undefined): Choice | undefined {
  return next && (old?.id !== next.id || old?.sourceVersion !== next.sourceVersion)
    ? { id: next.id, sourceVersion: next.sourceVersion }
    : undefined;
}
