<script setup lang="ts">
// Fonctionnement complet d'une campagne, et lexique des mutations.
import { parties } from '~/utils/fonctionnement'

definePageMeta({ layout: 'suivi', middleware: 'auth' })
useHead({ title: 'Fonctionnement d\'une campagne · MammoMesh' })
</script>

<template>
  <article class="doc">
    <h1>Fonctionnement d'une campagne</h1>
    <nav class="card sommaire" aria-label="Sommaire">
      <ol>
        <li v-for="p in parties" :key="p.id"><a :href="`#${p.id}`">{{ p.titre }}</a></li>
        <li><a href="#mutations">Lexique des mutations</a></li>
      </ol>
    </nav>
    <section v-for="p in parties" :id="p.id" :key="p.id" class="card">
      <h2>{{ p.titre }}</h2>
      <template v-for="(b, i) in p.blocs" :key="i">
        <p v-if="b.p">{{ b.p }}</p>
        <ol v-if="b.etapes"><li v-for="e in b.etapes" :key="e">{{ e }}</li></ol>
        <ul v-if="b.points"><li v-for="e in b.points" :key="e">{{ e }}</li></ul>
        <div v-if="b.table" class="table-wrap"><table>
          <thead><tr><th v-for="h in b.table.entetes" :key="h">{{ h }}</th></tr></thead>
          <tbody><tr v-for="(l, j) in b.table.lignes" :key="j"><td v-for="(c, k) in l" :key="k" :class="{ cle: k === 0 }">{{ c }}</td></tr></tbody>
        </table></div>
      </template>
    </section>
    <section id="mutations" class="card">
      <h2>Lexique des mutations</h2>
      <p>Chaque individu porte la liste des mutations qui l'ont produit. Un code dit quel gène a changé et dans quel sens : ↑ relevé, ↓ abaissé, + ajouté, − retiré.</p>
      <LexiqueMutations ouvert />
    </section>
    <p class="small muted">Voir aussi les guides d'écran : <NuxtLink to="/suivi/nouvelle">Nouvelle campagne</NuxtLink> et le suivi d'une campagne (bouton « ? » de chaque zone).</p>
  </article>
</template>

<style scoped>
.doc { max-width: 860px; display: grid; gap: 16px; }
.doc h1 { font-size: 1.4rem; margin: 0; }
.sommaire ol { margin: 0; padding-left: 20px; columns: 2 220px; }
ol, ul { padding-left: 20px; display: grid; gap: 6px; margin: 0 0 1em; }
td { vertical-align: top; }
td.cle { font-weight: 600; white-space: nowrap; }
section { scroll-margin-top: 16px; }
</style>
