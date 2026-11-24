/** Display only: pure business dates retain the API day, without timezone conversion. */
export function businessDate(value?: string | null) {
  if (!value) return '未提供';
  const parts = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  return parts ? value : '日期不可用';
}

export function businessPeriod(start?: string | null, endExclusive?: string | null, endInclusive?: string | null) {
  if (start && endExclusive === null) return `自 ${businessDate(start)} 起`;
  if (start && endExclusive && endInclusive) return `${businessDate(start)} 至 ${businessDate(endInclusive)}`;
  return '期间不可用';
}

const shanghai = new Intl.DateTimeFormat('zh-CN', {
  timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit',
  hour: '2-digit', minute: '2-digit', second: '2-digit', hourCycle: 'h23',
});
export function recordedTime(value?: string | null) {
  const date = value ? new Date(value) : null;
  if (!date || Number.isNaN(date.getTime())) return '记录时间不可用';
  const parts = Object.fromEntries(shanghai.formatToParts(date).map(p => [p.type, p.value]));
  return `${parts.year.padStart(4, '0')}-${parts.month}-${parts.day} ${parts.hour}:${parts.minute}:${parts.second}（北京时间 UTC+08:00）`;
}
