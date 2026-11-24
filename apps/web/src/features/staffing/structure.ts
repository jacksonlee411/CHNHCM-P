import { useEffect, useState } from 'react';
import { api, ApiError, sessionGuard, unwrap, type Models } from '@chnhcm/api-client';
import {submissionChannel,type Pending} from '../../shared/dated/submission';
import { paths } from './api';

export const structurePath = '/settings/staffing/position-hierarchies';
export type Node = Models['PositionStructureNode'];
export type Structure = Models['PositionStructure'];
export type Query = {
  mode: 'nodes' | 'search' | 'path' | 'relation' | 'versions' | 'candidates' | 'node';
  businessAt?: string; sourceVersion?: string; nodeId?: string; q?: string; page?: number; pageSize?: number;
};
export function structureQuery(query:Query, signal:AbortSignal) {
  return api.GET('/api/v1/position-hierarchies/reporting/structure', {params:{query},signal}).then(unwrap);
}
export function useStructure(query:Query, enabled=true, refresh=0) {
  const [result,setResult] = useState<{key:string; data?:Structure; error?:string; status?:number}>();
  const key = JSON.stringify([query,refresh,enabled]);
  useEffect(() => {
    if (!enabled) return;
    const controller = new AbortController(), same = sessionGuard();
    structureQuery(query,controller.signal).then(data => {
      if (same() && !controller.signal.aborted) setResult({key,data});
    }).catch(error => {
      if (same() && !controller.signal.aborted) setResult({key,status:error instanceof ApiError ? error.status:undefined,error:error instanceof Error ? error.message : '岗位结构暂时不可读'});
    });
    const clear = () => {controller.abort();setResult(undefined)};
    window.addEventListener('chnhcm-session-invalidated',clear);
    return () => {controller.abort();window.removeEventListener('chnhcm-session-invalidated',clear)};
  },[key]);
  return result?.key===key ? result : undefined;
}
export function nodeLabel(node: Pick<Node,'name'|'code'>) {return node.code ? node.name+'（'+node.code+'）' : node.name}
export function nodeHref(id:string, day:string, version:string, back:string) {
  const query = new URLSearchParams({tab:'relations',structureDate:day,back});
  if (version) query.set('structureVersion',version);
  return paths.position+'/'+id+'?'+query;
}

export const arrangementChannel=submissionChannel<string,Models['StaffingResult']>('staffing.positionhierarchy','chnhcm-pending-position-hierarchies');
export async function resolveArrangement(session:Models['Principal'], ref:Pending) {
  const result=unwrap(await api.POST('/api/v1/position-hierarchies/submissions/{id}/resolve',{
    params:{path:{id:ref.submissionId},header:{'X-CSRF-Token':session.csrf}}
  }));
  if(result.kind!=='positionhierarchy'||result.submissionId!==ref.submissionId) throw new Error('暂时无法确认保存结果');
  return result;
}
