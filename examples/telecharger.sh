#!/usr/bin/env bash
# Télécharge les DICOM d'exemple décrits dans manifeste.json (sources libres, voir README.md).
#   ./telecharger.sh               fichiers GitHub (vérifiés par SHA-256) puis séries TCIA
#   ./telecharger.sh --sans-tcia   fichiers GitHub seulement (~22 Mo)
#   ./telecharger.sh --tcia-seulement
# Les fichiers déjà présents et intègres ne sont pas retéléchargés. Dépendances : curl, python3, unzip.
#
# Banque d'entraînement (jalon 0) : collections complètes, plusieurs centaines de Go.
#   ./telecharger.sh --banque [tout | CBIS-DDSM | CMMD | NLBS]
#   BANQUE_DIR   dossier de destination (défaut : ./banque à la racine du dépôt ; prévoir un disque dédié)
#   PARALLELE    séries téléchargées en même temps (défaut : 4)
#   MARGE_GO     espace à laisser libre sur le disque de destination (défaut : 10)
# Reprise : relancer la même commande ; les séries terminées (marqueur .ok) sont sautées.
set -euo pipefail

ICI="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFESTE="$ICI/manifeste.json"
NBIA="${NBIA_URL:-https://services.cancerimagingarchive.net/nbia-api/services/v1}"
GITHUB=1
TCIA=1
BANQUE=""
usage() { echo "usage : $0 [--sans-tcia | --tcia-seulement | --banque [tout|CBIS-DDSM|CMMD|NLBS]]" >&2; exit 2; }
case "${1:-}" in
  --sans-tcia) TCIA=0 ;;
  --tcia-seulement) GITHUB=0 ;;
  --banque) BANQUE="${2:-tout}" ;;
  "") ;;
  *) usage ;;
esac

for outil in curl python3; do
  command -v "$outil" >/dev/null || { echo "outil manquant : $outil" >&2; exit 1; }
done
if { [ "$TCIA" = 1 ] || [ -n "$BANQUE" ]; } && ! command -v unzip >/dev/null; then
  echo "outil manquant pour TCIA : unzip (ou lancez avec --sans-tcia)" >&2; exit 1
fi

sha() { if command -v sha256sum >/dev/null; then sha256sum "$1" | cut -d' ' -f1; else shasum -a 256 "$1" | cut -d' ' -f1; fi; }
json() { python3 -I -c "import json,sys; d=json.load(open(sys.argv[1])); $1" "$MANIFESTE"; }
erreurs=0

