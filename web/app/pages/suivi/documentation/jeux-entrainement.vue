<script setup lang="ts">
import { etatJeux, partiesJeux, sourcesJeux } from '~/utils/jeuxEntrainement'

definePageMeta({ layout: 'suivi', middleware: 'auth' })
useHead({ title: 'Jeux d\'entraînement · MammoMesh' })
const sommaire = [...partiesJeux.map(p => ({ id: p.id, titre: p.titre })), { id: 'sources', titre: 'Sources' }]
</script>

<template>
  <DocPage titre="Jeux d'entraînement"
    :intro="`Les mammographies publiques retenues, à demander ou écartées pour la banque d'entraînement du CNN et de CephaloMesh, avec leurs licences et leurs sources. ${etatJeux}.`"
    :sommaire="sommaire">
    <section v-for="p in partiesJeux" :id="p.id" :key="p.id" class="card">
      <h2>{{ p.titre }}</h2>
      <DocBlocs :blocs="p.blocs" />
    </section>
    <section id="sources" class="card">
      <h2>Sources</h2>
      <ul>
        <li v-for="s in sourcesJeux" :key="s.url">
          <a :href="s.url" target="_blank" rel="noopener noreferrer">{{ s.titre }}</a>
        </li>
      </ul>
    </section>
  </DocPage>
</template>

<style scoped>
ul { padding-left: 20px; display: grid; gap: 6px; margin: 0; }
</style>
