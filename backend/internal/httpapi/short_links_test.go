package httpapi

import (
	"regexp"
	"testing"
)

func TestValidateShortLinkTarget(t *testing.T) {
	valid := []string{
		"https://example.com/promo",
		"http://example.com",
		"https://shop.example.net/a?utm_source=x",
		"  https://example.com/path  ",
	}
	for _, raw := range valid {
		if _, ok := validateShortLinkTarget(raw); !ok {
			t.Fatalf("validateShortLinkTarget(%q) = false, want true", raw)
		}
	}

	invalid := []string{
		"",
		"example.com",
		"ftp://example.com",
		"javascript:alert(1)",
		"//example.com",
		"https://",
	}
	for _, raw := range invalid {
		if _, ok := validateShortLinkTarget(raw); ok {
			t.Fatalf("validateShortLinkTarget(%q) = true, want false", raw)
		}
	}
}

func TestValidateShortLinkTargetTrims(t *testing.T) {
	got, ok := validateShortLinkTarget("  https://example.com/x  ")
	if !ok || got != "https://example.com/x" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
}

func TestNewShortLinkSlug(t *testing.T) {
	pattern := regexp.MustCompile(`^[a-z0-9]{8}$`)
	seen := make(map[string]bool, 100)

	for i := 0; i < 100; i++ {
		slug := newShortLinkSlug()
		if !pattern.MatchString(slug) {
			t.Fatalf("slug %q 不符合 8 位小写字母/数字格式", slug)
		}
		seen[slug] = true
	}

	if len(seen) < 95 {
		t.Fatalf("100 次生成只出现 %d 个不同值，随机性不足", len(seen))
	}
}

func TestShortLinkSlugPattern(t *testing.T) {
	valid := []string{"ab", "promo8", "a-b", "autumn-2026", "a234567890123456789012345678901234567890"}
	for _, slug := range valid {
		if !shortLinkSlugPattern.MatchString(slug) {
			t.Fatalf("slug %q 应该合法", slug)
		}
	}

	invalid := []string{
		"",
		"a",
		"-abc",
		"abc-",
		"ABC",
		"hello world",
		"a_b",
		"a2345678901234567890123456789012345678901",
	}
	for _, slug := range invalid {
		if shortLinkSlugPattern.MatchString(slug) {
			t.Fatalf("slug %q 应该非法", slug)
		}
	}
}
