package config

import "testing"

func TestConfig_Validate(t *testing.T) {
	// Valid config
	cfg := &Config{
		FirstRun:    false,
		CompanyType: "tech",
		AdminToken:  "secret",
		MTLS:        MTLSConfig{Enabled: false},
		TSGBridge:   TSGBridgeConfig{Enabled: false},
		Metrics:     MetricsConfig{Enabled: true, Port: 8888},
	}
	errs := cfg.Validate()
	if len(errs) != 0 {
		t.Errorf("valid config got %d errors: %+v", len(errs), errs)
	}

	// Missing admin token on non-first-run
	cfg2 := &Config{
		FirstRun:    false,
		CompanyType: "tech",
		AdminToken:  "",
		MTLS:        MTLSConfig{Enabled: false},
		TSGBridge:   TSGBridgeConfig{Enabled: false},
		Metrics:     MetricsConfig{Enabled: true, Port: 8888},
	}
	errs2 := cfg2.Validate()
	found := false
	for _, e := range errs2 {
		if e.Field == "admin_token" {
			found = true
			if e.FriendlyMsg == "" || e.Suggestion == "" {
				t.Error("admin_token error missing friendly msg or suggestion")
			}
		}
	}
	if !found {
		t.Error("expected admin_token validation error")
	}

	// mTLS custom_ca missing files
	cfg3 := &Config{
		FirstRun:    true,
		CompanyType: "tech",
		MTLS: MTLSConfig{
			Enabled:  true,
			Mode:     "custom_ca",
			CertFile: "",
			KeyFile:  "",
			CAFile:   "",
		},
	}
	errs3 := cfg3.Validate()
	foundMTLS := false
	for _, e := range errs3 {
		if e.Field == "mtls.cert_file" {
			foundMTLS = true
		}
	}
	if !foundMTLS {
		t.Error("expected mtls.cert_file validation error")
	}

	// Empty company type
	cfg4 := &Config{
		FirstRun:    true,
		CompanyType: "",
		MTLS:        MTLSConfig{Enabled: false},
	}
	errs4 := cfg4.Validate()
	foundCompany := false
	for _, e := range errs4 {
		if e.Field == "company_type" {
			foundCompany = true
		}
	}
	if !foundCompany {
		t.Error("expected company_type validation error")
	}

	// Bad metrics port
	cfg5 := &Config{
		FirstRun:    true,
		CompanyType: "tech",
		Metrics:     MetricsConfig{Enabled: true, Port: -1},
	}
	errs5 := cfg5.Validate()
	foundPort := false
	for _, e := range errs5 {
		if e.Field == "metrics.port" {
			foundPort = true
		}
	}
	if !foundPort {
		t.Error("expected metrics.port validation error")
	}
}
