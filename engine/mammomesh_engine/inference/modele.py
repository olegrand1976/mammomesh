"""Chargement d'un modèle entraîné (modele.pt), prédiction d'une image et zones d'attention (Grad-CAM)."""
from __future__ import annotations

import time
from dataclasses import dataclass
from pathlib import Path

import numpy as np

from ..cnn.pretraitement import a_retourner, ajuster, bornes_tissu, en_8_bits

SEUIL_ZONE = 0.5    # part du maximum de la carte d'attention au-dessus de laquelle une case compte
ZONES_MAX = 3


def choisir_modele(modeles: Path, nom: str | None = None) -> Path:
    """Dossier du modèle : celui demandé, sinon le plus récent entraînement fini (rapport.json),
    sinon le plus récent qui a déjà un modele.pt (entraînement en cours : meilleure époque à ce jour)."""
    if nom:
        d = modeles / nom
        if not (d / "modele.pt").exists():
            raise FileNotFoundError(f"{d / 'modele.pt'} introuvable")
        return d
    dossiers = sorted((d for d in modeles.iterdir() if (d / "modele.pt").exists()), key=lambda d: d.name, reverse=True)
    if not dossiers:
        raise FileNotFoundError(f"aucun modele.pt dans {modeles}")
    finis = [d for d in dossiers if (d / "rapport.json").exists()]
    return (finis or dossiers)[0]


def vers_type_source(px: np.ndarray, bits: int) -> np.ndarray:
    """Images 8 bits (CMMD) : l'entraînement les a lues en uint8, sans étirement des niveaux (en_8_bits) ;
    on fait de même, sinon le CNN verrait une image plus contrastée que celles qu'il a apprises."""
    if bits <= 8 and px.size and int(px.max()) <= 255:
        return px.astype(np.uint8)
    return px


@dataclass
class Geometrie:
    """Passage de l'entrée du CNN (toile H×W) aux pixels de l'image d'origine."""
    retourne: bool
    largeur_orig: int
    hauteur_orig: int
    ligne0: int
    col0: int
    k: float          # facteur de réduction (image recadrée → toile)
    haut: int         # marge verticale sur la toile
    nh: int
    nw: int

    def vers_origine(self, x: float, y: float, l: float, h: float) -> tuple[int, int, int, int]:
        """Boîte (toile) → boîte (image d'origine), bornée à l'image."""
        xr, yr = x / self.k, (y - self.haut) / self.k
        lr, hr = l / self.k, h / self.k
        xo, yo = xr + self.col0, yr + self.ligne0
        if self.retourne:
            xo = self.largeur_orig - (xo + lr)
        x0, y0 = max(0, int(round(xo))), max(0, int(round(yo)))
        x1 = min(self.largeur_orig, int(round(xo + lr)))
        y1 = min(self.hauteur_orig, int(round(yo + hr)))
        return x0, y0, max(1, x1 - x0), max(1, y1 - y0)


