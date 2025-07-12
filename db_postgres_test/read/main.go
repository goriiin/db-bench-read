package main

import (
	"context"
	"db-bench/lib"
	"db-bench/lib/conf"
	"log"
	"os"
	"sync"
)

func main() {
	configPath := os.Getenv("CONFIG_PATH")
	cfg, err := conf.LoadConfig("postgres", configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.TestDuration)
	defer cancel()
	tester, err := lib.GetTester("postgres", cfg)
	if err != nil {
		log.Fatalf("Failed to initialize postgres tester: %v", err)
	}
	defer tester.Close()
	var wg sync.WaitGroup
	tester.RunTest(ctx, &wg)
	wg.Wait()
	log.Println("Test for postgres completed.")
}
