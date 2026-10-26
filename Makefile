# Raccourcis de développement. La production passe par docker compose (voir README).
.PHONY: secrets up down logs test test-backend test-web dev-api dev-orchestrator dev-web ca

secrets:
	./scripts/init-secrets.sh

up: secrets
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f --tail=100

# Autorité interne de Caddy, à faire connaître aux postes (navigateur, Claude Code).
ca:
	docker compose cp caddy:/data/caddy/pki/authorities/local/root.crt ./mammomesh-root.crt
	@echo "Certificat racine copié dans ./mammomesh-root.crt"

test: test-backend test-web

# DATABASE_URL_TEST active le test de bout en bout (la base désignée est remise à zéro).
test-backend:
	cd backend && go vet ./... && go test -count=1 ./...

test-web:
	cd web && npx nuxt typecheck && npm run build

dev-api:
	cd backend && go run ./cmd/mammomesh api

dev-orchestrator:
	cd backend && go run ./cmd/mammomesh orchestrator

dev-web:
	cd web && npm run dev
