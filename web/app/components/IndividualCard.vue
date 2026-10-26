<script setup lang="ts">
// Panneau 5 · Fiche d'un individu : génome CephaloMesh, mesures avec intervalle, parent.
import { fr, type Individual } from '~/types'
import { expliquer } from '~/utils/mutations'

const props = defineProps<{ id: number | null }>()
const emit = defineEmits<{ close: [] }>()
const { api } = useApi()
const ind = ref<Individual | null>(null)
const error = ref('')
const carte = ref<HTMLElement | null>(null)
// Une explication par code distinct, dans l'ordre d'apparition.
const explications = computed(() => [...new Set(ind.value?.mutations ?? [])].map(code => ({ code, m: expliquer(code) })))

watch(() => props.id, async (id) => {
  ind.value = null
  error.value = ''
  if (!id) return
  await nextTick()
  carte.value?.scrollIntoView({ behavior: 'smooth', block: 'nearest' }) // la fiche s'ouvre sous les graphiques
  try {
    ind.value = (await api<{ individual: Individual }>(`/individuals/${id}`)).individual
  } catch (e) {
    error.value = (e as Error).message
  }
}, { immediate: true })
</script>

<template>
  <section v-if="id" ref="carte" class="card">
    <div class="row">
      <h2>Fiche {{ ind?.name ?? '…' }}</h2>
      <AideZone zone="fiche" />
      <div class="spacer" />
      <button class="btn" @click="emit('close')">Fermer</button>
    </div>
    <div id="aide-fiche-ici" />
    <p v-if="error" class="alert">{{ error }}</p>
    <template v-if="ind">
      <table>
        <tbody>
          <tr><th>Sensibilité (E-ajust)</th><td>{{ fr(ind.sensitivity) }} [{{ fr(ind.sensitivity_low) }} ; {{ fr(ind.sensitivity_high) }}]</td></tr>
          <tr><th>Sensibilité (E-val)</th><td>{{ fr(ind.sensitivity_eval) }}</td></tr>
          <tr><th>Coût par examen</th><td>{{ fr(ind.cost_ms, 0) }} ms</td></tr>
          <tr>
            <th>Mutations</th>
            <td>
              <span v-for="(m, i) in ind.mutations" :key="i"><template v-if="i"> › </template><MutationCode :code="m" /></span>
              <ul class="explic">
                <li v-for="e in explications" :key="e.code">
                  <code>{{ e.code }}</code> <strong>{{ e.m?.gene ?? 'code inconnu' }}</strong><template v-if="e.m"> ({{ e.m.partie }}) : {{ e.m.effet }}</template>
                </li>
              </ul>
            </td>
          </tr>
          <tr><th>Parent</th><td>{{ ind.parent_name ?? 'fondateur' }}</td></tr>
          <tr v-for="(v, k) in ind.genome" :key="k"><th>Gène {{ k }}</th><td>{{ Array.isArray(v) ? v.join(', ') : v }}</td></tr>
        </tbody>
      </table>
      <LexiqueMutations />
    </template>
  </section>
</template>

<style scoped>
.explic { margin: 6px 0 0; padding-left: 18px; font-size: 0.86rem; display: grid; gap: 3px; }
th { white-space: nowrap; vertical-align: top; }
</style>
