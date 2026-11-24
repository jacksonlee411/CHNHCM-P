import test from 'node:test';
import assert from 'node:assert/strict';
import { businessDate, businessPeriod, recordedTime } from '../packages/ui/src/date-display.ts';

test('pure dates retain their API day and all four year digits', () => {
  for (const day of ['2026-01-13', '2028-02-29', '0001-01-01', '9999-12-31']) assert.equal(businessDate(day), day);
  assert.equal(businessDate(null), '未提供');
  assert.equal(businessDate('2026/01/13'), '日期不可用');
});

test('recorded time keeps Shanghai time and seconds across UTC midnight', () => {
  assert.equal(recordedTime('2026-01-12T16:30:05.123456Z'), '2026-01-13 00:30:05（北京时间 UTC+08:00）');
  assert.equal(recordedTime('2026-01-13T08:30:05+08:00'), '2026-01-13 08:30:05（北京时间 UTC+08:00）');
  assert.equal(recordedTime(undefined), '记录时间不可用');
  assert.equal(recordedTime('invalid'), '记录时间不可用');
});

test('only an explicit open end can be rendered as open', () => {
  const start = '2024-02-28';
  assert.equal(businessPeriod(start, null, null), '自 2024-02-28 起');
  assert.equal(businessPeriod(start, '2024-03-01', '2024-02-29'), '2024-02-28 至 2024-02-29');
  for (const args of [[start, undefined], [start, '2024-03-01', null], [null, null, null]]) {
    assert.equal(businessPeriod(...args), '期间不可用');
  }
});
