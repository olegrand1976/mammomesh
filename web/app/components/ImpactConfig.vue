<script setup lang="ts">
// Effets calculés des réglages, mis à jour à chaque saisie (mêmes formules que l'API).
import type { CampaignConfig } from '~/types'
import { effets } from '~/utils/guideCampagne'
import { fr } from '~/types'

const props = defineProps<{ config: CampaignConfig }>()
const e = computed(() => effets(props.config))
const ev = computed(() => props.config.evolution)
const duree = (s: number) => s < 60 ? `${fr(s, s < 10 ? 1 : 0)} s` : s < 3600 ? `${fr(s / 60, 1)} min` : `${fr(s / 3600, 1)} h`

const alertes = computed(() => {
  const out: string[] = []
  const pr = e.value.premiersRealisables
  if (pr === null) {
    out.push(`Seuil ${fr(e.value.seuil, 2)} hors d'atteinte dans la simulation : aucun individu réalisable, arrêt par stagnation à la génération ${ev.value.arret_stagnation} sans résultat. Baissez le plancher ou augmentez la marge.`)
  } else if (pr >= ev.value.arret_stagnation) {
    out.push(`Premiers individus réalisables vers la génération ${pr} au plus tôt, mais l'arrêt par stagnation tombe à la génération ${ev.value.arret_stagnation} : la campagne risque de s'arrêter sans résultat. Montez « Arrêt si E-val stable » au-delà de ${pr}, ou assouplissez le plancher.`)
  }
  if (e.value.arretBudget !== null) {
    out.push(`Le budget (${fr(ev.value.budget_coeur_heures, 1)} cœur-h) arrêtera la campagne après la génération ${e.value.arretBudget}, avant le plafond de ${ev.value.generations_max}.`)
  }
  return out
})
</script>

<template>
  <section class="card impact" aria-labelledby="impact-titre">
    <h2 id="impact-titre">Effet de vos réglages</h2>
    <p class="muted small">Recalculé à chaque saisie. Les durées sont celles de la simulation ; les arrêts par stagnation ou budget peuvent survenir plus tôt.</p>
    <dl aria-live="polite">
      <div><dt>Seuil de réalisabilité</dt><dd>{{ fr(e.seuil, 2) }} <span class="muted">plancher {{ fr(config.projet.plancher_sensibilite, 2) }} − {{ fr(config.projet.marge_points, 1) }} point(s)</span></dd></div>
      <div><dt>Premiers réalisables (simulation)</dt><dd>{{ e.premiersRealisables === null ? 'jamais' : `génération ${e.premiersRealisables} au plus tôt` }}</dd></div>
      <div><dt>Une génération</dt><dd>{{ duree(e.dureeGenerationS) }} · {{ fr(e.coutGeneration, 2) }} cœur-h</dd></div>
      <div><dt>Campagne entière, au plus</dt><dd>{{ duree(e.dureeMaxMin * 60) }} · {{ fr(e.coutMax, 1) }} cœur-h <span class="muted">({{ ev.generations_max }} générations)</span></dd></div>
      <div><dt>Mutations par enfant</dt><dd>{{ fr(e.mutations, 2) }} en moyenne</dd></div>
    </dl>
    <ul v-if="alertes.length" class="alertes">
      <li v-for="a in alertes" :key="a">{{ a }}</li>
    </ul>
  </section>
</template>

<style scoped>
dl { display: grid; grid-template-columns: repeat(auto-fit, minmax(210px, 1fr)); gap: 10px 18px; margin: 0; }
dt { font-size: 0.78rem; color: var(--muted); }
dd { margin: 2px 0 0; font-variant-numeric: tabular-nums; }
dd .muted { font-size: 0.8rem; }
.alertes { margin: 14px 0 0; padding: 0; list-style: none; display: grid; gap: 8px; }
.alertes li { padding: 9px 12px; border-radius: 8px; background: color-mix(in srgb, var(--warn) 14%, transparent); font-size: 0.88rem; }
</style>
