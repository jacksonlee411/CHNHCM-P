package hireprep

// RegisterQuery separates the effective period from the original registration time.
// Recorded date bounds are inclusive civil dates in Asia/Shanghai.
type RegisterQuery struct {
	Ref, Kind, Status, BusinessFrom, BusinessTo, RecordedFrom, RecordedTo string
	Department, FromDepartment, ToDepartment, Operator, Sort              string
	Page, PageSize                                                        int
}

type RegisterRow struct {
	ID             string `json:"id"`
	Revision       string `json:"revision"`
	Source         string `json:"sourceVersion"`
	Kind           string `json:"kind"`
	Status         string `json:"status"`
	BusinessDate   string `json:"businessDate"`
	RecordedAt     string `json:"recordedAt"`
	Employment     string `json:"employment"`
	Assignment     string `json:"assignment"`
	RecordNumber   int    `json:"recordNumber"`
	Person         string `json:"person"`
	Name           string `json:"name"`
	Number         string `json:"number"`
	Department     string `json:"department,omitempty"`
	DepartmentName string `json:"departmentName,omitempty"`
	Operator       string `json:"operator,omitempty"`
	CanHistory     bool   `json:"canHistory"`
}
type RegisterOperator struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type RegisterResult struct {
	Rows            []RegisterRow        `json:"rows"`
	Departments     []EmployeeDepartment `json:"departments"`
	FromDepartments []EmployeeDepartment `json:"fromDepartments"`
	ToDepartments   []EmployeeDepartment `json:"toDepartments"`
	Operators       []RegisterOperator   `json:"operators"`
	Total           int                  `json:"total"`
	Keyword         string               `json:"keyword"`
}
