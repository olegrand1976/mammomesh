#!/usr/bin/env bash
# Active les tests d'images par le CNN entraîné, de bout en bout, et vérifie chaque maillon :
#   1. moteur : engine/.venv avec PyTorch, au moins un modele.pt dans modeles/ ;
#   2. service utilisateur mammomesh-inference (installé, démarré, /sante répond sur docker0) ;
#   3. API reconstruite et redémarrée (INFERENCE_URL), puis vérification depuis son conteneur.
#
#   ./scripts/activer-inference.sh               les 3 étapes
#   ./scripts/activer-inference.sh --sans-build  redémarre l'API sans reconstruire son image
#   ./scripts/activer-inference.sh --retirer     arrête et retire le service (tests de nouveau simulés si INFERENCE_URL=)
#   make inference [INFERENCE_OPTS="--sans-build"]
# Variables : MODELE=cnn-cmmd-… (modèle imposé), INFERENCE_APPAREIL=cuda, INFERENCE_PORT (défaut 8765).
set -euo pipefail
DEPOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$DEPOT"
DEST="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
UNITE=mammomesh-inference
PORT="${INFERENCE_PORT:-8765}"
PY="$DEPOT/engine/.venv/bin/python"

etape() { printf '\n\033[1m== %s ==\033[0m\n' "$*"; }
echec() { echo "  ✗ $*" >&2; exit 1; }
ok() { echo "  ✓ $*"; }

build=1
for arg in "$@"; do
  case "$arg" in
    --sans-build) build=0 ;;
    --retirer)
      systemctl --user disable --now "$UNITE.service" 2>/dev/null || true
      rm -f "$DEST/$UNITE.service"
      systemctl --user daemon-reload
      echo "Service $UNITE retiré. Pour des tests simulés : INFERENCE_URL= dans .env, puis docker compose up -d api."
      exit 0 ;;
    -h|--help) sed -n '2,12p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "Option inconnue : $arg (voir --help)" >&2; exit 2 ;;
  esac
done

etape "1/3 Moteur"
[[ -x "$PY" ]] || echec "engine/.venv absent : cd engine && python3 -m venv .venv && .venv/bin/pip install -e '.[cnn]' (voir README, « Premier CNN »)"
"$PY" -c "import torch, torchvision, numpy, PIL" 2>/dev/null \
  || echec "PyTorch absent de engine/.venv : engine/.venv/bin/pip install -e 'engine[cnn]'"
ok "PyTorch $("$PY" -c 'import torch; print(torch.__version__)')"
# Même dossier des modèles que la pile Docker (MODELES_DIR_HOTE, dans l'environnement ou dans .env).
[[ -n "${MODELES_DIR_HOTE:-}" ]] || MODELES_DIR_HOTE=$(grep -E '^MODELES_DIR_HOTE=' .env 2>/dev/null | cut -d= -f2- || true)
modeles="${MODELES_DIR_HOTE:-$DEPOT/modeles}"
[[ -d "$modeles" ]] || echec "dossier des modèles introuvable : $modeles"
if [[ -n "${MODELE:-}" ]]; then
  [[ -f "$modeles/$MODELE/modele.pt" ]] || echec "$modeles/$MODELE/modele.pt introuvable"
  ok "modèle imposé : $MODELE"
else
  n=$(find "$modeles" -mindepth 2 -maxdepth 2 -name modele.pt | wc -l)
  (( n > 0 )) || echec "aucun modele.pt dans $modeles : lance d'abord un entraînement (menu Entraînements)"
  ok "$n modèle(s) dans $modeles (le plus récent fini sera chargé)"
fi
hote=$(ip -4 -o addr show docker0 2>/dev/null | awk '{print $4}' | cut -d/ -f1)
[[ -n "$hote" ]] || echec "interface docker0 introuvable : Docker est-il installé et démarré ?"
ok "adresse d'écoute : $hote:$PORT (docker0, invisible depuis le réseau local)"

