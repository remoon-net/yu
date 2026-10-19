package probe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type httpConfig struct {
	URL             string            `json:"url"`
	Method          string            `json:"method"`
	FollowRedirects *bool             `json:"follow_redirects"`
	MaxRedirects    int               `json:"max_redirects"`
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"`
	ExpectedStatus  []int             `json:"expected_status"`
	BodyContains    string            `json:"body_contains"`
	BodyNotContains string            `json:"body_not_contains"`
	JSONPath        string            `json:"json_path"`
	JSONEquals      any               `json:"json_equals"`
	MaxLatencyMS    int               `json:"max_latency_ms"`
}

func parseHTTPConfig(raw json.RawMessage) (httpConfig, error) {
	var cfg httpConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("HTTP 配置格式不正确: %w", err)
	}

	cfg.URL = strings.TrimSpace(cfg.URL)
	if cfg.URL == "" {
		return cfg, errors.New("url 不能为空")
	}

	parsed, err := url.Parse(cfg.URL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return cfg, errors.New("url 必须是合法的 http(s) 地址")
	}

	if cfg.Method == "" {
		cfg.Method = http.MethodGet
	}
	cfg.Method = strings.ToUpper(strings.TrimSpace(cfg.Method))
	if strings.ContainsAny(cfg.Method, " \t\r\n") {
		return cfg, errors.New("method 不合法")
	}

	switch len(cfg.ExpectedStatus) {
	case 0:
		cfg.ExpectedStatus = []int{200, 299}
	case 1, 2:
	default:
		return cfg, errors.New("expected_status 需要 1 个或 2 个数字")
	}
	for _, status := range cfg.ExpectedStatus {
		if status < 100 || status > 599 {
			return cfg, errors.New("expected_status 需要在 100-599 之间")
		}
	}
	if len(cfg.ExpectedStatus) == 2 && cfg.ExpectedStatus[0] > cfg.ExpectedStatus[1] {
		return cfg, errors.New("expected_status 起始值不能大于结束值")
	}

	if len(cfg.Headers) > 50 {
		return cfg, errors.New("自定义 Header 不能超过 50 个")
	}
	headers := make(map[string]string, len(cfg.Headers))
	for key, value := range cfg.Headers {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return cfg, errors.New("Header 名称不能为空")
		}
		if !validHeaderName(key) {
			return cfg, errors.New("Header 名称不合法")
		}
		if len(key) > 100 || len(value) > 4096 {
			return cfg, errors.New("Header 名称或内容过长")
		}
		if strings.ContainsAny(key, "\r\n") || strings.ContainsAny(value, "\r\n") {
			return cfg, errors.New("Header 不能包含换行")
		}
		headers[key] = value
	}
	cfg.Headers = headers

	if len(cfg.Body) > 64<<10 {
		return cfg, errors.New("请求体不能超过 64KB")
	}

	if cfg.MaxLatencyMS < 0 {
		return cfg, errors.New("max_latency_ms 不能为负数")
	}

	if cfg.MaxRedirects <= 0 {
		cfg.MaxRedirects = 5
	}

	return cfg, nil
}

// validHeaderName 校验 RFC 7230 的 token 形式 Header 名称。
func validHeaderName(name string) bool {
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case strings.ContainsRune("!#$%&'*+-.^_`|~", r):
		default:
			return false
		}
	}
	return name != ""
}

