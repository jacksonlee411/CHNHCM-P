import subprocess
import tempfile
import unittest
from pathlib import Path
from design_revisions import historical_source_exists


class HistoricalSourceLinksTest(unittest.TestCase):
    def test_deleted_source_requires_final_report_and_real_baseline(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            def git(*args):
                return subprocess.check_output(['git', *args], cwd=root, stderr=subprocess.DEVNULL, text=True).strip()
            git('init', '-q')
            for folder in ('internal', 'docs'):
                (root / folder).mkdir()
                (root / folder / 'old.txt').write_text('retained only in Git')
            git('add', '.')
            git('-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.test', 'commit', '-qm', 'fixture')
            report = {'status': 'final', 'base_commit': git('rev-parse', 'HEAD')}
            (root / 'internal/old.txt').unlink()
            self.assertTrue(historical_source_exists(root, report, 'internal/old.txt'))
            self.assertTrue(historical_source_exists(root, report, 'internal'))
            self.assertFalse(historical_source_exists(root, {**report, 'base_commit': git('rev-parse', 'HEAD^{tree}')}, 'internal/old.txt'))
            for record, target in [(None, 'internal/old.txt'), ({**report, 'status': 'draft'}, 'internal/old.txt'), ({**report, 'base_commit': '0' * 40}, 'internal/old.txt'), (report, 'internal/missing.txt'), (report, 'docs/old.txt'), (report, 'internal/../docs/old.txt')]:
                self.assertFalse(historical_source_exists(root, record, target))
            (root / 'docs/verification').mkdir()
            path = 'docs/verification/result.md'
            (root / path).write_text('Final result references newly added code')
            (root / 'internal/new.txt').write_text('new source')
            git('add', '.')
            git('-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.test', 'commit', '-qm', 'report and code')
            (root / 'internal/new.txt').unlink()
            report['path'] = path
            self.assertTrue(historical_source_exists(root, report, 'internal/new.txt'))
            (root / path).write_text('Changed report cannot use the saved snapshot')
            self.assertFalse(historical_source_exists(root, report, 'internal/new.txt'))
