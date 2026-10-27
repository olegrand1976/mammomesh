<script setup lang="ts">
// Petite courbe AUC par sein + perte, pour un exemple du guide ou un entraînement.
import type { EpoqueEntrainement } from '~/utils/entrainements'
const props = defineProps<{ journal: EpoqueEntrainement[]; record?: number }>()
const p = usePalette()
const option = computed(() => {
  const j = props.journal
  const axe = { axisLine: { lineStyle: { color: p.value.grid } }, axisLabel: { color: p.value.muted }, nameTextStyle: { color: p.value.muted } }
  return {
    animation: false,
    textStyle: { color: p.value.text, fontFamily: 'inherit' },
    grid: { left: 40, right: 44, top: 28, bottom: 28 },
    legend: { top: 0, textStyle: { color: p.value.muted, fontSize: 11 } },
    tooltip: { trigger: 'axis', valueFormatter: (v: number) => v.toFixed(3) },
    xAxis: { type: 'category', data: j.map(e => e.epoque), ...axe },
    yAxis: [
      { type: 'value', min: 0.5, max: 0.85, splitLine: { lineStyle: { color: p.value.grid } }, ...axe },
      { type: 'value', scale: true, splitLine: { show: false }, ...axe },
    ],
    series: [
      {
        name: 'AUC par sein', type: 'line', data: j.map(e => e.auc_sein), lineStyle: { width: 2, color: p.value.accent }, itemStyle: { color: p.value.accent },
        markPoint: props.record ? { symbolSize: 34, label: { fontSize: 10 }, data: [{ name: 'gardée', coord: [String(props.record), j.find(e => e.epoque === props.record)?.auc_sein], value: 'gardée' }] } : undefined,
      },
      { name: 'Perte', type: 'line', yAxisIndex: 1, data: j.map(e => e.perte), lineStyle: { width: 1.5, type: 'dotted', color: p.value.second }, itemStyle: { color: p.value.second } },
    ],
  }
})
</script>

<template><EChart class="courbe" :option="option" /></template>

<style scoped>
.courbe { height: 180px; width: 100%; }
</style>
