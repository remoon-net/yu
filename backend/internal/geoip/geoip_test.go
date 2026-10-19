package geoip

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"testing"
	"time"
)

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

	lookup, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	if got := lookup.Location("8.8.8.8"); got != (Location{}) {
		t.Fatalf("empty Lookup = %+v, want empty", got)
	}
}

func TestDBIPCountryURL(t *testing.T) {
	got := dbipCountryURL(time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC))
	want := "https://download.db-ip.com/free/dbip-country-lite-2026-10.mmdb.gz"
	if got != want {
		t.Fatalf("dbipCountryURL() = %q, want %q", got, want)
	}
}

func TestWriteMMDBGzip(t *testing.T) {
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	if _, err := gz.Write([]byte("mmdb")); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := writeMMDB(&out, &compressed, "https://example.com/db.mmdb.gz"); err != nil {
		t.Fatal(err)
	}
	if out.String() != "mmdb" {
		t.Fatalf("writeMMDB() = %q", out.String())
	}
}

func TestWriteMMDBTarGz(t *testing.T) {
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	tw := tar.NewWriter(gz)
	body := []byte("maxmind")
	if err := tw.WriteHeader(&tar.Header{
		Name: "GeoLite2-City_20261019/GeoLite2-City.mmdb",
		Mode: 0o644,
		Size: int64(len(body)),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	url := "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&suffix=tar.gz"
	if err := writeMMDB(&out, &compressed, url); err != nil {
		t.Fatal(err)
	}
	if out.String() != string(body) {
		t.Fatalf("writeMMDB() = %q", out.String())
	}
}
