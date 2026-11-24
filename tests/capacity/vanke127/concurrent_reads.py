#!/usr/bin/env python3
"""Check overlapping reads in one fixed-case browser session; write no business facts."""
import argparse
import concurrent.futures
import json
from pathlib import Path
import threading
from urllib.parse import urlsplit

from load import Client


def run(destination, origin='http://127.0.0.1:4388'):
    address = urlsplit(origin)
    assert address.scheme == 'http' and address.hostname == '127.0.0.1' and address.port and not address.path and not address.query and not address.fragment and not address.username
    result = dict(origin=origin, businessWrites=0, batches=[], passed=False)
    for login in ['admin', 'reader']:
        client = Client(result['origin'])
        status, value, _ = client.call('POST', '/api/v1/session/login',
                                       dict(login=login, password='admin1234567'))
        assert status == 200, 'normal fixed-case login failed'
        client.csrf = value['csrf']
        for count in [2, 4]:
            # Context/session requests overlap all list reads, as on normal page
            # mount. Sharing one client means sharing its real session cookie.
            paths = ['/api/v1/departments'] * count + ['/api/v1/departments/context', '/api/v1/session']
            barrier = threading.Barrier(len(paths))

            def request(path):
                barrier.wait(timeout=10)
                status, value, elapsed = client.call('GET', path)
                ok = status == 200
                if path == '/api/v1/departments':
                    ok = ok and value.get('count') == 87343 and len(value.get('items', [])) == 20
                    version = value.get('queryKey'); ok = ok and 'selectionVersion' not in value
                else:
                    version = None
                return dict(route=path, status=status, ms=round(elapsed, 3), ok=ok,
                            count=value.get('count'), code=value.get('code')), version

            with concurrent.futures.ThreadPoolExecutor(len(paths)) as pool:
                replies = list(pool.map(request, paths))
            versions = [v for _, v in replies if v]
            batch = dict(login=login, listRequests=count, sameSession=True,
                         samples=[s for s, _ in replies],
                         sameQueryKey=len(versions) == count and len(set(versions)) == 1)
            batch['passed'] = all(s['ok'] for s, _ in replies) and batch['sameQueryKey']
            result['batches'].append(batch)
            destination.write_text(json.dumps(result, indent=2) + '\n')
    result['passed'] = all(b['passed'] for b in result['batches'])
    destination.write_text(json.dumps(result, indent=2) + '\n')
    print(json.dumps(dict(passed=result['passed'], requests=sum(len(b['samples']) for b in result['batches']),
                          failures=sum(not s['ok'] for b in result['batches'] for s in b['samples']))))
    assert result['passed'], 'same-session fixed-case reads failed; evidence retained'


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('destination', type=Path)
    parser.add_argument('--origin', default='http://127.0.0.1:4388')
    args = parser.parse_args(); run(args.destination, args.origin)
