"""Independent history-comparison counterexamples, without database traffic."""
import hashlib
import json
import unittest

from preserve import compare


def row(owner, source, number, value=None, kind='department'):
    digest = hashlib.sha256((value or source).encode()).hexdigest()
    return json.dumps([kind, owner, source, number, digest]).encode() + b'\n'


class HistoryComparisonTests(unittest.TestCase):
    def setUp(self):
        self.a = row('object-a', 'source-a', 1)
        self.b = row('object-a', 'source-b', 2)
        self.c = row('object-c', 'source-c', 1)
        self.original = [self.a, self.b, self.c]

    def test_additions_between_objects_and_at_the_tail(self):
        current = [self.a, self.b, row('object-a', 'source-new', 3),
                   row('object-b', 'source-new-object', 1), self.c,
                   row('object-c', 'source-tail', 2)]
        result = compare(iter(self.original), iter(current))
        self.assertEqual(result['originalSourcesPreserved'], 3)
        self.assertEqual(result['addedSources'], 3)
        self.assertEqual(result['originalUncompressedSHA256'],
                         hashlib.sha256(b''.join(self.original)).hexdigest())

    def test_original_changes_missing_sources_and_reordering_are_rejected(self):
        for current in [
            [row('object-a', 'source-a', 1, 'changed'), self.b, self.c],
            [self.a, self.c], [self.b, self.a, self.c],
            [row('object-a', 'replacement', 1), self.b, self.c],
        ]:
            with self.subTest(current=current):
                with self.assertRaises(AssertionError):
                    compare(iter(self.original), iter(current))

    def test_duplicate_original_cannot_fill_added_source_count(self):
        for current in [
            [self.a, self.b, self.b, self.c],
            [self.a, self.b, self.c, self.c],
            [self.a, self.b, row('object-a', 'source-a', 3), self.c],
        ]:
            with self.subTest(current=current):
                with self.assertRaises(AssertionError):
                    compare(iter(self.original), iter(current))

    def test_corrupt_original_stream_is_not_an_acceptable_oracle(self):
        for original in [
            [self.a, self.a], [self.b, self.a],
            [self.a, row('object-a', 'source-a', 3)],
        ]:
            with self.subTest(original=original):
                with self.assertRaises(AssertionError):
                    compare(iter(original), iter(original))

    def test_object_type_order_and_independent_source_numbers(self):
        original = [row('same-object', 'bu-source', 1, kind='BU'),
                    row('same-object', 'department-source', 1)]
        result = compare(iter(original), iter(original))
        self.assertEqual(result['originalSourcesPreserved'], 2)
        self.assertEqual(result['addedSources'], 0)

    def test_current_only_sources_must_also_be_unique_and_ordered(self):
        for current in [
            [self.a, self.b, row('object-b', 'new', 1), row('object-b', 'new', 2), self.c],
            [self.a, self.b, row('object-b', 'new', 2), row('object-b', 'different', 1), self.c],
            [self.a, self.b, row('object-b', 'new', 1), self.c, row('object-b', 'later', 2)],
        ]:
            with self.subTest(current=current):
                with self.assertRaises(AssertionError):
                    compare(iter(self.original), iter(current))


if __name__ == '__main__':
    unittest.main()
