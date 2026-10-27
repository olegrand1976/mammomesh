"""Adaptation automatique au matériel.

    from mammomesh_engine.materiel import detecter, planifier, Exigences
    plan = planifier(detecter(), Exigences(lot_effectif=32))
    plan = calibrer(plan, "efficientnet_b0", pas)   # mesure réelle du lot, mise en cache

Le plan dit sur quel appareil entraîner, en quelle précision, avec quel lot, combien
d'individus entraîner en parallèle et où faire l'inférence. Il ne dépend que du
matériel détecté : la même campagne tourne sur un portable comme sur un serveur.
"""
from .calibrage import calibrer, chercher_lot, est_manque_memoire
from .detection import Accelerateur, Materiel, detecter
from .plan import Exigences, Plan, planifier, selon_docker

__all__ = [
    "Accelerateur", "Materiel", "detecter",
    "Exigences", "Plan", "planifier", "selon_docker",
    "calibrer", "chercher_lot", "est_manque_memoire",
]
