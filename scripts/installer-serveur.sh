#!/usr/bin/env bash
# Installation complète de MammoMesh sur un nouveau serveur, avec détection du matériel.
# Voir docs/installation-serveur.md. Reprenable : chaque étape vérifie ce qui est déjà fait.
#
#   git clone https://github.com/olegrand1976/mammomesh.git && cd mammomesh
#   ./scripts/installer-serveur.sh                 # questions pour le nom du site et la banque
#   ./scripts/installer-serveur.sh --oui           # sans question (valeurs détectées)
#   ./scripts/installer-serveur.sh --simulation    # montre ce qui serait fait, sans rien changer
#
# Options :
#   --site NOM            nom du site sur le réseau local (SITE_ADDRESS) ; défaut : <nom de la machine>.lan
#   --ip ADRESSE          adresse d'écoute (BIND_ADDRESS) ; défaut : adresse de la machine sur le réseau local
#   --profils LISTE       profils compose en plus de ceux du matériel : obs, backup (séparés par des virgules)
#   --accel TYPE          forcer l'accélérateur : nvidia, amd, intel ou cpu (défaut : détecté)
#   --banque NOM          télécharger la banque à la fin : CMMD, CBIS-DDSM, NLBS ou tout (défaut : question)
#   --banque-dir DOSSIER  dossier de la banque (défaut : ./banque) ; l'espace libre est contrôlé avant
#   --sans-pilote         ne pas installer de pilote GPU (seulement le signaler)
#   --sans-moteur         pas d'environnement Python du moteur (entraînement du CNN)
#   --sans-services       pas de services de démarrage automatique
# Variables : TORCH_INDEX (index pip de PyTorch, remplace le choix automatique), MARGE_GO (défaut 10).
set -euo pipefail

DEPOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$DEPOT"

OUI=0 SIMU=0 SITE="" IP="" PROFILS_EN_PLUS="" ACCEL="" BANQUE="" BANQUE_DIR_OPT=""
SANS_PILOTE=0 SANS_MOTEUR=0 SANS_SERVICES=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    --oui) OUI=1 ;;
    --simulation) SIMU=1 ;;
    --site) SITE="${2:?--site NOM}"; shift ;;
    --ip) IP="${2:?--ip ADRESSE}"; shift ;;
    --profils) PROFILS_EN_PLUS="${2:?--profils LISTE}"; shift ;;
    --accel) ACCEL="${2:?--accel TYPE}"; shift ;;
    --banque) BANQUE="${2:?--banque NOM}"; shift ;;
    --banque-dir) BANQUE_DIR_OPT="${2:?--banque-dir DOSSIER}"; shift ;;
    --sans-pilote) SANS_PILOTE=1 ;;
    --sans-moteur) SANS_MOTEUR=1 ;;
    --sans-services) SANS_SERVICES=1 ;;
    -h|--help) sed -n '2,24p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "option inconnue : $1 (voir --help)" >&2; exit 2 ;;
  esac
  shift
done
case "$ACCEL" in ""|nvidia|amd|intel|cpu) ;; *) echo "--accel : nvidia, amd, intel ou cpu" >&2; exit 2 ;; esac

JOURNAL="$DEPOT/installation-$(date +%Y%m%d-%H%M%S).log"
[[ $SIMU == 1 ]] || exec > >(tee -a "$JOURNAL") 2>&1

# ---------- Outils ----------
GRAS=$'\e[1m' VERT=$'\e[32m' JAUNE=$'\e[33m' ROUGE=$'\e[31m' FIN=$'\e[0m'
[[ -t 1 ]] || { GRAS="" VERT="" JAUNE="" ROUGE="" FIN=""; }
NUM=0
etape() { NUM=$((NUM + 1)); echo; echo "${GRAS}== $NUM. $*${FIN}"; }
ok() { echo "  ${VERT}ok${FIN}     $*"; }
info() { echo "         $*"; }
alerte() { echo "  ${JAUNE}!${FIN}      $*"; ALERTES+=("$*"); }
erreur() { echo "  ${ROUGE}ÉCHEC${FIN}  $*" >&2; exit 1; }
ALERTES=() REDEMARRER=0 GROUPE_AJOUTE=0

