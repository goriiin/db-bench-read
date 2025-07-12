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
	tester, err := lib.GetTester("mongo", cfg)
	if err != nil {
		log.Fatalf("Failed to initialize mongo tester: %v", err)
	}
	defer tester.Close()
	if err := tester.Seed(context.Background()); err != nil {
		log.Fatalf("Seeding failed for mongo: %v", err)
	}
	log.Println("Seeding for mongo completed.")
}
