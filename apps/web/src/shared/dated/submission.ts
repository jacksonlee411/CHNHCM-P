/** Object-neutral submission identity protocol. Adapters own DTOs and decisions. */
export class SubmissionError extends Error {}
export const terminalOutcome = (value: unknown): boolean => ['accepted', 'rejected', 'no_change', 'not_applied'].includes(value as string);
export type Pending = { instanceRef: string; subjectRef: string; submissionId: string };
export type Owner = { instanceId: string; subjectId: string };
export type Outcome = { submissionId: string; outcome: 'accepted' | 'rejected' | 'no_change' | 'not_applied' | 'pending' };
export type RecoveryNotice<K extends string, R extends Outcome> = { kind: K; reference: Pending; result: R };
type StoragePort = Pick<Storage, 'getItem' | 'setItem'>;
export const sameReference = (a: Pending, b: Pending) => a.instanceRef === b.instanceRef && a.subjectRef === b.subjectRef && a.submissionId === b.submissionId;
export const ownedReference = (p: Pending, owner: Owner) => p.instanceRef === owner.instanceId && p.subjectRef === owner.subjectId;

export function submissionChannel<K extends string, R extends Outcome>(kind: K, key: string,
  platform: () => { storage: StoragePort; events: EventTarget } = () => ({ storage: localStorage, events: window })) {
  const changed = `chnhcm:${kind}:pending`, resolved = `chnhcm:${kind}:resolved`, denied = `chnhcm:${kind}:recovery-denied`;
  // A terminal already confirmed in this tab stays known even if clearing its
  // persistent reference fails. Refresh still discovers the retained identity.
  const settled = new Set<string>();
  const active = new Map<string, 'saving' | 'confirming'>();
  const notify = () => platform().events.dispatchEvent(new Event(changed));
  const identity = (p: Pending) => JSON.stringify([p.instanceRef, p.subjectRef, p.submissionId]);
  const read = (): Pending[] => {
    const value: unknown = JSON.parse(platform().storage.getItem(key) ?? '[]');
    if (!Array.isArray(value)) throw new Error('上次保存进度无法读取');
    // Persist only the three identity fields, never command bodies or results.
    const entries = value.flatMap(v => v && typeof v === 'object' && typeof v.instanceRef === 'string' && typeof v.subjectRef === 'string' && typeof v.submissionId === 'string'
      ? [{ instanceRef: v.instanceRef, subjectRef: v.subjectRef, submissionId: v.submissionId }] : []);
    const retained = new Set(entries.map(identity));
    for (const key of settled) if (!retained.has(key)) settled.delete(key);
    return entries.filter(p => !settled.has(identity(p)));
  };
  const write = (entries: Pending[]) => {
    platform().storage.setItem(key, JSON.stringify(entries));
    platform().events.dispatchEvent(new Event(changed));
  };
  return {
    kind, key, read,
    busy: (p: Pending) => active.has(identity(p)),
    phase: (p: Pending) => active.get(identity(p)),
    confirming(p: Pending) { active.set(identity(p), 'confirming'); notify(); },
    lock(p: Pending, phase: 'saving' | 'confirming' = 'confirming') { if (active.has(identity(p))) return false; active.set(identity(p), phase); notify(); return true; },
    unlock(p: Pending) { active.delete(identity(p)); notify(); },
    has: (p: Pending) => read().some(x => sameReference(x, p)),
    remember(p: Pending) { const entries = read(); if (!entries.some(x => sameReference(x, p))) entries.push({ instanceRef: p.instanceRef, subjectRef: p.subjectRef, submissionId: p.submissionId }); write(entries); },
    forget(p: Pending) {
      try { write(read().filter(x => !sameReference(x, p))); }
      catch {
        settled.add(identity(p));
        platform().events.dispatchEvent(new Event(changed));
      }
    },
    watch(listener: () => void) {
      const events = platform().events;
      events.addEventListener(changed, listener); events.addEventListener('storage', listener);
      return () => { events.removeEventListener(changed, listener); events.removeEventListener('storage', listener); };
    },
    publish(p: Pending, result: R) {
      if (result.submissionId !== p.submissionId || !terminalOutcome(result.outcome)) return;
      return !platform().events.dispatchEvent(new CustomEvent(resolved, { cancelable:true, detail: { kind, reference: p, result } satisfies RecoveryNotice<K, R> }));
    },
    deny(p: Pending) { platform().events.dispatchEvent(new CustomEvent(denied, { detail: { kind, reference: p } })); },
    subscribe(onResolved: (notice: RecoveryNotice<K, R>) => void | boolean, onDenied: (p: Pending) => void) {
      const events = platform().events;
      const accept = (event: Event) => {
        if (event instanceof CustomEvent) {
          const notice = event.detail as RecoveryNotice<K, R>;
          if (notice.kind === kind && notice.result.submissionId === notice.reference.submissionId && onResolved(notice) === true) event.preventDefault();
        }
      };
      const reject = (event: Event) => {
        if (event instanceof CustomEvent && event.detail.kind === kind) onDenied(event.detail.reference);
      };
      events.addEventListener(resolved, accept); events.addEventListener(denied, reject);
      return () => { events.removeEventListener(resolved, accept); events.removeEventListener(denied, reject); };
    },
  };
}
export type SubmissionChannel<K extends string, R extends Outcome> = ReturnType<typeof submissionChannel<K, R>>;
export type Attempt<C> = { id: string; command: C };
export type Failure = { error: unknown; definitive: boolean; wasUnknown: boolean };
type Completion<R> = { kind: 'saved'; result: R } | ({ kind: 'failed' } & Failure) | { kind: 'ignored' };

