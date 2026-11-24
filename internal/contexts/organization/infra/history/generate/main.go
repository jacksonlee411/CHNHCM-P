// Command generate emits fixed-field codecs. Reflection is used only during
// generation; production decoding has no type tags or generic JSON evaluator.
package main

import (
	"bytes"
	enterprise "chnhcm.local/chnhcm/internal/contexts/organization/domain/enterprise"
	"fmt"
	"go/format"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	bu "chnhcm.local/chnhcm/internal/contexts/organization/domain"
	country "chnhcm.local/chnhcm/internal/contexts/organization/domain/country"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	geoarea "chnhcm.local/chnhcm/internal/contexts/organization/domain/geoarea"
	legalentity "chnhcm.local/chnhcm/internal/contexts/organization/domain/legalentity"
	location "chnhcm.local/chnhcm/internal/contexts/organization/domain/location"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history/model"
)

type generator struct {
	code    bytes.Buffer
	imports map[string]string
	serial  int
}

func (g *generator) line(s string, args ...any) { fmt.Fprintf(&g.code, s+"\n", args...) }
func (g *generator) next() string               { g.serial++; return fmt.Sprintf("n%d", g.serial) }
func rangeFields(t reflect.Type) (string, string) {
	end, ok := t.FieldByName("End")
	if !ok || end.Type.Kind() != reflect.String {
		return "", ""
	}
	for _, name := range []string{"Date", "Start"} {
		f, ok := t.FieldByName(name)
		if ok && f.Type.Kind() == reflect.String {
			return name, "End"
		}
	}
	return "", ""
}
func (g *generator) pkg(path string) string {
	if name, ok := g.imports[path]; ok {
		return name
	}
	name := fmt.Sprintf("p%d", len(g.imports))
	g.imports[path] = name
	return name
}
func (g *generator) typ(t reflect.Type) string {
	if t == reflect.TypeOf(enterprise.Record{}) {
		return g.pkg("chnhcm.local/chnhcm/internal/contexts/organization/domain/enterprise") + ".Record"
	}
	if t == reflect.TypeOf(geoarea.Record{}) {
		return g.pkg("chnhcm.local/chnhcm/internal/contexts/organization/domain/geoarea") + ".Record"
	}
	if t == reflect.TypeOf(legalentity.Record{}) {
		return g.pkg("chnhcm.local/chnhcm/internal/contexts/organization/domain/legalentity") + ".Record"
	}
	if t == reflect.TypeOf(location.Record{}) {
		return g.pkg("chnhcm.local/chnhcm/internal/contexts/organization/domain/location") + ".Record"
	}
	if t == reflect.TypeOf(country.Record{}) {
		return g.pkg("chnhcm.local/chnhcm/internal/contexts/organization/domain/country") + ".Record"
	}
	if t == reflect.TypeOf(bu.Record{}) {
		return g.pkg("chnhcm.local/chnhcm/internal/contexts/organization/domain") + ".Record"
	}
	if t == reflect.TypeOf(d.Record{}) {
		return g.pkg("chnhcm.local/chnhcm/internal/contexts/organization/domain/department") + ".Record"
	}
	if t.Name() != "" {
		if t.PkgPath() == "" {
			return t.Name()
		}
		return g.pkg(t.PkgPath()) + "." + t.Name()
	}
	switch t.Kind() {
	case reflect.Slice:
		return "[]" + g.typ(t.Elem())
	case reflect.Pointer:
		return "*" + g.typ(t.Elem())
	case reflect.Map:
		return "map[" + g.typ(t.Key()) + "]" + g.typ(t.Elem())
	default:
		panic("unsupported type " + t.String())
	}
}
func (g *generator) write(t reflect.Type, x string) {
	if t == reflect.TypeOf(time.Time{}) {
		g.line("w.String(%s.UTC().Format(time.RFC3339Nano))", x)
		g.imports["time"] = "time"
		return
	}
	if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
		g.line("w.Bytes(%s)", x)
		return
	}
	switch t.Kind() {
	case reflect.String:
		g.line("w.String(string(%s))", x)
	case reflect.Int, reflect.Int32, reflect.Int64:
		g.line("w.Int(int64(%s))", x)
	case reflect.Bool:
		g.line("w.Bool(%s)", x)
	case reflect.Pointer:
		g.line("w.Bool(%s != nil); if %s != nil {", x, x)
		g.write(t.Elem(), "(*"+x+")")
		g.line("}")
	case reflect.Struct:
		start, end := rangeFields(t)
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.Tag.Get("json") == "-" {
				continue
			}
			if !f.IsExported() {
				panic("unexported persisted field " + f.Name)
			}
			if f.Name == end {
				continue
			}
			if f.Name == start {
				g.line("w.Range(%s.%s,%s.%s)", x, start, x, end)
				continue
			}
			g.write(f.Type, x+"."+f.Name)
		}
	case reflect.Slice:
		i := g.next()
		g.line("w.Bool(%s != nil); w.Uint(uint64(len(%s))); for %s := range %s {", x, x, i, x)
		g.write(t.Elem(), x+"["+i+"]")
		g.line("}")
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			panic("map key must be string")
		}
		g.imports["sort"] = "sort"
		keys, k, v := g.next(), g.next(), g.next()
		g.line("w.Bool(%s != nil); w.Uint(uint64(len(%s))); %s:=make([]string,0,len(%s)); for key:=range %s { %s=append(%s,key) }; sort.Strings(%s)", x, x, keys, x, x, keys, keys, keys)
		g.line("for _,%s:=range %s { w.String(%s); %s:=%s[%s]", k, keys, k, v, x, k)
		g.write(t.Elem(), v)
		g.line("}")
	default:
		panic("unsupported persisted kind " + t.String())
	}
}
func (g *generator) read(t reflect.Type, x string) {
	if t == reflect.TypeOf(time.Time{}) {
		g.imports["time"] = "time"
		g.line("if r.Err==nil { %s,r.Err=time.Parse(time.RFC3339Nano,r.String()) }", x)
		return
	}
	if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
		g.line("%s=r.Bytes()", x)
		return
	}
	switch t.Kind() {
	case reflect.String:
		g.line("%s=%s(r.String())", x, g.typ(t))
	case reflect.Int, reflect.Int32, reflect.Int64:
		g.line("%s=%s(r.Int())", x, g.typ(t))
	case reflect.Bool:
		g.line("%s=r.Bool()", x)
	case reflect.Pointer:
		g.line("if r.Bool() { %s=new(%s)", x, g.typ(t.Elem()))
		g.read(t.Elem(), "(*"+x+")")
		g.line("}")
	case reflect.Struct:
		start, end := rangeFields(t)
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.Tag.Get("json") == "-" {
				continue
			}
			if !f.IsExported() {
				panic(f.Name)
			}
			if f.Name == end {
				continue
			}
			if f.Name == start {
				g.line("%s.%s,%s.%s=r.Range()", x, start, x, end)
				continue
			}
			g.read(f.Type, x+"."+f.Name)
		}
	case reflect.Slice:
		present, n, i := g.next(), g.next(), g.next()
		g.line("%s,%s:=r.Bool(),r.Len(); if %s { %s=make(%s,%s) } else if %s!=0 { r.Err=compact.ErrCorrupt }; for %s:=range %s {", present, n, present, x, g.typ(t), n, n, i, x)
		g.read(t.Elem(), x+"["+i+"]")
		g.line("}")
	case reflect.Map:
		present, n, i, k, v := g.next(), g.next(), g.next(), g.next(), g.next()
		g.line("%s,%s:=r.Bool(),r.Len(); if %s { %s=make(%s,%s) } else if %s!=0 {r.Err=compact.ErrCorrupt}; for %s:=0; %s<%s; %s++ { %s:=r.String(); var %s %s", present, n, present, x, g.typ(t), n, n, i, i, n, i, k, v, g.typ(t.Elem()))
		g.read(t.Elem(), v)
		g.line("if _,exists:=%s[%s]; exists||%s==nil {r.Err=compact.ErrCorrupt} else { %s[%s]=%s }; }", x, k, x, x, k, v)
	default:
		panic(t.String())
	}
}
func main() {
	g := generator{imports: map[string]string{"context": "context", "chnhcm.local/chnhcm/internal/platform/compact": "compact"}}
	roots := []any{model.BUState{}, model.BUOperation{}, model.DepartmentState{}, model.DepartmentOperation{}, model.DepartmentEvidence{}, model.DepartmentRead{}, model.BURead{}, model.CountryState{}, model.CountryOperation{}, model.CountryTerminal{}, model.CountryEvidence{}, model.GeoAreaState{}, model.GeoAreaOperation{}, model.GeoAreaTerminal{}, model.GeoAreaEvidence{}, model.LocationState{}, model.LocationOperation{}, model.LocationTerminal{}, model.LocationEvidence{}, model.EnterpriseState{}, model.EnterpriseOperation{}, model.EnterpriseTerminal{}, model.EnterpriseEvidence{}, model.LegalEntityState{}, model.LegalEntityOperation{}, model.LegalEntityTerminal{}, model.LegalEntityEvidence{}}
	for i, root := range roots {
		t := reflect.TypeOf(root)
		name := t.Name()
		typ := g.typ(t)
		format := i + 1
		// 24/25 belong to the simplified location state/operation.
		// Enterprise keeps main's 20..23; new legal-entity shapes use 26..29.
		if strings.HasPrefix(name, "LegalEntity") {
			format += 2
		}
		if name == "LocationState" {
			format = 24
		}
		if name == "LocationOperation" {
			format = 25
		}
		g.line("func Encode%s(ctx context.Context,p *compact.Pool,v %s)([]byte,error) { w:=compact.NewWriter(ctx,p,%d)", name, typ, format)
		g.write(t, "v")
		g.line("return w.Finish() }")
		g.line("func Decode%s(ctx context.Context,p compact.Values,data []byte)(v %s,err error) { r:=compact.NewReader(ctx,p,data,%d)", name, typ, format)
		g.read(t, "v")
		g.line("err=r.Finish(); return }")
	}
	var out bytes.Buffer
	out.WriteString("// Code generated by history/generate; DO NOT EDIT.\npackage history\nimport (\n")
	paths := []string{}
	for p := range g.imports {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		fmt.Fprintf(&out, "%s %s\n", g.imports[p], strconv.Quote(p))
	}
	out.WriteString(")\n")
	out.Write(g.code.Bytes())
	data, err := format.Source(out.Bytes())
	if err != nil {
		fmt.Fprintln(os.Stderr, string(out.Bytes()))
		panic(err)
	}
	path := "internal/contexts/organization/infra/history/codec_generated.go"
	if len(os.Args) > 1 && strings.TrimSpace(os.Args[1]) == "--check" {
		old, e := os.ReadFile(path)
		if e != nil || !bytes.Equal(old, data) {
			panic("history codecs require regeneration")
		}
		return
	}
	if err = os.WriteFile(path, data, 0644); err != nil {
		panic(err)
	}
}
