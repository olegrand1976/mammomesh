"""Veilleur : reprend tout seul un entraînement inachevé dès que le GPU est libre.

    python -m mammomesh_engine.cnn.auto                      # en boucle (service mammomesh-entrainement)
    python -m mammomesh_engine.cnn.auto --une-fois           # un seul passage
    python -m mammomesh_engine.cnn.auto --abandonner NOM     # ne plus reprendre cet entraînement

Un entraînement est à reprendre s'il a ses réglages (parametres.json), pas de rapport.json, que son processus
n'existe plus (machine éteinte, veille, processus tué, plantage) et qu'il n'a pas été arrêté à la main
(Ctrl+C : statut « interrompu »). Il repart de sa dernière époque finie (reprise.pt), avec les mêmes réglages.

Le GPU est libre quand aucun calcul n'y tourne (nvidia-smi : aucune application de calcul) et qu'il garde au moins
--memoire-min Mio libres, deux fois de suite à 30 s d'intervalle. En attendant, l'interface affiche
« En attente du GPU » avec la raison. Après 3 reprises sans progrès, le veilleur abandonne (statut « echec »).
"""
from __future__ import annotations

import argparse
import fcntl
import os
import shutil
import subprocess
import sys
import time
from pathlib import Path

from .suivi import ecrire_etat, lire_etat, maintenant

DEPOT = Path(__file__).resolve().parents[3]
MOTEUR = DEPOT / "engine"
TENTATIVES_MAX = 3
EN_ATTENTE = "en_attente_gpu"


def _nvidia(*requete: str) -> list[list[str]] | None:
    if not shutil.which("nvidia-smi"):
        return None
    try:
        sortie = subprocess.run(["nvidia-smi", *requete, "--format=csv,noheader,nounits"],
                                capture_output=True, text=True, timeout=20, check=True).stdout
    except (subprocess.SubprocessError, OSError):
        return None
    return [[c.strip() for c in l.split(",")] for l in sortie.splitlines() if l.strip()]


def etat_gpu(memoire_min: int, indice: int = 0, *, lire=_nvidia) -> tuple[bool, str]:
    """(libre, raison). `lire` est remplaçable pour les tests."""
    mem = lire("--query-gpu=index,memory.total,memory.used")
    if mem is None:
        return False, "nvidia-smi ne répond pas (pilote absent, ou GPU en sommeil)"
    ligne = next((l for l in mem if l[0] == str(indice)), None)
    if ligne is None:
        return False, f"GPU {indice} introuvable"
    total, utilise = int(float(ligne[1])), int(float(ligne[2]))
    applis = [l for l in (lire("--query-compute-apps=gpu_uuid,pid,process_name,used_memory") or []) if len(l) >= 4]
    if applis:
        noms = ", ".join(f"{Path(a[2]).name} (pid {a[1]}, {a[3]} Mio)" for a in applis[:3])
        return False, f"GPU occupé par {noms}"
    if total - utilise < memoire_min:
        return False, f"{total - utilise} Mio libres sur le GPU, {memoire_min} demandés"
    return True, f"{total - utilise} Mio libres"


def processus_vivant(pid: int | None) -> bool:
    """Le processus existe et c'est bien un entraînement MammoMesh (pas un pid réutilisé)."""
    if not pid:
        return False
    try:
        cmd = Path(f"/proc/{int(pid)}/cmdline").read_bytes().replace(b"\0", b" ")
    except (OSError, ValueError):
        return False
    return b"mammomesh_engine" in cmd


def a_reprendre(modeles: Path, *, vivant=processus_vivant) -> list[Path]:
    """Entraînements inachevés à reprendre, le plus récent en tête."""
    out = []
    for d in sorted((p for p in modeles.iterdir() if p.is_dir()), reverse=True) if modeles.is_dir() else []:
        if not (d / "parametres.json").exists() or (d / "rapport.json").exists():
            continue
        e = lire_etat(d)
        if e.get("statut") in ("termine", "interrompu") or vivant(e.get("pid")):
            continue
        if e.get("statut") == "echec" and int(e.get("tentatives_auto", 0)) >= TENTATIVES_MAX:
            continue
        out.append(d)
    return out


def _maj(d: Path, **champs) -> dict:
    e = lire_etat(d)
    e.update(champs, maj=maintenant())
    ecrire_etat(d, e)
    return e


def abandonner(d: Path) -> None:
    _maj(d, statut="interrompu", message="Reprise automatique abandonnée à la main", fin=maintenant())


