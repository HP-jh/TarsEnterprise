package company

import "fmt"

// CompanyType 定义企业类型
type CompanyType string

const (
	Education CompanyType = "education"
	Tech      CompanyType = "tech"
	KPI       CompanyType = "kpi"
	Consulting CompanyType = "consulting"
	Finance   CompanyType = "finance"
	Gov       CompanyType = "gov"
)

var AllTypes = []CompanyType{Education, Tech, KPI, Consulting, Finance, Gov}

// Role 角色定义
type Role struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Level       int      `json:"level"`
	Permissions []string `json:"permissions"`
}

// Resource 资源定义
type Resource struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Sensitive bool  `json:"sensitive"`
}

// Action 动作定义
type Action string

const (
	ActionRead   Action = "read"
	ActionWrite  Action = "write"
	ActionDelete Action = "delete"
	ActionExec   Action = "exec"
	ActionAdmin  Action = "admin"
)

// CompanySchema 公司类型完整定义
type CompanySchema struct {
	Type      CompanyType `json:"type"`
	Name      string      `json:"name"`
	Roles     []Role      `json:"roles"`
	Resources []Resource  `json:"resources"`
	Actions   []Action    `json:"actions"`
}

func GetSchema(t CompanyType) (*CompanySchema, error) {
	switch t {
	case Education:
		return &CompanySchema{
			Type: Education, Name: "学校",
			Roles: []Role{
				{ID: "teacher", Name: "教师", Level: 2, Permissions: []string{"course:read", "course:write", "grade:read", "grade:write"}},
				{ID: "student", Name: "学生", Level: 1, Permissions: []string{"course:read", "grade:read"}},
				{ID: "parent", Name: "家长", Level: 1, Permissions: []string{"grade:read", "homework:read"}},
				{ID: "principal", Name: "校长", Level: 3, Permissions: []string{"*"}},
			},
			Resources: []Resource{
				{ID: "course", Name: "课程", Sensitive: false},
				{ID: "grade", Name: "成绩", Sensitive: true},
				{ID: "homework", Name: "作业", Sensitive: false},
				{ID: "attendance", Name: "考勤", Sensitive: false},
			},
			Actions: []Action{ActionRead, ActionWrite, ActionDelete, ActionAdmin},
		}, nil
	case Tech:
		return &CompanySchema{
			Type: Tech, Name: "互联网公司",
			Roles: []Role{
				{ID: "pm", Name: "产品经理", Level: 2, Permissions: []string{"project:read", "project:write", "incident:read"}},
				{ID: "dev", Name: "研发工程师", Level: 2, Permissions: []string{"project:read", "code:write", "incident:read"}},
				{ID: "ops", Name: "运营", Level: 1, Permissions: []string{"project:read", "data:read"}},
				{ID: "hr", Name: "HR", Level: 2, Permissions: []string{"people:read", "people:write"}},
				{ID: "cto", Name: "CTO", Level: 3, Permissions: []string{"*"}},
			},
			Resources: []Resource{
				{ID: "project", Name: "项目", Sensitive: false},
				{ID: "code", Name: "代码", Sensitive: true},
				{ID: "incident", Name: "事故", Sensitive: true},
				{ID: "people", Name: "人员", Sensitive: true},
			},
			Actions: []Action{ActionRead, ActionWrite, ActionDelete, ActionExec, ActionAdmin},
		}, nil
	case KPI:
		return &CompanySchema{
			Type: KPI, Name: "传统KPI考核公司",
			Roles: []Role{
				{ID: "employee", Name: "员工", Level: 1, Permissions: []string{"report:write:self", "report:read:self", "kpi:read:self"}},
				{ID: "lead", Name: "小组长", Level: 2, Permissions: []string{"report:read:team", "kpi:write:team", "report:write:self"}},
				{ID: "manager", Name: "部门经理", Level: 3, Permissions: []string{"report:read:dept", "kpi:write:dept", "kpi:read:all"}},
				{ID: "ceo", Name: "CEO", Level: 4, Permissions: []string{"*"}},
			},
			Resources: []Resource{
				{ID: "daily", Name: "日报", Sensitive: false},
				{ID: "weekly", Name: "周报", Sensitive: false},
				{ID: "monthly", Name: "月报", Sensitive: false},
				{ID: "kpi", Name: "KPI指标", Sensitive: true},
			},
			Actions: []Action{ActionRead, ActionWrite, ActionAdmin},
		}, nil
	case Consulting:
		return &CompanySchema{
			Type: Consulting, Name: "咨询公司",
			Roles: []Role{
				{ID: "consultant", Name: "顾问", Level: 2, Permissions: []string{"project:read", "project:write", "client:read"}},
				{ID: "pm", Name: "项目经理", Level: 3, Permissions: []string{"project:read", "project:write", "billing:read", "client:read"}},
				{ID: "partner", Name: "合伙人", Level: 4, Permissions: []string{"*"}},
			},
			Resources: []Resource{
				{ID: "project", Name: "项目", Sensitive: true},
				{ID: "client", Name: "客户", Sensitive: true},
				{ID: "billing", Name: "计费", Sensitive: true},
				{ID: "confidential", Name: "保密文档", Sensitive: true},
			},
			Actions: []Action{ActionRead, ActionWrite, ActionAdmin},
		}, nil
	case Finance:
		return &CompanySchema{
			Type: Finance, Name: "金融机构",
			Roles: []Role{
				{ID: "risk", Name: "风控", Level: 3, Permissions: []string{"trade:read", "risk:write", "report:read"}},
				{ID: "compliance", Name: "合规", Level: 3, Permissions: []string{"trade:read", "audit:read", "policy:write"}},
				{ID: "audit", Name: "审计", Level: 3, Permissions: []string{"trade:read", "audit:read", "audit:write"}},
				{ID: "trader", Name: "交易员", Level: 2, Permissions: []string{"trade:write:self", "trade:read:self"}},
				{ID: "cfo", Name: "CFO", Level: 4, Permissions: []string{"*"}},
			},
			Resources: []Resource{
				{ID: "trade", Name: "交易", Sensitive: true},
				{ID: "account", Name: "账户", Sensitive: true},
				{ID: "report", Name: "报告", Sensitive: true},
				{ID: "risk_model", Name: "风控模型", Sensitive: true},
			},
			Actions: []Action{ActionRead, ActionWrite, ActionExec, ActionAdmin},
		}, nil
	case Gov:
		return &CompanySchema{
			Type: Gov, Name: "政企机构",
			Roles: []Role{
				{ID: "clerk", Name: "科员", Level: 1, Permissions: []string{"doc:read", "doc:write:self", "flow:read"}},
				{ID: "director", Name: "处长", Level: 3, Permissions: []string{"doc:read:dept", "doc:write:dept", "flow:write", "approve:write"}},
				{ID: "approver", Name: "审批人", Level: 3, Permissions: []string{"doc:read", "approve:write", "flow:write"}},
				{ID: "chief", Name: "局长", Level: 4, Permissions: []string{"*"}},
			},
			Resources: []Resource{
				{ID: "doc", Name: "公文", Sensitive: true},
				{ID: "flow", Name: "流程", Sensitive: false},
				{ID: "permission", Name: "权限", Sensitive: true},
				{ID: "archive", Name: "档案", Sensitive: true},
			},
			Actions: []Action{ActionRead, ActionWrite, ActionExec, ActionAdmin},
		}, nil
	default:
		return nil, fmt.Errorf("unknown company type: %s", t)
	}
}

// ValidatePermission 验证角色是否有某资源的某动作权限
func ValidatePermission(schema *CompanySchema, roleID string, resourceID string, action Action) bool {
	for _, r := range schema.Roles {
		if r.ID != roleID {
			continue
		}
		for _, p := range r.Permissions {
			if p == "*" {
				return true
			}
			// 简单匹配: resource:action 或 resource:action:scope
			if p == fmt.Sprintf("%s:%s", resourceID, action) {
				return true
			}
			if p == fmt.Sprintf("%s:%s:self", resourceID, action) && action == ActionRead {
				return true
			}
		}
	}
	return false
}
