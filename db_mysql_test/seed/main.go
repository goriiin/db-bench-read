package main

import (
	"context"
	"db-bench/lib"
	"db-bench/lib/conf"
	"log"
	"os"
)

func main() {
	configPath := os.Getenv("CONFIG_PATH")
	cfg, err := conf.LoadConfig("mysql", configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	tester, err := lib.GetTester("mysql", cfg)
	if err != nil {
		log.Fatalf("Failed to initialize mysql tester: %v", err)
	}
	defer tester.Close()

	if err := tester.Seed(context.Background()); err != nil {
		log.Fatalf("Seeding failed for mysql: %v", err)
	}

	log.Println("Seeding for mysql completed.")
}
