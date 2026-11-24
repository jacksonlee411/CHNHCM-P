import copy
import importlib.util
from pathlib import Path
import unittest
spec=importlib.util.spec_from_file_location('geography200',Path(__file__).with_name('prepare-geography200.py'))
m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
class GeographyInputTest(unittest.TestCase):
    def setUp(self):
        self.c=[dict(sourceKey='cn',code='CN'),dict(sourceKey='us',code='US')]
        self.a=[dict(sourceKey='a',countryKey='cn',scheme='s',code='01',parentKey=None),dict(sourceKey='b',countryKey='cn',scheme='s',code='02',parentKey='a')]
    def test_missing_parent(self):
        self.a[1]['parentKey']='missing'
        with self.assertRaisesRegex(ValueError,'parent'):m.validate(self.c,self.a)
    def test_cycle(self):
        self.a[0]['parentKey']='b'
        with self.assertRaisesRegex(ValueError,'cycle'):m.validate(self.c,self.a)
    def test_scope(self):
        self.a[0]['countryKey']='us'
        with self.assertRaisesRegex(ValueError,'scope'):m.validate(self.c,self.a)
    def test_duplicate_code(self):
        self.a[1]['code']='01'
        with self.assertRaisesRegex(ValueError,'duplicate area code'):m.validate(self.c,self.a)
    def test_valid_does_not_mutate(self):
        before=copy.deepcopy(self.a);m.validate(self.c,self.a);self.assertEqual(before,self.a)
