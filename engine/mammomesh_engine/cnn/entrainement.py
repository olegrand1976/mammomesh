"""Entraînement et évaluation du CNN de base (EfficientNet-B0, sortie : probabilité de malignité).

Le plan matériel (materiel.planifier) choisit l'appareil, la précision et les workers ; sur GPU, calibrer()
mesure le plus grand lot qui tient. Si l'image entière ne tient pas, la résolution descend d'un cran.
"""
from __future__ import annotations

import csv
import json
import math
import random
import time
from dataclasses import replace
from datetime import datetime, timezone
from pathlib import Path

from ..materiel import Exigences, calibrer, detecter, planifier
from ..materiel import torch_outils as to
from .cmmd import Image
from .metriques import mesurer, seuil_pour_sensibilite
from .pretraitement import controler_place, nom_cache

ARCHITECTURE = "efficientnet_b0"
MOYENNE, ECART = (0.485, 0.456, 0.406), (0.229, 0.224, 0.225)   # normalisation ImageNet
PALIERS = ((1024, 832), (768, 624), (512, 416))


def creer_modele(preentraine: bool):
    import torch.nn as nn
    from torchvision.models import EfficientNet_B0_Weights, efficientnet_b0
    m = efficientnet_b0(weights=EfficientNet_B0_Weights.IMAGENET1K_V1 if preentraine else None)
    m.classifier[1] = nn.Linear(m.classifier[1].in_features, 1)
    return m


class JeuImages:
    """PNG du cache → tenseur (3, H, W) normalisé ; augmentations légères en entraînement."""

    def __init__(self, images: list[Image], cache: Path, taille: tuple[int, int], augmenter: bool):
        self.images, self.cache, self.taille, self.augmenter = images, cache, taille, augmenter
        self._aug = None

    def __len__(self) -> int:
        return len(self.images)

    def __getitem__(self, i: int):
        import numpy as np
        import torch
        import torch.nn.functional as F
        from PIL import Image as Pil
        im = self.images[i]
        x = torch.from_numpy(np.asarray(Pil.open(self.cache / nom_cache(im)), dtype=np.uint8).copy())[None]
        if tuple(x.shape[-2:]) != self.taille:
            x = F.interpolate(x[None].float(), size=self.taille, mode="area")[0].round().clamp(0, 255).to(torch.uint8)
        if self.augmenter:
            if self._aug is None:
                from torchvision.transforms import v2
                self._aug = v2.Compose([
                    v2.RandomVerticalFlip(0.5),
                    v2.RandomAffine(degrees=10, translate=(0.04, 0.04), scale=(0.9, 1.1)),
                    v2.ColorJitter(brightness=0.15, contrast=0.15),
                ])
            x = self._aug(x)
        x = x.float().div_(255.0).expand(3, -1, -1)
        moy = torch.tensor(MOYENNE).view(3, 1, 1)
        ec = torch.tensor(ECART).view(3, 1, 1)
        return (x - moy) / ec, torch.tensor(float(im.etiquette))


def _figer_batchnorm(modele) -> None:
    import torch.nn as nn
    for m in modele.modules():
        if isinstance(m, nn.modules.batchnorm._BatchNorm):
            m.eval()


def choisir_plan(*, resolution: tuple[int, int], lot_effectif: int, appareil: str | None, compiler: bool, journal=print):
    """Plan matériel + lot mesuré ; descend d'une résolution si l'image entière ne tient pas."""
    import torch
    m = detecter()
    paliers = [resolution] + [p for p in PALIERS if p[0] < resolution[0]]
    dernier = None
    for taille in paliers:
        plan = planifier(m, Exigences(lot_effectif=lot_effectif, resolution=taille), forcer=appareil)
        plan = replace(plan, entree="image", taille_entree=taille, checkpointing=False,
                       compile=plan.compile and compiler)
        if plan.appareil == "cpu":
            return plan   # pas de calibrage sur CPU : un essai trop gros ferait tuer le processus par le système
        dev = to.preparer(plan)

        def pas(n: int) -> None:
            mod = creer_modele(False).to(dev)
            if plan.channels_last:
                mod = mod.to(memory_format=torch.channels_last)
            opt = torch.optim.AdamW(mod.parameters(), lr=1e-4)
            sc = to.scaler(plan)
            x = torch.randn(n, 3, *taille, device=dev)
            with to.autocast(plan):
                perte = mod(x).float().mean()
            sc.scale(perte).backward()
            sc.step(opt)
            sc.update()
            del mod, opt, x

        try:
            return calibrer(plan, f"{ARCHITECTURE}-classif", pas)
        except MemoryError as e:
            journal(f"  {e} : résolution inférieure")
            dernier = e
    raise MemoryError(f"Aucune résolution ne tient sur cet appareil ({dernier})")


