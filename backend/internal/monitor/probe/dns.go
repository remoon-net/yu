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

type dnsConfig struct {
	Server         string   `json:"server"`
	Name           string   `json:"name"`
	RecordType     string   `json:"record_type"`
	ExpectedValues []string `json:"expected_values"`
}

var dnsRecordTypes = map[string]bool{
	"A": true, "AAAA": true, "CNAME": true, "MX": true, "TXT": true, "NS": true,
}

func parseDNSConfig(raw json.RawMessage) (dnsConfig, error) {
	var cfg dnsConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("DNS 配置格式不正确: %w", err)
	}

	cfg.Name = strings.TrimSpace(cfg.Name)
	if cfg.Name == "" {
		return cfg, errors.New("name 不能为空")
	}

	cfg.RecordType = strings.ToUpper(strings.TrimSpace(cfg.RecordType))
	if cfg.RecordType == "" {
		cfg.RecordType = "A"
	}
	if !dnsRecordTypes[cfg.RecordType] {
		return cfg, fmt.Errorf("不支持的记录类型: %s", cfg.RecordType)
	}

	return cfg, nil
}

func runDNS(ctx context.Context, raw json.RawMessage, timeout time.Duration) Result {
	cfg, err := parseDNSConfig(raw)
	if err != nil {
		return Result{Status: "down", Error: err.Error()}
	}

	resolver := net.DefaultResolver

	if server := strings.TrimSpace(cfg.Server); server != "" {
		host := server
		if parsedHost, _, err := net.SplitHostPort(server); err == nil {
			host = parsedHost
		}

		resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				dialer := net.Dialer{Timeout: timeout}
				return dialer.DialContext(ctx, "udp", net.JoinHostPort(host, "53"))
			},
		}
	}

	lookupCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	values, err := lookupDNS(lookupCtx, resolver, cfg)
	latency := int(time.Since(start).Milliseconds())

	if err != nil {
		message := err.Error()
		if errors.Is(err, context.DeadlineExceeded) {
			message = "DNS 查询超时"
		}
		return Result{Status: "down", LatencyMS: latency, Error: message}
	}

	if len(cfg.ExpectedValues) > 0 {
		present := make(map[string]bool, len(values))
		for _, value := range values {
			present[normalizeDNSValue(value)] = true
		}

		for _, expected := range cfg.ExpectedValues {
			if !present[normalizeDNSValue(expected)] {
				return Result{
					Status:    "down",
					LatencyMS: latency,
					Error:     fmt.Sprintf("未找到期望的记录值: %s", expected),
				}
			}
		}
	}

	return Result{Status: "up", LatencyMS: latency}
}

func lookupDNS(ctx context.Context, resolver *net.Resolver, cfg dnsConfig) ([]string, error) {
	switch cfg.RecordType {
	case "A":
		ips, err := resolver.LookupIP(ctx, "ip4", cfg.Name)
		if err != nil {
			return nil, err
		}
		values := make([]string, 0, len(ips))
		for _, ip := range ips {
			values = append(values, ip.String())
		}
		return values, nil

	case "AAAA":
		ips, err := resolver.LookupIP(ctx, "ip6", cfg.Name)
		if err != nil {
			return nil, err
		}
		values := make([]string, 0, len(ips))
		for _, ip := range ips {
			values = append(values, ip.String())
		}
		return values, nil

	case "CNAME":
		cname, err := resolver.LookupCNAME(ctx, cfg.Name)
		if err != nil {
			return nil, err
		}
		return []string{cname}, nil

	case "MX":
		records, err := resolver.LookupMX(ctx, cfg.Name)
		if err != nil {
			return nil, err
		}
		values := make([]string, 0, len(records))
		for _, record := range records {
			values = append(values, record.Host)
		}
		return values, nil

	case "TXT":
		return resolver.LookupTXT(ctx, cfg.Name)

	case "NS":
		records, err := resolver.LookupNS(ctx, cfg.Name)
		if err != nil {
			return nil, err
		}
		values := make([]string, 0, len(records))
		for _, record := range records {
			values = append(values, record.Host)
		}
		return values, nil
	}

	return nil, fmt.Errorf("不支持的记录类型: %s", cfg.RecordType)
}

func normalizeDNSValue(value string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(value), "."))
}
