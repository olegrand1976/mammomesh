<script setup lang="ts">
// Assistant simplifié : configuration → brouillon → contrôles avant lancement → lancement.
import type { Campaign, CampaignConfig, Draft } from '~/types'

definePageMeta({ layout: 'suivi', middleware: 'auth' })
useHead({ title: 'Nouvelle campagne · MammoMesh' })

const { api } = useApi()
const session = useSessionStore()
const config = ref<CampaignConfig | null>(null)
const draft = ref<Draft | null>(null)
const error = ref('')
const busy = ref(false)

onMounted(async () => {
  config.value = (await api<{ config: CampaignConfig }>('/config/default')).config
})

watch(config, () => { draft.value = null }, { deep: true })

async function verify() {
  error.value = ''
  busy.value = true
  try {
    const created = (await api<{ draft: Draft }>('/drafts', { method: 'POST', body: { config: config.value } })).draft
    draft.value = (await api<{ draft: Draft }>(`/drafts/${created.id}/verify`, { method: 'POST' })).draft
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    busy.value = false
  }
}

async function launch() {
  if (!draft.value) return
  busy.value = true
  try {
    const c = (await api<{ campaign: Campaign }>(`/drafts/${draft.value.id}/launch`, { method: 'POST' })).campaign
    await navigateTo(`/suivi/campagnes/${c.id}`)
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div>
    <h1>Nouvelle campagne</h1>
    <p class="muted">La configuration validée est figée et identifiée par son empreinte. La boucle CNN + CephaloMesh est simulée dans cette version du socle.</p>
    <p v-if="error" class="alert">{{ error }}</p>
    <form v-if="config" class="grid grid-2" @submit.prevent="verify">
      <section class="card">
        <h2>Projet</h2>
        <div class="field"><label>Nom</label><input v-model="config.projet.nom" required></div>
        <div class="field"><label>Objectif en une phrase</label><input v-model="config.projet.objectif"></div>
        <div class="field"><label>Lignée (majuscules)</label><input v-model="config.projet.lignee" pattern="[A-Z]{3,12}"></div>
        <div class="row">
          <div class="field"><label>Plancher de sensibilité</label><input v-model.number="config.projet.plancher_sensibilite" type="number" step="any" min="0.8" max="0.99"></div>
          <div class="field"><label>Marge (points)</label><input v-model.number="config.projet.marge_points" type="number" step="any" min="1" max="5"></div>
          <div class="field"><label>Taux de rappel cible</label><input v-model.number="config.projet.taux_rappel_cible" type="number" step="any" min="0.02" max="0.1"></div>
        </div>
      </section>
      <section class="card">
        <h2>Évolution</h2>
        <div class="row">
          <div class="field"><label>Population</label><input v-model.number="config.evolution.population" type="number" min="20" max="500"></div>
          <div class="field"><label>Générations au plus</label><input v-model.number="config.evolution.generations_max" type="number" min="10" max="2000"></div>
          <div class="field"><label>Arrêt si E-val stable (générations)</label><input v-model.number="config.evolution.arret_stagnation" type="number" min="3" max="100"></div>
        </div>
        <div class="row">
          <div class="field"><label>Budget (cœur-heures)</label><input v-model.number="config.evolution.budget_coeur_heures" type="number" step="any" min="0.1"></div>
          <div class="field"><label>μ mutations</label><input v-model.number="config.evolution.mutations_mu" type="number" step="any" min="0" max="4"></div>
          <div class="field"><label>Mutations au plus</label><input v-model.number="config.evolution.mutations_max" type="number" min="1" max="10"></div>
          <div class="field"><label>Graine</label><input v-model.number="config.evolution.graine" type="number"></div>
        </div>
        <h2>Simulation</h2>
        <div class="row">
          <div class="field"><label>Durée d'un lot (ms)</label><input v-model.number="config.simulation.duree_lot_ms" type="number" min="10"></div>
          <div class="field"><label>Lots par génération</label><input v-model.number="config.simulation.lots_par_generation" type="number" min="1" max="100"></div>
        </div>
        <button class="btn btn-primary" :disabled="busy">Vérifier la configuration</button>
      </section>
    </form>

    <section v-if="draft?.verification" class="card result">
      <h2>Contrôles avant lancement</h2>
      <table>
        <tbody>
          <tr v-for="c in draft.verification.controles" :key="c.nom">
            <td>{{ c.ok ? '✓' : '✗' }}</td><td>{{ c.nom }}</td><td class="muted">{{ c.detail }}</td>
          </tr>
        </tbody>
      </table>
      <p class="small">Durée estimée : {{ draft.verification.duree_estimee_min }} min au plus · {{ draft.verification.coeur_heures_estimees }} cœur-heures au plus.</p>
      <p v-if="draft.diff.length" class="small muted">Réglages modifiés : {{ draft.diff.map(d => d.champ).join(', ') }}</p>
      <button v-if="session.permissions?.lancer_campagnes" class="btn btn-primary" :disabled="busy || !draft.verified_ok" @click="launch">Lancer la campagne</button>
      <p v-else class="notice small">Le lancement est réservé au responsable scientifique et à l'administrateur.</p>
    </section>
  </div>
</template>

<style scoped>
h1 { font-size: 1.4rem; }
.row .field { flex: 1; min-width: 120px; }
.result { margin-top: 16px; }
td:first-child { width: 24px; }
</style>
