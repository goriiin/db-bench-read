.PHONY: all up down logs prune \
        seed-postgres seed-cassandra seed-mongo seed-etcd \
        test-postgres test-cassandra test-mongo test-etcd

# ВНИМАНИЕ: Эта команда удалит ВСЕ контейнеры и неиспользуемые Docker-объекты (образы, сети, volumes) в вашей системе.
prune:
	@echo "Stopping and removing ALL Docker containers..."
	@if [ -n "$$(docker ps -a -q)" ]; then docker rm -f $$(docker ps -a -q); else echo "No containers to remove."; fi
	@echo "Pruning Docker system (all unused containers, networks, images, volumes)..."
	docker system prune --volumes -a -f

# Запускает все сервисы проекта в фоновом режиме
up:
	@echo "Starting project services..."
	docker compose up -d

# Останавливает и удаляет сервисы, определенные в docker-compose.yml
down:
	@echo "Stopping project services..."
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
	docker compose logs -f tester