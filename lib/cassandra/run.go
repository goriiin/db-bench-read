package cassandra

import (
	"context"
	"fmt"
	"github.com/gocql/gocql"
	"sync"
	"time"
)

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
					t.cfg.ReadLatency.WithLabelValues(t.cfg.DB).Observe(time.Since(start).Seconds())
					if err != nil {
						t.cfg.ReadErrorsTotal.WithLabelValues(t.cfg.DB).Inc()
					} else {
						t.cfg.ReadsTotal.WithLabelValues(t.cfg.DB).Inc()
					}
				}
			}
		}()
	}
}
