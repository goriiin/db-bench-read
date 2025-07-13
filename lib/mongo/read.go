package mongo

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

func (t *MongoTester) RunTest(ctx context.Context, wg *sync.WaitGroup) {
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

					start := time.Now()
					var result bson.M
					err := t.collection.FindOne(ctx, bson.M{"id": id}).Decode(&result)
					duration := time.Since(start).Seconds()
					t.cfg.ReadLatency.WithLabelValues(t.cfg.DB).Observe(duration)

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
