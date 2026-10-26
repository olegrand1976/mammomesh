<script setup lang="ts">
// Panneau 1 · Progrès : meilleur et médiane par génération, et écart E-ajust − E-val (optimisme de la sélection).
const store = useCampagneStore()
const p = usePalette()

const option = computed(() => {
  const gens = store.generations
  const floor = store.campaign?.config.projet.plancher_sensibilite
  return {
    animation: false,
    textStyle: { color: p.value.text, fontFamily: 'inherit' },
    grid: { left: 48, right: 56, top: 56, bottom: 36 },
    legend: { top: 0, textStyle: { color: p.value.muted } },
    tooltip: { trigger: 'axis', valueFormatter: (v: number | null) => (v === null || v === undefined ? 'aucun réalisable' : v.toFixed(3)) },
    xAxis: { type: 'category', name: 'génération', nameLocation: 'middle', nameGap: 24, data: gens.map(g => g.number),
      axisLine: { lineStyle: { color: p.value.grid } }, axisLabel: { color: p.value.muted } },
    yAxis: [
      { type: 'value', name: 'sensibilité', scale: true, splitLine: { lineStyle: { color: p.value.grid } }, axisLabel: { color: p.value.muted }, nameTextStyle: { color: p.value.muted } },
      { type: 'value', name: 'écart', position: 'right', splitLine: { show: false }, axisLabel: { color: p.value.muted, formatter: (v: number) => (v * 100).toFixed(1) + ' pt' }, nameTextStyle: { color: p.value.muted } },
    ],
    series: [
      { name: 'Meilleur (E-val)', type: 'line', showSymbol: false, data: gens.map(g => g.best_sensitivity), lineStyle: { width: 2, color: p.value.accent }, itemStyle: { color: p.value.accent },
        markLine: floor ? { silent: true, symbol: 'none', label: { formatter: 'plancher', position: 'insideStartTop', color: p.value.muted }, lineStyle: { type: 'dashed', color: p.value.muted }, data: [{ yAxis: floor }] } : undefined },
      { name: 'Médiane', type: 'line', showSymbol: false, data: gens.map(g => g.median_sensitivity), lineStyle: { width: 1.5, color: p.value.muted }, itemStyle: { color: p.value.muted } },
      { name: 'Écart E-ajust − E-val', type: 'line', yAxisIndex: 1, showSymbol: false, data: gens.map(g => g.eval_gap), lineStyle: { width: 1.5, type: 'dotted', color: p.value.second }, itemStyle: { color: p.value.second } },
    ],
  }
})
</script>

<template>
  <section class="card">
    <h2>Progrès par génération</h2>
    <p v-if="!store.generations.length" class="muted small">Première génération en cours…</p>
    <EChart class="chart" :option="option" />
  </section>
</template>

<style scoped>
.chart { height: 320px; width: 100%; }
</style>
