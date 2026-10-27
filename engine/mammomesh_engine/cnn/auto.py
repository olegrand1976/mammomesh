"""Veilleur : reprend tout seul un entraînement inachevé dès que le GPU est libre, et exécute les demandes
de l'interface (relancer, supprimer) déposées par l'API dans modeles/.demandes/.

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
import json
import os
import re
import shlex
import shutil
import signal
import subprocess
import sys
import time
from dataclasses import dataclass, field
from datetime import datetime
from pathlib import Path

from .suivi import ecrire_etat, lire_etat, maintenant

DEPOT = Path(__file__).resolve().parents[3]
MOTEUR = DEPOT / "engine"
TENTATIVES_MAX = 3
EN_ATTENTE = "en_attente_gpu"
EN_PAUSE = "en_pause"
PAUSE = "pause"   # même nom que entrainement.PAUSE (sans importer PyTorch ici)
DEMANDES = ".demandes"
NOM_VALIDE = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$")
STABILITE_S = 30          # le GPU doit rester libre ce temps-là avant un départ
ARRET_S = 60              # délai laissé à un entraînement pour s'arrêter proprement (Ctrl+C) avant de le tuer


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
        if e.get("statut") in ("termine", "interrompu", EN_PAUSE) or vivant(e.get("pid")):
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


@dataclass
class Veilleur:
    modeles: Path
    memoire_min: int = 4500
    gpu: int = 0
    intervalle: int = 60
    journal: callable = print
    courant: tuple[subprocess.Popen, Path] | None = None
    libre_depuis: float | None = None
    attente_signalee: dict[str, str] = field(default_factory=dict)
    _prochaine_verif: float = 0.0

    # ---------- reprise ----------

    def lancer(self, d: Path) -> None:
        e = lire_etat(d)
        # Compteur de reprises sans progrès : remis à zéro dès qu'une époque de plus a été faite.
        epoque = int(e.get("epoque", 0))
        tentatives = 0 if epoque > int(e.get("epoque_derniere_tentative", -1)) else int(e.get("tentatives_auto", 0))
        _maj(d, tentatives_auto=tentatives + 1, epoque_derniere_tentative=epoque, message="")
        self.journal(f"{'départ' if e.get('file_attente') else 'reprise'} de {d.name} "
                     f"(tentative {tentatives + 1}/{TENTATIVES_MAX}, après l'époque {epoque})")
        log = open(d.parent / f"{d.name}.log", "a", encoding="utf-8")
        log.write(f"\n== {maintenant()} : lancé par le veilleur\n")
        log.flush()
        proc = subprocess.Popen(_commande(d), cwd=MOTEUR, stdout=log, stderr=subprocess.STDOUT, start_new_session=True)
        log.close()
        self.courant = (proc, d)
        self.attente_signalee.pop(d.name, None)

    def surveiller(self) -> None:
        if not self.courant:
            return
        proc, d = self.courant
        code = proc.poll()
        if code is None:
            return
        self.courant = None
        e = lire_etat(d)
        if d.exists() and code != 0 and int(e.get("tentatives_auto", 0)) >= TENTATIVES_MAX and not (d / "rapport.json").exists() \
                and e.get("statut") != "interrompu":
            _maj(d, statut="echec", message=f"Abandon après {TENTATIVES_MAX} reprises automatiques sans progrès "
                                            f"(dernier code de sortie {code}, voir {d.name}.log)")
        self.journal(f"{d.name} : fin du processus, code {code}")

    def verifier(self) -> None:
        """Choisit l'entraînement à reprendre et le lance quand le GPU est libre depuis STABILITE_S."""
        if self.courant:
            return
        candidats = a_reprendre(self.modeles)
        if not candidats:
            self.libre_depuis = None
            return
        d = candidats[0]
        libre, raison = etat_gpu(self.memoire_min, self.gpu)
        if libre:
            self.libre_depuis = self.libre_depuis or time.monotonic()
            if time.monotonic() - self.libre_depuis >= STABILITE_S:
                self.libre_depuis = None
                self.lancer(d)
                return
            raison = "GPU libre, départ dans quelques secondes"
        else:
            self.libre_depuis = None
        _maj(d, statut=EN_ATTENTE, message=f"En attente du GPU : {raison}")
        if self.attente_signalee.get(d.name) != raison:
            self.journal(f"{d.name} en attente : {raison}")
            self.attente_signalee[d.name] = raison

    # ---------- demandes de l'interface ----------

    def demandes(self) -> None:
        rep = self.modeles / DEMANDES
        for f in sorted(rep.glob("*.json")) if rep.is_dir() else []:
            try:
                dem = json.loads(f.read_text(encoding="utf-8"))
                nom, action, par = str(dem.get("nom", "")), dem.get("action"), str(dem.get("par", "?"))
            except (OSError, ValueError):
                nom, action, par = "", None, "?"
            d = self.modeles / nom
            if action == "nouveau":
                self.nouveau(dem.get("reglages") or {}, par)
            elif not NOM_VALIDE.match(nom) or not d.is_dir():
                self.journal(f"demande ignorée ({f.name}) : entraînement introuvable")
            elif action == "supprimer":
                self.supprimer(d, par)
            elif action == "relancer":
                self.relancer(d, par)
            elif action == "pause":
                self.pause(d, par, dem.get("mode") or "fin_epoque")
            else:
                self.journal(f"demande ignorée ({f.name}) : action inconnue")
            f.unlink(missing_ok=True)

    def _arreter(self, d: Path) -> None:
        """Arrête proprement (Ctrl+C) l'entraînement de ce dossier s'il tourne, de force au bout d'ARRET_S."""
        if self.courant and self.courant[1] == d:
            proc = self.courant[0]
            try:   # tout le groupe : systemd-inhibit, l'entraînement et ses workers
                os.killpg(proc.pid, signal.SIGINT)
            except ProcessLookupError:
                pass
            try:
                proc.wait(ARRET_S)
            except subprocess.TimeoutExpired:
                proc.kill()
                proc.wait()
            self.courant = None
            return
        pid = lire_etat(d).get("pid")
        if processus_vivant(pid):
            os.kill(int(pid), signal.SIGINT)
            fin = time.monotonic() + ARRET_S
            while processus_vivant(pid) and time.monotonic() < fin:
                time.sleep(1)
            if processus_vivant(pid):
                os.kill(int(pid), signal.SIGKILL)

    def supprimer(self, d: Path, par: str) -> None:
        self._arreter(d)
        shutil.rmtree(d, ignore_errors=True)
        for log in (d.parent / f"{d.name}.log", d.parent / f"entrainement-{d.name.removeprefix('cnn-')}.log"):
            log.unlink(missing_ok=True)
        self.attente_signalee.pop(d.name, None)
        self.journal(f"{d.name} supprimé (demande de {par})")

    def pause(self, d: Path, par: str, mode: str) -> None:
        """fin_epoque : l'entraînement s'arrête après l'époque en cours (rien de perdu) ;
        maintenant : arrêt immédiat, l'époque en cours sera refaite à la reprise."""
        e = lire_etat(d)
        if e.get("statut") in ("termine", "interrompu", EN_PAUSE) or (d / "rapport.json").exists():
            self.journal(f"pause de {d.name} ignorée : statut {e.get('statut')}")
            return
        tourne = (self.courant and self.courant[1] == d) or processus_vivant(e.get("pid"))
        if tourne and mode == "fin_epoque":
            (d / PAUSE).write_text(par, encoding="utf-8")
            self.journal(f"{d.name} : pause en fin d'époque demandée par {par}")
            return
        if tourne:
            self._arreter(d)
            epoque = int(lire_etat(d).get("epoque", 0))
            message = (f"Mis en pause immédiatement par {par} pendant l'époque {epoque} : "
                       f"« Reprendre » refera cette époque depuis le début")
        else:
            message = f"Mis en pause par {par} avant son départ"
        (d / PAUSE).unlink(missing_ok=True)
        _maj(d, statut=EN_PAUSE, message=message, fin=maintenant())
        self.attente_signalee.pop(d.name, None)
        self.journal(f"{d.name} : {message}")

    def relancer(self, d: Path, par: str) -> None:
        e = lire_etat(d)
        (d / PAUSE).unlink(missing_ok=True)
        if (self.courant and self.courant[1] == d) or processus_vivant(e.get("pid")):
            self.journal(f"relance de {d.name} ignorée : il tourne déjà")
            return
        if (d / "parametres.json").exists() and not (d / "rapport.json").exists():
            # Reprise à la dernière époque finie, compteurs remis à zéro (même après un arrêt à la main).
            _maj(d, statut=EN_ATTENTE, message=f"Reprise demandée par {par}", tentatives_auto=0,
                 epoque_derniere_tentative=-1, fin=None)
            self.journal(f"{d.name} : reprise demandée par {par}")
            return
        nouveau = nouvel_entrainement(self.modeles, d, par)
        self.journal(f"{d.name} : relancé en {nouveau.name} (demande de {par})")

    def nouveau(self, reglages: dict, par: str) -> None:
        d = nouvel_entrainement(self.modeles, None, par, reglages)
        self.journal(f"nouvel entraînement {d.name} (demande de {par})")

    # ---------- boucle ----------

    def tour(self) -> None:
        self.demandes()
        self.surveiller()
        if time.monotonic() >= self._prochaine_verif or self.libre_depuis is not None:
            self._prochaine_verif = time.monotonic() + self.intervalle
            self.verifier()


