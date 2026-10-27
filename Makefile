# Raccourcis de développement. La production passe par docker compose (voir README).
.PHONY: secrets up down logs test test-backend test-web test-engine e2e dev-api dev-orchestrator dev-web ca exemples banque materiel

secrets:
	./scripts/init-secrets.sh

# Profils compose choisis selon le matériel de l'hôte (scripts/detecter-materiel.sh).
up: secrets
	COMPOSE_PROFILES="$$(./scripts/detecter-materiel.sh)" docker compose up -d --build

# Matériel détecté et plan d'exécution (appareil, précision, lot, inférence).
materiel:
	@profils="$$(./scripts/detecter-materiel.sh)" && echo "Profils compose : $$profils" && \
	cd engine && python3 -m mammomesh_engine.materiel --docker "$$profils"

down:
	docker compose down

logs:
	docker compose logs -f --tail=100

# Autorité interne de Caddy, à faire connaître aux postes (navigateur, Claude Code).
ca:
	docker compose cp caddy:/data/caddy/pki/authorities/local/root.crt ./mammomesh-root.crt
	@echo "Certificat racine copié dans ./mammomesh-root.crt"

test: test-backend test-web test-engine

# Tests unitaires Go ; DATABASE_URL_TEST ajoute les tests d'intégration (base remise à zéro),
# REDIS_URL_TEST ceux qui passent par un vrai Redis.
test-backend:
	cd backend && go vet ./... && go test -count=1 ./...

# Moteur Python : adaptation au matériel (sans PyTorch ni GPU).
test-engine:
	cd engine && python3 -m unittest discover -s tests -t .

# Types, tests unitaires Vitest (stores, middleware), compilation.
test-web:
	cd web && npx nuxt typecheck && npm test && npm run build

# Parcours navigateur (Playwright) sur une pile complète démarrée à part : E2E_DATABASE_URL obligatoire
# (base jetable, remise à zéro), E2E_REDIS_URL facultatif. Navigateur : npx playwright install chromium.
e2e:
	cd backend && CGO_ENABLED=0 go build -o bin/mammomesh ./cmd/mammomesh
	cd web && npm run build && npx playwright test

dev-api:
	cd backend && go run ./cmd/mammomesh api

dev-orchestrator:
	cd backend && go run ./cmd/mammomesh orchestrator

dev-web:
	cd web && npm run dev

# DICOM d'exemple de sources libres (examples/README.md) ; EXEMPLES_OPTS=--sans-tcia pour GitHub seul.
exemples:
	./examples/telecharger.sh $(EXEMPLES_OPTS)

# Banque d'entraînement (jalon 0) : BANQUE=tout|CBIS-DDSM|CMMD|NLBS, BANQUE_DIR=disque dédié.
BANQUE ?= tout
banque:
	./examples/telecharger.sh --banque $(BANQUE)
