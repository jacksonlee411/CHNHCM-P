import test from 'node:test';
import assert from 'node:assert/strict';
import { SubmissionFlow, submissionChannel } from '../apps/web/src/shared/dated/submission.ts';

function setup(resolve) {
  const values = new Map(), events = new EventTarget();
  const channel = submissionChannel('test', 'pending', () => ({ events, storage: {
    getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value),
  } }));
  let writes = 0, confirms = 0;
  const flow = new SubmissionFlow({
    owner: { instanceId: 'instance', subjectId: 'user' }, channel,
    uuid: () => 'original', current: () => () => true, changed() {}, unwrap: value => value,
    definitive: () => false,
    send: async () => { writes++; throw new Error('response lost'); },
    resolve: async reference => { confirms++; return resolve(reference, channel); },
  });
  channel.subscribe(({ reference, result }) => flow.resolved(reference, result), () => {});
  return { channel, flow, counts: () => [writes, confirms] };
}

test('response loss confirms once without a second write and retains the exact terminal', async () => {
  const f = setup(async ref => ({ submissionId: ref.submissionId, outcome: 'accepted' }));
  await f.flow.submit({ name: '原内容' });
  assert.deepEqual(f.counts(), [1, 1]);
  assert.equal(f.flow.saved, true);
  assert.equal(f.flow.unknown, false);
  assert.deepEqual(f.channel.read(), []);
  assert.equal((await f.flow.submit({ name: '新内容' })).kind, 'ignored');
});

test('pending confirmation does not poll or repeat automatically on an exact retry', async () => {
  const f = setup(async ref => ({ submissionId: ref.submissionId, outcome: 'pending' }));
  await f.flow.submit({ name: '原内容' });
  assert.equal(f.flow.unknown, true);
  await f.flow.submit({ name: '不可替换原内容' });
  assert.deepEqual(f.counts(), [2, 1]);
  assert.deepEqual(f.flow.attempt.command, { name: '原内容' });
  assert.equal(f.channel.read().length, 1);
});

test('automatic confirmation excludes manual confirmation and exact retry until it finishes', async () => {
  let finish;
  const wait = new Promise(resolve => { finish = resolve; });
  const f = setup(async (ref, channel) => {
    assert.equal(channel.phase(ref), 'confirming');
    assert.equal(channel.lock(ref), false);
    await wait;
    return { submissionId: ref.submissionId, outcome: 'not_applied' };
  });
  const request = f.flow.submit({ name: '原内容' });
  await Promise.resolve();
  assert.equal((await f.flow.submit({ name: '重复点击' })).kind, 'ignored');
  finish(); await request;
  assert.deepEqual(f.counts(), [1, 1]);
  assert.equal(f.flow.busy, false);
  assert.equal(f.flow.unknown, false);
  assert.deepEqual(f.channel.read(), []);
});

test('mismatched confirmation preserves the original reference and remains unknown', async () => {
  const f = setup(async () => ({ submissionId: 'another', outcome: 'accepted' }));
  await f.flow.submit({ name: '原内容' });
  assert.equal(f.flow.saved, false);
  assert.equal(f.flow.unknown, true);
  assert.equal(f.channel.read()[0].submissionId, 'original');
  assert.deepEqual(f.counts(), [1, 1]);
});

for (const outcome of [undefined, 'unrecognized']) test(`invalid confirmation outcome ${outcome} cannot settle an operation`, async () => {
  const f = setup(async ref => ({ submissionId: ref.submissionId, outcome }));
  await f.flow.submit({ name: '原内容' });
  assert.equal(f.flow.saved, false);
  assert.equal(f.flow.unknown, true);
  assert.equal(f.channel.read()[0].submissionId, 'original');
  assert.deepEqual(f.counts(), [1, 1]);
});
