import createClient from "openapi-fetch";
import type { paths, components } from "./schema";
// Responses retain the session generation under which their request began.
// A delayed 401 from a previous session cannot expire a newer login.
let sessionGeneration = 0;
const sessionKey = 'chnhcm-session-generation';
// Only a random invalidation token is shared, never an identity or credential.
const sharedGeneration = () => localStorage.getItem(sessionKey);
const responseGenerations = new WeakMap<Response, () => boolean>();
function invalidateSessionConsumers(broadcast: 'none' | 'requests' | 'session') {
  sessionGeneration++;
  if (broadcast !== 'none') localStorage.setItem(sessionKey, `${broadcast}:${crypto.randomUUID()}`);
  window.dispatchEvent(new Event('chnhcm-session-invalidated'));
}
export function sessionChanged(broadcast = false) {
  invalidateSessionConsumers(broadcast ? 'session' : 'none');
}
// Starting logout invalidates pending reads/downloads across tabs. Identity
// refresh waits for completion, so a failed logout cannot destroy another
// tab's unsaved form merely by remounting its authenticated application.
export function sessionChangeStarted() {
  invalidateSessionConsumers('requests');
}
window.addEventListener('storage', event => {
  if (event.storageArea === localStorage && (event.key === sessionKey || event.key === null)) {
    sessionChanged();
    if (!event.newValue?.startsWith('requests:')) window.dispatchEvent(new Event('chnhcm-session-changed-elsewhere'));
  }
});
// Async owners also use this guard for transport failures and local side effects.
export function sessionGuard() {
  const generation = sessionGeneration;
  const shared = sharedGeneration();
  // Check storage synchronously too: a response can settle before storage's
  // event is delivered to this tab.
  return () => generation === sessionGeneration && shared === sharedGeneration();
}
export function isCurrentSessionResponse(response: Response) {
  return responseGenerations.get(response)?.() === true;
}
export const api = createClient<paths>({
  baseUrl: "",
  credentials: "same-origin",
  fetch: async (request) => {
    const generation = sessionGuard();
    let response: Response;
    try { response = await fetch(request); }
    catch (error) { if (error instanceof DOMException && error.name === 'AbortError') throw error; throw new ApiError('NETWORK_ERROR', '网络连接失败，请重试', 503); }
    responseGenerations.set(response, generation);
    return response;
  },
});
export type Models = components["schemas"];
export class ApiError extends Error {
  constructor(
    public code: string,
    message: string,
    public status: number,
    public field?: string,
    public failureDetails?: Models["FailureDetails"],
    public readFields?: string[],
    public existingPersonId?: string,
  ) {
    super(message);
  }
}
export function unwrap<T>(result: {
  data?: T;
  error?: Models["Error"] & Pick<Models["PersonError"], "existingPersonId">;
  response: Response;
}): T {
  if (result.error || !result.response.ok) {
    if (result.error?.code === 'AUTH_REQUEST_VALIDATION_FAILED' && isCurrentSessionResponse(result.response)) window.dispatchEvent(new Event('chnhcm-request-validation-failed'));
    if (result.response.status === 401 && isCurrentSessionResponse(result.response))
      window.dispatchEvent(new Event("chnhcm-auth-lost"));
    throw new ApiError(
      result.error?.code ?? "SERVICE_ERROR",
      result.error?.message ?? "请求失败，请重试",
      result.response.status,
      result.error?.field ?? result.error?.failureDetails?.issues[0]?.locations[0]?.fieldPath,
      result.error?.failureDetails,
      result.error?.readFields,
      result.error?.existingPersonId,
    );
  }
  if (result.data === undefined)
    throw new ApiError("SERVICE_ERROR", "返回结果不完整", 503);
  return result.data;
}
