package mongo

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (t *MongoTester) Seed(ctx context.Context) error {
	// Create index on id field for better read performance
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "id", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, err := t.collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		log.Printf("Warning: Failed to create index: %v", err)
	}

	log.Println("MongoDB: Writing documents...")

	// Prepare documents in batches for better performance
	batchSize := 1000
	documents := make([]interface{}, 0, batchSize)

	for i := 1; i <= t.cfg.RecordCount; i++ {
		// Create targeting rules as a map first
		targetingRulesMap := map[string]interface{}{"country": "US"}

		// Convert to JSON string if TargetingRules is defined as string
		targetingRulesJSON, err := json.Marshal(targetingRulesMap)
		if err != nil {
			return fmt.Errorf("failed to marshal targeting rules: %w", err)
		}

		doc := bson.M{
			"id":              int64(i),
			"experiment_name": fmt.Sprintf("Test %d", i),
			"targeting_rules": string(targetingRulesJSON),
		}
		documents = append(documents, doc)

		// Insert in batches
		if len(documents) == batchSize || i == t.cfg.RecordCount {
			opts := options.InsertMany().SetOrdered(false)
			_, err := t.collection.InsertMany(ctx, documents, opts)
			if err != nil {
				log.Printf("Warning: MongoDB batch insert failed: %v", err)
			}
			documents = documents[:0] // Clear slice but keep capacity
		}

		if i%10000 == 0 {
			log.Printf("MongoDB: %d records prepared...", i)
		}
	}

	return nil
}
