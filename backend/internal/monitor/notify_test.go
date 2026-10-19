package monitor

import (
	"encoding/json"
	"testing"
)

func TestSMTPOptionsIncludesTLS(t *testing.T) {
	opts, err := SMTPOptions(json.RawMessage(`{
		"host": "smtp.example.com",
		"port": 465,
		"username": "yu@example.com",
		"password": "secret",
		"to": ["ops@example.com"],
		"tls": true
	}`))
	if err != nil {
		t.Fatalf("SMTPOptions 返回错误: %v", err)
	}

	if !opts.TLS {
		t.Fatal("TLS = false, want true")
	}
	if opts.Port != 465 {
		t.Fatalf("Port = %d, want 465", opts.Port)
	}
	if opts.Username != "yu@example.com" || opts.Password != "secret" {
		t.Fatalf("认证信息未正确解析: %+v", opts)
	}
}

func TestValidateChannelConfigAllowsTLS(t *testing.T) {
	raw := json.RawMessage(`{
		"host": "smtp.example.com",
		"port": 465,
		"from": "Yu <yu@example.com>",
		"to": ["ops@example.com"],
		"tls": true
	}`)

	if err := ValidateChannelConfig("smtp", raw); err != nil {
		t.Fatalf("ValidateChannelConfig 返回错误: %v", err)
	}
}