# run : exécute, ou affiche en simulation. root : avec sudo si besoin.
run() { echo "  \$ $*"; [[ $SIMU == 1 ]] || "$@"; }
root() { if [[ $EUID == 0 ]]; then run "$@"; else run sudo "$@"; fi; }
ecrire_root() {  # ecrire_root FICHIER CONTENU
  echo "  \$ (écrit) $1"
  [[ $SIMU == 1 ]] && { sed 's/^/      /' <<<"$2"; return; }
  if [[ $EUID == 0 ]]; then printf '%s\n' "$2" > "$1"; else printf '%s\n' "$2" | sudo tee "$1" >/dev/null; fi
}
demander() {  # demander "question" défaut → réponse
  local rep
  if [[ $OUI == 1 || ! -t 0 ]]; then echo "$2"; return; fi
  read -rp "  ? $1 [$2] : " rep </dev/tty
  echo "${rep:-$2}"
}
oui_non() {  # oui_non "question" o|n → 0 si oui
  local r; r="$(demander "$1 (o/n)" "$2")"
  [[ "$r" =~ ^[oOyY] ]]
}
paquets_manquants() { local p m=(); for p in "$@"; do dpkg -s "$p" >/dev/null 2>&1 || m+=("$p"); done; echo "${m[*]-}"; }
apt_installer() {
  local m; m="$(paquets_manquants "$@")"
  if [[ -z "$m" ]]; then ok "déjà installés : $*"; return; fi
  # shellcheck disable=SC2086
  root env DEBIAN_FRONTEND=noninteractive apt-get install -y -q $m
}
dans_groupe() { id -nG "$UTIL" | tr ' ' '\n' | grep -qx "$1"; }
ajouter_groupe() {
  getent group "$1" >/dev/null || return 0
  if dans_groupe "$1"; then ok "$UTIL est dans le groupe $1"; else root usermod -aG "$1" "$UTIL"; GROUPE_AJOUTE=1; fi
}
# docker pour le compte courant, même si le groupe vient d'être ajouté dans cette session.
docker_ici() {
  if [[ $SIMU == 1 ]]; then echo "  \$ $*"; return; fi
  if docker info >/dev/null 2>&1; then "$@"; else sg docker -c "$(printf '%q ' "$@")"; fi
}
version_ge() { [[ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -1)" == "$2" ]]; }

echo "${GRAS}Installation de MammoMesh — $(hostname) — $(date '+%d/%m/%Y %H:%M')${FIN}"
[[ $SIMU == 1 ]] && echo "${JAUNE}Simulation : rien n'est modifié, les commandes sont seulement affichées.${FIN}" || echo "Journal : $JOURNAL"

# ---------- 1. Système ----------
etape "Système"
UTIL="$(id -un)"
if [[ $EUID == 0 ]]; then
  [[ $SIMU == 1 ]] || erreur "à lancer depuis le compte qui fera tourner MammoMesh, avec sudo (pas en root). Sur un serveur neuf :
         adduser mammomesh && usermod -aG sudo mammomesh && su - mammomesh"
  alerte "simulation lancée en root"
fi
[[ -r /etc/os-release ]] || erreur "système non reconnu (/etc/os-release absent)"
# shellcheck disable=SC1091
. /etc/os-release
FAMILLE=""
for i in ${ID:-} ${ID_LIKE:-}; do [[ $i == ubuntu || $i == debian ]] && { FAMILLE=$i; break; }; done
[[ -n "$FAMILLE" ]] || erreur "$PRETTY_NAME n'est pas pris en charge par ce script (Ubuntu ou Debian) : suivre docs/installation-serveur.md, section « Autres systèmes »"
CODENAME="${UBUNTU_CODENAME:-${VERSION_CODENAME:-}}"
[[ $FAMILLE == debian ]] && CODENAME="${VERSION_CODENAME:-}"
ARCH="$(dpkg --print-architecture)"
ok "$PRETTY_NAME ($ARCH), famille $FAMILLE ($CODENAME), compte $UTIL"
[[ $ARCH == amd64 || $ARCH == arm64 ]] || alerte "architecture $ARCH non testée"

CPU="$(grep -m1 'model name' /proc/cpuinfo 2>/dev/null | cut -d: -f2- | sed 's/^ *//')"
RAM_GO=$(( $(awk '/MemTotal/ {print $2}' /proc/meminfo) / 1024 / 1024 ))
LIBRE_GO=$(( $(df -Pk "$DEPOT" | awk 'NR==2 {print $4}') / 1024 / 1024 ))
ok "${CPU:-CPU inconnu}, $(nproc) cœurs, $RAM_GO Go de RAM, $LIBRE_GO Go libres sur $(df -P "$DEPOT" | awk 'NR==2 {print $6}')"
(( RAM_GO >= 8 )) || alerte "moins de 8 Go de RAM : la pile tourne, l'entraînement sera très limité"
(( LIBRE_GO >= 25 )) || erreur "moins de 25 Go libres : il en faut ~10 pour la pile, ~8 pour PyTorch et 10 de marge (SeaweedFS passe en lecture seule en dessous)"
(( LIBRE_GO >= 60 )) || alerte "moins de 60 Go libres : la banque d'images (23 Go pour CMMD, plusieurs centaines pour le reste) ira sur un autre disque (--banque-dir)"
if [[ $SIMU == 0 && $EUID != 0 ]]; then sudo -v || erreur "sudo est nécessaire"; fi
curl -fsS --max-time 15 -o /dev/null https://download.docker.com 2>/dev/null && ok "accès à Internet" \
  || alerte "download.docker.com injoignable : les téléchargements risquent d'échouer (proxy ?)"

# ---------- 2. Paquets de base ----------
etape "Paquets de base"
root env DEBIAN_FRONTEND=noninteractive apt-get update -q
apt_installer git curl ca-certificates gnupg openssl pciutils unzip p7zip-full python3 python3-venv python3-pip

# ---------- 3. Matériel ----------
etape "Matériel"
GPU_NVIDIA=0 GPU_AMD=0 GPU_INTEL=0
for d in /sys/bus/pci/devices/*; do
  [[ -r $d/class && "$(cat "$d/class")" == 0x03* ]] || continue   # contrôleurs d'affichage et 3D
  vendeur="$(cat "$d/vendor")"
  nom="$(lspci -s "${d##*/}" 2>/dev/null | cut -d' ' -f2- || true)"
  case "$vendeur" in
    0x10de) GPU_NVIDIA=1; ok "NVIDIA : ${nom:-${d##*/}}" ;;
    0x1002) GPU_AMD=1; ok "AMD : ${nom:-${d##*/}}" ;;
    0x8086) GPU_INTEL=1; ok "Intel : ${nom:-${d##*/}}" ;;
    *) info "autre : ${nom:-$vendeur}" ;;
  esac
