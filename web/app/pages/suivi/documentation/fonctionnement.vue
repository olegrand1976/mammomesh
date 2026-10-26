<script setup lang="ts">
import { parties } from '~/utils/fonctionnement'

definePageMeta({ layout: 'suivi', middleware: 'auth' })
useHead({ title: 'Fonctionnement d\'une campagne · MammoMesh' })
const sommaire = [...parties.map(p => ({ id: p.id, titre: p.titre })), { id: 'mutations', titre: 'Lexique des mutations' }]
</script>

<template>
  <DocPage titre="Fonctionnement d'une campagne" :sommaire="sommaire">
    <section v-for="p in parties" :id="p.id" :key="p.id" class="card">
      <h2>{{ p.titre }}</h2>
      <DocBlocs :blocs="p.blocs" />
    </section>
    <section id="mutations" class="card">
      <h2>Lexique des mutations</h2>
      <p>Chaque individu porte la liste des mutations qui l'ont produit. Un code dit quel gène a changé et dans quel sens : ↑ relevé, ↓ abaissé, + ajouté, − retiré.</p>
      <LexiqueMutations ouvert />
    </section>
    <p class="small muted">Formules détaillées : <NuxtLink to="/suivi/documentation/mathematiques">Modèle mathématique</NuxtLink>.</p>
  </DocPage>
</template>
