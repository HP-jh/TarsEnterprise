package wizard

import (
	"testing"

	"github.com/HP-jh/TarsEnterprise/pkg/company"
)

func TestSteps(t *testing.T) {
	steps := Steps()
	if len(steps) != 6 {
		t.Errorf("expected 6 steps, got %d", len(steps))
	}
	for i, s := range steps {
		if s.ID == "" {
			t.Errorf("step %d missing id", i)
		}
		if len(s.Fields) == 0 {
			t.Errorf("step %d no fields", i)
		}
	}
}

func TestRun(t *testing.T) {
	cfg := Run()
	if cfg.FirstRun {
		t.Error("wizard should set first_run=false")
	}
	if cfg.CompanyType == "" {
		t.Error("company type empty")
	}
}

func TestApplyAnswers(t *testing.T) {
	cfg := Run()
	answers := map[string]string{
		"company_type": string(company.Tech),
		"sandbox":      "seccomp",
		"tsg_bridge":   "false",
	}
	if err := ApplyAnswers(cfg, answers); err != nil {
		t.Fatal(err)
	}
	if cfg.CompanyType != string(company.Tech) {
		t.Error("company type not applied")
	}
	if cfg.Sandbox.Runtime != "seccomp" {
		t.Error("sandbox not applied")
	}
	if cfg.TSGBridge.Enabled {
		t.Error("tsg bridge should be disabled")
	}
}
