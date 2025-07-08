package main

import (
	"context"
	"encoding/json"
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
	// Подключения
	postgresURI   = "postgres://user:password@postgres-db:5432/ab_tests"
	cassandraHost = "cassandra-db:9042"
	mongoURI      = "mongodb://mongo-db:27017"
	etcdURI       = "http://etcd-db:2379"

	// Параметры теста
	keyspace       = "ab_tests"
	tableName      = "experiment_rules"
	recordCount    = 100000 // Количество записей для теста
	workerCount    = 100    // Количество параллельных воркеров
	testDuration   = 10 * time.Minute
	connectTimeout = 15 * time.Second
)

// --- Метрики Prometheus ---
var (
	readsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ab_reads_total",
		Help: "Total number of successful reads.",
	}, []string{"db"})
	readErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ab_read_errors_total",
		Help: "Total number of read errors.",
	}, []string{"db"})
	readLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ab_read_latency_seconds",
		Help:    "Read latency distribution.",
		Buckets: prometheus.DefBuckets,
	}, []string{"db"})
)

// --- Структура данных ---
type ExperimentRule struct {
	ID             int64  `bson:"_id" json:"id"`
	ExperimentName string `bson:"experiment_name" json:"experiment_name"`
	TargetingRules string `bson:"targeting_rules" json:"targeting_rules"`
}

// --- Интерфейс для тестера БД ---
type DatabaseTester interface {
	Seed(ctx context.Context) error
	RunTest(ctx context.Context, wg *sync.WaitGroup)
	Close()
}

// --- Main ---
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

// --- Реализация для PostgreSQL ---
type PostgresTester struct{ pool *pgxpool.Pool }

func NewPostgresTester(ctx context.Context) (*PostgresTester, error) {
	pool, err := pgxpool.New(ctx, postgresURI)
	if err != nil {
		return nil, err
	}
	return &PostgresTester{pool: pool}, nil
}
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
func (t *PostgresTester) RunTest(ctx context.Context, wg *sync.WaitGroup) {
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
					readLatency.WithLabelValues("postgres").Observe(time.Since(start).Seconds())
					if err != nil {
						readErrorsTotal.WithLabelValues("postgres").Inc()
					} else {
						readsTotal.WithLabelValues("postgres").Inc()
					}
				}
			}
		}()
	}
}
func (t *PostgresTester) Close() { t.pool.Close() }

// --- Реализация для Cassandra ---
type CassandraTester struct{ session *gocql.Session }

func NewCassandraTester(ctx context.Context) (*CassandraTester, error) {
	var session *gocql.Session
	var err error

	// --- 1. Подключение к системному кейспейсу с ретраями ---
	cluster := gocql.NewCluster(cassandraHost)
	cluster.Keyspace = "system"
	cluster.Timeout = 20 * time.Second
	cluster.ConnectTimeout = 15 * time.Second

	log.Println("Cassandra: attempting to connect to system keyspace...")
	for i := 0; i < 5; i++ {
		session, err = cluster.CreateSession()
		if err == nil {
			log.Println("Cassandra: system connection successful.")
			break // Успех
		}
		log.Printf("Cassandra connect failed (%v), retrying in 5s...", err)
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("cassandra connection failed after retries: %w", err)
	}
	defer session.Close()

	// --- 2. Создание кейспейса и таблицы ---
	err = session.Query(fmt.Sprintf(
		"CREATE KEYSPACE IF NOT EXISTS %s WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}",
		keyspace,
	)).Exec()
	if err != nil {
		return nil, fmt.Errorf("cassandra: CREATE KEYSPACE failed: %w", err)
	}
	log.Printf("Cassandra: Keyspace %s ensured.", keyspace)

	// --- 3. Финальное подключение к рабочему кейспейсу ---
	cluster.Keyspace = keyspace
	finalSession, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("cassandra: connection to keyspace '%s' failed: %w", keyspace, err)
	}
	log.Printf("Cassandra: Final session to keyspace '%s' created.", keyspace)

	return &CassandraTester{session: finalSession}, nil
}

// Замените существующий Seed на этот, чтобы он не создавал таблицу (она уже создана в New)
func (t *CassandraTester) Seed(ctx context.Context) error {
	// Таблица теперь создается при инициализации, но мы можем создать ее здесь для идемпотентности
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

func (t *CassandraTester) RunTest(ctx context.Context, wg *sync.WaitGroup) {
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
					readLatency.WithLabelValues("cassandra").Observe(time.Since(start).Seconds())
					if err != nil {
						if err != gocql.ErrNotFound {
							log.Printf("Cassandra read error: %v", err)
						}
						readErrorsTotal.WithLabelValues("cassandra").Inc()
					} else {
						readsTotal.WithLabelValues("cassandra").Inc()
					}
				}
			}
		}()
	}
}
func (t *CassandraTester) Close() { t.session.Close() }

// --- Реализация для MongoDB ---
type MongoTester struct {
	client *mongo.Client
	coll   *mongo.Collection
}

func NewMongoTester(ctx context.Context) (*MongoTester, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, err
	}
	coll := client.Database(keyspace).Collection(tableName)
	return &MongoTester{client: client, coll: coll}, nil
}
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
					readLatency.WithLabelValues("mongo").Observe(time.Since(start).Seconds())
					if err != nil {
						if err != mongo.ErrNoDocuments {
							log.Printf("Mongo read error: %v", err)
						}
						readErrorsTotal.WithLabelValues("mongo").Inc()
					} else {
						readsTotal.WithLabelValues("mongo").Inc()
					}
				}
			}
		}()
	}
}
func (t *MongoTester) Close() { t.client.Disconnect(context.Background()) }

// --- Реализация для etcd ---
type EtcdTester struct{ client *clientv3.Client }

func NewEtcdTester(ctx context.Context) (*EtcdTester, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdURI},
		DialTimeout: connectTimeout,
	})
	return &EtcdTester{client: client}, err
}
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
					_, err := t.client.Get(ctx, key)
					readLatency.WithLabelValues("etcd").Observe(time.Since(start).Seconds())
					if err != nil {
						readErrorsTotal.WithLabelValues("etcd").Inc()
					} else {
						readsTotal.WithLabelValues("etcd").Inc()
					}
				}
			}
		}()
	}
}
func (t *EtcdTester) Close() { t.client.Close() }
