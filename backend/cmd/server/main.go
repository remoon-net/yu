package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"remoon.net/yu/backend/internal/config"
	"remoon.net/yu/backend/internal/httpapi"
	"remoon.net/yu/backend/internal/mailer"
	"remoon.net/yu/backend/internal/monitor"
	"remoon.net/yu/backend/internal/store"
	"remoon.net/yu/backend/internal/version"
)

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)
	slog.Info("Yu 启动", "version", version.Full(), "buildTime", version.BuildTime)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("连接数据库失败", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	if err := st.Migrate(ctx); err != nil {
		slog.Error("执行迁移失败", "err", err)
		os.Exit(1)
	}

	if err := st.EnsureEventPartitions(ctx, 2); err != nil {
		slog.Error("创建事件分区失败", "err", err)
		os.Exit(1)
	}

	if err := st.EnsureMonitorCheckPartitions(ctx, 2); err != nil {
		slog.Error("创建监控结果分区失败", "err", err)
		os.Exit(1)
	}

	if removed, err := st.DeleteExpiredSessions(ctx); err != nil {
		slog.Warn("清理过期会话失败", "err", err)
	} else if removed > 0 {
		slog.Info("已清理过期会话", "count", removed)
	}

	if err := bootstrapAdmin(ctx, st, cfg); err != nil {
		slog.Error("初始化管理员失败", "err", err)
		os.Exit(1)
	}

	mail := mailer.New()

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpapi.New(cfg, st, mail),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	scheduler := monitor.NewScheduler(st, cfg.ProbeConcurrency, monitor.NewSender(mail))
	go scheduler.Run(ctx)

	go func() {
		slog.Info("HTTP 服务已启动", "addr", cfg.Addr, "version", version.Full())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP 服务异常退出", "err", err)
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("关闭 HTTP 服务失败", "err", err)
	}

	slog.Info("服务已停止")
}