# ---------- Banque d'entraînement (jalon 0) ----------
# L'espace libre est contrôlé avant chaque collection, puis avant chaque série : rien ne démarre
# si le disque de destination ne peut pas tout recevoir en gardant MARGE_GO libres.
if [ -n "$BANQUE" ]; then
  BANQUE_DIR="${BANQUE_DIR:-$(dirname "$ICI")/banque}"
  PARALLELE="${PARALLELE:-4}"
  MARGE_GO="${MARGE_GO:-10}"
  case "$PARALLELE$MARGE_GO" in *[!0-9]*) echo "PARALLELE et MARGE_GO doivent être des entiers" >&2; exit 2 ;; esac
  mkdir -p "$BANQUE_DIR"
  BANQUE_DIR="$(cd "$BANQUE_DIR" && pwd)"
  export NBIA BANQUE_DIR MARGE_GO

  libre_octets() { echo $(( $(df -Pk "$1" | awk 'NR==2 {print $4}') * 1024 )); }
  en_go() { awk -v x="$1" 'BEGIN { if (x >= 1e9) printf "%.1f Go\n", x / 1e9; else printf "%.0f Mo\n", x / 1e6 }'; }
  # assez_de_place <octets à écrire> <libellé> : refuse si l'écriture laisserait moins de MARGE_GO libres.
  assez_de_place() {
    local libre besoin
    libre=$(libre_octets "$BANQUE_DIR"); besoin=$(( $1 + MARGE_GO * 1000000000 ))
    echo "  espace    $2 : $(en_go "$1") à écrire, $(en_go "$libre") libres sur $BANQUE_DIR (marge $MARGE_GO Go)"
    if [ "$libre" -lt "$besoin" ]; then
      echo "  REFUS     $2 : il manque $(en_go $(( besoin - libre ))). Libérez de la place ou choisissez un autre disque (BANQUE_DIR=...)." >&2
      return 1
    fi
  }

  # Une série TCIA : zip téléchargé sur le disque de la banque, nombre d'images vérifié, marqueur .ok à la fin.
  telecharger_serie() {
    local c="$1" p="$2" uid="$3" n="$4" taille="$5"
    local base="$BANQUE_DIR/tcia/$c" dest="$BANQUE_DIR/tcia/$c/$p/$uid" tmp got libre
    [ -f "$dest/.ok" ] && return 0
    [ -f "$base/.arret" ] && return 0
    libre=$(( $(df -Pk "$BANQUE_DIR" | awk 'NR==2 {print $4}') * 1024 ))
    if [ "$libre" -lt $(( 2 * taille + MARGE_GO * 1000000000 )) ]; then
      echo "espace insuffisant avant $p/$uid" > "$base/.arret"
      echo "$p $uid espace" >> "$base/.echecs"; echo "  ARRÊT     $c : espace disque insuffisant" >&2; return 0
    fi
    tmp="$(mktemp -d "$BANQUE_DIR/.tmp/serie.XXXXXX")"
    if curl -fsS --retry 5 --retry-delay 15 --connect-timeout 30 -G "$NBIA/getImage" \
         --data-urlencode "SeriesInstanceUID=$uid" -o "$tmp/s.zip" && unzip -qo "$tmp/s.zip" -d "$tmp/z"; then
      got=$(find "$tmp/z" -type f -name '*.dcm' | wc -l | tr -d ' ')
      if [ "$got" -gt 0 ] && { [ "$n" = 0 ] || [ "$got" = "$n" ]; }; then
        rm -rf "$dest"; mkdir -p "$dest"
        find "$tmp/z" -type f -name '*.dcm' -exec mv {} "$dest/" \;
        if [ "$(find "$dest" -type f -name '*.dcm' | wc -l | tr -d ' ')" = "$got" ]; then
          echo "$got" > "$dest/.ok"; echo "  ok        $c/$p ($got image(s))"
        else
          echo "$p $uid noms" >> "$base/.echecs"; echo "  ÉCHEC     $c/$p/$uid (noms de fichiers en double)" >&2
        fi
      else
        echo "$p $uid nombre" >> "$base/.echecs"; echo "  ÉCHEC     $c/$p/$uid ($got image(s) reçue(s), $n attendue(s))" >&2
      fi
    else
      echo "$p $uid reseau" >> "$base/.echecs"; echo "  ÉCHEC     $c/$p/$uid (téléchargement)" >&2
    fi
    rm -rf "$tmp"
  }
  export -f telecharger_serie

  banque_tcia() {  # <nom> <collection> <taille_go> <url de métadonnées>...
    local nom="$1" c="$2" taille_go="$3"; shift 3
    local base="$BANQUE_DIR/tcia/$c" stats total faits reste sans_taille a_faire url
    echo "== $nom (TCIA, collection complète)"
    mkdir -p "$base/metadonnees" "$BANQUE_DIR/.tmp"
    rm -f "$base/.arret" "$base/.echecs"
    if ! curl -fsS --retry 3 --connect-timeout 30 -G "$NBIA/getSeries" --data-urlencode "Collection=$c" \
         --data-urlencode "format=json" -o "$base/series.json.part"; then
      echo "  ÉCHEC     $nom : liste des séries (TCIA injoignable ?)" >&2; return 1
    fi
    mv "$base/series.json.part" "$base/series.json"
    # Liste de travail : séries sans marqueur .ok. Identifiants assainis (données externes).
    stats=$(python3 -I -c '
import json, os, re, sys
base, sortie = sys.argv[1], sys.argv[2]
propre = lambda v: re.sub(r"[^A-Za-z0-9._-]", "_", str(v or "inconnu")).lstrip(".") or "inconnu"
series = json.load(open(os.path.join(base, "series.json")))
faits = reste = sans_taille = 0
with open(sortie, "w") as f:
    for s in series:
        p, uid = propre(s.get("PatientID")), propre(s.get("SeriesInstanceUID"))
        if os.path.exists(os.path.join(base, p, uid, ".ok")):
            faits += 1; continue
        taille = int(float(s.get("FileSize") or 0))
        n = int(s.get("ImageCount") or 0)
        sans_taille += taille == 0
        reste += taille
        f.write(f"{sys.argv[3]} {p} {uid} {n} {taille}\n")
print(len(series), faits, reste, sans_taille)' "$base" "$base/.a_faire" "$c")
    read -r total faits reste sans_taille <<<"$stats"
    a_faire=$(( total - faits ))
    echo "  séries    $total au total, $faits déjà là, $a_faire à télécharger"
    # Taille inconnue pour certaines séries : estimation au prorata de la taille publiée de la collection.
    if [ "$sans_taille" -gt 0 ]; then
      reste=$(python3 -I -c "import sys; t,f,r,s,g=map(float,sys.argv[1:]); print(int(max(r, g*1e9*(t-f)/t if t else 0)))" \
        "$total" "$faits" "$reste" "$sans_taille" "$taille_go")
    fi
    if [ "$a_faire" -gt 0 ]; then
      assez_de_place "$reste" "$nom" || return 1
      xargs -P "$PARALLELE" -n 5 bash -c 'telecharger_serie "$@"' _ < "$base/.a_faire" || true
    fi
    for url in "$@"; do
      [ -f "$base/metadonnees/$(basename "$url")" ] && continue
      curl -fsSL --retry 3 -o "$base/metadonnees/$(basename "$url").part" "$url" \
        && mv "$base/metadonnees/$(basename "$url").part" "$base/metadonnees/$(basename "$url")" \
        || { echo "  ÉCHEC     $nom : métadonnées $(basename "$url")" >&2; echo "meta $url" >> "$base/.echecs"; }
    done
    rm -f "$base/.a_faire"
    if [ -f "$base/.arret" ]; then
      echo "  ARRÊTÉ    $nom : espace disque insuffisant en cours de route ; libérez de la place puis relancez." >&2; return 1
    fi
    if [ -s "$base/.echecs" ]; then
      echo "  INCOMPLET $nom : $(wc -l < "$base/.echecs" | tr -d ' ') échec(s) listé(s) dans $base/.echecs ; relancez pour reprendre." >&2; return 1
    fi
    echo "  terminé   $nom : $total série(s) dans $base"
  }

  banque_frdr() {  # <nom> <taille_go> <page> <globus> : téléchargement par Globus, puis extraction ici.
    local nom="$1" taille_go="$2" page="$3" globus="$4"
    local base="$BANQUE_DIR/nlbs" a besoin=0 extrait
    local -a archives=()
    echo "== $nom (FRDR, téléchargement par Globus)"
    mkdir -p "$base/archives" "$base/donnees"
    # Volumes à extraire : archives .7z simples et premiers volumes .7z.001.
    while IFS= read -r a; do archives+=("$a"); done \
      < <(find "$base/archives" -maxdepth 1 -type f \( -name '*.7z' -o -name '*.7z.001' \) | sort)
    if [ "${#archives[@]}" = 0 ]; then
      echo "  à faire   FRDR ne propose pas de lien direct stable. Avant de lancer le transfert :"
      assez_de_place $(( taille_go * 1000000000 )) "$nom (archives, estimation)" \
        || echo "            Ne lancez pas le transfert avant d'avoir libéré cette place (l'extraction en demandera davantage)." >&2
      echo "            1. ouvrir $page (version 3) ;"
      echo "            2. transférer les archives .7z et le CSV par Globus ($globus)"
      echo "               ou « Download as Zip » (puis dézipper), vers $base/archives ;"
      echo "            3. relancer : $0 --banque NLBS (contrôle de place, extraction, reprise)."
      return 1
    fi
    command -v 7z >/dev/null || command -v 7za >/dev/null \
      || { echo "  outil manquant : 7z (paquet p7zip-full)" >&2; return 1; }
    local sz; sz=$(command -v 7z || command -v 7za)
    # Contrôle de place : somme des tailles décompressées des archives pas encore extraites.
    for a in "${archives[@]}"; do
      [ -f "$base/donnees/.ok_$(basename "$a")" ] && continue
      extrait=$("$sz" l -slt -ba "$a" | awk -F' = ' '$1=="Size" {s+=$2} END {printf "%d", s}')
      besoin=$(( besoin + extrait ))
    done
    if [ "$besoin" -gt 0 ]; then assez_de_place "$besoin" "$nom (extraction)" || return 1; fi
    for a in "${archives[@]}"; do
      [ -f "$base/donnees/.ok_$(basename "$a")" ] && { echo "  déjà là   $(basename "$a")"; continue; }
      if "$sz" x -y -bd -o"$base/donnees" "$a" >/dev/null; then
        touch "$base/donnees/.ok_$(basename "$a")"; echo "  ok        $(basename "$a")"
      else
        echo "  ÉCHEC     $(basename "$a") (extraction)" >&2; return 1
      fi
    done
    for a in "$base/archives"/*.csv; do
      [ -f "$a" ] && [ ! -e "$base/donnees/$(basename "$a")" ] && cp "$a" "$base/donnees/"
    done
    echo "  terminé   $nom : $(find "$base/donnees" -type f -name '*.dcm' | wc -l | tr -d ' ') image(s) dans $base/donnees"
    echo "            (les archives restent dans $base/archives : à supprimer vous-même pour récupérer la place)"
  }

  choix="$(echo "$BANQUE" | tr '[:lower:]' '[:upper:]')"
  trouve=0
  while IFS=$'\t' read -r nom source collection taille_go page globus metas; do
    [ "$choix" = "TOUT" ] || [ "$choix" = "$(echo "$nom" | tr '[:lower:]' '[:upper:]')" ] || continue
    trouve=1
    case "$source" in
      tcia) # shellcheck disable=SC2086
            banque_tcia "$nom" "$collection" "$taille_go" $metas || erreurs=$((erreurs + 1)) ;;
      frdr) banque_frdr "$nom" "$taille_go" "$page" "$globus" || erreurs=$((erreurs + 1)) ;;
    esac
  done < <(json 'print("\n".join("\t".join((e["nom"], e["source"], e.get("collection", "-"), str(int(e["taille_go"] + 0.999)), e.get("page", "-"), e.get("globus", "-"), " ".join(e.get("metadonnees", [])))) for e in d["banque"]))')
  [ "$trouve" = 1 ] || { echo "collection inconnue : $BANQUE" >&2; usage; }
  rmdir "$BANQUE_DIR/.tmp" 2>/dev/null || true
  if [ "$erreurs" -gt 0 ]; then echo "$erreurs collection(s) incomplète(s) : relancez la même commande pour reprendre." >&2; exit 1; fi
  echo "Banque à jour dans $BANQUE_DIR. Citez les sources (voir examples/README.md)."
  exit 0
fi

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
