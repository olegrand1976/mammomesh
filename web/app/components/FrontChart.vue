<script setup lang="ts">
// Panneau 2 · Front de Pareto : sensibilité sur la fenêtre de rappel contre coût par examen ; l'ancien front en gris.
import { fr, type Individual } from '~/types'

const emit = defineEmits<{ select: [id: number] }>()
const store = useCampagneStore()
const p = usePalette()

const pts = (list: Individual[]) => list.map(i => ({ value: [i.cost_ms, i.sensitivity], id: i.id, name: i.name }))

const option = computed(() => ({
  animation: false,
  textStyle: { color: p.value.text, fontFamily: 'inherit' },
  grid: { left: 56, right: 20, top: 56, bottom: 40 },
  legend: { top: 0, textStyle: { color: p.value.muted } },
  tooltip: {
    trigger: 'item',
    formatter: (it: { data: { name: string; value: [number, number] } }) =>
      `${it.data.name}<br>sensibilité ${it.data.value[1].toFixed(3)}<br>${it.data.value[0].toFixed(0)} ms par examen`,
  },
  xAxis: { type: 'value', name: 'coût par examen (ms)', nameLocation: 'middle', nameGap: 26, scale: true,
    splitLine: { lineStyle: { color: p.value.grid } }, axisLabel: { color: p.value.muted }, nameTextStyle: { color: p.value.muted } },
  yAxis: { type: 'value', name: 'sensibilité (E-ajust)', scale: true,
    splitLine: { lineStyle: { color: p.value.grid } }, axisLabel: { color: p.value.muted }, nameTextStyle: { color: p.value.muted } },
  series: [
    { name: 'Front précédent', type: 'line', step: 'end', symbolSize: 6, data: pts(store.previousFront),
      lineStyle: { color: p.value.muted, width: 1, opacity: 0.45 }, itemStyle: { color: p.value.muted, opacity: 0.45 } },
    { name: `Front, génération ${store.frontGeneration}`, type: 'line', step: 'end', symbolSize: 9, data: pts(store.front),
      lineStyle: { color: p.value.accent, width: 2 }, itemStyle: { color: p.value.accent } },
  ],
}))

function onClick(e: unknown) {
  const id = (e as { data?: { id?: number } }).data?.id
  if (id) emit('select', id)
}
</script>

<template>
  <section class="card">
    <div class="row titre"><h2>Front de Pareto</h2><AideZone zone="front" /></div>
    <div id="aide-front-ici" />
    <p class="muted small">Individus réalisables non dominés. Cliquer un point ouvre sa fiche.</p>
    <p v-if="!store.front.length" class="notice small">Aucun individu réalisable pour l'instant : le front apparaît dès qu'un individu dépasse le plancher de sensibilité.</p>
    <EChart class="chart" :option="option" @click="onClick" />
    <details v-if="store.front.length" class="liste">
      <summary>Liste des individus du front ({{ store.front.length }})</summary>
      <div class="table-wrap"><table>
        <thead><tr><th>Individu</th><th>Sensibilité (E-ajust)</th><th>Coût</th><th /></tr></thead>
        <tbody>
          <tr v-for="i in store.front" :key="i.id">
            <td>{{ i.name }}</td><td>{{ fr(i.sensitivity) }}</td><td>{{ fr(i.cost_ms, 0) }} ms</td>
            <td><button type="button" class="lien" @click="emit('select', i.id)">Fiche</button></td>
          </tr>
        </tbody>
      </table></div>
    </details>
  </section>
</template>

<style scoped>
.chart { height: 320px; width: 100%; }
.titre { gap: 8px; margin-bottom: 12px; }
.liste { margin-top: 8px; font-size: 0.88rem; }
.liste summary { cursor: pointer; color: var(--muted); }
.liste td, .liste th { padding: 5px 6px; }
.lien { border: 0; background: none; color: var(--accent); font: inherit; padding: 0; cursor: pointer; text-decoration: underline; }
.titre h2 { margin: 0; }
</style>
