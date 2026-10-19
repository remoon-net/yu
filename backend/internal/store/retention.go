package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RetentionCell 是留存矩阵中的一个周期。
type RetentionCell struct {
	Period   int     `json:"period"`
	Visitors int64   `json:"visitors"`
	Rate     float64 `json:"rate"`
}

// RetentionCohort 是一个留存队列。
type RetentionCohort struct {
	Cohort string          `json:"cohort"`
	Size   int64           `json:"size"`
	Cells  []RetentionCell `json:"cells"`
}

// RetentionResult 是留存分析结果。
type RetentionResult struct {
	Interval string            `json:"interval"`
	Periods  int               `json:"periods"`
	Cohorts  []RetentionCohort `json:"cohorts"`
}

// Retention 计算按天或按周的访客留存。
func (s *Store) Retention(ctx context.Context, siteID uuid.UUID, from, to time.Time, interval, timezone string, filter StatsFilter) (RetentionResult, error) {
	if interval != "week" {
		interval = "day"
	}

	maxPeriods := 30
	if interval == "week" {
		maxPeriods = 12
	}

	scope, scopeArgs := filter.scopeSQL(7)
	args := append([]any{siteID, from, to, interval, timezone, maxPeriods}, scopeArgs...)
	rows, err := s.Pool.Query(ctx, `
		with first_visit as (
			select
				visitor_id,
				date_trunc($4::text, min(started_at) at time zone $5)::date as cohort
			from sessions_web
			where site_id = $1 and started_at >= $2 and started_at < $3`+scope+`
			group by visitor_id
		), activity as (
			select distinct
				visitor_id,
				date_trunc($4::text, started_at at time zone $5)::date as bucket
			from sessions_web
			where site_id = $1 and started_at >= $2 and started_at < $3`+scope+`
		)
		select
			to_char(f.cohort, 'YYYY-MM-DD') as cohort,
			((a.bucket - f.cohort) / case when $4::text = 'week' then 7 else 1 end)::int as period,
			count(distinct a.visitor_id)
		from first_visit f
		join activity a on a.visitor_id = f.visitor_id and a.bucket >= f.cohort
		group by 1, 2
		having ((a.bucket - f.cohort) / case when $4::text = 'week' then 7 else 1 end)::int <= $6::int
		order by 1, 2`,
		args...,
	)
	if err != nil {
		return RetentionResult{}, err
	}
	defer rows.Close()

	type key struct {
		cohort string
	}
	counts := make(map[key]map[int]int64)

	for rows.Next() {
		var (
			cohort string
			period int
			count  int64
		)
		if err := rows.Scan(&cohort, &period, &count); err != nil {
			return RetentionResult{}, err
		}
		k := key{cohort: cohort}
		if counts[k] == nil {
			counts[k] = make(map[int]int64)
		}
		counts[k][period] = count
	}
	if err := rows.Err(); err != nil {
		return RetentionResult{}, err
	}
	rows.Close()

	sizeScope, sizeScopeArgs := filter.scopeSQL(6)
	sizeArgs := append([]any{siteID, from, to, interval, timezone}, sizeScopeArgs...)

	sizeRows, err := s.Pool.Query(ctx, `
		with first_visit as (
			select
				visitor_id,
				date_trunc($4::text, min(started_at) at time zone $5)::date as cohort
			from sessions_web
			where site_id = $1 and started_at >= $2 and started_at < $3`+sizeScope+`
			group by visitor_id
		)
		select to_char(cohort, 'YYYY-MM-DD'), count(*)
		from first_visit
		group by 1
		order by 1`,
		sizeArgs...,
	)
	if err != nil {
		return RetentionResult{}, err
	}
	defer sizeRows.Close()

	result := RetentionResult{Interval: interval, Periods: maxPeriods, Cohorts: make([]RetentionCohort, 0, 32)}

	for sizeRows.Next() {
		var (
			cohort string
			size   int64
		)
		if err := sizeRows.Scan(&cohort, &size); err != nil {
			return RetentionResult{}, err
		}

		elapsed := elapsedPeriods(cohort, to, timezone, interval)
		if elapsed > maxPeriods {
			elapsed = maxPeriods
		}
		if elapsed < 0 {
			elapsed = 0
		}

		cohortCounts := counts[key{cohort: cohort}]
		cells := make([]RetentionCell, 0, elapsed+1)
		for period := 0; period <= elapsed; period++ {
			visitors := cohortCounts[period]
			rate := float64(0)
			if size > 0 {
				rate = float64(visitors) / float64(size) * 100
			}
			cells = append(cells, RetentionCell{Period: period, Visitors: visitors, Rate: rate})
		}

		result.Cohorts = append(result.Cohorts, RetentionCohort{
			Cohort: cohort,
			Size:   size,
			Cells:  cells,
		})
	}

	return result, sizeRows.Err()
}

func elapsedPeriods(cohort string, to time.Time, timezone, interval string) int {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	start, err := time.ParseInLocation("2006-01-02", cohort, loc)
	if err != nil {
		return 0
	}

	end := to.In(loc)
	days := int(time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC).
		Sub(time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)).Hours() / 24)
	if interval == "week" {
		return days / 7
	}
	return days
}
