package collectscript

import _ "embed"

// JS 是内嵌的浏览器采集脚本。
//
//go:embed collect.js
var JS []byte
