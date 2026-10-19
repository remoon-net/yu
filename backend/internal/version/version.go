// Package version 保存构建期注入的版本信息。
//
// 版本号、提交与构建时间通过 go build -ldflags "-X ..." 写入，
// 本地 go run 时保持默认值。
package version

var (
	// Version 来自 git describe（例如 v1.2.3 或 v1.2.3-5-gabc1234）。
	Version = "dev"
	// Commit 是构建时的短提交哈希。
	Commit = "unknown"
	// BuildTime 是 UTC 构建时间。
	BuildTime = "unknown"
)

// Full 返回便于日志输出的版本描述。
func Full() string {
	return Version + " (" + Commit + ")"
}
