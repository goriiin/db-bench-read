package postgre

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

func (t *PostgresTester) RunTest(ctx context.Context, wg *sync.WaitGroup) {
	log.Printf("RunTest db %s", t.cfg.DBName)
	query := fmt.Sprintf("SELECT id FROM %s WHERE id = $1", t.cfg.TableName)
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
					id := rand.Int63n(int64(t.cfg.RecordCount)) + 1
					start := time.Now()
					err := t.pool.QueryRow(ctx, query, id).Scan(&idRead)
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
