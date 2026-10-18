package store

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrNotFound 表示记录不存在。
var ErrNotFound = errors.New("记录不存在")

// User 是系统用户。
type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
}

// Workspace 是资源隔离单元。
type Workspace struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

// WorkspaceWithRole 携带当前用户在该工作区中的角色。
type WorkspaceWithRole struct {
	Workspace
	Role string `json:"role"`
}

// Member 是工作区成员视图。
type Member struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// Invite 是待接受的邀请。
type Invite struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	Email       string     `json:"email"`
	Role        string     `json:"role"`
	ExpiresAt   time.Time  `json:"expires_at"`
	AcceptedAt  *time.Time `json:"accepted_at"`
	CreatedAt   time.Time  `json:"created_at"`
}
