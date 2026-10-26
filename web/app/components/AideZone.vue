<script setup lang="ts">
// Bouton « ? » d'une zone de l'écran de suivi et son aide : rôle de la zone et de chacun de ses éléments.
import type { Ref } from 'vue'
import { zoneParCle, type ZoneGuide } from '~/utils/guideSuivi'

const props = defineProps<{ zone: ZoneGuide['cle'] }>()
const z = computed(() => zoneParCle[props.zone])
const toutes = inject<Ref<boolean>>('aides-toutes', ref(false))
const ouvert = ref(toutes.value)
watch(toutes, v => { ouvert.value = v })
const id = computed(() => `aide-${props.zone}`)
const libelle = computed(() => `À quoi sert « ${z.value.titre} » ?`) // nom accessible stable ; aria-expanded dit l'état
</script>

<template>
  <button type="button" class="aide-btn" :aria-expanded="ouvert" :aria-controls="id"
    :title="libelle" :aria-label="libelle" @click="ouvert = !ouvert">?</button>
  <Teleport v-if="ouvert" defer :to="`#${id}-ici`">
    <div :id="id" class="aide-zone" role="note">
      <p class="resume">{{ z.resume }}</p>
      <dl>
        <template v-for="e in z.elements" :key="e.nom">
          <dt>{{ e.nom }}</dt>
          <dd>{{ e.description }}</dd>
        </template>
      </dl>
      <p v-if="z.aSavoir" class="a-savoir"><strong>À savoir.</strong> {{ z.aSavoir }}</p>
    </div>
  </Teleport>
</template>

<style scoped>
.aide-btn { width: 22px; height: 22px; flex: none; border-radius: 50%; border: 1px solid var(--border); background: var(--surface);
  color: var(--muted); font: 600 0.75rem/1 var(--font); cursor: pointer; padding: 0; }
.aide-btn[aria-expanded='true'] { background: var(--accent); border-color: var(--accent); color: #fff; }
.aide-zone { margin: 10px 0 12px; padding: 10px 14px; border-left: 3px solid var(--accent); background: var(--accent-soft);
  border-radius: 0 8px 8px 0; font-size: 0.86rem; }
.resume { margin: 0 0 8px; font-weight: 500; }
dl { margin: 0; display: grid; gap: 6px; }
dt { font-weight: 600; }
dd { margin: 0 0 4px; }
.a-savoir { margin: 8px 0 0; }
</style>
