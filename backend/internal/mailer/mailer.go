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

// Options 是一组 SMTP 连接参数。
type Options struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// Options 返回全局配置中的 SMTP 参数。
func (m *Mailer) Options() Options {
	return Options{
		Host:     m.cfg.SMTPHost,
		Port:     m.cfg.SMTPPort,
		Username: m.cfg.SMTPUser,
		Password: m.cfg.SMTPPass,
		From:     m.cfg.SMTPFrom,
	}
}

// Enabled 表示 SMTP 是否可用。
func (m *Mailer) Enabled() bool {
	return strings.TrimSpace(m.cfg.SMTPHost) != ""
}

// Send 使用全局 SMTP 配置发送一封纯文本邮件。
func (m *Mailer) Send(ctx context.Context, to, subject, body string) error {
	return m.SendWith(ctx, m.Options(), to, subject, body)
}

// SendWith 使用指定 SMTP 参数发送一封纯文本邮件。
func (m *Mailer) SendWith(ctx context.Context, opts Options, to, subject, body string) error {
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
