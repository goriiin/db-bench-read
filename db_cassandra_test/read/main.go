package main

import (
	"context"
	"db-bench/lib"
	"db-bench/lib/conf"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

func main() {
	configPath := os.Getenv("CONFIG_PATH")
	cfg, err := conf.LoadConfig("cassandra", configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Fatal(http.ListenAndServe(":8081", nil))
	}()

	time.Sleep(2 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.TestDuration)
	defer cancel()
	tester, err := lib.GetTester("cassandra", cfg)
	if err != nil {
		log.Fatalf("Failed to initialize cassandra tester: %v", err)
	}
	defer tester.Close()
	var wg sync.WaitGroup
	startTime := time.Now()

	tester.RunTest(ctx, &wg)
	wg.Wait()
	log.Println("Test for postgres completed.")

	duration := time.Since(startTime)
	log.Printf("Test completed. Duration: %v", duration)

	time.Sleep(10 * time.Second)
}
