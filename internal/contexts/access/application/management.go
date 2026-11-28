package application

import (
	"slices"
	"strings"
	"time"

	"chnhcm.local/chnhcm/internal/contexts/access/domain"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
)

var ManagementRights = []string{"account", "credential", "role", "policy", "assign", "delegate"}
var DepartmentFields = strings.Fields("code name shortName managementBuId parentDepartmentId departmentPath status effectiveDate operationType operationReason")
var DepartmentWrite = strings.Fields("code name shortName managementBuId parentDepartmentId effectiveDate operationType operationReason")
var QuotaFields = strings.Fields("code name departmentId status effectiveDate quota reason audit usageState sourceVersion")
var QuotaWrite = strings.Fields("effectiveDate quota reason")

// These entries are the daily delegation catalog. Migrated rights outside it retain their original meaning.
func ManagementCatalog() []access.CatalogAction {
	const dept = "部门"
	const quota = "设置 → 职务配置 → 部门编制分配"
	items := []access.CatalogAction{}
	add := func(key, name, menu, operation string) {
		items = append(items, access.CatalogAction{Key: key, Name: name, Menu: menu, API: operation})
	}
	add("bu.reference.read", "选择管理业务单元", dept, "departmentDepartmentReferences")
	for _, v := range [][3]string{
		{"read", "查看部门", "departmentDepartments, departmentDepartment, departmentTreeNodes, departmentNavigate"},
		{"history.read", "查看部门历史", "departmentDepartmentChanges"}, {"export", "导出部门", "departmentExportSelection"},
		{"reference.read", "选择上级部门", "departmentParentReferences"}, {"create", "新建部门", "departmentCreateDepartment"},
		{"record.add", "新增部门记录", "departmentAddEffectiveRecord"}, {"record.correct", "更正部门记录", "departmentCorrectRecord"},
		{"reassign", "变更管理业务单元", "departmentAddEffectiveRecord, departmentCorrectRecord"},
		{"deactivate", "停用部门", "departmentDeactivateDepartment"}, {"reactivate", "启用部门", "departmentReactivateDepartment"},
	} {
		add("department."+v[0], v[1], dept, v[2])
	}
	for _, part := range [][2]string{{"first", "首条"}, {"record", "普通"}, {"deactivation", "停用"}, {"reactivation", "启用"}} {
		prefix := "department." + part[0] + "."
		add(prefix+"redate", "调整"+part[1]+"记录日期", dept, "departmentRedateRecord")
		add(prefix+"reclassify", "更正"+part[1]+"记录分类", dept, "departmentCorrectRecord")
		if part[0] != "first" {
			add(prefix+"delete", "删除"+part[1]+"记录", dept, "departmentDeleteRecord")
		}
	}
	for _, v := range [][3]string{{"read", "查看岗位核编", "readDepartmentAllocation, getStaffingQuotaPosition"}, {"history.read", "查看核编历史", "getStaffingQuotaChanges"}, {"create", "建立核编", "saveStaffingQuotaCreate"}, {"adjust", "调整核编", "saveStaffingQuotaAdjust, saveDepartmentAllocation"}, {"correct", "更正核编", "saveStaffingQuotaCorrect"}, {"redate", "调整核编日期", "saveStaffingQuotaRedate"}, {"delete", "删除核编记录", "saveStaffingQuotaDelete"}, {"withdraw", "撤销核编", "saveStaffingQuotaWithdraw"}} {
		add("staffingquota."+v[0], v[1], quota, v[2])
	}
	return items
}

