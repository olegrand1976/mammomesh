<script setup lang="ts">
import { champs, sections, type Section } from '~/utils/guideCampagne'

definePageMeta({ layout: 'suivi', middleware: 'auth' })
useHead({ title: 'Guide : Nouvelle campagne · MammoMesh' })
const ordre: Section[] = ['projet', 'evolution', 'simulation']
const sommaire = [...ordre.map(s => ({ id: s, titre: sections[s].titre })), { id: 'repartir', titre: 'Repartir d\'une campagne finie' }]
const plage = (c: (typeof champs)[number]) => c.plage
  ? `${c.plage[0].toLocaleString('fr-BE')} à ${c.plage[1].toLocaleString('fr-BE')}${c.unite ? ' ' + c.unite : ''}`
  : c.format ?? 'libre'
</script>

<template>
  <DocPage titre="Guide : Nouvelle campagne"
    intro="Chaque réglage de l'écran « Nouvelle campagne » : à quoi il sert, ce que change une hausse ou une baisse, et son effet prévu avec la vraie boucle. Dans l'écran, le bouton « ? » de chaque champ affiche la même aide."
    :sommaire="sommaire">
    <section v-for="s in ordre" :id="s" :key="s" class="card">
      <h2>{{ sections[s].titre }}</h2>
      <p class="muted">{{ sections[s].intro }}</p>
      <div v-for="c in champs.filter(x => x.section === s)" :key="c.cle" class="champ">
        <h3>{{ c.libelle }} <span class="small muted">· {{ plage(c) }}</span></h3>
        <p><strong>À quoi ça sert.</strong> {{ c.role }}</p>
        <p><strong>Effet d'un changement.</strong> {{ c.impact }}</p>
        <p v-if="c.boucleReelle"><strong>Avec la vraie boucle.</strong> {{ c.boucleReelle }}</p>
      </div>
    </section>
    <section id="repartir" class="card">
      <h2>Repartir d'une campagne finie</h2>
      <p>Depuis une campagne terminée ou arrêtée, « Repartir de cette campagne » ouvre l'écran avec ses réglages comme référence (un réglage changé est marqué « modifié ») et une zone « Point de départ » : la génération dont le front de Pareto fournira les parents de la génération 1, la finale par défaut. Une génération sans individu réalisable ne peut pas servir. Le départ entre dans la configuration figée et son empreinte ; les contrôles ajoutent la ligne « Point de départ ».</p>
    </section>
  </DocPage>
</template>

<style scoped>
.champ { border-top: 1px solid var(--border); padding-top: 10px; margin-top: 10px; }
.champ h3 { font-size: 1rem; margin-bottom: 6px; }
.champ p { margin-bottom: 6px; }
</style>
