package main

import (
	"context"
	"log/slog"

	"remoon.net/yu/backend/internal/auth"
	"remoon.net/yu/backend/internal/config"
	"remoon.net/yu/backend/internal/store"
)

// bootstrapAdmin 在数据库为空时按环境变量创建初始管理员与默认工作区。
func bootstrapAdmin(ctx context.Context, st *store.Store, cfg config.Config) error {
	count, err := st.CountUsers(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	if cfg.BootstrapEmail == "" || cfg.BootstrapPassword == "" {
		slog.Warn("当前没有任何用户，请设置 YU_BOOTSTRAP_EMAIL 与 YU_BOOTSTRAP_PASSWORD 以创建初始管理员")
		return nil
	}

	passwordHash, err := auth.HashPassword(cfg.BootstrapPassword)
	if err != nil {
		return err
	}

	user, err := st.CreateUser(ctx, cfg.BootstrapEmail, "管理员", passwordHash)
	if err != nil {
		return err
	}

	workspace, err := st.CreateWorkspace(ctx, cfg.BootstrapWorkspace, "", user.ID)
	if err != nil {
		return err
	}

	slog.Info("已创建初始管理员", "email", user.Email, "workspace", workspace.Slug)
	return nil
}
