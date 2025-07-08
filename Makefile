.PHONY: all up down logs \
        seed-postgres seed-cassandra seed-mongo seed-etcd \
        test-postgres test-cassandra test-mongo test-etcd

# Запускает все сервисы в фоновом режиме
up:
	@echo "Starting all services..."
	docker compose up -d

# Останавливает и удаляет все сервисы
down:
	@echo "Stopping all services..."
	docker compose down --volumes

# Заполняет PostgreSQL данными
seed-postgres: up
	@echo "Seeding PostgreSQL..."
	docker compose run --rm tester --mode=seed --db=postgres

# Заполняет Cassandra данными
seed-cassandra: up
	@echo "Seeding Cassandra..."
	docker compose run --rm tester --mode=seed --db=cassandra

# Заполняет MongoDB данными
seed-mongo: up
	@echo "Seeding MongoDB..."
	docker compose run --rm tester --mode=seed --db=mongo

# Заполняет etcd данными
seed-etcd: up
	@echo "Seeding etcd..."
	docker compose run --rm tester --mode=seed --db=etcd

# Запускает тест на чтение для PostgreSQL
test-postgres:
	@echo "Starting PostgreSQL read test. View results at http://localhost:3000"
	docker compose run --rm tester --mode=test --db=postgres

# Запускает тест на чтение для Cassandra
test-cassandra:
	@echo "Starting Cassandra read test. View results at http://localhost:3000"
	docker compose run --rm tester --mode=test --db=cassandra

# Запускает тест на чтение для MongoDB
test-mongo:
	@echo "Starting MongoDB read test. View results at http://localhost:3000"
	docker compose run --rm tester --mode=test --db=mongo

# Запускает тест на чтение для etcd
test-etcd:
	@echo "Starting etcd read test. View results at http://localhost:3000"
	docker compose run --rm tester --mode=test --db=etcd

# Открывает логи тестера
logs:
	docker compose logs -f tester```

### `db-benchmark/go.mod`