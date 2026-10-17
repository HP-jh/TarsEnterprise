package voicecorrection

import (
	"context"
	"testing"
)

func TestCorrector_Correct(t *testing.T) {
	c := NewCorrector("")
	result, err := c.Correct(context.Background(), "mcp server and openclaw gateway")
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != "local" {
		t.Error("expected local source")
	}
	if result.Corrected == "" {
		t.Error("empty corrected")
	}
}

func TestCorrector_AsMCPTool(t *testing.T) {
	c := NewCorrector("")
	tool := c.AsMCPTool()
	if tool["name"] != "voice_correction" {
		t.Error("wrong tool name")
	}
}

func TestCorrector_Enabled(t *testing.T) {
	c := NewCorrector("")
	if c.Enabled() {
		t.Error("empty url should not be enabled")
	}
	c.SetServiceURL("http://localhost:8080")
	if !c.Enabled() {
		t.Error("should be enabled after setting url")
	}
}
