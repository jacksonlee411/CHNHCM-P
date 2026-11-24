#!/usr/bin/env python3
"""Compare every persisted source and path with an independent workbook oracle."""
import argparse
import bisect
import collections
import hashlib
import json
from pathlib import Path
import sqlite3


class Oracle:
    def __init__(self, private, creation_only=False):
        self.reference_path=private/'verification.sqlite'
        db=sqlite3.connect(private/'import.sqlite')
        self.objects={code:dict(id=oid,kind=kind) for code,oid,kind in db.execute('SELECT code,object_id,kind FROM objects')}
        self.codes={v['id']:code for code,v in self.objects.items()}
        self.frames={};self.heads={};self.events=collections.defaultdict(list)
        self.rows={(p,n):(json.loads(raw),json.loads(result) if result else None) for p,n,raw,result in db.execute('SELECT phase,row_no,payload,result FROM rows')}
        self.root=next(code for code,v in self.objects.items() if v['kind']=='business-units')
        for (phase,no),(row,result) in self.rows.items():
            if phase!='create':continue
            if creation_only and result is None:continue
            assert result and result['outcome']=='accepted','missing organization'
            record=(row['effective_date'],row['name'],row.get('parent_org_code',''),result['response']['targetRecordId'])
            self.add_frame(row['org_code'],result,[record],0,'CREATE')
        self.no_changes=0
        if creation_only:assert all(result is None for (phase,_),(_,result) in self.rows.items() if phase!='create')
        order=[] if creation_only else json.loads((private/'history-order.json').read_text())
        for seq,(phase,no) in enumerate(order,1):
            row,result=self.rows[phase,no];assert result,'missing historical command'
            code=row['org_code'];before=self.frames[self.heads[code]];records=list(before['records'])
            day=row.get('target_effective_date') or row['effective_date']
            ds=[r[0] for r in records];at=bisect.bisect_right(ds,day)-1;prev=records[at]
            if result['outcome']=='no_change':
                assert phase=='version' and not row.get('core.name'),'unexpected no-change source'
                assert result['response']['observedSourceVersion']==before['source']
                self.no_changes+=1;continue
            assert result['outcome']=='accepted'
            response=result['response']
            if phase=='version':
                assert day not in ds
                value=(day,row['core.name'],prev[2],response['targetRecordId']);records.insert(at+1,value);action='ADD'
            elif day in ds:
                assert row['operation']=='correct_parent'
                value=(day,prev[1],row['new_parent_org_code'],prev[3]);records[at]=value;action='CORRECT'
            else:
                value=(day,prev[1],row['new_parent_org_code'],response['targetRecordId']);records.insert(at+1,value);action='ADD'
            self.add_frame(code,result,records,seq,action)
        if not creation_only:
            assert len(self.objects)==87344 and self.no_changes==18
            assert len(self.frames)==132284
        self.event_times={code:[seq for seq,_ in rows] for code,rows in self.events.items()}
        db.close()

    def add_frame(self, code, result, records, sequence, action):
        response=result['response'];source=response['sourceVersion']
        assert source not in self.frames
        self.frames[source]=dict(code=code,source=source,previous=self.heads.get(code,''),number=len(self.events[code])+1,records=records,sequence=sequence,submission=result['requestKey'],action=action,target=response['targetRecordId'],revision=response['targetRevision'],response=response)
        self.heads[code]=source;self.events[code].append((sequence,source))

    def source_at(self, code, sequence):
        return self.events[code][bisect.bisect_right(self.event_times[code],sequence)-1][1]

    def record_at(self, frame, day):
        rows=frame['records'];return rows[bisect.bisect_right([r[0] for r in rows],day)-1]

    def path(self, code, source, day, current=False):
        sequence=self.frames[source]['sequence'];nodes=[];seen=set()
        while code!=self.root:
            assert code not in seen,'oracle cycle';seen.add(code)
            source=self.heads[code] if current else self.source_at(code,sequence);frame=self.frames[source];r=self.record_at(frame,day)
            nodes.append((self.objects[code]['id'],source,r[3],r[1]));code=r[2]
        return list(reversed(nodes))

    def check_fact(self, kind, fact):
        op=fact['Operation'];source=op['Source']['ID'];expected=self.frames[source];t=fact['State']['Timeline'];code=expected['code'];obj=self.objects[code]
        assert t['Code']==code and fact['Object']==obj['id'],'object identity mismatch'
        assert op['Source']['Submission']==expected['submission'] and op['Source']['Number']==expected['number'] and op['Source']['Previous']==expected['previous'] and op['Source']['Action']==expected['action'],'source identity mismatch'
        terminal=op['Terminal'];assert terminal['ID']==expected['submission'] and terminal['Outcome']=='accepted','terminal identity mismatch'
        for key in ['submissionId','outcome','buId','departmentId','sourceVersion','targetRecordId','targetRevision','selectedRecordId','effectiveDate','action']:
            assert terminal['Result'].get(key)==expected['response'].get(key),'original terminal result mismatch'
        assert t['Source']==source and len(t['Records'])==len(expected['records']),'timeline membership mismatch'
        revisions=[]
        for i,(actual,want) in enumerate(zip(t['Records'],expected['records'])):
            day,name,parent,rid=want;data=actual['Data']
            assert (actual['Date'],data['Name'],actual['ID'])==(day,name,rid),'date, name or record identity mismatch'
            assert actual['End']==(expected['records'][i+1][0] if i+1<len(expected['records']) else ''),'period mismatch'
            assert actual['State']=='active' and actual['Type']==('CREATE' if i==0 else 'CHANGE'),'lifecycle mismatch'
            if kind!='BU':
                assert data['BU']==self.objects[self.root]['id']
                assert data['Parent']==(self.objects[parent]['id'] if parent!=self.root else ''),'historical parent mismatch'
            if rid==expected['target']:assert actual['Revision']==expected['revision'],'target revision mismatch'
            revisions.append((source,rid,actual['Revision']))
        return expected,revisions

    def check_detail(self, code, value, source=None):
        source=source or self.heads[code];expected=self.frames[source]
        resolved=value.get('resolvedSourceVersion') if value.get('readOnly') else value['headSourceVersion']
        assert resolved==source and len(value['timeline'])==len(expected['records']),'HTTP source mismatch'
        for actual,want in zip(value['timeline'],expected['records']):
            assert actual['recordId']==want[3] and actual['effectiveDate']==want[0],'HTTP membership mismatch'
            if 'name' in actual:assert actual['name']==want[1],'HTTP name mismatch'
        selected=value['selectedRecord'];want=next(r for r in expected['records'] if r[3]==selected['recordId'])
        assert selected['name']==want[1],'HTTP selected name mismatch'
        if code!=self.root:
            assert selected['managementBu']['id']==self.objects[self.root]['id']
            got=selected.get('parentDepartment');assert (got['id'] if got else None)==(self.objects[want[2]]['id'] if want[2]!=self.root else None)
            segments=value['path']['pathSegments'];assert value['path']['pathMode']=='active' and segments
            index=expected['records'].index(want);end=expected['records'][index+1][0] if index+1<len(expected['records']) else ''
            assert segments[0]['start']==want[0] and (segments[-1]['endExclusive'] or '')==end,'HTTP path period mismatch'
            for i,segment in enumerate(segments):
                path=self.path(code,source,segment['start'],current=not value['readOnly'])
                assert 'nodes' not in segment, 'public link-only path nodes remain'
                assert segment['path']==' / '.join(node[3] for node in path),'HTTP path composition mismatch'
                # verify() separately compares every persisted exact source,
                # member and revision with the workbook oracle; the public API
                # exposes the authorized text and period only.
                self.check_path_boundary(path,segment['start'],segment['endExclusive'] or '')
                if i:assert segments[i-1]['endExclusive']==segment['start']
        return True

    def check_path_boundary(self, nodes, start, end):
        for _,source,_,_ in nodes:
            assert not any(start<r[0] and (not end or r[0]<end) for r in self.frames[source]['records']),'path segment skipped an ancestor record boundary'


