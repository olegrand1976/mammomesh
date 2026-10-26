<script setup lang="ts">
// ECharts n'est importé que par les pages de suivi : la page publique ne le charge jamais.
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, ScatterChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent, MarkLineComponent } from 'echarts/components'
import VChart from 'vue-echarts'

use([CanvasRenderer, LineChart, ScatterChart, GridComponent, TooltipComponent, LegendComponent, MarkLineComponent])

defineProps<{ option: Record<string, unknown> }>()
const emit = defineEmits<{ click: [e: unknown] }>()
</script>

<template>
  <VChart :option="option" autoresize @click="(e: unknown) => emit('click', e)" />
</template>
