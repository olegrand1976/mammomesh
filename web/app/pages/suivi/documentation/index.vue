<script setup lang="ts">
definePageMeta({ layout: 'suivi', middleware: 'auth' })
useHead({ title: 'Documentation · MammoMesh' })

const pages = [
  { to: '/suivi/documentation/fonctionnement', titre: 'Fonctionnement d\'une campagne', texte: 'Le processus de bout en bout : acteurs, cycle de vie, une génération pas à pas, E-ajust et E-val, arrêts, reprise, temps réel.' },
  { to: '/suivi/documentation/mathematiques', titre: 'Modèle mathématique', texte: 'Toutes les formules : coût et budget, mutations, simulation, réalisabilité, front de Pareto, stagnation, indicateurs, images, et le modèle prévu pour la vraie boucle.' },
  { to: '/suivi/documentation/nouvelle-campagne', titre: 'Guide : Nouvelle campagne', texte: 'Chaque réglage, à quoi il sert et ce que change une hausse ou une baisse ; repartir d\'une campagne finie.' },
  { to: '/suivi/documentation/suivi-campagne', titre: 'Guide : suivi d\'une campagne', texte: 'Chaque zone de l\'écran de suivi et ses éléments.' },
  { to: '/suivi/documentation/images', titre: 'Guide : Images', texte: 'Import pseudonymisé, visionneuse, test d\'un individu sur une mammographie.' },
  { to: '/suivi/documentation/mutations', titre: 'Lexique des mutations', texte: 'Chaque code de mutation : gène modifié, rôle, effet.' },
  { to: '/suivi/documentation/glossaire', titre: 'Glossaire', texte: 'Tous les sigles et mots techniques expliqués ; les mêmes explications s\'affichent au survol dans chaque écran.' },
  { to: '/suivi/documentation/installation-windows', titre: 'Installation sur Windows 11', texte: 'Faire tourner la pile sur un poste Windows 11 : WSL (Debian), Docker, GPU NVIDIA ; procédure téléchargeable en PDF.' },
  { to: '/suivi/documentation/jeux-entrainement', titre: 'Jeux d\'entraînement', texte: 'Mammographies publiques retenues, à demander ou écartées : contenu, volume, licence, téléchargement, citations et sources.' },
]
</script>

<template>
  <div class="doc">
    <h1>Documentation</h1>
    <p class="muted">Outil de recherche : aucun résultat ne sert à un diagnostic. Le contenu de ces pages se trouve aussi dans le dossier <code>docs/</code> du dépôt.</p>
    <div class="grid cartes">
      <NuxtLink v-for="p in pages" :key="p.to" :to="p.to" class="card carte">
        <h2>{{ p.titre }}</h2>
        <p class="small muted">{{ p.texte }}</p>
      </NuxtLink>
    </div>
  </div>
</template>

<style scoped>
.doc { max-width: 960px; }
.doc h1 { font-size: 1.4rem; }
.cartes { grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); }
.carte { color: var(--text); }
.carte:hover { text-decoration: none; border-color: var(--accent); }
.carte h2 { color: var(--accent); }
.carte p { margin: 0; }
</style>
