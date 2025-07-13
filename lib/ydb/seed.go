package ydb

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/ydb-platform/ydb-go-sdk/v3/table"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/options"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/types"
)

func (t *YDBTester) Seed(ctx context.Context) error {
	tablePath := t.getTablePath()

	// Create table
	err := t.db.Table().Do(ctx, func(ctx context.Context, s table.Session) error {
		return s.CreateTable(ctx, tablePath,
			options.WithColumn("id", types.TypeInt64),
			options.WithColumn("experiment_name", types.TypeUTF8),
			options.WithColumn("targeting_rules", types.TypeJSON),
			options.WithPrimaryKeyColumn("id"),
		)
	})
	if err != nil {
		log.Printf("Warning: Failed to create table (may already exist): %v", err)
	}

	log.Println("YDB: Writing rows...")

	// Batch size for YDB
	batchSize := 1000

	for i := 1; i <= t.cfg.RecordCount; i += batchSize {
		err := t.db.Table().Do(ctx, func(ctx context.Context, s table.Session) error {
			rows := make([]types.Value, 0, batchSize)

			for j := i; j <= i+batchSize-1 && j <= t.cfg.RecordCount; j++ {
				targetingRulesMap := map[string]interface{}{"country": "US"}
				targetingRulesJSON, err := json.Marshal(targetingRulesMap)
				if err != nil {
					return err
				}

				rows = append(rows, types.StructValue(
					types.StructFieldValue("id", types.Int64Value(int64(j))),
					types.StructFieldValue("experiment_name", types.UTF8Value(fmt.Sprintf("Test %d", j))),
					types.StructFieldValue("targeting_rules", types.JSONValue(string(targetingRulesJSON))),
				))
			}

			return s.BulkUpsert(ctx, tablePath, types.ListValue(rows...))
		})

		if err != nil {
			log.Printf("Warning: YDB bulk upsert failed for batch starting at %d: %v", i, err)
		}

		if i%10000 == 0 || i+batchSize > t.cfg.RecordCount {
			log.Printf("YDB: %d records prepared...", min(i+batchSize-1, t.cfg.RecordCount))
		}
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
