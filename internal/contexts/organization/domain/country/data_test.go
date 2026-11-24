package country

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"testing"
)

func text(s string) *string { return &s }
func TestDisplayPatchAndSourceIdentity(t *testing.T) {
	p := Policy{}
	old := Data{Name: "中国", NameEn: text("China"), PhoneCode: text("86")}
	changed, e := p.Apply(old, &Patch{"name": text("中国（显示名称）"), "nameEn": nil})
	if e != nil {
		t.Fatal(e)
	}
	if changed.NameEn != nil || old.NameEn == nil || *old.NameEn != "China" || *changed.PhoneCode != "86" {
		t.Fatal("patch mutated old or omitted field", old, changed)
	}
	*changed.PhoneCode = "1"
	if *old.PhoneCode != "86" {
		t.Fatal("clone aliases original")
	}
	for _, patch := range []Patch{{"code": text("XX")}, {"sourceKey": text("base.xx")}, {"name": nil}, {"phoneCode": text("+86")}} {
		if _, e := p.Apply(old, &patch); e == nil {
			t.Fatal("invalid patch accepted", patch)
		}
	}
	for _, code := range []string{"CN", "HK", "MO", "TW", "XI", "XK"} {
		o := Origin{Scheme: Scheme, Code: code, Key: "base." + lower(code), Name: "来源名", PackageVersion: "v1", PackageDigest: repeat('a'), ArchiveDigest: repeat('b')}
		if e := ValidateOrigin(o); e != nil {
			t.Fatal(code, e)
		}
	}
}
func lower(s string) string {
	b := []byte(s)
	for i := range b {
		b[i] += 32
	}
	return string(b)
}
func repeat(b byte) string {
	v := make([]byte, 64)
	for i := range v {
		v[i] = b
	}
	return string(v)
}
func TestDirectoryUsesSharedTimelineAndOwnReasons(t *testing.T) {
	ctx := context.Background()
	first := foundation.NewID()
	initial, e := Kernel.Create(ctx, first, "1890-01-01", MinDate, "SOURCE_INITIALIZATION", Data{Name: "来源名称"})
	if e != nil {
		t.Fatal(e)
	}
	initial.Records[0].Revision = foundation.NewID()
	stop, e := Kernel.Calculate(ctx, initial.Records, first, MinDate, []Item{{ID: "stop", Action: "ADD", Date: "2020-01-01", Type: "DEACTIVATE", Reason: "STOP_SELECTION"}})
	if e != nil {
		t.Fatal(e)
	}
	if len(stop.Records) != 2 || stop.Records[1].State != "inactive" || stop.Records[1].Data.Name != "来源名称" || stop.Records[1].TypeLabel != "停止选用" {
		t.Fatal(stop)
	}
	if _, e := Kernel.Calculate(ctx, initial.Records, first, MinDate, []Item{{ID: "wrong", Action: "ADD", Date: "2020-01-01", Type: "DEACTIVATE", Reason: "BUSINESS_CLOSURE"}}); e == nil {
		t.Fatal("BU reason leaked into directory")
	}
	if (Policy{}).OperationReasons("CHANGE")["DISPLAY_DATA_UPDATE"] != "显示资料更新" {
		t.Fatal("missing display reason")
	}
}
