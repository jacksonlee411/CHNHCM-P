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
export const groups = ['documents', 'addresses', 'contacts', 'family'] as const;
export type Group = (typeof groups)[number];
export const labelsByGroup: Record<Group, Record<string, string>> = {
  documents: documentLabels,
  contacts: contactLabels,
  addresses: addressLabels,
  family: familyLabels,
};
export const requiredByGroup: Record<Group, string[]> = {
  documents: ['documentKind', 'documentNumber'],
  contacts: ['kind', 'value'],
  addresses: ['addressType', 'addressCountry', 'street'],
  family: ['familyName', 'relationship'],
};
export const itemOptions: Record<string, string[]> = {
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
export type Places = Record<string, Models['PersonPlaceChoice'] | Models['PersonCountryAdoption'] | undefined>;
export type Fields = Record<string, string>;
export type Row = {
  key: string;
  revision?: string;
  locked?: string[];
  hidden?: string[];
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
};
export function fromProfile(p?: Profile): Draft {
  return {
    basic: { ...p?.basic },
    nationality: p?.nationalityAdoption,
    addresses: (p?.addresses || []).map(fromItem),
    family: (p?.family || []).map(fromItem),
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
    nationalityChoice: changedChoice(base.nationality, draft.nationality),
  };
}
export function applyRow(draft: Draft, group: Group, row: Row): Draft {
  const flag =
    group === 'documents' ? 'documentPrimary' : group === 'addresses' ? 'addressPrimary' : 'contactPrimary';
  const rows = draft[group].filter((x) => x.key !== row.key || !row.removed || x.revision);
  if (row.removed && !row.revision) return { ...draft, [group]: rows };
  // Replacing a primary affects only the current group. The server rechecks both rows.
  const next = rows.map((x) =>
    x.key === row.key
      ? row
      : row.fields[flag] && !row.removed && (group !== 'contacts' || row.fields.kind === x.fields.kind)
        ? { ...x, fields: { ...x.fields, [flag]: '' } }
        : x,
  );
  if (!next.some((x) => x.key === row.key)) next.push(row);
  return { ...draft, [group]: next };
}

function changedChoice(old: Choice | undefined, next: Choice | undefined): Choice | undefined {
  return next && (old?.id !== next.id || old?.sourceVersion !== next.sourceVersion)
    ? { id: next.id, sourceVersion: next.sourceVersion }
    : undefined;
}
