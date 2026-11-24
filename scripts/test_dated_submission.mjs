import test from 'node:test';
import assert from 'node:assert/strict';
import { SubmissionFlow, submissionChannel } from '../apps/web/src/shared/dated/submission.ts';

const owner = { instanceId: 'instance', subjectId: 'alice' };
const reference = id => ({ instanceRef: 'instance', subjectRef: 'alice', submissionId: id });
const terminal = (id, outcome = 'accepted') => ({ submissionId: id, outcome });
function fixture() {
  const values = new Map(), events = new EventTarget();
  const storage = { getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value) };
  const channel = submissionChannel('fixture.alpha', 'alpha', () => ({ storage, events }));
  let sequence = 0, generation = 0, alive = true;
  return { values, storage, events, channel,
    expire: () => generation++, unmount: () => { alive = false; },
    flow(send) { return new SubmissionFlow({ owner, channel, send, resolve: async ref => terminal(ref.submissionId, 'pending'), uuid: () => `submission-${++sequence}`, changed() {}, unwrap: x => x,
      current: () => { const own = generation; return () => alive && own === generation; }, definitive: e => e.status >= 400 && e.status < 500 }); },
  };
}
const deferred = () => { let resolve; const promise = new Promise(r => { resolve = r; }); return { promise, resolve }; };

test('pending storage retains only identity, deduplicates by all three fields, and isolates kinds', () => {
  const f = fixture(), beta = submissionChannel('fixture.beta', 'beta', () => f);
  f.channel.remember({ ...reference('same'), body: 'private' });
  f.channel.remember(reference('same'));
  f.channel.remember({ ...reference('same'), subjectRef: 'bob' }); beta.remember(reference('same'));
  assert.deepEqual(JSON.parse(f.values.get('alpha')), [reference('same'), { ...reference('same'), subjectRef: 'bob' }]);
  let a = 0, b = 0;
  f.channel.subscribe(() => a++, () => a++); beta.subscribe(() => b++, () => b++);
  beta.publish(reference('same'), terminal('same')); beta.deny(reference('same'));
  assert.equal(a, 0); assert.equal(b, 2);
  f.channel.forget(reference('same'));
  assert.deepEqual(f.channel.read(), [{ ...reference('same'), subjectRef: 'bob' }]); assert.deepEqual(beta.read(), [reference('same')]);
});

test('unknown retry sends the original immutable body and id; success removes only its reference', async () => {
  const f = fixture(), calls = [];
  const flow = f.flow(async (body, id) => { calls.push({ body: structuredClone(body), id }); if (calls.length === 1) throw new Error('lost'); return terminal(id); });
  const input = { names: ['original'] }; assert.equal((await flow.submit(input)).kind, 'failed'); input.names[0] = 'mutated';
  assert.equal(flow.unknown, true);
  assert.equal((await flow.submit({ names: ['new draft'] })).kind, 'saved');
  assert.deepEqual(calls, [{ body: { names: ['original'] }, id: 'submission-1' }, { body: { names: ['original'] }, id: 'submission-1' }]);
  assert.deepEqual(f.channel.read(), []); assert.equal(flow.saved, true);
});

test('definite rejection frees a fresh attempt; a rejection after an unknown reply stays pending', async () => {
  const f = fixture(); let lost = false;
  const flow = f.flow(async () => { if (lost) throw new Error('lost'); throw { status: 409 }; });
  assert.equal((await flow.submit({})).definitive, true); assert.deepEqual(f.channel.read(), []);
  lost = true; await flow.submit({}); lost = false;
  assert.equal((await flow.submit({})).definitive, false);
  assert.equal(flow.unknown, true); assert.deepEqual(f.channel.read(), [reference('submission-2')]);
});

test('resolve wins over an in-flight retry, clears busy, and permits a new attempt after not_applied', async () => {
  const f = fixture(), wait = deferred(); let calls = 0;
  const flow = f.flow(async (_, id) => { calls++; if (calls === 1) throw new Error('lost'); if (calls === 2) return wait.promise; return terminal(id); });
  await flow.submit({ old: true }); const retry = flow.submit({ ignored: true });
  f.channel.forget(reference('submission-1'));
  assert.equal(flow.resolved(reference('submission-1'), terminal('submission-1', 'not_applied')), true);
  const next = await flow.submit({ new: true }); assert.equal(next.result.submissionId, 'submission-2');
  wait.resolve(terminal('submission-1')); assert.equal((await retry).kind, 'ignored'); assert.equal(flow.saved, true); assert.deepEqual(f.channel.read(), []);
});

