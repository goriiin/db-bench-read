package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"db-bench/lib/conf"
)

func (t *EtcdTester) RunTest(ctx context.Context, wg *sync.WaitGroup) {
	log.Printf("RunTest db %s", t.cfg.DBName)

	for i := 0; i < t.cfg.WorkerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					id := rand.Int63n(int64(t.cfg.RecordCount)) + 1
					key := fmt.Sprintf("/%s/%d", t.cfg.TableName, id)

					start := time.Now()
					resp, err := t.client.Get(ctx, key)
					duration := time.Since(start).Seconds()
					t.cfg.ReadLatency.WithLabelValues(t.cfg.DB).Observe(duration)

					if err != nil {
						t.cfg.ReadErrorsTotal.WithLabelValues(t.cfg.DB).Inc()
						continue
					}

					if len(resp.Kvs) == 0 {
						t.cfg.ReadErrorsTotal.WithLabelValues(t.cfg.DB).Inc()
						continue
					}

					// Optionally validate the data
					var rule conf.ExperimentRule
					if err := json.Unmarshal(resp.Kvs[0].Value, &rule); err != nil {
						t.cfg.ReadErrorsTotal.WithLabelValues(t.cfg.DB).Inc()
					} else {
						t.cfg.ReadsTotal.WithLabelValues(t.cfg.DB).Inc()
					}
				}
			}
		}()
	}
}
