"""Mesure du plus grand lot qui tient réellement en mémoire.

L'estimation de plan.py suffit pour démarrer ; calibrer() fait de vrais pas
d'entraînement, cherche le plus grand lot par doublements puis dichotomie, garde
une marge de 10 % contre la fragmentation et met le résultat en cache par
(appareil, modèle, taille d'entrée, précision, checkpointing).
"""
from __future__ import annotations

import gc
import json
import os
from pathlib import Path
from typing import Callable

from .plan import Plan


def est_manque_memoire(exc: BaseException) -> bool:
    if type(exc).__name__ == "OutOfMemoryError":
        return True
    msg = str(exc).lower()
    return "out of memory" in msg or "not enough memory" in msg


def liberer_memoire(appareil: str = "") -> None:
    gc.collect()
    try:
        import torch
    except Exception:
        return
    type_ = appareil.split(":")[0]
    module = {"cuda": torch.cuda, "xpu": getattr(torch, "xpu", None), "mps": getattr(torch, "mps", None)}.get(type_)
    try:
        if module is not None:
            module.empty_cache()
    except Exception:
        pass


def chercher_lot(pas: Callable[[int], None], depart: int, maximum: int, *,
                 repetitions: int = 2, liberer: Callable[[], None] = lambda: None) -> int:
    """Plus grand lot ≤ maximum pour lequel `pas(lot)` passe ; 0 si même 1 échoue.

    `pas(lot)` fait un pas complet (forward, backward, optimizer.step) sur un lot
    factice. Il est répété (2 fois par défaut) : l'optimiseur alloue ses états au
    premier pas. Toute autre erreur qu'un manque de mémoire est propagée.
    """
    def essai(n: int) -> bool:
        try:
            for _ in range(repetitions):
                pas(n)
            return True
        except Exception as e:
            if not est_manque_memoire(e):
                raise
            liberer()
            return False
        finally:
            liberer()

    maximum = max(1, maximum)
    bon, mauvais = 0, maximum + 1
    n = max(1, min(depart, maximum))
    if essai(n):
        bon = n
        while bon < maximum:
            n = min(bon * 2, maximum)
            if not essai(n):
                mauvais = n
                break
            bon = n
    else:
        mauvais = n
        while mauvais > 1:
            n = mauvais // 2
            if essai(n):
                bon = n
                break
            mauvais = n
        if bon == 0:
            return 0
    while mauvais - bon > 1:
        n = (bon + mauvais) // 2
        if essai(n):
            bon = n
        else:
            mauvais = n
    return bon


def _chemin_cache(chemin: str | os.PathLike | None) -> Path:
    if chemin:
        return Path(chemin)
    if os.environ.get("MAMMOMESH_CACHE_CALIBRAGE"):
        return Path(os.environ["MAMMOMESH_CACHE_CALIBRAGE"])
    base = os.environ.get("XDG_CACHE_HOME") or Path.home() / ".cache"
    return Path(base) / "mammomesh" / "calibrage.json"


def calibrer(plan: Plan, modele: str, pas: Callable[[int], None], *, marge: float = 0.9,
             cache: str | os.PathLike | None = None) -> Plan:
    """Plan avec le lot mesuré (et l'accumulation recalculée). `modele` identifie
    l'architecture : pour la recherche évolutive, une empreinte du génome."""
    h, w = plan.taille_entree
    cle = "|".join([plan.nom_appareil, modele, f"{h}x{w}", plan.precision,
                    "ckpt" if plan.checkpointing else "plein"])
    fichier = _chemin_cache(cache)
    try:
        memo = json.loads(fichier.read_text(encoding="utf-8"))
    except (OSError, ValueError):
        memo = {}
    if cle in memo:
        return plan.avec_lot(int(memo[cle]))
    trouve = chercher_lot(pas, plan.lot, plan.lot_effectif, liberer=lambda: liberer_memoire(plan.appareil))
    if trouve == 0:
        raise MemoryError(f"Lot de 1 impossible sur {plan.nom_appareil} en {h}×{w} : "
                          "réduire la taille d'entrée ou activer le gradient checkpointing")
    lot = max(1, int(trouve * marge)) if trouve > 1 else 1
    memo[cle] = lot
    try:
        fichier.parent.mkdir(parents=True, exist_ok=True)
        fichier.write_text(json.dumps(memo, ensure_ascii=False, indent=2), encoding="utf-8")
    except OSError:
        pass
    return plan.avec_lot(lot)
