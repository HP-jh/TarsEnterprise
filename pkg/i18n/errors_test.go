package i18n

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMap_KnownErrors(t *testing.T) {
	cases := map[string]string{
		"POST required":               "这个接口只接受 POST 请求",
		"tool not found":              "找不到指定的工具",
		"unauthorized":                "身份验证失败",
		"admin token not configured":  "管理员令牌尚未配置",
		"knowledge base not found":    "知识库不存在",
		"user_id and role_id required": "用户 ID 和角色 ID 不能为空",
	}
	for input, want := range cases {
		got := Map(input)
		if !strings.Contains(got, want) {
			t.Errorf("Map(%q) = %q, want containing %q", input, got, want)
		}
	}
}

func TestMap_FuzzyMatch(t *testing.T) {
	if !strings.Contains(Map("connection refused to db"), "连接被拒绝") {
		t.Error("expected fuzzy match for connection refused")
	}
	if !strings.Contains(Map("the remote server returned a timeout error"), "超时") {
		t.Error("expected fuzzy match for timeout")
	}
}

func TestFriendlyError_WritesResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	FriendlyError(rec, http.StatusBadRequest, "tool not found")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "找不到指定的工具") {
		t.Fatalf("expected friendly message, got %q", body)
	}
}

func TestJSONError(t *testing.T) {
	rec := httptest.NewRecorder()
	JSONError(rec, http.StatusForbidden, "FORBIDDEN", "禁止访问")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "FORBIDDEN") || !strings.Contains(body, "禁止访问") {
		t.Fatalf("expected JSON error, got %q", body)
	}
}
