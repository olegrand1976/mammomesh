"""Plan d'exécution déduit du matériel.

Règles, dans l'ordre :
1. Entraînement sur le meilleur accélérateur dédié (le plus de mémoire libre), sinon un GPU
   intégré, sinon le CPU. Plusieurs GPU identiques : un individu de la population par GPU,
   plusieurs par GPU au-delà de 16 Gio.
2. Précision : BF16 si le matériel l'accélère, sinon FP16 (avec GradScaler), sinon FP32.
3. Entrée : l'image entière si un lot d'au moins 4 tient en mémoire, sinon avec gradient
   checkpointing, sinon des patchs de plus en plus petits. Le lot effectif est atteint par
   accumulation de gradient. La mémoire par image est une estimation : calibrer() la mesure.
4. Chargement : un cœur par entraînement, le reste en workers, bornés par la RAM libre.
5. Inférence : moteur le plus rapide disponible ; pendant un entraînement sur un petit GPU,
   l'inférence passe ailleurs (GPU Intel via OpenVINO, sinon CPU) pour ne pas lui prendre sa mémoire.
"""
from __future__ import annotations

import json
import math
from dataclasses import asdict, dataclass, replace

from .detection import Accelerateur, Materiel

# Mémoire d'activations par image 224 × 224 pour un CNN léger (EfficientNet-B0, ResNet-34)
# en entraînement 16 bits. Ordre de grandeur : calibrer() le remplace par une mesure.
COUT_224_MIO = 50.0
LOT_MIN = 4                 # en dessous, on passe au checkpointing puis aux patchs
LOT_MIN_BATCHNORM = 8       # en dessous, BatchNorm devient instable
GRAND_GPU_MIO = 16 * 1024   # au-delà : plusieurs individus par GPU, inférence sur le même GPU
MAX_PAR_GPU = 4
RAM_PAR_WORKER_MIO = 1536   # décodage DICOM, augmentations, file de préchargement


@dataclass(frozen=True)
class Exigences:
    """Ce que demande la tâche, indépendamment de la machine."""
    lot_effectif: int = 32
    resolution: tuple[int, int] = (1024, 832)
    patch: int = 512
    patch_min: int = 256
    cout_224_mio: float = COUT_224_MIO
    surcout_mio: int = 1024     # poids, états de l'optimiseur, contexte du pilote


@dataclass(frozen=True)
class Plan:
    appareil: str                       # cuda:0, xpu:0, mps, cpu
    appareils: tuple[str, ...]          # tous ceux qui entraînent en parallèle
    nom_appareil: str
    profils_compose: tuple[str, ...]    # nvidia, amd, intel, apple, cpu
    precision: str                      # bf16, fp16, fp32
    tf32: bool
    channels_last: bool
    compile: bool
    entree: str                         # image ou patch
    taille_entree: tuple[int, int]
    lot: int
    accumulation: int
    lot_effectif: int
    checkpointing: bool
    entrainements_paralleles: int
    workers: int
    threads_cpu: int
    pin_memory: bool
    prefetch: int
    inference: tuple[str, ...]          # moteurs par ordre de préférence
    inference_pendant_entrainement: str
    avertissements: tuple[str, ...] = ()

    def avec_lot(self, lot: int) -> "Plan":
        lot = max(1, min(lot, self.lot_effectif))
        return replace(self, lot=lot, accumulation=math.ceil(self.lot_effectif / lot))

    def to_dict(self) -> dict:
        return asdict(self)

    def to_json(self) -> str:
        return json.dumps(self.to_dict(), ensure_ascii=False, indent=2)

    def resume(self) -> str:
        h, w = self.taille_entree
        lignes = [
            f"Entraînement : {self.nom_appareil} ({self.appareil}), {self.entrainements_paralleles} en parallèle",
            f"Précision    : {self.precision}" + (" + TF32" if self.tf32 else "")
            + (", channels_last" if self.channels_last else "") + (", torch.compile" if self.compile else ""),
            f"Entrée       : {self.entree} {h}×{w}" + (", gradient checkpointing" if self.checkpointing else ""),
            f"Lot          : {self.lot} × {self.accumulation} accumulations = {self.lot * self.accumulation}",
            f"Données      : {self.workers} workers, {self.threads_cpu} threads CPU, pin_memory={self.pin_memory}",
            f"Inférence    : {', '.join(self.inference)}",
            f"  pendant un entraînement : {self.inference_pendant_entrainement}",
        ]
        lignes += [f"! {a}" for a in self.avertissements]
        return "\n".join(lignes)


