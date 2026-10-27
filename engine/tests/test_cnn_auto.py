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


class DemandesTest(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.r = Path(self._tmp.name)
        (self.r / auto.DEMANDES).mkdir()
        self.v = auto.Veilleur(self.r, journal=lambda m: None)

    def tearDown(self):
        self._tmp.cleanup()

    def demander(self, nom, action, k=1):
        (self.r / auto.DEMANDES / f"{k}-{action}.json").write_text(json.dumps({"nom": nom, "action": action, "par": "marie"}))

    def entrainement(self, nom, etat, parametres=None, rapport=False):
        d = self.r / nom
        d.mkdir()
        (d / "etat.json").write_text(json.dumps(etat))
        if parametres is not None:
            (d / "parametres.json").write_text(json.dumps(parametres))
        if rapport:
            (d / "rapport.json").write_text("{}")
        return d

    def test_supprimer(self):
        self.entrainement("cnn-cmmd-1", {"statut": "termine"}, rapport=True)
        (self.r / "cnn-cmmd-1.log").write_text("x")
        (self.r / "entrainement-cmmd-1.log").write_text("x")
        self.demander("cnn-cmmd-1", "supprimer")
        self.v.demandes()
        self.assertEqual(sorted(p.name for p in self.r.iterdir()), [auto.DEMANDES])
        self.assertEqual(list((self.r / auto.DEMANDES).iterdir()), [])

    def test_reprendre_apres_ctrl_c(self):
        d = self.entrainement("cnn-cmmd-2", {"statut": "interrompu", "epoque": 4, "tentatives_auto": 3}, parametres={})
        self.demander("cnn-cmmd-2", "relancer")
        self.v.demandes()
        e = lire_etat(d)
        self.assertEqual((e["statut"], e["tentatives_auto"]), (auto.EN_ATTENTE, 0))
        self.assertEqual([x.name for x in auto.a_reprendre(self.r, vivant=lambda _: False)], ["cnn-cmmd-2"])

    def test_relancer_un_entrainement_fini_cree_un_nouveau(self):
        self.entrainement("cnn-cmmd-3", {"statut": "termine"}, rapport=True,
                          parametres={"banque": "/b", "cache": "/c", "resolution": [768, 624], "epoques": 7, "lot_effectif": 32,
                                      "patience": 3, "lr": 0.001, "graine": 1, "sensibilite_cible": 0.9})
        self.demander("cnn-cmmd-3", "relancer")
        self.v.demandes()
        nouveaux = [p for p in self.r.iterdir() if p.name.startswith("cnn-cmmd-2")]
        self.assertEqual(len(nouveaux), 1)
        p = json.loads((nouveaux[0] / "parametres.json").read_text())
        self.assertEqual((p["epoques"], p["resolution"], p["patience"], p["banque"]), (7, [768, 624], 3, "/b"))
        e = lire_etat(nouveaux[0])
        self.assertEqual((e["statut"], e["file_attente"], e["source"]), (auto.EN_ATTENTE, True, "cnn-cmmd-3"))
        # Premier départ : ce n'est pas une reprise.
        s = Suivi(nouveaux[0], reprise=True)
        self.assertNotIn("reprises", s.etat)

    def test_relancer_un_ancien_sans_reglages(self):
        self.entrainement("cnn-cmmd-4", {"statut": "en_cours", "pid": 999999999, "commande": "--epoques 9"})
        self.demander("cnn-cmmd-4", "relancer")
        self.v.demandes()
        nouveau = next(p for p in self.r.iterdir() if p.name.startswith("cnn-cmmd-2"))
        self.assertEqual(json.loads((nouveau / "parametres.json").read_text())["epoques"], 9)
        # Deux relances dans la même minute : deux dossiers distincts.
        self.assertNotEqual(auto.nouvel_entrainement(self.r, self.r / "cnn-cmmd-4", "x"), nouveau)

    def test_demandes_invalides(self):
        self.demander("../etc", "supprimer")
        self.demander("absent", "relancer", 2)
        (self.r / auto.DEMANDES / "3-x.json").write_text("{pas du json")
        self.v.demandes()
        self.assertEqual(list((self.r / auto.DEMANDES).iterdir()), [])

    def test_depart_apres_stabilite(self):
        self.entrainement("cnn-cmmd-5", {"statut": "echec", "tentatives_auto": 0}, parametres={})
        lances = []
        self.v.lancer = lances.append
        ancien = auto.etat_gpu
        auto.etat_gpu = lambda *a, **k: (True, "libre")
        try:
            self.v.verifier()
            self.assertEqual(lances, [])
            self.assertEqual(lire_etat(self.r / "cnn-cmmd-5")["statut"], auto.EN_ATTENTE)
            self.v.libre_depuis -= auto.STABILITE_S
            self.v.verifier()
            self.assertEqual([d.name for d in lances], ["cnn-cmmd-5"])
        finally:
            auto.etat_gpu = ancien


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
