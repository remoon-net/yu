package store

import (
	"context"
	"fmt"
	"time"
)

// EnsureEventPartitions 确保未来 monthsAhead 个月的 events 分区存在。
func (s *Store) EnsureEventPartitions(ctx context.Context, monthsAhead int) error {
	now := time.Now().UTC()
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i <= monthsAhead; i++ {
		start := first.AddDate(0, i, 0)
		end := start.AddDate(0, 1, 0)
		table := "events_" + start.Format("200601")

		stmt := fmt.Sprintf(
			`create table if not exists %s partition of events for values from ('%s') to ('%s')`,
			table, start.Format("2006-01-02"), end.Format("2006-01-02"),
		)

		if _, err := s.Pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("创建分区 %s: %w", table, err)
		}
	}

	return nil
}
