import { Button } from '@mui/material';
import { ObjectDetailSection } from '@chnhcm/ui';
import {
  isProfileItem,
  labelsByGroup,
  requiredByGroup,
  kinds,
  type Fields,
  type Group,
  type Row,
} from './profileDraft';

type Props = {
  group: Group;
  title: string;
  rows: Row[];
  read: string[];
  write: string[];
  editing: boolean;
  disabled?: boolean;
  open: (row: Row, mode: '新增记录' | '编辑记录' | '记录详情') => void;
  apply: (row: Row) => void;
};

type SummaryColumn = { key: string; label: string; value: (fields: Fields) => string };

function summaryColumns(group: Group, read: string[]) {
  const columns: SummaryColumn[] = [];
  if (group === 'contacts') {
    columns.push({ key: 'kind', label: '联系渠道', value: (f) => kinds[f.kind] });
    if (read.includes('contactPurpose'))
      columns.push({ key: 'contactPurpose', label: '联系用途', value: (f) => f.contactPurpose });
  } else if (isProfileItem(group)) {
    const summaryKeys: Partial<Record<Group, string[]>> = {
      addresses: ['street', 'addressCountry', 'province', 'area', 'postalCode'],
      family: ['relationship', 'familyPhone', 'familyEmail', 'emergency', 'employer', 'familyJob'],
      education: ['educationType', 'major', 'educationLevel', 'highestEducation', 'degree', 'highestDegree'],
      credentials: [
        'credentialKind',
        'credentialLevel',
        'awardingBody',
        'validFrom',
        'validUntil',
        'indefinite',
      ],
    };
    const keys = summaryKeys[group] || [];
    for (const key of keys)
      if (read.includes(key))
        columns.push({
          key,
          label: labelsByGroup[group][key],
          value: (f) =>
            ['emergency', 'highestEducation', 'highestDegree', 'indefinite'].includes(key)
              ? f[key]
                ? '是'
                : '否'
              : f[key],
        });
  } else {
    if (read.includes('documentNumber') || read.includes('documentNumber.masked')) {
      columns.push({ key: 'documentNumber', label: '证件号码', value: () => '********' });
    }
    if (read.includes('issuingCountry'))
      columns.push({ key: 'issuingCountry', label: '签发国家/地区', value: (f) => f.issuingCountry });
    if (read.includes('longTerm') || read.includes('expiryDate')) {
      columns.push({
        key: 'validity',
        label: '有效期',
        value: (f) => {
          if (read.includes('longTerm') && f.longTerm) return '长期有效';
          return read.includes('expiryDate') && f.expiryDate ? '至 ' + f.expiryDate : '';
        },
      });
    }
  }
  return columns;
}

