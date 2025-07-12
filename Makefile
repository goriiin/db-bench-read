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
	docker compose -f docker-compose.postgres.yml down
	docker compose -f docker-compose.cassandra.yml down
	docker compose -f docker-compose.mongo.yml down
	docker compose -f docker-compose.etcd.yml down

logs:
	docker compose logs -f tester

run-postgres-seed:
	docker compose -f docker-compose.postgres.yml build seed_go
	docker compose -f docker-compose.postgres.yml up -d postgres
	docker compose -f docker-compose.postgres.yml up seed_go

run-postgres-read:
	docker compose -f docker-compose.postgres.yml build read_go
	docker compose -f docker-compose.postgres.yml up -d postgres-db
	docker compose -f docker-compose.postgres.yml up read_go

run-cassandra-seed:
	docker compose -f docker-compose.cassandra.yml build seed_go
	docker compose -f docker-compose.cassandra.yml up -d cassandra-db
	docker compose -f docker-compose.cassandra.yml up seed_go

run-cassandra-read:
	docker compose -f docker-compose.cassandra.yml build read_go
	docker compose -f docker-compose.cassandra.yml up -d cassandra-db
	docker compose -f docker-compose.cassandra.yml up read_go

run-mongo-seed:
	docker compose -f docker-compose.mongo.yml build seed_go
	docker compose -f docker-compose.mongo.yml up -d mongo-db
	docker compose -f docker-compose.mongo.yml up seed_go

run-mongo-read:
	docker compose -f docker-compose.mongo.yml build read_go
	docker compose -f docker-compose.mongo.yml up -d mongo-db
	docker compose -f docker-compose.mongo.yml up read_go

run-etcd-seed:
	docker compose -f docker-compose.etcd.yml build seed_go
	docker compose -f docker-compose.etcd.yml up -d etcd-db
	docker compose -f docker-compose.etcd.yml up seed_go

run-etcd-read:
	docker compose -f docker-compose.etcd.yml build read_go
	docker compose -f docker-compose.etcd.yml up -d etcd-db
	docker compose -f docker-compose.etcd.yml up read_go