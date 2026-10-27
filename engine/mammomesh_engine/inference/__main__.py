"""Service HTTP d'inférence du CNN (bibliothèque standard seulement, un calcul à la fois)."""
from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
import threading
import zlib
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import parse_qs, urlparse

import numpy as np

from .modele import Modele, choisir_modele

TAILLE_MAX = 512 * 1024 * 1024   # 16 000 × 16 000 pixels sur 16 bits, largement au-delà d'une mammographie


def adresse_docker() -> str | None:
    """Adresse de l'hôte sur docker0 : joignable depuis les conteneurs (host.docker.internal), pas depuis le réseau.
    None si docker0 n'existe pas (encore) : au démarrage de la machine, Docker peut être en retard sur ce service."""
    for ip in ("ip", "/usr/sbin/ip", "/sbin/ip"):
        try:
            sortie = subprocess.run([ip, "-4", "-o", "addr", "show", "docker0"], capture_output=True, text=True, timeout=5).stdout
            return sortie.split()[3].split("/")[0]
        except (OSError, IndexError, subprocess.SubprocessError):
            continue
    return None


def decompresser(brut: bytes, attendu: int) -> bytes:
    """gzip borné : jamais plus que la taille annoncée (+1 octet pour détecter un excès)."""
    if brut[:2] != b"\x1f\x8b":
        return brut
    d = zlib.decompressobj(16 + zlib.MAX_WBITS)
    return d.decompress(brut, attendu + 1)


def fabriquer(modele: Modele, jeton: str | None):
    verrou = threading.Lock()

    class Gestionnaire(BaseHTTPRequestHandler):
        server_version = "mammomesh-inference"

        def log_message(self, fmt, *args):  # journal sobre, sans contenu d'image
            print(f"{self.address_string()} {fmt % args}", flush=True)

        def _json(self, code: int, corps: dict) -> None:
            b = json.dumps(corps, ensure_ascii=False).encode()
            self.send_response(code)
            self.send_header("Content-Type", "application/json; charset=utf-8")
            self.send_header("Content-Length", str(len(b)))
            self.end_headers()
            self.wfile.write(b)

        def _autorise(self) -> bool:
            if jeton and self.headers.get("Authorization") != f"Bearer {jeton}":
                self._json(401, {"erreur": "jeton absent ou invalide"})
                return False
            return True

        def do_GET(self):
            if urlparse(self.path).path != "/sante":
                return self._json(404, {"erreur": "introuvable"})
            if not self._autorise():
                return
            with verrou:
                modele.a_jour()
                self._json(200, {"ok": True, **modele.infos()})

        def do_POST(self):
            u = urlparse(self.path)
            if u.path != "/predire":
                return self._json(404, {"erreur": "introuvable"})
            if not self._autorise():
                return
            q = {k: v[0] for k, v in parse_qs(u.query).items()}
            try:
                lignes, colonnes = int(q["lignes"]), int(q["colonnes"])
                n = int(self.headers.get("Content-Length", "0"))
            except (KeyError, ValueError):
                return self._json(400, {"erreur": "paramètres lignes et colonnes requis"})
            if not (0 < n <= TAILLE_MAX) or lignes <= 0 or colonnes <= 0 or 2 * lignes * colonnes > TAILLE_MAX:
                return self._json(400, {"erreur": "taille invalide"})
            try:
                brut = decompresser(self.rfile.read(n), 2 * lignes * colonnes)
            except zlib.error:
                return self._json(400, {"erreur": "gzip illisible"})
            if len(brut) != 2 * lignes * colonnes:
                return self._json(400, {"erreur": f"{len(brut)} octets reçus, {2 * lignes * colonnes} attendus"})
            px = np.frombuffer(brut, dtype="<u2").reshape(lignes, colonnes)
            try:
                with verrou:
                    modele.a_jour()
                    res = modele.predire(px, q.get("inverser") in ("1", "true"), int(q.get("bits", "16")))
            except Exception as e:  # une image ratée ne doit pas arrêter le service
                return self._json(500, {"erreur": f"inférence impossible : {e}"})
            self._json(200, res)

    return Gestionnaire


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--modeles", default=os.environ.get("MODELES_DIR", "../modeles"), help="dossier des entraînements")
    ap.add_argument("--modele", default=os.environ.get("MODELE"), help="nom du dossier ; défaut : le plus récent fini")
    ap.add_argument("--hote", default=os.environ.get("INFERENCE_HOTE", "127.0.0.1"),
                    help="adresse d'écoute ; « docker » : l'adresse de docker0 (pour l'API en conteneur)")
    ap.add_argument("--port", type=int, default=int(os.environ.get("INFERENCE_PORT", "8765")))
    ap.add_argument("--appareil", default=os.environ.get("INFERENCE_APPAREIL", "cpu"),
                    help="cpu (défaut : laisse le GPU aux entraînements) ou cuda")
    a = ap.parse_args()
    import torch
    torch.set_num_threads(max(1, min(8, (os.cpu_count() or 2) // 2)))
    dossier = choisir_modele(Path(a.modeles), a.modele)
    modele = Modele(dossier, a.appareil)
    hote = adresse_docker() if a.hote == "docker" else a.hote
    if hote is None:
        # Arrêt en erreur : systemd relance le service (RestartSec), Docker aura créé docker0 entre-temps.
        sys.exit("docker0 introuvable : Docker n'est pas (encore) démarré")
    jeton = os.environ.get("INFERENCE_JETON") or None
    srv = ThreadingHTTPServer((hote, a.port), fabriquer(modele, jeton))
    i = modele.infos()
    print(f"Inférence : {i['modele']} (époque {i['epoque']}, seuil {i['seuil']}, {'fini' if i['fini'] else 'en cours'})"
          f" sur {a.appareil}, http://{hote}:{a.port}", flush=True)
    srv.serve_forever()


if __name__ == "__main__":
    main()
