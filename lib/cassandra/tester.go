package cassandra

import (
	"context"
	"db-bench/lib/conf"
	"fmt"
	"time"

	"github.com/gocql/gocql"
)

type CassandraTester struct {
	session *gocql.Session
	cfg     *conf.Config
}

func NewCassandraTester(ctx context.Context, cfg *conf.Config) (*CassandraTester, error) {
	cluster := gocql.NewCluster(cfg.URI)
	cluster.Keyspace = "system"
	cluster.Timeout = 20 * time.Second
	cluster.ConnectTimeout = cfg.ConnectTimeout
	var session *gocql.Session
	var err error
	for i := 0; i < 5; i++ {
		session, err = cluster.CreateSession()
		if err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		return nil, err
	}
	defer session.Close()

	err = session.Query(
		fmt.Sprintf("CREATE KEYSPACE IF NOT EXISTS %s WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}",
			cfg.DBName)).Exec()
	if err != nil {
		return nil, err
	}
	cluster.Keyspace = cfg.DBName
	finalSession, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}
	return &CassandraTester{session: finalSession, cfg: cfg}, nil
}

func (t *CassandraTester) Close() { t.session.Close() }
