package postgre

import (
	"context"
	"db-bench/lib/conf"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresTester struct {
	pool *pgxpool.Pool
	cfg  *conf.Config
}

func NewPostgresTester(ctx context.Context, cfg *conf.Config) (*PostgresTester, error) {
	pool, err := pgxpool.New(ctx, cfg.URI)
	if err != nil {
		return nil, err
	}
	return &PostgresTester{pool: pool, cfg: cfg}, nil
}

func (t *PostgresTester) Close() { t.pool.Close() }
