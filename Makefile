.PHONY: init up down build test test-integration smoke seed logs clean
init:
	python3 scripts/init_env.py
up: init
	docker compose up --build -d --wait --wait-timeout 600
down:
	docker compose down
build:
	mvn -B package -DskipTests
	go build ./...
test:
	mvn -B test
	go test -race ./...
test-integration:
	mvn -B test
	go test -race -tags=integration ./...
smoke:
	python3 scripts/demo.py
seed:
	python3 scripts/seed.py
logs:
	docker compose logs -f --tail=100
clean:
	docker compose down --volumes --remove-orphans