def _commande(d: Path) -> list[str]:
    cmd = [sys.executable, "-m", "mammomesh_engine.cnn", "--reprendre", str(d)]
    # Pas de mise en veille pendant l'entraînement, si la session le permet.
    if shutil.which("systemd-inhibit") and subprocess.run(
            ["systemd-inhibit", "--what=sleep:idle", "--who=MammoMesh", "--why=essai", "true"],
            capture_output=True, timeout=20).returncode == 0:
        cmd = ["systemd-inhibit", "--what=sleep:idle", "--who=MammoMesh",
               f"--why=Entraînement du CNN ({d.name})", *cmd]
    return cmd


def reprendre(d: Path, journal=print) -> int:
    e = lire_etat(d)
    # Compteur de reprises sans progrès : remis à zéro dès qu'une époque de plus a été faite.
    epoque = int(e.get("epoque", 0))
    tentatives = 0 if epoque > int(e.get("epoque_derniere_tentative", -1)) else int(e.get("tentatives_auto", 0))
    _maj(d, tentatives_auto=tentatives + 1, epoque_derniere_tentative=epoque, message="")
    journal(f"reprise de {d.name} (tentative {tentatives + 1}/{TENTATIVES_MAX}, après l'époque {epoque})")
    with open(d.parent / f"{d.name}.log", "a", encoding="utf-8") as log:
        log.write(f"\n== {maintenant()} : reprise automatique\n")
        log.flush()
        code = subprocess.run(_commande(d), cwd=MOTEUR, stdout=log, stderr=subprocess.STDOUT).returncode
    e = lire_etat(d)
    if code != 0 and int(e.get("tentatives_auto", 0)) >= TENTATIVES_MAX and not (d / "rapport.json").exists():
        _maj(d, statut="echec", message=f"Abandon après {TENTATIVES_MAX} reprises automatiques sans progrès "
                                        f"(dernier code de sortie {code}, voir {d.name}.log)")
    journal(f"{d.name} : fin du processus, code {code}")
    return code


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(prog="mammomesh_engine.cnn.auto", description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--modeles", type=Path, default=DEPOT / "modeles")
    ap.add_argument("--memoire-min", type=int, default=4500, help="Mio libres demandés sur le GPU (défaut 4500)")
    ap.add_argument("--gpu", type=int, default=0, help="indice du GPU surveillé")
    ap.add_argument("--intervalle", type=int, default=60, help="secondes entre deux vérifications")
    ap.add_argument("--une-fois", action="store_true")
    ap.add_argument("--abandonner", metavar="NOM", help="ne plus reprendre cet entraînement")
    a = ap.parse_args(argv)
    journal = lambda m: print(f"[{time.strftime('%H:%M:%S')}] {m}", flush=True)

    if a.abandonner:
        d = a.modeles / Path(a.abandonner).name
        if not (d / "parametres.json").exists():
            print(f"{d} n'est pas un entraînement", file=sys.stderr)
            return 1
        abandonner(d)
        journal(f"{d.name} ne sera plus repris")
        return 0

    a.modeles.mkdir(parents=True, exist_ok=True)
    verrou = open(a.modeles / ".auto.lock", "w")
    try:
        fcntl.flock(verrou, fcntl.LOCK_EX | fcntl.LOCK_NB)
    except BlockingIOError:
        journal("un autre veilleur tourne déjà : rien à faire")
        return 0

    journal(f"veilleur démarré : {a.modeles}, GPU {a.gpu}, {a.memoire_min} Mio demandés")
    attente_signalee: dict[str, str] = {}
    while True:
        candidats = a_reprendre(a.modeles)
        if candidats:
            d = candidats[0]
            libre, raison = etat_gpu(a.memoire_min, a.gpu)
            if libre and not a.une_fois:
                time.sleep(30)   # le GPU doit rester libre : pas de départ sur un creux d'une seconde
                libre, raison = etat_gpu(a.memoire_min, a.gpu)
            if libre:
                attente_signalee.pop(d.name, None)
                reprendre(d, journal)
                continue
            _maj(d, statut=EN_ATTENTE, message=f"En attente du GPU : {raison}")
            if attente_signalee.get(d.name) != raison:
                journal(f"{d.name} en attente : {raison}")
                attente_signalee[d.name] = raison
        if a.une_fois:
            return 0
        time.sleep(a.intervalle)


if __name__ == "__main__":
    raise SystemExit(main())