func runHTTP(ctx context.Context, raw json.RawMessage, timeout time.Duration) Result {
	cfg, err := parseHTTPConfig(raw)
	if err != nil {
		return Result{Status: "down", Error: err.Error()}
	}

	follow := true
	if cfg.FollowRedirects != nil {
		follow = *cfg.FollowRedirects
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		IdleConnTimeout:       30 * time.Second,
		DisableKeepAlives:     true,
	}
	defer transport.CloseIdleConnections()

	client := &http.Client{Transport: transport, Timeout: timeout}
	redirects := 0
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if !follow {
			return http.ErrUseLastResponse
		}
		redirects++
		if redirects > cfg.MaxRedirects {
			return fmt.Errorf("重定向次数超过 %d", cfg.MaxRedirects)
		}
		return nil
	}

	var reqBody io.Reader
	if cfg.Body != "" {
		reqBody = strings.NewReader(cfg.Body)
	}

	req, err := http.NewRequestWithContext(ctx, cfg.Method, cfg.URL, reqBody)
	if err != nil {
		return Result{Status: "down", Error: err.Error()}
	}

	for key, value := range cfg.Headers {
		if strings.EqualFold(key, "Host") {
			req.Host = value
			continue
		}
		req.Header.Set(key, value)
	}

	// 未显式指定 Content-Type 时，为 JSON 请求体补上默认值。
	if cfg.Body != "" && req.Header.Get("Content-Type") == "" && json.Valid([]byte(cfg.Body)) {
		req.Header.Set("Content-Type", "application/json")
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		message := err.Error()
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(message, "Client.Timeout") {
			message = "请求超时"
		}
		return Result{
			Status:    "down",
			LatencyMS: int(time.Since(start).Milliseconds()),
			Error:     message,
		}
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	latencyMS := int(time.Since(start).Milliseconds())

	result := Result{
		Status:     "up",
		LatencyMS:  latencyMS,
		StatusCode: resp.StatusCode,
	}

	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		expires := resp.TLS.PeerCertificates[0].NotAfter
		result.CertExpiresAt = &expires
	}

	if readErr != nil {
		result.Status = "down"
		result.Error = "读取响应失败: " + readErr.Error()
		return result
	}

	statusOK := resp.StatusCode == cfg.ExpectedStatus[0]
	if len(cfg.ExpectedStatus) == 2 {
		statusOK = resp.StatusCode >= cfg.ExpectedStatus[0] && resp.StatusCode <= cfg.ExpectedStatus[1]
	}
	if !statusOK {
		result.Status = "down"
		result.Error = fmt.Sprintf("状态码 %d 不符合预期", resp.StatusCode)
		return result
	}

	text := string(body)

	if cfg.BodyContains != "" && !strings.Contains(text, cfg.BodyContains) {
		result.Status = "down"
		result.Error = "响应体不包含期望内容"
		return result
	}

	if cfg.BodyNotContains != "" && strings.Contains(text, cfg.BodyNotContains) {
		result.Status = "down"
		result.Error = "响应体包含被禁止的内容"
		return result
	}

	if cfg.JSONPath != "" {
		var data any
		if err := json.Unmarshal(body, &data); err != nil {
			result.Status = "down"
			result.Error = "响应不是合法 JSON"
			return result
		}

		value, found := jsonPathGet(data, cfg.JSONPath)
		if !found {
			result.Status = "down"
			result.Error = fmt.Sprintf("JSON 路径 %s 不存在", cfg.JSONPath)
			return result
		}

		if cfg.JSONEquals != nil && !jsonValueEqual(value, cfg.JSONEquals) {
			result.Status = "down"
			result.Error = fmt.Sprintf("JSON 路径 %s 的值不符合预期", cfg.JSONPath)
			return result
		}
	}

	if cfg.MaxLatencyMS > 0 && latencyMS > cfg.MaxLatencyMS {
		result.Status = "down"
		result.Error = fmt.Sprintf("响应耗时 %dms 超过阈值 %dms", latencyMS, cfg.MaxLatencyMS)
		return result
	}

	return result
}

// jsonPathGet 支持 $.a.b 与 $.a[0] 形式的简单路径。
func jsonPathGet(data any, path string) (any, bool) {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "$")
	path = strings.TrimPrefix(path, ".")
	if path == "" {
		return data, true
	}

	current := data

	for _, segment := range strings.Split(path, ".") {
		if segment == "" {
			continue
		}

		name := segment
		index := -1

		if open := strings.Index(segment, "["); open >= 0 && strings.HasSuffix(segment, "]") {
			name = segment[:open]
			rawIndex := segment[open+1 : len(segment)-1]
			parsed, err := strconv.Atoi(rawIndex)
			if err != nil {
				return nil, false
			}
			index = parsed
		}

		object, ok := current.(map[string]any)
		if name != "" {
			if !ok {
				return nil, false
			}
			current, ok = object[name]
			if !ok {
				return nil, false
			}
		}

		if index >= 0 {
			array, ok := current.([]any)
			if !ok || index >= len(array) {
				return nil, false
			}
			current = array[index]
		}
	}

	return current, true
}

func jsonValueEqual(actual, expected any) bool {
	if expected == nil {
		return actual == nil
	}

	actualJSON, err := json.Marshal(actual)
	if err != nil {
		return false
	}
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		return false
	}

	return string(actualJSON) == string(expectedJSON)
}