def verify(private, out, creation_only=False):
    oracle=Oracle(private,creation_only);ref_path=private/'verification.sqlite';refs=sqlite3.connect(ref_path);ref_path.chmod(0o600)
    refs.execute('DROP TABLE IF EXISTS revisions');refs.execute('CREATE TABLE revisions(source TEXT,record TEXT,revision TEXT,PRIMARY KEY(source,record))')
    seen=set();counts=collections.Counter();digest=hashlib.sha256();batch=[]
    with (private/'product-history.jsonl').open() as f:
        for line in f:
            digest.update(line.encode());item=json.loads(line);fact=item['Fact'];expected,rows=oracle.check_fact(item['Kind'],fact)
            assert expected['source'] not in seen;seen.add(expected['source']);counts[item['Kind']]+=1;batch.extend(rows)
            if len(batch)>=1000:refs.executemany('INSERT INTO revisions VALUES(?,?,?)',batch);batch=[]
    if batch:refs.executemany('INSERT INTO revisions VALUES(?,?,?)',batch)
    refs.commit();assert seen==set(oracle.frames),'missing or extra persisted source'
    path_segments,path_nodes=0,0
    with (private/'product-history.jsonl').open() as f:
        for line in f:
            item=json.loads(line)
            if item['Kind']=='BU':continue
            fact=item['Fact'];source=fact['Operation']['Source']['ID'];code=oracle.frames[source]['code'];groups=collections.defaultdict(list)
            for segment in fact['Evidence']['Segments']:
                groups[segment['RecordID']].append(segment)
                expected=oracle.path(code,source,segment['Start']);nodes=segment['Nodes']
                assert segment['Mode']=='active'
                assert nodes[0]['Kind']=='enterprise' and nodes[0]['Name']=='组织历史验证企业'
                assert len(nodes)==len(expected)+1,'frozen path depth mismatch'
                for node,want in zip(nodes[1:],expected):
                    assert node['Kind']=='department'
                    assert (node['DepartmentID'],node['Source'],node['RecordID'],node['Name'])==want,'frozen path reference or fact mismatch'
                    revision=refs.execute('SELECT revision FROM revisions WHERE source=? AND record=?',(want[1],want[2])).fetchone()
                    assert revision and revision[0]==node['Revision'],'frozen path revision mismatch'
                    path_nodes+=1
                oracle.check_path_boundary(expected,segment['Start'],segment['End'])
                path_segments+=1
            for record in fact['State']['Timeline']['Records']:
                segments=sorted(groups[record['ID']],key=lambda x:x['Start']);assert segments and segments[0]['Start']==record['Date'] and segments[-1]['End']==record['End'],'path coverage mismatch'
                for i,s in enumerate(segments):
                    assert s['Revision']==record['Revision']
                    if i:assert segments[i-1]['End']==s['Start'],'path coverage gap'
    result=dict(scope='imported creation prefix; historical changes not imported' if creation_only else 'complete three-workbook product history before measurement writes',completeHistoricalDataset=not creation_only,allSourcesVerified=True,allSourceValuesIndependent=True,counts=dict(counts),sourceCount=len(seen),organizations=len(oracle.objects),headRecordCount=sum(len(oracle.frames[s]['records']) for s in oracle.heads.values()),noChangeSourceRows=oracle.no_changes,frozenPathSegments=path_segments,frozenDepartmentNodes=path_nodes,allFrozenPathsAndExactReferencesVerified=True,productDumpSHA256=digest.hexdigest())
    out.write_text(json.dumps(result,indent=2)+'\n');print(json.dumps(result));refs.close()


if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('private',type=Path);p.add_argument('out',type=Path);p.add_argument('--creation-only',action='store_true');a=p.parse_args();verify(a.private,a.out,a.creation_only)
