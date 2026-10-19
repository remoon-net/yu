package store

import (
	"fmt"
	"strings"
)

// StatsFilter 是访客分段条件；所有条件按 AND 组合。
type StatsFilter struct {
	Path         string `json:"path"`
	ReferrerHost string `json:"referrer_host"`
	Country      string `json:"country"`
	Device       string `json:"device"`
	Browser      string `json:"browser"`
	UTMSource    string `json:"utm_source"`
}

// Empty 表示没有设置任何过滤条件。
func (f StatsFilter) Empty() bool {
	return strings.TrimSpace(f.Path) == "" &&
		strings.TrimSpace(f.ReferrerHost) == "" &&
		strings.TrimSpace(f.Country) == "" &&
		strings.TrimSpace(f.Device) == "" &&
		strings.TrimSpace(f.Browser) == "" &&
		strings.TrimSpace(f.UTMSource) == ""
}

// scopeSQL 生成“访客命中过滤条件”的子查询，参数从 next 开始编号。
// 调用方必须保证 $1/$2/$3 分别是 site_id、from、to。
func (f StatsFilter) scopeSQL(next int) (string, []any) {
	return f.scopeSQLFor("", next)
}

// scopeSQLFor 与 scopeSQL 相同，但可以用 alias 指定外层 visitor_id 的表别名。
func (f StatsFilter) scopeSQLFor(alias string, next int) (string, []any) {
	conditions := make([]string, 0, 6)
	args := make([]any, 0, 6)

	add := func(column, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		conditions = append(conditions, fmt.Sprintf("%s = $%d", column, next))
		args = append(args, value)
		next++
	}

	add("path", f.Path)
	add("referrer_host", f.ReferrerHost)
	add("country", f.Country)
	add("device", f.Device)
	add("browser", f.Browser)
	add("utm_source", f.UTMSource)

	if len(conditions) == 0 {
		return "", nil
	}

	visitorColumn := "visitor_id"
	if alias != "" {
		visitorColumn = alias + ".visitor_id"
	}

	return " and " + visitorColumn + " in (select visitor_id from events where site_id = $1 and created_at >= $2 and created_at < $3 and " +
		strings.Join(conditions, " and ") + ")", args
}
