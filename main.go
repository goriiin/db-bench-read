package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gocql/gocql"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// --- Конфигурация ---
const (
	postgresURI    = "postgres://user:password@postgres-db:5432/ab_tests"
	cassandraHost  = "cassandra-db:9042"
	mongoURI       = "mongodb://mongo-db:27017"
	etcdURI        = "http://etcd-db:2379"
	keyspace       = "ab_tests"
	tableName      = "experiment_rules"
	recordCount    = 100000
	workerCount    = 100
	testDuration   = 10 * time.Minute
	connectTimeout = 15 * time.Second
)

// --- Метрики Prometheus ---
var (
	readsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ab_reads_total", Help: "Total number of successful reads.",
	}, []string{"db"})
	readErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ab_read_errors_total", Help: "Total number of read errors.",
	}, []string{"db"})
	readLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "ab_read_latency_seconds", Help: "Read latency distribution.", Buckets: prometheus.DefBuckets,
	}, []string{"db"})
)

type ExperimentRule struct {
	ID             int64  `bson:"_id" json:"id"`
	ExperimentName string `bson:"experiment_name" json:"experiment_name"`
	TargetingRules string `bson:"targeting_rules" json:"targeting_rules"`
}

type DatabaseTester interface {
	Seed(ctx context.Context) error
	RunTest(ctx context.Context, wg *sync.WaitGroup)
	Close()
}

func main() {
	mode := flag.String("mode", "test", "Режим: 'seed' или 'test'")
	dbType := flag.String("db", "postgres", "БД: 'postgres', 'cassandra', 'mongo', 'etcd'")
	flag.Parse()

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(":8081", nil); err != nil {
			log.Fatalf("Metrics server failed: %v", err)
		}
	}()
	log.Println("Metrics server started on :8081")

	tester, err := getTester(*dbType)
	if err != nil {
		log.Fatalf("Failed to initialize tester: %v", err)
	}
	defer tester.Close()

	switch *mode {
	case "seed":
		log.Printf("Starting seed for %s...", *dbType)
		if err := tester.Seed(context.Background()); err != nil {
			log.Fatalf("Seeding failed for %s: %v", *dbType, err)
		}
		log.Printf("Seeding for %s completed.", *dbType)
	case "test":
		log.Printf("Starting test for %s for %v...", *dbType, testDuration)
		ctx, cancel := context.WithTimeout(context.Background(), testDuration)
		defer cancel()
		var wg sync.WaitGroup
		tester.RunTest(ctx, &wg)
		wg.Wait()
		log.Printf("Test for %s completed.", *dbType)
	default:
		log.Fatalf("Unknown mode: %s", *mode)
	}
}

func getTester(dbType string) (DatabaseTester, error) {
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()

	switch dbType {
	case "postgres":
		return NewPostgresTester(ctx)
	case "cassandra":
		return NewCassandraTester(ctx)
	case "mongo":
		return NewMongoTester(ctx)
	case "etcd":
		return NewEtcdTester(ctx)
	default:
		return nil, fmt.Errorf("unknown database type: %s", dbType)
	}
}

// --- PostgreSQL ---
type PostgresTester struct {
	pool   *pgxpool.Pool
	dbName string // ИЗМЕНЕНИЕ: Добавлено поле для имени БД
}

func NewPostgresTester(ctx context.Context) (*PostgresTester, error) {
	pool, err := pgxpool.New(ctx, postgresURI)
	if err != nil {
		return nil, err
	}
	return &PostgresTester{pool: pool, dbName: "postgres"}, nil // ИЗМЕНЕНИЕ: Инициализация поля
}

func (t *PostgresTester) RunTest(ctx context.Context, wg *sync.WaitGroup) {
	log.Printf("RunTest db %s", t.dbName)
	query := fmt.Sprintf("SELECT id FROM %s WHERE id = $1", tableName)
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var idRead int64
			for {
				select {
				case <-ctx.Done():
					return
				default:
					id := rand.Int63n(recordCount) + 1
					start := time.Now()
					err := t.pool.QueryRow(ctx, query, id).Scan(&idRead)
					// ИЗМЕНЕНИЕ: Используется поле dbName вместо строки "postgres"
					readLatency.WithLabelValues(t.dbName).Observe(time.Since(start).Seconds())
					if err != nil {
						readErrorsTotal.WithLabelValues(t.dbName).Inc()
					} else {
						readsTotal.WithLabelValues(t.dbName).Inc()
					}
				}
			}
		}()
	}
}

