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
	log.Printf("RunTest db %s", t.dbName)
	query := fmt.Sprintf("SELECT id FROM %s WHERE id = $1", t.tableName)
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var idRead int64
			for {
				select {
				case <-ctx.Done():
					return
				default:
					id := rand.Int63n(recordCount) + 1
					start := time.Now()
					err := t.pool.QueryRow(ctx, query, id).Scan(&idRead)
					// ИЗМЕНЕНИЕ: Используется поле dbName вместо строки "postgres"
					readLatency.WithLabelValues(t.dbName).Observe(time.Since(start).Seconds())
					if err != nil {
						readErrorsTotal.WithLabelValues(t.dbName).Inc()
					} else {
						readsTotal.WithLabelValues(t.dbName).Inc()
					}
				}
			}
		}()
	}
}
