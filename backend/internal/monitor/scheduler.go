package monitor

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"remoon.net/yu/backend/internal/monitor/probe"
	"remoon.net/yu/backend/internal/store"
)

const schedulerLockID int64 = 2026021802

// Scheduler 负责按周期执行监控探测。
type Scheduler struct {
	store       *store.Store
	concurrency int
	evaluator   *Evaluator
}

// NewScheduler 创建调度器。
func NewScheduler(st *store.Store, concurrency int, sender *Sender) *Scheduler {
	if concurrency <= 0 {
		concurrency = 16
	}
	return &Scheduler{store: st, concurrency: concurrency, evaluator: NewEvaluator(st, sender)}
}

// Run 获取调度锁后开始执行，退出时释放锁。
func (s *Scheduler) Run(ctx context.Context) {
	conn, err := s.acquireLock(ctx)
	if err != nil {
		if ctx.Err() == nil {
			slog.Error("获取调度锁失败", "err", err)
		}
		return
	}

	defer func() {
		if _, err := conn.Exec(context.Background(), `select pg_advisory_unlock($1)`, schedulerLockID); err != nil {
			slog.Warn("释放调度锁失败", "err", err)
		}
		conn.Release()
	}()

	slog.Info("监控调度器已启动", "concurrency", s.concurrency)

	sem := make(chan struct{}, s.concurrency)
	dispatchTicker := time.NewTicker(2 * time.Second)
	aggregateTicker := time.NewTicker(10 * time.Minute)
	defer dispatchTicker.Stop()
	defer aggregateTicker.Stop()

	s.aggregate(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-dispatchTicker.C:
			s.dispatch(ctx, sem)
		case <-aggregateTicker.C:
			s.aggregate(ctx)
		}
	}
}

func (s *Scheduler) acquireLock(ctx context.Context) (*pgxpool.Conn, error) {
	for {
		conn, err := s.store.Pool.Acquire(ctx)
		if err == nil {
			var acquired bool
			scanErr := conn.QueryRow(ctx, `select pg_try_advisory_lock($1)`, schedulerLockID).Scan(&acquired)
			if scanErr == nil && acquired {
				return conn, nil
			}
			conn.Release()
			if scanErr != nil {
				return nil, scanErr
			}
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Second):
		}
	}
}

func (s *Scheduler) dispatch(ctx context.Context, sem chan struct{}) {
	monitors, err := s.store.ListDueMonitors(ctx, 50)
	if err != nil {
		slog.Error("查询待执行监控失败", "err", err)
		return
	}

	for _, item := range monitors {
		next := time.Now().Add(time.Duration(item.IntervalSec) * time.Second)
		if err := s.store.SetNextRun(ctx, item.ID, next); err != nil {
			slog.Warn("更新下次执行时间失败", "err", err, "monitor", item.ID)
			continue
		}

		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			return
		}

		go func(m store.Monitor) {
			defer func() { <-sem }()
			defer func() {
				if recovered := recover(); recovered != nil {
					slog.Error("探测执行发生 panic", "monitor", m.ID, "panic", recovered)
				}
			}()
			s.runOne(ctx, m)
		}(item)
	}
}

func (s *Scheduler) runOne(ctx context.Context, m store.Monitor) {
	timeout := time.Duration(m.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	attempts := m.Retries + 1
	var result probe.Result

	for attempt := 0; attempt < attempts; attempt++ {
		result = probe.Run(ctx, m.Type, m.Config, timeout)
		if result.Status == "up" {
			break
		}

		if attempt < attempts-1 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
		}
	}

	checkedAt := time.Now().UTC()
	next := checkedAt.Add(time.Duration(m.IntervalSec) * time.Second)

	err := s.store.RecordCheck(ctx, m.ID, checkedAt, store.CheckResult{
		Status:        result.Status,
		LatencyMS:     result.LatencyMS,
		StatusCode:    result.StatusCode,
		Error:         result.Error,
		CertExpiresAt: result.CertExpiresAt,
	}, next)
	if err != nil {
		slog.Error("写入探测结果失败", "err", err, "monitor", m.ID)
		return
	}

	s.evaluator.Process(ctx, m, result, checkedAt)
}

func (s *Scheduler) aggregate(ctx context.Context) {
	now := time.Now().UTC().Truncate(time.Hour)
	from := now.Add(-3 * time.Hour)

	if err := s.store.AggregateMonitorHourly(ctx, from, now.Add(time.Hour)); err != nil {
		slog.Error("聚合监控小时统计失败", "err", err)
	}
}
