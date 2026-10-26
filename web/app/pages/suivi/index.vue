<script setup lang="ts">
import { fr, stateLabel, type Campaign } from '~/types'

definePageMeta({ layout: 'suivi', middleware: 'auth' })
useHead({ title: 'Campagnes · MammoMesh' })

const { api } = useApi()
const session = useSessionStore()
const campaigns = ref<Campaign[]>([])
const error = ref('')
let timer: ReturnType<typeof setInterval> | undefined

async function load() {
  try {
    campaigns.value = (await api<{ campaigns: Campaign[] }>('/campaigns')).campaigns
    error.value = ''
  } catch (e) {
    error.value = (e as Error).message
  }
}

onMounted(() => {
  void load()
  timer = setInterval(load, 10000)
})
onBeforeUnmount(() => clearInterval(timer))
</script>

<template>
  <div>
    <div class="row head">
      <h1>Campagnes</h1>
      <div class="spacer" />
      <NuxtLink v-if="session.permissions?.proposer_config" to="/suivi/nouvelle" class="btn btn-primary">Nouvelle campagne</NuxtLink>
    </div>
    <p v-if="error" class="alert">{{ error }}</p>
    <section class="card">
      <div v-if="campaigns.length" class="table-wrap"><table>
        <thead>
          <tr><th>Campagne</th><th>État</th><th>Génération</th><th>Meilleure sensibilité (E-val)</th><th class="hide-sm">Budget</th><th class="hide-sm">Lancée le</th></tr>
        </thead>
        <tbody>
          <tr v-for="c in campaigns" :key="c.id">
            <td><NuxtLink :to="`/suivi/campagnes/${c.id}`">{{ c.name }}</NuxtLink><div class="small muted">{{ c.lineage }} · {{ c.config_hash.slice(0, 8) }}</div></td>
            <td><span class="badge" :class="c.state">{{ stateLabel[c.state] }}</span></td>
            <td>{{ c.generation }} / {{ c.max_generations }}</td>
            <td>{{ fr(c.best_sensitivity) }}</td>
            <td class="hide-sm">{{ fr(c.budget_used, 1) }} / {{ fr(c.budget_total, 0) }} cœur-h</td>
            <td class="small hide-sm">{{ c.started_at ? new Date(c.started_at).toLocaleString('fr-BE') : '—' }}</td>
          </tr>
        </tbody>
      </table></div>
      <p v-else class="muted">Aucune campagne pour l'instant.<template v-if="session.permissions?.proposer_config"> Créez-en une avec « Nouvelle campagne ».</template></p>
    </section>
  </div>
</template>

<style scoped>
.head { margin-bottom: 16px; }
.head h1 { margin: 0; font-size: 1.4rem; }
</style>
