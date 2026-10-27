<script setup lang="ts">
// Panneau 3 · Journal en phrases : chaque événement arrive en tête, sans fusion.
import { decouperMessage } from '~/utils/mutations'
const store = useCampagneStore()
const filter = ref('')

const labels: Record<string, string> = {
  '': 'Tous',
  nouveau_meilleur: 'Nouveaux meilleurs',
  generation_terminee: 'Générations',
  individus_ecartes: 'Individus écartés',
  eval_stable: 'Stagnation',
  etat_change: 'Changements d\'état',
}

const items = computed(() => (filter.value ? store.journal.filter(e => e.type === filter.value) : store.journal).slice(0, 200))
const time = (s: string) => new Date(s).toLocaleTimeString('fr-BE')
</script>

<template>
  <section class="card">
    <div class="row head">
      <h2>Journal</h2>
      <AideZone zone="journal" />
      <div class="spacer" />
      <select v-model="filter" aria-label="Filtrer le journal">
        <option v-for="(l, k) in labels" :key="k" :value="k">{{ l }}</option>
      </select>
    </div>
    <div id="aide-journal-ici" />
    <ol class="journal">
      <li v-for="e in items" :key="e.id" :class="e.type">
        <time>{{ time(e.created_at) }}</time>
        <span><template v-for="(m, i) in decouperMessage(e.message)" :key="i"><MutationCode v-if="m.code" :code="m.code" /><TexteGlossaire v-else :texte="m.texte ?? ''" /></template></span>
      </li>
      <li v-if="!items.length" class="muted">Aucun événement.</li>
    </ol>
    <LexiqueMutations />
  </section>
</template>

<style scoped>
.head h2 { margin: 0; }
.head select { width: auto; padding: 4px 8px; font-size: 0.85rem; }
.journal { list-style: none; margin: 12px 0 0; padding: 0; max-height: 360px; overflow-y: auto; }
.journal li { display: grid; grid-template-columns: 70px 1fr; gap: 10px; padding: 7px 0; border-bottom: 1px solid var(--border); font-size: 0.9rem; }
.journal li:last-child { border-bottom: none; }
time { color: var(--muted); font-variant-numeric: tabular-nums; font-size: 0.82rem; }
.nouveau_meilleur span { color: var(--accent); font-weight: 500; }
.campagne_terminee span, .etat_change span { font-weight: 500; }
.campagne_en_echec span { color: var(--bad); }
</style>
