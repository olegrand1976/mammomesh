import json
import os
import tempfile
import unittest

from mammomesh_engine.materiel import calibrer, chercher_lot, est_manque_memoire, planifier
from tests.machines import P16V


class OutOfMemoryError(RuntimeError):
    pass


def pas_jusqua(limite, appels=None):
    def pas(n):
        if appels is not None:
            appels.append(n)
        if n > limite:
            raise OutOfMemoryError("CUDA out of memory. Tried to allocate 2.00 GiB")
    return pas


class CalibrageTest(unittest.TestCase):
    def test_reconnaissance(self):
        self.assertTrue(est_manque_memoire(OutOfMemoryError("x")))
        self.assertTrue(est_manque_memoire(RuntimeError("DefaultCPUAllocator: not enough memory")))
        self.assertTrue(est_manque_memoire(RuntimeError("MPS backend out of memory")))
        self.assertFalse(est_manque_memoire(ValueError("shape mismatch")))

    def test_recherche(self):
        for limite in (1, 3, 13, 31, 32, 200):
            for depart in (1, 4, 64):
                with self.subTest(limite=limite, depart=depart):
                    self.assertEqual(chercher_lot(pas_jusqua(limite), depart, 32), min(limite, 32))

    def test_rien_ne_tient(self):
        self.assertEqual(chercher_lot(pas_jusqua(0), 8, 32), 0)

    def test_autre_erreur_propagee(self):
        def pas(n):
            raise ValueError("bug du modèle")
        with self.assertRaises(ValueError):
            chercher_lot(pas, 4, 32)

    def test_cache(self):
        plan = planifier(P16V)
        with tempfile.TemporaryDirectory() as d:
            chemin = os.path.join(d, "calibrage.json")
            appels = []
            p1 = calibrer(plan, "effnet_b0", pas_jusqua(11, appels), cache=chemin)
            self.assertEqual(p1.lot, 9)          # 11 mesuré, marge de 10 %
            self.assertEqual(p1.accumulation, 4)
            self.assertTrue(appels)
            appels.clear()
            p2 = calibrer(plan, "effnet_b0", pas_jusqua(11, appels), cache=chemin)
            self.assertEqual((p2.lot, appels), (9, []))
            with open(chemin, encoding="utf-8") as f:
                self.assertEqual(len(json.load(f)), 1)

    def test_lot_impossible(self):
        with tempfile.TemporaryDirectory() as d, self.assertRaises(MemoryError):
            calibrer(planifier(P16V), "geant", pas_jusqua(0), cache=os.path.join(d, "c.json"))


if __name__ == "__main__":
    unittest.main()
