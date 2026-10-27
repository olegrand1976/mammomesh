#!/usr/bin/env bash
# Profils docker compose selon les accélérateurs de l'hôte ET leur prise en charge par Docker.
# Sortie : liste séparée par des virgules, fusionnée avec COMPOSE_PROFILES déjà défini (obs, backup…).
#   COMPOSE_PROFILES="$(./scripts/detecter-materiel.sh)" docker compose up -d
# MAMMOMESH_ACCEL=nvidia|amd|intel|cpu force le choix. Les avertissements vont sur stderr.
set -euo pipefail

profils=()
avertir() { echo "detecter-materiel : $*" >&2; }

if [[ -n "${MAMMOMESH_ACCEL:-}" ]]; then
  IFS=',' read -r -a profils <<<"$MAMMOMESH_ACCEL"
else
  docker_info="$(docker info 2>/dev/null || true)"

  # NVIDIA : pilote présent et runtime « nvidia » déclaré à Docker. Les services GPU passent par
  # runtime: nvidia, pas par --gpus : avec Docker 28.1, runc 1.2.5 et cgroup v2, --gpus prend le mode
  # legacy de nvidia-container-cli et échoue (« failed to add device rules »).
  if command -v nvidia-smi >/dev/null && nvidia-smi -L >/dev/null 2>&1; then
    if grep -qiE 'runtimes:.*nvidia' <<<"$docker_info"; then
      profils+=(nvidia)
    elif command -v nvidia-ctk >/dev/null; then
      avertir "GPU NVIDIA vu, nvidia-container-toolkit installé mais pas déclaré à Docker :" \
        "sudo nvidia-ctk runtime configure --runtime=docker && sudo systemctl restart docker"
    else
      avertir "GPU NVIDIA vu mais nvidia-container-toolkit absent : Docker ne pourra pas s'en servir" \
        "(installer nvidia-container-toolkit, puis sudo nvidia-ctk runtime configure --runtime=docker" \
        "&& sudo systemctl restart docker)"
    fi
  fi

  # AMD (ROCm) : /dev/kfd.
  [[ -e /dev/kfd ]] && profils+=(amd)

  # Intel : nœud de rendu d'un GPU Intel (vendeur 0x8086), utilisé par OpenVINO ou PyTorch XPU.
  for v in /sys/class/drm/renderD*/device/vendor; do
    [[ -r "$v" && "$(cat "$v")" == "0x8086" ]] && { profils+=(intel); break; }
  done

  # Aucun accélérateur dédié : le CPU entraîne.
  if [[ ! " ${profils[*]-} " =~ \ (nvidia|amd)\  ]]; then
    profils=(cpu ${profils[@]+"${profils[@]}"})
  fi
fi

# Fusion avec les profils déjà demandés (variable d'environnement ou .env), sans doublon.
existants="${COMPOSE_PROFILES:-}"
if [[ -z "$existants" && -f .env ]]; then
  existants="$(sed -n 's/^COMPOSE_PROFILES=//p' .env | tail -1 | tr -d '"'"'")"
fi
IFS=',' read -r -a deja <<<"$existants"
printf '%s\n' ${deja[@]+"${deja[@]}"} ${profils[@]+"${profils[@]}"} | sed '/^$/d' | awk '!vu[$0]++' | paste -sd, -
