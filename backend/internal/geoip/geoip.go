package geoip

import (
	"net"
	"strings"
	"sync"

	"github.com/oschwald/maxminddb-golang"
)

// Location 是一次 IP 地理位置查询结果。
type Location struct {
	Country string
	Region  string
	City    string
}

// Lookup 封装 MaxMind 兼容的 MMDB 数据库，支持更新后热切换。
type Lookup struct {
	mu     sync.RWMutex
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

// New 创建查询器并尝试加载 path；加载失败时返回空的查询器，仍可由更新器稍后填充。
func New(path string) (*Lookup, error) {
	lookup := &Lookup{}
	if strings.TrimSpace(path) == "" {
		return lookup, nil
	}
	if err := lookup.Load(path); err != nil {
		return lookup, err
	}
	return lookup, nil
}

// Open 打开 MMDB 数据库文件。path 为空时返回未启用的查询器。
func Open(path string) (*Lookup, error) {
	return New(path)
}

// Load 打开新数据库并替换当前 reader；成功后旧 reader 会被关闭。
func (l *Lookup) Load(path string) error {
	reader, err := maxminddb.Open(path)
	if err != nil {
		return err
	}

	l.mu.Lock()
	old := l.reader
	l.reader = reader
	l.mu.Unlock()

	if old != nil {
		_ = old.Close()
	}
	return nil
}

// Close 关闭数据库。
func (l *Lookup) Close() error {
	if l == nil {
		return nil
	}

	l.mu.Lock()
	reader := l.reader
	l.reader = nil
	l.mu.Unlock()

	if reader != nil {
		return reader.Close()
	}
	return nil
}

// Location 查询 IP 对应的大洲/国家/地区/城市；未启用或查询失败时返回空值。
func (l *Lookup) Location(ip string) Location {
	if l == nil {
		return Location{}
	}

	addr := net.ParseIP(strings.TrimSpace(ip))
	if addr == nil {
		return Location{}
	}

	l.mu.RLock()
	reader := l.reader
	l.mu.RUnlock()
	if reader == nil {
		return Location{}
	}

	var rec record
	if err := reader.Lookup(addr, &rec); err != nil {
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
