from pathlib import Path
import subprocess,os,shutil,sys
os.umask(0o077)
root=Path.cwd().resolve();dst=Path(sys.argv[1]).resolve();templates=Path(__file__).parent.resolve()
assert root not in dst.parents and not dst.exists()
dst.mkdir(mode=0o700)
p=subprocess.Popen(['git','archive','HEAD'],stdout=subprocess.PIPE)
subprocess.run(['tar','-x','-C',str(dst)],stdin=p.stdout,check=True);assert p.wait()==0
for f in ['internal/bootstrap/storagecase130/sql_observation.go','internal/bootstrap/capacitydiag/diagnostics.go','internal/contexts/organization/infra/history/storage.go','tests/capacity/remediation133/cold_save.py']:
 shutil.copyfile(root/f,dst/f)
shutil.copyfile(templates/'history.go.txt',dst/'internal/contexts/organization/infra/history/head_prototype133.go')
shutil.copyfile(templates/'modes.go.txt',dst/'internal/bootstrap/storagecase130/head_prototype133.go')
f=dst/'internal/contexts/organization/infra/history/read_segments.go';s=f.read_text();needle='func saveReadSegments(ctx context.Context, tx pgx.Tx, c Chunk, source string, full compact.Values, raw []byte) error {';assert s.count(needle)==1;s=s.replace(needle,needle+'\n if c.Kind == BusinessUnit { return SaveProof133(ctx, tx, c, source, full, raw, true) }');f.write_text(s)
f=dst/'internal/bootstrap/runtime.go';s=f.read_text();assert s.count('m02-compact-5')==2;f.write_text(s.replace('m02-compact-5','m02-head-proof133'))
subprocess.run(['gofmt','-w',str(dst/'internal/contexts/organization/infra/history/head_prototype133.go'),str(dst/'internal/bootstrap/storagecase130/head_prototype133.go'),str(dst/'internal/contexts/organization/infra/history/read_segments.go')],check=True)

f=dst/'internal/bootstrap/storagecase130/sql_observation.go';s=f.read_text().replace('[]string{"department_read_segment"','[]string{"proof133_node", "proof133_source", "department_read_segment"');f.write_text(s)
