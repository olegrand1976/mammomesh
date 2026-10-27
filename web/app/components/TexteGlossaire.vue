<script setup lang="ts">
// Texte dont chaque sigle du glossaire reçoit son infobulle (composant Sigle).
import { decouper } from '~/utils/glossaire'
const props = defineProps<{ texte: string; sansLien?: boolean }>()
const morceaux = computed(() => decouper(props.texte))
</script>

<template><template v-for="(m, i) in morceaux" :key="i"><Sigle v-if="m.sigle" :t="m.sigle.terme" :texte="m.texte" :sans-lien="sansLien" /><template v-else>{{ m.texte }}</template></template></template>
