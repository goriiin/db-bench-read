package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"db-bench/lib/conf"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdTester struct {
	client *clientv3.Client
	cfg    *conf.Config
}

func NewEtcdTester(ctx context.Context, cfg *conf.Config) (*EtcdTester, error) {
	client, err := clientv3.New(clientv3.Config{Endpoints: []string{cfg.URI}, DialTimeout: cfg.ConnectTimeout})
	if err != nil {
		return nil, err
	}
	return &EtcdTester{client: client, cfg: cfg}, nil
}

func (t *EtcdTester) Seed(ctx context.Context) error {
	log.Println("etcd: Writing rows...")
	for i := 1; i <= t.cfg.RecordCount; i++ {
		rule := conf.ExperimentRule{ID: int64(i), ExperimentName: fmt.Sprintf("Test %d", i), TargetingRules: `{"country":"US"}`}
		key := fmt.Sprintf("/experiments/%d", rule.ID)
		val, _ := json.Marshal(rule)
		_, err := t.client.Put(ctx, key, string(val))
		if err != nil {
			log.Printf("Warning: etcd put failed for key %s: %v", key, err)
		}
		if i%10000 == 0 {
			log.Printf("etcd: %d records inserted...", i)
		}
	}
	return nil
}

func (t *EtcdTester) RunTest(ctx context.Context, wg *sync.WaitGroup) {
	for i := 0; i < t.cfg.WorkerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					id := int64(time.Now().UnixNano())%int64(t.cfg.RecordCount) + 1
					key := fmt.Sprintf("/experiments/%d", id)
					start := time.Now()
					resp, err := t.client.Get(ctx, key)
					t.cfg.ReadLatency.WithLabelValues(t.cfg.DBName).Observe(time.Since(start).Seconds())
					if err != nil || resp.Count == 0 {
						t.cfg.ReadErrorsTotal.WithLabelValues(t.cfg.DBName).Inc()
					} else {
						t.cfg.ReadsTotal.WithLabelValues(t.cfg.DBName).Inc()
					}
				}
			}
		}()
	}
}

func (t *EtcdTester) Close() { t.client.Close() }
