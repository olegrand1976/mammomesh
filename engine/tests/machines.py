"""Machines types pour les tests (aucune dépendance à PyTorch)."""
from mammomesh_engine.materiel import Accelerateur, Materiel

IRIS_XE = Accelerateur("openvino", 0, "Intel(R) Iris(R) Xe Graphics", 0, 0, integre=True, ident="GPU", source="openvino")


def machine(*accs, coeurs=20, ram=47000, libre=31000, torch="2.9.0", openvino=(), ort=("CPUExecutionProvider",), cpu_bf16=False):
    return Materiel("linux", "CPU de test", coeurs, cpu_bf16, ram, libre, tuple(accs), tuple(openvino), tuple(ort), torch)


def nvidia(nom, mio, cap, index=0, libre=None, source="torch"):
    return Accelerateur("cuda", index, nom, mio, mio if libre is None else libre, cap,
                        bf16=cap >= (8, 0), fp16=cap >= (7, 0), source=source)


# Le portable de développement : ThinkPad P16v Gen 1
P16V = machine(nvidia("NVIDIA RTX A1000 6GB Laptop GPU", 6144, (8, 6), libre=5900), IRIS_XE,
               openvino=("CPU", "GPU"), ort=("CUDAExecutionProvider", "CPUExecutionProvider"))
SERVEUR = machine(nvidia("NVIDIA A100 80GB", 81920, (8, 0), 0), nvidia("NVIDIA A100 80GB", 81920, (8, 0), 1),
                  coeurs=64, ram=512000, libre=480000,
                  ort=("TensorrtExecutionProvider", "CUDAExecutionProvider", "CPUExecutionProvider"))
CPU_SEUL = machine(coeurs=8, ram=16000, libre=10000, torch=None, ort=())
PASCAL = machine(nvidia("NVIDIA GeForce GTX 1060 6GB", 6144, (6, 1)))
PETIT_GPU = machine(nvidia("NVIDIA GeForce MX450", 2048, (7, 5)))
