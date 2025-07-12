package mongo

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"db-bench/lib/conf"

	"github.com/prometheus/client_golang/prometheus"
)

func RunMongoRead(ctx context.Context, coll *mongo.Collection, dbName string, recordCount, workerCount int, readsTotal, readErrorsTotal *prometheus.CounterVec, readLatency *prometheus.HistogramVec) {
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var result conf.ExperimentRule
			for {
				select {
				case <-ctx.Done():
					return
				default:
					id := rand.Int63n(int64(recordCount)) + 1
					filter := bson.M{"_id": id}
					start := time.Now()
					err := coll.FindOne(ctx, filter).Decode(&result)
					readLatency.WithLabelValues(dbName).Observe(time.Since(start).Seconds())
					if err != nil {
						readErrorsTotal.WithLabelValues(dbName).Inc()
					} else {
						readsTotal.WithLabelValues(dbName).Inc()
					}
				}
			}
		}()
	}
	wg.Wait()
}
