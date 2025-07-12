package cassandra

import (
	"context"
	"db-bench/lib/conf"
	"fmt"
	"log"
	"sync"
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
	// Создание кейспейса
	err = session.Query(fmt.Sprintf("CREATE KEYSPACE IF NOT EXISTS %s WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}", cfg.DBName)).Exec()
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

func (t *CassandraTester) Seed(ctx context.Context) error {
	err := t.session.Query(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id bigint PRIMARY KEY,
		experiment_name text,
		targeting_rules text
	)`, t.cfg.TableName)).Exec()
	if err != nil {
		return fmt.Errorf("failed to create table in cassandra: %w", err)
	}
	log.Println("Cassandra: Writing rows...")
	query := fmt.Sprintf("INSERT INTO %s (id, experiment_name, targeting_rules) VALUES (?, ?, ?)", t.cfg.TableName)
	for i := 1; i <= t.cfg.RecordCount; i++ {
		rule := conf.ExperimentRule{ID: int64(i), ExperimentName: fmt.Sprintf("Test %d", i), TargetingRules: `{"country":"US"}`}
		if err := t.session.Query(query, rule.ID, rule.ExperimentName, rule.TargetingRules).Exec(); err != nil {
			log.Printf("Warning: Cassandra insert failed for key %d: %v", i, err)
		}
		if i%10000 == 0 {
			log.Printf("Cassandra: %d records inserted...", i)
		}
	}
	return nil
}

func (t *CassandraTester) RunTest(ctx context.Context, wg *sync.WaitGroup) {
	query := fmt.Sprintf("SELECT id FROM %s WHERE id = ?", t.cfg.TableName)
	for i := 0; i < t.cfg.WorkerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var idRead int64
			for {
				select {
				case <-ctx.Done():
					return
				default:
					id := int64(time.Now().UnixNano())%int64(t.cfg.RecordCount) + 1
					start := time.Now()
					err := t.session.Query(query, id).Consistency(gocql.One).Scan(&idRead)
					t.cfg.ReadLatency.WithLabelValues(t.cfg.DBName).Observe(time.Since(start).Seconds())
					if err != nil {
						t.cfg.ReadErrorsTotal.WithLabelValues(t.cfg.DBName).Inc()
					} else {
						t.cfg.ReadsTotal.WithLabelValues(t.cfg.DBName).Inc()
					}
				}
			}
		}()
	}
}

func (t *CassandraTester) Close() { t.session.Close() }
