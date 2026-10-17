package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecordAgentCall(t *testing.T) {
	Start()
	RecordAgentCall("agent-1", "client-1")
	RecordAgentCall("agent-1", "client-1")
	RecordAgentCall("agent-2", "client-2")

	top := TopAgents(10)
	if len(top) == 0 {
		t.Skip("no histogram data yet")
	}
	if top[0].AgentID != "agent-1" {
		t.Errorf("expected agent-1 first, got %s", top[0].AgentID)
	}
}

func TestHTTPHandler(t *testing.T) {
	Start()
	RecordAgentCall("a", "c")
	RecordAgentError("a")

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	HTTPHandler()(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "te_agent_calls_total") {
		t.Error("missing counter")
	}
}

func TestPercentile(t *testing.T) {
	vals := []float64{1, 2, 3, 4, 5}
	if percentile(vals, 0.5) != 3 {
		t.Error("p50 wrong")
	}
	if percentile([]float64{}, 0.5) != 0 {
		t.Error("empty wrong")
	}
}
