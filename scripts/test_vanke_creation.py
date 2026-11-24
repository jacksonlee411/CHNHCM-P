"""Creation recovery keeps requests durable and replays interrupted receipt batches."""
import io
import json
from pathlib import Path
import sqlite3
import tempfile
import threading
import sys
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'tests/capacity/vanke127'))
try:
    import create
    from load import source_db
finally:
    sys.path.pop(0)


class CreationJournalTest(unittest.TestCase):
    def test_lost_response_replays_same_durable_requests(self):
        with tempfile.TemporaryDirectory() as directory:
            private = Path(directory)
            (private / 'import.sqlite').touch()
            db = source_db(private, private)
            db.execute("INSERT INTO objects VALUES('ROOT','root-id','business-units','root-source')")
            for number, code in enumerate(['A', 'B'], 3):
                row = dict(org_code=code, name=code, parent_org_code='ROOT', effective_date='1984-01-01')
                db.execute("INSERT INTO rows VALUES('create',?,?,?,1,NULL)", (number, code, json.dumps(row)))
            db.commit()
            db.close()
            applied, lock, lose_response = {}, threading.Lock(), [True]
            test = self

            class Client:
                def call(self, method, path, body, key):
                    # A separate connection sees the complete batch before HTTP.
                    with sqlite3.connect(private / 'import.sqlite') as observer:
                        test.assertEqual(observer.execute('SELECT count(distinct row_no) FROM attempts').fetchone()[0], 2)
                        recorded = observer.execute('SELECT method,path,body FROM attempts WHERE key=? ORDER BY attempt DESC LIMIT 1', (key,)).fetchone()
                        test.assertEqual((recorded[0], recorded[1], json.loads(recorded[2])), (method, path, body))
                    with lock:
                        if key in applied:
                            test.assertEqual(applied[key][0], body)
                        else:
                            applied[key] = (body, dict(outcome='accepted', departmentId=body['code']+'-id', sourceVersion=body['code']+'-source'))
                        if body['code'] == 'B' and lose_response[0]:
                            lose_response[0] = False
                            raise RuntimeError('response lost after acceptance')
                        return 201, applied[key][1], 1.0

            class Server:
                stdout = io.StringIO('{"url":"http://127.0.0.1:1","apiConnectionLimit":12}\n')
                def send_signal(self, _): pass
                def wait(self, timeout): pass

            def run():
                with patch.object(create.subprocess, 'Popen', return_value=Server()), patch.object(create, 'authenticated_clients', return_value=[Client(), Client()]):
                    Server.stdout.seek(0)
                    create.run(private, private, private/'result.json', 2, 0)

            with self.assertRaisesRegex(RuntimeError, 'response lost'):
                run()
            with sqlite3.connect(private / 'import.sqlite') as observer:
                self.assertEqual(observer.execute('SELECT count(*) FROM attempts').fetchone()[0], 2)
                self.assertEqual(observer.execute('SELECT count(*) FROM rows WHERE result IS NOT NULL').fetchone()[0], 0)
                self.assertEqual(observer.execute('SELECT count(*) FROM objects').fetchone()[0], 1)
            run()
            self.assertEqual(len(applied), 2)
            self.assertTrue(json.loads((private/'result.json').read_text())['completed'])
            with sqlite3.connect(private / 'import.sqlite') as observer:
                self.assertEqual(observer.execute('SELECT count(*) FROM rows WHERE result IS NOT NULL').fetchone()[0], 2)
                self.assertEqual(observer.execute('SELECT count(*) FROM objects').fetchone()[0], 3)
                self.assertEqual(observer.execute('SELECT count(*) FROM attempts').fetchone()[0], 4)
                self.assertEqual(observer.execute('SELECT count(distinct key) FROM attempts').fetchone()[0], 2)


if __name__ == '__main__':
    unittest.main()
