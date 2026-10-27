#!/usr/bin/env bash
# Services utilisateur de MammoMesh (systemd --user), sans sudo sauf indication :
#   mammomesh-pile          : pile Docker au démarrage de la machine ;
#   mammomesh-entrainement  : reprise automatique des entraînements du CNN dès que le GPU est libre.
#   ./scripts/installer-services.sh             installe, active et démarre
#   ./scripts/installer-services.sh --sans-demarrer   active pour le prochain démarrage seulement
#   ./scripts/installer-services.sh --retirer   arrête et retire
# BANQUE_DIR=/disque/banque : banque des entraînements lancés depuis l'interface (défaut : ./banque du dépôt).
set -euo pipefail
DEPOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEST="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
SERVICES=(mammomesh-pile mammomesh-entrainement)
alerte() { echo "  ! $*"; }

DEMARRER=1
[[ "${1:-}" == "--sans-demarrer" ]] && DEMARRER=0
if [[ "${1:-}" == "--retirer" ]]; then
  for s in "${SERVICES[@]}"; do
    systemctl --user disable --now "$s.service" 2>/dev/null || true
    rm -f "$DEST/$s.service"
  done
  systemctl --user daemon-reload
  echo "Services retirés (les conteneurs gardent leur politique de redémarrage)."
  exit 0
fi

[[ "$DEPOT" != *" "* ]] || { echo "Le chemin du dépôt contient une espace : non pris en charge par les unités." >&2; exit 1; }
command -v systemctl >/dev/null || { echo "systemd introuvable" >&2; exit 1; }

echo "== Vérifications"
if [[ "$(systemctl is-enabled docker.service 2>/dev/null)" != enabled ]]; then
  alerte "Docker ne démarre pas avec la machine : sudo systemctl enable docker.service containerd.service"
fi
id -nG | tr ' ' '\n' | grep -qx docker || alerte "ton compte n'est pas dans le groupe docker : sudo usermod -aG docker $USER (puis se reconnecter)"
if [[ "$(loginctl show-user "$USER" -p Linger --value 2>/dev/null)" != yes ]]; then
  # Sans « linger », les services utilisateur attendent l'ouverture de ta session.
  loginctl enable-linger "$USER" 2>/dev/null && echo "  linger activé : les services démarrent avec la machine, sans ouvrir de session" \
    || alerte "sudo loginctl enable-linger $USER   (pour démarrer sans ouvrir de session)"
fi
installer=("${SERVICES[@]}")
if [[ ! -x "$DEPOT/engine/.venv/bin/python" ]]; then
  alerte "engine/.venv absent : mammomesh-entrainement n'est pas installé (voir README, « Premier CNN »)"
  installer=(mammomesh-pile)
fi

echo "== Installation dans $DEST"
mkdir -p "$DEST"
for s in "${installer[@]}"; do
  sed -e "s#@DEPOT@#$DEPOT#g" -e "s#@BANQUE_DIR@#${BANQUE_DIR:-$DEPOT/banque}#g" "$DEPOT/deploy/systemd/$s.service" > "$DEST/$s.service"
  echo "  $s.service"
done
systemctl --user daemon-reload
systemctl --user enable "${installer[@]/%/.service}"
if [[ $DEMARRER == 0 ]]; then
  echo "Services activés : ils démarreront avec la machine."
  exit 0
fi
systemctl --user restart mammomesh-pile.service &   # peut construire des images : on n'attend pas
for s in "${installer[@]}"; do
  [[ "$s" == mammomesh-pile ]] || systemctl --user restart "$s.service"
done

echo "== État"
sleep 3
systemctl --user --no-pager --lines=0 status "${installer[@]/%/.service}" || true
echo
echo "Journaux : journalctl --user -u mammomesh-entrainement -f    (et -u mammomesh-pile)"
