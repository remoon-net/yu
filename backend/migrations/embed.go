package migrations

import "embed"

// FS 内嵌全部 SQL 迁移文件，文件名格式为 0001_name.sql。
//
//go:embed *.sql
var FS embed.FS
