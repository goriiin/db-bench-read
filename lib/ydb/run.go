package ydb

import (
	"context"
	"fmt"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/types"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/ydb-platform/ydb-go-sdk/v3/table"
)

func (t *YDBTester) RunTest(ctx context.Context, wg *sync.WaitGroup) {
	log.Printf("RunTest db %s", t.cfg.DBName)
	tablePath := t.getTablePath()

	query := fmt.Sprintf(`
		DECLARE $id AS Int64;
		SELECT id, experiment_name, targeting_rules 
		FROM %s 
		WHERE id = $id;
	`, "`"+tablePath+"`")

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

					err := t.db.Table().Do(ctx, func(ctx context.Context, s table.Session) error {
						_, res, err := s.Execute(ctx, table.DefaultTxControl(), query,
							table.NewQueryParameters(
								table.ValueParam("$id", types.Int64Value(id)),
							),
						)
						if err != nil {
							return err
						}
						defer res.Close()

						if res.NextResultSet(ctx) && res.NextRow() {
							// Successfully read the row
							return nil
						}
						return fmt.Errorf("no rows found")
					})

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