/** One state machine for create, single-record and fixed-pair submissions. */
export class SubmissionFlow<C, R extends Outcome, Wire> {
  attempt?: Attempt<C>;
  submittedId?: string;
  busy = false;
  unknown = false;
  saved = false;
  private serial = 0;
  private readonly autoConfirmed = new Set<string>();
  private readonly ports;
  constructor(ports: {
    owner: Owner; channel: Pick<SubmissionChannel<string, R>, 'read' | 'remember' | 'forget' | 'lock' | 'unlock' | 'publish' | 'confirming' | 'busy'>;
    current: () => () => boolean; changed: () => void; uuid: () => string;
    send: (command: C, id: string) => Promise<Wire>; unwrap: (wire: Wire) => R;
    definitive: (error: unknown) => boolean;
    resolve: (reference: Pending) => Promise<R>;
  }) { this.ports = ports; }
  get inFlight() { return this.busy || Boolean(this.submittedId && this.ports.channel.busy(this.reference(this.submittedId))); }
  reference(id: string): Pending { return { instanceRef: this.ports.owner.instanceId, subjectRef: this.ports.owner.subjectId, submissionId: id }; }
  clearPayload() {
    this.attempt = undefined; this.serial++;
    if (this.submittedId) {
      // Discarding sensitive input also abandons the in-flight response owner.
      // If navigation/logout fails, the mounted form must offer recovery.
      this.busy = false; this.unknown = true; this.ports.changed();
    }
  }
  resolved(reference: Pending, result: R) {
    if (!ownedReference(reference, this.ports.owner) || reference.submissionId !== this.submittedId || result.submissionId !== reference.submissionId || !terminalOutcome(result.outcome)) return false;
    this.clearPayload(); this.submittedId = undefined; this.busy = false; this.unknown = false;
    this.saved = result.outcome === 'accepted' || result.outcome === 'no_change'; this.ports.changed(); return true;
  }
  async submit(command: C): Promise<Completion<R>> {
    if (this.busy || this.saved) return { kind: 'ignored' };
    const current = this.ports.current(), wasUnknown = this.unknown;
    const attempt = this.attempt ?? { id: this.ports.uuid(), command: structuredClone(command) };
    const reference = this.reference(attempt.id);
    try {
      if (!this.attempt && this.ports.channel.read().some(p => ownedReference(p, this.ports.owner))) {
        return { kind: 'failed', error: new SubmissionError('请先查看上次保存结果'), definitive: false, wasUnknown };
      }
      this.ports.channel.remember(reference);
    } catch { return { kind: 'failed', error: new SubmissionError('暂时无法开始保存，请重试'), definitive: false, wasUnknown }; }
    if (!this.ports.channel.lock(reference, 'saving')) return { kind: 'ignored' };
    this.attempt = attempt; this.submittedId = attempt.id; this.busy = true;
    const serial = ++this.serial;
    const owns = () => current() && serial === this.serial && this.submittedId === attempt.id;
    this.ports.changed();
    try {
      const wire = await this.ports.send(attempt.command, attempt.id);
      if (!owns()) return { kind: 'ignored' };
      const result = this.ports.unwrap(wire);
      if (result.submissionId !== attempt.id || !['accepted', 'no_change'].includes(result.outcome)) throw new Error('暂时无法确认保存结果');
      this.attempt = undefined; this.submittedId = undefined; this.busy = false; this.unknown = false; this.saved = true;
      // The exact accepted result is authoritative. A cleanup failure leaves an
      // identity to recover later; it cannot make this confirmed write unknown.
      try { this.ports.channel.forget(reference); } catch { /* Keep the recoverable identity. */ }
      this.ports.changed(); return { kind: 'saved', result };
    } catch (error) {
      if (!owns()) return { kind: 'ignored' };
      let definitive = !wasUnknown && this.ports.definitive(error);
      if (definitive) {
        try { this.ports.channel.forget(reference); }
        catch { definitive = false; }
      }
      if (!definitive && !this.autoConfirmed.has(attempt.id)) {
        this.autoConfirmed.add(attempt.id);
        this.ports.channel.confirming(reference);
        this.unknown = true; this.ports.changed();
        try {
          const result = await this.ports.resolve(reference);
          if (!owns()) return { kind: 'ignored' };
          if (result.submissionId === attempt.id && terminalOutcome(result.outcome)) {
            this.ports.channel.forget(reference);
            this.ports.channel.publish(reference, result);
            return { kind: 'ignored' };
          }
        } catch { /* Keep the original identity and payload for manual recovery. */ }
        if (!owns()) return { kind: 'ignored' };
      }
      this.busy = false; this.unknown = !definitive;
      if (definitive) { this.attempt = undefined; this.submittedId = undefined; }
      this.ports.changed(); return { kind: 'failed', error, definitive, wasUnknown };
    } finally { this.ports.channel.unlock(reference); }
  }
}