done
if [[ -z "$ACCEL" ]]; then
  if [[ $GPU_NVIDIA == 1 ]]; then ACCEL=nvidia
  elif [[ $GPU_AMD == 1 ]]; then ACCEL=amd
  elif [[ $GPU_INTEL == 1 ]]; then ACCEL=intel
  else ACCEL=cpu; fi
fi
ok "accélérateur retenu pour l'entraînement : $ACCEL"
[[ $ACCEL == cpu ]] && alerte "aucun GPU utilisable : la pile tourne, l'entraînement du CNN sera très lent (CPU)"

# ---------- 4. Docker ----------
etape "Docker Engine et compose"
if command -v docker >/dev/null && docker compose version >/dev/null 2>&1; then
  ok "$(docker --version), $(docker compose version --short 2>/dev/null | sed 's/^/compose /')"
else
  root install -m 0755 -d /etc/apt/keyrings
  root curl -fsSL "https://download.docker.com/linux/$FAMILLE/gpg" -o /etc/apt/keyrings/docker.asc
  root chmod a+r /etc/apt/keyrings/docker.asc
  ecrire_root /etc/apt/sources.list.d/docker.list \
    "deb [arch=$ARCH signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/$FAMILLE $CODENAME stable"
  root apt-get update -q
  root env DEBIAN_FRONTEND=noninteractive apt-get install -y -q docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
fi
for s in docker.service containerd.service; do
  if [[ "$(systemctl is-enabled "$s" 2>/dev/null)" == enabled ]]; then ok "$s démarre avec la machine"; else root systemctl enable --now "$s"; fi
done
ajouter_groupe docker

