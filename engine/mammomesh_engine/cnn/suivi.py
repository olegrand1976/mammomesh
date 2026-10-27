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


def _maintenant() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="seconds")


class Suivi:
    def __init__(self, dossier: str | Path, **infos):
        self.dossier = Path(dossier)
        self.etat: dict = {"version": 1, "statut": "en_cours", "phase": "preparation", "debut": _maintenant(),
                           "pid": os.getpid(), **infos}
        self._derniere = 0.0
        self.ecrire()

    def ecrire(self) -> None:
        self.etat["maj"] = _maintenant()
        try:
            self.dossier.mkdir(parents=True, exist_ok=True)
            tmp = self.dossier / (FICHIER + ".tmp")
            tmp.write_text(json.dumps(self.etat, ensure_ascii=False, indent=1), encoding="utf-8")
            tmp.replace(self.dossier / FICHIER)
        except OSError:
            pass   # le suivi ne doit jamais arrêter un entraînement
        self._derniere = time.monotonic()

    def phase(self, phase: str, **infos) -> None:
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
