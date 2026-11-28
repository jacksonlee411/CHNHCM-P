package application

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"testing"
	"time"
)

func TestManagementCeilingCompletePermission(t *testing.T) {
	now := time.Now().UTC()
	until := now.Add(24 * time.Hour)
	late := until.Add(time.Second)
	c := ManagementCeiling{ManagementLimit: access.ManagementLimit{Owner: "A", MaxProxyDays: 1, Until: &until}, TargetKind: "bu", TargetID: "A", Actions: []string{"department.record.correct"}, Read: []string{"code", "name"}, Write: []string{"name"}}
	base := PermissionTerms{Owner: "A", Purpose: "department", Source: "proxy", Actions: []string{"department.record.correct"}, TargetKind: "bu", TargetID: "A", Read: []string{"name"}, Write: []string{"name"}, From: &now, Until: &until}
	if !c.Covers(base) {
		t.Fatal("complete permission rejected")
	}
	for name, change := range map[string]func(*PermissionTerms){
		"other account owner": func(p *PermissionTerms) { p.Owner = "B" }, "other object": func(p *PermissionTerms) { p.TargetID = "B" },
		"wrong purpose": func(p *PermissionTerms) { p.Purpose = "staffingquota" }, "extra action": func(p *PermissionTerms) { p.Actions = append(p.Actions, "department.create") },
		"unavailable field": func(p *PermissionTerms) { p.Write = append(p.Write, "parentDepartmentId") },
		"longer proxy":      func(p *PermissionTerms) { p.Until = &late }, "unlimited proxy": func(p *PermissionTerms) { p.Until = nil },
		"unlimited duty from finite mandate": func(p *PermissionTerms) { p.Source = "duty"; p.Until = nil },
	} {
		t.Run(name, func(t *testing.T) {
			p := base
			change(&p)
			if c.Covers(p) {
				t.Fatal("ceiling expanded")
			}
		})
	}
}
