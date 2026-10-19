package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FunnelStep 是漏斗的一步。
type FunnelStep struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// FunnelVisitors 计算每一步的访客数，要求后一步发生在前一步之后。
func (s *Store) FunnelVisitors(ctx context.Context, siteID uuid.UUID, from, to time.Time, steps []FunnelStep, filter StatsFilter) ([]int64, error) {
	if len(steps) == 0 {
		return nil, nil
	}

	args := make([]any, 0, len(steps)+9)
	args = append(args, siteID, from, to)
	for _, step := range steps {
		args = append(args, step.Value)
	}

	scope, scopeArgs := filter.scopeSQLFor("e", len(steps)+4)
	args = append(args, scopeArgs...)

	ctes := make([]string, 0, len(steps))
	selects := make([]string, 0, len(steps))

	predicate := func(step FunnelStep, index int) string {
		if step.Type == "event" {
			return fmt.Sprintf("e.type = 'custom' and e.name = $%d", index)
		}
		return fmt.Sprintf("e.type = 'pageview' and e.path = $%d", index)
	}

	for i, step := range steps {
		argIndex := i + 4

		if i == 0 {
			ctes = append(ctes, fmt.Sprintf(`step1 as (
				select e.visitor_id, min(e.created_at) as at
				from events e
				where e.site_id = $1 and e.created_at >= $2 and e.created_at < $3 and %s%s
				group by e.visitor_id)`, predicate(step, argIndex), scope))
		} else {
			ctes = append(ctes, fmt.Sprintf(`step%d as (
				select e.visitor_id, min(e.created_at) as at
				from events e
				join step%d p on p.visitor_id = e.visitor_id
				where e.site_id = $1 and e.created_at >= p.at and e.created_at < $3 and %s%s
				group by e.visitor_id)`, i+1, i, predicate(step, argIndex), scope))
		}

		selects = append(selects, fmt.Sprintf("(select count(*) from step%d)", i+1))
	}

	query := "with " + strings.Join(ctes, ",\n") + " select " + strings.Join(selects, ", ")

	counts := make([]int64, len(steps))
	dest := make([]any, len(steps))
	for i := range counts {
		dest[i] = &counts[i]
	}

	if err := s.Pool.QueryRow(ctx, query, args...).Scan(dest...); err != nil {
		return nil, err
	}

	return counts, nil
}
