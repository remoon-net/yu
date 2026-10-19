package httpapi

import "testing"

func TestTrafficFilterShouldSkip(t *testing.T) {
	filter := newTrafficFilter(true, []string{"192.168.1.0/24", "10.0.0.5", "::1"})

	cases := []struct {
		ip    string
		isBot bool
		want  bool
	}{
		{"203.0.113.10", false, false},
		{"203.0.113.10", true, true},
		{"192.168.1.20", false, true},
		{"10.0.0.5", false, true},
		{"10.0.0.6", false, false},
		{"::1", false, true},
		{"not-an-ip", false, false},
	}

	for _, tc := range cases {
		if got := filter.shouldSkip(tc.ip, tc.isBot); got != tc.want {
			t.Fatalf("shouldSkip(%q, %v) = %v, want %v", tc.ip, tc.isBot, got, tc.want)
		}
	}
}

func TestTrafficFilterBotsDisabled(t *testing.T) {
	filter := newTrafficFilter(false, nil)
	if filter.shouldSkip("203.0.113.10", true) {
		t.Fatal("禁用 bot 过滤时不应跳过请求")
	}
}
