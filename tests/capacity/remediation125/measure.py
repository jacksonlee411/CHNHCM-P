#!/usr/bin/env python3
"""125 current-model lane. Counts, dates, identities and values come from private
source bundles; measured HTTP responses are never used as the read oracle.
Only the explicitly disposable storage126 candidate can be restarted/evicted.
"""
import argparse
import csv
import concurrent.futures
import copy
import gzip
import hashlib
import importlib.util
import json
import math
import os
import re
from pathlib import Path
import subprocess
import time
import uuid
import io
import urllib.request
import urllib.error
from datetime import datetime, date, timedelta
from zoneinfo import ZoneInfo

ROOT = Path(__file__).resolve().parents[3]
spec = importlib.util.spec_from_file_location('compact_case', ROOT/'tests/capacity/storage126/measure.py')
case = importlib.util.module_from_spec(spec)
spec.loader.exec_module(case)
ACTIONS = ['list', 'detail', 'source', 'save', 'pair', 'resolve']

def today():
    return datetime.now(ZoneInfo('Asia/Shanghai')).strftime('%Y-%m-%d')


def save(path, value):
    temporary = path.with_suffix('.writing')
    temporary.write_text(json.dumps(value, indent=2)+'\n')
    temporary.replace(path)


def operation_target_ms(kind, action, *, sessions):
    if kind == 'dep' and action == 'list':
        return {1: 300, 20: 1000}[sessions]
    return 300 if action in ['list', 'detail'] else 500


def stats(samples, target):
    values = sorted(s['ms'] for s in samples if s.get('ms') is not None)
    p95 = values[math.ceil(len(values)*.95)-1] if values else None
    return dict(requests=len(samples), timed=len(values), failures=sum(not s['ok'] for s in samples),
                httpErrors=sum(s.get('httpError', False) for s in samples),
                semanticErrors=sum(s.get('semanticError', False) for s in samples),
                p95MS=p95, maxMS=max(values) if values else None, targetMS=target,
                passed=bool(values) and all(s['ok'] for s in samples) and (not target or p95 <= target), samples=samples)


def parallel(jobs):
    with concurrent.futures.ThreadPoolExecutor(max_workers=20) as pool:
        return list(pool.map(lambda fn: fn(), jobs))


def measured(client, method, path, body=None, key=None, check=None):
    start = time.perf_counter()
    try:
        value, elapsed = client.call(method, path, body, key)
    except Exception as error:
        return dict(ok=False, ms=getattr(error,'elapsed_ms',(time.perf_counter()-start)*1000), httpError=not isinstance(error,json.JSONDecodeError),semanticError=isinstance(error,json.JSONDecodeError),
                    status=getattr(error,'status',None),code=getattr(error,'code',None),
                    errorType=type(error).__name__, request=getattr(client, 'request_id', None)), None
    sample = dict(ok=True, ms=elapsed, request=getattr(client, 'request_id', None))
    if check:
        try:
            check(value)
        except Exception as error:
            sample.update(ok=False, semanticError=True, errorType=type(error).__name__)
    return sample, value


def _failure_sample(error):
    status = getattr(error, 'status', None)
    http_error = status is not None or isinstance(error, (urllib.error.URLError, TimeoutError, ConnectionError))
    return dict(ok=False, ms=getattr(error, 'elapsed_ms', None), httpError=http_error,
                semanticError=not http_error, status=status, code=getattr(error, 'code', None),
                errorType=type(error).__name__)


class PreparationFailure(RuntimeError):
    def __init__(self, sample):
        super().__init__('resolve preparation failed')
        self.sample = sample


