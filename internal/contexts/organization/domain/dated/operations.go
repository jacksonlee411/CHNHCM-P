package dated

type ObjectKind string

const (
	BusinessUnit ObjectKind = "BUSINESS_UNIT"
	Department   ObjectKind = "DEPARTMENT"
)

// OperationReasons returns a fresh object-scoped view of the CHNHCM-117 directory.
// Stored snapshots and submission results never resolve labels through this view.
func OperationReasons(object ObjectKind, kind string) map[string]string {
	if object != BusinessUnit && object != Department {
		return map[string]string{}
	}
	switch kind {
	case "CREATE":
		return map[string]string{"NEW_ORGANIZATION": "新设组织"}
	case "CHANGE":
		reasons := map[string]string{"NAME_ADJUSTMENT": "名称调整", "OTHER_DATA_ADJUSTMENT": "其他数据调整"}
		if object == Department {
			reasons["ORGANIZATION_REASSIGNMENT"] = "变更归属组织"
		}
		return reasons
	case "DEACTIVATE":
		return map[string]string{"ORG_RESTRUCTURING": "机构撤并", "BUSINESS_CLOSURE": "业务终止"}
	case "REACTIVATE":
		return map[string]string{"BUSINESS_RESUMPTION": "业务恢复"}
	default:
		return map[string]string{}
	}
}
func OperationLabel(kind string) string {
	return map[string]string{"CREATE": "新建", "CHANGE": "数据更改", "DEACTIVATE": "停用", "REACTIVATE": "重新启用"}[kind]
}
