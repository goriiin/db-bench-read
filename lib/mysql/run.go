package mysql

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

func (t *MySQLTester) RunTest(ctx context.Context, wg *sync.WaitGroup) {
	log.Printf("RunTest db %s", t.cfg.DBName)

	query := fmt.Sprintf("SELECT id FROM %s WHERE id = ?", t.cfg.TableName)

	for i := 0; i < t.cfg.WorkerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Each goroutine gets its own prepared statement
			stmt, err := t.db.PrepareContext(ctx, query)
			if err != nil {
				log.Printf("Failed to prepare statement: %v", err)
				return
			}
			defer stmt.Close()

			var idRead int64
			for {
				select {
				case <-ctx.Done():
					return
				default:
					id := rand.Int63n(int64(t.cfg.RecordCount)) + 1
					start := time.Now()
					err := stmt.QueryRowContext(ctx, id).Scan(&idRead)
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
