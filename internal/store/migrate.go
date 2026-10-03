package store

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"finguard.ai/migrations"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
		  version text PRIMARY KEY,
		  applied_at timestamptz NOT NULL DEFAULT now()
		)`); err != nil {
		return err
	}
	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		return err
	}
	var ups []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".sql") && !strings.HasSuffix(name, "_down.sql") {
			ups = append(ups, name)
		}
	}
	sort.Strings(ups)
	for _, name := range ups {
		version := strings.TrimSuffix(name, ".sql")
		var one int
		err := pool.QueryRow(ctx, `SELECT 1 FROM schema_migrations WHERE version = $1`, version).Scan(&one)
		if err == nil {
			continue
		}
		if err != pgx.ErrNoRows {
			return err
		}
		body, err := migrations.FS.ReadFile(name)
		if err != nil {
			return err
		}
		script := "BEGIN;\n" + string(body) + fmt.Sprintf("\nINSERT INTO schema_migrations (version) VALUES ('%s');\nCOMMIT;\n", version)
		if err := execScript(ctx, pool, script); err != nil {
			return fmt.Errorf("migrate %s: %w", version, err)
		}
	}
	return nil
}

func RollbackLatest(ctx context.Context, pool *pgxpool.Pool) error {
	var version string
	err := pool.QueryRow(ctx, `SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1`).Scan(&version)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	body, err := migrations.FS.ReadFile(version + "_down.sql")
	if err != nil {
		return fmt.Errorf("down migration %s: %w", version, err)
	}
	script := "BEGIN;\n" + string(body) + fmt.Sprintf("\nDELETE FROM schema_migrations WHERE version = '%s';\nCOMMIT;\n", version)
	if err := execScript(ctx, pool, script); err != nil {
		return err
	}
	return nil
}

func execScript(ctx context.Context, pool *pgxpool.Pool, script string) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	results, err := conn.Conn().PgConn().Exec(ctx, script).ReadAll()
	if err != nil {
		_, _ = conn.Conn().PgConn().Exec(ctx, "ROLLBACK").ReadAll()
		return err
	}
	_ = results
	return nil
}

func WithDatabase(raw, name string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	u.Path = "/" + name
	return u.String(), nil
}

func EnsureAppRole(ctx context.Context, adminURL string) error {
	conn, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	if _, err = conn.Exec(ctx, `DO $$ BEGIN
		CREATE ROLE finguard_app LOGIN PASSWORD 'finguard_app';
	EXCEPTION WHEN duplicate_object THEN NULL; END $$;`); err != nil {
		return err
	}
	u, err := url.Parse(adminURL)
	if err != nil {
		return err
	}
	name := strings.Trim(u.Path, "/")
	if name == "" {
		return nil
	}
	_, err = conn.Exec(ctx, `GRANT CONNECT ON DATABASE `+pgx.Identifier{name}.Sanitize()+` TO finguard_app`)
	return err
}

func EnsureDatabase(ctx context.Context, adminURL, name string) error {
	conn, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	var exists int
	err = conn.QueryRow(ctx, `SELECT 1 FROM pg_database WHERE datname = $1`, name).Scan(&exists)
	if err == pgx.ErrNoRows {
		if _, err = conn.Exec(ctx, `CREATE DATABASE `+pgx.Identifier{name}.Sanitize()); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	_, err = conn.Exec(ctx, `GRANT CONNECT ON DATABASE `+pgx.Identifier{name}.Sanitize()+` TO finguard_app`)
	return err
}

func TableExists(ctx context.Context, pool *pgxpool.Pool, name string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1)`, name).Scan(&exists)
	return exists, err
}
