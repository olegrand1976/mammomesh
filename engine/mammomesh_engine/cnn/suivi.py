"""État d'un entraînement, lu par l'interface de suivi (API : GET /api/entrainements).

`etat.json` est réécrit en entier, de façon atomique, à chaque changement de phase et au plus toutes les
10 s pendant une époque. L'API le lit avec `journal.csv` (une ligne par époque) et `rapport.json` (à la fin).
"""
from __future__ import annotations

import json
import os
import time
from datetime import datetime, timezone
from pathlib import Path

FICHIER = "etat.json"
INTERVALLE_S = 10.0


def maintenant() -> str:
    return _maintenant()


def _maintenant() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="seconds")


def lire_etat(dossier: str | Path) -> dict:
    try:
        return json.loads((Path(dossier) / FICHIER).read_text(encoding="utf-8"))
    except (OSError, ValueError):
        return {}


def ecrire_etat(dossier: str | Path, etat: dict) -> None:
    """Écriture atomique (aussi utilisée par le veilleur auto.py)."""
    d = Path(dossier)
    d.mkdir(parents=True, exist_ok=True)
    tmp = d / (FICHIER + ".tmp")
    tmp.write_text(json.dumps(etat, ensure_ascii=False, indent=1), encoding="utf-8")
    tmp.replace(d / FICHIER)


class Suivi:
    def __init__(self, dossier: str | Path, *, reprise: bool = False, **infos):
        """reprise=True : garde l'état existant (début, compteurs du veilleur) et compte une reprise de plus."""
        self.dossier = Path(dossier)
        ancien = lire_etat(self.dossier) if reprise else {}
        for cle in ("statut", "message", "fin", "pid", "etape"):
            ancien.pop(cle, None)
        self.etat: dict = {"version": 1, "phase": "preparation", "debut": _maintenant(), **ancien,
                           "statut": "en_cours", "pid": os.getpid(), **infos}
        if reprise:
            self.etat["reprises"] = int(self.etat.get("reprises", 0)) + 1
            self.etat["derniere_reprise"] = _maintenant()
        self._derniere = 0.0
        self.ecrire()

    def ecrire(self) -> None:
        self.etat["maj"] = _maintenant()
        try:
            ecrire_etat(self.dossier, self.etat)
        except OSError:
            pass   # le suivi ne doit jamais arrêter un entraînement
        self._derniere = time.monotonic()

    def phase(self, phase: str, **infos) -> None:
        if phase != self.etat.get("phase"):
            infos.setdefault("debut_phase", _maintenant())   # sert à l'estimation des temps de fin
        self.etat.update(phase=phase, **infos)
        self.ecrire()

    def progres(self, **infos) -> None:
        """Mise à jour fréquente (pas dans l'époque) : écrite au plus toutes les INTERVALLE_S secondes."""
        self.etat.update(infos)
        if time.monotonic() - self._derniere >= INTERVALLE_S:
            self.ecrire()

    def fin(self, statut: str, message: str = "") -> None:
        self.etat.update(statut=statut, message=message, fin=_maintenant())
        self.ecrire()


class SansSuivi(Suivi):
    """Suivi inactif (appel direct d'entrainer() sans dossier de suivi)."""

    def __init__(self):
        self.etat, self._derniere = {}, 0.0

    def ecrire(self) -> None:
        pass
