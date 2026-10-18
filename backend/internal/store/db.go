package store

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"remoon.net/yu/backend/migrations"
)

const migrationLockID int64 = 2026021801

// Store 封装数据库连接池与查询方法。
type Store struct {
	Pool *pgxpool.Pool
}

// querier 让同一段 SQL 既能在连接池上执行，也能在事务中执行。
type querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Open 建立连接池并验证连通性。
func Open(ctx context.Context, dsn string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("解析数据库连接串: %w", err)
	}

	cfg.MaxConns = 16
	cfg.MinConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("创建连接池: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("连接数据库: %w", err)
	}

	return &Store{Pool: pool}, nil
}

// Close 关闭连接池。
func (s *Store) Close() {
	s.Pool.Close()
}

// Migrate 按版本号顺序应用尚未执行的迁移，使用 advisory lock 防止并发执行。
func (s *Store) Migrate(ctx context.Context) error {
	conn, err := s.Pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `select pg_advisory_lock($1)`, migrationLockID); err != nil {
		return fmt.Errorf("获取迁移锁: %w", err)
	}
	defer func() {
		_, _ = conn.Exec(context.Background(), `select pg_advisory_unlock($1)`, migrationLockID)
	}()

	if _, err := conn.Exec(ctx, `
		create table if not exists schema_migrations (
			version integer primary key,
			applied_at timestamptz not null default now()
		)`); err != nil {
		return fmt.Errorf("创建迁移记录表: %w", err)
	}

	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return err
	}

	type migration struct {
		version int
		name    string
	}

	var pending []migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		prefix := strings.SplitN(entry.Name(), "_", 2)[0]
		version, err := strconv.Atoi(prefix)
		if err != nil {
			return fmt.Errorf("迁移文件名不合法: %s", entry.Name())
		}
		pending = append(pending, migration{version: version, name: entry.Name()})
	}

	sort.Slice(pending, func(i, j int) bool { return pending[i].version < pending[j].version })

	for _, m := range pending {
		var applied bool
		if err := conn.QueryRow(ctx,
			`select exists (select 1 from schema_migrations where version = $1)`, m.version,
		).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}

		body, err := migrations.FS.ReadFile(m.name)
		if err != nil {
			return err
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("执行迁移 %s: %w", m.name, err)
		}

		if _, err := tx.Exec(ctx,
			`insert into schema_migrations (version) values ($1)`, m.version,
		); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}

		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}

	return nil
}

// inTx 在事务中执行 fn，出错自动回滚。
func (s *Store) inTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// isUniqueViolation 判断错误是否为唯一约束冲突。
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// IsUniqueViolation 暴露唯一约束冲突判断给 API 层。
func IsUniqueViolation(err error) bool {
	return isUniqueViolation(err)
}