def selon_docker(plan: Plan, profils: str) -> Plan:
    """Signale un plan que les conteneurs ne pourront pas suivre (profils retenus par detecter-materiel.sh)."""
    retenus = [x.strip() for x in profils.split(",") if x.strip()]
    if plan.appareil == "cpu" or plan.profils_compose[0] in retenus:
        return plan
    return replace(plan, avertissements=plan.avertissements + (
        f"Docker ne verra pas {plan.nom_appareil} (profils {','.join(retenus) or 'aucun'}) : "
        "ce plan vaut hors conteneur, dans Docker l'entraînement se fera sans ce GPU",))


def _puissance2(n: int) -> int:
    return 1 << (n.bit_length() - 1) if n >= 1 else 0


def _par_gpu(a: Accelerateur) -> int:
    return max(1, min(MAX_PAR_GPU, a.memoire_libre_mio // GRAND_GPU_MIO))


def planifier(m: Materiel, ex: Exigences = Exigences(), *, forcer: str | None = None) -> Plan:
    avert: list[str] = []
    candidats = sorted((a for a in m.accelerateurs if a.entrainement),
                       key=lambda a: (a.integre, -a.memoire_libre_mio))
    if forcer:
        candidats = [a for a in candidats if a.appareil == forcer]
        if forcer != "cpu" and not candidats:
            avert.append(f"Appareil demandé {forcer} introuvable : plan calculé pour le CPU")
    p = candidats[0] if candidats else None

    # 1. Appareils
    if p:
        groupe = [a for a in candidats if a.type == p.type and a.integre == p.integre
                  and a.memoire_libre_mio >= p.memoire_libre_mio // 2] if not forcer else [p]
        paralleles = sum(_par_gpu(a) for a in groupe)
        budget = p.memoire_libre_mio / _par_gpu(p) * 0.85 - ex.surcout_mio
        if p.memoire_libre_mio < 0.8 * p.memoire_mio:
            avert.append(f"{p.memoire_mio - p.memoire_libre_mio} Mio déjà pris sur {p.nom} par d'autres programmes"
                         " (affichage, autre calcul)")
        if p.source != "torch":
            avert.append("PyTorch absent : installer la version qui correspond au GPU" if m.torch is None else
                         f"PyTorch {m.torch} ne voit pas {p.nom} : installer sa version CUDA/ROCm/XPU")
        if p.integre and p.type != "mps":
            avert.append(f"Entraînement sur un GPU intégré ({p.nom}) : lent, à réserver aux essais")
    else:
        groupe, paralleles = [], 1
        budget = m.ram_libre_mio * 0.5 - ex.surcout_mio
        avert.append("Aucun GPU utilisable : entraînement sur CPU, très lent, à réserver aux essais")

    # 2. Précision
    if p:
        precision = "bf16" if p.bf16 else "fp16" if p.fp16 else "fp32"
    else:
        precision = "bf16" if m.cpu_bf16 else "fp32"
    facteur = 2.0 if precision == "fp32" else 1.0

    # 3. Entrée et lot
    H, W = ex.resolution
    tailles = sorted({ex.patch, max(ex.patch_min, ex.patch * 3 // 4 // 32 * 32), ex.patch_min}, reverse=True)
    options = [("image", (H, W), False), ("image", (H, W), True)]
    options += [("patch", (t, t), c) for t in tailles for c in (False, True)]

    def lot_max(taille, ckpt):
        cout = ex.cout_224_mio * taille[0] * taille[1] / (224 * 224) * facteur * (0.4 if ckpt else 1.0)
        return int(max(0.0, budget) // cout)

    choix = next((o for o in options if lot_max(o[1], o[2]) >= LOT_MIN), None)
    if choix is None:
        choix = next((o for o in options if lot_max(o[1], o[2]) >= 1), options[-1])
        avert.append("Mémoire très juste : lot inférieur à 4 même en patchs avec checkpointing")
    entree, taille, ckpt = choix
    lot = max(1, min(_puissance2(lot_max(taille, ckpt)), ex.lot_effectif))
    if lot < LOT_MIN_BATCHNORM:
        avert.append(f"Lot de {lot} : préférer GroupNorm ou figer les statistiques BatchNorm")

    # 4. Chargement des données
    if p:
        dispo = max(0, m.coeurs - 1 - paralleles)
        workers = dispo // paralleles
        threads = max(1, min(4, m.coeurs // paralleles))
    else:
        workers = min(2, max(0, m.coeurs // 4))
        threads = max(1, m.coeurs - workers)
    workers = max(0, min(workers, 12, int(m.ram_libre_mio * 0.5 // (RAM_PAR_WORKER_MIO * paralleles))))
    accel = bool(p) and p.type in ("cuda", "rocm", "xpu")

    # 5. Inférence
    ort = set(m.onnxruntime)
    moteurs: list[str] = []
    if p and p.type == "cuda":
        moteurs += [x for f, x in (("TensorrtExecutionProvider", "onnxruntime-tensorrt"),
                                   ("CUDAExecutionProvider", "onnxruntime-cuda")) if f in ort]
        moteurs.append("torch-cuda")
    elif p and p.type == "rocm":
        moteurs += [x for f, x in (("MIGraphXExecutionProvider", "onnxruntime-migraphx"),
                                   ("ROCMExecutionProvider", "onnxruntime-rocm")) if f in ort]
        moteurs.append("torch-rocm")
    elif p and p.type in ("xpu", "mps"):
        moteurs.append(f"torch-{p.type}")
    ov_gpu = [a for a in m.accelerateurs if a.type == "openvino"]
    moteurs += [f"openvino-{a.ident}" for a in ov_gpu]
    if "CPU" in m.openvino:
        moteurs.append("openvino-CPU")
    if "CPUExecutionProvider" in ort:
        moteurs.append("onnxruntime-cpu")
    moteurs.append("torch-cpu")
    moteurs = list(dict.fromkeys(moteurs))

    if p is None or p.memoire_mio >= GRAND_GPU_MIO:
        pendant = moteurs[0]
    else:
        prefixe = {"cuda": "-cuda", "rocm": "-rocm", "xpu": "torch-xpu", "mps": "torch-mps"}[p.type]
        hors = [x for x in moteurs if prefixe not in x and x not in ("onnxruntime-tensorrt", "onnxruntime-migraphx")]
        if p.type == "xpu":   # OpenVINO GPU = le même GPU Intel
            hors = [x for x in hors if not x.startswith("openvino-GPU")] or hors
        pendant = hors[0]

    familles = [p.famille] if p else ["cpu"]
    if pendant.startswith("openvino-GPU"):
        familles.append("intel")

    return Plan(
        appareil=p.appareil if p else "cpu",
        appareils=tuple(a.appareil for a in groupe) or ("cpu",),
        nom_appareil=p.nom if p else m.cpu,
        profils_compose=tuple(dict.fromkeys(familles)),
        precision=precision,
        tf32=bool(p) and p.type == "cuda" and p.capacite is not None and p.capacite >= (8, 0),
        channels_last=(p is None) or p.type == "xpu" or (p.type == "cuda" and (p.capacite or (0, 0)) >= (7, 0)),
        compile=bool(p) and p.type in ("cuda", "rocm") and m.systeme == "linux"
                and (p.capacite is None or p.capacite >= (7, 0)),
        entree=entree,
        taille_entree=taille,
        lot=lot,
        accumulation=math.ceil(ex.lot_effectif / lot),
        lot_effectif=ex.lot_effectif,
        checkpointing=ckpt,
        entrainements_paralleles=paralleles,
        workers=workers,
        threads_cpu=threads,
        pin_memory=accel,
        prefetch=4 if m.ram_libre_mio > 16384 else 2,
        inference=tuple(moteurs),
        inference_pendant_entrainement=pendant,
        avertissements=tuple(avert),
    )
