package mongo

import (
	"context"
	"db-bench/lib/conf"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
)

func SeedMongo(ctx context.Context, coll *mongo.Collection, recordCount int) error {
	log.Println("MongoDB: Writing rows...")
	var models []mongo.WriteModel
	for i := 1; i <= recordCount; i++ {
		rule := conf.ExperimentRule{ID: int64(i), ExperimentName: fmt.Sprintf("Test %d", i), TargetingRules: `{"country":"US"}`}
		models = append(models, mongo.NewInsertOneModel().SetDocument(rule))
		if i%10000 == 0 || i == recordCount {
			_, err := coll.BulkWrite(ctx, models)
			if err != nil {
				log.Printf("Warning: Mongo bulk write failed: %v", err)
			}
			log.Printf("MongoDB: %d records prepared...", i)
			models = nil // reset batch
		}
	}
	return nil
}
