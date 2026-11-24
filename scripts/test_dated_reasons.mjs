import test from 'node:test';
import assert from 'node:assert/strict';
import { selectedReason } from '../apps/web/src/features/organization/dated/reasons.ts';

test('a sole choice is not inferred as a default by the client', () => {
  assert.equal(selectedReason('', [{ code: 'one', default: false }]), '');
  assert.equal(selectedReason('', [{ code: 'one', default: true }]), 'one');
  assert.equal(selectedReason('', []), '');
});
test('a current valid selection survives a new server default; a removed one does not', () => {
  const choices = [{ code: 'chosen', default: false }, { code: 'default', default: true }];
  assert.equal(selectedReason('chosen', choices), 'chosen');
  assert.equal(selectedReason('removed', choices), 'default');
  assert.equal(selectedReason('removed', choices.map(value => ({ ...value, default: false }))), '');
});
