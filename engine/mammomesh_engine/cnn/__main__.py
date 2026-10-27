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
from .suivi import Suivi

DEPOT = Path(__file__).resolve().parents[3]
PARAMETRES = "parametres.json"
CLINIQUE_REL = Path("tcia") / "CMMD" / cmmd.CLINIQUE   # présence de CMMD dans une banque
REGLAGES = ("banque", "cache", "resolution", "epoques", "lot_effectif", "patience", "lr", "graine",
            "sensibilite_cible", "limite", "appareil", "sans_preentrainement", "sans_compile")


def _ecrire_reglages(a: argparse.Namespace, sortie: Path) -> None:
    """Réglages du lancement, relus par --reprendre (et par le veilleur auto.py)."""
    sortie.mkdir(parents=True, exist_ok=True)
    r = {k: getattr(a, k) for k in REGLAGES}
    r.update(banque=str(Path(a.banque).resolve()), cache=str(Path(a.cache).resolve()), resolution=list(a.resolution))
    (sortie / PARAMETRES).write_text(json.dumps(r, ensure_ascii=False, indent=1), encoding="utf-8")


def _reglages_de(dossier: Path) -> argparse.Namespace:
    r = json.loads((dossier / PARAMETRES).read_text(encoding="utf-8"))
    return argparse.Namespace(
        banque=Path(r["banque"]), cache=Path(r["cache"]), resolution=tuple(r["resolution"]), epoques=r["epoques"],
        lot_effectif=r["lot_effectif"], patience=r["patience"], lr=r["lr"], graine=r["graine"],
        sensibilite_cible=r["sensibilite_cible"], limite=r.get("limite"), appareil=r.get("appareil"),
        sans_preentrainement=r.get("sans_preentrainement", False), sans_compile=r.get("sans_compile", False),
        sortie=dossier, jusqua=None, reprendre=dossier)


def _taille(s: str) -> tuple[int, int]:
    h, w = (int(x) for x in s.lower().replace("×", "x").split("x"))
    if h % 32 or w % 32:
        raise argparse.ArgumentTypeError("hauteur et largeur multiples de 32")
    return h, w


def analyseur() -> argparse.ArgumentParser:
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
    ap.add_argument("--reprendre", type=Path, metavar="DOSSIER",
                    help="reprendre un entraînement interrompu (mêmes réglages, depuis la dernière époque finie)")
    return ap


def main(argv: list[str] | None = None) -> int:
    a = analyseur().parse_args(argv)
    if a.reprendre:
        try:
            a = _reglages_de(a.reprendre)
        except (OSError, ValueError) as e:
            print(f"Reprise impossible : {a.reprendre / PARAMETRES} illisible ({e})", file=sys.stderr)
            return 1

    dossier = a.banque / "tcia" / "CMMD"
    if not (dossier / cmmd.CLINIQUE).exists():
        print(f"CMMD introuvable dans {dossier} (lancer : ./examples/telecharger.sh --banque CMMD)", file=sys.stderr)
        return 1
    h, w = a.resolution
    cache = a.cache / f"cmmd-{h}x{w}"
    sortie = a.sortie or DEPOT / "modeles" / f"cnn-cmmd-{datetime.now():%Y%m%d-%H%M}"
    suivi = None
    if a.jusqua is None:
        commande = " ".join(sys.argv[1:] if argv is None else argv)
        suivi = Suivi(sortie, reprise=a.reprendre is not None, jeu="CMMD", **({} if a.reprendre else {"commande": commande}))
        if not a.reprendre:
            _ecrire_reglages(a, sortie)
    try:
        if suivi:
            suivi.phase("index")
        print(f"== Index CMMD ({dossier})")
        if a.reprendre and (sortie / "index.csv").exists():
            images = cmmd.lire_index(sortie / "index.csv")   # exactement le même découpage qu'au lancement
            print(f"  {len(images)} images, index repris du premier lancement")
        else:
            images, stats = cmmd.indexer(dossier, limite_patientes=a.limite)
            images = cmmd.decouper(images, graine=a.graine)
            print(f"  {stats['fichiers']} fichiers, {len(images)} images étiquetées, "
                  f"{stats['sans_etiquette']} sans étiquette (sein opposé), {stats['illisibles']} illisibles")
        cmmd.ecrire_index(images, cache / "index.csv")
        if suivi:
            cmmd.ecrire_index(images, sortie / "index.csv")
        for s, r in cmmd.resumer(images).items():
            print(f"  {s:12} : {r['patientes']} patientes, {r['seins']} seins dont {r['seins_malins']} malins, {r['images']} images")
        if a.jusqua == "index":
            return 0

        print(f"== Prétraitement {h}×{w} → {cache}")
        if suivi:
            suivi.phase("pretraitement", donnees=cmmd.resumer(images))

        def journal_pre(msg: str) -> None:
            print(msg)
            if suivi and "/" in msg:
                fait, total = msg.split()[-1].split("/")
                suivi.progres(pretraites=int(fait), a_pretraiter=int(total))

        p = pretraiter(images, dossier, cache, h, w, journal=journal_pre)
        print(f"  {p['faits']} faits, {p['deja_la']} déjà là, {len(p['echecs'])} échec(s)")
        for e in p["echecs"][:10]:
            print(f"  ÉCHEC {e}", file=sys.stderr)
        if p["echecs"]:
            images = [im for im in images if (cache / nom_cache(im)).exists()]
        if a.jusqua == "pretraitement":
            return 0

        from .entrainement import entrainer
        print(f"== Entraînement → {sortie}")
        r = entrainer(images, cache, sortie, resolution=a.resolution, epoques=a.epoques, lot_effectif=a.lot_effectif,
                      patience=a.patience, lr=a.lr, preentraine=not a.sans_preentrainement, appareil=a.appareil,
                      compiler=not a.sans_compile, sensibilite_cible=a.sensibilite_cible, graine=a.graine,
                      suivi=suivi, reprendre=a.reprendre is not None)
    except PlaceInsuffisante as e:
        print(f"REFUS : {e}", file=sys.stderr)
        if suivi:
            suivi.fin("echec", f"Place insuffisante : {e}")
        return 2
    except KeyboardInterrupt:
        if suivi:
            suivi.fin("interrompu", "Arrêté à la main (Ctrl+C)")
        raise
    except BaseException as e:
        if suivi:
            suivi.fin("echec", f"{type(e).__name__} : {e}")
        raise
    if suivi:
        suivi.fin("termine")
    v = r["validation"]
    fmt = lambda x: "—" if x is None else f"{x:.3f}"
    print(f"== Validation (E-val, jamais vue pendant l'entraînement) : AUC {fmt(v['auc_sein'])} par sein, "
          f"{fmt(v['auc_image'])} par image ; au seuil {v['seuil']:.3f} : sensibilité {fmt(v.get('sensibilite_sein'))}, "
          f"spécificité {fmt(v.get('specificite_sein'))}")
    print(f"Rapport : {sortie / 'rapport.json'}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
