import { expect, type Page } from './fixtures';
import { execFileSync } from 'node:child_process';

export async function accessFixture(page:Page){
 const session=await page.evaluate(async()=>{const s=await(await fetch('/api/v1/session')).json();return {subject:s.subjectId,instance:s.instanceId}});
 for(const id of [session.subject,session.instance])expect(id).toMatch(/^[0-9a-f-]{36}$/);
 const connection=JSON.parse(execFileSync('python3',['scripts/local-bu.py','connection'],{encoding:'utf8'}));
 const container=execFileSync('docker',['ps','-q','--filter','label=com.docker.compose.project='+connection.project,'--filter','label=com.docker.compose.service=postgres'],{encoding:'utf8'}).trim();expect(container).toMatch(/^[0-9a-f]+$/);
 const sql=(q:string)=>execFileSync('docker',['exec','-i',container,'psql','-U','postgres','-d','chnhcm','-X','-Atq','-v','ON_ERROR_STOP=1'],{input:q,encoding:'utf8'}).trim();
 const where=`instance_id='${session.instance}' AND principal_id='${session.subject}'`;
 const grants=sql(`SELECT json_agg(g) FROM access.subject_grant g WHERE ${where}`);
 const mutate=(q:string)=>sql(`BEGIN;SELECT principal_id FROM access.policy_guard WHERE ${where} FOR UPDATE;${q};UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE ${where};COMMIT;`);
 return {sql,where,session,mutate,restore:()=>mutate(`DELETE FROM access.subject_grant WHERE ${where};INSERT INTO access.subject_grant SELECT * FROM json_populate_recordset(NULL::access.subject_grant,'${grants.replaceAll("'","''")}')`)};
}
