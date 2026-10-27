<script setup lang="ts">
// Encadré « Lecture » sous le graphique d'un entraînement : ses gains et ce que la courbe laisse prévoir, en mots simples.
import type { EpoqueEntrainement } from '~/utils/entrainements'
import { lireProgression } from '~/utils/progression'
const props = defineProps<{ journal: EpoqueEntrainement[]; patience?: number }>()
const lu = computed(() => lireProgression(props.journal, props.patience || 5))
</script>

<template>
  <div v-if="lu" class="lecture" :class="lu.verdict">
    <strong>Lecture : {{ lu.titre }}</strong>
    <p v-for="ph in lu.phrases" :key="ph" class="small"><TexteGlossaire :texte="ph" /></p>
  </div>
</template>

<style scoped>
.lecture { border-left: 4px solid var(--accent); background: var(--accent-soft); border-radius: 8px; padding: 10px 12px; margin-top: 10px; }
.lecture.surapprentissage, .lecture.stagne { border-left-color: var(--warn); }
.lecture p { margin: 4px 0 0; }
</style>
