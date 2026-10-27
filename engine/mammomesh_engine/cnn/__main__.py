"""python -m mammomesh_engine.cnn [options] : index CMMD, prétraitement, entraînement, évaluation."""
from __future__ import annotations

import argparse
import json
import os
import sys
from datetime import datetime
from pathlib import Path

from . import cmmd
from .pretraitement import PlaceInsuffisante, nom_cache, pretraiter

DEPOT = Path(__file__).resolve().parents[3]


def _taille(s: str) -> tuple[int, int]:
    h, w = (int(x) for x in s.lower().replace("×", "x").split("x"))
    if h % 32 or w % 32:
        raise argparse.ArgumentTypeError("hauteur et largeur multiples de 32")
    return h, w


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(prog="mammomesh_engine.cnn", description=__doc__)
    ap.add_argument("--banque", type=Path, default=Path(os.environ.get("BANQUE_DIR", DEPOT / "banque")),
                    help="dossier de la banque (contient tcia/CMMD) ; défaut : $BANQUE_DIR ou ./banque")
    ap.add_argument("--cache", type=Path, default=DEPOT / "cache" / "cnn", help="PNG prétraités (disque local)")
    ap.add_argument("--sortie", type=Path, help="dossier du modèle ; défaut : ./modeles/cnn-cmmd-AAAAMMJJ-HHMM")
    ap.add_argument("--resolution", type=_taille, default=(1024, 832), help="hauteur×largeur (défaut 1024x832)")
    ap.add_argument("--epoques", type=int, default=15)
    ap.add_argument("--lot-effectif", type=int, default=32)
    ap.add_argument("--patience", type=int, default=5)
    ap.add_argument("--lr", type=float, default=3e-4)
    ap.add_argument("--graine", type=int, default=42)
    ap.add_argument("--sensibilite-cible", type=float, default=0.85, help="pour choisir le seuil en ajustement")
    ap.add_argument("--limite", type=int, help="nombre de patientes (essai rapide)")
    ap.add_argument("--appareil", help="forcer un appareil (cuda:0, cpu…)")
    ap.add_argument("--sans-preentrainement", action="store_true", help="poids initiaux aléatoires au lieu d'ImageNet")
    ap.add_argument("--sans-compile", action="store_true", help="ne pas utiliser torch.compile")
    ap.add_argument("--jusqua", choices=["index", "pretraitement"], help="s'arrêter après cette étape")
    a = ap.parse_args(argv)

    dossier = a.banque / "tcia" / "CMMD"
    if not (dossier / cmmd.CLINIQUE).exists():
        print(f"CMMD introuvable dans {dossier} (lancer : ./examples/telecharger.sh --banque CMMD)", file=sys.stderr)
        return 1
    h, w = a.resolution
    cache = a.cache / f"cmmd-{h}x{w}"
    try:
        print(f"== Index CMMD ({dossier})")
        images, stats = cmmd.indexer(dossier, limite_patientes=a.limite)
        images = cmmd.decouper(images, graine=a.graine)
        cmmd.ecrire_index(images, cache / "index.csv")
        print(f"  {stats['fichiers']} fichiers, {len(images)} images étiquetées, "
              f"{stats['sans_etiquette']} sans étiquette (sein opposé), {stats['illisibles']} illisibles")
        for s, r in cmmd.resumer(images).items():
            print(f"  {s:12} : {r['patientes']} patientes, {r['seins']} seins dont {r['seins_malins']} malins, {r['images']} images")
        if a.jusqua == "index":
            return 0

        print(f"== Prétraitement {h}×{w} → {cache}")
        p = pretraiter(images, dossier, cache, h, w)
        print(f"  {p['faits']} faits, {p['deja_la']} déjà là, {len(p['echecs'])} échec(s)")
        for e in p["echecs"][:10]:
            print(f"  ÉCHEC {e}", file=sys.stderr)
        if p["echecs"]:
            images = [im for im in images if (cache / nom_cache(im)).exists()]
        if a.jusqua == "pretraitement":
            return 0

        from .entrainement import entrainer
        sortie = a.sortie or DEPOT / "modeles" / f"cnn-cmmd-{datetime.now():%Y%m%d-%H%M}"
        print(f"== Entraînement → {sortie}")
        r = entrainer(images, cache, sortie, resolution=a.resolution, epoques=a.epoques, lot_effectif=a.lot_effectif,
                      patience=a.patience, lr=a.lr, preentraine=not a.sans_preentrainement, appareil=a.appareil,
                      compiler=not a.sans_compile, sensibilite_cible=a.sensibilite_cible, graine=a.graine)
    except PlaceInsuffisante as e:
        print(f"REFUS : {e}", file=sys.stderr)
        return 2
    v = r["validation"]
    fmt = lambda x: "—" if x is None else f"{x:.3f}"
    print(f"== Validation (E-val, jamais vue pendant l'entraînement) : AUC {fmt(v['auc_sein'])} par sein, "
          f"{fmt(v['auc_image'])} par image ; au seuil {v['seuil']:.3f} : sensibilité {fmt(v.get('sensibilite_sein'))}, "
          f"spécificité {fmt(v.get('specificite_sein'))}")
    print(f"Rapport : {sortie / 'rapport.json'}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
