package lib

import (
	"context"
	"db-bench/lib/cassandra"
	"db-bench/lib/conf"
	"db-bench/lib/etcd"
	"db-bench/lib/mongo"
	"db-bench/lib/postgre"
	"fmt"
	"sync"
)

type DatabaseTester interface {
	Seed(ctx context.Context) error
	RunTest(ctx context.Context, wg *sync.WaitGroup)
	Close()
}

func GetTester(dbType string, cfg *conf.Config) (DatabaseTester, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	switch dbType {
	case "postgres":
		return postgre.NewPostgresTester(ctx, cfg)
	case "cassandra":
		return cassandra.NewCassandraTester(ctx, cfg)
	case "mongo":
		return mongo.NewMongoTester(ctx, cfg)
	case "etcd":
		return etcd.NewEtcdTester(ctx, cfg)
	default:
		return nil, fmt.Errorf("unknown database type: %s", dbType)
	}
}
