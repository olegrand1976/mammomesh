<script setup lang="ts">
// Guide de l'écran de suivi d'une campagne, pour un non-spécialiste : notions, lecture des graphiques, exemples.
import { exempleFront, exemplesCampagne } from '~/utils/progressionCampagne'
const fr2 = (x: number) => x.toLocaleString('fr-BE', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
</script>

<template>
  <details class="card guide">
    <summary>Comprendre cet écran : générations, sensibilité, compromis, avec des exemples</summary>

    <h3>Ce que fait une campagne</h3>
    <p>
      Une campagne cherche les meilleurs réglages de CephaloMesh, la partie de MammoMesh qui raisonne sur ce que le <Sigle t="CNN" /> a vu.
      Chaque combinaison de réglages est une <strong>version</strong>, appelée <strong>individu</strong> (ses réglages sont ses gènes).
      La campagne procède comme une sélection naturelle accélérée, par <strong>générations</strong> :
    </p>
    <ol>
      <li>elle part des meilleures versions trouvées jusque-là (les parents) ;</li>
      <li>elle en fabrique de nouvelles en changeant un réglage (<NuxtLink to="/suivi/documentation/mutations">mutation</NuxtLink>) ou en mélangeant deux parents (croisement) ;</li>
      <li>elle teste chaque version sur des mammographies dont le diagnostic est connu ;</li>
      <li>elle garde les meilleures, qui deviennent les parents de la génération suivante.</li>
    </ol>
    <p>Elle tourne seule, jour et nuit, jusqu'à ce qu'une condition d'arrêt soit atteinte : plus de progrès, temps de calcul épuisé, ou nombre maximal de générations.</p>

    <h3>Les mots de l'écran</h3>
    <ul>
      <li><strong>Sensibilité</strong> : la part des cancers trouvés. 0,88 = sur 100 cancers, 88 trouvés et 12 manqués. Plus haut, c'est mieux.</li>
      <li><strong>Coût</strong> : le temps de calcul par examen, en millisecondes. Plus bas, c'est mieux.</li>
      <li><strong>Minimum exigé</strong> (« plancher ») : la sensibilité en dessous de laquelle une version est écartée d'office, même si elle est très rapide.
        Une version qui l'atteint est dite <strong>réalisable</strong>.</li>
      <li><strong>Deux groupes d'images</strong> : la recherche choisit ses versions sur un premier groupe (<Sigle t="E-ajust" />), puis on les mesure sur un second,
        qu'elle n'a jamais utilisé pour choisir (<Sigle t="E-val" />, les images de contrôle). Comme un élève qui révise sur des exercices puis passe un examen
        avec d'autres questions : seule la note de l'examen (<Sigle t="E-val" />) dit ce qu'il sait vraiment.</li>
    </ul>

    <h3>Le compromis entre sûreté et rapidité (front de Pareto)</h3>
    <p>
      Une version plus sûre est souvent plus lente. Il n'y a donc pas une « meilleure » version, mais une famille de bons compromis :
      celles qu'aucune autre ne bat sur les deux points à la fois. C'est le <strong>front de Pareto</strong>, l'escalier bleu du graphique de droite.
    </p>
    <div class="table-wrap"><table class="small">
      <thead><tr><th>Version</th><th>Sensibilité</th><th>Coût par examen</th><th>Dans le front ?</th></tr></thead>
      <tbody>
        <tr v-for="v in exempleFront" :key="v.nom" :class="{ hors: !v.front }">
          <td>{{ v.nom }}</td><td>{{ fr2(v.sensibilite) }}</td><td>{{ v.cout.toLocaleString('fr-BE') }} ms</td><td>{{ v.front ? 'Oui' : 'Non' }} — {{ v.pourquoi }}</td>
        </tr>
      </tbody>
    </table></div>

    <h3>Lire l'évolution, génération après génération</h3>
    <p>
      Le graphique de gauche suit la meilleure sensibilité en contrôle (<Sigle t="E-val" />) à chaque génération. Le plus souvent : progrès rapides au début,
      puis de plus en plus lents, puis un palier. Un point de sensibilité en plus, c'est un cancer de plus trouvé sur 100.
      Sous les graphiques, l'encadré « Lecture » fait cette analyse sur les vrais chiffres de la campagne.
    </p>
    <p class="small muted">Exemples ci-dessous : chiffres illustratifs.</p>
    <div class="exemples">
      <article v-for="e in exemplesCampagne" :key="e.titre" class="exemple">
        <h4>{{ e.titre }}</h4>
        <CourbeCampagne :points="e.points" />
        <p class="small">{{ e.lecture }}</p>
      </article>
    </div>

    <h3>Les zones de l'écran</h3>
    <ol>
      <li><strong>Bandeau</strong> : état, commandes et chiffres clés (génération, temps de calcul, meilleure sensibilité, fin estimée).</li>
      <li><strong>Progrès par génération</strong> : la meilleure version et la moyenne de la population progressent-elles ?</li>
      <li><strong>Front de Pareto</strong> : les meilleurs compromis ; un clic sur un point ouvre la fiche de la version.</li>
      <li><strong>Fiche d'un individu</strong> : ses résultats et la liste des mutations qui l'ont produit.</li>
      <li><strong>Journal</strong> : chaque événement en une phrase, le plus récent en tête.</li>
    </ol>
    <p class="small muted">
      Chaque zone a son bouton <span class="q">?</span> pour le détail. Tout se met à jour en direct.
      Pour l'instant, les mesures des campagnes sont simulées (sans valeur) : le <Sigle t="CNN" /> entraîné n'y est pas encore branché.
    </p>
  </details>
</template>

<style scoped>
.guide { margin-bottom: 14px; }
.guide summary { cursor: pointer; font-weight: 600; }
.guide h3 { margin: 18px 0 6px; font-size: 1rem; }
.guide p, .guide ul, .guide ol { margin: 8px 0 0; }
.guide ul, .guide ol { padding-left: 20px; display: grid; gap: 4px; }
tr.hors td { color: var(--muted); }
.exemples { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(300px, 100%), 1fr)); gap: 12px; margin-top: 8px; }
.exemple { border: 1px solid var(--border); border-radius: var(--radius); padding: 10px 12px; }
.exemple h4 { margin: 0 0 4px; font-size: 0.95rem; }
.q { display: inline-grid; place-items: center; width: 18px; height: 18px; border-radius: 50%; border: 1px solid var(--border); font-size: 0.7rem; font-weight: 600; }
</style>
