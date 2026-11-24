import test from 'node:test';
import assert from 'node:assert/strict';
import { trimDetailFields, trimDetailReferences } from '../apps/web/src/features/organization/businessUnitProjection.ts';

function detail() {
  const record = { recordId: 'record', state: 'active', name: '仍可读名称', values: { name: '仍可读名称' } };
  return {
    status: 'active', capabilities: { readFields: ['name', 'status'] },
    selectedRecord: { ...record }, timeline: [{ ...record }, { ...record, state: 'inactive' }],
    viewFact: { ...record }, lifecycleContext: { defaults: { ...record } },
    additionContext: { defaults: { ...record } },
  };
}

test('revoked department status is removed from every cached projection and cannot return without a fresh read', () => {
  const original = detail(), before = structuredClone(original);
  const safe = trimDetailFields(original, ['name', 'effectiveDate'], 'status');
  const cached = [safe.selectedRecord, ...safe.timeline, safe.viewFact, safe.lifecycleContext.defaults, safe.additionContext.defaults];
  assert.equal(safe.status, undefined);
  for (const record of cached) {
    assert.equal(record.state, undefined);
    assert.equal(record.name, '仍可读名称');
  }
  assert.deepEqual(original, before);
  const readableAgain = trimDetailFields(safe, ['name', 'effectiveDate', 'status'], 'status');
  assert.equal(readableAgain.status, undefined);
  assert.equal(readableAgain.selectedRecord.state, undefined);
});

test('BU status retains its own read contract when other fields are removed', () => {
  const safe = trimDetailFields(detail(), [], undefined);
  assert.equal(safe.status, 'active');
  assert.equal(safe.selectedRecord.state, 'active');
  assert.equal(safe.timeline[1].state, 'inactive');
  assert.equal(safe.selectedRecord.name, undefined);
});

test('reference revocation clears labels throughout the detail while preserving readable values, relationship IDs and source identity', () => {
  const record = { recordId: 'record', revision: 'revision', name: '可读部门名称', state: 'active', effectiveDate: '2026-01-01',
    values: { name: '可读部门名称', managementBuId: 'bu', parentDepartmentId: 'parent' }, references: { managementBuId: { id: 'bu', code: 'BU', name: '旧BU名称' }, parentDepartmentId: { id: 'parent', code: 'P', name: '旧上级名称' } } };
  const original = { ...detail(), headSourceVersion: 'source', selectedRecord: record, timeline: [record], viewFact: record,
    additionContext: { defaults: record }, lifecycleContext: { defaults: record },
    pairDeletion: { deactivation: record, reactivation: record, changes: [{ before: record, after: record }] },
    related: (record, fields, restricted) => ({ record, fields, restricted }) };
  const before = { ...original, selectedRecord: structuredClone(record) }, safe = trimDetailReferences(original);
  for (const r of [safe.selectedRecord, ...safe.timeline, safe.viewFact, safe.additionContext.defaults, safe.lifecycleContext.defaults,
    safe.pairDeletion.deactivation, safe.pairDeletion.reactivation, safe.pairDeletion.changes[0].before, safe.pairDeletion.changes[0].after]) {
    assert.deepEqual(r, { ...record, references: { managementBuId: { id: 'bu' }, parentDepartmentId: { id: 'parent' } } });
  }
  assert.equal(safe.related, original.related);
  assert.equal(safe.relatedReferencesRestricted, true);
  const rendered = safe.related(safe.selectedRecord, safe.capabilities.readFields, safe.relatedReferencesRestricted);
  assert.equal(rendered.record.references.parentDepartmentId.name, undefined);
  assert.equal(rendered.restricted, true);
  assert.equal(safe.headSourceVersion, 'source');
  assert.deepEqual(safe.capabilities, original.capabilities);
  assert.deepEqual(original, before);
});

test('country display projection clears any revoked field including a new object-owned field', () => {
  const original = detail(); original.selectedRecord.values = { name: '国家名称', phoneCode: '86', nameEn: 'China' };
  const safe = trimDetailFields(original, ['name'], 'state');
  assert.deepEqual(safe.selectedRecord.values, { name: '国家名称' });
  assert.deepEqual(trimDetailFields(safe, ['name','phoneCode'], 'state').selectedRecord.values, { name: '国家名称' });
  assert.equal(original.selectedRecord.values.phoneCode, '86');
});

test('field revocation retains the object-owned section while rendering only current fields and values', () => {
  const original = detail();
  original.selectedRecord.values.parentDepartmentId = 'parent';
  original.related = (record, fields) => fields.includes('parentDepartmentId') ? record.values.parentDepartmentId : '无读取权限';
  const safe = trimDetailFields(original, ['name'], 'status');
  assert.equal(safe.related(safe.selectedRecord, safe.capabilities.readFields), '无读取权限');
  assert.equal(safe.selectedRecord.values.parentDepartmentId, undefined);
  assert.equal(safe.relatedReferencesRestricted, true);
});
