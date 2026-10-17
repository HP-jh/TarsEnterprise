package mtls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

// Pair 证书密钥对
type Pair struct {
	CertPEM []byte
	KeyPEM  []byte
}

// GenerateSelfSigned 生成自签名证书（开发/测试用）
func GenerateSelfSigned(commonName string) (*Pair, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"TarsEnterprise"},
			CommonName:   commonName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return &Pair{CertPEM: certPEM, KeyPEM: keyPEM}, nil
}

// LoadTLSConfig 加载 mTLS 配置（双向认证）
func LoadTLSConfig(certPEM, keyPEM, caPEM []byte) (*tls.Config, error) {
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("load key pair: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("parse CA cert")
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// Dial 建立 mTLS 连接
func Dial(addr string, cfg *tls.Config) (*tls.Conn, error) {
	conn, err := tls.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, fmt.Errorf("mtls dial %s: %w", addr, err)
	}
	return conn, nil
}

// Frame mTLS 消息帧格式（TSG ↔ TE 互通）
type Frame struct {
	Magic     [4]byte  // "TE\x00\x01"
	Version   uint16   // 1
	Type      uint8    // 1=msg, 2=cmd, 3=ack
	RoleToken [32]byte // 角色令牌哈希
	Length    uint32   // payload 长度
	Payload   []byte
}

func NewFrame(msgType uint8, roleToken string, payload []byte) *Frame {
	f := &Frame{
		Magic:   [4]byte{'T', 'E', 0x00, 0x01},
		Version: 1,
		Type:    msgType,
		Length:  uint32(len(payload)),
		Payload: payload,
	}
	copy(f.RoleToken[:], roleToken)
	return f
}

func (f *Frame) Encode() []byte {
	buf := make([]byte, 4+2+1+32+4+len(f.Payload))
	copy(buf[0:4], f.Magic[:])
	buf[4] = byte(f.Version >> 8)
	buf[5] = byte(f.Version)
	buf[6] = f.Type
	copy(buf[7:39], f.RoleToken[:])
	buf[39] = byte(f.Length >> 24)
	buf[40] = byte(f.Length >> 16)
	buf[41] = byte(f.Length >> 8)
	buf[42] = byte(f.Length)
	copy(buf[43:], f.Payload)
	return buf
}

func DecodeFrame(data []byte) (*Frame, error) {
	if len(data) < 43 {
		return nil, fmt.Errorf("frame too short")
	}
	if data[0] != 'T' || data[1] != 'E' {
		return nil, fmt.Errorf("invalid magic")
	}
	length := uint32(data[39])<<24 | uint32(data[40])<<16 | uint32(data[41])<<8 | uint32(data[42])
	if len(data) < 43+int(length) {
		return nil, fmt.Errorf("incomplete frame")
	}
	f := &Frame{
		Version: uint16(data[4])<<8 | uint16(data[5]),
		Type:    data[6],
		Length:  length,
		Payload: make([]byte, length),
	}
	copy(f.Magic[:], data[0:4])
	copy(f.RoleToken[:], data[7:39])
	copy(f.Payload, data[43:43+length])
	return f, nil
}
