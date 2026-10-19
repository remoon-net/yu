package geoip

import (
	"net"
	"strings"

	"github.com/oschwald/maxminddb-golang"
)

// Location 是一次 IP 地理位置查询结果。
type Location struct {
	Country string
	Region  string
	City    string
}

// Lookup 封装 MaxMind GeoLite2 数据库。
type Lookup struct {
	reader *maxminddb.Reader
}

type record struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
	Subdivisions []struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"subdivisions"`
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
}

// Open 打开 GeoLite2 数据库文件。path 为空时返回 nil，表示未启用。
func Open(path string) (*Lookup, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}

	reader, err := maxminddb.Open(path)
	if err != nil {
		return nil, err
	}
	return &Lookup{reader: reader}, nil
}

// Close 关闭数据库。
func (l *Lookup) Close() error {
	if l == nil || l.reader == nil {
		return nil
	}
	return l.reader.Close()
}

// Location 查询 IP 对应的大洲/国家/地区/城市；未启用或查询失败时返回空值。
func (l *Lookup) Location(ip string) Location {
	if l == nil || l.reader == nil {
		return Location{}
	}

	addr := net.ParseIP(strings.TrimSpace(ip))
	if addr == nil {
		return Location{}
	}

	var rec record
	if err := l.reader.Lookup(addr, &rec); err != nil {
		return Location{}
	}

	location := Location{Country: strings.TrimSpace(rec.Country.ISOCode)}
	if len(rec.Subdivisions) > 0 {
		location.Region = localizedName(rec.Subdivisions[0].Names)
	}
	location.City = localizedName(rec.City.Names)

	return location
}

func localizedName(names map[string]string) string {
	for _, key := range []string{"zh-CN", "zh", "en"} {
		if value := strings.TrimSpace(names[key]); value != "" {
			return value
		}
	}
	return ""
}
