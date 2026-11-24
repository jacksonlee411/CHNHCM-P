//go:build integration

package bootstrap_test

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/internal/contexts/organization/infra/countryseed"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/geoareaseed"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
)

// Build the candidate oracle from source rows and parent keys, without calling
// the product candidate, path or adoption algorithms. Identity mapping was
// independently checked by geoAllSources203 before this measurement stage.
func geographyCandidateOracle205(t *testing.T, f *fixture, country, shape, parent, query string) func(map[string]any) {
	t.Helper()
	areas, e := geoareaseed.Fixed()
	must(t, e)
	countries, e := countryseed.Fixed()
	must(t, e)
	rows := map[string]geoareaseed.Row{}
	parents := map[string]bool{}
	parentKey := ""
	for _, row := range areas.Areas {
		rows[row.SourceKey] = row
		parents[row.ParentKey] = true
		if row.CountryKey == "base."+strings.ToLower(country) && row.Code == parent {
			parentKey = row.SourceKey
		}
	}
	selected := []geoareaseed.Row{}
	for _, row := range areas.Areas {
		if row.CountryKey != "base."+strings.ToLower(country) {
			continue
		}
		if shape == "roots" && row.ParentKey == "" || shape == "children" && row.ParentKey == parentKey || shape == "search" && (strings.Contains(row.Code, query) || strings.Contains(row.Name, query)) {
			selected = append(selected, row)
		}
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].Code < selected[j].Code })
	more := len(selected) > 20
	selected = selected[:min(20, len(selected))]
	var countryName string
	for _, row := range countries.Countries {
		if row.Code == country {
			countryName = row.Name
		}
	}
	want := []org.GeoAreaCandidate{}
	for _, row := range selected {
		chain := []geoareaseed.Row{row}
		for node := row; node.ParentKey != ""; {
			node = rows[node.ParentKey]
			chain = append(chain, node)
		}
		path := []org.GeoAreaPathNode{{ID: geoCountry203(t, f, country), Code: country, Name: countryName}}
		for i := len(chain) - 1; i >= 0; i-- {
			node := chain[i]
			path = append(path, org.GeoAreaPathNode{ID: geoID203(t, f, node.Code), Code: node.Code, Name: node.Name, Level: node.Level})
		}
		nature, scheme := row.Nature, row.Scheme
		want = append(want, org.GeoAreaCandidate{ID: geoID203(t, f, row.Code), Code: row.Code, Name: row.Name, Level: row.Level, Nature: &nature, Scheme: &scheme, Path: path, Terminal: !parents[row.SourceKey]})
	}
	return func(value map[string]any) {
		raw, e := json.Marshal(value)
		must(t, e)
		var got org.GeoAreaCandidates
		must(t, json.Unmarshal(raw, &got))
		if got.Mode != "areas" || got.HasMore != more || (got.Cursor != nil) != more || !reflect.DeepEqual(got.Items, want) {
			t.Fatalf("G5 independent candidate oracle differs for %s/%s: got=%+v want=%+v", country, shape, got, want)
		}
	}
}