class Lane:
    def __init__(self, private, output, binary):
        case.BIN = binary
        self.private, self.output, self.binary = private, output, binary
        self.oracle = case.Oracle(private)
        # Resolve frame metadata once, outside all timed traffic.
        self.codes = {k: sorted(o['code'] for o in self.oracle.owners[k]) for k in ['bu','dep']}
        self.current = {}
        for code in self.oracle.bycode:
            kind=self.oracle.bycode[code][0]
            background=re.fullmatch('R6_[BD]_[0-9]{5}',code) is not None
            # Clone templates share their values; only typed UUID identities
            # are remapped. Do not recursively copy operation/history evidence
            # merely to prepare a current-list expectation.
            frame=(self.oracle.proto[kind]['Frames'][-1] if background else self.oracle.frame(code))['State']['Timeline']
            applicable=[r for r in frame['Records'] if r['Date']<=today() and (not r['End'] or today()<r['End'])]
            record=applicable[0] if applicable else frame['Records'][0]
            self.current[code]=dict(name=record['Name'] if kind=='bu' else record['Data']['Name'], date=record['Date'],
                                    shortName=record.get('Data',{}).get('ShortName'),parent=record.get('Data',{}).get('Parent',''),
                                    status=record['State'] if applicable else 'not_effective',
                                    bu=(case.mapped(f'R6_B_{int(code[-5:])%2000:05d}:',self.oracle.proto['bu']['Owner']['bu_id']) if background else record['Data']['BU']) if kind=='dep' else None)
        self.detailExpected={}
        self.byid={o['bu_id' if k=='bu' else 'department_id']:code for code,(k,o) in self.oracle.bycode.items()}
        with gzip.open(private/'organization.enterprise.jsonl.gz','rt') as enterprise_file:
            enterprises=[json.loads(line) for line in enterprise_file]
        assert len(enterprises)==1
        self.enterpriseName=enterprises[0]['name']
        self.pathExpected={}
        self.server = case.Server(private, private)
        self.clients = {}
        self.result = dict(apiConnections=12, sessions=20,
                           binarySHA256=hashlib.sha256(binary.read_bytes()).hexdigest(),
                           completed=False, groups={}, coldResets=[], preparation=[], cleanup=[],
                           timing='complete real HTTP and JSON decode; semantic validation follows the timed response',
                           before=case.cli('size', private))
        self.receipts = []
        self.server.start()
        assert self.server.info['apiConnectionLimit']==12
        self.result['startup']=self.server.info
        self.result['model']=self.result['before'].get('model',self.server.info.get('model'))
        assert self.result['model'], 'record the actual admitted model'

    def close(self):
        self.server.close()
        self.result['after'] = case.cli('size', self.private)
        if self.result.get('lastAuthenticationBaseline'):
            previous=self.result['lastAuthenticationBaseline']
            self.result['securityEventGrowthAfterAuthentication']={k:self.result['after']['counts'][k]-previous[k] for k in ['securityEvents','authenticationEvents','securityChangeEvents']}
        self.result['passed'] = self.result['completed'] and all(g['passed'] for g in self.result['groups'].values()) and all(s['ok'] for s in self.result['preparation']+self.result['cleanup'])
        save(self.output, self.result)
        # Complete receipts are needed to check original identities after restart.
        receipt_file=self.private/('125-receipts-'+self.output.stem+'.json')
        with os.fdopen(os.open(receipt_file,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600),'w') as f:json.dump(self.receipts,f)

    def login(self, role='admin'):
        if role not in self.clients:
            values = []
            for _ in range(20):
                c = case.Client(); c.base = self.server.base
                reply, _ = c.call('POST','/api/v1/session/login',dict(login=role,password='admin1234567'))
                c.csrf = reply['csrf']; values.append(c)
            self.clients[role] = values
            self.result['lastAuthenticationBaseline']=case.cli('size',self.private)['counts']
        return self.clients[role]

    def cold(self, tag):
        container = self.server.info['container']
        allowed = {'chnhcm-storage126-product': ('chnhcm-storage126-product-data','storage126-product'),
                   'chnhcm-storage130-b': ('chnhcm-storage130-b-data','storage130')}
        assert container in allowed
        volume, task = allowed[container]
        inspect = json.loads(subprocess.check_output(['docker','inspect','--format','{{json .Config.Labels}}',container]))
        assert inspect.get('io.chnhcm.task') == task and inspect.get('io.chnhcm.temporary') == 'true'
        if container == 'chnhcm-storage130-b':
            assert inspect.get('io.chnhcm.parent-task') == 'storage130'
        mounts = json.loads(subprocess.check_output(['docker','inspect','--format','{{json .Mounts}}',container]))
        assert any(m['Type']=='volume' and m['Name']==volume and m['Destination']=='/var/lib/postgresql' for m in mounts)
        helper = (ROOT/'.local/bin/125-cache').resolve()
        assert helper.is_file()
        self.server.stop()
        subprocess.run(['docker','stop','--time','30',container],stdout=subprocess.DEVNULL,check=True)
        image = (ROOT/'deploy/local/postgres-image.txt').read_text().strip()
        try:
            completed = subprocess.run(['docker','run','--rm','--network','none','--read-only','--user','postgres',
                '--mount',f'type=volume,src={volume},dst=/data,readonly',
                '--mount',f'type=bind,src={helper},dst=/125-cache,readonly',
                '--env','CHNHCM_T07_CACHE=1','--entrypoint','/125-cache',image],capture_output=True,text=True)
            reset = json.loads(completed.stdout)
            reset.update(tag=tag, helperExit=completed.returncode, apiRebuilt=False,
                         apiBinarySHA256=self.result['binarySHA256'], volume=volume)
            self.result['coldResets'].append(reset)
            save(self.output,self.result)
            assert completed.returncode==0 and reset['passed'] is True
        finally:
            subprocess.run(['docker','start',container],stdout=subprocess.DEVNULL,check=True)
        for _ in range(120):
            if subprocess.run(['docker','exec',container,'pg_isready','-h','127.0.0.1','-U','postgres','-d','storage126'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL).returncode==0:
                break
            time.sleep(.25)
        else:
            raise RuntimeError('owned database did not resume')
        self.server.start()
        for clients in self.clients.values():
            for c in clients: c.base=self.server.base

    def check_list(self, kind, value, page=1):
        codes = self.codes[kind]
        assert value['count']==len(codes) and value['page']==page and value['pageSize']==20
        expected = codes[(page-1)*20:page*20]
        assert [row['code'] for row in value['items']]==expected
        assert len(value['queryKey'])==64
        if kind=='bu': assert len(value['selectionVersion'])==64
        else: assert 'selectionVersion' not in value
        for row, code in zip(value['items'],expected):
            record=self.current[code]
            assert value['businessAt']==today()
            assert row['id']==self.oracle.bycode[code][1]['bu_id' if kind=='bu' else 'department_id']
            assert row['name']==record['name']
            assert row['effectiveDate']==record['date'] and row['status']==record['status']
            if kind=='dep':
                assert row.get('shortName')==record['shortName']
                bu_code=self.byid[record['bu']]
                assert row['managementBu']==dict(id=record['bu'],code=bu_code,name=self.current[bu_code]['name'])
                path_day=today() if record['status']=='active' else record['date'] if record['status']=='not_effective' else (date.fromisoformat(record['date'])-timedelta(days=1)).isoformat()
                mode={'active':'active','inactive':'inactive_before_stop','not_effective':'future_first'}[record['status']]
                assert row['path']['departmentPath']==self.expected_path(code,path_day)
                assert row['path']['pathMode']==mode and row['path']['pathBusinessAt']==path_day
                assert len(row['path']['pathVersion'])==64 and not row['path'].get('pathSegments')

    def expected_path(self,code,day):
        key=(code,day)
        if key in self.pathExpected:return self.pathExpected[key]
        kind=self.oracle.bycode[code][0]
        frame=(self.oracle.proto[kind]['Frames'][-1] if re.fullmatch('R6_[BD]_[0-9]{5}',code) else self.oracle.frame(code))['State']['Timeline']
        r=next((r for r in reversed(frame['Records']) if r['Date']<=day),frame['Records'][0])
        parent=r['Data']['Parent']
        value=(self.expected_path(self.byid[parent],day)+' / ' if parent else '')+r['Data']['Name']
        self.pathExpected[key]=value;return value

    def detail_check(self,code,source):
        key=(code,source)
        if key not in self.detailExpected:
            frame=self.oracle.frame(code,source);t=frame['State']['Timeline'];kind=self.oracle.bycode[code][0]
            records=[]
            for r in t['Records']:
                data=r if kind=='bu' else r['Data']
                records.append(dict(recordId=r['ID'],revision=r['Revision'],effectiveDate=r['Date'],endExclusive=r['End'] or None,operationType=r['Type'],operationReason=r['Reason'],state=r['State'],name=data['Name'],bu=data.get('BU'),parent=data.get('Parent','')))
            selected=next((r for r in reversed(records) if r['effectiveDate']<=today()),records[0])
            paths=[] if kind=='bu' else [' / '.join(n['Name'] for n in p['Nodes'] if n['Kind']=='department') for p in frame['Evidence']['Segments'] if p['RecordID']==selected['recordId'] and p['Revision']==selected['revision']]
            self.detailExpected[key]=dict(source=t['Source'],records=records,selected=selected,paths=paths)
        expected=self.detailExpected[key];kind,owner=self.oracle.bycode[code]
        def check(value):
            assert value['id' if kind=='bu' else 'departmentId']==owner['bu_id' if kind=='bu' else 'department_id']
            assert value['resolvedSourceVersion' if source else 'headSourceVersion']==expected['source']
            assert value['currentDate']==today() and len(value['timeline'])==len(expected['records'])
            for actual,want in zip(value['timeline'],expected['records'],strict=True):
                for field in ['recordId','revision','effectiveDate','endExclusive','operationType','operationReason','state','name']:assert actual.get(field)==want[field]
                if kind=='dep':assert actual['managementBuId']==want['bu'] and (actual.get('parentDepartmentId') or '')==want['parent']
            selected=value['selectedRecord'];want=expected['selected']
            assert (selected['recordId'],selected['revision'],selected['name'])==(want['recordId'],want['revision'],want['name'])
            assert value['selectedRecordIsFuture']==(today()<expected['records'][0]['effectiveDate'])
            if source:assert value['readOnly'] is True
            if kind=='dep':
                assert selected['managementBu']['id']==want['bu']
                if source:assert value['path']['departmentPath'] in expected['paths']
                else:
                    day=want['effectiveDate'] if today()<expected['records'][0]['effectiveDate'] else (date.fromisoformat(want['effectiveDate'])-timedelta(days=1)).isoformat() if want['state']=='inactive' else today()
                    assert value['path']['departmentPath']==self.expected_path(code,day)
        return check

    def group(self,name,samples,target):
        self.result['groups'][name]=stats(samples,target)
        save(self.output,self.result)
        s=self.result['groups'][name]
        print(json.dumps(dict(group=name, requests=s['requests'], failures=s['failures'], p95MS=s['p95MS'])),flush=True)

    def context(self,value,role):
        assert value['enterprise']['id']==self.oracle.owners['bu'][0]['enterprise_id']
        assert value['currentDate']==today()
        assert value['timezone']=='Asia/Shanghai'
        assert value['capabilities']['create']==(role=='admin')
        assert set(value['fieldCatalog'])=={'code','name','shortName','managementBuId','parentDepartmentId','departmentPath','status','effectiveDate','operationType','operationReason'}
        assert 'name' in value['capabilities']['readFields']

    def diagnose(self):
        clients=self.login()
        self.result['diagnosticFirstSourceLoad']=measured(clients[0],'GET','/api/v1/departments',check=lambda v:self.check_list('dep',v))[0]
        for phase,n,rounds in [('single',1,5),('twenty',20,3)]:
            samples=[]
            for _ in range(rounds):
                samples += parallel([lambda c=c: measured(c,'GET','/api/v1/departments',check=lambda v:self.check_list('dep',v))[0] for c in clients[:n]])
            self.group('diagnostic-list-'+phase,samples,300)
        self.references(clients)

    def references(self,clients):
        bu=self.oracle.bycode['R6_B_00000'][1]['bu_id']
        target=self.oracle.bycode['R6_D_00000'][1]['department_id']
        frame=self.oracle.frame('R6_D_00000')['State']['Timeline']; record=frame['Records'][0]
        contexts=[('create',dict(mode='create',effectiveDate=today(),managementBuId=bu),10),
                  ('correct',dict(mode='maintain',action='CORRECT',departmentId=target,expectedSourceVersion=frame['Source'],recordId=record['ID'],expectedRecordRevision=record['Revision']),9)]
        for mode,ctx,count in contexts:
            for search,page in [('',1),('R6_D_00000',1),('',2)]:
                def check(v):
                    expected=count if not search else int(mode=='create')
                    assert v['applicable'] is True and v['count']==expected
                    assert len(v['items'])==(expected if page==1 else 0)
                    assert v['selectedReference']['id']==target
                    for row in v['items']: assert row['eligibility']=='eligible'
                body=dict(context=ctx,page=page,pageSize=20,q=search,selectedId=target)
                values=parallel([lambda c=c:measured(c,'POST','/api/v1/departments/references/parents',body,check=check)[0] for c in clients])
                self.group(f'parents-{mode}-search{bool(search)}-page{page}',values,300)

    def probe(self):
        clients=self.login()
        for kind in ['bu','dep']:
            self.result['preparation'].append(measured(clients[0],'GET','/api/v1/'+('business-units' if kind=='bu' else 'departments'),check=lambda v,k=kind:self.check_list(k,v))[0])
        for action in ['save','pair','resolve']:
            prepared=[self.write_job(c,'dep' if w%2 else 'bu',action,0,w) for w,c in enumerate(clients)]
            values=parallel([job for job,_ in prepared])
            self.group('probe-'+action,[v[0] for v in values],500 if action!='resolve' else 0)
            for _,cleanup in prepared:cleanup()
        for role in ['admin','reader']:
            c=self.login(role)[0]
            self.result['preparation'].append(measured(c,'GET','/api/v1/departments/context',check=lambda v,r=role:self.context(v,r))[0])

    def contexts(self):
        for role in ['admin','reader']:
            clients=self.login(role)
            for cache,rounds in [('warm',10),('cold',5)]:
                samples=[]
                if cache=='warm':
                    self.result['preparation'].append(measured(clients[0],'GET','/api/v1/departments/context',check=lambda v:self.context(v,role))[0])
                for round in range(rounds):
                    if cache=='cold': self.cold(f'context-{role}-{round}')
                    if cache=='cold':
                        first=measured(clients[0],'GET','/api/v1/departments/context',check=lambda v:self.context(v,role))[0]
                        self.result.setdefault('contextFirstRequests',[]).append(dict(role=role,round=round,**first))
                        values=[first]+parallel([lambda c=c:measured(c,'GET','/api/v1/departments/context',check=lambda v:self.context(v,role))[0] for c in clients[1:]])
                    else:
                        values=parallel([lambda c=c:measured(c,'GET','/api/v1/departments/context',check=lambda v:self.context(v,role))[0] for c in clients])
                    samples+=values
                self.group(f'context-{role}-{cache}',samples,300)

    def read_job(self,c,kind,action,round,w):
        # Reads avoid objects mutated in the timed save lane.
        code=f'R6_{"B" if kind=="bu" else "D"}_{(round*20+w)%400:05d}'
        if action=='list':
            page=1+(round*20+w)%(len(self.codes[kind])//20)
            path='/api/v1/'+('business-units' if kind=='bu' else 'departments')+f'?page={page}&pageSize=20'
            return lambda:measured(c,'GET',path,check=lambda v:self.check_list(kind,v,page))
        source=25 if action=='source' else None
        path=self.oracle.path(code,source);check=self.detail_check(code,source)
        return lambda:measured(c,'GET',path,check=check)

    def write_job(self,c,kind,action,round,w):
        code=f'R6_SOLO_{w:02d}' if kind=='bu' and action=='pair' else f'R6_{"B" if kind=="bu" else "D"}_{600+((round*20+w)%1300):05d}'
        path=self.oracle.path(code); before,ms=c.call('GET',path)
        self.result['preparation'].append(dict(ok=True,ms=ms))
        key=str(uuid.uuid4()); receipt={}; baseline=copy.deepcopy(before['timeline'])
        route='/api/v1/'+('organization' if kind=='bu' else 'departments')+'/submissions/'+key+'/resolve'
        if action=='pair':
            body=dict(expectedSourceVersion=before['headSourceVersion'],anchorRecordId=baseline[-1]['recordId'],deactivationDate='2035-01-01',deactivationReason='ORG_RESTRUCTURING',reactivationDate='2035-02-01',reactivationReason='BUSINESS_RESUMPTION')
            endpoint=path+'/lifecycle-pairs'
        else:
            name='125容量核验'+uuid.uuid4().hex
            body=dict(expectedSourceVersion=before['headSourceVersion'],expectedRecordRevision=baseline[0]['revision'],changes=dict(name=name))
            endpoint=path+'/records/'+baseline[0]['recordId']+'/correct'
        def accepted(v):
            assert v['outcome']=='accepted' and v['submissionId']==key and v['sourceVersion']!=body['expectedSourceVersion']
            assert v['departmentId' if kind=='dep' else 'buId']==before['departmentId' if kind=='dep' else 'id']
            receipt.update(v)
        def cleanup():
            if not receipt:
                recovered,value=measured(c,'POST',route)
                self.result['cleanup'].append(recovered)
                if value and value.get('outcome')=='accepted': accepted(value)
                else: return
            self.receipts.append(dict(route=route,path=path,receipt=dict(receipt)))
            exact,_=c.call('GET',path+'?sourceVersion='+receipt['sourceVersion'])
            assert exact['resolvedSourceVersion']==receipt['sourceVersion'] and exact['readOnly'] is True
            assert len(exact['timeline'])==len(baseline)+(2 if action=='pair' else 0)
            current,_=c.call('GET',path)
            assert current['headSourceVersion']==receipt['sourceVersion']
            final_head=current['headSourceVersion']
            if action=='pair':
                a,z=current['timeline'][-2:]
                assert a['effectiveDate']=='2035-01-01' and z['effectiveDate']=='2035-02-01' and a['state']=='inactive' and z['state']=='active'
                assert len(current['timeline'])==len(baseline)+2
                data=dict(expectedSourceVersion=current['headSourceVersion'],deactivationRecordId=a['recordId'],deactivationRevision=a['revision'],reactivationRecordId=z['recordId'],reactivationRevision=z['revision'])
                deletion_key=str(uuid.uuid4())
                deleted,_=c.call('POST',path+'/lifecycle-pairs/delete',data,deletion_key)
                assert deleted['outcome']=='accepted' and deleted['submissionId']==deletion_key
                # Cleanup is a real business write too. Keep its original
                # receipt for the same post-restart source/recovery checks.
                self.receipts.append(dict(route=route.replace(key,deletion_key),path=path,receipt=dict(deleted)))
                restored,_=c.call('GET',path)
                final_head=restored['headSourceVersion']
                assert [(r['recordId'],r['effectiveDate'],r.get('endExclusive'),r['name'],r['state']) for r in restored['timeline']]==[(r['recordId'],r['effectiveDate'],r.get('endExclusive'),r['name'],r['state']) for r in baseline]
            else:
                row=next(r for r in current['timeline'] if r['recordId']==baseline[0]['recordId'])
                assert row['name']==body['changes']['name'] and row['revision']==receipt['targetRevision']
                assert len(current['timeline'])==len(baseline)
                assert [(r['recordId'],r['effectiveDate'],r.get('endExclusive')) for r in current['timeline']]==[(r['recordId'],r['effectiveDate'],r.get('endExclusive')) for r in baseline]
            resolved,value=measured(c,'POST',route,check=lambda v:self.exact_receipt(v,receipt))
            if resolved['ok']:
                resolved['ok']=value['currentChanged']==(final_head!=receipt['sourceVersion'])
                if not resolved['ok']:resolved['semanticError']=True
            self.result['cleanup'].append(resolved)
        if action=='resolve':
            prep,value=measured(c,'POST',endpoint,body,key,accepted);self.result['preparation'].append(prep)
            if not prep['ok']:
                # A failed HTTP response does not prove the write was rejected.
                # Resolve that same key before discarding this planned sample;
                # never resubmit the mutation or turn recovery into a pass.
                try:cleanup()
                except Exception as error:
                    self.result['cleanup'].append(dict(_failure_sample(error),
                        scope='failed resolve preparation recovery'))
                raise PreparationFailure(prep)
            job=lambda:measured(c,'POST',route,check=lambda v:self.exact_receipt(v,receipt))
        else:
            job=lambda:measured(c,'POST',endpoint,body,key,accepted)
        return job,cleanup

    @staticmethod
    def exact_receipt(value,receipt):
        for key,expected in receipt.items():
            if key!='currentChanged':assert value.get(key)==expected

    def matrix(self,phase,*,actions=None,modes=None):
        clients=self.login(); cold=phase=='cold'
        selected_actions=ACTIONS if actions is None else actions
        selected_modes=(['mixed-actions'] if phase=='mixed' else ['BU','Department','mixed']+([] if cold else ['mixed-actions'])) if modes is None else modes
        assert selected_actions and len(set(selected_actions))==len(selected_actions) and set(selected_actions)<=set(ACTIONS)
        assert selected_modes and len(set(selected_modes))==len(selected_modes) and set(selected_modes)<={'BU','Department','mixed','mixed-actions'}
        expected=sum(1200 if mode=='mixed-actions' else len(selected_actions)*(100 if cold else 200) for mode in selected_modes)
        self.result.update(expectedTimedRequests=expected,
                           actualTimedRequests=0,unattemptedRequests=[],allScheduledGroupsAttempted=False)
        if actions is not None or modes is not None:
            self.result['partition']=dict(actions=selected_actions,modes=selected_modes,
                originalPhaseRequests={'warm':4800,'cold':1800,'mixed':1200}[phase],
                fullPhaseAcceptance=False,scope='explicit affected-write partition; other original groups not rerun or relabelled')
        if not cold:
            self.result['preparation'].append(measured(clients[0],'GET','/api/v1/departments',check=lambda v:self.check_list('dep',v))[0])
        for mode in selected_modes:
            for category in (['all'] if mode=='mixed-actions' else selected_actions):
                samples=[]; scheduled=[]
                rounds=60 if mode=='mixed-actions' else 5 if cold else 10
                name=f'{phase}/{mode}/{category}'
                # Mixed groups with different per-object targets are judged by
                # their complete object/action subsets, never a blended P95.
                target=None if category=='all' or mode=='mixed' and category=='list' else operation_target_ms('dep' if mode=='Department' else 'bu',category,sessions=20)
                def retain(scheduled_complete=False):
                    group=stats(samples,target)
                    group.update(expectedRequests=rounds*20,completed=len(samples)==rounds*20,
                                 allScheduledRoundsAttempted=scheduled_complete)
                    group['passed']=group['passed'] and group['completed']
                    self.result['groups'][name]=group
                    save(self.output,self.result)
                retain()
                for round in range(rounds):
                    jobs=[]; cleanups=[]; attributes=[]
                    for w,c in enumerate(clients):
                        kind='dep' if mode=='Department' or mode.startswith('mixed') and w%2 else 'bu'
                        action=ACTIONS[(w+round)%6] if category=='all' else category
                        attribute=dict(kind=kind,action=action)
                        scheduled.append(attribute)
                        cleanup=lambda:None
                        try:
                            if action in ['list','detail','source']: job=self.read_job(c,kind,action,round,w)
                            else: job,cleanup=self.write_job(c,kind,action,round,w)
                        except Exception as error:
                            sample=error.sample if isinstance(error,PreparationFailure) else _failure_sample(error)
                            failure=dict(group=name,round=round,worker=w,**attribute,
                                         errorType=sample['errorType'],
                                         status=sample.get('status'),code=sample.get('code'))
                            sample.update(failure)
                            if not isinstance(error,PreparationFailure):self.result['preparation'].append(sample)
                            self.result['unattemptedRequests'].append(dict(reason='preparation failed before timed request',**failure))
                            continue
                        jobs.append(job);cleanups.append(cleanup);attributes.append(attribute)
                    # Preserve preparation failures even if the following cold
                    # reset fails. No placeholder is a timed HTTP request.
                    save(self.output,self.result)
                    if cold:self.cold(f'{mode}-{category}-{round}')
                    values=parallel(jobs)
                    for (sample,_),attributes_ in zip(values,attributes): sample.update(attributes_);samples.append(sample)
                    self.result['actualTimedRequests']+=len(values)
                    retain()
                    for cleanup in cleanups:
                        try: cleanup()
                        except Exception as error:self.result['cleanup'].append(_failure_sample(error))
                    save(self.output,self.result)
                retain(True)
                if category=='all' or mode=='mixed':
                    for kind in ['bu','dep']:
                        for action in ACTIONS if category=='all' else [category]:
                            subset=[s for s in samples if s['kind']==kind and s['action']==action]
                            expected=sum(s['kind']==kind and s['action']==action for s in scheduled)
                            tag=f'{phase}/{mode}/{kind}/{action}'
                            group=stats(subset,operation_target_ms(kind,action,sessions=20))
                            group.update(expectedRequests=expected,completed=len(subset)==expected,
                                         allScheduledRoundsAttempted=True)
                            group['passed']=group['passed'] and group['completed']
                            self.result['groups'][tag]=group
                    save(self.output,self.result)
        self.result['allScheduledGroupsAttempted']=True
        save(self.output,self.result)

    def specials(self):
        from vanke_csv import protected_text
        clients=self.login()
        codes=['R6_DEPTH_099','R6_FAN_ROOT','R6_LONG_HISTORY']
        assert all(code in self.oracle.extra for code in codes)
        for code in codes:
            for source in [None,1]:
                values=[]
                for _ in range(3):
                    path=self.oracle.path(code,source);check=self.detail_check(code,source)
                    values+=parallel([lambda c=c:measured(c,'GET',path,check=check)[0] for c in clients])
                self.group(f'special-{code}-source{source}',values,300 if source is None else 500)
        client=clients[0]
        selected=[];draft=None
        query=dict(q='R6_D_',status='all')
        for page in range(1,8):
            value,elapsed=client.call('GET',f'/api/v1/departments?q=R6_D_&status=all&page={page}&pageSize=20')
            assert value['count']==20000 and len(value['items'])==20
            assert 'selectionVersion' not in value
            selected += [v['id'] for v in value['items']][:min(20,137-len(selected))]
            self.result['preparation'].append(dict(ok=True,ms=elapsed))
        assert len(selected)==137 and len(set(selected))==137
        established,elapsed=client.call('POST','/api/v1/departments/selection/start',dict(query=query,selection=dict(mode='explicit',ids=selected),page=1,pageSize=20,sort='code'))
        draft=established['summary']
        assert established['summary']['selectedCount']==137
        self.result['preparation'].append(dict(kind='department-selection-start',ok=True,ms=elapsed))
        body=dict(query=query,draftId=draft['draftId'],expectedRevision=draft['revision'],sort='code')
        req=urllib.request.Request(client.base+'/api/v1/departments/export',method='POST',data=json.dumps(body).encode(),headers={'Content-Type':'application/json','Origin':client.base,'X-CSRF-Token':client.csrf})
        started=time.perf_counter()
        with client.http.open(req,timeout=20) as response:
            assert response.status==200 and 'text/csv' in response.headers.get('Content-Type','')
            data=response.read()
        elapsed=(time.perf_counter()-started)*1000
        rows=list(csv.reader(io.StringIO(data.decode('utf-8-sig'))))
        assert rows[0]==['编码','名称','简称','BU编码','BU名称','部门路径','状态','生效日期'] and len(rows)==138
        expected_codes=[f'R6_D_{n:05d}' for n in range(137)]
        byid={v[1]['bu_id']:code for code,v in self.oracle.bycode.items() if v[0]=='bu'}
        for row,code in zip(rows[1:],expected_codes,strict=True):
            record=self.current[code];bu_code=byid[record['bu']];bu=self.current[bu_code]
            frame=self.oracle.frame(code)['State']['Timeline'];r=next(r for r in frame['Records'] if r['Date']==record['date'])
            assert r['Data']['Parent']==''
            expected=[code,record['name'],r['Data'].get('ShortName') or '',bu_code,bu['name'],record['name'],'启用',record['date']]
            assert row==[protected_text(value) for value in expected]
        self.result['csv137']=dict(passed=True,rows=137,pages=7,columns=8,allFieldsCompared=True,duplicates=0,missing=0,ms=elapsed,bytes=len(data),sha256=hashlib.sha256(data).hexdigest(),singleSample=True)
        wrong=dict(body,draftId='00000000-0000-0000-0000-000000000000')
        req=urllib.request.Request(client.base+'/api/v1/departments/export',method='POST',data=json.dumps(wrong).encode(),headers={'Content-Type':'application/json','Origin':client.base,'X-CSRF-Token':client.csrf})
        try:
            client.http.open(req,timeout=20)
        except urllib.error.HTTPError as response:
            error=json.load(response)
            assert response.code==409 and error['code']=='M02_SELECTION_CHANGED' and 'Content-Disposition' not in response.headers
        else:raise AssertionError('stale selection produced a file')
        self.result['csv137']['staleSelectionZeroFile']=True


if __name__=='__main__':
    parser=argparse.ArgumentParser()
    parser.add_argument('private',type=Path);parser.add_argument('output',type=Path)
    parser.add_argument('--binary',type=Path,required=True)
    parser.add_argument('--phase',choices=['diagnose','probe','warm','cold','mixed','context','specials'],required=True)
    args=parser.parse_args()
    assert args.private.is_absolute() and ROOT not in args.private.parents and args.private.stat().st_mode&0o777==0o700
    assert not args.output.exists(),'first results must not be overwritten'
    lane=Lane(args.private,args.output,args.binary.resolve())
    try:
        # Prove that successful HTTP with a wrong count is rejected.
        try: lane.check_list('dep',dict(count=0,page=1,pageSize=20,items=[]))
        except (AssertionError,KeyError): lane.result['wrongHTTP200Rejected']=True
        else: raise AssertionError('capacity oracle accepted a false empty collection')
        if args.phase=='diagnose':lane.diagnose()
        elif args.phase=='probe':lane.probe()
        elif args.phase=='context':lane.contexts()
        elif args.phase=='specials':lane.specials()
        else:lane.matrix(args.phase)
        lane.result['completed']=(lane.result['actualTimedRequests']==lane.result['expectedTimedRequests']
                                  if args.phase in ['warm','cold','mixed'] else True)
    finally:lane.close()
    if not lane.result['passed']:raise SystemExit(1)
