package mailer

import (
	"context"
	"io"
	"net"
	"strconv"
	"testing"
	"time"
)

func TestSendWithImplicitTLS(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("创建监听失败: %v", err)
	}
	defer listener.Close()

	firstByte := make(chan byte, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		buf := make([]byte, 1)
		if _, err := io.ReadFull(conn, buf); err == nil {
			firstByte <- buf[0]
		}
	}()

	host, rawPort, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("解析监听地址失败: %v", err)
	}
	port, err := strconv.Atoi(rawPort)
	if err != nil {
		t.Fatalf("解析端口失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = New().Send(ctx, Options{
		Host: host,
		Port: port,
		From: "Yu <yu@example.com>",
		TLS:  true,
	}, "ops@example.com", "subject", "body")
	if err == nil {
		t.Fatal("服务端立即关闭时应返回 TLS 握手错误")
	}

	select {
	case b := <-firstByte:
		// TLS ClientHello 的 record type 为 0x16。
		if b != 0x16 {
			t.Fatalf("首个字节 = %#x, want 0x16（TLS 握手）", b)
		}
	case <-time.After(time.Second):
		t.Fatal("服务端未收到 TLS 握手数据")
	}
}