# Réglages qu'une demande « nouveau » de l'interface peut fixer (le reste vient du serveur : banque, cache).
REGLAGES_INTERFACE = {"epoques": int, "patience": int, "lr": float, "graine": int, "sensibilite_cible": float,
                      "limite": int, "sans_preentrainement": bool, "resolution": str}


def nouvel_entrainement(modeles: Path, source: Path | None, par: str, reglages: dict | None = None) -> Path:
    """Nouveau dossier, mis en file d'attente : le veilleur le lancera quand le GPU sera libre.
    Réglages : ceux de `source` (parametres.json, sinon sa commande), sinon ceux par défaut, puis `reglages`."""
    from .__main__ import CLINIQUE_REL, _ecrire_reglages, _taille, analyseur
    a = analyseur().parse_args([])
    for k, v in (reglages or {}).items():
        if k not in REGLAGES_INTERFACE or v is None:
            continue
        if k == "resolution":
            a.resolution = _taille(str(v))
        elif k == "limite":
            a.limite = int(v) or None
        else:
            setattr(a, k, REGLAGES_INTERFACE[k](v))
    if source is None:
        pass
    elif (source / "parametres.json").exists():
        r = json.loads((source / "parametres.json").read_text(encoding="utf-8"))
        for k, v in r.items():
            if hasattr(a, k) and v is not None:
                setattr(a, k, tuple(v) if k == "resolution" else Path(v) if k in ("banque", "cache") else v)
    else:
        try:
            a = analyseur().parse_args(shlex.split(lire_etat(source).get("commande", "") or ""))
        except SystemExit:
            pass   # commande illisible : réglages par défaut
    nom = f"cnn-cmmd-{datetime.now():%Y%m%d-%H%M}"
    d, k = modeles / nom, 2
    while d.exists():
        d, k = modeles / f"{nom}-{k}", k + 1
    _ecrire_reglages(a, d)
    quoi = f"Relance de {source.name}" if source else "Nouvel entraînement"
    etat = {"version": 1, "statut": EN_ATTENTE, "phase": "preparation", "debut": maintenant(), "maj": maintenant(),
            "jeu": "CMMD", "file_attente": True, "demande_par": par, "epoques": a.epoques,
            "message": f"{quoi} demandé par {par} : en attente du GPU"}
    if source:
        etat["source"] = source.name
    if not (Path(a.banque) / CLINIQUE_REL).exists():
        etat.update(statut="echec", tentatives_auto=TENTATIVES_MAX, fin=maintenant(),
                    message=f"Banque introuvable : {Path(a.banque) / CLINIQUE_REL} (lancer ./examples/telecharger.sh --banque CMMD,"
                            " ou régler BANQUE_DIR du service mammomesh-entrainement)")
    ecrire_etat(d, etat)
    return d


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(prog="mammomesh_engine.cnn.auto", description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--modeles", type=Path, default=DEPOT / "modeles")
    ap.add_argument("--memoire-min", type=int, default=4500, help="Mio libres demandés sur le GPU (défaut 4500)")
    ap.add_argument("--gpu", type=int, default=0, help="indice du GPU surveillé")
    ap.add_argument("--intervalle", type=int, default=60, help="secondes entre deux vérifications du GPU")
    ap.add_argument("--une-fois", action="store_true", help="un seul passage (demandes et vérification)")
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

    (a.modeles / DEMANDES).mkdir(parents=True, exist_ok=True)
    try:
        os.chmod(a.modeles / DEMANDES, 0o777)   # l'API (autre compte, dans Docker) y dépose les demandes
    except OSError as e:
        journal(f"droits de {a.modeles / DEMANDES} non modifiables ({e}) : l'interface ne pourra pas y écrire")
    verrou = open(a.modeles / ".auto.lock", "w")
    try:
        fcntl.flock(verrou, fcntl.LOCK_EX | fcntl.LOCK_NB)
    except BlockingIOError:
        journal("un autre veilleur tourne déjà : rien à faire")
        return 0

    v = Veilleur(a.modeles, a.memoire_min, a.gpu, a.intervalle, journal)
    journal(f"veilleur démarré : {a.modeles}, GPU {a.gpu}, {a.memoire_min} Mio demandés")
    arreter = []
    signal.signal(signal.SIGTERM, lambda *_: arreter.append(True))
    while not arreter:
        v.tour()
        if a.une_fois:
            break
        time.sleep(5)
    if v.courant:   # arrêt du service : l'entraînement s'arrête aussi et reprendra plus tard
        try:
            os.killpg(v.courant[0].pid, signal.SIGTERM)
        except ProcessLookupError:
            pass
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
