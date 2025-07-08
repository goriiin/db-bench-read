.PHONY: all up down logs prune \
        seed-postgres seed-cassandra seed-mongo seed-etcd \
        test-postgres test-cassandra test-mongo test-etcd

# ВНИМАНИЕ: Эта команда удалит ВСЕ неиспользуемые Docker-объекты (контейнеры, сети, образы, volumes)
prune:
	@echo "Stopping and removing project containers..."
	docker compose down --volumes
	@echo "Pruning Docker system..."
	docker system prune -a -f --volumes

# Запускает все сервисы проекта в фоновом режиме
up:
	@echo "Starting project services..."
	docker compose up -d

# Останавливает и удаляет сервисы, определенные в docker-compose.yml
down:
	@echo "Stopping project services..."
	docker compose down --volumes

# --- SEED TARGETS ---
# Заполняет PostgreSQL данными
seed-postgres: up
	@echo "Seeding PostgreSQL..."
	docker compose run --build --rm tester --mode=seed --db=postgres

# Заполняет Cassandra данными
seed-cassandra: up
	@echo "Seeding Cassandra..."
	docker compose run --build --rm tester --mode=seed --db=cassandra

# Заполняет MongoDB данными
seed-mongo: up
	@echo "Seeding MongoDB..."
	docker compose run --build --rm tester --mode=seed --db=mongo

# Заполняет etcd данными
seed-etcd: up
	@echo "Seeding etcd..."
	docker compose run --build --rm tester --mode=seed --db=etcd

# --- TEST TARGETS ---
# Запускает тест на чтение для PostgreSQL
test-postgres:
	@echo "Starting PostgreSQL read test. View results at http://localhost:3000"
	docker compose run --build --rm tester --mode=test --db=postgres

# Запускает тест на чтение для Cassandra
test-cassandra:
	@echo "Starting Cassandra read test. View results at http://localhost:3000"
	docker compose run --build --rm tester --mode=test --db=cassandra

# Запускает тест на чтение для MongoDB
test-mongo:
	@echo "Starting MongoDB read test. View results at http://localhost:3000"
	docker compose run --build --rm tester --mode=test --db=mongo

# Запускает тест на чтение для etcd
test-etcd:
	@echo "Starting etcd read test. View results at http://localhost:3000"
	docker compose run --build --rm tester --mode=test --db=etcd

# Открывает логи тестера
logs:
	docker compose logs -f tester