package mysql

import (
	"context"
	"database/sql"
	"db-bench/lib/conf"
	_ "github.com/go-sql-driver/mysql"
)

type MySQLTester struct {
	db  *sql.DB
	cfg *conf.Config
}

func NewMySQLTester(ctx context.Context, cfg *conf.Config) (*MySQLTester, error) {
	db, err := sql.Open("mysql", cfg.URI)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.WorkerCount + 10)
	db.SetMaxIdleConns(cfg.WorkerCount / 2)

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return &MySQLTester{db: db, cfg: cfg}, nil
}

func (t *MySQLTester) Close() {
	t.db.Close()
}
