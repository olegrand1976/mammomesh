<script setup lang="ts">
import { ancre, glossaire } from '~/utils/glossaire'

definePageMeta({ layout: 'suivi', middleware: 'auth' })
useHead({ title: 'Glossaire · MammoMesh' })

const filtre = ref('')
const norm = (s: string) => s.normalize('NFD').replace(/[̀-ͯ]/g, '').toLowerCase()
const trouves = computed(() => glossaire.filter(t => !filtre.value
  || norm(`${t.terme} ${t.forme ?? ''} ${t.definition}`).includes(norm(filtre.value))))
const parties = computed(() => [
  { id: 'sigles', titre: 'Sigles', termes: trouves.value.filter(t => t.sigle).sort((a, b) => a.terme.localeCompare(b.terme, 'fr')) },
  { id: 'mots', titre: 'Mots', termes: trouves.value.filter(t => !t.sigle).sort((a, b) => a.terme.localeCompare(b.terme, 'fr')) },
])
</script>

<template>
  <DocPage titre="Glossaire"
    intro="Sigles et mots techniques de l'interface. Dans chaque écran, un sigle souligné en pointillé affiche son explication au survol ou au clavier (Tab)."
    :sommaire="[{ id: 'sigles', titre: 'Sigles' }, { id: 'mots', titre: 'Mots' }]">
    <label class="recherche">
      <span class="small muted">Chercher</span>
      <input v-model="filtre" type="search" placeholder="par exemple : AUC, époque, Pareto">
    </label>
    <section v-for="p in parties" :id="p.id" :key="p.id" class="card">
      <h2>{{ p.titre }} <span class="small muted">({{ p.termes.length }})</span></h2>
      <dl>
        <template v-for="t in p.termes" :key="t.terme">
          <dt :id="ancre(t.terme)">{{ t.terme }}<span v-if="t.forme" class="forme"> : {{ t.forme }}</span></dt>
          <dd><TexteGlossaire :texte="t.definition" /></dd>
        </template>
      </dl>
      <p v-if="!p.termes.length" class="muted small">Aucun terme ne correspond.</p>
    </section>
  </DocPage>
</template>

<style scoped>
.recherche { display: grid; gap: 4px; max-width: 360px; }
dl { margin: 0; display: grid; gap: 4px; }
dt { font-weight: 600; margin-top: 8px; scroll-margin-top: 16px; }
dt:target { color: var(--accent); }
.forme { font-weight: 400; color: var(--muted); }
dd { margin: 0; }
</style>
