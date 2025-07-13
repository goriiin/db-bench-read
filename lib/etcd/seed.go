package etcd

import (
	"context"
	"db-bench/lib/conf"
	"encoding/json"
	"fmt"
	"log"
)

func (t *EtcdTester) Seed(ctx context.Context) error {
	log.Println("Etcd: Writing keys...")

	for i := 1; i <= t.cfg.RecordCount; i++ {
		targetingRulesMap := map[string]interface{}{"country": "US"}

		targetingRulesJSON, err := json.Marshal(targetingRulesMap)
		if err != nil {
			return fmt.Errorf("failed to marshal targeting rules: %w", err)
		}

		rule := conf.ExperimentRule{
			ID:             int64(i),
			ExperimentName: fmt.Sprintf("Test %d", i),
			TargetingRules: string(targetingRulesJSON),
		}

		value, err := json.Marshal(rule)
		if err != nil {
			return fmt.Errorf("failed to marshal rule %d: %w", i, err)
		}

		key := fmt.Sprintf("/%s/%d", t.cfg.TableName, i)
		_, err = t.client.Put(ctx, key, string(value))
		if err != nil {
			log.Printf("Warning: Etcd put failed for key %s: %v", key, err)
		}

		if i%10000 == 0 {
			log.Printf("Etcd: %d records prepared...", i)
		}
	}

	return nil
}
