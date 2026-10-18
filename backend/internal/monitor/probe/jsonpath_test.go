package probe

import "testing"

func TestJSONPathGet(t *testing.T) {
	data := map[string]any{
		"status": "ok",
		"data": map[string]any{
			"version": "1.2.3",
			"items": []any{
				map[string]any{"id": "a"},
				map[string]any{"id": "b"},
			},
		},
		"count": 42,
	}

	cases := []struct {
		path  string
		want  any
		found bool
	}{
		{"$.status", "ok", true},
		{"$.data.version", "1.2.3", true},
		{"$.data.items[0].id", "a", true},
		{"$.data.items[1].id", "b", true},
		{"$.count", 42, true},
		{"status", "ok", true},
		{"$.missing", nil, false},
		{"$.data.items[9].id", nil, false},
		{"$.data.items[x]", nil, false},
		{"$.status.deep", nil, false},
	}

	for _, tc := range cases {
		got, found := jsonPathGet(data, tc.path)
		if found != tc.found {
			t.Fatalf("jsonPathGet(%q) found = %v, want %v", tc.path, found, tc.found)
		}
		if tc.found && got != tc.want {
			t.Fatalf("jsonPathGet(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestJSONPathGetOnNonObject(t *testing.T) {
	if _, found := jsonPathGet("plain string", "$.a"); found {
		t.Fatal("非对象数据应该返回 not found")
	}
}

func TestJSONPathGetRootPath(t *testing.T) {
	data := map[string]any{"a": 1}
	got, found := jsonPathGet(data, "$.")
	if !found {
		t.Fatal("$. 应该返回整个对象")
	}
	if _, ok := got.(map[string]any); !ok {
		t.Fatalf("$. 返回 %T, want map[string]any", got)
	}
}
