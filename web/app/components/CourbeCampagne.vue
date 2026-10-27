<script setup lang="ts">
// Petite courbe « meilleure sensibilité par génération » pour un exemple du guide, avec la ligne du minimum exigé.
const props = defineProps<{ points: [number, number | null][]; plancher?: number }>()
const p = usePalette()
const option = computed(() => {
  const axe = { axisLine: { lineStyle: { color: p.value.grid } }, axisLabel: { color: p.value.muted }, nameTextStyle: { color: p.value.muted } }
  return {
    animation: false,
    textStyle: { color: p.value.text, fontFamily: 'inherit' },
    grid: { left: 44, right: 12, top: 16, bottom: 28 },
    tooltip: { trigger: 'axis', valueFormatter: (v: number | null) => (v == null ? 'aucune version assez sûre' : v.toFixed(3)) },
    xAxis: { type: 'category', name: 'génération', nameLocation: 'middle', nameGap: 20, data: props.points.map(x => x[0]), ...axe },
    yAxis: { type: 'value', min: 0.8, max: 0.9, splitLine: { lineStyle: { color: p.value.grid } }, ...axe },
    series: [{
      name: 'Meilleure sensibilité (E-val)', type: 'line', connectNulls: false, data: props.points.map(x => x[1]),
      lineStyle: { width: 2, color: p.value.accent }, itemStyle: { color: p.value.accent },
      markLine: { symbol: 'none', silent: true, label: { formatter: 'minimum', color: p.value.muted, fontSize: 10 }, lineStyle: { type: 'dashed', color: p.value.muted }, data: [{ yAxis: props.plancher ?? 0.85 }] },
    }],
  }
})
</script>

<template><EChart class="courbe" :option="option" /></template>

<style scoped>
.courbe { height: 170px; width: 100%; }
</style>