// Legacy grants can contain business actions outside the publication catalog.
// These names are for reading their original binding. Publication still uses ManagementCatalog.
func ManagementActionName(key string) string {
	object, operation, _ := strings.Cut(key, ".")
	objects := map[string]string{"enterprise": "企业", "bu": "业务单元", "department": "部门", "legalentity": "法律实体", "location": "地点", "jobfunction": "职类", "jobfamily": "职务系列", "job": "职务", "grade": "职级", "gradeladder": "职级阶梯", "position": "岗位", "positionhierarchy": "岗位层级", "country": "国家/地区", "geoarea": "行政区划", "replenishment": "岗位补员", "assignmentpolicy": "任职与占编规则", "legalpurpose": "法律实体用途", "staffingquota": "岗位核编"}
	name := objects[object]
	if name == "" {
		return key
	}
	verbs := map[string]string{"read": "查看", "export": "导出", "create": "新建", "deactivate": "停用", "reactivate": "启用", "maintain": "维护", "check": "核对", "publish": "发布"}
	if verb := verbs[operation]; verb != "" {
		return verb + name
	}
	switch operation {
	case "history.read":
		return "查看" + name + "历史"
	case "lifecycle.pair":
		return "维护" + name + "启停记录"
	}
	part, action, _ := strings.Cut(operation, ".")
	records := map[string]string{"first": "首条记录", "record": "记录", "deactivation": "停用记录", "reactivation": "启用记录"}
	if record := records[part]; record != "" {
		switch action {
		case "add":
			return "新增" + name + record
		case "correct":
			return "更正" + name + record
		case "delete":
			return "删除" + name + record
		case "redate":
			return "调整" + name + record + "日期"
		case "reclassify":
			return "更正" + name + record + "分类"
		}
	}
	return key
}

// A mandate is one complete ceiling. Authorization never forms a Cartesian product of different mandates.
type ManagementCeiling struct {
	access.ManagementLimit
	Actions, Read, Write []string
	TargetKind, TargetID string
}
type PermissionTerms struct {
	Owner, Purpose, Source string
	Actions, Read, Write   []string
	TargetKind, TargetID   string
	From, Until            *time.Time
}

func (c ManagementCeiling) Covers(p PermissionTerms) bool {
	if c.Owner != p.Owner || c.TargetKind != p.TargetKind || c.TargetID != p.TargetID || !domain.ContainsAll(c.Actions, p.Actions) || !domain.ContainsAll(c.Read, p.Read) || !domain.ContainsAll(c.Write, p.Write) {
		return false
	}
	if c.Until != nil && (p.Until == nil || p.Until.After(*c.Until)) {
		return false
	}
	if p.Purpose != "" && !slices.ContainsFunc(c.Actions, func(a string) bool { return strings.HasPrefix(a, p.Purpose+".") }) {
		return false
	}
	return ProxyWithin(c.MaxProxyDays, p.Source, p.From, p.Until)
}
func ProxyWithin(days int, source string, from, until *time.Time) bool {
	if source != "proxy" {
		return true
	}
	return from != nil && until != nil && from.Before(*until) && (days == 0 || until.Sub(*from) <= time.Duration(days)*24*time.Hour)
}
func ValidPublication(role, scope, fields access.Definition) bool {
	if len(role.Actions) == 0 {
		return false
	}
	catalog := ManagementCatalog()
	for _, a := range role.Actions {
		if !slices.ContainsFunc(catalog, func(c access.CatalogAction) bool { return c.Key == a }) {
			return false
		}
		switch {
		case strings.HasPrefix(a, "staffingquota."):
			if scope.TargetKind != "position" || !domain.ContainsAll(QuotaFields, fields.Read) || !domain.ContainsAll(QuotaWrite, fields.Write) {
				return false
			}
		case a == "bu.reference.read":
			if !slices.Contains([]string{"enterprise", "bu"}, scope.TargetKind) || !domain.ContainsAll([]string{"code", "name"}, fields.Read) || len(fields.Write) > 0 {
				return false
			}
		default:
			if !slices.Contains([]string{"enterprise", "bu", "department"}, scope.TargetKind) || !domain.ContainsAll(DepartmentFields, fields.Read) || !domain.ContainsAll(DepartmentWrite, fields.Write) {
				return false
			}
		}
	}
	return true
}
