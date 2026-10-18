package probe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type pingConfig struct {
	Host  string `json:"host"`
	Count int    `json:"count"`
}

func parsePingConfig(raw json.RawMessage) (pingConfig, error) {
	var cfg pingConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("Ping 配置格式不正确: %w", err)
	}

	cfg.Host = strings.TrimSpace(cfg.Host)
	if cfg.Host == "" {
		return cfg, errors.New("host 不能为空")
	}

	if cfg.Count <= 0 {
		cfg.Count = 3
	}
	if cfg.Count > 10 {
		cfg.Count = 10
	}

	return cfg, nil
}

func runPing(ctx context.Context, raw json.RawMessage, timeout time.Duration) Result {
	cfg, err := parsePingConfig(raw)
	if err != nil {
		return Result{Status: "down", Error: err.Error()}
	}

	target, err := net.ResolveIPAddr("ip4", cfg.Host)
	if err != nil {
		return Result{Status: "down", Error: "解析主机失败: " + err.Error()}
	}

	conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	network := "udp4"
	if err != nil {
		conn, err = icmp.ListenPacket("ip4:icmp", "0.0.0.0")
		network = "ip4:icmp"
	}
	if err != nil {
		return Result{
			Status: "down",
			Error:  "无法创建 ICMP 套接字（需要 CAP_NET_RAW 或 net.ipv4.ping_group_range 配置）: " + err.Error(),
		}
	}
	defer conn.Close()

	var destination net.Addr = target
	if network == "udp4" {
		destination = &net.UDPAddr{IP: target.IP}
	}

	deadline := time.Now().Add(timeout)
	_ = conn.SetDeadline(deadline)

	id := os.Getpid() & 0xffff
	received := 0
	var total time.Duration

	for seq := 0; seq < cfg.Count; seq++ {
		if ctx.Err() != nil {
			break
		}

		message := icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   id,
				Seq:  seq,
				Data: []byte("yu-ping"),
			},
		}

		payload, err := message.Marshal(nil)
		if err != nil {
			continue
		}

		start := time.Now()
		if _, err := conn.WriteTo(payload, destination); err != nil {
			continue
		}

		buffer := make([]byte, 1500)
		for {
			n, peer, err := conn.ReadFrom(buffer)
			if err != nil {
				break
			}
			if !sameIP(peer, target.IP) {
				continue
			}

			parsed, err := icmp.ParseMessage(ipv4.ICMPTypeEcho.Protocol(), buffer[:n])
			if err != nil {
				continue
			}

			echo, ok := parsed.Body.(*icmp.Echo)
			if !ok || echo.Seq != seq {
				continue
			}

			received++
			total += time.Since(start)
			break
		}

		if time.Now().After(deadline) {
			break
		}
	}

	if received == 0 {
		return Result{Status: "down", Error: "未收到 ICMP 响应", LatencyMS: int(timeout.Milliseconds())}
	}

	return Result{
		Status:    "up",
		LatencyMS: int((total / time.Duration(received)).Milliseconds()),
	}
}

func sameIP(addr net.Addr, ip net.IP) bool {
	switch value := addr.(type) {
	case *net.UDPAddr:
		return value.IP.Equal(ip)
	case *net.IPAddr:
		return value.IP.Equal(ip)
	default:
		return false
	}
}