export function ProfileCollection({
  group,
  title,
  rows,
  read,
  write,
  editing,
  disabled,
  open,
  apply,
}: Props) {
  const doc = group === 'documents';
  const item = isProfileItem(group);
  const flag = doc ? 'documentPrimary' : group === 'addresses' ? 'addressPrimary' : 'contactPrimary';
  const columns = summaryColumns(group, read).filter(
    (column) => !rows.length || rows.some((row) => !row.hidden?.includes(column.key)),
  );
  const canAdd =
    editing &&
    (item
      ? requiredByGroup[group].every((k) => write.includes(k))
      : doc
        ? write.includes('documentKind') && write.includes('documentNumber')
        : write.includes('phone') || write.includes('email'));
  const records = rows.map((row) => {
    const writable = write.filter((k) => !row.locked?.includes(k));
    const field = row.fields;
    const canEdit =
      editing &&
      (item
        ? Object.keys(labelsByGroup[group]).some((k) => writable.includes(k))
        : doc
          ? Object.keys(labelsByGroup.documents).some((k) => writable.includes(k))
          : writable.includes(field.kind));
    const canMark =
      editing &&
      (group === 'addresses'
        ? writable.includes('addressPrimary')
        : doc && writable.includes('documentNumber') && writable.includes('documentPrimary'));
    const canRemove =
      canEdit &&
      (item
        ? !row.revision || row.canRemove
        : doc
          ? writable.includes('documentKind') &&
            writable.includes('documentNumber') &&
            Object.entries(field).every(([k, v]) => !v || writable.includes(k))
          : (!field.contactPurpose || writable.includes('contactPurpose')) &&
            (!field.contactPrimary || writable.includes('contactPrimary')));
    return { row, canEdit, canMark, canRemove };
  });
  const actions = records.some((r) => r.canMark || r.canRemove);
  return (
    <ObjectDetailSection title={title}>
      <div
        className="profile-collection"
        id={'profile-' + group}
        role="group"
        aria-label={title}
        aria-describedby={'profile-count-' + group}
        tabIndex={-1}
      >
        <span className="profile-count" id={'profile-count-' + group}>
          {records.length} 条
        </span>
        {!records.length ? (
          <p className="profile-empty">暂无记录</p>
        ) : (
          <div className="profile-table-scroll">
            <table>
              <thead>
                <tr>
                  <th scope="col">
                    {item ? labelsByGroup[group][requiredByGroup[group][0]] : doc ? '证件类型' : '号码或邮箱'}
                  </th>
                  {columns.map((c) => (
                    <th key={c.key} scope="col">
                      {c.label}
                    </th>
                  ))}
                  {actions && <th scope="col">操作</th>}
                </tr>
              </thead>
              <tbody>
                {records.map(({ row, canEdit, canMark, canRemove }) => {
                  const f = row.fields;
                  const name = item ? f[requiredByGroup[group][0]] : doc ? kinds[f.documentKind] : f.value;
                  const primary = read.includes(flag) && !!f[flag];
                  return (
                    <tr key={row.key}>
                      <td className="profile-row-name">
                        <button
                          type="button"
                          id={'profile-row-' + row.key}
                          disabled={disabled}
                          aria-label={(canEdit ? '编辑' : '查看') + ' · ' + name}
                          onClick={() => open(row, canEdit ? '编辑记录' : '记录详情')}
                        >
                          {name}
                        </button>
                        {primary && <span className="profile-primary">主要</span>}
                      </td>
                      {columns.map((c) => {
                        if (row.hidden?.includes(c.key))
                          return (
                            <td key={c.key} aria-hidden="true" className="profile-metadata profile-missing" />
                          );
                        const value = c.value(f);
                        return (
                          <td key={c.key} className={'profile-metadata' + (value ? '' : ' profile-missing')}>
                            <span className="profile-meta-label" aria-hidden="true">
                              {c.label}
                            </span>
                            <span>{value || <span className="profile-empty">未填写</span>}</span>
                          </td>
                        );
                      })}
                      {actions && (
                        <td className="profile-row-actions">
                          {canMark && (
                            <Button
                              size="small"
                              disabled={disabled}
                              onClick={() =>
                                apply({ ...row, fields: { ...f, [flag]: primary ? '' : 'yes' } })
                              }
                            >
                              {primary ? '撤下主要' : '设为主要'}
                            </Button>
                          )}
                          {canRemove && (
                            <Button
                              size="small"
                              disabled={disabled}
                              onClick={() => apply({ ...row, removed: true })}
                            >
                              移除
                            </Button>
                          )}
                        </td>
                      )}
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
        {canAdd && (
          <Button
            size="small"
            disabled={disabled}
            onClick={() =>
              open(
                {
                  key: crypto.randomUUID(),
                  fields:
                    group === 'education'
                      ? { educationType: '学习经历' }
                      : item
                        ? {}
                        : doc
                          ? { documentKind: '', documentNumber: '' }
                          : { kind: write.includes('phone') ? 'phone' : 'email', value: '' },
                },
                '新增记录',
              )
            }
          >
            添加一行
          </Button>
        )}
      </div>
    </ObjectDetailSection>
  );
}
