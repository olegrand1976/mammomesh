#!/usr/bin/env bash
# Test de compression, niveau B : télécharge seulement les séries CBIS-DDSM utiles (image complète + masque
# de la lésion, en CC et en MLO) pour N seins appariés, au lieu des ~164 Go de la collection.
#   scripts/telecharger-cbis-paires.sh [N=200]
# Même disposition et mêmes marqueurs .ok que examples/telecharger.sh : un téléchargement complet ultérieur
# reprendra ces séries sans les refaire. Ensembles de test officiels exclus (règle de la banque).
# Variables : BANQUE_DIR (défaut ./banque), MARGE_GO (10), PARALLELE (4), NBIA_URL.
set -euo pipefail

N="${1:-200}"
ICI="$(cd "$(dirname "$0")/.." && pwd)"
BANQUE_DIR="${BANQUE_DIR:-$ICI/banque}"
MARGE_GO="${MARGE_GO:-10}"
PARALLELE="${PARALLELE:-4}"
NBIA="${NBIA_URL:-https://services.cancerimagingarchive.net/nbia-api/services/v1}"
BASE="$BANQUE_DIR/tcia/CBIS-DDSM"
LISTE="$ICI/cache/audit/cbis-a-telecharger.txt"
case "$N$MARGE_GO$PARALLELE" in *[!0-9]*) echo "N, MARGE_GO et PARALLELE doivent être des entiers" >&2; exit 2 ;; esac

mkdir -p "$BASE" "$BANQUE_DIR/.tmp" "$(dirname "$LISTE")"
if [ ! -s "$BASE/series.json" ]; then
  echo "== liste des séries CBIS-DDSM (TCIA)"
  curl -fsS --retry 3 --connect-timeout 30 -G "$NBIA/getSeries" --data-urlencode "Collection=CBIS-DDSM" \
    --data-urlencode "format=json" -o "$BASE/series.json.part" && mv "$BASE/series.json.part" "$BASE/series.json"
fi

PY="${PYTHON:-python3}"
[ -x "$ICI/engine/.venv/bin/python" ] && PY="$ICI/engine/.venv/bin/python"
(cd "$ICI/engine" && "$PY" -m mammomesh_engine.audit cbis-selection --series "$BASE/series.json" --n "$N" --liste "$LISTE")

# Ce qui reste à faire et sa taille.
read -r a_faire octets < <(awk -v base="$BASE" '{ if (system("test -f \"" base "/" $2 "/" $3 "/.ok\"") != 0) { n++; t += $5 } } END { print n+0, t+0 }' "$LISTE")
libre=$(( $(df -Pk "$BANQUE_DIR" | awk 'NR==2 {print $4}') * 1024 ))
echo "== $a_faire série(s) à télécharger, $(( octets / 1000000 )) Mo ; $(( libre / 1000000000 )) Go libres (marge $MARGE_GO Go)"
if [ "$libre" -lt $(( octets + MARGE_GO * 1000000000 )) ]; then
  echo "REFUS : place insuffisante sur $BANQUE_DIR. Réduisez N ou choisissez un autre disque (BANQUE_DIR=...)." >&2
  exit 1
fi
[ "$a_faire" = 0 ] && { echo "== rien à faire"; exit 0; }

serie() {   # <collection> <patient> <uid> <n images> <taille>
  local p="$2" uid="$3" n="$4" dest tmp got
  dest="$BASE/$p/$uid"
  [ -f "$dest/.ok" ] && return 0
  tmp="$(mktemp -d "$BANQUE_DIR/.tmp/serie.XXXXXX")"
  if curl -fsS --retry 5 --retry-delay 15 --connect-timeout 30 -G "$NBIA/getImage" \
       --data-urlencode "SeriesInstanceUID=$uid" -o "$tmp/s.zip" && unzip -qo "$tmp/s.zip" -d "$tmp/z"; then
    got=$(find "$tmp/z" -type f -name '*.dcm' | wc -l | tr -d ' ')
    if [ "$got" -gt 0 ] && { [ "$n" = 0 ] || [ "$got" = "$n" ]; }; then
      rm -rf "$dest"; mkdir -p "$dest"
      find "$tmp/z" -type f -name '*.dcm' -exec mv {} "$dest/" \;
      echo "$got" > "$dest/.ok"; echo "  ok        $p"
    else
      echo "$p $uid nombre" >> "$BASE/.echecs"; echo "  ÉCHEC     $p ($got image(s), $n attendue(s))" >&2
    fi
  else
    echo "$p $uid reseau" >> "$BASE/.echecs"; echo "  ÉCHEC     $p (téléchargement)" >&2
  fi
  rm -rf "$tmp"
}
export -f serie
export BASE BANQUE_DIR NBIA
xargs -P "$PARALLELE" -n 5 bash -c 'serie "$@"' _ < "$LISTE" || true
echo "== terminé ; ensuite : (cd engine && $PY -m mammomesh_engine.audit lesions)"
