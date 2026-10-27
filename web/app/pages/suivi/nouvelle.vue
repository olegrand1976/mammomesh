<script setup lang="ts">
// Assistant simplifié : configuration → brouillon → contrôles avant lancement → lancement.
import { fr, type Campaign, type CampaignConfig, type Draft, type Generation } from '~/types'
import { champDuControle, lire, sections } from '~/utils/guideCampagne'

definePageMeta({ layout: 'suivi', middleware: 'auth' })
useHead({ title: 'Nouvelle campagne · MammoMesh' })

const { api } = useApi()
const route = useRoute()
const session = useSessionStore()
// Repartir d'une campagne finie (?depart=ID) : ses réglages servent de base, le front de la génération
// choisie (la finale par défaut) fournit les parents de la génération 1.
const source = ref<Campaign | null>(null)
const sourceGens = ref<Generation[]>([])
const config = ref<CampaignConfig | null>(null)
const defaut = ref<CampaignConfig | null>(null)
const aidesToutes = ref(false)
provide('aides-toutes', aidesToutes)
const def = (cle: string) => (defaut.value ? lire(defaut.value, cle) : undefined)
const libelleControle = (nom: string) => champDuControle(nom)?.libelle ?? ({ budget: 'Budget et plafond', depart: 'Point de départ' } as Record<string, string>)[nom] ?? nom
const draft = ref<Draft | null>(null)
const error = ref('')
const busy = ref(false)

async function charger() {
  error.value = ''
  draft.value = null
  source.value = null
  sourceGens.value = []
  const id = Number(route.query.depart)
  if (!id) {
    const d = (await api<{ config: CampaignConfig }>('/config/default')).config
    defaut.value = structuredClone(d)
    config.value = d
    return
  }
  try {
    const [c, g] = await Promise.all([
      api<{ campaign: Campaign }>(`/campaigns/${id}`),
      api<{ generations: Generation[] }>(`/campaigns/${id}/generations`),
    ])
    if (c.campaign.state !== 'terminee' && c.campaign.state !== 'arretee') {
      error.value = 'On ne repart que d\'une campagne terminée ou arrêtée.'
    }
    source.value = c.campaign
    sourceGens.value = g.generations
    const base: CampaignConfig = structuredClone(toRaw(c.campaign.config))
    delete base.depart
    defaut.value = structuredClone(base)
    config.value = { ...structuredClone(base), depart: { campagne_id: id, generation: c.campaign.generation } }
    config.value.projet.nom = `Suite de ${c.campaign.name}`
  } catch (e) {
    error.value = (e as Error).message
  }
}
watch(() => route.query.depart, charger, { immediate: true })

const finale = computed(() => source.value?.generation ?? 0)
// Générations proposées, de la finale à la première ; sans individu réalisable, pas de front : non sélectionnable.
const choixGenerations = computed(() => [...sourceGens.value].reverse().map(g => ({
  numero: g.number,
  libelle: `Génération ${g.number}${g.number === finale.value ? ' (finale)' : ''} · meilleure ${fr(g.best_sensitivity)} · ${g.feasible} réalisables`,
  possible: g.feasible > 0,
})))

watch(config, () => { draft.value = null }, { deep: true })

async function verify() {
  error.value = ''
  busy.value = true
  try {
    const created = (await api<{ draft: Draft }>('/drafts', { method: 'POST', body: { config: config.value, base_campaign_id: source.value?.id } })).draft
    draft.value = (await api<{ draft: Draft }>(`/drafts/${created.id}/verify`, { method: 'POST' })).draft
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    busy.value = false
  }
}

