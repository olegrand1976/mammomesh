<script setup lang="ts">
// Panneau 5 · Fiche d'un individu : génome CephaloMesh, mesures avec intervalle, parent.
import { fr, type Individual } from '~/types'

const props = defineProps<{ id: number | null }>()
const emit = defineEmits<{ close: [] }>()
const { api } = useApi()
const ind = ref<Individual | null>(null)
const error = ref('')

watch(() => props.id, async (id) => {
  ind.value = null
  error.value = ''
  if (!id) return
  try {
    ind.value = (await api<{ individual: Individual }>(`/individuals/${id}`)).individual
  } catch (e) {
    error.value = (e as Error).message
  }
}, { immediate: true })
</script>

<template>
  <section v-if="id" class="card">
    <div class="row">
      <h2>Fiche {{ ind?.name ?? '…' }}</h2>
      <div class="spacer" />
      <button class="btn" @click="emit('close')">Fermer</button>
    </div>
    <p v-if="error" class="alert">{{ error }}</p>
    <template v-if="ind">
      <table>
        <tbody>
          <tr><th>Sensibilité (E-ajust)</th><td>{{ fr(ind.sensitivity) }} [{{ fr(ind.sensitivity_low) }} ; {{ fr(ind.sensitivity_high) }}]</td></tr>
          <tr><th>Sensibilité (E-val)</th><td>{{ fr(ind.sensitivity_eval) }}</td></tr>
          <tr><th>Coût par examen</th><td>{{ fr(ind.cost_ms, 0) }} ms</td></tr>
          <tr><th>Mutations</th><td><code>{{ ind.mutations.join(' › ') }}</code></td></tr>
          <tr><th>Parent</th><td>{{ ind.parent_name ?? 'fondateur' }}</td></tr>
          <tr v-for="(v, k) in ind.genome" :key="k"><th>Gène {{ k }}</th><td>{{ Array.isArray(v) ? v.join(', ') : v }}</td></tr>
        </tbody>
      </table>
    </template>
  </section>
</template>
