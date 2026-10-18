package mailer

import (
	"context"
	"errors"
	"fmt"
	"net/smtp"
	"strings"
)

// ErrNotConfigured 表示没有配置 SMTP。
var ErrNotConfigured = errors.New("SMTP 未配置")

// Mailer 负责发送纯文本邮件。
type Mailer struct{}

// New 创建 Mailer。
func New() *Mailer {
	return &Mailer{}
}

// Options 是一组 SMTP 连接参数。
type Options struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// Send 使用指定 SMTP 参数发送一封纯文本邮件。
func (m *Mailer) Send(ctx context.Context, opts Options, to, subject, body string) error {
	opts.Host = strings.TrimSpace(opts.Host)
	if opts.Host == "" {
		return ErrNotConfigured
	}
	if opts.Port == 0 {
		opts.Port = 587
	}
	if strings.TrimSpace(opts.From) == "" {
		opts.From = "Yu <noreply@localhost>"
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", opts.Host, opts.Port)

	var auth smtp.Auth
	if opts.Username != "" {
		auth = smtp.PlainAuth("", opts.Username, opts.Password, opts.Host)
	}

	message := strings.Join([]string{
		"From: " + opts.From,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	return smtp.SendMail(addr, auth, envelopeFrom(opts.From), []string{to}, []byte(message))
}

func envelopeFrom(from string) string {
	if start := strings.Index(from, "<"); start >= 0 {
		if end := strings.Index(from[start:], ">"); end >= 0 {
			return from[start+1 : start+end]
		}
	}
	return strings.TrimSpace(from)
}
