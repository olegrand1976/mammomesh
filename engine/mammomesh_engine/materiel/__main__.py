"""python -m mammomesh_engine.materiel [--resume | --profils | --json]"""
from __future__ import annotations

import argparse
import json

from .detection import detecter
from .plan import Exigences, planifier, selon_docker


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(prog="mammomesh_engine.materiel", description="Matériel détecté et plan d'exécution")
    sortie = ap.add_mutually_exclusive_group()
    sortie.add_argument("--resume", action="store_true", help="résumé lisible (défaut)")
    sortie.add_argument("--json", action="store_true", help="matériel et plan en JSON")
    sortie.add_argument("--profils", action="store_true", help="profils docker compose, séparés par des virgules")
    ap.add_argument("--lot-effectif", type=int, default=32)
    ap.add_argument("--resolution", default="1024x832", help="hauteur×largeur de l'image entière")
    ap.add_argument("--patch", type=int, default=512)
    ap.add_argument("--appareil", help="forcer un appareil (cuda:1, cpu…)")
    ap.add_argument("--docker", metavar="PROFILS", help="profils compose de l'hôte : signale un GPU que Docker ne verra pas")
    a = ap.parse_args(argv)

    h, w = (int(x) for x in a.resolution.lower().replace("×", "x").split("x"))
    m = detecter()
    plan = planifier(m, Exigences(lot_effectif=a.lot_effectif, resolution=(h, w), patch=a.patch), forcer=a.appareil)
    if a.docker is not None:
        plan = selon_docker(plan, a.docker)
    if a.profils:
        print(",".join(plan.profils_compose))
    elif a.json:
        print(json.dumps({"materiel": m.to_dict(), "plan": plan.to_dict()}, ensure_ascii=False, indent=2))
    else:
        accs = ", ".join(f"{x.nom} ({x.memoire_mio} Mio, {x.source})" for x in m.accelerateurs) or "aucun"
        print(f"Machine : {m.cpu}, {m.coeurs} cœurs, {m.ram_mio // 1024} Gio de RAM "
              f"({m.ram_libre_mio // 1024} libres), accélérateurs : {accs}")
        print(f"PyTorch : {m.torch or 'absent'} ; OpenVINO : {', '.join(m.openvino) or 'absent'} ; "
              f"ONNX Runtime : {', '.join(m.onnxruntime) or 'absent'}\n")
        print(plan.resume())
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
