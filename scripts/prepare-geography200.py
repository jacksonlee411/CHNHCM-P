#!/usr/bin/env python3
"""Prepare the frozen G0 geographic reference input; never connect to a service."""
import argparse
import collections
import gzip
import hashlib
import json
from pathlib import Path
import zipfile

ROOT = Path(__file__).resolve().parents[1]
EXCLUDED = {'mca-2024:710000', 'mca-2025:810000', 'mca-2025:820000'}

def validate(countries, areas):
    cs = {x['sourceKey']: x for x in countries}
    nodes = {x['sourceKey']: x for x in areas}
    if len(cs) != len(countries) or len(nodes) != len(areas):
        raise ValueError('duplicate source key')
    if len({x['code'] for x in countries}) != len(countries):
        raise ValueError('duplicate country code')
    seen_codes = set()
    for x in areas:
        if x['countryKey'] not in cs or cs[x['countryKey']]['code'] not in ('CN', 'HK', 'MO', 'TW'):
            raise ValueError('country outside scope')
        key = (x['countryKey'], x['scheme'], x['code'])
        if key in seen_codes:
            raise ValueError('duplicate area code')
        seen_codes.add(key)
        chain = {x['sourceKey']}
        parent = x['parentKey']
        while parent:
            if parent in chain:
                raise ValueError('cycle')
            if parent not in nodes or nodes[parent]['countryKey'] != x['countryKey']:
                raise ValueError('missing or cross-country parent')
            chain.add(parent)
            parent = nodes[parent]['parentKey']

def build(archive_path=None):
    index = json.loads((ROOT / 'docs/reference/200-geography-sources.json').read_text())
    source = index['cubebox']
    archive_path = archive_path or ROOT / source['archive']
    if not archive_path.is_file():
        raise ValueError('provide the authorized offline archive using --source-archive')
    data = archive_path.read_bytes()
    if hashlib.sha256(data).hexdigest() != source['archive_sha256']:
        raise ValueError('archive digest mismatch')
    with zipfile.ZipFile(archive_path) as z:
        for member in source['members']:
            if hashlib.sha256(z.read(member['path'])).hexdigest() != member['sha256']:
                raise ValueError('source member digest mismatch')
        def read(name):
            return json.loads(z.read('apps/prototype/src/data/geography/' + name + '.json'))
        baseline, official, supplements = read('baseline'), read('official-divisions'), read('supplements')
    countries = [dict(sourceKey=x['id'], scheme='odoo-country', code=x['code'], name=x['name'],
                      nameEn=x.get('nameEn'), phoneCode=x.get('phoneCode'), effectiveDate='1900-01-01')
                 for x in baseline['rows'] if x['kind'] == 'country']
    areas = []
    for x in official['records']:
        if x['id'] in EXCLUDED:
            continue
        areas.append(dict(sourceKey=x['id'], countryKey=x['countryId'], parentKey=x['parentId'] or None,
                          scheme=official['sources'][x['source']]['scheme'], code=x['code'], name=x['name'],
                          level=x['level'], nature='address_region' if x['level']=='region' else 'administrative',
                          effectiveDate='1900-01-01', sourceVersion=official['sources'][x['source']]['version']))
    countries.sort(key=lambda x:x['sourceKey']);areas.sort(key=lambda x:x['sourceKey'])
    validate(countries, areas)
    if len(countries)!=251 or len(areas)!=3620:
        raise ValueError('frozen input counts changed')
    dispositions = [dict(sourceKey=x['id'], name=x['name'], disposition='excluded_cross_reference')
                    for x in official['records'] if x['id'] in EXCLUDED]
    dispositions += [dict(sourceKey=x['id'], name=x['name'], disposition='held_unverified_address_area',
                          officialCode=None, reason='No current item-level verification; no inferred alias or English translation')
                     for x in supplements['cities'] if x['country']=='base.mo']
    unmatched = sorted(k for k,v in official['mappings'].items() if not v.get('officialId'))
    package = dict(schema=1, version='chnhcm-geography-g0-v1', countries=countries, areas=areas,
                   dispositions=dispositions, unmappedLegacyKeys=unmatched,
                   sourceArchiveSHA256=source['archive_sha256'])
    payload = gzip.compress((json.dumps(package,ensure_ascii=False,sort_keys=True,separators=(',',':'))+'\n').encode(),mtime=0)
    counts = collections.Counter(x['countryKey'] for x in areas)
    manifest = dict(version=package['version'], defaultEffectiveDate='1900-01-01',
                    archive=source['archive'], sourceArchiveSHA256=source['archive_sha256'],
                    package='package.json.gz', sha256=hashlib.sha256(payload).hexdigest(),
                    countryCount=len(countries), areaCounts=dict(counts), excludedCrossReferences=3,
                    macauHeld=8, unmappedLegacyCount=len(unmatched),
                    locationCount=0, runtimePublished=False,
                    limits=['Fixed source editions, not latest official coverage or complete historical geography.',
                            'MO supports country plus detailed address; all eight old areas are held.',
                            'Source keys are not product UUIDs; Go ingestion and database publication belong to G2/G3.'])
    return {'package.json.gz':payload,'manifest.json':(json.dumps(manifest,ensure_ascii=False,indent=2)+'\n').encode()}

def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--output',type=Path,default=ROOT/'tests/fixtures/geography/v1')
    p.add_argument('--check',action='store_true')
    p.add_argument('--source-archive', type=Path, help='authorized offline source archive')
    args=p.parse_args();outputs=build(args.source_archive)
    if args.check:
        for name,data in outputs.items():
            if (args.output/name).read_bytes()!=data:
                raise ValueError('generated geography drift: '+name)
    else:
        args.output.mkdir(parents=True,exist_ok=True)
        for name,data in outputs.items():(args.output/name).write_bytes(data)
    print(json.dumps({'result':'passed','check_only':args.check,'files':list(outputs)}))
if __name__=='__main__':main()
