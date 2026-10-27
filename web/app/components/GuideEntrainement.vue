<script setup lang="ts">
// Guide de lecture de la page Entraînements : CNN, AUC, perte, évolution au fil des époques, exemples.
import { exemples, lireProgression } from '~/utils/progression'
const lus = exemples.map(e => ({ ...e, lu: lireProgression(e.journal)! }))
const fr3 = (x: number) => x.toLocaleString('fr-BE', { minimumFractionDigits: 3, maximumFractionDigits: 3 })
const gain = (j: { auc_sein: number | null }[], i: number) => (i === 0 ? '' : ((j[i]!.auc_sein! - j[i - 1]!.auc_sein!) >= 0 ? '+' : '−') + fr3(Math.abs(j[i]!.auc_sein! - j[i - 1]!.auc_sein!)))
</script>

<template>
  <details class="card guide">
    <summary>Comprendre cette page : le CNN, l'AUC, les époques, avec des exemples</summary>

    <h3>Le CNN et son apprentissage</h3>
    <p>
      Un <strong><Sigle t="CNN" /></strong> (réseau de neurones convolutif) est un modèle d'apprentissage automatique, une branche de l'intelligence artificielle, conçu pour les images :
      un empilement de couches de calcul qui reconnaissent des motifs de plus en plus complexes. Ses millions de réglages internes (les poids)
      ne sont pas écrits à la main : on lui présente des mammographies dont on connaît la réponse, et l'entraînement corrige ses poids
      pour qu'il se trompe de moins en moins. Dans MammoMesh, il est le <strong>regard</strong> ; CephaloMesh, ensuite, <strong>raisonne</strong> sur ce qu'il a vu.
    </p>
    <p>
      Le <Sigle t="CNN" /> entraîné ici (EfficientNet-B0, qui sait déjà reconnaître des photos ordinaires) apprend, sur la banque <Sigle t="CMMD" />,
      une tâche précise : face à un sein qui porte une anomalie, dire si elle est plutôt <strong>maligne ou bénigne</strong> (la réponse est connue par biopsie).
      C'est une tâche difficile, même pour un radiologue : toutes les images montrent déjà quelque chose d'anormal.
    </p>
    <p>Les patientes sont réparties une fois pour toutes en trois groupes, sans jamais mélanger les images d'une même patiente :</p>
    <ul>
      <li><strong>Entraînement</strong> (70 %) : les images sur lesquelles le <Sigle t="CNN" /> apprend et se corrige.</li>
      <li><strong>Ajustement</strong> (<Sigle t="E-ajust" />, 15 %) : des images qu'il ne voit jamais pendant l'apprentissage, mesurées après chaque époque pour suivre ses vrais progrès. C'est la courbe du graphique.</li>
      <li><strong>Validation</strong> (<Sigle t="E-val" />, 15 %) : des patientes jamais vues, mesurées une seule fois, à la fin. C'est la note finale.</li>
    </ul>

    <h3>L'<Sigle t="AUC" />, expliquée simplement</h3>
    <p>
      Imaginez qu'on tire au hasard <strong>un sein malin et un sein bénin</strong>, et qu'on demande au <Sigle t="CNN" /> lequel est le plus suspect.
      L'<Sigle t="AUC" /> est la proportion de fois où il désigne le bon. Elle va de 0,5 à 1 :
    </p>
    <div class="table-wrap"><table class="echelle">
      <thead><tr><th><Sigle t="AUC" /></th><th>Ce que ça veut dire</th><th>Sur 100 paires</th></tr></thead>
      <tbody>
        <tr><td>0,50</td><td>Pile ou face : le <Sigle t="CNN" /> n'a rien appris.</td><td>50 bien classées</td></tr>
        <tr><td>0,60 à 0,70</td><td>Il a trouvé quelques indices, encore peu fiables.</td><td>60 à 70</td></tr>
        <tr><td>0,70 à 0,80</td><td>Correct : il distingue souvent les deux.</td><td>70 à 80</td></tr>
        <tr><td>0,80 à 0,90</td><td>Bon.</td><td>80 à 90</td></tr>
        <tr><td>0,90 et plus</td><td>Excellent, rarement atteint sur une tâche aussi difficile.</td><td>90 et plus</td></tr>
      </tbody>
    </table></div>
    <ul>
      <li><strong>Pourquoi l'<Sigle t="AUC" /> plutôt qu'un « pourcentage de bonnes réponses » ?</strong> Le <Sigle t="CNN" /> donne un score, pas un oui ou non.
        L'<Sigle t="AUC" /> juge s'il range bien les seins du plus suspect au moins suspect, sans dépendre du seuil d'alerte choisi. Le seuil est fixé à la fin.</li>
      <li><strong>Par sein ou par image ?</strong> Chaque sein est radiographié sous deux angles (<Sigle t="CC" /> et <Sigle t="MLO" />). L'<Sigle t="AUC" /> par image juge chaque image seule ;
        l'<Sigle t="AUC" /> par sein fait la moyenne des deux scores d'un même sein. C'est la plus utile, car c'est le sein qu'on diagnostique : c'est elle qui décide de l'époque gardée.</li>
      <li><strong>Que vaut un gain ?</strong> +0,01 d'<Sigle t="AUC" />, c'est une paire de plus bien classée sur 100. Cela paraît peu ; à l'échelle d'un programme de dépistage,
        quelques points d'<Sigle t="AUC" /> font une vraie différence en cancers trouvés ou en fausses alertes évitées.</li>
    </ul>

    <h3>La perte</h3>
    <p>
      La perte mesure l'erreur moyenne du <Sigle t="CNN" /> sur les images d'<strong>entraînement</strong>, celles qu'il est en train d'apprendre. Plus elle est basse, mieux c'est ;
      elle doit baisser au fil des époques. Mais une perte qui baisse ne suffit pas : seul le progrès sur des images inconnues (l'<Sigle t="AUC" /> d'ajustement) compte vraiment.
    </p>
    <p>
      Concrètement, c'est une entropie croisée : une mauvaise réponse coûte d'autant plus que le <Sigle t="CNN" /> en était sûr. Les seins malins y pèsent davantage
      pour compenser leur proportion, si bien que sa valeur absolue n'a pas de sens seule : on regarde sa tendance. Sur le graphique, elle suit l'axe de droite,
      zoomé sur les valeurs atteintes : une baisse de 2 % peut y paraître spectaculaire. Formules : <NuxtLink to="/suivi/documentation/mathematiques#cnn">modèle mathématique</NuxtLink>.
    </p>

    <h3>L'évolution au fil des époques</h3>
    <p>Une <strong>époque</strong> est terminée quand le <Sigle t="CNN" /> a vu une fois chacune des images d'entraînement. Le plus souvent, la courbe suit trois phases :</p>
    <ol>
      <li><strong>Les premières époques</strong> : gros progrès. Le <Sigle t="CNN" /> découvre les indices les plus évidents.</li>
      <li><strong>Le ralentissement</strong> : les gains deviennent petits. Il affine des détails.</li>
      <li><strong>Le plateau</strong> : plus de progrès, voire un recul. S'il continue, il risque d'apprendre les images par cœur au lieu de comprendre (surapprentissage).</li>
    </ol>
    <p>
      Deux garde-fous automatiques : la version gardée est toujours celle de la <strong>meilleure époque</strong> (pas forcément la dernière),
      et l'entraînement s'arrête de lui-même s'il n'y a plus de record pendant 5 époques (<strong>arrêt anticipé</strong>, réglable par la « patience »).
    </p>

    <h3>Trois exemples pour apprendre à lire la courbe</h3>
    <p class="small muted">Chiffres illustratifs. Sous le graphique d'un entraînement réel, l'encadré « Lecture » fait la même analyse sur ses vrais chiffres.</p>
    <div class="exemples">
      <article v-for="e in lus" :key="e.titre" class="exemple">
        <h4>{{ e.titre }}</h4>
        <CourbeExemple :journal="e.journal" :record="e.lu.epoqueMeilleure" />
        <div class="table-wrap"><table class="small">
          <thead><tr><th>Époque</th><th><Sigle t="AUC" /> par sein</th><th>Gain</th><th>Perte</th></tr></thead>
          <tbody>
            <tr v-for="(l, i) in e.journal" :key="l.epoque" :class="{ record: l.epoque === e.lu.epoqueMeilleure }">
              <td>{{ l.epoque }}</td><td>{{ fr3(l.auc_sein!) }}</td><td>{{ gain(e.journal, i) }}</td><td>{{ fr3(l.perte) }}</td>
            </tr>
          </tbody>
        </table></div>
        <p class="small">{{ e.lecture }}</p>
      </article>
    </div>

    <h3>Le résultat final</h3>
    <p>
      À la fin, le <Sigle t="CNN" /> reçoit un <strong>seuil</strong> : au-dessus, il s'alerte. Ce seuil est choisi sur l'ajustement pour trouver 85 % des seins malins.
      Puis il passe l'examen final sur les patientes de validation :
    </p>
    <ul>
      <li><strong>Sensibilité</strong> : la part des seins malins signalés. 0,85 = sur 100 seins malins, 85 signalés et 15 manqués.</li>
      <li><strong>Spécificité</strong> : la part des seins bénins reconnus comme tels. 0,60 = sur 100 seins bénins, 60 laissés tranquilles et 40 fausses alertes.</li>
      <li>Les deux s'opposent : baisser le seuil trouve plus de cancers mais donne plus de fausses alertes. Une meilleure <Sigle t="AUC" /> améliore les deux à la fois.</li>
    </ul>
    <p class="small muted">Outil de recherche : aucun résultat ne sert à un diagnostic.</p>
  </details>
</template>

<style scoped>
.guide summary { cursor: pointer; font-weight: 600; }
.guide h3 { margin: 18px 0 6px; font-size: 1rem; }
.guide p, .guide ul, .guide ol { margin: 8px 0 0; }
.guide ul, .guide ol { padding-left: 20px; display: grid; gap: 4px; }
.echelle td:first-child, .echelle td:last-child { white-space: nowrap; }
.exemples { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(280px, 100%), 1fr)); gap: 12px; margin-top: 8px; }
.exemple { border: 1px solid var(--border); border-radius: var(--radius); padding: 10px 12px; }
.exemple h4 { margin: 0 0 4px; font-size: 0.95rem; }
.exemple table { margin-top: 6px; }
tr.record td { font-weight: 600; color: var(--accent); }
</style>
