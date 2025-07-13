package mysql

import (
	"context"
	"db-bench/lib/conf"
	"fmt"
	"log"
)

func (t *MySQLTester) Seed(ctx context.Context) error {
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id BIGINT PRIMARY KEY,
			experiment_name TEXT,
			targeting_rules JSON
		)`, t.cfg.TableName)

	if _, err := t.db.ExecContext(ctx, createTableSQL); err != nil {
		return err
	}

	log.Println("MySQL: Writing rows...")

	// Prepare statement for better performance
	stmt, err := t.db.PrepareContext(ctx,
		fmt.Sprintf("INSERT IGNORE INTO %s (id, experiment_name, targeting_rules) VALUES (?, ?, ?)", t.cfg.TableName))
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i := 1; i <= t.cfg.RecordCount; i++ {
		rule := conf.ExperimentRule{
			ID:             int64(i),
			ExperimentName: fmt.Sprintf("Test %d", i),
			TargetingRules: `{"country":"US"}`,
		}

		_, err := stmt.ExecContext(ctx, rule.ID, rule.ExperimentName, rule.TargetingRules)
		if err != nil {
			log.Printf("Warning: MySQL insert failed for key %d: %v", i, err)
		}

		if i%10000 == 0 {
			log.Printf("MySQL: %d records prepared...", i)
		}
	}

	return nil
}
