<script setup lang="ts">
import { zones } from '~/utils/guideSuivi'

definePageMeta({ layout: 'suivi', middleware: 'auth' })
useHead({ title: 'Guide : suivi d\'une campagne · MammoMesh' })
</script>

<template>
  <DocPage titre="Guide : suivi d'une campagne"
    intro="L'écran d'une campagne se met à jour en direct. Chaque zone a un bouton « ? » qui affiche la même aide dans l'écran."
    :sommaire="zones.map(z => ({ id: z.cle, titre: z.titre }))">
    <DocZones :zones="zones" />
  </DocPage>
</template>
