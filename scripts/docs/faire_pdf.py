"""PDF de la procédure Windows 11, tiré des mêmes données que la page de documentation (scripts/generer-docs.sh).

    python3 scripts/docs/faire_pdf.py DONNEES.json docs/installation-windows-11.pdf   (reportlab, polices DejaVu)
"""
import json, os, sys
from xml.sax.saxutils import escape
from reportlab.lib import colors
from reportlab.lib.enums import TA_LEFT
from reportlab.lib.pagesizes import A4
from reportlab.lib.styles import ParagraphStyle
from reportlab.lib.units import mm
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.ttfonts import TTFont
from reportlab.platypus import (KeepTogether, ListFlowable, ListItem, Paragraph, Preformatted, SimpleDocTemplate,
                                Spacer, Table, TableStyle)

D = os.environ.get("POLICES_DEJAVU", "/usr/share/fonts/truetype/dejavu/")   # paquet fonts-dejavu-core
pdfmetrics.registerFont(TTFont("Sans", D + "DejaVuSans.ttf"))
pdfmetrics.registerFont(TTFont("Sans-Bold", D + "DejaVuSans-Bold.ttf"))
pdfmetrics.registerFont(TTFont("Mono", D + "DejaVuSansMono.ttf"))
from reportlab.pdfbase.pdfmetrics import registerFontFamily
registerFontFamily("Sans", normal="Sans", bold="Sans-Bold", italic="Sans", boldItalic="Sans-Bold")

ACCENT = colors.HexColor("#2463a6"); GRIS = colors.HexColor("#66645e"); FOND = colors.HexColor("#f0efec"); BORD = colors.HexColor("#e2e0db")
st = {
    "titre": ParagraphStyle("titre", fontName="Sans-Bold", fontSize=20, leading=25, textColor=colors.HexColor("#1d1d1b"), spaceAfter=4),
    "sous": ParagraphStyle("sous", fontName="Sans", fontSize=9.5, leading=13, textColor=GRIS, spaceAfter=14),
    "h2": ParagraphStyle("h2", fontName="Sans-Bold", fontSize=13.5, leading=17, textColor=ACCENT, spaceBefore=12, spaceAfter=6),
    "p": ParagraphStyle("p", fontName="Sans", fontSize=9.6, leading=13.6, spaceAfter=6, alignment=TA_LEFT),
    "cell": ParagraphStyle("cell", fontName="Sans", fontSize=8.6, leading=11.4),
    "cellb": ParagraphStyle("cellb", fontName="Sans-Bold", fontSize=8.6, leading=11.4),
    "code": ParagraphStyle("code", fontName="Mono", fontSize=7.5, leading=10.2),
}
P = lambda t, s="p": Paragraph(escape(t), st[s])

def blocs(b, largeur):
    out = []
    if b.get("p"): out.append(P(b["p"]))
    for cle, type_ in (("etapes", "1"), ("points", "bullet")):
        if b.get(cle):
            out.append(ListFlowable([ListItem(P(e), leftIndent=14) for e in b[cle]], bulletType=type_, start=1 if type_ == "1" else None,
                                    bulletFontName="Sans", bulletFontSize=9, leftIndent=14, spaceAfter=6))
    if b.get("table"):
        t = b["table"]; n = len(t["entetes"])
        larg = [largeur * x for x in ({2: (0.32, 0.68), 3: (0.28, 0.2, 0.52)}.get(n) or [1 / n] * n)]
        data = [[P(h, "cellb") for h in t["entetes"]]] + [[P(c, "cellb" if i == 0 else "cell") for i, c in enumerate(l)] for l in t["lignes"]]
        tab = Table(data, colWidths=larg, repeatRows=1)
        tab.setStyle(TableStyle([("BACKGROUND", (0, 0), (-1, 0), FOND), ("GRID", (0, 0), (-1, -1), 0.5, BORD),
                                 ("VALIGN", (0, 0), (-1, -1), "TOP"), ("TOPPADDING", (0, 0), (-1, -1), 4), ("BOTTOMPADDING", (0, 0), (-1, -1), 4)]))
        out += [tab, Spacer(1, 8)]
    if b.get("code"):
        pre = Preformatted(b["code"], st["code"], maxLineLength=118, newLineChars="  ")
        boite = Table([[pre]], colWidths=[largeur])
        boite.setStyle(TableStyle([("BACKGROUND", (0, 0), (-1, -1), FOND), ("BOX", (0, 0), (-1, -1), 0.6, BORD),
                                   ("LEFTPADDING", (0, 0), (-1, -1), 8), ("TOPPADDING", (0, 0), (-1, -1), 6), ("BOTTOMPADDING", (0, 0), (-1, -1), 6)]))
        out += [boite, Spacer(1, 8)]
    return out

def pied(c, doc):
    c.saveState(); c.setFont("Sans", 7.5); c.setFillColor(GRIS)
    c.drawString(18 * mm, 10 * mm, "MammoMesh — installation sur Windows 11 — outil de recherche, aucun résultat ne sert à un diagnostic")
    c.drawRightString(A4[0] - 18 * mm, 10 * mm, f"page {doc.page}")
    c.restoreState()

def main(src, dest):
    d = json.load(open(src, encoding="utf-8"))
    doc = SimpleDocTemplate(dest, pagesize=A4, leftMargin=18 * mm, rightMargin=18 * mm, topMargin=16 * mm, bottomMargin=18 * mm,
                            title="Installer MammoMesh sur Windows 11", author="MammoMesh", subject="WSL 2 (Debian), Docker, GPU NVIDIA")
    largeur = A4[0] - 36 * mm
    story = [P("Installer MammoMesh sur Windows 11", "titre"),
             P(f"WSL 2 (Debian), Docker Engine, GPU NVIDIA, démarrage automatique. {d['etat']}. "
               "Même contenu que la page Documentation › Installation sur Windows 11 de l'interface.", "sous")]
    for p in d["parties"]:
        corps = [x for b in p["blocs"] for x in blocs(b, largeur)]
        story.append(KeepTogether([P(p["titre"], "h2")] + corps[:1]))
        story += corps[1:]
    doc.build(story, onFirstPage=pied, onLaterPages=pied)

main(sys.argv[1], sys.argv[2])
