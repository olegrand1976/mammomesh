<script setup lang="ts">
import { fr, stateLabel, type Campaign } from '~/types'

definePageMeta({ layout: 'suivi', middleware: 'auth' })
useHead({ title: 'Campagnes · MammoMesh' })

const { api } = useApi()
const session = useSessionStore()
const campaigns = ref<Campaign[]>([])
const error = ref('')
const info = ref('')
const voirDesactivees = ref(false)
const enCours = (c: Campaign) => c.state === 'en_cours' || c.state === 'en_pause'
const visibles = computed(() => campaigns.value.filter(c => voirDesactivees.value || !c.disabled_at))
const nbDesactivees = computed(() => campaigns.value.filter(c => c.disabled_at).length)
let timer: ReturnType<typeof setInterval> | undefined

async function load() {
  try {
    campaigns.value = (await api<{ campaigns: Campaign[] }>('/campaigns', { query: { desactivees: 1 } })).campaigns
    error.value = ''
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function basculer(c: Campaign) {
  const desactiver = !c.disabled_at
  if (desactiver && !window.confirm(`Désactiver « ${c.name} » ?\n\nElle reste consultable ici, mais n'est plus prise en compte : accueil, tests d'images, départ d'une nouvelle campagne et MCP. Réversible.`)) return
  try {
    await api(`/campaigns/${c.id}/control`, { method: 'POST', body: { action: desactiver ? 'desactiver' : 'reactiver' } })
    info.value = desactiver ? `« ${c.name} » est désactivée.` : `« ${c.name} » est réactivée.`
    await load()
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function supprimer(c: Campaign) {
  if (!window.confirm(`Supprimer définitivement « ${c.name} » ?\n\nSes ${c.generation} générations, ses individus, son journal et les tests d'images faits avec ses individus seront effacés. Cette action est irréversible ; pour seulement l'écarter, utilisez « Désactiver ».`)) return
  try {
    await api(`/campaigns/${c.id}`, { method: 'DELETE' })
    info.value = `« ${c.name} » est supprimée.`
    await load()
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
    <p v-if="info" class="attente">{{ info }}</p>
    <label v-if="nbDesactivees" class="small muted bascule"><input v-model="voirDesactivees" type="checkbox"> Afficher les campagnes désactivées ({{ nbDesactivees }})</label>
    <section class="card">
      <div v-if="visibles.length" class="table-wrap"><table>
        <thead>
          <tr><th>Campagne</th><th>État</th><th>Génération</th><th>Meilleure sensibilité (<Sigle t="E-val" />)</th><th class="hide-sm">Budget</th><th class="hide-sm">Lancée le</th><th v-if="session.permissions?.lancer_campagnes" class="actions">Actions</th></tr>
        </thead>
        <tbody>
          <tr v-for="c in visibles" :key="c.id" :class="{ desactivee: c.disabled_at }">
            <td><NuxtLink :to="`/suivi/campagnes/${c.id}`">{{ c.name }}</NuxtLink><div class="small muted">{{ c.lineage }} · {{ c.config_hash.slice(0, 8) }}</div></td>
            <td><span class="badge" :class="c.state">{{ stateLabel[c.state] }}</span> <span v-if="c.disabled_at" class="badge">Désactivée</span></td>
            <td>{{ c.generation }} / {{ c.max_generations }}</td>
            <td>{{ fr(c.best_sensitivity) }}</td>
            <td class="hide-sm">{{ fr(c.budget_used, 1) }} / {{ fr(c.budget_total, 0) }} cœur-h</td>
            <td class="small hide-sm">{{ c.started_at ? new Date(c.started_at).toLocaleString('fr-BE') : '—' }}</td>
            <td v-if="session.permissions?.lancer_campagnes" class="actions">
              <template v-if="!enCours(c)">
                <button type="button" class="btn btn-sm" :aria-label="`${c.disabled_at ? 'Réactiver' : 'Désactiver'} ${c.name}`" @click="basculer(c)">{{ c.disabled_at ? 'Réactiver' : 'Désactiver' }}</button>
                <button type="button" class="btn btn-sm btn-danger" :aria-label="`Supprimer ${c.name}`" @click="supprimer(c)">Supprimer</button>
              </template>
              <small v-else class="muted">Arrêtez-la d'abord</small>
            </td>
          </tr>
        </tbody>
      </table></div>
      <p v-else-if="campaigns.length" class="muted">Toutes les campagnes sont désactivées.</p>
      <p v-else class="muted">Aucune campagne pour l'instant.<template v-if="session.permissions?.proposer_config"> Créez-en une avec « Nouvelle campagne ».</template></p>
    </section>
  </div>
</template>

<style scoped>
.head { margin-bottom: 16px; }
.head h1 { margin: 0; font-size: 1.4rem; }
.bascule { display: inline-flex; align-items: center; gap: 6px; margin-bottom: 8px; }
tr.desactivee td { opacity: 0.6; }
.actions { white-space: nowrap; text-align: right; }
.actions .btn + .btn { margin-left: 6px; }
.btn-sm { padding: 3px 8px; font-size: 0.8rem; }
.attente { background: var(--accent-soft); color: var(--accent); padding: 8px 12px; border-radius: 8px; }
</style>
