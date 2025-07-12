package conf

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/spf13/viper"
)

type Config struct {
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

func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.AddConfigPath(".")
	}
	v.AutomaticEnv()
	v.SetDefault("URI", "")
	v.SetDefault("DBName", "")
	v.SetDefault("WorkerCount", 100)
	v.SetDefault("RecordCount", 100000)
	v.SetDefault("TableName", "experiment_rules")
	v.SetDefault("TestDuration", "10m")
	v.SetDefault("ConnectTimeout", "15s")

	if err := v.ReadInConfig(); err != nil {
		// Не критично, если файла нет, читаем только env
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	cfg := &Config{
		URI:            v.GetString("URI"),
		DBName:         v.GetString("DBName"),
		WorkerCount:    v.GetInt("WorkerCount"),
		RecordCount:    v.GetInt("RecordCount"),
		TableName:      v.GetString("TableName"),
		TestDuration:   v.GetDuration("TestDuration"),
		ConnectTimeout: v.GetDuration("ConnectTimeout"),
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
