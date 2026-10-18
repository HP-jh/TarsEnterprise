package mtls

import "errors"

// 小白化错误定义：技术黑话 → 人话提示
var (
	// ErrServerCertExpired 服务端证书过期
	ErrServerCertExpired = errors.New("服务器安全证书已过期，请联系管理员更新证书后再试")

	// ErrClientCertRequired 客户端未提供证书
	ErrClientCertRequired = errors.New("连接需要客户端证书，请确保已正确配置并上传您的身份证书")

	// ErrClientCertExpired 客户端证书过期
	ErrClientCertExpired = errors.New("您的身份证书已过期，请联系管理员重新签发证书")

	// ErrClientCertNotYetValid 客户端证书尚未生效
	ErrClientCertNotYetValid = errors.New("您的身份证书尚未生效，请确认系统时间是否正确，或联系管理员")

	// ErrClientCertInvalid 客户端证书无效（非过期类通用错误）
	ErrClientCertInvalid = errors.New("您的身份证书无法通过验证，请检查证书是否损坏或由正确的机构签发")
)

// FriendlyError 将 mTLS 错误转换为人话
func FriendlyError(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, ErrServerCertExpired):
		return "服务器安全证书已过期，请联系管理员更新证书后再试"
	case errors.Is(err, ErrClientCertRequired):
		return "连接需要客户端证书，请确保已正确配置并上传您的身份证书"
	case errors.Is(err, ErrClientCertExpired):
		return "您的身份证书已过期，请联系管理员重新签发证书"
	case errors.Is(err, ErrClientCertNotYetValid):
		return "您的身份证书尚未生效，请确认系统时间是否正确，或联系管理员"
	case errors.Is(err, ErrClientCertInvalid):
		return "您的身份证书无法通过验证，请检查证书是否损坏或由正确的机构签发"
	default:
		return "安全连接验证失败，请检查证书配置或联系管理员协助排查"
	}
}
