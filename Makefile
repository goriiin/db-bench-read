.PHONY: all up down logs prune \
        seed-postgres seed-cassandra seed-mongo seed-etcd \
        test-postgres test-cassandra test-mongo test-etcd

prune:
	@echo "Stopping and removing project containers..."
	docker compose down --volumes
	@echo "Pruning Docker system..."
	docker system prune -a -f --volumes

up:
	@echo "Starting project services..."
	docker compose up -d

down:
	@echo "Stopping project services..."
	docker compose down --volumes

seed-postgres: up
	@echo "Seeding PostgreSQL..."
	go run db_postgres_test/seed/main.go

seed-cassandra: up
	@echo "Seeding Cassandra..."
	go run db_cassandra_test/seed/main.go

seed-mongo: up
	@echo "Seeding MongoDB..."
	go run db_mongo_test/seed/main.go

seed-etcd: up
	@echo "Seeding etcd..."
	go run db_etcd_test/seed/main.go

test-postgres:
	@echo "Starting PostgreSQL read test. View results at http://localhost:3000"
	go run db_postgres_test/read/main.go

test-cassandra:
	@echo "Starting Cassandra read test. View results at http://localhost:3000"
	go run db_cassandra_test/read/main.go

test-mongo:
	@echo "Starting MongoDB read test. View results at http://localhost:3000"
	go run db_mongo_test/read/main.go

test-etcd:
	@echo "Starting etcd read test. View results at http://localhost:3000"
	go run db_etcd_test/read/main.go

logs:
	docker compose logs -f tester

run-postgres-seed:
	docker compose -f docker-compose.postgres.yml build tester-seed
	docker compose -f docker-compose.postgres.yml up -d postgres-db
	docker compose -f docker-compose.postgres.yml up tester-seed

run-postgres-read:
	docker compose -f docker-compose.postgres.yml build tester-read
	docker compose -f docker-compose.postgres.yml up -d postgres-db
	docker compose -f docker-compose.postgres.yml up tester-read

run-cassandra-seed:
	docker compose -f docker-compose.cassandra.yml build tester-seed
	docker compose -f docker-compose.cassandra.yml up -d cassandra-db
	docker compose -f docker-compose.cassandra.yml up tester-seed

run-cassandra-read:
	docker compose -f docker-compose.cassandra.yml build tester-read
	docker compose -f docker-compose.cassandra.yml up -d cassandra-db
	docker compose -f docker-compose.cassandra.yml up tester-read

run-mongo-seed:
	docker compose -f docker-compose.mongo.yml build tester-seed
	docker compose -f docker-compose.mongo.yml up -d mongo-db
	docker compose -f docker-compose.mongo.yml up tester-seed

run-mongo-read:
	docker compose -f docker-compose.mongo.yml build tester-read
	docker compose -f docker-compose.mongo.yml up -d mongo-db
	docker compose -f docker-compose.mongo.yml up tester-read

run-etcd-seed:
	docker compose -f docker-compose.etcd.yml build tester-seed
	docker compose -f docker-compose.etcd.yml up -d etcd-db
	docker compose -f docker-compose.etcd.yml up tester-seed

run-etcd-read:
	docker compose -f docker-compose.etcd.yml build tester-read
	docker compose -f docker-compose.etcd.yml up -d etcd-db
	docker compose -f docker-compose.etcd.yml up tester-read