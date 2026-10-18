package monitor

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"remoon.net/yu/backend/internal/mailer"
	"remoon.net/yu/backend/internal/store"
)

// SMTPChannelConfig 是邮件渠道配置。
type SMTPChannelConfig struct {
	Host     string   `json:"host"`
	Port     int      `json:"port"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	From     string   `json:"from"`
	To       []string `json:"to"`
}

// WebhookChannelConfig 是 Webhook 渠道配置。
type WebhookChannelConfig struct {
	URL     string            `json:"url"`
	Secret  string            `json:"secret"`
	Headers map[string]string `json:"headers"`
}

// ValidateChannelConfig 校验通知渠道配置。
func ValidateChannelConfig(channelType string, raw json.RawMessage) error {
	switch channelType {
	case "smtp":
		var cfg SMTPChannelConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return fmt.Errorf("邮件渠道配置格式不正确: %w", err)
		}
		if len(cfg.To) == 0 {
			return errors.New("收件人不能为空")
		}
		for _, address := range cfg.To {
			if _, err := mail.ParseAddress(strings.TrimSpace(address)); err != nil {
				return fmt.Errorf("收件人邮箱不合法: %s", address)
			}
		}
		if strings.TrimSpace(cfg.Host) == "" {
			return errors.New("SMTP 主机不能为空")
		}
		if cfg.Port != 0 && (cfg.Port < 1 || cfg.Port > 65535) {
			return errors.New("SMTP 端口不合法")
		}
		if strings.TrimSpace(cfg.From) != "" {
			if _, err := mail.ParseAddress(cfg.From); err != nil {
				return errors.New("发件人邮箱不合法")
			}
		}
		return nil

	case "webhook":
		var cfg WebhookChannelConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return fmt.Errorf("Webhook 配置格式不正确: %w", err)
		}
		parsed, err := url.Parse(strings.TrimSpace(cfg.URL))
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return errors.New("webhook url 必须是合法的 http(s) 地址")
		}
		return nil

	default:
		return fmt.Errorf("不支持的通知类型: %s", channelType)
	}
}

// AlertMessage 是通知内容。
type AlertMessage struct {
	Event         string
	WorkspaceName string
	MonitorName   string
	MonitorType   string
	Target        string
	Status        string
	LatencyMS     int
	Error         string
	CheckedAt     time.Time
	CertExpiresAt *time.Time
	DaysLeft      int
}

// Sender 负责把通知发送到各个渠道。
type Sender struct {
	mail   *mailer.Mailer
	client *http.Client
}

// NewSender 创建发送器。
func NewSender(m *mailer.Mailer) *Sender {
	return &Sender{
		mail:   m,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// SendAll 向所有启用的渠道发送通知，失败只记录日志。
func (s *Sender) SendAll(ctx context.Context, channels []store.NotificationChannel, msg AlertMessage) {
	for _, channel := range channels {
		if !channel.Enabled {
			continue
		}

		if err := s.Send(ctx, channel, msg); err != nil {
			slog.Warn("发送通知失败", "channel", channel.ID, "type", channel.Type, "err", err)
		}
	}
}

// Send 向单个渠道发送通知。
func (s *Sender) Send(ctx context.Context, channel store.NotificationChannel, msg AlertMessage) error {
	switch channel.Type {
	case "smtp":
		return s.sendSMTP(ctx, channel.Config, msg)
	case "webhook":
		return s.sendWebhook(ctx, channel.Config, msg)
	default:
		return fmt.Errorf("不支持的通知类型: %s", channel.Type)
	}
}

func (s *Sender) sendSMTP(ctx context.Context, raw json.RawMessage, msg AlertMessage) error {
	cfg, options, err := parseSMTPConfig(raw)
	if err != nil {
		return err
	}

	var failures []string
	for _, to := range cfg.To {
		if err := s.mail.Send(ctx, options, strings.TrimSpace(to), msg.subject(), msg.body()); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", to, err))
		}
	}

	if len(failures) > 0 {
		return errors.New(strings.Join(failures, "; "))
	}
	return nil
}

// SMTPOptions 解析邮件渠道配置为发送参数，供告警与邀请邮件共用。
func SMTPOptions(raw json.RawMessage) (mailer.Options, error) {
	_, options, err := parseSMTPConfig(raw)
	return options, err
}

func parseSMTPConfig(raw json.RawMessage) (SMTPChannelConfig, mailer.Options, error) {
	var cfg SMTPChannelConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, mailer.Options{}, err
	}

	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		return cfg, mailer.Options{}, errors.New("SMTP 主机未配置")
	}

	options := mailer.Options{
		Host:     host,
		Port:     cfg.Port,
		Username: strings.TrimSpace(cfg.Username),
		Password: cfg.Password,
		From:     strings.TrimSpace(cfg.From),
	}
	if options.From == "" {
		options.From = options.Username
	}

	return cfg, options, nil
}

func (s *Sender) sendWebhook(ctx context.Context, raw json.RawMessage, msg AlertMessage) error {
	var cfg WebhookChannelConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return err
	}

	payload := map[string]any{
		"event":           msg.Event,
		"workspace":       msg.WorkspaceName,
		"status":          msg.Status,
		"latency_ms":      msg.LatencyMS,
		"error":           msg.Error,
		"checked_at":      msg.CheckedAt,
		"cert_expires_at": msg.CertExpiresAt,
		"days_left":       msg.DaysLeft,
		"monitor": map[string]any{
			"name":   msg.MonitorName,
			"type":   msg.MonitorType,
			"target": msg.Target,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Yu-Monitor/1.0")
	for key, value := range cfg.Headers {
		req.Header.Set(key, value)
	}

	if cfg.Secret != "" {
		mac := hmac.New(sha256.New, []byte(cfg.Secret))
		mac.Write(body)
		req.Header.Set("X-Yu-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook 返回状态码 %d", resp.StatusCode)
	}

	return nil
}

func (m AlertMessage) subject() string {
	switch m.Event {
	case "monitor.down":
		return "[Yu] 服务不可用：" + m.MonitorName
	case "monitor.up":
		return "[Yu] 服务已恢复：" + m.MonitorName
	case "monitor.ssl_expiring":
		return "[Yu] 证书即将到期：" + m.MonitorName
	default:
		return "[Yu] 通知测试"
	}
}

func (m AlertMessage) body() string {
	var b strings.Builder

	switch m.Event {
	case "monitor.down":
		b.WriteString("监控项不可用。\n\n")
	case "monitor.up":
		b.WriteString("监控项已恢复。\n\n")
	case "monitor.ssl_expiring":
		b.WriteString("HTTPS 证书即将到期。\n\n")
	default:
		b.WriteString("这是一条测试通知。\n\n")
	}

	fmt.Fprintf(&b, "工作区：%s\n", m.WorkspaceName)
	fmt.Fprintf(&b, "监控项：%s（%s）\n", m.MonitorName, m.MonitorType)
	fmt.Fprintf(&b, "目标：%s\n", m.Target)

	if m.Status != "" {
		fmt.Fprintf(&b, "状态：%s\n", m.Status)
	}
	if m.LatencyMS > 0 {
		fmt.Fprintf(&b, "响应时间：%d ms\n", m.LatencyMS)
	}
	if m.Error != "" {
		fmt.Fprintf(&b, "错误：%s\n", m.Error)
	}
	if m.Event == "monitor.ssl_expiring" {
		fmt.Fprintf(&b, "到期时间：%s（剩余 %d 天）\n", m.CertExpiresAt.Format(time.RFC3339), m.DaysLeft)
	}

	fmt.Fprintf(&b, "时间：%s\n", m.CheckedAt.Format(time.RFC3339))

	return b.String()
}
