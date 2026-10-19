package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Goal 是一个转化目标。
type Goal struct {
	ID         uuid.UUID `json:"id"`
	SiteID     uuid.UUID `json:"site_id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	MatchValue string    `json:"match_value"`
	CreatedAt  time.Time `json:"created_at"`
}

// GoalInput 是创建或更新目标的输入。
type GoalInput struct {
	SiteID     uuid.UUID
	Name       string
	Type       string
	MatchValue string
}

// GoalStat 是目标在区间内的转化统计。
type GoalStat struct {
	Goal
	Conversions    int64   `json:"conversions"`
	Visitors       int64   `json:"visitors"`
	ConversionRate float64 `json:"conversion_rate"`
}

const goalColumns = `id, site_id, name, type, match_value, created_at`

func scanGoal(row pgx.Row) (*Goal, error) {
	goal := &Goal{}
	if err := row.Scan(&goal.ID, &goal.SiteID, &goal.Name, &goal.Type, &goal.MatchValue, &goal.CreatedAt); err != nil {
		return nil, err
	}
	return goal, nil
}

// CreateGoal 创建目标。
func (s *Store) CreateGoal(ctx context.Context, in GoalInput) (*Goal, error) {
	id := uuid.New()
	if _, err := s.Pool.Exec(ctx, `
		insert into site_goals (id, site_id, name, type, match_value, created_at)
		values ($1, $2, $3, $4, $5, now())`,
		id, in.SiteID, in.Name, in.Type, in.MatchValue,
	); err != nil {
		return nil, err
	}
	return s.GetGoal(ctx, in.SiteID, id)
}

// ListGoals 返回站点下的目标。
func (s *Store) ListGoals(ctx context.Context, siteID uuid.UUID) ([]Goal, error) {
	rows, err := s.Pool.Query(ctx, `
		select `+goalColumns+`
		from site_goals
		where site_id = $1
		order by created_at`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Goal, 0, 8)
	for rows.Next() {
		goal, err := scanGoal(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *goal)
	}
	return list, rows.Err()
}

// GetGoal 按站点与 ID 查询目标。
func (s *Store) GetGoal(ctx context.Context, siteID, goalID uuid.UUID) (*Goal, error) {
	return scanGoal(s.Pool.QueryRow(ctx, `
		select `+goalColumns+`
		from site_goals
		where id = $1 and site_id = $2`, goalID, siteID))
}

// UpdateGoal 更新目标。
func (s *Store) UpdateGoal(ctx context.Context, goalID uuid.UUID, in GoalInput) (*Goal, error) {
	tag, err := s.Pool.Exec(ctx, `
		update site_goals set name = $2, type = $3, match_value = $4
		where id = $1 and site_id = $5`,
		goalID, in.Name, in.Type, in.MatchValue, in.SiteID,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.GetGoal(ctx, in.SiteID, goalID)
}

// DeleteGoal 删除目标。
func (s *Store) DeleteGoal(ctx context.Context, siteID, goalID uuid.UUID) error {
	tag, err := s.Pool.Exec(ctx, `
		delete from site_goals where id = $1 and site_id = $2`, goalID, siteID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GoalStats 返回站点目标的转化统计。
func (s *Store) GoalStats(ctx context.Context, siteID uuid.UUID, from, to time.Time, filter StatsFilter) ([]GoalStat, error) {
	scope, scopeArgs := filter.scopeSQL(4)
	args := append([]any{siteID, from, to}, scopeArgs...)

	var uv int64
	if err := s.Pool.QueryRow(ctx, `
		select count(distinct visitor_id)
		from events
		where site_id = $1 and type = 'pageview' and created_at >= $2 and created_at < $3`+scope,
		args...,
	).Scan(&uv); err != nil {
		return nil, err
	}

	scope, scopeArgs = filter.scopeSQLFor("e", 4)
	args = append([]any{siteID, from, to}, scopeArgs...)

	rows, err := s.Pool.Query(ctx, `
		select
			g.id, g.site_id, g.name, g.type, g.match_value, g.created_at,
			count(e.id) as conversions,
			count(distinct e.visitor_id) as visitors
		from site_goals g
		left join events e
			on e.site_id = g.site_id
			and e.created_at >= $2 and e.created_at < $3
			and (
				(g.type = 'event' and e.type = 'custom' and e.name = g.match_value)
				or (g.type = 'page' and e.type = 'pageview' and e.path = g.match_value)
			)`+scope+`
		where g.site_id = $1
		group by g.id, g.site_id, g.name, g.type, g.match_value, g.created_at
		order by g.created_at`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]GoalStat, 0, 8)
	for rows.Next() {
		var item GoalStat
		if err := rows.Scan(
			&item.ID, &item.SiteID, &item.Name, &item.Type, &item.MatchValue, &item.CreatedAt,
			&item.Conversions, &item.Visitors,
		); err != nil {
			return nil, err
		}
		if uv > 0 {
			item.ConversionRate = float64(item.Visitors) / float64(uv) * 100
		}
		list = append(list, item)
	}

	return list, rows.Err()
}
