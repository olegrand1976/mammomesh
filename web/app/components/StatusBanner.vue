<script setup lang="ts">
import { fr, stateLabel } from '~/types'

const store = useCampagneStore()
const session = useSessionStore()
const busy = ref(false)
const error = ref('')

const c = computed(() => store.campaign!)
// La progression est fusionnée (un message toutes les 2 s) : on ignore un message plus ancien que l'état.
const live = computed(() => {
  const p = store.progress
  return p && c.value.state === 'en_cours' && p.generation === c.value.generation + 1 ? p : null
})
const budgetUsed = computed(() => Math.max(live.value?.budget_used ?? 0, c.value.budget_used))
const budgetPct = computed(() => Math.min(100, (budgetUsed.value / c.value.budget_total) * 100))
const genPct = computed(() => Math.min(100, (c.value.generation / c.value.max_generations) * 100))
const eta = computed(() => {
  const gens = store.generations
  if (c.value.state !== 'en_cours' || gens.length < 2) return '—'
  const recent = gens.slice(-10)
  const ms = recent.reduce((t, g) => t + (Date.parse(g.finished_at) - Date.parse(g.started_at)), 0) / recent.length
  const left = (c.value.max_generations - c.value.generation) * ms
  return left < 60000 ? `${Math.ceil(left / 1000)} s au plus` : `${Math.ceil(left / 60000)} min au plus`
})

async function act(action: 'pause' | 'reprise' | 'arret' | 'budget') {
  error.value = ''
  let hausse = 0
  if (action === 'arret' && !confirm('Arrêter définitivement cette campagne ?')) return
  if (action === 'budget') {
    const v = prompt('Hausse du budget, en cœur-heures :', '10')
    hausse = Number(v?.replace(',', '.'))
    if (!hausse || hausse <= 0) return
  }
  busy.value = true
  try {
    await store.control(action, hausse)
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <section class="card banner">
    <div class="row">
      <h1>{{ c.name }}</h1>
      <span class="badge" :class="c.state">{{ stateLabel[c.state] }}</span>
      <span v-if="c.state === 'en_cours' || c.state === 'en_pause'" class="live" :class="{ on: store.connected }" :title="store.connected ? 'Flux temps réel connecté' : 'Reconnexion…'">
        {{ store.connected ? 'En direct' : 'Reconnexion…' }}
      </span>
      <AideZone zone="bandeau" />
      <div class="spacer" />
      <template v-if="session.permissions?.lancer_campagnes">
        <button v-if="c.state === 'en_cours'" class="btn" :disabled="busy" @click="act('pause')">Pause</button>
        <button v-if="c.state === 'en_pause'" class="btn" :disabled="busy" @click="act('reprise')">Reprendre</button>
        <button v-if="c.state === 'en_cours' || c.state === 'en_pause'" class="btn" :disabled="busy" @click="act('budget')">Budget +</button>
        <button v-if="c.state === 'en_cours' || c.state === 'en_pause'" class="btn btn-danger" :disabled="busy" @click="act('arret')">Arrêter</button>
      </template>
      <NuxtLink v-if="(c.state === 'terminee' || c.state === 'arretee') && session.permissions?.proposer_config"
        :to="`/suivi/nouvelle?depart=${c.id}`" class="btn btn-primary">Repartir de cette campagne</NuxtLink>
    </div>
    <div id="aide-bandeau-ici" />
    <p v-if="error" class="alert">{{ error }}</p>
    <div class="stats">
      <div>
        <span class="label">Génération</span>
        <strong>{{ c.generation }} / {{ c.max_generations }}</strong>
        <div class="bar"><i :style="{ width: genPct + '%' }" /></div>
        <span v-if="live" class="small muted">
          génération {{ live.generation }} : lot {{ live.lot }} sur {{ live.lots }}
        </span>
      </div>
      <div>
        <span class="label">Budget</span>
        <strong>{{ fr(budgetUsed, 2) }} / {{ fr(c.budget_total, 0) }} cœur-h</strong>
        <div class="bar"><i :style="{ width: budgetPct + '%' }" /></div>
      </div>
      <div>
        <span class="label">Meilleure sensibilité (<Sigle t="E-val" />)</span>
        <strong>{{ fr(c.best_sensitivity) }}</strong>
        <span class="small muted">{{ c.stagnation }} génération(s) sans progrès</span>
      </div>
      <div>
        <span class="label">Fin estimée</span>
        <strong>{{ eta }}</strong>
        <span class="small muted">lignée {{ c.lineage }} · configuration <code>{{ c.config_hash.slice(0, 8) }}</code></span>
        <span v-if="c.config.depart" class="small muted">départ :
          <NuxtLink :to="`/suivi/campagnes/${c.config.depart.campagne_id}`">campagne {{ c.config.depart.campagne_id }}</NuxtLink>,
          génération {{ c.config.depart.generation }}</span>
      </div>
    </div>
  </section>
</template>

<style scoped>
.banner h1 { font-size: 1.35rem; margin: 0; }
.stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 18px; margin-top: 16px; }
.stats > div { display: flex; flex-direction: column; gap: 3px; }
.label { font-size: 0.8rem; color: var(--muted); }
strong { font-size: 1.1rem; font-variant-numeric: tabular-nums; }
.bar { height: 6px; background: var(--surface-2); border-radius: 3px; overflow: hidden; }
.bar i { display: block; height: 100%; background: var(--accent); transition: width 0.4s ease; }
.live { font-size: 0.78rem; color: var(--muted); display: inline-flex; align-items: center; gap: 6px; }
.live::before { content: ''; width: 8px; height: 8px; border-radius: 50%; background: var(--warn); }
.live.on::before { background: var(--good); box-shadow: 0 0 0 3px color-mix(in srgb, var(--good) 25%, transparent); }
</style>
