import json
import os
import tempfile
import unittest
from unittest import mock

from mammomesh_engine.materiel import Materiel, detecter
from mammomesh_engine.materiel.detection import lire_nvidia_smi
from tests.machines import P16V


class DetectionTest(unittest.TestCase):
    def test_nvidia_smi(self):
        accs = lire_nvidia_smi("0, NVIDIA RTX A1000 6GB Laptop GPU, 6144, 5900, 8.6\n1, Tesla P100, 16384, [N/A], 6.0\n")
        self.assertEqual(len(accs), 1)
        a = accs[0]
        self.assertEqual((a.nom, a.memoire_mio, a.capacite, a.bf16, a.source),
                         ("NVIDIA RTX A1000 6GB Laptop GPU", 6144, (8, 6), True, "nvidia-smi"))

    def test_aller_retour_json(self):
        self.assertEqual(Materiel.from_dict(json.loads(json.dumps(P16V.to_dict()))), P16V)

    def test_machine_simulee(self):
        with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as f:
            json.dump(P16V.to_dict(), f)
        try:
            with mock.patch.dict(os.environ, {"MAMMOMESH_MATERIEL": f.name}):
                self.assertEqual(detecter(), P16V)
        finally:
            os.unlink(f.name)

    def test_detection_reelle_sans_erreur(self):
        with mock.patch.dict(os.environ, {}, clear=False):
            os.environ.pop("MAMMOMESH_MATERIEL", None)
            m = detecter()
        self.assertGreaterEqual(m.coeurs, 1)
        self.assertGreater(m.ram_mio, 0)


if __name__ == "__main__":
    unittest.main()
