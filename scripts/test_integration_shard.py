import importlib.util
import json
import re
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location('integration_shard', Path(__file__).with_name('integration-shard.py'))
sharding = importlib.util.module_from_spec(spec)
spec.loader.exec_module(sharding)


class IntegrationShardingTests(unittest.TestCase):
    def test_full_preflight_moves_exact_roots_and_rejects_a_renamed_root(self):
        early = sorted(sharding.PREFLIGHT_ROOTS)
        rest = [('other', early[0][1]), ('other', 'TestOther')]
        self.assertEqual(sharding.remaining_roots(early + rest), rest)
        with self.assertRaisesRegex(ValueError, 'missing preflight'):
            sharding.remaining_roots(early[1:] + rest)

    def test_filtered_roots_preserve_packages_and_empty_filter_fails(self):
        roots = [('a', 'TestCountryOne'), ('b', 'TestCountryOne'), ('a', 'TestBUOne')]
        selected = sharding.select_roots(roots, '^TestCountry')
        self.assertEqual(selected, roots[:2])
        self.assertEqual(sorted(sum([sharding.partition(selected, i, 2) for i in [1, 2]], [])), sorted(selected))
        with self.assertRaises(ValueError):
            sharding.select_roots(roots, '^Missing')

    def test_complete_disjoint_deterministic_partition(self):
        roots = [('pkg' + str(i % 2), 'Test' + str(i)) for i in range(247)]
        shards = [sharding.partition(roots, i, 8) for i in range(1, 9)]
        self.assertEqual(sorted(sum(shards, [])), sorted(roots))
        self.assertEqual(sum(map(len, shards)), len(set(sum(shards, []))))
        self.assertTrue(all(shards))
        self.assertEqual(shards[0], sharding.partition(roots[::-1], 1, 8))

    def test_invalid_or_empty_partition_is_not_success(self):
        for roots, shard, total in [([], 1, 8), ([('a', 'TestA')], 0, 1),
                                    ([('a', 'TestA')], 2, 1), ([('a', 'TestA')], 1, 2),
                                    ([('a', 'TestA')] * 2, 1, 1)]:
            with self.subTest(shard=shard, total=total), self.assertRaises(ValueError):
                sharding.partition(roots, shard, total)

    def test_discovery_preserves_package_identity_and_ignores_build_output(self):
        events = [{'Action': 'start', 'Package': 'a'},
                  {'Action': 'output', 'Package': 'a', 'Output': 'TestSame\n'},
                  {'Action': 'output', 'Package': 'b', 'Output': 'TestSame\n'},
                  {'Action': 'output', 'Package': 'a', 'Output': 'ok a 0.1s\n'}]
        self.assertEqual(sharding.discover('\n'.join(map(json.dumps, events))),
                         [('a', 'TestSame'), ('b', 'TestSame')])
        for data in ['', json.dumps(events[1]) + '\n' + json.dumps(events[1])]:
            with self.assertRaises(ValueError):
                sharding.discover(data)

    def test_scope_keeps_package_identity_and_rejects_empty_or_invalid_filter(self):
        roots = [('a', 'TestLocationA'), ('b', 'TestLocationA'), ('b', 'TestOther')]
        self.assertEqual(sharding.select_roots(roots, '^TestLocation'), roots[:2])
        self.assertEqual(sharding.select_roots(roots, '^Test'), roots)
        with self.assertRaises(ValueError):
            sharding.select_roots(roots, '^TestMissing')
        with self.assertRaises(re.error):
            sharding.select_roots(roots, '[')
