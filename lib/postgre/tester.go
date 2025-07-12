package postgre

import (
	"context"
	"db-bench/lib/conf"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresTester struct {
	pool *pgxpool.Pool
	conf conf.Config
}

func NewPostgresTester(ctx context.Context, uri string, dbName string) (*PostgresTester, error) {
	pool, err := pgxpool.New(ctx, uri)
	if err != nil {
		return nil, err
	}

	return &PostgresTester{pool: pool, dbName: dbName}, nil
}
