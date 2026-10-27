<script setup lang="ts">
import { partiesCouple } from '~/utils/couple'

definePageMeta({ layout: 'suivi', middleware: 'auth' })
useHead({ title: 'CNN et CephaloMesh · MammoMesh' })
const sommaire = partiesCouple.map(p => ({ id: p.id, titre: p.titre }))
</script>

<template>
  <DocPage titre="CNN et CephaloMesh : voir, puis raisonner"
    intro="Ce que fait chaque partie de MammoMesh, ce qui lui manque, et pourquoi les combiner. Page écrite pour être lue sans connaissance préalable en intelligence artificielle ni en radiologie."
    :sommaire="sommaire">
    <section v-for="p in partiesCouple" :id="p.id" :key="p.id" class="card">
      <h2>{{ p.titre }}</h2>
      <DocBlocs :blocs="p.blocs" />
    </section>
    <p class="small muted">
      Pour aller plus loin : <NuxtLink to="/suivi/documentation/mutations">Lexique des mutations</NuxtLink>,
      <NuxtLink to="/suivi/documentation/fonctionnement">Fonctionnement d'une campagne</NuxtLink>,
      <NuxtLink to="/suivi/documentation/glossaire">Glossaire</NuxtLink>.
    </p>
  </DocPage>
</template>
