package mailer

import (
	"context"
	"errors"
	"fmt"
	"net/smtp"
	"strings"

	"remoon.net/yu/backend/internal/config"
)

// ErrNotConfigured 表示没有配置 SMTP。
var ErrNotConfigured = errors.New("SMTP 未配置")

// Mailer 负责发送纯文本邮件。
type Mailer struct {
	cfg config.Config
}

// New 创建 Mailer。
func New(cfg config.Config) *Mailer {
	return &Mailer{cfg: cfg}
}

// Enabled 表示 SMTP 是否可用。
func (m *Mailer) Enabled() bool {
	return strings.TrimSpace(m.cfg.SMTPHost) != ""
}

// Send 发送一封纯文本邮件。
func (m *Mailer) Send(ctx context.Context, to, subject, body string) error {
	if !m.Enabled() {
		return ErrNotConfigured
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", m.cfg.SMTPHost, m.cfg.SMTPPort)

	var auth smtp.Auth
	if m.cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", m.cfg.SMTPUser, m.cfg.SMTPPass, m.cfg.SMTPHost)
	}

	message := strings.Join([]string{
		"From: " + m.cfg.SMTPFrom,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	return smtp.SendMail(addr, auth, envelopeFrom(m.cfg.SMTPFrom), []string{to}, []byte(message))
}

func envelopeFrom(from string) string {
	if start := strings.Index(from, "<"); start >= 0 {
		if end := strings.Index(from[start:], ">"); end >= 0 {
			return from[start+1 : start+end]
		}
	}
	return strings.TrimSpace(from)
}
