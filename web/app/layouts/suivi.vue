<script setup lang="ts">
import { roleLabel } from '~/types'
const session = useSessionStore()
</script>

<template>
  <div>
    <header class="app">
      <div class="container row">
        <NuxtLink to="/suivi" class="brand"><img src="/favicon.svg" alt="" width="22" height="22"> MammoMesh</NuxtLink>
        <nav class="row">
          <NuxtLink to="/suivi">Campagnes</NuxtLink>
          <NuxtLink v-if="session.permissions?.proposer_config" to="/suivi/nouvelle">Nouvelle campagne</NuxtLink>
          <NuxtLink to="/suivi/jetons">Jetons MCP</NuxtLink>
        </nav>
        <div class="spacer" />
        <NuxtLink v-if="session.user" to="/suivi/compte" class="small muted account">{{ session.user.display_name || session.user.username }} · {{ roleLabel[session.user.role] }}</NuxtLink>
        <button class="btn" @click="session.logout()">Déconnexion</button>
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
nav a.router-link-exact-active { color: var(--text); background: var(--surface-2); text-decoration: none; }
.account { color: var(--muted); }
.research-band { background: var(--surface-2); color: var(--muted); font-size: 0.78rem; text-align: center; padding: 3px; }
.main { padding-top: 20px; padding-bottom: 40px; }
</style>
