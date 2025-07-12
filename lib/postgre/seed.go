package postgre

import (
	"context"
	"db-bench/lib/conf"
	"fmt"
	"log"
)

func (t *PostgresTester) Seed(ctx context.Context) error {
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id BIGINT PRIMARY KEY,
			experiment_name TEXT,
			targeting_rules JSONB
		)`, t.cfg.TableName)
	if _, err := t.pool.Exec(ctx, createTableSQL); err != nil {
		return err
	}
	log.Println("Postgres: Writing rows...")
	for i := 1; i <= t.cfg.RecordCount; i++ {
		rule := conf.ExperimentRule{ID: int64(i), ExperimentName: fmt.Sprintf("Test %d", i), TargetingRules: `{"country":"US"}`}
		_, err := t.pool.Exec(ctx,
			fmt.Sprintf("INSERT INTO %s (id, experiment_name, targeting_rules) VALUES ($1, $2, $3) ON CONFLICT (id) DO NOTHING", t.cfg.TableName),
			rule.ID, rule.ExperimentName, rule.TargetingRules,
		)
		if err != nil {
			log.Printf("Warning: Postgres insert failed for key %d: %v", i, err)
		}
		if i%10000 == 0 {
			log.Printf("Postgres: %d records prepared...", i)
		}
	}
	return nil
}
