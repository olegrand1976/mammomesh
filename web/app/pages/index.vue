<script setup lang="ts">
// Page de présentation publique : HTML statique pré-rendu, aucune donnée de santé, aucun appel à l'API.
// Écrite pour un lecteur non spécialiste ; état du projet tenu à jour avec le README et docs/jeux-entrainement.md.
useHead({ title: 'MammoMesh · présentation' })

const parties = [
  { t: 'Le CNN : l\'œil', d: 'Un modèle d\'apprentissage automatique (une branche de l\'intelligence artificielle) conçu pour les images. Il a appris, sur des milliers de mammographies dont le diagnostic est connu, à repérer ce qui ressemble à une lésion. Très fin, très rapide, mais il juge chaque image séparément.' },
  { t: 'CephaloMesh : le raisonnement', d: 'Il reprend ce que le CNN a repéré et le recoupe comme un radiologue : la lésion se voit-elle sur les deux vues du même sein ? Les deux seins se ressemblent-ils ? Était-elle déjà là au dernier examen ? Chaque étape est traçable.' },
  { t: 'Ensemble', d: 'Le CNN apporte la vue, CephaloMesh le jugement. Moins de fausses alertes (un défaut sur une seule image ne suffit plus), des indices faibles qui se confirment entre eux, et un examen détaillé seulement là où c\'est utile.' },
]

const steps = [
  { n: '1', t: 'Premier coup d\'œil', d: 'Le CNN regarde les quatre images de l\'examen (deux vues par sein) en basse résolution et signale les zones qui l\'interpellent.' },
  { n: '2', t: 'Mise en carte', d: 'Chaque zone signalée est placée sur une carte du sein, une sorte de cible centrée sur le mamelon.' },
  { n: '3', t: 'Vérifications', d: 'CephaloMesh recoupe : les deux vues, les deux seins, l\'examen précédent, la cohérence de la description (vocabulaire BI-RADS des radiologues).' },
  { n: '4', t: 'Second regard', d: 'Les zones encore douteuses sont réexaminées par le CNN en pleine résolution, puis le raisonnement est refait avec ces résultats.' },
]

const etat = [
  { t: 'Fait', d: 'L\'interface de suivi en direct ; un premier CNN (EfficientNet-B0) qui s\'entraîne réellement sur la carte graphique du serveur, à partir des mammographies CMMD ; l\'import et l\'affichage de mammographies, sans le nom ni l\'identifiant de la patiente.' },
  { t: 'En rodage', d: 'Les campagnes d\'évolution tournent déjà, mais leurs mesures sont simulées, donc sans valeur, tant que le CNN entraîné n\'y est pas branché.' },
  { t: 'À venir', d: 'Constituer toute la banque d\'images, brancher le CNN sur CephaloMesh, puis lancer les premières vraies campagnes.' },
]

const guards = [
  { t: 'Pas d\'examen sur des images déjà vues', d: 'CephaloMesh n\'est jamais réglé à partir de résultats que le CNN a obtenus sur ses propres images d\'apprentissage.' },
  { t: 'Une patiente, un seul groupe', d: 'Les images d\'une même patiente restent toujours dans le même groupe (apprentissage, sélection ou test final) : le test porte vraiment sur des inconnues.' },
  { t: 'Un minimum de cancers trouvés', d: 'Fixé et justifié avant chaque campagne. Une version qui trouve moins de cancers n\'est jamais retenue, même si elle est plus rapide.' },
  { t: 'Comparaison équitable', d: 'MammoMesh est comparé à des méthodes plus simples, comme le CNN seul, qui reçoivent le même effort de réglage.' },
  { t: 'Un seul test final', d: 'Le critère de réussite est écrit à l\'avance ; le jeu de test caché ne sert qu\'une fois, puis les résultats sont vérifiés sur des données d\'un autre centre.' },
  { t: 'Décision humaine', d: 'L\'évolution propose des candidats ; seule une personne habilitée peut en retenir un.' },
]
</script>