# ---------- 5. GPU ----------
etape "Pilote et accès des conteneurs au GPU ($ACCEL)"
case "$ACCEL" in
  nvidia)
    if nvidia-smi -L >/dev/null 2>&1; then
      ok "pilote NVIDIA $(nvidia-smi --query-gpu=driver_version --format=csv,noheader | head -1) : $(nvidia-smi -L | head -1)"
    elif [[ $SANS_PILOTE == 1 ]]; then
      alerte "pilote NVIDIA absent (--sans-pilote) : l'installer, redémarrer, puis relancer ce script"
    elif [[ $FAMILLE == ubuntu ]]; then
      apt_installer ubuntu-drivers-common
      # Serveur sans bureau : pilote « gpgpu » (calcul) ; poste avec bureau : pilote recommandé.
      if [[ "$(systemctl get-default)" == graphical.target ]]; then root ubuntu-drivers install; else root ubuntu-drivers install --gpgpu; fi
      REDEMARRER=1
    else
      if apt-cache policy nvidia-driver 2>/dev/null | grep -q 'Candidate: [0-9]'; then
        apt_installer "linux-headers-$(uname -r)" nvidia-driver firmware-misc-nonfree
        REDEMARRER=1
      else
        alerte "Debian : activer les composants « contrib non-free non-free-firmware » dans /etc/apt/sources.list, puis relancer ce script"
      fi
    fi
    if command -v nvidia-ctk >/dev/null; then
      ok "nvidia-container-toolkit $(nvidia-ctk --version 2>/dev/null | head -1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' || true)"
    else
      root sh -c "curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | gpg --dearmor --yes -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg"
      root sh -c "curl -fsSL https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list \
        | sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' \
        > /etc/apt/sources.list.d/nvidia-container-toolkit.list"
      root apt-get update -q
      apt_installer nvidia-container-toolkit
    fi
    # Les services GPU passent par « runtime: nvidia » (voir compose.yaml et scripts/detecter-materiel.sh).
    if [[ $SIMU == 0 ]] && sudo docker info 2>/dev/null | grep -qiE 'runtimes:.*nvidia'; then
      ok "runtime nvidia déclaré à Docker"
    else
      root nvidia-ctk runtime configure --runtime=docker
      root systemctl restart docker
    fi
    ;;
  amd)
    ajouter_groupe render; ajouter_groupe video
    if [[ -e /dev/kfd ]]; then ok "ROCm : /dev/kfd présent"
    else alerte "AMD : ROCm absent (/dev/kfd). Installer amdgpu-install (https://rocm.docs.amd.com, « Quick start installation »), redémarrer, puis relancer ce script"; fi
    ;;
  intel)
    ajouter_groupe render; ajouter_groupe video
    info "GPU Intel : inférence par OpenVINO ; l'entraînement se fera sur CPU (ou PyTorch XPU sur une carte Arc)"
    ;;
  cpu) ok "rien à installer" ;;
esac

# ---------- 6. Réglages du système ----------
etape "Réglages du système"
if [[ "$(sysctl -n vm.overcommit_memory 2>/dev/null)" == 1 ]]; then
  ok "vm.overcommit_memory=1 (Redis)"
else
  ecrire_root /etc/sysctl.d/99-mammomesh-redis.conf "vm.overcommit_memory = 1"
  root sysctl -q -w vm.overcommit_memory=1
fi

# ---------- 7. Configuration de la pile ----------
etape "Configuration (.env, secrets, nom du site)"
if [[ -f .env ]]; then ok ".env existe : gardé, seules les valeurs demandées changent"; else run cp .env.example .env; fi
valeur_env() { sed -n "s/^$1=//p" .env 2>/dev/null | tail -1; }
regler_env() {  # regler_env CLÉ VALEUR
  if [[ "$(valeur_env "$1")" == "$2" ]]; then ok "$1=$2"; return; fi
  echo "  \$ .env : $1=$2"
  [[ $SIMU == 1 ]] && return
  if grep -q "^$1=" .env; then
    local v; v="$(printf '%s' "$2" | sed 's/[\/&|]/\\&/g')"
    sed -i "s|^$1=.*|$1=$v|" .env
  else
    printf '%s=%s\n' "$1" "$2" >> .env
  fi
}
[[ -f .env ]] || [[ $SIMU == 1 ]] || erreur ".env manquant"
ancien_site="$(valeur_env SITE_ADDRESS)"
[[ -z "$SITE" ]] && SITE="$(demander "Nom du site sur le réseau local" "$([[ -n "$ancien_site" && "$ancien_site" != mammomesh.lan ]] && echo "$ancien_site" || echo "$(hostname -s).lan")")"
IP_DETECTEE="$(ip -4 route get 1.1.1.1 2>/dev/null | grep -oP 'src \K[0-9.]+' || true)"
[[ -n "$IP_DETECTEE" ]] || IP_DETECTEE="$(hostname -I 2>/dev/null | awk '{print $1}')"
[[ -z "$IP" ]] && IP="$(demander "Adresse d'écoute (0.0.0.0 = toutes)" "${IP_DETECTEE:-0.0.0.0}")"
regler_env SITE_ADDRESS "$SITE"
regler_env BIND_ADDRESS "$IP"
if [[ -n "$PROFILS_EN_PLUS" ]]; then
  fusion="$(printf '%s\n' "$(valeur_env COMPOSE_PROFILES | tr ',' '\n')" "$(tr ',' '\n' <<<"$PROFILS_EN_PLUS")" | sed '/^$/d' | awk '!vu[$0]++' | paste -sd, -)"
  regler_env COMPOSE_PROFILES "$fusion"