etape "2/3 Service $UNITE"
command -v systemctl >/dev/null || echec "systemd introuvable"
[[ "$DEPOT" != *" "* ]] || echec "le chemin du dépôt contient une espace : non pris en charge par les unités"
mkdir -p "$DEST"
sed -e "s#@DEPOT@#$DEPOT#g" "$DEPOT/deploy/systemd/$UNITE.service" > "$DEST/$UNITE.service"
# Réglages facultatifs, ajoutés à l'unité s'ils sont donnés.
{
  [[ -n "${MODELES_DIR_HOTE:-}" ]] && echo "Environment=MODELES_DIR=$MODELES_DIR_HOTE"
  [[ -n "${MODELE:-}" ]] && echo "Environment=MODELE=$MODELE"
  [[ -n "${INFERENCE_APPAREIL:-}" ]] && echo "Environment=INFERENCE_APPAREIL=$INFERENCE_APPAREIL"
  [[ -n "${INFERENCE_PORT:-}" ]] && echo "Environment=INFERENCE_PORT=$INFERENCE_PORT"
  jeton=$(grep -E '^INFERENCE_JETON=' .env 2>/dev/null | cut -d= -f2- || true)
  [[ -n "$jeton" ]] && echo "Environment=INFERENCE_JETON=$jeton"
  true
} > /tmp/mammomesh-inference.env.$$
if [[ -s /tmp/mammomesh-inference.env.$$ ]]; then
  sed -i "/^ExecStart=/e cat /tmp/mammomesh-inference.env.$$" "$DEST/$UNITE.service"
fi
rm -f /tmp/mammomesh-inference.env.$$
systemctl --user daemon-reload
systemctl --user enable "$UNITE.service" >/dev/null
systemctl --user restart "$UNITE.service"
ok "installé dans $DEST, activé au démarrage"
echo "  chargement du modèle…"
for _ in $(seq 1 60); do
  if curl -fsS --max-time 2 ${jeton:+-H "Authorization: Bearer $jeton"} "http://$hote:$PORT/sante" -o /tmp/mammomesh-sante.$$ 2>/dev/null; then
    break
  fi
  if ! systemctl --user is-active --quiet "$UNITE.service"; then
    journalctl --user -u "$UNITE" -n 20 --no-pager >&2 || true
    echec "le service s'est arrêté (journal ci-dessus)"
  fi
  sleep 1
done
[[ -s /tmp/mammomesh-sante.$$ ]] || { journalctl --user -u "$UNITE" -n 20 --no-pager >&2 || true; echec "http://$hote:$PORT/sante ne répond pas après 60 s"; }
ok "répond : $(cat /tmp/mammomesh-sante.$$)"
rm -f /tmp/mammomesh-sante.$$

etape "3/3 API"
if grep -qE '^INFERENCE_URL=$' .env 2>/dev/null; then
  echec "INFERENCE_URL est vide dans .env (tests simulés) : retire cette ligne ou mets http://host.docker.internal:$PORT"
fi
if [[ "$PORT" != 8765 ]] && ! grep -qE '^INFERENCE_URL=' .env 2>/dev/null; then
  echo "INFERENCE_URL=http://host.docker.internal:$PORT" >> .env
  ok "INFERENCE_URL ajouté à .env (port $PORT)"
fi
if (( build )); then
  docker compose up -d --build api web
else
  docker compose up -d api
fi
echo "  attente de l'API…"
for _ in $(seq 1 60); do
  [[ "$(docker inspect -f '{{.State.Health.Status}}' "$(docker compose ps -q api)" 2>/dev/null)" == healthy ]] && break
  sleep 2
done
if docker compose exec -T api /usr/local/bin/mammomesh verifier-inference; then
  ok "l'API joint le moteur"
else
  if command -v ufw >/dev/null && ufw status 2>/dev/null | grep -q "Status: active"; then
    echo "  Pare-feu ufw actif : sudo ufw allow in on br+ to any port $PORT proto tcp ; sudo ufw allow in on docker0 to any port $PORT proto tcp" >&2
  fi
  echec "l'API ne joint pas le moteur depuis son conteneur"
fi

etape "Prêt"
echo "Menu Images → choisir une image → « Tester sur cette image (CNN) »."
echo "Journal du moteur : journalctl --user -u $UNITE -f"