// --- Cassandra ---
type CassandraTester struct {
	session *gocql.Session
	dbName  string // ИЗМЕНЕНИЕ: Добавлено поле для имени БД
}

func NewCassandraTester(ctx context.Context) (*CassandraTester, error) {
	cluster := gocql.NewCluster(cassandraHost)
	cluster.Keyspace = "system"
	cluster.Timeout = 20 * time.Second
	cluster.ConnectTimeout = 15 * time.Second
	var session *gocql.Session
	var err error
	for i := 0; i < 5; i++ {
		session, err = cluster.CreateSession()
		if err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		return nil, err
	}
	defer session.Close()
	err = session.Query(fmt.Sprintf("CREATE KEYSPACE IF NOT EXISTS %s WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}", keyspace)).Exec()
	if err != nil {
		return nil, err
	}
	cluster.Keyspace = keyspace
	finalSession, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}

	return &CassandraTester{session: finalSession, dbName: "cassandra"}, nil // ИЗМЕНЕНИЕ: Инициализация поля
}

func (t *CassandraTester) RunTest(ctx context.Context, wg *sync.WaitGroup) {
	log.Printf("RunTest db %s", t.dbName)
	query := fmt.Sprintf("SELECT id FROM %s WHERE id = ?", tableName)
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var idRead int64
			for {
				select {
				case <-ctx.Done():
					return
				default:
					id := rand.Int63n(recordCount) + 1
					start := time.Now()
					err := t.session.Query(query, id).Consistency(gocql.One).Scan(&idRead)
					readLatency.WithLabelValues(t.dbName).Observe(time.Since(start).Seconds())
					if err != nil {
						if !errors.Is(err, gocql.ErrNotFound) {
							log.Printf("Cassandra read error: %v", err)
						}
						readErrorsTotal.WithLabelValues(t.dbName).Inc()
					} else {
						readsTotal.WithLabelValues(t.dbName).Inc()
					}
				}
			}
		}()
	}
}

// --- MongoDB ---
type MongoTester struct {
	coll   *mongo.Collection
	client *mongo.Client
	dbName string // ИЗМЕНЕНИЕ: Добавлено поле для имени БД
}

func NewMongoTester(ctx context.Context) (*MongoTester, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, err
	}
	coll := client.Database(keyspace).Collection(tableName)
	return &MongoTester{client: client, coll: coll, dbName: "mongo"}, nil // ИЗМЕНЕНИЕ: Инициализация поля
}

func (t *MongoTester) RunTest(ctx context.Context, wg *sync.WaitGroup) {
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var result ExperimentRule
			for {
				select {
				case <-ctx.Done():
					return
				default:
					id := rand.Int63n(recordCount) + 1
					filter := bson.M{"_id": id}
					start := time.Now()
					err := t.coll.FindOne(ctx, filter).Decode(&result)
					// ИЗМЕНЕНИЕ: Используется поле dbName вместо строки "mongo"
					readLatency.WithLabelValues(t.dbName).Observe(time.Since(start).Seconds())
					if err != nil {
						if !errors.Is(err, mongo.ErrNoDocuments) {
							log.Printf("Mongo read error: %v", err)
						}
						readErrorsTotal.WithLabelValues(t.dbName).Inc()
					} else {
						readsTotal.WithLabelValues(t.dbName).Inc()
					}
				}
			}
		}()
	}
}

// --- etcd ---
type EtcdTester struct {
	client *clientv3.Client
	dbName string // ИЗМЕНЕНИЕ: Добавлено поле для имени БД
}

func NewEtcdTester(ctx context.Context) (*EtcdTester, error) {
	client, err := clientv3.New(clientv3.Config{Endpoints: []string{etcdURI}, DialTimeout: connectTimeout})
	if err != nil {
		return nil, err
	}
	return &EtcdTester{client: client, dbName: "etcd"}, nil // ИЗМЕНЕНИЕ: Инициализация поля
}