<template>
  <main>
    <section class="hero">
      <div class="container">
        <p class="kicker">Projet de recherche · dépistage du cancer du sein</p>
        <h1>Une intelligence artificielle qui regarde, une autre qui raisonne, et une évolution qui les accorde</h1>
        <p class="lead">
          MammoMesh cherche, de lui-même, la meilleure façon de repérer les signes de cancer du sein sur une mammographie.
          Il associe deux parties : <strong>le <Sigle t="CNN" sans-lien /></strong>, qui regarde l'image et repère ce qui pourrait être une lésion,
          et <strong>CephaloMesh</strong>, qui raisonne sur ces indices comme un radiologue qui compare les vues, les deux seins et l'examen précédent.
        </p>
        <p class="status">Outil de recherche : aucun résultat ne sert à un diagnostic. Un usage à l'hôpital exigerait le marquage <Sigle t="CE" sans-lien /> (règlement (UE) 2017/745) et une étude clinique dédiée.</p>
      </div>
    </section>

    <section class="container block">
      <h2>Deux parties complémentaires</h2>
      <div class="steps">
        <article v-for="p in parties" :key="p.t" class="card">
          <h3>{{ p.t }}</h3>
          <p><TexteGlossaire :texte="p.d" sans-lien /></p>
        </article>
      </div>
    </section>

    <section class="container block">
      <h2>Le parcours d'un examen</h2>
      <div class="steps">
        <article v-for="s in steps" :key="s.n" class="card">
          <span class="num">{{ s.n }}</span>
          <h3>{{ s.t }}</h3>
          <p><TexteGlossaire :texte="s.d" sans-lien /></p>
        </article>
      </div>
      <p class="muted">Résultat : un score par zone et par sein, une catégorie <Sigle t="BI-RADS" sans-lien /> suggérée et les raisons de chaque décision. C'est une suggestion pour un radiologue, jamais une décision.</p>
    </section>

    <section class="container block two">
      <div>
        <h2>Ce qui évolue</h2>
        <p>
          Le couple a beaucoup de réglages : à partir de quand s'alerter, quelles vérifications faire, combien de seconds regards demander,
          comment est construit le <Sigle t="CNN" sans-lien />… Impossible de tout essayer à la main.
        </p>
        <p>
          MammoMesh procède comme une sélection naturelle accélérée : il crée des variantes du couple en changeant un réglage à la fois,
          les mesure, et garde celles qui trouvent le plus de cancers dans le moins de temps. Une variante qui en rate trop est écartée d'office.
        </p>
        <ul>
          <li><strong>CephaloMesh évolue vite</strong> : il se teste en quelques dixièmes de milliseconde par examen.</li>
          <li><strong>Le <Sigle t="CNN" sans-lien /> évolue lentement</strong> : chaque changement de structure demande de le réentraîner.</li>
        </ul>
      </div>
      <div class="card">
        <h3>Les images utilisées</h3>
        <p class="small">Des mammographies publiques dont le diagnostic est connu, sous licence libre (y compris pour un usage commercial) :</p>
        <ul class="small">
          <li><Sigle t="CMMD" sans-lien /> (Chine) : 1 775 patientes, lésion bénigne ou maligne prouvée par biopsie ; sert au premier <Sigle t="CNN" sans-lien />.</li>
          <li><Sigle t="CBIS-DDSM" sans-lien /> (États-Unis) : films numérisés avec le contour des lésions, pour apprendre à les localiser.</li>
          <li><Sigle t="NLBS" sans-lien /> (Canada) : un vrai programme de dépistage, avec peu de cancers et beaucoup d'examens normaux.</li>
          <li>Demandés : <Sigle t="CSAW-CC" sans-lien /> (Suède, historique de dépistage) et VinDr-Mammo (Vietnam).</li>
        </ul>
        <p class="small muted">Un résultat ne vaut que pour les populations et les appareils testés.</p>
      </div>
    </section>

    <section class="container block">
      <h2>Où en est le projet</h2>
      <div class="steps">
        <article v-for="e in etat" :key="e.t" class="card">
          <h3>{{ e.t }}</h3>
          <p><TexteGlossaire :texte="e.d" sans-lien /></p>
        </article>
      </div>
    </section>

    <section class="container block">
      <h2>Garde-fous</h2>
      <div class="guards">
        <article v-for="g in guards" :key="g.t">
          <h3>{{ g.t }}</h3>
          <p><TexteGlossaire :texte="g.d" sans-lien /></p>
        </article>
      </div>
      <p class="muted small">Résultats rapportés selon les recommandations <Sigle t="CLAIM" sans-lien /> 2024 et <Sigle t="TRIPOD" sans-lien />+AI. Données pseudonymisées (nom et identifiant supprimés), traitées sur le réseau local, avec avis éthique et analyse d'impact avant tout import.</p>
    </section>

    <section class="container block">
      <div class="card cta row">
        <div>
          <h3>Équipe du projet</h3>
          <p class="muted small">L'interface de suivi montre les entraînements et les campagnes en direct, avec une documentation complète. Accès sur le réseau local, avec un compte nominatif.</p>
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
