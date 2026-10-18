package probe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

type tcpConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func parseTCPConfig(raw json.RawMessage) (tcpConfig, error) {
	var cfg tcpConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("TCP 配置格式不正确: %w", err)
	}

	cfg.Host = strings.TrimSpace(cfg.Host)
	if cfg.Host == "" {
		return cfg, errors.New("host 不能为空")
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return cfg, errors.New("port 必须在 1-65535 之间")
	}

	return cfg, nil
}

func runTCP(ctx context.Context, raw json.RawMessage, timeout time.Duration) Result {
	cfg, err := parseTCPConfig(raw)
	if err != nil {
		return Result{Status: "down", Error: err.Error()}
	}

	address := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	dialer := &net.Dialer{Timeout: timeout}

	start := time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", address)
	latency := int(time.Since(start).Milliseconds())

	if err != nil {
		message := err.Error()
		if errors.Is(err, context.DeadlineExceeded) {
			message = "TCP 连接超时"
		}
		return Result{Status: "down", LatencyMS: latency, Error: message}
	}
	_ = conn.Close()

	return Result{Status: "up", LatencyMS: latency}
}
