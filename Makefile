.PHONY: sec-up sec-down sec-seed sec-logs

sec-up:
	docker compose -f docker-compose.spire.yaml up -d
	docker compose -f docker-compose.vault.yaml up -d

sec-down:
	docker compose -f docker-compose.spire.yaml down
	docker compose -f docker-compose.vault.yaml down

sec-seed:
	bash ./scripts/sec-seed.sh

sec-logs:
	docker compose -f docker-compose.spire.yaml logs -f
