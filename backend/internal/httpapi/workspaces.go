package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"remoon.net/yu/backend/internal/auth"
	"remoon.net/yu/backend/internal/mailer"
	"remoon.net/yu/backend/internal/monitor"
	"remoon.net/yu/backend/internal/store"
)

var workspaceSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,38}[a-z0-9]$`)

var roleRank = map[string]int{"member": 1, "admin": 2, "owner": 3}

func roleAtLeast(role, min string) bool {
	return roleRank[role] >= roleRank[min]
}

func validRole(role string) bool {
	_, ok := roleRank[role]
	return ok
}

// workspaceAccess 校验当前用户对路径中工作区的权限。
func (s *Server) workspaceAccess(w http.ResponseWriter, r *http.Request, minRole string) (*store.Workspace, string, bool) {
	wsID, err := uuid.Parse(chi.URLParam(r, "wsID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "工作区不存在")
		return nil, "", false
	}

	ws, err := s.store.GetWorkspace(r.Context(), wsID)
	if err != nil {
		writeError(w, http.StatusNotFound, "工作区不存在")
		return nil, "", false
	}

	user := userFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "未登录或会话已过期")
		return nil, "", false
	}

	role, err := s.store.MemberRole(r.Context(), wsID, user.ID)
	if err != nil {
		writeError(w, http.StatusNotFound, "工作区不存在")
		return nil, "", false
	}

	if !roleAtLeast(role, minRole) {
		writeError(w, http.StatusForbidden, "没有权限执行该操作")
		return nil, "", false
	}

	return ws, role, true
}

func (s *Server) handleListWorkspaces(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "未登录或会话已过期")
		return
	}

	list, err := s.store.ListWorkspacesByUser(r.Context(), user.ID)
	if err != nil {
		slog.Error("查询工作区失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取工作区失败")
		return
	}

	writeJSON(w, http.StatusOK, list)
}

type createWorkspaceRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (s *Server) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	var req createWorkspaceRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	name := trimSpace(req.Name)
	if name == "" || len([]rune(name)) > 60 {
		writeError(w, http.StatusBadRequest, "工作区名称需要 1-60 个字符")
		return
	}

	slug := strings.ToLower(trimSpace(req.Slug))
	if slug != "" && !workspaceSlugPattern.MatchString(slug) {
		writeError(w, http.StatusBadRequest, "工作区标识只能包含小写字母、数字与连字符，长度 2-40")
		return
	}

	user := userFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "未登录或会话已过期")
		return
	}

	ws, err := s.store.CreateWorkspace(r.Context(), name, slug, user.ID)
	if err != nil {
		if errors.Is(err, store.ErrSlugTaken) {
			writeError(w, http.StatusConflict, "工作区标识已被占用")
			return
		}
		slog.Error("创建工作区失败", "err", err)
		writeError(w, http.StatusInternalServerError, "创建工作区失败")
		return
	}

	writeJSON(w, http.StatusCreated, ws)
}

func (s *Server) handleDeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	ws, _, ok := s.workspaceAccess(w, r, "owner")
	if !ok {
		return
	}

	if err := s.store.DeleteWorkspace(r.Context(), ws.ID); err != nil {
		slog.Error("删除工作区失败", "err", err)
		writeError(w, http.StatusInternalServerError, "删除工作区失败")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListMembers(w http.ResponseWriter, r *http.Request) {
	ws, _, ok := s.workspaceAccess(w, r, "member")
	if !ok {
		return
	}

	members, err := s.store.ListMembers(r.Context(), ws.ID)
	if err != nil {
		slog.Error("查询成员失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取成员失败")
		return
	}

	writeJSON(w, http.StatusOK, members)
}

type updateMemberRoleRequest struct {
	Role string `json:"role"`
}

func (s *Server) handleUpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	ws, requesterRole, ok := s.workspaceAccess(w, r, "admin")
	if !ok {
		return
	}

	targetID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "成员不存在")
		return
	}

	var req updateMemberRoleRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if !validRole(req.Role) {
		writeError(w, http.StatusBadRequest, "角色不合法")
		return
	}

	if req.Role == "owner" && requesterRole != "owner" {
		writeError(w, http.StatusForbidden, "只有 owner 可以授予 owner 角色")
		return
	}

	targetRole, err := s.store.MemberRole(r.Context(), ws.ID, targetID)
	if err != nil {
		writeError(w, http.StatusNotFound, "成员不存在")
		return
	}

	if targetRole == "owner" && req.Role != "owner" {
		owners, err := s.store.CountOwners(r.Context(), ws.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "读取成员失败")
			return
		}
		if owners <= 1 {
			writeError(w, http.StatusConflict, "不能降级最后一个 owner")
			return
		}
	}

	if err := s.store.SetMemberRole(r.Context(), ws.ID, targetID, req.Role); err != nil {
		slog.Error("修改成员角色失败", "err", err)
		writeError(w, http.StatusInternalServerError, "修改成员角色失败")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	ws, requesterRole, ok := s.workspaceAccess(w, r, "member")
	if !ok {
		return
	}

	targetID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "成员不存在")
		return
	}

	requester := userFromContext(r.Context())
	isSelf := requester != nil && requester.ID == targetID

	if !isSelf && !roleAtLeast(requesterRole, "admin") {
		writeError(w, http.StatusForbidden, "没有权限移除该成员")
		return
	}

	targetRole, err := s.store.MemberRole(r.Context(), ws.ID, targetID)
	if err != nil {
		writeError(w, http.StatusNotFound, "成员不存在")
		return
	}

	if targetRole == "owner" {
		if requesterRole != "owner" {
			writeError(w, http.StatusForbidden, "只有 owner 可以移除 owner")
			return
		}

		owners, err := s.store.CountOwners(r.Context(), ws.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "读取成员失败")
			return
		}
		if owners <= 1 {
			writeError(w, http.StatusConflict, "不能移除最后一个 owner")
			return
		}
	}

	if err := s.store.RemoveMember(r.Context(), ws.ID, targetID); err != nil {
		slog.Error("移除成员失败", "err", err)
		writeError(w, http.StatusInternalServerError, "移除成员失败")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListInvites(w http.ResponseWriter, r *http.Request) {
	ws, _, ok := s.workspaceAccess(w, r, "admin")
	if !ok {
		return
	}

	invites, err := s.store.ListPendingInvites(r.Context(), ws.ID)
	if err != nil {
		slog.Error("查询邀请失败", "err", err)
		writeError(w, http.StatusInternalServerError, "读取邀请失败")
		return
	}

	writeJSON(w, http.StatusOK, invites)
}

type createInviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (s *Server) handleCreateInvite(w http.ResponseWriter, r *http.Request) {
	ws, requesterRole, ok := s.workspaceAccess(w, r, "admin")
	if !ok {
		return
	}

	var req createInviteRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	addr, err := mail.ParseAddress(trimSpace(req.Email))
	if err != nil {
		writeError(w, http.StatusBadRequest, "邮箱格式不正确")
		return
	}

	role := trimSpace(req.Role)
	if role == "" {
		role = "member"
	}
	if !validRole(role) {
		writeError(w, http.StatusBadRequest, "角色不合法")
		return
	}
	if role == "owner" && requesterRole != "owner" {
		writeError(w, http.StatusForbidden, "只有 owner 可以授予 owner 角色")
		return
	}

	token, err := auth.NewToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建邀请失败")
		return
	}

	requester := userFromContext(r.Context())
	invite, err := s.store.CreateInvite(
		r.Context(), ws.ID, addr.Address, role, auth.HashToken(token), requester.ID,
		time.Now().Add(7*24*time.Hour),
	)
	if err != nil {
		slog.Error("创建邀请失败", "err", err)
		writeError(w, http.StatusInternalServerError, "创建邀请失败")
		return
	}

	link := s.cfg.PublicBaseURL + "/invite/" + token + "/"
	emailSent := s.sendInviteEmail(r.Context(), ws.ID, invite.Email, ws.Name, link)

	writeJSON(w, http.StatusCreated, map[string]any{
		"invite_url": link,
		"email":      invite.Email,
		"role":       invite.Role,
		"expires_at": invite.ExpiresAt,
		"email_sent": emailSent,
	})
}

func (s *Server) sendInviteEmail(ctx context.Context, workspaceID uuid.UUID, to, workspaceName, link string) bool {
	channels, err := s.store.ListChannels(ctx, workspaceID)
	if err != nil {
		slog.Warn("读取通知渠道失败", "err", err, "workspace", workspaceID)
		return false
	}

	var smtpOptions mailer.Options
	found := false
	for _, channel := range channels {
		if channel.Type != "smtp" || !channel.Enabled {
			continue
		}
		opts, err := monitor.SMTPOptions(channel.Config)
		if err != nil {
			slog.Warn("跳过不可用的邮件渠道", "err", err, "channel", channel.ID)
			continue
		}
		smtpOptions = opts
		found = true
		break
	}

	if !found {
		return false
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	subject := "邀请你加入工作区 " + workspaceName
	body := "你被邀请加入 Yu 工作区「" + workspaceName + "」。\n\n" +
		"点击下面的链接设置密码并加入：\n\n" + link + "\n\n链接 7 天内有效。"

	if err := s.mail.Send(ctx, smtpOptions, to, subject, body); err != nil {
		slog.Warn("发送邀请邮件失败", "err", err, "to", to)
		return false
	}

	return true
}
