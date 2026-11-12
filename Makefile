.PHONY: test test-integration test-db-up test-db-down

test:
	cd server && go test ./...

test-integration: test-db-up
	bash scripts/test-integration.sh

test-db-up:
	docker compose -f docker-compose.test.yml up -d --wait --wait-timeout 60 postgres

test-db-down:
	docker compose -f docker-compose.test.yml down
