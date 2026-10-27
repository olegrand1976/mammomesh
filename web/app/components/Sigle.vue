<script setup lang="ts">
// Sigle avec son explication : bulle au survol et au focus (clavier), lue aussi par les lecteurs d'écran.
import { ancre, terme } from '~/utils/glossaire'

// sansLien : page publique (le glossaire demande une connexion).
const props = defineProps<{ t: string; texte?: string; sansLien?: boolean }>()
const def = computed(() => terme(props.t))
const id = useId()
</script>

<template>
  <span v-if="def" class="sigle" tabindex="0" :aria-describedby="id">
    <abbr>{{ texte ?? t }}</abbr>
    <span :id="id" role="tooltip" class="bulle">
      <strong>{{ def.terme }}</strong><template v-if="def.forme"> : {{ def.forme }}</template>.
      {{ def.definition }}
      <NuxtLink v-if="!sansLien" :to="`/suivi/documentation/glossaire#${ancre(def.terme)}`" class="lien" tabindex="-1">Glossaire</NuxtLink>
    </span>
  </span>
  <span v-else>{{ texte ?? t }}</span>
</template>

<style scoped>
.sigle { position: relative; display: inline; cursor: help; }
abbr { text-decoration: underline dotted; text-underline-offset: 2px; text-decoration-color: color-mix(in srgb, currentColor 55%, transparent); }
.sigle:focus-visible { outline: 2px solid var(--accent); outline-offset: 1px; border-radius: 3px; }
.bulle { display: none; position: absolute; left: 50%; top: calc(100% + 6px); transform: translateX(-50%); z-index: 40;
  width: max-content; max-width: min(320px, 80vw); padding: 8px 10px; border-radius: 8px; background: var(--text); color: var(--surface);
  font-size: 0.8rem; font-weight: 400; line-height: 1.4; text-align: left; white-space: normal; box-shadow: var(--shadow); }
.sigle:hover .bulle, .sigle:focus .bulle, .sigle:focus-within .bulle { display: block; }
.bulle .lien { color: inherit; text-decoration: underline; margin-left: 2px; }
</style>
