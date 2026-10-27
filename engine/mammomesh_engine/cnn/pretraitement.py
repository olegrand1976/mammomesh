"""DICOM → PNG 8 bits prêt pour le CNN : sein orienté à gauche, recadré sur le tissu, mis à la taille d'entrée.

Le cache est écrit une fois (reprise : les PNG présents sont gardés) et la place libre est contrôlée avant :
on refuse d'écrire si le disque garderait moins de MARGE_GO Go libres.
"""
from __future__ import annotations

import hashlib
import os
import shutil
from concurrent.futures import ProcessPoolExecutor
from pathlib import Path

import numpy as np

from .cmmd import Image

SEUIL_TISSU = 20        # niveau (0-255) au-dessus duquel un pixel compte comme tissu
PART_TISSU = 0.01       # une ligne ou colonne garde sa place si plus de 1 % de ses pixels sont du tissu


class PlaceInsuffisante(RuntimeError):
    pass


def marge_octets() -> int:
    return int(float(os.environ.get("MARGE_GO", "10")) * 1e9)


def controler_place(dossier: str | Path, besoin: int, quoi: str) -> None:
    """Refuse si écrire `besoin` octets dans `dossier` laisserait moins de MARGE_GO libres."""
    d = Path(dossier)
    d.mkdir(parents=True, exist_ok=True)
    libre = shutil.disk_usage(d).free
    if libre - besoin < marge_octets():
        raise PlaceInsuffisante(
            f"{quoi} : {besoin / 1e9:.1f} Go à écrire, {libre / 1e9:.1f} Go libres sur {d} "
            f"(marge {marge_octets() / 1e9:.0f} Go). Libérez de la place ou choisissez un autre dossier.")


def en_8_bits(px: np.ndarray, monochrome1: bool) -> np.ndarray:
    if px.dtype != np.uint8:
        px = px.astype(np.float32)
        bas, haut = np.percentile(px, (0.5, 99.5))
        px = np.clip((px - bas) / max(haut - bas, 1e-6) * 255.0, 0, 255).astype(np.uint8)
    return 255 - px if monochrome1 else px


def orienter(img: np.ndarray) -> np.ndarray:
    """Met le sein à gauche (le côté le plus clair) : toutes les images se ressemblent pour le CNN."""
    m = img.shape[1] // 2
    return img[:, ::-1] if img[:, m:].mean() > img[:, :m].mean() else img


def recadrer(img: np.ndarray) -> np.ndarray:
    tissu = img > SEUIL_TISSU
    lignes = np.flatnonzero(tissu.mean(axis=1) > PART_TISSU)
    cols = np.flatnonzero(tissu.mean(axis=0) > PART_TISSU)
    if len(lignes) < 16 or len(cols) < 16:
        return img
    return img[lignes[0]:lignes[-1] + 1, cols[0]:cols[-1] + 1]


def ajuster(img: np.ndarray, hauteur: int, largeur: int) -> np.ndarray:
    """Réduit en gardant les proportions, colle à gauche et centre verticalement sur fond noir."""
    from PIL import Image as Pil
    h, w = img.shape
    k = min(hauteur / h, largeur / w)
    nh, nw = max(1, round(h * k)), max(1, round(w * k))
    petit = np.asarray(Pil.fromarray(np.ascontiguousarray(img)).resize((nw, nh), Pil.Resampling.LANCZOS))
    toile = np.zeros((hauteur, largeur), dtype=np.uint8)
    haut = (hauteur - nh) // 2
    toile[haut:haut + nh, :nw] = petit
    return toile


def preparer(px: np.ndarray, hauteur: int, largeur: int, monochrome1: bool = False) -> np.ndarray:
    return ajuster(recadrer(orienter(en_8_bits(px, monochrome1))), hauteur, largeur)


def nom_cache(im: Image) -> str:
    h = hashlib.sha1(im.fichier.encode()).hexdigest()[:10]
    return f"{im.patiente}_{im.cote}_{im.vue}_{h}.png"


def _une(args: tuple[str, str, int, int]) -> str | None:
    source, cible, hauteur, largeur = args
    try:
        import pydicom
        from PIL import Image as Pil
        ds = pydicom.dcmread(source)
        img = preparer(ds.pixel_array, hauteur, largeur, str(getattr(ds, "PhotometricInterpretation", "")) == "MONOCHROME1")
        tmp = cible + ".tmp.png"
        Pil.fromarray(img).save(tmp, optimize=False, compress_level=3)
        os.replace(tmp, cible)
        return None
    except Exception as e:  # une image illisible ne doit pas arrêter les autres
        return f"{source} : {e}"


def pretraiter(images: list[Image], dossier_cmmd: str | Path, cache: str | Path, hauteur: int, largeur: int,
               *, processus: int | None = None, journal=print) -> dict:
    cache = Path(cache)
    cache.mkdir(parents=True, exist_ok=True)
    a_faire = [im for im in images if not (cache / nom_cache(im)).exists()]
    stats = {"deja_la": len(images) - len(a_faire), "faits": 0, "echecs": []}
    if not a_faire:
        return stats
    # Majorant : PNG non compressé.
    controler_place(cache, len(a_faire) * hauteur * largeur, f"Prétraitement de {len(a_faire)} image(s)")
    taches = [(str(Path(dossier_cmmd) / im.fichier), str(cache / nom_cache(im)), hauteur, largeur) for im in a_faire]
    processus = processus or max(1, min(8, (os.cpu_count() or 2) - 1))
    with ProcessPoolExecutor(max_workers=processus) as ex:
        for k, err in enumerate(ex.map(_une, taches, chunksize=8), 1):
            if err:
                stats["echecs"].append(err)
            else:
                stats["faits"] += 1
            if k % 250 == 0 or k == len(taches):
                journal(f"  prétraitement {k}/{len(taches)}")
    return stats
