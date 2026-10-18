package store

import (
	"context"
	"fmt"
	"time"
)

var partitionTables = map[string]bool{
	"events":         true,
	"monitor_checks": true,
}

// EnsureEventPartitions 确保未来 monthsAhead 个月的 events 分区存在。
func (s *Store) EnsureEventPartitions(ctx context.Context, monthsAhead int) error {
	return s.EnsureMonthlyPartitions(ctx, "events", monthsAhead)
}

// EnsureMonitorCheckPartitions 确保未来 monthsAhead 个月的 monitor_checks 分区存在。
func (s *Store) EnsureMonitorCheckPartitions(ctx context.Context, monthsAhead int) error {
	return s.EnsureMonthlyPartitions(ctx, "monitor_checks", monthsAhead)
}

// EnsureMonthlyPartitions 为按月分区的表创建未来若干个月的分区。
func (s *Store) EnsureMonthlyPartitions(ctx context.Context, table string, monthsAhead int) error {
	if !partitionTables[table] {
		return fmt.Errorf("不支持的分区表: %s", table)
	}

	now := time.Now().UTC()
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i <= monthsAhead; i++ {
		start := first.AddDate(0, i, 0)
		end := start.AddDate(0, 1, 0)
		name := table + "_" + start.Format("200601")

		stmt := fmt.Sprintf(
			`create table if not exists %s partition of %s for values from ('%s') to ('%s')`,
			name, table, start.Format("2006-01-02"), end.Format("2006-01-02"),
		)

		if _, err := s.Pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("创建分区 %s: %w", name, err)
		}
	}

	return nil
}
