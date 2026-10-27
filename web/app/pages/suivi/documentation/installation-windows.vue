<script setup lang="ts">
import { etatWindows, partiesWindows, pdfWindows } from '~/utils/installationWindows'

definePageMeta({ layout: 'suivi', middleware: 'auth' })
useHead({ title: 'Installation sur Windows 11 · MammoMesh' })
</script>

<template>
  <DocPage titre="Installation sur Windows 11"
    :intro="`Faire tourner la pile MammoMesh sur un poste Windows 11 : WSL 2 (Debian), Docker Engine, GPU NVIDIA, démarrage automatique. ${etatWindows}.`"
    :sommaire="partiesWindows.map(p => ({ id: p.id, titre: p.titre }))">
    <p><a :href="pdfWindows" download class="btn btn-primary">Télécharger la procédure en PDF</a></p>
    <section v-for="p in partiesWindows" :id="p.id" :key="p.id" class="card">
      <h2>{{ p.titre }}</h2>
      <DocBlocs :blocs="p.blocs" />
    </section>
  </DocPage>
</template>
