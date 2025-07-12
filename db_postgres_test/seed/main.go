package main

import (
	"context"
	"db-bench/lib"
	"db-bench/lib/conf"
	"log"
)

func main() {
	cfg, err := conf.LoadConfig("")
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
