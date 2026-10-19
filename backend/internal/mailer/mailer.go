package mailer

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/smtp"
	"strconv"
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
	// TLS 为 true 时使用隐式 TLS（SMTPS，通常端口 465）；
	// 为 false 时使用明文连接，并在服务器支持时自动升级 STARTTLS。
	TLS bool
}

// Send 使用指定 SMTP 参数发送一封纯文本邮件。
func (m *Mailer) Send(ctx context.Context, opts Options, to, subject, body string) error {
	opts.Host = strings.TrimSpace(opts.Host)
	if opts.Host == "" {
		return ErrNotConfigured
	}
	if opts.Port == 0 {
		if opts.TLS {
			opts.Port = 465
		} else {
			opts.Port = 587
		}
	}
	if strings.TrimSpace(opts.From) == "" {
		opts.From = "Yu <noreply@localhost>"
	}
	if err := ctx.Err(); err != nil {
		return err
	}

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

	return sendMail(ctx, opts, auth, envelopeFrom(opts.From), []string{to}, []byte(message))
}

func sendMail(ctx context.Context, opts Options, auth smtp.Auth, from string, to []string, message []byte) error {
	addr := net.JoinHostPort(opts.Host, strconv.Itoa(opts.Port))
	dialer := &net.Dialer{}

	var (
		conn net.Conn
		err  error
	)
	if opts.TLS {
		tlsDialer := &tls.Dialer{
			NetDialer: dialer,
			Config:    &tls.Config{ServerName: opts.Host},
		}
		conn, err = tlsDialer.DialContext(ctx, "tcp", addr)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return err
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	client, err := smtp.NewClient(conn, opts.Host)
	if err != nil {
		return err
	}
	defer client.Close()

	// 隐式 TLS 已完成加密；否则在服务器支持时自动升级到 STARTTLS。
	if !opts.TLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: opts.Host}); err != nil {
				return err
			}
		}
	}

	if auth != nil {
		if ok, _ := client.Extension("AUTH"); !ok {
			return errors.New("SMTP 服务器不支持认证")
		}
		if err := client.Auth(auth); err != nil {
			return err
		}
	}

	if err := client.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return err
		}
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	return client.Quit()
}

func envelopeFrom(from string) string {
	if start := strings.Index(from, "<"); start >= 0 {
		if end := strings.Index(from[start:], ">"); end >= 0 {
			return from[start+1 : start+end]
		}
	}
	return strings.TrimSpace(from)
}
