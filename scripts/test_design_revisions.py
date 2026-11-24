import unittest

from design_revisions import resolve_hashes


class DesignRevisionTests(unittest.TestCase):
    def setUp(self):
        self.path = 'docs/architecture/target/modules/m02.md'
        self.base = [{'target_path': self.path, 'adopted_sha256': 'a' * 64}]

    def revision(self, before='a', after='b', path=None, rid='CHNHCM-009'):
        return {'id': rid, 'files': [{'path': path or self.path, 'before_sha256': before * 64, 'after_sha256': after * 64}]}

    def test_unchanged_source_and_valid_local_chain(self):
        self.assertEqual(resolve_hashes(self.base, [])[self.path], 'a' * 64)
        changes = [self.revision(), self.revision('b', 'c', rid='CHNHCM-014')]
        self.assertEqual(resolve_hashes(self.base, changes)[self.path], 'c' * 64)
        self.assertEqual(self.base[0]['adopted_sha256'], 'a' * 64)

    def test_reject_skipping_an_existing_revision(self):
        with self.assertRaisesRegex(ValueError, 'broken revision chain'):
            resolve_hashes(self.base, [self.revision('c', 'b')])

    def test_reject_unknown_file_and_duplicate_path(self):
        with self.assertRaisesRegex(ValueError, 'unknown or duplicate'):
            resolve_hashes(self.base, [self.revision(path='../../outside.md')])
        row = self.revision()
        row['files'] *= 2
        with self.assertRaisesRegex(ValueError, 'unknown or duplicate'):
            resolve_hashes(self.base, [row])

    def test_reject_duplicate_revision_and_invalid_digest(self):
        with self.assertRaisesRegex(ValueError, 'duplicate or invalid'):
            resolve_hashes(self.base, [self.revision(), self.revision('b', 'c')])
        with self.assertRaisesRegex(ValueError, 'invalid or unchanged'):
            resolve_hashes(self.base, [self.revision(after='z')])

    def test_append_revision_of_same_document_preserves_chain(self):
        changes = [self.revision(), self.revision('b', 'c', rid='CHNHCM-014')]
        changes.append({**self.revision('c', 'd'), 'revision': 2})
        self.assertEqual(resolve_hashes(self.base, changes)[self.path], 'd' * 64)
        self.assertNotIn('revision', changes[0])
        self.assertEqual(self.base[0]['adopted_sha256'], 'a' * 64)
        changes[-1]['files'][0]['before_sha256'] = 'b' * 64
        with self.assertRaisesRegex(ValueError, 'broken revision chain'):
            resolve_hashes(self.base, changes)

    def test_reject_invalid_missing_and_duplicate_sequence(self):
        for value in (0, -1, True, 1.5, '2', None):
            with self.subTest(value=value), self.assertRaisesRegex(ValueError, 'invalid local revision number'):
                resolve_hashes(self.base, [{**self.revision(), 'revision': value}])
        for changes in (
            [{**self.revision(), 'revision': 2}],
            [self.revision(), {**self.revision('b', 'c'), 'revision': 3}],
            [self.revision(), {**self.revision('b', 'c'), 'revision': 2},
             {**self.revision('c', 'd'), 'revision': 2}],
        ):
            with self.subTest(changes=changes), self.assertRaisesRegex(ValueError, 'duplicate or invalid local revision sequence'):
                resolve_hashes(self.base, changes)


if __name__ == '__main__':
    unittest.main()
