<script setup lang="ts">
// Page de présentation scientifique : HTML statique pré-rendu, aucune donnée de santé, aucun appel à l'API.
useHead({ title: 'MammoMesh · présentation scientifique' })

const steps = [
  { n: '1', t: 'Passe rapide', d: 'Le CNN analyse les 4 vues en résolution réduite et signale les zones suspectes.' },
  { n: '2', t: 'Onde CephaloMesh', d: 'Les détections deviennent des impulsions dans un cube par sein : position autour du mamelon, distance, type de constat.' },
  { n: '3', t: 'Contrôles', d: 'Concordance des vues, symétrie gauche-droite, évolution depuis l\'examen précédent, cohérence BI-RADS.' },
  { n: '4', t: 'Réanalyse ciblée', d: 'Les zones retenues sont réanalysées par le CNN en pleine résolution, puis une seconde onde intègre ces résultats.' },
]

const guards = [
  { t: 'Sorties hors échantillon', d: 'CephaloMesh n\'est jamais réglé sur des sorties du CNN calculées sur ses propres images d\'entraînement.' },
  { t: 'Jeux séparés par patiente', d: 'Évolution, sélection et test caché : une patiente reste dans un seul jeu, pour toujours.' },
  { t: 'Plancher de sensibilité', d: 'Fixé et sourcé avant chaque campagne ; un candidat sous le plancher n\'est jamais retenu.' },
  { t: 'Références à budget égal', d: 'CNN seul, cascade apprise, raisonnement multi-vues appris, fusion simple : toutes reçoivent le même effort de réglage.' },
  { t: 'Un seul test confirmatoire', d: 'Critère principal écrit d\'avance, test caché lu une fois, puis jeu externe dans la population cible.' },
  { t: 'Décision humaine', d: 'L\'évolution produit des candidats ; seule une personne habilitée peut en promouvoir un.' },
]
</script>

<template>
  <main>
    <section class="hero">
      <div class="container">
        <p class="kicker">Projet de recherche · mammographie de dépistage</p>
        <h1>Un réseau qui voit, un réseau qui raisonne, et une évolution qui les accorde</h1>
        <p class="lead">
          MammoMesh fait évoluer, par algorithme génétique et recherche d'architecture, un couple
          <strong>CNN + CephaloMesh</strong> qui repère les signes de cancer du sein sur mammographie.
          Le CNN voit les lésions possibles ; CephaloMesh raisonne sur ce qui a été vu, comme le fait un radiologue
          qui compare les vues, les deux seins et l'examen précédent.
        </p>
        <p class="status">Outil de recherche : aucun résultat ne sert à un diagnostic. Un usage clinique exigerait le marquage CE (règlement (UE) 2017/745) et une étude dédiée.</p>
      </div>
    </section>

    <section class="container block">
      <h2>Le cycle d'un examen</h2>
      <div class="steps">
        <article v-for="s in steps" :key="s.n" class="card">
          <span class="num">{{ s.n }}</span>
          <h3>{{ s.t }}</h3>
          <p>{{ s.d }}</p>
        </article>
      </div>
      <p class="muted">Sortie : un score par zone et par sein, une catégorie BI-RADS suggérée et le récit de l'onde, qui dit pourquoi chaque zone est retenue ou écartée. C'est une suggestion à un radiologue, jamais une décision.</p>
    </section>

    <section class="container block two">
      <div>
        <h2>Ce qui évolue</h2>
        <p>Chaque individu est un couple décrit par un génome à deux chromosomes, qui évoluent à deux vitesses.</p>
        <ul>
          <li><strong>CNN, lentement</strong> : étages dégelés, cou, blocs ajoutés à l'identique, tête, résolutions. Le tronc pré-entraîné ne mute pas.</li>
          <li><strong>CephaloMesh, rapidement</strong> : seuils, portée des liens, contrôles actifs, budget de réanalyse, découpage du cube. Quelques dixièmes de milliseconde par examen.</li>
        </ul>
        <p>La sélection cherche le meilleur compromis entre sensibilité au taux de rappel du programme et temps par examen, sous un plancher de sensibilité non négociable.</p>
      </div>
      <div class="card">
        <h3>Données prévues</h3>
        <ul class="small">
          <li>EMBED (États-Unis) : 2D, 2D synthétique, tomosynthèse, lésions liées à la pathologie</li>
          <li>OPTIMAM (Royaume-Uni) : séries de dépistage avec cancers d'intervalle</li>
          <li>VinDr-Mammo (Vietnam) : entraînement seulement, sans pathologie</li>
          <li>Centre partenaire belge : jeu externe, population cible</li>
        </ul>
        <p class="small muted">Un résultat ne vaut que pour les populations et les appareils testés.</p>
      </div>
    </section>

    <section class="container block">
      <h2>Garde-fous méthodologiques</h2>
      <div class="guards">
        <article v-for="g in guards" :key="g.t">
          <h3>{{ g.t }}</h3>
          <p>{{ g.d }}</p>
        </article>
      </div>
      <p class="muted small">Comptes rendus conformes à CLAIM 2024 et TRIPOD+AI. Données pseudonymisées, traitées sur le réseau local, avec avis éthique et analyse d'impact avant tout import.</p>
    </section>

    <section class="container block">
      <div class="card cta row">
        <div>
          <h3>Équipe du projet</h3>
          <p class="muted small">L'interface de suivi montre l'évolution en direct. Accès sur le réseau local, avec un compte nominatif.</p>
        </div>
        <div class="spacer" />
        <NuxtLink to="/login" class="btn btn-primary">Se connecter</NuxtLink>
      </div>
    </section>
  </main>
</template>

<style scoped>
.hero { background: var(--surface); border-bottom: 1px solid var(--border); padding: 56px 0 44px; }
.kicker { color: var(--accent); font-weight: 600; font-size: 0.85rem; letter-spacing: 0.02em; }
.hero h1 { font-size: clamp(1.6rem, 3.4vw, 2.4rem); max-width: 820px; }
.lead { font-size: 1.1rem; max-width: 760px; color: var(--muted); }
.lead strong { color: var(--text); }
.status { display: inline-block; margin-top: 8px; padding: 8px 12px; border-radius: 8px; background: var(--surface-2); font-size: 0.88rem; }
.block { margin-top: 44px; }
.block h2 { font-size: 1.35rem; margin-bottom: 16px; }
.steps { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 14px; margin-bottom: 14px; }
.steps h3 { font-size: 1rem; margin-top: 8px; }
.steps p { color: var(--muted); font-size: 0.92rem; margin: 0; }
.num { display: inline-grid; place-items: center; width: 28px; height: 28px; border-radius: 50%; background: var(--accent-soft); color: var(--accent); font-weight: 600; }
.two { display: grid; grid-template-columns: 1.4fr 1fr; gap: 24px; align-items: start; }
@media (max-width: 760px) { .two { grid-template-columns: 1fr; } }
ul { padding-left: 20px; }
li { margin-bottom: 6px; }
.guards { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 4px 28px; }
.guards h3 { font-size: 0.98rem; margin: 10px 0 4px; }
.guards p { color: var(--muted); font-size: 0.92rem; }
.cta h3 { margin-bottom: 4px; }
.cta p { margin: 0; }
</style>
