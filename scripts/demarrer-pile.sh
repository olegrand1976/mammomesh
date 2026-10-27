#!/usr/bin/env bash
# Démarre (ou complète) la pile Docker locale avec les profils du matériel. Idempotent.
# Lancé au démarrage de la machine par le service utilisateur mammomesh-pile (scripts/installer-services.sh).
#   ./scripts/demarrer-pile.sh            sans reconstruire les images
#   ./scripts/demarrer-pile.sh --build    en les reconstruisant (comme make up)
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

# Au démarrage, Docker peut ne pas être encore prêt : jusqu'à 5 minutes d'attente.
# Un refus d'accès au socket ne se règle pas en attendant : arrêt immédiat, avec la marche à suivre.
erreur=""
for _ in $(seq 1 60); do
  if erreur="$(docker info 2>&1 >/dev/null)"; then erreur=""; break; fi
  if grep -qi "permission denied" <<<"$erreur"; then
    echo "Accès refusé au socket Docker pour $(id -un) (groupes : $(id -Gn))." >&2
    echo "Si ton compte vient d'être ajouté au groupe docker, le gestionnaire de services utilisateur ne le sait pas" >&2
    echo "encore : redémarre la machine, ou « sudo systemctl restart user@$(id -u).service »." >&2
    exit 1
  fi
  sleep 5
done
[[ -z "$erreur" ]] || { echo "Docker ne répond pas : $erreur" >&2; exit 1; }

# Demandes de l'interface (relancer, supprimer un entraînement) : l'API, qui ne tourne pas sous ton compte,
# doit pouvoir y écrire ; le veilleur de l'hôte les lit et les efface.
mkdir -p modeles/.demandes && chmod 0777 modeles/.demandes

profils="$(./scripts/detecter-materiel.sh)"
echo "Profils compose : ${profils:-aucun}"
COMPOSE_PROFILES="$profils" docker compose up -d ${1:+"$1"}
docker compose ps --format 'table {{.Service}}\t{{.State}}\t{{.Status}}'
