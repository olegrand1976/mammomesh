<script setup lang="ts">
import { zonesImages } from '~/utils/guideImages'

definePageMeta({ layout: 'suivi', middleware: 'auth' })
useHead({ title: 'Guide : Images · MammoMesh' })
</script>

<template>
  <DocPage titre="Guide : Images"
    intro="Importer une mammographie sans rien garder d'identifiant, la lire, et y tester une version du couple CNN + CephaloMesh choisie dans une campagne (génération finale par défaut)."
    :sommaire="zonesImages.map(z => ({ id: z.cle, titre: z.titre }))">
    <DocZones :zones="zonesImages" />
  </DocPage>
</template>
