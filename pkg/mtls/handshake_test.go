package mtls

import (
	"bytes"
	"testing"
)

func TestGenerateSelfSigned(t *testing.T) {
	pair, err := GenerateSelfSigned("test.te.local")
	if err != nil {
		t.Fatal(err)
	}
	if len(pair.CertPEM) == 0 {
		t.Error("cert empty")
	}
	if len(pair.KeyPEM) == 0 {
		t.Error("key empty")
	}
}

func TestFrame_EncodeDecode(t *testing.T) {
	payload := []byte("hello tsg bridge")
	f := NewFrame(1, "role-token-1234567890abcdef", payload)
	enc := f.Encode()
	dec, err := DecodeFrame(enc)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Type != f.Type {
		t.Error("type mismatch")
	}
	if !bytes.Equal(dec.Payload, payload) {
		t.Error("payload mismatch")
	}
}

func TestDecodeFrame_Short(t *testing.T) {
	_, err := DecodeFrame([]byte{1, 2, 3})
	if err == nil {
		t.Error("expected error for short frame")
	}
}

func TestDecodeFrame_BadMagic(t *testing.T) {
	data := make([]byte, 50)
	data[0] = 'X'
	_, err := DecodeFrame(data)
	if err == nil {
		t.Error("expected error for bad magic")
	}
}
