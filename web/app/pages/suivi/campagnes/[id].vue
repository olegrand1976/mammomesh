<script setup lang="ts">
definePageMeta({ layout: 'suivi', middleware: 'auth' })

const route = useRoute()
const store = useCampagneStore()
const error = ref('')
const selected = ref<number | null>(null)

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
    <p class="crumbs small"><NuxtLink to="/suivi">Campagnes</NuxtLink> ›</p>
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
</style>
