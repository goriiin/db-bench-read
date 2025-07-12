package main

import (
	"context"
	"log"

	"db-bench/lib"
	"db-bench/lib/conf"
)

func main() {
	cfg, err := conf.LoadConfig("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	tester, err := lib.GetTester("etcd", cfg)
	if err != nil {
		log.Fatalf("Failed to initialize etcd tester: %v", err)
	}
	defer tester.Close()
	if err := tester.Seed(context.Background()); err != nil {
		log.Fatalf("Seeding failed for etcd: %v", err)
	}
	log.Println("Seeding for etcd completed.")
}
