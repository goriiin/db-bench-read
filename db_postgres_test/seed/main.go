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
	cfg, err := conf.LoadConfig("postgres", configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	tester, err := lib.GetTester("postgres", cfg)
	if err != nil {
		log.Fatalf("Failed to initialize postgres tester: %v", err)
	}
	defer tester.Close()
	if err := tester.Seed(context.Background()); err != nil {
		log.Fatalf("Seeding failed for postgres: %v", err)
	}
	log.Println("Seeding for postgres completed.")
}
