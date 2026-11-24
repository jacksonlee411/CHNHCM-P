#!/usr/bin/env python3
"""Measure actual allocation growth and history retention in the task candidate."""
import argparse
import json
import uuid
from pathlib import Path

from measure import Client, Oracle, Server, cli, save, check_resolve


def run(directory, capacity_path, output):
    capacity = json.loads(capacity_path.read_text())
    assert capacity.get('completed'), 'finish the concurrent measurement first'
    oracle = Oracle(directory)
    result = {'completed': False, 'scope': 'same candidate, ordinary maintenance; all histories retained',
              'originalResults': [], 'originalSources': [], 'created': []}
    result['beforeMaintenance'] = cli('size', directory)
    result['maintenance'] = cli('maintain', directory)
    server = Server(directory, output.parent)
    client = Client()
    try:
        client.base = server.start()
        session, _ = client.call('POST', '/api/v1/session/login',
                                 {'login': 'admin', 'password': 'admin1234567'})
        client.csrf = session['csrf']
        # Check every saved original identity, including a pair that has since
        # been deleted, against the source returned by that original command.
        for item in capacity['writeIdentities']:
            kind = oracle.bycode[item['code']][0]
            route = '/api/v1/' + ('organization' if kind == 'bu' else 'departments')
            value, _ = client.call('POST', route + '/submissions/' + item['key'] + '/resolve')
            check_resolve(value, {'sourceVersion': item['source'], 'submissionId': item['key']})
            result['originalResults'].append({'code': item['code'], 'source': item['source'], 'ok': True})
        for code in sorted({item['code'] for item in capacity['writeIdentities']}):
            value, _ = client.call('GET', oracle.path(code, 1))
            oracle.check_detail(code, value, 1)
            result['originalSources'].append({'code': code, 'source': 1, 'ok': True})
        result['beforeReadBatch'] = cli('size', directory)
        for i in range(100):
            code = f'R6_{"B" if i % 2 == 0 else "D"}_{i:05d}'
            value, _ = client.call('GET', oracle.path(code, 25))
            oracle.check_detail(code, value, 25)
        result['readBatchRequests'] = 100
        result['afterReadBatch'] = cli('size', directory)
        assert result['afterReadBatch']['counts']['securityEvents'] == result['beforeReadBatch']['counts']['securityEvents']
        # Distinct names, independent IDs, current Go authorization and actual
        # business events; authorization calculations are not persisted.
        created_bu = []
        for kind in ['bu', 'dep']:
            result['beforeCreate_' + kind] = cli('size', directory)
            for i in range(10):
                code = 'S126_' + uuid.uuid4().hex[:20]
                body = {'code': code, 'name': '独立新对象' + uuid.uuid4().hex,
                        'effectiveDate': '2026-01-01', 'operationType': 'CREATE',
                        'operationReason': 'NEW_ORGANIZATION'}
                if kind == 'dep':
                    body.update(managementBuId=created_bu[i], parentDepartmentId=None)
                route = '/api/v1/' + ('business-units' if kind == 'bu' else 'departments')
                key = str(uuid.uuid4())
                value, _ = client.call('POST', route, body, key, want=201)
                assert value['outcome'] == 'accepted' and value['submissionId'] == key
                object_id = value['buId' if kind == 'bu' else 'departmentId']
                if kind == 'bu':
                    created_bu.append(object_id)
                detail, _ = client.call('GET', route + '/' + object_id)
                assert detail['selectedRecord']['name'] == body['name']
                assert detail['headSourceVersion'] == value['sourceVersion']
                result['created'].append({'kind': kind, 'id': object_id, 'source': value['sourceVersion']})
                save(output, result)
            result['afterCreate_' + kind] = cli('size', directory)
        server.stop()
        result['finalMaintenance'] = cli('maintain', directory)
        result['completed'] = True
    except Exception as error:
        result['error'] = str(error)
        raise
    finally:
        server.close()
        save(output, result)


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('directory', type=Path)
    parser.add_argument('capacity', type=Path)
    parser.add_argument('output', type=Path)
    args = parser.parse_args()
    run(args.directory, args.capacity, args.output)
