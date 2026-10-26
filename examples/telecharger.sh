#!/usr/bin/env bash
# Télécharge les DICOM d'exemple décrits dans manifeste.json (sources libres, voir README.md).
#   ./telecharger.sh               fichiers GitHub (vérifiés par SHA-256) puis séries TCIA
#   ./telecharger.sh --sans-tcia   fichiers GitHub seulement (~22 Mo)
#   ./telecharger.sh --tcia-seulement
# Les fichiers déjà présents et intègres ne sont pas retéléchargés. Dépendances : curl, python3, unzip.
set -euo pipefail

ICI="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFESTE="$ICI/manifeste.json"
NBIA="https://services.cancerimagingarchive.net/nbia-api/services/v1"
GITHUB=1
TCIA=1
case "${1:-}" in
  --sans-tcia) TCIA=0 ;;
  --tcia-seulement) GITHUB=0 ;;
  "") ;;
  *) echo "usage : $0 [--sans-tcia | --tcia-seulement]" >&2; exit 2 ;;
esac

for outil in curl python3; do
  command -v "$outil" >/dev/null || { echo "outil manquant : $outil" >&2; exit 1; }
done
if [ "$TCIA" = 1 ] && ! command -v unzip >/dev/null; then
  echo "outil manquant pour TCIA : unzip (ou lancez avec --sans-tcia)" >&2; exit 1
fi

sha() { if command -v sha256sum >/dev/null; then sha256sum "$1" | cut -d' ' -f1; else shasum -a 256 "$1" | cut -d' ' -f1; fi; }
json() { python3 -I -c "import json,sys; d=json.load(open(sys.argv[1])); $1" "$MANIFESTE"; }
erreurs=0

# ---------- GitHub : URL figées sur un commit, empreinte vérifiée ----------
if [ "$GITHUB" = 1 ]; then
  echo "== Fichiers GitHub"
  while IFS=$'\t' read -r fichier url attendu; do
    dest="$ICI/$fichier"
    if [ -f "$dest" ] && [ "$(sha "$dest")" = "$attendu" ]; then
      echo "  déjà là   $fichier"; continue
    fi
    mkdir -p "$(dirname "$dest")"
    if curl -fsSL --retry 3 -o "$dest.part" "$url" && [ "$(sha "$dest.part")" = "$attendu" ]; then
      mv "$dest.part" "$dest"; echo "  ok        $fichier"
    else
      rm -f "$dest.part"; echo "  ÉCHEC     $fichier (téléchargement ou empreinte)" >&2; erreurs=$((erreurs + 1))
    fi
  done < <(json 'print("\n".join("\t".join((e["fichier"], e["url"], e["sha256"])) for e in d["github"]))')
fi

# ---------- TCIA : API publique NBIA v1 (sans compte) ----------
if [ "$TCIA" = 1 ]; then
  echo "== Séries TCIA (CC BY ; citez les sources, voir README.md)"
  if ! curl -fsS --max-time 20 -o /dev/null "$NBIA/getCollectionValues?format=json"; then
    echo "  TCIA injoignable depuis cette machine : partie ignorée." >&2
    exit $(( erreurs > 0 ))
  fi
  tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
  while IFS=$'\t' read -r collection patient filtre; do
    dossier="$ICI/tcia/$collection"
    if ls "$dossier/${patient}"_*.dcm >/dev/null 2>&1; then
      echo "  déjà là   $collection/$patient"; continue
    fi
    mkdir -p "$dossier"
    if ! curl -fsS --retry 3 -G "$NBIA/getSeries" --data-urlencode "Collection=$collection" \
         --data-urlencode "PatientID=$patient" --data-urlencode "format=json" -o "$tmp/series.json"; then
      echo "  ÉCHEC     $collection/$patient (liste des séries)" >&2; erreurs=$((erreurs + 1)); continue
    fi
    # Séries MG ; pour CBIS-DDSM, seulement l'image complète (pas les découpes ni les masques).
    uids=$(python3 -I -c '
import json, sys
filtre = sys.argv[2].lower()
for s in json.load(open(sys.argv[1])):
    if s.get("Modality") == "MG" and (not filtre or filtre in s.get("SeriesDescription", "").lower()):
        print(s["SeriesInstanceUID"])' "$tmp/series.json" "$filtre")
    if [ -z "$uids" ]; then
      echo "  ÉCHEC     $collection/$patient (aucune série retenue)" >&2; erreurs=$((erreurs + 1)); continue
    fi
    n=0
    for uid in $uids; do
      rm -rf "$tmp/z" && mkdir -p "$tmp/z"
      if ! curl -fsS --retry 3 -G "$NBIA/getImage" --data-urlencode "SeriesInstanceUID=$uid" -o "$tmp/s.zip" \
         || ! unzip -qo "$tmp/s.zip" -d "$tmp/z"; then
        echo "  ÉCHEC     $collection/$patient (série $uid)" >&2; erreurs=$((erreurs + 1)); continue
      fi
      while IFS= read -r f; do
        n=$((n + 1)); mv "$f" "$dossier/${patient}_$n.dcm"
      done < <(find "$tmp/z" -type f -name '*.dcm' | sort)
    done
    echo "  ok        $collection/$patient ($n image(s))"
  done < <(json 'print("\n".join("\t".join((e["collection"], e["patient"], e.get("filtre", ""))) for e in d["tcia"]))')
fi

if [ "$erreurs" -gt 0 ]; then echo "$erreurs échec(s)." >&2; exit 1; fi
echo "Terminé."
