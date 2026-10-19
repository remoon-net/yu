package geoip

import "testing"

func TestLocalizedName(t *testing.T) {
	cases := []struct {
		names map[string]string
		want  string
	}{
		{map[string]string{"zh-CN": "上海", "en": "Shanghai"}, "上海"},
		{map[string]string{"zh": "上海", "en": "Shanghai"}, "上海"},
		{map[string]string{"en": "Shanghai"}, "Shanghai"},
		{map[string]string{"fr": "Shanghai"}, ""},
		{nil, ""},
	}

	for _, tc := range cases {
		if got := localizedName(tc.names); got != tc.want {
			t.Fatalf("localizedName(%v) = %q, want %q", tc.names, got, tc.want)
		}
	}
}

func TestLookupWithoutDatabase(t *testing.T) {
	var lookup *Lookup
	if got := lookup.Location("8.8.8.8"); got != (Location{}) {
		t.Fatalf("nil Lookup = %+v, want empty", got)
	}
}
