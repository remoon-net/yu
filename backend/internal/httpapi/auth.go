package httpapi

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"remoon.net/yu/backend/internal/auth"
	"remoon.net/yu/backend/internal/store"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	user, err := s.store.GetUserByEmail(r.Context(), req.Email)
	if err != nil || !auth.VerifyPassword(user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "邮箱或密码错误")
		return
	}

	if !s.startSession(w, r, user) {
		return
	}

	s.writeSessionPayload(w, r, user, http.StatusOK)
}

type acceptInviteRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func (s *Server) handleAcceptInvite(w http.ResponseWriter, r *http.Request) {
	var req acceptInviteRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if trimSpace(req.Token) == "" {
		writeError(w, http.StatusBadRequest, "邀请链接无效")
		return
	}

	invite, err := s.store.GetInviteByTokenHash(r.Context(), auth.HashToken(req.Token))
	if err != nil || invite.AcceptedAt != nil || time.Now().After(invite.ExpiresAt) {
		writeError(w, http.StatusBadRequest, "邀请无效或已过期")
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	name := trimSpace(req.Name)
	if name == "" {
		name = strings.SplitN(invite.Email, "@", 2)[0]
	}

	user, err := s.store.AcceptInvite(r.Context(), invite.ID, invite.Email, name, passwordHash)
	if err != nil {
		slog.Error("接受邀请失败", "err", err)
		writeError(w, http.StatusInternalServerError, "接受邀请失败")
		return
	}

	if !s.startSession(w, r, user) {
		return
	}

	s.writeSessionPayload(w, r, user, http.StatusOK)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "未登录或会话已过期")
		return
	}

	if _, err := r.Cookie(csrfCookie); err != nil {
		s.setCSRFCookie(w)
	}

	s.writeSessionPayload(w, r, user, http.StatusOK)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
		if err := s.store.DeleteAuthSession(r.Context(), auth.HashToken(c.Value)); err != nil {
			slog.Warn("删除会话失败", "err", err)
		}
	}

	s.clearCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

// startSession 创建会话并写入 Cookie。
func (s *Server) startSession(w http.ResponseWriter, r *http.Request, user *store.User) bool {
	token, err := auth.NewToken()
	if err != nil {
		slog.Error("生成会话 token 失败", "err", err)
		writeError(w, http.StatusInternalServerError, "创建会话失败")
		return false
	}

	expires := time.Now().Add(s.cfg.SessionTTL)
	if err := s.store.CreateAuthSession(r.Context(), auth.HashToken(token), user.ID, expires); err != nil {
		slog.Error("保存会话失败", "err", err)
		writeError(w, http.StatusInternalServerError, "创建会话失败")
		return false
	}

	s.setSessionCookie(w, token, expires)
	s.setCSRFCookie(w)

	return true
}

// writeSessionPayload 返回当前用户与其工作区列表。
func (s *Server) writeSessionPayload(w http.ResponseWriter, r *http.Request, user *store.User, status int) {
	workspaces, err := s.store.ListWorkspacesByUser(r.Context(), user.ID)
	if err != nil {
		slog.Error("查询工作区失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取工作区失败")
		return
	}

	writeJSON(w, status, map[string]any{
		"user":       user,
		"workspaces": workspaces,
	})
}