fi
if [[ ",$(valeur_env COMPOSE_PROFILES)," == *,backup,* ]]; then
  bdir="$(valeur_env BACKUP_DIR)"; bdir="${bdir:-./backup}"
  [[ -d "$bdir" ]] && ok "dossier de sauvegarde $bdir" || run mkdir -p "$bdir"
  [[ "$bdir" == ./* ]] && alerte "les sauvegardes restent sur ce serveur ($bdir) : prévoir un disque ou un NAS (BACKUP_DIR, BACKUP_COPIE)"
fi
run ./scripts/init-secrets.sh
run mkdir -p modeles/.demandes
run chmod 0777 modeles/.demandes
# Le serveur se joint lui-même par son nom (essais, certificat).
if [[ "$SITE" != localhost ]] && ! getent hosts "$SITE" >/dev/null; then
  root sh -c "printf '127.0.1.1 %s\n' '$SITE' >> /etc/hosts"
fi

# ---------- 8. Moteur (entraînement du CNN) ----------
if [[ $SANS_MOTEUR == 0 ]]; then
  etape "Moteur Python (engine/.venv) et PyTorch pour $ACCEL"
  version_ge "$(python3 -c 'import sys; print("%d.%d" % sys.version_info[:2])')" 3.10 || erreur "Python 3.10 ou plus récent requis pour le moteur"
  extra="cnn"
  if [[ -n "${TORCH_INDEX:-}" ]]; then
    index="$TORCH_INDEX"
  else
    case "$ACCEL" in
      nvidia)
        cuda="$(nvidia-smi 2>/dev/null | grep -oP 'CUDA Version: \K[0-9.]+' || true)"
        if [[ -z "$cuda" ]]; then index=""; info "pilote pas encore chargé : PyTorch CUDA par défaut (pilote récent attendu)"
        elif version_ge "$cuda" 13.0; then index=""
        elif version_ge "$cuda" 12.8; then index="https://download.pytorch.org/whl/cu128"
        elif version_ge "$cuda" 12.6; then index="https://download.pytorch.org/whl/cu126"
        else index="https://download.pytorch.org/whl/cpu"; alerte "pilote NVIDIA trop ancien (CUDA $cuda) : PyTorch CPU ; mettre le pilote à jour puis relancer"; fi
        ;;
      amd) index="https://download.pytorch.org/whl/rocm6.4" ;;
      intel) index="https://download.pytorch.org/whl/cpu"; extra="cnn,openvino" ;;
      *) index="https://download.pytorch.org/whl/cpu" ;;
    esac
  fi
  [[ -x engine/.venv/bin/python ]] && ok "engine/.venv existe" || run python3 -m venv engine/.venv
  if [[ $SIMU == 0 ]] && engine/.venv/bin/python -c 'import torch' 2>/dev/null; then
    ok "PyTorch $(engine/.venv/bin/python -c 'import torch; print(torch.__version__)') déjà installé"
  else
    run engine/.venv/bin/pip install -q --upgrade pip
    if [[ -n "$index" ]]; then run engine/.venv/bin/pip install -q torch torchvision --index-url "$index"
    else run engine/.venv/bin/pip install -q torch torchvision; fi
  fi
  run engine/.venv/bin/pip install -q -e "engine[$extra]"
  if [[ $SIMU == 0 ]]; then
    (cd engine && .venv/bin/python -m mammomesh_engine.materiel) | sed 's/^/         /' || alerte "plan matériel illisible"
  fi
fi

# ---------- 9. Pile Docker ----------
etape "Pile Docker (construction et démarrage)"
info "première construction : plusieurs minutes"
docker_ici ./scripts/demarrer-pile.sh --build
if [[ $SIMU == 0 ]]; then
  for _ in $(seq 1 60); do
    curl -fsk --max-time 5 --resolve "$SITE:443:127.0.0.1" "https://$SITE/api/health" >/dev/null 2>&1 && break
    sleep 5
  done
  if curl -fsk --max-time 5 --resolve "$SITE:443:127.0.0.1" "https://$SITE/api/health" >/dev/null 2>&1; then
    ok "https://$SITE répond"
    docker_ici docker compose cp caddy:/data/caddy/pki/authorities/local/root.crt ./mammomesh-root.crt \
      && ok "certificat racine du site : $DEPOT/mammomesh-root.crt (à importer sur les postes)"
  else
    alerte "https://$SITE ne répond pas encore : voir « docker compose ps » et « docker compose logs api caddy »"
  fi
fi

# ---------- 10. Démarrage automatique ----------
if [[ $SANS_SERVICES == 0 ]]; then
  etape "Démarrage automatique (pile au démarrage, reprise des entraînements)"
  if [[ "$(loginctl show-user "$UTIL" -p Linger --value 2>/dev/null)" == yes ]]; then ok "linger actif"; else root loginctl enable-linger "$UTIL"; fi
  if [[ $GROUPE_AJOUTE == 1 ]]; then
    # Le gestionnaire de services de ce compte ne connaît pas encore ses nouveaux groupes : on active
    # sans démarrer, tout partira au redémarrage.
    run env BANQUE_DIR="${BANQUE_DIR_OPT:-$DEPOT/banque}" ./scripts/installer-services.sh --sans-demarrer
    REDEMARRER=1
  else
    run env BANQUE_DIR="${BANQUE_DIR_OPT:-$DEPOT/banque}" ./scripts/installer-services.sh
  fi
fi

# ---------- 11. Banque d'entraînement ----------
etape "Banque d'entraînement"
if [[ -z "$BANQUE" ]] && oui_non "Télécharger maintenant CMMD (23 Go, ~1 h) ?" n; then BANQUE=CMMD; fi
if [[ -n "$BANQUE" ]]; then
  bd="${BANQUE_DIR_OPT:-$DEPOT/banque}"
  run env BANQUE_DIR="$bd" ./examples/telecharger.sh --banque "$BANQUE" || alerte "banque incomplète : relancer « BANQUE_DIR=$bd ./examples/telecharger.sh --banque $BANQUE » (reprise)"
else
  info "plus tard : BANQUE_DIR=/disque/banque ./examples/telecharger.sh --banque CMMD   (voir docs/jeux-entrainement.md)"
fi

# ---------- Résumé ----------
echo
echo "${GRAS}== Résumé${FIN}"
echo "  Site        : https://$SITE  (écoute sur $IP:443)"
ip_postes="$IP"; [[ "$ip_postes" == 0.0.0.0 ]] && ip_postes="${IP_DETECTEE:-<adresse du serveur>}"
echo "  Postes      : ajouter « $ip_postes $SITE » à leur fichier hosts (ou au DNS local), puis importer mammomesh-root.crt"
echo "  Connexion   : $(valeur_env ADMIN_USERNAME 2>/dev/null || echo admin) / mot de passe de secrets/admin_password.txt (à changer à la première connexion)"
echo "  Matériel    : $ACCEL ; profils compose : $(./scripts/detecter-materiel.sh 2>/dev/null || echo '?')"
[[ $SANS_MOTEUR == 0 ]] && echo "  Entraîner   : cd engine && .venv/bin/python -m mammomesh_engine.cnn --banque ${BANQUE_DIR_OPT:-../banque}"
[[ $SIMU == 1 ]] || echo "  Journal     : $JOURNAL"
if (( ${#ALERTES[@]} )); then
  echo; echo "${JAUNE}Points à reprendre :${FIN}"; printf '  - %s\n' "${ALERTES[@]}"
fi
if [[ $REDEMARRER == 1 ]]; then
  echo; echo "${GRAS}Redémarrer le serveur (sudo reboot)${FIN} : pilote GPU et/ou nouveaux groupes. La pile et le veilleur repartiront seuls ;"
  echo "relancer ensuite ce script pour vérifier (il ne refait que ce qui manque)."
fi
