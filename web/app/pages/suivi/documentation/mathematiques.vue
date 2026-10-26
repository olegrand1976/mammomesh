<script setup lang="ts">
import { partiesMath } from '~/utils/mathematique'

definePageMeta({ layout: 'suivi', middleware: 'auth' })
useHead({ title: 'Modèle mathématique · MammoMesh' })
</script>

<template>
  <DocPage titre="Modèle mathématique"
    intro="Les formules exactes de ce que calcule le socle (code de l'orchestrateur, de l'API et de l'interface), puis le modèle prévu par la note de conception pour la vraie boucle."
    :sommaire="partiesMath.map(p => ({ id: p.id, titre: p.titre }))">
    <section v-for="p in partiesMath" :id="p.id" :key="p.id" class="card">
      <h2>{{ p.titre }}</h2>
      <DocBlocs :blocs="p.blocs" />
    </section>
  </DocPage>
</template>
