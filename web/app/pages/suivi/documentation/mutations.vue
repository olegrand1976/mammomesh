<script setup lang="ts">
definePageMeta({ layout: 'suivi', middleware: 'auth' })
useHead({ title: 'Lexique des mutations · MammoMesh' })
</script>

<template>
  <DocPage titre="Lexique des mutations"
    intro="MammoMesh essaie de nombreuses versions de lui-même, un peu comme une sélection naturelle accélérée. Chaque version s'appelle un individu, et ses réglages sont ses gènes. Pour créer un nouvel individu, on copie un individu existant et on change un réglage : c'est une mutation. Cette page explique chaque code de mutation.">
    <section class="card">
      <h2>Lire un code</h2>
      <ul class="lire">
        <li>Les lettres disent <strong>quel réglage</strong> a changé (SEU pour le seuil de déclenchement, SYM pour le contrôle de symétrie…).</li>
        <li>Le symbole dit <strong>dans quel sens</strong> : ↑ augmenté, ↓ diminué, + vérification ou élément ajouté, − retiré.</li>
        <li>Exemple : « SEU↑ › BLK+ » veut dire qu'on a d'abord relevé le seuil de déclenchement, puis ajouté un bloc au <Sigle t="CNN" />.</li>
        <li>Chaque réglage appartient à l'une des deux parties : le <Sigle t="CNN" />, l'« œil » qui repère sur l'image ce qui pourrait être une lésion, ou CephaloMesh, le « raisonnement » qui juge ces indices
          (<NuxtLink to="/suivi/documentation/cnn-cephalomesh">CNN et CephaloMesh</NuxtLink>).</li>
        <li>Presque chaque réglage est un compromis entre trois choses : <strong>trouver plus de cancers</strong>, <strong>donner moins de fausses alertes</strong> et <strong>aller plus vite</strong>. La campagne garde les individus qui offrent les meilleurs compromis.</li>
        <li>Dans la fiche d'un individu et dans le journal, il suffit de survoler un code pour voir son explication.</li>
      </ul>
    </section>
    <section class="card"><LexiqueMutations ouvert /></section>
  </DocPage>
</template>

<style scoped>
.lire { padding-left: 20px; display: grid; gap: 6px; margin: 0; }
</style>
