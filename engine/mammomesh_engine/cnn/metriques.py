"""Mesures sans dépendance : AUC, seuil pour une sensibilité cible, sensibilité et spécificité."""
from __future__ import annotations

from collections import defaultdict
from typing import Iterable, Sequence


def auc(scores: Sequence[float], etiquettes: Sequence[int]) -> float | None:
    """Aire sous la courbe ROC (statistique de Mann-Whitney, ex aequo comptés pour moitié)."""
    paires = sorted(zip(scores, etiquettes), key=lambda x: x[0])
    n_pos = sum(1 for _, e in paires if e == 1)
    n_neg = len(paires) - n_pos
    if n_pos == 0 or n_neg == 0:
        return None
    somme_rangs, i = 0.0, 0
    while i < len(paires):
        j = i
        while j + 1 < len(paires) and paires[j + 1][0] == paires[i][0]:
            j += 1
        rang = (i + j) / 2 + 1
        somme_rangs += rang * sum(1 for k in range(i, j + 1) if paires[k][1] == 1)
        i = j + 1
    return (somme_rangs - n_pos * (n_pos + 1) / 2) / (n_pos * n_neg)


def sens_spec(scores: Sequence[float], etiquettes: Sequence[int], seuil: float) -> tuple[float | None, float | None]:
    vp = sum(1 for s, e in zip(scores, etiquettes) if e == 1 and s >= seuil)
    vn = sum(1 for s, e in zip(scores, etiquettes) if e == 0 and s < seuil)
    pos = sum(etiquettes)
    neg = len(etiquettes) - pos
    return (vp / pos if pos else None, vn / neg if neg else None)


def seuil_pour_sensibilite(scores: Sequence[float], etiquettes: Sequence[int], cible: float) -> float:
    """Plus haut seuil dont la sensibilité atteint la cible (donc la meilleure spécificité à cette sensibilité)."""
    positifs = sorted((s for s, e in zip(scores, etiquettes) if e == 1), reverse=True)
    if not positifs:
        return 0.5
    besoin = max(1, -(-int(round(cible * len(positifs) * 1e6)) // 1_000_000))  # plafond, sans erreur d'arrondi
    return positifs[min(besoin, len(positifs)) - 1]


def par_sein(lignes: Iterable[tuple[str, int, float]]) -> tuple[list[str], list[int], list[float]]:
    """(sein, étiquette, score par image) → score par sein = moyenne de ses vues."""
    cumul: dict[str, list[float]] = defaultdict(list)
    etiq: dict[str, int] = {}
    for sein, e, s in lignes:
        cumul[sein].append(s)
        etiq[sein] = e
    seins = sorted(cumul)
    return seins, [etiq[s] for s in seins], [sum(cumul[s]) / len(cumul[s]) for s in seins]


def mesurer(lignes: list[tuple[str, int, float]], seuil: float | None = None) -> dict:
    """AUC par image et par sein ; sensibilité et spécificité par sein au seuil donné."""
    _, e_img, s_img = zip(*lignes) if lignes else ((), (), ())
    _, e_sein, s_sein = par_sein(lignes)
    out = {"images": len(lignes), "seins": len(e_sein), "seins_malins": sum(e_sein),
           "auc_image": auc(s_img, e_img), "auc_sein": auc(s_sein, e_sein)}
    if seuil is not None:
        se, sp = sens_spec(s_sein, e_sein, seuil)
        out.update(seuil=seuil, sensibilite_sein=se, specificite_sein=sp)
    return out
