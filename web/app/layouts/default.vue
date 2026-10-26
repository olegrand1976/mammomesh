<template>
  <div>
    <header class="site">
      <div class="container row">
        <NuxtLink to="/" class="brand"><img src="/favicon.svg" alt="" width="24" height="24"> MammoMesh</NuxtLink>
        <span class="research-tag">Outil de recherche</span>
        <div class="spacer" />
        <NuxtLink to="/suivi" class="btn">Interface de suivi</NuxtLink>
      </div>
    </header>
    <slot />
    <footer class="site">
      <div class="container small muted">
        MammoMesh est un projet de recherche. Aucun résultat ne sert à un diagnostic ni à une décision clinique.
      </div>
    </footer>
  </div>
</template>

<style scoped>
header.site { border-bottom: 1px solid var(--border); background: var(--surface); padding: 12px 0; }
.brand { display: inline-flex; align-items: center; gap: 8px; font-weight: 600; color: var(--text); font-size: 1.05rem; }
.brand:hover { text-decoration: none; }
footer.site { border-top: 1px solid var(--border); padding: 24px 0; margin-top: 48px; }
</style>
