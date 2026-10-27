"""Index de CMMD : une ligne par image, étiquette au niveau du sein, découpage par patiente.

Les données cliniques de CMMD (CMMD_clinicaldata_revision.xlsx) donnent, pour chaque sein qui porte
une anomalie, bénin ou malin prouvé par biopsie. Le sein opposé n'a pas de ligne : ses images sont
écartées (rien ne prouve qu'il soit sain). La tâche est donc « malin ou bénin, parmi les seins qui
portent une anomalie », pas « cancer ou sein normal ».
"""
from __future__ import annotations

import csv
import random
from dataclasses import asdict, dataclass, fields, replace
from pathlib import Path

ETIQUETTES = {"Malignant": 1, "Benign": 0}
SOUS_ENSEMBLES = ("entrainement", "ajustement", "validation")
PARTS_DEFAUT = (0.70, 0.15, 0.15)
VUES = {"cranio-caudal": "CC", "medio-lateral oblique": "MLO", "mediolateral oblique": "MLO"}
CLINIQUE = "metadonnees/CMMD_clinicaldata_revision.xlsx"


@dataclass(frozen=True)
class Image:
    patiente: str
    cote: str           # L ou R
    vue: str            # CC, MLO, ou ? si l'en-tête ne le dit pas
    fichier: str        # chemin relatif au dossier de la collection CMMD
    etiquette: int      # 1 malin, 0 bénin
    anomalie: str       # mass, calcification, both
    sous_ensemble: str = ""

    @property
    def sein(self) -> str:
        return f"{self.patiente}/{self.cote}"


def lire_clinique(xlsx: str | Path) -> dict[tuple[str, str], tuple[int, str]]:
    """(patiente, côté) → (étiquette, anomalie). Deux lignes pour le même sein : la maligne l'emporte."""
    import openpyxl
    wb = openpyxl.load_workbook(xlsx, read_only=True, data_only=True)
    lignes = wb.active.iter_rows(values_only=True)
    entetes = [str(h).strip() if h is not None else "" for h in next(lignes)]
    col = {h: k for k, h in enumerate(entetes)}
    manque = [c for c in ("ID1", "LeftRight", "classification", "abnormality") if c not in col]
    if manque:
        raise ValueError(f"{xlsx} : colonnes absentes {manque}")
    out: dict[tuple[str, str], tuple[int, str]] = {}
    for l in lignes:
        pid, cote, cls = l[col["ID1"]], l[col["LeftRight"]], l[col["classification"]]
        if not pid or cls not in ETIQUETTES or str(cote).strip().upper() not in ("L", "R"):
            continue
        cle = (str(pid).strip(), str(cote).strip().upper())
        val = (ETIQUETTES[cls], str(l[col["abnormality"]] or "").strip())
        if cle not in out or val[0] > out[cle][0]:
            out[cle] = val
    wb.close()
    return out


def _vue(ds) -> str:
    try:
        sens = str(ds.ViewCodeSequence[0].CodeMeaning).strip().lower()
        return VUES.get(sens, "?")
    except (AttributeError, IndexError):
        pos = str(getattr(ds, "ViewPosition", "")).upper()
        return pos if pos in ("CC", "MLO") else "?"


def indexer(dossier_cmmd: str | Path, *, limite_patientes: int | None = None) -> tuple[list[Image], dict]:
    """Parcourt les séries terminées (marqueur .ok) et garde les images dont le sein a une étiquette."""
    import pydicom
    dossier = Path(dossier_cmmd)
    clinique = lire_clinique(dossier / CLINIQUE)
    images: list[Image] = []
    stats = {"fichiers": 0, "sans_etiquette": 0, "illisibles": 0}
    patientes = sorted(p for p in dossier.iterdir() if p.is_dir() and p.name != "metadonnees")
    if limite_patientes:
        patientes = patientes[:limite_patientes]
    for p in patientes:
        for serie in sorted(s for s in p.iterdir() if (s / ".ok").exists()):
            for f in sorted(serie.glob("*.dcm")):
                stats["fichiers"] += 1
                try:
                    ds = pydicom.dcmread(f, stop_before_pixels=True)
                except Exception:
                    stats["illisibles"] += 1
                    continue
                pid = str(getattr(ds, "PatientID", p.name)).strip() or p.name
                cote = str(getattr(ds, "ImageLaterality", "") or getattr(ds, "Laterality", "")).strip().upper()
                if (pid, cote) not in clinique:
                    stats["sans_etiquette"] += 1
                    continue
                etiquette, anomalie = clinique[(pid, cote)]
                images.append(Image(pid, cote, _vue(ds), f.relative_to(dossier).as_posix(), etiquette, anomalie))
    return images, stats


def decouper(images: list[Image], parts: tuple[float, float, float] = PARTS_DEFAUT, graine: int = 42) -> list[Image]:
    """Découpage par patiente (jamais par image), stratifié sur « au moins un sein malin », reproductible."""
    if abs(sum(parts) - 1) > 1e-6:
        raise ValueError("les parts doivent faire 1")
    par_patiente: dict[str, int] = {}
    for im in images:
        par_patiente[im.patiente] = max(par_patiente.get(im.patiente, 0), im.etiquette)
    rng = random.Random(graine)
    affectation: dict[str, str] = {}
    for strate in (0, 1):
        groupe = sorted(p for p, e in par_patiente.items() if e == strate)
        rng.shuffle(groupe)
        n = len(groupe)
        n_entr = round(n * parts[0])
        n_ajust = round(n * parts[1])
        for k, p in enumerate(groupe):
            affectation[p] = SOUS_ENSEMBLES[0] if k < n_entr else SOUS_ENSEMBLES[1] if k < n_entr + n_ajust else SOUS_ENSEMBLES[2]
    return [replace(im, sous_ensemble=affectation[im.patiente]) for im in images]


def resumer(images: list[Image]) -> dict:
    out = {}
    for s in SOUS_ENSEMBLES:
        sel = [im for im in images if im.sous_ensemble == s]
        seins = {im.sein: im.etiquette for im in sel}
        out[s] = {"patientes": len({im.patiente for im in sel}), "seins": len(seins),
                  "seins_malins": sum(seins.values()), "images": len(sel)}
    return out


def ecrire_index(images: list[Image], chemin: str | Path) -> None:
    chemin = Path(chemin)
    chemin.parent.mkdir(parents=True, exist_ok=True)
    tmp = chemin.with_suffix(".tmp")
    with open(tmp, "w", newline="", encoding="utf-8") as f:
        w = csv.DictWriter(f, fieldnames=[x.name for x in fields(Image)])
        w.writeheader()
        for im in images:
            w.writerow(asdict(im))
    tmp.replace(chemin)


def lire_index(chemin: str | Path) -> list[Image]:
    with open(chemin, newline="", encoding="utf-8") as f:
        return [Image(**{**r, "etiquette": int(r["etiquette"])}) for r in csv.DictReader(f)]
