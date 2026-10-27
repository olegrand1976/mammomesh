<script setup lang="ts">
// Zones d'un écran et leurs éléments (guides du suivi et des images).
import type { ElementGuide } from '~/utils/guideSuivi'
defineProps<{ zones: { cle: string; titre: string; resume: string; elements: ElementGuide[]; aSavoir?: string }[] }>()
</script>

<template>
  <section v-for="(z, i) in zones" :id="z.cle" :key="z.cle" class="card">
    <h2>{{ i + 1 }}. {{ z.titre }}</h2>
    <p class="muted"><TexteGlossaire :texte="z.resume" /></p>
    <dl>
      <template v-for="e in z.elements" :key="e.nom">
        <dt><TexteGlossaire :texte="e.nom" /></dt>
        <dd><TexteGlossaire :texte="e.description" /></dd>
      </template>
    </dl>
    <p v-if="z.aSavoir"><strong>À savoir.</strong> <TexteGlossaire :texte="z.aSavoir" /></p>
  </section>
</template>

<style scoped>
dl { margin: 0 0 1em; display: grid; gap: 4px; }
dt { font-weight: 600; margin-top: 6px; }
dd { margin: 0; }
</style>
