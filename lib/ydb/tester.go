package ydb

import (
	"context"
	"db-bench/lib/conf"
	"fmt"

	"github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/table"
)

type YDBTester struct {
	db  *ydb.Driver
	cfg *conf.Config
}

func NewYDBTester(ctx context.Context, cfg *conf.Config) (*YDBTester, error) {
	db, err := ydb.Open(ctx, cfg.URI)
	if err != nil {
		return nil, err
	}

	return &YDBTester{
		db:  db,
		cfg: cfg,
	}, nil
}

func (t *YDBTester) Close() {
	if t.db != nil {
		ctx := context.Background()
		t.db.Close(ctx)
	}
}

func (t *YDBTester) getTablePath() string {
	return fmt.Sprintf("%s/%s", t.cfg.DBName, t.cfg.TableName)
}
