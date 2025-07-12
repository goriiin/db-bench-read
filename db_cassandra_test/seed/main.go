package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"db-bench/lib"
	"db-bench/lib/conf"
)

func main() {
	configPath := os.Getenv("CONFIG_PATH")
	cfg, err := conf.LoadConfig("cassandra", configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Println(cfg)
	tester, err := lib.GetTester("cassandra", cfg)
	if err != nil {
		log.Fatalf("Failed to initialize cassandra tester: %v", err)
	}
	defer tester.Close()
	if err := tester.Seed(context.Background()); err != nil {
		log.Fatalf("Seeding failed for cassandra: %v", err)
	}
	log.Println("Seeding for cassandra completed.")
}
