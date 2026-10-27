<script setup lang="ts">
import { roleLabel } from '~/types'
const session = useSessionStore()
</script>

<template>
  <div>
    <header class="app">
      <div class="container row">
        <NuxtLink to="/suivi" class="brand"><img src="/favicon.svg" alt="" width="22" height="22"> MammoMesh</NuxtLink>
        <nav v-if="!session.user?.must_change_password" class="row" aria-label="Navigation du suivi">
          <NuxtLink to="/suivi">Accueil</NuxtLink>
          <NuxtLink to="/suivi/campagnes">Campagnes</NuxtLink>
          <NuxtLink v-if="session.permissions?.proposer_config" to="/suivi/nouvelle">Nouvelle campagne</NuxtLink>
          <NuxtLink to="/suivi/jetons">Jetons MCP</NuxtLink>
          <NuxtLink to="/suivi/images">Images</NuxtLink>
          <NuxtLink to="/suivi/entrainements">Entraînements</NuxtLink>
          <NuxtLink to="/suivi/documentation">Documentation</NuxtLink>
        </nav>
        <div class="spacer" />
        <NuxtLink v-if="session.user" to="/suivi/compte" class="small muted account" title="Mon compte">{{ session.user.display_name || session.user.username }}<span class="hide-sm"> · {{ roleLabel[session.user.role] }}</span></NuxtLink>
        <button class="btn logout" @click="session.logout()">Déconnexion</button>
      </div>
    </header>
    <div class="research-band">Recherche uniquement : aucun résultat ne sert à un diagnostic.</div>
    <main class="container main">
      <slot />
    </main>
  </div>
</template>

<style scoped>
header.app { background: var(--surface); border-bottom: 1px solid var(--border); padding: 10px 0; }
.brand { display: inline-flex; align-items: center; gap: 8px; font-weight: 600; color: var(--text); margin-right: 16px; }
.brand:hover { text-decoration: none; }
nav a { color: var(--muted); padding: 4px 6px; border-radius: 6px; }
nav a.router-link-exact-active,
nav a.router-link-active:not([href="/suivi"]) { color: var(--text); background: var(--surface-2); text-decoration: none; }
.account { color: var(--muted); }
.research-band { background: var(--surface-2); color: var(--muted); font-size: 0.78rem; text-align: center; padding: 3px; }
.main { padding-top: 20px; padding-bottom: 40px; }
/* Téléphone : logo et déconnexion sur une ligne, navigation dessous sur une seule ligne défilante. */
@media (max-width: 640px) {
  header.app { padding: 6px 0; }
  header.app .container { row-gap: 4px; }
  nav { order: 3; width: 100%; flex-wrap: nowrap; overflow-x: auto; gap: 4px; scrollbar-width: none; }
  nav a { white-space: nowrap; font-size: 0.92rem; }
  .logout { padding: 5px 10px; font-size: 0.9rem; }
  .main { padding-top: 14px; }
}
</style>
