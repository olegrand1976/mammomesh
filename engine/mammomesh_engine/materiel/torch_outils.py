"""Application d'un plan à PyTorch (importé seulement ici)."""
from __future__ import annotations

from contextlib import nullcontext

from .plan import Plan


def preparer(plan: Plan):
    """Réglages globaux de PyTorch ; renvoie le torch.device d'entraînement."""
    import torch
    if plan.tf32:
        torch.backends.cuda.matmul.allow_tf32 = True
        torch.backends.cudnn.allow_tf32 = True
    if plan.appareil.startswith("cuda"):
        torch.backends.cudnn.benchmark = True
    torch.set_num_threads(plan.threads_cpu)
    return torch.device(plan.appareil)


def _type(plan: Plan) -> str:
    return plan.appareil.split(":")[0]


def autocast(plan: Plan):
    if plan.precision == "fp32":
        return nullcontext()
    import torch
    dtype = torch.bfloat16 if plan.precision == "bf16" else torch.float16
    return torch.autocast(device_type=_type(plan), dtype=dtype)


def scaler(plan: Plan):
    """GradScaler, actif seulement en FP16 (inutile en BF16 et FP32)."""
    import torch
    return torch.amp.GradScaler(_type(plan), enabled=plan.precision == "fp16")


def preparer_modele(plan: Plan, modele):
    import torch
    modele = modele.to(plan.appareil)
    if plan.channels_last:
        modele = modele.to(memory_format=torch.channels_last)
    if plan.compile:
        modele = torch.compile(modele)
    return modele


def options_dataloader(plan: Plan) -> dict:
    opts = {"num_workers": plan.workers, "pin_memory": plan.pin_memory}
    if plan.workers > 0:
        opts.update(persistent_workers=True, prefetch_factor=plan.prefetch)
    return opts
