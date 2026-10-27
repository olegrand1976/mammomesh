import unittest

from mammomesh_engine.materiel import Accelerateur, Exigences, planifier, selon_docker
from tests.machines import CPU_SEUL, IRIS_XE, P16V, PASCAL, PETIT_GPU, SERVEUR, machine, nvidia


class PlanTest(unittest.TestCase):
    def verifier_commun(self, plan, ex=Exigences()):
        self.assertGreaterEqual(plan.lot, 1)
        self.assertGreaterEqual(plan.lot * plan.accumulation, ex.lot_effectif)
        self.assertLess(plan.lot * (plan.accumulation - 1), ex.lot_effectif)
        self.assertIn(plan.inference_pendant_entrainement, plan.inference)
        self.assertEqual(plan.inference[-1], "torch-cpu")

    def test_portable_a1000(self):
        p = planifier(P16V)
        self.verifier_commun(p)
        self.assertEqual(p.appareil, "cuda:0")
        self.assertEqual(p.precision, "bf16")
        self.assertTrue(p.tf32 and p.channels_last and p.compile)
        self.assertGreaterEqual(p.lot, 4)
        self.assertEqual(p.entrainements_paralleles, 1)
        # Inférence sur l'Iris Xe pendant que la A1000 entraîne
        self.assertEqual(p.inference_pendant_entrainement, "openvino-GPU")
        self.assertEqual(p.profils_compose, ("nvidia", "intel"))
        self.assertTrue(any("BatchNorm" in a for a in p.avertissements))

    def test_docker_sans_le_gpu(self):
        p = planifier(P16V)
        self.assertIs(selon_docker(p, "obs,nvidia,intel"), p)
        self.assertIn("Docker ne verra pas NVIDIA RTX A1000", selon_docker(p, "cpu,intel").avertissements[-1])
        cpu = planifier(CPU_SEUL)
        self.assertIs(selon_docker(cpu, "cpu"), cpu)

    def test_serveur_multi_gpu(self):
        p = planifier(SERVEUR)
        self.verifier_commun(p)
        self.assertEqual(p.appareils, ("cuda:0", "cuda:1"))
        self.assertEqual(p.entrainements_paralleles, 8)
        self.assertEqual(p.entree, "image")
        self.assertFalse(p.checkpointing)
        self.assertEqual(p.inference[0], "onnxruntime-tensorrt")
        self.assertEqual(p.inference_pendant_entrainement, "onnxruntime-tensorrt")
        self.assertEqual(p.profils_compose, ("nvidia",))
        self.assertGreater(p.workers, 0)

    def test_cpu_seul(self):
        p = planifier(CPU_SEUL)
        self.verifier_commun(p)
        self.assertEqual(p.appareil, "cpu")
        self.assertEqual(p.precision, "fp32")
        self.assertFalse(p.pin_memory)
        self.assertEqual(p.profils_compose, ("cpu",))
        self.assertTrue(any("CPU" in a for a in p.avertissements))
        self.assertEqual(p.inference, ("torch-cpu",))

    def test_cpu_bf16(self):
        self.assertEqual(planifier(machine(cpu_bf16=True)).precision, "bf16")

    def test_pascal_sans_tensor_cores(self):
        p = planifier(PASCAL)
        self.assertEqual(p.precision, "fp32")
        self.assertFalse(p.tf32)

    def test_petit_gpu_passe_aux_patchs(self):
        p = planifier(PETIT_GPU)
        self.verifier_commun(p)
        self.assertTrue(p.entree == "patch" or p.checkpointing)
        self.assertEqual(p.precision, "fp16")

    def test_memoire_occupee_signalee(self):
        p = planifier(machine(nvidia("GPU", 8192, (8, 9), libre=4000)))
        self.assertTrue(any("déjà pris" in a for a in p.avertissements))

    def test_gpu_vu_sans_pytorch(self):
        p = planifier(machine(nvidia("RTX", 8192, (8, 9), source="nvidia-smi"), torch=None))
        self.assertEqual(p.profils_compose, ("nvidia",))
        self.assertTrue(any("PyTorch" in a for a in p.avertissements))

    def test_forcer_cpu(self):
        p = planifier(P16V, forcer="cpu")
        self.assertEqual(p.appareil, "cpu")

    def test_iris_xe_seul(self):
        p = planifier(machine(IRIS_XE, openvino=("CPU", "GPU")))
        self.assertEqual(p.appareil, "cpu")
        self.assertEqual(p.inference[0], "openvino-GPU")
        self.assertEqual(p.profils_compose, ("cpu", "intel"))

    def test_xpu_arc(self):
        arc = Accelerateur("xpu", 0, "Intel(R) Arc(TM) A770 Graphics", 16384, 16000, bf16=True)
        p = planifier(machine(arc, openvino=("CPU", "GPU")))
        self.assertEqual(p.appareil, "xpu:0")
        self.assertEqual(p.precision, "bf16")
        self.assertNotIn("openvino-GPU", p.inference_pendant_entrainement)

    def test_avec_lot(self):
        p = planifier(P16V).avec_lot(6)
        self.assertEqual((p.lot, p.accumulation), (6, 6))

    def test_exigences_plus_grandes(self):
        ex = Exigences(lot_effectif=64, resolution=(2048, 1664))
        p = planifier(P16V, ex)
        self.verifier_commun(p, ex)
        self.assertTrue(p.entree == "patch" or p.checkpointing)


if __name__ == "__main__":
    unittest.main()
