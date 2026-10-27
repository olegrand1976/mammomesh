<script setup lang="ts">
// Lecture en mots simples de la campagne affichée et, une fois finie, sa synthèse globale.
import { lireCampagne, syntheseCampagne } from '~/utils/progressionCampagne'
const store = useCampagneStore()
const c = computed(() => store.campaign!)
const finie = computed(() => ['terminee', 'arretee', 'en_echec'].includes(c.value.state))
const lu = computed(() => lireCampagne(store.generations, c.value.config?.projet?.plancher_sensibilite ?? 0.85, c.value.config?.evolution?.arret_stagnation ?? 15))
const synthese = computed(() => (finie.value ? syntheseCampagne(c.value, store.generations, store.front) : []))
</script>

<template>
  <section v-if="finie" class="card cadre synthese">
    <h2>Synthèse de la campagne</h2>
    <ul><li v-for="ph in synthese" :key="ph"><TexteGlossaire :texte="ph" /></li></ul>
    <p class="small muted">Pour l'instant, les mesures des campagnes sont simulées : cette synthèse montre la forme du bilan, pas un résultat.</p>
  </section>
  <section v-else-if="lu" class="card cadre">
    <h2>Lecture : {{ lu.titre }}</h2>
    <ul><li v-for="ph in lu.phrases" :key="ph"><TexteGlossaire :texte="ph" /></li></ul>
  </section>
</template>

<style scoped>
.cadre { border-left: 4px solid var(--accent); background: var(--accent-soft); }
.cadre h2 { margin: 0 0 6px; }
.cadre ul { margin: 0; padding-left: 18px; display: grid; gap: 4px; font-size: 0.92rem; }
.cadre p { margin: 8px 0 0; }
</style>
