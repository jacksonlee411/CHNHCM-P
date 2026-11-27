"""为隔离的浏览器样本准备第二位管理员，密码与本地公开演示账号相同。"""
from pathlib import Path
import runpy
import sys

root = Path(__file__).resolve().parents[4]
sys.path.insert(0, str(root / "scripts"))
local = runpy.run_path(str(root / "scripts/local-bu.py"))
runtime = local["runtime"]
if not runtime.temporary_profile():
    raise RuntimeError("双人首次核编样本需要临时 profile")
conn = runtime.connection()
runtime.verify_target(conn)
config = local["configuration"]()
enterprise = config.get("enterprise", dict(EnterpriseCode="LOCAL", EnterpriseName="本地学习企业"))
local["run"]([str(local["BIN"] / "upgrader"), "-mode", "initialize"],
             local["environment"](config, conn),
             dict(Instance=config["instance"], **enterprise, Login="quota-315-admin",
                  Password=local["DEVELOPMENT_LOGIN_PASSWORD"], ReadOnly=False, DevelopmentFixture=True))
