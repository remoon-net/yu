package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 保存服务运行所需的全部配置，来自环境变量。
type Config struct {
	Addr               string
	DatabaseURL        string
	PublicBaseURL      string
	AllowedOrigins     []string
	SecureCookies      bool
	SessionTTL         time.Duration
	SMTPHost           string
	SMTPPort           int
	SMTPUser           string
	SMTPPass           string
	SMTPFrom           string
	BootstrapEmail     string
	BootstrapPassword  string
	BootstrapWorkspace string
}

// Load 从环境变量读取配置并填充默认值。
func Load() Config {
	cfg := Config{
		Addr:               env("YU_ADDR", ":8080"),
		DatabaseURL:        env("YU_DATABASE_URL", "postgres://yu:yu@127.0.0.1:5432/yu?sslmode=disable"),
		PublicBaseURL:      strings.TrimRight(env("YU_PUBLIC_BASE_URL", "http://localhost:8080"), "/"),
		SecureCookies:      envBool("YU_SECURE_COOKIES", false),
		SessionTTL:         envDuration("YU_SESSION_TTL", 720*time.Hour),
		SMTPHost:           env("YU_SMTP_HOST", ""),
		SMTPPort:           envInt("YU_SMTP_PORT", 587),
		SMTPUser:           env("YU_SMTP_USER", ""),
		SMTPPass:           env("YU_SMTP_PASS", ""),
		SMTPFrom:           env("YU_SMTP_FROM", "Yu <noreply@localhost>"),
		BootstrapEmail:     env("YU_BOOTSTRAP_EMAIL", ""),
		BootstrapPassword:  env("YU_BOOTSTRAP_PASSWORD", ""),
		BootstrapWorkspace: env("YU_BOOTSTRAP_WORKSPACE", "默认工作区"),
	}

	for _, origin := range strings.Split(env("YU_ALLOWED_ORIGINS", ""), ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			cfg.AllowedOrigins = append(cfg.AllowedOrigins, strings.TrimRight(origin, "/"))
		}
	}

	return cfg
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return v
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func envDuration(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return v
}
