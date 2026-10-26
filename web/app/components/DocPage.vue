<script setup lang="ts">
// Gabarit d'une page de documentation : fil d'Ariane, titre, introduction, sommaire, contenu.
defineProps<{ titre: string; intro?: string; sommaire?: { id: string; titre: string }[] }>()
</script>

<template>
  <article class="doc">
    <p class="small crumbs"><NuxtLink to="/suivi/documentation">Documentation</NuxtLink> ›</p>
    <h1>{{ titre }}</h1>
    <p v-if="intro" class="muted">{{ intro }}</p>
    <nav v-if="sommaire?.length" class="card sommaire" aria-label="Sommaire">
      <ol><li v-for="s in sommaire" :key="s.id"><a :href="`#${s.id}`">{{ s.titre }}</a></li></ol>
    </nav>
    <slot />
  </article>
</template>

<style scoped>
.doc { max-width: 900px; display: grid; gap: 16px; }
.doc h1 { font-size: 1.4rem; margin: 0; }
.crumbs { margin: 0; }
.sommaire ol { margin: 0; padding-left: 20px; columns: 2 240px; }
.doc :deep(section) { scroll-margin-top: 16px; }
</style>
