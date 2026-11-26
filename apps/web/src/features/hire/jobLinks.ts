const registerKeys = [
  'ref',
  'page',
  'pageSize',
  'sort',
  'kind',
  'status',
  'department',
  'fromDepartment',
  'toDepartment',
  'operator',
  'businessFrom',
  'businessTo',
  'recordedFrom',
  'recordedTo',
  'selected',
];
const id = '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}';
const registerDetail = new RegExp('^/personnel/actions/' + id + '$', 'i');
const assignmentDetail = new RegExp('^/personnel/employees/' + id + '/assignments/' + id + '$', 'i');

function pick(params: URLSearchParams, names: string[]) {
  const result = new URLSearchParams();
  for (const name of names) {
    const value = params.get(name);
    if (value) result.set(name, value);
  }
  return result;
}
export const registerListParams = (params: URLSearchParams) => pick(params, registerKeys);

export function profileReturn(target: string | null | undefined, person: string) {
  if(!target || !new RegExp('^'+id+'$','i').test(person))return undefined;
  const path='/personnel/employees/'+person;
  if(target!==path && !target.startsWith(path+'?'))return undefined;
  const url=new URL(target,location.origin);
  const query=pick(url.searchParams,['ref','page','pageSize','sort','selected','department','primary','from','assignment','assignmentDay','tab']);
  return path+(query.size?'?'+query:'');
}
export function employmentHistoryReturn(target: string | null | undefined, person: string) {
  const path = '/personnel/employees/' + person + '/employment';
  if (!target?.startsWith(path + '?')) return undefined;
  const url = new URL(target, location.origin);
  if (url.origin !== location.origin || url.pathname !== path || url.searchParams.get('view') !== 'history') return undefined;
  const query = pick(url.searchParams, [...registerKeys, 'view', 'action', 'assignmentDay', 'from', 'person', 'primary', 'assignment', 'employer']);
  const parent = personnelReturn(url.searchParams.get('returnTo')) || profileReturn(url.searchParams.get('returnTo'),person);
  if (parent) query.set('returnTo', parent);
  return path + '?' + query;
}

export function personnelReturn(target?: string | null) {
  if (!target || !target.startsWith('/personnel/')) return undefined;
  const url = new URL(target, location.origin);
  if (
    url.origin !== location.origin ||
    !(registerDetail.test(url.pathname) || assignmentDetail.test(url.pathname))
  )
    return undefined;
  const allowed = [
    ...registerKeys,
    'fromRegister',
    'returnAction',
    'assignmentDay',
    'action',
    'adoptedRef',
    'sourceVersion',
    'sourceEntry',
    'from',
    'view',
    'person',
    'primary',
    'assignment',
    'employer',
  ];
  const query = new URLSearchParams();
  url.searchParams.forEach((value, name) => {
    if (allowed.includes(name)) query.set(name, value);
  });
  const parent = url.searchParams.get('returnTo');
  if (assignmentDetail.test(url.pathname) && parent?.startsWith('/personnel/actions/')) {
    const action = new URL(parent, location.origin);
    if (
      action.origin === location.origin &&
      registerDetail.test(action.pathname) &&
      action.pathname === '/personnel/actions/' + query.get('returnAction')
    ) {
      const origin = pick(action.searchParams, allowed);
      query.set('returnTo', action.pathname + (origin.size ? '?' + origin : ''));
    }
  }
  return url.pathname + (query.size ? '?' + query : '');
}
export function adoptedOfferURL(offer: string, version: string, returnTo: string) {
  const query = new URLSearchParams({ sourceVersion: version });
  const back = personnelReturn(returnTo);
  if (back) query.set('returnTo', back);
  return '/recruiting/offers/' + encodeURIComponent(offer) + '?' + query;
}
