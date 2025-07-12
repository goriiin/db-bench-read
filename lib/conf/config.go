package conf

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/spf13/viper"
)

type Config struct {
	DB              string
	URI             string
	DBName          string
	WorkerCount     int
	RecordCount     int
	TableName       string
	TestDuration    time.Duration
	ConnectTimeout  time.Duration
	ReadsTotal      *prometheus.CounterVec
	ReadErrorsTotal *prometheus.CounterVec
	ReadLatency     *prometheus.HistogramVec
}

type ExperimentRule struct {
	ID             int64  `bson:"_id" json:"id"`
	ExperimentName string `bson:"experiment_name" json:"experiment_name"`
	TargetingRules string `bson:"targeting_rules" json:"targeting_rules"`
}

func LoadConfig(db string, configPath string) (*Config, error) {
	v := viper.New()
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.AddConfigPath(".")
	}
	v.AutomaticEnv()
	// Общие параметры
	v.SetDefault("workerCount", 100)
	v.SetDefault("recordCount", 100000)
	v.SetDefault("tableName", "experiment_rules")
	v.SetDefault("testDuration", "10m")
	v.SetDefault("connectTimeout", "15s")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	// Получаем параметры для выбранной базы
	uri := v.GetString(fmt.Sprintf("%s.uri", db))
	dbName := v.GetString(fmt.Sprintf("%s.dbName", db))
	workerCount := v.GetInt("workerCount")
	recordCount := v.GetInt("recordCount")
	tableName := v.GetString("tableName")
	testDuration := v.GetDuration("testDuration")
	connectTimeout := v.GetDuration("connectTimeout")

	cfg := &Config{
		DB:             db,
		URI:            uri,
		DBName:         dbName,
		WorkerCount:    workerCount,
		RecordCount:    recordCount,
		TableName:      tableName,
		TestDuration:   testDuration,
		ConnectTimeout: connectTimeout,
		ReadsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "ab_reads_total", Help: "Total number of successful reads.",
		}, []string{"db"}),
		ReadErrorsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "ab_read_errors_total", Help: "Total number of read errors.",
		}, []string{"db"}),
		ReadLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "ab_read_latency_seconds",
			Help:    "Read latency distribution.",
			Buckets: prometheus.DefBuckets,
		}, []string{"db"}),
	}
	return cfg, nil
}
