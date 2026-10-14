package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"regexp"
	"time"

	docsql "github.com/hellokitty980808-ctrl/im-core/docs/sql"

	driver "github.com/go-sql-driver/mysql"
)

// Migrate 执行新增迁移；校验已执行文件，失败可重试，不采纳未知数据库为基线。
func Migrate(ctx context.Context, service, dsn string) error {
	cfg, err := driver.ParseDSN(dsn)
	if err != nil {
		return err
	}
	database := cfg.DBName
	if !regexp.MustCompile(`^[A-Za-z0-9_]+$`).MatchString(database) {
		return fmt.Errorf("invalid database identifier")
	}
	cfg.DBName = "mysql"
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return err
	}
	defer db.Close()
	// GET_LOCK 必须与 RELEASE_LOCK 使用同一物理连接。
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	var locked int
	key := "imcore:v2:migrate:" + database
	if err = conn.QueryRowContext(ctx, "SELECT GET_LOCK(?,30)", key).Scan(&locked); err != nil {
		return err
	}
	if locked != 1 {
		return fmt.Errorf("migration lock timeout")
	}
	defer conn.ExecContext(context.Background(), "SELECT RELEASE_LOCK(?)", key)
	if _, err = conn.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS `"+database+"` CHARACTER SET utf8mb4"); err != nil {
		return err
	}
	if _, err = conn.ExecContext(ctx, "USE `"+database+"`"); err != nil {
		return err
	}
	_, err = conn.ExecContext(
		ctx,
		`CREATE TABLE IF NOT EXISTS schema_migrations (name VARCHAR(128) PRIMARY KEY, checksum CHAR(64) NOT NULL, applied_at BIGINT NOT NULL)`,
	)
	if err != nil {
		return err
	}
	files, err := docsql.ListServiceFiles(service)
	if err != nil {
		return err
	}
	for _, file := range files {
		content, err := docsql.ReadFile(file.Path)
		if err != nil {
			return err
		}
		checksum := fmt.Sprintf("%x", sha256.Sum256(content))
		var existing string
		err = conn.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE name=?", file.Name).Scan(&existing)
		if err == nil {
			if existing != checksum {
				return fmt.Errorf("migration checksum mismatch: %s", file.Name)
			}
			continue
		}
		if err != sql.ErrNoRows {
			return err
		}
		if _, err = conn.ExecContext(ctx, string(content)); err != nil {
			return fmt.Errorf("migration %s: %w", file.Name, err)
		}
		if _, err = conn.ExecContext(
			ctx,
			"INSERT INTO schema_migrations VALUES (?,?,?)",
			file.Name,
			checksum,
			time.Now().UnixMilli(),
		); err != nil {
			return err
		}
	}
	return nil
}
