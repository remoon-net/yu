package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Result 是一次探测的结果。
type Result struct {
	Status        string
	LatencyMS     int
	StatusCode    int
	Error         string
	CertExpiresAt *time.Time
}

// Run 根据监控类型执行一次探测。
func Run(ctx context.Context, monitorType string, rawConfig json.RawMessage, timeout time.Duration) Result {
	switch monitorType {
	case "http":
		return runHTTP(ctx, rawConfig, timeout)
	case "tcp":
		return runTCP(ctx, rawConfig, timeout)
	case "ping":
		return runPing(ctx, rawConfig, timeout)
	case "dns":
		return runDNS(ctx, rawConfig, timeout)
	default:
		return Result{Status: "down", Error: "不支持的监控类型"}
	}
}

// Validate 校验监控配置。
func Validate(monitorType string, rawConfig json.RawMessage) error {
	switch monitorType {
	case "http":
		_, err := parseHTTPConfig(rawConfig)
		return err
	case "tcp":
		_, err := parseTCPConfig(rawConfig)
		return err
	case "ping":
		_, err := parsePingConfig(rawConfig)
		return err
	case "dns":
		_, err := parseDNSConfig(rawConfig)
		return err
	default:
		return fmt.Errorf("不支持的监控类型: %s", monitorType)
	}
}
