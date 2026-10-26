<script setup lang="ts">
// Formule LaTeX rendue par KaTeX (MathML pour les lecteurs d'écran, HTML pour l'affichage).
import katex from 'katex'
import 'katex/dist/katex.min.css'

const props = defineProps<{ tex: string }>()
const html = computed(() => katex.renderToString(props.tex, { displayMode: true, throwOnError: false, output: 'htmlAndMathml', strict: 'ignore' }))
</script>

<template>
  <!-- eslint-disable-next-line vue/no-v-html -- sortie de KaTeX, à partir de formules du dépôt -->
  <div class="formule" v-html="html" />
</template>

<style scoped>
.formule { overflow-x: auto; overflow-y: hidden; padding: 4px 0; margin: 0 0 1em; }
</style>
