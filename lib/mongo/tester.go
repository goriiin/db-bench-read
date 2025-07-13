package mongo

import (
	"context"
	"db-bench/lib/conf"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoTester struct {
	client     *mongo.Client
	collection *mongo.Collection
	cfg        *conf.Config
}

func NewMongoTester(ctx context.Context, cfg *conf.Config) (*MongoTester, error) {
	clientOptions := options.Client().ApplyURI(cfg.URI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Test connection
	err = client.Ping(ctx, nil)
	if err != nil {
		client.Disconnect(ctx)
		return nil, err
	}

	db := client.Database(cfg.DBName)
	collection := db.Collection(cfg.TableName)

	return &MongoTester{
		client:     client,
		collection: collection,
		cfg:        cfg,
	}, nil
}

func (t *MongoTester) Close() {
	if t.client != nil {
		ctx := context.Background()
		t.client.Disconnect(ctx)
	}
}
