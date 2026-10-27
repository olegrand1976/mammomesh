#!/usr/bin/env bash
# Démarre (ou complète) la pile Docker locale avec les profils du matériel. Idempotent.
# Lancé au démarrage de la machine par le service utilisateur mammomesh-pile (scripts/installer-services.sh).
#   ./scripts/demarrer-pile.sh            sans reconstruire les images
#   ./scripts/demarrer-pile.sh --build    en les reconstruisant (comme make up)
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

# Au démarrage, Docker peut ne pas être encore prêt : jusqu'à 5 minutes d'attente.
for _ in $(seq 1 60); do
  docker info >/dev/null 2>&1 && break
  sleep 5
done
docker info >/dev/null 2>&1 || { echo "Docker ne répond pas (service arrêté, ou compte hors du groupe docker)" >&2; exit 1; }

profils="$(./scripts/detecter-materiel.sh)"
echo "Profils compose : ${profils:-aucun}"
COMPOSE_PROFILES="$profils" docker compose up -d ${1:+"$1"}
docker compose ps --format 'table {{.Service}}\t{{.State}}\t{{.Status}}'
