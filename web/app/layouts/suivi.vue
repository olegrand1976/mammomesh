<script setup lang="ts">
// Interface connectée : barre latérale repliable (icônes seules), tiroir sur téléphone.
import { roleLabel } from '~/types'
import { navigation } from '~/utils/navigation'

const session = useSessionStore()
const route = useRoute()
const CLE = 'mammomesh.barre-repliee'

const repliee = ref(false)   // bureau : icônes seules
const ouverte = ref(false)   // téléphone : tiroir ouvert

const groupes = computed(() => session.user?.must_change_password
  ? []
  : navigation
      .map(g => ({ ...g, entrees: g.entrees.filter(e => !e.permission || session.permissions?.[e.permission]) }))
      .filter(g => g.entrees.length))

onMounted(() => {
  try { repliee.value = localStorage.getItem(CLE) === '1' } catch { /* stockage indisponible : barre dépliée */ }
})
function basculer() {
  repliee.value = !repliee.value
  try { localStorage.setItem(CLE, repliee.value ? '1' : '0') } catch { /* sans conséquence */ }
}
watch(() => route.fullPath, () => { ouverte.value = false })
</script>

<template>
  <div class="coquille" :class="{ repliee, ouverte }">
    <header class="barre-mobile">
      <button class="btn icone-seule" :aria-expanded="ouverte" aria-controls="barre-laterale" aria-label="Menu" @click="ouverte = true">
        <IconeNav nom="menu" />
      </button>
      <NuxtLink to="/suivi" class="marque"><img src="/favicon.svg" alt="" width="22" height="22"> MammoMesh</NuxtLink>
    </header>

    <aside id="barre-laterale" class="barre" aria-label="Navigation du suivi">
      <div class="tete">
        <NuxtLink to="/suivi" class="marque" title="Accueil"><img src="/favicon.svg" alt="" width="24" height="24"><span class="libelle">MammoMesh</span></NuxtLink>
        <button class="bouton-barre bureau" :aria-label="repliee ? 'Déplier le menu' : 'Replier le menu'" :title="repliee ? 'Déplier le menu' : 'Replier le menu'"
          :aria-expanded="!repliee" @click="basculer">
          <IconeNav :nom="repliee ? 'deplier' : 'replier'" />
        </button>
        <button class="bouton-barre mobile" aria-label="Fermer le menu" @click="ouverte = false"><IconeNav nom="fermer" /></button>
      </div>

      <nav class="liens">
        <div v-for="(g, i) in groupes" :key="i" class="groupe">
          <p v-if="g.titre" class="titre-groupe"><span class="libelle">{{ g.titre }}</span></p>
          <NuxtLink v-for="e in g.entrees" :key="e.to" :to="e.to" class="entree" :title="repliee ? e.titre : undefined">
            <IconeNav :nom="e.icone" /><span class="libelle">{{ e.titre }}</span>
          </NuxtLink>
        </div>
      </nav>

      <div class="pied">
        <NuxtLink v-if="session.user" to="/suivi/compte" class="entree compte" title="Mon compte">
          <IconeNav nom="compte" />
          <span class="libelle">
            <span class="nom">{{ session.user.display_name || session.user.username }}</span>
            <span class="role">{{ roleLabel[session.user.role] }}</span>
          </span>
        </NuxtLink>
        <button class="entree sortie" :title="repliee ? 'Déconnexion' : undefined" @click="session.logout()">
          <IconeNav nom="sortie" /><span class="libelle">Déconnexion</span>
        </button>
      </div>
    </aside>
    <div v-if="ouverte" class="voile" aria-hidden="true" @click="ouverte = false" />

    <div class="contenu">
      <div class="research-band">Recherche uniquement : aucun résultat ne sert à un diagnostic.</div>
      <main class="container main">
        <slot />
      </main>
    </div>
  </div>
</template>

<style scoped>
.coquille { --largeur: 232px; display: grid; grid-template-columns: var(--largeur) minmax(0, 1fr); min-height: 100vh; transition: grid-template-columns 0.18s; }
.coquille.repliee { --largeur: 64px; }

