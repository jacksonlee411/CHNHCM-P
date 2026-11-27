from pathlib import Path
import subprocess
import tempfile
import unittest

from project_skills import project_skill_errors


class ProjectSkillsTest(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name) / 'checkout'
        subprocess.run(['git', 'init', '-q', str(self.root)], check=True)
        self.skill = self.root / '.agents/skills/chnhcm-p-example'
        self.skill.mkdir(parents=True)
        self.entry = self.skill / 'SKILL.md'
        self.entry.write_text('---\nname: chnhcm-p-example\ndescription: Example\n---\n')

    def test_repository_files_and_relative_resource(self):
        guide = self.root / 'guide.md'
        guide.write_text('example')
        (self.skill / 'guide.md').symlink_to('../../../guide.md')
        self.assertEqual([], project_skill_errors(self.root))

    def test_external_link_and_mismatched_entry(self):
        (self.skill / 'outside').symlink_to(self.root.parent)
        self.entry.write_text('---\nname: other\ndescription: Example\n---\n')
        errors = project_skill_errors(self.root)
        self.assertTrue(any('outside' in error for error in errors))
        self.assertTrue(any('SKILL.md' in error for error in errors))

    def test_global_installer_and_misplaced_skill(self):
        (self.root / 'SKILL.md').write_text('misplaced')
        script = self.root / 'setup.sh'
        account_home = 'CODEX' + '_HOME'
        script.write_text('dest="${' + account_home + ':-$HOME/.codex}"\nmkdir -p "$dest/skills"\n')
        errors = project_skill_errors(self.root)
        self.assertTrue(any('setup.sh' in error for error in errors))
        self.assertTrue(any('SKILL.md' in error for error in errors))

    def test_shared_destinations_in_workflows_and_path_builders(self):
        shared = '.agents'
        samples = {
            'workflow.yml': f'steps:\n  - run: mkdir -p "$HOME/{shared}/skills/example"\n',
            'setup.py': f'(Path.home() / "{shared}" / "skills" / "example").mkdir()\n',
            'setup.mjs': f'fs.mkdirSync(path.join(os.homedir(), "{shared}", "skills", "example"));\n',
            'parent.sh': f'mkdir -p "../{shared}/skills/example"\n',
            'sibling.sh': f'cp -r source "../other-checkout/{shared}/skills/example"\n',
        }
        for name, source in samples.items():
            with self.subTest(name=name):
                path = self.root / name
                path.write_text(source)
                self.assertTrue(any(name in error for error in project_skill_errors(self.root)))
                path.unlink()
        (self.root / 'local.sh').write_text(f'mkdir -p "{shared}/skills/example"\n')
        self.assertEqual([], project_skill_errors(self.root))
