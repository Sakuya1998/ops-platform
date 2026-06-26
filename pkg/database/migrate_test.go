package database

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestLoadMigrationsSortsSQLFiles(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "002_second.sql"), "SELECT 2;")
	writeTestFile(t, filepath.Join(dir, "001_first.sql"), "SELECT 1;")
	writeTestFile(t, filepath.Join(dir, "README.md"), "ignore")

	migrations, err := LoadMigrations(dir)
	if err != nil {
		t.Fatalf("LoadMigrations: %v", err)
	}
	got := []string{}
	for _, migration := range migrations {
		got = append(got, migration.Version)
	}
	if !reflect.DeepEqual(got, []string{"001_first", "002_second"}) {
		t.Fatalf("unexpected migration order: %+v", got)
	}
}

func TestResolveMigrationsDirReturnsFirstExistingDirectory(t *testing.T) {
	dir := t.TempDir()
	got, err := ResolveMigrationsDir(filepath.Join(dir, "missing"), dir)
	if err != nil {
		t.Fatalf("ResolveMigrationsDir: %v", err)
	}
	if got != dir {
		t.Fatalf("expected %s, got %s", dir, got)
	}
}

func TestApplyMigrationsSkipsAppliedAndRecordsNewVersions(t *testing.T) {
	executor := &fakeMigrationExecutor{
		applied: map[string]bool{"001_init": true},
	}
	migrations := []Migration{
		{Version: "001_init", SQL: "SELECT 1;"},
		{Version: "002_add_column", SQL: "SELECT 2;"},
		{Version: "003_blank", SQL: "   "},
	}

	result, err := ApplyMigrations(context.Background(), executor, migrations)
	if err != nil {
		t.Fatalf("ApplyMigrations: %v", err)
	}
	if !reflect.DeepEqual(result.Applied, []string{"002_add_column"}) {
		t.Fatalf("unexpected applied versions: %+v", result.Applied)
	}
	if !reflect.DeepEqual(result.Skipped, []string{"001_init", "003_blank"}) {
		t.Fatalf("unexpected skipped versions: %+v", result.Skipped)
	}
	if !reflect.DeepEqual(executor.executed, []string{
		"CREATE TABLE IF NOT EXISTS schema_migrations (\n    version VARCHAR(255) PRIMARY KEY,\n    applied_at TIMESTAMPTZ NOT NULL\n)",
		"SELECT 2;",
	}) {
		t.Fatalf("unexpected executed SQL: %+v", executor.executed)
	}
	if !reflect.DeepEqual(executor.recorded, []string{"002_add_column"}) {
		t.Fatalf("unexpected recorded versions: %+v", executor.recorded)
	}
	if executor.transactions != 1 {
		t.Fatalf("expected one transaction, got %d", executor.transactions)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

type fakeMigrationExecutor struct {
	applied      map[string]bool
	executed     []string
	recorded     []string
	transactions int
}

func (e *fakeMigrationExecutor) Exec(ctx context.Context, sql string, args ...interface{}) error {
	e.executed = append(e.executed, sql)
	return nil
}

func (e *fakeMigrationExecutor) QueryAppliedVersions(ctx context.Context) (map[string]bool, error) {
	out := map[string]bool{}
	for version, applied := range e.applied {
		out[version] = applied
	}
	return out, nil
}

func (e *fakeMigrationExecutor) RecordApplied(ctx context.Context, version string, appliedAt time.Time) error {
	e.recorded = append(e.recorded, version)
	return nil
}

func (e *fakeMigrationExecutor) Transaction(ctx context.Context, fn func(context.Context) error) error {
	e.transactions++
	return fn(ctx)
}