func (t *EtcdTester) RunTest(ctx context.Context, wg *sync.WaitGroup) {
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					id := rand.Int63n(recordCount) + 1
					key := fmt.Sprintf("/experiments/%d", id)
					start := time.Now()
					resp, err := t.client.Get(ctx, key)
					// ИЗМЕНЕНИЕ: Используется поле dbName вместо строки "etcd"
					readLatency.WithLabelValues(t.dbName).Observe(time.Since(start).Seconds())
					if err != nil || resp.Count == 0 {
						readErrorsTotal.WithLabelValues(t.dbName).Inc()
					} else {
						readsTotal.WithLabelValues(t.dbName).Inc()
					}
				}
			}
		}()
	}
}

// Методы Seed и Close оставлены без изменений, добавил только не изменные для полноты
func (t *PostgresTester) Seed(ctx context.Context) error {
	createTableSQL := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
		id BIGINT PRIMARY KEY,
		experiment_name TEXT,
		targeting_rules JSONB
	)`, tableName)
	if _, err := t.pool.Exec(ctx, createTableSQL); err != nil {
		return err
	}
	log.Println("Postgres: Writing rows...")
	for i := 1; i <= recordCount; i++ {
		rule := ExperimentRule{ID: int64(i), ExperimentName: fmt.Sprintf("Test %d", i), TargetingRules: `{"country":"US"}`}
		_, err := t.pool.Exec(ctx,
			fmt.Sprintf("INSERT INTO %s (id, experiment_name, targeting_rules) VALUES ($1, $2, $3) ON CONFLICT (id) DO NOTHING", tableName),
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

func (t *PostgresTester) Close() { t.pool.Close() }

func (t *CassandraTester) Seed(ctx context.Context) error {
	err := t.session.Query(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.%s (
		id bigint PRIMARY KEY,
		experiment_name text,
		targeting_rules text
	)`, keyspace, tableName)).Exec()
	if err != nil {
		return fmt.Errorf("failed to create table in cassandra: %w", err)
	}

	log.Println("Cassandra: Writing rows...")
	query := fmt.Sprintf("INSERT INTO %s (id, experiment_name, targeting_rules) VALUES (?, ?, ?)", tableName)
	for i := 1; i <= recordCount; i++ {
		rule := ExperimentRule{ID: int64(i), ExperimentName: fmt.Sprintf("Test %d", i), TargetingRules: `{"country":"US"}`}
		if err := t.session.Query(query, rule.ID, rule.ExperimentName, rule.TargetingRules).Exec(); err != nil {
			log.Printf("Warning: Cassandra insert failed for key %d: %v", i, err)
		}
		if i%10000 == 0 {
			log.Printf("Cassandra: %d records inserted...", i)
		}
	}
	return nil
}

func (t *CassandraTester) Close() { t.session.Close() }

func (t *MongoTester) Seed(ctx context.Context) error {
	log.Println("MongoDB: Writing rows...")
	var models []mongo.WriteModel
	for i := 1; i <= recordCount; i++ {
		rule := ExperimentRule{ID: int64(i), ExperimentName: fmt.Sprintf("Test %d", i), TargetingRules: `{"country":"US"}`}
		models = append(models, mongo.NewInsertOneModel().SetDocument(rule))
		if i%10000 == 0 || i == recordCount {
			_, err := t.coll.BulkWrite(ctx, models)
			if err != nil {
				log.Printf("Warning: Mongo bulk write failed: %v", err)
			}
			log.Printf("MongoDB: %d records prepared...", i)
			models = nil // reset batch
		}
	}
	return nil
}

func (t *MongoTester) Close() { t.client.Disconnect(context.Background()) }

func (t *EtcdTester) Seed(ctx context.Context) error {
	log.Println("etcd: Writing rows...")
	for i := 1; i <= recordCount; i++ {
		rule := ExperimentRule{ID: int64(i), ExperimentName: fmt.Sprintf("Test %d", i), TargetingRules: `{"country":"US"}`}
		key := fmt.Sprintf("/experiments/%d", rule.ID)
		val, _ := json.Marshal(rule)
		_, err := t.client.Put(ctx, key, string(val))
		if err != nil {
			log.Printf("Warning: etcd put failed for key %s: %v", key, err)
		}
		if i%10000 == 0 {
			log.Printf("etcd: %d records inserted...", i)
		}
	}
	return nil
}

func (t *EtcdTester) Close() { t.client.Close() }
