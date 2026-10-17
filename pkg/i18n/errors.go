package i18n

import (
	"fmt"
	"net/http"
)

// FriendlyError 将生硬的底层错误转换为人话提示
func FriendlyError(w http.ResponseWriter, status int, internal string) {
	msg := Map(internal)
	http.Error(w, msg, status)
}

// Map 错误映射表：技术黑话 → 人话
func Map(internal string) string {
	switch internal {
	case "POST required":
		return "这个接口只接受 POST 请求，请确认请求方式是否正确"
	case "GET required":
		return "这个接口只接受 GET 请求，请确认请求方式是否正确"
	case "tool not found":
		return "找不到指定的工具，请检查工具 ID 是否输入正确"
	case "unauthorized":
		return "身份验证失败，请确认管理员令牌是否正确"
	case "admin token not configured":
		return "管理员令牌尚未配置，无法执行敏感操作。请先设置 admin_token"
	case "user_id and role_id required":
		return "用户 ID 和角色 ID 不能为空，请补充完整后再提交"
	case "knowledge base not found":
		return "知识库不存在，请先导入文档或检查公司类型配置"
	case "parse config":
		return "配置文件解析出错，请检查 JSON 格式是否合法"
	}

	// 模糊匹配常见前缀
	if len(internal) > 20 {
		if contains(internal, "connection refused") {
			return "连接被拒绝，可能是目标服务未启动或网络不通"
		}
		if contains(internal, "timeout") {
			return "请求超时，请检查网络或稍后重试"
		}
		if contains(internal, "not found") || contains(internal, "no such") {
			return "找不到请求的资源，请确认路径或参数是否正确"
		}
		if contains(internal, "permission denied") {
			return "权限不足，请联系管理员开通对应权限"
		}
	}

	// 兜底：保留原始信息但加上友好前缀
	return fmt.Sprintf("操作遇到问题：%s。如果反复出现，请联系管理员或查看 troubleshooting 文档", internal)
}

// JSONError 写入 JSON 格式的友好错误
func JSONError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error_code":"%s","message":"%s"}`+"\n", code, msg)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOfSubstr(s, substr) >= 0
}

func indexOfSubstr(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
