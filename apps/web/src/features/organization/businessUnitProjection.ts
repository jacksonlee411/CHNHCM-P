import type { DetailView, RecordView } from "./dated/presentation";

export const canReadStatus = (fields: string[], statusReadField: string | undefined) =>
  statusReadField === undefined || fields.includes(statusReadField);

// Reference-label authority is separate from the department field's read right.
// Keep relationship IDs and drafts; clear cached labels until a fresh detail read.
export function trimDetailReferences(value: DetailView): DetailView {
  const record = <T extends Partial<RecordView>>(r: T): T => ({ ...r, references: Object.fromEntries(Object.entries(r.references ?? {}).map(([key,v]) => [key, v ? { id: v.id } : v])) });
  return {
    ...value,
    selectedRecord: value.selectedRecord && record(value.selectedRecord),
    timeline: value.timeline.map(record),
    viewFact: value.viewFact && record(value.viewFact),
    relatedReferencesRestricted: true,
    additionContext: value.additionContext && { ...value.additionContext, defaults: value.additionContext.defaults && record(value.additionContext.defaults) },
    lifecycleContext: value.lifecycleContext && { ...value.lifecycleContext, defaults: value.lifecycleContext.defaults && record(value.lifecycleContext.defaults) },
    pairDeletion: value.pairDeletion && {
      ...value.pairDeletion, deactivation: record(value.pairDeletion.deactivation), reactivation: record(value.pairDeletion.reactivation),
      changes: value.pairDeletion.changes.map(change => ({ ...change, before: record(change.before), after: record(change.after) })),
    },
  };
}

// Clear cached presentation fields using the server's current read-field list.
// The following reload obtains authoritative capabilities and source contents.
export function trimDetailFields(value: DetailView, fields: string[], statusReadField: string | undefined) {
  const can = (field: string) => fields.includes(field);
  const statusReadable = canReadStatus(fields, statusReadField);
  const record = <T extends Partial<RecordView>>(r: T) => ({
    ...r,
    state: statusReadable ? r.state : undefined,
    name: can("name") ? r.name : undefined,
    values: Object.fromEntries(Object.entries(r.values ?? {}).filter(([key]) => can(key))),
    references: Object.fromEntries(Object.entries(r.references ?? {}).filter(([key]) => can(key))),
    effectiveDate: can("effectiveDate") ? r.effectiveDate : undefined,
    endExclusive: can("effectiveDate") ? r.endExclusive : null,
    endInclusive: can("effectiveDate") ? r.endInclusive : null,
    operationType: can("operationType") ? r.operationType : "",
    typeLabel: can("operationType") ? r.typeLabel : "",
    operationReason: can("operationReason") ? r.operationReason : "",
    reasonLabel: can("operationReason") ? r.reasonLabel : "",
  });
  const out = { ...value, status: statusReadable ? value.status : undefined, capabilities: { ...value.capabilities, readFields: fields }, timeline: value.timeline.map(record) };
  out.relatedReferencesRestricted = true;
  delete out.pairInsertion;delete out.pairDeletion;
  if(out.lifecycleContext){
    const c={...out.lifecycleContext,defaults:{...out.lifecycleContext.defaults}};
    if(!can("effectiveDate")){delete c.minEffectiveDate;delete c.maxEffectiveDate;delete c.defaultEffectiveDate;}
    if (out.lifecycleContext.defaults) c.defaults = record(out.lifecycleContext.defaults);out.lifecycleContext=c;
  }
  if(out.additionContext){
    if(!can('effectiveDate'))delete out.additionContext;
    else if(out.additionContext.defaults){const d=record(out.additionContext.defaults);out.additionContext={...out.additionContext,defaults:d};}
  }
  if (out.capabilities.correction) {
    const c={...out.capabilities.correction,facets:[...out.capabilities.correction.facets]};
    if(!can('effectiveDate')){delete c.minEffectiveDate;delete c.maxEffectiveDate;c.facets=c.facets.filter(f=>f!=='REDATE');}
    if(!can('operationType')||!can('operationReason')){c.classificationTypes=[];c.facets=c.facets.filter(f=>f!=='RECLASSIFY');}
    if(!Object.keys(value.selectedRecord?.values ?? {}).some(can))c.facets=c.facets.filter(f=>f!=='CORRECT');
    out.capabilities.correction=c;
  }
  if (out.selectedRecord) out.selectedRecord = record(out.selectedRecord);
  if (out.viewFact) out.viewFact = record(out.viewFact);
  if (!can("code")) out.code = undefined;
  return out;
}
