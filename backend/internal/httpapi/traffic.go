package httpapi

import (
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
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		if prefix, err := netip.ParsePrefix(raw); err == nil {
			filter.excluded = append(filter.excluded, prefix)
			continue
		}

		addr, err := netip.ParseAddr(raw)
		if err != nil {
			slog.Warn("忽略不合法的排除 IP/CIDR", "value", raw, "err", err)
			continue
		}

		bits := 32
		if addr.Is6() {
			bits = 128
		}
		filter.excluded = append(filter.excluded, netip.PrefixFrom(addr, bits))
	}

	return filter
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
