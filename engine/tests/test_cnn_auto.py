"""Veilleur de reprise : GPU libre ou occupé, choix de l'entraînement à reprendre, suivi repris (sans PyTorch)."""
import json
import tempfile
import unittest
from pathlib import Path

from mammomesh_engine.cnn import auto
from mammomesh_engine.cnn.suivi import Suivi, lire_etat


def faux_nvidia(memoire, applis=()):
    def lire(requete, *_):
        if requete.startswith("--query-gpu"):
            return [["0", str(memoire[0]), str(memoire[1])]]
        return [list(a) for a in applis]
    return lire


class GpuTest(unittest.TestCase):
    def test_libre(self):
        libre, raison = auto.etat_gpu(4500, lire=faux_nvidia((6144, 300)))
        self.assertTrue(libre, raison)

    def test_occupe_par_un_calcul(self):
        libre, raison = auto.etat_gpu(4500, lire=faux_nvidia((6144, 5000), [("GPU-x", "1234", "/usr/bin/python3", "4800")]))
        self.assertFalse(libre)
        self.assertIn("python3 (pid 1234", raison)

    def test_memoire_insuffisante(self):
        libre, raison = auto.etat_gpu(4500, lire=faux_nvidia((6144, 2000)))
        self.assertFalse(libre)
        self.assertIn("4144 Mio libres", raison)

    def test_nvidia_smi_muet(self):
        self.assertFalse(auto.etat_gpu(4500, lire=lambda *_: None)[0])


class ChoixTest(unittest.TestCase):
    def dossier(self, racine, nom, etat=None, parametres=True, rapport=False):
        d = Path(racine) / nom
        d.mkdir()
        if parametres:
            (d / "parametres.json").write_text("{}")
        if rapport:
            (d / "rapport.json").write_text("{}")
        if etat is not None:
            (d / "etat.json").write_text(json.dumps(etat))
        return d

    def test_a_reprendre(self):
        with tempfile.TemporaryDirectory() as r:
            self.dossier(r, "cnn-a-tue", {"statut": "en_cours", "pid": 111})
            self.dossier(r, "cnn-b-vivant", {"statut": "en_cours", "pid": 222})
            self.dossier(r, "cnn-c-fini", {"statut": "termine"}, rapport=True)
            self.dossier(r, "cnn-d-ctrlc", {"statut": "interrompu", "pid": 333})
            self.dossier(r, "cnn-e-ancien", {"statut": "en_cours"}, parametres=False)
            self.dossier(r, "cnn-f-echec", {"statut": "echec", "tentatives_auto": 1})
            self.dossier(r, "cnn-g-abandon", {"statut": "echec", "tentatives_auto": auto.TENTATIVES_MAX})
            self.dossier(r, "cnn-h-attente", {"statut": auto.EN_ATTENTE, "pid": 444})
            vus = [d.name for d in auto.a_reprendre(Path(r), vivant=lambda pid: pid == 222)]
        self.assertEqual(vus, ["cnn-h-attente", "cnn-f-echec", "cnn-a-tue"])

    def test_abandonner(self):
        with tempfile.TemporaryDirectory() as r:
            d = self.dossier(r, "cnn-x", {"statut": "en_cours", "epoque": 3})
            auto.abandonner(d)
            self.assertEqual(lire_etat(d)["statut"], "interrompu")
            self.assertEqual(lire_etat(d)["epoque"], 3)
            self.assertEqual(auto.a_reprendre(Path(r), vivant=lambda _: False), [])

    def test_processus_vivant(self):
        self.assertFalse(auto.processus_vivant(None))
        self.assertFalse(auto.processus_vivant(2 ** 22 + 12345))   # au-delà de pid_max : n'existe pas


class SuiviRepriseTest(unittest.TestCase):
    def test_garde_le_debut_et_compte_les_reprises(self):
        with tempfile.TemporaryDirectory() as r:
            s = Suivi(r, jeu="CMMD", commande="--epoques 15")
            s.phase("entrainement", epoque=4, epoques=15)
            s.etat["tentatives_auto"] = 1
            s.fin("echec", "OutOfMemoryError")
            debut = s.etat["debut"]
            s2 = Suivi(r, reprise=True)
            e = lire_etat(r)
            self.assertEqual((e["statut"], e["debut"], e["epoque"], e["commande"], e["reprises"], e["tentatives_auto"]),
                             ("en_cours", debut, 4, "--epoques 15", 1, 1))
            self.assertNotIn("message", e)
            Suivi(r, reprise=True)
            self.assertEqual(lire_etat(r)["reprises"], 2)
            del s2


if __name__ == "__main__":
    unittest.main()
