import { lazyCredentials } from './credentials.mjs';
import { deferAutomaticConfirmation } from './fixtures';
import { readRecord } from './fixtures';
import {readCurrentDate,  test, expect, openMoreActions, expectCurrentDate, type Page } from "./fixtures";
import { readFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { randomUUID } from "node:crypto";
const credentials = lazyCredentials();
const listPath="/settings/organization/business-units";
async function menu(page:Page) {
 await page.getByRole("button",{name:"切换应用",exact:true}).click();
 await page.getByRole("menuitem",{name:"设置",exact:true}).click();
 await page.getByRole("button",{name:"组织配置",exact:true}).click();
 await page.getByRole("menuitem",{name:"业务单元",exact:true}).click();
 expect(await readCurrentDate(page)).toMatch(/^\d{4}-\d{2}-\d{2}$/);
}
async function open(page:Page) {
 await new Promise(r=>setTimeout(r,3100));
 await page.goto("/");await page.getByLabel("登录名").fill(credentials.login);await page.getByLabel("密码").fill(credentials.password);await page.getByRole("button",{name:"登录",exact:true}).click();await menu(page);
 return readCurrentDate(page);
}
async function create(page:Page) {
 const code="LIFE_"+randomUUID().slice(0,8);
 await page.getByRole("link",{name:"新建",exact:true}).click();
 await page.getByRole("textbox",{name:"编码",exact:true}).fill(code);
 await page.getByRole("textbox",{name:"中文名称",exact:true}).fill("甲");
 await page.getByRole("textbox",{name:"英文名称",exact:true}).fill("Alpha");
 await page.getByLabel("生效日期").and(page.locator("input")).fill("2026-01-01");
 await page.getByRole("combobox",{name:"操作原因",exact:true}).click();await page.getByRole("option",{name:"新设组织",exact:true}).click();
 await page.getByRole("button",{name:"保存",exact:true}).click();await expect(page.locator(".object-form")).toHaveCount(0);await expect(page.getByRole("button",{name:"更多操作",exact:true})).toBeEnabled();
 return code;
}
function shift(date:string,days:number) { const value=new Date(date+"T12:00:00Z");value.setUTCDate(value.getUTCDate()+days);return value.toISOString().slice(0,10); }
function dateLabel(date:string) {const [y,m,d]=date.split("-");return `${y}-${m}-${d}`;}
async function form(page:Page,kind:"DEACTIVATE"|"REACTIVATE",date:string,name?:string) {
 await openMoreActions(page);
 await page.getByRole("menuitem",{name:kind==="DEACTIVATE"?"停用业务单元":"重新启用业务单元",exact:true}).click();
 await page.getByLabel("生效日期").and(page.locator("input")).fill(date);
 await page.getByRole("combobox",{name:"操作原因",exact:true}).click();
 await page.getByRole("option",{name:kind==="DEACTIVATE"?"业务终止":"业务恢复",exact:true}).click();
 if(kind==="DEACTIVATE") await expect(page.getByRole("textbox",{name:"中文名称",exact:true})).toHaveCount(0);
 if(name!==undefined) await page.getByRole("textbox",{name:"中文名称",exact:true}).fill(name);
}
async function save(page:Page,kind:"DEACTIVATE"|"REACTIVATE") {
 await expect(page.getByRole('button', {name:'预览',exact:true})).toHaveCount(0);
 const response=page.waitForResponse(r=>r.request().method()==="POST"&&r.url().endsWith("/"+kind.toLowerCase()));
 await page.getByRole("button",{name:kind==="DEACTIVATE"?"确认停用":"确认重新启用",exact:true}).click();
 const result=await response;expect(result.status()).toBe(201);const body=await result.json();await expect(page.locator(".object-form")).toHaveCount(0);await expect(page.getByRole("button",{name:"更多操作",exact:true})).toBeEnabled();return body;
}
async function read(page: Page, date?: string, source?: string) { return readRecord(page, date, source); }
async function correct(page:Page,date:string,name:string) {
 await page.getByRole("button",{name:dateLabel(date),exact:true}).click();await openMoreActions(page);await page.getByRole("menuitem",{name:"更正记录",exact:true}).click();await page.getByRole("textbox",{name:"中文名称",exact:true}).fill(name);await expect(page.getByRole('button', {name:'预览',exact:true})).toHaveCount(0);await page.getByRole("button",{name:"保存",exact:true}).click();await expect(page.locator(".object-form")).toHaveCount(0);await expect(page.getByRole("button",{name:"更多操作",exact:true})).toBeEnabled();
}

test("正常菜单今日启用安排未来多轮启停、准确继承、独立恢复快照及旧来源",async({page})=>{
 const today=await open(page);await create(page);
 const s1=shift(today,2),r1=shift(today,3),s2=shift(today,4),r2=shift(today,5);
 await form(page,"DEACTIVATE",s1);const stopped=await save(page,"DEACTIVATE");
 await expect(page.getByText("所选记录尚未生效。",{exact:true})).toBeVisible();await expect(page.locator(".object-detail-field").filter({has:page.getByText("状态",{exact:true})})).toContainText("停用");
 await expect(page.getByRole("button",{name:dateLabel(s1),exact:true})).toHaveAttribute("aria-current","date");
 await openMoreActions(page);await page.getByRole("menuitem",{name:"更正记录",exact:true}).click();await expect(page.getByRole("textbox",{name:"中文名称",exact:true})).toBeDisabled();await expect(page.getByLabel("生效日期").and(page.locator("input"))).toHaveValue(s1);await page.getByRole("button",{name:"取消编辑",exact:true}).click();
 await form(page,"REACTIVATE",r1,"乙");await save(page,"REACTIVATE");
 await form(page,"DEACTIVATE",s2);await save(page,"DEACTIVATE");
 await form(page,"REACTIVATE",r2);const last=await save(page,"REACTIVATE");expect(last.outcome).toBe("accepted");
 await correct(page,"2026-01-01","甲2");await correct(page,r1,"乙2");
 for(const [date,state,name] of [[s1,"inactive","甲2"],[r1,"active","乙2"],[s2,"inactive","乙2"],[r2,"active","乙"]]) {const fact=(await read(page,date)).selectedRecord;expect(fact.state).toBe(state);expect(fact.name).toBe(name);expect(fact.nameEn).toBe("Alpha");}
 const old=await read(page,s1,stopped.sourceVersion);expect(old.selectedRecord.name).toBe("甲");expect(old.selectedRecord.endExclusive).toBeNull();
 await page.reload();await expectCurrentDate(page,today);
 await page.getByRole("tab",{name:"变更日志",exact:true}).click();await page.getByRole("link",{name:"查看当次资料",exact:true}).last().click();await expect(page.getByText("只读",{exact:false}).first()).toBeVisible();await expect(page.getByRole('button',{name:'更多操作',exact:true})).toHaveCount(0);await expect(page.getByRole("menuitem",{name:"停用业务单元",exact:true})).toHaveCount(0);await expect(page.getByRole("menuitem",{name:"重新启用业务单元",exact:true})).toHaveCount(0);
});

test("当天停用、未来恢复不提前生效与未保存保护，过去启停仍可办理",async({page})=>{
 const today=await open(page);await create(page);
 await form(page,"DEACTIVATE",today);await page.getByRole("button",{name:"取消编辑",exact:true}).click();await expect(page.getByRole("dialog")).toBeVisible();await page.getByRole("button",{name:"继续编辑",exact:true}).click();await expect(page.getByLabel("生效日期").and(page.locator("input"))).toHaveValue(today);await save(page,"DEACTIVATE");
 await expect(page.getByText(/当日状态：/)).toHaveCount(0);
 await form(page,"REACTIVATE",shift(today,-1));await expect(page.getByRole("button",{name:"生效日期须晚于末条记录日期并在允许范围内",exact:true})).toBeVisible();await expect(page.getByRole("button",{name:"确认重新启用",exact:true})).toBeDisabled();await expect(page.getByLabel("生效日期").and(page.locator("input"))).toHaveValue(shift(today,-1));
 await page.getByLabel("生效日期").and(page.locator("input")).fill(shift(today,1));await page.getByRole("textbox",{name:"中文名称",exact:true}).fill("恢复𠮷祥事业部");await save(page,"REACTIVATE");await expect(page.getByText(/当日状态：/)).toHaveCount(0);expect((await read(page,shift(today,1))).selectedRecord.name).toBe("恢复𠮷祥事业部");
 await page.goto(listPath);await create(page);await form(page,"DEACTIVATE","2026-03-01");await save(page,"DEACTIVATE");await form(page,"REACTIVATE","2026-04-01","过去恢复");await save(page,"REACTIVATE");expect((await read(page,"2026-03-15")).selectedRecord.state).toBe("inactive");expect((await read(page,"2026-04-01")).selectedRecord.name).toBe("过去恢复");
});

test("停用与重新启用丢响应后的刷新查回定位原记录，原来源只读",async({page})=>{
  await deferAutomaticConfirmation(page);
 await open(page);await create(page);
 for(const [kind,date] of [["DEACTIVATE","2026-03-01"],["REACTIVATE","2026-04-01"]] as const) {
  await form(page,kind,date);await expect(page.getByRole('button', {name:'预览',exact:true})).toHaveCount(0);
  let result:any;const pattern=`**/api/v1/business-units/*/${kind.toLowerCase()}`;
  await page.route(pattern,async route=>{const response=await route.fetch();expect(response.status()).toBe(201);result=await response.json();await route.abort("failed")});
  const failed=page.waitForEvent("requestfailed",r=>r.url().endsWith("/"+kind.toLowerCase()));await page.getByRole("button",{name:kind==="DEACTIVATE"?"确认停用":"确认重新启用",exact:true}).click();await failed;await expect(page.getByText(/暂时无法确认保存结果/,{exact:true})).toBeVisible();
  await page.reload();await page.getByRole("button",{name:"查看保存结果",exact:true}).click();await expect(page.getByText("已保存", { exact: true })).toBeVisible();await page.getByRole("link",{name:"查看保存结果",exact:true}).click();await expect(page).toHaveURL(u=>u.searchParams.get("sourceVersion")===result.sourceVersion&&u.searchParams.get("selectedRecordId")===result.selectedRecordId);await expect(page.getByRole('button',{name:'更多操作',exact:true})).toHaveCount(0);await expect(page.getByRole("menuitem",{name:"重新启用业务单元",exact:true})).toHaveCount(0);await expect(page.getByRole("menuitem",{name:"停用业务单元",exact:true})).toHaveCount(0);
  await page.unroute(pattern);await page.getByRole("button",{name:"返回当前资料",exact:true}).click();await expect(page.locator(".object-form")).toHaveCount(0);await expect(page.getByRole("button",{name:"更多操作",exact:true})).toBeEnabled();
 }
});

test("单条越过末条日期就地反馈并保留输入，调整合法后直接确认",async({page})=>{
 await open(page);await create(page);
 await page.getByRole("button",{name:/^(新增记录|插入记录)$/,exact:true}).click();
 await page.getByLabel("生效日期").and(page.locator("input")).fill("2026-10-01");
 await page.getByRole("combobox",{name:"操作原因",exact:true}).click();await page.getByRole("option",{name:"名称调整",exact:true}).click();
 await page.getByRole("textbox",{name:"中文名称",exact:true}).fill("后继资料");
 await expect(page.getByRole('button', {name:'预览',exact:true})).toHaveCount(0);await page.getByRole("button",{name:"保存",exact:true}).click();await expect(page.locator(".object-form")).toHaveCount(0);await expect(page.getByRole("button",{name:"更多操作",exact:true})).toBeEnabled();
 await form(page,"DEACTIVATE","2026-03-01");
 await expect(page.getByRole("button",{name:"生效日期须晚于末条记录日期并在允许范围内",exact:true})).toBeVisible();
 // Automatic assistance preserves the last input focus. The explicit error
 // link still lets the user move directly to the invalid date.
 await expect(page.getByRole("combobox",{name:"操作原因",exact:true})).toBeFocused();
 await page.getByRole("button",{name:"生效日期须晚于末条记录日期并在允许范围内",exact:true}).click();
 await expect(page.getByLabel("生效日期").and(page.locator("input"))).toBeFocused();
 await expect(page.getByLabel("生效日期").and(page.locator("input"))).toHaveValue("2026-03-01");await expect(page.getByRole("button",{name:"确认停用",exact:true})).toBeDisabled();
 await page.getByLabel("生效日期").and(page.locator("input")).fill("2026-12-01");await save(page,"DEACTIVATE");
 expect((await read(page,"2026-12-01")).selectedRecord.name).toBe("后继资料");expect((await read(page,"2026-10-01")).selectedRecord.state).toBe("active");
});

test("查询者正常菜单可读资料但不开放生命周期入口",async({page})=>{
 await open(page);const code=await create(page);
 await page.getByRole("button",{name:"退出",exact:true}).click();
 await page.getByLabel("登录名").fill(credentials.reader_login);await page.getByLabel("密码").fill(credentials.reader_password);await page.getByRole("button",{name:"登录",exact:true}).click();await menu(page);
 await page.getByLabel("搜索编码或名称").fill(code);await page.getByRole("button",{name:"查询",exact:true}).click();
 await page.getByRole("row").filter({hasText:code}).getByRole("link",{name:"甲",exact:true}).click();
 await expect(page.getByRole("tab",{name:"基本资料",exact:true})).toBeVisible();
 await expect(page.getByRole("button",{name:"更多操作",exact:true})).toHaveCount(0);
 await expect(page.getByRole("menuitem",{name:"停用业务单元",exact:true})).toHaveCount(0);await expect(page.getByRole("menuitem",{name:"重新启用业务单元",exact:true})).toHaveCount(0);
});

test("重新启用撤字段读权后清除默认和详情缓存、保留可读输入并忽略旧响应", async ({page}) => {
 await open(page);const code=await create(page);
 await form(page,"DEACTIVATE","2026-03-01");await save(page,"DEACTIVATE");
 await form(page,"REACTIVATE","2026-04-01");
 await expect(page.getByRole("textbox",{name:"英文名称",exact:true})).toHaveValue("Alpha");
 await page.getByRole("textbox",{name:"中文名称",exact:true}).fill("保留中文");
 await page.getByRole("textbox",{name:"英文名称",exact:true}).fill("Wanted");
 // Real DB permission changes use the local runtime's container and policy lock.
 // No test-only management endpoint or credentials are added to the application.
 const subject=await page.evaluate(async()=> (await (await fetch("/api/v1/session")).json()).subjectId);
 for(const id of [subject,credentials.instance]) expect(id).toMatch(/^[0-9a-f-]{36}$/);
 const credentialPath=execFileSync("python3",["scripts/local-bu.py","credentials-path"],{encoding:"utf8"}).trim();
 const key=credentialPath.split("/").at(-2)!;
 expect(key).toMatch(/^[0-9a-f]{12}$/);
 const container=execFileSync("docker",["ps","-q","--filter",`label=com.docker.compose.project=chnhcm-bu-${key}`,"--filter","label=com.docker.compose.service=postgres"],{encoding:"utf8"}).trim();
 expect(container).toMatch(/^[0-9a-f]+$/);
 const sql=(query:string)=>execFileSync("docker",["exec","-i",container,"psql","-U","postgres","-d","chnhcm","-X","-Atq","-v","ON_ERROR_STOP=1"],{input:query,encoding:"utf8"}).trim();
 const where=`instance_id='${credentials.instance}' AND principal_id='${subject}'`;
 const grants=sql(`SELECT json_agg(g) FROM (SELECT grant_id,read_fields,write_fields FROM access.subject_grant WHERE ${where}) g;`);
 const mutate=(query:string)=>sql(`BEGIN;SELECT principal_id FROM access.policy_guard WHERE ${where} FOR UPDATE;${query};UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE ${where};COMMIT;`);
 let release!:()=>void;
 let received!:()=>void;
 const held=new Promise<void>(r=>{release=r});
 const fetched=new Promise<void>(r=>{received=r});
 let first=true;
 const pattern="**/api/v1/business-units/*/preview";
 await page.route(pattern,async route=>{
  if(!first) {await route.continue();return;}
  first=false;
  const response=await route.fetch();expect(response.status()).toBe(200);
  received();await held;await route.fulfill({response});
 });
 try {
  await page.getByRole("textbox",{name:"中文名称",exact:true}).fill("保留中文1");await fetched;
  mutate(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'nameEn') WHERE ${where} AND action IN ('bu.read','bu.history.read');UPDATE access.subject_grant SET write_fields=array_remove(write_fields,'nameEn') WHERE ${where} AND action='bu.reactivate'`);
  const denied=page.waitForResponse(r=>r.request().method()==="POST"&&r.url().endsWith("/preview")&&r.status()===403);
  await page.getByRole("textbox",{name:"中文名称",exact:true}).fill("保留中文");await denied;
  await expect(page.getByText("已重读当前资料，请核对后保存。",{exact:true})).toHaveCount(0);
  const checkCleared=async()=>{
   await expect(page.getByRole("textbox",{name:"英文名称",exact:true})).toHaveCount(0);
   await expect(page.locator("body")).not.toContainText("Alpha");
   await expect(page.locator("body")).not.toContainText("Wanted");
   await expect(page.getByRole("textbox",{name:"中文名称",exact:true})).toHaveValue("保留中文");
   await expect(page.getByRole("button",{name:"确认重新启用",exact:true})).toBeEnabled();
  };
  await checkCleared();
  const late=page.waitForResponse(r=>r.request().method()==="POST"&&r.url().endsWith("/preview")&&r.status()===200);
  release();await late;
  // Flush browser rendering after the old response was delivered.
  await page.evaluate(()=>new Promise<void>(r=>requestAnimationFrame(()=>requestAnimationFrame(()=>r()))));
  await checkCleared();await page.unroute(pattern);
  const screenshot=test.info().outputPath("field-revocation-masked.png");
  await page.screenshot({path:screenshot,mask:[page.locator(".app-account-name")]});
  await test.info().attach("撤权后缓存清理",{path:screenshot,contentType:"image/png"});
  const result=await save(page,"REACTIVATE");
  const fresh=await read(page,"2026-04-01");
  expect(fresh.capabilities.readFields).not.toContain("nameEn");
  expect(fresh.selectedRecord.nameEn).toBeUndefined();
  expect(fresh.selectedRecord.name).toBe("保留中文");
  console.log(JSON.stringify({code,buId:result.buId,sourceVersion:result.sourceVersion,readFields:fresh.capabilities.readFields,cachedEnglish:false,readableInputSaved:true,latePreviewIgnored:true}));
 } finally {
  release();await page.unroute(pattern);
  // Restore the exact original fields, including their order, even after failure.
  mutate(`UPDATE access.subject_grant g SET read_fields=s.read_fields,write_fields=s.write_fields FROM json_to_recordset('${grants.replaceAll("'","''")}') AS s(grant_id uuid,read_fields text[],write_fields text[]) WHERE g.instance_id='${credentials.instance}' AND g.grant_id=s.grant_id`);
 }
 expect((await read(page,"2026-04-01")).selectedRecord.nameEn).toBe("Alpha");
});


test("单条入口按末条生效后状态，进入带值边界与真实编辑取消保护",async({page})=>{
 const today=await open(page);await create(page);
 await openMoreActions(page);
 await expect(page.getByRole('menuitem',{name:'停用业务单元',exact:true})).toBeVisible();
 await expect(page.getByRole('menuitem',{name:'重新启用业务单元',exact:true})).toHaveCount(0);
 await page.getByRole('menuitem',{name:'停用业务单元',exact:true}).click();
 const date=page.getByLabel('生效日期').and(page.locator('input'));
 await expect(date).toHaveValue(today);await expect(date).toHaveAttribute('min','2026-01-02');await expect(date).toHaveAttribute('max','9999-12-31');
 await expect(page.getByRole('combobox',{name:'操作原因',exact:true}).locator('..').locator('input')).toHaveValue('');
 await page.getByRole('button',{name:'取消编辑',exact:true}).click();await expect(page.getByRole('dialog')).toHaveCount(0);await expect(page.locator('.object-form')).toHaveCount(0);
 const stop=shift(today,2),resume=shift(today,3);
 await form(page,'DEACTIVATE',stop);await save(page,'DEACTIVATE');
 await expect(page.locator('.object-detail-field').filter({has:page.getByText('状态',{exact:true})})).toContainText('停用');await expect(page.getByText('所选记录尚未生效。',{exact:true})).toBeVisible();expect((await read(page)).status).toBe('active');
 await openMoreActions(page);await expect(page.getByRole('menuitem',{name:'停用业务单元',exact:true})).toHaveCount(0);await page.getByRole('menuitem',{name:'重新启用业务单元',exact:true}).click();
 await expect(date).toHaveValue(resume);await expect(date).toHaveAttribute('min',resume);await expect(date).toHaveAttribute('max','9999-12-31');
 await expect(page.getByRole('textbox',{name:'中文名称',exact:true})).toHaveValue('甲');await expect(page.getByRole('textbox',{name:'英文名称',exact:true})).toHaveValue('Alpha');
 await expect(page.getByRole('combobox',{name:'操作原因',exact:true}).locator('..').locator('input')).toHaveValue('BUSINESS_RESUMPTION');
 await page.getByRole('button',{name:'取消编辑',exact:true}).click();await expect(page.getByRole('dialog')).toHaveCount(0);
 await form(page,'REACTIVATE',resume,'恢复草稿');await page.getByRole('button',{name:'取消编辑',exact:true}).click();
 await expect(page.getByRole('dialog')).toBeVisible();await page.getByRole('button',{name:'继续编辑',exact:true}).click();await expect(page.getByRole('textbox',{name:'中文名称',exact:true})).toHaveValue('恢复草稿');
 await page.getByRole('button',{name:'取消编辑',exact:true}).click();await page.getByRole('button',{name:'放弃修改',exact:true}).click();await expect(page.locator('.object-form')).toHaveCount(0);
 await page.getByRole('button',{name:dateLabel('2026-01-01'),exact:true}).click();
 await openMoreActions(page);await expect(page.getByRole('menuitem',{name:'停用业务单元',exact:true})).toHaveCount(0);await expect(page.getByRole('menuitem',{name:'重新启用业务单元',exact:true})).toHaveCount(0);
 await expect(page.getByRole('menuitem',{name:'插入停用与重新启用',exact:true})).toBeVisible();await page.keyboard.press('Escape');
 const d=await read(page,today);expect(d.timeline).toHaveLength(2);expect(d.timeline.at(-1)).toMatchObject({effectiveDate:stop,state:'inactive',name:'甲'});
});


test("单条来源冲突重读保留日期和原锚点，明确重选末条后才能追加",async({page})=>{
 const today=await open(page);await create(page);await form(page,'DEACTIVATE','2028-09-01');
 const status=await page.evaluate(async()=>{
  const id=location.pathname.split('/').at(-1),s=await(await fetch('/api/v1/session')).json(),d=await(await fetch('/api/v1/business-units/'+id)).json();
  return(await fetch('/api/v1/business-units/'+id+'/records',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':s.csrf,'Idempotency-Key':crypto.randomUUID()},body:JSON.stringify({anchorRecordId:d.timeline.at(-1).recordId,expectedSourceVersion:d.headSourceVersion,effectiveDate:'2028-06-01',operationType:'CHANGE',operationReason:'NAME_ADJUSTMENT',changes:{name:'并发资料'}})})).status;
 });expect(status).toBe(201);
 await page.getByLabel('生效日期').and(page.locator('input')).fill('2028-09-02');
 await expect(page.getByText('资料已更新，请核对后保存',{exact:true})).toBeVisible();await expect(page.getByRole('button',{name:'确认停用',exact:true})).toBeDisabled();
 await page.getByRole('button',{name:'重新核对当前资料',exact:true}).click();
 await expect(page.getByText('此处无法完成当前操作，可取消编辑',{exact:true})).toBeVisible();
 await expect(page.getByLabel('生效日期').and(page.locator('input'))).toHaveValue('2028-09-02');await expect(page.getByRole('button',{name:'确认停用',exact:true})).toBeDisabled();
 await page.getByRole('button',{name:'取消编辑',exact:true}).click();await page.getByRole('button',{name:'放弃修改',exact:true}).click();
 await page.getByRole('button',{name:dateLabel('2028-06-01'),exact:true}).click();await form(page,'DEACTIVATE','2028-09-02');await save(page,'DEACTIVATE');
 const d=await read(page,today);expect(d.timeline).toHaveLength(3);expect(d.timeline.at(-1)).toMatchObject({effectiveDate:'2028-09-02',state:'inactive',name:'并发资料'});
});
