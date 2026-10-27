"""CNN de base : découpage par patiente, données cliniques, mesures, prétraitement (sans PyTorch)."""
import importlib.util
import os
import tempfile
import unittest
from pathlib import Path

from mammomesh_engine.cnn import cmmd
from mammomesh_engine.cnn.metriques import auc, mesurer, par_sein, sens_spec, seuil_pour_sensibilite

A_NUMPY = importlib.util.find_spec("numpy") is not None and importlib.util.find_spec("PIL") is not None
A_OPENPYXL = importlib.util.find_spec("openpyxl") is not None


def fausses_images(n_patientes=200):
    out = []
    for k in range(n_patientes):
        pid = f"D1-{k:04d}"
        e = 1 if k % 10 < 7 else 0          # 70 % malins, comme CMMD
        for vue in ("CC", "MLO"):
            out.append(cmmd.Image(pid, "L", vue, f"{pid}/s/{vue}.dcm", e, "mass"))
        if k % 20 == 0:                     # quelques patientes avec les deux seins étiquetés
            for vue in ("CC", "MLO"):
                out.append(cmmd.Image(pid, "R", vue, f"{pid}/s/R{vue}.dcm", 0, "calcification"))
    return out


class DecoupageTest(unittest.TestCase):
    def test_par_patiente_et_reproductible(self):
        a = cmmd.decouper(fausses_images())
        b = cmmd.decouper(fausses_images())
        self.assertEqual(a, b)
        par_patiente = {}
        for im in a:
            par_patiente.setdefault(im.patiente, set()).add(im.sous_ensemble)
        self.assertTrue(all(len(s) == 1 for s in par_patiente.values()), "une patiente des deux côtés")
        self.assertNotEqual(cmmd.decouper(fausses_images(), graine=7), a)

    def test_proportions_et_strates(self):
        r = cmmd.resumer(cmmd.decouper(fausses_images()))
        self.assertEqual(r["entrainement"]["patientes"], 140)
        self.assertEqual(r["ajustement"]["patientes"], 30)
        self.assertEqual(r["validation"]["patientes"], 30)
        for s in cmmd.SOUS_ENSEMBLES:   # la part de malins suit celle de l'ensemble
            self.assertAlmostEqual(r[s]["seins_malins"] / r[s]["patientes"], 0.7, delta=0.05)

    def test_index_aller_retour(self):
        ims = cmmd.decouper(fausses_images(20))
        with tempfile.TemporaryDirectory() as d:
            cmmd.ecrire_index(ims, Path(d) / "index.csv")
            self.assertEqual(cmmd.lire_index(Path(d) / "index.csv"), ims)

    @unittest.skipUnless(A_OPENPYXL, "openpyxl absent")
    def test_donnees_cliniques(self):
        import openpyxl
        wb = openpyxl.Workbook()
        ws = wb.active
        ws.append(["ID1", "LeftRight", "Age", "number", "abnormality", "classification", "subtype"])
        ws.append(["D1-0001", "R", 44, 2, "calcification", "Benign", None])
        ws.append(["D2-0002", "L", 50, 2, "mass", "Malignant", "Luminal A"])
        ws.append(["D2-0002", "L", 50, 2, "both", "Benign", None])      # doublon : le malin l'emporte
        ws.append(["D2-0003", "X", 50, 2, "mass", "Malignant", None])   # côté inconnu : ignoré
        with tempfile.TemporaryDirectory() as d:
            wb.save(Path(d) / "c.xlsx")
            c = cmmd.lire_clinique(Path(d) / "c.xlsx")
        self.assertEqual(c, {("D1-0001", "R"): (0, "calcification"), ("D2-0002", "L"): (1, "mass")})


class MetriquesTest(unittest.TestCase):
    def test_auc(self):
        self.assertAlmostEqual(auc([0.1, 0.4, 0.35, 0.8], [0, 0, 1, 1]), 0.75)
        self.assertEqual(auc([0.2, 0.9], [0, 1]), 1.0)
        self.assertEqual(auc([0.5, 0.5], [0, 1]), 0.5)
        self.assertIsNone(auc([0.1, 0.2], [1, 1]))

    def test_seuil_et_sensibilite(self):
        s, e = [0.9, 0.8, 0.3, 0.2, 0.1], [1, 1, 1, 0, 0]
        seuil = seuil_pour_sensibilite(s, e, 0.85)
        self.assertEqual(seuil, 0.3)
        self.assertEqual(sens_spec(s, e, seuil), (1.0, 1.0))
        self.assertEqual(seuil_pour_sensibilite(s, e, 0.6), 0.8)

    def test_par_sein(self):
        seins, e, s = par_sein([("A/L", 1, 0.2), ("A/L", 1, 0.6), ("B/R", 0, 0.1)])
        self.assertEqual((seins, e), (["A/L", "B/R"], [1, 0]))
        self.assertAlmostEqual(s[0], 0.4)
        m = mesurer([("A/L", 1, 0.2), ("A/L", 1, 0.6), ("B/R", 0, 0.1)], seuil=0.3)
        self.assertEqual((m["seins"], m["auc_sein"], m["sensibilite_sein"]), (2, 1.0, 1.0))


@unittest.skipUnless(A_NUMPY, "numpy ou Pillow absent")
class PretraitementTest(unittest.TestCase):
    def test_oriente_recadre_ajuste(self):
        import numpy as np
        from mammomesh_engine.cnn.pretraitement import preparer
        img = np.zeros((600, 500), dtype=np.uint8)
        img[100:500, 300:500] = 180          # « sein » à droite
        out = preparer(img, 256, 224)
        self.assertEqual(out.shape, (256, 224))
        self.assertGreater(out[:, :20].mean(), 100, "le tissu doit être collé à gauche")
        self.assertLess(out[:, -20:].mean(), 10)

    def test_monochrome1_et_16_bits(self):
        import numpy as np
        from mammomesh_engine.cnn.pretraitement import en_8_bits
        px = np.linspace(0, 4095, 100 * 100).reshape(100, 100).astype(np.uint16)
        out = en_8_bits(px, monochrome1=True)
        self.assertEqual(out.dtype, np.uint8)
        self.assertGreater(out[0, 0], out[-1, -1])

    def test_refus_sans_place(self):
        from mammomesh_engine.cnn.pretraitement import PlaceInsuffisante, controler_place
        with tempfile.TemporaryDirectory() as d:
            ancien = os.environ.get("MARGE_GO")
            os.environ["MARGE_GO"] = "100000"
            try:
                with self.assertRaises(PlaceInsuffisante):
                    controler_place(d, 1, "essai")
            finally:
                if ancien is None:
                    os.environ.pop("MARGE_GO")
                else:
                    os.environ["MARGE_GO"] = ancien


if __name__ == "__main__":
    unittest.main()
