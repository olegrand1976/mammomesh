<script setup lang="ts">
// Panneau 2 · Front de Pareto : sensibilité sur la fenêtre de rappel contre coût par examen ; l'ancien front en gris.
import type { Individual } from '~/types'

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
      lineStyle: { color: p.value.faint, width: 1 }, itemStyle: { color: p.value.faint } },
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
    <h2>Front de Pareto</h2>
    <p class="muted small">Individus réalisables non dominés. Cliquer un point ouvre sa fiche.</p>
    <EChart class="chart" :option="option" @click="onClick" />
  </section>
</template>

<style scoped>
.chart { height: 320px; width: 100%; }
</style>
