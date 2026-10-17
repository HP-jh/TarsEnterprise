package role

import (
	"fmt"
	"strings"

	"github.com/HP-jh/TarsEnterprise/pkg/company"
)

// Identity 企业中的身份
type Identity struct {
	UserID      string            `json:"user_id"`
	RoleID      string            `json:"role_id"`
	CompanyType company.CompanyType `json:"company_type"`
	Department  string            `json:"department"`
	Attributes  map[string]string `json:"attributes"`
}

// RoleMatrix 角色-资源-动作三轴权限矩阵
type RoleMatrix struct {
	RoleID     string         `json:"role_id"`
	ResourceID string         `json:"resource_id"`
	Actions    []company.Action `json:"actions"`
}

// Manager 角色管理器
type Manager struct {
	identities map[string]*Identity
	matrix     map[string][]RoleMatrix // key = companyType:roleID
}

func NewManager() *Manager {
	return &Manager{
		identities: make(map[string]*Identity),
		matrix:     make(map[string][]RoleMatrix),
	}
}

func (m *Manager) RegisterIdentity(id *Identity) error {
	if id.UserID == "" || id.RoleID == "" {
		return fmt.Errorf("user_id and role_id required")
	}
	m.identities[id.UserID] = id
	return nil
}

func (m *Manager) GetIdentity(userID string) (*Identity, bool) {
	id, ok := m.identities[userID]
	return id, ok
}

func (m *Manager) BuildMatrix(ctype company.CompanyType) error {
	schema, err := company.GetSchema(ctype)
	if err != nil {
		return err
	}
	key := string(ctype)
	var mats []RoleMatrix
	for _, r := range schema.Roles {
		resActions := make(map[string][]company.Action)
		for _, perm := range r.Permissions {
			if perm == "*" {
				for _, res := range schema.Resources {
					resActions[res.ID] = schema.Actions
				}
				break
			}
			parts := strings.SplitN(perm, ":", 3)
			if len(parts) < 2 {
				continue
			}
			act := company.Action(parts[1])
			resActions[parts[0]] = append(resActions[parts[0]], act)
		}
		for resID, acts := range resActions {
			mats = append(mats, RoleMatrix{
				RoleID:     r.ID,
				ResourceID: resID,
				Actions:    uniqueActions(acts),
			})
		}
	}
	m.matrix[key] = mats
	return nil
}

func (m *Manager) Can(userID string, resourceID string, action company.Action) bool {
	id, ok := m.identities[userID]
	if !ok {
		return false
	}
	key := string(id.CompanyType)
	mats, ok := m.matrix[key]
	if !ok {
		return false
	}
	for _, mat := range mats {
		if mat.RoleID == id.RoleID && mat.ResourceID == resourceID {
			for _, a := range mat.Actions {
				if a == action {
					return true
				}
			}
		}
	}
	return false
}

func uniqueActions(actions []company.Action) []company.Action {
	seen := make(map[company.Action]bool)
	var out []company.Action
	for _, a := range actions {
		if !seen[a] {
			seen[a] = true
			out = append(out, a)
		}
	}
	return out
}

// BindToProfession 将角色绑定到职业系统（KingdoM 界门纲目科属种）
func BindToProfession(roleID string) string {
	// 简化映射：role → 界门纲目科属种路径
	mapping := map[string]string{
		"teacher":    "Animalia/Chordata/Mammalia/Primates/Hominidae/Homo/sapiens-educator",
		"student":    "Animalia/Chordata/Mammalia/Primates/Hominidae/Homo/sapiens-learner",
		"dev":        "Animalia/Chordata/Mammalia/Primates/Hominidae/Homo/sapiens-engineer",
		"pm":         "Animalia/Chordata/Mammalia/Primates/Hominidae/Homo/sapiens-planner",
		"cto":        "Animalia/Chordata/Mammalia/Primates/Hominidae/Homo/sapiens-architect",
		"employee":   "Animalia/Chordata/Mammalia/Primates/Hominidae/Homo/sapiens-worker",
		"manager":    "Animalia/Chordata/Mammalia/Primates/Hominidae/Homo/sapiens-leader",
		"ceo":        "Animalia/Chordata/Mammalia/Primates/Hominidae/Homo/sapiens-executive",
		"consultant": "Animalia/Chordata/Mammalia/Primates/Hominidae/Homo/sapiens-advisor",
		"partner":    "Animalia/Chordata/Mammalia/Primates/Hominidae/Homo/sapiens-stakeholder",
		"risk":       "Animalia/Chordata/Mammalia/Primates/Hominidae/Homo/sapiens-guardian",
		"trader":     "Animalia/Chordata/Mammalia/Primates/Hominidae/Homo/sapiens-trader",
		"cfo":        "Animalia/Chordata/Mammalia/Primates/Hominidae/Homo/sapiens-financier",
		"clerk":      "Animalia/Chordata/Mammalia/Primates/Hominidae/Homo/sapiens-clerk",
		"director":   "Animalia/Chordata/Mammalia/Primates/Hominidae/Homo/sapiens-director",
		"chief":      "Animalia/Chordata/Mammalia/Primates/Hominidae/Homo/sapiens-chief",
	}
	if p, ok := mapping[roleID]; ok {
		return p
	}
	return "Animalia/Chordata/Mammalia/Primates/Hominidae/Homo/sapiens"
}