for (const boundary of ['expire', 'unmount']) test(`${boundary} drops a late success without clearing its recoverable reference`, async () => {
  const f = fixture(), wait = deferred(), flow = f.flow(() => wait.promise);
  const request = flow.submit({}); f[boundary](); wait.resolve(terminal('submission-1'));
  assert.equal((await request).kind, 'ignored'); assert.deepEqual(f.channel.read(), [reference('submission-1')]); assert.equal(flow.saved, false);
});

test('cross-owner, mismatched id and pending resolve cannot settle another form', async () => {
  const f = fixture(), flow = f.flow(async () => { throw new Error('lost'); }); await flow.submit({});
  assert.equal(flow.resolved({ ...reference('submission-1'), subjectRef: 'bob' }, terminal('submission-1')), false);
  assert.equal(flow.resolved(reference('submission-2'), terminal('submission-2')), false);
  assert.equal(flow.resolved(reference('submission-1'), terminal('submission-2')), false);
  assert.equal(flow.resolved(reference('submission-1'), terminal('submission-1', 'pending')), false);
  flow.clearPayload(); assert.equal(flow.attempt, undefined);
  assert.equal(flow.resolved(reference('submission-1'), terminal('submission-1', 'rejected')), true); assert.equal(flow.unknown, false);
});

test('corrupt/unavailable storage and existing same-kind pending prevent a new write', async () => {
  const f = fixture(); let calls = 0; const flow = f.flow(async (_, id) => { calls++; return terminal(id); });
  f.values.set('alpha', '{}'); const unreadable = await flow.submit({});
  assert.equal(unreadable.kind, 'failed'); assert.equal(unreadable.error.message, '暂时无法开始保存，请重试'); assert.equal(calls, 0);
  f.values.set('alpha', JSON.stringify([reference('other')])); await flow.submit({}); assert.equal(calls, 0);
  f.values.delete('alpha'); f.values.set('beta', JSON.stringify([reference('other')])); await flow.submit({}); assert.equal(calls, 1);
});

test('a successful transport is not a saved result without accepted/no_change for the exact id', async () => {
  for (const outcome of ['pending', 'rejected', 'not_applied', 'wrong-id']) {
    const f = fixture(), flow = f.flow(async (_, id) => outcome === 'wrong-id' ? terminal('different') : terminal(id, outcome));
    assert.equal((await flow.submit({})).kind, 'failed');
    assert.equal(flow.saved, false); assert.equal(flow.unknown, true); assert.deepEqual(f.channel.read(), [reference('submission-1')]);
  }
});

test('storage cleanup failure preserves confirmed acceptance and prevents resubmission', async () => {
  const f = fixture(), wait = deferred(), flow = f.flow(() => wait.promise); const request = flow.submit({});
  assert.equal((await flow.submit({ second: true })).kind, 'ignored');
  const write = f.storage.setItem; f.storage.setItem = () => { throw new Error('quota'); };
  wait.resolve(terminal('submission-1')); assert.equal((await request).kind, 'saved'); assert.equal(flow.unknown, false);
  assert.equal(flow.saved, true); assert.equal(flow.attempt, undefined);
  assert.equal((await flow.submit({ duplicate: true })).kind, 'ignored');
  assert.deepEqual(f.channel.read(), []);
  assert.deepEqual(JSON.parse(f.values.get('alpha')), [reference('submission-1')]);
  const refreshed = submissionChannel('fixture.alpha', 'alpha', () => f);
  assert.deepEqual(refreshed.read(), [reference('submission-1')]);
  f.storage.setItem = write; refreshed.forget(reference('submission-1'));
  assert.deepEqual(refreshed.read(), []);
});

test('discarding an in-flight body keeps only a recoverable identity if the form remains mounted', async () => {
  const f = fixture(), wait = deferred(), flow = f.flow(() => wait.promise);
  const request = flow.submit({ sensitive: 'draft' });
  flow.clearPayload();
  assert.equal(flow.busy, false); assert.equal(flow.unknown, true); assert.equal(flow.attempt, undefined);
  assert.deepEqual(f.channel.read(), [reference('submission-1')]);
  wait.resolve(terminal('submission-1'));
  assert.equal((await request).kind, 'ignored');
  assert.equal(flow.resolved(reference('submission-1'), terminal('submission-1')), true);
  assert.equal(flow.saved, true); assert.equal(flow.unknown, false);
});