async function launch() {
  if (!draft.value) return
  busy.value = true
  try {
    const c = (await api<{ campaign: Campaign }>(`/drafts/${draft.value.id}/launch`, { method: 'POST' })).campaign
    await navigateTo(`/suivi/campagnes/${c.id}`)
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div>
    <div class="row head">
      <h1>{{ source ? `Repartir de « ${source.name} »` : 'Nouvelle campagne' }}</h1>
      <div class="spacer" />
      <button v-if="config" type="button" class="btn" :aria-pressed="aidesToutes" @click="aidesToutes = !aidesToutes">
        {{ aidesToutes ? 'Masquer les aides' : 'Afficher toutes les aides' }}
      </button>
    </div>
    <details class="card parcours">
      <summary>Comment ça marche</summary>
      <ol>
        <li><strong>Régler.</strong> Chaque réglage montre sa plage et sa valeur par défaut ; le bouton <span class="q">?</span> explique à quoi il sert et ce que change une hausse ou une baisse. Un réglage modifié est signalé et se rétablit d'un clic.</li>
        <li><strong>Vérifier.</strong> « Vérifier la configuration » fige un brouillon identifié par son empreinte, contrôle chaque réglage et estime la durée et le calcul. Rien n'est lancé. Toute modification ensuite efface la vérification : il faut vérifier à nouveau.</li>
        <li><strong>Lancer.</strong> Réservé au responsable scientifique et à l'administrateur ; un brouillon ne se lance qu'une fois. L'orchestrateur prend la campagne dans la seconde et le suivi s'ouvre en direct.</li>
      </ol>
      <p class="muted small">La boucle <Sigle t="CNN" /> + CephaloMesh est simulée dans cette version : les exemples chiffrés des aides viennent de la simulation. Fonctionnement complet d'une campagne : <NuxtLink to="/suivi/documentation">Documentation</NuxtLink> · guide détaillé : <code>docs/guide-nouvelle-campagne.md</code>.</p>
    </details>
    <p v-if="error" class="alert">{{ error }}</p>
    <section v-if="source && config?.depart" class="card depart" aria-labelledby="depart-titre">
      <h2 id="depart-titre">Point de départ</h2>
      <p class="small">Les individus du front de Pareto de la génération choisie deviennent les parents de la première génération ; leur filiation reste visible dans les fiches. Les réglages ci-dessous partent de ceux de
        <NuxtLink :to="`/suivi/campagnes/${source.id}`">« {{ source.name }} »</NuxtLink> : un réglage changé est marqué « modifié ».</p>
      <div class="field">
        <label for="depart-generation">Génération de départ</label>
        <select id="depart-generation" v-model.number="config.depart.generation">
          <option v-for="g in choixGenerations" :key="g.numero" :value="g.numero" :disabled="!g.possible">{{ g.libelle }}</option>
        </select>
        <p class="small muted">Par défaut la génération finale. Une génération intermédiaire repart d'un front moins avancé, utile pour explorer une autre direction depuis un palier.</p>
      </div>
      <NuxtLink to="/suivi/nouvelle" class="small">Partir de zéro à la place</NuxtLink>
    </section>
    <p v-if="session.permissions && !session.permissions.proposer_config" class="notice">
      Votre rôle permet de suivre les campagnes, pas d'en préparer. Demandez au responsable scientifique ou à l'administrateur.
    </p>
    <form v-else-if="config" class="grid" @submit.prevent="verify">
      <div class="grid grid-2">
        <section class="card">
          <h2>{{ sections.projet.titre }}</h2>
          <p class="muted small">{{ sections.projet.intro }}</p>
          <ChampConfig v-model="config.projet.nom" cle="projet.nom" :defaut="def('projet.nom')" required />
          <ChampConfig v-model="config.projet.objectif" cle="projet.objectif" :defaut="def('projet.objectif')" />
          <ChampConfig v-model="config.projet.lignee" cle="projet.lignee" :defaut="def('projet.lignee')" pattern="[A-Z]{3,12}" />
          <div class="cols">
            <ChampConfig v-model="config.projet.plancher_sensibilite" cle="projet.plancher_sensibilite" :defaut="def('projet.plancher_sensibilite')" type="number" step="any" min="0.8" max="0.99" />
            <ChampConfig v-model="config.projet.marge_points" cle="projet.marge_points" :defaut="def('projet.marge_points')" type="number" step="any" min="1" max="5" />
            <ChampConfig v-model="config.projet.taux_rappel_cible" cle="projet.taux_rappel_cible" :defaut="def('projet.taux_rappel_cible')" type="number" step="any" min="0.02" max="0.1" />
          </div>
        </section>
        <section class="card">
          <h2>{{ sections.evolution.titre }}</h2>
          <p class="muted small">{{ sections.evolution.intro }}</p>
          <div class="cols">
            <ChampConfig v-model="config.evolution.population" cle="evolution.population" :defaut="def('evolution.population')" type="number" min="20" max="500" />
            <ChampConfig v-model="config.evolution.generations_max" cle="evolution.generations_max" :defaut="def('evolution.generations_max')" type="number" min="10" max="2000" />
            <ChampConfig v-model="config.evolution.arret_stagnation" cle="evolution.arret_stagnation" :defaut="def('evolution.arret_stagnation')" type="number" min="3" max="100" />
            <ChampConfig v-model="config.evolution.budget_coeur_heures" cle="evolution.budget_coeur_heures" :defaut="def('evolution.budget_coeur_heures')" type="number" step="any" min="0.1" />
            <ChampConfig v-model="config.evolution.mutations_mu" cle="evolution.mutations_mu" :defaut="def('evolution.mutations_mu')" type="number" step="any" min="0" max="4" />
            <ChampConfig v-model="config.evolution.mutations_max" cle="evolution.mutations_max" :defaut="def('evolution.mutations_max')" type="number" min="1" max="10" />
            <ChampConfig v-model="config.evolution.iles" cle="evolution.iles" :defaut="def('evolution.iles')" type="number" min="1" max="8" />
            <ChampConfig v-model="config.evolution.graine" cle="evolution.graine" :defaut="def('evolution.graine')" type="number" step="1" />
          </div>
          <h2 class="sous">{{ sections.simulation.titre }}</h2>
          <p class="muted small">{{ sections.simulation.intro }}</p>
          <div class="cols">
            <ChampConfig v-model="config.simulation.duree_lot_ms" cle="simulation.duree_lot_ms" :defaut="def('simulation.duree_lot_ms')" type="number" min="10" max="60000" />
            <ChampConfig v-model="config.simulation.lots_par_generation" cle="simulation.lots_par_generation" :defaut="def('simulation.lots_par_generation')" type="number" min="1" max="100" />
          </div>
        </section>
      </div>
      <ImpactConfig :config="config" />
      <div class="row">
        <button class="btn btn-primary" :disabled="busy">Vérifier la configuration</button>
        <span class="small muted">Contrôle et estime, sans rien lancer.</span>
      </div>
    </form>

    <section v-if="draft?.verification" class="card result">
      <h2>Contrôles avant lancement</h2>
      <p class="small muted">Chaque réglage est comparé à sa plage autorisée. Un seul contrôle en échec bloque le lancement.</p>
      <div class="table-wrap"><table>
        <tbody>
          <tr v-for="c in draft.verification.controles" :key="c.nom" :class="{ ko: !c.ok }">
            <td :aria-label="c.ok ? 'conforme' : 'en échec'">{{ c.ok ? '✓' : '✗' }}</td><td>{{ libelleControle(c.nom) }}</td><td class="muted">{{ c.detail }}</td>
          </tr>
        </tbody>
      </table></div>
      <p class="small">Durée estimée : {{ draft.verification.duree_estimee_min }} min au plus · {{ draft.verification.coeur_heures_estimees }} cœur-heures au plus.
        <span class="muted">« Au plus » : la stagnation ou le budget peuvent arrêter la campagne avant.</span></p>
      <p v-if="draft.diff?.length" class="small muted">Réglages modifiés : {{ draft.diff.map(d => d.champ).join(', ') }}</p>
      <button v-if="session.permissions?.lancer_campagnes" class="btn btn-primary" :disabled="busy || !draft.verified_ok" @click="launch">Lancer la campagne</button>
      <p v-else class="notice small">Le lancement est réservé au responsable scientifique et à l'administrateur.</p>
    </section>
  </div>
</template>

<style scoped>
.head { margin-bottom: 12px; }
.head h1 { font-size: 1.4rem; margin: 0; }
.parcours { margin-bottom: 16px; padding: 12px 18px; }
.parcours summary { cursor: pointer; font-weight: 600; }
.parcours ol { margin: 10px 0 8px; padding-left: 20px; display: grid; gap: 6px; }
.q { display: inline-grid; place-items: center; width: 18px; height: 18px; border-radius: 50%; border: 1px solid var(--border); font-size: 0.7rem; font-weight: 600; }
.cols { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 0 14px; align-items: start; }
h2.sous { margin-top: 10px; }
.result { margin-top: 16px; }
.depart { margin-bottom: 16px; border-left: 3px solid var(--accent); }
.depart select { max-width: 560px; }
td:first-child { width: 24px; }
tr.ko td { color: var(--bad); }
</style>
