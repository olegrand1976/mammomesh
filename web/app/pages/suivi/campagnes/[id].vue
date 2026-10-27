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
      <p class="small"><NuxtLink to="/suivi/campagnes">Campagnes</NuxtLink> ›</p>
      <div class="spacer" />
      <button v-if="store.campaign" type="button" class="btn btn-sm" :aria-pressed="aidesToutes" @click="aidesToutes = !aidesToutes">
        {{ aidesToutes ? 'Masquer les aides' : 'Afficher toutes les aides' }}
      </button>
    </div>
    <GuideCampagne v-if="store.campaign" />
    <p v-if="error" class="alert">{{ error }}</p>
    <div v-if="store.campaign" class="grid">
      <StatusBanner />
      <LectureCampagne v-if="store.campaign.state !== 'en_cours' && store.campaign.state !== 'en_pause'" />
      <div class="grid grid-2">
        <ProgressChart />
        <FrontChart @select="selected = $event" />
      </div>
      <LectureCampagne v-if="store.campaign.state === 'en_cours' || store.campaign.state === 'en_pause'" />
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
</style>
