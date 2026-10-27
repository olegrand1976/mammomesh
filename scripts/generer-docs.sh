#!/usr/bin/env bash
# Régénère les documents du dépôt tirés des données de l'interface (web/app/utils) :
#   docs/glossaire.md, docs/installation-windows-11.md, docs/installation-windows-11.pdf
#   (et sa copie web/public/docs/, servie par l'interface pour le téléchargement).
# Dépendances : Node 22.6 ou plus récent, Python 3 avec reportlab (sudo apt install python3-reportlab
# ou pip install reportlab), polices DejaVu (paquet fonts-dejavu-core).
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
node --experimental-strip-types --no-warnings scripts/docs/generer.mjs docs "$tmp/windows.json"
python3 -c 'import reportlab' 2>/dev/null || { echo "reportlab absent : sudo apt install python3-reportlab" >&2; exit 1; }
python3 scripts/docs/faire_pdf.py "$tmp/windows.json" docs/installation-windows-11.pdf
mkdir -p web/public/docs
cp docs/installation-windows-11.pdf web/public/docs/
echo "À jour : docs/glossaire.md, docs/installation-windows-11.md, docs/installation-windows-11.pdf (+ web/public/docs/)"
