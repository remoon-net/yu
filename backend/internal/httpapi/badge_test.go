package httpapi

import (
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func TestBadgeStatus(t *testing.T) {
	cases := []struct {
		status string
		text   string
		color  string
	}{
		{"up", "up", badgeGreen},
		{"down", "down", badgeRed},
		{"pending", "pending", badgeYellow},
		{"recovering", "recovering", badgeYellow},
		{"", "unknown", badgeGrey},
		{"weird", "unknown", badgeGrey},
	}

	for _, tc := range cases {
		text, color := badgeStatus(tc.status)
		if text != tc.text || color != tc.color {
			t.Fatalf("badgeStatus(%q) = (%q, %q), want (%q, %q)", tc.status, text, color, tc.text, tc.color)
		}
	}
}

func TestRenderBadgeEscapesLabel(t *testing.T) {
	svg := renderBadge(badgeSpec{label: `API <prod> & "x"`, message: "up", color: badgeGreen})

	if strings.Contains(svg, "<prod>") {
		t.Fatalf("label was not escaped: %s", svg)
	}
	if !strings.Contains(svg, "API &lt;prod&gt; &amp; &#34;x&#34;") {
		t.Fatalf("label missing from svg: %s", svg)
	}
	if !strings.Contains(svg, `fill="`+badgeGreen+`"`) {
		t.Fatalf("status color missing from svg: %s", svg)
	}
}

func TestBadgeTextWidth(t *testing.T) {
	if got := badgeTextWidth("up"); got != 13 {
		t.Fatalf("badgeTextWidth(up) = %d, want 13", got)
	}
	// 中文字符按 11px 全角计算
	if got := badgeTextWidth("正常"); got != 22 {
		t.Fatalf("badgeTextWidth(正常) = %d, want 22", got)
	}
}

func TestRenderBadgeIsValidXML(t *testing.T) {
	svg := renderBadge(badgeSpec{label: `API <prod> & "x"`, message: "up 42ms", color: badgeGreen})

	decoder := xml.NewDecoder(strings.NewReader(svg))
	for {
		_, err := decoder.Token()
		if err == io.EOF {
			return
		}
		if err != nil {
			t.Fatalf("invalid svg xml: %v\n%s", err, svg)
		}
	}
}
