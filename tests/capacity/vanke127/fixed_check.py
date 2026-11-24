#!/usr/bin/env python3
"""Repeatable normal-API checks; retain the fixed case's business facts."""
import argparse
import json
from pathlib import Path
from urllib.parse import urlsplit

from load import Client
from measure import measured
from runtime import cli
from verify import Oracle


def run(private, out, origin='http://127.0.0.1:4388'):
    address = urlsplit(origin)
    assert address.scheme == 'http' and address.hostname == '127.0.0.1' and address.port and not address.path and not address.query and not address.fragment and not address.username
    oracle = Oracle(private)
    before = cli('size', private)
    admission = cli('source-audit', private)
    assert before['counts']['BU'] == 1
    assert before['counts']['department'] == 87343
    assert before['counts']['buSources'] == 1
    assert before['counts']['departmentSources'] == 132283
    codes = sorted(c for c in oracle.objects if c != oracle.root)
    sampled = list(dict.fromkeys([oracle.root, max(codes, key=lambda c: len(oracle.events[c]))] +
                                [codes[i * len(codes) // 10] for i in range(10)]))
    result = dict(origin=origin, scope='full-history case, normal same-origin API, two roles',
                  businessWrites=0, checks=[], readGrowth=[], passed=False)

    def save():
        out.write_text(json.dumps(result, indent=2) + '\n')

    def check(client, label, path, assertion, method='GET', body=None):
        sample, value = measured(client, method, path, body=body, check=assertion)
        result['checks'].append(dict(label=label, **sample))
        save()
        return value

    def route(code):
        obj = oracle.objects[code]
        return '/api/v1/' + obj['kind'] + '/' + obj['id']

    for login in ['admin', 'reader']:
        client = Client(result['origin'])
        status, value, _ = client.call('POST', '/api/v1/session/login',
                                       dict(login=login, password='admin1234567'))
        assert status == 200, 'fixed runtime login failed'
        client.csrf = value['csrf']
        read_before = cli('size', private)
        lists = {}
        for kind, expected in [('business-units', [oracle.root]), ('departments', codes)]:
            def context(value):
                assert value['enterprise']['code'] == 'VANKE127'
                assert value['capabilities']['create'] == (login == 'admin')
            resource = 'organization' if kind == 'business-units' else kind
            check(client, login + ':' + kind + ':context', '/api/v1/' + resource + '/context', context)

            def listing(value):
                assert value['count'] == len(expected)
                assert [item['code'] for item in value['items']] == expected[:20]
                for item, code in zip(value['items'], expected):
                    record = oracle.record_at(oracle.frames[oracle.heads[code]], value['businessAt'])
                    assert item['name'] == record[1] and item['id'] == oracle.objects[code]['id']
            lists[kind] = check(client, login + ':' + kind + ':list', '/api/v1/' + kind, listing)
        for index, code in enumerate(sampled):
            for old in [False, True]:
                source = oracle.events[code][0][1] if old else None
                def detail(value):
                    oracle.check_detail(code, value, source)
                    if login == 'reader':
                        assert not any(value['capabilities'].get(k) for k in
                                       ['create', 'add', 'correct', 'delete', 'deactivate', 'reactivate', 'redate'])
                check(client, login + ':sample-' + str(index) + (':original' if old else ':current'),
                      route(code) + ('?sourceVersion=' + source if source else ''), detail)
        if login == 'admin':
            def original_result(expected):
                def compare(value):
                    for key in ['submissionId', 'outcome', 'sourceVersion', 'observedSourceVersion',
                                'targetRecordId', 'targetRevision', 'selectedRecordId']:
                        assert value.get(key) == expected.get(key)
                return compare
            for index, code in enumerate(sampled):
                frame = oracle.frames[oracle.events[code][0][1]]
                resource = 'organization' if code == oracle.root else 'departments'
                check(client, 'original-submission:' + str(index),
                      '/api/v1/' + resource + '/submissions/' + frame['submission'] + '/resolve',
                      original_result(frame['response']), method='POST')
            for (phase, number), (raw, stored) in oracle.rows.items():
                if stored['outcome'] == 'no_change':
                    check(client, 'no-change-submission:' + phase + ':' + str(number),
                          '/api/v1/departments/submissions/' + stored['requestKey'] + '/resolve',
                          original_result(stored['response']), method='POST')
            for field, numbers in admission['invalidRowNumbers'].items():
                phase = field.split('.')[0]
                for no in numbers:
                    raw, stored = oracle.rows[phase, no]
                    code = raw['org_code']; source = stored['response']['sourceVersion']
                    target = stored['response']['targetRecordId']
                    check(client, 'historical-name:' + phase + ':' + str(no),
                          route(code) + '?sourceVersion=' + source + '&selectedRecordId=' + target,
                          lambda value: oracle.check_detail(code, value, source))
        # Repeat auxiliary reads under one fixed session and current policy.
        for repeat in range(2):
            for kind, expected in [('business-units', 1), ('departments', 87343)]:
                if kind=='departments':
                    def full_capture(v):assert v['summary']['selectedCount']==expected
                    start=check(client,login+':selection-start:'+str(repeat),'/api/v1/departments/selection/start',full_capture,'POST',dict(query={},selection=dict(mode='allMatching',excludedIds=[]),page=1,pageSize=20))
                    draft=start['summary']
                    choice=dict(query={},draftId=draft['draftId'],expectedRevision=draft['revision'],page=1,pageSize=20)
                else:
                    choice=dict(query={},selection=dict(mode='allMatching',excludedIds=[]),expectedSelectionVersion=lists[kind]['selectionVersion'])
                def selection(value):
                    assert (value['summary'] if kind=='departments' else value)['selectedCount']==expected
                check(client, login + ':selection:' + kind + ':' + str(repeat), '/api/v1/' + kind + '/selection/resolve', selection, 'POST', choice)
            code = sampled[-1]
            def changes(value):
                assert value['total'] >= 1
            check(client, login + ':changes:' + str(repeat), route(code) + '/changes', changes)
            if login == 'admin':
                def references(value):
                    assert len(value['items']) >= 1
                body = dict(context=dict(mode='create', effectiveDate=lists['departments']['businessAt'], managementBuId=oracle.objects[oracle.root]['id']), page=1, pageSize=20)
                for kind in ['business-units', 'parents']:
                    if kind=='parents':body.pop('page',None)
                    check(client, 'references:' + kind + ':' + str(repeat), '/api/v1/departments/references/' + kind, references, 'POST', body)
                detail = check(client, 'preview-context:' + str(repeat), route(code), lambda value: oracle.check_detail(code, value))
                row = detail['selectedRecord']
                preview = dict(action='CORRECT', expectedSourceVersion=detail['headSourceVersion'], recordId=row['recordId'], expectedRecordRevision=row['revision'], changes=dict(shortName='128预览检查'))
                def valid(value):
                    assert value['valid'] is True
                check(client, 'preview:' + str(repeat), route(code) + '/preview', valid, 'POST', preview)
        read_after = cli('size', private)
        growth = dict(role=login, securityEvents=read_after['counts']['securityEvents']-read_before['counts']['securityEvents'], sessionUpdates=read_after['sessionUpdates']-read_before['sessionUpdates'], databaseBytes=read_after['databaseBytes']-read_before['databaseBytes'])
        result['readGrowth'].append(growth); save()
        assert growth['securityEvents'] == 0, 'read-only auxiliary operations persisted authorization events'
    after = cli('size', private)
    keys = ['BU', 'department', 'buSources', 'departmentSources', 'sourceRows',
            'standaloneBuTerminals', 'standaloneDepartmentTerminals']
    result['businessCountsBefore'] = {k: before['counts'][k] for k in keys}
    result['businessCountsAfter'] = {k: after['counts'][k] for k in keys}
    result['businessCountsUnchanged'] = result['businessCountsBefore'] == result['businessCountsAfter']
    result['passed'] = result['businessCountsUnchanged'] and all(c['ok'] for c in result['checks'])
    save()
    print(json.dumps(dict(passed=result['passed'], checks=len(result['checks']),
                          failed=sum(not c['ok'] for c in result['checks']), businessWrites=0)))
    assert result['passed'], 'fixed case checks failed; aggregate evidence preserved'


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('private', type=Path); p.add_argument('out', type=Path)
    p.add_argument('--origin', default='http://127.0.0.1:4388')
    a = p.parse_args(); run(a.private, a.out, a.origin)
