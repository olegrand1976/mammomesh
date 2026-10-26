<script setup lang="ts">
definePageMeta({ layout: 'suivi', middleware: 'auth' })
useHead({ title: 'Lexique des mutations · MammoMesh' })
</script>

<template>
  <DocPage titre="Lexique des mutations"
    intro="Chaque individu porte la liste des mutations qui l'ont produit. Un code dit quel gène a changé et dans quel sens : ↑ relevé, ↓ abaissé, + ajouté, − retiré. Dans la fiche d'un individu et dans le journal, chaque code s'explique aussi au survol.">
    <section class="card"><LexiqueMutations ouvert /></section>
  </DocPage>
</template>
