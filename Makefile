.PHONY: sec-up sec-down sec-seed sec-logs

sec-up:
	mkdir -p .dev/spire/server
	mkdir -p .dev/spire/agent
	cp infra/spire/server/server.conf .dev/spire/server/server.conf
	cp infra/spire/agent/agent.conf .dev/spire/agent/agent.conf
	perl -0pi -e 's/(agent \{\n)/$1  insecure_bootstrap = true\n/' .dev/spire/agent/agent.conf
	docker compose -f docker-compose.spire.yaml up -d spire-server
	docker compose -f docker-compose.vault.yaml up -d

sec-down:
	docker compose -f docker-compose.spire.yaml down
	docker compose -f docker-compose.vault.yaml down

sec-seed:
	bash ./scripts/sec-seed.sh

sec-logs:
	docker compose -f docker-compose.spire.yaml logs -f
