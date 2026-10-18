package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"remoon.net/yu/backend/internal/store"
)

func (s *Server) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	ws, _, ok := s.workspaceAccess(w, r, "admin")
	if !ok {
		return
	}

	keys, err := s.store.ListAPIKeys(r.Context(), ws.ID)
	if err != nil {
		slog.Error("查询 API Key 失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取 API Key 失败")
		return
	}

	writeJSON(w, http.StatusOK, keys)
}

func (s *Server) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	ws, _, ok := s.workspaceAccess(w, r, "admin")
	if !ok {
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	name := trimSpace(req.Name)
	if name == "" || len([]rune(name)) > 60 {
		writeError(w, http.StatusBadRequest, "名称需要 1-60 个字符")
		return
	}

	user := userFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "未登录或会话已过期")
		return
	}

	key, token, err := s.store.CreateAPIKey(r.Context(), ws.ID, user.ID, name)
	if err != nil {
		slog.Error("创建 API Key 失败", "err", err)
		writeError(w, http.StatusInternalServerError, "创建 API Key 失败")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"key":   key,
		"token": token,
	})
}

func (s *Server) handleDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	ws, _, ok := s.workspaceAccess(w, r, "admin")
	if !ok {
		return
	}

	keyID, err := uuid.Parse(chi.URLParam(r, "keyID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "API Key 不存在")
		return
	}

	if err := s.store.DeleteAPIKey(r.Context(), ws.ID, keyID); err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "API Key 不存在")
			return
		}
		slog.Error("删除 API Key 失败", "err", err)
		writeError(w, http.StatusInternalServerError, "删除 API Key 失败")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
