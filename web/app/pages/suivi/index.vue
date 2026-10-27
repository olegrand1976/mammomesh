<script setup lang="ts">
// Accueil de l'interface connectée : à quoi sert chaque menu, comment il marche, où aller ensuite.
import type { Campaign } from '~/types'
import { comparaison, menus } from '~/utils/accueil'
import { estFinalise, type ResumeEntrainement } from '~/utils/entrainements'

definePageMeta({ layout: 'suivi', middleware: 'auth' })
useHead({ title: 'Accueil · MammoMesh' })

const { api } = useApi()
const session = useSessionStore()
const campagnesEnCours = ref<number | null>(null)
const entrainementsEnCours = ref<number | null>(null)

const visibles = computed(() => menus.filter(m => !m.permission || session.permissions?.[m.permission]))

onMounted(async () => {
  // Chiffres du moment, sans bloquer la page si un service ne répond pas.
  api<{ campaigns: Campaign[] }>('/campaigns')
    .then(r => { campagnesEnCours.value = r.campaigns.filter(c => c.state === 'en_cours' || c.state === 'en_pause').length })
    .catch(() => {})
  api<{ entrainements: ResumeEntrainement[] }>('/entrainements')
    .then(r => { entrainementsEnCours.value = r.entrainements.filter(e => !estFinalise(e.statut)).length })
    .catch(() => {})
})

const pluriel = (n: number, mot: string) => `${n} ${mot}${n > 1 ? 's' : ''}`
</script>

<template>
  <div class="accueil">
    <h1>Accueil</h1>
    <p class="intro">
      Bonjour {{ session.user?.display_name || session.user?.username }}. MammoMesh cherche, de façon autonome, un couple
      <strong>CNN + CephaloMesh</strong> qui repère les signes de cancer du sein sur mammographie : le CNN <em>voit</em> les lésions
      possibles, CephaloMesh <em>raisonne</em> sur ce qui a été vu (concordance des vues, symétrie, évolution, cohérence BI-RADS).
      Outil de recherche : aucun résultat ne sert à un diagnostic.
    </p>

    <div class="grid etat">
      <NuxtLink to="/suivi/campagnes" class="card chiffre">
        <span class="valeur">{{ campagnesEnCours ?? '—' }}</span>
        <span class="small muted">{{ campagnesEnCours === null ? 'campagnes' : pluriel(campagnesEnCours, 'campagne') }} en cours ou en pause</span>
      </NuxtLink>
      <NuxtLink to="/suivi/entrainements" class="card chiffre">
        <span class="valeur">{{ entrainementsEnCours ?? '—' }}</span>
        <span class="small muted">{{ entrainementsEnCours === null ? 'entraînements' : pluriel(entrainementsEnCours, 'entraînement') }} en cours ou en attente</span>
      </NuxtLink>
    </div>

    <section id="campagne-ou-entrainement" class="card">
      <h2>Campagne ou entraînement ?</h2>
      <p>
        Les deux améliorent MammoMesh, mais pas la même moitié. Un <strong>entraînement</strong> apprend au CNN à regarder :
        il ajuste les poids du réseau sur des mammographies dont on connaît la réponse. Une <strong>campagne</strong> cherche
        comment raisonner sur ce regard : elle fait évoluer une population de réglages de CephaloMesh et garde les meilleurs
        compromis entre sensibilité et temps de calcul. On entraîne d'abord le CNN ; les campagnes s'appuieront ensuite sur lui.
      </p>
      <div class="table-wrap"><table>
        <thead><tr><th /><th>Campagne</th><th>Entraînement</th></tr></thead>
        <tbody>
          <tr v-for="l in comparaison" :key="l.aspect"><td class="cle">{{ l.aspect }}</td><td>{{ l.campagne }}</td><td>{{ l.entrainement }}</td></tr>
        </tbody>
      </table></div>
    </section>

    <section class="card">
      <h2>Le parcours</h2>
      <ol class="parcours">
        <li><strong>Banque d'images</strong> : les mammographies publiques sont téléchargées sur le serveur
          (<NuxtLink to="/suivi/documentation/jeux-entrainement">jeux d'entraînement</NuxtLink>).</li>
        <li><strong>Entraîner le CNN</strong> : menu <NuxtLink to="/suivi/entrainements">Entraînements</NuxtLink>, « Nouvel entraînement ».</li>
        <li><strong>Lancer une campagne</strong> : menu <NuxtLink to="/suivi/nouvelle">Nouvelle campagne</NuxtLink>, réglages vérifiés avant lancement.</li>
        <li><strong>Suivre et comparer</strong> : menu <NuxtLink to="/suivi/campagnes">Campagnes</NuxtLink>, front de Pareto et fiches des individus.</li>
        <li><strong>Essayer sur une image</strong> : menu <NuxtLink to="/suivi/images">Images</NuxtLink>, test d'un individu sur une mammographie.</li>
      </ol>
    </section>

    <h2 class="titre-menus">Les menus</h2>
    <div class="grid menus">
      <section v-for="m in visibles" :id="`menu-${m.cle}`" :key="m.cle" class="card menu">
        <h3><NuxtLink :to="m.to">{{ m.titre }}</NuxtLink></h3>
        <p class="resume">{{ m.resume }}</p>
        <ul><li v-for="f in m.fonctionnement" :key="f">{{ f }}</li></ul>
        <p class="liens small">
          <NuxtLink :to="m.to" class="btn">Ouvrir</NuxtLink>
          <NuxtLink v-for="l in m.liens" :key="l.to" :to="l.to">{{ l.titre }}</NuxtLink>
        </p>
      </section>
    </div>
  </div>
</template>

<style scoped>
.accueil { display: grid; grid-template-columns: minmax(0, 1fr); gap: 16px; }
.accueil h1 { margin: 0; font-size: 1.4rem; }
.intro { margin: 0; max-width: 900px; }
.etat { grid-template-columns: repeat(auto-fit, minmax(min(220px, 100%), 1fr)); gap: 12px; }
.chiffre { display: grid; gap: 2px; color: var(--text); }
.chiffre:hover { text-decoration: none; border-color: var(--accent); }
.valeur { font-size: 1.6rem; font-weight: 600; }
td.cle { font-weight: 600; white-space: nowrap; vertical-align: top; }
td { vertical-align: top; }
.parcours { padding-left: 20px; display: grid; gap: 6px; margin: 0; }
.titre-menus { margin: 8px 0 0; }
.menus { grid-template-columns: repeat(auto-fit, minmax(min(300px, 100%), 1fr)); gap: 12px; }
.menu h3 { margin: 0 0 6px; font-size: 1.05rem; }
.menu .resume { margin: 0 0 8px; }
.menu ul { padding-left: 18px; margin: 0 0 10px; display: grid; gap: 4px; font-size: 0.9rem; }
.liens { display: flex; flex-wrap: wrap; gap: 10px; align-items: center; margin: 0; }
</style>
