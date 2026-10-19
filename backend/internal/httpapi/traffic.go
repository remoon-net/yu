package httpapi

import (
	"fmt"
	"log/slog"
	"net/netip"
	"strings"
)

// trafficFilter 判断一次请求是否应跳过统计写入。
type trafficFilter struct {
	filterBots bool
	excluded   []netip.Prefix
}

func newTrafficFilter(filterBots bool, cidrs []string) trafficFilter {
	filter := trafficFilter{filterBots: filterBots}

	for _, raw := range cidrs {
		prefix, err := parseTrafficPrefix(raw)
		if err != nil {
			slog.Warn("忽略不合法的排除 IP/CIDR", "value", raw, "err", err)
			continue
		}
		filter.excluded = append(filter.excluded, prefix)
	}

	return filter
}

func parseTrafficPrefix(raw string) (netip.Prefix, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return netip.Prefix{}, fmt.Errorf("空的 IP/CIDR")
	}

	if prefix, err := netip.ParsePrefix(raw); err == nil {
		return prefix.Masked(), nil
	}

	addr, err := netip.ParseAddr(raw)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("不是合法的 IP 或 CIDR")
	}

	bits := 32
	if addr.Is6() {
		bits = 128
	}
	return netip.PrefixFrom(addr, bits), nil
}

// normalizeTrafficCIDRs 校验并规范化面板里填写的排除 IP/CIDR。
func normalizeTrafficCIDRs(values []string) ([]string, error) {
	out := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))

	for _, raw := range values {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		prefix, err := parseTrafficPrefix(raw)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", strings.TrimSpace(raw), err)
		}

		value := prefix.String()
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}

	if len(out) > 64 {
		return nil, fmt.Errorf("最多支持 64 个排除 IP/CIDR")
	}

	return out, nil
}

func (f trafficFilter) shouldSkip(ip string, isBot bool) bool {
	if f.filterBots && isBot {
		return true
	}

	addr, err := netip.ParseAddr(strings.TrimSpace(ip))
	if err != nil {
		return false
	}
	addr = addr.Unmap()

	for _, prefix := range f.excluded {
		if prefix.Contains(addr) {
			return true
		}
	}

	return false
}
