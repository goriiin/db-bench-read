package mongo

import (
	"context"
	"db-bench/lib/conf"
	"sync"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoTester struct {
	coll   *mongo.Collection
	client *mongo.Client
	cfg    *conf.Config
}

func NewMongoTester(ctx context.Context, cfg *conf.Config) (*MongoTester, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI))
	if err != nil {
		return nil, err
	}
	coll := client.Database(cfg.DBName).Collection(cfg.TableName)
	return &MongoTester{client: client, coll: coll, cfg: cfg}, nil
}

func (t *MongoTester) Seed(ctx context.Context) error {
	return SeedMongo(ctx, t.coll, t.cfg.RecordCount)
}

func (t *MongoTester) RunTest(ctx context.Context, wg *sync.WaitGroup) {
	RunMongoRead(ctx, t.coll, t.cfg.DBName, t.cfg.RecordCount, t.cfg.WorkerCount, t.cfg.ReadsTotal, t.cfg.ReadErrorsTotal, t.cfg.ReadLatency)
}

func (t *MongoTester) Close() {
	_ = t.client.Disconnect(context.Background())
}
