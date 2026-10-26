<script setup lang="ts">
definePageMeta({ layout: 'suivi', middleware: 'auth' })

const route = useRoute()
const store = useCampagneStore()
const error = ref('')
const selected = ref<number | null>(null)
const aidesToutes = ref(false)
provide('aides-toutes', aidesToutes)

useHead(() => ({ title: store.campaign ? `${store.campaign.name} · MammoMesh` : 'Campagne · MammoMesh' }))

onMounted(async () => {
  try {
    await store.load(Number(route.params.id))
  } catch (e) {
    error.value = (e as Error).message
  }
})
onBeforeUnmount(() => store.disconnect())
</script>

<template>
  <div>
    <div class="row crumbs">
      <p class="small"><NuxtLink to="/suivi">Campagnes</NuxtLink> ›</p>
      <div class="spacer" />
      <button v-if="store.campaign" type="button" class="btn btn-sm" :aria-pressed="aidesToutes" @click="aidesToutes = !aidesToutes">
        {{ aidesToutes ? 'Masquer les aides' : 'Afficher toutes les aides' }}
      </button>
    </div>
    <details v-if="store.campaign" class="card lecture">
      <summary>Comment lire cet écran</summary>
      <ol>
        <li><strong>Bandeau</strong> : état, commandes et chiffres clés (génération, budget, meilleure sensibilité, fin estimée).</li>
        <li><strong>Progrès par génération</strong> : le meilleur et la médiane progressent-ils, l'optimisme de la sélection reste-t-il faible ?</li>
        <li><strong>Front de Pareto</strong> : les meilleurs compromis sensibilité / coût ; un clic sur un point ouvre sa fiche.</li>
        <li><strong>Journal</strong> : chaque événement en phrase, le plus récent en tête.</li>
      </ol>
      <p class="muted small">Tout se met à jour en direct, sans recharger. Chaque zone a son bouton <span class="q">?</span>. Fonctionnement complet et lexique des mutations : <NuxtLink to="/suivi/fonctionnement">Fonctionnement</NuxtLink> · guide détaillé : <code>docs/guide-suivi-campagne.md</code>.</p>
    </details>
    <p v-if="error" class="alert">{{ error }}</p>
    <div v-if="store.campaign" class="grid">
      <StatusBanner />
      <div class="grid grid-2">
        <ProgressChart />
        <FrontChart @select="selected = $event" />
      </div>
      <IndividualCard :id="selected" @close="selected = null" />
      <JournalList />
    </div>
    <p v-else-if="!error" class="muted">Chargement…</p>
  </div>
</template>

<style scoped>
.crumbs { margin: 0 0 10px; }
.crumbs p { margin: 0; }
.btn-sm { padding: 4px 10px; font-size: 0.85rem; }
.lecture { margin-bottom: 14px; padding: 12px 18px; }
.lecture summary { cursor: pointer; font-weight: 600; }
.lecture ol { margin: 10px 0 8px; padding-left: 20px; display: grid; gap: 4px; }
.q { display: inline-grid; place-items: center; width: 18px; height: 18px; border-radius: 50%; border: 1px solid var(--border); font-size: 0.7rem; font-weight: 600; }
</style>
