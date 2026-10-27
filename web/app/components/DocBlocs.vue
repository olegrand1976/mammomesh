<script setup lang="ts">
// Affiche les blocs d'une partie de documentation : paragraphes, étapes, points, tableaux, formules.
import type { BlocMath } from '~/utils/mathematique'
defineProps<{ blocs: BlocMath[] }>()
</script>

<template>
  <template v-for="(b, i) in blocs" :key="i">
    <p v-if="b.p"><TexteGlossaire :texte="b.p" /></p>
    <ol v-if="b.etapes"><li v-for="e in b.etapes" :key="e"><TexteGlossaire :texte="e" /></li></ol>
    <ul v-if="b.points"><li v-for="e in b.points" :key="e"><TexteGlossaire :texte="e" /></li></ul>
    <div v-if="b.table" class="table-wrap"><table>
      <thead><tr><th v-for="h in b.table.entetes" :key="h">{{ h }}</th></tr></thead>
      <tbody><tr v-for="(l, j) in b.table.lignes" :key="j"><td v-for="(c, k) in l" :key="k" :class="{ cle: k === 0 }"><TexteGlossaire :texte="c" /></td></tr></tbody>
    </table></div>
    <FormuleMath v-if="b.formule" :tex="b.formule" />
    <pre v-if="b.code" class="code"><code>{{ b.code }}</code></pre>
  </template>
</template>

<style scoped>
ol, ul { padding-left: 20px; display: grid; gap: 6px; margin: 0 0 1em; }
td { vertical-align: top; }
td.cle { font-weight: 600; white-space: nowrap; }
.code { background: var(--surface-2); border: 1px solid var(--border); border-radius: 8px; padding: 10px 12px; overflow-x: auto;
  font-size: 0.82rem; line-height: 1.45; margin: 0 0 1em; }
</style>
