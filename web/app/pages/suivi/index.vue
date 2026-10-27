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
    <section class="card intro-cadre">
      <p class="bonjour">Bonjour {{ session.user?.display_name || session.user?.username }}.</p>
      <p>
        MammoMesh est un outil de recherche qui cherche, de lui-même, la meilleure façon de repérer
        les signes de cancer du sein sur une mammographie.
      </p>
      <p>Il réunit deux parties :</p>
      <ul class="duo">
        <li>
          <strong>le <Sigle t="CNN" /></strong>, un modèle d'apprentissage automatique (une branche de l'intelligence artificielle) qui <em>regarde</em> l'image
          et repère ce qui pourrait être une lésion ;
        </li>
        <li>
          <strong>CephaloMesh</strong>, qui <em>raisonne</em> sur ces indices comme un radiologue :
          la lésion se voit-elle sur les deux vues ? Les deux seins se ressemblent-ils ? Était-elle déjà là au dernier examen ?
        </li>
      </ul>

      <h2 class="titre-exemples">Deux exemples complets</h2>
      <div class="exemples">
        <article class="exemple retenu">
          <h3>Une alerte confirmée</h3>
          <ol>
            <li><strong>Le <Sigle t="CNN" /> regarde</strong> : sur la vue de dessus (<Sigle t="CC" />) du sein gauche,
              il repère une petite zone dense, à quelques centimètres du mamelon. Il n'en est qu'à moitié sûr.</li>
            <li><strong>CephaloMesh vérifie</strong> :
              <ul>
                <li>une zone semblable apparaît au même endroit sur la vue de biais (<Sigle t="MLO" />) : <em>renforcée</em> ;</li>
                <li>rien de comparable dans le sein droit : <em>renforcée</em> ;</li>
                <li>elle n'existait pas sur l'examen d'il y a deux ans : <em>renforcée</em>.</li>
              </ul>
            </li>
            <li><strong>Second regard</strong> : le <Sigle t="CNN" /> réexamine la zone en pleine résolution et confirme des contours irréguliers.</li>
            <li><strong>Résultat</strong> : zone retenue, score de suspicion élevé, catégorie <Sigle t="BI-RADS" /> 4 suggérée (anomalie suspecte),
              avec les trois raisons ci-dessus. Le radiologue décide de la suite.</li>
          </ol>
        </article>
        <article class="exemple ecarte">
          <h3>Une fausse alerte écartée</h3>
          <ol>
            <li><strong>Le <Sigle t="CNN" /> regarde</strong> : sur la vue de biais (<Sigle t="MLO" />) du sein droit,
              il repère une tache claire. Là aussi, il n'en est qu'à moitié sûr.</li>
            <li><strong>CephaloMesh vérifie</strong> :
              <ul>
                <li>rien au même endroit sur la vue de dessus : sans doute une superposition de tissus, <em>affaiblie</em> ;</li>
                <li>une tache identique existe dans le sein gauche : <em>affaiblie</em> ;</li>
                <li>elle est inchangée depuis trois ans : <em>affaiblie</em>.</li>
              </ul>
            </li>
            <li><strong>Pas de second regard</strong> : la zone n'est plus douteuse, aucun calcul supplémentaire n'est dépensé.</li>
            <li><strong>Résultat</strong> : zone écartée, catégorie <Sigle t="BI-RADS" /> 1 suggérée (rien d'anormal),
              avec les raisons de cet écart. Le <Sigle t="CNN" /> seul aurait donné une fausse alerte.</li>
          </ol>
        </article>
      </div>

      <p class="small muted note">
        Exemples illustratifs : ils montrent le fonctionnement visé. Aujourd'hui, le <Sigle t="CNN" /> s'entraîne réellement et peut déjà
        noter une image (menu Images), mais CephaloMesh est encore simulé. Aucun résultat ne sert à poser un diagnostic.
        <NuxtLink to="/suivi/documentation/cnn-cephalomesh">En savoir plus sur le CNN et CephaloMesh</NuxtLink>.
      </p>
    </section>

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
        Les deux améliorent MammoMesh, mais pas la même partie. Un <strong>entraînement</strong> apprend au <Sigle t="CNN" /> à regarder :
        on lui montre des mammographies dont on connaît le diagnostic, et il se corrige à chaque erreur. Une <strong>campagne</strong>
        cherche la meilleure façon de raisonner sur ce qu'il voit : elle essaie de nombreuses variantes des réglages de CephaloMesh
        et garde celles qui trouvent le plus de cancers dans le moins de temps. On entraîne d'abord le <Sigle t="CNN" /> ; les campagnes s'appuient ensuite sur lui.
      </p>
      <div class="table-wrap"><table>
        <thead><tr><th /><th>Campagne</th><th>Entraînement</th></tr></thead>
        <tbody>
          <tr v-for="l in comparaison" :key="l.aspect"><td class="cle">{{ l.aspect }}</td><td><TexteGlossaire :texte="l.campagne" /></td><td><TexteGlossaire :texte="l.entrainement" /></td></tr>
        </tbody>
      </table></div>
    </section>

    <section class="card">
      <h2>Le parcours</h2>
      <ol class="parcours">
        <li><strong>Rassembler les images</strong> : des mammographies publiques, dont le diagnostic est connu, sont copiées sur le serveur
          (<NuxtLink to="/suivi/documentation/jeux-entrainement">jeux d'entraînement</NuxtLink>).</li>
        <li><strong>Entraîner le <Sigle t="CNN" /></strong> : menu <NuxtLink to="/suivi/entrainements">Entraînements</NuxtLink>, « Nouvel entraînement ».</li>
        <li><strong>Lancer une campagne</strong> : menu <NuxtLink to="/suivi/nouvelle">Nouvelle campagne</NuxtLink> ; les réglages sont vérifiés avant le lancement.</li>
        <li><strong>Suivre et comparer</strong> : menu <NuxtLink to="/suivi/campagnes">Campagnes</NuxtLink>, pour voir les meilleures versions trouvées.</li>
        <li><strong>Essayer sur une image</strong> : menu <NuxtLink to="/suivi/images">Images</NuxtLink>, pour tester une version sur une mammographie.</li>
      </ol>
    </section>

    <h2 class="titre-menus">Les menus</h2>
    <div class="grid menus">
      <section v-for="m in visibles" :id="`menu-${m.cle}`" :key="m.cle" class="card menu">
        <h3><NuxtLink :to="m.to">{{ m.titre }}</NuxtLink></h3>
        <p class="resume"><TexteGlossaire :texte="m.resume" /></p>
        <ul><li v-for="f in m.fonctionnement" :key="f"><TexteGlossaire :texte="f" /></li></ul>
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
.intro-cadre { border-left: 4px solid var(--accent); background: var(--accent-soft); display: grid; gap: 10px; }
.intro-cadre p { margin: 0; max-width: 900px; }
.intro-cadre .bonjour { font-weight: 600; font-size: 1.05rem; }
.duo { margin: 0; padding-left: 20px; display: grid; gap: 6px; max-width: 900px; }
.titre-exemples { margin: 8px 0 0; font-size: 1rem; }
.exemples { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(320px, 100%), 1fr)); gap: 12px; }
.exemple { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius); padding: 12px 14px; font-size: 0.92rem; }
.exemple h3 { margin: 0 0 8px; font-size: 0.98rem; }
.exemple.retenu h3 { color: var(--warn); }
.exemple.ecarte h3 { color: var(--accent); }
.exemple ol { margin: 0; padding-left: 20px; display: grid; gap: 6px; }
.exemple ul { margin: 4px 0 0; padding-left: 18px; }
.note { margin-top: 2px; }
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
