package health

import (
	"fmt"
	"time"
)

// Status represents the health status of a subsystem.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
	StatusUnknown   Status = "unknown"
)

// Check represents a single subsystem health check.
type Check struct {
	Name        string                 `json:"name"`
	Status      Status                 `json:"status"`
	Message     string                 `json:"message"`
	FriendlyMsg string                 `json:"friendly_msg"`
	Details     map[string]interface{} `json:"details,omitempty"`
	CheckedAt   time.Time              `json:"checked_at"`
}

// Report aggregates health checks from all subsystems.
type Report struct {
	Overall Status  `json:"overall"`
	Checks  []Check `json:"checks"`
	Version string  `json:"version"`
	Time    int64   `json:"time"`
}

// Checker is a function that returns a health check for a subsystem.
type Checker func() Check

// Registry holds all registered health checkers.
type Registry struct {
	checkers []Checker
}

// NewRegistry creates a new health check registry.
func NewRegistry() *Registry {
	return &Registry{checkers: make([]Checker, 0)}
}

// Register adds a health checker to the registry.
func (r *Registry) Register(c Checker) {
	r.checkers = append(r.checkers, c)
}

// Run executes all registered checks and returns a report.
func (r *Registry) Run(version string) Report {
	checks := make([]Check, 0, len(r.checkers))
	overall := StatusHealthy
	now := time.Now()

	for _, checker := range r.checkers {
		check := checker()
		if check.CheckedAt.IsZero() {
			check.CheckedAt = now
		}
		checks = append(checks, check)
		if check.Status == StatusUnhealthy {
			overall = StatusUnhealthy
		} else if check.Status == StatusDegraded && overall == StatusHealthy {
			overall = StatusDegraded
		}
	}

	return Report{
		Overall: overall,
		Checks:  checks,
		Version: version,
		Time:    now.Unix(),
	}
}

// SimpleCheck creates a basic check with a given status and message.
func SimpleCheck(name string, status Status, message string) Check {
	return Check{
		Name:        name,
		Status:      status,
		Message:     message,
		FriendlyMsg: message,
		CheckedAt:   time.Now(),
	}
}

// Healthy returns a healthy check.
func Healthy(name string, details ...string) Check {
	msg := fmt.Sprintf("%s 运行正常", name)
	if len(details) > 0 {
		msg = details[0]
	}
	return SimpleCheck(name, StatusHealthy, msg)
}

// Unhealthy returns an unhealthy check with a friendly message.
func Unhealthy(name string, err error) Check {
	return Check{
		Name:        name,
		Status:      StatusUnhealthy,
		Message:     err.Error(),
		FriendlyMsg: fmt.Sprintf("%s 出现了一些问题：%s", name, err.Error()),
		CheckedAt:   time.Now(),
	}
}

// Unknown returns an unknown status check.
func Unknown(name string, reason string) Check {
	return Check{
		Name:        name,
		Status:      StatusUnknown,
		Message:     reason,
		FriendlyMsg: fmt.Sprintf("%s 状态未知：%s", name, reason),
		CheckedAt:   time.Now(),
	}
}
