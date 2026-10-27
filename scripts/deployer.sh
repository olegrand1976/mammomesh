#!/usr/bin/env bash
# Déploiement local en 3 étapes, arrêt à la première erreur :
#   1. tests du backend Go (make test-backend) ;
#   2. reconstruction et redémarrage de la pile Docker (scripts/demarrer-pile.sh --build),
#      puis redémarrage du veilleur d'entraînement s'il est installé ;
#   3. envoi des commits sur le dépôt distant (git push).
#
#   ./scripts/deployer.sh               les 3 étapes
#   ./scripts/deployer.sh --sans-tests  saute l'étape 1
#   ./scripts/deployer.sh --sans-push   saute l'étape 3
#   make deployer [DEPLOYER_OPTS="--sans-push"]
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

tests=1 push=1
for arg in "$@"; do
  case "$arg" in
    --sans-tests) tests=0 ;;
    --sans-push)  push=0 ;;
    -h|--help)    sed -n '2,11p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "Option inconnue : $arg (voir --help)" >&2; exit 2 ;;
  esac
done

etape() { printf '\n\033[1m== %s ==\033[0m\n' "$*"; }
debut=$SECONDS

# Modifications non commitées : elles seront déployées mais pas poussées.
if [[ -n "$(git status --porcelain --untracked-files=no)" ]]; then
  echo "Attention : des fichiers suivis ont des modifications non commitées (déployées, mais pas envoyées par git push)."
  git status --short --untracked-files=no
fi

if (( tests )); then
  etape "1/3 Tests du backend"
  command -v go >/dev/null || { echo "Go introuvable : installe-le ou relance avec --sans-tests." >&2; exit 1; }
  make test-backend
else
  etape "1/3 Tests du backend : sautés"
fi

etape "2/3 Pile Docker (reconstruction)"
./scripts/demarrer-pile.sh --build
if systemctl --user cat mammomesh-entrainement.service >/dev/null 2>&1; then
  systemctl --user restart mammomesh-entrainement.service
  echo "Veilleur d'entraînement redémarré ($(systemctl --user is-active mammomesh-entrainement.service))."
else
  echo "Veilleur d'entraînement non installé (scripts/installer-services.sh) : rien à redémarrer."
fi

if (( push )); then
  etape "3/3 Envoi sur le dépôt distant"
  branche="$(git rev-parse --abbrev-ref HEAD)"
  if git rev-parse --abbrev-ref --symbolic-full-name '@{u}' >/dev/null 2>&1; then
    git fetch --quiet
    avance="$(git rev-list --count '@{u}..HEAD')" retard="$(git rev-list --count 'HEAD..@{u}')"
    if (( retard > 0 )); then
      echo "La branche $branche a $retard commit(s) de retard sur le distant : fais « git pull --rebase » puis relance." >&2
      exit 1
    fi
    if (( avance == 0 )); then
      echo "Rien à envoyer : $branche est à jour."
    else
      git push
    fi
  else
    git push -u origin "$branche"
  fi
else
  etape "3/3 Envoi sur le dépôt distant : sauté"
fi

etape "Terminé en $(( SECONDS - debut )) s"
