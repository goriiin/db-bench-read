package cassandra

import (
	"context"
	"db-bench/lib/conf"
	"fmt"
	"log"
)

func (t *CassandraTester) Seed(_ context.Context) error {
	err := t.session.Query(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id bigint PRIMARY KEY,
		experiment_name text,
		targeting_rules text
	)`, t.cfg.TableName)).Exec()
	if err != nil {
		return fmt.Errorf("failed to create table in cassandra: %w", err)
	}
	log.Println("Cassandra: Writing rows...")
	query := fmt.Sprintf("INSERT INTO %s (id, experiment_name, targeting_rules) VALUES (?, ?, ?)", t.cfg.TableName)
	for i := 1; i <= t.cfg.RecordCount; i++ {
		rule := conf.ExperimentRule{ID: int64(i), ExperimentName: fmt.Sprintf("Test %d", i), TargetingRules: `{"country":"US"}`}
		if err := t.session.Query(query, rule.ID, rule.ExperimentName, rule.TargetingRules).Exec(); err != nil {
			log.Printf("Warning: Cassandra insert failed for key %d: %v", i, err)
		}
		if i%10000 == 0 {
			log.Printf("Cassandra: %d records inserted...", i)
		}
	}
	return nil
}
