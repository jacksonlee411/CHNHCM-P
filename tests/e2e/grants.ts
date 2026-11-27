import {expect,type Page} from './fixtures';
import {execFileSync} from 'node:child_process';
import {readFileSync} from 'node:fs';

export async function withPolicy(page:Page,run:(where:string,mutate:(sql:string)=>void)=>Promise<void>){
 const credentials=JSON.parse(readFileSync(execFileSync('python3',['scripts/local-bu.py','credentials-path'],{encoding:'utf8'}).trim(),'utf8'));
 const connection=JSON.parse(execFileSync('python3',['scripts/local-bu.py','connection'],{encoding:'utf8'}));const container=execFileSync('docker',['ps','-q','--filter','label=com.docker.compose.project='+connection.project,'--filter','label=com.docker.compose.service=postgres'],{encoding:'utf8'}).trim();expect(container).toMatch(/^[0-9a-f]+$/);
 const subject=await page.evaluate(async()=>(await(await fetch('/api/v1/session')).json()).subjectId);for(const id of [subject,credentials.instance])expect(id).toMatch(/^[0-9a-f-]{36}$/);
 const sql=(input:string)=>execFileSync('docker',['exec','-i',container,'psql','-U','postgres','-d','chnhcm','-X','-Atq','-v','ON_ERROR_STOP=1'],{input,encoding:'utf8'}).trim();const where=`instance_id='${credentials.instance}' AND principal_id='${subject}'`;
 const snapshot=sql(`SELECT json_agg(g) FROM access.subject_grant g WHERE ${where}`);
 const mutate=(statement:string)=>{sql(`BEGIN;SELECT principal_id FROM access.policy_guard WHERE ${where} FOR UPDATE;${statement};UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE ${where};COMMIT;`)};
 try{await run(where,mutate)}finally{mutate(`INSERT INTO access.subject_grant SELECT * FROM json_populate_recordset(NULL::access.subject_grant,'${snapshot.replaceAll("'","''")}')`)}
}
