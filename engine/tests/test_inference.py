"""Inférence : géométrie (toile → image d'origine) et zones d'attention, sans PyTorch."""
import unittest

try:
    import numpy as np
    from PIL import Image  # noqa: F401  (ajuster redimensionne avec Pillow)
except ImportError:  # pragma: no cover
    np = None


@unittest.skipIf(np is None, "numpy et Pillow requis")
class TestInference(unittest.TestCase):
    def image(self, droite: bool):
        img = np.zeros((400, 300), dtype=np.uint16)
        if droite:
            img[50:350, 200:300] = 3000   # sein à droite
            img[150:170, 250:270] = 4000
        else:
            img[50:350, 0:100] = 3000
            img[150:170, 30:50] = 4000
        return img

    def test_geometrie_aller_retour(self):
        from mammomesh_engine.inference.modele import preparer
        for droite in (False, True):
            toile, geo = preparer(self.image(droite), 128, 104, False)
            self.assertEqual(toile.shape, (128, 104))
            self.assertEqual(geo.retourne, droite)
            # Toute la zone de tissu de la toile revient sur la bande de tissu d'origine.
            x, y, l, h = geo.vers_origine(0, geo.haut, geo.nw, geo.nh)
            self.assertAlmostEqual(x, 200 if droite else 0, delta=2)
            self.assertAlmostEqual(y, 50, delta=2)
            self.assertAlmostEqual(l, 100, delta=3)
            self.assertAlmostEqual(h, 300, delta=3)

    def test_zones(self):
        from mammomesh_engine.inference.modele import preparer, zones
        _, geo = preparer(self.image(False), 128, 104, False)
        carte = np.zeros((4, 4))
        carte[1, 0] = 1.0     # dans le sein
        carte[3, 3] = 5.0     # hors du sein (marge noire) : ignoré
        z = zones(carte, geo, 128, 104)
        self.assertEqual(len(z), 1)
        self.assertEqual(z[0]["score"], 1.0)
        self.assertLess(z[0]["x"], 100)

    def test_huit_bits(self):
        from mammomesh_engine.inference.modele import vers_type_source
        px = np.array([[0, 200]], dtype=np.uint16)
        self.assertEqual(vers_type_source(px, 8).dtype, np.uint8)
        self.assertEqual(vers_type_source(px, 12).dtype, np.uint16)


if __name__ == "__main__":
    unittest.main()