def preparer(px: np.ndarray, hauteur: int, largeur: int, inverser: bool) -> tuple[np.ndarray, Geometrie]:
    """Même chaîne que l'entraînement (pretraitement.preparer), en gardant la géométrie."""
    img = en_8_bits(px, inverser)
    H0, W0 = img.shape
    retourne = a_retourner(img)
    if retourne:
        img = img[:, ::-1]
    b = bornes_tissu(img)
    l0 = c0 = 0
    if b is not None:
        l0, c0 = b[0], b[2]
        img = img[b[0]:b[1] + 1, b[2]:b[3] + 1]
    h, w = img.shape
    k = min(hauteur / h, largeur / w)
    nh, nw = max(1, round(h * k)), max(1, round(w * k))
    toile = ajuster(img, hauteur, largeur)
    return toile, Geometrie(retourne, W0, H0, l0, c0, k, (hauteur - nh) // 2, nh, nw)


def zones(carte: np.ndarray, geo: Geometrie, hauteur: int, largeur: int) -> list[dict]:
    """Régions d'attention : cases contiguës au-dessus de SEUIL_ZONE × max, les plus fortes d'abord."""
    hc, wc = carte.shape
    ch, cw = hauteur / hc, largeur / wc
    # Hors du sein (marges noires de la toile) : ignoré.
    utile = np.zeros_like(carte, dtype=bool)
    for i in range(hc):
        for j in range(wc):
            cy, cx = (i + 0.5) * ch, (j + 0.5) * cw
            utile[i, j] = geo.haut <= cy < geo.haut + geo.nh and cx < geo.nw
    c = np.where(utile, carte, 0.0)
    if c.max() <= 0:
        return []
    c = c / c.max()
    masque = c >= SEUIL_ZONE
    vu = np.zeros_like(masque)
    out = []
    for i in range(hc):
        for j in range(wc):
            if not masque[i, j] or vu[i, j]:
                continue
            pile, cases = [(i, j)], []
            vu[i, j] = True
            while pile:
                a, b = pile.pop()
                cases.append((a, b))
                for da, db in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                    na, nb = a + da, b + db
                    if 0 <= na < hc and 0 <= nb < wc and masque[na, nb] and not vu[na, nb]:
                        vu[na, nb] = True
                        pile.append((na, nb))
            ls, cs = [a for a, _ in cases], [b for _, b in cases]
            x, y = min(cs) * cw, min(ls) * ch
            l, h = (max(cs) - min(cs) + 1) * cw, (max(ls) - min(ls) + 1) * ch
            xo, yo, lo, ho = geo.vers_origine(x, y, l, h)
            out.append({"x": xo, "y": yo, "largeur": lo, "hauteur": ho,
                        "score": round(float(max(c[a, b] for a, b in cases)), 3), "type": "attention du CNN"})
    out.sort(key=lambda z: -z["score"])
    return out[:ZONES_MAX]


class Modele:
    """Modèle chargé en mémoire ; recharge seul si modele.pt change (entraînement en cours)."""

    def __init__(self, dossier: Path, appareil: str = "cpu"):
        self.dossier, self.appareil = dossier, appareil
        self._mtime = None
        self.charger()

    def charger(self) -> None:
        import torch
        from ..cnn.entrainement import creer_modele
        fichier = self.dossier / "modele.pt"
        mtime = fichier.stat().st_mtime
        etat = torch.load(fichier, map_location="cpu", weights_only=True)
        m = creer_modele(False)
        m.load_state_dict(etat["etat"])
        m.eval().to(self.appareil)
        self.reseau = m
        self.taille = tuple(int(v) for v in etat.get("taille_entree", (1024, 832)))
        norm = etat.get("normalisation") or {"moyenne": (0.485, 0.456, 0.406), "ecart": (0.229, 0.224, 0.225)}
        self.moyenne, self.ecart = tuple(norm["moyenne"]), tuple(norm["ecart"])
        self.epoque = etat.get("epoque")
        self.seuil = float(etat["seuil"]) if etat.get("seuil") is not None else None
        self.fini = (self.dossier / "rapport.json").exists()
        self._mtime = mtime

    def a_jour(self) -> None:
        try:
            if (self.dossier / "modele.pt").stat().st_mtime != self._mtime:
                self.charger()
        except FileNotFoundError:
            pass

    def infos(self) -> dict:
        return {"modele": self.dossier.name, "epoque": self.epoque, "seuil": self.seuil, "fini": self.fini,
                "appareil": self.appareil, "taille_entree": list(self.taille)}

    def predire(self, px: np.ndarray, inverser: bool, bits: int = 16) -> dict:
        import torch
        import torch.nn.functional as F
        t0 = time.perf_counter()
        px = vers_type_source(px, bits)
        h, w = self.taille
        toile, geo = preparer(px, h, w, inverser)
        x = torch.from_numpy(toile).float().div_(255.0)[None, None].expand(1, 3, -1, -1)
        x = (x - torch.tensor(self.moyenne).view(1, 3, 1, 1)) / torch.tensor(self.ecart).view(1, 3, 1, 1)
        x = x.to(self.appareil)
        # Grad-CAM sur la dernière couche de convolution : poids = gradient moyen du logit par canal.
        caract = self.reseau.features(x)
        caract.retain_grad()
        logit = self.reseau.classifier(torch.flatten(self.reseau.avgpool(caract), 1))[0, 0]
        logit.backward()
        poids = caract.grad.mean(dim=(2, 3), keepdim=True)
        carte = F.relu((poids * caract).sum(dim=1))[0].detach().float().cpu().numpy()
        score = float(torch.sigmoid(logit.detach()).item())
        res = {"score": round(score, 4), "seuil": self.seuil, "zones": zones(carte, geo, h, w),
               "duree_ms": round((time.perf_counter() - t0) * 1000, 1), **self.infos()}
        if self.seuil is not None:
            res["signale"] = score >= self.seuil
        return res
