package health

import (
	"errors"
	"testing"
)

func TestRegistry_Run(t *testing.T) {
	r := NewRegistry()
	r.Register(func() Check {
		return Healthy("mTLS", "双向认证正常工作")
	})
	r.Register(func() Check {
		return Unhealthy("NATS", errors.New("connection refused"))
	})

	report := r.Run("v4.33.0")

	if report.Version != "v4.33.0" {
		t.Errorf("version = %s, want v4.33.0", report.Version)
	}
	if report.Overall != StatusUnhealthy {
		t.Errorf("overall = %s, want unhealthy", report.Overall)
	}
	if len(report.Checks) != 2 {
		t.Errorf("checks = %d, want 2", len(report.Checks))
	}

	// Verify individual checks
	found := map[string]bool{}
	for _, c := range report.Checks {
		found[c.Name] = true
		if c.Name == "mTLS" && c.Status != StatusHealthy {
			t.Errorf("mTLS status = %s, want healthy", c.Status)
		}
		if c.Name == "NATS" {
			if c.Status != StatusUnhealthy {
				t.Errorf("NATS status = %s, want unhealthy", c.Status)
			}
			if c.FriendlyMsg == "" {
				t.Error("NATS friendly_msg empty")
			}
		}
	}
	if !found["mTLS"] || !found["NATS"] {
		t.Error("missing expected checks")
	}
}

func TestHealthy(t *testing.T) {
	c := Healthy("Test", "custom message")
	if c.Status != StatusHealthy {
		t.Errorf("status = %s", c.Status)
	}
	if c.Message != "custom message" {
		t.Errorf("message = %s", c.Message)
	}
}

func TestUnhealthy(t *testing.T) {
	c := Unhealthy("Test", errors.New("boom"))
	if c.Status != StatusUnhealthy {
		t.Errorf("status = %s", c.Status)
	}
	if c.FriendlyMsg == "" {
		t.Error("friendly_msg empty")
	}
}

func TestSimpleCheck(t *testing.T) {
	c := SimpleCheck("X", StatusDegraded, "slow")
	if c.Name != "X" || c.Status != StatusDegraded || c.Message != "slow" {
		t.Error("SimpleCheck mismatch")
	}
}
