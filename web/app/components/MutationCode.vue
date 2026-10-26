<script setup lang="ts">
// Code de mutation avec son explication au survol et au focus clavier (lexique complet : LexiqueMutations).
import { infobulle } from '~/utils/mutations'
defineProps<{ code: string }>()
</script>

<template>
  <abbr class="mut" tabindex="0" :title="infobulle(code)" :aria-label="`${code} : ${infobulle(code)}`">{{ code }}</abbr>
</template>

<style scoped>
.mut { font-family: var(--mono); font-size: 0.86em; text-decoration: underline dotted; text-underline-offset: 3px; cursor: help; white-space: nowrap; }
.mut:focus-visible { outline: 2px solid var(--accent); outline-offset: 1px; border-radius: 3px; }
</style>