def _predire(modele, chargeur, plan, dev) -> list[float]:
    import torch
    modele.eval()
    sorties: list[float] = []
    with torch.no_grad(), to.autocast(plan):
        for x, _ in chargeur:
            x = x.to(dev, non_blocking=True)
            if plan.channels_last:
                x = x.contiguous(memory_format=torch.channels_last)
            sorties += torch.sigmoid(modele(x).float().squeeze(1)).cpu().tolist()
    return sorties


def _lignes(images: list[Image], scores: list[float]) -> list[tuple[str, int, float]]:
    return [(im.sein, im.etiquette, s) for im, s in zip(images, scores)]


def entrainer(images: list[Image], cache: Path, sortie: Path, *, resolution: tuple[int, int] = PALIERS[0],
              epoques: int = 15, lot_effectif: int = 32, patience: int = 5, lr: float = 3e-4,
              preentraine: bool = True, appareil: str | None = None, compiler: bool = True,
              sensibilite_cible: float = 0.85, graine: int = 42, journal=print) -> dict:
    import torch
    from torch.utils.data import DataLoader

    random.seed(graine)
    torch.manual_seed(graine)
    controler_place(sortie, 200_000_000, "Modèle et journaux")   # poids (~16 Mo), copies, journaux
    sortie.mkdir(parents=True, exist_ok=True)

    plan = choisir_plan(resolution=resolution, lot_effectif=lot_effectif, appareil=appareil, compiler=compiler, journal=journal)
    taille = plan.taille_entree
    journal(plan.resume())
    dev = to.preparer(plan)

    sel = {s: [im for im in images if im.sous_ensemble == s] for s in ("entrainement", "ajustement", "validation")}
    opts = to.options_dataloader(plan)
    ch_entr = DataLoader(JeuImages(sel["entrainement"], cache, taille, True), batch_size=plan.lot, shuffle=True,
                         drop_last=len(sel["entrainement"]) > plan.lot, **opts)
    ch_ajust = DataLoader(JeuImages(sel["ajustement"], cache, taille, False), batch_size=plan.lot, **opts)
    ch_val = DataLoader(JeuImages(sel["validation"], cache, taille, False), batch_size=plan.lot, **opts)

    modele = creer_modele(preentraine)
    figer_bn = preentraine and plan.lot < 8
    modele = to.preparer_modele(plan, modele)
    n_pos = sum(im.etiquette for im in sel["entrainement"])
    n_neg = len(sel["entrainement"]) - n_pos
    critere = torch.nn.BCEWithLogitsLoss(pos_weight=torch.tensor(n_neg / max(n_pos, 1), device=dev))
    opt = torch.optim.AdamW(modele.parameters(), lr=lr, weight_decay=1e-4)
    pas_par_epoque = max(1, math.ceil(len(ch_entr) / plan.accumulation))
    total, chauffe = pas_par_epoque * epoques, pas_par_epoque
    sched = torch.optim.lr_scheduler.LambdaLR(opt, lambda k: min(1.0, (k + 1) / chauffe)
                                              * 0.5 * (1 + math.cos(math.pi * min(k, total) / total)))
    sc = to.scaler(plan)

    meilleur, sans_progres, historique = -1.0, 0, []
    poids = sortie / "modele.pt"
    with open(sortie / "journal.csv", "w", newline="", encoding="utf-8") as jf:
        jw = csv.writer(jf)
        jw.writerow(["epoque", "perte", "auc_image_ajustement", "auc_sein_ajustement", "duree_s"])
        for ep in range(1, epoques + 1):
            t0 = time.time()
            modele.train()
            if figer_bn:
                _figer_batchnorm(modele)
            opt.zero_grad(set_to_none=True)
            cumul, n = 0.0, 0
            for k, (x, y) in enumerate(ch_entr, 1):
                x, y = x.to(dev, non_blocking=True), y.to(dev, non_blocking=True)
                if plan.channels_last:
                    x = x.contiguous(memory_format=torch.channels_last)
                with to.autocast(plan):
                    logits = modele(x).float().squeeze(1)
                perte = critere(logits, y) / plan.accumulation
                sc.scale(perte).backward()
                if k % plan.accumulation == 0 or k == len(ch_entr):
                    sc.step(opt)
                    sc.update()
                    opt.zero_grad(set_to_none=True)
                    sched.step()
                cumul += perte.item() * plan.accumulation * len(y)
                n += len(y)
            m_aj = mesurer(_lignes(sel["ajustement"], _predire(modele, ch_ajust, plan, dev)))
            duree = time.time() - t0
            auc_s = m_aj["auc_sein"] or 0.0
            historique.append({"epoque": ep, "perte": cumul / max(n, 1), **m_aj, "duree_s": duree})
            jw.writerow([ep, f"{cumul / max(n, 1):.4f}", f"{m_aj['auc_image'] or 0:.4f}", f"{auc_s:.4f}", f"{duree:.0f}"])
            jf.flush()
            mieux = auc_s > meilleur
            journal(f"  époque {ep}/{epoques} : perte {cumul / max(n, 1):.4f}, AUC ajustement {auc_s:.3f} par sein"
                    f" ({m_aj['auc_image'] or 0:.3f} par image), {duree:.0f} s" + (" ← meilleure" if mieux else ""))
            if mieux:
                meilleur, sans_progres = auc_s, 0
                etat = getattr(modele, "_orig_mod", modele).state_dict()
                torch.save({"etat": etat, "architecture": ARCHITECTURE, "taille_entree": taille,
                            "normalisation": {"moyenne": MOYENNE, "ecart": ECART}, "epoque": ep}, poids)
            else:
                sans_progres += 1
                if sans_progres >= patience:
                    journal(f"  arrêt : pas de progrès depuis {patience} époques")
                    break

    # Meilleur modèle : seuil choisi en ajustement, mesures finales en validation (jamais vues avant).
    meilleur_etat = torch.load(poids, map_location=dev, weights_only=True)
    getattr(modele, "_orig_mod", modele).load_state_dict(meilleur_etat["etat"])
    lignes_aj = _lignes(sel["ajustement"], _predire(modele, ch_ajust, plan, dev))
    from .metriques import par_sein
    _, e_aj, s_aj = par_sein(lignes_aj)
    seuil = seuil_pour_sensibilite(s_aj, e_aj, sensibilite_cible)
    m_aj = mesurer(lignes_aj, seuil)
    m_val = mesurer(_lignes(sel["validation"], _predire(modele, ch_val, plan, dev)), seuil)
    meilleur_etat["seuil"] = seuil
    torch.save(meilleur_etat, poids)

    rapport = {
        "date": datetime.now(timezone.utc).isoformat(timespec="seconds"),
        "tache": "malin ou bénin, parmi les seins de CMMD qui portent une anomalie (biopsie)",
        "avertissement": "Outil de recherche : aucun résultat ne sert à un diagnostic.",
        "donnees": {"jeu": "CMMD", "licence": "CC BY 4.0", "doi": "10.7937/tcia.eqde-4b16",
                    **{s: {"images": len(v), "seins_malins": len({im.sein for im in v if im.etiquette})}
                       for s, v in sel.items()}},
        "modele": {"architecture": ARCHITECTURE, "preentraine_imagenet": preentraine, "taille_entree": list(taille),
                   "epoque_retenue": meilleur_etat["epoque"], "poids": str(poids.name)},
        "plan": plan.to_dict(),
        "reglages": {"epoques_max": epoques, "lot_effectif": lot_effectif, "patience": patience, "lr": lr,
                     "graine": graine, "sensibilite_cible": sensibilite_cible, "batchnorm_figee": figer_bn},
        "ajustement": m_aj,
        "validation": m_val,
        "historique": historique,
    }
    (sortie / "rapport.json").write_text(json.dumps(rapport, ensure_ascii=False, indent=2), encoding="utf-8")
    return rapport
