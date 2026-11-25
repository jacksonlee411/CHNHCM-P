//go:build integration

package bootstrap_test

import f "chnhcm.local/chnhcm/internal/foundation"

type lifecycleSample struct{ assignmentFixture }

func (x lifecycleSample) view(id, day, source string) map[string]any {
	return x.h.b.call("GET", "/api/v1/assignments/"+id+"?businessAt="+day+"&sourceVersion="+source, nil, "", 200)
}
func (x lifecycleSample) post(id, operation, day string, body map[string]any, status int) map[string]any {
	body["expectedSourceVersion"] = x.view(id, day, "")["sourceVersion"]
	return x.h.b.call("POST", "/api/v1/assignments/"+id+"/"+operation, body, f.NewID(), status)
}
func (x lifecycleSample) hirePerson(name string, grade string) map[string]any {
	type M = map[string]any
	person := x.h.b.call("POST", "/api/v1/persons", M{"submissionId": f.NewID(), "name": name, "primaryDocument": M{"kind": "passport", "number": f.NewID()}}, "", 200)["id"]
	return x.h.b.call("POST", "/api/v1/hires", M{"person": person, "employer": x.entity, "department": x.dep, "job": x.job, "grade": grade, "arrival": "2027-04-01", "notice": "合成独立任用决定"}, f.NewID(), 200)
}
func (x lifecycleSample) policy(purpose string) {
	type M = map[string]any
	p := x.h.b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
	x.h.b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": p["version"], "row": M{"id": f.NewID(), "category": "Employee", "purpose": purpose, "mode": "JobOnly", "from": "2027-04-01"}}, f.NewID(), 200)
}
func (x lifecycleSample) exit(id, op, date, meaning string, code int) map[string]any {
	return x.post(id, op, date, map[string]any{"date": date, "meaning": meaning, "notice": "原任职退出通知"}, code)
}
