package metrics

import "testing"

func TestCounter(t *testing.T) {
	ResetRequests()
	IncrRequest()
	IncrRequest()
	if GetRequests() != 2 {
		t.Fatalf("expected 2, got %d", GetRequests())
	}
}
