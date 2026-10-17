package lang

import "testing"

func TestDetect(t *testing.T) {
	e := Detect()
	if !e.Go {
		t.Error("go should always be available")
	}
	if e.Versions["go"] == "" {
		t.Error("go version empty")
	}
}

func TestEnvInfo_Summary(t *testing.T) {
	e := &EnvInfo{Go: true, Versions: map[string]string{"go": "go1.23"}}
	summary := e.Summary()
	if len(summary) == 0 {
		t.Error("expected summary")
	}
}
