#!/usr/bin/env python3
"""Create/stop only the explicitly disposable V127 container and volume."""
import argparse
import json
import os
from pathlib import Path
import secrets
import subprocess
import time
import uuid
from urllib.parse import urlsplit, urlunsplit

ROOT = Path(__file__).resolve().parents[3]
NAME = 'chnhcm-vanke127'
VOLUME = NAME + '-data'
BIN = Path(os.environ.get('CHNHCM_VANKE_BINARY', str(ROOT / '.local/vanke-127/vanke127'))).resolve()


def docker(*args):
    return subprocess.check_output(['docker', *args], stderr=subprocess.DEVNULL, text=True).strip()


def command(mode, private):
    # The 130 A/B harness uses the same business API and its own explicitly
    # owned task configuration. This is a verification-tool argument adapter.
    candidate=private/'candidate-private.json'
    if candidate.is_file() and json.loads(candidate.read_text()).get('Role') in ['A','B']:
        return [str(BIN),'-mode',mode,'-input',str(private)]
    return [str(BIN),'-mode',mode,'-private',str(private)]


def owned_resource(private):
    candidate=private/'candidate-private.json'
    if candidate.is_file():
        config=json.loads(candidate.read_text())
        if config.get('Role') in ['A','B']:
            expected={'A':('chnhcm-storage126-product','storage126-product'), 'B':('chnhcm-storage130-b','storage130')}
            container,task=expected[config['Role']]
            assert config['Container']==container and config['Volume']==container+'-data'
            labels=json.loads(docker('inspect','--format','{{json .Config.Labels}}',container))
            assert labels.get('io.chnhcm.task')==task and labels.get('io.chnhcm.parent-task')=='storage130' and labels.get('io.chnhcm.temporary')=='true'
            return container,'storage126'
    assert docker('inspect','--format','{{index .Config.Labels "io.chnhcm.task"}}/{{index .Config.Labels "io.chnhcm.temporary"}}',NAME)=='vanke127/true'
    return NAME,'vanke127'


def cli(mode, private):
    return json.loads(subprocess.check_output(command(mode,private), cwd=ROOT))


def prepare(private):
    private = private.resolve()
    assert ROOT not in private.parents and private.stat().st_mode & 0o777 == 0o700
    for kind, name in [('container', NAME), ('volume', VOLUME)]:
        assert subprocess.run(['docker', kind, 'inspect', name], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL).returncode != 0, 'refusing existing resource'
    password, api_password = secrets.token_hex(32), secrets.token_hex(32)
    env = private / 'postgres.env'
    fd = os.open(env, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(fd, 'w') as f:
        f.write('POSTGRES_DB=vanke127\nPOSTGRES_PASSWORD=' + password + '\nPOSTGRES_INITDB_ARGS=--encoding=UTF8 --locale=C\n')
    labels = ['--label', 'io.chnhcm.task=vanke127', '--label', 'io.chnhcm.temporary=true']
    docker('volume', 'create', *labels, VOLUME)
    try:
        image = (ROOT / 'deploy/local/postgres-image.txt').read_text().strip()
        docker('run', '-d', '--name', NAME, *labels, '-p', '127.0.0.1::5432', '--env-file', str(env), '-v', VOLUME + ':/var/lib/postgresql', image, 'postgres', '-c', 'synchronous_commit=on', '-c', 'timezone=UTC')
        port = docker('port', NAME, '5432/tcp').rsplit(':', 1)[1]
        config = dict(AdminDSN=f'postgres://postgres:{password}@127.0.0.1:{port}/vanke127?sslmode=disable', APIDSN=f'postgres://chnhcm_api:{api_password}@127.0.0.1:{port}/vanke127?sslmode=disable', Instance=str(uuid.uuid4()))
        fd = os.open(private / 'database-private.json', os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(fd, 'w') as f:
            json.dump(config, f)
        for _ in range(100):
            # initdb briefly starts a socket-only server before the final TCP
            # server. Do not mistake that phase for an API-connectable database.
            if subprocess.run(['docker', 'exec', NAME, 'pg_isready', '-h', '127.0.0.1', '-U', 'postgres', '-d', 'vanke127'], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL).returncode == 0:
                break
            time.sleep(.2)
        else:
            raise RuntimeError('task database startup timeout')
        result = cli('initialize', private)
        print(json.dumps(dict(container=NAME, volume=VOLUME, temporary=True, port=port, **result)))
    finally:
        env.unlink()


def resume(private):
    labels = json.loads(docker('inspect', '--format', '{{json .Config.Labels}}', NAME))
    assert labels.get('io.chnhcm.task') == 'vanke127' and labels.get('io.chnhcm.temporary') == 'true'
    docker('start', NAME)
    port = docker('port', NAME, '5432/tcp')
    assert port.startswith('127.0.0.1:')
    path = private / 'database-private.json'
    config = json.loads(path.read_text())
    for key in ['AdminDSN', 'APIDSN']:
        u = urlsplit(config[key])
        assert u.hostname == '127.0.0.1' and u.path == '/vanke127'
        config[key] = urlunsplit((u.scheme, u.netloc.rsplit('@', 1)[0] + '@' + port, u.path, u.query, u.fragment))
    path.write_text(json.dumps(config))
    assert path.stat().st_mode & 0o777 == 0o600
    for _ in range(100):
        if subprocess.run(['docker','exec',NAME,'pg_isready','-U','postgres','-d','vanke127'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL).returncode == 0:
            print(json.dumps(dict(container=NAME, resumed=True, port=port)))
            return
        time.sleep(.2)
    raise RuntimeError('task database readiness timeout')


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('private', type=Path)
    p.add_argument('--mode', default='prepare')
    a = p.parse_args()
    if a.mode == 'prepare':
        prepare(a.private)
    elif a.mode == 'resume':
        resume(a.private)
    else:
        print(json.dumps(cli(a.mode, a.private)))
