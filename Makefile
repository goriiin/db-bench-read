.PHONY: all build up down seed-ydb seed-cassandra test-ydb test-cassandra logs

# Запускает все сервисы в фоновом режиме
up:
	@echo "Starting all services..."
	docker compose up -d

# Останавливает и удаляет все сервисы
down:
	@echo "Stopping all services..."
	docker compose down --volumes

# Заполняет YDB начальными данными
seed-ydb: up
	@echo "Seeding YDB with 1,000,000 records..."
	docker compose run --rm tester --mode=seed --db=ydb

# Заполняет Cassandra начальными данными
seed-cassandra: up
	@echo "Waiting for Cassandra to be fully ready..."
	@sleep 30
	@echo "Seeding Cassandra with 1,000,000 records..."
	docker compose run --rm tester --mode=seed --db=cassandra

# Запускает 10-минутный тест на чтение для YDB
test-ydb:
	@echo "Starting YDB read test for 10 minutes. View results at http://localhost:3000"
	docker compose run --rm tester --mode=test --db=ydb

# Запускает 10-минутный тест на чтение для Cassandra
test-cassandra:
	@echo "Starting Cassandra read test for 10 minutes. View results at http://localhost:3000"
	docker compose run --rm tester --mode=test --db=cassandra

# Открывает логи тестера
logs:
	docker compose logs -f tester