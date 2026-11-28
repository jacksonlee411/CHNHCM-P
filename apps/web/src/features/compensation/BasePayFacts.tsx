import { Link } from 'react-router-dom';
import { ObjectDetailSection, ObjectDetailField } from '@chnhcm/ui';
import type { Basis } from './basePay';
import { money, ratePath } from './rates';
import { planPath } from './plans';

export function PayBasisFacts({
  basis,
  title,
  day,
  original = false,
}: {
  basis: Basis;
  title: string;
  day: string;
  original?: boolean;
}) {
  const link = (
    path: string,
    id: string | undefined,
    source: string | undefined,
    record: string | undefined,
    text: string,
  ) => {
    if (!id || !text) return text;
    const query = new URLSearchParams();
    if (original && source) query.set('sourceVersion', source);
    else if (path === planPath) query.set('referenceDate', day);
    else if (record) query.set('selectedRecordId', record);
    const back = new URLSearchParams(location.search);
    back.set('payFocus', path === planPath ? 'plan' : 'rate');
    query.set('back', location.pathname + '?' + back);
    return (
      <Link
        id={'pay-basis-' + (path === planPath ? 'plan' : 'rate') + '-' + id}
        to={path + '/' + id + '?' + query}
      >
        {text}
      </Link>
    );
  };
  return (
    <ObjectDetailSection title={title}>
      {(basis.planName || basis.planCode) && (
        <ObjectDetailField label="薪酬计划">
          {link(
            planPath,
            basis.planId,
            basis.planSource,
            basis.planRecord,
            [basis.planCode, basis.planName].filter(Boolean).join(' · '),
          )}
        </ObjectDetailField>
      )}
      {basis.grade && <ObjectDetailField label="实际职级">{basis.grade}</ObjectDetailField>}
      {basis.componentName && <ObjectDetailField label="基本工资项">{basis.componentName}</ObjectDetailField>}
      {(basis.minimum || basis.maximum) && (
        <ObjectDetailField label="月薪带">
          {link(
            ratePath,
            basis.rateId,
            basis.rateSource,
            basis.rateRecord,
            [basis.minimum && money(basis.minimum), basis.maximum && money(basis.maximum)]
              .filter(Boolean)
              .join(' – ') + ' 元/月',
          )}
        </ObjectDetailField>
      )}
      {basis.guidance && <ObjectDetailField label="指导值">{money(basis.guidance)} 元/月</ObjectDetailField>}
      {basis.allowances?.map((a, i) => (
        <ObjectDetailField key={i} label={a.name + '（计划参考）'}>
          {money(a.amount)} 元/月
        </ObjectDetailField>
      ))}
    </ObjectDetailSection>
  );
}
