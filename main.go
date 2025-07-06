package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gocql/gocql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/table"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/options"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/types"
)

const (
	ydbEndpoint   = "grpc://ydb-db:2136"
	ydbDatabase   = "/local"
	cassandraHost = "cassandra-db:9042"
	keyspace      = "ab_tests"
	tableName     = "experiment_rules"
	recordCount   = 1000000
	workerCount   = 100
	testDuration  = 10 * time.Minute
)

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

func main() {
	mode := flag.String("mode", "test", "Mode to run: 'seed' or 'test'")
	dbType := flag.String("db", "ydb", "Database to use: 'ydb' or 'cassandra'")
	flag.Parse()

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(":8081", nil); err != nil {
			log.Fatalf("Metrics server failed: %v", err)
		}
	}()
	log.Println("Metrics server started on :8081")

	switch *mode {
	case "seed":
		seedDatabase(*dbType)
	case "test":
		runTest(*dbType)
	default:
		log.Fatalf("Unknown mode: %s", *mode)
	}
}

func seedDatabase(dbType string) {
	log.Printf("Starting seed for %s...", dbType)
	switch dbType {
	case "ydb":
		seedYDB()
	case "cassandra":
		seedCassandra()
	}
	log.Printf("Seeding for %s completed.", dbType)
}

func runTest(dbType string) {
	log.Printf("Starting test for %s for %v...", dbType, testDuration)
	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	var wg sync.WaitGroup
	switch dbType {
	case "ydb":
		runYDBTest(ctx, &wg)
	case "cassandra":
		runCassandraTest(ctx, &wg)
	}
	wg.Wait()
	log.Printf("Test for %s completed.", dbType)
}

func connectYDB() (*ydb.Driver, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return ydb.New(ctx, ydb.WithEndpoint(ydbEndpoint), ydb.WithDatabase(ydbDatabase), ydb.WithAnonymousCredentials())
}

func seedYDB() {
	ctx := context.Background()
	db, err := connectYDB()
	if err != nil {
		log.Fatalf("Failed to connect to YDB: %v", err)
	}
	defer db.Close(ctx)

	// Create Table
	err = db.Table().Do(ctx, func(ctx context.Context, s table.Session) error {
		return s.CreateTable(ctx, db.Name()+"/"+tableName,
			options.WithColumn("experiment_id", types.TypeInt64), // NOT OPTIONAL
			options.WithColumn("experiment_name", types.Optional(types.TypeText)),
			options.WithColumn("targeting_rules", types.Optional(types.TypeJSON)),
			options.WithColumn("buckets", types.Optional(types.TypeText)),
			options.WithPrimaryKeyColumn("experiment_id"),
		)
	})
	if err != nil {
		log.Printf("YDB CreateTable failed (might already exist, which is fine): %v", err)
	}

	// Prepare data for bulk upsert
	rules := make([]types.Value, 0, recordCount)
	for i := 0; i < recordCount; i++ {
		rules = append(rules, types.StructValue(
			types.StructFieldValue("experiment_id", types.Int64Value(int64(i+1))),
			types.StructFieldValue("experiment_name", types.OptionalValue(types.TextValue(fmt.Sprintf("Test %d", i+1)))),
			types.StructFieldValue("targeting_rules", types.OptionalValue(types.JSONValue(`{"country":"US"}`))),
			types.StructFieldValue("buckets", types.OptionalValue(types.TextValue(`[{"id":0,"share":50}]`))),
		))
	}

	log.Println("YDB: Writing rows...")
	err = db.Table().Do(ctx, func(ctx context.Context, s table.Session) error {
		return s.BulkUpsert(ctx, ydbDatabase+"/"+tableName, types.ListValue(rules...))
	})
	if err != nil {
		log.Fatalf("YDB BulkUpsert failed: %v", err)
	}
}

