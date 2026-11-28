"""4496 人员名录的合成数据。重入时沿用原请求正文、业务提交号与回执。"""
import argparse
import json
import os
from pathlib import Path
from http.cookiejar import CookieJar
from urllib.error import HTTPError
from urllib.parse import urlsplit
from urllib.request import build_opener, HTTPCookieProcessor, ProxyHandler, Request
from uuid import NAMESPACE_URL, uuid5



def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("origin")
    parser.add_argument("journal", type=Path)
    parser.add_argument("--prepare-only", action="store_true")
    args = parser.parse_args()
    origin = args.origin.rstrip("/")
    if urlsplit(origin).hostname != "127.0.0.1":
        parser.error("请使用本机验收环境")
    os.umask(0o077)
    client = build_opener(ProxyHandler({}), HTTPCookieProcessor(CookieJar()))
    headers = {"Origin": origin, "Content-Type": "application/json"}
    base = origin + "/api/v1/"

    def call(path, body=None, key=None):
        content = None if body is None else json.dumps(body).encode()
        request = Request(base + path, data=content, headers=headers)
        if key:
            request.add_header("Idempotency-Key", key)
        try:
            with client.open(request, timeout=40) as response:
                return json.load(response)
        except HTTPError as error:
            detail = json.load(error).get("message", "")
            raise RuntimeError(f"{path} HTTP {error.code} {detail}") from error

    read = call
    session = call("session/login", {"login": "admin", "password": "admin1234567"})
    headers["X-CSRF-Token"] = session["csrf"]
    instance = session["instanceId"]
    state = json.loads(args.journal.read_text()) if args.journal.exists() else {"instance": instance, "origin": origin, "steps": {}}
    if (instance, origin) != (state["instance"], state["origin"]):
        raise ValueError("回执与当前环境不匹配")

    def identity(label):
        return str(uuid5(NAMESPACE_URL, instance + "/405/" + label))

    def save():
        args.journal.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
        pending = args.journal.with_suffix(".tmp")
        pending.write_text(json.dumps(state, ensure_ascii=False))
        pending.replace(args.journal)

    def post(label, path, body):
        step = state["steps"].setdefault(label, {"path": path, "input": body})
        if step["path"] != path:
            raise ValueError(label + " 的原件路径已变化")
        if "result" not in step:
            save()
            key = identity(label + "/" + json.dumps(step["input"], sort_keys=True))
            step["result"] = call(path, step["input"], key)
            save()
        return step["result"]

    def create(label, path, name, reason, key, **fields):
        return post(label, path, dict(code="D405"+label.upper(), name=name, effectiveDate="2026-01-01",
                    operationType="CREATE", operationReason=reason, **fields))[key]

    bu = create("bu", "business-units", "名录示例事业部", "NEW_ORGANIZATION", "buId")
    deps = {}
    for label, name, parent in [
        ("d1", "示例运营总部", None), ("d2", "华东区域", "d1"), ("d3", "客户服务中心", "d2"),
        ("d4", "服务交付部", "d3"), ("d5", "一线服务组", "d4"), ("east", "质量支持组", "d4"),
    ]:
        deps[label] = create(label, "departments", name, "NEW_ORGANIZATION", "departmentId",
                             managementBuId=bu, parentDepartmentId=deps.get(parent))
    cn = next(item["id"] for item in read("country-regions?q=CN")["items"] if item["code"] == "CN")
    mo = next(item["id"] for item in read("country-regions?q=MO")["items"] if item["code"] == "MO")
    location = create("loc", "locations", "示例服务园区", "NEW_LOCATION", "locationId", countryRegionId=mo, addressLine="示例园区一号")
    family = create("family", "job-families", "客户服务序列", "NEW_JOB_FAMILY", "objectId", description="名录验收")
    function = create("func", "job-functions", "客户服务职能", "NEW_JOB_FUNCTION", "objectId", description="名录验收")
    job = create("job", "jobs", "客户服务专员", "NEW_JOB", "objectId", jobFamilyId=family, jobFunctionId=function, responsibilities="受理客户服务", requirements="服务沟通")
    positions = [create("pos"+str(i), "positions", name, "NEW_POSITION", "objectId", departmentId=deps[dept], locationId=location, jobId=job)
                 for i, (name, dept) in enumerate([("服务值班岗", "d5"), ("服务质检岗", "east")])]
    employer = create("employer", "legal-entities", "名录示例用人单位", "NEW_LEGAL_ENTITY", "legalEntityId", countryRegionId=cn,
                      subjectType="legal_person", registrationType="登记证", registrationNumber="D405SYNTH", establishingOrganization="合成设立单位")
    post("purpose", "legal-entities/"+employer+"/purposes", {"expectedVersion": "", "row": {
        "id": identity("purpose"), "role":"employer", "kind":"declaration", "basis":"示例用工资格", "from":"2026-01-01"}})

    for purpose, mode in [("hire", "JobOnly"), ("secondary", "PositionExempt")]:
        policies = read("assignment-policies")
        existing = next((row for row in policies["rows"] if row["category"] == "Employee" and row["purpose"] == purpose and row["from"] <= "2026-06-01" < (row.get("until") or "9999-12-31")), None)
        if existing:
            if existing["mode"] != mode or (existing.get("until") or "9999-12-31") < "2027-04-01":
                raise ValueError("已有政策期间不兼容：" + purpose)
        else:
            post("policy"+purpose, "assignment-policies", {"expectedVersion": policies["version"], "row": {
                "id":identity(purpose),"category":"Employee","purpose":purpose,"mode":mode,"from":"2026-01-01","until":"2027-04-01"}})

    people = []
    for i, name in enumerate(["示例陈舒", "示例刘宁", "示例周平", "示例林青"]):
        label = "person" + str(i)
        people.append(post(label, "persons", {"submissionId": identity(label), "name": name,
                       "primaryDocument": {"kind":"passport","number":"SYN-D405-0"+str(i)}})["id"])
    if args.prepare_only:
        return
    for i, (person, dept, date) in enumerate(zip(people, ["d4", "d5", "east", "d4"], ["2026-06-08", "2026-07-13", "2026-09-07", "2026-10-19"])):
        hired = post("hire"+str(i), "hires", {"person": person, "employer":employer, "department":deps[dept], "job":job, "location":location,
                      "arrival":date, "lastDay":"2027-03-31", "notice":"名录示例入职通知"})
        if i == 0:
            path = "assignments/" + hired["assignment"]
            for n, (position, start) in enumerate(zip(positions, ["2026-09-14", "2026-10-12"])):
                source = read(path + "?businessAt=" + start)
                post("secondary"+str(n), path+"/additional-assignments", {"from":start,"lastDay":"2027-03-31","expectedSourceVersion":source["sourceVersion"],
                     "parallel":False,"position":position,"notice":"名录示例兼岗通知"})
    print("四笔首聘、两笔兼岗已取得原提交回执")


if __name__ == "__main__":
    main()
