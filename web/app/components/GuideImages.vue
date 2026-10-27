<script setup lang="ts">
// Guide de la page Images, pour un non-spécialiste : à quoi sert chaque partie, pourquoi tester un individu, comment lire le résultat, limites actuelles.
</script>

<template>
  <details class="card guide">
    <summary>Comprendre cette page : importer, regarder, tester une version de MammoMesh</summary>

    <h3>À quoi sert cette page</h3>
    <p>
      Les campagnes mesurent des milliers de versions de MammoMesh avec des chiffres. Cette page permet de voir, <strong>sur une image précise</strong>,
      ce qu'une de ces versions remarque : comme essayer une voiture après avoir lu sa fiche technique. Elle se lit de gauche à droite :
    </p>
    <ol>
      <li><strong>Importer une mammographie</strong> (colonne de gauche) : un fichier <Sigle t="DICOM" />, le format des appareils de radiologie.
        À l'import, tout ce qui pourrait identifier la patiente est supprimé (nom, identifiant, dates, établissement, nom du fichier) :
        seuls l'image et quelques informations techniques sont gardés (taille, vue, côté, fabricant de l'appareil). Pas d'image ? Des exemples synthétiques, fabriqués de toutes pièces, sont proposés.</li>
      <li><strong>Images importées</strong> : la liste des images déjà disponibles, avec leur taille, leur vue (<Sigle t="CC" /> de dessus, <Sigle t="MLO" /> de biais), le côté (L gauche, R droit) et le nombre de tests déjà faits. Un clic l'ouvre.</li>
      <li><strong>La visionneuse</strong> (au centre) : l'image en pleine résolution, pour la regarder comme un radiologue (zoom, contraste, inversion noir et blanc).</li>
      <li><strong>Tester un individu</strong> (à droite) : faire analyser l'image par une version de MammoMesh choisie dans une campagne, puis voir les zones qu'elle signale sur l'image.</li>
    </ol>

    <h3>Pourquoi « tester un individu » ?</h3>
    <p>
      Un <strong>individu</strong> est une version complète de MammoMesh (un <Sigle t="CNN" /> et des réglages de CephaloMesh) produite par une campagne.
      Chaque campagne ne propose pas une seule version mais plusieurs bons compromis : certaines trouvent plus de cancers, d'autres vont plus vite.
      Tester un individu sur une image permet :
    </p>
    <ul>
      <li>de <strong>voir concrètement</strong> ce qu'il signale, et où, au lieu de se fier seulement à sa sensibilité moyenne ;</li>
      <li>de <strong>comparer</strong> deux versions sur la même image, par exemple la plus sûre et la plus rapide d'une campagne : la plus sûre signale en général plus de zones ;</li>
      <li>de <strong>vérifier sur un cas dont on connaît la réponse</strong> : un radiologue peut juger si les zones signalées ont un sens.</li>
    </ul>
    <p>
      Par défaut, la page choisit la dernière campagne terminée, sa génération finale et, parmi les meilleurs compromis de cette génération (le front),
      la version qui trouve le plus de cancers sur les images de contrôle (<Sigle t="E-val" />). Seules les versions du front sont proposées : les autres font moins bien sur tous les points.
    </p>

    <h3>Lire le résultat</h3>
    <ul>
      <li><strong>Score</strong> : le degré de suspicion de l'image, de 0 (rien de suspect) à 1 (très suspect). Ce n'est pas une probabilité de cancer : il sert à classer les images entre elles.</li>
      <li><strong>Zones</strong> : les endroits signalés, numérotés, chacun avec son type et son propre score. Cliquer sur une zone, dans l'image, la met en évidence.
        Types : <em>masse</em> (une grosseur), <em>microcalcifications</em> (minuscules dépôts de calcium, parfois groupés), <em>distorsion architecturale</em> (le tissu paraît tiré ou déformé),
        <em>asymétrie</em> (une zone plus dense que du côté opposé).</li>
      <li><strong>Durée</strong> : le temps de calcul de cette version pour cette image, en millisecondes.</li>
      <li><strong>Tests précédents</strong> : chaque test est gardé ; on peut revoir le résultat d'une autre version sur la même image.</li>
    </ul>

    <h3>Limites actuelles</h3>
    <ul>
      <li><strong>Score du <Sigle t="CNN" /> ou simulation.</strong> Si le service d'inférence tourne (mammomesh-inference), le score vient du <Sigle t="CNN" /> entraîné et les zones montrent où il a regardé ; CephaloMesh n'est pas encore branché, l'individu choisi n'y change donc rien. Sinon, score et zones sont tirés au hasard (toujours les mêmes pour une même image et une même version) et n'ont aucune valeur.</li>
      <li><strong>Une image, pas un examen.</strong> Un examen complet compte quatre images (deux vues par sein). Plusieurs vérifications de CephaloMesh, comme l'accord entre les deux vues ou la comparaison des deux seins, demandent l'examen entier :
        sur une image seule, elles ne peuvent pas s'appliquer. Le test par examen complet viendra avec le vrai moteur.</li>
      <li>Outil de recherche : aucun résultat ne sert à un diagnostic.</li>
    </ul>
    <p class="small muted">Détail de chaque élément : <NuxtLink to="/suivi/documentation/images">Guide : Images</NuxtLink>.</p>
  </details>
</template>

<style scoped>
.guide { margin-bottom: 14px; }
.guide summary { cursor: pointer; font-weight: 600; }
.guide h3 { margin: 18px 0 6px; font-size: 1rem; }
.guide p, .guide ul, .guide ol { margin: 8px 0 0; }
.guide ul, .guide ol { padding-left: 20px; display: grid; gap: 4px; }
</style>