func runYDBTest(ctx context.Context, wg *sync.WaitGroup) {
	db, err := connectYDB()
	if err != nil {
		log.Fatalf("Failed to connect to YDB: %v", err)
	}
	defer db.Close(ctx)

	// FIX: Use correct transaction control constructor
	txControl := table.TxControl()
	readQuery := fmt.Sprintf("DECLARE $id AS Int64; SELECT experiment_id FROM `%s` WHERE experiment_id = $id;", tableName)

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					start := time.Now()
					id := rand.Int63n(recordCount) + 1
					err := db.Table().Do(ctx, func(ctx context.Context, s table.Session) error {
						_, res, err := s.Execute(ctx, txControl, readQuery, table.NewQueryParameters(table.ValueParam("$id", types.Int64Value(id))))
						if err == nil {
							_ = res.Close()
						}
						return err
					})
					readLatency.WithLabelValues("ydb").Observe(time.Since(start).Seconds())
					if err != nil {
						readErrorsTotal.WithLabelValues("ydb").Inc()
					} else {
						readsTotal.WithLabelValues("ydb").Inc()
					}
				}
			}
		}()
	}
}

func connectCassandra() (*gocql.Session, error) {
	cluster := gocql.NewCluster(cassandraHost)
	cluster.Keyspace = "system"
	cluster.Timeout = 20 * time.Second
	var session *gocql.Session
	var err error
	// Retry loop for Cassandra connection
	for i := 0; i < 10; i++ {
		session, err = cluster.CreateSession()
		if err == nil {
			break
		}
		log.Printf("Cassandra connection attempt %d failed: %v. Retrying in 10s...", i+1, err)
		time.Sleep(10 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("could not connect to cassandra after multiple retries: %w", err)
	}

	// FIX: Use double quotes for Go string literal
	err = session.Query(fmt.Sprintf("CREATE KEYSPACE IF NOT EXISTS %s WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}", keyspace)).Exec()
	if err != nil {
		log.Fatalf("Cassandra CREATE KEYSPACE failed: %v", err)
	}
	session.Close()

	cluster.Keyspace = keyspace
	return cluster.CreateSession()
}

func seedCassandra() {
	session, err := connectCassandra()
	if err != nil {
		log.Fatalf("Failed to connect to Cassandra: %v", err)
	}
	defer session.Close()

	err = session.Query(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.%s (
		experiment_id bigint PRIMARY KEY, 
		experiment_name text, 
		targeting_rules text, 
		buckets text
	)`, keyspace, tableName)).Exec()
	if err != nil {
		log.Fatalf("Cassandra CREATE TABLE failed: %v", err)
	}

	log.Println("Cassandra: Writing rows...")
	insertQuery := fmt.Sprintf("INSERT INTO %s (experiment_id, experiment_name, targeting_rules, buckets) VALUES (?, ?, ?, ?)", tableName)

	for i := 1; i <= recordCount; i++ {
		if err := session.Query(insertQuery, int64(i), fmt.Sprintf("Test %d", i), `{"country":"US"}`, `[{"id":0,"share":50}]`).Exec(); err != nil {
			log.Printf("Warning: Cassandra insert failed for key %d: %v", i, err)
		}
		if i%1000 == 0 {
			log.Printf("Cassandra: %d records inserted...", i)
		}
	}
}

func runCassandraTest(ctx context.Context, wg *sync.WaitGroup) {
	session, err := connectCassandra()
	if err != nil {
		log.Fatalf("Failed to connect to Cassandra: %v", err)
	}
	defer session.Close()
	readQuery := fmt.Sprintf("SELECT experiment_id FROM %s WHERE experiment_id = ?", tableName)

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
					start := time.Now()
					id := rand.Int63n(recordCount) + 1
					err := session.Query(readQuery, id).Consistency(gocql.One).Scan(&idRead)
					readLatency.WithLabelValues("cassandra").Observe(time.Since(start).Seconds())
					if err != nil {
						// gocql.ErrNotFound is a common case, not necessarily a system error
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
