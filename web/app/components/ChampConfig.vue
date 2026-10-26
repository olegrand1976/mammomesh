<script setup lang="ts">
// Un réglage de campagne : libellé, saisie, plage et valeur par défaut, et l'aide du guide
// (rôle, effet d'un changement) dépliable ; « Afficher toutes les aides » les ouvre ensemble.
import type { Ref } from 'vue'
import { champParCle } from '~/utils/guideCampagne'

defineOptions({ inheritAttrs: false })
const props = defineProps<{ cle: string; defaut?: unknown; type?: 'text' | 'number' }>()
const model = defineModel<string | number>({ required: true })

const guide = computed(() => champParCle[props.cle])
const id = computed(() => 'cfg-' + props.cle.replace('.', '-'))
const toutes = inject<Ref<boolean>>('aides-toutes', ref(false))
const ouvert = ref(false)
watch(toutes, v => { ouvert.value = v })

const modifie = computed(() => props.defaut !== undefined && model.value !== props.defaut)
const saisie = computed(() => {
  const g = guide.value
  if (!g?.plage) return g?.format ?? ''
  return `${nombre(g.plage[0])} à ${nombre(g.plage[1])}${g.unite ? ' ' + g.unite : ''}`
})
const nombre = (v: unknown) => typeof v === 'number' ? v.toLocaleString('fr-BE', { maximumFractionDigits: 4 }) : String(v)

function onInput(e: Event) {
  const v = (e.target as HTMLInputElement).value
  model.value = props.type === 'number' && v !== '' ? Number(v) : v
}
</script>

<template>
  <div v-if="guide" class="field champ" :class="{ modifie }">
    <div class="tete">
      <label :for="id">{{ guide.libelle }}</label>
      <button type="button" class="aide-btn" :aria-expanded="ouvert" :aria-controls="id + '-aide'"
        :title="`À quoi sert « ${guide.libelle} » ?`" :aria-label="`Aide : ${guide.libelle}`" @click="ouvert = !ouvert">?</button>
    </div>
    <input :id="id" v-bind="$attrs" :type="type ?? 'text'" :value="model" :aria-describedby="id + '-info'" @input="onInput">
    <p :id="id + '-info'" class="info">
      <span>{{ saisie }}</span>
      <span v-if="defaut !== undefined">{{ ` · défaut ${nombre(defaut)}` }}</span>
      <template v-if="modifie">
        · <strong>modifié</strong>
        <button type="button" class="lien" @click="model = defaut as string | number">rétablir</button>
      </template>
    </p>
    <div v-show="ouvert" :id="id + '-aide'" class="aide" role="note">
      <p><strong>À quoi ça sert.</strong> {{ guide.role }}</p>
      <p><strong>Effet d'un changement.</strong> {{ guide.impact }}</p>
      <p v-if="guide.boucleReelle"><strong>Avec la vraie boucle.</strong> {{ guide.boucleReelle }}</p>
    </div>
  </div>
</template>

<style scoped>
.champ { min-width: 0; }
.tete { display: flex; align-items: center; gap: 6px; }
.tete label { margin: 0; }
.aide-btn { width: 20px; height: 20px; flex: none; border-radius: 50%; border: 1px solid var(--border); background: var(--surface);
  color: var(--muted); font: 600 0.72rem/1 var(--font); cursor: pointer; padding: 0; }
.aide-btn[aria-expanded='true'] { background: var(--accent); border-color: var(--accent); color: #fff; }
.tete + input { margin-top: 4px; }
.info { margin: 3px 0 0; font-size: 0.76rem; color: var(--muted); }
.modifie input { border-color: var(--accent); }
.modifie strong { color: var(--accent); font-weight: 600; }
.lien { border: 0; background: none; color: var(--accent); font: inherit; padding: 0; cursor: pointer; text-decoration: underline; }
.aide { margin-top: 6px; padding: 8px 10px; border-left: 3px solid var(--accent); background: var(--accent-soft); border-radius: 0 6px 6px 0; font-size: 0.84rem; }
.aide p { margin: 0 0 6px; }
.aide p:last-child { margin: 0; }
</style>
