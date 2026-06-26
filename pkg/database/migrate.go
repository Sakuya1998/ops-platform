package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

const schemaMigrationsTable = "schema_migrations"

type Migration struct {
	Version string
	Path    string
	SQL     string
}

type MigrationResult struct {
	Applied []string
	Skipped []string
}

func LogMigrationResult(serviceName, dir string, result MigrationResult) {
	log.Printf(
		"[%s] database migrations completed dir=%s applied=%d skipped=%d applied_versions=%s skipped_versions=%s",
		serviceName,
		dir,
		len(result.Applied),
		len(result.Skipped),
		strings.Join(result.Applied, ","),
		strings.Join(result.Skipped, ","),
	)
}

type migrationExecutor interface {
	Exec(ctx context.Context, sql string, args ...interface{}) error
	QueryAppliedVersions(ctx context.Context) (map[string]bool, error)
	RecordApplied(ctx context.Context, version string, appliedAt time.Time) error
	Transaction(ctx context.Context, fn func(context.Context) error) error
}

func RunMigrations(ctx context.Context, db *gorm.DB, dir string) (MigrationResult, error) {
	migrations, err := LoadMigrations(dir)
	if err != nil {
		return MigrationResult{}, err
	}
	return ApplyMigrations(ctx, gormMigrationExecutor{db: db}, migrations)
}

func ResolveMigrationsDir(candidates ...string) (string, error) {
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("migrations directory not found in candidates: %v", candidates)
}

func LoadMigrations(dir string) ([]Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}
	migrations := make([]Migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		version := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		migrations = append(migrations, Migration{
			Version: version,
			Path:    path,
			SQL:     string(data),
		})
	}
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})
	return migrations, nil
}

func ApplyMigrations(ctx context.Context, executor migrationExecutor, migrations []Migration) (MigrationResult, error) {
	result := MigrationResult{}
	if err := ensureSchemaMigrations(ctx, executor); err != nil {
		return result, err
	}
	applied, err := executor.QueryAppliedVersions(ctx)
	if err != nil {
		return result, fmt.Errorf("query applied migrations: %w", err)
	}
	for _, migration := range migrations {
		if strings.TrimSpace(migration.SQL) == "" || applied[migration.Version] {
			result.Skipped = append(result.Skipped, migration.Version)
			continue
		}
		if err := executor.Transaction(ctx, func(txCtx context.Context) error {
			if err := executor.Exec(txCtx, migration.SQL); err != nil {
				return fmt.Errorf("execute migration %s: %w", migration.Version, err)
			}
			if err := executor.RecordApplied(txCtx, migration.Version, time.Now()); err != nil {
				return fmt.Errorf("record migration %s: %w", migration.Version, err)
			}
			return nil
		}); err != nil {
			return result, err
		}
		applied[migration.Version] = true
		result.Applied = append(result.Applied, migration.Version)
	}
	return result, nil
}

func ensureSchemaMigrations(ctx context.Context, executor migrationExecutor) error {
	return executor.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL
)`)
}

type gormMigrationExecutor struct {
	db *gorm.DB
}

func (e gormMigrationExecutor) Exec(ctx context.Context, sql string, args ...interface{}) error {
	return e.dbFromContext(ctx).WithContext(ctx).Exec(sql, args...).Error
}

func (e gormMigrationExecutor) QueryAppliedVersions(ctx context.Context) (map[string]bool, error) {
	rows, err := e.dbFromContext(ctx).WithContext(ctx).Table(schemaMigrationsTable).Select("version").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := map[string]bool{}
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}
	return applied, rows.Err()
}

func (e gormMigrationExecutor) RecordApplied(ctx context.Context, version string, appliedAt time.Time) error {
	return e.dbFromContext(ctx).WithContext(ctx).Table(schemaMigrationsTable).Create(map[string]interface{}{
		"version":    version,
		"applied_at": appliedAt,
	}).Error
}

func (e gormMigrationExecutor) Transaction(ctx context.Context, fn func(context.Context) error) error {
	return e.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ctxWithTx(ctx, tx))
	})
}

func (e gormMigrationExecutor) dbFromContext(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txContextKey{}).(*gorm.DB); ok && tx != nil {
		return tx
	}
	return e.db
}

type txContextKey struct{}

func ctxWithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}
