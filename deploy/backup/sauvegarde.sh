#!/bin/sh
# Sauvegarde de PostgreSQL dans un dépôt restic (profil compose « backup »).
#   sauvegarde quotidienne   chaque jour à BACKUP_HOUR h (heure de TZ) ; au démarrage aussi si le dépôt est vide
#   sauvegarde une           une seule sauvegarde, tout de suite : docker compose run --rm backup une
#   sauvegarde copie         copie vers RESTIC_COPIE seulement
# Rétention : GARDER_JOURS quotidiennes, GARDER_SEMAINES hebdomadaires, GARDER_MOIS mensuelles.
# RESTIC_COPIE : second dépôt hors de la machine (NAS monté à la demande). Après chaque sauvegarde, et au
# démarrage, les instantanés qui lui manquent y sont copiés ; s'il est injoignable, ils le seront au passage suivant.
# Les images (SeaweedFS) ne sont pas comprises (docs/guide-images.md). Restauration : README.md de ce dossier.
set -eu

: "${BACKUP_HOUR:=2}" "${GARDER_JOURS:=7}" "${GARDER_SEMAINES:=4}" "${GARDER_MOIS:=12}"
: "${RESTIC_REPOSITORY:?}" "${RESTIC_PASSWORD_FILE:?}" "${PGDATABASE:?}"
PGPASSWORD="$(cat "${PGPASSWORD_FILE:?}")"
export PGPASSWORD

journal() { echo "$(date '+%Y-%m-%d %H:%M:%S') $*"; }

preparer() {
  until pg_isready -q; do sleep 2; done
  if ! restic cat config >/dev/null 2>&1; then
    mkdir -p "$RESTIC_REPOSITORY"
    restic init
    journal "dépôt restic créé dans $RESTIC_REPOSITORY"
  fi
}

# Sans instantané si pg_dump échoue (--stdin-from-command).
sauvegarder() {
  journal "sauvegarde de la base $PGDATABASE"
  restic backup --quiet --tag postgres --stdin-filename "$PGDATABASE.dump" \
    --stdin-from-command -- pg_dump --format=custom "$PGDATABASE" || return 1
  restic forget --quiet --tag postgres --keep-daily "$GARDER_JOURS" --keep-weekly "$GARDER_SEMAINES" \
    --keep-monthly "$GARDER_MOIS" --prune || return 1
  restic check --quiet || return 1
  journal "sauvegarde terminée ($(restic snapshots --tag postgres --compact | tail -n 1))"
  copier || journal "ÉCHEC de la copie vers $RESTIC_COPIE ; nouvel essai au passage suivant"
}

copier() {
  [ -n "${RESTIC_COPIE:-}" ] || return 0
  # Essai d'écriture : un dossier qui existe déjà ne prouve pas qu'on peut y écrire, et restic insisterait.
  if ! timeout 60 sh -c 'mkdir -p "$1" && touch "$1/.essai" && rm "$1/.essai"' _ "$RESTIC_COPIE" 2>/dev/null; then
    journal "copie : $RESTIC_COPIE injoignable, rattrapage au passage suivant"
    return 0
  fi
  if ! restic -r "$RESTIC_COPIE" cat config >/dev/null 2>&1; then
    restic -r "$RESTIC_COPIE" init --from-repo "$RESTIC_REPOSITORY" --from-password-file "$RESTIC_PASSWORD_FILE" \
      --copy-chunker-params >/dev/null || return 1
    journal "dépôt de copie créé dans $RESTIC_COPIE"
  fi
  restic -r "$RESTIC_COPIE" copy --quiet --tag postgres --from-repo "$RESTIC_REPOSITORY" \
    --from-password-file "$RESTIC_PASSWORD_FILE" || return 1
  restic -r "$RESTIC_COPIE" forget --quiet --tag postgres --keep-daily "$GARDER_JOURS" \
    --keep-weekly "$GARDER_SEMAINES" --keep-monthly "$GARDER_MOIS" --prune || return 1
  restic -r "$RESTIC_COPIE" check --quiet || return 1
  journal "copie à jour dans $RESTIC_COPIE ($(restic -r "$RESTIC_COPIE" snapshots --tag postgres --compact | tail -n 1))"
}

# Secondes jusqu'au prochain BACKUP_HOUR:00.
attente() {
  maintenant=$(date +%s)
  cible=$(date -d "$(date +%Y-%m-%d) $(printf '%02d' "$BACKUP_HOUR"):00" +%s)
  [ "$cible" -gt "$maintenant" ] || cible=$((cible + 86400))
  echo $((cible - maintenant))
}

trap 'journal "arrêt"; exit 0' INT TERM

case "${1:-quotidienne}" in
  une)
    preparer
    sauvegarder ;;
  copie)
    copier ;;
  quotidienne)
    case "$BACKUP_HOUR" in [0-9]|1[0-9]|2[0-3]) ;; *) echo "BACKUP_HOUR : heure de 0 à 23" >&2; exit 2 ;; esac
    preparer
    if ! restic snapshots --tag postgres --json | grep -q '"id"'; then
      sauvegarder || journal "ÉCHEC de la première sauvegarde"
    else
      copier || journal "ÉCHEC de la copie vers $RESTIC_COPIE ; nouvel essai au passage suivant"
    fi
    while :; do
      s=$(attente)
      journal "prochaine sauvegarde dans $((s / 3600)) h $((s % 3600 / 60)) min"
      sleep "$s" & wait $!
      sauvegarder || journal "ÉCHEC de la sauvegarde ; nouvel essai demain"
    done ;;
  *)
    echo "usage : sauvegarde [quotidienne | une | copie]" >&2; exit 2 ;;
esac