.barre { position: sticky; top: 0; height: 100vh; display: flex; flex-direction: column; gap: 8px; background: var(--surface);
  border-right: 1px solid var(--border); padding: 12px 10px; overflow: hidden auto; z-index: 20; }
.tete { display: flex; align-items: center; gap: 6px; min-height: 36px; }
.marque { display: inline-flex; align-items: center; gap: 8px; font-weight: 600; color: var(--text); white-space: nowrap; padding: 0 6px; }
.marque:hover { text-decoration: none; }
.bouton-barre { margin-left: auto; display: inline-flex; padding: 6px; border: 0; border-radius: 8px; background: none; color: var(--muted); cursor: pointer; }
.bouton-barre:hover { background: var(--surface-2); color: var(--text); }
.bouton-barre.mobile { display: none; }

.liens { display: grid; gap: 12px; align-content: start; flex: 1; }
.groupe { display: grid; gap: 2px; }
.titre-groupe { margin: 0 0 2px; padding: 0 10px; font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.05em; color: var(--muted); white-space: nowrap; }
.entree { display: flex; align-items: center; gap: 10px; padding: 8px 10px; border-radius: 8px; color: var(--muted); white-space: nowrap;
  font: inherit; border: 0; background: none; cursor: pointer; text-align: left; width: 100%; }
.entree:hover { background: var(--surface-2); color: var(--text); text-decoration: none; }
.entree.router-link-exact-active,
.entree.router-link-active:not([href="/suivi"]) { background: var(--accent-soft); color: var(--accent); font-weight: 600; }
.pied { display: grid; gap: 2px; border-top: 1px solid var(--border); padding-top: 8px; }
.compte .libelle { display: grid; line-height: 1.2; }
.compte .nom { color: var(--text); overflow: hidden; text-overflow: ellipsis; }
.compte .role { font-size: 0.75rem; }

/* Repliée : icônes seules ; les libellés restent lus par les lecteurs d'écran. */
.repliee .libelle { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0 0 0 0); white-space: nowrap; }
.repliee .titre-groupe { height: 1px; margin: 4px 10px; padding: 0; background: var(--border); }
.repliee .tete { flex-direction: column; }
.repliee .bouton-barre { margin-left: 0; }
.repliee .entree { justify-content: center; padding: 9px 0; }

.barre-mobile { display: none; }
.contenu { min-width: 0; }
.research-band { background: var(--surface-2); color: var(--muted); font-size: 0.78rem; text-align: center; padding: 3px; }
.main { padding-top: 20px; padding-bottom: 40px; }
.voile { display: none; }

/* Téléphone et petite tablette : barre compacte en haut, menu en tiroir. */
@media (max-width: 760px) {
  .coquille, .coquille.repliee { grid-template-columns: minmax(0, 1fr); --largeur: 260px; }
  .barre-mobile { display: flex; align-items: center; gap: 8px; position: sticky; top: 0; z-index: 15; padding: 6px 12px;
    background: var(--surface); border-bottom: 1px solid var(--border); }
  .icone-seule { padding: 6px; }
  .barre { position: fixed; left: 0; top: 0; width: var(--largeur); transform: translateX(-100%); visibility: hidden;
    transition: transform 0.2s, visibility 0s 0.2s; box-shadow: var(--shadow); }
  .ouverte .barre { transform: none; visibility: visible; transition: transform 0.2s; }
  .bouton-barre.bureau { display: none; }
  .bouton-barre.mobile { display: inline-flex; }
  .repliee .libelle { position: static; width: auto; height: auto; clip: auto; }
  .repliee .titre-groupe { height: auto; margin: 0 0 2px; padding: 0 10px; background: none; }
  .repliee .tete { flex-direction: row; }
  .repliee .entree { justify-content: flex-start; padding: 8px 10px; }
  .voile { display: block; position: fixed; inset: 0; background: rgb(0 0 0 / 0.35); z-index: 19; }
  .main { padding-top: 14px; }
}
</style>
